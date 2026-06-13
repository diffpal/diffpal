package markdown

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/diffpal/diffpal/internal/findings"
)

const (
	maxSnippetLines = 25
	maxSnippetChars = 4000
)

type CodeSnippet struct {
	Language string
	Code     string
}

type SnippetProvider interface {
	Snippet(findings.Finding) (CodeSnippet, bool)
}

type SnippetFunc func(findings.Finding) (CodeSnippet, bool)

func (fn SnippetFunc) Snippet(finding findings.Finding) (CodeSnippet, bool) {
	if fn == nil {
		return CodeSnippet{}, false
	}
	return fn(finding)
}

type WorktreeSnippetProvider struct {
	Root string
}

func NewWorktreeSnippetProvider(root string) WorktreeSnippetProvider {
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	return WorktreeSnippetProvider{Root: root}
}

func (p WorktreeSnippetProvider) Snippet(finding findings.Finding) (CodeSnippet, bool) {
	if finding.StartLine <= 0 || strings.TrimSpace(finding.Path) == "" {
		return CodeSnippet{}, false
	}
	cleanPath, ok := cleanRelativePath(finding.Path)
	if !ok {
		return CodeSnippet{}, false
	}
	target, ok := safeSnippetPath(p.Root, cleanPath)
	if !ok {
		return CodeSnippet{}, false
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		return CodeSnippet{}, false
	}
	lines := splitSourceLines(string(raw))
	if finding.StartLine > len(lines) {
		return CodeSnippet{}, false
	}
	end := finding.EndLine
	if end <= 0 || end < finding.StartLine {
		end = finding.StartLine
	}
	if end > len(lines) {
		end = len(lines)
	}
	if end-finding.StartLine+1 > maxSnippetLines {
		end = finding.StartLine + maxSnippetLines - 1
	}
	code := strings.Join(lines[finding.StartLine-1:end], "\n")
	code = trimSnippetChars(code)
	if strings.TrimSpace(code) == "" {
		return CodeSnippet{}, false
	}
	return CodeSnippet{
		Language: languageForPath(cleanPath),
		Code:     code,
	}, true
}

func cleanRelativePath(path string) (string, bool) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "." || filepath.IsAbs(path) || strings.HasPrefix(path, ".."+string(filepath.Separator)) || path == ".." {
		return "", false
	}
	return path, true
}

func safeSnippetPath(root, cleanPath string) (string, bool) {
	cleanRoot := filepath.Clean(root)
	rootReal, err := filepath.EvalSymlinks(cleanRoot)
	if err != nil {
		return "", false
	}
	targetReal, err := filepath.EvalSymlinks(filepath.Join(cleanRoot, cleanPath))
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(rootReal, targetReal)
	if err != nil {
		return "", false
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return "", false
	}
	return targetReal, true
}

func splitSourceLines(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	lines := strings.Split(raw, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		return lines[:len(lines)-1]
	}
	return lines
}

func trimSnippetChars(code string) string {
	if len([]rune(code)) <= maxSnippetChars {
		return code
	}
	runes := []rune(code)
	return string(runes[:maxSnippetChars])
}

func languageForPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "ts"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "js"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".md", ".markdown":
		return "markdown"
	case ".sh", ".bash":
		return "bash"
	case ".py":
		return "python"
	case ".rb":
		return "ruby"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".kt", ".kts":
		return "kotlin"
	case ".cs":
		return "csharp"
	case ".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx":
		return "cpp"
	case ".c", ".h":
		return "c"
	case ".sql":
		return "sql"
	default:
		return ""
	}
}
