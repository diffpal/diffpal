package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/diffpal/diffpal/internal/config"
	"github.com/diffpal/diffpal/internal/findings"
	"github.com/diffpal/diffpal/internal/platform/azure"
	"github.com/diffpal/diffpal/internal/platform/github"
	gitlabpub "github.com/diffpal/diffpal/internal/platform/gitlab"
	"github.com/diffpal/diffpal/internal/platformauth"
	"github.com/diffpal/diffpal/internal/reviewer"
	"github.com/spf13/cobra"
)

type reviewRunner func(context.Context, config.Config, reviewer.Options) (reviewer.Result, error)

type reviewExecution struct {
	Config        config.Config
	Repo          string
	OutPath       string
	BlockOn       string
	Gate          bool
	ReviewChannel string
	Result        reviewer.Result
}

func newReviewCommand() *cobra.Command {
	return newReviewCommandWithRunner(reviewer.Run)
}

func newReviewCommandWithRunner(run reviewRunner) *cobra.Command {
	review := &cobra.Command{
		Use:   "review",
		Short: "Run review locally or for a CI host",
		Long:  "Run DiffPal review locally or emit host-specific review artifacts for GitHub, GitLab, or Azure DevOps.",
	}

	review.AddCommand(
		newLocalReviewSubcommand(run),
		newHostReviewSubcommand(run, "github", "github", nil),
		newHostReviewSubcommand(run, "gitlab", "gitlab", nil),
		newHostReviewSubcommand(run, "ado", "azure", []string{"azure"}),
	)

	silenceRuntimeErrorOutput(review)
	return review
}

func newLocalReviewSubcommand(run reviewRunner) *cobra.Command {
	local := &cobra.Command{
		Use:   "local",
		Short: "Review a local diff and write findings.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReviewOnly(cmd, "local", run)
		},
	}
	addReviewAnalysisFlags(local, "local")
	addReviewPolicyFlags(local)
	return local
}

func newHostReviewSubcommand(run reviewRunner, name, platform string, aliases []string) *cobra.Command {
	host := &cobra.Command{
		Use:     name,
		Aliases: aliases,
		Short:   fmt.Sprintf("Review and emit %s artifacts", hostDisplayName(name)),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHostReview(cmd, platform, name, run)
		},
	}
	addReviewAnalysisFlags(host, name)
	addReviewPolicyFlags(host)
	addReviewPublishFlags(host)
	if platform == "github" {
		addGitHubReviewPublishFlags(host)
	}
	return host
}

func addReviewAnalysisFlags(cmd *cobra.Command, defaultReviewID string) {
	cmd.Flags().String("base", "", "Base revision")
	cmd.Flags().String("head", "", "Head revision")
	cmd.Flags().String("language", "", "Language for generated review findings")
	cmd.Flags().String("instructions", "", "Additional review instructions for local prompt tuning")
	cmd.Flags().String("instructions-file", "", "Path to additional review instructions")
	cmd.Flags().String("out", findings.DefaultBundlePath, "Output findings bundle path")
	cmd.Flags().String("repo", "local", "Repository id for deterministic fingerprints")
	cmd.Flags().String("review-id", defaultReviewID, "Review identifier for deterministic outputs")
}

func addReviewPublishFlags(cmd *cobra.Command) {
	cmd.Flags().String("mode", "", "Comma-separated publish modes for the selected host")
	cmd.Flags().String("feedback", string(FeedbackBalanced), "Review feedback shape: summary, balanced, or inline")
	cmd.Flags().Bool("summary-overview", true, "Include a semantic change overview in review summaries")
	cmd.Flags().Bool("dry-run", false, "Print the host review markdown without publishing")
}

func addGitHubReviewPublishFlags(cmd *cobra.Command) {
	cmd.Flags().String("review-channel", github.DefaultReviewChannel, "GitHub publishing channel used for check runs and pull request reviews")
}

func addReviewPolicyFlags(cmd *cobra.Command) {
	cmd.Flags().String("block-on", "high", "Mark findings at this severity and above as blocking")
	cmd.Flags().Bool("gate", false, "Return non-zero when blocking findings exist")
}

func runReviewOnly(cmd *cobra.Command, defaultReviewID string, run reviewRunner) error {
	execution, err := executeReview(cmd, defaultReviewID, run)
	if err != nil {
		return err
	}
	blocking := countBlockingFindings(execution.Result.Bundle)
	if execution.Gate && blocking > 0 {
		return newReviewBlockedError(blocking)
	}
	if blocking > 0 {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "status=blocked blocking=%d\n", blocking); err != nil {
			return withExitCode(5, err)
		}
		return nil
	}
	return nil
}

