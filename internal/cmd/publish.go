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
	"github.com/diffpal/diffpal/internal/policy"
	"github.com/diffpal/diffpal/internal/sarif"
)

type publishOutput struct {
	Surface string `json:"surface"`
	Path    string `json:"path,omitempty"`
	Status  string `json:"status,omitempty"`
}

type FeedbackProfile string

const (
	FeedbackReview  FeedbackProfile = "review"
	FeedbackSummary FeedbackProfile = "summary"
)

func publishBundleToFiles(platform string, bundle findings.FindingsBundle, repo string, blockOn string, gateEnabled bool, feedback string, summaryOverview bool, out string, reviewChannel string) ([]publishOutput, int, error) {
	platform = strings.ToLower(platform)
	blockOn, err := normalizeSeverity(blockOn)
	if err != nil {
		return nil, 0, err
	}
	bundle, err = normalizeBundleBlocking(bundle, blockOn)
	if err != nil {
		return nil, 0, err
	}
	blockThresholds := []string{blockOn}
	blocking := 0
	surfaces, profile, err := resolvePublishSurfaces(platform, feedback)
	if err != nil {
		return nil, 0, err
	}
	outputs := make([]publishOutput, 0, len(surfaces))
	if strings.TrimSpace(out) != "" && len(surfaces) > 1 {
		return nil, 0, fmt.Errorf("--out cannot be used when feedback publishes multiple surfaces")
	}
	summary, err := renderPublishSummary(platform, bundle, profile, surfaces, summaryOverview, reviewChannel, repo)
	if err != nil {
		return nil, 0, err
	}
	decision := gitlabpub.SummarizeDecision(bundle, blockThresholds)

	for _, surface := range surfaces {
		normalized := normalizePublishSurface(platform, surface)
		targetOut := out
		if targetOut == "" {
			targetOut = defaultSurfaceOutput(platform, normalized)
		}
		switch normalized {
		case "summary":
			if err := findings.WriteStringBundle(targetOut, summary); err != nil {
				return nil, 0, err
			}
			outputs = append(outputs, publishOutput{Surface: normalized, Path: targetOut, Status: "published"})
		case "github_comments":
			blocking = max(blocking, decision.BlockCount)
			existing, err := github.LoadExistingState(targetOut)
			if err != nil {
				return nil, 0, err
			}
			plan := github.PlanInlineCommentsWithOptions(existing, publishableInlineFindings(bundle.Findings), github.CommentOptions{
				Links:       githubLinkProvider(platform, bundle, repo),
				AllFindings: true,
			})
			raw, err := json.MarshalIndent(plan, "", "  ")
			if err != nil {
				return nil, 0, err
			}
			if err := findings.WriteStringBundle(targetOut, string(raw)); err != nil {
				return nil, 0, err
			}
			outputs = append(outputs, publishOutput{Surface: normalized, Path: targetOut, Status: "published"})
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
			outputs = append(outputs, publishOutput{Surface: normalized, Path: targetOut, Status: string(decision.Decision)})
		case "code_quality", "code-quality":
			report, err := codequality.ToJSON(bundle, repo)
			if err != nil {
				return nil, 0, err
			}
			if err := findings.WriteStringBundle(targetOut, string(report)); err != nil {
				return nil, 0, err
			}
			outputs = append(outputs, publishOutput{Surface: "code-quality", Path: targetOut, Status: "published"})
		case "sarif":
			converted := sarif.ToReport(bundle)
			raw, err := sarif.ToJSON(converted)
			if err != nil {
				return nil, 0, err
			}
			if err := findings.WriteStringBundle(targetOut, string(raw)); err != nil {
				return nil, 0, err
			}
			outputs = append(outputs, publishOutput{Surface: normalized, Path: targetOut, Status: "published"})
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
			threads := azure.PlanThreads(existing, bundle.Findings, ctx)
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
			outputs = append(outputs, publishOutput{Surface: normalized, Path: targetOut, Status: "published"})
		case "status":
			blocking = max(blocking, decision.BlockCount)
			payload := azure.PolicyStatus(azure.PolicyContext{BlockOn: blockOn, GateEnabled: gateEnabled, FatalOnFailures: true}, decision.BlockCount, decision.AdvisoryCount, false)
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
			outputs = append(outputs, publishOutput{Surface: normalized, Path: targetOut, Status: string(payload.State)})
		default:
			return nil, 0, fmt.Errorf("unsupported surface %q for platform %s", surface, platform)
		}
	}

	return outputs, blocking, nil
}

