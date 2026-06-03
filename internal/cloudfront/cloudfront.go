package cloudfront

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	acmtypes "github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/xenos76/aws-probe/internal/awsutil"
	"github.com/xenos76/aws-probe/internal/output"
)

// CertificateReport represents the properties of a CloudFront distribution certificate.
type CertificateReport struct {
	DistributionID        string   `json:"distributionId"`
	DomainName            string   `json:"domainName"`
	CertificateDomainName string   `json:"certificateDomainName"`
	AlternateDomainNames  []string `json:"alternateDomainNames"`
	ExpiryDate            string   `json:"expiryDate"`
	RenewalStatus         string   `json:"renewalStatus"`
	SecurityPolicy        string   `json:"securityPolicy"`
}

// ClientAPI defines the interface for CloudFront API operations.
type ClientAPI interface {
	ListDistributions(ctx context.Context, params *cloudfront.ListDistributionsInput,
		optFns ...func(*cloudfront.Options)) (*cloudfront.ListDistributionsOutput, error)
	GetDistribution(ctx context.Context, params *cloudfront.GetDistributionInput,
		optFns ...func(*cloudfront.Options)) (*cloudfront.GetDistributionOutput, error)
}

// ACMClientAPI defines the interface for ACM API operations.
type ACMClientAPI interface {
	DescribeCertificate(ctx context.Context, params *acm.DescribeCertificateInput,
		optFns ...func(*acm.Options)) (*acm.DescribeCertificateOutput, error)
}

// NewClient creates a new CloudFront client.
func NewClient(cfg aws.Config) *cloudfront.Client {
	return cloudfront.NewFromConfig(cfg)
}

// NewACMClient creates a new ACM client.
func NewACMClient(cfg aws.Config) *acm.Client {
	return acm.NewFromConfig(cfg)
}

// CollectCertificates gathers certificate data for all CloudFront distributions.
func CollectCertificates(ctx context.Context, cfClient ClientAPI, acmClient ACMClientAPI) (
	[]CertificateReport, error,
) {
	var (
		reports []CertificateReport
		marker  *string
	)

	for {
		resp, err := cfClient.ListDistributions(ctx, &cloudfront.ListDistributionsInput{
			Marker: marker,
		})
		if err != nil {
			return nil, err
		}

		if resp.DistributionList != nil && len(resp.DistributionList.Items) > 0 {
			for _, dist := range resp.DistributionList.Items {
				report := processDistribution(ctx, cfClient, acmClient, dist)
				reports = append(reports, report)
			}
		}

		if resp.DistributionList == nil || resp.DistributionList.IsTruncated == nil ||
			!*resp.DistributionList.IsTruncated {
			break
		}

		marker = resp.DistributionList.NextMarker
	}

	return reports, nil
}

// ListCertificates gathers certificate data for all CloudFront distributions.
func ListCertificates(ctx context.Context, cfClient ClientAPI, acmClient ACMClientAPI,
	out io.Writer, format, theme string,
) error {
	reports, err := CollectCertificates(ctx, cfClient, acmClient)
	if err != nil {
		return err
	}

	headers := []string{
		"Distribution ID",
		"Distribution Domain Name",
		"Certificate Domain Name",
		"Alternate Domain Names",
		"Expiration",
		"Renewal Status",
		"Security Policy",
	}

	var rows [][]string

	for _, r := range reports {
		altNames := strings.Join(r.AlternateDomainNames, "\n")
		if format == "csv" {
			altNames = strings.Join(r.AlternateDomainNames, ", ")
		}

		rows = append(rows, []string{
			r.DistributionID,
			r.DomainName,
			r.CertificateDomainName,
			altNames,
			r.ExpiryDate,
			r.RenewalStatus,
			r.SecurityPolicy,
		})
	}

	data := output.TableData{
		Headers: headers,
		Rows:    rows,
		Raw:     reports,
	}

	return output.Print(out, format, theme, data)
}