func runHostReview(cmd *cobra.Command, platform, defaultReviewID string, run reviewRunner) error {
	modeSpec, _ := cmd.Flags().GetString("mode")
	modes := parseModeList(modeSpec)
	feedback, _ := cmd.Flags().GetString("feedback")
	summaryOverview, _ := cmd.Flags().GetBool("summary-overview")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun && platform != "github" {
		return withExitCode(2, fmt.Errorf("--dry-run is only supported for github review"))
	}
	if len(modes) == 0 {
		if _, err := normalizeFeedback(feedback); err != nil {
			return withExitCode(2, err)
		}
	}
	if cmd.Flags().Lookup("review-channel") != nil {
		reviewChannel, _ := cmd.Flags().GetString("review-channel")
		if _, err := github.NewReviewIdentity(reviewChannel); err != nil {
			return withExitCode(2, err)
		}
	}
	if platform == "github" {
		skipReview, err := shouldSkipGitHubReview(cmd)
		if err != nil {
			return withExitCode(4, err)
		}
		if skipReview {
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "publish=skipped-fork"); err != nil {
				return withExitCode(5, err)
			}
			return nil
		}
	}

	execution, err := executeReview(cmd, defaultReviewID, run)
	if err != nil {
		return err
	}

	bundle, err := normalizeBundleBlocking(execution.Result.Bundle, execution.BlockOn)
	if err != nil {
		return withExitCode(2, err)
	}
	blocking := countBlockingFindings(bundle)

	if dryRun {
		resolvedModes, profile, err := resolvePublishModes(platform, modes, feedback)
		if err != nil {
			return withExitCode(2, err)
		}
		summary, err := renderPublishSummary(platform, bundle, profile, resolvedModes, summaryOverview, execution.ReviewChannel, execution.Repo)
		if err != nil {
			return withExitCode(2, err)
		}
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(summary)); err != nil {
			return withExitCode(5, err)
		}
		if execution.Gate && blocking > 0 {
			return newReviewBlockedError(blocking)
		}
		return nil
	}

	if platform == "github" {
		skipPublish, err := shouldSkipGitHubPublish(bundle)
		if err != nil {
			return withExitCode(4, err)
		}
		if skipPublish {
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), "publish=skipped-fork"); err != nil {
				return withExitCode(5, err)
			}
			if execution.Gate && blocking > 0 {
				return newReviewBlockedError(blocking)
			}
			if blocking > 0 {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "status=blocked blocking=%d\n", blocking); err != nil {
					return withExitCode(5, err)
				}
				return nil
			}
			return nil
		}
	}

	outputs, _, err := publishBundleToFiles(platform, bundle, execution.Repo, execution.BlockOn, execution.Gate, modes, feedback, summaryOverview, "", execution.ReviewChannel)
	if err != nil {
		return withExitCode(4, err)
	}
	auth, err := platformauth.Resolve(execution.Config, platform)
	if err != nil {
		return withExitCode(2, err)
	}
	if err := publishBundleToAPI(cmd.Context(), auth, platform, execution.Config, bundle, execution.Repo, execution.BlockOn, execution.Gate, modes, feedback, summaryOverview, execution.ReviewChannel); err != nil {
		return withExitCode(4, err)
	}
	for _, item := range outputs {
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "mode=%s path=%s\n", item.Mode, item.Path)
		if err != nil {
			return withExitCode(5, err)
		}
	}
	if execution.Gate && blocking > 0 {
		return newReviewBlockedError(blocking)
	}
	if blocking > 0 {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "status=blocked blocking=%d\n", blocking); err != nil {
			return withExitCode(5, err)
		}
		return nil
	}
	return nil
}

