package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSTSClient implements stsCallerIdentityAPI for testing.
type mockSTSClient struct {
	output *sts.GetCallerIdentityOutput
	err    error
}

func (m *mockSTSClient) GetCallerIdentity(
	_ context.Context,
	_ *sts.GetCallerIdentityInput,
	_ ...func(*sts.Options),
) (*sts.GetCallerIdentityOutput, error) {
	return m.output, m.err
}

func TestRunWhoami(t *testing.T) {
	tests := []struct {
		name       string
		client     stsCallerIdentityAPI
		want       string
		wantStderr string
		wantErr    bool
	}{
		{
			name: "successful identity",
			client: &mockSTSClient{
				output: &sts.GetCallerIdentityOutput{
					Account: aws.String("123456789012"),
					Arn:     aws.String("arn:aws:iam::123456789012:user/testuser"),
					UserId:  aws.String("AIDAJEXAMPLE"),
				},
			},
			want: "Account:  123456789012\n" +
				"Arn:      arn:aws:iam::123456789012:user/testuser\n" +
				"UserId:   AIDAJEXAMPLE\n",
		},
		{
			name: "API error",
			client: &mockSTSClient{
				err: errors.New("access denied"),
			},
			wantErr: true,
		},
		{
			name: "nil fields in output",
			client: &mockSTSClient{
				output: &sts.GetCallerIdentityOutput{},
			},
			want: "Account:  \nArn:      \nUserId:   \n",
		},
		{
			name: "no credentials found",
			client: &mockSTSClient{
				err: fmt.Errorf(
					"operation error STS: %w",
					errors.New("failed to refresh cached credentials"),
				),
			},
			wantStderr: noCredentialsMessage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, runErr := captureWhoamiOutput(t, tt.client)

			if tt.wantErr {
				assert.Error(t, runErr)

				return
			}

			require.NoError(t, runErr)
			assert.Equal(t, tt.want, stdout)
			assert.Equal(t, tt.wantStderr, stderr)
		})
	}
}

// captureWhoamiOutput runs runWhoami while capturing stdout and stderr.
func captureWhoamiOutput(t *testing.T, client stsCallerIdentityAPI) (stdout, stderr string, runErr error) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)

	stderrR, stderrW, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = stdoutW
	os.Stderr = stderrW

	t.Cleanup(func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	})

	runErr = runWhoami(context.Background(), client)

	stdoutW.Close()
	stderrW.Close()

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var stdoutBuf, stderrBuf bytes.Buffer

	_, err = io.Copy(&stdoutBuf, stdoutR)
	require.NoError(t, err)

	_, err = io.Copy(&stderrBuf, stderrR)
	require.NoError(t, err)

	return stdoutBuf.String(), stderrBuf.String(), runErr
}

func TestIsCredentialError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "cached credentials error",
			err:  errors.New("failed to refresh cached credentials"),
			want: true,
		},
		{
			name: "no IMDS role",
			err:  errors.New("no EC2 IMDS role found"),
			want: true,
		},
		{
			name: "no providers",
			err:  errors.New("no credential providers"),
			want: true,
		},
		{
			name: "anonymous credentials",
			err:  errors.New("AnonymousCredentials"),
			want: true,
		},
		{
			name: "wrapped credential error",
			err: fmt.Errorf(
				"operation error: %w",
				errors.New("failed to refresh cached credentials"),
			),
			want: true,
		},
		{
			name: "unrelated error",
			err:  errors.New("network timeout"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isCredentialError(tt.err))
		})
	}
}

func TestDerefString(t *testing.T) {
	tests := []struct {
		name  string
		input *string
		want  string
	}{
		{name: "non-nil", input: aws.String("hello"), want: "hello"},
		{name: "nil", input: nil, want: ""},
		{name: "empty string", input: aws.String(""), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, derefString(tt.input))
		})
	}
}
