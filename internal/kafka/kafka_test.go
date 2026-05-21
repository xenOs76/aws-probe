package kafka

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

type mockFailingCreds struct{}

func (mockFailingCreds) Retrieve(_ context.Context) (aws.Credentials, error) {
	return aws.Credentials{}, errors.New("mock error")
}

func TestNewService(t *testing.T) {
	t.Run("with logger", func(t *testing.T) {
		logger := slog.Default()
		s := NewService(aws.Config{}, logger)
		assert.Equal(t, logger, s.logger)
		assert.NotNil(t, s.clientFactory)
	})

	t.Run("with nil logger", func(t *testing.T) {
		s := NewService(aws.Config{}, nil)
		assert.NotNil(t, s.logger)
		assert.Equal(t, slog.Default(), s.logger)
		assert.NotNil(t, s.clientFactory)
	})
}

func TestGetClientOptions(t *testing.T) {
	cfg := aws.Config{
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider("AKIA", "SECRET", "TOKEN")),
	}
	svc := NewService(cfg, slog.Default())

	ctx := context.Background()

	t.Run("Basic", func(t *testing.T) {
		kcfg := Config{
			Brokers: []string{"localhost:9092"},
			Auth:    "none",
			UseTLS:  false,
		}

		opts, err := svc.getClientOptions(ctx, kcfg)
		require.NoError(t, err)
		assert.NotEmpty(t, opts) // Should at least have SeedBrokers
	})

	t.Run("IAM", func(t *testing.T) {
		kcfg := Config{
			Brokers: []string{"localhost:9092"},
			Auth:    "iam",
		}

		opts, err := svc.getClientOptions(ctx, kcfg)
		require.NoError(t, err)
		assert.NotEmpty(t, opts) // Should include SASL and TLS
	})

	t.Run("IAM without credentials", func(t *testing.T) {
		badCfg := aws.Config{
			Credentials: aws.NewCredentialsCache(mockFailingCreds{}),
		}

		badSvc := NewService(badCfg, slog.Default())

		kcfg := Config{Auth: "iam"}
		opts, err := badSvc.getClientOptions(ctx, kcfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "mock error")
		assert.Nil(t, opts)
	})

	t.Run("TLS Only", func(t *testing.T) {
		kcfg := Config{UseTLS: true}
		opts, err := svc.getClientOptions(ctx, kcfg)
		require.NoError(t, err)
		assert.NotEmpty(t, opts)
	})
}

func TestProduceClientError(t *testing.T) {
	svc := NewService(aws.Config{}, slog.Default())
	svc.clientFactory = func(...kgo.Opt) (*kgo.Client, error) {
		return nil, errors.New("factory error")
	}

	err := svc.Produce(context.Background(), Config{Acks: 1}, []byte("key"), []byte("val"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "factory error")
}

func TestConsumeClientError(t *testing.T) {
	svc := NewService(aws.Config{}, slog.Default())
	svc.clientFactory = func(...kgo.Opt) (*kgo.Client, error) {
		return nil, errors.New("factory error")
	}

	err := svc.Consume(context.Background(), Config{Group: "g", FromBeginning: true}, func(*Record) {})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "factory error")
}

func TestKgoLogger(t *testing.T) {
	svc := NewService(aws.Config{}, slog.Default())
	kl := &kgoLogger{s: svc}

	assert.Equal(t, kgo.LogLevelDebug, kl.Level())

	// Just exercise the switch statement
	kl.Log(kgo.LogLevelError, "test error")
	kl.Log(kgo.LogLevelWarn, "test warn")
	kl.Log(kgo.LogLevelInfo, "test info")
	kl.Log(kgo.LogLevelDebug, "test debug")
	kl.Log(kgo.LogLevelNone, "test none fallback to debug")
}
