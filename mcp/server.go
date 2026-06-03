package mcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register adds aws-probe tools, resources, and prompts to an existing MCP server.
func Register(server *mcp.Server, deps *Deps) {
	d := deps.withDefaults()
	registerTools(server, d)
	registerResources(server)
	registerPrompts(server)
}

// Run creates an MCP server, registers aws-probe features, and serves over stdio.
func Run(ctx context.Context, version string, deps *Deps) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "aws-probe",
		Version: version,
		Title:   "aws-probe MCP",
	}, nil)

	Register(server, deps)

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		return fmt.Errorf("mcp server: %w", err)
	}

	return nil
}
