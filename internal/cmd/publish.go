package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/diffpal/diffpal/internal/codequality"
	"github.com/diffpal/diffpal/internal/findings"
	"github.com/diffpal/diffpal/internal/markdown"
	"github.com/diffpal/diffpal/internal/platform/azure"
	"github.com/diffpal/diffpal/internal/platform/github"
	gitlabpub "github.com/diffpal/diffpal/internal/platform/gitlab"
	"github.com/diffpal/diffpal/internal/sarif"
)

type publishOutput struct {
	Mode   string `json:"mode"`
	Path   string `json:"path,omitempty"`
	Status string `json:"status,omitempty"`
}

type FeedbackProfile string

const (
	FeedbackBalanced FeedbackProfile = "balanced"
	FeedbackSummary  FeedbackProfile = "summary"
	FeedbackInline   FeedbackProfile = "inline"
)

func publishBundleToFiles(platform string, bundle findings.FindingsBundle, repo string, blockOn string, modes []string, feedback string, summaryOverview bool, out string, reviewChannel string) ([]publishOutput, int, error) {
	platform = strings.ToLower(platform)
	blockOn, err := normalizeSeverity(blockOn)
	if err != nil {
		return nil, 0, err
	}
	blockThresholds := []string{blockOn}
	outputs := make([]publishOutput, 0, len(modes))
	blocking := 0
	modes, profile, err := resolvePublishModes(platform, modes, feedback)
	if err != nil {
		return nil, 0, err
	}
	if strings.TrimSpace(out) != "" && len(modes) > 1 {
		return nil, 0, fmt.Errorf("--out cannot be used with multiple publish modes")
	}
	summary, err := renderPublishSummary(platform, bundle, profile, modes, summaryOverview, reviewChannel, repo)
	if err != nil {
		return nil, 0, err
	}
	decision := gitlabpub.SummarizeDecision(bundle, blockThresholds)

	for _, mode := range modes {
		normalized := normalizePublishMode(platform, mode)
		targetOut := out
		if targetOut == "" {
			targetOut = defaultModeOutput(platform, normalized)
		}
		switch normalized {
		case "summary":
			if err := findings.WriteStringBundle(targetOut, summary); err != nil {
				return nil, 0, err
			}
			outputs = append(outputs, publishOutput{Mode: normalized, Path: targetOut, Status: "published"})
		case "check_run":
			blocking = max(blocking, decision.BlockCount)
			ctx, err := github.ResolveContext(bundle.BaseSHA, bundle.HeadSHA)
			if err != nil {
				ctx = github.Context{}
				ctx.HeadSHA = bundle.HeadSHA
			}
			identity, err := github.NewReviewIdentity(reviewChannel)
			if err != nil {
				return nil, 0, err
			}
			payload := github.BuildCheckRunPayloadWithIdentity(ctx, bundle, summary, identity)
			raw, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return nil, 0, err
			}
			if err := findings.WriteStringBundle(targetOut, string(raw)); err != nil {
				return nil, 0, err
			}
			outputs = append(outputs, publishOutput{Mode: normalized, Path: targetOut, Status: payload.Conclusion})
		case "github_comments":
			blocking = max(blocking, decision.BlockCount)
			existing, err := github.LoadExistingState(targetOut)
			if err != nil {
				return nil, 0, err
			}
			plan := github.PlanInlineCommentsWithOptions(existing, bundle.Findings, github.CommentOptions{
				Profile: string(profile),
				Links:   githubLinkProvider(platform, bundle, repo),
			})
			raw, err := json.MarshalIndent(plan, "", "  ")
			if err != nil {
				return nil, 0, err
			}
			if err := findings.WriteStringBundle(targetOut, string(raw)); err != nil {
				return nil, 0, err
			}
			outputs = append(outputs, publishOutput{Mode: normalized, Path: targetOut, Status: "published"})
		case "discussions":
			existing, err := gitlabpub.LoadExistingState(targetOut)
			if err != nil {
				return nil, 0, err
			}
			plan := gitlabpub.PlanDiscussions(existing, bundle.Findings, blockThresholds)
			blocking = max(blocking, decision.BlockCount)
			payload := map[string]interface{}{
				"decision":       string(decision.Decision),
				"blocking_count": decision.BlockCount,
				"advisory_count": decision.AdvisoryCount,
				"plan":           plan,
			}
			raw, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return nil, 0, err
			}
			if err := findings.WriteStringBundle(targetOut, string(raw)); err != nil {
				return nil, 0, err
			}
			outputs = append(outputs, publishOutput{Mode: normalized, Path: targetOut, Status: string(decision.Decision)})
		case "code_quality", "code-quality":
			report, err := codequality.ToJSON(bundle, repo)
			if err != nil {
				return nil, 0, err
			}
			if err := findings.WriteStringBundle(targetOut, string(report)); err != nil {
				return nil, 0, err
			}
			outputs = append(outputs, publishOutput{Mode: "code-quality", Path: targetOut, Status: "published"})
		case "sarif":
			converted := sarif.ToReport(bundle)
			raw, err := sarif.ToJSON(converted)
			if err != nil {
				return nil, 0, err
			}
			if err := findings.WriteStringBundle(targetOut, string(raw)); err != nil {
				return nil, 0, err
			}
			outputs = append(outputs, publishOutput{Mode: normalized, Path: targetOut, Status: "published"})
		case "threads":
			existing, err := azure.LoadExistingState(targetOut)
			if err != nil {
				return nil, 0, err
			}
			ctx, err := azure.ResolveContext(bundle.BaseSHA, bundle.HeadSHA)
			if err != nil {
				ctx = azure.Context{
					BaseSHA: bundle.BaseSHA,
					HeadSHA: bundle.HeadSHA,
				}
			}
			threads := azure.PlanThreadsWithProfile(existing, bundle.Findings, ctx, string(profile))
			payload := map[string]interface{}{
				"threads": threads,
			}
			raw, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return nil, 0, err
			}
			if err := findings.WriteStringBundle(targetOut, string(raw)); err != nil {
				return nil, 0, err
			}
			outputs = append(outputs, publishOutput{Mode: normalized, Path: targetOut, Status: "published"})
		case "status":
			blocking = max(blocking, decision.BlockCount)
			payload := azure.PolicyStatus(azure.PolicyContext{BlockOn: blockOn, FatalOnFailures: true}, decision.BlockCount, decision.AdvisoryCount, false)
			decisions := map[string]interface{}{
				"decision": decision.Decision,
				"status":   payload,
				"blocking": decision.BlockCount,
				"advisory": decision.AdvisoryCount,
			}
			raw, err := json.MarshalIndent(decisions, "", "  ")
			if err != nil {
				return nil, 0, err
			}
			if err := findings.WriteStringBundle(targetOut, string(raw)); err != nil {
				return nil, 0, err
			}
			outputs = append(outputs, publishOutput{Mode: normalized, Path: targetOut, Status: string(payload.State)})
		default:
			return nil, 0, fmt.Errorf("unsupported mode %q for platform %s", mode, platform)
		}
	}

	return outputs, blocking, nil
}

