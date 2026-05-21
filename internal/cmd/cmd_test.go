package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewS3Cmd(t *testing.T) {
	cmd := newS3Cmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "s3", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("list-buckets"))
	assert.NotNil(t, cmd.Flags().Lookup("list-bucket"))
	assert.NotNil(t, cmd.Flags().Lookup("get-metadata"))
	assert.NotNil(t, cmd.Flags().Lookup("path"))
	assert.NotNil(t, cmd.Flags().Lookup("recursive"))
	assert.NotNil(t, cmd.Flags().Lookup("key"))
}

func TestNewSqsCmd(t *testing.T) {
	cmd := newSqsCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "sqs", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("list-queues"))
	assert.NotNil(t, cmd.Flags().Lookup("get-queue-url"))
	assert.NotNil(t, cmd.Flags().Lookup("receive-message"))
}

func TestNewSecretsCmd(t *testing.T) {
	cmd := newSecretsCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "secrets", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("list-secrets"))
	assert.NotNil(t, cmd.Flags().Lookup("get-secret-value"))
}

func TestNewMskCmd(t *testing.T) {
	cmd := newMskCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "msk", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("list-clusters"))
	assert.NotNil(t, cmd.Flags().Lookup("list-topics"))
	assert.NotNil(t, cmd.Flags().Lookup("produce"))
	assert.NotNil(t, cmd.Flags().Lookup("consume"))
}

func checkCommonKafkaFlags(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	assert.NotNil(t, cmd.Flags().Lookup("brokers"))
	assert.NotNil(t, cmd.Flags().Lookup("cluster-arn"))
	assert.NotNil(t, cmd.Flags().Lookup("topic"))
	assert.NotNil(t, cmd.Flags().Lookup("auth"))
	assert.NotNil(t, cmd.Flags().Lookup("tls"))
	assert.NotNil(t, cmd.Flags().Lookup("acks"))
	assert.NotNil(t, cmd.Flags().Lookup("verbose"))
}

func TestNewRootCmd(t *testing.T) {
	cmd := newRootCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "aws-probe", cmd.Use)
	assert.NotNil(t, cmd.Commands())
	assert.Len(t, cmd.Commands(), 7)
}

func TestNewSnsCmd(t *testing.T) {
	cmd := newSnsCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "sns", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("list-topics"))
	assert.NotNil(t, cmd.Flags().Lookup("list-subscriptions"))
}

func TestNewCloudfrontCmd(t *testing.T) {
	cmd := newCloudfrontCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "cloudfront", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("list-certificates"))
}

//nolint:revive // maximum number of lines per function exceeded is acceptable for test case exhaustive coverage
func TestCommandRunE_Error(t *testing.T) {
	oldPrepare := PrepareAWSConfig

	defer func() { PrepareAWSConfig = oldPrepare }()

	PrepareAWSConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, errors.New("load error")
	}

	setFlag := func(t *testing.T, cmd *cobra.Command, name, value string) {
		t.Helper()
		require.NoError(t, cmd.Flags().Set(name, value))
	}

	tests := []struct {
		name string
		cmd  *cobra.Command
		args []string
	}{
		{"cloudfront --list-certificates", func() *cobra.Command {
			c := newCloudfrontCmd()
			setFlag(t, c, "list-certificates", "true")

			return c
		}(), []string{}},
		{"sns --list-topics", func() *cobra.Command {
			c := newSnsCmd()
			setFlag(t, c, "list-topics", "true")

			return c
		}(), []string{}},
		{"sns --list-subscriptions", func() *cobra.Command {
			c := newSnsCmd()
			setFlag(t, c, "list-subscriptions", "arn")

			return c
		}(), []string{}},
		{"msk --list-clusters", func() *cobra.Command {
			c := newMskCmd()
			setFlag(t, c, "list-clusters", "true")

			return c
		}(), []string{}},
		{"msk --list-topics", func() *cobra.Command {
			c := newMskCmd()
			setFlag(t, c, "list-topics", "arn")

			return c
		}(), []string{}},
		{"msk --produce", func() *cobra.Command {
			c := newMskCmd()
			setFlag(t, c, "produce", "true")
			setFlag(t, c, "topic", "t")
			setFlag(t, c, "message", "m")

			return c
		}(), []string{}},
		{"msk --consume", func() *cobra.Command {
			c := newMskCmd()
			setFlag(t, c, "consume", "true")
			setFlag(t, c, "topic", "t")

			return c
		}(), []string{}},
		{"secrets --list-secrets", func() *cobra.Command {
			c := newSecretsCmd()
			setFlag(t, c, "list-secrets", "true")

			return c
		}(), []string{}},
		{"secrets --get-secret-value", func() *cobra.Command {
			c := newSecretsCmd()
			setFlag(t, c, "get-secret-value", "id")

			return c
		}(), []string{}},
		{"sqs --list-queues", func() *cobra.Command {
			c := newSqsCmd()
			setFlag(t, c, "list-queues", "true")

			return c
		}(), []string{}},
		{"sqs --get-queue-url", func() *cobra.Command {
			c := newSqsCmd()
			setFlag(t, c, "get-queue-url", "queue-name")

			return c
		}(), []string{}},
		{"sqs --receive-message", func() *cobra.Command {
			c := newSqsCmd()
			setFlag(t, c, "receive-message", "https://sqs.us-east-1.amazonaws.com/123456789012/queue-name")

			return c
		}(), []string{}},
		{"s3 --list-buckets", func() *cobra.Command {
			c := newS3Cmd()
			setFlag(t, c, "list-buckets", "true")

			return c
		}(), []string{}},
		{"s3 --list-bucket", func() *cobra.Command {
			c := newS3Cmd()
			setFlag(t, c, "list-bucket", "b")

			return c
		}(), []string{}},
		{"s3 --get-metadata", func() *cobra.Command {
			c := newS3Cmd()
			setFlag(t, c, "get-metadata", "b")
			setFlag(t, c, "key", "k")

			return c
		}(), []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cmd.RunE(tt.cmd, tt.args)
			require.Error(t, err)
		})
	}
}
