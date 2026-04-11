package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewS3Cmd(t *testing.T) {
	cmd := newS3Cmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "s3", cmd.Use)
	assert.NotNil(t, cmd.Commands())
	assert.Len(t, cmd.Commands(), 1)
	assert.Equal(t, "list-buckets", cmd.Commands()[0].Use)
}

func TestNewListBucketsCmd(t *testing.T) {
	cmd := newListBucketsCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "list-buckets", cmd.Use)
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
	assert.Len(t, cmd.Commands(), 2)
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

func TestNewRootCmd(t *testing.T) {
	cmd := newRootCmd()

	require.NotNil(t, cmd)
	assert.Equal(t, "aws-probe", cmd.Use)
	assert.NotNil(t, cmd.Commands())
	assert.Len(t, cmd.Commands(), 5)
}
