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
			if err := writeTopicRow(tw, &hasTopics, awsutil.DerefString(topic.TopicArn)); err != nil {
				return err
			}
		}
	}

	if !hasTopics {
		if _, err := fmt.Fprintln(w, "No SNS topics found."); err != nil {
			return err
		}

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
			err := writeSubscriptionRow(
				tw, &hasSubs, sub.TopicArn, sub.Protocol, sub.Endpoint, sub.Owner,
			)
			if err != nil {
				return err
			}
		}
	}

	if !hasSubs {
		if _, err := fmt.Fprintln(w, "No subscriptions found."); err != nil {
			return err
		}

		return nil
	}

	return tw.Flush()
}

func writeTopicRow(tw *tabwriter.Writer, hasTopics *bool, topicArn string) error {
	if !*hasTopics {
		if _, err := fmt.Fprint(tw, "TOPIC ARN\n"); err != nil {
			return err
		}

		*hasTopics = true
	}

	_, err := fmt.Fprintf(tw, "%s\n", topicArn)

	return err
}

func writeSubscriptionRow(
	tw *tabwriter.Writer,
	hasSubs *bool,
	topicArn, protocol, endpoint, owner *string,
) error {
	if !*hasSubs {
		if _, err := fmt.Fprint(tw, "TOPIC ARN\tPROTOCOL\tENDPOINT\tOWNER\n"); err != nil {
			return err
		}

		*hasSubs = true
	}

	_, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
		awsutil.DerefString(topicArn),
		awsutil.DerefString(protocol),
		awsutil.DerefString(endpoint),
		awsutil.DerefString(owner),
	)

	return err
}

// NewClient creates a new SNS client.
func NewClient(cfg aws.Config) *sns.Client {
	return sns.NewFromConfig(cfg)
}
