package mcp

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	internalsecrets "github.com/xenos76/aws-probe/internal/secrets"
)

type secretEntry struct {
	Name string `json:"name"`
	ARN  string `json:"arn"`
}

type listSecretsOutput struct {
	Secrets []secretEntry `json:"secrets"`
}

func registerSecretsTools(server *mcp.Server, deps *Deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "aws_probe_secrets_list",
		Description: "List Secrets Manager secret names and ARNs (values not included)",
	}, secretsListHandler(deps))
}

func secretsListHandler(deps *Deps) func(
	context.Context, *mcp.CallToolRequest, struct{},
) (*mcp.CallToolResult, listSecretsOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (
		*mcp.CallToolResult, listSecretsOutput, error,
	) {
		cfg, err := loadAWS(ctx, deps)
		if err != nil {
			return nil, listSecretsOutput{}, err
		}

		secrets, err := listSecretsForMCP(ctx, internalsecrets.NewClient(cfg))
		if err != nil {
			return nil, listSecretsOutput{}, err
		}

		return nil, listSecretsOutput{Secrets: secrets}, nil
	}
}

func listSecretsForMCP(ctx context.Context, client internalsecrets.Lister) ([]secretEntry, error) {
	secrets := make([]secretEntry, 0)

	input := &secretsmanager.ListSecretsInput{}
	for {
		out, err := client.ListSecrets(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("listing secrets: %w", err)
		}

		for _, s := range out.SecretList {
			if s.Name == nil || s.ARN == nil {
				continue
			}

			secrets = append(secrets, secretEntry{Name: *s.Name, ARN: *s.ARN})
		}

		if out.NextToken == nil || *out.NextToken == "" {
			break
		}

		input.NextToken = out.NextToken
	}

	return secrets, nil
}
