package gitlab

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Context struct {
	Provider           string
	ProjectID          string
	ProjectPath        string
	MergeRequestIID    string
	MergeRequestID     string
	Repo               string
	SourceBranch       string
	TargetBranch       string
	BaseSHA            string
	HeadSHA            string
	SourceCommitSHA    string
	TargetCommitSHA    string
	WebURL             string
	CanApprove         bool
	ApproverIdentityID string
	TokenMode          string
}

func ResolveContext(baseArg, headArg, projectArg, mrArg string) (Context, error) {
	ctx := Context{
		Provider: "gitlab",
	}
	if baseArg != "" {
		ctx.BaseSHA = baseArg
	}
	if headArg != "" {
		ctx.HeadSHA = headArg
	}
	if projectArg != "" {
		ctx.Repo = projectArg
	}
	if mrArg != "" {
		ctx.MergeRequestIID = mrArg
	}

	if ctx.BaseSHA == "" {
		ctx.BaseSHA = os.Getenv("CI_MERGE_REQUEST_DIFF_BASE_SHA")
	}
	if ctx.HeadSHA == "" {
		ctx.HeadSHA = firstEnv(
			"CI_MERGE_REQUEST_SOURCE_BRANCH_SHA",
			"CI_COMMIT_SHA",
			"CI_COMMIT_SHORT_SHA",
		)
	}
	if ctx.MergeRequestIID == "" {
		ctx.MergeRequestIID = os.Getenv("CI_MERGE_REQUEST_IID")
	}
	if ctx.Repo == "" {
		ctx.Repo = os.Getenv("CI_PROJECT_PATH")
	}
	ctx.ProjectPath = ctx.Repo
	ctx.TokenMode = detectTokenMode()
	ctx.CanApprove = ctx.TokenMode == "gitlab_token"
	ctx.ApproverIdentityID = os.Getenv("DIFFPAL_GITLAB_APPROVER_ID")

	eventPath := firstEnv("GITLAB_EVENT_PATH", "CI_MERGE_REQUEST_EVENT_PATH")
	if eventPath != "" {
		payload, err := os.ReadFile(filepath.Clean(eventPath))
		if err != nil {
			return Context{}, err
		}
		var doc map[string]any
		if err := json.Unmarshal(payload, &doc); err != nil {
			return Context{}, err
		}
		projectNode := doc
		if projectMap, ok := doc["project"].(map[string]any); ok {
			projectNode = projectMap
		}
		if ctx.Repo == "" {
			ctx.Repo = digString(projectNode, "path_with_namespace")
		}
		ctx.ProjectID = digString(projectNode, "id")
		if ctx.ProjectID == "" {
			ctx.ProjectID = digString(projectNode, "path")
		}
		ctx.WebURL = digString(projectNode, "web_url")
		objAttr := doc
		if mr, ok := doc["object_attributes"].(map[string]any); ok {
			objAttr = mr
		}
		if ctx.MergeRequestIID == "" {
			ctx.MergeRequestIID = digString(objAttr, "iid")
		}
		if ctx.MergeRequestID == "" {
			ctx.MergeRequestID = digString(objAttr, "id")
		}
		ctx.SourceBranch = digString(objAttr, "source_branch")
		ctx.TargetBranch = digString(objAttr, "target_branch")
		ctx.SourceCommitSHA = digString(objAttr, "last_commit", "id")
		ctx.TargetCommitSHA = digString(objAttr, "target_commit", "id")
		if ctx.BaseSHA == "" {
			ctx.BaseSHA = digString(objAttr, "oldrev")
		}
		if ctx.HeadSHA == "" {
			ctx.HeadSHA = digString(objAttr, "last_commit", "id")
		}
	}

	if ctx.BaseSHA == "" {
		return ctx, fmt.Errorf("missing base sha")
	}
	if ctx.HeadSHA == "" {
		return ctx, fmt.Errorf("missing head sha")
	}
	if ctx.Repo == "" {
		return ctx, fmt.Errorf("missing project/repository")
	}
	return ctx, nil
}

func detectTokenMode() string {
	switch {
	case os.Getenv("GITLAB_TOKEN") != "":
		return "gitlab_token"
	case os.Getenv("CI_JOB_TOKEN") != "":
		return "ci_job_token"
	default:
		return "none"
	}
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if val := os.Getenv(key); val != "" {
			return val
		}
	}
	return ""
}

func digString(doc map[string]any, path ...string) string {
	var current any = doc
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		next, ok := m[key]
		if !ok {
			return ""
		}
		current = next
	}
	switch v := current.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatInt(int64(v), 10)
	default:
		return ""
	}
}
