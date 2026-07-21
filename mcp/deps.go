package mcp

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/xenos76/aws-probe/internal/awsutil"
	internalcf "github.com/xenos76/aws-probe/internal/cloudfront"
	internaliam "github.com/xenos76/aws-probe/internal/iam"
	internalmsk "github.com/xenos76/aws-probe/internal/msk"
	internals3 "github.com/xenos76/aws-probe/internal/s3"
	internalsecrets "github.com/xenos76/aws-probe/internal/secrets"
	internalsns "github.com/xenos76/aws-probe/internal/sns"
	internalsqs "github.com/xenos76/aws-probe/internal/sqs"
	"github.com/xenos76/aws-probe/internal/whoami"
)

// Config holds optional defaults for MCP tool behavior.
type Config struct {
	Region string
	Output string // table, json, csv (reserved for future formatted tools)
	Theme  string
}

// s3API is the S3 surface used by MCP tools.
type s3API interface {
	internals3.BucketsLister
	internals3.ObjectsLister
	internals3.ObjectHeader
}

// sqsAPI is the SQS surface used by MCP tools.
type sqsAPI interface {
	internalsqs.Lister
	internalsqs.QueueURLGetter
	internalsqs.MessageReceiver
}

// secretsAPI is the Secrets Manager surface used by MCP list tools.
type secretsAPI interface {
	internalsecrets.Lister
}

// mskAPI is the MSK surface used by MCP tools.
type mskAPI interface {
	internalmsk.ClustersLister
	internalmsk.TopicsLister
	internalmsk.BrokersGetter
}

// iamAPI is the IAM surface used by MCP tools.
type iamAPI interface {
	internaliam.RolesLister
	internaliam.PoliciesLister
	internaliam.PolicyDumper
	internaliam.RoleDumper
}

// Deps bundles dependencies for MCP registration and tool handlers.
type Deps struct {
	LoadConfig func(context.Context) (aws.Config, error)
	Config     Config

	// Optional factories for unit tests. Nil uses production SDK clients.
	S3         func(aws.Config) s3API
	SQS        func(aws.Config) sqsAPI
	Secrets    func(aws.Config) secretsAPI
	MSK        func(aws.Config) mskAPI
	IAM        func(aws.Config) iamAPI
	SNS        func(aws.Config) internalsns.TopicsLister
	STS        func(aws.Config) whoami.STSAPI
	CloudFront func(aws.Config) (internalcf.ClientAPI, internalcf.ACMClientAPI)
}

// DefaultLoadConfig returns the standard aws-probe AWS config loader with credential checks.
func DefaultLoadConfig() func(context.Context) (aws.Config, error) {
	return func(ctx context.Context) (aws.Config, error) {
		return awsutil.PrepareAWSConfig(ctx)
	}
}

// withDefaults fills zero Deps fields with production defaults.
func (d *Deps) withDefaults() *Deps {
	if d == nil {
		d = &Deps{}
	}

	out := *d
	if out.LoadConfig == nil {
		out.LoadConfig = DefaultLoadConfig()
	}

	return &out
}

func (d *Deps) s3(cfg aws.Config) s3API {
	if d != nil && d.S3 != nil {
		return d.S3(cfg)
	}

	return internals3.NewClient(cfg)
}

func (d *Deps) sqs(cfg aws.Config) sqsAPI {
	if d != nil && d.SQS != nil {
		return d.SQS(cfg)
	}

	return internalsqs.NewClient(cfg)
}

func (d *Deps) secrets(cfg aws.Config) secretsAPI {
	if d != nil && d.Secrets != nil {
		return d.Secrets(cfg)
	}

	return internalsecrets.NewClient(cfg)
}

func (d *Deps) msk(cfg aws.Config) mskAPI {
	if d != nil && d.MSK != nil {
		return d.MSK(cfg)
	}

	return internalmsk.NewClient(cfg)
}

func (d *Deps) iam(cfg aws.Config) iamAPI {
	if d != nil && d.IAM != nil {
		return d.IAM(cfg)
	}

	return internaliam.NewClient(cfg)
}

func (d *Deps) sns(cfg aws.Config) internalsns.TopicsLister {
	if d != nil && d.SNS != nil {
		return d.SNS(cfg)
	}

	return internalsns.NewClient(cfg)
}

func (d *Deps) sts(cfg aws.Config) whoami.STSAPI {
	if d != nil && d.STS != nil {
		return d.STS(cfg)
	}

	return whoami.NewSTSClient(cfg)
}

func (d *Deps) cloudFront(cfg aws.Config) (internalcf.ClientAPI, internalcf.ACMClientAPI) {
	if d != nil && d.CloudFront != nil {
		return d.CloudFront(cfg)
	}

	return internalcf.NewClient(cfg), internalcf.NewACMClient(cfg)
}
