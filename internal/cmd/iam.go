package cmd

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/xenos76/aws-probe/internal/iam"
)

// newIamCmd creates the IAM command.
func newIamCmd() *cobra.Command {
	var (
		listRoles    bool
		listPolicies bool
		dumpPolicy   string
		dumpRole     string
		scope        string
		outputFormat string
		theme        string
	)

	cmd := &cobra.Command{
		Use:   "iam",
		Short: "Inspect IAM roles and policies",
		Long: `List and dump IAM roles and managed policies.

Requires IAM read permissions on the caller principal, for example:
iam:ListRoles, iam:ListPolicies, iam:GetRole, iam:GetPolicy,
iam:GetPolicyVersion, iam:ListAttachedRolePolicies, iam:ListRolePolicies,
and iam:GetRolePolicy.`,
		Example: `  # List customer-managed policies
  aws-probe iam --list-policies

  # List all roles as JSON
  aws-probe iam --list-roles --output json

  # Dump a managed policy document and granted services
  aws-probe iam --dump-policy aws-probe-s3-read-only

  # Dump a role trust policy and attached/inline policies
  aws-probe iam --dump-role aws-probe-s3-read-only --output json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !listRoles && !listPolicies && dumpPolicy == "" && dumpRole == "" {
				return errors.New(
					"an action flag is required: use one of --list-roles, --list-policies, --dump-policy, --dump-role",
				)
			}

			cfg, err := PrepareAWSConfig(cmd.Context())
			if err != nil {
				return err
			}

			client := iam.NewClient(cfg)
			out := cmd.OutOrStdout()

			switch {
			case listRoles:
				return iam.ListRoles(cmd.Context(), client, out, outputFormat, theme)
			case listPolicies:
				return iam.ListPolicies(cmd.Context(), client, scope, out, outputFormat, theme)
			case dumpPolicy != "":
				return iam.DumpPolicy(cmd.Context(), client, dumpPolicy, out, outputFormat, theme)
			default:
				return iam.DumpRole(cmd.Context(), client, dumpRole, out, outputFormat, theme)
			}
		},
	}

	cmd.Flags().BoolVar(&listRoles, "list-roles", false, "List IAM roles")
	cmd.Flags().BoolVar(&listPolicies, "list-policies", false, "List IAM managed policies")
	cmd.Flags().StringVar(&dumpPolicy, "dump-policy", "", "Dump a managed policy by name or ARN")
	cmd.Flags().StringVar(&dumpRole, "dump-role", "", "Dump an IAM role by name or ARN")
	cmd.Flags().StringVar(&scope, "scope", "Local",
		"Policy scope for --list-policies (Local, AWS, All)")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, csv, json)")
	cmd.Flags().StringVarP(&theme, "theme", "t", "catppuccin-frappe",
		"Output theme (catppuccin-frappe, dracula, nord, none)")

	cmd.MarkFlagsMutuallyExclusive("list-roles", "list-policies", "dump-policy", "dump-role")

	return cmd
}
