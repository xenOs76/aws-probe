package mcp

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	internaliam "github.com/xenos76/aws-probe/internal/iam"
)

var errIAMNameOrArnRequired = errors.New("nameOrArn is required")

type listRolesOutput struct {
	Roles []internaliam.RoleSummary `json:"roles"`
}

type listPoliciesInput struct {
	Scope string `json:"scope,omitempty" jsonschema:"Policy scope: Local, AWS, or All (default Local)"`
}

type listPoliciesOutput struct {
	Policies []internaliam.PolicySummary `json:"policies"`
}

type dumpPolicyInput struct {
	NameOrArn string `json:"nameOrArn" jsonschema:"Managed policy name or ARN"`
}

type dumpRoleInput struct {
	NameOrArn string `json:"nameOrArn" jsonschema:"IAM role name or ARN"`
}

func registerIAMTools(server *mcp.Server, deps *Deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "aws_probe_iam_list_roles",
		Description: "List IAM roles in the account",
	}, iamListRolesHandler(deps))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "aws_probe_iam_list_policies",
		Description: "List IAM managed policies (scope: Local, AWS, or All)",
	}, iamListPoliciesHandler(deps))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "aws_probe_iam_dump_policy",
		Description: "Dump a managed IAM policy document and the AWS services it grants",
	}, iamDumpPolicyHandler(deps))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "aws_probe_iam_dump_role",
		Description: "Dump an IAM role trust policy plus attached and inline policies",
	}, iamDumpRoleHandler(deps))
}

func iamListRolesHandler(deps *Deps) func(
	context.Context, *mcp.CallToolRequest, struct{},
) (*mcp.CallToolResult, listRolesOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (
		*mcp.CallToolResult, listRolesOutput, error,
	) {
		cfg, err := loadAWS(ctx, deps)
		if err != nil {
			return nil, listRolesOutput{}, err
		}

		roles, err := internaliam.CollectRoles(ctx, deps.iam(cfg))
		if err != nil {
			return nil, listRolesOutput{}, err
		}

		return nil, listRolesOutput{Roles: roles}, nil
	}
}

func iamListPoliciesHandler(deps *Deps) func(
	context.Context, *mcp.CallToolRequest, listPoliciesInput,
) (*mcp.CallToolResult, listPoliciesOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in listPoliciesInput) (
		*mcp.CallToolResult, listPoliciesOutput, error,
	) {
		scope := in.Scope
		if scope == "" {
			scope = "Local"
		}

		cfg, err := loadAWS(ctx, deps)
		if err != nil {
			return nil, listPoliciesOutput{}, err
		}

		policies, err := internaliam.CollectPolicies(ctx, deps.iam(cfg), scope)
		if err != nil {
			return nil, listPoliciesOutput{}, err
		}

		return nil, listPoliciesOutput{Policies: policies}, nil
	}
}

func iamDumpPolicyHandler(deps *Deps) func(
	context.Context, *mcp.CallToolRequest, dumpPolicyInput,
) (*mcp.CallToolResult, internaliam.PolicyDump, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in dumpPolicyInput) (
		*mcp.CallToolResult, internaliam.PolicyDump, error,
	) {
		if in.NameOrArn == "" {
			return nil, internaliam.PolicyDump{}, errIAMNameOrArnRequired
		}

		cfg, err := loadAWS(ctx, deps)
		if err != nil {
			return nil, internaliam.PolicyDump{}, err
		}

		dump, err := internaliam.CollectPolicyDump(ctx, deps.iam(cfg), in.NameOrArn)
		if err != nil {
			return nil, internaliam.PolicyDump{}, err
		}

		return nil, dump, nil
	}
}

func iamDumpRoleHandler(deps *Deps) func(
	context.Context, *mcp.CallToolRequest, dumpRoleInput,
) (*mcp.CallToolResult, internaliam.RoleDump, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in dumpRoleInput) (
		*mcp.CallToolResult, internaliam.RoleDump, error,
	) {
		if in.NameOrArn == "" {
			return nil, internaliam.RoleDump{}, errIAMNameOrArnRequired
		}

		cfg, err := loadAWS(ctx, deps)
		if err != nil {
			return nil, internaliam.RoleDump{}, err
		}

		dump, err := internaliam.CollectRoleDump(ctx, deps.iam(cfg), in.NameOrArn)
		if err != nil {
			return nil, internaliam.RoleDump{}, err
		}

		return nil, dump, nil
	}
}
