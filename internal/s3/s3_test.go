package s3

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/require"
)

type mockS3Client struct {
	ListBucketsFunc func(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (
		*s3.ListBucketsOutput, error)
	ListObjectsV2Func func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (
		*s3.ListObjectsV2Output, error)
	HeadObjectFunc func(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (
		*s3.HeadObjectOutput, error)
}

func (m *mockS3Client) ListBuckets(ctx context.Context, params *s3.ListBucketsInput,
	optFns ...func(*s3.Options),
) (*s3.ListBucketsOutput, error) {
	return m.ListBucketsFunc(ctx, params, optFns...)
}

func (m *mockS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input,
	optFns ...func(*s3.Options),
) (*s3.ListObjectsV2Output, error) {
	return m.ListObjectsV2Func(ctx, params, optFns...)
}

func (m *mockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput,
	optFns ...func(*s3.Options),
) (*s3.HeadObjectOutput, error) {
	return m.HeadObjectFunc(ctx, params, optFns...)
}

type mockKMSClient struct {
	DescribeKeyFunc func(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (
		*kms.DescribeKeyOutput, error)
	ListAliasesFunc func(ctx context.Context, params *kms.ListAliasesInput, optFns ...func(*kms.Options)) (
		*kms.ListAliasesOutput, error)
}

func (m *mockKMSClient) DescribeKey(ctx context.Context, params *kms.DescribeKeyInput,
	optFns ...func(*kms.Options),
) (*kms.DescribeKeyOutput, error) {
	return m.DescribeKeyFunc(ctx, params, optFns...)
}

func (m *mockKMSClient) ListAliases(ctx context.Context, params *kms.ListAliasesInput,
	optFns ...func(*kms.Options),
) (*kms.ListAliasesOutput, error) {
	return m.ListAliasesFunc(ctx, params, optFns...)
}

func TestListBuckets(t *testing.T) {
	creationDate := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name            string
		mockListBuckets func(
			ctx context.Context,
			params *s3.ListBucketsInput,
			optFns ...func(*s3.Options),
		) (*s3.ListBucketsOutput, error)
		wantOutput string
		wantErr    bool
	}{
		{
			name: "success",
			mockListBuckets: func(_ context.Context, _ *s3.ListBucketsInput,
				_ ...func(*s3.Options),
			) (*s3.ListBucketsOutput, error) {
				return &s3.ListBucketsOutput{
					Buckets: []s3types.Bucket{
						{Name: aws.String("bucket1"), CreationDate: &creationDate},
					},
				}, nil
			},
			wantOutput: "NAME    CREATED\nbucket1  2023-01-01 12:00:00\n",
			wantErr:    false,
		},
		{
			name: "no buckets",
			mockListBuckets: func(_ context.Context, _ *s3.ListBucketsInput,
				_ ...func(*s3.Options),
			) (*s3.ListBucketsOutput, error) {
				return &s3.ListBucketsOutput{}, nil
			},
			wantOutput: "No S3 buckets found.\n",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &mockS3Client{ListBucketsFunc: tt.mockListBuckets}

			var buf bytes.Buffer

			err := ListBuckets(context.Background(), api, &buf)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.name == "no buckets" {
				require.Contains(t, buf.String(), "No S3 buckets found.")
			} else {
				require.Contains(t, buf.String(), "NAME")
				require.Contains(t, buf.String(), "bucket1")
				require.Contains(t, buf.String(), "2023-01-01 12:00:00")
			}
		})
	}
}

func TestListBucket(t *testing.T) {
	lastModified := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name              string
		mockListObjectsV2 func(
			ctx context.Context,
			params *s3.ListObjectsV2Input,
			optFns ...func(*s3.Options),
		) (*s3.ListObjectsV2Output, error)
		wantErr bool
	}{
		{
			name: "success",
			mockListObjectsV2: func(_ context.Context, _ *s3.ListObjectsV2Input,
				_ ...func(*s3.Options),
			) (*s3.ListObjectsV2Output, error) {
				return &s3.ListObjectsV2Output{
					Contents: []s3types.Object{
						{Key: aws.String("file1.txt"), LastModified: &lastModified, Size: aws.Int64(1024)},
					},
					CommonPrefixes: []s3types.CommonPrefix{
						{Prefix: aws.String("folder1/")},
					},
				}, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &mockS3Client{ListObjectsV2Func: tt.mockListObjectsV2}

			var buf bytes.Buffer

			err := ListBucket(context.Background(), "my-bucket", "", false, api, &buf)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Contains(t, buf.String(), "KEY")
			require.Contains(t, buf.String(), "folder1/")
			require.Contains(t, buf.String(), "file1.txt")
		})
	}
}

func TestGetObjectMetadata(t *testing.T) {
	lastModified := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name           string
		mockHeadObject func(ctx context.Context, params *s3.HeadObjectInput,
			optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
		wantOutput string
		wantErr    bool
	}{
		{
			name: "success",
			mockHeadObject: func(_ context.Context, _ *s3.HeadObjectInput,
				_ ...func(*s3.Options),
			) (*s3.HeadObjectOutput, error) {
				return &s3.HeadObjectOutput{
					ContentLength: aws.Int64(1024),
					ContentType:   aws.String("text/plain"),
					LastModified:  &lastModified,
					ETag:          aws.String("\"etag123\""),
				}, nil
			},
			wantOutput: `
GENERAL
KEY                     test-key
SIZE                    1.0 KB
ETAG                    etag123

CONTENT
CONTENT-TYPE            text/plain

STORAGE
LAST MODIFIED           2023-01-01 12:00:00 UTC

ENCRYPTION
SERVER-SIDE ENCRYPTION  None

VERSIONING
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &mockS3Client{HeadObjectFunc: tt.mockHeadObject}
			kmsMock := &mockKMSClient{
				DescribeKeyFunc: func(_ context.Context, _ *kms.DescribeKeyInput,
					_ ...func(*kms.Options),
				) (*kms.DescribeKeyOutput, error) {
					return &kms.DescribeKeyOutput{
						KeyMetadata: &kmstypes.KeyMetadata{Arn: aws.String("arn:kms")},
					}, nil
				},
				ListAliasesFunc: func(_ context.Context, _ *kms.ListAliasesInput,
					_ ...func(*kms.Options),
				) (*kms.ListAliasesOutput, error) {
					return &kms.ListAliasesOutput{}, nil
				},
			}

			var buf bytes.Buffer

			err := GetObjectMetadata(context.Background(), "my-bucket", "test-key", api, kmsMock, kmsMock, &buf)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			// Use Contains because there might be some variation in exact spacing or trailing newlines
			require.Contains(t, buf.String(), "GENERAL")
			require.Contains(t, buf.String(), "KEY                     test-key")
		})
	}
}
