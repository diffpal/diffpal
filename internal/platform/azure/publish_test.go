package azure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/identity"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/location"
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
			Evidence:   findings.NewEvidence("client may be nil"),
			Blocking:   true,
		},
		{
			ID:         "fp-update",
			Category:   "security",
			Severity:   "high",
			Confidence: 0.91,
			Path:       "internal/db/query.go",
			StartLine:  20,
			Message:    "unsafe SQL concatenation",
			Evidence:   findings.NewEvidence("query concatenates input"),
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

	if len(plan.Actions) != 3 {
		t.Fatalf("len(Actions) = %d, want 3 thread actions", len(plan.Actions))
	}
	if plan.Actions[0].Type != ActionCreate {
		t.Fatalf("first action = %q, want create", plan.Actions[0].Type)
	}
	if plan.Actions[1].Type != ActionUpdate {
		t.Fatalf("second action = %q, want update", plan.Actions[1].Type)
	}
	if plan.Actions[1].Status != ThreadStatusClosed {
		t.Fatalf("second status = %q, want closed", plan.Actions[1].Status)
	}
	if plan.Actions[0].Status != ThreadStatusActive {
		t.Fatalf("first status = %q, want active", plan.Actions[0].Status)
	}
	if plan.Actions[2].Status != ThreadStatusClosed {
		t.Fatalf("third status = %q, want closed", plan.Actions[2].Status)
	}
	if plan.Comparison.PullRequestID != "42" || plan.Comparison.BaseSHA != "base-a" || plan.Comparison.HeadSHA != "head-a" {
		t.Fatalf("unexpected comparison: %+v", plan.Comparison)
	}
	if len(plan.State) != 3 {
		t.Fatalf("len(State) = %d, want 3 thread states", len(plan.State))
	}
}

func TestThreadBodyIncludesImpactWhenPresent(t *testing.T) {
	t.Parallel()

	body := threadBody(findings.Finding{
		Title:      "possible regression",
		Category:   "security",
		Severity:   "high",
		Confidence: 0.91,
		Message:    "unsafe SQL concatenation",
		Evidence:   findings.NewEvidence("query concatenates input"),
		Impact:     findings.NewImpact("malicious users can delete unrelated sessions"),
		Suggestion: "Use a parameterized query.",
	})
	if !strings.Contains(body, "unsafe SQL concatenation") {
		t.Fatalf("thread body missing finding message:\n%s", body)
	}
	if !strings.Contains(body, "- **Evidence**: query concatenates input") {
		t.Fatalf("thread body missing compact evidence:\n%s", body)
	}
	if !strings.Contains(body, "- **Impact**: malicious users can delete unrelated sessions") {
		t.Fatalf("thread body missing impact:\n%s", body)
	}
	if !strings.Contains(body, "- **Suggestion**: Use a parameterized query.") {
		t.Fatalf("thread body missing suggestion:\n%s", body)
	}
	for _, noisy := range []string{"Category:", "Severity:", "Confidence:", "High security", "changed_line"} {
		if strings.Contains(body, noisy) {
			t.Fatalf("thread body contains noisy %q:\n%s", noisy, body)
		}
	}
}