func executeReview(cmd *cobra.Command, defaultReviewID string, run reviewRunner) (reviewExecution, error) {
	base, _ := cmd.Flags().GetString("base")
	head, _ := cmd.Flags().GetString("head")
	language, _ := cmd.Flags().GetString("language")
	instructionsFlag, _ := cmd.Flags().GetString("instructions")
	instructionsFile, _ := cmd.Flags().GetString("instructions-file")
	outPath, _ := cmd.Flags().GetString("out")
	repo, _ := cmd.Flags().GetString("repo")
	reviewID, _ := cmd.Flags().GetString("review-id")
	reviewChannel := github.DefaultReviewChannel
	if cmd.Flags().Lookup("review-channel") != nil {
		reviewChannel, _ = cmd.Flags().GetString("review-channel")
		if _, err := github.NewReviewIdentity(reviewChannel); err != nil {
			return reviewExecution{}, withExitCode(2, err)
		}
	}

	cfg, err := loadRequiredConfig()
	if err != nil {
		return reviewExecution{}, withExitCode(2, err)
	}
	if !cmd.Flags().Changed("language") {
		language = cfg.ReviewLanguage()
	}
	instructions := cfg.ReviewInstructions()
	if cmd.Flags().Changed("instructions") {
		instructions = instructionsFlag
	}
	if cmd.Flags().Changed("instructions-file") {
		fileInstructions, err := readReviewInstructionsFile(instructionsFile)
		if err != nil {
			return reviewExecution{}, withExitCode(2, err)
		}
		instructions = joinReviewInstructions(instructions, fileInstructions)
	}
	blockOn := cfg.BlockOn()
	flagValue, _ := cmd.Flags().GetString("block-on")
	if cmd.Flags().Changed("block-on") {
		blockOn = flagValue
	} else if blockOn == "" {
		blockOn = flagValue
	}
	gate, _ := cmd.Flags().GetBool("gate")
	if reviewID == "" {
		reviewID = defaultReviewID
	}
	if outPath == "" {
		outPath = findings.DefaultBundlePath
	}
	runCtx := cmd.Context()
	if runCtx == nil {
		runCtx = context.Background()
	}
	result, err := run(runCtx, cfg, reviewer.Options{
		Repo:          repo,
		ReviewID:      reviewID,
		BaseSHA:       base,
		HeadSHA:       head,
		BlockOn:       blockOn,
		Language:      language,
		ReviewTimeout: cfg.ReviewTimeout(),
		Instructions:  instructions,
	})
	if err != nil {
		return reviewExecution{}, reviewExitError(err)
	}
	if err := findings.WriteBundle(outPath, result.Bundle, repo); err != nil {
		return reviewExecution{}, withExitCode(5, err)
	}
	if !reviewDryRun(cmd) {
		if err := emitReviewSummary(cmd, result, outPath); err != nil {
			return reviewExecution{}, err
		}
	}

	return reviewExecution{
		Config:        cfg,
		Repo:          repo,
		OutPath:       outPath,
		BlockOn:       blockOn,
		Gate:          gate,
		ReviewChannel: reviewChannel,
		Result:        result,
	}, nil
}

func reviewDryRun(cmd *cobra.Command) bool {
	if cmd.Flags().Lookup("dry-run") == nil {
		return false
	}
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	return dryRun
}

func readReviewInstructionsFile(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("--instructions-file path is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read instructions file %q: %w", path, err)
	}
	return strings.TrimSpace(string(raw)), nil
}

func joinReviewInstructions(items ...string) string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return strings.Join(out, "\n\n")
}

func emitReviewSummary(cmd *cobra.Command, result reviewer.Result, outPath string) error {
	if result.Bundle.Prompt != nil && result.Bundle.Prompt.PromptID != "" {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "prompt_id=%s\n", result.Bundle.Prompt.PromptID)
		if err != nil {
			return withExitCode(5, err)
		}
	}
	if result.Bundle.Prompt != nil && result.Bundle.Prompt.PromptVersion != "" {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "prompt_version=%s\n", result.Bundle.Prompt.PromptVersion)
		if err != nil {
			return withExitCode(5, err)
		}
	}
	if result.Bundle.Prompt != nil && result.Bundle.Prompt.SchemaVersion != "" {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "prompt_schema_version=%s\n", result.Bundle.Prompt.SchemaVersion)
		if err != nil {
			return withExitCode(5, err)
		}
	}
	for _, file := range result.Files {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s -> %s\n", file.FromPath, file.ToPath)
		if err != nil {
			return withExitCode(5, err)
		}
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "files=%d\n", result.ChangedFiles)
	if err != nil {
		return withExitCode(5, err)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "findings=%d\n", len(result.Bundle.Findings))
	if err != nil {
		return withExitCode(5, err)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "bundle=%s\n", outPath)
	if err != nil {
		return withExitCode(5, err)
	}
	return nil
}

func countBlockingFindings(bundle findings.FindingsBundle) int {
	count := 0
	for _, item := range bundle.Findings {
		if item.Blocking {
			count++
		}
	}
	return count
}

func shouldSkipGitHubPublish(bundle findings.FindingsBundle) (bool, error) {
	ctx, err := github.ResolveContext(bundle.BaseSHA, bundle.HeadSHA)
	if err != nil {
		return false, fmt.Errorf("resolve github context for fork safety: %w", err)
	}
	if !ctx.IsFork {
		return false, nil
	}
	return true, nil
}

func shouldSkipGitHubReview(cmd *cobra.Command) (bool, error) {
	base, _ := cmd.Flags().GetString("base")
	head, _ := cmd.Flags().GetString("head")
	ctx, err := github.ResolveContext(base, head)
	if err != nil {
		return false, fmt.Errorf("resolve github context for fork safety: %w", err)
	}
	return ctx.IsFork, nil
}

