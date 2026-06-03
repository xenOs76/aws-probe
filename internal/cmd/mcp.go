package cmd

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"
	awsmcp "github.com/xenos76/aws-probe/mcp"
)

func newMcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run Model Context Protocol server for AI agents",
		Long: `Start an MCP server over stdin/stdout that exposes aws-probe AWS inspection tools,
resources, and prompts to AI clients (Cursor, Claude Desktop, etc.).`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return awsmcp.Run(cmd.Context(), Version, &awsmcp.Deps{
				LoadConfig: func(ctx context.Context) (aws.Config, error) {
					return PrepareAWSConfig(ctx)
				},
			})
		},
	}
}
