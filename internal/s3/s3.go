package s3

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/xenos76/aws-probe/internal/awsutil"
)

// BucketsLister defines the interface for listing S3 buckets.
type BucketsLister interface {
	ListBuckets(
		ctx context.Context,
		params *s3.ListBucketsInput,
		optFns ...func(*s3.Options),
	) (*s3.ListBucketsOutput, error)
}

// ObjectsLister defines the interface for listing S3 objects.
type ObjectsLister interface {
	ListObjectsV2(
		ctx context.Context,
		params *s3.ListObjectsV2Input,
		optFns ...func(*s3.Options),
	) (*s3.ListObjectsV2Output, error)
}

const defaultPageSize = 50

// ListBuckets lists S3 buckets using the provided API client.
func ListBuckets(ctx context.Context, api BucketsLister, w io.Writer) error {
	output, err := api.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return fmt.Errorf("listing S3 buckets: %w", err)
	}

	if len(output.Buckets) == 0 {
		fmt.Fprintln(w, "No S3 buckets found.")

		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	fmt.Fprint(tw, "NAME\tCREATED\n")

	for _, bucket := range output.Buckets {
		created := ""
		if bucket.CreationDate != nil {
			created = bucket.CreationDate.Format("2006-01-02 15:04:05")
		}

		fmt.Fprintf(tw, "%s\t%s\n", awsutil.DerefString(bucket.Name), created)
	}

	return tw.Flush()
}

// ListBucket lists objects in an S3 bucket using the provided API client.
//
//nolint:revive // recursive is not a control flag here, it's a functional option
func ListBucket(ctx context.Context, bucket, prefix string, recursive bool, api ObjectsLister, w io.Writer) error {
	input := &s3.ListObjectsV2Input{
		Bucket: &bucket,
	}

	if prefix != "" {
		input.Prefix = &prefix
	}

	if !recursive {
		delimiter := "/"
		input.Delimiter = &delimiter
	}

	paginator := s3.NewListObjectsV2Paginator(api, input, func(o *s3.ListObjectsV2PaginatorOptions) {
		o.Limit = defaultPageSize
	})

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	hasContent := false

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			_ = tw.Flush()

			return fmt.Errorf("listing S3 objects: %w", err)
		}

		if !hasContent {
			fmt.Fprint(tw, "KEY\tLAST MODIFIED\tSIZE\n")

			hasContent = true
		}

		renderCommonPrefixes(tw, output.CommonPrefixes, prefix)
		renderContents(tw, output.Contents, prefix)
	}

	if !hasContent {
		fmt.Fprintln(w, "No objects found.")

		return nil
	}

	return tw.Flush()
}

// StripPrefix removes the given prefix from a key if present.
func StripPrefix(key, prefix string) string {
	if prefix == "" {
		return key
	}

	return strings.TrimPrefix(key, prefix)
}

// FormatSize formats byte size to human-readable string.
func FormatSize(size int64) string {
	const unit = 1024

	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func renderCommonPrefixes(tw *tabwriter.Writer, prefixes []s3types.CommonPrefix, basePrefix string) {
	for _, cp := range prefixes {
		if cp.Prefix != nil {
			displayKey := StripPrefix(awsutil.DerefString(cp.Prefix), basePrefix)
			fmt.Fprintf(tw, "%s\t-\t0\n", displayKey)
		}
	}
}

func renderContents(tw *tabwriter.Writer, contents []s3types.Object, basePrefix string) {
	for _, obj := range contents {
		modified := ""
		if obj.LastModified != nil {
			modified = obj.LastModified.Format("2006-01-02 15:04:05")
		}

		size := FormatSize(awsutil.DerefInt64(obj.Size))
		displayKey := StripPrefix(awsutil.DerefString(obj.Key), basePrefix)

		fmt.Fprintf(tw, "%s\t%s\t%s\n", displayKey, modified, size)
	}
}

// NewClient creates a new S3 client.
func NewClient(cfg aws.Config) *s3.Client {
	return s3.NewFromConfig(cfg)
}
