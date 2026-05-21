package awsutil

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

//nolint:revive // test functions with many subtests can be long
func TestDetectAuthMethod(t *testing.T) {
	// Backup env vars
	envVars := []string{
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
	backup := make(map[string]string)

	for _, v := range envVars {
		backup[v] = os.Getenv(v)
		os.Unsetenv(v)
	}

	defer func() {
		for k, v := range backup {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	}()

	t.Run("EKS IRSA", func(t *testing.T) {
		os.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", "/tmp/token")
		os.Setenv("AWS_ROLE_ARN", "arn:role")

		method := DetectAuthMethod()
		assert.Equal(t, AuthMethodEKSIRSA, method.Type)
		os.Unsetenv("AWS_WEB_IDENTITY_TOKEN_FILE")
		os.Unsetenv("AWS_ROLE_ARN")
	})

	t.Run("Static Creds", func(t *testing.T) {
		os.Setenv("AWS_ACCESS_KEY_ID", "key")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "secret")

		method := DetectAuthMethod()
		assert.Equal(t, AuthMethodStaticCreds, method.Type)
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	})
	t.Run("ECS", func(t *testing.T) {
		os.Setenv("ECS_CONTAINER_METADATA_URI", "http://169.254.170.2")

		method := DetectAuthMethod()
		assert.Equal(t, AuthMethodECS, method.Type)
		os.Unsetenv("ECS_CONTAINER_METADATA_URI")
	})

	t.Run("AWS Profile", func(t *testing.T) {
		os.Setenv("AWS_PROFILE", "myprofile")

		method := DetectAuthMethod()
		assert.Equal(t, AuthMethodAWSProfile, method.Type)
		os.Unsetenv("AWS_PROFILE")
	})

	t.Run("AWS SSO Environment", func(t *testing.T) {
		os.Setenv("AWS_SSO_START_URL", "url")

		method := DetectAuthMethod()
		assert.Equal(t, AuthMethodSSO, method.Type)
		os.Unsetenv("AWS_SSO_START_URL")
	})

	t.Run("EC2 Role", func(t *testing.T) {
		method := DetectAuthMethod()
		assert.Equal(t, AuthMethodEC2Role, method.Type)
	})
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
