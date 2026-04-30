package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/service/kafka"
	kafkatypes "github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/spf13/cobra"
	"github.com/xenos76/aws-probe/internal/service"
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
	cmd.AddCommand(newProduceCmd())
	cmd.AddCommand(newConsumeCmd())

	return cmd
}

var (
	mskBrokers    string
	mskClusterArn string
	mskTopic      string
	mskAuth       string
	mskTLS        bool
	mskMessage    string
	mskKey        string
	mskGroup      string
	mskFromStart  bool
	mskAcks       int16
	mskVerbose    bool
)

func addCommonKafkaFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&mskBrokers, "brokers", "", "Comma-separated list of Kafka brokers")
	cmd.Flags().StringVar(&mskClusterArn, "cluster-arn", "", "MSK cluster ARN to fetch brokers from")
	cmd.Flags().StringVar(&mskTopic, "topic", "", "Kafka topic")
	cmd.Flags().StringVar(&mskAuth, "auth", "iam", "Authentication method (iam, none)")
	cmd.Flags().BoolVar(&mskTLS, "tls", true, "Enable TLS (ignored if auth is iam)")
	cmd.Flags().Int16Var(&mskAcks, "acks", -1, "Required acks (-1=all, 0=none, 1=one)")
	cmd.Flags().BoolVar(&mskVerbose, "verbose", false, "Enable verbose logging")
}

func newProduceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "produce [topic] [message]",
		Short: "Produce a message to a Kafka topic",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				mskTopic = args[0]
			}

			if len(args) > 1 {
				mskMessage = args[1]
			}

			if len(args) > 2 {
				mskKey = args[2]
			}

			if mskTopic == "" {
				return errors.New("topic is required")
			}

			if mskMessage == "" {
				return errors.New("message is required")
			}

			return runProduce(cmd.Context())
		},
	}

	addCommonKafkaFlags(cmd)
	cmd.Flags().StringVar(&mskMessage, "message", "", "Message to produce")
	cmd.Flags().StringVar(&mskKey, "key", "", "Key for the message")

	return cmd
}

func newConsumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consume [topic]",
		Short: "Consume messages from a Kafka topic",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				mskTopic = args[0]
			}

			if mskTopic == "" {
				return errors.New("topic is required")
			}

			return runConsume(cmd.Context())
		},
	}

	addCommonKafkaFlags(cmd)
	cmd.Flags().StringVar(&mskGroup, "group", "", "Consumer group ID")
	cmd.Flags().BoolVar(&mskFromStart, "from-beginning", false, "Start consuming from the beginning")

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
	cfg, err := PrepareAWSConfig(ctx)
	if err != nil {
		return err
	}

	return listClusters(ctx, kafka.NewFromConfig(cfg))
}

// runListTopics executes the list-topics command.
func runListTopics(ctx context.Context, clusterArn string) error {
	cfg, err := PrepareAWSConfig(ctx)
	if err != nil {
		return err
	}

	return listTopics(ctx, clusterArn, kafka.NewFromConfig(cfg))
}

// listClusters lists MSK clusters using the provided API client.
func listClusters(ctx context.Context, api kafkaClustersLister) error {
	var allClusters []kafkatypes.Cluster

	input := &kafka.ListClustersV2Input{}
	for {
		output, err := api.ListClustersV2(ctx, input)
		if err != nil {
			return fmt.Errorf("listing MSK clusters: %w", err)
		}

		allClusters = append(allClusters, output.ClusterInfoList...)

		if output.NextToken == nil || *output.NextToken == "" {
			break
		}

		input.NextToken = output.NextToken
	}

	if len(allClusters) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "No MSK clusters found.")

		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprint(tw, "CLUSTER NAME\tARN\tSTATUS\n")

	for _, cluster := range allClusters {
		fmt.Fprintf(tw, "%s\t%s\t%s\n",
			derefString(cluster.ClusterName),
			derefString(cluster.ClusterArn),
			cluster.State,
		)
	}

	return tw.Flush()
}

