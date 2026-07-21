package mcp

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	acmtypes "github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	kafkatypes "github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	internalcf "github.com/xenos76/aws-probe/internal/cloudfront"
	internalsns "github.com/xenos76/aws-probe/internal/sns"
	"github.com/xenos76/aws-probe/internal/whoami"
)

type stubAWS struct {
	mockBucketsLister
	mockObjectsLister
	mockObjectHeader
	mockSQSLister
	mockQueueURLGetter
	mockMessageReceiver
	mockSecretsLister
	mockMSKClustersLister
	mockMSKTopicsLister
}

func (*stubAWS) GetBootstrapBrokers(
	_ context.Context, _ *kafka.GetBootstrapBrokersInput, _ ...func(*kafka.Options),
) (*kafka.GetBootstrapBrokersOutput, error) {
	return &kafka.GetBootstrapBrokersOutput{
		BootstrapBrokerStringTls: aws.String("b1:9098"),
	}, nil
}

type stubIAM struct {
	roles    []iamtypes.Role
	policies []iamtypes.Policy
	policy   *iamtypes.Policy
	doc      string
	role     *iamtypes.Role
}

func (s *stubIAM) ListRoles(
	_ context.Context, _ *iam.ListRolesInput, _ ...func(*iam.Options),
) (*iam.ListRolesOutput, error) {
	return &iam.ListRolesOutput{Roles: s.roles}, nil
}

func (s *stubIAM) ListPolicies(
	_ context.Context, _ *iam.ListPoliciesInput, _ ...func(*iam.Options),
) (*iam.ListPoliciesOutput, error) {
	return &iam.ListPoliciesOutput{Policies: s.policies}, nil
}

func (s *stubIAM) GetPolicy(
	_ context.Context, _ *iam.GetPolicyInput, _ ...func(*iam.Options),
) (*iam.GetPolicyOutput, error) {
	return &iam.GetPolicyOutput{Policy: s.policy}, nil
}

func (s *stubIAM) GetPolicyVersion(
	_ context.Context, _ *iam.GetPolicyVersionInput, _ ...func(*iam.Options),
) (*iam.GetPolicyVersionOutput, error) {
	enc := url.QueryEscape(s.doc)

	return &iam.GetPolicyVersionOutput{
		PolicyVersion: &iamtypes.PolicyVersion{Document: &enc},
	}, nil
}

func (s *stubIAM) GetRole(
	_ context.Context, _ *iam.GetRoleInput, _ ...func(*iam.Options),
) (*iam.GetRoleOutput, error) {
	return &iam.GetRoleOutput{Role: s.role}, nil
}

func (*stubIAM) ListAttachedRolePolicies(
	_ context.Context, _ *iam.ListAttachedRolePoliciesInput, _ ...func(*iam.Options),
) (*iam.ListAttachedRolePoliciesOutput, error) {
	return &iam.ListAttachedRolePoliciesOutput{}, nil
}

func (*stubIAM) ListRolePolicies(
	_ context.Context, _ *iam.ListRolePoliciesInput, _ ...func(*iam.Options),
) (*iam.ListRolePoliciesOutput, error) {
	return &iam.ListRolePoliciesOutput{}, nil
}

func (*stubIAM) GetRolePolicy(
	_ context.Context, _ *iam.GetRolePolicyInput, _ ...func(*iam.Options),
) (*iam.GetRolePolicyOutput, error) {
	return &iam.GetRolePolicyOutput{}, nil
}

type stubSTS struct{}

func (stubSTS) GetCallerIdentity(
	_ context.Context, _ *sts.GetCallerIdentityInput, _ ...func(*sts.Options),
) (*sts.GetCallerIdentityOutput, error) {
	return &sts.GetCallerIdentityOutput{
		Account: aws.String("123"), Arn: aws.String("arn:aws:iam::123:user/u"), UserId: aws.String("AID"),
	}, nil
}

type stubSNS struct{}

func (stubSNS) ListTopics(
	_ context.Context, _ *sns.ListTopicsInput, _ ...func(*sns.Options),
) (*sns.ListTopicsOutput, error) {
	return &sns.ListTopicsOutput{
		Topics: []snstypes.Topic{{TopicArn: aws.String("arn:aws:sns:us-east-1:1:t")}},
	}, nil
}

type stubCF struct{}

func (stubCF) ListDistributions(
	_ context.Context, _ *cloudfront.ListDistributionsInput, _ ...func(*cloudfront.Options),
) (*cloudfront.ListDistributionsOutput, error) {
	return &cloudfront.ListDistributionsOutput{
		DistributionList: &cftypes.DistributionList{
			Items: []cftypes.DistributionSummary{{
				Id:         aws.String("E1"),
				DomainName: aws.String("d.cloudfront.net"),
				Aliases:    &cftypes.Aliases{Items: []string{"ex.com"}, Quantity: aws.Int32(1)},
				ViewerCertificate: &cftypes.ViewerCertificate{
					ACMCertificateArn:      aws.String("arn:aws:acm:us-east-1:1:certificate/x"),
					MinimumProtocolVersion: cftypes.MinimumProtocolVersionTLSv122021,
					SSLSupportMethod:       cftypes.SSLSupportMethodSniOnly,
				},
			}},
			IsTruncated: aws.Bool(false),
		},
	}, nil
}

