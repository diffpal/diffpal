package github

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type ForkSafetyMode string

const (
	ForkSafetyWrite    ForkSafetyMode = "write"
	ForkSafetyReadOnly ForkSafetyMode = "read_only"
)

type Context struct {
	Provider       string
	BaseSHA        string
	HeadSHA        string
	PRNumber       int
	Repo           string
	BaseRepo       string
	HeadRepo       string
	IsFork         bool
	MergeCommit    string
	EventName      string
	ForkSafetyMode ForkSafetyMode
}

func ResolveContext(baseArg, headArg string) (Context, error) {
	ctx := Context{
		Provider:  "github",
		EventName: os.Getenv("GITHUB_EVENT_NAME"),
		Repo:      os.Getenv("GITHUB_REPOSITORY"),
	}
	if baseArg != "" {
		ctx.BaseSHA = baseArg
	}
	if headArg != "" {
		ctx.HeadSHA = headArg
	}
	if ctx.BaseSHA == "" && os.Getenv("GITHUB_BASE_SHA") != "" {
		ctx.BaseSHA = os.Getenv("GITHUB_BASE_SHA")
	}
	if ctx.HeadSHA == "" && os.Getenv("GITHUB_HEAD_SHA") != "" {
		ctx.HeadSHA = os.Getenv("GITHUB_HEAD_SHA")
	}

	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		finalizeForkSafety(&ctx)
		return validateProvided(ctx)
	}
	payload, err := os.ReadFile(filepath.Clean(eventPath))
	if err != nil {
		return Context{}, err
	}
	var doc map[string]any
	if err := json.Unmarshal(payload, &doc); err != nil {
		return Context{}, err
	}
	if ctx.BaseSHA == "" || ctx.HeadSHA == "" {
		if sha := digString(doc, "pull_request", "base", "sha"); sha != "" {
			ctx.BaseSHA = sha
		}
		if sha := digString(doc, "pull_request", "head", "sha"); sha != "" {
			ctx.HeadSHA = sha
		}
		if ctx.MergeCommit == "" {
			ctx.MergeCommit = digString(doc, "pull_request", "merge_commit_sha")
		}
	}
	if ctx.BaseRepo == "" {
		ctx.BaseRepo = digString(doc, "pull_request", "base", "repo", "full_name")
	}
	if ctx.HeadRepo == "" {
		ctx.HeadRepo = digString(doc, "pull_request", "head", "repo", "full_name")
	}
	if ctx.Repo == "" {
		ctx.Repo = digString(doc, "repository", "full_name")
	}
	if ctx.Repo == "" {
		ctx.Repo = ctx.BaseRepo
	}
	if ctx.BaseRepo == "" {
		ctx.BaseRepo = ctx.Repo
	}
	if ctx.PRNumber == 0 {
		ctx.PRNumber = int(digFloat(doc, "number"))
	}
	if ctx.PRNumber == 0 {
		ctx.PRNumber = int(digFloat(doc, "pull_request", "number"))
	}
	ctx.IsFork = ctx.BaseRepo != "" && ctx.HeadRepo != "" && ctx.BaseRepo != ctx.HeadRepo
	finalizeForkSafety(&ctx)
	return validateProvided(ctx)
}

func validateProvided(ctx Context) (Context, error) {
	if ctx.BaseSHA == "" {
		return ctx, fmt.Errorf("missing base sha")
	}
	if ctx.HeadSHA == "" {
		return ctx, fmt.Errorf("missing head sha")
	}
	if ctx.Repo == "" {
		return ctx, fmt.Errorf("missing repository")
	}
	if ctx.PRNumber < 0 {
		return ctx, fmt.Errorf("invalid pull request number")
	}
	return ctx, nil
}

func finalizeForkSafety(ctx *Context) {
	if ctx.BaseRepo == "" {
		ctx.BaseRepo = ctx.Repo
	}
	switch {
	case !ctx.IsFork:
		ctx.ForkSafetyMode = ForkSafetyWrite
	case ctx.EventName == "pull_request_target" && os.Getenv("GITHUB_TOKEN") != "":
		ctx.ForkSafetyMode = ForkSafetyWrite
	default:
		ctx.ForkSafetyMode = ForkSafetyReadOnly
	}
}

func digString(doc map[string]any, path ...string) string {
	current := any(doc)
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		v, ok := m[key]
		if !ok {
			return ""
		}
		current = v
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

func digFloat(doc map[string]any, path ...string) float64 {
	current := any(doc)
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return 0
		}
		v, ok := m[key]
		if !ok {
			return 0
		}
		current = v
	}
	f, ok := current.(float64)
	if !ok {
		return 0
	}
	return f
}
