package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/xenos76/aws-probe/internal/awsutil"
)

// PrepareAWSConfig is a variable that points to awsutil.PrepareAWSConfig.
// It is a variable to allow mocking in tests.
var PrepareAWSConfig = awsutil.PrepareAWSConfig

// DefaultAWSRegion is the default region used when no region is specified.
var DefaultAWSRegion = func() string {
	if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		return region
	}

	return "eu-central-1"
}()

// Version is the current version of the aws-probe tool.
var Version = "dev"

var (
	// OutputFormat specifies the output format for all commands.
	OutputFormat string
	// Theme specifies the color theme for table output.
	Theme string
)

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
	rootCmd.AddCommand(newSnsCmd())
	rootCmd.AddCommand(newCloudfrontCmd())

	rootCmd.PersistentFlags().StringVarP(&OutputFormat, "output", "o", "table", "Output format (table, csv, json)")
	rootCmd.PersistentFlags().StringVarP(&Theme, "theme", "t", "catppuccin-frappe",
		"Output theme (catppuccin-frappe, dracula, nord, none)")

	return rootCmd
}

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}
