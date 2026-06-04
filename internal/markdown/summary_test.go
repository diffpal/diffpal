package markdown

import (
	"strings"
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
)

func TestRenderSummaryGroupsBySeverityFileAndRule(t *testing.T) {
	t.Parallel()

	bundle := findings.FindingsBundle{
		ReviewID: "review-123",
		BaseSHA:  "base-1",
		HeadSHA:  "head-1",
		Findings: []findings.Finding{
			{
				RuleID:    "security.sql",
				Severity:  "high",
				Path:      "internal/db/query.go",
				StartLine: 20,
				EndLine:   20,
				Message:   "query concatenates untrusted input",
			},
			{
				RuleID:    "security.sql",
				Severity:  "high",
				Path:      "internal/db/query.go",
				StartLine: 28,
				EndLine:   28,
				Message:   "second unsafe SQL sink",
			},
			{
				RuleID:    "correctness.nil",
				Severity:  "critical",
				Path:      "internal/app/service.go",
				StartLine: 8,
				EndLine:   8,
				Message:   "possible nil dereference",
			},
			{
				RuleID:    "maintainability.deadcode",
				Severity:  "low",
				Path:      "internal/app/service.go",
				StartLine: 41,
				EndLine:   41,
				Message:   "branch is unreachable",
			},
		},
	}

	got := RenderSummary(bundle)

	assertContains(t, got, "# DiffPal Findings Summary")
	assertContains(t, got, "review_id: review-123")
	assertContains(t, got, "## CRITICAL (1)")
	assertContains(t, got, "### internal/app/service.go")
	assertContains(t, got, "- `correctness.nil` (1)")
	assertContains(t, got, "## HIGH (2)")
	assertContains(t, got, "### internal/db/query.go")
	assertContains(t, got, "- `security.sql` (2)")
	assertContains(t, got, "  - [20-20] query concatenates untrusted input")
	assertContains(t, got, "  - [28-28] second unsafe SQL sink")
	assertContains(t, got, "## LOW (1)")

	if strings.Index(got, "## CRITICAL") > strings.Index(got, "## HIGH") {
		t.Fatalf("severity order is unstable:\n%s", got)
	}
	if strings.Index(got, "## HIGH") > strings.Index(got, "## LOW") {
		t.Fatalf("severity order is unstable:\n%s", got)
	}
}

func TestRenderSummaryHandlesEmptyBundle(t *testing.T) {
	t.Parallel()

	got := RenderSummary(findings.FindingsBundle{
		ReviewID: "review-empty",
	})

	assertContains(t, got, "# DiffPal Findings Summary")
	assertContains(t, got, "review_id: review-empty")
	assertContains(t, got, "No findings.")
}

func assertContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("summary missing %q:\n%s", want, got)
	}
}
