package azure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
)

func TestPlanThreadsUsesComparisonContextAndReconciles(t *testing.T) {
	t.Parallel()

	findingsList := []findings.Finding{
		{
			ID:         "fp-create",
			Category:   "correctness",
			Severity:   "high",
			Confidence: 0.9,
			Path:       "internal/app/service.go",
			StartLine:  10,
			Message:    "possible nil dereference",
			Evidence:   "client may be nil",
		},
		{
			ID:         "fp-update",
			Category:   "security",
			Severity:   "high",
			Confidence: 0.91,
			Path:       "internal/db/query.go",
			StartLine:  20,
			Message:    "unsafe SQL concatenation",
			Evidence:   "query concatenates input",
		},
		{
			ID:         "fp-low",
			Category:   "style",
			Severity:   "low",
			Confidence: 0.4,
			Path:       "internal/app/service.go",
			StartLine:  30,
			Message:    "style note",
		},
	}
	existing := map[string]string{
		threadKey("internal/db/query.go", 20, "security", "fp-update"): "old-fp",
	}

	plan := PlanThreads(existing, findingsList, Context{
		PullRequestID: "42",
		BaseSHA:       "base-a",
		HeadSHA:       "head-a",
	})

	if len(plan.Actions) != 2 {
		t.Fatalf("len(Actions) = %d, want 2 actionable threads", len(plan.Actions))
	}
	if plan.Actions[0].Type != ActionCreate {
		t.Fatalf("first action = %q, want create", plan.Actions[0].Type)
	}
	if plan.Actions[1].Type != ActionUpdate {
		t.Fatalf("second action = %q, want update", plan.Actions[1].Type)
	}
	if plan.Comparison.PullRequestID != "42" || plan.Comparison.BaseSHA != "base-a" || plan.Comparison.HeadSHA != "head-a" {
		t.Fatalf("unexpected comparison: %+v", plan.Comparison)
	}
	if len(plan.State) != 2 {
		t.Fatalf("len(State) = %d, want 2 actionable states", len(plan.State))
	}
}

func TestPlanThreadsKeepsSameLineFindingsDistinct(t *testing.T) {
	t.Parallel()

	items := []findings.Finding{
		{
			ID:         "fp-a",
			Category:   "security",
			Severity:   "high",
			Confidence: 0.95,
			Path:       "main.go",
			StartLine:  12,
			Message:    "first issue",
		},
		{
			ID:         "fp-b",
			Category:   "security",
			Severity:   "high",
			Confidence: 0.95,
			Path:       "main.go",
			StartLine:  12,
			Message:    "second issue",
		},
	}

	plan := PlanThreads(nil, items, Context{})
	if len(plan.State) != 2 {
		t.Fatalf("state = %d, want 2", len(plan.State))
	}
	if plan.State[0].ThreadID == plan.State[1].ThreadID {
		t.Fatalf("same-line findings share thread id %q", plan.State[0].ThreadID)
	}
}

func TestLoadExistingStateReadsPriorThreadPlan(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "threads.json")
	raw := []byte(`{
  "actions": [],
  "state": [
    {"thread_id":"a.go:10:rule-a","finding_id":"fp-a"}
  ],
  "comparison": {"pull_request_id":"11","base_sha":"b","head_sha":"h"}
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

func TestPlanThreadsWithProfileUsesExpandedInlineThreshold(t *testing.T) {
	t.Parallel()

	items := []findings.Finding{{
		ID:         "fp-inline",
		Category:   "correctness",
		Severity:   "medium",
		Confidence: 0.7,
		Path:       "main.go",
		StartLine:  12,
		Message:    "edge case",
	}}
	if got := PlanThreadsWithProfile(nil, items, Context{}, "balanced"); len(got.Actions) != 0 {
		t.Fatalf("balanced actions = %d, want 0", len(got.Actions))
	}
	if got := PlanThreadsWithProfile(nil, items, Context{}, "inline"); len(got.Actions) != 1 {
		t.Fatalf("inline actions = %d, want 1", len(got.Actions))
	}
}

func TestPolicyStatusDistinguishesBlockedReviewAndToolingError(t *testing.T) {
	t.Parallel()

	policyFail := PolicyStatus(PolicyContext{
		BlockOn:         "high",
		FatalOnFailures: true,
	}, 2, 1, false)
	if policyFail.State != StatusStateFailed {
		t.Fatalf("policy failure state = %q, want failed", policyFail.State)
	}
	if policyFail.Context != "diffpal/review" {
		t.Fatalf("Context = %q, want diffpal/review", policyFail.Context)
	}

	toolError := PolicyStatus(PolicyContext{
		BlockOn:         "high",
		FatalOnFailures: true,
	}, 0, 0, true)
	if toolError.State != StatusStateFailed {
		t.Fatalf("tooling error state = %q, want failed", toolError.State)
	}

	advisoryOnly := PolicyStatus(PolicyContext{
		BlockOn:         "high",
		FatalOnFailures: true,
	}, 0, 2, false)
	if advisoryOnly.State != StatusStateSucceeded {
		t.Fatalf("advisory state = %q, want succeeded", advisoryOnly.State)
	}
}
