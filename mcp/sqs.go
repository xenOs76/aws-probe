package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	internalsqs "github.com/xenos76/aws-probe/internal/sqs"
)

const maxSQSReceiveMessages = 10

var (
	errSQSQueueNameRequired = errors.New("queueName is required")
	errSQSQueueURLRequired  = errors.New("queueUrl is required")
)

type listSQSQueuesOutput struct {
	QueueURLs []string `json:"queueUrls"`
}

type getQueueURLInput struct {
	QueueName string `json:"queueName" jsonschema:"SQS queue name"`
}

type getQueueURLOutput struct {
	QueueURL string `json:"queueUrl"`
}

type receiveMessageInput struct {
	QueueURL    string `json:"queueUrl" jsonschema:"Full SQS queue URL"`
	MaxMessages int    `json:"maxMessages,omitempty" jsonschema:"Maximum messages to receive (1-10)"`
}

type sqsMessage struct {
	MessageID     string `json:"messageId"`
	Body          string `json:"body"`
	ReceiptHandle string `json:"receiptHandle,omitempty"`
}

type receiveMessageOutput struct {
	Messages []sqsMessage `json:"messages"`
}

func registerSQSTools(server *mcp.Server, deps *Deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "aws_probe_sqs_list_queues",
		Description: "List all SQS queue URLs in the account",
	}, sqsListQueuesHandler(deps))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "aws_probe_sqs_get_queue_url",
		Description: "Resolve an SQS queue name to its URL",
	}, sqsGetQueueURLHandler(deps))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "aws_probe_sqs_receive_message",
		Description: "Receive messages from an SQS queue (does not delete messages; capped batch size)",
	}, sqsReceiveMessageHandler(deps))
}

func sqsListQueuesHandler(deps *Deps) func(
	context.Context, *mcp.CallToolRequest, struct{},
) (*mcp.CallToolResult, listSQSQueuesOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (
		*mcp.CallToolResult, listSQSQueuesOutput, error,
	) {
		cfg, err := loadAWS(ctx, deps)
		if err != nil {
			return nil, listSQSQueuesOutput{}, err
		}

		urls, err := listSQSQueueURLs(ctx, internalsqs.NewClient(cfg))
		if err != nil {
			return nil, listSQSQueuesOutput{}, err
		}

		return nil, listSQSQueuesOutput{QueueURLs: urls}, nil
	}
}

func listSQSQueueURLs(ctx context.Context, client internalsqs.Lister) ([]string, error) {
	var urls []string

	input := &sqs.ListQueuesInput{}
	for {
		out, err := client.ListQueues(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("listing SQS queues: %w", err)
		}

		urls = append(urls, out.QueueUrls...)

		if out.NextToken == nil || *out.NextToken == "" {
			break
		}

		input.NextToken = out.NextToken
	}

	return urls, nil
}

func sqsGetQueueURLHandler(deps *Deps) func(
	context.Context, *mcp.CallToolRequest, getQueueURLInput,
) (*mcp.CallToolResult, getQueueURLOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in getQueueURLInput) (
		*mcp.CallToolResult, getQueueURLOutput, error,
	) {
		if in.QueueName == "" {
			return nil, getQueueURLOutput{}, errSQSQueueNameRequired
		}

		cfg, err := loadAWS(ctx, deps)
		if err != nil {
			return nil, getQueueURLOutput{}, err
		}

		url, err := getSQSQueueURL(ctx, internalsqs.NewClient(cfg), in.QueueName)
		if err != nil {
			return nil, getQueueURLOutput{}, err
		}

		return nil, getQueueURLOutput{QueueURL: url}, nil
	}
}

func getSQSQueueURL(ctx context.Context, client internalsqs.QueueURLGetter, queueName string) (string, error) {
	out, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return "", fmt.Errorf("getting queue URL: %w", err)
	}

	return aws.ToString(out.QueueUrl), nil
}

func sqsReceiveMessageHandler(deps *Deps) func(
	context.Context, *mcp.CallToolRequest, receiveMessageInput,
) (*mcp.CallToolResult, receiveMessageOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in receiveMessageInput) (
		*mcp.CallToolResult, receiveMessageOutput, error,
	) {
		if in.QueueURL == "" {
			return nil, receiveMessageOutput{}, errSQSQueueURLRequired
		}

		maxMessages := clampSQSMaxMessages(in.MaxMessages)

		cfg, err := loadAWS(ctx, deps)
		if err != nil {
			return nil, receiveMessageOutput{}, err
		}

		messages, err := receiveSQSMessages(ctx, internalsqs.NewClient(cfg), in.QueueURL, maxMessages)
		if err != nil {
			return nil, receiveMessageOutput{}, err
		}

		return nil, receiveMessageOutput{Messages: messages}, nil
	}
}

func clampSQSMaxMessages(requested int) int32 {
	if requested <= 0 {
		return 1
	}

	if requested > maxSQSReceiveMessages {
		return maxSQSReceiveMessages
	}

	return int32(requested)
}

func receiveSQSMessages(
	ctx context.Context,
	client internalsqs.MessageReceiver,
	queueURL string,
	maxMessages int32,
) ([]sqsMessage, error) {
	out, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: maxMessages,
	})
	if err != nil {
		return nil, fmt.Errorf("receiving messages: %w", err)
	}

	messages := make([]sqsMessage, 0, len(out.Messages))
	for _, m := range out.Messages {
		messages = append(messages, sqsMessage{
			MessageID:     aws.ToString(m.MessageId),
			Body:          aws.ToString(m.Body),
			ReceiptHandle: aws.ToString(m.ReceiptHandle),
		})
	}

	return messages, nil
}