func resolvePublishModes(platform string, modes []string, feedback string) ([]string, FeedbackProfile, error) {
	profile, err := normalizeFeedback(feedback)
	if err != nil {
		return nil, "", err
	}
	if len(modes) > 0 {
		return modes, profile, nil
	}
	return modesForFeedback(platform, profile), profile, nil
}

func renderPublishSummary(platform string, bundle findings.FindingsBundle, profile FeedbackProfile, modes []string, summaryOverview bool, reviewChannel string, repo string) (string, error) {
	title := ""
	if strings.ToLower(strings.TrimSpace(platform)) == "github" {
		identity, err := github.NewReviewIdentity(reviewChannel)
		if err != nil {
			return "", err
		}
		title = identity.SummaryTitle()
	}
	opts := markdown.SummaryOptions{
		Title:           title,
		FeedbackProfile: string(profile),
		PublishSurfaces: publishSurfaceLabels(modes),
		HideOverview:    !summaryOverview,
		Links:           githubLinkProvider(platform, bundle, repo),
	}
	return markdown.RenderSummaryWithOptions(bundle, opts), nil
}

func githubLinkProvider(platform string, bundle findings.FindingsBundle, repo string) markdown.FindingLinkProvider {
	if strings.ToLower(strings.TrimSpace(platform)) != "github" {
		return nil
	}
	ctx, err := github.ResolveContext(bundle.BaseSHA, bundle.HeadSHA)
	if err != nil {
		ctx = github.Context{
			Repo:    strings.TrimSpace(repo),
			HeadSHA: bundle.HeadSHA,
		}
	}
	if strings.TrimSpace(ctx.Repo) == "" {
		ctx.Repo = strings.TrimSpace(os.Getenv("GITHUB_REPOSITORY"))
	}
	if strings.TrimSpace(ctx.HeadSHA) == "" {
		ctx.HeadSHA = bundle.HeadSHA
	}
	return github.NewPermanentLinkProvider(ctx)
}

