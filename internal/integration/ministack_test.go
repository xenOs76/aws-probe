//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xenos76/aws-probe/internal/awsutil"
	internaliam "github.com/xenos76/aws-probe/internal/iam"
	internals3 "github.com/xenos76/aws-probe/internal/s3"
	internalsecrets "github.com/xenos76/aws-probe/internal/secrets"
	internalsqs "github.com/xenos76/aws-probe/internal/sqs"
	"github.com/xenos76/aws-probe/internal/whoami"
	awsmcp "github.com/xenos76/aws-probe/mcp"
)

func TestMain(m *testing.M) {
	if os.Getenv("AWS_ENDPOINT_URL") == "" {
		os.Stderr.WriteString("skipping integration tests: AWS_ENDPOINT_URL not set\n")
		os.Exit(0)
	}

	if !ministackHealthy() {
		os.Stderr.WriteString("skipping integration tests: Ministack health check failed\n")
		os.Exit(0)
	}

	os.Exit(m.Run())
}

func ministackHealthy() bool {
	endpoint := strings.TrimRight(os.Getenv("AWS_ENDPOINT_URL"), "/")
	client := &http.Client{Timeout: 2 * time.Second}

	resp, err := client.Get(endpoint + "/_ministack/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func loadCfg(t *testing.T) aws.Config {
	t.Helper()

	ctx := context.Background()
	cfg, err := awsutil.PrepareAWSConfig(ctx)
	require.NoError(t, err)

	return cfg
}

func TestWhoami(t *testing.T) {
	cfg := loadCfg(t)
	id, err := whoami.GetCallerIdentity(context.Background(), whoami.NewSTSClient(cfg))
	require.NoError(t, err)
	assert.NotEmpty(t, id.Account)
	assert.NotEmpty(t, id.Arn)
}

func TestIAM(t *testing.T) {
	cfg := loadCfg(t)
	client := internaliam.NewClient(cfg)
	ctx := context.Background()

	roles, err := internaliam.CollectRoles(ctx, client)
	require.NoError(t, err)

	foundRole := false
	for _, r := range roles {
		if strings.HasPrefix(r.RoleName, "aws-probe-") {
			foundRole = true

			break
		}
	}
	assert.True(t, foundRole, "expected aws-probe-* role from terraform")

	policies, err := internaliam.CollectPolicies(ctx, client, "Local")
	require.NoError(t, err)

	foundPolicy := false
	for _, p := range policies {
		if p.PolicyName == "aws-probe-s3-read-only" {
			foundPolicy = true

			break
		}
	}
	assert.True(t, foundPolicy)

	dump, err := internaliam.CollectPolicyDump(ctx, client, "aws-probe-s3-read-only")
	require.NoError(t, err)
	assert.Contains(t, dump.Services, "s3")

	roleDump, err := internaliam.CollectRoleDump(ctx, client, "aws-probe-s3-read-only")
	require.NoError(t, err)
	assert.Equal(t, "aws-probe-s3-read-only", roleDump.Role.RoleName)
	assert.NotEmpty(t, roleDump.AttachedPolicies)
}

func TestS3SQSSecrets(t *testing.T) {
	cfg := loadCfg(t)
	ctx := context.Background()
	s3Client := internals3.NewClient(cfg)

	var buckets strings.Builder
	require.NoError(t, internals3.ListBuckets(ctx, s3Client, &buckets))
	assert.NotContains(t, buckets.String(), "No S3 buckets found.")

	// Use first terraform-seeded object path if bucket listing succeeded.
	bucketName := firstWordAfterHeader(buckets.String())
	if bucketName != "" {
		var objects strings.Builder
		require.NoError(t, internals3.ListBucket(ctx, bucketName, "samples/", false, s3Client, &objects))
		assert.Contains(t, objects.String(), "hello.txt")

		var meta strings.Builder
		require.NoError(t, internals3.GetObjectMetadata(
			ctx, bucketName, "samples/hello.txt",
			s3Client, internals3.NewKMSClient(cfg), internals3.NewKMSClient(cfg), &meta,
		))
		assert.Contains(t, meta.String(), "GENERAL")
	}

	var queues strings.Builder
	require.NoError(t, internalsqs.ListQueues(ctx, internalsqs.NewClient(cfg), &queues))
	assert.Contains(t, queues.String(), "sample-queue")

	var secretsOut strings.Builder
	require.NoError(t, internalsecrets.ListSecrets(ctx, internalsecrets.NewClient(cfg), &secretsOut))
	assert.NotContains(t, secretsOut.String(), "No secrets found.")
}

func firstWordAfterHeader(table string) string {
	for _, line := range strings.Split(table, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "NAME") || strings.HasPrefix(line, "No ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			return fields[0]
		}
	}

	return ""
}

func TestMCPToolsAgainstMinistack(t *testing.T) {
	cfg := loadCfg(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	server := mcp.NewServer(&mcp.Implementation{Name: "integ", Version: "dev"}, nil)
	awsmcp.Register(server, &awsmcp.Deps{
		LoadConfig: func(_ context.Context) (aws.Config, error) { return cfg, nil },
	})

	go func() {
		ss, err := server.Connect(ctx, serverTransport, nil)
		if err != nil {
			return
		}
		<-ctx.Done()
		_ = ss.Close()
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "integ-client", Version: "dev"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	call := func(name string, args map[string]any) map[string]any {
		t.Helper()
		res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		require.NoError(t, err)
		require.False(t, res.IsError, "%s: %v", name, contentText(res))
		text := contentText(res)
		var out map[string]any
		require.NoError(t, json.Unmarshal([]byte(text), &out))

		return out
	}

	who := call("aws_probe_whoami", map[string]any{})
	assert.NotEmpty(t, who["account"])

	roles := call("aws_probe_iam_list_roles", map[string]any{})
	assert.NotEmpty(t, roles["roles"])

	policies := call("aws_probe_iam_list_policies", map[string]any{"scope": "Local"})
	assert.NotEmpty(t, policies["policies"])

	policy := call("aws_probe_iam_dump_policy", map[string]any{"nameOrArn": "aws-probe-s3-read-only"})
	assert.NotEmpty(t, policy)

	role := call("aws_probe_iam_dump_role", map[string]any{"nameOrArn": "aws-probe-s3-read-only"})
	assert.NotEmpty(t, role)

	buckets := call("aws_probe_s3_list_buckets", map[string]any{})
	assert.NotEmpty(t, buckets["buckets"])

	queues := call("aws_probe_sqs_list_queues", map[string]any{})
	assert.NotEmpty(t, queues["queueUrls"])

	secrets := call("aws_probe_secrets_list", map[string]any{})
	assert.NotEmpty(t, secrets["secrets"])

	certs := call("aws_probe_cloudfront_list_certificates", map[string]any{})
	assert.NotNil(t, certs["certificates"])
}

func contentText(res *mcp.CallToolResult) string {
	if res == nil || len(res.Content) == 0 {
		return ""
	}

	if t, ok := res.Content[0].(*mcp.TextContent); ok {
		return t.Text
	}

	return ""
}
