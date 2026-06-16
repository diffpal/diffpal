package cmd

import (
	"github.com/spf13/cobra"
)

var (
	rootConfigDir string
	rootProfile   string
)

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "diffpal",
		Short: "DiffPal CLI for local and CI code review",
		Long:  "DiffPal runs provider-backed review locally and emits host-specific artifacts for GitHub, GitLab, and Azure DevOps.",
	}
	root.PersistentFlags().StringVar(&rootConfigDir, "config-dir", "", "extra config root directory (highest priority)")
	root.PersistentFlags().StringVar(&rootProfile, "profile", "", "config profile name")

	root.AddCommand(
		newInitCommand(),
		newReviewCommand(),
		newSARIFCommand(),
		newDoctorCommand(),
		newDebugCommand(),
		newVersionCommand(),
	)

	return root
}
