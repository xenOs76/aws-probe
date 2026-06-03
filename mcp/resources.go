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

const ministackSummary = `# Ministack local AWS

aws-probe can be tested against [Ministack](https://ministack.org) using Terraform under terraform/ in this repository.
Apply the sample stack to create S3 and related resources, then point AWS credentials at the Ministack endpoint.
`

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

	server.AddResource(&mcp.Resource{
		URI:         "aws-probe://examples/ministack",
		Name:        "aws-probe-ministack",
		Description: "Local AWS testing with Ministack and sample Terraform",
		MIMEType:    "text/markdown",
	}, func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      "aws-probe://examples/ministack",
				MIMEType: "text/markdown",
				Text:     ministackSummary,
			}},
		}, nil
	})
}
