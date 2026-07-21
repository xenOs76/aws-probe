package s3

import (
	"bytes"
	"context"
	"errors"
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
		{
			name: "no objects",
			mockListObjectsV2: func(_ context.Context, _ *s3.ListObjectsV2Input,
				_ ...func(*s3.Options),
			) (*s3.ListObjectsV2Output, error) {
				return &s3.ListObjectsV2Output{}, nil
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

			if tt.name == "no objects" {
				require.Contains(t, buf.String(), "No objects found.")
				require.NotContains(t, buf.String(), "KEY")
			} else {
				require.Contains(t, buf.String(), "KEY")
				require.Contains(t, buf.String(), "folder1/")
				require.Contains(t, buf.String(), "file1.txt")
			}
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

func TestGetKMSKeyARN(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		kmsMock := &mockKMSClient{
			DescribeKeyFunc: func(_ context.Context, in *kms.DescribeKeyInput, _ ...func(*kms.Options)) (
				*kms.DescribeKeyOutput, error,
			) {
				require.Equal(t, "key-1", aws.ToString(in.KeyId))

				return &kms.DescribeKeyOutput{
					KeyMetadata: &kmstypes.KeyMetadata{Arn: aws.String("arn:aws:kms:us-east-1:1:key/key-1")},
				}, nil
			},
		}
		arn, err := getKMSKeyARN(ctx, kmsMock, "key-1")
		require.NoError(t, err)
		require.Equal(t, "arn:aws:kms:us-east-1:1:key/key-1", arn)
	})

	t.Run("describe error", func(t *testing.T) {
		kmsMock := &mockKMSClient{
			DescribeKeyFunc: func(_ context.Context, _ *kms.DescribeKeyInput, _ ...func(*kms.Options)) (
				*kms.DescribeKeyOutput, error,
			) {
				return nil, errors.New("denied")
			},
		}
		_, err := getKMSKeyARN(ctx, kmsMock, "key-1")
		require.Error(t, err)
	})

	t.Run("empty metadata", func(t *testing.T) {
		kmsMock := &mockKMSClient{
			DescribeKeyFunc: func(_ context.Context, _ *kms.DescribeKeyInput, _ ...func(*kms.Options)) (
				*kms.DescribeKeyOutput, error,
			) {
				return &kms.DescribeKeyOutput{}, nil
			},
		}
		arn, err := getKMSKeyARN(ctx, kmsMock, "key-1")
		require.NoError(t, err)
		require.Empty(t, arn)
	})
}

func TestGetKMSKeyName(t *testing.T) {
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		kmsMock := &mockKMSClient{
			ListAliasesFunc: func(_ context.Context, _ *kms.ListAliasesInput, _ ...func(*kms.Options)) (
				*kms.ListAliasesOutput, error,
			) {
				return &kms.ListAliasesOutput{
					Aliases: []kmstypes.AliasListEntry{{
						AliasName:   aws.String("alias/my-key"),
						TargetKeyId: aws.String("key-1"),
					}},
				}, nil
			},
		}
		name, err := getKMSKeyName(ctx, kmsMock, "key-1")
		require.NoError(t, err)
		require.Equal(t, "my-key", name)
	})

	t.Run("not found", func(t *testing.T) {
		kmsMock := &mockKMSClient{
			ListAliasesFunc: func(_ context.Context, _ *kms.ListAliasesInput, _ ...func(*kms.Options)) (
				*kms.ListAliasesOutput, error,
			) {
				return &kms.ListAliasesOutput{}, nil
			},
		}
		_, err := getKMSKeyName(ctx, kmsMock, "key-1")
		require.Error(t, err)
	})

	t.Run("list error", func(t *testing.T) {
		kmsMock := &mockKMSClient{
			ListAliasesFunc: func(_ context.Context, _ *kms.ListAliasesInput, _ ...func(*kms.Options)) (
				*kms.ListAliasesOutput, error,
			) {
				return nil, errors.New("fail")
			},
		}
		_, err := getKMSKeyName(ctx, kmsMock, "key-1")
		require.Error(t, err)
	})
}

func TestGetObjectMetadata_KMSAndExtras(t *testing.T) {
	retain := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	api := &mockS3Client{
		HeadObjectFunc: func(_ context.Context, _ *s3.HeadObjectInput, _ ...func(*s3.Options)) (
			*s3.HeadObjectOutput, error,
		) {
			return &s3.HeadObjectOutput{
				ContentLength:             aws.Int64(10),
				ETag:                      aws.String(`"e"`),
				ServerSideEncryption:      s3types.ServerSideEncryptionAwsKms,
				SSEKMSKeyId:               aws.String("key-1"),
				BucketKeyEnabled:          aws.Bool(true),
				ObjectLockMode:            s3types.ObjectLockModeCompliance,
				ObjectLockLegalHoldStatus: s3types.ObjectLockLegalHoldStatusOn,
				ObjectLockRetainUntilDate: &retain,
				ChecksumSHA256:            aws.String("abc"),
				Metadata:                  map[string]string{"owner": "x"},
				DeleteMarker:              aws.Bool(true),
			}, nil
		},
	}
	kmsMock := &mockKMSClient{
		DescribeKeyFunc: func(_ context.Context, _ *kms.DescribeKeyInput, _ ...func(*kms.Options)) (
			*kms.DescribeKeyOutput, error,
		) {
			return &kms.DescribeKeyOutput{
				KeyMetadata: &kmstypes.KeyMetadata{Arn: aws.String("arn:kms:key-1")},
			}, nil
		},
		ListAliasesFunc: func(_ context.Context, _ *kms.ListAliasesInput, _ ...func(*kms.Options)) (
			*kms.ListAliasesOutput, error,
		) {
			return &kms.ListAliasesOutput{
				Aliases: []kmstypes.AliasListEntry{{
					AliasName: aws.String("alias/prod"), TargetKeyId: aws.String("key-1"),
				}},
			}, nil
		},
	}

	var buf bytes.Buffer

	err := GetObjectMetadata(context.Background(), "b", "k", api, kmsMock, kmsMock, &buf)
	require.NoError(t, err)

	out := buf.String()
	require.Contains(t, out, "SSE-KMS KEY ARN")
	require.Contains(t, out, "prod")
	require.Contains(t, out, "OBJECT LOCK")
	require.Contains(t, out, "CUSTOM METADATA")
	require.Contains(t, out, "DELETE MARKER")
}

func TestFormatSizeAndStripPrefix(t *testing.T) {
	require.Equal(t, "0 B", FormatSize(0))
	require.Equal(t, "1.0 KB", FormatSize(1024))
	require.Equal(t, "file", StripPrefix("pre/file", "pre/"))
	require.Equal(t, "pre/file", StripPrefix("pre/file", ""))
}

func TestGetObjectMetadata_HeadError(t *testing.T) {
	api := &mockS3Client{
		HeadObjectFunc: func(_ context.Context, _ *s3.HeadObjectInput, _ ...func(*s3.Options)) (
			*s3.HeadObjectOutput, error,
		) {
			return nil, errors.New("missing")
		},
	}
	err := GetObjectMetadata(context.Background(), "b", "k", api, nil, nil, &bytes.Buffer{})
	require.Error(t, err)
}
