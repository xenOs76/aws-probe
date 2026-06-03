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

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "dev"}, nil)
	Register(server, &Deps{
		LoadConfig: func(_ context.Context) (aws.Config, error) {
			return aws.Config{}, nil
		},
	})

	serverErr := make(chan error, 1)

	go func() {
		ss, err := server.Connect(ctx, serverTransport, nil)
		if err != nil {
			serverErr <- err

			return
		}

		<-ctx.Done()

		_ = ss.Close()

		serverErr <- ctx.Err()
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "dev"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	require.NoError(t, err)
	require.NotNil(t, tools)

	toolNames := make([]string, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		toolNames = append(toolNames, tool.Name)
	}

	assert.Contains(t, toolNames, "aws_probe_whoami")

	prompts, err := session.ListPrompts(ctx, &mcp.ListPromptsParams{})
	require.NoError(t, err)
	require.NotNil(t, prompts)

	promptNames := make([]string, 0, len(prompts.Prompts))
	for _, prompt := range prompts.Prompts {
		promptNames = append(promptNames, prompt.Name)
	}

	assert.Contains(t, promptNames, "aws_probe_prompt_check_credentials")
	assert.Contains(t, promptNames, "aws_probe_prompt_audit_s3_prefix")
	assert.Contains(t, promptNames, "aws_probe_prompt_cloudfront_cert_report")

	cancel()

	select {
	case err := <-serverErr:
		require.ErrorIs(t, err, context.Canceled)
	default:
	}
}
