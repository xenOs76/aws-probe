package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister_AllToolsResourcesPrompts(t *testing.T) {
	t.Parallel()

	session, _ := startTestSession(t, nil)
	ctx := context.Background()

	tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	require.NoError(t, err)

	wantTools := []string{
		"aws_probe_whoami",
		"aws_probe_s3_list_buckets",
		"aws_probe_s3_list_objects",
		"aws_probe_s3_get_object_metadata",
		"aws_probe_sqs_list_queues",
		"aws_probe_sqs_get_queue_url",
		"aws_probe_sqs_receive_message",
		"aws_probe_secrets_list",
		"aws_probe_sns_list_topics",
		"aws_probe_msk_list_clusters",
		"aws_probe_msk_list_topics",
		"aws_probe_msk_consume",
		"aws_probe_cloudfront_list_certificates",
		"aws_probe_iam_list_roles",
		"aws_probe_iam_list_policies",
		"aws_probe_iam_dump_policy",
		"aws_probe_iam_dump_role",
	}

	got := make(map[string]bool, len(tools.Tools))
	for _, tool := range tools.Tools {
		got[tool.Name] = true
	}

	for _, name := range wantTools {
		assert.Truef(t, got[name], "missing tool %s", name)
	}

	assert.Len(t, tools.Tools, len(wantTools))

	prompts, err := session.ListPrompts(ctx, &mcp.ListPromptsParams{})
	require.NoError(t, err)

	promptNames := map[string]bool{}
	for _, p := range prompts.Prompts {
		promptNames[p.Name] = true
	}

	assert.True(t, promptNames["aws_probe_prompt_check_credentials"])
	assert.True(t, promptNames["aws_probe_prompt_audit_s3_prefix"])
	assert.True(t, promptNames["aws_probe_prompt_cloudfront_cert_report"])

	resources, err := session.ListResources(ctx, &mcp.ListResourcesParams{})
	require.NoError(t, err)

	uris := map[string]bool{}
	for _, r := range resources.Resources {
		uris[r.URI] = true
	}

	assert.True(t, uris["aws-probe://docs/agents"])
	assert.True(t, uris["aws-probe://docs/cli-reference"])
}

func TestReadResources(t *testing.T) {
	t.Parallel()

	session, _ := startTestSession(t, nil)
	ctx := context.Background()

	for _, uri := range []string{
		"aws-probe://docs/agents",
		"aws-probe://docs/cli-reference",
	} {
		res, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
		require.NoError(t, err, uri)
		require.NotEmpty(t, res.Contents)
		assert.NotEmpty(t, res.Contents[0].Text, uri)
	}
}

func TestGetPrompts(t *testing.T) {
	t.Parallel()

	session, _ := startTestSession(t, nil)
	ctx := context.Background()

	res, err := session.GetPrompt(ctx, &mcp.GetPromptParams{Name: "aws_probe_prompt_check_credentials"})
	require.NoError(t, err)
	require.NotEmpty(t, res.Messages)

	res, err = session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name:      "aws_probe_prompt_audit_s3_prefix",
		Arguments: map[string]string{"bucket": "b", "prefix": "p/"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.Messages)
	text, ok := res.Messages[0].Content.(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "b")
	assert.Contains(t, text.Text, "p/")

	_, err = session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "aws_probe_prompt_audit_s3_prefix",
	})
	require.Error(t, err)

	res, err = session.GetPrompt(ctx, &mcp.GetPromptParams{Name: "aws_probe_prompt_cloudfront_cert_report"})
	require.NoError(t, err)
	require.NotEmpty(t, res.Messages)
}

