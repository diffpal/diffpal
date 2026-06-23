package gitlab

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
)

func TestPlanDiscussionsCreatesAllFindingsAndResolvesAdvisories(t *testing.T) {
	t.Parallel()

	findingsList := []findings.Finding{
		{
			ID:         "fp-update",
			Category:   "security",
			Severity:   "high",
			Confidence: 0.9,
			Path:       "internal/db/query.go",
			StartLine:  12,
			Message:    "unsafe SQL concatenation",
		},
		{
			ID:         "fp-skip",
			Category:   "correctness",
			Severity:   "critical",
			Confidence: 0.95,
			Path:       "internal/app/service.go",
			StartLine:  8,
			Message:    "possible nil dereference",
			Blocking:   true,
		},
		{
			ID:         "fp-advisory",
			Category:   "maintainability",
			Severity:   "medium",
			Confidence: 0.7,
			Path:       "internal/app/service.go",
			StartLine:  30,
			Message:    "branch is unreachable",
		},
	}

	existing := map[string]string{
		discussionKey("internal/db/query.go", 12, "security", "fp-update"):    "old-fp",
		discussionKey("internal/app/service.go", 8, "correctness", "fp-skip"): "fp-skip",
	}
	plan := PlanDiscussions(existing, findingsList, []string{"high"})

	if len(plan.Actions) != 3 {
		t.Fatalf("len(Actions) = %d, want 3 actions", len(plan.Actions))
	}
	if plan.Actions[0].Type != ActionUpdate {
		t.Fatalf("first action = %q, want update", plan.Actions[0].Type)
	}
	if plan.Actions[1].Type != ActionSkip {
		t.Fatalf("second action = %q, want skip", plan.Actions[1].Type)
	}
	if plan.Actions[2].Blocking || !plan.Actions[2].Resolved {
		t.Fatalf("advisory action = %+v, want non-blocking resolved", plan.Actions[2])
	}
	if len(plan.State) != 3 {
		t.Fatalf("len(State) = %d, want 3 states", len(plan.State))
	}
}

func TestDiscussionBodyUsesSafeFenceForBackticks(t *testing.T) {
	t.Parallel()

	body := discussionBody(findings.Finding{
		Category:   "security",
		Severity:   "high",
		Confidence: 0.9,
		Message:    "unsafe markdown",
		Evidence:   findings.NewEvidence("```go\nfmt.Println(\"x\")\n```"),
		Impact:     findings.NewImpact("reviewers can see the concrete consequence"),
		Suggestion: "````suggestion\nx\n````",
	})

	if !strings.Contains(body, "**Impact**: reviewers can see the concrete consequence") {
		t.Fatalf("body missing impact:\n%s", body)
	}
	if !strings.Contains(body, "- **Suggestion**: ````suggestion\nx\n````") {
		t.Fatalf("body missing suggestion:\n%s", body)
	}
	if strings.Contains(body, "Confidence") || strings.Contains(body, "Provider") {
		t.Fatalf("body contains noisy fields:\n%s", body)
	}
}

func TestDiscussionBodyFallsBackToTitle(t *testing.T) {
	t.Parallel()

	body := discussionBody(findings.Finding{
		Category:   "correctness",
		Severity:   "medium",
		Confidence: 0.9,
		Title:      "title only finding",
	})
	if !strings.Contains(body, "title only finding") {
		t.Fatalf("body missing title fallback:\n%s", body)
	}
}

func TestPlanDiscussionsUpdatesSinglePriorLocationWhenFindingIDChanges(t *testing.T) {
	t.Parallel()

	items := []findings.Finding{{
		ID:         "fp-new",
		Category:   "security",
		Severity:   "high",
		Confidence: 0.95,
		Path:       "main.go",
		StartLine:  12,
		Message:    "updated issue",
	}}
	existing := map[string]string{
		discussionKey("main.go", 12, "security", "fp-old"): "fp-old",
	}

	plan := PlanDiscussions(existing, items, []string{"high"})
	if len(plan.Actions) != 1 {
		t.Fatalf("actions = %d, want 1", len(plan.Actions))
	}
	if plan.Actions[0].Type != ActionUpdate {
		t.Fatalf("action = %q, want update", plan.Actions[0].Type)
	}
	if plan.Actions[0].ThreadHash != discussionKey("main.go", 12, "security", "fp-old") {
		t.Fatalf("ThreadHash = %q, want prior thread hash", plan.Actions[0].ThreadHash)
	}
	if plan.State[0].ThreadHash != plan.Actions[0].ThreadHash {
		t.Fatalf("state ThreadHash = %q, want action ThreadHash %q", plan.State[0].ThreadHash, plan.Actions[0].ThreadHash)
	}
}

