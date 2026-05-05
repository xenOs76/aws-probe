package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xenos76/aws-probe/internal/sqs"
)

// newSqsCmd creates the SQS command.
func newSqsCmd() *cobra.Command {
	var listQueuesFlag bool

	cmd := &cobra.Command{
		Use:   "sqs",
		Short: "Manage SQS queues",
		Long:  `List SQS queues in the current AWS account.`,
		Example: `  # List all queues
  aws-probe sqs --list-queues`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !listQueuesFlag {
				return cmd.Help()
			}

			cfg, err := PrepareAWSConfig(cmd.Context())
			if err != nil {
				return err
			}

			client := sqs.NewClient(cfg)

			return sqs.ListQueues(cmd.Context(), client, cmd.OutOrStdout())
		},
	}

	cmd.Flags().BoolVar(&listQueuesFlag, "list-queues", false, "List all SQS queues")

	return cmd
}