func TestCallTool_ValidationErrors(t *testing.T) {
	t.Parallel()

	session, _ := startTestSession(t, nil)
	ctx := context.Background()

	cases := []struct {
		name string
		tool string
		args any
	}{
		{"s3 objects missing bucket", "aws_probe_s3_list_objects", map[string]any{}},
		{"s3 metadata missing", "aws_probe_s3_get_object_metadata", map[string]any{"bucket": "b"}},
		{"sqs queue name", "aws_probe_sqs_get_queue_url", map[string]any{}},
		{"sqs queue url", "aws_probe_sqs_receive_message", map[string]any{}},
		{"msk topics", "aws_probe_msk_list_topics", map[string]any{}},
		{"msk consume topic", "aws_probe_msk_consume", map[string]any{}},
		{"iam dump policy", "aws_probe_iam_dump_policy", map[string]any{}},
		{"iam dump role", "aws_probe_iam_dump_role", map[string]any{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tc.tool, Arguments: tc.args})
			require.NoError(t, err)
			require.True(t, res.IsError, "expected tool error for %s", tc.tool)
		})
	}
}

func TestCallTool_LoadConfigError(t *testing.T) {
	t.Parallel()

	session, _ := startTestSession(t, loadConfigErr(errors.New("no aws")))
	ctx := context.Background()

	tools := []struct {
		name string
		args any
	}{
		{"aws_probe_whoami", map[string]any{}},
		{"aws_probe_s3_list_buckets", map[string]any{}},
		{"aws_probe_s3_list_objects", map[string]any{"bucket": "b"}},
		{"aws_probe_s3_get_object_metadata", map[string]any{"bucket": "b", "key": "k"}},
		{"aws_probe_sqs_list_queues", map[string]any{}},
		{"aws_probe_sqs_get_queue_url", map[string]any{"queueName": "q"}},
		{"aws_probe_sqs_receive_message", map[string]any{"queueUrl": "https://q"}},
		{"aws_probe_secrets_list", map[string]any{}},
		{"aws_probe_sns_list_topics", map[string]any{}},
		{"aws_probe_msk_list_clusters", map[string]any{}},
		{"aws_probe_msk_list_topics", map[string]any{"clusterArn": "arn:c"}},
		{"aws_probe_msk_consume", map[string]any{"topic": "t", "brokers": "localhost:9092", "auth": "none"}},
		{"aws_probe_cloudfront_list_certificates", map[string]any{}},
		{"aws_probe_iam_list_roles", map[string]any{}},
		{"aws_probe_iam_list_policies", map[string]any{}},
		{"aws_probe_iam_dump_policy", map[string]any{"nameOrArn": "p"}},
		{"aws_probe_iam_dump_role", map[string]any{"nameOrArn": "r"}},
	}

	for _, tc := range tools {
		t.Run(tc.name, func(t *testing.T) {
			res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tc.name, Arguments: tc.args})
			require.NoError(t, err)
			require.True(t, res.IsError, tc.name)
			require.NotEmpty(t, res.Content)
		})
	}
}

func TestIAMDumpPolicy_RequiredErrorMessage(t *testing.T) {
	t.Parallel()

	session, _ := startTestSession(t, nil)
	ctx := context.Background()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "aws_probe_iam_dump_policy",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	require.True(t, res.IsError)

	text, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "nameOrArn")
}

func TestIAMListPolicies_DefaultScopeReachesLoadConfig(t *testing.T) {
	t.Parallel()

	session, _ := startTestSession(t, loadConfigErr(errors.New("cfg")))
	ctx := context.Background()

	// empty scope should default to Local and still hit LoadConfig
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "aws_probe_iam_list_policies",
		Arguments: map[string]any{},
	})
	require.NoError(t, err)
	require.True(t, res.IsError)

	res, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "aws_probe_iam_list_policies",
		Arguments: map[string]any{"scope": "Local"},
	})
	require.NoError(t, err)
	require.True(t, res.IsError)
}

func TestCallToolResultJSONShape(t *testing.T) {
	t.Parallel()
	// sanity: Arguments marshaling used by CallTool works for nested maps
	b, err := json.Marshal(map[string]any{"bucket": "b", "key": "k"})
	require.NoError(t, err)
	assert.Contains(t, string(b), "bucket")
}
