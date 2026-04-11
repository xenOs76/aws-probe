package cmd

import (
	"github.com/spf13/cobra"
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
	}

	rootCmd.AddCommand(newWhoamiCmd())
	rootCmd.AddCommand(newListCmd())

	return rootCmd
}

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}
