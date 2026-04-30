package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureCredentials(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
	}{
		{
			name:    "no credentials - EC2 role assumed",
			envVars: map[string]string{},
			wantErr: false,
		},
		{
			name: "with access key",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID":     "AKIATEST",
				"AWS_SECRET_ACCESS_KEY": "testsecret",
			},
			wantErr: false,
		},
		{
			name: "with profile",
			envVars: map[string]string{
				"AWS_PROFILE": "test-profile",
			},
			wantErr: false,
		},
		{
			name: "with EKS IRSA",
			envVars: map[string]string{
				"AWS_WEB_IDENTITY_TOKEN_FILE": "/tmp/token",
				"AWS_ROLE_ARN":                "arn:aws:iam::123:role/test",
			},
			wantErr: false,
		},
		{
			name: "with ECS",
			envVars: map[string]string{
				"ECS_CONTAINER_METADATA_URI": "http://169.254.169.254/v3/abc",
			},
			wantErr: false,
		},
		{
			name: "with SSO env vars",
			envVars: map[string]string{
				"AWS_SSO_START_URL": "https://my.awsapps.com",
				"AWS_SSO_REGION":    "us-east-1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldEnv := cleanEnvForTest(t)
			t.Cleanup(func() { restoreEnvForTest(t, oldEnv) })

			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			err := EnsureCredentials()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadAWSConfig(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
	}{
		{
			name:    "default region",
			envVars: map[string]string{},
			wantErr: false,
		},
		{
			name: "with access key",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID":     "AKIATEST",
				"AWS_SECRET_ACCESS_KEY": "testsecret",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldEnv := cleanEnvForTest(t)
			t.Cleanup(func() { restoreEnvForTest(t, oldEnv) })

			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err := LoadAWSConfig(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPrepareAWSConfig(t *testing.T) {
	oldEnv := cleanEnvForTest(t)
	t.Cleanup(func() { restoreEnvForTest(t, oldEnv) })

	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIATEST")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "testsecret")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := PrepareAWSConfig(ctx)
	assert.NoError(t, err)
}

func TestIsSSOProfile(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	awsDir := filepath.Join(tempHome, ".aws")
	err := os.MkdirAll(awsDir, 0o755)
	require.NoError(t, err)

	configContent := `
[profile sso-test]
sso_start_url = https://test.awsapps.com/start
sso_region = us-east-1
sso_account_id = 123456789012
sso_role_name = Admin

[profile other]
region = us-east-1
`
	err = os.WriteFile(filepath.Join(awsDir, "config"), []byte(configContent), 0o644)
	require.NoError(t, err)

	assert.True(t, isSSOProfile("sso-test"))
	assert.False(t, isSSOProfile("other"))
	assert.False(t, isSSOProfile("non-existent"))
}
