package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
)

func TestLanguageForPath(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"main.go":       "go",
		"app.tsx":       "ts",
		"worker.mjs":    "js",
		"data.json":     "json",
		"config.yml":    "yaml",
		"README.md":     "markdown",
		"script.bash":   "bash",
		"tool.py":       "python",
		"task.rb":       "ruby",
		"lib.rs":        "rust",
		"Main.java":     "java",
		"build.kts":     "kotlin",
		"Program.cs":    "csharp",
		"native.hpp":    "cpp",
		"header.h":      "c",
		"migration.sql": "sql",
		"LICENSE":       "",
	}
	for path, want := range tests {
		if got := languageForPath(path); got != want {
			t.Errorf("languageForPath(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestWorktreeSnippetProviderRejectsSymlinkEscape(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.go")
	if err := os.WriteFile(outside, []byte("package secret\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(outside) error = %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link.go")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	provider := NewWorktreeSnippetProvider(root)
	if snippet, ok := provider.Snippet(findings.Finding{Path: "link.go", StartLine: 1, EndLine: 1}); ok {
		t.Fatalf("Snippet() ok = true for symlink escape: %+v", snippet)
	}
}

func TestWorktreeSnippetProviderExtractsBoundedLines(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte(strings.Join([]string{
		"package main",
		"",
		"func main() {",
		"println(\"hello\")",
		"}",
	}, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	snippet, ok := NewWorktreeSnippetProvider(root).Snippet(findings.Finding{
		Path:      "main.go",
		StartLine: 3,
		EndLine:   4,
	})
	if !ok {
		t.Fatal("Snippet() ok = false, want true")
	}
	if snippet.Language != "go" {
		t.Fatalf("Language = %q, want go", snippet.Language)
	}
	if snippet.Code != "func main() {\nprintln(\"hello\")" {
		t.Fatalf("Code = %q", snippet.Code)
	}
}
