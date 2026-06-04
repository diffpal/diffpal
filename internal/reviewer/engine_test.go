package reviewer

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	dpconfig "github.com/diffpal/diffpal/internal/config"
	"github.com/diffpal/diffpal/internal/findings"
	"github.com/normahq/norma/pkg/runtime/agentconfig"
)

func TestRunWithRuntimeAggregatesFindingsAndAppliesBlocking(t *testing.T) {
	repo := newGitRepo(t)
	writeRepoFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() {\n\tprintln(\"before\")\n}\n")
	runGitCmd(t, repo, "add", "main.go")
	runGitCmd(t, repo, "commit", "-m", "initial")

	writeRepoFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() {\n\tprintln(\"after\")\n}\n")

	runtime := &fakeRuntime{
		outputs: []ChunkOutput{{
			Findings: []ChunkFinding{{
				RuleID:     "correctness.behavior-change",
				Category:   "correctness",
				Severity:   "high",
				Confidence: 0.94,
				Path:       "main.go",
				StartLine:  4,
				EndLine:    4,
				Title:      "behavior changed without guard",
				Message:    "the modified print path is not guarded by a flag",
				Evidence:   "line 4 now emits a different string",
			}},
		}},
	}

	result, err := RunWithRuntime(context.Background(), testConfig(), Options{
		WorkingDir:       repo,
		Repo:             "repo-a",
		ReviewID:         "review-a",
		MaxFiles:         20,
		ContextLines:     3,
		MaxPatchChars:    12000,
		MaxFilesPerChunk: 20,
		BlockOn:          "high",
	}, runtime)
	if err != nil {
		t.Fatalf("RunWithRuntime() error = %v", err)
	}
	if len(runtime.inputs) != 1 {
		t.Fatalf("runtime calls = %d, want 1", len(runtime.inputs))
	}
	if result.ChangedFiles != 1 || result.ReviewableFiles != 1 {
		t.Fatalf("file counts = changed %d reviewable %d, want 1/1", result.ChangedFiles, result.ReviewableFiles)
	}
	if result.ContextChunks != 1 {
		t.Fatalf("ContextChunks = %d, want 1", result.ContextChunks)
	}
	if result.TestSummary != "no_tests_in_diff" {
		t.Fatalf("TestSummary = %q, want no_tests_in_diff", result.TestSummary)
	}
	if len(result.Bundle.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1", len(result.Bundle.Findings))
	}
	got := result.Bundle.Findings[0]
	if !got.Blocking {
		t.Fatal("Blocking = false, want true")
	}
	if got.Provider != "openai-fast" {
		t.Fatalf("Provider = %q, want openai-fast", got.Provider)
	}
	if got.ReviewID != "review-a" {
		t.Fatalf("ReviewID = %q, want review-a", got.ReviewID)
	}
}

func TestRunWithRuntimeDropsInvalidFindingsAndSkipsDeletedFiles(t *testing.T) {
	repo := newGitRepo(t)
	writeRepoFile(t, filepath.Join(repo, "keep.go"), "package main\n\nfunc keep() {\n\tprintln(\"before\")\n}\n")
	writeRepoFile(t, filepath.Join(repo, "gone.go"), "package main\n")
	runGitCmd(t, repo, "add", ".")
	runGitCmd(t, repo, "commit", "-m", "initial")

	writeRepoFile(t, filepath.Join(repo, "keep.go"), "package main\n\nfunc keep() {\n\tprintln(\"after\")\n}\n")
	if err := os.Remove(filepath.Join(repo, "gone.go")); err != nil {
		t.Fatalf("Remove(gone.go) error = %v", err)
	}

	runtime := &fakeRuntime{
		outputs: []ChunkOutput{{
			Findings: []ChunkFinding{
				{
					RuleID:     "maintainability.output-change",
					Category:   "maintainability",
					Severity:   "medium",
					Confidence: 0.88,
					Path:       "keep.go",
					StartLine:  4,
					EndLine:    4,
					Title:      "output changed",
					Message:    "the function output changed",
					Evidence:   "line 4 was edited",
				},
				{
					RuleID:     "maintainability.output-change",
					Category:   "maintainability",
					Severity:   "medium",
					Confidence: 0.88,
					Path:       "keep.go",
					StartLine:  4,
					EndLine:    4,
					Title:      "output changed",
					Message:    "the function output changed",
					Evidence:   "line 4 was edited",
				},
				{
					RuleID:     "security.bad-category",
					Category:   "unknown",
					Severity:   "high",
					Confidence: 0.9,
					Path:       "keep.go",
					StartLine:  4,
					EndLine:    4,
					Title:      "bad category",
					Message:    "bad category",
					Evidence:   "bad category",
				},
				{
					RuleID:     "security.out-of-range",
					Category:   "security",
					Severity:   "high",
					Confidence: 0.9,
					Path:       "gone.go",
					StartLine:  1,
					EndLine:    1,
					Title:      "deleted file finding",
					Message:    "deleted file should be ignored",
					Evidence:   "file is deleted",
				},
			},
		}},
	}

	result, err := RunWithRuntime(context.Background(), testConfig(), Options{
		WorkingDir:       repo,
		Repo:             "repo-b",
		ReviewID:         "review-b",
		MaxFiles:         20,
		ContextLines:     3,
		MaxPatchChars:    12000,
		MaxFilesPerChunk: 20,
		BlockOn:          "high",
	}, runtime)
	if err != nil {
		t.Fatalf("RunWithRuntime() error = %v", err)
	}
	if result.ChangedFiles != 2 {
		t.Fatalf("ChangedFiles = %d, want 2", result.ChangedFiles)
	}
	if result.ReviewableFiles != 1 {
		t.Fatalf("ReviewableFiles = %d, want 1", result.ReviewableFiles)
	}
	if len(result.Bundle.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1", len(result.Bundle.Findings))
	}
	if result.Bundle.Findings[0].Path != "keep.go" {
		t.Fatalf("finding path = %q, want keep.go", result.Bundle.Findings[0].Path)
	}
}

func TestRunWithRuntimeRetriesTransientRuntimeFailures(t *testing.T) {
	repo := newGitRepo(t)
	writeRepoFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() {\n\tprintln(\"before\")\n}\n")
	runGitCmd(t, repo, "add", "main.go")
	runGitCmd(t, repo, "commit", "-m", "initial")
	writeRepoFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() {\n\tprintln(\"after\")\n}\n")

	runtime := &fakeRuntime{
		errs: []error{
			wrapError(KindTransient, errors.New("429 rate limit")),
			nil,
		},
		outputs: []ChunkOutput{
			{},
			{
				Findings: []ChunkFinding{{
					RuleID:     "testing.retry-success",
					Category:   "testing",
					Severity:   "low",
					Confidence: 0.5,
					Path:       "main.go",
					StartLine:  4,
					EndLine:    4,
					Title:      "retry recovered",
					Message:    "the second attempt succeeded",
					Evidence:   "transient failure was retried",
				}},
			},
		},
	}

	result, err := RunWithRuntime(context.Background(), testConfig(), Options{
		WorkingDir:       repo,
		Repo:             "repo-c",
		ReviewID:         "review-c",
		MaxFiles:         20,
		ContextLines:     3,
		MaxPatchChars:    12000,
		MaxFilesPerChunk: 20,
		BlockOn:          "high",
	}, runtime)
	if err != nil {
		t.Fatalf("RunWithRuntime() error = %v", err)
	}
	if runtime.calls != 2 {
		t.Fatalf("runtime calls = %d, want 2", runtime.calls)
	}
	if len(result.Bundle.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1", len(result.Bundle.Findings))
	}
}

type fakeRuntime struct {
	outputs []ChunkOutput
	errs    []error
	inputs  []ChunkInput
	calls   int
}

func (f *fakeRuntime) ReviewChunk(_ context.Context, _ RuntimeConfig, input ChunkInput) (ChunkOutput, RuntimeUsage, error) {
	f.inputs = append(f.inputs, input)
	idx := f.calls
	f.calls++

	var err error
	if idx < len(f.errs) {
		err = f.errs[idx]
	}
	if err != nil {
		return ChunkOutput{}, RuntimeUsage{}, err
	}

	if idx >= len(f.outputs) {
		return ChunkOutput{}, RuntimeUsage{}, nil
	}
	return f.outputs[idx], RuntimeUsage{}, nil
}

func testConfig() dpconfig.Config {
	return dpconfig.Config{
		Version: "v1",
		Defaults: dpconfig.DefaultsConfig{
			Provider: "openai-fast",
			Policy:   "default",
		},
		Providers: map[string]dpconfig.ProviderConfig{
			"openai-fast": {
				Type: "openai",
				OpenAI: &agentconfig.LocalAPIConfig{
					Model:  "gpt-5",
					APIKey: "test-key",
				},
			},
		},
		Policies: map[string]dpconfig.PolicyConfig{
			"default": {BlockOn: "high"},
		},
		Review: dpconfig.ReviewConfig{
			ContextLines: 20,
			MaxFiles:     200,
		},
	}
}

func newGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGitCmd(t, dir, "init")
	runGitCmd(t, dir, "config", "user.email", "test@example.com")
	runGitCmd(t, dir, "config", "user.name", "DiffPal Test")
	return dir
}

func writeRepoFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}

func runGitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestDedupeAndSortFindingsKeepsStableOrder(t *testing.T) {
	items := []findings.Finding{
		{RuleID: "style.beta", Message: "beta", Evidence: "e2", Path: "b.go", StartLine: 3, EndLine: 3},
		{RuleID: "style.alpha", Message: "alpha", Evidence: "e1", Path: "a.go", StartLine: 2, EndLine: 2},
		{RuleID: "style.alpha", Message: "alpha", Evidence: "e1", Path: "a.go", StartLine: 2, EndLine: 2},
	}

	got := dedupeAndSortFindings(items, "repo", "review", "head")
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].Path != "a.go" || got[1].Path != "b.go" {
		t.Fatalf("sorted paths = %q, %q; want a.go then b.go", got[0].Path, got[1].Path)
	}
}
