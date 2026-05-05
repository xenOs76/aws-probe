package msk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	kafkatypes "github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/xenos76/aws-probe/internal/awsutil"
	internalkafka "github.com/xenos76/aws-probe/internal/kafka"
)

// ClustersLister defines the interface for listing MSK clusters.
type ClustersLister interface {
	ListClustersV2(
		ctx context.Context,
		params *kafka.ListClustersV2Input,
		optFns ...func(*kafka.Options),
	) (*kafka.ListClustersV2Output, error)
}

// TopicsLister defines the interface for listing MSK topics.
type TopicsLister interface {
	ListTopics(
		ctx context.Context,
		params *kafka.ListTopicsInput,
		optFns ...func(*kafka.Options),
	) (*kafka.ListTopicsOutput, error)
}

// BrokersGetter defines the interface for getting bootstrap brokers.
type BrokersGetter interface {
	GetBootstrapBrokers(
		ctx context.Context,
		params *kafka.GetBootstrapBrokersInput,
		optFns ...func(*kafka.Options),
	) (*kafka.GetBootstrapBrokersOutput, error)
}

// ProduceConfig holds configuration for producing a message.
type ProduceConfig struct {
	Brokers []string
	Topic   string
	Auth    string
	UseTLS  bool
	Acks    int16
	Key     string
	Message string
	Verbose bool
}

// ConsumeConfig holds configuration for consuming messages.
type ConsumeConfig struct {
	Brokers       []string
	Topic         string
	Auth          string
	UseTLS        bool
	Acks          int16
	Group         string
	FromBeginning bool
	Verbose       bool
}

// ListClusters lists MSK clusters using the provided API client.
func ListClusters(ctx context.Context, api ClustersLister, w io.Writer) error {
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
		fmt.Fprintln(w, "No MSK clusters found.")

		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	fmt.Fprint(tw, "CLUSTER NAME\tARN\tSTATUS\n")

	for _, cluster := range allClusters {
		fmt.Fprintf(tw, "%s\t%s\t%s\n",
			awsutil.DerefString(cluster.ClusterName),
			awsutil.DerefString(cluster.ClusterArn),
			cluster.State,
		)
	}

	return tw.Flush()
}

// ListTopics lists MSK topics for a given cluster using the provided API client.
func ListTopics(ctx context.Context, clusterArn string, api TopicsLister, w io.Writer) error {
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
		fmt.Fprintln(w, "No topics found.")

		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	fmt.Fprint(tw, "TOPIC NAME\tPARTITIONS\tREPLICATION\n")

	for _, topic := range allTopics {
		fmt.Fprintf(tw, "%s\t%d\t%d\n",
			awsutil.DerefString(topic.TopicName),
			awsutil.DerefInt32(topic.PartitionCount),
			awsutil.DerefInt32(topic.ReplicationFactor),
		)
	}

	return tw.Flush()
}

// ResolveBrokers resolves Kafka brokers from explicit list or MSK cluster ARN.
func ResolveBrokers(
	ctx context.Context,
	brokers, clusterArn, auth string,
	useTLS bool,
	api BrokersGetter,
) ([]string, error) {
	if brokers != "" {
		return strings.Split(brokers, ","), nil
	}

	if clusterArn == "" {
		return nil, errors.New("either --brokers or --cluster-arn is required")
	}

	result, err := api.GetBootstrapBrokers(ctx, &kafka.GetBootstrapBrokersInput{
		ClusterArn: &clusterArn,
	})
	if err != nil {
		return nil, fmt.Errorf("getting bootstrap brokers: %w", err)
	}

	return selectBrokerString(result, auth, useTLS)
}

// revive:disable:flag-parameter
func selectBrokerString(result *kafka.GetBootstrapBrokersOutput, auth string, useTLS bool) ([]string, error) {
	var brokerString string

	switch auth {
	case "iam":
		if result.BootstrapBrokerStringSaslIam == nil {
			return nil, errors.New("cluster does not support IAM authentication")
		}

		brokerString = *result.BootstrapBrokerStringSaslIam
	default:
		if useTLS {
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
	// revive:enable:flag-parameter

	return strings.Split(brokerString, ","), nil
}

// Produce sends a message to a Kafka topic.
func Produce(ctx context.Context, cfg aws.Config, pcfg ProduceConfig, w io.Writer) error {
	logger := buildLogger(pcfg.Verbose, w)
	svc := internalkafka.NewService(cfg, logger)

	kcfg := internalkafka.Config{
		Brokers: pcfg.Brokers,
		Topic:   pcfg.Topic,
		Auth:    pcfg.Auth,
		UseTLS:  pcfg.UseTLS,
		Acks:    pcfg.Acks,
	}

	var key []byte
	if pcfg.Key != "" {
		key = []byte(pcfg.Key)
	}

	return svc.Produce(ctx, kcfg, key, []byte(pcfg.Message))
}

// Consume reads messages from a Kafka topic.
func Consume(ctx context.Context, cfg aws.Config, ccfg ConsumeConfig, w io.Writer) error {
	logger := buildLogger(ccfg.Verbose, w)
	svc := internalkafka.NewService(cfg, logger)

	kcfg := internalkafka.Config{
		Brokers:       ccfg.Brokers,
		Topic:         ccfg.Topic,
		Auth:          ccfg.Auth,
		UseTLS:        ccfg.UseTLS,
		Acks:          ccfg.Acks,
		Group:         ccfg.Group,
		FromBeginning: ccfg.FromBeginning,
	}

	return svc.Consume(ctx, kcfg, func(r *internalkafka.Record) {
		fmt.Fprintf(w, "Partition: %d | Offset: %d | Key: %s | Value: %s\n",
			r.Partition, r.Offset, string(r.Key), string(r.Value))
	})
}

// NewClient creates a new Kafka/MSK client.
func NewClient(cfg aws.Config) *kafka.Client {
	return kafka.NewFromConfig(cfg)
}

// revive:disable:flag-parameter
func buildLogger(verbose bool, w io.Writer) *slog.Logger {
	if verbose {
		return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	return slog.Default()
	// revive:enable:flag-parameter
}
