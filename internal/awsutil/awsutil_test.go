package awsutil

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var authMethodEnvVars = []string{
	"AWS_WEB_IDENTITY_TOKEN_FILE",
	"AWS_ROLE_ARN",
	"AWS_SERVICE_NAME",
	"AWS_PROFILE",
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",
	"ECS_CONTAINER_METADATA_URI",
	"AWS_SSO_START_URL",
	"AWS_SSO_TOKEN",
	"AWS_SSO_ACCOUNT_ID",
	"AWS_SSO_ROLE_NAME",
}

func TestIsCredentialError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "credential error",
			err:  errors.New("failed to refresh cached credentials"),
			want: true,
		},
		{
			name: "random error",
			err:  errors.New("something went wrong"),
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
	s := "test"
	assert.Equal(t, "test", DerefString(&s))
	assert.Empty(t, DerefString(nil))
}

func TestDerefInt32(t *testing.T) {
	var i int32 = 42
	assert.Equal(t, int32(42), DerefInt32(&i))
	assert.Equal(t, int32(0), DerefInt32(nil))
}

func TestDerefInt64(t *testing.T) {
	var i int64 = 42
	assert.Equal(t, int64(42), DerefInt64(&i))
	assert.Equal(t, int64(0), DerefInt64(nil))
}

func TestDetectAuthMethod(t *testing.T) {
	tests := []struct {
		name   string
		setup  map[string]string
		expect AuthMethodType
	}{
		{
			name: "EKS IRSA",
			setup: map[string]string{
				"AWS_WEB_IDENTITY_TOKEN_FILE": "/tmp/token",
				"AWS_ROLE_ARN":                "arn:role",
			},
			expect: AuthMethodEKSIRSA,
		},
		{
			name: "Static Creds",
			setup: map[string]string{
				"AWS_ACCESS_KEY_ID":     "key",
				"AWS_SECRET_ACCESS_KEY": "secret",
			},
			expect: AuthMethodStaticCreds,
		},
		{
			name: "ECS",
			setup: map[string]string{
				"ECS_CONTAINER_METADATA_URI": "http://169.254.170.2",
			},
			expect: AuthMethodECS,
		},
		{
			name: "AWS Profile",
			setup: map[string]string{
				"AWS_PROFILE": "myprofile",
			},
			expect: AuthMethodAWSProfile,
		},
		{
			name: "AWS SSO Environment",
			setup: map[string]string{
				"AWS_SSO_START_URL": "url",
			},
			expect: AuthMethodSSO,
		},
		{
			name:   "EC2 Role",
			setup:  map[string]string{},
			expect: AuthMethodEC2Role,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, envVar := range authMethodEnvVars {
				t.Setenv(envVar, "")
			}

			for envVar, value := range tt.setup {
				t.Setenv(envVar, value)
			}

			method := DetectAuthMethod()
			assert.Equal(t, tt.expect, method.Type)
		})
	}
}

func TestLoadAWSConfig(t *testing.T) {
	ctx := context.Background()
	cfg, err := LoadAWSConfig(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, cfg.Region)
}

func TestEnsureCredentials(t *testing.T) {
	err := EnsureCredentials()
	require.NoError(t, err)
}

func TestPrepareAWSConfig(t *testing.T) {
	ctx := context.Background()
	cfg, err := PrepareAWSConfig(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, cfg.Region)
}

func TestIsSSOProfile(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	err := os.Mkdir(tempHome+"/.aws", 0o755)
	require.NoError(t, err)

	configContent := `[default]
sso_start_url = https://example.awsapps.com/start

[profile dev]
region = us-east-1`

	err = os.WriteFile(tempHome+"/.aws/config", []byte(configContent), 0o644)
	require.NoError(t, err)

	assert.True(t, isSSOProfile("default"))
	assert.False(t, isSSOProfile("dev"))
	assert.False(t, isSSOProfile("unknown"))
}
