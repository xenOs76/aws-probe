package sqs

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/require"
)

type mockSqsLister struct {
	ListQueuesFunc func(ctx context.Context, params *sqs.ListQueuesInput, optFns ...func(*sqs.Options)) (
		*sqs.ListQueuesOutput, error)
}

type mockQueueURLGetter struct {
	GetQueueURLFunc func(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (
		*sqs.GetQueueUrlOutput, error)
}

// AWS SDK exposes GetQueueUrl (non-idiomatic capitalization); mirror it for mocks.
//
//nolint:revive // Matches aws-sdk-go-v2/service/sqs Client.GetQueueUrl
func (m *mockQueueURLGetter) GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput,
	optFns ...func(*sqs.Options),
) (*sqs.GetQueueUrlOutput, error) {
	return m.GetQueueURLFunc(ctx, params, optFns...)
}

type mockMessageReceiver struct {
	ReceiveMessageFunc func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (
		*sqs.ReceiveMessageOutput, error)
}

func (m *mockMessageReceiver) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput,
	optFns ...func(*sqs.Options),
) (*sqs.ReceiveMessageOutput, error) {
	return m.ReceiveMessageFunc(ctx, params, optFns...)
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

func TestGetQueueURL_Success(t *testing.T) {
	tests := []struct {
		name      string
		queueName string
		mockGet   func(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (
			*sqs.GetQueueUrlOutput, error)
		wantOutput string
	}{
		{
			name:      "success returns queue URL",
			queueName: "my-queue",
			mockGet: func(_ context.Context, params *sqs.GetQueueUrlInput,
				_ ...func(*sqs.Options),
			) (*sqs.GetQueueUrlOutput, error) {
				require.Equal(t, "my-queue", aws.ToString(params.QueueName))

				return &sqs.GetQueueUrlOutput{
					QueueUrl: aws.String("https://sqs.us-east-1.amazonaws.com/123456789012/my-queue"),
				}, nil
			},
			wantOutput: "QUEUE URL\nhttps://sqs.us-east-1.amazonaws.com/123456789012/my-queue\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &mockQueueURLGetter{GetQueueURLFunc: tt.mockGet}

			var buf bytes.Buffer

			err := GetQueueURL(context.Background(), api, tt.queueName, &buf)
			require.NoError(t, err)
			require.Equal(t, tt.wantOutput, buf.String())
		})
	}
}

func TestGetQueueURL_Error(t *testing.T) {
	api := &mockQueueURLGetter{
		GetQueueURLFunc: func(_ context.Context, _ *sqs.GetQueueUrlInput,
			_ ...func(*sqs.Options),
		) (*sqs.GetQueueUrlOutput, error) {
			return nil, errors.New("api error")
		},
	}

	var buf bytes.Buffer

	err := GetQueueURL(context.Background(), api, "my-queue", &buf)
	require.Error(t, err)
	require.Contains(t, err.Error(), "api error")
}

func TestReceiveMessage_Success(t *testing.T) {
	tests := []struct {
		name        string
		queueURL    string
		mockReceive func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (
			*sqs.ReceiveMessageOutput, error)
		wantOutput string
	}{
		{
			name:     "messages found",
			queueURL: "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue",
			mockReceive: func(_ context.Context, params *sqs.ReceiveMessageInput,
				_ ...func(*sqs.Options),
			) (*sqs.ReceiveMessageOutput, error) {
				wantQueueURL := "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue"
				require.Equal(t, wantQueueURL, aws.ToString(params.QueueUrl))

				return &sqs.ReceiveMessageOutput{
					Messages: []types.Message{
						{MessageId: aws.String("msg-1"), Body: aws.String("hello")},
						{MessageId: aws.String("msg-2"), Body: aws.String("world")},
					},
				}, nil
			},
			wantOutput: "MESSAGE ID  BODY\nmsg-1       hello\nmsg-2       world\n",
		},
		{
			name:     "no messages found",
			queueURL: "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue",
			mockReceive: func(_ context.Context, _ *sqs.ReceiveMessageInput,
				_ ...func(*sqs.Options),
			) (*sqs.ReceiveMessageOutput, error) {
				return &sqs.ReceiveMessageOutput{}, nil
			},
			wantOutput: "No SQS messages found.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &mockMessageReceiver{ReceiveMessageFunc: tt.mockReceive}

			var buf bytes.Buffer

			err := ReceiveMessage(context.Background(), api, tt.queueURL, &buf)
			require.NoError(t, err)
			require.Equal(t, tt.wantOutput, buf.String())
		})
	}
}

func TestReceiveMessage_Error(t *testing.T) {
	api := &mockMessageReceiver{
		ReceiveMessageFunc: func(_ context.Context, _ *sqs.ReceiveMessageInput,
			_ ...func(*sqs.Options),
		) (*sqs.ReceiveMessageOutput, error) {
			return nil, errors.New("api error")
		},
	}

	var buf bytes.Buffer

	err := ReceiveMessage(context.Background(), api, "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue", &buf)
	require.Error(t, err)
	require.Contains(t, err.Error(), "api error")
}
