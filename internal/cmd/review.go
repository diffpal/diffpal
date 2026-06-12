package cmd

import (
	"context"
	"fmt"

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
	Config  config.Config
	Repo    string
	OutPath string
	BlockOn string
	Gate    bool
	Result  reviewer.Result
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
	return host
}

func addReviewAnalysisFlags(cmd *cobra.Command, defaultReviewID string) {
	cmd.Flags().String("base", "", "Base revision")
	cmd.Flags().String("head", "", "Head revision")
	cmd.Flags().Int("max-files", 0, "Maximum files from diff")
	cmd.Flags().Int("context-lines", 0, "Context lines to enrich each changed file")
	cmd.Flags().Int("max-patch-chars", 12000, "Maximum context characters per chunk")
	cmd.Flags().Int("max-files-per-chunk", 20, "Maximum files per context chunk")
	cmd.Flags().String("language", "", "Language for generated review findings")
	cmd.Flags().String("review-checks", "", "Comma-separated checks to run: bugs, performance, best-practices")
	cmd.Flags().String("out", findings.DefaultBundlePath, "Output findings bundle path")
	cmd.Flags().String("repo", "local", "Repository id for deterministic fingerprints")
	cmd.Flags().String("review-id", defaultReviewID, "Review identifier for deterministic outputs")
}

func addReviewPublishFlags(cmd *cobra.Command) {
	cmd.Flags().String("mode", "", "Comma-separated publish modes for the selected host")
	cmd.Flags().String("feedback", string(FeedbackBalanced), "Review feedback shape: summary, balanced, or inline")
	cmd.Flags().Bool("summary-overview", true, "Include a semantic change overview in review summaries")
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
		return withExitCode(1, fmt.Errorf("review blocked: blocking findings detected: %d", blocking))
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
	if len(modes) == 0 {
		if _, err := normalizeFeedback(feedback); err != nil {
			return withExitCode(2, err)
		}
	}

	execution, err := executeReview(cmd, defaultReviewID, run)
	if err != nil {
		return err
	}

	blocking := countBlockingFindings(execution.Result.Bundle)

	if platform == "github" && shouldSkipGitHubPublish(execution.Config, execution.Result.Bundle) {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "publish=skipped-fork"); err != nil {
			return withExitCode(5, err)
		}
		if execution.Gate && blocking > 0 {
			return withExitCode(1, fmt.Errorf("review blocked: blocking findings detected: %d", blocking))
		}
		if blocking > 0 {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "status=blocked blocking=%d\n", blocking); err != nil {
				return withExitCode(5, err)
			}
			return nil
		}
		return nil
	}

	outputs, _, err := publishBundleToFiles(platform, execution.Result.Bundle, execution.Repo, execution.BlockOn, modes, feedback, summaryOverview, "")
	if err != nil {
		return withExitCode(4, err)
	}
	auth, err := platformauth.Resolve(execution.Config, platform)
	if err != nil {
		return withExitCode(2, err)
	}
	if err := publishBundleToAPI(cmd.Context(), auth, platform, execution.Config, execution.Result.Bundle, execution.BlockOn, modes, feedback, summaryOverview); err != nil {
		return withExitCode(4, err)
	}
	for _, item := range outputs {
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "mode=%s path=%s\n", item.Mode, item.Path)
		if err != nil {
			return withExitCode(5, err)
		}
	}
	if execution.Gate && blocking > 0 {
		return withExitCode(1, fmt.Errorf("review blocked: blocking findings detected: %d", blocking))
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
	maxFiles, _ := cmd.Flags().GetInt("max-files")
	contextLines, _ := cmd.Flags().GetInt("context-lines")
	maxPatchChars, _ := cmd.Flags().GetInt("max-patch-chars")
	maxFilesPerChunk, _ := cmd.Flags().GetInt("max-files-per-chunk")
	language, _ := cmd.Flags().GetString("language")
	reviewChecksSpec, _ := cmd.Flags().GetString("review-checks")
	outPath, _ := cmd.Flags().GetString("out")
	repo, _ := cmd.Flags().GetString("repo")
	reviewID, _ := cmd.Flags().GetString("review-id")

	cfg, err := loadRequiredConfig()
	if err != nil {
		return reviewExecution{}, withExitCode(2, err)
	}
	if !cmd.Flags().Changed("max-files") {
		maxFiles = cfg.Review.MaxFiles
	}
	if !cmd.Flags().Changed("context-lines") {
		contextLines = cfg.Review.ContextLines
	}
	if !cmd.Flags().Changed("max-patch-chars") {
		maxPatchChars = cfg.Review.Chunking.MaxPatchChars
	}
	if !cmd.Flags().Changed("max-files-per-chunk") {
		maxFilesPerChunk = cfg.Review.Chunking.MaxFilesPerChunk
	}
	if !cmd.Flags().Changed("language") {
		language = cfg.ReviewLanguage()
	}
	var reviewChecks []string
	if cmd.Flags().Changed("review-checks") {
		var err error
		reviewChecks, err = config.NormalizeReviewChecks(parseModeList(reviewChecksSpec))
		if err != nil {
			return reviewExecution{}, withExitCode(2, err)
		}
	} else {
		reviewChecks = cfg.ReviewChecks()
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
		Repo:             repo,
		ReviewID:         reviewID,
		BaseSHA:          base,
		HeadSHA:          head,
		MaxFiles:         maxFiles,
		ContextLines:     contextLines,
		MaxPatchChars:    maxPatchChars,
		MaxFilesPerChunk: maxFilesPerChunk,
		BlockOn:          blockOn,
		Language:         language,
		ReviewChecks:     reviewChecks,
	})
	if err != nil {
		return reviewExecution{}, reviewExitError(err)
	}
	if err := findings.WriteBundle(outPath, result.Bundle, repo); err != nil {
		return reviewExecution{}, withExitCode(5, err)
	}
	if err := emitReviewSummary(cmd, result, contextLines, outPath); err != nil {
		return reviewExecution{}, err
	}

	return reviewExecution{
		Config:  cfg,
		Repo:    repo,
		OutPath: outPath,
		BlockOn: blockOn,
		Gate:    gate,
		Result:  result,
	}, nil
}

