package cmd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockS3MetadataClient struct {
	headObjectOutput *s3.HeadObjectOutput
	err              error
}

func (m *mockS3MetadataClient) HeadObject(
	_ context.Context,
	_ *s3.HeadObjectInput,
	_ ...func(*s3.Options),
) (*s3.HeadObjectOutput, error) {
	return m.headObjectOutput, m.err
}

type mockKMSClient struct {
	describeKeyOutput *kms.DescribeKeyOutput
	listAliasesOutput *kms.ListAliasesOutput
	err               error
}

func (m *mockKMSClient) DescribeKey(
	_ context.Context,
	_ *kms.DescribeKeyInput,
	_ ...func(*kms.Options),
) (*kms.DescribeKeyOutput, error) {
	return m.describeKeyOutput, m.err
}

func (m *mockKMSClient) ListAliases(
	_ context.Context,
	_ *kms.ListAliasesInput,
	_ ...func(*kms.Options),
) (*kms.ListAliasesOutput, error) {
	return m.listAliasesOutput, m.err
}

func TestGetObjectMetadata_Basic(t *testing.T) {
	lastModified := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	s3Client := &mockS3MetadataClient{
		headObjectOutput: &s3.HeadObjectOutput{
			ContentLength: aws.Int64(1024),
			ContentType:   aws.String("text/plain"),
			LastModified:  &lastModified,
			ETag:          aws.String("\"etag\""),
			StorageClass:  s3types.StorageClassStandard,
			Metadata: map[string]string{
				"custom": "value",
			},
		},
	}
	kmsClient := &mockKMSClient{}

	out, err := captureCmdOutput(t, func() error {
		return getObjectMetadata(context.Background(), "bucket", "test-key", s3Client, kmsClient, kmsClient)
	})

	require.NoError(t, err)

	wantOut := "\nGENERAL\n" +
		"KEY                     test-key\n" +
		"SIZE                    1.0 KB\n" +
		"ETAG                    etag\n" +
		"\nCONTENT\n" +
		"CONTENT-TYPE            text/plain\n" +
		"\nSTORAGE\n" +
		"STORAGE CLASS           STANDARD\n" +
		"LAST MODIFIED           2023-01-01 12:00:00 UTC\n" +
		"\nENCRYPTION\n" +
		"SERVER-SIDE ENCRYPTION  None\n" +
		"\nVERSIONING\n" +
		"\nCUSTOM METADATA\n" +
		"  custom                value\n"
	assert.Equal(t, wantOut, out.stdout)
}

func TestGetObjectMetadata_Encryption(t *testing.T) {
	s3Client := &mockS3MetadataClient{
		headObjectOutput: &s3.HeadObjectOutput{
			ServerSideEncryption: s3types.ServerSideEncryptionAwsKms,
			SSEKMSKeyId:          aws.String("kms-key-id"),
		},
	}
	kmsClient := &mockKMSClient{
		describeKeyOutput: &kms.DescribeKeyOutput{
			KeyMetadata: &kmstypes.KeyMetadata{
				Arn: aws.String("arn:aws:kms:region:123:key/kms-key-id"),
			},
		},
		listAliasesOutput: &kms.ListAliasesOutput{
			Aliases: []kmstypes.AliasListEntry{
				{
					AliasName:   aws.String("alias/my-key"),
					TargetKeyId: aws.String("kms-key-id"),
				},
			},
		},
	}

	out, err := captureCmdOutput(t, func() error {
		return getObjectMetadata(context.Background(), "bucket", "test-key", s3Client, kmsClient, kmsClient)
	})

	require.NoError(t, err)

	wantOut := "\nGENERAL\n" +
		"KEY                     test-key\n" +
		"SIZE                    0 B\n" +
		"ETAG                    -\n" +
		"\nCONTENT\n" +
		"\nSTORAGE\n" +
		"\nENCRYPTION\n" +
		"SERVER-SIDE ENCRYPTION  aws:kms\n" +
		"SSE-KMS KEY ID          kms-key-id\n" +
		"SSE-KMS KEY ARN         arn:aws:kms:region:123:key/kms-key-id\n" +
		"SSE-KMS KEY NAME        my-key\n" +
		"\nVERSIONING\n"
	assert.Equal(t, wantOut, out.stdout)
}

