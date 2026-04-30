package cmd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	kafkatypes "github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockS3ListBucketsClient struct {
	output *s3.ListBucketsOutput
	err    error
}

func (m *mockS3ListBucketsClient) ListBuckets(
	_ context.Context,
	_ *s3.ListBucketsInput,
	_ ...func(*s3.Options),
) (*s3.ListBucketsOutput, error) {
	return m.output, m.err
}

type mockS3ListObjectsClient struct {
	output *s3.ListObjectsV2Output
	err    error
}

func (m *mockS3ListObjectsClient) ListObjectsV2(
	_ context.Context,
	_ *s3.ListObjectsV2Input,
	_ ...func(*s3.Options),
) (*s3.ListObjectsV2Output, error) {
	return m.output, m.err
}

func TestListBuckets(t *testing.T) {
	createdAt := time.Date(2025, 3, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		client  s3Lister
		wantOut string
		wantErr bool
	}{
		{
			name: "lists buckets",
			client: &mockS3ListBucketsClient{
				output: &s3.ListBucketsOutput{
					Buckets: []s3types.Bucket{
						{Name: aws.String("my-bucket"), CreationDate: &createdAt},
						{Name: aws.String("logs-bucket"), CreationDate: &createdAt},
					},
				},
			},
			wantOut: "NAME         CREATED\n" +
				"my-bucket    2025-03-15 10:30:00\n" +
				"logs-bucket  2025-03-15 10:30:00\n",
		},
		{
			name: "bucket with nil creation date",
			client: &mockS3ListBucketsClient{
				output: &s3.ListBucketsOutput{
					Buckets: []s3types.Bucket{
						{Name: aws.String("my-bucket")},
					},
				},
			},
			wantOut: "NAME       CREATED\nmy-bucket  \n",
		},
		{
			name:    "empty list",
			client:  &mockS3ListBucketsClient{output: &s3.ListBucketsOutput{}},
			wantOut: "",
		},
		{
			name:    "API error",
			client:  &mockS3ListBucketsClient{err: errors.New("service unavailable")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := captureCmdOutput(t, func() error {
				return listBuckets(context.Background(), tt.client)
			})

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOut, out.stdout)
		})
	}
}

func TestStripPrefix(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		prefix string
		want   string
	}{
		{
			name:   "empty prefix returns key unchanged",
			key:    "folder/file.txt",
			prefix: "",
			want:   "folder/file.txt",
		},
		{
			name:   "strips matching prefix",
			key:    "k3s-argo/backups/",
			prefix: "k3s-argo/",
			want:   "backups/",
		},
		{
			name:   "prefix not found returns key unchanged",
			key:    "other/data.txt",
			prefix: "k3s-argo/",
			want:   "other/data.txt",
		},
		{
			name:   "key equals prefix returns empty string",
			key:    "k3s-argo/",
			prefix: "k3s-argo/",
			want:   "",
		},
		{
			name:   "nested prefix",
			key:    "a/b/c/d.txt",
			prefix: "a/b/",
			want:   "c/d.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripPrefix(tt.key, tt.prefix)
			assert.Equal(t, tt.want, got)
		})
	}
}

