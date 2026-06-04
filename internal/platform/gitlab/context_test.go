package gitlab

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveContextFromGitLabCIEnv(t *testing.T) {
	t.Setenv("CI_MERGE_REQUEST_DIFF_BASE_SHA", "base-env")
	t.Setenv("CI_COMMIT_SHA", "head-env")
	t.Setenv("CI_MERGE_REQUEST_IID", "17")
	t.Setenv("CI_PROJECT_PATH", "acme/diffpal")
	t.Setenv("CI_JOB_TOKEN", "job-token")
	t.Setenv("GITLAB_EVENT_PATH", "")
	t.Setenv("CI_MERGE_REQUEST_EVENT_PATH", "")

	ctx, err := ResolveContext("", "", "", "")
	if err != nil {
		t.Fatalf("ResolveContext() error = %v", err)
	}
	if ctx.BaseSHA != "base-env" || ctx.HeadSHA != "head-env" {
		t.Fatalf("unexpected SHAs: base=%q head=%q", ctx.BaseSHA, ctx.HeadSHA)
	}
	if ctx.MergeRequestIID != "17" {
		t.Fatalf("MergeRequestIID = %q, want 17", ctx.MergeRequestIID)
	}
	if ctx.Repo != "acme/diffpal" {
		t.Fatalf("Repo = %q, want acme/diffpal", ctx.Repo)
	}
	if ctx.TokenMode != "ci_job_token" {
		t.Fatalf("TokenMode = %q, want ci_job_token", ctx.TokenMode)
	}
	if ctx.CanApprove {
		t.Fatal("CanApprove = true, want false for CI_JOB_TOKEN")
	}
}

func TestResolveContextFromGitLabEventPayload(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "token")
	eventPath := writeGitLabEvent(t, `{
  "project": {
    "id": 15,
    "path_with_namespace": "acme/diffpal",
    "web_url": "https://gitlab.example/acme/diffpal"
  },
  "object_attributes": {
    "id": 99,
    "iid": 23,
    "source_branch": "feature/x",
    "target_branch": "main",
    "oldrev": "base-event",
    "last_commit": {"id": "head-event"}
  }
}`)
	t.Setenv("GITLAB_EVENT_PATH", eventPath)

	ctx, err := ResolveContext("", "", "", "")
	if err != nil {
		t.Fatalf("ResolveContext() error = %v", err)
	}
	if ctx.ProjectID != "15" {
		t.Fatalf("ProjectID = %q, want 15", ctx.ProjectID)
	}
	if ctx.MergeRequestID != "99" || ctx.MergeRequestIID != "23" {
		t.Fatalf("unexpected MR ids: id=%q iid=%q", ctx.MergeRequestID, ctx.MergeRequestIID)
	}
	if ctx.BaseSHA != "base-event" || ctx.HeadSHA != "head-event" {
		t.Fatalf("unexpected SHAs: base=%q head=%q", ctx.BaseSHA, ctx.HeadSHA)
	}
	if ctx.TokenMode != "gitlab_token" {
		t.Fatalf("TokenMode = %q, want gitlab_token", ctx.TokenMode)
	}
	if !ctx.CanApprove {
		t.Fatal("CanApprove = false, want true for GITLAB_TOKEN")
	}
}

func writeGitLabEvent(t *testing.T, payload string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gitlab-event.json")
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
