package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/spf13/cobra"
)

// newS3Cmd creates the S3 command.
func newS3Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "s3",
		Short: "List S3 buckets",
		Long:  `List S3 buckets in the current AWS account.`,
	}

	cmd.AddCommand(newListBucketsCmd())
	cmd.AddCommand(newListBucketCmd())
	cmd.AddCommand(newGetObjectMetadataCmd())

	return cmd
}

// newListBucketsCmd creates the list-buckets subcommand.
func newListBucketsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-buckets",
		Short: "List S3 buckets",
		Long:  `List all S3 buckets in the current AWS account.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runListBuckets(cmd.Context())
		},
	}
}

// runListBuckets executes the list-buckets command.
func runListBuckets(ctx context.Context) error {
	cfg, err := PrepareAWSConfig(ctx)
	if err != nil {
		return err
	}

	return listBuckets(ctx, s3.NewFromConfig(cfg))
}

// listBuckets lists S3 buckets using the provided API client.
func listBuckets(ctx context.Context, api s3Lister) error {
	output, err := api.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return fmt.Errorf("listing S3 buckets: %w", err)
	}

	if len(output.Buckets) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "No S3 buckets found.")

		return nil
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	fmt.Fprint(tw, "NAME\tCREATED\n")

	for _, bucket := range output.Buckets {
		created := ""
		if bucket.CreationDate != nil {
			created = bucket.CreationDate.Format("2006-01-02 15:04:05")
		}

		fmt.Fprintf(tw, "%s\t%s\n", derefString(bucket.Name), created)
	}

	return tw.Flush()
}

const defaultPageSize = 50

// newListBucketCmd creates the list-bucket subcommand.
func newListBucketCmd() *cobra.Command {
	var recursive bool

	cmd := &cobra.Command{
		Use:   "list-bucket [bucket-name] [path]",
		Short: "List objects in an S3 bucket",
		Long: `List objects in an S3 bucket. The path is an S3 key prefix 
(e.g., "logs/" or "data/files/").`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := ""
			if len(args) > 1 {
				path = args[1]
			}

			return runListBucket(cmd.Context(), args[0], path, recursive)
		},
	}

	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "List all objects recursively")

	return cmd
}

// runListBucket executes the list-bucket command.
func runListBucket(ctx context.Context, bucket, path string, recursive bool) error {
	cfg, err := PrepareAWSConfig(ctx)
	if err != nil {
		return err
	}

	return listBucket(ctx, bucket, path, recursive, s3.NewFromConfig(cfg))
}

// listBucket lists objects in an S3 bucket using the provided API client.
//
//nolint:revive // recursive is not a control flag in this context, it's a functional option
func listBucket(ctx context.Context, bucket, prefix string, recursive bool, api s3ObjectsLister) error {
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

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

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
		_, _ = fmt.Fprintln(os.Stderr, "No objects found.")

		return nil
	}

	return tw.Flush()
}

// stripPrefix removes the given prefix from a key if present.
func stripPrefix(key, prefix string) string {
	if prefix == "" {
		return key
	}

	return strings.TrimPrefix(key, prefix)
}

// formatSize formats byte size to human-readable string.
func formatSize(size int64) string {
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
			displayKey := stripPrefix(derefString(cp.Prefix), basePrefix)
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

		size := formatSize(derefInt64(obj.Size))
		displayKey := stripPrefix(derefString(obj.Key), basePrefix)

		fmt.Fprintf(tw, "%s\t%s\t%s\n",
			displayKey,
			modified,
			size,
		)
	}
}
