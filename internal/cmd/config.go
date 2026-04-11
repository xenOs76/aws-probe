package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

type AuthMethodType string

const (
	AuthMethodEC2Role     AuthMethodType = "ec2_iam_role"
	AuthMethodEKSIRSA     AuthMethodType = "eks_irsa"
	AuthMethodSSO         AuthMethodType = "aws_sso"
	AuthMethodStaticCreds AuthMethodType = "static_credentials"
	AuthMethodAWSProfile  AuthMethodType = "aws_profile"
	AuthMethodECS         AuthMethodType = "ecs_task_role"
	AuthMethodUnknown     AuthMethodType = "unknown"
)

type AuthMethod struct {
	Type           AuthMethodType
	IdentitySource string
	RoleARN        string
	ServiceAccount string
}

var credentialErrorHints = []string{
	"no EC2 IMDS role found",
	"failed to refresh cached credentials",
	"no credential providers",
	"AnonymousCredentials",
}

const noCredentialsMessage = `No active AWS credentials found.

Configure credentials using one of the following methods:
  • Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables
  • Run "aws configure" to create ~/.aws/credentials
  • Set AWS_PROFILE to use a named profile
  • Run "aws sso login" if using AWS IAM Identity Center
  • Ensure EC2 instance has IAM role attached
  • Ensure EKS Pod has IRSA configured
`

func IsCredentialError(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()
	for _, hint := range credentialErrorHints {
		if strings.Contains(msg, hint) {
			return true
		}
	}

	return false
}

func LoadAWSConfig(ctx context.Context, optFns ...func(*config.LoadOptions) error) (aws.Config, error) {
	options := []func(*config.LoadOptions) error{
		config.WithRegion(DefaultAWSRegion),
	}
	options = append(options, optFns...)

	return config.LoadDefaultConfig(ctx, options...)
}

func EnsureCredentials() error {
	auth := DetectAuthMethod()

	switch auth.Type {
	case AuthMethodUnknown:
		fmt.Fprint(os.Stderr, noCredentialsMessage)

		return errors.New("checking credentials: no credentials available")
	default:
		return nil
	}
}

func DetectAuthMethod() AuthMethod {
	if tokenFile := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE"); tokenFile != "" {
		return AuthMethod{
			Type:           AuthMethodEKSIRSA,
			IdentitySource: "EKS IRSA (IAM Role for Service Account)",
			RoleARN:        os.Getenv("AWS_ROLE_ARN"),
			ServiceAccount: os.Getenv("AWS_SERVICE_NAME"),
		}
	}

	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		if isSSOProfile(profile) {
			return AuthMethod{
				Type:           AuthMethodSSO,
				IdentitySource: fmt.Sprintf("AWS IAM Identity Center (SSO) - profile: %s", profile),
				RoleARN:        os.Getenv("AWS_ROLE_ARN"),
			}
		}

		return AuthMethod{
			Type:           AuthMethodAWSProfile,
			IdentitySource: fmt.Sprintf("AWS Profile: %s", profile),
		}
	}

	if _, hasKey := os.LookupEnv("AWS_ACCESS_KEY_ID"); hasKey {
		if isSSOEnvironment() {
			return AuthMethod{
				Type:           AuthMethodSSO,
				IdentitySource: "AWS IAM Identity Center (SSO)",
			}
		}

		return AuthMethod{
			Type:           AuthMethodStaticCreds,
			IdentitySource: "Static credentials (environment variables)",
		}
	}

	if os.Getenv("ECS_CONTAINER_METADATA_URI") != "" {
		return AuthMethod{
			Type:           AuthMethodECS,
			IdentitySource: "ECS Task Role",
		}
	}

	if isSSOEnvironment() {
		return AuthMethod{
			Type:           AuthMethodSSO,
			IdentitySource: "AWS IAM Identity Center (SSO)",
		}
	}

	return AuthMethod{
		Type:           AuthMethodEC2Role,
		IdentitySource: "EC2 IAM Role (via IMDS)",
	}
}

func isSSOProfile(profile string) bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	configPath := homeDir + "/.aws/config"

	data, err := os.ReadFile(configPath)
	if err != nil {
		configPath = homeDir + "/.aws/credentials"

		data, err = os.ReadFile(configPath)
		if err != nil {
			return false
		}
	}

	content := string(data)
	profileSection := "[profile " + profile + "]"

	ssoStartIdx := strings.Index(content, profileSection)
	if ssoStartIdx == -1 {
		profileSection = "[" + profile + "]"
		ssoStartIdx = strings.Index(content, profileSection)
	}

	if ssoStartIdx == -1 {
		return false
	}

	nextSectionIdx := strings.Index(content[ssoStartIdx+len(profileSection):], "[")

	sectionContent := content[ssoStartIdx:]
	if nextSectionIdx != -1 {
		sectionContent = content[ssoStartIdx : ssoStartIdx+nextSectionIdx]
	}

	return strings.Contains(sectionContent, "sso_start_url") ||
		strings.Contains(sectionContent, "sso_session") ||
		strings.Contains(sectionContent, "credential_process")
}

func isSSOEnvironment() bool {
	return os.Getenv("AWS_SSO_START_URL") != "" ||
		os.Getenv("AWS_SSO_TOKEN") != "" ||
		os.Getenv("AWS_SSO_ACCOUNT_ID") != "" ||
		os.Getenv("AWS_SSO_ROLE_NAME") != ""
}
