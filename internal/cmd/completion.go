package cmd

import (
	"github.com/spf13/cobra"
)

// newCompletionCmd creates a command for generating shell completion scripts.
func newCompletionCmd(root *cobra.Command) *cobra.Command {
	completionCmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell autocompletion scripts",
		Long: `Generate shell autocompletion scripts for aws-probe.

The generated script depends on your shell and can be loaded
for the current session or installed in your shell configuration.`,
		Example: `  aws-probe completion bash
  aws-probe completion zsh
  aws-probe completion fish
  aws-probe completion powershell`,
		Args: cobra.NoArgs,
	}

	completionCmd.AddCommand(
		&cobra.Command{
			Use:   "bash",
			Short: "Generate Bash completion script",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				return root.GenBashCompletionV2(cmd.OutOrStdout(), true)
			},
		},
		&cobra.Command{
			Use:   "zsh",
			Short: "Generate Zsh completion script",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				return root.GenZshCompletion(cmd.OutOrStdout())
			},
		},
		&cobra.Command{
			Use:   "fish",
			Short: "Generate Fish completion script",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				return root.GenFishCompletion(cmd.OutOrStdout(), true)
			},
		},
		&cobra.Command{
			Use:   "powershell",
			Short: "Generate PowerShell completion script",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				return root.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			},
		},
	)

	return completionCmd
}
