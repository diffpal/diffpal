package cmd

import (
	"reflect"
	"strings"
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
)

func TestDefaultModesMatchProductContract(t *testing.T) {
	t.Parallel()

	cases := []struct {
		platform string
		want     []string
	}{
		{platform: "github", want: []string{"check-run", "comments", "sarif", "summary"}},
		{platform: "gitlab", want: []string{"code-quality", "discussions", "sarif", "summary"}},
		{platform: "azure", want: []string{"threads", "status", "summary"}},
	}
	for _, tc := range cases {
		if got := defaultModes(tc.platform); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("defaultModes(%q) = %v, want %v", tc.platform, got, tc.want)
		}
	}
}

func TestResolvePublishModesUsesFeedbackWhenModeIsEmpty(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		platform string
		feedback string
		want     []string
	}{
		{name: "github summary", platform: "github", feedback: "summary", want: []string{"check-run", "sarif", "summary"}},
		{name: "github balanced", platform: "github", feedback: "balanced", want: []string{"check-run", "comments", "sarif", "summary"}},
		{name: "azure summary", platform: "azure", feedback: "summary", want: []string{"status", "summary"}},
		{name: "azure balanced", platform: "azure", feedback: "balanced", want: []string{"threads", "status", "summary"}},
		{name: "gitlab summary", platform: "gitlab", feedback: "summary", want: []string{"code-quality", "sarif", "summary"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, profile, err := resolvePublishModes(tc.platform, nil, tc.feedback)
			if err != nil {
				t.Fatalf("resolvePublishModes() error = %v", err)
			}
			if profile != FeedbackProfile(tc.feedback) {
				t.Fatalf("profile = %q, want %q", profile, tc.feedback)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("modes = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolvePublishModesExplicitModesOverrideFeedback(t *testing.T) {
	t.Parallel()

	want := []string{"status"}
	got, profile, err := resolvePublishModes("azure", want, "summary")
	if err != nil {
		t.Fatalf("resolvePublishModes() error = %v", err)
	}
	if profile != FeedbackProfile("summary") {
		t.Fatalf("profile = %q, want summary", profile)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("modes = %v, want %v", got, want)
	}
}

func TestResolvePublishModesRejectsInvalidFeedback(t *testing.T) {
	t.Parallel()

	if _, _, err := resolvePublishModes("github", nil, "verbose"); err == nil {
		t.Fatal("resolvePublishModes() error = nil, want invalid feedback error")
	}
}

func TestRenderPublishSummaryHidesMetadataByDefault(t *testing.T) {
	t.Parallel()

	got, err := renderPublishSummary("github", findings.FindingsBundle{
		ReviewID: "github-pr-2",
		ChangeSummary: []string{
			"Documented the GitHub setup flow for DiffPal users.",
		},
		Files: []findings.ReviewedFile{
			{Path: "README.md"},
		},
	}, FeedbackBalanced, []string{"check-run", "comments", "sarif", "summary"}, true, "", "")
	if err != nil {
		t.Fatalf("renderPublishSummary() error = %v", err)
	}

	for _, unwanted := range []string{
		"- Feedback profile: balanced",
		"- Publish surfaces: check-run, comments, sarif, summary",
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("summary contains hidden metadata %q:\n%s", unwanted, got)
		}
	}
	if !strings.Contains(got, "- Documented the GitHub setup flow for DiffPal users.") {
		t.Fatalf("summary missing change overview:\n%s", got)
	}
}

func TestRenderPublishSummaryCanHideOverview(t *testing.T) {
	t.Parallel()

	got, err := renderPublishSummary("github", findings.FindingsBundle{
		ReviewID: "github-pr-2",
		ChangeSummary: []string{
			"Documented the GitHub setup flow for DiffPal users.",
		},
		Files: []findings.ReviewedFile{
			{Path: "README.md"},
		},
	}, FeedbackBalanced, []string{"check-run", "comments", "sarif", "summary"}, false, "", "")
	if err != nil {
		t.Fatalf("renderPublishSummary() error = %v", err)
	}

	if strings.Contains(got, "## Summary of Changes") {
		t.Fatalf("summary contains hidden overview:\n%s", got)
	}
}

func TestRenderPublishSummaryUsesReviewChannelTitle(t *testing.T) {
	t.Parallel()

	got, err := renderPublishSummary("github", findings.FindingsBundle{
		ReviewID: "github-pr-2-diffpal-dev",
		Files: []findings.ReviewedFile{
			{Path: "README.md"},
		},
	}, FeedbackBalanced, []string{"check-run", "summary"}, true, "diffpal-dev", "")
	if err != nil {
		t.Fatalf("renderPublishSummary() error = %v", err)
	}

	if !strings.Contains(got, "# DiffPal Dev Review Summary") {
		t.Fatalf("summary missing channel title:\n%s", got)
	}
}

func TestRenderPublishSummaryRejectsInvalidReviewChannel(t *testing.T) {
	t.Parallel()

	_, err := renderPublishSummary("github", findings.FindingsBundle{
		ReviewID: "github-pr-2",
	}, FeedbackBalanced, []string{"summary"}, true, "bad/channel", "")
	if err == nil {
		t.Fatal("renderPublishSummary() error = nil, want invalid review channel error")
	}
}

func TestRenderPublishSummaryUsesRepoFallbackForGitHubLinks(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "")

	got, err := renderPublishSummary("github", findings.FindingsBundle{
		ReviewID: "github-pr-2",
		HeadSHA:  "head-a",
		Findings: []findings.Finding{
			{
				Severity:  "medium",
				Category:  "correctness",
				Path:      "internal/file.go",
				StartLine: 7,
				EndLine:   7,
				Title:     "finding",
				Message:   "message",
			},
		},
	}, FeedbackBalanced, []string{"summary"}, true, "", "acme/diffpal")
	if err != nil {
		t.Fatalf("renderPublishSummary() error = %v", err)
	}
	if !strings.Contains(got, "https://github.com/acme/diffpal/blob/head-a/internal/file.go#L7") {
		t.Fatalf("summary missing repo fallback link:\n%s", got)
	}
}
