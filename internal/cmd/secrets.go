package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/spf13/cobra"
)

// newSecretsCmd creates the secrets command.
func newSecretsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "List Secrets Manager secrets",
		Long:  `List secrets in AWS Secrets Manager.`,
	}

	cmd.AddCommand(newListSecretsCmd())

	return cmd
}

// newListSecretsCmd creates the list-secrets subcommand.
func newListSecretsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-secrets",
		Short: "List Secrets Manager secrets",
		Long:  `List all secrets in AWS Secrets Manager.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runListSecrets(cmd.Context())
		},
	}
}

// runListSecrets executes the list-secrets command.
func runListSecrets(ctx context.Context) error {
	cfg, err := PrepareAWSConfig(ctx)
	if err != nil {
		return err
	}

	return listSecrets(ctx, secretsmanager.NewFromConfig(cfg))
}

// listSecrets lists secrets using the provided API client.
func listSecrets(ctx context.Context, api secretsLister) error {
	paginator := secretsmanager.NewListSecretsPaginator(api, &secretsmanager.ListSecretsInput{})

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	hasSecrets := false

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			_ = tw.Flush()
			return fmt.Errorf("listing secrets: %w", err)
		}

		for _, secret := range output.SecretList {
			if !hasSecrets {
				fmt.Fprint(tw, "NAME\tARN\n")

				hasSecrets = true
			}

			fmt.Fprintf(tw, "%s\t%s\n", derefString(secret.Name), derefString(secret.ARN))
		}
	}

	if !hasSecrets {
		_, _ = fmt.Fprintln(os.Stderr, "No secrets found.")

		return nil
	}

	return tw.Flush()
}
