package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSecretsClient struct {
	output *secretsmanager.ListSecretsOutput
	err    error
}

func (m *mockSecretsClient) ListSecrets(
	_ context.Context,
	_ *secretsmanager.ListSecretsInput,
	_ ...func(*secretsmanager.Options),
) (*secretsmanager.ListSecretsOutput, error) {
	return m.output, m.err
}

type mockSQSClient struct {
	output *sqs.ListQueuesOutput
	err    error
}

func (m *mockSQSClient) ListQueues(
	_ context.Context,
	_ *sqs.ListQueuesInput,
	_ ...func(*sqs.Options),
) (*sqs.ListQueuesOutput, error) {
	return m.output, m.err
}

type mockS3Client struct {
	output *s3.ListBucketsOutput
	err    error
}

func (m *mockS3Client) ListBuckets(
	_ context.Context,
	_ *s3.ListBucketsInput,
	_ ...func(*s3.Options),
) (*s3.ListBucketsOutput, error) {
	return m.output, m.err
}

type outputCapture struct {
	stdout string
	stderr string
}

func captureOutput(t *testing.T, fn func() error) (outputCapture, error) {
	t.Helper()

	oldStdout, oldStderr := os.Stdout, os.Stderr

	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)

	stderrR, stderrW, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = stdoutW
	os.Stderr = stderrW

	t.Cleanup(func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	})

	fnErr := fn()

	stdoutW.Close()
	stderrW.Close()

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var stdoutBuf, stderrBuf bytes.Buffer

	_, err = io.Copy(&stdoutBuf, stdoutR)
	require.NoError(t, err)

	_, err = io.Copy(&stderrBuf, stderrR)
	require.NoError(t, err)

	return outputCapture{stdout: stdoutBuf.String(), stderr: stderrBuf.String()}, fnErr
}

func TestListSecrets(t *testing.T) {
	tests := []struct {
		name       string
		client     secretsListAPI
		wantOut    string
		wantStderr string
		wantErr    bool
	}{
		{
			name: "lists secrets",
			client: &mockSecretsClient{
				output: &secretsmanager.ListSecretsOutput{
					SecretList: []smtypes.SecretListEntry{
						{
							Name: aws.String("db-password"),
							ARN:  aws.String("arn:aws:secretsmanager:us-east-1:123:secret:db"),
						},
						{
							Name: aws.String("api-key"),
							ARN:  aws.String("arn:aws:secretsmanager:us-east-1:123:secret:api"),
						},
					},
				},
			},
			wantOut: "NAME         ARN\n" +
				"db-password  arn:aws:secretsmanager:us-east-1:123:secret:db\n" +
				"api-key      arn:aws:secretsmanager:us-east-1:123:secret:api\n",
		},
		{
			name: "empty list",
			client: &mockSecretsClient{
				output: &secretsmanager.ListSecretsOutput{},
			},
			wantStderr: "No secrets found.\n",
		},
		{
			name: "API error",
			client: &mockSecretsClient{
				err: errors.New("access denied"),
			},
			wantErr: true,
		},
		{
			name: "credential error",
			client: &mockSecretsClient{
				err: errors.New("failed to refresh cached credentials"),
			},
			wantStderr: noCredentialsMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := captureOutput(t, func() error {
				return listSecrets(context.Background(), tt.client)
			})

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOut, out.stdout)
			assert.Equal(t, tt.wantStderr, out.stderr)
		})
	}
}

func TestListQueues(t *testing.T) {
	tests := []struct {
		name       string
		client     sqsListAPI
		wantOut    string
		wantStderr string
		wantErr    bool
	}{
		{
			name: "lists queues",
			client: &mockSQSClient{
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
			name: "empty list",
			client: &mockSQSClient{
				output: &sqs.ListQueuesOutput{},
			},
			wantStderr: "No SQS queues found.\n",
		},
		{
			name: "API error",
			client: &mockSQSClient{
				err: errors.New("throttled"),
			},
			wantErr: true,
		},
		{
			name: "credential error",
			client: &mockSQSClient{
				err: errors.New("no credential providers"),
			},
			wantStderr: noCredentialsMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := captureOutput(t, func() error {
				return listQueues(context.Background(), tt.client)
			})

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOut, out.stdout)
			assert.Equal(t, tt.wantStderr, out.stderr)
		})
	}
}

func TestListBuckets(t *testing.T) {
	createdAt := time.Date(2025, 3, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name       string
		client     s3ListAPI
		wantOut    string
		wantStderr string
		wantErr    bool
	}{
		{
			name: "lists buckets",
			client: &mockS3Client{
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
			client: &mockS3Client{
				output: &s3.ListBucketsOutput{
					Buckets: []s3types.Bucket{
						{Name: aws.String("my-bucket")},
					},
				},
			},
			wantOut: "NAME       CREATED\nmy-bucket  \n",
		},
		{
			name: "empty list",
			client: &mockS3Client{
				output: &s3.ListBucketsOutput{},
			},
			wantStderr: "No S3 buckets found.\n",
		},
		{
			name: "API error",
			client: &mockS3Client{
				err: errors.New("service unavailable"),
			},
			wantErr: true,
		},
		{
			name: "credential error",
			client: &mockS3Client{
				err: errors.New("AnonymousCredentials"),
			},
			wantStderr: noCredentialsMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := captureOutput(t, func() error {
				return listBuckets(context.Background(), tt.client)
			})

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOut, out.stdout)
			assert.Equal(t, tt.wantStderr, out.stderr)
		})
	}
}

