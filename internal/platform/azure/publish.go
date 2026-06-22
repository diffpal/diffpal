package azure

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/diffpal/diffpal/internal/findings"
)

type ThreadActionType string
type ThreadStatus string

const (
	ActionCreate ThreadActionType = "create"
	ActionUpdate ThreadActionType = "update"
	ActionSkip   ThreadActionType = "skip"

	ThreadStatusActive ThreadStatus = "active"
	ThreadStatusClosed ThreadStatus = "closed"
)

const (
	fallbackBlockingThreadID = "fallback:blocking"
	fallbackAdvisoryThreadID = "fallback:advisory"
)

type ThreadAction struct {
	Type       ThreadActionType `json:"type"`
	Status     ThreadStatus     `json:"status"`
	FindingID  string           `json:"finding_id"`
	FindingIDs []string         `json:"finding_ids,omitempty"`
	Path       string           `json:"path"`
	Line       int              `json:"line"`
	EndLine    int              `json:"end_line"`
	Body       string           `json:"body"`
	ThreadID   string           `json:"thread_id"`
}

type ThreadState struct {
	ThreadID   string       `json:"thread_id"`
	FindingID  string       `json:"finding_id,omitempty"`
	FindingIDs []string     `json:"finding_ids,omitempty"`
	Status     ThreadStatus `json:"status,omitempty"`
}

type Comparison struct {
	PullRequestID string `json:"pull_request_id,omitempty"`
	BaseSHA       string `json:"base_sha,omitempty"`
	HeadSHA       string `json:"head_sha,omitempty"`
}

type ThreadPlan struct {
	Actions    []ThreadAction `json:"actions"`
	State      []ThreadState  `json:"state"`
	Comparison Comparison     `json:"comparison"`
}

func PlanThreads(existing map[string]string, findingsList []findings.Finding, ctx Context) ThreadPlan {
	return planThreads(existing, findingsList, ctx)
}

func planThreads(existing map[string]string, findingsList []findings.Finding, ctx Context) ThreadPlan {
	out := make([]ThreadAction, 0, len(findingsList))
	state := make([]ThreadState, 0, len(findingsList))
	blockingFallback := make([]findings.Finding, 0, len(findingsList))
	advisoryFallback := make([]findings.Finding, 0, len(findingsList))
	for _, f := range findingsList {
		if !hasCanonicalThreadLocation(f) {
			if f.Blocking {
				blockingFallback = append(blockingFallback, f)
			} else {
				advisoryFallback = append(advisoryFallback, f)
			}
			continue
		}
		out, state = appendFindingThread(out, state, existing, f)
	}
	out, state = appendFallbackThread(out, state, existing, fallbackBlockingThreadID, ThreadStatusActive, blockingFallback)
	out, state = appendFallbackThread(out, state, existing, fallbackAdvisoryThreadID, ThreadStatusClosed, advisoryFallback)
	return ThreadPlan{
		Actions: out,
		State:   state,
		Comparison: Comparison{
			PullRequestID: ctx.PullRequestID,
			BaseSHA:       ctx.BaseSHA,
			HeadSHA:       ctx.HeadSHA,
		},
	}
}

func appendFindingThread(out []ThreadAction, state []ThreadState, existing map[string]string, f findings.Finding) ([]ThreadAction, []ThreadState) {
	status := threadStatusForFinding(f)
	signature := threadStateSignature([]string{f.ID}, status)
	key := threadKey(f.Path, f.StartLine, f.Category, f.ID)
	actionThreadID := key
	action := ActionCreate
	prior, ok := existing[key]
	if !ok {
		var priorThreadID string
		priorThreadID, prior, ok = singleExistingForLocation(existing, threadLocationKey(f.Path, f.StartLine, f.Category))
		if ok {
			actionThreadID = priorThreadID
		}
	}
	if ok {
		if prior == signature {
			action = ActionSkip
		} else {
			action = ActionUpdate
		}
	}
	endLine := f.EndLine
	if endLine < f.StartLine {
		endLine = f.StartLine
	}
	state = append(state, ThreadState{
		ThreadID:   actionThreadID,
		FindingID:  f.ID,
		FindingIDs: []string{f.ID},
		Status:     status,
	})
	out = append(out, ThreadAction{
		Type:       action,
		Status:     status,
		FindingID:  f.ID,
		FindingIDs: []string{f.ID},
		Path:       f.Path,
		Line:       f.StartLine,
		EndLine:    endLine,
		Body:       threadBody(f),
		ThreadID:   actionThreadID,
	})
	return out, state
}

