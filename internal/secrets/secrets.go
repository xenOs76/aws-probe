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

type secretEntry struct {
	name string
	arn  string
}

// ListSecrets lists Secrets Manager secrets using the provided API client.
func ListSecrets(ctx context.Context, api Lister, w io.Writer) error {
	secrets, err := fetchAllSecrets(ctx, api)
	if err != nil {
		return err
	}

	if len(secrets) == 0 {
		if _, err := fmt.Fprintln(w, "No secrets found."); err != nil {
			return err
		}

		return nil
	}

	return writeSecretsTable(w, secrets)
}

func fetchAllSecrets(ctx context.Context, api Lister) ([]secretEntry, error) {
	var secrets []secretEntry

	input := &secretsmanager.ListSecretsInput{}

	for {
		output, err := api.ListSecrets(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("listing secrets: %w", err)
		}

		for _, s := range output.SecretList {
			if s.Name == nil || s.ARN == nil {
				continue
			}

			secrets = append(secrets, secretEntry{name: *s.Name, arn: *s.ARN})
		}

		if output.NextToken == nil || *output.NextToken == "" {
			break
		}

		input.NextToken = output.NextToken
	}

	return secrets, nil
}

func writeSecretsTable(w io.Writer, secrets []secretEntry) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	if _, err := fmt.Fprint(tw, "NAME\tARN\n"); err != nil {
		return err
	}

	for _, s := range secrets {
		if _, err := fmt.Fprintf(tw, "%s\t%s\n", s.name, s.arn); err != nil {
			return err
		}
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

	if output.SecretString == nil {
		return nil
	}

	secretStr := *output.SecretString
	if isJSON(secretStr) {
		if prettyJSON, err := json.MarshalIndent(json.RawMessage(secretStr), "", "  "); err == nil {
			_, writeErr := fmt.Fprintln(w, string(prettyJSON))

			return writeErr
		}
	}

	_, writeErr := fmt.Fprintln(w, secretStr)

	return writeErr
}

func isJSON(s string) bool {
	var js json.RawMessage

	return json.Unmarshal([]byte(s), &js) == nil
}

// NewClient creates a new Secrets Manager client.
func NewClient(cfg aws.Config) *secretsmanager.Client {
	return secretsmanager.NewFromConfig(cfg)
}
