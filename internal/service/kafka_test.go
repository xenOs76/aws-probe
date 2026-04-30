package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestKafkaService_getClientOptions(t *testing.T) {
	tests := []struct {
		name    string
		kcfg    KafkaConfig
		wantErr bool
	}{
		{
			name: "basic config",
			kcfg: KafkaConfig{
				Brokers: []string{"localhost:9092"},
			},
			wantErr: false,
		},
		{
			name: "with TLS",
			kcfg: KafkaConfig{
				Brokers: []string{"localhost:9092"},
				UseTLS:  true,
			},
			wantErr: false,
		},
		{
			name: "with IAM auth",
			kcfg: KafkaConfig{
				Brokers: []string{"localhost:9092"},
				Auth:    "iam",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewKafkaService(aws.Config{
				Credentials: aws.NewCredentialsCache(
					aws.CredentialsProviderFunc(func(_ context.Context) (aws.Credentials, error) {
						return aws.Credentials{
							AccessKeyID:     "test",
							SecretAccessKey: "test",
						}, nil
					}),
				),
			}, nil)

			opts, err := s.getClientOptions(context.Background(), tt.kcfg)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, opts)
			}
		})
	}
}

func TestKafkaLogger(t *testing.T) {
	s := NewKafkaService(aws.Config{}, slog.Default())
	l := &kgoLogger{s: s}
	assert.Equal(t, kgo.LogLevelDebug, l.Level())
	l.Log(kgo.LogLevelError, "test error")
	l.Log(kgo.LogLevelWarn, "test warn")
	l.Log(kgo.LogLevelInfo, "test info")
	l.Log(kgo.LogLevelDebug, "test debug")
}

func TestKafkaService_Produce_ClientError(t *testing.T) {
	s := NewKafkaService(aws.Config{}, nil)
	s.clientFactory = func(_ ...kgo.Opt) (*kgo.Client, error) {
		return nil, errors.New("client error")
	}

	err := s.Produce(context.Background(), KafkaConfig{}, nil, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "client error")
}

func TestKafkaService_Consume_ClientError(t *testing.T) {
	s := NewKafkaService(aws.Config{}, nil)
	s.clientFactory = func(_ ...kgo.Opt) (*kgo.Client, error) {
		return nil, errors.New("client error")
	}

	err := s.Consume(context.Background(), KafkaConfig{}, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "client error")
}
