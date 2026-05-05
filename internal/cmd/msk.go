package cmd

import (
	"context"
	"errors"
	"io"

	"github.com/spf13/cobra"
	"github.com/xenos76/aws-probe/internal/msk"
)

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

// newMskCmd creates the MSK command.
func newMskCmd() *cobra.Command {
	var (
		listClustersFlag bool
		listTopicsFlag   string
		produceFlag      bool
		consumeFlag      bool
	)

	cmd := &cobra.Command{
		Use:   "msk",
		Short: "Manage MSK resources",
		Long:  `List MSK clusters and topics, produce and consume messages.`,
		Example: `  # List all clusters
  aws-probe msk --list-clusters

  # List topics for a cluster
  aws-probe msk --list-topics <cluster-arn>

  # Produce a message
  aws-probe msk --produce --topic <topic> --message <msg> --cluster-arn <arn>

  # Consume messages
  aws-probe msk --consume --topic <topic> --cluster-arn <arn> --from-beginning`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleMskRun(cmd, args, listClustersFlag, listTopicsFlag, produceFlag, consumeFlag)
		},
	}

	cmd.Flags().BoolVar(&listClustersFlag, "list-clusters", false, "List all MSK clusters")
	cmd.Flags().StringVar(&listTopicsFlag, "list-topics", "", "List topics for the specified cluster ARN")
	cmd.Flags().BoolVar(&produceFlag, "produce", false, "Produce a message to a topic")
	cmd.Flags().BoolVar(&consumeFlag, "consume", false, "Consume messages from a topic")

	addCommonKafkaFlags(cmd)
	cmd.Flags().StringVar(&mskMessage, "message", "", "Message to produce")
	cmd.Flags().StringVar(&mskKey, "key", "", "Key for the message")
	cmd.Flags().StringVar(&mskGroup, "group", "", "Consumer group ID")
	cmd.Flags().BoolVar(&mskFromStart, "from-beginning", false, "Start consuming from the beginning")

	cmd.MarkFlagsMutuallyExclusive("list-clusters", "list-topics", "produce", "consume")

	return cmd
}

func addCommonKafkaFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&mskBrokers, "brokers", "", "Comma-separated list of Kafka brokers")
	cmd.Flags().StringVar(&mskClusterArn, "cluster-arn", "", "MSK cluster ARN to fetch brokers from")
	cmd.Flags().StringVar(&mskTopic, "topic", "", "Kafka topic")
	cmd.Flags().StringVar(&mskAuth, "auth", "iam", "Authentication method (iam, none)")
	cmd.Flags().BoolVar(&mskTLS, "tls", true, "Enable TLS (ignored if auth is iam)")
	cmd.Flags().Int16Var(&mskAcks, "acks", -1, "Required acks (-1=all, 0=none, 1=one)")
	cmd.Flags().BoolVar(&mskVerbose, "verbose", false, "Enable verbose logging")
}

// handleMskRun handles the execution logic for the MSK command.
//
//nolint:revive // listClusters is a control flag derived from CLI arguments
func handleMskRun(
	cmd *cobra.Command,
	args []string,
	listClusters bool,
	listTopics string,
	produce bool,
	consume bool,
) error {
	ctx := cmd.Context()
	out := cmd.OutOrStdout()

	if listClusters || listTopics != "" {
		cfg, err := PrepareAWSConfig(ctx)
		if err != nil {
			return err
		}

		client := msk.NewClient(cfg)
		if listClusters {
			return msk.ListClusters(ctx, client, out)
		}

		return msk.ListTopics(ctx, listTopics, client, out)
	}

	if produce {
		return handleMskProduce(ctx, args, out)
	}

	if consume {
		return handleMskConsume(ctx, args, out)
	}

	return cmd.Help()
}

func handleMskProduce(ctx context.Context, args []string, out io.Writer) error {
	topic := mskTopic
	message := mskMessage
	key := mskKey

	if len(args) > 0 {
		topic = args[0]
	}

	if len(args) > 1 {
		message = args[1]
	}

	if len(args) > 2 {
		key = args[2]
	}

	if topic == "" {
		return errors.New("topic is required")
	}

	if message == "" {
		return errors.New("message is required")
	}

	cfg, err := PrepareAWSConfig(ctx)
	if err != nil {
		return err
	}

	brokers, err := msk.ResolveBrokers(ctx, mskBrokers, mskClusterArn, mskAuth, mskTLS, msk.NewClient(cfg))
	if err != nil {
		return err
	}

	return msk.Produce(ctx, cfg, msk.ProduceConfig{
		Brokers: brokers,
		Topic:   topic,
		Auth:    mskAuth,
		UseTLS:  mskTLS,
		Acks:    mskAcks,
		Key:     key,
		Message: message,
		Verbose: mskVerbose,
	}, out)
}

func handleMskConsume(ctx context.Context, args []string, out io.Writer) error {
	topic := mskTopic

	if len(args) > 0 {
		topic = args[0]
	}

	if topic == "" {
		return errors.New("topic is required")
	}

	cfg, err := PrepareAWSConfig(ctx)
	if err != nil {
		return err
	}

	brokers, err := msk.ResolveBrokers(ctx, mskBrokers, mskClusterArn, mskAuth, mskTLS, msk.NewClient(cfg))
	if err != nil {
		return err
	}

	return msk.Consume(ctx, cfg, msk.ConsumeConfig{
		Brokers:       brokers,
		Topic:         topic,
		Auth:          mskAuth,
		UseTLS:        mskTLS,
		Acks:          mskAcks,
		Group:         mskGroup,
		FromBeginning: mskFromStart,
		Verbose:       mskVerbose,
	}, out)
}