func appendFallbackThread(out []ThreadAction, state []ThreadState, existing map[string]string, threadID string, status ThreadStatus, items []findings.Finding) ([]ThreadAction, []ThreadState) {
	if len(items) == 0 {
		return out, state
	}
	ordered := orderedFindings(items)
	findingIDs := findingIDsForState(ordered)
	signature := threadStateSignature(findingIDs, status)
	action := ActionCreate
	if prior, ok := existing[threadID]; ok {
		if prior == signature {
			action = ActionSkip
		} else {
			action = ActionUpdate
		}
	}
	state = append(state, ThreadState{
		ThreadID:   threadID,
		FindingIDs: findingIDs,
		Status:     status,
	})
	out = append(out, ThreadAction{
		Type:       action,
		Status:     status,
		FindingIDs: findingIDs,
		Body:       fallbackThreadBody(ordered, status),
		ThreadID:   threadID,
	})
	return out, state
}

func hasCanonicalThreadLocation(f findings.Finding) bool {
	return strings.TrimSpace(f.Path) != "" && f.StartLine > 0 && strings.TrimSpace(f.Category) != ""
}

func threadStatusForFinding(f findings.Finding) ThreadStatus {
	if f.Blocking {
		return ThreadStatusActive
	}
	return ThreadStatusClosed
}

func threadKey(path string, line int, category string, findingID string) string {
	return threadLocationKey(path, line, category) + ":" + findingID
}

func threadLocationKey(path string, line int, category string) string {
	return fmt.Sprintf("%s:%d:%s", path, line, category)
}

func singleExistingForLocation(existing map[string]string, locationKey string) (string, string, bool) {
	var priorKey string
	var prior string
	found := false
	prefix := locationKey + ":"
	for key, findingID := range existing {
		if key != locationKey && !strings.HasPrefix(key, prefix) {
			continue
		}
		if found {
			return "", "", false
		}
		priorKey = key
		prior = findingID
		found = true
	}
	return priorKey, prior, found
}

func orderedFindings(items []findings.Finding) []findings.Finding {
	out := append([]findings.Finding(nil), items...)
	sort.SliceStable(out, func(i, j int) bool {
		left, right := out[i], out[j]
		if left.Blocking != right.Blocking {
			return left.Blocking
		}
		if left.Category != right.Category {
			return left.Category < right.Category
		}
		if left.Severity != right.Severity {
			return left.Severity < right.Severity
		}
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.StartLine != right.StartLine {
			return left.StartLine < right.StartLine
		}
		if left.Message != right.Message {
			return left.Message < right.Message
		}
		return left.ID < right.ID
	})
	return out
}

func findingIDsForState(items []findings.Finding) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func threadStateSignature(findingIDs []string, status ThreadStatus) string {
	parts := append([]string(nil), findingIDs...)
	sort.Strings(parts)
	return strings.Join(parts, ",") + "|" + string(status)
}

func threadBody(f findings.Finding) string {
	out := strings.Builder{}
	fmt.Fprintf(&out, "%s\n", firstAzureNonEmpty(strings.TrimSpace(f.Message), strings.TrimSpace(f.Title)))
	if evidence := compactEvidenceText(f); evidence != "" {
		fmt.Fprintf(&out, "- **Evidence**: %s\n", evidence)
	}
	if impact := compactImpactText(f); impact != "" {
		fmt.Fprintf(&out, "- **Impact**: %s\n", impact)
	}
	if suggestion := strings.TrimSpace(f.Suggestion); suggestion != "" {
		fmt.Fprintf(&out, "- **Suggestion**: %s\n", suggestion)
	}
	return strings.TrimSpace(out.String())
}

