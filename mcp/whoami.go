package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xenos76/aws-probe/internal/awsutil"
	"github.com/xenos76/aws-probe/internal/whoami"
)

type whoamiOutput struct {
	Account     string `json:"account"`
	Arn         string `json:"arn"`
	UserID      string `json:"userId"`
	AuthMethod  string `json:"authMethod"`
	AuthType    string `json:"authType"`
	RoleARN     string `json:"roleArn,omitempty"`
	ServiceAcct string `json:"serviceAccount,omitempty"`
}

func registerWhoamiTools(server *mcp.Server, deps *Deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "aws_probe_whoami",
		Description: "Return the current AWS caller identity and detected authentication method",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, whoamiOutput, error) {
		cfg, err := loadAWS(ctx, deps)
		if err != nil {
			return nil, whoamiOutput{}, err
		}

		identity, err := whoami.GetCallerIdentity(ctx, whoami.NewSTSClient(cfg))
		if err != nil {
			return nil, whoamiOutput{}, err
		}

		auth := awsutil.DetectAuthMethod()

		return nil, whoamiOutput{
			Account:     identity.Account,
			Arn:         identity.Arn,
			UserID:      identity.UserID,
			AuthMethod:  auth.IdentitySource,
			AuthType:    string(auth.Type),
			RoleARN:     auth.RoleARN,
			ServiceAcct: auth.ServiceAccount,
		}, nil
	})
}
