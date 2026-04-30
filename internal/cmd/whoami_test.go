package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testEnvVars = []string{
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",
	"AWS_SESSION_TOKEN",
	"AWS_PROFILE",
	"AWS_WEB_IDENTITY_TOKEN_FILE",
	"AWS_ROLE_ARN",
	"AWS_SERVICE_NAME",
	"AWS_SSO_TOKEN",
	"AWS_SSO_START_URL",
	"AWS_SSO_REGION",
	"AWS_SSO_ACCOUNT_ID",
	"AWS_SSO_ROLE_NAME",
	"ECS_CONTAINER_METADATA_URI",
}

func cleanEnvForTest(t *testing.T) map[string]string {
	t.Helper()

	oldEnv := make(map[string]string)

	for _, k := range testEnvVars {
		if v, ok := os.LookupEnv(k); ok {
			oldEnv[k] = v
			os.Unsetenv(k)
		}
	}

	return oldEnv
}

func restoreEnvForTest(t *testing.T, oldEnv map[string]string) {
	t.Helper()

	for _, k := range testEnvVars {
		os.Unsetenv(k)
	}

	for k, v := range oldEnv {
		os.Setenv(k, v)
	}
}

type mockSTSClient struct {
	output *sts.GetCallerIdentityOutput
	err    error
}

func (m *mockSTSClient) GetCallerIdentity(
	_ context.Context,
	_ *sts.GetCallerIdentityInput,
	_ ...func(*sts.Options),
) (*sts.GetCallerIdentityOutput, error) {
	return m.output, m.err
}

func TestRunWhoami(t *testing.T) {
	tests := []struct {
		name       string
		client     stsIdentityGetter
		want       string
		wantStderr string
		wantErr    bool
	}{
		{
			name: "successful identity",
			client: &mockSTSClient{
				output: &sts.GetCallerIdentityOutput{
					Account: aws.String("123456789012"),
					Arn:     aws.String("arn:aws:iam::123456789012:user/testuser"),
					UserId:  aws.String("AIDAJEXAMPLE"),
				},
			},
			want: "Account:  123456789012\n" +
				"Arn:      arn:aws:iam::123456789012:user/testuser\n" +
				"UserId:   AIDAJEXAMPLE\n",
			wantStderr: "Authentication: EC2 IAM Role (via IMDS)\n\n",
		},
		{
			name: "API error",
			client: &mockSTSClient{
				err: errors.New("access denied"),
			},
			wantErr: true,
		},
		{
			name: "nil fields in output",
			client: &mockSTSClient{
				output: &sts.GetCallerIdentityOutput{},
			},
			want:       "Account:  \nArn:      \nUserId:   \n",
			wantStderr: "Authentication: EC2 IAM Role (via IMDS)\n\n",
		},
		{
			name: "no credentials found",
			client: &mockSTSClient{
				err: fmt.Errorf(
					"operation error STS: %w",
					errors.New("failed to refresh cached credentials"),
				),
			},
			wantStderr: noCredentialsMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldEnv := cleanEnvForTest(t)
			t.Cleanup(func() { restoreEnvForTest(t, oldEnv) })

			stdout, stderr, runErr := captureWhoamiOutput(t, tt.client)

			if tt.wantErr {
				assert.Error(t, runErr)

				return
			}

			require.NoError(t, runErr)
			assert.Equal(t, tt.want, stdout)
			assert.Equal(t, tt.wantStderr, stderr)
		})
	}
}

func captureWhoamiOutput(t *testing.T, client stsIdentityGetter) (stdout, stderr string, runErr error) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)

	stderrR, stderrW, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = stdoutW
	os.Stderr = stderrW

	t.Cleanup(func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	})

	runErr = runWhoami(context.Background(), client)

	stdoutW.Close()
	stderrW.Close()

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var stdoutBuf, stderrBuf bytes.Buffer

	_, err = io.Copy(&stdoutBuf, stdoutR)
	require.NoError(t, err)

	_, err = io.Copy(&stderrBuf, stderrR)
	require.NoError(t, err)

	return stdoutBuf.String(), stderrBuf.String(), runErr
}

