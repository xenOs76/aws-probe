package cloudfront

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockClientAPI struct {
	mock.Mock
}

func (m *MockClientAPI) ListDistributions(
	ctx context.Context,
	params *cloudfront.ListDistributionsInput,
	_ ...func(*cloudfront.Options),
) (*cloudfront.ListDistributionsOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	var out *cloudfront.ListDistributionsOutput
	if val, ok := args.Get(0).(*cloudfront.ListDistributionsOutput); ok {
		out = val
	}

	return out, args.Error(1)
}

func (m *MockClientAPI) GetDistribution(
	ctx context.Context,
	params *cloudfront.GetDistributionInput,
	_ ...func(*cloudfront.Options),
) (*cloudfront.GetDistributionOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	var out *cloudfront.GetDistributionOutput
	if val, ok := args.Get(0).(*cloudfront.GetDistributionOutput); ok {
		out = val
	}

	return out, args.Error(1)
}

type MockACMClientAPI struct {
	mock.Mock
}

func (m *MockACMClientAPI) DescribeCertificate(
	ctx context.Context,
	params *acm.DescribeCertificateInput,
	_ ...func(*acm.Options),
) (*acm.DescribeCertificateOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	var out *acm.DescribeCertificateOutput
	if val, ok := args.Get(0).(*acm.DescribeCertificateOutput); ok {
		out = val
	}

	return out, args.Error(1)
}

func TestListCertificates_Success(t *testing.T) {
	ctx := context.Background()
	cfMock := new(MockClientAPI)
	acmMock := new(MockACMClientAPI)

	now := time.Now()
	expiryStr := now.Format(time.RFC3339)

	cfMock.On("ListDistributions", ctx, mock.Anything).Return(&cloudfront.ListDistributionsOutput{
		DistributionList: &cftypes.DistributionList{
			IsTruncated: aws.Bool(false),
			Items: []cftypes.DistributionSummary{
				{
					Id:         aws.String("D123"),
					DomainName: aws.String("test.cloudfront.net"),
					Aliases: &cftypes.Aliases{
						Items: []string{"test.example.com"},
					},
					ViewerCertificate: &cftypes.ViewerCertificate{
						MinimumProtocolVersion: cftypes.MinimumProtocolVersionTLSv122021,
						ACMCertificateArn:      aws.String("arn:aws:acm:us-east-1:123:certificate/123"),
					},
				},
			},
		},
	}, nil).Once()

	acmMock.On("DescribeCertificate", ctx, mock.Anything).Return(&acm.DescribeCertificateOutput{
		Certificate: &types.CertificateDetail{
			NotAfter: aws.Time(now),
		},
	}, nil).Once()

	var out bytes.Buffer

	err := ListCertificates(ctx, cfMock, acmMock, &out, "json", "none")
	require.NoError(t, err)

	require.Contains(t, out.String(), "test.example.com")
	require.Contains(t, out.String(), expiryStr)
	require.Contains(t, out.String(), "TLSv1.2_2021")

	cfMock.AssertExpectations(t)
	acmMock.AssertExpectations(t)
}

func TestListCertificates_ListDistributionsError(t *testing.T) {
	ctx := context.Background()
	cfMock := new(MockClientAPI)
	acmMock := new(MockACMClientAPI)

	cfMock.On("ListDistributions", ctx, mock.Anything).Return(nil, errors.New("API error")).Once()

	var out bytes.Buffer

	err := ListCertificates(ctx, cfMock, acmMock, &out, "json", "none")
	require.Error(t, err)
	require.Equal(t, "API error", err.Error())

	cfMock.AssertExpectations(t)
}

func TestListCertificates_FallbackGetDistribution(t *testing.T) {
	ctx := context.Background()
	cfMock := new(MockClientAPI)
	acmMock := new(MockACMClientAPI)

	cfMock.On("ListDistributions", ctx, mock.Anything).Return(&cloudfront.ListDistributionsOutput{
		DistributionList: &cftypes.DistributionList{
			IsTruncated: aws.Bool(false),
			Items: []cftypes.DistributionSummary{
				{
					Id:         aws.String("D456"),
					DomainName: aws.String("fallback.cloudfront.net"),
				},
			},
		},
	}, nil).Once()

	cfMock.On("GetDistribution", ctx, mock.Anything).Return(&cloudfront.GetDistributionOutput{
		Distribution: &cftypes.Distribution{
			DistributionConfig: &cftypes.DistributionConfig{
				ViewerCertificate: &cftypes.ViewerCertificate{
					CloudFrontDefaultCertificate: aws.Bool(true),
				},
			},
		},
	}, nil).Once()

	var out bytes.Buffer

	err := ListCertificates(ctx, cfMock, acmMock, &out, "json", "none")
	require.NoError(t, err)
	require.Contains(t, out.String(), "fallback.cloudfront.net")
	require.Contains(t, out.String(), "Managed by AWS")
	require.Contains(t, out.String(), "TLSv1") // Defaults to TLSv1

	cfMock.AssertExpectations(t)
}
