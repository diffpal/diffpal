package cmd

import (
	"github.com/diffpal/diffpal/internal/reviewer"
	"github.com/spf13/cobra"
)

func newUncommittedReviewSubcommand(run reviewRunner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uncommitted",
		Short: "Review uncommitted changes with the configured provider",
		Long:  "Ask the configured provider to review the uncommitted state in its backend-provided workspace snapshot.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runReviewOnlyWithMode(cmd, "uncommitted", run, reviewer.ModeUncommitted)
		},
	}
	addReviewAnalysisFlags(cmd, "uncommitted")
	addReviewPolicyFlags(cmd)
	addReviewFeedbackFlag(cmd)
	return cmd
}
