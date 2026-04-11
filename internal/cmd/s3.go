package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/cobra"
)

func newS3Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "s3",
		Short: "List S3 buckets",
		Long:  `List S3 buckets in the current AWS account.`,
	}

	cmd.AddCommand(newListBucketsCmd())

	return cmd
}

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

func runListBuckets(ctx context.Context) error {
	if err := EnsureCredentials(); err != nil {
		return err
	}

	cfg, err := LoadAWSConfig(ctx)
	if err != nil {
		return fmt.Errorf("loading AWS config: %w", err)
	}

	return listBuckets(ctx, s3.NewFromConfig(cfg))
}

func listBuckets(ctx context.Context, api s3ListAPI) error {
	output, err := api.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return fmt.Errorf("listing S3 buckets: %w", err)
	}

	if len(output.Buckets) == 0 {
		fmt.Fprintln(os.Stderr, "No S3 buckets found.")

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
