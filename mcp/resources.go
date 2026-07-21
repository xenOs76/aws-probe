package mcp

import (
	"context"
	_ "embed"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed embed/cli-reference.md
var cliReferenceMD []byte

//go:embed embed/agents.md
var agentsMD []byte

func registerResources(server *mcp.Server) {
	server.AddResource(&mcp.Resource{
		URI:         "aws-probe://docs/agents",
		Name:        "aws-probe-agents",
		Description: "Agent and contributor guide for aws-probe",
		MIMEType:    "text/markdown",
	}, func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      "aws-probe://docs/agents",
				MIMEType: "text/markdown",
				Text:     string(agentsMD),
			}},
		}, nil
	})

	server.AddResource(&mcp.Resource{
		URI:         "aws-probe://docs/cli-reference",
		Name:        "aws-probe-cli-reference",
		Description: "Summary of aws-probe CLI commands and flags",
		MIMEType:    "text/markdown",
	}, func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      "aws-probe://docs/cli-reference",
				MIMEType: "text/markdown",
				Text:     string(cliReferenceMD),
			}},
		}, nil
	})
}
