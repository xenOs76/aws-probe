package mcp

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/xenos76/aws-probe/internal/awsutil"
)

// Config holds optional defaults for MCP tool behavior.
type Config struct {
	Region string
	Output string // table, json, csv (reserved for future formatted tools)
	Theme  string
}

// Deps bundles dependencies for MCP registration and tool handlers.
type Deps struct {
	LoadConfig func(context.Context) (aws.Config, error)
	Config     Config
}

// DefaultLoadConfig returns the standard aws-probe AWS config loader with credential checks.
func DefaultLoadConfig() func(context.Context) (aws.Config, error) {
	return func(ctx context.Context) (aws.Config, error) {
		return awsutil.PrepareAWSConfig(ctx)
	}
}

// withDefaults fills zero Deps fields with production defaults.
func (d *Deps) withDefaults() *Deps {
	if d == nil {
		d = &Deps{}
	}

	out := *d
	if out.LoadConfig == nil {
		out.LoadConfig = DefaultLoadConfig()
	}

	return &out
}
