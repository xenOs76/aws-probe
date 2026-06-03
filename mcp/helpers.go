package mcp

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func loadAWS(ctx context.Context, deps *Deps) (aws.Config, error) {
	cfg, err := deps.LoadConfig(ctx)
	if err != nil {
		return aws.Config{}, fmt.Errorf("loading AWS config: %w", err)
	}

	return cfg, nil
}
