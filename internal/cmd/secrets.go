//nolint:dupl // CLI handlers follow a similar pattern
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xenos76/aws-probe/internal/secrets"
)

// newSecretsCmd creates the secrets command.
//
//nolint:dupl // CLI handlers follow a similar pattern
func newSecretsCmd() *cobra.Command {
	var (
		listSecrets bool
		getSecret   string
	)

	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage Secrets Manager secrets",
		Long:  `List secrets and retrieve secret values from AWS Secrets Manager.`,
		Example: `  # List all secrets
  aws-probe secrets --list-secrets

  # Get secret value
  aws-probe secrets --get-secret-value my-secret-id`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !listSecrets && getSecret == "" {
				return cmd.Help()
			}

			cfg, err := PrepareAWSConfig(cmd.Context())
			if err != nil {
				return err
			}

			client := secrets.NewClient(cfg)

			if listSecrets {
				return secrets.ListSecrets(cmd.Context(), client, cmd.OutOrStdout())
			}

			return secrets.GetSecretValue(cmd.Context(), client, getSecret, cmd.OutOrStdout())
		},
	}

	cmd.Flags().BoolVar(&listSecrets, "list-secrets", false, "List all secrets")
	cmd.Flags().StringVar(&getSecret, "get-secret-value", "", "Retrieve the value of a secret")

	cmd.MarkFlagsMutuallyExclusive("list-secrets", "get-secret-value")

	return cmd
}
