package reviewer

import (
	"os"
	"path/filepath"
	"testing"
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