func TestLoadExistingStateReadsPriorDiscussionPlan(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "discussions.json")
	raw := []byte(`{
  "actions": [],
  "state": [
    {"thread_hash":"a.go:10:rule-a","finding_id":"fp-a"}
  ]
}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	state, err := LoadExistingState(path)
	if err != nil {
		t.Fatalf("LoadExistingState() error = %v", err)
	}
	if state["a.go:10:rule-a"] != "fp-a" {
		t.Fatalf("unexpected state map: %#v", state)
	}
}

func TestSummarizeDecisionAndApprovalPolicy(t *testing.T) {
	t.Parallel()

	bundle := findings.FindingsBundle{
		HeadSHA: "head-a",
		Findings: []findings.Finding{
			{
				Category: "security",
				Severity: "high",
				Path:     "internal/db/query.go",
				Message:  "unsafe SQL concatenation",
			},
			{
				Category: "maintainability",
				Severity: "medium",
				Path:     "internal/app/service.go",
				Message:  "branch is unreachable",
			},
		},
	}
	result := SummarizeDecision(bundle, []string{"high"})
	if result.Decision != MergeDecisionFail {
		t.Fatalf("Decision = %q, want fail", result.Decision)
	}
	if result.BlockCount != 1 || result.AdvisoryCount != 1 {
		t.Fatalf("unexpected counts: %+v", result)
	}

	if CanAutoApprove(ApprovalPolicy{
		Enabled:       true,
		RequireSHA:    "head-a",
		ApproverID:    "bot-1",
		ApproveOnPass: true,
	}, bundle, "head-a") {
		t.Fatal("CanAutoApprove() = true, want false when findings exist")
	}

	if !CanAutoApprove(ApprovalPolicy{
		Enabled:       true,
		RequireSHA:    "head-b",
		ApproverID:    "bot-1",
		ApproveOnPass: true,
	}, findings.FindingsBundle{HeadSHA: "head-b"}, "head-b") {
		t.Fatal("CanAutoApprove() = false, want true for clean pass on matching SHA")
	}
}

func TestSummarizeDecisionIgnoresUnknownBlockOnSeverity(t *testing.T) {
	t.Parallel()

	result := SummarizeDecision(findings.FindingsBundle{Findings: []findings.Finding{{
		Category: "maintainability",
		Severity: "medium",
		Path:     "main.go",
		Message:  "advisory",
	}}}, []string{"unknown"})
	if result.Decision != MergeDecisionWarn {
		t.Fatalf("Decision = %q, want warn", result.Decision)
	}
	if result.BlockCount != 0 || result.AdvisoryCount != 1 {
		t.Fatalf("unexpected counts: %+v", result)
	}
}

func TestPolicyStatus(t *testing.T) {
	t.Parallel()

	blocked := PolicyStatus(2, 1, true, "https://gitlab.example/job/1")
	if blocked.State != "failed" || blocked.Context != "diffpal/review" || blocked.TargetURL == "" {
		t.Fatalf("blocked status = %+v, want failed diffpal/review with target URL", blocked)
	}

	advisory := PolicyStatus(0, 1, true, "")
	if advisory.State != "success" || !strings.Contains(advisory.Description, "Advisory findings") {
		t.Fatalf("advisory status = %+v, want success advisory description", advisory)
	}
}

func TestParseDiffLinesIgnoresNoNewlineMarker(t *testing.T) {
	t.Parallel()

	lines := parseDiffLines("@@ -1,2 +1,3 @@\n old\n+new\n\\ No newline at end of file\n")
	if len(lines) != 2 {
		t.Fatalf("len(lines) = %d, want 2: %+v", len(lines), lines)
	}
	if lines[1].NewLine != 2 || lines[1].OldLine != 0 {
		t.Fatalf("added line = %+v, want new line 2 only", lines[1])
	}
}
