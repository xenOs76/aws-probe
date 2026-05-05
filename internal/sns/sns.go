package sns

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/xenos76/aws-probe/internal/awsutil"
)

// TopicsLister defines the interface for listing SNS topics.
type TopicsLister interface {
	ListTopics(
		ctx context.Context,
		params *sns.ListTopicsInput,
		optFns ...func(*sns.Options),
	) (*sns.ListTopicsOutput, error)
}

// SubscriptionsLister defines the interface for listing SNS subscriptions.
type SubscriptionsLister interface {
	ListSubscriptionsByTopic(
		ctx context.Context,
		params *sns.ListSubscriptionsByTopicInput,
		optFns ...func(*sns.Options),
	) (*sns.ListSubscriptionsByTopicOutput, error)
}

// ListTopics lists SNS topics using the provided API client.
func ListTopics(ctx context.Context, api TopicsLister, w io.Writer) error {
	paginator := sns.NewListTopicsPaginator(api, &sns.ListTopicsInput{})

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

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
				awsutil.DerefString(topic.TopicArn),
			)
		}
	}

	if !hasTopics {
		fmt.Fprintln(w, "No SNS topics found.")

		return nil
	}

	return tw.Flush()
}

// ListSubscriptions lists SNS subscriptions using the provided API client.
func ListSubscriptions(ctx context.Context, topicArn string, api SubscriptionsLister, w io.Writer) error {
	paginator := sns.NewListSubscriptionsByTopicPaginator(api, &sns.ListSubscriptionsByTopicInput{
		TopicArn: &topicArn,
	})

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

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
				awsutil.DerefString(sub.TopicArn),
				awsutil.DerefString(sub.Protocol),
				awsutil.DerefString(sub.Endpoint),
				awsutil.DerefString(sub.Owner),
			)
		}
	}

	if !hasSubs {
		fmt.Fprintln(w, "No subscriptions found.")

		return nil
	}

	return tw.Flush()
}

// NewClient creates a new SNS client.
func NewClient(cfg aws.Config) *sns.Client {
	return sns.NewFromConfig(cfg)
}