func (stubCF) GetDistribution(
	_ context.Context, _ *cloudfront.GetDistributionInput, _ ...func(*cloudfront.Options),
) (*cloudfront.GetDistributionOutput, error) {
	return &cloudfront.GetDistributionOutput{}, nil
}

type stubACM struct{}

func (stubACM) DescribeCertificate(
	_ context.Context, _ *acm.DescribeCertificateInput, _ ...func(*acm.Options),
) (*acm.DescribeCertificateOutput, error) {
	exp := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	return &acm.DescribeCertificateOutput{
		Certificate: &acmtypes.CertificateDetail{
			DomainName: aws.String("ex.com"),
			NotAfter:   &exp,
		},
	}, nil
}

func callToolJSON(t *testing.T, session *mcp.ClientSession, name string, args any, dest any) {
	t.Helper()

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err)
	require.False(t, res.IsError, "%s: %v", name, res.Content)
	require.NotEmpty(t, res.Content)

	text, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	require.NoError(t, json.Unmarshal([]byte(text.Text), dest))
}

func TestHandlers_SuccessWithInjectedClients(t *testing.T) { //nolint:revive // exercises all MCP tool handlers
	t.Parallel()

	created := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	stub := &stubAWS{}
	stub.mockBucketsLister.fn = func(_ context.Context, _ *s3.ListBucketsInput, _ ...func(*s3.Options)) (
		*s3.ListBucketsOutput, error,
	) {
		return &s3.ListBucketsOutput{Buckets: []s3types.Bucket{{Name: aws.String("b1"), CreationDate: &created}}}, nil
	}
	stub.mockObjectsLister.fn = func(_ context.Context, _ *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (
		*s3.ListObjectsV2Output, error,
	) {
		return &s3.ListObjectsV2Output{
			Contents: []s3types.Object{{Key: aws.String("k"), Size: aws.Int64(1)}},
		}, nil
	}
	stub.mockObjectHeader.fn = func(_ context.Context, _ *s3.HeadObjectInput, _ ...func(*s3.Options)) (
		*s3.HeadObjectOutput, error,
	) {
		return &s3.HeadObjectOutput{ContentLength: aws.Int64(1), ETag: aws.String(`"e"`)}, nil
	}
	stub.mockSQSLister.fn = func(_ context.Context, _ *sqs.ListQueuesInput, _ ...func(*sqs.Options)) (
		*sqs.ListQueuesOutput, error,
	) {
		return &sqs.ListQueuesOutput{QueueUrls: []string{"https://q"}}, nil
	}
	stub.mockQueueURLGetter.fn = func(_ context.Context, _ *sqs.GetQueueUrlInput, _ ...func(*sqs.Options)) (
		*sqs.GetQueueUrlOutput, error,
	) {
		return &sqs.GetQueueUrlOutput{QueueUrl: aws.String("https://q")}, nil
	}
	stub.mockMessageReceiver.fn = func(_ context.Context, _ *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (
		*sqs.ReceiveMessageOutput, error,
	) {
		return &sqs.ReceiveMessageOutput{
			Messages: []sqstypes.Message{{MessageId: aws.String("m"), Body: aws.String("b")}},
		}, nil
	}
	stub.mockSecretsLister.fn = func(_ context.Context, _ *secretsmanager.ListSecretsInput,
		_ ...func(*secretsmanager.Options),
	) (*secretsmanager.ListSecretsOutput, error) {
		return &secretsmanager.ListSecretsOutput{
			SecretList: []smtypes.SecretListEntry{{Name: aws.String("s"), ARN: aws.String("arn:s")}},
		}, nil
	}
	stub.mockMSKClustersLister.fn = func(_ context.Context, _ *kafka.ListClustersV2Input, _ ...func(*kafka.Options)) (
		*kafka.ListClustersV2Output, error,
	) {
		return &kafka.ListClustersV2Output{
			ClusterInfoList: []kafkatypes.Cluster{{
				ClusterName: aws.String("c"), ClusterArn: aws.String("arn:c"), State: kafkatypes.ClusterStateActive,
			}},
		}, nil
	}
	stub.mockMSKTopicsLister.fn = func(_ context.Context, _ *kafka.ListTopicsInput, _ ...func(*kafka.Options)) (
		*kafka.ListTopicsOutput, error,
	) {
		return &kafka.ListTopicsOutput{
			Topics: []kafkatypes.TopicInfo{{TopicName: aws.String("t"), PartitionCount: aws.Int32(1)}},
		}, nil
	}

	iamStub := &stubIAM{
		roles: []iamtypes.Role{{RoleName: aws.String("r1"), Arn: aws.String("arn:r1"), Path: aws.String("/")}},
		policies: []iamtypes.Policy{{
			PolicyName:       aws.String("p1"),
			Arn:              aws.String("arn:aws:iam::1:policy/p1"),
			DefaultVersionId: aws.String("v1"),
		}},
		policy: &iamtypes.Policy{
			PolicyName:       aws.String("p1"),
			Arn:              aws.String("arn:aws:iam::1:policy/p1"),
			DefaultVersionId: aws.String("v1"),
		},
		doc: `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:GetObject","Resource":"*"}]}`,
		role: &iamtypes.Role{
			RoleName: aws.String("r1"), Arn: aws.String("arn:r1"), Path: aws.String("/"),
		},
	}

	deps := &Deps{
		LoadConfig: func(_ context.Context) (aws.Config, error) { return aws.Config{}, nil },
		S3:         func(aws.Config) s3API { return stub },
		SQS:        func(aws.Config) sqsAPI { return stub },
		Secrets:    func(aws.Config) secretsAPI { return stub },
		MSK:        func(aws.Config) mskAPI { return stub },
		IAM:        func(aws.Config) iamAPI { return iamStub },
		SNS:        func(aws.Config) internalsns.TopicsLister { return stubSNS{} },
		STS:        func(aws.Config) whoami.STSAPI { return stubSTS{} },
		CloudFront: func(aws.Config) (internalcf.ClientAPI, internalcf.ACMClientAPI) {
			return stubCF{}, stubACM{}
		},
	}

	session, _ := startTestSession(t, deps)

	var buckets listBucketsOutput
	callToolJSON(t, session, "aws_probe_s3_list_buckets", map[string]any{}, &buckets)
	require.Len(t, buckets.Buckets, 1)

	var objects listObjectsOutput
	callToolJSON(t, session, "aws_probe_s3_list_objects", map[string]any{"bucket": "b1"}, &objects)
	require.NotEmpty(t, objects.Objects)

	var meta objectMetadataOutput
	callToolJSON(t, session, "aws_probe_s3_get_object_metadata", map[string]any{"bucket": "b1", "key": "k"}, &meta)
	assert.Equal(t, "k", meta.Key)

	var queues listSQSQueuesOutput
	callToolJSON(t, session, "aws_probe_sqs_list_queues", map[string]any{}, &queues)
	require.Len(t, queues.QueueURLs, 1)

	var qURL getQueueURLOutput
	callToolJSON(t, session, "aws_probe_sqs_get_queue_url", map[string]any{"queueName": "q"}, &qURL)
	assert.Equal(t, "https://q", qURL.QueueURL)

	var msgs receiveMessageOutput
	callToolJSON(t, session, "aws_probe_sqs_receive_message", map[string]any{"queueUrl": "https://q"}, &msgs)
	require.Len(t, msgs.Messages, 1)

	var secretsOut listSecretsOutput
	callToolJSON(t, session, "aws_probe_secrets_list", map[string]any{}, &secretsOut)
	require.Len(t, secretsOut.Secrets, 1)

	var topics listTopicsOutput
	callToolJSON(t, session, "aws_probe_sns_list_topics", map[string]any{}, &topics)
	require.Len(t, topics.TopicARNs, 1)

	var clusters listMSKClustersOutput
	callToolJSON(t, session, "aws_probe_msk_list_clusters", map[string]any{}, &clusters)
	require.Len(t, clusters.Clusters, 1)

	var mskTopics listMSKTopicsOutput
	callToolJSON(t, session, "aws_probe_msk_list_topics", map[string]any{"clusterArn": "arn:c"}, &mskTopics)
	require.Len(t, mskTopics.Topics, 1)

	var who whoamiOutput
	callToolJSON(t, session, "aws_probe_whoami", map[string]any{}, &who)
	assert.Equal(t, "123", who.Account)

	var roles listRolesOutput
	callToolJSON(t, session, "aws_probe_iam_list_roles", map[string]any{}, &roles)
	require.Len(t, roles.Roles, 1)

	var policies listPoliciesOutput
	callToolJSON(t, session, "aws_probe_iam_list_policies", map[string]any{}, &policies)
	require.Len(t, policies.Policies, 1)

	var policyDump map[string]any
	callToolJSON(t, session, "aws_probe_iam_dump_policy", map[string]any{"nameOrArn": "p1"}, &policyDump)
	assert.NotEmpty(t, policyDump)

	var roleDump map[string]any
	callToolJSON(t, session, "aws_probe_iam_dump_role", map[string]any{"nameOrArn": "r1"}, &roleDump)
	assert.NotEmpty(t, roleDump)

	var certs listCloudFrontCertsOutput
	callToolJSON(t, session, "aws_probe_cloudfront_list_certificates", map[string]any{}, &certs)
	require.NotEmpty(t, certs.Certificates)
}
