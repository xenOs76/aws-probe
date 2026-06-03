package mcp

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xenos76/aws-probe/internal/awsutil"
	internalkafka "github.com/xenos76/aws-probe/internal/kafka"
	internalmsk "github.com/xenos76/aws-probe/internal/msk"
)

const (
	defaultConsumeTimeout    = 30 * time.Second
	defaultConsumeMaxRecords = 10
)

var (
	errMSKClusterARNRequired = errors.New("clusterArn is required")
	errMSKTopicRequired      = errors.New("topic is required")
)

type mskClusterEntry struct {
	Name   string `json:"name"`
	ARN    string `json:"arn"`
	Status string `json:"status"`
}

type listMSKClustersOutput struct {
	Clusters []mskClusterEntry `json:"clusters"`
}

type listMSKTopicsInput struct {
	ClusterARN string `json:"clusterArn" jsonschema:"MSK cluster ARN"`
}

type mskTopicEntry struct {
	Name              string `json:"name"`
	PartitionCount    int32  `json:"partitionCount"`
	ReplicationFactor int32  `json:"replicationFactor"`
}

type listMSKTopicsOutput struct {
	Topics []mskTopicEntry `json:"topics"`
}

type mskConsumeInput struct {
	ClusterARN    string `json:"clusterArn,omitempty" jsonschema:"MSK cluster ARN (used to resolve brokers)"`
	Brokers       string `json:"brokers,omitempty" jsonschema:"Comma-separated broker list (alternative to clusterArn)"`
	Topic         string `json:"topic" jsonschema:"Kafka topic name"`
	Group         string `json:"group,omitempty" jsonschema:"Consumer group ID"`
	FromBeginning bool   `json:"fromBeginning,omitempty" jsonschema:"Start from beginning of topic"`
	Auth          string `json:"auth,omitempty" jsonschema:"Authentication: iam or none (default iam)"`
	TLS           *bool  `json:"tls,omitempty" jsonschema:"Enable TLS (default true)"`
	TimeoutSec    int    `json:"timeoutSec,omitempty" jsonschema:"Max seconds to poll (default 30)"`
	MaxRecords    int    `json:"maxRecords,omitempty" jsonschema:"Max records to return (default 10)"`
}

