package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	dpconfig "github.com/diffpal/diffpal/internal/config"
	"github.com/diffpal/diffpal/internal/reviewer"
)

func TestUncommittedCommandSelectsPromptOnlyMode(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeTestConfig(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var got reviewer.Options
	command := newReviewCommandWithRunner(func(_ context.Context, _ dpconfig.Config, opts reviewer.Options) (reviewer.Result, error) {
		got = opts
		return testReviewResult(opts.ReviewID), nil
	})
	uncommitted, _, err := command.Find([]string{"uncommitted"})
	if err != nil {
		t.Fatal(err)
	}
	for _, removed := range []string{"base", "head", "scope", "path", "untracked"} {
		if uncommitted.Flags().Lookup(removed) != nil {
			t.Fatalf("uncommitted command exposes removed --%s flag", removed)
		}
	}

	var out bytes.Buffer
	command.SetOut(&out)
	command.SetErr(&out)
	command.SetArgs([]string{"uncommitted", "--out", filepath.Join(dir, "findings.json")})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Mode != reviewer.ModeUncommitted {
		t.Fatalf("Mode = %q, want %q", got.Mode, reviewer.ModeUncommitted)
	}
	if got.WorkingDir != "" {
		t.Fatalf("WorkingDir = %q, want normal runtime resolution", got.WorkingDir)
	}
	if got.BaseSHA != "" || got.HeadSHA != "" {
		t.Fatalf("range = %q..%q, want no CLI-selected range", got.BaseSHA, got.HeadSHA)
	}
}
