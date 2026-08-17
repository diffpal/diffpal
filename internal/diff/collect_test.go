package diff

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeDiffFilesParsesExactSideAwareChangedSpans(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		header     string
		diff       string
		want       []LineSpan
		wantStatus ChangeStatus
	}{
		{
			name:       "replacement excludes context",
			header:     "diff --git a/file.txt b/file.txt\n",
			diff:       "@@ -10,4 +10,5 @@ func example() {\n keep\n-old one\n-old two\n+new one\n+new two\n+new three\n keep too\n",
			wantStatus: ChangeModified,
			want: []LineSpan{
				{Start: 11, End: 12, Side: SideLeft},
				{Start: 11, End: 13, Side: SideRight},
			},
		},
		{
			name:       "multiple hunks",
			header:     "diff --git a/file.txt b/file.txt\n",
			diff:       "@@ -1,2 +1,2 @@\n-old\n+new\n context\n@@ -20,2 +20,2 @@\n context\n-old later\n+new later\n",
			wantStatus: ChangeModified,
			want: []LineSpan{
				{Start: 1, End: 1, Side: SideLeft},
				{Start: 1, End: 1, Side: SideRight},
				{Start: 21, End: 21, Side: SideLeft},
				{Start: 21, End: 21, Side: SideRight},
			},
		},
		{
			name:       "new file zero old count",
			header:     "diff --git a/new.txt b/new.txt\n",
			diff:       "new file mode 100644\n--- /dev/null\n+++ b/new.txt\n@@ -0,0 +1,2 @@\n+one\n+two\n\\ No newline at end of file\n",
			want:       []LineSpan{{Start: 1, End: 2, Side: SideRight}},
			wantStatus: ChangeAdded,
		},
		{
			name:       "deleted file zero new count",
			header:     "diff --git a/old.txt b/old.txt\n",
			diff:       "deleted file mode 100644\n--- a/old.txt\n+++ /dev/null\n@@ -3,2 +2,0 @@\n-three\n-four\n\\ No newline at end of file\n",
			want:       []LineSpan{{Start: 3, End: 4, Side: SideLeft}},
			wantStatus: ChangeDeleted,
		},
		{
			name:       "rename with edit",
			header:     "diff --git a/old.txt b/new.txt\n",
			diff:       "similarity index 80%\nrename from old.txt\nrename to new.txt\n--- a/old.txt\n+++ b/new.txt\n@@ -5 +5 @@\n-before\n+after\n",
			wantStatus: ChangeRenamed,
			want: []LineSpan{
				{Start: 5, End: 5, Side: SideLeft},
				{Start: 5, End: 5, Side: SideRight},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := fmt.Sprintf("%sindex 1111111..2222222 100644\n%s", tt.header, tt.diff)
			files := normalizeDiffFiles(raw)
			if len(files) != 1 {
				t.Fatalf("len(files) = %d, want 1", len(files))
			}
			if got := fmt.Sprint(files[0].ChangedLineSpans); got != fmt.Sprint(tt.want) {
				t.Fatalf("ChangedLineSpans = %v, want %v", files[0].ChangedLineSpans, tt.want)
			}
			if files[0].Status != tt.wantStatus {
				t.Fatalf("Status = %q, want %q", files[0].Status, tt.wantStatus)
			}
		})
	}
}

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
	if got, want := file.ChangedLineSpans, []LineSpan{{Start: 2, End: 2, Side: SideRight}}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("ChangedLineSpans = %v, want %v", got, want)
	}
	if !strings.Contains(result.RawDiff, "diff --git a/old.txt b/new.txt") {
		t.Fatalf("RawDiff missing rename header:\n%s", result.RawDiff)
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
	if got, want := result.Files[0].ChangedLineSpans, []LineSpan{
		{Start: 1, End: 1, Side: SideLeft},
		{Start: 1, End: 1, Side: SideRight},
	}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("ChangedLineSpans = %v, want %v", got, want)
	}
}

func TestCollectParsesAddedFileRightSpan(t *testing.T) {
	t.Parallel()

	repo := newGitRepo(t)
	writeFile(t, filepath.Join(repo, "tracked.txt"), "tracked\n")
	runGitCmd(t, repo, "add", "tracked.txt")
	runGitCmd(t, repo, "commit", "-m", "initial")

	writeFile(t, filepath.Join(repo, "added.txt"), "one\ntwo\n")
	runGitCmd(t, repo, "add", "added.txt")

	result, err := Collect(Options{WorkDir: repo})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(result.Files) != 1 || result.Files[0].Status != ChangeAdded {
		t.Fatalf("Files = %+v, want one added file", result.Files)
	}
	if got, want := result.Files[0].ChangedLineSpans, []LineSpan{{Start: 1, End: 2, Side: SideRight}}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("ChangedLineSpans = %v, want %v", got, want)
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
	if got, want := result.Files[0].ChangedLineSpans, []LineSpan{{Start: 1, End: 1, Side: SideLeft}}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("ChangedLineSpans = %v, want %v", got, want)
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
