package azure

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/diffpal/diffpal/internal/findings"
)

const MinThreadConfidence = 0.8

type ThreadActionType string

const (
	ActionCreate ThreadActionType = "create"
	ActionUpdate ThreadActionType = "update"
	ActionSkip   ThreadActionType = "skip"
)

type ThreadAction struct {
	Type      ThreadActionType
	FindingID string
	Path      string
	Line      int
	Body      string
	ThreadID  string
}

type ThreadState struct {
	ThreadID  string `json:"thread_id"`
	FindingID string `json:"finding_id"`
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
	out := make([]ThreadAction, 0, len(findingsList))
	state := make([]ThreadState, 0, len(findingsList))
	for _, f := range findingsList {
		if f.Path == "" || f.StartLine <= 0 || f.RuleID == "" || f.Confidence < MinThreadConfidence {
			continue
		}
		key := threadKey(f.Path, f.StartLine, f.RuleID)
		action := ActionCreate
		if prior, ok := existing[key]; ok {
			if prior == f.ID {
				action = ActionSkip
			} else {
				action = ActionUpdate
			}
		}
		state = append(state, ThreadState{ThreadID: key, FindingID: f.ID})
		out = append(out, ThreadAction{
			Type:      action,
			FindingID: f.ID,
			Path:      f.Path,
			Line:      f.StartLine,
			Body:      threadBody(f),
			ThreadID:  key,
		})
	}
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

func threadKey(path string, line int, ruleID string) string {
	return fmt.Sprintf("%s:%d:%s", path, line, ruleID)
}

func threadBody(f findings.Finding) string {
	return "### " + f.RuleID + "\n\n" +
		"Category: **" + f.Category + "**\n\n" +
		"Severity: **" + f.Severity + "**\n\n" +
		f.Message + "\n\n" +
		"Evidence: " + f.Evidence + "\n\n" +
		"Confidence: " + formatConfidence(f.Confidence)
}

func formatConfidence(v float64) string {
	if v <= 0 {
		return "0.00"
	}
	return fmt.Sprintf("%.2f", v)
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
	FatalOnFailures bool
}

type StatusPayload struct {
	Name        string      `json:"name"`
	State       StatusState `json:"state"`
	Description string      `json:"description"`
	Context     string      `json:"context"`
}

func EvaluateStatus(bundleCount int, blockingCount int, failed bool) StatusPayload {
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
	return EvaluateStatus(blockingCount+advisoryCount, blockingCount, toolError)
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
		out[item.ThreadID] = item.FindingID
	}
	return out, nil
}
