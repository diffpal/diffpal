package github

import (
	"encoding/json"
	"fmt"
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
		bundle.Findings = append(bundle.Findings, findings.Finding{
			RuleID:    fmt.Sprintf("rule-%d", i),
			Category:  "correctness",
			Severity:  severity,
			Blocking:  blocking,
			Path:      fmt.Sprintf("internal/file-%02d.go", i),
			StartLine: i + 1,
			EndLine:   i + 1,
			Title:     fmt.Sprintf("finding %d", i),
			Message:   fmt.Sprintf("message %d", i),
		})
	}

	payload := BuildCheckRunPayload(Context{HeadSHA: "head-a"}, bundle, CheckRunSummary(bundle))

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
	if payload.Annotations[0].AnnotationLevel != "error" {
		t.Fatalf("first annotation level = %q, want error", payload.Annotations[0].AnnotationLevel)
	}
	if payload.Annotations[1].AnnotationLevel != "warning" {
		t.Fatalf("second annotation level = %q, want warning", payload.Annotations[1].AnnotationLevel)
	}
	encoded, err := json.Marshal(payload.Annotations[0])
	if err != nil {
		t.Fatalf("Marshal annotation error = %v", err)
	}
	if !strings.Contains(string(encoded), `"annotation_level":"error"`) {
		t.Fatalf("annotation JSON missing annotation_level: %s", encoded)
	}
	if strings.Contains(string(encoded), `"level"`) {
		t.Fatalf("annotation JSON contains unsupported level field: %s", encoded)
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
				RuleID:    "security.sql",
				Severity:  "critical",
				Path:      "internal/db/query.go",
				StartLine: 9,
				EndLine:   9,
				Message:   "unsanitized SQL input",
			},
		},
	}

	summary := CheckRunSummary(bundle)
	if !strings.Contains(summary, "# DiffPal Findings Summary") {
		t.Fatalf("summary missing title:\n%s", summary)
	}
	if !strings.Contains(summary, "## CRITICAL (1)") {
		t.Fatalf("summary missing severity section:\n%s", summary)
	}
	if !strings.Contains(summary, "### internal/db/query.go") {
		t.Fatalf("summary missing file section:\n%s", summary)
	}
}
