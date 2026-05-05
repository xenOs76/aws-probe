package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xenos76/aws-probe/internal/awsutil"
	"github.com/xenos76/aws-probe/internal/whoami"
)

// newWhoamiCmd creates the whoami command.
func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Display the current AWS caller identity",
		Long: `Display information about the current AWS caller identity,
including the account ID, IAM ARN, and user ID.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := PrepareAWSConfig(cmd.Context())
			if err != nil {
				return err
			}

			client := whoami.NewSTSClient(cfg)
			auth := awsutil.DetectAuthMethod()

			return whoami.DisplayCallerIdentity(cmd.Context(), client, auth, cmd.OutOrStdout())
		},
	}
}
