package azure

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/diffpal/diffpal/internal/findings"
	"github.com/diffpal/diffpal/internal/markdown"
)

const MinThreadConfidence = 0.8
const MinExpandedThreadConfidence = 0.65

type ThreadActionType string
type ThreadStatus string

const (
	ActionCreate ThreadActionType = "create"
	ActionUpdate ThreadActionType = "update"
	ActionSkip   ThreadActionType = "skip"

	ThreadStatusActive ThreadStatus = "active"
	ThreadStatusClosed ThreadStatus = "closed"
)

type ThreadAction struct {
	Type      ThreadActionType `json:"type"`
	Status    ThreadStatus     `json:"status"`
	FindingID string           `json:"finding_id"`
	Path      string           `json:"path"`
	Line      int              `json:"line"`
	Body      string           `json:"body"`
	ThreadID  string           `json:"thread_id"`
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
	return PlanThreadsWithProfile(existing, findingsList, ctx, "")
}

func PlanThreadsWithProfile(existing map[string]string, findingsList []findings.Finding, ctx Context, profile string) ThreadPlan {
	return planThreads(existing, findingsList, ctx, threadConfidenceThreshold(profile))
}

func planThreads(existing map[string]string, findingsList []findings.Finding, ctx Context, minConfidence float64) ThreadPlan {
	out := make([]ThreadAction, 0, len(findingsList))
	state := make([]ThreadState, 0, len(findingsList))
	for _, f := range findingsList {
		if !f.Blocking || f.Path == "" || f.StartLine <= 0 || f.Category == "" || f.Confidence < minConfidence {
			continue
		}
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
			if prior == f.ID {
				action = ActionSkip
			} else {
				action = ActionUpdate
			}
		}
		state = append(state, ThreadState{ThreadID: actionThreadID, FindingID: f.ID})
		out = append(out, ThreadAction{
			Type:      action,
			Status:    ThreadStatusActive,
			FindingID: f.ID,
			Path:      f.Path,
			Line:      f.StartLine,
			Body:      threadBody(f),
			ThreadID:  actionThreadID,
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

func threadConfidenceThreshold(profile string) float64 {
	if profile == "inline" {
		return MinExpandedThreadConfidence
	}
	return MinThreadConfidence
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

func threadBody(f findings.Finding) string {
	return markdown.RenderFindingDetail(f, markdown.FindingDetailOptions{
		HideConfidence: true,
	})
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
		out[item.ThreadID] = item.FindingID
	}
	return out, nil
}