func publishBundleToAPI(ctx context.Context, auth platformauth.Resolved, platform string, cfg config.Config, bundle findings.FindingsBundle, repo string, blockOn string, gateEnabled bool, modes []string, feedback string, summaryOverview bool, reviewChannel string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var err error
	bundle, err = normalizeBundleBlocking(bundle, blockOn)
	if err != nil {
		return err
	}
	resolvedModes, profile, err := resolvePublishModes(platform, modes, feedback)
	if err != nil {
		return err
	}
	modes = resolvedModes
	summary, err := renderPublishSummary(platform, bundle, profile, modes, summaryOverview, reviewChannel, repo)
	if err != nil {
		return err
	}

	switch platform {
	case "github":
		reviewCtx, err := github.ResolveContext(bundle.BaseSHA, bundle.HeadSHA)
		if err != nil {
			return err
		}
		identity, err := github.NewReviewIdentity(reviewChannel)
		if err != nil {
			return err
		}
		return auth.WithToken(func(token string) error {
			publishReview := false
			includeInline := false
			for _, mode := range modes {
				switch normalizePublishMode(platform, mode) {
				case "github_comments":
					publishReview = true
					includeInline = true
				case "summary":
					publishReview = true
				}
			}
			if publishReview {
				plan := github.CommentPlan{}
				if includeInline {
					inlineFindings := publishableInlineFindings(bundle.Findings)
					if err := github.ValidateInlineFindings(inlineFindings); err != nil {
						return err
					}
					existing := github.ActiveReviewThreadState(ctx, token, reviewCtx, identity, inlineFindings, nil)
					plan = github.PlanInlineCommentsWithOptions(existing, inlineFindings, github.CommentOptions{
						Profile:     string(profile),
						AllFindings: true,
					})
				}
				// GitHub Actions cannot approve pull requests with GITHUB_TOKEN.
				// DiffPal publishes comment reviews; --gate is enforced by the
				// workflow exit status instead of sticky PR review state.
				if err := github.PublishPullRequestReviewWithIdentity(ctx, token, reviewCtx, summary, identity, plan, nil); err != nil {
					return err
				}
			}
			return nil
		})
	case "gitlab":
		reviewCtx, err := gitlabpub.ResolveContext(bundle.BaseSHA, bundle.HeadSHA, "", "")
		if err != nil {
			return err
		}
		existing, err := gitlabpub.LoadExistingState(defaultModeOutput(platform, "discussions"))
		if err != nil {
			return err
		}
		blockThresholds := []string{blockOn}
		plan := gitlabpub.PlanDiscussions(existing, bundle.Findings, blockThresholds)
		return auth.WithToken(func(token string) error {
			for _, mode := range modes {
				switch normalizePublishMode(platform, mode) {
				case "discussions":
					if err := gitlabpub.PublishDiscussions(ctx, auth.Mode, token, reviewCtx, plan, nil); err != nil {
						return err
					}
				case "summary":
					if err := gitlabpub.PublishSummaryDiscussion(ctx, auth.Mode, token, reviewCtx, summary, nil); err != nil {
						return err
					}
				}
			}
			return nil
		})
	case "azure":
		reviewCtx, err := azure.ResolveContext(bundle.BaseSHA, bundle.HeadSHA)
		if err != nil {
			return err
		}
		existing, err := azure.LoadExistingState(defaultModeOutput(platform, "threads"))
		if err != nil {
			return err
		}
		plan := azure.PlanThreadsWithProfile(existing, bundle.Findings, reviewCtx, string(profile))
		blocking := countBlockingFindings(bundle)
		status := azure.PolicyStatus(azure.PolicyContext{BlockOn: blockOn, GateEnabled: gateEnabled, FatalOnFailures: true}, blocking, len(bundle.Findings)-blocking, false)
		return auth.WithToken(func(token string) error {
			for _, mode := range modes {
				switch normalizePublishMode(platform, mode) {
				case "threads":
					if err := azure.PublishThreads(ctx, auth.Mode, token, reviewCtx, plan, nil); err != nil {
						return err
					}
				case "status":
					if err := azure.PublishStatus(ctx, auth.Mode, token, reviewCtx, status, nil); err != nil {
						return err
					}
				case "summary":
					if err := azure.PublishSummaryThread(ctx, auth.Mode, token, reviewCtx, summary, nil); err != nil {
						return err
					}
				}
			}
			if gateEnabled {
				vote := 10
				if blocking > 0 {
					vote = -5
				}
				if err := azure.PublishGateVote(ctx, auth.Mode, token, reviewCtx, vote, nil); err != nil {
					return err
				}
			}
			return nil
		})
	default:
		return fmt.Errorf("unsupported platform %q", platform)
	}
}

func hostDisplayName(name string) string {
	switch name {
	case "ado":
		return "Azure DevOps"
	case "gitlab":
		return "GitLab"
	default:
		return "GitHub"
	}
}
