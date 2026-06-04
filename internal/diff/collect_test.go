package diff

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectResolvesRefsAndParsesRename(t *testing.T) {
	t.Parallel()

	repo := newGitRepo(t)
	writeFile(t, filepath.Join(repo, "old.txt"), "one\n")
	runGitCmd(t, repo, "add", "old.txt")
	runGitCmd(t, repo, "commit", "-m", "initial")
	base := strings.TrimSpace(runGitCmd(t, repo, "rev-parse", "HEAD"))

	runGitCmd(t, repo, "mv", "old.txt", "new.txt")
	writeFile(t, filepath.Join(repo, "new.txt"), "one\ntwo\n")
	runGitCmd(t, repo, "commit", "-am", "rename")
	head := strings.TrimSpace(runGitCmd(t, repo, "rev-parse", "HEAD"))

	result, err := Collect(Options{
		BaseSHA: "HEAD~1",
		HeadSHA: "HEAD",
		WorkDir: repo,
	})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if result.BaseSHA != base {
		t.Fatalf("BaseSHA = %q, want %q", result.BaseSHA, base)
	}
	if result.HeadSHA != head {
		t.Fatalf("HeadSHA = %q, want %q", result.HeadSHA, head)
	}
	if result.ChangedFiles != 1 {
		t.Fatalf("ChangedFiles = %d, want 1", result.ChangedFiles)
	}
	if len(result.Files) != 1 {
		t.Fatalf("len(Files) = %d, want 1", len(result.Files))
	}
	file := result.Files[0]
	if !file.IsRename || file.Status != ChangeRenamed {
		t.Fatalf("rename file = %+v, want renamed", file)
	}
	if file.FromPath != "old.txt" || file.ToPath != "new.txt" {
		t.Fatalf("file paths = %+v, want old.txt -> new.txt", file)
	}
	if len(file.ChangedLineSpans) == 0 {
		t.Fatalf("ChangedLineSpans = nil, want parsed spans")
	}
	if !strings.Contains(result.RawDiff, "diff --git a/old.txt b/new.txt") {
		t.Fatalf("RawDiff missing rename header:\n%s", result.RawDiff)
	}
}

func TestCollectLimitsFilesDeterministically(t *testing.T) {
	t.Parallel()

	repo := newGitRepo(t)
	writeFile(t, filepath.Join(repo, "a.txt"), "a\n")
	writeFile(t, filepath.Join(repo, "b.txt"), "b\n")
	runGitCmd(t, repo, "add", ".")
	runGitCmd(t, repo, "commit", "-m", "initial")

	writeFile(t, filepath.Join(repo, "a.txt"), "aa\n")
	writeFile(t, filepath.Join(repo, "b.txt"), "bb\n")

	result, err := Collect(Options{WorkDir: repo, MaxFiles: 1})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if result.ChangedFiles != 1 {
		t.Fatalf("ChangedFiles = %d, want 1", result.ChangedFiles)
	}
	if len(result.Files) != 1 {
		t.Fatalf("len(Files) = %d, want 1", len(result.Files))
	}
}

func TestCollectDefaultsHeadToHEAD(t *testing.T) {
	t.Parallel()

	repo := newGitRepo(t)
	writeFile(t, filepath.Join(repo, "a.txt"), "a\n")
	runGitCmd(t, repo, "add", "a.txt")
	runGitCmd(t, repo, "commit", "-m", "initial")
	head := strings.TrimSpace(runGitCmd(t, repo, "rev-parse", "HEAD"))

	writeFile(t, filepath.Join(repo, "a.txt"), "changed\n")
	result, err := Collect(Options{WorkDir: repo})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if result.HeadSHA != head {
		t.Fatalf("HeadSHA = %q, want %q", result.HeadSHA, head)
	}
	if len(result.Files) != 1 || result.Files[0].Status != ChangeModified {
		t.Fatalf("Files = %+v, want one modified file", result.Files)
	}
}

func TestCollectMarksDeletedFiles(t *testing.T) {
	t.Parallel()

	repo := newGitRepo(t)
	writeFile(t, filepath.Join(repo, "gone.txt"), "gone\n")
	runGitCmd(t, repo, "add", "gone.txt")
	runGitCmd(t, repo, "commit", "-m", "initial")

	if err := os.Remove(filepath.Join(repo, "gone.txt")); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	result, err := Collect(Options{WorkDir: repo})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(result.Files) != 1 {
		t.Fatalf("len(Files) = %d, want 1", len(result.Files))
	}
	if result.Files[0].Status != ChangeDeleted {
		t.Fatalf("Status = %q, want %q", result.Files[0].Status, ChangeDeleted)
	}
	if result.Files[0].ToPath != "/dev/null" {
		t.Fatalf("ToPath = %q, want /dev/null", result.Files[0].ToPath)
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

func writeFile(t *testing.T, path string, content string) {
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
	return string(out)
}
