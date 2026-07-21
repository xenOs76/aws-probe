package mcp

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func startTestSession(t *testing.T, deps *Deps) (*mcp.ClientSession, context.CancelFunc) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	if deps == nil {
		deps = &Deps{
			LoadConfig: func(_ context.Context) (aws.Config, error) {
				return aws.Config{}, nil
			},
		}
	}

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "dev"}, nil)
	Register(server, deps)

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

	t.Cleanup(func() {
		cancel()

		select {
		case <-serverErr:
		default:
		}
	})

	return session, cancel
}

func loadConfigErr(err error) *Deps {
	return &Deps{
		LoadConfig: func(_ context.Context) (aws.Config, error) {
			return aws.Config{}, err
		},
	}
}
