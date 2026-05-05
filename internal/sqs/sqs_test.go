package sqs

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/require"
)

type mockSqsLister struct {
	ListQueuesFunc func(ctx context.Context, params *sqs.ListQueuesInput, optFns ...func(*sqs.Options)) (
		*sqs.ListQueuesOutput, error)
}

func (m *mockSqsLister) ListQueues(ctx context.Context, params *sqs.ListQueuesInput,
	optFns ...func(*sqs.Options),
) (*sqs.ListQueuesOutput, error) {
	return m.ListQueuesFunc(ctx, params, optFns...)
}

func TestListQueues_Success(t *testing.T) {
	tests := []struct {
		name           string
		mockListQueues func(ctx context.Context, params *sqs.ListQueuesInput, optFns ...func(*sqs.Options)) (
			*sqs.ListQueuesOutput, error)
		wantOutput string
	}{
		{
			name: "success with multiple queues",
			mockListQueues: func(_ context.Context, _ *sqs.ListQueuesInput,
				_ ...func(*sqs.Options),
			) (*sqs.ListQueuesOutput, error) {
				return &sqs.ListQueuesOutput{
					QueueUrls: []string{
						"https://sqs.us-east-1.amazonaws.com/123456789012/queue1",
						"https://sqs.us-east-1.amazonaws.com/123456789012/queue2",
					},
				}, nil
			},
			wantOutput: "QUEUE URL\n" +
				"https://sqs.us-east-1.amazonaws.com/123456789012/queue1\n" +
				"https://sqs.us-east-1.amazonaws.com/123456789012/queue2\n",
		},
		{
			name: "success with pagination",
			mockListQueues: func(_ context.Context, params *sqs.ListQueuesInput,
				_ ...func(*sqs.Options),
			) (*sqs.ListQueuesOutput, error) {
				if params.NextToken == nil {
					return &sqs.ListQueuesOutput{
						QueueUrls: []string{"https://queue1"},
						NextToken: aws.String("token"),
					}, nil
				}

				return &sqs.ListQueuesOutput{
					QueueUrls: []string{"https://queue2"},
				}, nil
			},
			wantOutput: "QUEUE URL\nhttps://queue1\nhttps://queue2\n",
		},
		{
			name: "no queues found",
			mockListQueues: func(_ context.Context, _ *sqs.ListQueuesInput,
				_ ...func(*sqs.Options),
			) (*sqs.ListQueuesOutput, error) {
				return &sqs.ListQueuesOutput{}, nil
			},
			wantOutput: "No SQS queues found.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &mockSqsLister{ListQueuesFunc: tt.mockListQueues}

			var buf bytes.Buffer

			err := ListQueues(context.Background(), api, &buf)
			require.NoError(t, err)
			require.Equal(t, tt.wantOutput, buf.String())
		})
	}
}

func TestListQueues_Error(t *testing.T) {
	api := &mockSqsLister{
		ListQueuesFunc: func(_ context.Context, _ *sqs.ListQueuesInput,
			_ ...func(*sqs.Options),
		) (*sqs.ListQueuesOutput, error) {
			return nil, errors.New("api error")
		},
	}

	var buf bytes.Buffer

	err := ListQueues(context.Background(), api, &buf)
	require.Error(t, err)
	require.Contains(t, err.Error(), "api error")
}
