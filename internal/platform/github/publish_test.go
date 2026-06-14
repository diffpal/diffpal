package github

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
)

func TestBuildCheckRunPayloadBatchesAnnotationsAndFailsOnBlockingFindings(t *testing.T) {
	t.Parallel()

	bundle := findings.FindingsBundle{
		ReviewID: "review-github",
		BaseSHA:  "base-a",
		HeadSHA:  "head-a",
		Findings: make([]findings.Finding, 0, 55),
	}
	for i := 0; i < 55; i++ {
		severity := "medium"
		blocking := false
		if i == 0 {
			severity = "high"
			blocking = true
		}
		message := "message"
		if i == 0 {
			message = "message 0"
		}
		bundle.Findings = append(bundle.Findings, findings.Finding{
			Category:  "correctness",
			Severity:  severity,
			Blocking:  blocking,
			Path:      "internal/file.go",
			StartLine: i + 1,
			EndLine:   i + 1,
			Title:     "finding",
			Message:   message,
		})
	}

	payload := BuildCheckRunPayload(Context{HeadSHA: "head-a"}, bundle, CheckRunSummary(bundle))

	if payload.Name != "diffpal-checks" {
		t.Fatalf("Name = %q, want diffpal-checks", payload.Name)
	}
	if payload.Conclusion != "failure" {
		t.Fatalf("Conclusion = %q, want failure", payload.Conclusion)
	}
	if payload.HeadSHA != "head-a" {
		t.Fatalf("HeadSHA = %q, want head-a", payload.HeadSHA)
	}
	if payload.Count != 55 {
		t.Fatalf("Count = %d, want 55", payload.Count)
	}
	if len(payload.AnnotationBatches) != 2 {
		t.Fatalf("AnnotationBatches = %d, want 2", len(payload.AnnotationBatches))
	}
	if len(payload.AnnotationBatches[0].Annotations) != 50 {
		t.Fatalf("first batch size = %d, want 50", len(payload.AnnotationBatches[0].Annotations))
	}
	if len(payload.AnnotationBatches[1].Annotations) != 5 {
		t.Fatalf("second batch size = %d, want 5", len(payload.AnnotationBatches[1].Annotations))
	}
	if len(payload.Annotations) != 50 {
		t.Fatalf("Annotations = %d, want 50 primary annotations", len(payload.Annotations))
	}
	if payload.Annotations[0].AnnotationLevel != "failure" {
		t.Fatalf("first annotation level = %q, want failure", payload.Annotations[0].AnnotationLevel)
	}
	if payload.Annotations[1].AnnotationLevel != "warning" {
		t.Fatalf("second annotation level = %q, want warning", payload.Annotations[1].AnnotationLevel)
	}
	if payload.Annotations[0].Message != "message 0" {
		t.Fatalf("first annotation message = %q, want message 0", payload.Annotations[0].Message)
	}
	encoded, err := json.Marshal(payload.Annotations[0])
	if err != nil {
		t.Fatalf("Marshal annotation error = %v", err)
	}
	if !strings.Contains(string(encoded), `"annotation_level":"failure"`) {
		t.Fatalf("annotation JSON missing annotation_level: %s", encoded)
	}
	if strings.Contains(string(encoded), `"level"`) {
		t.Fatalf("annotation JSON contains unsupported level field: %s", encoded)
	}
}

func TestBuildCheckRunPayloadUsesReviewChannel(t *testing.T) {
	t.Parallel()

	identity, err := NewReviewIdentity("diffpal-dev")
	if err != nil {
		t.Fatalf("NewReviewIdentity() error = %v", err)
	}
	payload := BuildCheckRunPayloadWithIdentity(Context{HeadSHA: "head-a"}, findings.FindingsBundle{
		ReviewID: "review-github-dev",
		HeadSHA:  "head-a",
	}, "# DiffPal Dev Review Summary", identity)

	if payload.Name != "diffpal-dev-checks" {
		t.Fatalf("Name = %q, want diffpal-dev-checks", payload.Name)
	}
}

func TestAnnotationMessageFallsBackToTitle(t *testing.T) {
	t.Parallel()

	got := annotationMessage(findings.Finding{Title: "title only"})
	if got != "title only" {
		t.Fatalf("annotationMessage() = %q, want title only", got)
	}
}

func TestBuildCheckRunPayloadDefaultsUnknownSeverityToWarning(t *testing.T) {
	t.Parallel()

	bundle := findings.FindingsBundle{
		ReviewID: "review-github",
		BaseSHA:  "base-a",
		HeadSHA:  "head-a",
		Findings: []findings.Finding{
			{
				Category:  "correctness",
				Severity:  "unexpected",
				Path:      "internal/file.go",
				StartLine: 1,
				EndLine:   1,
				Title:     "custom finding",
				Message:   "custom message",
			},
		},
	}

	payload := BuildCheckRunPayload(Context{HeadSHA: "head-a"}, bundle, CheckRunSummary(bundle))
	if payload.Annotations[0].AnnotationLevel != "warning" {
		t.Fatalf("annotation level = %q, want warning", payload.Annotations[0].AnnotationLevel)
	}
}

func TestBuildCheckRunPayloadFailsOnFailureLevelAnnotation(t *testing.T) {
	t.Parallel()

	bundle := findings.FindingsBundle{
		ReviewID: "review-github",
		BaseSHA:  "base-a",
		HeadSHA:  "head-a",
		Findings: []findings.Finding{
			{
				Category:  "correctness",
				Severity:  "high",
				Blocking:  false,
				Path:      "internal/file.go",
				StartLine: 1,
				EndLine:   1,
				Title:     "high finding",
				Message:   "high message",
			},
		},
	}

	payload := BuildCheckRunPayload(Context{HeadSHA: "head-a"}, bundle, CheckRunSummary(bundle))
	if payload.Conclusion != "failure" {
		t.Fatalf("Conclusion = %q, want failure", payload.Conclusion)
	}
}

func TestCheckRunSummaryUsesMarkdownGrouping(t *testing.T) {
	t.Parallel()

	bundle := findings.FindingsBundle{
		ReviewID: "review-summary",
		BaseSHA:  "base-b",
		HeadSHA:  "head-b",
		Findings: []findings.Finding{
			{
				Severity:  "critical",
				Path:      "internal/db/query.go",
				StartLine: 9,
				EndLine:   9,
				Message:   "unsanitized SQL input",
			},
		},
	}

	summary := CheckRunSummary(bundle)
	assertStringContains(t, summary, "# DiffPal Review Summary", "title")
	assertStringContains(t, summary, "## Feedback on Files", "file feedback section")
	assertStringContains(t, summary, "| `internal/db/query.go` | Needs attention | critical: 1 |", "file feedback row")
	assertStringContains(t, summary, "## Detailed Comments", "detailed comments")
}

func assertStringContains(t *testing.T, got, want, label string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("summary missing %s:\n%s", label, got)
	}
}
