package mcp

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	t.Parallel()

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "dev"}, nil)
	Register(server, &Deps{
		LoadConfig: func(_ context.Context) (aws.Config, error) {
			return aws.Config{}, nil
		},
	})

	require.NotNil(t, server)
	assert.NotNil(t, server)
}
