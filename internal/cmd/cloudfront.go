package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xenos76/aws-probe/internal/cloudfront"
)

// newCloudfrontCmd creates the CloudFront command.
func newCloudfrontCmd() *cobra.Command {
	var listCertificatesFlag bool

	cmd := &cobra.Command{
		Use:   "cloudfront",
		Short: "Manage CloudFront resources",
		Long:  `Manage CloudFront distributions and associated resources.`,
		Example: `  # List all certificates for distributions
  aws-probe cloudfront --list-certificates`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if listCertificatesFlag {
				return runListCertificates(cmd)
			}

			return cmd.Help()
		},
	}

	cmd.Flags().BoolVar(&listCertificatesFlag, "list-certificates", false,
		"List certificates of all CloudFront distributions")

	return cmd
}

func runListCertificates(cmd *cobra.Command) error {
	cfg, err := PrepareAWSConfig(cmd.Context())
	if err != nil {
		return err
	}

	cfClient := cloudfront.NewClient(cfg)
	acmClient := cloudfront.NewACMClient(cfg)

	return cloudfront.ListCertificates(
		cmd.Context(),
		cfClient,
		acmClient,
		cmd.OutOrStdout(),
		OutputFormat,
		Theme,
	)
}
