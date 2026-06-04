package azure

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveContextFromAzurePipelineEnv(t *testing.T) {
	t.Setenv("SYSTEM_TEAMPROJECT", "DiffPal")
	t.Setenv("BUILD_REPOSITORY_NAME", "diffpal")
	t.Setenv("BUILD_REPOSITORY_ID", "repo-1")
	t.Setenv("SYSTEM_PULLREQUEST_PULLREQUESTID", "44")
	t.Setenv("BUILD_SOURCEVERSION", "head-env")
	t.Setenv("SYSTEM_PULLREQUEST_TARGETCOMMITID", "base-env")
	t.Setenv("SYSTEM_ACCESSTOKEN", "system-token")
	t.Setenv("SYSTEM_PULLREQUEST_EVENT_PAYLOAD", "")

	ctx, err := ResolveContext("", "")
	if err != nil {
		t.Fatalf("ResolveContext() error = %v", err)
	}
	if ctx.PullRequestID != "44" {
		t.Fatalf("PullRequestID = %q, want 44", ctx.PullRequestID)
	}
	if ctx.HeadSHA != "head-env" || ctx.BaseSHA != "base-env" {
		t.Fatalf("unexpected SHAs: base=%q head=%q", ctx.BaseSHA, ctx.HeadSHA)
	}
	if ctx.RepoID != "repo-1" || ctx.RepositoryID != "repo-1" {
		t.Fatalf("unexpected repo ids: repo=%q repository=%q", ctx.RepoID, ctx.RepositoryID)
	}
	if ctx.TokenSource != "system_access_token" {
		t.Fatalf("TokenSource = %q, want system_access_token", ctx.TokenSource)
	}
	if !ctx.UsesSystemToken {
		t.Fatal("UsesSystemToken = false, want true")
	}
}

func TestResolveContextFromAzureEventPayload(t *testing.T) {
	t.Setenv("SYSTEM_PULLREQUEST_PULLREQUESTID", "")
	t.Setenv("SYSTEM_PULLREQUEST_SOURCECOMMITID", "")
	t.Setenv("SYSTEM_PULLREQUEST_TARGETCOMMITID", "")
	t.Setenv("BUILD_SOURCEVERSION", "")
	t.Setenv("AZURE_DEVOPS_EXT_PAT", "pat-token")
	eventPath := writeAzureEvent(t, `{
  "resource": {
    "pullRequestId": "72",
    "sourceRefName": "refs/heads/feature/a",
    "targetRefName": "refs/heads/main",
    "lastMergeSourceCommit": {"commitId": "head-event"},
    "lastMergeTargetCommit": {"commitId": "base-event"},
    "repository": {
      "name": "diffpal",
      "url": "https://dev.azure.example/org/project/_git/diffpal"
    }
  }
}`)
	t.Setenv("SYSTEM_PULLREQUEST_EVENT_PAYLOAD", eventPath)

	ctx, err := ResolveContext("", "")
	if err != nil {
		t.Fatalf("ResolveContext() error = %v", err)
	}
	if ctx.PullRequestID != "72" {
		t.Fatalf("PullRequestID = %q, want 72", ctx.PullRequestID)
	}
	if ctx.HeadSHA != "head-event" || ctx.BaseSHA != "base-event" {
		t.Fatalf("unexpected SHAs: base=%q head=%q", ctx.BaseSHA, ctx.HeadSHA)
	}
	if ctx.RepoName != "diffpal" {
		t.Fatalf("RepoName = %q, want diffpal", ctx.RepoName)
	}
	if ctx.TokenSource != "pat" {
		t.Fatalf("TokenSource = %q, want pat", ctx.TokenSource)
	}
	if ctx.UsesSystemToken {
		t.Fatal("UsesSystemToken = true, want false for PAT")
	}
}

func writeAzureEvent(t *testing.T, payload string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "azure-event.json")
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
