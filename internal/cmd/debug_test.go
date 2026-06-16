package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDebugPromptRendersReviewArtifactsOffline(t *testing.T) {
	repo := newCommandGitRepo(t)
	writeCommandFile(t, filepath.Join(repo, ".config/diffpal/config.yaml"), `version: v1

runtime:
  providers:
    codex-acp:
      type: codex_acp
      codex_acp:
        reasoning_effort: low

diffpal:
  provider: codex-acp
  gate:
    block_on: high
  review:
    language: en
    checks:
      - security
`)
	writeCommandFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() {\n\tprintln(\"before\")\n}\n")
	runCommandGit(t, repo, "add", ".")
	runCommandGit(t, repo, "commit", "-m", "initial")
	writeCommandFile(t, filepath.Join(repo, "main.go"), "package main\n\nfunc main() {\n\tprintln(\"after\")\n}\n")
	runCommandGit(t, repo, "add", "main.go")
	runCommandGit(t, repo, "commit", "-m", "change output")

	previousWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("Chdir(%s) error = %v", repo, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previousWorkingDir); err != nil {
			t.Fatalf("restore working dir: %v", err)
		}
		rootConfigDir = ""
		rootProfile = ""
	})

	outPath := filepath.Join(repo, ".artifacts/diffpal/debug-findings.json")
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"debug", "prompt",
		"--base", "HEAD~1",
		"--head", "HEAD",
		"--out", outPath,
		"--format", "text",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v\n%s", err, out.String())
	}
	text := out.String()
	for _, needle := range []string{
		"## System Prompt",
		"# Provider adapter instructions",
		"## Task Snapshot",
		"DiffPal review task snapshot",
		"## Mock Bundle",
		`"version": "v2"`,
		`"schema_version": "findings.v2"`,
		"Debug harness rendered the review task without contacting a provider.",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("debug prompt output missing %q:\n%s", needle, text)
		}
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("debug prompt did not write bundle: %v", err)
	}
}

func newCommandGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runCommandGit(t, dir, "init")
	runCommandGit(t, dir, "config", "user.email", "test@example.com")
	runCommandGit(t, dir, "config", "user.name", "DiffPal Test")
	return dir
}

func runCommandGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s error = %v\n%s", strings.Join(args, " "), err, string(out))
	}
}

func writeCommandFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}
