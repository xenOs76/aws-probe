package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunFunctions_ConfigError(t *testing.T) {
	oldPrepare := PrepareAWSConfig

	defer func() { PrepareAWSConfig = oldPrepare }()

	PrepareAWSConfig = func(_ context.Context, _ ...func(*config.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, errors.New("forced config error")
	}

	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"runListBuckets", func() error { return runListBuckets(ctx) }},
		{"runListBucket", func() error { return runListBucket(ctx, "b", "p", false) }},
		{"runGetObjectMetadata", func() error { return runGetObjectMetadata(ctx, "b", "k") }},
		{"runListSecrets", func() error { return runListSecrets(ctx) }},
		{"runListQueues", func() error { return runListQueues(ctx) }},
		{"runSnsListTopics", func() error { return runSnsListTopics(ctx) }},
		{"runSnsListSubscriptions", func() error { return runSnsListSubscriptions(ctx, "arn") }},
		{"runListClusters", func() error { return runListClusters(ctx) }},
		{"runListTopics", func() error { return runListTopics(ctx, "arn") }},
		{"runProduce", func() error { return runProduce(ctx) }},
		{"runConsume", func() error { return runConsume(ctx) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "forced config error")
		})
	}
}
