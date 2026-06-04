package github

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveContextFromPullRequestEvent(t *testing.T) {
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_REPOSITORY", "acme/diffpal")
	t.Setenv("GITHUB_TOKEN", "")

	eventPath := writeEventFile(t, `{
  "number": 42,
  "repository": {"full_name": "acme/diffpal"},
  "pull_request": {
    "number": 42,
    "merge_commit_sha": "merge-123",
    "base": {
      "sha": "base-123",
      "repo": {"full_name": "acme/diffpal"}
    },
    "head": {
      "sha": "head-123",
      "repo": {"full_name": "contrib/diffpal"}
    }
  }
}`)
	t.Setenv("GITHUB_EVENT_PATH", eventPath)

	ctx, err := ResolveContext("", "")
	if err != nil {
		t.Fatalf("ResolveContext() error = %v", err)
	}

	if ctx.Repo != "acme/diffpal" {
		t.Fatalf("Repo = %q, want acme/diffpal", ctx.Repo)
	}
	if ctx.BaseRepo != "acme/diffpal" {
		t.Fatalf("BaseRepo = %q, want acme/diffpal", ctx.BaseRepo)
	}
	if ctx.HeadRepo != "contrib/diffpal" {
		t.Fatalf("HeadRepo = %q, want contrib/diffpal", ctx.HeadRepo)
	}
	if ctx.PRNumber != 42 {
		t.Fatalf("PRNumber = %d, want 42", ctx.PRNumber)
	}
	if ctx.BaseSHA != "base-123" || ctx.HeadSHA != "head-123" {
		t.Fatalf("unexpected SHAs: base=%q head=%q", ctx.BaseSHA, ctx.HeadSHA)
	}
	if !ctx.IsFork {
		t.Fatal("IsFork = false, want true")
	}
	if ctx.ForkSafetyMode != ForkSafetyReadOnly {
		t.Fatalf("ForkSafetyMode = %q, want %q", ctx.ForkSafetyMode, ForkSafetyReadOnly)
	}
}

func TestResolveContextFromPullRequestTargetAllowsWrite(t *testing.T) {
	t.Setenv("GITHUB_EVENT_NAME", "pull_request_target")
	t.Setenv("GITHUB_REPOSITORY", "acme/diffpal")
	t.Setenv("GITHUB_TOKEN", "token")

	eventPath := writeEventFile(t, `{
  "number": 7,
  "repository": {"full_name": "acme/diffpal"},
  "pull_request": {
    "base": {
      "sha": "base-777",
      "repo": {"full_name": "acme/diffpal"}
    },
    "head": {
      "sha": "head-777",
      "repo": {"full_name": "contrib/diffpal"}
    }
  }
}`)
	t.Setenv("GITHUB_EVENT_PATH", eventPath)

	ctx, err := ResolveContext("", "")
	if err != nil {
		t.Fatalf("ResolveContext() error = %v", err)
	}
	if ctx.ForkSafetyMode != ForkSafetyWrite {
		t.Fatalf("ForkSafetyMode = %q, want %q", ctx.ForkSafetyMode, ForkSafetyWrite)
	}
}

func TestResolveContextFromArgsAndEnvWithoutEvent(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "acme/diffpal")
	t.Setenv("GITHUB_EVENT_PATH", "")
	t.Setenv("GITHUB_BASE_SHA", "env-base")
	t.Setenv("GITHUB_HEAD_SHA", "env-head")

	ctx, err := ResolveContext("", "")
	if err != nil {
		t.Fatalf("ResolveContext() error = %v", err)
	}
	if ctx.BaseSHA != "env-base" || ctx.HeadSHA != "env-head" {
		t.Fatalf("unexpected SHAs: base=%q head=%q", ctx.BaseSHA, ctx.HeadSHA)
	}
	if ctx.Repo != "acme/diffpal" {
		t.Fatalf("Repo = %q, want acme/diffpal", ctx.Repo)
	}
	if ctx.ForkSafetyMode != ForkSafetyWrite {
		t.Fatalf("ForkSafetyMode = %q, want %q", ctx.ForkSafetyMode, ForkSafetyWrite)
	}
}

func writeEventFile(t *testing.T, payload string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "event.json")
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
