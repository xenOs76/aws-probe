package cmd

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// secretsLister defines the interface for listing secrets.
type secretsLister interface {
	ListSecrets(
		ctx context.Context,
		params *secretsmanager.ListSecretsInput,
		optFns ...func(*secretsmanager.Options),
	) (*secretsmanager.ListSecretsOutput, error)
}

// sqsLister defines the interface for listing SQS queues.
type sqsLister interface {
	ListQueues(
		ctx context.Context,
		params *sqs.ListQueuesInput,
		optFns ...func(*sqs.Options),
	) (*sqs.ListQueuesOutput, error)
}

// s3Lister defines the interface for listing S3 buckets.
type s3Lister interface {
	ListBuckets(
		ctx context.Context,
		params *s3.ListBucketsInput,
		optFns ...func(*s3.Options),
	) (*s3.ListBucketsOutput, error)
}

// s3ObjectsLister defines the interface for listing S3 objects.
type s3ObjectsLister interface {
	ListObjectsV2(
		ctx context.Context,
		params *s3.ListObjectsV2Input,
		optFns ...func(*s3.Options),
	) (*s3.ListObjectsV2Output, error)
}

// s3ObjectHeader defines the interface for getting S3 object metadata.
type s3ObjectHeader interface {
	HeadObject(
		ctx context.Context,
		params *s3.HeadObjectInput,
		optFns ...func(*s3.Options),
	) (*s3.HeadObjectOutput, error)
}

// kmsKeyDescriber defines the interface for describing KMS keys.
type kmsKeyDescriber interface {
	DescribeKey(
		ctx context.Context,
		params *kms.DescribeKeyInput,
		optFns ...func(*kms.Options),
	) (*kms.DescribeKeyOutput, error)
}

// kmsAliasesLister defines the interface for listing KMS aliases.
type kmsAliasesLister interface {
	ListAliases(
		ctx context.Context,
		params *kms.ListAliasesInput,
		optFns ...func(*kms.Options),
	) (*kms.ListAliasesOutput, error)
}

// kafkaClustersLister defines the interface for listing MSK clusters.
type kafkaClustersLister interface {
	ListClustersV2(
		ctx context.Context,
		params *kafka.ListClustersV2Input,
		optFns ...func(*kafka.Options),
	) (*kafka.ListClustersV2Output, error)
}

// kafkaTopicsLister defines the interface for listing MSK topics.
type kafkaTopicsLister interface {
	ListTopics(
		ctx context.Context,
		params *kafka.ListTopicsInput,
		optFns ...func(*kafka.Options),
	) (*kafka.ListTopicsOutput, error)
}

// kafkaBrokersGetter defines the interface for getting bootstrap brokers.
type kafkaBrokersGetter interface {
	GetBootstrapBrokers(
		ctx context.Context,
		params *kafka.GetBootstrapBrokersInput,
		optFns ...func(*kafka.Options),
	) (*kafka.GetBootstrapBrokersOutput, error)
}

// snsTopicsLister defines the interface for listing SNS topics.
type snsTopicsLister interface {
	ListTopics(
		ctx context.Context,
		params *sns.ListTopicsInput,
		optFns ...func(*sns.Options),
	) (*sns.ListTopicsOutput, error)
}

// snsSubscriptionsLister defines the interface for listing SNS subscriptions.
type snsSubscriptionsLister interface {
	ListSubscriptionsByTopic(
		ctx context.Context,
		params *sns.ListSubscriptionsByTopicInput,
		optFns ...func(*sns.Options),
	) (*sns.ListSubscriptionsByTopicOutput, error)
}

// derefInt32 dereferences an int32 pointer, returning 0 if nil.
func derefInt32(i *int32) int32 {
	if i == nil {
		return 0
	}

	return *i
}

// derefInt64 dereferences an int64 pointer, returning 0 if nil.
func derefInt64(i *int64) int64 {
	if i == nil {
		return 0
	}

	return *i
}
