package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/spf13/cobra"
)

// newSqsCmd creates the SQS command.
func newSqsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sqs",
		Short: "List SQS queues",
		Long:  `List SQS queues in the current AWS account.`,
	}

	cmd.AddCommand(newListQueuesCmd())

	return cmd
}

// newListQueuesCmd creates the list-queues subcommand.
func newListQueuesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-queues",
		Short: "List SQS queues",
		Long:  `List all SQS queues in the current AWS account.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runListQueues(cmd.Context())
		},
	}
}

// runListQueues executes the list-queues command.
func runListQueues(ctx context.Context) error {
	if err := EnsureCredentials(); err != nil {
		return err
	}

	cfg, err := LoadAWSConfig(ctx)
	if err != nil {
		return fmt.Errorf("loading AWS config: %w", err)
	}

	return listQueues(ctx, sqs.NewFromConfig(cfg))
}

// listQueues lists SQS queues using the provided API client.
func listQueues(ctx context.Context, api sqsListAPI) error {
	var allQueueUrls []string

	input := &sqs.ListQueuesInput{}

	for {
		output, err := api.ListQueues(ctx, input)
		if err != nil {
			return fmt.Errorf("listing SQS queues: %w", err)
		}

		allQueueUrls = append(allQueueUrls, output.QueueUrls...)

		if output.NextToken == nil || *output.NextToken == "" {
			break
		}

		input.NextToken = output.NextToken
	}

	if len(allQueueUrls) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "No SQS queues found.")

		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprint(tw, "QUEUE URL\n")

	for _, url := range allQueueUrls {
		fmt.Fprintf(tw, "%s\n", url)
	}

	return tw.Flush()
}
