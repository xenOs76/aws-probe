package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xenos76/aws-probe/internal/awsutil"
	internals3 "github.com/xenos76/aws-probe/internal/s3"
)

var (
	errS3BucketRequired    = errors.New("bucket is required")
	errS3BucketKeyRequired = errors.New("bucket and key are required")
)

type listBucketsOutput struct {
	Buckets []bucketEntry `json:"buckets"`
}

type bucketEntry struct {
	Name    string `json:"name"`
	Created string `json:"created,omitempty"`
}

type listObjectsInput struct {
	Bucket    string `json:"bucket" jsonschema:"S3 bucket name"`
	Prefix    string `json:"prefix,omitempty" jsonschema:"Key prefix"`
	Recursive bool   `json:"recursive,omitempty" jsonschema:"List recursively without delimiter"`
}

type objectEntry struct {
	Key          string `json:"key"`
	LastModified string `json:"lastModified,omitempty"`
	Size         string `json:"size"`
	IsPrefix     bool   `json:"isPrefix,omitempty"`
}

type listObjectsOutput struct {
	Objects []objectEntry `json:"objects"`
}

type getObjectMetadataInput struct {
	Bucket string `json:"bucket" jsonschema:"S3 bucket name"`
	Key    string `json:"key" jsonschema:"Object key"`
}

type objectMetadataOutput struct {
	Key              string            `json:"key"`
	Size             string            `json:"size"`
	ETag             string            `json:"etag,omitempty"`
	ContentType      string            `json:"contentType,omitempty"`
	StorageClass     string            `json:"storageClass,omitempty"`
	LastModified     string            `json:"lastModified,omitempty"`
	ServerEncryption string            `json:"serverSideEncryption,omitempty"`
	VersionID        string            `json:"versionId,omitempty"`
	CustomMetadata   map[string]string `json:"customMetadata,omitempty"`
}

func registerS3Tools(server *mcp.Server, deps *Deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "aws_probe_s3_list_buckets",
		Description: "List all S3 buckets in the account",
	}, s3ListBucketsHandler(deps))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "aws_probe_s3_list_objects",
		Description: "List objects and common prefixes in an S3 bucket",
	}, s3ListObjectsHandler(deps))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "aws_probe_s3_get_object_metadata",
		Description: "Return metadata for an S3 object (HeadObject)",
	}, s3GetObjectMetadataHandler(deps))
}

func s3ListBucketsHandler(deps *Deps) func(
	context.Context, *mcp.CallToolRequest, struct{},
) (*mcp.CallToolResult, listBucketsOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (
		*mcp.CallToolResult, listBucketsOutput, error,
	) {
		cfg, err := loadAWS(ctx, deps)
		if err != nil {
			return nil, listBucketsOutput{}, err
		}

		entries, err := listS3Buckets(ctx, internals3.NewClient(cfg))
		if err != nil {
			return nil, listBucketsOutput{}, err
		}

		return nil, listBucketsOutput{Buckets: entries}, nil
	}
}

func listS3Buckets(ctx context.Context, client internals3.BucketsLister) ([]bucketEntry, error) {
	out, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("listing S3 buckets: %w", err)
	}

	entries := make([]bucketEntry, 0, len(out.Buckets))
	for _, b := range out.Buckets {
		created := ""
		if b.CreationDate != nil {
			created = b.CreationDate.Format(time.RFC3339)
		}

		entries = append(entries, bucketEntry{
			Name:    awsutil.DerefString(b.Name),
			Created: created,
		})
	}

	return entries, nil
}

func s3ListObjectsHandler(deps *Deps) func(
	context.Context, *mcp.CallToolRequest, listObjectsInput,
) (*mcp.CallToolResult, listObjectsOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in listObjectsInput) (
		*mcp.CallToolResult, listObjectsOutput, error,
	) {
		if in.Bucket == "" {
			return nil, listObjectsOutput{}, errS3BucketRequired
		}

		cfg, err := loadAWS(ctx, deps)
		if err != nil {
			return nil, listObjectsOutput{}, err
		}

		delimiter := "/"

		var delimPtr *string
		if !in.Recursive {
			delimPtr = &delimiter
		}

		client := internals3.NewClient(cfg)

		objects, err := collectS3Objects(ctx, client, in.Bucket, in.Prefix, delimPtr)
		if err != nil {
			return nil, listObjectsOutput{}, err
		}

		return nil, listObjectsOutput{Objects: objects}, nil
	}
}

func s3GetObjectMetadataHandler(deps *Deps) func(
	context.Context, *mcp.CallToolRequest, getObjectMetadataInput,
) (*mcp.CallToolResult, objectMetadataOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in getObjectMetadataInput) (
		*mcp.CallToolResult, objectMetadataOutput, error,
	) {
		if in.Bucket == "" || in.Key == "" {
			return nil, objectMetadataOutput{}, errS3BucketKeyRequired
		}

		cfg, err := loadAWS(ctx, deps)
		if err != nil {
			return nil, objectMetadataOutput{}, err
		}

		out, err := headS3Object(ctx, internals3.NewClient(cfg), in.Bucket, in.Key)
		if err != nil {
			return nil, objectMetadataOutput{}, err
		}

		return nil, out, nil
	}
}

func headS3Object(
	ctx context.Context,
	client internals3.ObjectHeader,
	bucket, key string,
) (objectMetadataOutput, error) {
	head, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return objectMetadataOutput{}, fmt.Errorf("head object: %w", err)
	}

	out := objectMetadataOutput{
		Key:            key,
		Size:           internals3.FormatSize(awsutil.DerefInt64(head.ContentLength)),
		ETag:           strings.Trim(awsutil.DerefString(head.ETag), `"`),
		ContentType:    awsutil.DerefString(head.ContentType),
		StorageClass:   string(head.StorageClass),
		CustomMetadata: head.Metadata,
	}

	if head.LastModified != nil {
		out.LastModified = head.LastModified.Format(time.RFC3339)
	}

	if head.ServerSideEncryption != "" {
		out.ServerEncryption = string(head.ServerSideEncryption)
	}

	out.VersionID = awsutil.DerefString(head.VersionId)

	return out, nil
}

func collectS3Objects(
	ctx context.Context,
	api internals3.ObjectsLister,
	bucket, prefix string,
	delimiter *string,
) ([]objectEntry, error) {
	input := &s3.ListObjectsV2Input{Bucket: aws.String(bucket)}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	input.Delimiter = delimiter

	paginator := s3.NewListObjectsV2Paginator(api, input)

	objects := make([]objectEntry, 0)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing S3 objects: %w", err)
		}

		for _, cp := range page.CommonPrefixes {
			objects = append(objects, objectEntry{
				Key:      internals3.StripPrefix(awsutil.DerefString(cp.Prefix), prefix),
				Size:     "0",
				IsPrefix: true,
			})
		}

		for _, obj := range page.Contents {
			modified := ""
			if obj.LastModified != nil {
				modified = obj.LastModified.Format(time.RFC3339)
			}

			objects = append(objects, objectEntry{
				Key:          internals3.StripPrefix(awsutil.DerefString(obj.Key), prefix),
				LastModified: modified,
				Size:         internals3.FormatSize(awsutil.DerefInt64(obj.Size)),
			})
		}
	}

	return objects, nil
}
