//nolint:dupl // CLI handlers follow a similar pattern
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xenos76/aws-probe/internal/sns"
)

// newSnsCmd creates the SNS command.
//
//nolint:dupl // CLI handlers follow a similar pattern
func newSnsCmd() *cobra.Command {
	var (
		listTopicsFlag        bool
		listSubscriptionsFlag string
	)

	cmd := &cobra.Command{
		Use:   "sns",
		Short: "Manage SNS resources",
		Long:  `List SNS topics and subscriptions in the current AWS account.`,
		Example: `  # List all topics
  aws-probe sns --list-topics

  # List subscriptions for a topic
  aws-probe sns --list-subscriptions <topic-arn>`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !listTopicsFlag && listSubscriptionsFlag == "" {
				return cmd.Help()
			}

			cfg, err := PrepareAWSConfig(cmd.Context())
			if err != nil {
				return err
			}

			client := sns.NewClient(cfg)

			if listTopicsFlag {
				return sns.ListTopics(cmd.Context(), client, cmd.OutOrStdout())
			}

			return sns.ListSubscriptions(cmd.Context(), listSubscriptionsFlag, client, cmd.OutOrStdout())
		},
	}

	cmd.Flags().BoolVar(&listTopicsFlag, "list-topics", false, "List all SNS topics")
	cmd.Flags().StringVar(&listSubscriptionsFlag, "list-subscriptions", "",
		"List subscriptions for the specified topic ARN")

	cmd.MarkFlagsMutuallyExclusive("list-topics", "list-subscriptions")

	return cmd
}