func publishSurfaceLabels(modes []string) []string {
	labels := make([]string, 0, len(modes))
	seen := map[string]struct{}{}
	for _, mode := range modes {
		label := publishSurfaceLabel(mode)
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return labels
}

func publishSurfaceLabel(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "check_run", "check-run", "checks":
		return "check-run"
	case "github_comments", "comments", "review-comments":
		return "comments"
	case "code_quality", "code-quality":
		return "code-quality"
	case "discussions":
		return "discussions"
	case "threads":
		return "threads"
	case "status":
		return "status"
	case "sarif":
		return "sarif"
	case "summary":
		return "summary"
	default:
		return strings.TrimSpace(mode)
	}
}

func normalizeFeedback(value string) (FeedbackProfile, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(FeedbackBalanced):
		return FeedbackBalanced, nil
	case string(FeedbackSummary):
		return FeedbackSummary, nil
	case string(FeedbackInline):
		return FeedbackInline, nil
	default:
		return "", fmt.Errorf("invalid feedback %q", value)
	}
}

func modesForFeedback(platform string, feedback FeedbackProfile) []string {
	switch strings.ToLower(platform) {
	case "gitlab":
		switch feedback {
		case FeedbackSummary:
			return []string{"code-quality", "sarif", "summary"}
		default:
			return []string{"code-quality", "discussions", "sarif", "summary"}
		}
	case "azure":
		switch feedback {
		case FeedbackSummary:
			return []string{"status", "summary"}
		default:
			return []string{"threads", "status", "summary"}
		}
	default:
		switch feedback {
		case FeedbackSummary:
			return []string{"check-run", "sarif", "summary"}
		default:
			return []string{"check-run", "comments", "sarif", "summary"}
		}
	}
}

func parseModeList(raw string) []string {
	parts := strings.Split(raw, ",")
	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		clean := strings.TrimSpace(part)
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	sort.Strings(out)
	return out
}

func normalizePublishMode(platform string, mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "check", "checkrun", "check-run", "check_run":
		return "check_run"
	case "comments", "inline", "inline-comments", "inline_comments":
		return "github_comments"
	case "discussion", "discussions", "threads":
		if platform == "gitlab" {
			return "discussions"
		}
		if platform == "azure" {
			return "threads"
		}
	}
	switch mode {
	case "summary":
		return "summary"
	case "codequality", "code_quality", "code-quality":
		if platform == "gitlab" {
			return "code-quality"
		}
		return mode
	case "sarif":
		return "sarif"
	case "status":
		return "status"
	}
	if platform == "github" && mode == "comments" {
		return "github_comments"
	}
	if platform == "azure" && mode == "status" {
		return "status"
	}
	if platform == "gitlab" && mode == "discussions" {
		return "discussions"
	}
	return mode
}

func defaultModes(platform string) []string {
	return modesForFeedback(platform, FeedbackBalanced)
}

func defaultModeOutput(platform string, mode string) string {
	switch mode {
	case "summary":
		return ".artifacts/diffpal/summary.md"
	case "sarif":
		return ".artifacts/diffpal/diffpal.sarif"
	case "code-quality":
		return ".artifacts/diffpal/codequality.json"
	case "check_run":
		return ".artifacts/diffpal/github-checkrun.json"
	case "github_comments":
		return ".artifacts/diffpal/github-comments.json"
	case "discussions":
		return ".artifacts/diffpal/gitlab-discussions.json"
	case "threads":
		return ".artifacts/diffpal/azure-threads.json"
	case "status":
		return ".artifacts/diffpal/azure-status.json"
	default:
		base := strings.ReplaceAll(mode, " ", "-")
		if base == "" {
			base = "publish"
		}
		return filepath.Join(".artifacts", "diffpal", base+".json")
	}
}

func normalizeSeverity(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low", "medium", "high", "critical":
		return strings.ToLower(strings.TrimSpace(value)), nil
	default:
		return "", fmt.Errorf("invalid block-on severity %q", value)
	}
}
