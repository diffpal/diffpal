package github

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
)

func TestPlanInlineCommentsFiltersAndReconciles(t *testing.T) {
	t.Parallel()

	findingsList := []findings.Finding{
		{
			ID:         "fp-create",
			RuleID:     "correctness.nil",
			Severity:   "high",
			Confidence: 0.95,
			Path:       "internal/app/service.go",
			StartLine:  10,
			Message:    "possible nil dereference",
		},
		{
			ID:         "fp-update",
			RuleID:     "security.sql",
			Severity:   "high",
			Confidence: 0.91,
			Path:       "internal/db/query.go",
			StartLine:  22,
			Message:    "unsafe SQL concatenation",
		},
		{
			ID:         "fp-skip",
			RuleID:     "maintainability.deadcode",
			Severity:   "medium",
			Confidence: 0.88,
			Path:       "internal/app/service.go",
			StartLine:  31,
			Message:    "unreachable branch",
		},
		{
			ID:         "fp-low-confidence",
			RuleID:     "style.nit",
			Severity:   "low",
			Confidence: 0.4,
			Path:       "internal/app/service.go",
			StartLine:  40,
			Message:    "style note",
		},
	}

	existing := map[string]string{
		commentKey("internal/db/query.go", 22, "security.sql"):                "fp-old",
		commentKey("internal/app/service.go", 31, "maintainability.deadcode"): "fp-skip",
	}
	plan := PlanInlineComments(existing, findingsList)

	if len(plan.Actions) != 3 {
		t.Fatalf("len(Actions) = %d, want 3", len(plan.Actions))
	}
	if plan.Actions[0].Type != ActionCreate {
		t.Fatalf("first action = %q, want create", plan.Actions[0].Type)
	}
	if plan.Actions[1].Type != ActionUpdate {
		t.Fatalf("second action = %q, want update", plan.Actions[1].Type)
	}
	if plan.Actions[2].Type != ActionSkip {
		t.Fatalf("third action = %q, want skip", plan.Actions[2].Type)
	}
	if len(plan.State) != 3 {
		t.Fatalf("len(State) = %d, want 3 high-confidence findings", len(plan.State))
	}
}

func TestLoadExistingStateReadsPriorPlan(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "comments.json")
	raw := []byte(`{
  "actions": [],
  "state": [
    {"key":"a.go:10:rule-a","finding_id":"fp-a"},
    {"key":"b.go:20:rule-b","finding_id":"fp-b"}
  ]
}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	state, err := LoadExistingState(path)
	if err != nil {
		t.Fatalf("LoadExistingState() error = %v", err)
	}
	if len(state) != 2 {
		t.Fatalf("len(state) = %d, want 2", len(state))
	}
	if state["a.go:10:rule-a"] != "fp-a" {
		t.Fatalf("unexpected state map: %#v", state)
	}
}
