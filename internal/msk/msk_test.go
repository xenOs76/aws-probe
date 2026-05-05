package msk

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	kafkatypes "github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/stretchr/testify/require"
)

type mockMskClient struct {
	ListClustersV2Func func(ctx context.Context, params *kafka.ListClustersV2Input, optFns ...func(*kafka.Options)) (
		*kafka.ListClustersV2Output, error)
	ListTopicsFunc func(ctx context.Context, params *kafka.ListTopicsInput, optFns ...func(*kafka.Options)) (
		*kafka.ListTopicsOutput, error)
}

func (m *mockMskClient) ListClustersV2(ctx context.Context, params *kafka.ListClustersV2Input,
	optFns ...func(*kafka.Options),
) (*kafka.ListClustersV2Output, error) {
	return m.ListClustersV2Func(ctx, params, optFns...)
}

func (m *mockMskClient) ListTopics(ctx context.Context, params *kafka.ListTopicsInput,
	optFns ...func(*kafka.Options),
) (*kafka.ListTopicsOutput, error) {
	return m.ListTopicsFunc(ctx, params, optFns...)
}

func TestListClusters(t *testing.T) {
	tests := []struct {
		name               string
		mockListClustersV2 func(
			ctx context.Context,
			params *kafka.ListClustersV2Input,
			optFns ...func(*kafka.Options),
		) (*kafka.ListClustersV2Output, error)
		wantOutput string
		wantErr    bool
	}{
		{
			name: "success",
			mockListClustersV2: func(_ context.Context, _ *kafka.ListClustersV2Input,
				_ ...func(*kafka.Options),
			) (*kafka.ListClustersV2Output, error) {
				return &kafka.ListClustersV2Output{
					ClusterInfoList: []kafkatypes.Cluster{
						{
							ClusterName: aws.String("cluster1"),
							ClusterArn:  aws.String("arn:cluster1"),
							State:       kafkatypes.ClusterStateActive,
						},
					},
				}, nil
			},
			wantOutput: "CLUSTER NAME  ARN           STATUS\ncluster1      arn:cluster1  ACTIVE\n",
			wantErr:    false,
		},
		{
			name: "no clusters",
			mockListClustersV2: func(_ context.Context, _ *kafka.ListClustersV2Input,
				_ ...func(*kafka.Options),
			) (*kafka.ListClustersV2Output, error) {
				return &kafka.ListClustersV2Output{}, nil
			},
			wantOutput: "No MSK clusters found.\n",
			wantErr:    false,
		},
		{
			name: "error",
			mockListClustersV2: func(_ context.Context, _ *kafka.ListClustersV2Input,
				_ ...func(*kafka.Options),
			) (*kafka.ListClustersV2Output, error) {
				return nil, errors.New("api error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &mockMskClient{ListClustersV2Func: tt.mockListClustersV2}

			var buf bytes.Buffer

			err := ListClusters(context.Background(), api, &buf)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Contains(t, buf.String(), tt.wantOutput)
		})
	}
}

func TestListTopics(t *testing.T) {
	tests := []struct {
		name           string
		mockListTopics func(ctx context.Context, params *kafka.ListTopicsInput, optFns ...func(*kafka.Options)) (
			*kafka.ListTopicsOutput, error)
		wantOutput string
		wantErr    bool
	}{
		{
			name: "success",
			mockListTopics: func(_ context.Context, _ *kafka.ListTopicsInput,
				_ ...func(*kafka.Options),
			) (*kafka.ListTopicsOutput, error) {
				return &kafka.ListTopicsOutput{
					Topics: []kafkatypes.TopicInfo{
						{
							TopicName:         aws.String("topic1"),
							PartitionCount:    aws.Int32(3),
							ReplicationFactor: aws.Int32(2),
						},
					},
				}, nil
			},
			wantOutput: "TOPIC NAME  PARTITIONS  REPLICATION\ntopic1      3           2\n",
			wantErr:    false,
		},
		{
			name: "no topics",
			mockListTopics: func(_ context.Context, _ *kafka.ListTopicsInput,
				_ ...func(*kafka.Options),
			) (*kafka.ListTopicsOutput, error) {
				return &kafka.ListTopicsOutput{}, nil
			},
			wantOutput: "No topics found.\n",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &mockMskClient{ListTopicsFunc: tt.mockListTopics}

			var buf bytes.Buffer

			err := ListTopics(context.Background(), "arn:cluster1", api, &buf)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Contains(t, buf.String(), tt.wantOutput)
		})
	}
}