func emitReviewSummary(cmd *cobra.Command, result reviewer.Result, contextLines int, outPath string) error {
	if contextLines > 0 {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "context_files=%d\n", result.ContextFiles)
		if err != nil {
			return withExitCode(5, err)
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "context_chunks=%d\n", result.ContextChunks)
		if err != nil {
			return withExitCode(5, err)
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "test_summary=%s\n", result.TestSummary)
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
	return withExitCode(5, err)
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

func shouldSkipGitHubPublish(cfg config.Config, bundle findings.FindingsBundle) bool {
	ctx, err := github.ResolveContext(bundle.BaseSHA, bundle.HeadSHA)
	if err != nil {
		return false
	}
	if !ctx.IsFork {
		return false
	}
	if ctx.EventName == "pull_request_target" {
		_, authErr := platformauth.Resolve(cfg, "github")
		return authErr != nil
	}
	return true
}

func publishBundleToAPI(ctx context.Context, auth platformauth.Resolved, platform string, cfg config.Config, bundle findings.FindingsBundle, blockOn string, modes []string, feedback string, summaryOverview bool) error {
	if ctx == nil {
		ctx = context.Background()
	}
	resolvedModes, profile, err := resolvePublishModes(platform, modes, feedback)
	if err != nil {
		return err
	}
	modes = resolvedModes
	summary := renderPublishSummary(bundle, profile, modes, summaryOverview)

	switch platform {
	case "github":
		reviewCtx, err := github.ResolveContext(bundle.BaseSHA, bundle.HeadSHA)
		if err != nil {
			return err
		}
		existingComments, err := github.LoadExistingState(defaultModeOutput(platform, "github_comments"))
		if err != nil {
			return err
		}
		commentPlan := github.PlanInlineCommentsWithProfile(existingComments, bundle.Findings, string(profile))
		summaryCommentEnabled := cfg.Platforms.GitHub.SummaryCommentEnabled()
		return auth.WithToken(func(token string) error {
			for _, mode := range modes {
				switch normalizePublishMode(platform, mode) {
				case "check_run":
					if err := github.PublishCheckRun(ctx, token, reviewCtx, github.BuildCheckRunPayload(reviewCtx, bundle, summary), nil); err != nil {
						return err
					}
				case "github_comments":
					if err := github.PublishInlineComments(ctx, token, reviewCtx, commentPlan, nil); err != nil {
						return err
					}
				case "summary":
					if !summaryCommentEnabled {
						continue
					}
					if err := github.PublishSummaryComment(ctx, token, reviewCtx, summary, nil); err != nil {
						return err
					}
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
		status := azure.PolicyStatus(azure.PolicyContext{BlockOn: blockOn, FatalOnFailures: true}, blocking, len(bundle.Findings)-blocking, false)
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
