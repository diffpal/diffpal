package github

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
	"github.com/diffpal/diffpal/internal/markdown"
)

func TestPlanInlineCommentsFiltersAndReconciles(t *testing.T) {
	t.Parallel()

	findingsList := []findings.Finding{
		{
			ID:         "fp-create",
			Category:   "correctness",
			Severity:   "high",
			Confidence: 0.95,
			Path:       "internal/app/service.go",
			StartLine:  10,
			Message:    "possible nil dereference",
		},
		{
			ID:         "fp-update",
			Category:   "security",
			Severity:   "high",
			Confidence: 0.91,
			Path:       "internal/db/query.go",
			StartLine:  22,
			Message:    "unsafe SQL concatenation",
		},
		{
			ID:         "fp-skip",
			Category:   "maintainability",
			Severity:   "medium",
			Confidence: 0.88,
			Path:       "internal/app/service.go",
			StartLine:  31,
			Message:    "unreachable branch",
		},
		{
			ID:         "fp-low-confidence",
			Category:   "style",
			Severity:   "low",
			Confidence: 0.4,
			Path:       "internal/app/service.go",
			StartLine:  40,
			Message:    "style note",
		},
	}

	existing := map[string]string{
		commentKey("internal/db/query.go", 22, "security", "fp-update"):         "fp-old",
		commentKey("internal/app/service.go", 31, "maintainability", "fp-skip"): "fp-skip",
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

func TestPlanInlineCommentsKeepsSameLineFindingsDistinct(t *testing.T) {
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

	plan := PlanInlineComments(nil, items)
	if len(plan.State) != 2 {
		t.Fatalf("state = %d, want 2", len(plan.State))
	}
	if plan.State[0].Key == plan.State[1].Key {
		t.Fatalf("same-line findings share key %q", plan.State[0].Key)
	}
}

func TestPlanInlineCommentsUpdatesSinglePriorLocationWhenFindingIDChanges(t *testing.T) {
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
		commentKey("main.go", 12, "security", "fp-old"): "fp-old",
	}

	plan := PlanInlineComments(existing, items)
	if len(plan.Actions) != 1 {
		t.Fatalf("actions = %d, want 1", len(plan.Actions))
	}
	if plan.Actions[0].Type != ActionUpdate {
		t.Fatalf("action = %q, want update", plan.Actions[0].Type)
	}
}

func TestPlanInlineCommentsCreatesWhenPriorLocationIsAmbiguous(t *testing.T) {
	t.Parallel()

	items := []findings.Finding{{
		ID:         "fp-new",
		Category:   "security",
		Severity:   "high",
		Confidence: 0.95,
		Path:       "main.go",
		StartLine:  12,
		Message:    "third issue",
	}}
	existing := map[string]string{
		commentKey("main.go", 12, "security", "fp-old-a"): "fp-old-a",
		commentKey("main.go", 12, "security", "fp-old-b"): "fp-old-b",
	}

	plan := PlanInlineComments(existing, items)
	if len(plan.Actions) != 1 {
		t.Fatalf("actions = %d, want 1", len(plan.Actions))
	}
	if plan.Actions[0].Type != ActionCreate {
		t.Fatalf("action = %q, want create", plan.Actions[0].Type)
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

func TestPlanInlineCommentsWithProfileUsesExpandedInlineThreshold(t *testing.T) {
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
	if got := PlanInlineCommentsWithProfile(nil, items, "balanced"); len(got.Actions) != 0 {
		t.Fatalf("balanced actions = %d, want 0", len(got.Actions))
	}
	if got := PlanInlineCommentsWithProfile(nil, items, "inline"); len(got.Actions) != 1 {
		t.Fatalf("inline actions = %d, want 1", len(got.Actions))
	}
}

func TestPlanInlineCommentsCanPublishAllFindings(t *testing.T) {
	t.Parallel()

	items := []findings.Finding{{
		ID:         "fp-low",
		Category:   "correctness",
		Severity:   "medium",
		Confidence: 0.2,
		Path:       "main.go",
		StartLine:  12,
		Message:    "low confidence but provider emitted it",
	}}

	plan := PlanInlineCommentsWithOptions(nil, items, CommentOptions{AllFindings: true})
	if len(plan.Actions) != 1 {
		t.Fatalf("actions = %d, want 1", len(plan.Actions))
	}
}

func TestValidateInlineFindingsRejectsUnplaceableFindings(t *testing.T) {
	t.Parallel()

	err := ValidateInlineFindings([]findings.Finding{{ID: "fp-no-line", Path: "main.go"}})
	if err == nil {
		t.Fatal("ValidateInlineFindings() error = nil, want missing line error")
	}
	if !strings.Contains(err.Error(), "missing start line") {
		t.Fatalf("error = %v, want missing start line", err)
	}

	err = ValidateInlineFindings([]findings.Finding{{ID: "fp-no-path", StartLine: 12}})
	if err == nil {
		t.Fatal("ValidateInlineFindings() error = nil, want missing path error")
	}
	if !strings.Contains(err.Error(), "missing path") {
		t.Fatalf("error = %v, want missing path", err)
	}
}

func TestPlanInlineCommentsCanIncludePermanentLink(t *testing.T) {
	t.Parallel()

	plan := PlanInlineCommentsWithOptions(nil, []findings.Finding{{
		ID:         "fp-sql",
		Category:   "security",
		Severity:   "high",
		Confidence: 0.95,
		Path:       "internal/db/query.go",
		StartLine:  12,
		EndLine:    17,
		Message:    "query concatenates untrusted input",
		Evidence:   findings.NewEvidence("Line 17 builds SQL by concatenating user input."),
		Suggestion: "Use a parameterized statement.",
	}}, CommentOptions{
		Profile: "balanced",
		Links: markdown.FindingLinkFunc(func(findings.Finding) (string, bool) {
			return "https://github.com/acme/diffpal/blob/head-a/internal/db/query.go#L12-L17", true
		}),
	})

	if len(plan.Actions) != 1 {
		t.Fatalf("actions = %d, want 1", len(plan.Actions))
	}
	body := plan.Actions[0].Body
	for _, want := range []string{
		"**High security**: query concatenates untrusted input",
		"https://github.com/acme/diffpal/blob/head-a/internal/db/query.go#L12-L17",
		"- **Evidence**: Line 17 builds SQL by concatenating user input.",
		"- **Suggestion**: Use a parameterized statement.",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("comment body missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "```") {
		t.Fatalf("comment body contains fenced code block:\n%s", body)
	}
	if strings.Contains(body, "`L12-L17`") {
		t.Fatalf("comment body repeats linked line range in header:\n%s", body)
	}
	if strings.Contains(body, "**Confidence**") {
		t.Fatalf("comment body contains confidence:\n%s", body)
	}
}

func TestPlanInlineCommentsKeepsFindingLineRange(t *testing.T) {
	t.Parallel()

	plan := PlanInlineComments(nil, []findings.Finding{{
		ID:         "fp-range",
		Category:   "correctness",
		Severity:   "high",
		Confidence: 0.95,
		Path:       "internal/cmd/review.go",
		StartLine:  473,
		EndLine:    475,
		Message:    "range issue",
	}})

	if len(plan.Actions) != 1 {
		t.Fatalf("actions = %d, want 1", len(plan.Actions))
	}
	action := plan.Actions[0]
	if action.Line != 473 || action.EndLine != 475 {
		t.Fatalf("action range = %d-%d, want 473-475", action.Line, action.EndLine)
	}
}
