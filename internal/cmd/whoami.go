package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/spf13/cobra"
)

type stsCallerIdentityAPI interface {
	GetCallerIdentity(
		ctx context.Context,
		params *sts.GetCallerIdentityInput,
		optFns ...func(*sts.Options),
	) (*sts.GetCallerIdentityOutput, error)
}

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Display the current AWS caller identity",
		Long: `Calls AWS STS GetCallerIdentity and displays the Account,
ARN, and UserId associated with the currently configured
AWS credentials. Also detects and displays the authentication
method used (EC2 role, EKS IRSA, SSO, etc.).`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			cfg, err := LoadAWSConfig(ctx)
			if err != nil {
				return fmt.Errorf("loading AWS config: %w", err)
			}

			client := sts.NewFromConfig(cfg)

			return runWhoami(ctx, client)
		},
	}
}

func runWhoami(ctx context.Context, api stsCallerIdentityAPI) error {
	auth := DetectAuthMethod()

	output, err := api.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		if IsCredentialError(err) {
			fmt.Fprint(os.Stderr, noCredentialsMessage)

			return nil
		}

		return fmt.Errorf("calling STS GetCallerIdentity: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Authentication: %s\n", auth.IdentitySource)

	if auth.RoleARN != "" {
		fmt.Fprintf(os.Stderr, "IAM Role: %s\n", auth.RoleARN)
	}

	if auth.ServiceAccount != "" {
		fmt.Fprintf(os.Stderr, "Service Account: %s\n", auth.ServiceAccount)
	}

	fmt.Fprintln(os.Stderr)

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprintf(tw, "Account:\t%s\n", derefString(output.Account))
	fmt.Fprintf(tw, "Arn:\t%s\n", derefString(output.Arn))
	fmt.Fprintf(tw, "UserId:\t%s\n", derefString(output.UserId))

	return tw.Flush()
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}
