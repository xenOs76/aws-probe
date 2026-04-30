package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSNSClient struct {
	listTopicsOutput        *sns.ListTopicsOutput
	listSubscriptionsOutput *sns.ListSubscriptionsByTopicOutput
	err                     error
}

func (m *mockSNSClient) ListTopics(
	_ context.Context,
	_ *sns.ListTopicsInput,
	_ ...func(*sns.Options),
) (*sns.ListTopicsOutput, error) {
	return m.listTopicsOutput, m.err
}

func (m *mockSNSClient) ListSubscriptionsByTopic(
	_ context.Context,
	_ *sns.ListSubscriptionsByTopicInput,
	_ ...func(*sns.Options),
) (*sns.ListSubscriptionsByTopicOutput, error) {
	return m.listSubscriptionsOutput, m.err
}

func TestListSnsTopics(t *testing.T) {
	tests := []struct {
		name    string
		client  *mockSNSClient
		wantOut string
		wantErr bool
	}{
		{
			name: "single topic",
			client: &mockSNSClient{
				listTopicsOutput: &sns.ListTopicsOutput{
					Topics: []snstypes.Topic{
						{TopicArn: aws.String("arn:aws:sns:region:123:mytopic")},
					},
				},
			},
			wantOut: "TOPIC ARN\narn:aws:sns:region:123:mytopic\n",
		},
		{
			name: "no topics",
			client: &mockSNSClient{
				listTopicsOutput: &sns.ListTopicsOutput{},
			},
			wantOut: "",
		},
		{
			name: "API error",
			client: &mockSNSClient{
				err: errors.New("api error"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := captureCmdOutput(t, func() error {
				return listSnsTopics(context.Background(), tt.client)
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

func TestListSnsSubscriptions(t *testing.T) {
	tests := []struct {
		name    string
		client  *mockSNSClient
		wantOut string
		wantErr bool
	}{
		{
			name: "single subscription",
			client: &mockSNSClient{
				listSubscriptionsOutput: &sns.ListSubscriptionsByTopicOutput{
					Subscriptions: []snstypes.Subscription{
						{
							TopicArn: aws.String("arn:aws:sns:region:123:topic"),
							Protocol: aws.String("lambda"),
							Endpoint: aws.String("arn:aws:lambda:region:123:func"),
							Owner:    aws.String("123"),
						},
					},
				},
			},
			wantOut: "TOPIC ARN                     PROTOCOL  ENDPOINT                        OWNER\n" +
				"arn:aws:sns:region:123:topic  lambda    arn:aws:lambda:region:123:func  123\n",
		},
		{
			name: "no subscriptions",
			client: &mockSNSClient{
				listSubscriptionsOutput: &sns.ListSubscriptionsByTopicOutput{},
			},
			wantOut: "",
		},
		{
			name: "API error",
			client: &mockSNSClient{
				err: errors.New("api error"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := captureCmdOutput(t, func() error {
				return listSnsSubscriptions(context.Background(), "arn:aws:sns:region:123:topic", tt.client)
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
