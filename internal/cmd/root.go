package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var DefaultAWSRegion = func() string {
	if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		return region
	}
	return "eu-central-1"
}()

var Version = "dev"

// newRootCmd creates the top-level aws-probe command.
func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "aws-probe",
		Short: "AWS diagnostic and inspection toolkit",
		Long: `aws-probe is a CLI toolkit for inspecting and diagnosing
your AWS environment. It provides subcommands to query
AWS APIs and display useful information about your account,
resources, and configuration.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
	}

	rootCmd.AddCommand(newWhoamiCmd())
	rootCmd.AddCommand(newS3Cmd())
	rootCmd.AddCommand(newSqsCmd())
	rootCmd.AddCommand(newSecretsCmd())
	rootCmd.AddCommand(newMskCmd())

	return rootCmd
}

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}