func TestGetObjectMetadata_Complex(t *testing.T) {
	lastModified := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	s3Client := &mockS3MetadataClient{
		headObjectOutput: &s3.HeadObjectOutput{
			DeleteMarker:              aws.Bool(true),
			ObjectLockMode:            s3types.ObjectLockModeCompliance,
			ObjectLockRetainUntilDate: &lastModified,
			ObjectLockLegalHoldStatus: s3types.ObjectLockLegalHoldStatusOn,
			ReplicationStatus:         s3types.ReplicationStatusCompleted,
			WebsiteRedirectLocation:   aws.String("https://example.com"),
			ChecksumSHA256:            aws.String("sha256sum"),
			ChecksumType:              s3types.ChecksumTypeFullObject,
			Expiration:                aws.String("expiry"),
			Restore:                   aws.String("restore-ongoing"),
		},
	}
	kmsClient := &mockKMSClient{}

	out, err := captureCmdOutput(t, func() error {
		return getObjectMetadata(context.Background(), "bucket", "test-key", s3Client, kmsClient, kmsClient)
	})

	require.NoError(t, err)

	wantOut := "\nGENERAL\n" +
		"KEY                     test-key\n" +
		"SIZE                    0 B\n" +
		"ETAG                    -\n" +
		"DELETE MARKER           true\n" +
		"EXPIRATION              expiry\n" +
		"RESTORE                 restore-ongoing\n" +
		"\nCONTENT\n" +
		"\nSTORAGE\n" +
		"\nENCRYPTION\n" +
		"SERVER-SIDE ENCRYPTION  None\n" +
		"\nVERSIONING\n" +
		"REPLICATION STATUS      COMPLETED\n" +
		"\nOBJECT LOCK\n" +
		"LEGAL HOLD STATUS       ON\n" +
		"LOCK MODE               COMPLIANCE\n" +
		"RETAIN UNTIL DATE       2023-01-01 12:00:00 UTC\n" +
		"\nOTHER\n" +
		"WEBSITE REDIRECT        https://example.com\n" +
		"CHECKSUM TYPE           FULL_OBJECT\n" +
		"CHECKSUM SHA256         sha256sum\n"
	assert.Equal(t, wantOut, out.stdout)
}

func TestGetObjectMetadata_Error(t *testing.T) {
	s3Client := &mockS3MetadataClient{
		err: errors.New("api error"),
	}
	kmsClient := &mockKMSClient{}

	_, err := captureCmdOutput(t, func() error {
		return getObjectMetadata(context.Background(), "bucket", "test-key", s3Client, kmsClient, kmsClient)
	})

	require.Error(t, err)
}

func TestFormatETag(t *testing.T) {
	assert.Equal(t, "etag", formatETag(aws.String("\"etag\"")))
	assert.Equal(t, "etag", formatETag(aws.String("etag")))
	assert.Equal(t, "-", formatETag(nil))
}

func TestFormatTime(t *testing.T) {
	now := time.Now()
	assert.Equal(t, now.Format("2006-01-02 15:04:05 MST"), formatTime(&now))
	assert.Equal(t, "-", formatTime(nil))
}

func TestGetKMSKeyARN(t *testing.T) {
	client := &mockKMSClient{
		describeKeyOutput: &kms.DescribeKeyOutput{
			KeyMetadata: &kmstypes.KeyMetadata{
				Arn: aws.String("arn"),
			},
		},
	}
	arn, err := getKMSKeyARN(context.Background(), client, "id")
	require.NoError(t, err)
	assert.Equal(t, "arn", arn)
}

func TestGetKMSKeyName(t *testing.T) {
	client := &mockKMSClient{
		listAliasesOutput: &kms.ListAliasesOutput{
			Aliases: []kmstypes.AliasListEntry{
				{
					AliasName:   aws.String("alias/name"),
					TargetKeyId: aws.String("id"),
				},
			},
		},
	}
	name, err := getKMSKeyName(context.Background(), client, "id")
	require.NoError(t, err)
	assert.Equal(t, "name", name)

	_, err = getKMSKeyName(context.Background(), client, "other")
	require.Error(t, err)
}
