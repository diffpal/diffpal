package azure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Context struct {
	Provider        string
	CollectionURI   string
	ProjectName     string
	RepoName        string
	RepoID          string
	PullRequestID   string
	RepositoryID    string
	SourceRefName   string
	TargetRefName   string
	SourceCommitSHA string
	TargetCommitSHA string
	HeadSHA         string
	BaseSHA         string
	SourceBranch    string
	TargetBranch    string
	WebURL          string
	BuildID         string
	SupportsThreads bool
	UsesSystemToken bool
	TokenSource     string
}

func ResolveContext(baseArg, headArg string) (Context, error) {
	ctx := Context{
		Provider:        "azure",
		CollectionURI:   os.Getenv("SYSTEM_COLLECTIONURI"),
		ProjectName:     os.Getenv("SYSTEM_TEAMPROJECT"),
		RepoName:        os.Getenv("BUILD_REPOSITORY_NAME"),
		RepoID:          os.Getenv("BUILD_REPOSITORY_ID"),
		PullRequestID:   os.Getenv("SYSTEM_PULLREQUEST_PULLREQUESTID"),
		SourceBranch:    os.Getenv("SYSTEM_PULLREQUEST_SOURCEBRANCH"),
		TargetBranch:    os.Getenv("SYSTEM_PULLREQUEST_TARGETBRANCH"),
		SourceCommitSHA: os.Getenv("SYSTEM_PULLREQUEST_SOURCECOMMITID"),
		TargetCommitSHA: os.Getenv("SYSTEM_PULLREQUEST_TARGETCOMMITID"),
		BuildID:         os.Getenv("BUILD_BUILDID"),
		SupportsThreads: true,
	}
	ctx.TokenSource = detectTokenSource()
	ctx.UsesSystemToken = ctx.TokenSource == "system_access_token"
	if baseArg != "" {
		ctx.BaseSHA = baseArg
	}
	if headArg != "" {
		ctx.HeadSHA = headArg
	}
	if ctx.HeadSHA == "" {
		ctx.HeadSHA = firstNonEmpty(
			"BUILD_SOURCEVERSION",
			"SYSTEM_PULLREQUEST_SOURCECOMMITID",
		)
	}
	if ctx.BaseSHA == "" {
		ctx.BaseSHA = firstNonEmpty(
			"SYSTEM_PULLREQUEST_TARGETCOMMITID",
			"BUILD_SOURCEVERSION",
		)
	}
	ctx.RepositoryID = ctx.RepoID
	ctx.WebURL = firstNonEmpty(
		"BUILD_REPOSITORY_URI",
		"SYSTEM_COLLECTIONURI",
	)
	if ctx.WebURL == "" {
		ctx.WebURL = ctx.CollectionURI
	}

	eventPath := os.Getenv("SYSTEM_PULLREQUEST_EVENT_PAYLOAD")
	if eventPath != "" {
		payload, err := os.ReadFile(filepath.Clean(eventPath))
		if err != nil {
			return Context{}, err
		}
		var doc map[string]any
		if err := json.Unmarshal(payload, &doc); err != nil {
			return Context{}, err
		}
		if raw, ok := doc["resource"].(map[string]any); ok {
			if ctx.PullRequestID == "" {
				ctx.PullRequestID = digString(raw, "pullRequestId")
			}
			if ctx.SourceBranch == "" {
				ctx.SourceBranch = digString(raw, "sourceRefName")
			}
			if ctx.TargetBranch == "" {
				ctx.TargetBranch = digString(raw, "targetRefName")
			}
			if ctx.SourceCommitSHA == "" {
				ctx.SourceCommitSHA = digString(raw, "lastMergeSourceCommit", "commitId")
			}
			if ctx.TargetCommitSHA == "" {
				ctx.TargetCommitSHA = digString(raw, "lastMergeTargetCommit", "commitId")
			}
			if ctx.RepoName == "" {
				ctx.RepoName = digString(raw, "repository", "name")
			}
			if ctx.WebURL == "" {
				ctx.WebURL = digString(raw, "repository", "url")
			}
			if ctx.BaseSHA == "" {
				ctx.BaseSHA = ctx.TargetCommitSHA
			}
			if ctx.HeadSHA == "" {
				ctx.HeadSHA = ctx.SourceCommitSHA
			}
		}
	}

	if ctx.PullRequestID == "" {
		return ctx, fmt.Errorf("missing pull request id")
	}
	if ctx.HeadSHA == "" {
		return ctx, fmt.Errorf("missing head sha")
	}
	if ctx.BaseSHA == "" {
		return ctx, fmt.Errorf("missing base sha")
	}
	return ctx, nil
}

func detectTokenSource() string {
	switch {
	case os.Getenv("SYSTEM_ACCESSTOKEN") != "":
		return "system_access_token"
	case os.Getenv("AZURE_DEVOPS_EXT_PAT") != "":
		return "pat"
	default:
		return "none"
	}
}

func firstNonEmpty(keys ...string) string {
	for _, key := range keys {
		if val := os.Getenv(key); val != "" {
			return val
		}
	}
	return ""
}

func digString(doc map[string]any, path ...string) string {
	current := any(doc)
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
	if s, ok := current.(string); ok {
		return s
	}
	return ""
}
