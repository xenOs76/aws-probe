package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/xenos76/aws-probe/internal/awsutil"
)

const metadataFieldFormat = "%-24s%s\n"

// ObjectHeader defines the interface for getting S3 object metadata.
type ObjectHeader interface {
	HeadObject(
		ctx context.Context,
		params *s3.HeadObjectInput,
		optFns ...func(*s3.Options),
	) (*s3.HeadObjectOutput, error)
}

// KMSKeyDescriber defines the interface for describing KMS keys.
type KMSKeyDescriber interface {
	DescribeKey(
		ctx context.Context,
		params *kms.DescribeKeyInput,
		optFns ...func(*kms.Options),
	) (*kms.DescribeKeyOutput, error)
}

// KMSAliasesLister defines the interface for listing KMS aliases.
type KMSAliasesLister interface {
	ListAliases(
		ctx context.Context,
		params *kms.ListAliasesInput,
		optFns ...func(*kms.Options),
	) (*kms.ListAliasesOutput, error)
}

// GetObjectMetadata retrieves and displays metadata for an S3 object.
func GetObjectMetadata(
	ctx context.Context,
	bucket string,
	key string,
	s3Client ObjectHeader,
	kmsClient KMSKeyDescriber,
	kmsAliasesClient KMSAliasesLister,
	w io.Writer,
) error {
	output, err := s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return fmt.Errorf("getting S3 object metadata: %w", err)
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

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

// NewKMSClient creates a new KMS client.
func NewKMSClient(cfg aws.Config) *kms.Client {
	return kms.NewFromConfig(cfg)
}

func displayGeneralInfo(tw *tabwriter.Writer, key string, output *s3.HeadObjectOutput) {
	fmt.Fprintln(tw, "\nGENERAL")
	fmt.Fprintf(tw, metadataFieldFormat, "KEY", key)
	fmt.Fprintf(tw, metadataFieldFormat, "SIZE", FormatSize(awsutil.DerefInt64(output.ContentLength)))
	fmt.Fprintf(tw, metadataFieldFormat, "ETAG", formatETag(output.ETag))

	if output.DeleteMarker != nil && *output.DeleteMarker {
		fmt.Fprintf(tw, metadataFieldFormat, "DELETE MARKER", "true")
	}

	if output.Expiration != nil {
		fmt.Fprintf(tw, metadataFieldFormat, "EXPIRATION", awsutil.DerefString(output.Expiration))
	}

	if output.Restore != nil {
		fmt.Fprintf(tw, metadataFieldFormat, "RESTORE", awsutil.DerefString(output.Restore))
	}
}

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
		fmt.Fprintf(tw, metadataFieldFormat, "EXPIRES (STRING)", awsutil.DerefString(output.ExpiresString))
	}
}

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

func displayEncryptionInfo(
	ctx context.Context,
	tw *tabwriter.Writer,
	output *s3.HeadObjectOutput,
	kmsClient KMSKeyDescriber,
	kmsAliasesClient KMSAliasesLister,
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
		kmsKeyID := awsutil.DerefString(output.SSEKMSKeyId)
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

func displayVersioningInfo(tw *tabwriter.Writer, output *s3.HeadObjectOutput) {
	fmt.Fprintln(tw, "\nVERSIONING")
	fmtField(tw, "VERSION ID", output.VersionId)

	replicationStatus := string(output.ReplicationStatus)
	if replicationStatus != "" {
		fmt.Fprintf(tw, metadataFieldFormat, "REPLICATION STATUS", replicationStatus)
	}
}

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

func displayCustomMetadata(tw *tabwriter.Writer, metadata map[string]string) {
	if len(metadata) == 0 {
		return
	}

	fmt.Fprintln(tw, "\nCUSTOM METADATA")

	for k, v := range metadata {
		fmt.Fprintf(tw, "  %-22s%s\n", k, v)
	}
}

func fmtField(tw *tabwriter.Writer, label string, value *string) {
	if value != nil && *value != "" {
		fmt.Fprintf(tw, metadataFieldFormat, label, *value)
	}
}

func formatETag(etag *string) string {
	if etag == nil {
		return "-"
	}

	return strings.Trim(*etag, `"`)
}

func formatTime(t *time.Time) string {
	if t == nil {
		return "-"
	}

	return t.Format("2006-01-02 15:04:05 MST")
}

func getKMSKeyARN(ctx context.Context, api KMSKeyDescriber, keyID string) (string, error) {
	output, err := api.DescribeKey(ctx, &kms.DescribeKeyInput{KeyId: &keyID})
	if err != nil {
		return "", err
	}

	if output.KeyMetadata != nil && output.KeyMetadata.Arn != nil {
		return awsutil.DerefString(output.KeyMetadata.Arn), nil
	}

	return "", nil
}

func getKMSKeyName(ctx context.Context, api KMSAliasesLister, keyID string) (string, error) {
	paginator := kms.NewListAliasesPaginator(api, &kms.ListAliasesInput{})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return "", err
		}

		for _, alias := range output.Aliases {
			if alias.TargetKeyId != nil && *alias.TargetKeyId == keyID {
				if alias.AliasName != nil {
					return strings.TrimPrefix(awsutil.DerefString(alias.AliasName), "alias/"), nil
				}
			}
		}
	}

	return "", errors.New("alias not found")
}
