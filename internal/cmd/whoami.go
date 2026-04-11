package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/spf13/cobra"
)

// stsCallerIdentityAPI defines the minimal interface for STS GetCallerIdentity.
// This enables unit testing without hitting real AWS endpoints.
type stsCallerIdentityAPI interface {
	GetCallerIdentity(
		ctx context.Context,
		params *sts.GetCallerIdentityInput,
		optFns ...func(*sts.Options),
	) (*sts.GetCallerIdentityOutput, error)
}

// newWhoamiCmd creates the `whoami` subcommand.
func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Display the current AWS caller identity",
		Long: `Calls AWS STS GetCallerIdentity and displays the Account,
ARN, and UserId associated with the currently configured
AWS credentials.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			cfg, err := config.LoadDefaultConfig(ctx)
			if err != nil {
				return fmt.Errorf("loading AWS config: %w", err)
			}

			client := sts.NewFromConfig(cfg)

			return runWhoami(ctx, client)
		},
	}
}

// credentialErrorHints lists substrings found in AWS SDK v2 errors
// when no valid credentials are available.
var credentialErrorHints = []string{
	"no EC2 IMDS role found",
	"failed to refresh cached credentials",
	"no credential providers",
	"AnonymousCredentials",
}

// isCredentialError returns true if err looks like a missing-credential error.
func isCredentialError(err error) bool {
	msg := err.Error()

	for _, hint := range credentialErrorHints {
		if strings.Contains(msg, hint) {
			return true
		}
	}

	return false
}

// noCredentialsMessage is printed when no active AWS credentials are found.
const noCredentialsMessage = `No active AWS credentials found.

Configure credentials using one of the following methods:
  • Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables
  • Run "aws configure" to create ~/.aws/credentials
  • Run "aws sso login" if using AWS IAM Identity Center
`

// runWhoami queries STS and prints caller identity details.
func runWhoami(ctx context.Context, api stsCallerIdentityAPI) error {
	output, err := api.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		if isCredentialError(err) {
			fmt.Fprint(os.Stderr, noCredentialsMessage)

			return nil
		}

		return fmt.Errorf("calling STS GetCallerIdentity: %w", err)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprintf(tw, "Account:\t%s\n", derefString(output.Account))
	fmt.Fprintf(tw, "Arn:\t%s\n", derefString(output.Arn))
	fmt.Fprintf(tw, "UserId:\t%s\n", derefString(output.UserId))

	return tw.Flush()
}

// derefString safely dereferences a *string, returning "" if nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}
