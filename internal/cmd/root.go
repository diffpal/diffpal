package cmd

import (
	"github.com/diffpal/diffpal/internal/logging"
	"github.com/spf13/cobra"
)

var (
	rootConfigDir string
	rootProfile   string
)

func NewRootCommand() *cobra.Command {
	var debug bool
	root := &cobra.Command{
		Use:           "diffpal",
		Short:         "DiffPal CLI for local and CI code review",
		Long:          "DiffPal runs provider-backed review locally and emits host-specific artifacts for GitHub, GitLab, and Azure DevOps.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&rootConfigDir, "config-dir", "", "extra config root directory (highest priority)")
	root.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")
	root.PersistentFlags().StringVar(&rootProfile, "profile", "", "config profile name")
	root.PersistentPreRun = func(cmd *cobra.Command, _ []string) {
		cmd.SetContext(logging.Init(cmd.Context(), debug))
	}

	root.AddCommand(
		newInitCommand(),
		newReviewCommand(),
		newSARIFCommand(),
		newDoctorCommand(),
		newDebugCommand(),
		newVersionCommand(),
	)

	silenceRuntimeErrorOutput(root)
	return root
}

func silenceRuntimeErrorOutput(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	for _, child := range cmd.Commands() {
		silenceRuntimeErrorOutput(child)
	}
}