func TestPrintAvailableResources(t *testing.T) {
	out, err := captureOutput(t, func() error {
		printAvailableResources()

		return nil
	})

	require.NoError(t, err)
	assert.Empty(t, out.stdout)
	assert.Contains(t, out.stderr, "secrets")
	assert.Contains(t, out.stderr, "sqs")
	assert.Contains(t, out.stderr, "s3")
	assert.Contains(t, out.stderr, "msk-clusters")
	assert.Contains(t, out.stderr, "msk-topics")
	assert.Contains(t, out.stderr, "Usage:")
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

func TestListMSKClusters(t *testing.T) {
	tests := []struct {
		name       string
		client     kafkaListClustersAPI
		wantOut    string
		wantStderr string
		wantErr    bool
	}{
		{
			name: "lists clusters",
			client: &mockKafkaListClustersClient{
				output: &kafka.ListClustersV2Output{
					ClusterInfoList: []kafkatypes.Cluster{
						{
							ClusterName: aws.String("my-cluster"),
							ClusterArn:  aws.String("arn:aws:kafka:us-east-1:123:cluster/my-cluster/abc"),
							State:       kafkatypes.ClusterStateActive,
						},
						{
							ClusterName: aws.String("prod-cluster"),
							ClusterArn:  aws.String("arn:aws:kafka:us-east-1:123:cluster/prod-cluster/xyz"),
							State:       kafkatypes.ClusterStateActive,
						},
					},
				},
			},
			wantOut: "CLUSTER NAME  ARN                                                   STATUS\n" +
				"my-cluster    arn:aws:kafka:us-east-1:123:cluster/my-cluster/abc    ACTIVE\n" +
				"prod-cluster  arn:aws:kafka:us-east-1:123:cluster/prod-cluster/xyz  ACTIVE\n",
		},
		{
			name: "empty list",
			client: &mockKafkaListClustersClient{
				output: &kafka.ListClustersV2Output{},
			},
			wantStderr: "No MSK clusters found.\n",
		},
		{
			name: "API error",
			client: &mockKafkaListClustersClient{
				err: errors.New("access denied"),
			},
			wantErr: true,
		},
		{
			name: "credential error",
			client: &mockKafkaListClustersClient{
				err: errors.New("no credential providers"),
			},
			wantStderr: noCredentialsMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := captureOutput(t, func() error {
				return listMSKClusters(context.Background(), tt.client)
			})

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOut, out.stdout)
			assert.Equal(t, tt.wantStderr, out.stderr)
		})
	}
}

func TestListMSKTopics(t *testing.T) {
	tests := []struct {
		name       string
		client     kafkaListTopicsAPI
		clusterArn string
		wantOut    string
		wantStderr string
		wantErr    bool
	}{
		{
			name:       "lists topics",
			clusterArn: "arn:aws:kafka:us-east-1:123:cluster/my-cluster/abc",
			client: &mockKafkaListTopicsClient{
				output: &kafka.ListTopicsOutput{
					Topics: []kafkatypes.TopicInfo{
						{
							TopicName:         aws.String("orders"),
							TopicArn:          aws.String("arn:aws:kafka:us-east-1:123:topic/my-cluster/orders"),
							PartitionCount:    aws.Int32(6),
							ReplicationFactor: aws.Int32(3),
						},
						{
							TopicName:         aws.String("users"),
							TopicArn:          aws.String("arn:aws:kafka:us-east-1:123:topic/my-cluster/users"),
							PartitionCount:    aws.Int32(12),
							ReplicationFactor: aws.Int32(3),
						},
					},
				},
			},
			wantOut: "TOPIC NAME  PARTITIONS  REPLICATION\n" +
				"orders      6           3\n" +
				"users       12          3\n",
		},
		{
			name:       "empty list",
			clusterArn: "arn:aws:kafka:us-east-1:123:cluster/my-cluster/abc",
			client: &mockKafkaListTopicsClient{
				output: &kafka.ListTopicsOutput{},
			},
			wantStderr: "No topics found.\n",
		},
		{
			name:       "API error",
			clusterArn: "arn:aws:kafka:us-east-1:123:cluster/my-cluster/abc",
			client: &mockKafkaListTopicsClient{
				err: errors.New("access denied"),
			},
			wantErr: true,
		},
		{
			name:       "credential error",
			clusterArn: "arn:aws:kafka:us-east-1:123:cluster/my-cluster/abc",
			client: &mockKafkaListTopicsClient{
				err: errors.New("failed to refresh cached credentials"),
			},
			wantStderr: noCredentialsMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := captureOutput(t, func() error {
				return listMSKTopics(context.Background(), tt.clusterArn, tt.client)
			})

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOut, out.stdout)
			assert.Equal(t, tt.wantStderr, out.stderr)
		})
	}
}