func fallbackThreadBody(items []findings.Finding, status ThreadStatus) string {
	out := strings.Builder{}
	if status == ThreadStatusActive {
		out.WriteString("Blocking findings without canonical file/line mapping")
	} else {
		out.WriteString("Non-blocking findings without canonical file/line mapping")
	}
	for _, item := range items {
		fmt.Fprintf(&out, "\n\n%s", threadBody(item))
	}
	return strings.TrimSpace(out.String())
}

func compactEvidenceText(f findings.Finding) string {
	parts := make([]string, 0, 3)
	appendUnique := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		for _, part := range parts {
			if strings.EqualFold(part, value) {
				return
			}
		}
		parts = append(parts, value)
	}

	appendUnique(f.Evidence.Anchor)
	appendUnique(f.Evidence.ReasoningBasis)
	source := strings.TrimSpace(f.Evidence.Source)
	if source != "" && source != "changed_line" && source != "legacy" {
		appendUnique(source)
	}
	return strings.Join(parts, " ")
}

func compactImpactText(f findings.Finding) string {
	summary := strings.TrimSpace(f.Impact.Summary)
	scope := strings.TrimSpace(f.Impact.Scope)
	switch scope {
	case "", "legacy", "changed behavior":
		return summary
	}
	if summary == "" {
		return scope
	}
	return summary + " Scope: " + scope
}

func firstAzureNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

type StatusState string

const (
	StatusStateSucceeded StatusState = "succeeded"
	StatusStatePending   StatusState = "pending"
	StatusStateFailed    StatusState = "failed"
	StatusStateError     StatusState = "error"
)

type PolicyContext struct {
	StatusName      string
	BlockOn         string
	GateEnabled     bool
	FatalOnFailures bool
}

type StatusPayload struct {
	Name        string      `json:"name"`
	State       StatusState `json:"state"`
	Description string      `json:"description"`
	Context     string      `json:"context"`
}

func EvaluateStatus(bundleCount int, blockingCount int, gateEnabled bool, failed bool) StatusPayload {
	name := "DiffPal Review"
	ctx := "diffpal/review"
	if failed {
		return StatusPayload{
			Name:        name,
			State:       StatusStateFailed,
			Description: "DiffPal tooling error",
			Context:     ctx,
		}
	}
	if blockingCount > 0 {
		if !gateEnabled {
			return StatusPayload{
				Name:        name,
				State:       StatusStateSucceeded,
				Description: fmt.Sprintf("%d blocking findings; gate disabled", blockingCount),
				Context:     ctx,
			}
		}
		return StatusPayload{
			Name:        name,
			State:       StatusStateFailed,
			Description: fmt.Sprintf("%d blocking findings", blockingCount),
			Context:     ctx,
		}
	}
	if bundleCount > 0 {
		return StatusPayload{
			Name:        name,
			State:       StatusStateSucceeded,
			Description: "Advisory findings present without merge blockers",
			Context:     ctx,
		}
	}
	return StatusPayload{
		Name:        name,
		State:       StatusStateSucceeded,
		Description: "DiffPal completed with no findings",
		Context:     ctx,
	}
}

func IsPass(blockingCount int, failedPolicy bool, toolError bool) bool {
	if failedPolicy || toolError {
		return false
	}
	return blockingCount == 0
}

func PolicyStatus(ctx PolicyContext, blockingCount int, advisoryCount int, toolFailure bool) StatusPayload {
	toolError := toolFailure && ctx.FatalOnFailures
	return EvaluateStatus(blockingCount+advisoryCount, blockingCount, ctx.GateEnabled, toolError)
}

func LoadExistingState(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var plan ThreadPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		return nil, err
	}
	if len(plan.State) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(plan.State))
	for _, item := range plan.State {
		ids := item.FindingIDs
		if len(ids) == 0 && item.FindingID != "" {
			ids = []string{item.FindingID}
		}
		out[item.ThreadID] = threadStateSignature(ids, item.Status)
	}
	return out, nil
}
