package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/spf13/cobra"
)

// newMskCmd creates the MSK command.
func newMskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "msk",
		Short: "List MSK clusters and topics",
		Long:  `List MSK clusters and topics in the current AWS account.`,
	}

	cmd.AddCommand(newListClustersCmd())
	cmd.AddCommand(newListTopicsCmd())

	return cmd
}

// newListClustersCmd creates the list-clusters subcommand.
func newListClustersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-clusters",
		Short: "List MSK clusters",
		Long:  `List all MSK clusters in the current AWS account.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runListClusters(cmd.Context())
		},
	}
}

// newListTopicsCmd creates the list-topics subcommand.
func newListTopicsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-topics [cluster-arn]",
		Short: "List MSK topics",
		Long:  `List topics for an MSK cluster. Requires cluster ARN as argument.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListTopics(cmd.Context(), args[0])
		},
	}
}

// runListClusters executes the list-clusters command.
func runListClusters(ctx context.Context) error {
	if err := EnsureCredentials(); err != nil {
		return err
	}

	cfg, err := LoadAWSConfig(ctx)
	if err != nil {
		return fmt.Errorf("loading AWS config: %w", err)
	}

	return listClusters(ctx, kafka.NewFromConfig(cfg))
}

// runListTopics executes the list-topics command.
func runListTopics(ctx context.Context, clusterArn string) error {
	if err := EnsureCredentials(); err != nil {
		return err
	}

	cfg, err := LoadAWSConfig(ctx)
	if err != nil {
		return fmt.Errorf("loading AWS config: %w", err)
	}

	return listTopics(ctx, clusterArn, kafka.NewFromConfig(cfg))
}

// listClusters lists MSK clusters using the provided API client.
func listClusters(ctx context.Context, api kafkaListClustersAPI) error {
	output, err := api.ListClustersV2(ctx, &kafka.ListClustersV2Input{})
	if err != nil {
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

// listTopics lists MSK topics for a given cluster using the provided API client.
func listTopics(ctx context.Context, clusterArn string, api kafkaListTopicsAPI) error {
	output, err := api.ListTopics(ctx, &kafka.ListTopicsInput{
		ClusterArn: &clusterArn,
	})
	if err != nil {
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
