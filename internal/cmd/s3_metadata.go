package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/cobra"
)

const metadataFieldFormat = "%-24s%s\n"

func newGetObjectMetadataCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-object-metadata [bucket-name] [key]",
		Short: "Get metadata for an S3 object",
		Long: `Get metadata for an S3 object. Displays all available
metadata information including size, content type, storage class,
and encryption details.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGetObjectMetadata(cmd.Context(), args[0], args[1])
		},
	}
}

func runGetObjectMetadata(ctx context.Context, bucket, key string) error {
	cfg, err := PrepareAWSConfig(ctx)
	if err != nil {
		return err
	}

	s3Client := s3.NewFromConfig(cfg)
	kmsClient := kms.NewFromConfig(cfg)

	return getObjectMetadata(ctx, bucket, key, s3Client, kmsClient, kmsClient)
}

// getObjectMetadata retrieves and displays metadata for an S3 object.
func getObjectMetadata(
	ctx context.Context,
	bucket string,
	key string,
	s3Client s3ObjectHeader,
	kmsClient kmsKeyDescriber,
	kmsAliasesClient kmsAliasesLister,
) error {
	output, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return fmt.Errorf("getting S3 object metadata: %w", err)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	displayGeneralInfo(tw, key, output)
	displayContentInfo(tw, output)
	displayStorageInfo(tw, output)
	displayEncryptionInfo(ctx, tw, output, kmsClient, kmsAliasesClient)
	displayVersioningInfo(tw, output)
	displayObjectLockInfo(tw, output)
	displayOtherInfo(tw, output)
	displayCustomMetadata(tw, output.Metadata)

	return tw.Flush()
}

// displayGeneralInfo prints general information about the object.
func displayGeneralInfo(tw *tabwriter.Writer, key string, output *s3.HeadObjectOutput) {
	fmt.Fprintln(tw, "\nGENERAL")
	fmt.Fprintf(tw, metadataFieldFormat, "KEY", key)
	fmt.Fprintf(tw, metadataFieldFormat, "SIZE", formatSize(derefInt64(output.ContentLength)))
	fmt.Fprintf(tw, metadataFieldFormat, "ETAG", formatETag(output.ETag))

	if output.DeleteMarker != nil && *output.DeleteMarker {
		fmt.Fprintf(tw, metadataFieldFormat, "DELETE MARKER", "true")
	}

	if output.Expiration != nil {
		fmt.Fprintf(tw, metadataFieldFormat, "EXPIRATION", derefString(output.Expiration))
	}

	if output.Restore != nil {
		fmt.Fprintf(tw, metadataFieldFormat, "RESTORE", derefString(output.Restore))
	}
}

// displayContentInfo prints content-related information about the object.
func displayContentInfo(tw *tabwriter.Writer, output *s3.HeadObjectOutput) {
	fmt.Fprintln(tw, "\nCONTENT")
	fmtField(tw, "CONTENT-TYPE", output.ContentType)
	fmtField(tw, "CONTENT-ENCODING", output.ContentEncoding)
	fmtField(tw, "CONTENT-LANGUAGE", output.ContentLanguage)
	fmtField(tw, "CONTENT-DISPOSITION", output.ContentDisposition)
	fmtField(tw, "CONTENT-RANGE", output.ContentRange)
	fmtField(tw, "CACHE-CONTROL", output.CacheControl)
	fmtField(tw, "ACCEPT-RANGES", output.AcceptRanges)

	if output.Expires != nil {
		fmt.Fprintf(tw, metadataFieldFormat, "EXPIRES", output.Expires.Format(time.RFC1123))
	}

	if output.ExpiresString != nil {
		fmt.Fprintf(tw, metadataFieldFormat, "EXPIRES (STRING)", derefString(output.ExpiresString))
	}
}

// displayStorageInfo prints storage-related information about the object.
func displayStorageInfo(tw *tabwriter.Writer, output *s3.HeadObjectOutput) {
	fmt.Fprintln(tw, "\nSTORAGE")

	if output.StorageClass != "" {
		fmt.Fprintf(tw, metadataFieldFormat, "STORAGE CLASS", string(output.StorageClass))
	}

	if output.LastModified != nil {
		fmt.Fprintf(tw, metadataFieldFormat, "LAST MODIFIED", formatTime(output.LastModified))
	}

	if output.PartsCount != nil {
		fmt.Fprintf(tw, "%-24s%d\n", "PARTS COUNT", *output.PartsCount)
	}

	if output.MissingMeta != nil {
		fmt.Fprintf(tw, "%-24s%d\n", "MISSING META", *output.MissingMeta)
	}
}

// displayEncryptionInfo prints encryption-related information about the object.
func displayEncryptionInfo(
	ctx context.Context,
	tw *tabwriter.Writer,
	output *s3.HeadObjectOutput,
	kmsClient kmsKeyDescriber,
	kmsAliasesClient kmsAliasesLister,
) {
	fmt.Fprintln(tw, "\nENCRYPTION")

	sse := string(output.ServerSideEncryption)
	if sse == "" {
		fmt.Fprintf(tw, metadataFieldFormat, "SERVER-SIDE ENCRYPTION", "None")
	} else {
		fmt.Fprintf(tw, metadataFieldFormat, "SERVER-SIDE ENCRYPTION", sse)
	}

	if output.BucketKeyEnabled != nil {
		fmt.Fprintf(tw, "%-24s%t\n", "BUCKET KEY ENABLED", *output.BucketKeyEnabled)
	}

	fmtField(tw, "SSE-CUSTOMER ALGORITHM", output.SSECustomerAlgorithm)
	fmtField(tw, "SSE-CUSTOMER KEY MD5", output.SSECustomerKeyMD5)

	if output.SSEKMSKeyId != nil {
		kmsKeyID := derefString(output.SSEKMSKeyId)
		fmt.Fprintf(tw, metadataFieldFormat, "SSE-KMS KEY ID", kmsKeyID)

		kmsKeyARN, err := getKMSKeyARN(ctx, kmsClient, kmsKeyID)
		if err == nil && kmsKeyARN != "" {
			fmt.Fprintf(tw, metadataFieldFormat, "SSE-KMS KEY ARN", kmsKeyARN)
		}

		keyName, err := getKMSKeyName(ctx, kmsAliasesClient, kmsKeyID)
		if err == nil && keyName != "" {
			fmt.Fprintf(tw, metadataFieldFormat, "SSE-KMS KEY NAME", keyName)
		}
	}
}

// displayVersioningInfo prints versioning-related information about the object.
func displayVersioningInfo(tw *tabwriter.Writer, output *s3.HeadObjectOutput) {
	fmt.Fprintln(tw, "\nVERSIONING")
	fmtField(tw, "VERSION ID", output.VersionId)

	replicationStatus := string(output.ReplicationStatus)
	if replicationStatus != "" {
		fmt.Fprintf(tw, metadataFieldFormat, "REPLICATION STATUS", replicationStatus)
	}
}

// displayObjectLockInfo prints object lock-related information about the object.
func displayObjectLockInfo(tw *tabwriter.Writer, output *s3.HeadObjectOutput) {
	hasLockInfo := output.ObjectLockLegalHoldStatus != "" ||
		output.ObjectLockMode != "" ||
		output.ObjectLockRetainUntilDate != nil

	if !hasLockInfo {
		return
	}

	fmt.Fprintln(tw, "\nOBJECT LOCK")

	if output.ObjectLockLegalHoldStatus != "" {
		fmt.Fprintf(tw, metadataFieldFormat, "LEGAL HOLD STATUS",
			string(output.ObjectLockLegalHoldStatus))
	}

	if output.ObjectLockMode != "" {
		fmt.Fprintf(tw, metadataFieldFormat, "LOCK MODE", string(output.ObjectLockMode))
	}

	if output.ObjectLockRetainUntilDate != nil {
		fmt.Fprintf(tw, metadataFieldFormat, "RETAIN UNTIL DATE",
			output.ObjectLockRetainUntilDate.Format("2006-01-02 15:04:05 MST"))
	}
}

// displayOtherInfo prints other miscellaneous information about the object.
func displayOtherInfo(tw *tabwriter.Writer, output *s3.HeadObjectOutput) {
	hasOtherInfo := output.WebsiteRedirectLocation != nil ||
		output.ChecksumCRC32 != nil ||
		output.ChecksumCRC32C != nil ||
		output.ChecksumCRC64NVME != nil ||
		output.ChecksumSHA1 != nil ||
		output.ChecksumSHA256 != nil

	if !hasOtherInfo {
		return
	}

	fmt.Fprintln(tw, "\nOTHER")
	fmtField(tw, "WEBSITE REDIRECT", output.WebsiteRedirectLocation)

	if output.ChecksumType != "" {
		fmt.Fprintf(tw, metadataFieldFormat, "CHECKSUM TYPE", string(output.ChecksumType))
	}

	fmtField(tw, "CHECKSUM CRC32", output.ChecksumCRC32)
	fmtField(tw, "CHECKSUM CRC32C", output.ChecksumCRC32C)
	fmtField(tw, "CHECKSUM CRC64NVME", output.ChecksumCRC64NVME)
	fmtField(tw, "CHECKSUM SHA1", output.ChecksumSHA1)
	fmtField(tw, "CHECKSUM SHA256", output.ChecksumSHA256)
}

// displayCustomMetadata prints custom metadata key-value pairs.
func displayCustomMetadata(tw *tabwriter.Writer, metadata map[string]string) {
	if len(metadata) == 0 {
		return
	}

	fmt.Fprintln(tw, "\nCUSTOM METADATA")

	for k, v := range metadata {
		fmt.Fprintf(tw, "  %-22s%s\n", k, v)
	}
}

// fmtField prints a labeled field if the value is not nil or empty.
func fmtField(tw *tabwriter.Writer, label string, value *string) {
	if value != nil && *value != "" {
		fmt.Fprintf(tw, metadataFieldFormat, label, *value)
	}
}

// formatETag removes quotes from the ETag string.
func formatETag(etag *string) string {
	if etag == nil {
		return "-"
	}

	return strings.Trim(*etag, `"`)
}

