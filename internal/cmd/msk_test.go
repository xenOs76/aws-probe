package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockKafkaGetBrokersAPI struct {
	GetBootstrapBrokersFn func(
		ctx context.Context,
		params *kafka.GetBootstrapBrokersInput,
		optFns ...func(*kafka.Options),
	) (*kafka.GetBootstrapBrokersOutput, error)
}

func (m *mockKafkaGetBrokersAPI) GetBootstrapBrokers(
	ctx context.Context,
	params *kafka.GetBootstrapBrokersInput,
	optFns ...func(*kafka.Options),
) (*kafka.GetBootstrapBrokersOutput, error) {
	return m.GetBootstrapBrokersFn(ctx, params, optFns...)
}

var resolveBrokersTests = []struct {
	name        string
	mskBrokers  string
	mskArn      string
	mskAuth     string
	mskTLS      bool
	mockResp    *kafka.GetBootstrapBrokersOutput
	mockErr     error
	wantBrokers []string
	wantErr     bool
}{
	{
		name:        "Explicit brokers",
		mskBrokers:  "b1:9092,b2:9092",
		wantBrokers: []string{"b1:9092", "b2:9092"},
	},
	{
		name:    "IAM auth from MSK",
		mskArn:  "arn:aws:kafka:region:account:cluster/name/id",
		mskAuth: "iam",
		mockResp: &kafka.GetBootstrapBrokersOutput{
			BootstrapBrokerStringSaslIam: aws.String("iam1:9098,iam2:9098"),
		},
		wantBrokers: []string{"iam1:9098", "iam2:9098"},
	},
	{
		name:    "TLS auth from MSK",
		mskArn:  "arn:aws:kafka:region:account:cluster/name/id",
		mskAuth: "none",
		mskTLS:  true,
		mockResp: &kafka.GetBootstrapBrokersOutput{
			BootstrapBrokerStringTls: aws.String("tls1:9094,tls2:9094"),
		},
		wantBrokers: []string{"tls1:9094", "tls2:9094"},
	},
	{
		name:    "Plaintext auth from MSK",
		mskArn:  "arn:aws:kafka:region:account:cluster/name/id",
		mskAuth: "none",
		mskTLS:  false,
		mockResp: &kafka.GetBootstrapBrokersOutput{
			BootstrapBrokerString: aws.String("p1:9092,p2:9092"),
		},
		wantBrokers: []string{"p1:9092", "p2:9092"},
	},
	{
		name:    "No brokers or ARN",
		wantErr: true,
	},
	{
		name:    "API error",
		mskArn:  "arn:aws:kafka:region:account:cluster/name/id",
		mockErr: errors.New("api error"),
		wantErr: true,
	},
}

func TestResolveBrokers(t *testing.T) {
	ctx := context.Background()

	for _, tt := range resolveBrokersTests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore global variables
			oldBrokers := mskBrokers
			oldArn := mskClusterArn
			oldAuth := mskAuth
			oldTLS := mskTLS

			defer func() {
				mskBrokers = oldBrokers
				mskClusterArn = oldArn
				mskAuth = oldAuth
				mskTLS = oldTLS
			}()

			mskBrokers = tt.mskBrokers
			mskClusterArn = tt.mskArn
			mskAuth = tt.mskAuth
			mskTLS = tt.mskTLS

			mockAPI := &mockKafkaGetBrokersAPI{
				GetBootstrapBrokersFn: func(
					_ context.Context,
					_ *kafka.GetBootstrapBrokersInput,
					_ ...func(*kafka.Options),
				) (*kafka.GetBootstrapBrokersOutput, error) {
					return tt.mockResp, tt.mockErr
				},
			}

			got, err := resolveBrokers(ctx, mockAPI)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantBrokers, got)
			}
		})
	}
}
