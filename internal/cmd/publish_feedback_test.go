package cmd

import (
	"reflect"
	"strings"
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
)

func TestResolvePublishSurfacesUsesFeedback(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		platform string
		feedback string
		want     []string
	}{
		{name: "github summary", platform: "github", feedback: "summary", want: []string{"sarif", "summary"}},
		{name: "github review", platform: "github", feedback: "review", want: []string{"comments", "sarif", "summary"}},
		{name: "azure summary", platform: "azure", feedback: "summary", want: []string{"status", "summary"}},
		{name: "azure review", platform: "azure", feedback: "review", want: []string{"threads", "status", "summary"}},
		{name: "gitlab summary", platform: "gitlab", feedback: "summary", want: []string{"code-quality", "sarif", "status", "summary"}},
		{name: "gitlab review", platform: "gitlab", feedback: "review", want: []string{"code-quality", "discussions", "status", "sarif", "summary"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, profile, err := resolvePublishSurfaces(tc.platform, tc.feedback)
			if err != nil {
				t.Fatalf("resolvePublishSurfaces() error = %v", err)
			}
			if profile != FeedbackProfile(tc.feedback) {
				t.Fatalf("profile = %q, want %q", profile, tc.feedback)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("surfaces = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolvePublishSurfacesRejectsInvalidFeedback(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"verbose", "balanced", "inline"} {
		if _, _, err := resolvePublishSurfaces("github", value); err == nil {
			t.Fatalf("resolvePublishSurfaces(%q) error = nil, want invalid feedback error", value)
		}
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
	}, FeedbackReview, []string{"comments", "sarif", "summary"}, true, "", "")
	if err != nil {
		t.Fatalf("renderPublishSummary() error = %v", err)
	}

	for _, unwanted := range []string{
		"- Feedback profile: review",
		"- Publish surfaces: comments, sarif, summary",
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
	}, FeedbackReview, []string{"comments", "sarif", "summary"}, false, "", "")
	if err != nil {
		t.Fatalf("renderPublishSummary() error = %v", err)
	}

	if strings.Contains(got, "## Summary of Changes") {
		t.Fatalf("summary contains hidden overview:\n%s", got)
	}
}

func TestRenderPublishSummarySummaryOnlyOmitsFindingsResult(t *testing.T) {
	t.Parallel()

	got, err := renderPublishSummary("github", findings.FindingsBundle{
		ReviewID: "github-pr-2",
		ChangeSummary: []string{
			"Documented the GitHub setup flow for DiffPal users.",
		},
		Findings: []findings.Finding{{
			Severity:  "high",
			Category:  "security",
			Path:      "internal/file.go",
			StartLine: 7,
			EndLine:   7,
			Message:   "query concatenates user input",
			Blocking:  true,
		}},
	}, FeedbackSummary, []string{"summary", "sarif"}, true, "", "")
	if err != nil {
		t.Fatalf("renderPublishSummary() error = %v", err)
	}

	for _, unwanted := range []string{
		"## Review Result",
		"## Detailed Comments",
		"DiffPal found 1 actionable finding(s)",
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("summary-only output contains %q:\n%s", unwanted, got)
		}
	}
	if !strings.Contains(got, "## Summary of Changes") {
		t.Fatalf("summary-only output missing change overview:\n%s", got)
	}
}

func TestRenderPublishSummaryOmitsDetailedCommentsWhenFileThreadsArePublished(t *testing.T) {
	t.Parallel()

	got, err := renderPublishSummary("azure", findings.FindingsBundle{
		ReviewID: "azure-pr-1921",
		ChangeSummary: []string{
			"Added trade session synchronization workflow.",
		},
		Findings: []findings.Finding{{
			Severity:   "medium",
			Category:   "reliability",
			Path:       "internal/tradesessionssyncservice/temporal/activities/load_from_s3.go",
			StartLine:  29,
			EndLine:    32,
			Message:    "S3 ListObjectsV2 is called only once without pagination.",
			Impact:     findings.NewImpact("Some trade sessions can remain unprocessed."),
			Suggestion: "Use s3.NewListObjectsV2Paginator.",
		}},
	}, FeedbackReview, []string{"threads", "status", "summary"}, true, "", "")
	if err != nil {
		t.Fatalf("renderPublishSummary() error = %v", err)
	}

	for _, unwanted := range []string{
		"## Detailed Comments",
		"internal/tradesessionssyncservice/temporal/activities/load_from_s3.go",
		"S3 ListObjectsV2 is called only once without pagination.",
		"Use s3.NewListObjectsV2Paginator.",
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("summary duplicates file-thread detail %q:\n%s", unwanted, got)
		}
	}
	for _, want := range []string{
		"## Summary of Changes",
		"## Review Result",
		"DiffPal found 1 actionable finding(s).",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary missing %q:\n%s", want, got)
		}
	}
}

func TestRenderPublishSummaryUsesReviewChannelTitle(t *testing.T) {
	t.Parallel()

	got, err := renderPublishSummary("github", findings.FindingsBundle{
		ReviewID: "github-pr-2-diffpal-dev",
		Files: []findings.ReviewedFile{
			{Path: "README.md"},
		},
	}, FeedbackReview, []string{"comments", "summary"}, true, "diffpal-dev", "")
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
	}, FeedbackReview, []string{"summary"}, true, "bad/channel", "")
	if err == nil {
		t.Fatal("renderPublishSummary() error = nil, want invalid review channel error")
	}
}

func TestRenderPublishSummaryUsesRepoFallbackForGitHubLinks(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "")
	t.Setenv("GITHUB_EVENT_PATH", "")
	t.Setenv("GITHUB_BASE_SHA", "")
	t.Setenv("GITHUB_HEAD_SHA", "")

	got, err := renderPublishSummary("github", findings.FindingsBundle{
		ReviewID: "github-pr-2",
		BaseSHA:  "base-a",
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
	}, FeedbackReview, []string{"summary"}, true, "", "acme/diffpal")
	if err != nil {
		t.Fatalf("renderPublishSummary() error = %v", err)
	}
	if strings.Contains(got, "https://github.com/acme/diffpal/blob/head-a/internal/file.go#L7") {
		t.Fatalf("github summary includes detailed finding link:\n%s", got)
	}
	if strings.Contains(got, "## Detailed Comments") {
		t.Fatalf("github summary includes detailed comments:\n%s", got)
	}
}