// formatTime formats a time pointer to a human-readable string.
func formatTime(t *time.Time) string {
	if t == nil {
		return "-"
	}

	return t.Format("2006-01-02 15:04:05 MST")
}

// getKMSKeyARN retrieves the ARN of a KMS key.
func getKMSKeyARN(ctx context.Context, api kmsKeyDescriber, keyID string) (string, error) {
	input := &kms.DescribeKeyInput{
		KeyId: &keyID,
	}

	output, err := api.DescribeKey(ctx, input)
	if err != nil {
		return "", err
	}

	if output.KeyMetadata != nil && output.KeyMetadata.Arn != nil {
		return derefString(output.KeyMetadata.Arn), nil
	}

	return "", nil
}

// getKMSKeyName retrieves the alias name of a KMS key.
func getKMSKeyName(ctx context.Context, api kmsAliasesLister, keyID string) (string, error) {
	paginator := kms.NewListAliasesPaginator(api, &kms.ListAliasesInput{})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return "", err
		}

		for _, alias := range output.Aliases {
			if alias.TargetKeyId != nil && *alias.TargetKeyId == keyID {
				if alias.AliasName != nil {
					return strings.TrimPrefix(derefString(alias.AliasName), "alias/"), nil
				}
			}
		}
	}

	return "", errors.New("alias not found")
}
