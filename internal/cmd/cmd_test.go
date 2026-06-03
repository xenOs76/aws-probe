package cmd

import (
	"bytes"
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
	assert.Len(t, cmd.Commands(), 9)
	assert.NotNil(t, cmd.Commands()[7])
}

func TestCompletionCommand(t *testing.T) {
	cmd := newRootCmd()
	completionCmd, _, err := cmd.Find([]string{"completion"})
	require.NoError(t, err)
	require.NotNil(t, completionCmd)
	assert.Equal(t, "completion", completionCmd.Name())

	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		shellCmd, _, findErr := cmd.Find([]string{"completion", shell})
		require.NoError(t, findErr)
		require.NotNil(t, shellCmd)
		assert.Equal(t, shell, shellCmd.Name())

		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs([]string{"completion", shell})

		execErr := cmd.Execute()
		require.NoError(t, execErr)
		assert.NotEmpty(t, out.String())
		assert.Contains(t, out.String(), "aws-probe")
	}
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
	assert.NotNil(t, cmd.Flags().Lookup("output"))
	assert.NotNil(t, cmd.Flags().Lookup("theme"))
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
			setFlag(t, c, "list-topics", "true")
			setFlag(t, c, "cluster-arn", "arn")

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
		{"msk positional args are rejected", func() *cobra.Command {
			c := newMskCmd()
			setFlag(t, c, "produce", "true")
			setFlag(t, c, "topic", "t")
			setFlag(t, c, "message", "m")

			return c
		}(), []string{"extra"}},
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

type validateMSKCase struct {
	name    string
	args    []string
	opts    mskOptions
	wantErr string
}

func mskValidationCases() []validateMSKCase {
	return append(mskValidationCoreCases(), mskValidationProduceConsumeCases()...)
}

func mskValidationCoreCases() []validateMSKCase {
	return []validateMSKCase{
		{
			name: "flags without action are rejected (acks)",
			opts: mskOptions{
				acks: 0,
			},
			wantErr: "an action flag is required",
		},
		{
			name: "flags without action are rejected (auth)",
			opts: mskOptions{
				auth: "none",
				acks: -1,
			},
			wantErr: "an action flag is required",
		},
		{
			name: "flags without action are rejected (key)",
			opts: mskOptions{
				auth: "iam",
				acks: -1,
				key:  "test",
			},
			wantErr: "an action flag is required",
		},
		{
			name: "valid produce",
			opts: mskOptions{
				produce: true,
				topic:   "topic",
				message: "message",
				auth:    "iam",
				acks:    -1,
			},
		},
		{
			name: "list-topics requires cluster arn",
			opts: mskOptions{
				listTopics: true,
				auth:       "iam",
				acks:       -1,
			},
			wantErr: "list-topics mode requires --cluster-arn",
		},
		{
			name: "invalid auth",
			opts: mskOptions{
				listClusters: true,
				auth:         "scram",
				acks:         -1,
			},
			wantErr: "invalid --auth value",
		},
		{
			name: "invalid acks",
			opts: mskOptions{
				listClusters: true,
				auth:         "iam",
				acks:         2,
			},
			wantErr: "invalid --acks value",
		},
	}
}

func mskValidationProduceConsumeCases() []validateMSKCase {
	return []validateMSKCase{
		{
			name: "produce requires topic",
			opts: mskOptions{
				produce: true,
				message: "message",
				auth:    "iam",
				acks:    -1,
			},
			wantErr: "produce mode requires --topic",
		},
		{
			name: "produce requires message",
			opts: mskOptions{
				produce: true,
				topic:   "topic",
				auth:    "iam",
				acks:    -1,
			},
			wantErr: "produce mode requires --message",
		},
		{
			name: "consume requires topic",
			opts: mskOptions{
				consume: true,
				auth:    "none",
				acks:    1,
			},
			wantErr: "consume mode requires --topic",
		},
		{
			name: "positional args rejected",
			args: []string{"topic"},
			opts: mskOptions{
				consume: true,
				topic:   "topic",
				auth:    "none",
				acks:    1,
			},
			wantErr: "positional arguments are not supported",
		},
	}
}

func TestValidateMSKOptions(t *testing.T) {
	tests := mskValidationCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMSKOptions(tt.args, tt.opts)
			if tt.wantErr == "" {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)

			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestCommandActionRequiredErrors(t *testing.T) {
	tests := []struct {
		name    string
		cmd     *cobra.Command
		wantErr string
	}{
		{
			name:    "s3 requires action",
			cmd:     newS3Cmd(),
			wantErr: "an action flag is required",
		},
		{
			name:    "sqs requires action",
			cmd:     newSqsCmd(),
			wantErr: "an action flag is required",
		},
		{
			name:    "sns requires action",
			cmd:     newSnsCmd(),
			wantErr: "an action flag is required",
		},
		{
			name:    "secrets requires action",
			cmd:     newSecretsCmd(),
			wantErr: "an action flag is required",
		},
		{
			name:    "cloudfront requires action",
			cmd:     newCloudfrontCmd(),
			wantErr: "an action flag is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cmd.RunE(tt.cmd, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