func resolvePublishSurfaces(platform string, feedback string) ([]string, FeedbackProfile, error) {
	profile, err := normalizeFeedback(feedback)
	if err != nil {
		return nil, "", err
	}
	return surfacesForFeedback(platform, profile), profile, nil
}

func renderPublishSummary(platform string, bundle findings.FindingsBundle, profile FeedbackProfile, surfaces []string, summaryOverview bool, reviewChannel string, repo string) (string, error) {
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
		PublishSurfaces: publishSurfaceLabels(surfaces),
		HideOverview:    !summaryOverview,
		HideResult:      !publishesFileLevelFindings(platform, surfaces),
		HideDetails:     true,
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

func publishSurfaceLabels(surfaces []string) []string {
	labels := make([]string, 0, len(surfaces))
	seen := map[string]struct{}{}
	for _, surface := range surfaces {
		label := publishSurfaceLabel(surface)
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

func publishSurfaceLabel(surface string) string {
	switch strings.ToLower(strings.TrimSpace(surface)) {
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
		return strings.TrimSpace(surface)
	}
}

func normalizeFeedback(value string) (FeedbackProfile, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(FeedbackReview):
		return FeedbackReview, nil
	case string(FeedbackSummary):
		return FeedbackSummary, nil
	default:
		return "", fmt.Errorf("invalid feedback %q", value)
	}
}

func surfacesForFeedback(platform string, feedback FeedbackProfile) []string {
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
			return []string{"sarif", "summary"}
		default:
			return []string{"comments", "sarif", "summary"}
		}
	}
}

func normalizePublishSurface(platform string, surface string) string {
	surface = strings.ToLower(strings.TrimSpace(surface))
	switch surface {
	case "comments", "review-comments":
		return "github_comments"
	case "discussion", "discussions", "threads":
		if platform == "gitlab" {
			return "discussions"
		}
		if platform == "azure" {
			return "threads"
		}
	}
	switch surface {
	case "summary":
		return "summary"
	case "codequality", "code_quality", "code-quality":
		if platform == "gitlab" {
			return "code-quality"
		}
		return surface
	case "sarif":
		return "sarif"
	case "status":
		return "status"
	}
	if platform == "github" && surface == "comments" {
		return "github_comments"
	}
	if platform == "azure" && surface == "status" {
		return "status"
	}
	if platform == "gitlab" && surface == "discussions" {
		return "discussions"
	}
	return surface
}

func publishesFileLevelFindings(platform string, surfaces []string) bool {
	for _, surface := range surfaces {
		switch normalizePublishSurface(platform, surface) {
		case "github_comments", "threads", "discussions":
			return true
		}
	}
	return false
}

func defaultSurfaceOutput(platform string, surface string) string {
	switch surface {
	case "summary":
		return ".artifacts/diffpal/summary.md"
	case "sarif":
		return ".artifacts/diffpal/diffpal.sarif"
	case "code-quality":
		return ".artifacts/diffpal/codequality.json"
	case "github_comments":
		return ".artifacts/diffpal/github-comments.json"
	case "discussions":
		return ".artifacts/diffpal/gitlab-discussions.json"
	case "threads":
		return ".artifacts/diffpal/azure-threads.json"
	case "status":
		return ".artifacts/diffpal/azure-status.json"
	default:
		base := strings.ReplaceAll(surface, " ", "-")
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

func normalizeBundleBlocking(bundle findings.FindingsBundle, blockOn string) (findings.FindingsBundle, error) {
	sev, err := policy.ParseSeverity(strings.ToLower(strings.TrimSpace(blockOn)))
	if err != nil {
		return findings.FindingsBundle{}, err
	}
	items := make([]policy.Finding, 0, len(bundle.Findings))
	for _, item := range bundle.Findings {
		parsed, err := policy.ParseSeverity(item.Severity)
		if err != nil {
			return findings.FindingsBundle{}, err
		}
		items = append(items, policy.Finding{
			Severity:   parsed,
			Confidence: item.Confidence,
			Path:       item.Path,
		})
	}
	decisions := policy.ApplyPolicy(policy.Policy{BlockOn: sev}, items)
	out := bundle
	out.Findings = append([]findings.Finding(nil), bundle.Findings...)
	for i := range out.Findings {
		out.Findings[i].Blocking = decisions[i].Action == "block"
	}
	return out, nil
}

func publishableInlineFindings(items []findings.Finding) []findings.Finding {
	out := make([]findings.Finding, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Path) == "" || item.StartLine <= 0 {
			continue
		}
		out = append(out, item)
	}
	return out
}
