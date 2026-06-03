package mcp

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xenos76/aws-probe/internal/awsutil"
	internalsns "github.com/xenos76/aws-probe/internal/sns"
)

type listTopicsOutput struct {
	TopicARNs []string `json:"topicArns"`
}

func registerSNSTools(server *mcp.Server, deps *Deps) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "aws_probe_sns_list_topics",
		Description: "List SNS topic ARNs in the account",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, listTopicsOutput, error) {
		cfg, err := loadAWS(ctx, deps)
		if err != nil {
			return nil, listTopicsOutput{}, err
		}

		client := internalsns.NewClient(cfg)
		paginator := sns.NewListTopicsPaginator(client, &sns.ListTopicsInput{})

		arns := make([]string, 0)

		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, listTopicsOutput{}, fmt.Errorf("listing SNS topics: %w", err)
			}

			for _, t := range page.Topics {
				arns = append(arns, awsutil.DerefString(t.TopicArn))
			}
		}

		return nil, listTopicsOutput{TopicARNs: arns}, nil
	})
}
