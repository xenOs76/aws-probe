package whoami

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/xenos76/aws-probe/internal/awsutil"
)

// CallerIdentity holds the information about the AWS caller identity.
type CallerIdentity struct {
	Account string
	Arn     string
	UserID  string
}

// stsClient defines the interface for the STS GetCallerIdentity operation.
type stsClient interface {
	GetCallerIdentity(
		ctx context.Context,
		params *sts.GetCallerIdentityInput,
		optFns ...func(*sts.Options),
	) (*sts.GetCallerIdentityOutput, error)
}

// GetCallerIdentity retrieves the current AWS caller identity using the provided API client.
func GetCallerIdentity(ctx context.Context, api stsClient) (*CallerIdentity, error) {
	output, err := api.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("getting caller identity: %w", err)
	}

	return &CallerIdentity{
		Account: awsutil.DerefString(output.Account),
		Arn:     awsutil.DerefString(output.Arn),
		UserID:  awsutil.DerefString(output.UserId),
	}, nil
}

// DisplayCallerIdentity retrieves and displays the current AWS caller identity.
func DisplayCallerIdentity(ctx context.Context, api stsClient, authMethod awsutil.AuthMethod, w io.Writer) error {
	identity, err := GetCallerIdentity(ctx, api)
	if err != nil {
		return err
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintf(tw, "ACCOUNT\t%s\n", identity.Account)
	fmt.Fprintf(tw, "ARN\t%s\n", identity.Arn)
	fmt.Fprintf(tw, "USER ID\t%s\n", identity.UserID)
	fmt.Fprintf(tw, "AUTH METHOD\t%s\n", authMethod.IdentitySource)

	return tw.Flush()
}

// NewSTSClient creates a new STS client from the provided AWS configuration.
func NewSTSClient(cfg aws.Config) *sts.Client {
	return sts.NewFromConfig(cfg)
}
