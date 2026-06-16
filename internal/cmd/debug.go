package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/diffpal/diffpal/internal/config"
	"github.com/diffpal/diffpal/internal/findings"
	"github.com/diffpal/diffpal/internal/reviewer"
	"github.com/spf13/cobra"
)

func newDebugCommand() *cobra.Command {
	debug := &cobra.Command{
		Use:   "debug",
		Short: "Inspect DiffPal runtime artifacts without a provider",
	}
	debug.AddCommand(newDebugPromptCommand())
	return debug
}

func newDebugPromptCommand() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "prompt",
		Short: "Render review prompts and task snapshots offline",
		RunE: func(cmd *cobra.Command, args []string) error {
			var debugResult reviewer.DebugResult
			runner := func(ctx context.Context, cfg config.Config, opts reviewer.Options) (reviewer.Result, error) {
				var err error
				debugResult, err = reviewer.DebugPrompt(ctx, cfg, opts)
				if err != nil {
					return reviewer.Result{}, err
				}
				return reviewer.Result{Bundle: debugResult.Bundle}, nil
			}
			if _, err := executeReview(cmd, "debug", runner); err != nil {
				return err
			}
			switch strings.ToLower(strings.TrimSpace(format)) {
			case "", "text":
				return printDebugPromptText(cmd, debugResult)
			case "json":
				return printDebugPromptJSON(cmd, debugResult)
			default:
				return withExitCode(2, fmt.Errorf("unsupported debug format %q", format))
			}
		},
	}
	addReviewAnalysisFlags(cmd, "debug")
	addReviewPolicyFlags(cmd)
	cmd.Flags().Bool("dry-run", true, "Suppress normal review summary output")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text or json")
	return cmd
}

func printDebugPromptText(cmd *cobra.Command, result reviewer.DebugResult) error {
	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintln(out, "## System Prompt"); err != nil {
		return withExitCode(5, err)
	}
	if _, err := fmt.Fprintln(out, result.SystemPrompt); err != nil {
		return withExitCode(5, err)
	}
	if _, err := fmt.Fprintln(out, "\n## Task Snapshot"); err != nil {
		return withExitCode(5, err)
	}
	if _, err := fmt.Fprintln(out, result.TaskSnapshot); err != nil {
		return withExitCode(5, err)
	}
	if _, err := fmt.Fprintln(out, "\n## Mock Bundle"); err != nil {
		return withExitCode(5, err)
	}
	raw, err := findings.FormatBundle(result.Bundle, "debug")
	if err != nil {
		return withExitCode(5, err)
	}
	if _, err := fmt.Fprintln(out, string(raw)); err != nil {
		return withExitCode(5, err)
	}
	return nil
}

func printDebugPromptJSON(cmd *cobra.Command, result reviewer.DebugResult) error {
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return withExitCode(5, err)
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(raw)); err != nil {
		return withExitCode(5, err)
	}
	return nil
}