func TestIsCredentialError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "cached credentials error",
			err:  errors.New("failed to refresh cached credentials"),
			want: true,
		},
		{
			name: "no IMDS role",
			err:  errors.New("no EC2 IMDS role found"),
			want: true,
		},
		{
			name: "no providers",
			err:  errors.New("no credential providers"),
			want: true,
		},
		{
			name: "anonymous credentials",
			err:  errors.New("AnonymousCredentials"),
			want: true,
		},
		{
			name: "wrapped credential error",
			err: fmt.Errorf(
				"operation error: %w",
				errors.New("failed to refresh cached credentials"),
			),
			want: true,
		},
		{
			name: "unrelated error",
			err:  errors.New("network timeout"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsCredentialError(tt.err))
		})
	}
}

func TestDerefString(t *testing.T) {
	tests := []struct {
		name  string
		input *string
		want  string
	}{
		{name: "non-nil", input: aws.String("hello"), want: "hello"},
		{name: "nil", input: nil, want: ""},
		{name: "empty string", input: aws.String(""), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, derefString(tt.input))
		})
	}
}

func TestDetectAuthMethod(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected AuthMethodType
	}{
		{
			name: "EKS IRSA with token file and role ARN",
			envVars: map[string]string{
				"AWS_WEB_IDENTITY_TOKEN_FILE": "/var/run/secrets/kubernetes.io/serviceaccount/token",
				"AWS_ROLE_ARN":                "arn:aws:iam::123456789012:role/my-app-role",
				"AWS_SERVICE_NAME":            "my-app",
			},
			expected: AuthMethodEKSIRSA,
		},
		{
			name: "Static credentials",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID":     "AKIAIOSFODNN7EXAMPLE",
				"AWS_SECRET_ACCESS_KEY": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			expected: AuthMethodStaticCreds,
		},
		{
			name: "SSO with token",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID": "AKIAIOSFODNN7EXAMPLE",
				"AWS_SSO_TOKEN":     "some-sso-token",
			},
			expected: AuthMethodSSO,
		},
		{
			name: "ECS task role",
			envVars: map[string]string{
				"ECS_CONTAINER_METADATA_URI": "http://169.254.169.254/v3/...",
			},
			expected: AuthMethodECS,
		},
		{
			name:     "Default (no special env - EC2 assumed)",
			envVars:  map[string]string{},
			expected: AuthMethodEC2Role,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldEnv := cleanEnvForTest(t)
			t.Cleanup(func() { restoreEnvForTest(t, oldEnv) })

			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			result := DetectAuthMethod()
			assert.Equal(t, tt.expected, result.Type)
		})
	}
}

func TestDetectAuthMethod_EKSIRSA_Output(t *testing.T) {
	oldEnv := cleanEnvForTest(t)
	t.Cleanup(func() { restoreEnvForTest(t, oldEnv) })

	envVars := map[string]string{
		"AWS_WEB_IDENTITY_TOKEN_FILE": "/var/run/secrets/kubernetes.io/serviceaccount/token",
		"AWS_ROLE_ARN":                "arn:aws:iam::123456789012:role/production-app",
		"AWS_SERVICE_NAME":            "my-service",
	}

	for k, v := range envVars {
		t.Setenv(k, v)
	}

	result := DetectAuthMethod()

	assert.Equal(t, AuthMethodEKSIRSA, result.Type)
	assert.Equal(t, "EKS IRSA (IAM Role for Service Account)", result.IdentitySource)
	assert.Equal(t, "arn:aws:iam::123456789012:role/production-app", result.RoleARN)
	assert.Equal(t, "my-service", result.ServiceAccount)
}

func TestDetectAuthMethod_StaticCreds_Output(t *testing.T) {
	oldEnv := cleanEnvForTest(t)
	t.Cleanup(func() { restoreEnvForTest(t, oldEnv) })

	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")

	result := DetectAuthMethod()

	assert.Equal(t, AuthMethodStaticCreds, result.Type)
	assert.Equal(t, "Static credentials (environment variables)", result.IdentitySource)
	assert.Empty(t, result.RoleARN)
	assert.Empty(t, result.ServiceAccount)
}

