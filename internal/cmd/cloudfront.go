package cmd

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/xenos76/aws-probe/internal/cloudfront"
)

// newCloudfrontCmd creates the CloudFront command.
func newCloudfrontCmd() *cobra.Command {
	var (
		listCertificatesFlag bool
		outputFormat         string
		theme                string
	)

	cmd := &cobra.Command{
		Use:   "cloudfront",
		Short: "Manage CloudFront resources",
		Long:  `Manage CloudFront distributions and associated resources.`,
		Example: `  # List all certificates for distributions
  aws-probe cloudfront --list-certificates

  # List certificates with specific output format and theme
  aws-probe cloudfront --list-certificates --output json --theme dracula`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if listCertificatesFlag {
				return runListCertificates(cmd, outputFormat, theme)
			}

			return errors.New("an action flag is required: use --list-certificates")
		},
	}

	cmd.Flags().BoolVar(&listCertificatesFlag, "list-certificates", false,
		"List certificates of all CloudFront distributions")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, csv, json)")
	cmd.Flags().StringVarP(&theme, "theme", "t", "catppuccin-frappe",
		"Output theme (catppuccin-frappe, dracula, nord, none)")

	return cmd
}

func runListCertificates(cmd *cobra.Command, outputFormat, theme string) error {
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
		outputFormat,
		theme,
	)
}
