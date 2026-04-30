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
	assert.NotNil(t, cmd.Commands())
	assert.Len(t, cmd.Commands(), 3)
}

func TestNewListBucketsCmd(t *testing.T) {
	cmd := newListBucketsCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "list-buckets", cmd.Use)
	assert.NotNil(t, cmd.RunE)
}

func TestNewListBucketCmd(t *testing.T) {
	cmd := newListBucketCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "list-bucket [bucket-name] [path]", cmd.Use)
	assert.NotNil(t, cmd.RunE)
	assert.NotNil(t, cmd.Flags().Lookup("recursive"))
}

func TestNewGetObjectMetadataCmd(t *testing.T) {
	cmd := newGetObjectMetadataCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "get-object-metadata [bucket-name] [key]", cmd.Use)
	assert.NotNil(t, cmd.RunE)
}

func TestNewSqsCmd(t *testing.T) {
	cmd := newSqsCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "sqs", cmd.Use)
	assert.NotNil(t, cmd.Commands())
	assert.Len(t, cmd.Commands(), 1)
	assert.Equal(t, "list-queues", cmd.Commands()[0].Use)
}

func TestNewListQueuesCmd(t *testing.T) {
	cmd := newListQueuesCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "list-queues", cmd.Use)
	assert.NotNil(t, cmd.RunE)
}

func TestNewSecretsCmd(t *testing.T) {
	cmd := newSecretsCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "secrets", cmd.Use)
	assert.NotNil(t, cmd.Commands())
	assert.Len(t, cmd.Commands(), 1)
	assert.Equal(t, "list-secrets", cmd.Commands()[0].Use)
}

func TestNewListSecretsCmd(t *testing.T) {
	cmd := newListSecretsCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "list-secrets", cmd.Use)
	assert.NotNil(t, cmd.RunE)
}

func TestNewMskCmd(t *testing.T) {
	cmd := newMskCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "msk", cmd.Use)
	assert.NotNil(t, cmd.Commands())
	assert.Len(t, cmd.Commands(), 4)
}

func TestNewListClustersCmd(t *testing.T) {
	cmd := newListClustersCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "list-clusters", cmd.Use)
	assert.NotNil(t, cmd.RunE)
}

func TestNewListTopicsCmd(t *testing.T) {
	cmd := newListTopicsCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "list-topics [cluster-arn]", cmd.Use)
	assert.NotNil(t, cmd.RunE)
}

func TestNewProduceCmd(t *testing.T) {
	cmd := newProduceCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "produce [topic] [message]", cmd.Use)
	assert.NotNil(t, cmd.RunE)
	checkCommonKafkaFlags(t, cmd)
	assert.NotNil(t, cmd.Flags().Lookup("message"))
	assert.NotNil(t, cmd.Flags().Lookup("key"))
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

func TestNewConsumeCmd(t *testing.T) {
	cmd := newConsumeCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "consume [topic]", cmd.Use)
	assert.NotNil(t, cmd.RunE)
	checkCommonKafkaFlags(t, cmd)
	assert.NotNil(t, cmd.Flags().Lookup("group"))
	assert.NotNil(t, cmd.Flags().Lookup("from-beginning"))
}

func TestNewRootCmd(t *testing.T) {
	cmd := newRootCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "aws-probe", cmd.Use)
	assert.NotNil(t, cmd.Commands())
	assert.Len(t, cmd.Commands(), 6)
}

func TestNewSnsCmd(t *testing.T) {
	cmd := newSnsCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "sns", cmd.Use)
	assert.NotNil(t, cmd.Commands())
	assert.Len(t, cmd.Commands(), 2)
}

func TestNewSnsListTopicsCmd(t *testing.T) {
	cmd := newSnsListTopicsCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "list-topics", cmd.Use)
	assert.NotNil(t, cmd.RunE)
}

func TestNewSnsListSubscriptionsCmd(t *testing.T) {
	cmd := newSnsListSubscriptionsCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "list-subscriptions [topic-arn]", cmd.Use)
	assert.NotNil(t, cmd.RunE)
}

func TestCommandRunE_Error(t *testing.T) {
	oldPrepare := PrepareAWSConfig

	defer func() { PrepareAWSConfig = oldPrepare }()

	PrepareAWSConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, errors.New("load error")
	}

	tests := []struct {
		name string
		cmd  *cobra.Command
		args []string
	}{
		{"sns list-topics", newSnsListTopicsCmd(), []string{}},
		{"sns list-subscriptions", newSnsListSubscriptionsCmd(), []string{"arn"}},
		{"msk list-clusters", newListClustersCmd(), []string{}},
		{"msk list-topics", newListTopicsCmd(), []string{"arn"}},
		{"msk produce", newProduceCmd(), []string{}},
		{"msk consume", newConsumeCmd(), []string{}},
		{"secrets list-secrets", newListSecretsCmd(), []string{}},
		{"sqs list-queues", newListQueuesCmd(), []string{}},
		{"s3 get-object-metadata", newGetObjectMetadataCmd(), []string{"b", "k"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cmd.RunE(tt.cmd, tt.args)
			require.Error(t, err)
		})
	}
}
