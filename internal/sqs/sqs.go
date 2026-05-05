package sqs

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// Lister defines the interface for listing SQS queues.
type Lister interface {
	ListQueues(
		ctx context.Context,
		params *sqs.ListQueuesInput,
		optFns ...func(*sqs.Options),
	) (*sqs.ListQueuesOutput, error)
}

// ListQueues lists SQS queues using the provided API client.
func ListQueues(ctx context.Context, api Lister, w io.Writer) error {
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
		fmt.Fprintln(w, "No SQS queues found.")

		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	fmt.Fprint(tw, "QUEUE URL\n")

	for _, url := range allQueueUrls {
		fmt.Fprintf(tw, "%s\n", url)
	}

	return tw.Flush()
}

// NewClient creates a new SQS client from the provided AWS configuration.
func NewClient(cfg aws.Config) *sqs.Client {
	return sqs.NewFromConfig(cfg)
}