//nolint:revive // function length acceptable for test clarity
func TestListBucket(t *testing.T) {
	modifiedAt := time.Date(2025, 3, 15, 10, 30, 0, 0, time.UTC)

	t.Run("lists objects with delimiter", func(t *testing.T) {
		client := &mockS3ListObjectsClient{
			output: &s3.ListObjectsV2Output{
				CommonPrefixes: []s3types.CommonPrefix{
					{Prefix: aws.String("folder/")},
				},
				Contents: []s3types.Object{
					{Key: aws.String("README.md"), LastModified: &modifiedAt, Size: aws.Int64(1024)},
				},
			},
		}

		out, err := captureCmdOutput(t, func() error {
			return listBucket(context.Background(), "my-bucket", "", false, client)
		})

		require.NoError(t, err)
		assert.Contains(t, out.stdout, "KEY")
		assert.Contains(t, out.stdout, "folder/")
		assert.Contains(t, out.stdout, "README.md")
	})

	t.Run("lists objects recursively", func(t *testing.T) {
		client := &mockS3ListObjectsClient{
			output: &s3.ListObjectsV2Output{
				Contents: []s3types.Object{
					{Key: aws.String("folder/file.txt"), LastModified: &modifiedAt, Size: aws.Int64(512)},
					{Key: aws.String("data/backup.tar.gz"), LastModified: &modifiedAt, Size: aws.Int64(1048576)},
				},
			},
		}

		out, err := captureCmdOutput(t, func() error {
			return listBucket(context.Background(), "my-bucket", "", true, client)
		})

		require.NoError(t, err)
		assert.Contains(t, out.stdout, "folder/file.txt")
		assert.Contains(t, out.stdout, "data/backup.tar.gz")
	})

	t.Run("empty list", func(t *testing.T) {
		client := &mockS3ListObjectsClient{
			output: &s3.ListObjectsV2Output{},
		}

		_, err := captureCmdOutput(t, func() error {
			return listBucket(context.Background(), "my-bucket", "", false, client)
		})

		require.NoError(t, err)
	})

	t.Run("API error", func(t *testing.T) {
		client := &mockS3ListObjectsClient{err: errors.New("access denied")}

		err := listBucket(context.Background(), "my-bucket", "", false, client)

		assert.Error(t, err)
	})

	t.Run("strips prefix from CommonPrefixes", func(t *testing.T) {
		client := &mockS3ListObjectsClient{
			output: &s3.ListObjectsV2Output{
				CommonPrefixes: []s3types.CommonPrefix{
					{Prefix: aws.String("k3s-argo/backups/")},
					{Prefix: aws.String("k3s-argo/configs/")},
				},
			},
		}

		out, err := captureCmdOutput(t, func() error {
			return listBucket(context.Background(), "my-bucket", "k3s-argo/", false, client)
		})

		require.NoError(t, err)
		assert.Contains(t, out.stdout, "backups/")
		assert.Contains(t, out.stdout, "configs/")
		assert.NotContains(t, out.stdout, "k3s-argo/")
	})

	t.Run("strips prefix from Contents", func(t *testing.T) {
		client := &mockS3ListObjectsClient{
			output: &s3.ListObjectsV2Output{
				Contents: []s3types.Object{
					{Key: aws.String("k3s-argo/config.yaml"), LastModified: &modifiedAt, Size: aws.Int64(512)},
				},
			},
		}

		out, err := captureCmdOutput(t, func() error {
			return listBucket(context.Background(), "my-bucket", "k3s-argo/", false, client)
		})

		require.NoError(t, err)
		assert.Contains(t, out.stdout, "config.yaml")
		assert.NotContains(t, out.stdout, "k3s-argo/")
	})
}

type mockSQSListQueuesClient struct {
	output *sqs.ListQueuesOutput
	err    error
}

func (m *mockSQSListQueuesClient) ListQueues(
	_ context.Context,
	_ *sqs.ListQueuesInput,
	_ ...func(*sqs.Options),
) (*sqs.ListQueuesOutput, error) {
	return m.output, m.err
}

func TestListQueues(t *testing.T) {
	tests := []struct {
		name    string
		client  sqsLister
		wantOut string
		wantErr bool
	}{
		{
			name: "lists queues",
			client: &mockSQSListQueuesClient{
				output: &sqs.ListQueuesOutput{
					QueueUrls: []string{
						"https://sqs.us-east-1.amazonaws.com/123/queue-a",
						"https://sqs.us-east-1.amazonaws.com/123/queue-b",
					},
				},
			},
			wantOut: "QUEUE URL\n" +
				"https://sqs.us-east-1.amazonaws.com/123/queue-a\n" +
				"https://sqs.us-east-1.amazonaws.com/123/queue-b\n",
		},
		{
			name:    "empty list",
			client:  &mockSQSListQueuesClient{output: &sqs.ListQueuesOutput{}},
			wantOut: "",
		},
		{
			name:    "API error",
			client:  &mockSQSListQueuesClient{err: errors.New("throttled")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := captureCmdOutput(t, func() error {
				return listQueues(context.Background(), tt.client)
			})

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOut, out.stdout)
		})
	}
}

type mockSecretsListClient struct {
	output *secretsmanager.ListSecretsOutput
	err    error
}

func (m *mockSecretsListClient) ListSecrets(
	_ context.Context,
	_ *secretsmanager.ListSecretsInput,
	_ ...func(*secretsmanager.Options),
) (*secretsmanager.ListSecretsOutput, error) {
	return m.output, m.err
}

