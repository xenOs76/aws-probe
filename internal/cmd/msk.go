package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/xenos76/aws-probe/internal/msk"
)

type mskOptions struct {
	listClusters bool
	listTopics   bool
	produce      bool
	consume      bool

	brokers    string
	clusterARN string
	topic      string
	auth       string
	tls        bool
	message    string
	key        string
	group      string
	fromStart  bool
	acks       int16
	verbose    bool
}

// newMskCmd creates the MSK command.
func newMskCmd() *cobra.Command {
	opts := mskOptions{}

	cmd := &cobra.Command{
		Use:   "msk",
		Short: "Manage MSK resources",
		Long:  `List MSK clusters and topics, produce and consume messages.`,
		Example: `  # List all clusters
  aws-probe msk --list-clusters

  # List topics for a cluster
  aws-probe msk --list-topics --cluster-arn <cluster-arn>

  # Produce a message
  aws-probe msk --produce --topic <topic> --message <msg> --cluster-arn <arn>

  # Consume messages
  aws-probe msk --consume --topic <topic> --cluster-arn <arn> --from-beginning`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleMskRun(cmd, args, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.listClusters, "list-clusters", false, "List all MSK clusters")
	cmd.Flags().BoolVar(&opts.listTopics, "list-topics", false, "List topics for the specified cluster")
	cmd.Flags().BoolVar(&opts.produce, "produce", false, "Produce a message to a topic")
	cmd.Flags().BoolVar(&opts.consume, "consume", false, "Consume messages from a topic")

	addCommonKafkaFlags(cmd, &opts)
	cmd.Flags().StringVar(&opts.message, "message", "", "Message to produce")
	cmd.Flags().StringVar(&opts.key, "key", "", "Key for the message")
	cmd.Flags().StringVar(&opts.group, "group", "", "Consumer group ID")
	cmd.Flags().BoolVar(&opts.fromStart, "from-beginning", false, "Start consuming from the beginning")

	cmd.MarkFlagsMutuallyExclusive("list-clusters", "list-topics", "produce", "consume")

	return cmd
}

func addCommonKafkaFlags(cmd *cobra.Command, opts *mskOptions) {
	cmd.Flags().StringVar(&opts.brokers, "brokers", "", "Comma-separated list of Kafka brokers")
	cmd.Flags().StringVar(&opts.clusterARN, "cluster-arn", "", "MSK cluster ARN to fetch brokers from")
	cmd.Flags().StringVar(&opts.topic, "topic", "", "Kafka topic")
	cmd.Flags().StringVar(&opts.auth, "auth", "iam", "Authentication method (iam, none)")
	cmd.Flags().BoolVar(&opts.tls, "tls", true, "Enable TLS (ignored if auth is iam)")
	cmd.Flags().Int16Var(&opts.acks, "acks", -1, "Required acks (-1=all, 0=none, 1=one)")
	cmd.Flags().BoolVar(&opts.verbose, "verbose", false, "Enable verbose logging")
}

// handleMskRun handles the execution logic for the MSK command.
func handleMskRun(cmd *cobra.Command, args []string, opts mskOptions) error {
	if err := validateMSKOptions(args, opts); err != nil {
		return err
	}

	ctx := cmd.Context()
	out := cmd.OutOrStdout()

	return runMSKAction(ctx, cmd, out, opts)
}

func handleMskProduce(ctx context.Context, out io.Writer, opts mskOptions) error {
	cfg, err := PrepareAWSConfig(ctx)
	if err != nil {
		return fmt.Errorf("preparing AWS config: %w", err)
	}

	brokers, err := msk.ResolveBrokers(ctx, opts.brokers, opts.clusterARN, opts.auth, opts.tls, msk.NewClient(cfg))
	if err != nil {
		return err
	}

	return msk.Produce(ctx, cfg, msk.ProduceConfig{
		Brokers: brokers,
		Topic:   opts.topic,
		Auth:    opts.auth,
		UseTLS:  opts.tls,
		Acks:    opts.acks,
		Key:     opts.key,
		Message: opts.message,
		Verbose: opts.verbose,
	}, out)
}

func handleMskConsume(ctx context.Context, out io.Writer, opts mskOptions) error {
	cfg, err := PrepareAWSConfig(ctx)
	if err != nil {
		return fmt.Errorf("preparing AWS config: %w", err)
	}

	brokers, err := msk.ResolveBrokers(ctx, opts.brokers, opts.clusterARN, opts.auth, opts.tls, msk.NewClient(cfg))
	if err != nil {
		return err
	}

	return msk.Consume(ctx, cfg, msk.ConsumeConfig{
		Brokers:       brokers,
		Topic:         opts.topic,
		Auth:          opts.auth,
		UseTLS:        opts.tls,
		Acks:          opts.acks,
		Group:         opts.group,
		FromBeginning: opts.fromStart,
		Verbose:       opts.verbose,
	}, out)
}

func validateMSKOptions(args []string, opts mskOptions) error {
	if len(args) > 0 {
		return errors.New("positional arguments are not supported; use flags such as --topic and --message")
	}

	if err := validateMSKActionSelection(opts); err != nil {
		return err
	}

	if opts.auth != "iam" && opts.auth != "none" {
		return fmt.Errorf("invalid --auth value %q (allowed: iam, none)", opts.auth)
	}

	if opts.acks != -1 && opts.acks != 0 && opts.acks != 1 {
		return fmt.Errorf("invalid --acks value %d (allowed: -1, 0, 1)", opts.acks)
	}

	if opts.produce {
		if opts.topic == "" {
			return errors.New("produce mode requires --topic")
		}

		if opts.message == "" {
			return errors.New("produce mode requires --message")
		}
	}

	if opts.consume && opts.topic == "" {
		return errors.New("consume mode requires --topic")
	}

	if opts.listTopics && opts.clusterARN == "" {
		return errors.New("list-topics mode requires --cluster-arn")
	}

	return nil
}

func validateMSKActionSelection(opts mskOptions) error {
	actionSelected := opts.listClusters || opts.listTopics || opts.produce || opts.consume
	if actionSelected || !hasMSKParameterFlags(opts) {
		return nil
	}

	return errors.New("an action flag is required: use one of --list-clusters, --list-topics, --produce, --consume")
}

func hasMSKParameterFlags(opts mskOptions) bool {
	return opts.brokers != "" ||
		opts.clusterARN != "" ||
		opts.topic != "" ||
		opts.auth != "iam" ||
		!opts.tls ||
		opts.message != "" ||
		opts.key != "" ||
		opts.group != "" ||
		opts.fromStart ||
		opts.acks != -1 ||
		opts.verbose
}

func runMSKAction(ctx context.Context, cmd *cobra.Command, out io.Writer, opts mskOptions) error {
	if opts.listClusters || opts.listTopics {
		cfg, err := PrepareAWSConfig(ctx)
		if err != nil {
			return fmt.Errorf("preparing AWS config: %w", err)
		}

		client := msk.NewClient(cfg)
		if opts.listClusters {
			return msk.ListClusters(ctx, client, out)
		}

		return msk.ListTopics(ctx, opts.clusterARN, client, out)
	}

	if opts.produce {
		return handleMskProduce(ctx, out, opts)
	}

	if opts.consume {
		return handleMskConsume(ctx, out, opts)
	}

	return cmd.Help()
}
