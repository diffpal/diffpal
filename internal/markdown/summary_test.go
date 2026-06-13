package markdown

import (
	"strings"
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
)

func TestRenderSummaryGroupsBySeverityFileAndRule(t *testing.T) {
	t.Parallel()

	bundle := findings.FindingsBundle{
		ReviewID:     "review-123",
		BaseSHA:      "base-1",
		HeadSHA:      "head-1",
		Language:     "en",
		ReviewChecks: []string{"bugs", "performance", "best-practices"},
		ChangeSummary: []string{
			"Changed service request handling and database query behavior.",
		},
		Files: []findings.ReviewedFile{
			{Path: "internal/app/service.go"},
			{Path: "internal/db/query.go"},
			{Path: "internal/web/handler.go"},
		},
		Findings: []findings.Finding{
			{
				Category:  "security",
				Severity:  "high",
				Blocking:  true,
				Path:      "internal/db/query.go",
				StartLine: 20,
				EndLine:   20,
				Title:     "unsafe query",
				Message:   "query concatenates untrusted input",
				Evidence:  "db.Query(\"select \" + input)",
			},
			{
				Category:  "security",
				Severity:  "high",
				Blocking:  true,
				Path:      "internal/db/query.go",
				StartLine: 28,
				EndLine:   28,
				Title:     "second unsafe query",
				Message:   "second unsafe SQL sink",
				Evidence:  "db.Query(raw)",
			},
			{
				Category:  "correctness",
				Severity:  "critical",
				Blocking:  true,
				Path:      "internal/app/service.go",
				StartLine: 8,
				EndLine:   8,
				Title:     "nil dereference",
				Message:   "possible nil dereference",
				Evidence:  "cfg.Client.Do(req)",
			},
			{
				Category:  "maintainability",
				Severity:  "low",
				Path:      "internal/app/service.go",
				StartLine: 41,
				EndLine:   41,
				Title:     "unreachable branch",
				Message:   "branch is unreachable",
				Evidence:  "return before if",
			},
		},
	}

	got := RenderSummary(bundle)

	assertContains(t, got, "# DiffPal Review Summary")
	assertContains(t, got, "## Summary of Changes")
	assertContains(t, got, "- Changed service request handling and database query behavior.")
	assertContains(t, got, "## Review Result")
	assertContains(t, got, "DiffPal found 4 actionable finding(s), including 3 blocking finding(s).")
	assertNotContains(t, got, "review_id: review-123")
	assertNotContains(t, got, "- Reviewed files: 3")
	assertNotContains(t, got, "- Findings: 4")
	assertNotContains(t, got, "- Blocking findings: 3")
	assertNotContains(t, got, "- Review checks: bugs, performance, best-practices")
	assertContains(t, got, "## Feedback on Files")
	assertContains(t, got, "| `internal/app/service.go` | Blocked | critical: 1, low: 1 |")
	assertContains(t, got, "| `internal/db/query.go` | Blocked | high: 2 |")
	assertContains(t, got, "| `internal/web/handler.go` | Passed | No actionable findings. |")
	assertContains(t, got, "## Detailed Comments")
	assertContains(t, got, "### internal/app/service.go")
	assertContains(t, got, "- **Critical correctness** `L8`: possible nil dereference")
	assertContains(t, got, "  - **Evidence**: cfg.Client.Do(req)")
	assertContains(t, got, "### internal/db/query.go")
	assertContains(t, got, "- **High security** `L20`: query concatenates untrusted input")
}

func TestRenderSummaryHandlesEmptyBundle(t *testing.T) {
	t.Parallel()

	got := RenderSummary(findings.FindingsBundle{
		ReviewID: "review-empty",
		ChangeSummary: []string{
			"Refined application service behavior without actionable review findings.",
		},
		Files: []findings.ReviewedFile{
			{Path: "internal/app/service.go"},
		},
	})

	assertContains(t, got, "# DiffPal Review Summary")
	assertContains(t, got, "- Refined application service behavior without actionable review findings.")
	assertNotContains(t, got, "review_id: review-empty")
	assertContains(t, got, "DiffPal found no actionable issues in the reviewed diff.")
	assertContains(t, got, "| `internal/app/service.go` | Passed | No actionable findings. |")
	if strings.Contains(got, "## Detailed Comments") {
		t.Fatalf("empty summary includes detailed comments:\n%s", got)
	}
}