func processDistribution(ctx context.Context, cfClient ClientAPI, acmClient ACMClientAPI,
	dist types.DistributionSummary,
) CertificateReport {
	report := CertificateReport{
		DistributionID: awsutil.DerefString(dist.Id),
		DomainName:     awsutil.DerefString(dist.DomainName),
	}

	resolveMissingData(ctx, cfClient, &dist)

	if dist.Aliases != nil && len(dist.Aliases.Items) > 0 {
		report.AlternateDomainNames = dist.Aliases.Items
	}

	if dist.ViewerCertificate != nil {
		report.SecurityPolicy = string(dist.ViewerCertificate.MinimumProtocolVersion)
		if report.SecurityPolicy == "" && dist.ViewerCertificate.CloudFrontDefaultCertificate != nil &&
			*dist.ViewerCertificate.CloudFrontDefaultCertificate {
			report.SecurityPolicy = "TLSv1"
		}

		acmDetails := fetchCertificateDetails(ctx, acmClient, dist.ViewerCertificate)
		report.ExpiryDate = acmDetails.ExpiryDate
		report.CertificateDomainName = acmDetails.CertificateDomainName
		report.RenewalStatus = acmDetails.RenewalStatus
	} else {
		report.SecurityPolicy = "N/A"
		report.ExpiryDate = "N/A"
		report.CertificateDomainName = "N/A"
		report.RenewalStatus = "N/A"
	}

	return report
}

type certDetails struct {
	ExpiryDate            string
	CertificateDomainName string
	RenewalStatus         string
}

func fetchCertificateDetails(ctx context.Context, acmClient ACMClientAPI, cert *types.ViewerCertificate) certDetails {
	if cert.ACMCertificateArn != nil {
		return fetchACMDetails(ctx, acmClient, awsutil.DerefString(cert.ACMCertificateArn))
	}

	if cert.CloudFrontDefaultCertificate != nil && *cert.CloudFrontDefaultCertificate {
		return certDetails{
			ExpiryDate:            "Managed by AWS",
			CertificateDomainName: "Managed by AWS",
			RenewalStatus:         "Managed by AWS",
		}
	}

	return certDetails{
		ExpiryDate:            "N/A",
		CertificateDomainName: "N/A",
		RenewalStatus:         "N/A",
	}
}

func fetchACMDetails(ctx context.Context, acmClient ACMClientAPI, arn string) certDetails {
	details := certDetails{
		ExpiryDate:            "Unknown",
		CertificateDomainName: "Unknown",
		RenewalStatus:         "Unknown",
	}

	var optFns []func(*acm.Options)

	parts := strings.Split(arn, ":")
	if len(parts) > 3 && parts[3] != "" {
		region := parts[3]

		optFns = append(optFns, func(o *acm.Options) {
			o.Region = region
		})
	}

	resp, err := acmClient.DescribeCertificate(ctx, &acm.DescribeCertificateInput{
		CertificateArn: aws.String(arn),
	}, optFns...)
	if err != nil || resp.Certificate == nil {
		return details
	}

	cert := resp.Certificate

	if cert.NotAfter != nil {
		details.ExpiryDate = cert.NotAfter.Format(time.RFC3339)
	}

	if cert.DomainName != nil {
		details.CertificateDomainName = *cert.DomainName
	}

	details.RenewalStatus = resolveRenewalStatus(cert)

	return details
}

func resolveRenewalStatus(cert *acmtypes.CertificateDetail) string {
	if cert.RenewalSummary != nil {
		return string(cert.RenewalSummary.RenewalStatus)
	}

	if cert.RenewalEligibility != "" {
		return string(cert.RenewalEligibility)
	}

	return "Unknown"
}

func resolveMissingData(ctx context.Context, cfClient ClientAPI, dist *types.DistributionSummary) {
	if dist.ViewerCertificate != nil && dist.Aliases != nil {
		return
	}

	getResp, err := cfClient.GetDistribution(ctx, &cloudfront.GetDistributionInput{
		Id: dist.Id,
	})
	if err != nil || getResp.Distribution == nil || getResp.Distribution.DistributionConfig == nil {
		return
	}

	if dist.ViewerCertificate == nil {
		dist.ViewerCertificate = getResp.Distribution.DistributionConfig.ViewerCertificate
	}

	if dist.Aliases == nil {
		dist.Aliases = getResp.Distribution.DistributionConfig.Aliases
	}
}
