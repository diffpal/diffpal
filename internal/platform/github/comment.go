package github

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/diffpal/diffpal/internal/findings"
	"github.com/diffpal/diffpal/internal/markdown"
)

const MinInlineConfidence = 0.8
const MinExpandedInlineConfidence = 0.65

type CommentActionType string

const (
	ActionCreate CommentActionType = "create"
	ActionUpdate CommentActionType = "update"
	ActionSkip   CommentActionType = "skip"
)

type CommentAction struct {
	Type      CommentActionType
	FindingID string
	Body      string
	Path      string
	Line      int
	EndLine   int
}

type CommentState struct {
	Key       string `json:"key"`
	FindingID string `json:"finding_id"`
}

type CommentPlan struct {
	Actions []CommentAction `json:"actions"`
	State   []CommentState  `json:"state"`
}

func PlanInlineComments(existing map[string]string, findings []findings.Finding) CommentPlan {
	return PlanInlineCommentsWithProfile(existing, findings, "")
}

func PlanInlineCommentsWithProfile(existing map[string]string, findings []findings.Finding, profile string) CommentPlan {
	return PlanInlineCommentsWithOptions(existing, findings, CommentOptions{Profile: profile})
}

type CommentOptions struct {
	Profile     string
	Links       markdown.FindingLinkProvider
	AllFindings bool
}

func PlanInlineCommentsWithOptions(existing map[string]string, findings []findings.Finding, opts CommentOptions) CommentPlan {
	minConfidence := inlineConfidenceThreshold(opts.Profile)
	if opts.AllFindings {
		minConfidence = -1
	}
	return planInlineComments(existing, findings, minConfidence, opts.Links)
}

func planInlineComments(existing map[string]string, findings []findings.Finding, minConfidence float64, links markdown.FindingLinkProvider) CommentPlan {
	out := make([]CommentAction, 0, len(findings))
	state := make([]CommentState, 0, len(findings))
	for _, f := range findings {
		if f.Path == "" {
			continue
		}
		if f.StartLine <= 0 || f.Confidence < minConfidence {
			continue
		}
		key := commentKey(f.Path, f.StartLine, f.Category, f.ID)
		body := formatBody(f, links)
		endLine := f.EndLine
		if endLine < f.StartLine {
			endLine = f.StartLine
		}
		state = append(state, CommentState{Key: key, FindingID: f.ID})
		if existing == nil {
			out = append(out, CommentAction{Type: ActionCreate, FindingID: f.ID, Body: body, Path: f.Path, Line: f.StartLine, EndLine: endLine})
			continue
		}
		prior, ok := existing[key]
		if !ok {
			prior, ok = singleExistingForLocation(existing, commentLocationKey(f.Path, f.StartLine, f.Category))
		}
		if ok && prior == f.ID {
			out = append(out, CommentAction{Type: ActionSkip, FindingID: f.ID, Body: body, Path: f.Path, Line: f.StartLine, EndLine: endLine})
			continue
		}
		if ok {
			out = append(out, CommentAction{Type: ActionUpdate, FindingID: f.ID, Body: body, Path: f.Path, Line: f.StartLine, EndLine: endLine})
			continue
		}
		out = append(out, CommentAction{Type: ActionCreate, FindingID: f.ID, Body: body, Path: f.Path, Line: f.StartLine, EndLine: endLine})
	}
	return CommentPlan{
		Actions: out,
		State:   state,
	}
}

func inlineConfidenceThreshold(profile string) float64 {
	if profile == "inline" {
		return MinExpandedInlineConfidence
	}
	return MinInlineConfidence
}

func ValidateInlineFindings(items []findings.Finding) error {
	for _, item := range items {
		if strings.TrimSpace(item.Path) == "" {
			return fmt.Errorf("github finding %q cannot be published inline: missing path", item.ID)
		}
		if item.StartLine <= 0 {
			return fmt.Errorf("github finding %q cannot be published inline: missing start line", item.ID)
		}
	}
	return nil
}

func LoadExistingState(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var plan CommentPlan
	if err := json.Unmarshal(raw, &plan); err != nil {
		return nil, err
	}
	if len(plan.State) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(plan.State))
	for _, item := range plan.State {
		out[item.Key] = item.FindingID
	}
	return out, nil
}

func commentKey(path string, line int, category string, findingID string) string {
	return commentLocationKey(path, line, category) + ":" + findingID
}

func commentLocationKey(path string, line int, category string) string {
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

func formatBody(f findings.Finding, links markdown.FindingLinkProvider) string {
	return markdown.RenderFindingDetail(f, markdown.FindingDetailOptions{
		Link: linkForFinding(links, f),
	})
}

func linkForFinding(provider markdown.FindingLinkProvider, finding findings.Finding) string {
	if provider == nil {
		return ""
	}
	link, ok := provider.Link(finding)
	if !ok {
		return ""
	}
	return link
}

func findingMarker(identity ReviewIdentity, findingID string) string {
	id := strings.NewReplacer("--", "-", "\n", " ", "\r", " ").Replace(strings.TrimSpace(findingID))
	if id == "" {
		return ""
	}
	return "<!-- diffpal:finding:" + identity.channel() + " id:" + id + " -->"
}