type kafkaRecord struct {
	Topic     string `json:"topic"`
	Partition int32  `json:"partition"`
	Offset    int64  `json:"offset"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

type mskConsumeOutput struct {
	Records []kafkaRecord `json:"records"`
}

func registerMSKTools(server *mcp.Server, deps *Deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "aws_probe_msk_list_clusters",
		Description: "List MSK clusters in the account",
	}, mskListClustersHandler(deps))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "aws_probe_msk_list_topics",
		Description: "List topics for an MSK cluster",
	}, mskListTopicsHandler(deps))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "aws_probe_msk_consume",
		Description: "Consume messages from an MSK/Kafka topic (bounded by timeout and maxRecords)",
	}, mskConsumeHandler(deps))
}

func mskListClustersHandler(deps *Deps) func(
	context.Context, *mcp.CallToolRequest, struct{},
) (*mcp.CallToolResult, listMSKClustersOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (
		*mcp.CallToolResult, listMSKClustersOutput, error,
	) {
		cfg, err := loadAWS(ctx, deps)
		if err != nil {
			return nil, listMSKClustersOutput{}, err
		}

		clusters, err := listMSKClustersForMCP(ctx, internalmsk.NewClient(cfg))
		if err != nil {
			return nil, listMSKClustersOutput{}, err
		}

		return nil, listMSKClustersOutput{Clusters: clusters}, nil
	}
}

func listMSKClustersForMCP(ctx context.Context, client internalmsk.ClustersLister) ([]mskClusterEntry, error) {
	clusters := make([]mskClusterEntry, 0)

	input := &kafka.ListClustersV2Input{}
	for {
		out, err := client.ListClustersV2(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("listing MSK clusters: %w", err)
		}

		for _, c := range out.ClusterInfoList {
			clusters = append(clusters, mskClusterEntry{
				Name:   awsutil.DerefString(c.ClusterName),
				ARN:    awsutil.DerefString(c.ClusterArn),
				Status: string(c.State),
			})
		}

		if out.NextToken == nil || *out.NextToken == "" {
			break
		}

		input.NextToken = out.NextToken
	}

	return clusters, nil
}

func mskListTopicsHandler(deps *Deps) func(
	context.Context, *mcp.CallToolRequest, listMSKTopicsInput,
) (*mcp.CallToolResult, listMSKTopicsOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in listMSKTopicsInput) (
		*mcp.CallToolResult, listMSKTopicsOutput, error,
	) {
		if in.ClusterARN == "" {
			return nil, listMSKTopicsOutput{}, errMSKClusterARNRequired
		}

		cfg, err := loadAWS(ctx, deps)
		if err != nil {
			return nil, listMSKTopicsOutput{}, err
		}

		topics, err := listMSKTopicsForMCP(ctx, internalmsk.NewClient(cfg), in.ClusterARN)
		if err != nil {
			return nil, listMSKTopicsOutput{}, err
		}

		return nil, listMSKTopicsOutput{Topics: topics}, nil
	}
}

func listMSKTopicsForMCP(
	ctx context.Context,
	client internalmsk.TopicsLister,
	clusterARN string,
) ([]mskTopicEntry, error) {
	topics := make([]mskTopicEntry, 0)

	input := &kafka.ListTopicsInput{ClusterArn: &clusterARN}
	for {
		out, err := client.ListTopics(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("listing MSK topics: %w", err)
		}

		for _, t := range out.Topics {
			topics = append(topics, mskTopicEntry{
				Name:              awsutil.DerefString(t.TopicName),
				PartitionCount:    awsutil.DerefInt32(t.PartitionCount),
				ReplicationFactor: awsutil.DerefInt32(t.ReplicationFactor),
			})
		}

		if out.NextToken == nil || *out.NextToken == "" {
			break
		}

		input.NextToken = out.NextToken
	}

	return topics, nil
}

func mskConsumeHandler(deps *Deps) func(
	context.Context, *mcp.CallToolRequest, mskConsumeInput,
) (*mcp.CallToolResult, mskConsumeOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in mskConsumeInput) (
		*mcp.CallToolResult, mskConsumeOutput, error,
	) {
		opts, err := parseMSKConsumeInput(in)
		if err != nil {
			return nil, mskConsumeOutput{}, err
		}

		cfg, err := loadAWS(ctx, deps)
		if err != nil {
			return nil, mskConsumeOutput{}, err
		}

		mskClient := internalmsk.NewClient(cfg)

		brokers, err := internalmsk.ResolveBrokers(
			ctx, in.Brokers, in.ClusterARN, opts.auth, opts.useTLS, mskClient,
		)
		if err != nil {
			return nil, mskConsumeOutput{}, err
		}

		consumeCtx, cancel := context.WithTimeout(ctx, opts.timeout)
		defer cancel()

		records, err := consumeBounded(consumeCtx, cfg, internalkafka.Config{
			Brokers:       brokers,
			Topic:         in.Topic,
			Auth:          opts.auth,
			UseTLS:        opts.useTLS,
			Group:         in.Group,
			FromBeginning: in.FromBeginning,
		}, opts.maxRecords)
		if err != nil {
			return nil, mskConsumeOutput{}, err
		}

		return nil, mskConsumeOutput{Records: records}, nil
	}
}

type mskConsumeOptions struct {
	auth       string
	useTLS     bool
	timeout    time.Duration
	maxRecords int
}

func parseMSKConsumeInput(in mskConsumeInput) (mskConsumeOptions, error) {
	if in.Topic == "" {
		return mskConsumeOptions{}, errMSKTopicRequired
	}

	auth := in.Auth
	if auth == "" {
		auth = "iam"
	}

	if auth != "iam" && auth != "none" {
		return mskConsumeOptions{}, fmt.Errorf("invalid auth %q (allowed: iam, none)", auth)
	}

	useTLS := true
	if in.TLS != nil {
		useTLS = *in.TLS
	}

	if auth == "iam" {
		useTLS = true
	}

	timeout := defaultConsumeTimeout
	if in.TimeoutSec > 0 {
		timeout = time.Duration(in.TimeoutSec) * time.Second
	}

	maxRecords := defaultConsumeMaxRecords
	if in.MaxRecords > 0 {
		maxRecords = in.MaxRecords
	}

	return mskConsumeOptions{
		auth:       auth,
		useTLS:     useTLS,
		timeout:    timeout,
		maxRecords: maxRecords,
	}, nil
}

func consumeBounded(ctx context.Context, cfg aws.Config, kcfg internalkafka.Config, maxRecords int) (
	[]kafkaRecord, error,
) {
	svc := internalkafka.NewService(cfg, nil)

	var (
		collected []kafkaRecord
		mu        sync.Mutex
	)

	consumeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- svc.Consume(consumeCtx, kcfg, func(r *internalkafka.Record) {
			mu.Lock()
			defer mu.Unlock()

			if len(collected) >= maxRecords {
				cancel()

				return
			}

			collected = append(collected, kafkaRecord{
				Topic:     r.Topic,
				Partition: r.Partition,
				Offset:    r.Offset,
				Key:       string(r.Key),
				Value:     string(r.Value),
			})

			if len(collected) >= maxRecords {
				cancel()
			}
		})
	}()

	err := <-errCh
	if err != nil && consumeCtx.Err() == nil {
		return nil, fmt.Errorf("consume: %w", err)
	}

	if len(collected) == 0 && ctx.Err() != nil {
		return nil, fmt.Errorf("consume: %w", ctx.Err())
	}

	return collected, nil
}
