package reviewer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	dpconfig "github.com/diffpal/diffpal/internal/config"
	"github.com/diffpal/diffpal/internal/reviewer/promptpack"
)

func TestFileToolsetListReadAndSearch(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "app.go"), []byte("package main\n\nfunc main() {\n\tprintln(\"needle\")\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tools, err := newFileToolset(root)
	if err != nil {
		t.Fatalf("newFileToolset() error = %v", err)
	}

	list, err := tools.listFiles(nil, listFilesArgs{Path: "src"})
	if err != nil {
		t.Fatalf("listFiles() error = %v", err)
	}
	if len(list.Entries) != 1 || list.Entries[0].Path != "src/app.go" {
		t.Fatalf("listFiles() = %#v, want src/app.go", list)
	}

	read, err := tools.readFile(nil, readFileArgs{Path: "src/app.go", StartLine: 3, EndLine: 4})
	if err != nil {
		t.Fatalf("readFile() error = %v", err)
	}
	if read.StartLine != 3 || read.EndLine != 4 || len(read.Lines) != 2 {
		t.Fatalf("readFile() = %#v, want lines 3-4", read)
	}

	found, err := tools.searchFiles(nil, searchFilesArgs{Query: "needle", Path: "src"})
	if err != nil {
		t.Fatalf("searchFiles() error = %v", err)
	}
	if len(found.Matches) != 1 || found.Matches[0].Path != "src/app.go" || found.Matches[0].Line != 4 {
		t.Fatalf("searchFiles() = %#v, want src/app.go:4", found)
	}

	regexFound, err := tools.searchFiles(nil, searchFilesArgs{Query: `print.*needle`, Path: "src", Regex: true})
	if err != nil {
		t.Fatalf("searchFiles(regex) error = %v", err)
	}
	if len(regexFound.Matches) != 1 {
		t.Fatalf("searchFiles(regex) = %#v, want one match", regexFound)
	}
}

func TestFileToolsetRejectsPathEscape(t *testing.T) {
	root := t.TempDir()
	tools, err := newFileToolset(root)
	if err != nil {
		t.Fatalf("newFileToolset() error = %v", err)
	}

	if _, _, err := tools.resolvePath("../outside"); err == nil {
		t.Fatal("resolvePath() error = nil, want traversal rejection")
	}
	if _, _, err := tools.resolvePath(filepath.Join(root, "inside")); err == nil {
		t.Fatal("resolvePath() error = nil, want absolute path rejection")
	}
}

func TestFileToolsetRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "secret.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	tools, err := newFileToolset(root)
	if err != nil {
		t.Fatalf("newFileToolset() error = %v", err)
	}
	if _, err := tools.readFile(nil, readFileArgs{Path: "secret.txt"}); err == nil {
		t.Fatal("readFile() error = nil, want symlink escape rejection")
	}
}

func TestReviewToolsetGitDiffIsScopedToChangedFiles(t *testing.T) {
	repo := newGitRepo(t)
	writeRepoFile(t, filepath.Join(repo, "app.go"), "package main\n\nfunc main() {\n\tprintln(\"before\")\n}\n")
	writeRepoFile(t, filepath.Join(repo, "other.go"), "package main\n")
	runGitCmd(t, repo, "add", ".")
	runGitCmd(t, repo, "commit", "-m", "initial")
	base := runGitCmd(t, repo, "rev-parse", "HEAD")
	writeRepoFile(t, filepath.Join(repo, "app.go"), "package main\n\nfunc main() {\n\tprintln(\"after\")\n}\n")
	runGitCmd(t, repo, "add", "app.go")
	runGitCmd(t, repo, "commit", "-m", "change app")
	head := runGitCmd(t, repo, "rev-parse", "HEAD")

	ts, err := newReviewToolset(reviewToolOptions{
		Root:    repo,
		BaseSHA: base,
		HeadSHA: head,
		ChangedFiles: []ChunkFile{{
			Path:   "app.go",
			Status: "modified",
			Spans:  []ChunkSpan{{Start: 4, End: 4}},
		}},
	})
	if err != nil {
		t.Fatalf("newReviewToolset() error = %v", err)
	}

	changed, err := ts.gitChangedFiles(nil, gitChangedFilesArgs{})
	if err != nil {
		t.Fatalf("gitChangedFiles() error = %v", err)
	}
	if len(changed.Files) != 1 || changed.Files[0].Path != "app.go" || !changed.Untrusted {
		t.Fatalf("gitChangedFiles() = %+v, want app.go untrusted metadata", changed)
	}

	got, err := ts.gitDiff(nil, gitDiffArgs{Path: "app.go"})
	if err != nil {
		t.Fatalf("gitDiff() error = %v", err)
	}
	if !strings.Contains(got.Diff, `println("after")`) || !strings.Contains(got.Diff, promptpack.UntrustedInputStart) {
		t.Fatalf("gitDiff() = %+v, want untrusted diff with changed line", got)
	}
	if _, err := ts.gitDiff(nil, gitDiffArgs{Path: "other.go"}); err == nil {
		t.Fatal("gitDiff(other.go) error = nil, want unchanged path rejection")
	}
	if _, err := ts.gitDiff(nil, gitDiffArgs{Path: "../app.go"}); err == nil {
		t.Fatal("gitDiff(../app.go) error = nil, want traversal rejection")
	}
}

func TestReviewToolsForProviderOnlyInjectsHostedTools(t *testing.T) {
	hosted, err := reviewToolsForProvider(dpconfig.ProviderConfig{Type: "openai"}, RuntimeConfig{WorkingDir: t.TempDir()})
	if err != nil {
		t.Fatalf("reviewToolsForProvider(openai) error = %v", err)
	}
	if len(hosted) == 0 {
		t.Fatal("reviewToolsForProvider(openai) returned no tools")
	}
	acp, err := reviewToolsForProvider(dpconfig.ProviderConfig{Type: "codex_acp"}, RuntimeConfig{WorkingDir: t.TempDir()})
	if err != nil {
		t.Fatalf("reviewToolsForProvider(codex_acp) error = %v", err)
	}
	if len(acp) != 0 {
		t.Fatalf("reviewToolsForProvider(codex_acp) returned %d tools, want none", len(acp))
	}
}
