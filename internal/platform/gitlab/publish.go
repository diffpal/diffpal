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
	Type       DiscussionActionType
	FindingID  string
	Body       string
	Path       string
	Line       int
	Blocking   bool
	ThreadHash string
}

type DiscussionState struct {
	ThreadHash string `json:"thread_hash"`
	FindingID  string `json:"finding_id"`
}

type DiscussionPlan struct {
	Actions         []DiscussionAction `json:"actions"`
	State           []DiscussionState  `json:"state"`
	AdvisorySummary string             `json:"advisory_summary,omitempty"`
}

func PlanDiscussions(existing map[string]string, findingsList []findings.Finding, blockOn []string) DiscussionPlan {
	out := make([]DiscussionAction, 0, len(findingsList))
	state := make([]DiscussionState, 0, len(findingsList))
	advisory := make([]findings.Finding, 0, len(findingsList))
	for _, finding := range findingsList {
		if finding.Path == "" || finding.StartLine <= 0 || finding.Category == "" {
			continue
		}
		blocking := finding.Blocking || isLevelOrAbove(finding.Severity, blockOn)
		if !blocking {
			advisory = append(advisory, finding)
			continue
		}
		thread := discussionKey(finding.Path, finding.StartLine, finding.Category, finding.ID)
		actionType := ActionCreate
		prior, ok := existing[thread]
		if !ok {
			prior, ok = singleExistingForLocation(existing, discussionLocationKey(finding.Path, finding.StartLine, finding.Category))
		}
		if ok {
			if prior == finding.ID {
				actionType = ActionSkip
			} else {
				actionType = ActionUpdate
			}
		}
		state = append(state, DiscussionState{ThreadHash: thread, FindingID: finding.ID})
		out = append(out, DiscussionAction{
			Type:       actionType,
			FindingID:  finding.ID,
			Body:       discussionBody(finding),
			Path:       finding.Path,
			Line:       finding.StartLine,
			Blocking:   blocking,
			ThreadHash: thread,
		})
	}
	var advisorySummary string
	if len(advisory) > 0 {
		advisorySummary = markdown.RenderSummary(findings.FindingsBundle{Findings: advisory})
	}
	return DiscussionPlan{
		Actions:         out,
		State:           state,
		AdvisorySummary: advisorySummary,
	}
}

func discussionKey(path string, line int, category string, findingID string) string {
	return discussionLocationKey(path, line, category) + ":" + findingID
}

func discussionLocationKey(path string, line int, category string) string {
	return fmt.Sprintf("%s:%d:%s", path, line, category)
}

func singleExistingForLocation(existing map[string]string, locationKey string) (string, bool) {
	var prior string
	found := false
	prefix := locationKey + ":"
	for key, findingID := range existing {
		if key != locationKey && !strings.HasPrefix(key, prefix) {
			continue
		}
		if found {
			return "", false
		}
		prior = findingID
		found = true
	}
	return prior, found
}

func discussionBody(f findings.Finding) string {
	lines := []string{
		fmt.Sprintf("**%s %s**", strings.ToUpper(f.Severity), f.Category),
		"",
		f.Message,
		"",
		"**Confidence**: " + formatConfidence(f.Confidence),
		"**Provider**: " + f.Provider,
	}
	if f.Evidence != "" {
		fence := markdownFence(f.Evidence)
		lines = append(lines, "", "**Evidence:**", fence, f.Evidence, fence)
	}
	if f.Suggestion != "" {
		fence := markdownFence(f.Suggestion)
		lines = append(lines, "", "**Suggestion:**", fence, f.Suggestion, fence)
	}
	return strings.Join(lines, "\n")
}

func markdownFence(content string) string {
	maxRun := 0
	current := 0
	for _, r := range content {
		if r == '`' {
			current++
			if current > maxRun {
				maxRun = current
			}
			continue
		}
		current = 0
	}
	if maxRun < 3 {
		return "```"
	}
	return strings.Repeat("`", maxRun+1)
}

func formatConfidence(v float64) string {
	if v <= 0 {
		return "unset"
	}
	return fmt.Sprintf("%.2f", v)
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
