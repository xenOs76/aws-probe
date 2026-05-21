// Package sqs implements read-only SQS operations used by the aws-probe CLI.
// Callers pass an io.Writer for human-readable tabular output; AWS errors are
// wrapped with context. Narrow interfaces (Lister, QueueURLGetter, MessageReceiver)
// allow unit tests to mock the AWS SDK without pulling in the real client.
package sqs

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// Lister is the subset of the AWS SQS API needed for ListQueues pagination.
type Lister interface {
	ListQueues(
		ctx context.Context,
		params *sqs.ListQueuesInput,
		optFns ...func(*sqs.Options),
	) (*sqs.ListQueuesOutput, error)
}

// QueueURLGetter is the subset of the AWS SQS API needed for GetQueueUrl.
// Method name GetQueueUrl matches the aws-sdk-go-v2 client surface.
type QueueURLGetter interface {
	GetQueueUrl(
		ctx context.Context,
		params *sqs.GetQueueUrlInput,
		optFns ...func(*sqs.Options),
	) (*sqs.GetQueueUrlOutput, error)
}

// MessageReceiver is the subset of the AWS SQS API needed for ReceiveMessage.
type MessageReceiver interface {
	ReceiveMessage(
		ctx context.Context,
		params *sqs.ReceiveMessageInput,
		optFns ...func(*sqs.Options),
	) (*sqs.ReceiveMessageOutput, error)
}

// ListQueues collects all queue URLs by following NextToken until exhausted.
// It writes a tabwriter table with header "QUEUE URL", or "No SQS queues found." if empty.
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

// GetQueueURL resolves queueName to a URL via GetQueueUrl and prints a single
// "QUEUE URL" section followed by the URL line.
func GetQueueURL(ctx context.Context, api QueueURLGetter, queueName string, w io.Writer) error {
	output, err := api.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return fmt.Errorf("getting SQS queue URL: %w", err)
	}

	if _, err := fmt.Fprint(w, "QUEUE URL\n"); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%s\n", aws.ToString(output.QueueUrl)); err != nil {
		return err
	}

	return nil
}

// ReceiveMessage pulls one ReceiveMessage batch (SDK defaults) for queueURL.
// With no messages it prints "No SQS messages found."; otherwise it writes a
// tabwriter table with columns MESSAGE ID and BODY.
func ReceiveMessage(ctx context.Context, api MessageReceiver, queueURL string, w io.Writer) error {
	output, err := api.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl: aws.String(queueURL),
	})
	if err != nil {
		return fmt.Errorf("receiving SQS message: %w", err)
	}

	if len(output.Messages) == 0 {
		fmt.Fprintln(w, "No SQS messages found.")

		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	fmt.Fprint(tw, "MESSAGE ID\tBODY\n")

	for _, message := range output.Messages {
		safeBody := strings.ReplaceAll(aws.ToString(message.Body), "\n", "\\n")
		safeBody = strings.ReplaceAll(safeBody, "\t", "\\t")
		fmt.Fprintf(tw, "%s\t%s\n", aws.ToString(message.MessageId), safeBody)
	}

	return tw.Flush()
}

// NewClient returns an AWS SDK v2 SQS client built from cfg.
func NewClient(cfg aws.Config) *sqs.Client {
	return sqs.NewFromConfig(cfg)
}