func TestRenderSummaryCanHideChangeOverview(t *testing.T) {
	t.Parallel()

	got := RenderSummaryWithOptions(findings.FindingsBundle{
		ReviewID: "review-feedback",
		ChangeSummary: []string{
			"Refined application service behavior.",
		},
		Files: []findings.ReviewedFile{
			{Path: "internal/app/service.go"},
		},
	}, SummaryOptions{
		HideOverview: true,
	})

	assertNotContains(t, got, "## Summary of Changes")
	assertNotContains(t, got, "Refined application service behavior.")
	assertContains(t, got, "## Review Result")
}

func TestRenderSummaryWithOptionsShowsMetadata(t *testing.T) {
	t.Parallel()

	got := RenderSummaryWithOptions(findings.FindingsBundle{
		ReviewID: "review-feedback",
		BaseSHA:  "base-a",
		HeadSHA:  "head-a",
		Language: "en",
		ReviewChecks: []string{
			"bugs",
			"performance",
			"best-practices",
		},
		ChangeSummary: []string{
			"Refined application service behavior.",
		},
		Files: []findings.ReviewedFile{
			{Path: "internal/app/service.go"},
		},
	}, SummaryOptions{
		FeedbackProfile: "balanced",
		PublishSurfaces: []string{
			"check-run",
			"comments",
			"sarif",
			"summary",
		},
		ShowMetadata: true,
	})

	assertContains(t, got, "## Review Metadata")
	assertContains(t, got, "- Review ID: review-feedback")
	assertContains(t, got, "- Reviewed files: 1")
	assertContains(t, got, "- Feedback profile: balanced")
	assertContains(t, got, "- Publish surfaces: check-run, comments, sarif, summary")
}

func TestRenderSummaryFallsBackToSemanticChangeOverview(t *testing.T) {
	t.Parallel()

	got := RenderSummary(findings.FindingsBundle{
		ReviewID: "review-empty-overview",
		Files: []findings.ReviewedFile{
			{Path: "README.md"},
			{Path: "internal/app/service.go"},
		},
	})

	assertContains(t, got, "## Summary of Changes")
	assertContains(t, got, "- Updated user-facing documentation and setup guidance.")
	assertContains(t, got, "- Updated DiffPal implementation files.")
	assertNotContains(t, got, "Updated `README.md`.")
	assertNotContains(t, got, "Updated `internal/app/service.go`.")
}

func TestRenderSummaryIncludesFindingCodeSnippet(t *testing.T) {
	t.Parallel()

	bundle := findings.FindingsBundle{
		ReviewID: "review-snippet",
		Files: []findings.ReviewedFile{
			{Path: "internal/platformapi/admin_debug.go"},
		},
		Findings: []findings.Finding{
			{
				Category:   "security",
				Severity:   "high",
				Confidence: 0.98,
				Path:       "internal/platformapi/admin_debug.go",
				StartLine:  12,
				EndLine:    17,
				Message:    "query concatenates untrusted input",
				Evidence:   "Line 17 builds SQL by concatenating user input.",
				Suggestion: "Use a parameterized statement.",
			},
		},
	}
	got := RenderSummaryWithOptions(bundle, SummaryOptions{
		Snippets: SnippetFunc(func(finding findings.Finding) (CodeSnippet, bool) {
			if finding.Path != "internal/platformapi/admin_debug.go" {
				return CodeSnippet{}, false
			}
			return CodeSnippet{
				Language: "go",
				Code:     "user := r.URL.Query().Get(\"user\")\n_, _ = db.Exec(\"DELETE FROM sessions WHERE user = '\" + user + \"'\")",
			}, true
		}),
	})

	assertContains(t, got, "- **High security** `L12-L17`: query concatenates untrusted input")
	assertContains(t, got, "  ```go\n  user := r.URL.Query().Get(\"user\")\n  _, _ = db.Exec(\"DELETE FROM sessions WHERE user = '\" + user + \"'\")\n  ```")
	assertContains(t, got, "  - **Evidence**: Line 17 builds SQL by concatenating user input.")
	assertContains(t, got, "  - **Suggestion**: Use a parameterized statement.")
}

