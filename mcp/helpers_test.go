package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	kafkatypes "github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	internalkafka "github.com/xenos76/aws-probe/internal/kafka"
)

type mockBucketsLister struct {
	fn func(context.Context, *s3.ListBucketsInput, ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
}

func (m *mockBucketsLister) ListBuckets(
	ctx context.Context, in *s3.ListBucketsInput, opts ...func(*s3.Options),
) (*s3.ListBucketsOutput, error) {
	return m.fn(ctx, in, opts...)
}

type mockObjectsLister struct {
	fn func(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

func (m *mockObjectsLister) ListObjectsV2(
	ctx context.Context, in *s3.ListObjectsV2Input, opts ...func(*s3.Options),
) (*s3.ListObjectsV2Output, error) {
	return m.fn(ctx, in, opts...)
}

type mockObjectHeader struct {
	fn func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

func (m *mockObjectHeader) HeadObject(
	ctx context.Context, in *s3.HeadObjectInput, opts ...func(*s3.Options),
) (*s3.HeadObjectOutput, error) {
	return m.fn(ctx, in, opts...)
}

type mockSQSLister struct {
	fn func(context.Context, *sqs.ListQueuesInput, ...func(*sqs.Options)) (*sqs.ListQueuesOutput, error)
}

func (m *mockSQSLister) ListQueues(
	ctx context.Context, in *sqs.ListQueuesInput, opts ...func(*sqs.Options),
) (*sqs.ListQueuesOutput, error) {
	return m.fn(ctx, in, opts...)
}

type mockQueueURLGetter struct {
	fn func(context.Context, *sqs.GetQueueUrlInput, ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error)
}

//nolint:revive // GetQueueUrl matches the AWS SDK method name.
func (m *mockQueueURLGetter) GetQueueUrl(
	ctx context.Context, in *sqs.GetQueueUrlInput, opts ...func(*sqs.Options),
) (*sqs.GetQueueUrlOutput, error) {
	return m.fn(ctx, in, opts...)
}

type mockMessageReceiver struct {
	fn func(context.Context, *sqs.ReceiveMessageInput, ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
}

func (m *mockMessageReceiver) ReceiveMessage(
	ctx context.Context, in *sqs.ReceiveMessageInput, opts ...func(*sqs.Options),
) (*sqs.ReceiveMessageOutput, error) {
	return m.fn(ctx, in, opts...)
}

type mockSecretsLister struct {
	fn func(context.Context, *secretsmanager.ListSecretsInput, ...func(*secretsmanager.Options)) (
		*secretsmanager.ListSecretsOutput, error)
}

func (m *mockSecretsLister) ListSecrets(
	ctx context.Context, in *secretsmanager.ListSecretsInput, opts ...func(*secretsmanager.Options),
) (*secretsmanager.ListSecretsOutput, error) {
	return m.fn(ctx, in, opts...)
}

type mockMSKClustersLister struct {
	fn func(context.Context, *kafka.ListClustersV2Input, ...func(*kafka.Options)) (*kafka.ListClustersV2Output, error)
}

func (m *mockMSKClustersLister) ListClustersV2(
	ctx context.Context, in *kafka.ListClustersV2Input, opts ...func(*kafka.Options),
) (*kafka.ListClustersV2Output, error) {
	return m.fn(ctx, in, opts...)
}

type mockMSKTopicsLister struct {
	fn func(context.Context, *kafka.ListTopicsInput, ...func(*kafka.Options)) (*kafka.ListTopicsOutput, error)
}

func (m *mockMSKTopicsLister) ListTopics(
	ctx context.Context, in *kafka.ListTopicsInput, opts ...func(*kafka.Options),
) (*kafka.ListTopicsOutput, error) {
	return m.fn(ctx, in, opts...)
}

func TestListS3Buckets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	created := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	entries, err := listS3Buckets(ctx, &mockBucketsLister{
		fn: func(_ context.Context, _ *s3.ListBucketsInput, _ ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
			return &s3.ListBucketsOutput{
				Buckets: []s3types.Bucket{{Name: aws.String("b1"), CreationDate: &created}},
			}, nil
		},
	})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "b1", entries[0].Name)
	assert.Equal(t, "2024-01-01T00:00:00Z", entries[0].Created)

	_, err = listS3Buckets(ctx, &mockBucketsLister{
		fn: func(_ context.Context, _ *s3.ListBucketsInput, _ ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
			return nil, errors.New("boom")
		},
	})
	require.Error(t, err)
}

func TestCollectS3Objects(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	modified := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	delim := "/"

	objects, err := collectS3Objects(ctx, &mockObjectsLister{
		fn: func(
			_ context.Context, in *s3.ListObjectsV2Input, _ ...func(*s3.Options),
		) (*s3.ListObjectsV2Output, error) {
			assert.Equal(t, "bucket", aws.ToString(in.Bucket))
			assert.Equal(t, "pre/", aws.ToString(in.Prefix))
			assert.Equal(t, &delim, in.Delimiter)

			return &s3.ListObjectsV2Output{
				CommonPrefixes: []s3types.CommonPrefix{{Prefix: aws.String("pre/folder/")}},
				Contents: []s3types.Object{{
					Key: aws.String("pre/file.txt"), Size: aws.Int64(100), LastModified: &modified,
				}},
			}, nil
		},
	}, "bucket", "pre/", &delim)
	require.NoError(t, err)
	require.Len(t, objects, 2)
	assert.True(t, objects[0].IsPrefix)
	assert.Equal(t, "folder/", objects[0].Key)
	assert.Equal(t, "file.txt", objects[1].Key)

	_, err = collectS3Objects(ctx, &mockObjectsLister{
		fn: func(_ context.Context, _ *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
			return nil, errors.New("list failed")
		},
	}, "bucket", "", nil)
	require.Error(t, err)
}

func TestHeadS3Object(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	modified := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)

	out, err := headS3Object(ctx, &mockObjectHeader{
		fn: func(_ context.Context, in *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
			assert.Equal(t, "b", aws.ToString(in.Bucket))
			assert.Equal(t, "k", aws.ToString(in.Key))

			return &s3.HeadObjectOutput{
				ContentLength:        aws.Int64(42),
				ETag:                 aws.String(`"abc"`),
				ContentType:          aws.String("text/plain"),
				StorageClass:         s3types.StorageClassStandard,
				LastModified:         &modified,
				ServerSideEncryption: s3types.ServerSideEncryptionAes256,
				VersionId:            aws.String("v1"),
				Metadata:             map[string]string{"owner": "x"},
			}, nil
		},
	}, "b", "k")
	require.NoError(t, err)
	assert.Equal(t, "k", out.Key)
	assert.Equal(t, "abc", out.ETag)
	assert.Equal(t, "AES256", out.ServerEncryption)
	assert.Equal(t, "v1", out.VersionID)
	assert.Equal(t, map[string]string{"owner": "x"}, out.CustomMetadata)

	_, err = headS3Object(ctx, &mockObjectHeader{
		fn: func(_ context.Context, _ *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
			return nil, errors.New("missing")
		},
	}, "b", "k")
	require.Error(t, err)
}

func TestSQSHelpers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	urls, err := listSQSQueueURLs(ctx, &mockSQSLister{
		fn: func(_ context.Context, in *sqs.ListQueuesInput, _ ...func(*sqs.Options)) (*sqs.ListQueuesOutput, error) {
			if in.NextToken == nil {
				return &sqs.ListQueuesOutput{
					QueueUrls: []string{"https://q1"},
					NextToken: aws.String("n"),
				}, nil
			}

			return &sqs.ListQueuesOutput{QueueUrls: []string{"https://q2"}}, nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"https://q1", "https://q2"}, urls)

	_, err = listSQSQueueURLs(ctx, &mockSQSLister{
		fn: func(_ context.Context, _ *sqs.ListQueuesInput, _ ...func(*sqs.Options)) (*sqs.ListQueuesOutput, error) {
			return nil, errors.New("fail")
		},
	})
	require.Error(t, err)

	url, err := getSQSQueueURL(ctx, &mockQueueURLGetter{
		fn: func(_ context.Context, in *sqs.GetQueueUrlInput, _ ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
			assert.Equal(t, "my-q", aws.ToString(in.QueueName))

			return &sqs.GetQueueUrlOutput{QueueUrl: aws.String("https://my-q")}, nil
		},
	}, "my-q")
	require.NoError(t, err)
	assert.Equal(t, "https://my-q", url)

	_, err = getSQSQueueURL(ctx, &mockQueueURLGetter{
		fn: func(_ context.Context, _ *sqs.GetQueueUrlInput, _ ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
			return nil, errors.New("nope")
		},
	}, "my-q")
	require.Error(t, err)

	msgs, err := receiveSQSMessages(ctx, &mockMessageReceiver{
		fn: func(
			_ context.Context, in *sqs.ReceiveMessageInput, _ ...func(*sqs.Options),
		) (*sqs.ReceiveMessageOutput, error) {
			assert.Equal(t, int32(2), in.MaxNumberOfMessages)

			return &sqs.ReceiveMessageOutput{
				Messages: []sqstypes.Message{{
					MessageId: aws.String("m1"), Body: aws.String("hi"), ReceiptHandle: aws.String("rh"),
				}},
			}, nil
		},
	}, "https://q", 2)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "m1", msgs[0].MessageID)

	_, err = receiveSQSMessages(ctx, &mockMessageReceiver{
		fn: func(
			_ context.Context, _ *sqs.ReceiveMessageInput, _ ...func(*sqs.Options),
		) (*sqs.ReceiveMessageOutput, error) {
			return nil, errors.New("recv fail")
		},
	}, "https://q", 1)
	require.Error(t, err)
}

func TestClampSQSMaxMessages(t *testing.T) {
	t.Parallel()
	assert.Equal(t, int32(1), clampSQSMaxMessages(0))
	assert.Equal(t, int32(1), clampSQSMaxMessages(-1))
	assert.Equal(t, int32(5), clampSQSMaxMessages(5))
	assert.Equal(t, int32(10), clampSQSMaxMessages(99))
}

func TestListSecretsForMCP(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	secrets, err := listSecretsForMCP(ctx, &mockSecretsLister{
		fn: func(_ context.Context, in *secretsmanager.ListSecretsInput, _ ...func(*secretsmanager.Options)) (
			*secretsmanager.ListSecretsOutput, error,
		) {
			if in.NextToken == nil {
				return &secretsmanager.ListSecretsOutput{
					SecretList: []smtypes.SecretListEntry{
						{Name: aws.String("a"), ARN: aws.String("arn:a")},
						{Name: nil, ARN: aws.String("arn:skip")},
					},
					NextToken: aws.String("n"),
				}, nil
			}

			return &secretsmanager.ListSecretsOutput{
				SecretList: []smtypes.SecretListEntry{{Name: aws.String("b"), ARN: aws.String("arn:b")}},
			}, nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []secretEntry{{Name: "a", ARN: "arn:a"}, {Name: "b", ARN: "arn:b"}}, secrets)

	_, err = listSecretsForMCP(ctx, &mockSecretsLister{
		fn: func(_ context.Context, _ *secretsmanager.ListSecretsInput, _ ...func(*secretsmanager.Options)) (
			*secretsmanager.ListSecretsOutput, error,
		) {
			return nil, errors.New("secrets fail")
		},
	})
	require.Error(t, err)
}

func TestMSKListHelpers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	clusters, err := listMSKClustersForMCP(ctx, &mockMSKClustersLister{
		fn: func(_ context.Context, _ *kafka.ListClustersV2Input, _ ...func(*kafka.Options)) (
			*kafka.ListClustersV2Output, error,
		) {
			return &kafka.ListClustersV2Output{
				ClusterInfoList: []kafkatypes.Cluster{
					{
						ClusterName: aws.String("c1"),
						ClusterArn:  aws.String("arn:c1"),
						State:       kafkatypes.ClusterStateActive,
					},
				},
			}, nil
		},
	})
	require.NoError(t, err)
	require.Len(t, clusters, 1)
	assert.Equal(t, "c1", clusters[0].Name)
	assert.Equal(t, "ACTIVE", clusters[0].Status)

	_, err = listMSKClustersForMCP(ctx, &mockMSKClustersLister{
		fn: func(_ context.Context, _ *kafka.ListClustersV2Input, _ ...func(*kafka.Options)) (
			*kafka.ListClustersV2Output, error,
		) {
			return nil, errors.New("msk fail")
		},
	})
	require.Error(t, err)

	topics, err := listMSKTopicsForMCP(ctx, &mockMSKTopicsLister{
		fn: func(_ context.Context, in *kafka.ListTopicsInput, _ ...func(*kafka.Options)) (
			*kafka.ListTopicsOutput, error,
		) {
			assert.Equal(t, "arn:cluster", aws.ToString(in.ClusterArn))

			return &kafka.ListTopicsOutput{
				Topics: []kafkatypes.TopicInfo{{
					TopicName: aws.String("t1"), PartitionCount: aws.Int32(3), ReplicationFactor: aws.Int32(2),
				}},
			}, nil
		},
	}, "arn:cluster")
	require.NoError(t, err)
	require.Len(t, topics, 1)
	assert.Equal(t, "t1", topics[0].Name)

	_, err = listMSKTopicsForMCP(ctx, &mockMSKTopicsLister{
		fn: func(_ context.Context, _ *kafka.ListTopicsInput, _ ...func(*kafka.Options)) (
			*kafka.ListTopicsOutput, error,
		) {
			return nil, errors.New("topics fail")
		},
	}, "arn:cluster")
	require.Error(t, err)
}

func TestParseMSKConsumeInput(t *testing.T) {
	t.Parallel()

	_, err := parseMSKConsumeInput(mskConsumeInput{})
	require.ErrorIs(t, err, errMSKTopicRequired)

	_, err = parseMSKConsumeInput(mskConsumeInput{Topic: "t", Auth: "bad"})
	require.Error(t, err)

	tlsFalse := false
	opts, err := parseMSKConsumeInput(mskConsumeInput{
		Topic: "t", Auth: "none", TLS: &tlsFalse, TimeoutSec: 5, MaxRecords: 3,
	})
	require.NoError(t, err)
	assert.Equal(t, "none", opts.auth)
	assert.False(t, opts.useTLS)
	assert.Equal(t, 5*time.Second, opts.timeout)
	assert.Equal(t, 3, opts.maxRecords)

	opts, err = parseMSKConsumeInput(mskConsumeInput{Topic: "t", Auth: "iam", TLS: &tlsFalse})
	require.NoError(t, err)
	assert.True(t, opts.useTLS) // IAM forces TLS

	opts, err = parseMSKConsumeInput(mskConsumeInput{Topic: "t"})
	require.NoError(t, err)
	assert.Equal(t, "iam", opts.auth)
	assert.Equal(t, defaultConsumeTimeout, opts.timeout)
	assert.Equal(t, defaultConsumeMaxRecords, opts.maxRecords)
}

func TestLoadAWSAndDeps(t *testing.T) {
	t.Parallel()

	cfg, err := loadAWS(context.Background(), &Deps{
		LoadConfig: func(_ context.Context) (aws.Config, error) {
			return aws.Config{Region: "eu-west-1"}, nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "eu-west-1", cfg.Region)

	_, err = loadAWS(context.Background(), loadConfigErr(errors.New("no creds")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading AWS config")

	d := (*Deps)(nil).withDefaults()
	require.NotNil(t, d.LoadConfig)

	d2 := (&Deps{Config: Config{Region: "us-east-1"}}).withDefaults()
	assert.Equal(t, "us-east-1", d2.Config.Region)
	require.NotNil(t, d2.LoadConfig)

	require.NotNil(t, DefaultLoadConfig())
}

func TestConsumeBoundedWith(t *testing.T) {
	t.Parallel()

	t.Run("collects up to max", func(t *testing.T) {
		svc := &stubKafkaConsumer{records: []*internalkafka.Record{
			{Topic: "t", Partition: 0, Offset: 1, Key: []byte("k"), Value: []byte("v1")},
			{Topic: "t", Partition: 0, Offset: 2, Key: []byte("k"), Value: []byte("v2")},
			{Topic: "t", Partition: 0, Offset: 3, Key: []byte("k"), Value: []byte("v3")},
		}}
		got, err := consumeBoundedWith(context.Background(), svc, internalkafka.Config{Topic: "t"}, 2)
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "v1", got[0].Value)
		assert.Equal(t, "v2", got[1].Value)
	})

	t.Run("consume error", func(t *testing.T) {
		svc := &stubKafkaConsumer{err: errors.New("broker down")}
		_, err := consumeBoundedWith(context.Background(), svc, internalkafka.Config{Topic: "t"}, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "broker down")
	})

	t.Run("context canceled with no records", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		svc := &stubKafkaConsumer{err: context.Canceled}
		_, err := consumeBoundedWith(ctx, svc, internalkafka.Config{Topic: "t"}, 1)
		require.Error(t, err)
	})
}

type stubKafkaConsumer struct {
	records []*internalkafka.Record
	err     error
}

func (s *stubKafkaConsumer) Consume(
	ctx context.Context, _ internalkafka.Config, handler func(*internalkafka.Record),
) error {
	for _, r := range s.records {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		handler(r)
	}

	return s.err
}

func TestDepsDefaultFactories(t *testing.T) {
	t.Parallel()

	d := &Deps{}
	cfg := aws.Config{Region: "us-east-1"}
	require.NotNil(t, d.s3(cfg))
	require.NotNil(t, d.sqs(cfg))
	require.NotNil(t, d.secrets(cfg))
	require.NotNil(t, d.msk(cfg))
	require.NotNil(t, d.iam(cfg))
	require.NotNil(t, d.sns(cfg))
	require.NotNil(t, d.sts(cfg))
	cf, acm := d.cloudFront(cfg)
	require.NotNil(t, cf)
	require.NotNil(t, acm)

	fn := DefaultLoadConfig()
	require.NotNil(t, fn)
	// May succeed or fail depending on ambient AWS credentials; just exercise the path.
	_, _ = fn(context.Background())
}

func TestRun_CanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Run(ctx, "dev", &Deps{
		LoadConfig: func(_ context.Context) (aws.Config, error) {
			return aws.Config{}, nil
		},
	})
	require.Error(t, err)
}

func TestMSKConsumeHandler_ResolveAndConsumeError(t *testing.T) {
	t.Parallel()

	stub := &stubAWS{}
	deps := &Deps{
		LoadConfig: func(_ context.Context) (aws.Config, error) { return aws.Config{}, nil },
		MSK:        func(aws.Config) mskAPI { return stub },
	}
	session, _ := startTestSession(t, deps)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "aws_probe_msk_consume",
		Arguments: map[string]any{
			"topic": "t", "brokers": "127.0.0.1:1", "auth": "none", "tls": false,
			"timeoutSec": 1, "maxRecords": 1,
		},
	})
	require.NoError(t, err)
	require.True(t, res.IsError)
}
