package secrets

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/stretchr/testify/require"
)

type mockSecretsAPI struct {
	listSecretsFunc func(ctx context.Context, params *secretsmanager.ListSecretsInput,
		optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error)
	getSecretValueFunc func(ctx context.Context, params *secretsmanager.GetSecretValueInput,
		optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

func (m *mockSecretsAPI) ListSecrets(ctx context.Context, params *secretsmanager.ListSecretsInput,
	optFns ...func(*secretsmanager.Options),
) (*secretsmanager.ListSecretsOutput, error) {
	return m.listSecretsFunc(ctx, params, optFns...)
}

func (m *mockSecretsAPI) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput,
	optFns ...func(*secretsmanager.Options),
) (*secretsmanager.GetSecretValueOutput, error) {
	return m.getSecretValueFunc(ctx, params, optFns...)
}

func TestListSecrets(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		api := &mockSecretsAPI{
			listSecretsFunc: func(_ context.Context, _ *secretsmanager.ListSecretsInput,
				_ ...func(*secretsmanager.Options),
			) (*secretsmanager.ListSecretsOutput, error) {
				return &secretsmanager.ListSecretsOutput{
					SecretList: []types.SecretListEntry{
						{Name: aws.String("secret1"), ARN: aws.String("arn1")},
						{Name: aws.String("secret2"), ARN: aws.String("arn2")},
					},
				}, nil
			},
		}

		var buf bytes.Buffer

		err := ListSecrets(context.Background(), api, &buf)
		require.NoError(t, err)

		output := buf.String()
		require.Contains(t, output, "NAME")
		require.Contains(t, output, "secret1")
		require.Contains(t, output, "arn1")
		require.Contains(t, output, "secret2")
		require.Contains(t, output, "arn2")
	})

	t.Run("no secrets", func(t *testing.T) {
		api := &mockSecretsAPI{
			listSecretsFunc: func(_ context.Context, _ *secretsmanager.ListSecretsInput,
				_ ...func(*secretsmanager.Options),
			) (*secretsmanager.ListSecretsOutput, error) {
				return &secretsmanager.ListSecretsOutput{
					SecretList: []types.SecretListEntry{},
				}, nil
			},
		}

		var buf bytes.Buffer

		err := ListSecrets(context.Background(), api, &buf)
		require.NoError(t, err)
		require.Contains(t, buf.String(), "No secrets found.")
	})

	t.Run("error", func(t *testing.T) {
		api := &mockSecretsAPI{
			listSecretsFunc: func(_ context.Context, _ *secretsmanager.ListSecretsInput,
				_ ...func(*secretsmanager.Options),
			) (*secretsmanager.ListSecretsOutput, error) {
				return nil, errors.New("api error")
			},
		}

		var buf bytes.Buffer

		err := ListSecrets(context.Background(), api, &buf)
		require.Error(t, err)
		require.Contains(t, err.Error(), "api error")
	})
}

func TestGetSecretValue(t *testing.T) {
	t.Run("success string", func(t *testing.T) {
		api := &mockSecretsAPI{
			getSecretValueFunc: func(_ context.Context, _ *secretsmanager.GetSecretValueInput,
				_ ...func(*secretsmanager.Options),
			) (*secretsmanager.GetSecretValueOutput, error) {
				return &secretsmanager.GetSecretValueOutput{
					SecretString: aws.String("my-secret-value"),
				}, nil
			},
		}

		var buf bytes.Buffer

		err := GetSecretValue(context.Background(), api, "my-secret", &buf)
		require.NoError(t, err)
		require.Equal(t, "my-secret-value\n", buf.String())
	})

	t.Run("success json", func(t *testing.T) {
		api := &mockSecretsAPI{
			getSecretValueFunc: func(_ context.Context, _ *secretsmanager.GetSecretValueInput,
				_ ...func(*secretsmanager.Options),
			) (*secretsmanager.GetSecretValueOutput, error) {
				return &secretsmanager.GetSecretValueOutput{
					SecretString: aws.String(`{"key":"value"}`),
				}, nil
			},
		}

		var buf bytes.Buffer

		err := GetSecretValue(context.Background(), api, "my-secret", &buf)
		require.NoError(t, err)
		require.Contains(t, buf.String(), `"key": "value"`)
	})

	t.Run("error", func(t *testing.T) {
		api := &mockSecretsAPI{
			getSecretValueFunc: func(_ context.Context, _ *secretsmanager.GetSecretValueInput,
				_ ...func(*secretsmanager.Options),
			) (*secretsmanager.GetSecretValueOutput, error) {
				return nil, errors.New("api error")
			},
		}

		var buf bytes.Buffer

		err := GetSecretValue(context.Background(), api, "my-secret", &buf)
		require.Error(t, err)
		require.Contains(t, err.Error(), "api error")
	})
}
