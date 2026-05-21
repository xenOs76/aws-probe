package cmd

import (
	"github.com/spf13/cobra"
	"github.com/xenos76/aws-probe/internal/sqs"
)

// newSqsCmd returns the cobra command for SQS probes. Exactly one of
// --list-queues, --get-queue-url <name>, or --receive-message <queue-url> must
// be set; without any, RunE prints help. Operations are mutually exclusive.
func newSqsCmd() *cobra.Command {
	var (
		listQueuesFlag     bool
		getQueueURLFlag    string
		receiveMessageFlag string
	)

	cmd := &cobra.Command{
		Use:   "sqs",
		Short: "Manage SQS queues",
		Long:  `List SQS queues, resolve queue URLs, and receive queue messages.`,
		Example: `  # List all queues
  aws-probe sqs --list-queues

  # Get queue URL by queue name
  aws-probe sqs --get-queue-url my-queue

  # Receive messages from queue URL
  aws-probe sqs --receive-message https://sqs.us-east-1.amazonaws.com/123456789012/my-queue`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !listQueuesFlag && getQueueURLFlag == "" && receiveMessageFlag == "" {
				return cmd.Help()
			}

			cfg, err := PrepareAWSConfig(cmd.Context())
			if err != nil {
				return err
			}

			client := sqs.NewClient(cfg)

			if listQueuesFlag {
				return sqs.ListQueues(cmd.Context(), client, cmd.OutOrStdout())
			}

			if getQueueURLFlag != "" {
				return sqs.GetQueueURL(cmd.Context(), client, getQueueURLFlag, cmd.OutOrStdout())
			}

			return sqs.ReceiveMessage(cmd.Context(), client, receiveMessageFlag, cmd.OutOrStdout())
		},
	}

	cmd.Flags().BoolVar(&listQueuesFlag, "list-queues", false, "List all SQS queues")
	cmd.Flags().StringVar(&getQueueURLFlag, "get-queue-url", "", "Get queue URL for the specified queue name")
	cmd.Flags().StringVar(&receiveMessageFlag, "receive-message", "", "Receive messages from the specified queue URL")
	cmd.MarkFlagsMutuallyExclusive("list-queues", "get-queue-url", "receive-message")

	return cmd
}
