package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// Lister defines the interface for listing secrets.
type Lister interface {
	ListSecrets(
		ctx context.Context,
		params *secretsmanager.ListSecretsInput,
		optFns ...func(*secretsmanager.Options),
	) (*secretsmanager.ListSecretsOutput, error)
}

// Getter defines the interface for getting secret values.
type Getter interface {
	GetSecretValue(
		ctx context.Context,
		params *secretsmanager.GetSecretValueInput,
		optFns ...func(*secretsmanager.Options),
	) (*secretsmanager.GetSecretValueOutput, error)
}

// ListSecrets lists Secrets Manager secrets using the provided API client.
func ListSecrets(ctx context.Context, api Lister, w io.Writer) error {
	var (
		allSecrets []string
		allArns    []string
	)

	input := &secretsmanager.ListSecretsInput{}

	for {
		output, err := api.ListSecrets(ctx, input)
		if err != nil {
			return fmt.Errorf("listing secrets: %w", err)
		}

		for _, s := range output.SecretList {
			if s.Name == nil || s.ARN == nil {
				continue
			}

			allSecrets = append(allSecrets, *s.Name)
			allArns = append(allArns, *s.ARN)
		}

		if output.NextToken == nil || *output.NextToken == "" {
			break
		}

		input.NextToken = output.NextToken
	}

	if len(allSecrets) == 0 {
		fmt.Fprintln(w, "No secrets found.")

		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	fmt.Fprint(tw, "NAME\tARN\n")

	for i := range allSecrets {
		fmt.Fprintf(tw, "%s\t%s\n", allSecrets[i], allArns[i])
	}

	return tw.Flush()
}

// GetSecretValue retrieves and displays a secret value.
func GetSecretValue(ctx context.Context, api Getter, secretID string, w io.Writer) error {
	output, err := api.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretID,
	})
	if err != nil {
		return fmt.Errorf("getting secret value: %w", err)
	}

	if output.SecretString != nil {
		secretStr := *output.SecretString

		if isJSON(secretStr) {
			var prettyJSON []byte

			prettyJSON, err = json.MarshalIndent(json.RawMessage(secretStr), "", "  ")
			if err == nil {
				fmt.Fprintln(w, string(prettyJSON))

				return nil
			}
		}

		fmt.Fprintln(w, secretStr)
	}

	return nil
}

func isJSON(s string) bool {
	var js json.RawMessage

	return json.Unmarshal([]byte(s), &js) == nil
}

// NewClient creates a new Secrets Manager client.
func NewClient(cfg aws.Config) *secretsmanager.Client {
	return secretsmanager.NewFromConfig(cfg)
}