func TestDetectAuthMethod_SSO_Output(t *testing.T) {
	oldEnv := cleanEnvForTest(t)
	t.Cleanup(func() { restoreEnvForTest(t, oldEnv) })

	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	t.Setenv("AWS_SSO_TOKEN", "token-value")

	result := DetectAuthMethod()

	assert.Equal(t, AuthMethodSSO, result.Type)
	assert.Equal(t, "AWS IAM Identity Center (SSO)", result.IdentitySource)
}

func TestDetectAuthMethod_ECS_Output(t *testing.T) {
	oldEnv := cleanEnvForTest(t)
	t.Cleanup(func() { restoreEnvForTest(t, oldEnv) })

	t.Setenv("ECS_CONTAINER_METADATA_URI", "http://169.254.169.254/v3/abc123")

	result := DetectAuthMethod()

	assert.Equal(t, AuthMethodECS, result.Type)
	assert.Equal(t, "ECS Task Role", result.IdentitySource)
}

func TestDetectAuthMethod_EC2_Output(t *testing.T) {
	oldEnv := cleanEnvForTest(t)
	t.Cleanup(func() { restoreEnvForTest(t, oldEnv) })

	result := DetectAuthMethod()

	assert.Equal(t, AuthMethodEC2Role, result.Type)
	assert.Equal(t, "EC2 IAM Role (via IMDS)", result.IdentitySource)
}

func TestRunWhoami_WithEKSIRSA(t *testing.T) {
	oldEnv := cleanEnvForTest(t)
	t.Cleanup(func() { restoreEnvForTest(t, oldEnv) })

	t.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", "/var/run/secrets/token")
	t.Setenv("AWS_ROLE_ARN", "arn:aws:iam::123456789012:role/eks-app")
	t.Setenv("AWS_SERVICE_NAME", "my-app")

	client := &mockSTSClient{
		output: &sts.GetCallerIdentityOutput{
			Account: aws.String("123456789012"),
			Arn:     aws.String("arn:aws:iam::123456789012:role/eks-app"),
			UserId:  aws.String("AROAEXAMPLE"),
		},
	}

	stdout, stderr, err := captureWhoamiOutput(t, client)

	require.NoError(t, err)
	assert.Contains(t, stderr, "Authentication: EKS IRSA (IAM Role for Service Account)")
	assert.Contains(t, stderr, "IAM Role: arn:aws:iam::123456789012:role/eks-app")
	assert.Contains(t, stderr, "Service Account: my-app")
	assert.Contains(t, stdout, "Account:")
	assert.Contains(t, stdout, "Arn:")
	assert.Contains(t, stdout, "UserId:")
}

func TestDetectAuthMethod_AWSProfile(t *testing.T) {
	oldEnv := cleanEnvForTest(t)
	t.Cleanup(func() { restoreEnvForTest(t, oldEnv) })

	t.Setenv("AWS_PROFILE", "my-profile")

	result := DetectAuthMethod()

	assert.Equal(t, AuthMethodAWSProfile, result.Type)
	assert.Equal(t, "AWS Profile: my-profile", result.IdentitySource)
}

func TestDetectAuthMethod_SSOEnvironmentVars(t *testing.T) {
	oldEnv := cleanEnvForTest(t)
	t.Cleanup(func() { restoreEnvForTest(t, oldEnv) })

	t.Setenv("AWS_SSO_START_URL", "https://my-company.awsapps.com/start")
	t.Setenv("AWS_SSO_REGION", "us-east-1")
	t.Setenv("AWS_SSO_ACCOUNT_ID", "123456789012")
	t.Setenv("AWS_SSO_ROLE_NAME", "AdministratorAccess")

	result := DetectAuthMethod()

	assert.Equal(t, AuthMethodSSO, result.Type)
	assert.Equal(t, "AWS IAM Identity Center (SSO)", result.IdentitySource)
}
