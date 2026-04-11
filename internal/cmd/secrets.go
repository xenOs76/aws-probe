package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/spf13/cobra"
)

func newSecretsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "List Secrets Manager secrets",
		Long:  `List secrets in AWS Secrets Manager.`,
	}

	cmd.AddCommand(newListSecretsCmd())

	return cmd
}

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

func runListSecrets(ctx context.Context) error {
	if err := EnsureCredentials(); err != nil {
		return err
	}

	cfg, err := LoadAWSConfig(ctx)
	if err != nil {
		return fmt.Errorf("loading AWS config: %w", err)
	}

	return listSecrets(ctx, secretsmanager.NewFromConfig(cfg))
}

func listSecrets(ctx context.Context, api secretsListAPI) error {
	output, err := api.ListSecrets(ctx, &secretsmanager.ListSecretsInput{})
	if err != nil {
		return fmt.Errorf("listing secrets: %w", err)
	}

	if len(output.SecretList) == 0 {
		fmt.Fprintln(os.Stderr, "No secrets found.")

		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprint(tw, "NAME\tARN\n")

	for _, secret := range output.SecretList {
		fmt.Fprintf(tw, "%s\t%s\n", derefString(secret.Name), derefString(secret.ARN))
	}

	return tw.Flush()
}
