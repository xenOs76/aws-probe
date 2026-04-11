package cmd

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// secretsListAPI defines the interface for listing secrets.
type secretsListAPI interface {
	ListSecrets(
		ctx context.Context,
		params *secretsmanager.ListSecretsInput,
		optFns ...func(*secretsmanager.Options),
	) (*secretsmanager.ListSecretsOutput, error)
}

// sqsListAPI defines the interface for listing SQS queues.
type sqsListAPI interface {
	ListQueues(
		ctx context.Context,
		params *sqs.ListQueuesInput,
		optFns ...func(*sqs.Options),
	) (*sqs.ListQueuesOutput, error)
}

// s3ListAPI defines the interface for listing S3 buckets.
type s3ListAPI interface {
	ListBuckets(
		ctx context.Context,
		params *s3.ListBucketsInput,
		optFns ...func(*s3.Options),
	) (*s3.ListBucketsOutput, error)
}

// kafkaListClustersAPI defines the interface for listing MSK clusters.
type kafkaListClustersAPI interface {
	ListClustersV2(
		ctx context.Context,
		params *kafka.ListClustersV2Input,
		optFns ...func(*kafka.Options),
	) (*kafka.ListClustersV2Output, error)
}

// kafkaListTopicsAPI defines the interface for listing MSK topics.
type kafkaListTopicsAPI interface {
	ListTopics(
		ctx context.Context,
		params *kafka.ListTopicsInput,
		optFns ...func(*kafka.Options),
	) (*kafka.ListTopicsOutput, error)
}

// derefInt32 dereferences an int32 pointer, returning 0 if nil.
func derefInt32(i *int32) int32 {
	if i == nil {
		return 0
	}

	return *i
}
