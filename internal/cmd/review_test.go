package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
			t.Fatalf("MaxFiles = %d, want 200 default", opts.MaxFiles)
		}
		if opts.Language != "en" {
			t.Fatalf("Language = %q, want en from config", opts.Language)
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

func TestReviewLocalSubcommandPassesLanguageAndInstructionsFlags(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeTestConfig(t, dir)
	instructionsPath := filepath.Join(dir, "diffpal-instructions.md")
	if err := os.WriteFile(instructionsPath, []byte("Prefer security findings over style comments.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(instructions) error = %v", err)
	}

	cmd := newReviewCommandWithRunner(func(_ context.Context, _ dpconfig.Config, opts reviewer.Options) (reviewer.Result, error) {
		if opts.Language != "French" {
			t.Fatalf("Language = %q, want French", opts.Language)
		}
		wantInstructions := "Focus on request handlers.\n\nPrefer security findings over style comments."
		if opts.Instructions != wantInstructions {
			t.Fatalf("Instructions = %q, want %q", opts.Instructions, wantInstructions)
		}
		return testReviewResult("local"), nil
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"local",
		"--language", "French",
		"--instructions", "Focus on request handlers.",
		"--instructions-file", instructionsPath,
		"--out", filepath.Join(dir, "findings.json"),
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
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
	var coder interface{ ExitCode() int }
	if !errors.As(err, &coder) {
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
	t.Setenv("GITHUB_EVENT_PATH", writeGitHubEvent(t, `{"number":10,"repository":{"full_name":"acme/diffpal"}}`))

	var requests atomic.Int32
	var checkRunName string
	var summaryBody string
	handlerErrs := make(chan error, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			handlerErrs <- fmt.Errorf("Authorization = %q, want Bearer token", got)
			http.Error(w, "unexpected authorization", http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/diffpal/check-runs":
			var payload struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				handlerErrs <- fmt.Errorf("decode check run: %w", err)
				http.Error(w, "bad payload", http.StatusBadRequest)
				return
			}
			checkRunName = payload.Name
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/issues/10/comments":
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/diffpal/issues/10/comments":
			var payload struct {
				Body string `json:"body"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				handlerErrs <- fmt.Errorf("decode summary comment: %w", err)
				http.Error(w, "bad payload", http.StatusBadRequest)
				return
			}
			summaryBody = payload.Body
			w.WriteHeader(http.StatusCreated)
		default:
			handlerErrs <- fmt.Errorf("request = %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
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
		"--review-channel", "diffpal-dev",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := requests.Load(); got != 3 {
		t.Fatalf("requests = %d, want 3", got)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
	if checkRunName != "diffpal-dev-checks" {
		t.Fatalf("check run name = %q, want diffpal-dev-checks", checkRunName)
	}
	if !strings.Contains(summaryBody, "<!-- diffpal:summary:diffpal-dev -->") {
		t.Fatalf("summary body missing channel marker:\n%s", summaryBody)
	}
	if !strings.Contains(summaryBody, "# DiffPal Dev Review Summary") {
		t.Fatalf("summary body missing channel title:\n%s", summaryBody)
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

func TestReviewChannelFlagIsGitHubOnly(t *testing.T) {
	t.Parallel()

	cmd := newReviewCommandWithRunner(func(_ context.Context, _ dpconfig.Config, _ reviewer.Options) (reviewer.Result, error) {
		t.Fatal("review runner was called for help")
		return reviewer.Result{}, nil
	})

	for _, args := range [][]string{
		{"github", "--help"},
		{"gitlab", "--help"},
		{"ado", "--help"},
	} {
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute(%v) error = %v", args, err)
		}
		hasFlag := strings.Contains(out.String(), "--review-channel")
		if args[0] == "github" && !hasFlag {
			t.Fatalf("github help missing --review-channel:\n%s", out.String())
		}
		if args[0] != "github" && hasFlag {
			t.Fatalf("%s help includes GitHub-only --review-channel:\n%s", args[0], out.String())
		}
	}
}

func TestReviewGitHubRejectsInvalidReviewChannelBeforeRunningReview(t *testing.T) {
	t.Parallel()

	called := false
	cmd := newReviewCommandWithRunner(func(_ context.Context, _ dpconfig.Config, _ reviewer.Options) (reviewer.Result, error) {
		called = true
		return reviewer.Result{}, nil
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"github", "--review-channel", "bad/channel", "--dry-run"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want invalid review channel error")
	}
	if called {
		t.Fatal("review runner was called for invalid review channel")
	}
	var coder interface{ ExitCode() int }
	if !errors.As(err, &coder) {
		t.Fatalf("error does not expose ExitCode(): %T", err)
	}
	if coder.ExitCode() != 2 {
		t.Fatalf("ExitCode() = %d, want 2", coder.ExitCode())
	}
	if !strings.Contains(err.Error(), "invalid review channel") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReviewGitHubDryRunPrintsMarkdownWithoutPublishing(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeHostTestConfig(t, dir)
	t.Setenv("GITHUB_REPOSITORY", "acme/diffpal")
	t.Setenv("GITHUB_BASE_SHA", "base-a")
	t.Setenv("GITHUB_HEAD_SHA", "head-a")
	t.Setenv("GITHUB_EVENT_PATH", writeGitHubEvent(t, `{"number":10,"repository":{"full_name":"acme/diffpal"}}`))

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "unexpected request", http.StatusBadRequest)
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	cmd := newReviewCommandWithRunner(func(_ context.Context, _ dpconfig.Config, _ reviewer.Options) (reviewer.Result, error) {
		return testReviewResult("github"), nil
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"github",
		"--dry-run",
		"--out", filepath.Join(dir, "findings.json"),
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("requests = %d, want 0", got)
	}
	text := out.String()
	for _, want := range []string{
		"# DiffPal Review Summary",
		"https://github.com/acme/diffpal/blob/head-a/main.go#L4",
		"example evidence",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("dry-run output missing %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{"findings=", "bundle=", "```"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("dry-run output contains %q:\n%s", forbidden, text)
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

	var called atomic.Bool
	cmd := newReviewCommandWithRunner(func(_ context.Context, _ dpconfig.Config, _ reviewer.Options) (reviewer.Result, error) {
		called.Store(true)
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
	if called.Load() {
		t.Fatal("review runner was called for fork PR")
	}
}

func TestReviewGitHubSkipsPublishForForkPullRequestTargetWithToken(t *testing.T) {
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
	t.Setenv("GITHUB_EVENT_NAME", "pull_request_target")
	t.Setenv("GITHUB_TOKEN", "token")
	t.Setenv("GITHUB_REPOSITORY", "acme/diffpal")
	t.Setenv("GITHUB_EVENT_PATH", eventPath)

	var called atomic.Bool
	cmd := newReviewCommandWithRunner(func(_ context.Context, _ dpconfig.Config, _ reviewer.Options) (reviewer.Result, error) {
		called.Store(true)
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
	if called.Load() {
		t.Fatal("review runner was called for fork PR")
	}
}

func TestReviewGitHubForkSafetyFailsClosedOnContextError(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeTestConfig(t, dir)

	eventPath := filepath.Join(dir, "event.json")
	if err := os.WriteFile(eventPath, []byte(`{`), 0o644); err != nil {
		t.Fatalf("WriteFile(event.json) error = %v", err)
	}
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_REPOSITORY", "acme/diffpal")
	t.Setenv("GITHUB_EVENT_PATH", eventPath)

	var called atomic.Bool
	cmd := newReviewCommandWithRunner(func(_ context.Context, _ dpconfig.Config, _ reviewer.Options) (reviewer.Result, error) {
		called.Store(true)
		result := testReviewResult("github")
		result.Bundle.BaseSHA = "base-a"
		result.Bundle.HeadSHA = "head-a"
		return result, nil
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"github", "--out", filepath.Join(dir, "findings.json")})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want fork safety context error")
	}
	var coder interface{ ExitCode() int }
	if !errors.As(err, &coder) {
		t.Fatalf("Execute() error = %T, want exit coder", err)
	}
	if coder.ExitCode() != 4 {
		t.Fatalf("ExitCode() = %d, want 4", coder.ExitCode())
	}
	if !strings.Contains(err.Error(), "resolve github context for fork safety") {
		t.Fatalf("error missing fork safety context: %v", err)
	}
	if strings.Contains(out.String(), "mode=summary") {
		t.Fatalf("output shows publish artifacts despite context error:\n%s", out.String())
	}
	if called.Load() {
		t.Fatal("review runner was called after fork safety context error")
	}
}

func TestReviewGitHubSummaryCommentCanBeDisabled(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("GITHUB_TOKEN", "token")
	writeHostTestConfigWithGitHubSummary(t, dir, false)
	t.Setenv("GITHUB_REPOSITORY", "acme/diffpal")
	t.Setenv("GITHUB_BASE_SHA", "base-a")
	t.Setenv("GITHUB_HEAD_SHA", "head-a")
	t.Setenv("GITHUB_EVENT_PATH", writeGitHubEvent(t, `{"number":10,"repository":{"full_name":"acme/diffpal"}}`))

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "unexpected request", http.StatusBadRequest)
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	cmd := newReviewCommandWithRunner(func(_ context.Context, _ dpconfig.Config, _ reviewer.Options) (reviewer.Result, error) {
		return testReviewResult("github"), nil
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"github",
		"--out", filepath.Join(dir, "findings.json"),
		"--mode", "summary",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := requests.Load(); got != 0 {
		t.Fatalf("requests = %d, want 0", got)
	}
	if !strings.Contains(out.String(), "mode=summary path=.artifacts/diffpal/summary.md") {
		t.Fatalf("output missing summary artifact:\n%s", out.String())
	}
}

func TestReviewGitHubRequiresConfiguredAuthEnv(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("DIFFPAL_GITLAB_TOKEN_TEST", "unused")
	t.Setenv("DIFFPAL_ADO_PAT_TEST", "unused")
	writeHostTestConfig(t, dir)
	t.Setenv("GITHUB_REPOSITORY", "acme/diffpal")
	t.Setenv("GITHUB_BASE_SHA", "base-a")
	t.Setenv("GITHUB_HEAD_SHA", "head-a")
	t.Setenv("GITHUB_EVENT_PATH", writeGitHubEvent(t, `{"number":10,"repository":{"full_name":"acme/diffpal"}}`))

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
	var coder interface{ ExitCode() int }
	if !errors.As(err, &coder) {
		t.Fatalf("error does not expose ExitCode(): %T", err)
	}
	if coder.ExitCode() != 2 {
		t.Fatalf("ExitCode() = %d, want 2", coder.ExitCode())
	}
	if !strings.Contains(err.Error(), "diffpal.platforms.github.auth.token or GITHUB_TOKEN is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReviewHostRejectsInvalidFeedbackBeforeRunningReview(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeTestConfig(t, dir)

	called := false
	cmd := newReviewCommandWithRunner(func(_ context.Context, _ dpconfig.Config, _ reviewer.Options) (reviewer.Result, error) {
		called = true
		return testReviewResult("github"), nil
	})

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"github", "--feedback", "verbose", "--out", filepath.Join(dir, "findings.json")})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want invalid feedback error")
	}
	if called {
		t.Fatal("review runner was called for invalid feedback")
	}
	var coder interface{ ExitCode() int }
	if !errors.As(err, &coder) {
		t.Fatalf("error does not expose ExitCode(): %T", err)
	}
	if coder.ExitCode() != 2 {
		t.Fatalf("ExitCode() = %d, want 2", coder.ExitCode())
	}
	if !strings.Contains(err.Error(), "invalid feedback") {
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
		if r.URL.EscapedPath() != "/api/v4/projects/acme%2Fdiffpal/merge_requests/42/discussions" {
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
		_, _ = w.Write([]byte(`{"id":"discussion-1","notes":[]}`))
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
	t.Setenv("SYSTEM_TEAMPROJECT", "proj")
	t.Setenv("BUILD_REPOSITORY_ID", "repo-1")
	t.Setenv("SYSTEM_PULLREQUEST_PULLREQUESTID", "55")
	t.Setenv("SYSTEM_PULLREQUEST_TARGETCOMMITID", "base-a")
	t.Setenv("BUILD_SOURCEVERSION", "head-a")

	var requests atomic.Int32
	handlerErrs := make(chan error, 2)
	serverURL := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Basic ") {
			handlerErrs <- fmt.Errorf("Authorization = %q, want Basic auth", got)
			http.Error(w, "unexpected authorization", http.StatusUnauthorized)
			return
		}
		switch {
		case r.Method == http.MethodOptions && r.URL.Path == "/_apis":
			_, _ = w.Write([]byte(`{
  "count": 2,
  "value": [
    {
      "id": "e81700f7-3be2-46de-8624-2eb35882fcaa",
      "area": "Location",
      "resourceName": "ResourceAreas",
      "routeTemplate": "_apis/resourceAreas",
      "minVersion": "1.0",
      "maxVersion": "7.1",
      "releasedVersion": "7.1",
      "resourceVersion": 1
    },
    {
      "id": "b5f6bb4f-8d1e-4d79-8d11-4c9172c99c35",
      "area": "git",
      "resourceName": "pullRequestStatuses",
      "routeTemplate": "{project}/_apis/git/repositories/{repositoryId}/pullRequests/{pullRequestId}/statuses",
      "minVersion": "1.0",
      "maxVersion": "7.1",
      "releasedVersion": "7.1",
      "resourceVersion": 1
    }
  ]
}`))
		case r.Method == http.MethodGet && r.URL.Path == "/_apis/resourceAreas":
			_, _ = w.Write([]byte(`{
  "count": 1,
  "value": [
    {
      "id": "4e080c62-fa21-4fbc-8fef-2a10a2b38049",
      "locationUrl": "` + serverURL + `",
      "name": "git"
    }
  ]
}`))
		case r.Method == http.MethodPost && r.URL.Path == "/proj/_apis/git/repositories/repo-1/pullRequests/55/statuses":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":1}`))
		default:
			handlerErrs <- fmt.Errorf("request = %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	serverURL = server.URL
	t.Setenv("SYSTEM_COLLECTIONURI", server.URL+"/")

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
	if got := requests.Load(); got != 3 {
		t.Fatalf("requests = %d, want 3 SDK discovery/status requests", got)
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

	var coder interface{ ExitCode() int }
	if !errors.As(err, &coder) {
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
				Category:   "correctness",
				Severity:   "high",
				Confidence: 0.9,
				Path:       "main.go",
				StartLine:  4,
				EndLine:    4,
				Title:      "example finding",
				Message:    "example message",
				Evidence:   findings.NewEvidence("example evidence"),
				Provider:   "openai-fast",
				Blocking:   true,
			}},
		},
		Files: []diff.FileChange{{
			FromPath: "main.go",
			ToPath:   "main.go",
		}},
		ChangedFiles: 1,
	}
}

func writeTestConfig(t *testing.T, dir string) {
	t.Helper()
	configDir := filepath.Join(dir, ".config", "diffpal")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", configDir, err)
	}
	const payload = `version: v1
runtime:
  providers:
    openai-fast:
      type: openai
      openai:
        model: gpt-5
        api_key: test-key
diffpal:
  provider: openai-fast
  gate:
    block_on: high
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(payload), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) error = %v", err)
	}
}

func writeHostTestConfig(t *testing.T, dir string) {
	writeHostTestConfigWithGitHubSummary(t, dir, true)
}

func writeHostTestConfigWithGitHubSummary(t *testing.T, dir string, enabled bool) {
	t.Helper()
	configDir := filepath.Join(dir, ".config", "diffpal")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", configDir, err)
	}
	payload := fmt.Sprintf(`version: v1
runtime:
  providers:
    openai-fast:
      type: openai
      openai:
        model: gpt-5
        api_key: test-key
diffpal:
  provider: openai-fast
  gate:
    block_on: high
  platforms:
    github:
      summary_comment:
        enabled: %t
    gitlab: {}
    azure: {}
`, enabled)
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
