package whoami

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/require"
	"github.com/xenos76/aws-probe/internal/awsutil"
)

type mockStsClient struct {
	GetCallerIdentityFunc func(ctx context.Context, params *sts.GetCallerIdentityInput,
		optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

func (m *mockStsClient) GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput,
	optFns ...func(*sts.Options),
) (*sts.GetCallerIdentityOutput, error) {
	return m.GetCallerIdentityFunc(ctx, params, optFns...)
}

func TestGetCallerIdentity(t *testing.T) {
	tests := []struct {
		name                  string
		mockGetCallerIdentity func(ctx context.Context, params *sts.GetCallerIdentityInput,
			optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
		want    *CallerIdentity
		wantErr bool
	}{
		{
			name: "success",
			mockGetCallerIdentity: func(_ context.Context, _ *sts.GetCallerIdentityInput,
				_ ...func(*sts.Options),
			) (*sts.GetCallerIdentityOutput, error) {
				return &sts.GetCallerIdentityOutput{
					Account: aws.String("123456789012"),
					Arn:     aws.String("arn:aws:iam::123456789012:user/test"),
					UserId:  aws.String("AIDA..."),
				}, nil
			},
			want: &CallerIdentity{
				Account: "123456789012",
				Arn:     "arn:aws:iam::123456789012:user/test",
				UserID:  "AIDA...",
			},
			wantErr: false,
		},
		{
			name: "error",
			mockGetCallerIdentity: func(_ context.Context, _ *sts.GetCallerIdentityInput,
				_ ...func(*sts.Options),
			) (*sts.GetCallerIdentityOutput, error) {
				return nil, errors.New("api error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &mockStsClient{GetCallerIdentityFunc: tt.mockGetCallerIdentity}
			got, err := GetCallerIdentity(context.Background(), api)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDisplayCallerIdentity(t *testing.T) {
	tests := []struct {
		name                  string
		mockGetCallerIdentity func(ctx context.Context, params *sts.GetCallerIdentityInput,
			optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
		authMethod awsutil.AuthMethod
		wantOutput string
		wantErr    bool
	}{
		{
			name: "success",
			mockGetCallerIdentity: func(_ context.Context, _ *sts.GetCallerIdentityInput,
				_ ...func(*sts.Options),
			) (*sts.GetCallerIdentityOutput, error) {
				return &sts.GetCallerIdentityOutput{
					Account: aws.String("123456789012"),
					Arn:     aws.String("arn:aws:iam::123456789012:user/test"),
					UserId:  aws.String("AIDA..."),
				}, nil
			},
			authMethod: awsutil.AuthMethod{IdentitySource: "Env Vars"},
			wantOutput: "ACCOUNT      123456789012\n" +
				"ARN          arn:aws:iam::123456789012:user/test\n" +
				"USER ID      AIDA...\n" +
				"AUTH METHOD  Env Vars\n",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &mockStsClient{GetCallerIdentityFunc: tt.mockGetCallerIdentity}

			var buf bytes.Buffer

			err := DisplayCallerIdentity(context.Background(), api, tt.authMethod, &buf)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantOutput, buf.String())
		})
	}
}
