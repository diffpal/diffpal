package gitlab

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/diffpal/diffpal/internal/findings"
	"github.com/diffpal/diffpal/internal/markdown"
)

type DiscussionActionType string

const (
	ActionCreate DiscussionActionType = "create"
	ActionUpdate DiscussionActionType = "update"
	ActionSkip   DiscussionActionType = "skip"
)

type DiscussionAction struct {
	Type       DiscussionActionType `json:"type"`
	FindingID  string               `json:"finding_id"`
	Body       string               `json:"body"`
	Path       string               `json:"path"`
	Line       int                  `json:"line"`
	EndLine    int                  `json:"end_line"`
	Blocking   bool                 `json:"blocking"`
	Resolved   bool                 `json:"resolved"`
	ThreadHash string               `json:"thread_hash"`
}

type DiscussionState struct {
	ThreadHash string `json:"thread_hash"`
	FindingID  string `json:"finding_id"`
}

type DiscussionPlan struct {
	Actions []DiscussionAction `json:"actions"`
	State   []DiscussionState  `json:"state"`
}

func PlanDiscussions(existing map[string]string, findingsList []findings.Finding, blockOn []string) DiscussionPlan {
	out := make([]DiscussionAction, 0, len(findingsList))
	state := make([]DiscussionState, 0, len(findingsList))
	for _, finding := range findingsList {
		if finding.Path == "" || finding.StartLine <= 0 || finding.Category == "" {
			continue
		}
		blocking := finding.Blocking || isLevelOrAbove(finding.Severity, blockOn)
		thread := discussionKey(finding.Path, finding.StartLine, finding.Category, finding.ID)
		actionThread := thread
		actionType := ActionCreate
		prior, ok := existing[thread]
		if !ok {
			var priorThread string
			priorThread, prior, ok = singleExistingForLocation(existing, discussionLocationKey(finding.Path, finding.StartLine, finding.Category))
			if ok {
				actionThread = priorThread
			}
		}
		if ok {
			if prior == finding.ID {
				actionType = ActionSkip
			} else {
				actionType = ActionUpdate
			}
		}
		state = append(state, DiscussionState{ThreadHash: actionThread, FindingID: finding.ID})
		out = append(out, DiscussionAction{
			Type:       actionType,
			FindingID:  finding.ID,
			Body:       discussionBody(finding),
			Path:       finding.Path,
			Line:       finding.StartLine,
			EndLine:    max(finding.EndLine, finding.StartLine),
			Blocking:   blocking,
			Resolved:   !blocking,
			ThreadHash: actionThread,
		})
	}
	return DiscussionPlan{
		Actions: out,
		State:   state,
	}
}

func discussionKey(path string, line int, category string, findingID string) string {
	return discussionLocationKey(path, line, category) + ":" + findingID
}

func discussionLocationKey(path string, line int, category string) string {
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

func discussionBody(f findings.Finding) string {
	return markdown.RenderFindingDetail(f, markdown.FindingDetailOptions{
		HideConfidence: true,
	})
}

func isLevelOrAbove(level string, blockOn []string) bool {
	severityRank := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}
	current, ok := severityRank[strings.ToLower(level)]
	if !ok {
		return false
	}
	for _, candidate := range blockOn {
		target, ok := severityRank[strings.ToLower(candidate)]
		if !ok {
			continue
		}
		if current >= target {
			return true
		}
	}
	return false
}

type MergeDecision string

const (
	MergeDecisionPass MergeDecision = "pass"
	MergeDecisionWarn MergeDecision = "warn"
	MergeDecisionFail MergeDecision = "fail"
)

type PublishResult struct {
	Decision      MergeDecision
	BlockCount    int
	AdvisoryCount int
	Blocking      []findings.Finding
	Advisory      []findings.Finding
}

type StatusPayload struct {
	State       string `json:"state"`
	Name        string `json:"name"`
	Context     string `json:"context"`
	Description string `json:"description"`
	TargetURL   string `json:"target_url,omitempty"`
}

func PolicyStatus(blockingCount int, advisoryCount int, gateEnabled bool, targetURL string) StatusPayload {
	payload := StatusPayload{
		State:   "success",
		Name:    "DiffPal Review",
		Context: "diffpal/review",
	}
	switch {
	case blockingCount > 0 && gateEnabled:
		payload.State = "failed"
		payload.Description = fmt.Sprintf("%d blocking findings", blockingCount)
	case blockingCount > 0:
		payload.Description = fmt.Sprintf("%d blocking findings; gate disabled", blockingCount)
	case advisoryCount > 0:
		payload.Description = "Advisory findings present without merge blockers"
	default:
		payload.Description = "DiffPal completed with no findings"
	}
	payload.TargetURL = strings.TrimSpace(targetURL)
	return payload
}

func SummarizeDecision(bundle findings.FindingsBundle, blockOn []string) PublishResult {
	out := PublishResult{}
	for _, f := range bundle.Findings {
		if isLevelOrAbove(f.Severity, blockOn) || f.Blocking {
			out.BlockCount++
			out.Blocking = append(out.Blocking, f)
			continue
		}
		out.AdvisoryCount++
		out.Advisory = append(out.Advisory, f)
	}
	switch {
	case out.BlockCount > 0:
		out.Decision = MergeDecisionFail
	case out.AdvisoryCount > 0:
		out.Decision = MergeDecisionWarn
	default:
		out.Decision = MergeDecisionPass
	}
	return out
}

type ApprovalPolicy struct {
	Enabled       bool
	RequireSHA    string
	ApproverID    string
	ApproveOnPass bool
}

func CanAutoApprove(cfg ApprovalPolicy, bundle findings.FindingsBundle, headSHA string) bool {
	if !cfg.Enabled || cfg.ApproverID == "" {
		return false
	}
	if cfg.RequireSHA != "" && cfg.RequireSHA != headSHA {
		return false
	}
	if len(bundle.Findings) > 0 {
		return false
	}
	if cfg.ApproveOnPass {
		return true
	}
	return false
}

func LoadExistingState(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var plan DiscussionPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		return nil, err
	}
	if len(plan.State) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(plan.State))
	for _, item := range plan.State {
		out[item.ThreadHash] = item.FindingID
	}
	return out, nil
}
