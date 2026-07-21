package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIAMTools_ValidationAndLoadConfig(t *testing.T) {
	t.Parallel()

	t.Run("dump_policy empty nameOrArn handler", func(t *testing.T) {
		h := iamDumpPolicyHandler(nil)
		_, _, err := h(context.Background(), nil, dumpPolicyInput{})
		require.ErrorIs(t, err, errIAMNameOrArnRequired)
	})

	t.Run("dump_role empty nameOrArn handler", func(t *testing.T) {
		h := iamDumpRoleHandler(nil)
		_, _, err := h(context.Background(), nil, dumpRoleInput{})
		require.ErrorIs(t, err, errIAMNameOrArnRequired)
	})

	t.Run("dump_role CallTool schema requires nameOrArn", func(t *testing.T) {
		session, _ := startTestSession(t, nil)
		res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "aws_probe_iam_dump_role",
			Arguments: map[string]any{},
		})
		require.NoError(t, err)
		require.True(t, res.IsError)
		text, ok := res.Content[0].(*mcp.TextContent)
		require.True(t, ok)
		assert.Contains(t, text.Text, "nameOrArn")
	})

	t.Run("list_roles load config error", func(t *testing.T) {
		session, _ := startTestSession(t, loadConfigErr(errors.New("no iam")))
		res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "aws_probe_iam_list_roles",
			Arguments: map[string]any{},
		})
		require.NoError(t, err)
		require.True(t, res.IsError)
	})

	t.Run("list_policies default scope then load config error", func(t *testing.T) {
		h := iamListPoliciesHandler(loadConfigErr(errors.New("no iam")))
		_, _, err := h(context.Background(), nil, listPoliciesInput{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "loading AWS config")
	})

	t.Run("dump_policy load config error", func(t *testing.T) {
		session, _ := startTestSession(t, loadConfigErr(errors.New("no iam")))
		res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "aws_probe_iam_dump_policy",
			Arguments: map[string]any{"nameOrArn": "aws-probe-s3-read-only"},
		})
		require.NoError(t, err)
		require.True(t, res.IsError)
	})

	t.Run("dump_role load config error", func(t *testing.T) {
		session, _ := startTestSession(t, loadConfigErr(errors.New("no iam")))
		res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
			Name:      "aws_probe_iam_dump_role",
			Arguments: map[string]any{"nameOrArn": "aws-probe-s3-read-only"},
		})
		require.NoError(t, err)
		require.True(t, res.IsError)
	})
}