func TestListSecrets(t *testing.T) {
	tests := []struct {
		name    string
		client  secretsLister
		wantOut string
		wantErr bool
	}{
		{
			name: "lists secrets",
			client: &mockSecretsListClient{
				output: &secretsmanager.ListSecretsOutput{
					SecretList: []smtypes.SecretListEntry{
						{Name: aws.String("db-password"), ARN: aws.String("arn:aws:secretsmanager:us:123:secret:db")},
						{Name: aws.String("api-key"), ARN: aws.String("arn:aws:secretsmanager:us:123:secret:api")},
					},
				},
			},
			wantOut: "NAME         ARN\n" +
				"db-password  arn:aws:secretsmanager:us:123:secret:db\n" +
				"api-key      arn:aws:secretsmanager:us:123:secret:api\n",
		},
		{
			name:    "empty list",
			client:  &mockSecretsListClient{output: &secretsmanager.ListSecretsOutput{}},
			wantOut: "",
		},
		{
			name:    "API error",
			client:  &mockSecretsListClient{err: errors.New("access denied")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := captureCmdOutput(t, func() error {
				return listSecrets(context.Background(), tt.client)
			})

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOut, out.stdout)
		})
	}
}

type mockKafkaListClustersClient struct {
	output *kafka.ListClustersV2Output
	err    error
}

func (m *mockKafkaListClustersClient) ListClustersV2(
	_ context.Context,
	_ *kafka.ListClustersV2Input,
	_ ...func(*kafka.Options),
) (*kafka.ListClustersV2Output, error) {
	return m.output, m.err
}

func TestListClusters(t *testing.T) {
	tests := []struct {
		name    string
		client  kafkaClustersLister
		wantOut string
		wantErr bool
	}{
		{
			name: "lists clusters",
			client: &mockKafkaListClustersClient{
				output: &kafka.ListClustersV2Output{
					ClusterInfoList: []kafkatypes.Cluster{
						{
							ClusterName: aws.String("my-cluster"),
							ClusterArn:  aws.String("arn:aws:kafka:us:123:cluster/my/abc"),
							State:       kafkatypes.ClusterStateActive,
						},
					},
				},
			},
			wantOut: "CLUSTER NAME  ARN                                  STATUS\n" +
				"my-cluster    arn:aws:kafka:us:123:cluster/my/abc  ACTIVE\n",
		},
		{
			name:    "empty list",
			client:  &mockKafkaListClustersClient{output: &kafka.ListClustersV2Output{}},
			wantOut: "",
		},
		{
			name:    "API error",
			client:  &mockKafkaListClustersClient{err: errors.New("access denied")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := captureCmdOutput(t, func() error {
				return listClusters(context.Background(), tt.client)
			})

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOut, out.stdout)
		})
	}
}

type mockKafkaListTopicsClient struct {
	output *kafka.ListTopicsOutput
	err    error
}

func (m *mockKafkaListTopicsClient) ListTopics(
	_ context.Context,
	_ *kafka.ListTopicsInput,
	_ ...func(*kafka.Options),
) (*kafka.ListTopicsOutput, error) {
	return m.output, m.err
}

func TestListTopics(t *testing.T) {
	tests := []struct {
		name       string
		client     kafkaTopicsLister
		clusterArn string
		wantOut    string
		wantErr    bool
	}{
		{
			name:       "lists topics",
			clusterArn: "arn:aws:kafka:us:123:cluster/my/abc",
			client: &mockKafkaListTopicsClient{
				output: &kafka.ListTopicsOutput{
					Topics: []kafkatypes.TopicInfo{
						{
							TopicName:         aws.String("orders"),
							PartitionCount:    aws.Int32(6),
							ReplicationFactor: aws.Int32(3),
						},
					},
				},
			},
			wantOut: "TOPIC NAME  PARTITIONS  REPLICATION\n" +
				"orders      6           3\n",
		},
		{
			name:       "empty list",
			clusterArn: "arn:aws:kafka:us:123:cluster/my/abc",
			client:     &mockKafkaListTopicsClient{output: &kafka.ListTopicsOutput{}},
			wantOut:    "",
		},
		{
			name:       "API error",
			clusterArn: "arn:aws:kafka:us:123:cluster/my/abc",
			client:     &mockKafkaListTopicsClient{err: errors.New("access denied")},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := captureCmdOutput(t, func() error {
				return listTopics(context.Background(), tt.clusterArn, tt.client)
			})

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOut, out.stdout)
		})
	}
}

func TestS3Cmd_RunE_Error(t *testing.T) {
	oldPrepare := PrepareAWSConfig

	defer func() { PrepareAWSConfig = oldPrepare }()

	PrepareAWSConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, errors.New("load error")
	}

	t.Run("list-buckets", func(t *testing.T) {
		cmd := newListBucketsCmd()
		err := cmd.RunE(cmd, []string{})
		require.Error(t, err)
	})

	t.Run("list-bucket", func(t *testing.T) {
		cmd := newListBucketCmd()
		err := cmd.RunE(cmd, []string{"b"})
		require.Error(t, err)
	})
}