func TestThreadPayloadAttachesFileContext(t *testing.T) {
	t.Parallel()

	payload := threadPayload("body", ThreadStatusActive, "internal/app/service.go", 27, 29)
	if payload.ThreadContext == nil {
		t.Fatal("ThreadContext = nil, want file context")
	}
	if payload.ThreadContext.FilePath == nil || *payload.ThreadContext.FilePath != "internal/app/service.go" {
		t.Fatalf("FilePath = %v, want internal/app/service.go", payload.ThreadContext.FilePath)
	}
	if payload.ThreadContext.RightFileStart == nil || payload.ThreadContext.RightFileStart.Line == nil || *payload.ThreadContext.RightFileStart.Line != 27 {
		t.Fatalf("RightFileStart = %#v, want line 27", payload.ThreadContext.RightFileStart)
	}
	if payload.ThreadContext.RightFileEnd == nil || payload.ThreadContext.RightFileEnd.Line == nil || *payload.ThreadContext.RightFileEnd.Line != 29 {
		t.Fatalf("RightFileEnd = %#v, want line 29", payload.ThreadContext.RightFileEnd)
	}
	if payload.Status == nil || string(*payload.Status) != string(ThreadStatusActive) {
		t.Fatalf("Status = %v, want active", payload.Status)
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
			Blocking:   true,
		},
		{
			ID:         "fp-b",
			Category:   "security",
			Severity:   "high",
			Confidence: 0.95,
			Path:       "main.go",
			StartLine:  12,
			Message:    "second issue",
			Blocking:   true,
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

func TestPlanThreadsUpdatesSinglePriorLocationWhenFindingIDChanges(t *testing.T) {
	t.Parallel()

	items := []findings.Finding{{
		ID:         "fp-new",
		Category:   "security",
		Severity:   "high",
		Confidence: 0.95,
		Path:       "main.go",
		StartLine:  12,
		Message:    "updated issue",
		Evidence:   findings.NewEvidence("same location"),
		Blocking:   true,
	}}
	existing := map[string]string{
		threadKey("main.go", 12, "security", "fp-old"): threadStateSignature([]string{"fp-old"}, ThreadStatusActive),
	}

	plan := PlanThreads(existing, items, Context{})
	if len(plan.Actions) != 1 {
		t.Fatalf("actions = %d, want 1", len(plan.Actions))
	}
	if plan.Actions[0].Type != ActionUpdate {
		t.Fatalf("action = %q, want update", plan.Actions[0].Type)
	}
	if plan.Actions[0].ThreadID != threadKey("main.go", 12, "security", "fp-old") {
		t.Fatalf("ThreadID = %q, want prior thread id", plan.Actions[0].ThreadID)
	}
	if plan.State[0].ThreadID != plan.Actions[0].ThreadID {
		t.Fatalf("state ThreadID = %q, want action ThreadID %q", plan.State[0].ThreadID, plan.Actions[0].ThreadID)
	}
}

func TestPlanThreadsUpdatesWhenBlockingStatusChanges(t *testing.T) {
	t.Parallel()

	items := []findings.Finding{{
		ID:         "fp-same",
		Category:   "security",
		Severity:   "high",
		Confidence: 0.95,
		Path:       "main.go",
		StartLine:  12,
		Message:    "escalated issue",
		Blocking:   true,
	}}
	existing := map[string]string{
		threadKey("main.go", 12, "security", "fp-same"): threadStateSignature([]string{"fp-same"}, ThreadStatusClosed),
	}

	plan := PlanThreads(existing, items, Context{})
	if len(plan.Actions) != 1 {
		t.Fatalf("actions = %d, want 1", len(plan.Actions))
	}
	if plan.Actions[0].Type != ActionUpdate {
		t.Fatalf("action = %q, want update", plan.Actions[0].Type)
	}
	if plan.Actions[0].Status != ThreadStatusActive {
		t.Fatalf("status = %q, want active", plan.Actions[0].Status)
	}
}

func TestPlanThreadsKeepsLineRange(t *testing.T) {
	t.Parallel()

	items := []findings.Finding{{
		ID:         "fp-range",
		Category:   "correctness",
		Severity:   "high",
		Confidence: 0.95,
		Path:       "main.go",
		StartLine:  12,
		EndLine:    16,
		Message:    "range issue",
		Blocking:   true,
	}}

	plan := PlanThreads(nil, items, Context{})
	if len(plan.Actions) != 1 {
		t.Fatalf("actions = %d, want 1", len(plan.Actions))
	}
	if plan.Actions[0].Line != 12 || plan.Actions[0].EndLine != 16 {
		t.Fatalf("action range = %d-%d, want 12-16", plan.Actions[0].Line, plan.Actions[0].EndLine)
	}
}

func TestLoadExistingStateReadsPriorThreadPlan(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "threads.json")
	raw := []byte(`{
  "actions": [],
  "state": [
    {"thread_id":"a.go:10:rule-a","finding_id":"fp-a","finding_ids":["fp-a"],"status":"active"}
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
	if state["a.go:10:rule-a"] != "fp-a|active" {
		t.Fatalf("unexpected state map: %#v", state)
	}
}

func TestPlanThreadsWithProfilePublishesAllFindingsInBothProfiles(t *testing.T) {
	t.Parallel()

	items := []findings.Finding{{
		ID:         "fp-inline",
		Category:   "correctness",
		Severity:   "medium",
		Confidence: 0.1,
		Path:       "main.go",
		StartLine:  12,
		Message:    "edge case",
	}}
	if got := PlanThreadsWithProfile(nil, items, Context{}, "balanced"); len(got.Actions) != 1 {
		t.Fatalf("balanced actions = %d, want 1", len(got.Actions))
	}
	if got := PlanThreadsWithProfile(nil, items, Context{}, "inline"); len(got.Actions) != 1 {
		t.Fatalf("inline actions = %d, want 1", len(got.Actions))
	}
}

func TestPlanThreadsClosesAdvisoryFinding(t *testing.T) {
	t.Parallel()

	items := []findings.Finding{{
		ID:         "fp-advisory",
		Category:   "security",
		Severity:   "medium",
		Confidence: 0.01,
		Path:       "main.go",
		StartLine:  18,
		Message:    "advisory issue",
	}}

	plan := PlanThreadsWithProfile(nil, items, Context{}, "balanced")
	if len(plan.Actions) != 1 {
		t.Fatalf("actions = %d, want 1", len(plan.Actions))
	}
	if plan.Actions[0].Type != ActionCreate {
		t.Fatalf("action = %q, want create", plan.Actions[0].Type)
	}
	if plan.Actions[0].Status != ThreadStatusClosed {
		t.Fatalf("status = %q, want closed", plan.Actions[0].Status)
	}
}

func TestPlanThreadsGroupsFallbackThreadsByBlockingStatus(t *testing.T) {
	t.Parallel()

	items := []findings.Finding{
		{
			ID:         "fp-blocking-no-path",
			Category:   "security",
			Severity:   "high",
			Confidence: 0.0,
			StartLine:  12,
			Message:    "missing path",
			Blocking:   true,
		},
		{
			ID:         "fp-advisory-no-line",
			Category:   "security",
			Severity:   "high",
			Confidence: 0.0,
			Path:       "main.go",
			Message:    "missing line",
		},
		{
			ID:         "fp-advisory-no-category",
			Severity:   "high",
			Confidence: 0.0,
			Path:       "main.go",
			StartLine:  14,
			Message:    "missing category",
		},
	}

	plan := PlanThreads(nil, items, Context{})
	if len(plan.Actions) != 2 {
		t.Fatalf("actions = %d, want 2", len(plan.Actions))
	}
	if plan.Actions[0].ThreadID != fallbackBlockingThreadID || plan.Actions[0].Status != ThreadStatusActive {
		t.Fatalf("first fallback action = %#v, want blocking active fallback", plan.Actions[0])
	}
	if plan.Actions[1].ThreadID != fallbackAdvisoryThreadID || plan.Actions[1].Status != ThreadStatusClosed {
		t.Fatalf("second fallback action = %#v, want advisory closed fallback", plan.Actions[1])
	}
	if len(plan.Actions[0].FindingIDs) != 1 || plan.Actions[0].FindingIDs[0] != "fp-blocking-no-path" {
		t.Fatalf("blocking fallback finding ids = %#v, want blocking finding", plan.Actions[0].FindingIDs)
	}
	if len(plan.Actions[1].FindingIDs) != 2 {
		t.Fatalf("advisory fallback finding ids = %#v, want 2 advisory findings", plan.Actions[1].FindingIDs)
	}
	if plan.Actions[1].Path != "" || plan.Actions[1].Line != 0 || plan.Actions[1].EndLine != 0 {
		t.Fatalf("advisory fallback location = %q:%d-%d, want no file context", plan.Actions[1].Path, plan.Actions[1].Line, plan.Actions[1].EndLine)
	}
	if !strings.Contains(plan.Actions[1].Body, "Non-blocking findings without canonical file/line mapping") {
		t.Fatalf("advisory fallback body = %q, want fallback heading", plan.Actions[1].Body)
	}
}

func TestPolicyStatusDistinguishesBlockedReviewAndToolingError(t *testing.T) {
	t.Parallel()

	policyFail := PolicyStatus(PolicyContext{
		BlockOn:         "high",
		GateEnabled:     true,
		FatalOnFailures: true,
	}, 2, 1, false)
	if policyFail.State != StatusStateFailed {
		t.Fatalf("policy failure state = %q, want failed", policyFail.State)
	}
	if policyFail.Context != "diffpal/review" {
		t.Fatalf("Context = %q, want diffpal/review", policyFail.Context)
	}

	gateDisabled := PolicyStatus(PolicyContext{
		BlockOn:         "high",
		GateEnabled:     false,
		FatalOnFailures: true,
	}, 2, 1, false)
	if gateDisabled.State != StatusStateSucceeded {
		t.Fatalf("gate-disabled state = %q, want succeeded", gateDisabled.State)
	}
	if !strings.Contains(gateDisabled.Description, "gate disabled") {
		t.Fatalf("gate-disabled description = %q, want gate disabled", gateDisabled.Description)
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

func TestReviewerIDFromConnectionDataPrefersAuthorizedUser(t *testing.T) {
	t.Parallel()

	authorizedID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	authenticatedID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	got, err := reviewerIDFromConnectionData(&location.ConnectionData{
		AuthorizedUser: &identity.Identity{
			Id: &authorizedID,
		},
		AuthenticatedUser: &identity.Identity{
			Id: &authenticatedID,
		},
	})
	if err != nil {
		t.Fatalf("reviewerIDFromConnectionData() error = %v", err)
	}
	if got != authorizedID.String() {
		t.Fatalf("reviewer id = %q, want authorized user %q", got, authorizedID.String())
	}
}