func TestRenderSummaryFallsBackWhenSnippetMissing(t *testing.T) {
	t.Parallel()

	got := RenderSummaryWithOptions(findings.FindingsBundle{
		ReviewID: "review-no-snippet",
		Findings: []findings.Finding{
			{
				Category:   "correctness",
				Severity:   "medium",
				Confidence: 0.8,
				Path:       "internal/app/service.go",
				StartLine:  4,
				Message:    "possible nil dereference",
				Evidence:   "cfg.Client.Do(req)",
			},
		},
	}, SummaryOptions{
		Snippets: SnippetFunc(func(findings.Finding) (CodeSnippet, bool) {
			return CodeSnippet{}, false
		}),
	})

	assertContains(t, got, "- **Medium correctness** `L4`: possible nil dereference")
	assertContains(t, got, "  - **Evidence**: cfg.Client.Do(req)")
	assertNotContains(t, got, "```")
}

func TestRenderSummaryUsesReadableLinkedFindingHeader(t *testing.T) {
	t.Parallel()

	got := RenderSummaryWithOptions(findings.FindingsBundle{
		ReviewID: "review-link",
		Findings: []findings.Finding{
			{
				Category:   "security",
				Severity:   "high",
				Confidence: 0.98,
				Path:       "internal/platformapi/admin_debug.go",
				StartLine:  12,
				EndLine:    17,
				Message:    "query concatenates untrusted input",
				Evidence:   "line 17 concatenates the user query parameter into SQL",
			},
		},
	}, SummaryOptions{
		Links: FindingLinkFunc(func(findings.Finding) (string, bool) {
			return "https://github.com/acme/diffpal/blob/head-a/internal/platformapi/admin_debug.go#L12-L17", true
		}),
	})

	assertContains(t, got, "- **High security**: query concatenates untrusted input")
	assertContains(t, got, "  - **Evidence**: line 17 concatenates the user query parameter into SQL")
	assertContains(t, got, "  - **Source**:")
	assertContains(t, got, "  - https://github.com/acme/diffpal/blob/head-a/internal/platformapi/admin_debug.go#L12-L17")
	assertNotContains(t, got, "**[high][")
}

func TestRenderSummaryUsesLongerFenceForBackticks(t *testing.T) {
	t.Parallel()

	got := RenderSummaryWithOptions(findings.FindingsBundle{
		ReviewID: "review-backticks",
		Findings: []findings.Finding{
			{
				Severity:   "low",
				Confidence: 0.7,
				Path:       "README.md",
				StartLine:  3,
				Message:    "example contains a fence",
				Evidence:   "nested fence",
			},
		},
	}, SummaryOptions{
		Snippets: SnippetFunc(func(findings.Finding) (CodeSnippet, bool) {
			return CodeSnippet{Language: "markdown", Code: "```go\nfmt.Println(\"x\")\n```"}, true
		}),
	})

	assertContains(t, got, "  ````markdown\n  ```go\n  fmt.Println(\"x\")\n  ```\n  ````")
}

func assertContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("summary missing %q:\n%s", want, got)
	}
}

func assertNotContains(t *testing.T, got string, unwanted string) {
	t.Helper()
	if strings.Contains(got, unwanted) {
		t.Fatalf("summary contains %q:\n%s", unwanted, got)
	}
}
