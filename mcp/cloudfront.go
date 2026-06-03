package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	internalcf "github.com/xenos76/aws-probe/internal/cloudfront"
)

type listCloudFrontCertsOutput struct {
	Certificates []internalcf.CertificateReport `json:"certificates"`
}

func registerCloudFrontTools(server *mcp.Server, deps *Deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "aws_probe_cloudfront_list_certificates",
		Description: "List TLS certificate details for all CloudFront distributions",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (
		*mcp.CallToolResult, listCloudFrontCertsOutput, error,
	) {
		cfg, err := loadAWS(ctx, deps)
		if err != nil {
			return nil, listCloudFrontCertsOutput{}, err
		}

		reports, err := internalcf.CollectCertificates(ctx, internalcf.NewClient(cfg), internalcf.NewACMClient(cfg))
		if err != nil {
			return nil, listCloudFrontCertsOutput{}, err
		}

		return nil, listCloudFrontCertsOutput{Certificates: reports}, nil
	})
}
