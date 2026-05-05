package sns

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/stretchr/testify/require"
)

type mockSnsLister struct {
	ListTopicsFunc func(ctx context.Context, params *sns.ListTopicsInput,
		optFns ...func(*sns.Options)) (*sns.ListTopicsOutput, error)
	ListSubscriptionsByTopicFunc func(ctx context.Context, params *sns.ListSubscriptionsByTopicInput,
		optFns ...func(*sns.Options)) (*sns.ListSubscriptionsByTopicOutput, error)
}

func (m *mockSnsLister) ListTopics(ctx context.Context, params *sns.ListTopicsInput,
	optFns ...func(*sns.Options),
) (*sns.ListTopicsOutput, error) {
	return m.ListTopicsFunc(ctx, params, optFns...)
}

func (m *mockSnsLister) ListSubscriptionsByTopic(ctx context.Context, params *sns.ListSubscriptionsByTopicInput,
	optFns ...func(*sns.Options),
) (*sns.ListSubscriptionsByTopicOutput, error) {
	return m.ListSubscriptionsByTopicFunc(ctx, params, optFns...)
}

func TestListTopics(t *testing.T) {
	tests := []struct {
		name           string
		mockListTopics func(ctx context.Context, params *sns.ListTopicsInput,
			optFns ...func(*sns.Options)) (*sns.ListTopicsOutput, error)
		wantOutput string
		wantErr    bool
	}{
		{
			name: "success",
			mockListTopics: func(_ context.Context, _ *sns.ListTopicsInput,
				_ ...func(*sns.Options),
			) (*sns.ListTopicsOutput, error) {
				return &sns.ListTopicsOutput{
					Topics: []types.Topic{
						{TopicArn: aws.String("arn:aws:sns:us-east-1:123456789012:topic1")},
					},
				}, nil
			},
			wantOutput: "TOPIC ARN\narn:aws:sns:us-east-1:123456789012:topic1\n",
			wantErr:    false,
		},
		{
			name: "no topics",
			mockListTopics: func(
				_ context.Context,
				_ *sns.ListTopicsInput,
				_ ...func(*sns.Options),
			) (*sns.ListTopicsOutput, error) {
				return &sns.ListTopicsOutput{}, nil
			},
			wantOutput: "No SNS topics found.\n",
			wantErr:    false,
		},
		{
			name: "error",
			mockListTopics: func(
				_ context.Context,
				_ *sns.ListTopicsInput,
				_ ...func(*sns.Options),
			) (*sns.ListTopicsOutput, error) {
				return nil, errors.New("api error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &mockSnsLister{ListTopicsFunc: tt.mockListTopics}

			var buf bytes.Buffer

			err := ListTopics(context.Background(), api, &buf)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantOutput, buf.String())
		})
	}
}

func TestListSubscriptions(t *testing.T) {
	topicArn := "arn:aws:sns:us-east-1:123456789012:topic1"
	tests := []struct {
		name                  string
		mockListSubscriptions func(ctx context.Context, params *sns.ListSubscriptionsByTopicInput,
			optFns ...func(*sns.Options)) (*sns.ListSubscriptionsByTopicOutput, error)
		wantOutput string
		wantErr    bool
	}{
		{
			name: "success",
			mockListSubscriptions: func(_ context.Context, _ *sns.ListSubscriptionsByTopicInput,
				_ ...func(*sns.Options),
			) (*sns.ListSubscriptionsByTopicOutput, error) {
				return &sns.ListSubscriptionsByTopicOutput{
					Subscriptions: []types.Subscription{
						{
							TopicArn: aws.String(topicArn),
							Protocol: aws.String("email"),
							Endpoint: aws.String("test@example.com"),
							Owner:    aws.String("123456789012"),
						},
					},
				}, nil
			},
			wantOutput: "TOPIC ARN                                  PROTOCOL  ENDPOINT          OWNER\n" +
				"arn:aws:sns:us-east-1:123456789012:topic1  email     test@example.com  123456789012\n",
			wantErr: false,
		},
		{
			name: "no subscriptions",
			mockListSubscriptions: func(
				_ context.Context,
				_ *sns.ListSubscriptionsByTopicInput,
				_ ...func(*sns.Options),
			) (*sns.ListSubscriptionsByTopicOutput, error) {
				return &sns.ListSubscriptionsByTopicOutput{}, nil
			},
			wantOutput: "No subscriptions found.\n",
			wantErr:    false,
		},
		{
			name: "error",
			mockListSubscriptions: func(
				_ context.Context,
				_ *sns.ListSubscriptionsByTopicInput,
				_ ...func(*sns.Options),
			) (*sns.ListSubscriptionsByTopicOutput, error) {
				return nil, errors.New("api error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &mockSnsLister{ListSubscriptionsByTopicFunc: tt.mockListSubscriptions}

			var buf bytes.Buffer

			err := ListSubscriptions(context.Background(), topicArn, api, &buf)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantOutput, buf.String())
		})
	}
}
