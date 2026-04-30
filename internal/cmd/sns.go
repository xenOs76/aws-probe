package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/spf13/cobra"
)

// newSnsCmd creates the SNS command.
func newSnsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sns",
		Short: "List SNS topics and subscriptions",
		Long:  `List SNS topics and subscriptions in the current AWS account.`,
	}

	cmd.AddCommand(newSnsListTopicsCmd())
	cmd.AddCommand(newSnsListSubscriptionsCmd())

	return cmd
}

// newSnsListTopicsCmd creates the list-topics subcommand for SNS.
func newSnsListTopicsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-topics",
		Short: "List SNS topics",
		Long:  `List all SNS topics in the current AWS account.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSnsListTopics(cmd.Context())
		},
	}
}

// newSnsListSubscriptionsCmd creates the list-subscriptions subcommand for SNS.
func newSnsListSubscriptionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-subscriptions [topic-arn]",
		Short: "List SNS subscriptions",
		Long:  `List subscriptions for an SNS topic. Requires topic ARN as argument.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSnsListSubscriptions(cmd.Context(), args[0])
		},
	}
}

// runSnsListTopics executes the list-topics command for SNS.
func runSnsListTopics(ctx context.Context) error {
	cfg, err := PrepareAWSConfig(ctx)
	if err != nil {
		return err
	}

	return listSnsTopics(ctx, sns.NewFromConfig(cfg))
}

// runSnsListSubscriptions executes the list-subscriptions command for SNS.
func runSnsListSubscriptions(ctx context.Context, topicArn string) error {
	cfg, err := PrepareAWSConfig(ctx)
	if err != nil {
		return err
	}

	return listSnsSubscriptions(ctx, topicArn, sns.NewFromConfig(cfg))
}

// listSnsTopics lists SNS topics using the provided API client.
func listSnsTopics(ctx context.Context, api snsTopicsLister) error {
	paginator := sns.NewListTopicsPaginator(api, &sns.ListTopicsInput{})

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	hasTopics := false

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			_ = tw.Flush()
			return fmt.Errorf("listing SNS topics: %w", err)
		}

		for _, topic := range output.Topics {
			if !hasTopics {
				fmt.Fprint(tw, "TOPIC ARN\n")

				hasTopics = true
			}

			fmt.Fprintf(tw, "%s\n",
				derefString(topic.TopicArn),
			)
		}
	}

	if !hasTopics {
		_, _ = fmt.Fprintln(os.Stderr, "No SNS topics found.")
		return nil
	}

	return tw.Flush()
}

// listSnsSubscriptions lists SNS subscriptions using the provided API client.
func listSnsSubscriptions(ctx context.Context, topicArn string, api snsSubscriptionsLister) error {
	paginator := sns.NewListSubscriptionsByTopicPaginator(api, &sns.ListSubscriptionsByTopicInput{
		TopicArn: &topicArn,
	})

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	hasSubs := false

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			_ = tw.Flush()
			return fmt.Errorf("listing SNS subscriptions: %w", err)
		}

		for _, sub := range output.Subscriptions {
			if !hasSubs {
				fmt.Fprint(tw, "TOPIC ARN\tPROTOCOL\tENDPOINT\tOWNER\n")

				hasSubs = true
			}

			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
				derefString(sub.TopicArn),
				derefString(sub.Protocol),
				derefString(sub.Endpoint),
				derefString(sub.Owner),
			)
		}
	}

	if !hasSubs {
		_, _ = fmt.Fprintln(os.Stderr, "No subscriptions found.")
		return nil
	}

	return tw.Flush()
}
