package cmd

import (
	"errors"

	"github.com/spf13/cobra"
	internals3 "github.com/xenos76/aws-probe/internal/s3"
)

// newS3Cmd creates the S3 command.
func newS3Cmd() *cobra.Command {
	var (
		listBucketsFlag bool
		listBucketFlag  string
		getMetadataFlag string
		path            string
		recursive       bool
		key             string
	)

	cmd := &cobra.Command{
		Use:   "s3",
		Short: "Manage S3 resources",
		Long:  `Manage S3 buckets and objects.`,
		Example: `  # List all buckets
  aws-probe s3 --list-buckets

  # List objects in a bucket
  aws-probe s3 --list-bucket my-bucket --path logs/ --recursive

  # Get object metadata
  aws-probe s3 --get-metadata my-bucket --key my-file.txt`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if listBucketsFlag {
				return runListBuckets(cmd)
			}

			if listBucketFlag != "" {
				return runListBucket(cmd, listBucketFlag, path, recursive)
			}

			if getMetadataFlag != "" {
				return runGetObjectMetadata(cmd, getMetadataFlag, key)
			}

			return cmd.Help()
		},
	}

	cmd.Flags().BoolVar(&listBucketsFlag, "list-buckets", false, "List all S3 buckets")
	cmd.Flags().StringVar(&listBucketFlag, "list-bucket", "", "List objects in the specified bucket")
	cmd.Flags().StringVar(&getMetadataFlag, "get-metadata", "", "Get metadata for an object in the specified bucket")
	cmd.Flags().StringVar(&path, "path", "", "Path prefix for listing objects (use with --list-bucket)")
	cmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "List objects recursively (use with --list-bucket)")
	cmd.Flags().StringVar(&key, "key", "", "Object key (use with --get-metadata)")

	cmd.MarkFlagsMutuallyExclusive("list-buckets", "list-bucket", "get-metadata")

	return cmd
}

func runListBuckets(cmd *cobra.Command) error {
	cfg, err := PrepareAWSConfig(cmd.Context())
	if err != nil {
		return err
	}

	client := internals3.NewClient(cfg)

	return internals3.ListBuckets(cmd.Context(), client, cmd.OutOrStdout())
}

func runListBucket(cmd *cobra.Command, bucket, path string, recursive bool) error {
	cfg, err := PrepareAWSConfig(cmd.Context())
	if err != nil {
		return err
	}

	client := internals3.NewClient(cfg)

	return internals3.ListBucket(cmd.Context(), bucket, path, recursive, client, cmd.OutOrStdout())
}

func runGetObjectMetadata(cmd *cobra.Command, bucket, key string) error {
	if key == "" {
		return errors.New("--key is required when using --get-metadata")
	}

	cfg, err := PrepareAWSConfig(cmd.Context())
	if err != nil {
		return err
	}

	s3Client := internals3.NewClient(cfg)
	kmsClient := internals3.NewKMSClient(cfg)

	return internals3.GetObjectMetadata(
		cmd.Context(),
		bucket,
		key,
		s3Client,
		kmsClient,
		kmsClient,
		cmd.OutOrStdout(),
	)
}
