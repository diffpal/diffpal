package cmd

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	dpconfig "github.com/diffpal/diffpal/internal/config"
	"github.com/diffpal/diffpal/internal/diff"
	"github.com/diffpal/diffpal/internal/findings"
	"github.com/diffpal/diffpal/internal/reviewer"
)

func TestReviewLocalSubcommandUsesLocalBehavior(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeTestConfig(t, dir)

	cmd := newReviewCommandWithRunner(func(_ context.Context, _ dpconfig.Config, opts reviewer.Options) (reviewer.Result, error) {
		if opts.ReviewID != "local" {
			t.Fatalf("ReviewID = %q, want local default", opts.ReviewID)
		}
		if opts.MaxFiles != 200 {
			t.Fatalf("MaxFiles = %d, want 200 from config", opts.MaxFiles)
		}
		if opts.ContextLines != 20 {
			t.Fatalf("ContextLines = %d, want 20 from config", opts.ContextLines)
		}
		return testReviewResult("local"), nil
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"local", "--out", filepath.Join(dir, "findings.json")})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "findings=1") {
		t.Fatalf("output missing findings count:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "status=blocked blocking=1") {
		t.Fatalf("output missing blocked status:\n%s", out.String())
	}
}

func TestReviewLocalGateExitsBlocked(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeTestConfig(t, dir)

	cmd := newReviewCommandWithRunner(func(_ context.Context, _ dpconfig.Config, _ reviewer.Options) (reviewer.Result, error) {
		return testReviewResult("local"), nil
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"local", "--gate", "--out", filepath.Join(dir, "findings.json")})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want blocked gate error")
	}
	coder, ok := err.(interface{ ExitCode() int })
	if !ok {
		t.Fatalf("error does not expose ExitCode(): %T", err)
	}
	if coder.ExitCode() != 1 {
		t.Fatalf("ExitCode() = %d, want 1", coder.ExitCode())
	}
	if !strings.Contains(err.Error(), "review blocked: blocking findings detected: 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReviewGitHubPublishesSelectedHostArtifacts(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("GITHUB_TOKEN", "token")
	t.Setenv("DIFFPAL_GITLAB_TOKEN_TEST", "unused")
	t.Setenv("DIFFPAL_ADO_PAT_TEST", "unused")
	writeHostTestConfig(t, dir)
	t.Setenv("GITHUB_REPOSITORY", "acme/diffpal")
	t.Setenv("GITHUB_BASE_SHA", "base-a")
	t.Setenv("GITHUB_HEAD_SHA", "head-a")

	var requests atomic.Int32
	handlerErrs := make(chan error, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/repos/acme/diffpal/check-runs" {
			handlerErrs <- fmt.Errorf("path = %q, want /repos/acme/diffpal/check-runs", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusBadRequest)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			handlerErrs <- fmt.Errorf("Authorization = %q, want Bearer token", got)
			http.Error(w, "unexpected authorization", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	cmd := newReviewCommandWithRunner(func(_ context.Context, _ dpconfig.Config, opts reviewer.Options) (reviewer.Result, error) {
		if opts.BlockOn != "high" {
			t.Fatalf("BlockOn = %q, want high", opts.BlockOn)
		}
		return testReviewResult("github"), nil
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"github",
		"--out", filepath.Join(dir, "findings.json"),
		"--mode", "check-run,summary",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
	text := out.String()
	for _, needle := range []string{
		"findings=1",
		"bundle=",
		"mode=check_run path=.artifacts/diffpal/github-checkrun.json",
		"mode=summary path=.artifacts/diffpal/summary.md",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("output missing %q:\n%s", needle, text)
		}
	}
}

func TestReviewGitHubSkipsPublishForForks(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeTestConfig(t, dir)

	eventPath := writeGitHubEvent(t, `{
  "number": 10,
  "repository": {"full_name": "acme/diffpal"},
  "pull_request": {
    "base": {"sha": "base-a", "repo": {"full_name": "acme/diffpal"}},
    "head": {"sha": "head-a", "repo": {"full_name": "fork/diffpal"}}
  }
}`)
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_REPOSITORY", "acme/diffpal")
	t.Setenv("GITHUB_EVENT_PATH", eventPath)

	cmd := newReviewCommandWithRunner(func(_ context.Context, _ dpconfig.Config, _ reviewer.Options) (reviewer.Result, error) {
		result := testReviewResult("github")
		result.Bundle.BaseSHA = "base-a"
		result.Bundle.HeadSHA = "head-a"
		return result, nil
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"github", "--out", filepath.Join(dir, "findings.json")})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "publish=skipped-fork") {
		t.Fatalf("output missing fork skip marker:\n%s", out.String())
	}
}

func TestReviewGitHubRequiresConfiguredAuthEnv(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("DIFFPAL_GITLAB_TOKEN_TEST", "unused")
	t.Setenv("DIFFPAL_ADO_PAT_TEST", "unused")
	writeHostTestConfig(t, dir)

	cmd := newReviewCommandWithRunner(func(_ context.Context, _ dpconfig.Config, _ reviewer.Options) (reviewer.Result, error) {
		return testReviewResult("github"), nil
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"github", "--out", filepath.Join(dir, "findings.json"), "--mode", "summary"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want host auth failure")
	}
	coder, ok := err.(interface{ ExitCode() int })
	if !ok {
		t.Fatalf("error does not expose ExitCode(): %T", err)
	}
	if coder.ExitCode() != 2 {
		t.Fatalf("ExitCode() = %d, want 2", coder.ExitCode())
	}
	if !strings.Contains(err.Error(), "platforms.github.auth.token or GITHUB_TOKEN is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReviewGitLabPublishesSelectedHostArtifacts(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("GITLAB_TOKEN", "gitlab-token")
	t.Setenv("DIFFPAL_GITHUB_TOKEN_TEST", "unused")
	t.Setenv("DIFFPAL_ADO_PAT_TEST", "unused")
	writeHostTestConfig(t, dir)
	t.Setenv("CI_PROJECT_PATH", "acme/diffpal")
	t.Setenv("CI_MERGE_REQUEST_IID", "42")
	t.Setenv("CI_MERGE_REQUEST_DIFF_BASE_SHA", "base-a")
	t.Setenv("CI_COMMIT_SHA", "head-a")

	var requests atomic.Int32
	handlerErrs := make(chan error, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/projects/acme/diffpal/merge_requests/42/discussions" {
			handlerErrs <- fmt.Errorf("path = %q, want GitLab discussions endpoint", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusBadRequest)
			return
		}
		if got := r.Header.Get("PRIVATE-TOKEN"); got != "gitlab-token" {
			handlerErrs <- fmt.Errorf("PRIVATE-TOKEN = %q, want gitlab-token", got)
			http.Error(w, "unexpected token", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITLAB_API_URL", server.URL)

	cmd := newReviewCommandWithRunner(func(_ context.Context, _ dpconfig.Config, _ reviewer.Options) (reviewer.Result, error) {
		return testReviewResult("gitlab"), nil
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"gitlab", "--out", filepath.Join(dir, "findings.json"), "--mode", "discussions"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
}

func TestReviewADOPublishesSelectedHostArtifacts(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("AZURE_DEVOPS_EXT_PAT", "ado-token")
	t.Setenv("DIFFPAL_GITHUB_TOKEN_TEST", "unused")
	t.Setenv("DIFFPAL_GITLAB_TOKEN_TEST", "unused")
	writeHostTestConfig(t, dir)
	t.Setenv("SYSTEM_COLLECTIONURI", "https://dev.azure.com/acme/")
	t.Setenv("SYSTEM_TEAMPROJECT", "proj")
	t.Setenv("BUILD_REPOSITORY_ID", "repo-1")
	t.Setenv("SYSTEM_PULLREQUEST_PULLREQUESTID", "55")
	t.Setenv("SYSTEM_PULLREQUEST_TARGETCOMMITID", "base-a")
	t.Setenv("BUILD_SOURCEVERSION", "head-a")

	var requests atomic.Int32
	handlerErrs := make(chan error, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if !strings.HasSuffix(r.URL.Path, "/statuses") {
			handlerErrs <- fmt.Errorf("path = %q, want ADO statuses endpoint", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusBadRequest)
			return
		}
		if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Basic ") {
			handlerErrs <- fmt.Errorf("Authorization = %q, want Basic auth", got)
			http.Error(w, "unexpected authorization", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_ADO_API_URL", server.URL+"/_apis/git/repositories/repo-1/pullRequests/55")

	cmd := newReviewCommandWithRunner(func(_ context.Context, _ dpconfig.Config, _ reviewer.Options) (reviewer.Result, error) {
		return testReviewResult("ado"), nil
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"ado", "--out", filepath.Join(dir, "findings.json"), "--mode", "status"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
}

func TestReviewRequiresConfigWithExitCode2(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cmd := newReviewCommand()
	cmd.SetArgs([]string{"local"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want config failure")
	}

	coder, ok := err.(interface{ ExitCode() int })
	if !ok {
		t.Fatalf("error does not expose ExitCode(): %T", err)
	}
	if coder.ExitCode() != 2 {
		t.Fatalf("ExitCode() = %d, want 2", coder.ExitCode())
	}
}

func testReviewResult(reviewID string) reviewer.Result {
	return reviewer.Result{
		Bundle: findings.FindingsBundle{
			Version:  findings.VersionV1,
			ReviewID: reviewID,
			BaseSHA:  "base-a",
			HeadSHA:  "head-a",
			Findings: []findings.Finding{{
				ReviewID:   reviewID,
				RuleID:     "correctness.example",
				Category:   "correctness",
				Severity:   "high",
				Confidence: 0.9,
				Path:       "main.go",
				StartLine:  4,
				EndLine:    4,
				Title:      "example finding",
				Message:    "example message",
				Evidence:   "example evidence",
				Provider:   "openai-fast",
				Blocking:   true,
			}},
		},
		Files: []diff.FileChange{{
			FromPath: "main.go",
			ToPath:   "main.go",
		}},
		ChangedFiles:  1,
		ContextFiles:  1,
		ContextChunks: 1,
		TestSummary:   "no_tests_in_diff",
	}
}

func writeTestConfig(t *testing.T, dir string) {
	t.Helper()
	configDir := filepath.Join(dir, ".config", "diffpal")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", configDir, err)
	}
	const payload = `version: v1
defaults:
  provider: openai-fast
  policy: default
providers:
  openai-fast:
    type: openai
    openai:
      model: gpt-5
      api_key: test-key
policies:
  default:
    block_on: high
review:
  context_lines: 20
  max_files: 200
  chunking:
    max_patch_chars: 12000
    max_files_per_chunk: 20
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(payload), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) error = %v", err)
	}
}

func writeHostTestConfig(t *testing.T, dir string) {
	t.Helper()
	configDir := filepath.Join(dir, ".config", "diffpal")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", configDir, err)
	}
	const payload = `version: v1
defaults:
  provider: openai-fast
  policy: default
providers:
  openai-fast:
    type: openai
    openai:
      model: gpt-5
      api_key: test-key
policies:
  default:
    block_on: high
review:
  context_lines: 20
  max_files: 200
  chunking:
    max_patch_chars: 12000
    max_files_per_chunk: 20
platforms:
  github: {}
  gitlab: {}
  azure: {}
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(payload), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) error = %v", err)
	}
}

func writeGitHubEvent(t *testing.T, payload string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "event.json")
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
