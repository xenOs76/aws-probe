package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/spf13/cobra"
)

var validResources = []string{"secrets", "sqs", "s3", "msk-clusters", "msk-topics"}

type secretsListAPI interface {
	ListSecrets(
		ctx context.Context,
		params *secretsmanager.ListSecretsInput,
		optFns ...func(*secretsmanager.Options),
	) (*secretsmanager.ListSecretsOutput, error)
}

type sqsListAPI interface {
	ListQueues(
		ctx context.Context,
		params *sqs.ListQueuesInput,
		optFns ...func(*sqs.Options),
	) (*sqs.ListQueuesOutput, error)
}

type s3ListAPI interface {
	ListBuckets(
		ctx context.Context,
		params *s3.ListBucketsInput,
		optFns ...func(*s3.Options),
	) (*s3.ListBucketsOutput, error)
}

type kafkaListClustersAPI interface {
	ListClustersV2(
		ctx context.Context,
		params *kafka.ListClustersV2Input,
		optFns ...func(*kafka.Options),
	) (*kafka.ListClustersV2Output, error)
}

type kafkaListTopicsAPI interface {
	ListTopics(
		ctx context.Context,
		params *kafka.ListTopicsInput,
		optFns ...func(*kafka.Options),
	) (*kafka.ListTopicsOutput, error)
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "list [resource] [args...]",
		Short:     "List AWS resources",
		Long:      `List AWS resources by type. Supported resources: secrets, sqs, s3, msk-clusters, msk-topics.`,
		ValidArgs: validResources,
		Args:      cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				printAvailableResources()

				return nil
			}

			return executeList(cmd.Context(), args)
		},
	}
}

func printAvailableResources() {
	fmt.Fprintln(os.Stderr, "Available resources:")

	for _, r := range validResources {
		fmt.Fprintf(os.Stderr, "  • %s\n", r)
	}

	fmt.Fprintln(os.Stderr, "\nUsage: aws-probe list <resource> [args]")
}

func executeList(ctx context.Context, args []string) error {
	resource := args[0]

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("loading AWS config: %w", err)
	}

	switch strings.ToLower(resource) {
	case "secrets":
		return listSecrets(ctx, secretsmanager.NewFromConfig(cfg))
	case "sqs":
		return listQueues(ctx, sqs.NewFromConfig(cfg))
	case "s3":
		return listBuckets(ctx, s3.NewFromConfig(cfg))
	case "msk-clusters":
		return listMSKClusters(ctx, kafka.NewFromConfig(cfg))
	case "msk-topics":
		if len(args) < 2 {
			return fmt.Errorf("msk-topics requires a cluster ARN as second argument\nUsage: aws-probe list msk-topics <cluster-arn>")
		}

		return listMSKTopics(ctx, args[1], kafka.NewFromConfig(cfg))
	default:
		return fmt.Errorf("unknown resource %q, valid options: %s", resource, strings.Join(validResources, ", "))
	}
}

func listSecrets(ctx context.Context, api secretsListAPI) error {
	output, err := api.ListSecrets(ctx, &secretsmanager.ListSecretsInput{})
	if err != nil {
		if isCredentialError(err) {
			fmt.Fprint(os.Stderr, noCredentialsMessage)

			return nil
		}

		return fmt.Errorf("listing secrets: %w", err)
	}

	if len(output.SecretList) == 0 {
		fmt.Fprintln(os.Stderr, "No secrets found.")

		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprint(tw, "NAME\tARN\n")

	for _, secret := range output.SecretList {
		fmt.Fprintf(tw, "%s\t%s\n", derefString(secret.Name), derefString(secret.ARN))
	}

	return tw.Flush()
}

func listQueues(ctx context.Context, api sqsListAPI) error {
	output, err := api.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err != nil {
		if isCredentialError(err) {
			fmt.Fprint(os.Stderr, noCredentialsMessage)

			return nil
		}

		return fmt.Errorf("listing SQS queues: %w", err)
	}

	if len(output.QueueUrls) == 0 {
		fmt.Fprintln(os.Stderr, "No SQS queues found.")

		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprint(tw, "QUEUE URL\n")

	for _, url := range output.QueueUrls {
		fmt.Fprintf(tw, "%s\n", url)
	}

	return tw.Flush()
}

func listBuckets(ctx context.Context, api s3ListAPI) error {
	output, err := api.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		if isCredentialError(err) {
			fmt.Fprint(os.Stderr, noCredentialsMessage)

			return nil
		}

		return fmt.Errorf("listing S3 buckets: %w", err)
	}

	if len(output.Buckets) == 0 {
		fmt.Fprintln(os.Stderr, "No S3 buckets found.")

		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprint(tw, "NAME\tCREATED\n")

	for _, bucket := range output.Buckets {
		created := ""
		if bucket.CreationDate != nil {
			created = bucket.CreationDate.Format("2006-01-02 15:04:05")
		}

		fmt.Fprintf(tw, "%s\t%s\n", derefString(bucket.Name), created)
	}

	return tw.Flush()
}

func listMSKClusters(ctx context.Context, api kafkaListClustersAPI) error {
	output, err := api.ListClustersV2(ctx, &kafka.ListClustersV2Input{})
	if err != nil {
		if isCredentialError(err) {
			fmt.Fprint(os.Stderr, noCredentialsMessage)

			return nil
		}

		return fmt.Errorf("listing MSK clusters: %w", err)
	}

	if len(output.ClusterInfoList) == 0 {
		fmt.Fprintln(os.Stderr, "No MSK clusters found.")

		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprint(tw, "CLUSTER NAME\tARN\tSTATUS\n")

	for _, cluster := range output.ClusterInfoList {
		fmt.Fprintf(tw, "%s\t%s\t%s\n",
			derefString(cluster.ClusterName),
			derefString(cluster.ClusterArn),
			cluster.State,
		)
	}

	return tw.Flush()
}

func listMSKTopics(ctx context.Context, clusterArn string, api kafkaListTopicsAPI) error {
	output, err := api.ListTopics(ctx, &kafka.ListTopicsInput{
		ClusterArn: &clusterArn,
	})
	if err != nil {
		if isCredentialError(err) {
			fmt.Fprint(os.Stderr, noCredentialsMessage)

			return nil
		}

		return fmt.Errorf("listing MSK topics: %w", err)
	}

	if len(output.Topics) == 0 {
		fmt.Fprintln(os.Stderr, "No topics found.")

		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprint(tw, "TOPIC NAME\tPARTITIONS\tREPLICATION\n")

	for _, topic := range output.Topics {
		fmt.Fprintf(tw, "%s\t%d\t%d\n",
			derefString(topic.TopicName),
			derefInt32(topic.PartitionCount),
			derefInt32(topic.ReplicationFactor),
		)
	}

	return tw.Flush()
}

func derefInt32(i *int32) int32 {
	if i == nil {
		return 0
	}

	return *i
}
