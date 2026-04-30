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

// AuthMethodType represents the type of AWS authentication being used.
type AuthMethodType string

const (
	// AuthMethodEC2Role indicates the use of an EC2 IAM role via IMDS.
	AuthMethodEC2Role AuthMethodType = "ec2_iam_role"
	// AuthMethodEKSIRSA indicates the use of EKS IAM Roles for Service Accounts.
	AuthMethodEKSIRSA AuthMethodType = "eks_irsa"
	// AuthMethodSSO indicates the use of AWS IAM Identity Center (SSO).
	AuthMethodSSO AuthMethodType = "aws_sso"
	// AuthMethodStaticCreds indicates the use of static credentials via environment variables.
	AuthMethodStaticCreds AuthMethodType = "static_credentials"
	// AuthMethodAWSProfile indicates the use of a named AWS profile.
	AuthMethodAWSProfile AuthMethodType = "aws_profile"
	// AuthMethodECS indicates the use of an ECS task role.
	AuthMethodECS AuthMethodType = "ecs_task_role"
	// AuthMethodUnknown indicates that the authentication method could not be determined.
	AuthMethodUnknown AuthMethodType = "unknown"
)

// AuthMethod contains information about the detected AWS authentication method.
type AuthMethod struct {
	// Type is the categorized type of authentication.
	Type AuthMethodType
	// IdentitySource is a human-readable description of where the identity comes from.
	IdentitySource string
	// RoleARN is the ARN of the IAM role being used, if available.
	RoleARN string
	// ServiceAccount is the name of the EKS service account, if applicable.
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

// IsCredentialError checks if the given error is related to missing or invalid credentials.
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

// LoadAWSConfig loads the AWS configuration.
var LoadAWSConfig = func(ctx context.Context, optFns ...func(*config.LoadOptions) error) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return cfg, err
	}

	if cfg.Region == "" {
		cfg.Region = DefaultAWSRegion
	}

	return cfg, nil
}

// EnsureCredentials checks if AWS credentials are available.
var EnsureCredentials = func() error {
	auth := DetectAuthMethod()

	if auth.Type == AuthMethodUnknown {
		printCredentialsMessage()

		return errors.New("checking credentials: no credentials available")
	}

	return nil
}

// PrepareAWSConfig combines EnsureCredentials and LoadAWSConfig.
var PrepareAWSConfig = func(ctx context.Context, optFns ...func(*config.LoadOptions) error) (aws.Config, error) {
	if err := EnsureCredentials(); err != nil {
		return aws.Config{}, err
	}

	cfg, err := LoadAWSConfig(ctx, optFns...)
	if err != nil {
		return cfg, fmt.Errorf("loading AWS config: %w", err)
	}

	return cfg, nil
}

// printCredentialsMessage prints the no credentials message to stderr.
func printCredentialsMessage() {
	_, _ = fmt.Fprint(os.Stderr, noCredentialsMessage)
}

// DetectAuthMethod detects the current AWS authentication method based on environment variables.
func DetectAuthMethod() AuthMethod {
	if os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE") != "" {
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

	if os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "" {
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

	nextSectionBase := ssoStartIdx + len(profileSection)
	nextSectionIdx := strings.Index(content[nextSectionBase:], "[")

	sectionContent := content[nextSectionBase:]
	if nextSectionIdx != -1 {
		sectionContent = content[nextSectionBase : nextSectionBase+nextSectionIdx]
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