// listTopics lists MSK topics for a given cluster using the provided API client.
func listTopics(ctx context.Context, clusterArn string, api kafkaTopicsLister) error {
	var allTopics []kafkatypes.TopicInfo

	input := &kafka.ListTopicsInput{
		ClusterArn: &clusterArn,
	}

	for {
		output, err := api.ListTopics(ctx, input)
		if err != nil {
			return fmt.Errorf("listing MSK topics: %w", err)
		}

		allTopics = append(allTopics, output.Topics...)

		if output.NextToken == nil || *output.NextToken == "" {
			break
		}

		input.NextToken = output.NextToken
	}

	if len(allTopics) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "No topics found.")

		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprint(tw, "TOPIC NAME\tPARTITIONS\tREPLICATION\n")

	for _, topic := range allTopics {
		fmt.Fprintf(tw, "%s\t%d\t%d\n",
			derefString(topic.TopicName),
			derefInt32(topic.PartitionCount),
			derefInt32(topic.ReplicationFactor),
		)
	}

	return tw.Flush()
}

func runProduce(ctx context.Context) error {
	cfg, err := PrepareAWSConfig(ctx)
	if err != nil {
		return err
	}

	brokers, err := resolveBrokers(ctx, kafka.NewFromConfig(cfg))
	if err != nil {
		return err
	}

	logger := slog.Default()
	if mskVerbose {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	svc := service.NewKafkaService(cfg, logger)
	kcfg := service.KafkaConfig{
		Brokers: brokers,
		Topic:   mskTopic,
		Auth:    mskAuth,
		UseTLS:  mskTLS,
		Acks:    mskAcks,
	}

	var key []byte
	if mskKey != "" {
		key = []byte(mskKey)
	}

	return svc.Produce(ctx, kcfg, key, []byte(mskMessage))
}

func runConsume(ctx context.Context) error {
	cfg, err := PrepareAWSConfig(ctx)
	if err != nil {
		return err
	}

	brokers, err := resolveBrokers(ctx, kafka.NewFromConfig(cfg))
	if err != nil {
		return err
	}

	logger := slog.Default()

	if mskVerbose {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	svc := service.NewKafkaService(cfg, logger)

	kcfg := service.KafkaConfig{
		Brokers:       brokers,
		Topic:         mskTopic,
		Auth:          mskAuth,
		UseTLS:        mskTLS,
		Acks:          mskAcks,
		Group:         mskGroup,
		FromBeginning: mskFromStart,
	}

	return svc.Consume(ctx, kcfg, func(r *service.Record) {
		fmt.Printf("Partition: %d | Offset: %d | Key: %s | Value: %s\n",
			r.Partition, r.Offset, string(r.Key), string(r.Value))
	})
}

func resolveBrokers(ctx context.Context, api kafkaBrokersGetter) ([]string, error) {
	if mskBrokers != "" {
		return strings.Split(mskBrokers, ","), nil
	}

	if mskClusterArn == "" {
		return nil, errors.New("either --brokers or --cluster-arn is required")
	}

	input := &kafka.GetBootstrapBrokersInput{
		ClusterArn: &mskClusterArn,
	}

	result, err := api.GetBootstrapBrokers(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("getting bootstrap brokers: %w", err)
	}

	var brokerString string

	switch mskAuth {
	case "iam":
		if result.BootstrapBrokerStringSaslIam == nil {
			return nil, errors.New("cluster does not support IAM authentication")
		}

		brokerString = *result.BootstrapBrokerStringSaslIam
	default:
		if mskTLS {
			if result.BootstrapBrokerStringTls == nil {
				return nil, errors.New("cluster does not support TLS")
			}

			brokerString = *result.BootstrapBrokerStringTls
		} else {
			if result.BootstrapBrokerString == nil {
				return nil, errors.New("cluster does not support plaintext")
			}

			brokerString = *result.BootstrapBrokerString
		}
	}

	return strings.Split(brokerString, ","), nil
}
