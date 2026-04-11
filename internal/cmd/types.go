package cmd

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

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

func derefInt32(i *int32) int32 {
	if i == nil {
		return 0
	}

	return *i
}
