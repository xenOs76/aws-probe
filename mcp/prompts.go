package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerPrompts(server *mcp.Server) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "aws_probe_prompt_check_credentials",
		Description: "Workflow to verify AWS credentials via aws_probe_whoami",
	}, func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Description: "Check AWS credentials and interpret the auth method",
			Messages: []*mcp.PromptMessage{{
				Role: "user",
				Content: &mcp.TextContent{
					Text: "Run the aws_probe_whoami tool. Report account, ARN, and auth method. " +
						"If credentials are missing, list ways to configure AWS access (env vars, profile, SSO, IRSA).",
				},
			}},
		}, nil
	})

	server.AddPrompt(&mcp.Prompt{
		Name:        "aws_probe_prompt_audit_s3_prefix",
		Description: "Audit objects under an S3 bucket prefix",
		Arguments: []*mcp.PromptArgument{
			{Name: "bucket", Description: "S3 bucket name", Required: true},
			{Name: "prefix", Description: "Key prefix", Required: false},
		},
	}, func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		bucket := promptArg(req, "bucket")
		prefix := promptArg(req, "prefix")

		return &mcp.GetPromptResult{
			Description: "List and inspect S3 objects under a prefix",
			Messages: []*mcp.PromptMessage{{
				Role: "user",
				Content: &mcp.TextContent{
					Text: "Using aws_probe_s3_list_objects with bucket \"" + bucket +
						"\" and prefix \"" + prefix + "\", list objects. " +
						"For interesting keys, call aws_probe_s3_get_object_metadata. " +
						"Summarize findings.",
				},
			}},
		}, nil
	})

	server.AddPrompt(&mcp.Prompt{
		Name:        "aws_probe_prompt_cloudfront_cert_report",
		Description: "Review CloudFront distribution certificates and TLS policies",
	}, func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Description: "CloudFront certificate and security policy report",
			Messages: []*mcp.PromptMessage{{
				Role: "user",
				Content: &mcp.TextContent{
					Text: "Run aws_probe_cloudfront_list_certificates. Highlight distributions with soon-to-expire " +
						"certificates, weak minimum TLS versions, or missing ACM details.",
				},
			}},
		}, nil
	})
}

func promptArg(req *mcp.GetPromptRequest, name string) string {
	if req == nil || req.Params == nil || req.Params.Arguments == nil {
		return ""
	}

	return req.Params.Arguments[name]
}
