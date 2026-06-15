package reviewer

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

const (
	reviewToolMaxEntries       = 200
	reviewToolMaxReadBytes     = 64 * 1024
	reviewToolMaxSearchResults = 50
	reviewToolMaxSearchBytes   = 1024 * 1024
)

type fileToolset struct {
	root string
}

type listFilesArgs struct {
	Path string `json:"path,omitempty"`
}

type listFilesResult struct {
	Entries   []fileEntry `json:"entries"`
	Truncated bool        `json:"truncated"`
}

type fileEntry struct {
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size,omitempty"`
}

type readFileArgs struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

type readFileResult struct {
	Path      string   `json:"path"`
	StartLine int      `json:"start_line"`
	EndLine   int      `json:"end_line"`
	Lines     []string `json:"lines"`
	Truncated bool     `json:"truncated"`
}

type searchFilesArgs struct {
	Query string `json:"query"`
	Path  string `json:"path,omitempty"`
	Regex bool   `json:"regex,omitempty"`
}

type searchFilesResult struct {
	Matches   []searchMatch `json:"matches"`
	Truncated bool          `json:"truncated"`
}

type searchMatch struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

func newReviewTools(root string) ([]tool.Tool, error) {
	ts, err := newFileToolset(root)
	if err != nil {
		return nil, err
	}

	listFiles, err := functiontool.New(functiontool.Config{
		Name:        "list_files",
		Description: "List files under the review working directory.",
	}, ts.listFiles)
	if err != nil {
		return nil, fmt.Errorf("create list_files tool: %w", err)
	}
	readFile, err := functiontool.New(functiontool.Config{
		Name:        "read_file",
		Description: "Read a text file under the review working directory.",
	}, ts.readFile)
	if err != nil {
		return nil, fmt.Errorf("create read_file tool: %w", err)
	}
	searchFiles, err := functiontool.New(functiontool.Config{
		Name:        "search_files",
		Description: "Search text files under the review working directory.",
	}, ts.searchFiles)
	if err != nil {
		return nil, fmt.Errorf("create search_files tool: %w", err)
	}

	return []tool.Tool{listFiles, readFile, searchFiles}, nil
}

func newFileToolset(root string) (*fileToolset, error) {
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve tool root: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("resolve tool root symlinks: %w", err)
	}
	return &fileToolset{root: resolved}, nil
}

func (t *fileToolset) listFiles(_ tool.Context, args listFilesArgs) (listFilesResult, error) {
	dir, rel, err := t.resolvePath(args.Path)
	if err != nil {
		return listFilesResult{}, err
	}
	info, err := os.Stat(dir)
	if err != nil {
		return listFilesResult{}, fmt.Errorf("stat %q: %w", rel, err)
	}
	if !info.IsDir() {
		return listFilesResult{}, fmt.Errorf("%q is not a directory", rel)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return listFilesResult{}, fmt.Errorf("list %q: %w", rel, err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	result := listFilesResult{Entries: make([]fileEntry, 0, min(len(entries), reviewToolMaxEntries))}
	for i, entry := range entries {
		if i >= reviewToolMaxEntries {
			result.Truncated = true
			break
		}
		entryRel := filepath.ToSlash(filepath.Join(rel, entry.Name()))
		if rel == "." {
			entryRel = entry.Name()
		}
		item := fileEntry{Path: entryRel, IsDir: entry.IsDir()}
		if info, err := entry.Info(); err == nil && !info.IsDir() {
			item.Size = info.Size()
		}
		result.Entries = append(result.Entries, item)
	}
	return result, nil
}

func (t *fileToolset) readFile(_ tool.Context, args readFileArgs) (readFileResult, error) {
	path, rel, err := t.resolvePath(args.Path)
	if err != nil {
		return readFileResult{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return readFileResult{}, fmt.Errorf("stat %q: %w", rel, err)
	}
	if info.IsDir() {
		return readFileResult{}, fmt.Errorf("%q is a directory", rel)
	}
	if info.Size() > reviewToolMaxReadBytes {
		return readFileResult{}, fmt.Errorf("%q exceeds read limit of %d bytes", rel, reviewToolMaxReadBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return readFileResult{}, fmt.Errorf("read %q: %w", rel, err)
	}
	if !utf8.Valid(data) {
		return readFileResult{}, fmt.Errorf("%q is not valid UTF-8 text", rel)
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	start := args.StartLine
	if start <= 0 {
		start = 1
	}
	end := args.EndLine
	if end <= 0 || end > len(lines) {
		end = len(lines)
	}
	if start > end || start > len(lines) {
		return readFileResult{Path: rel, StartLine: start, EndLine: end, Lines: []string{}}, nil
	}

	selected := append([]string(nil), lines[start-1:end]...)
	truncated := false
	if len(strings.Join(selected, "\n")) > reviewToolMaxReadBytes {
		truncated = true
		selected = trimLinesToBytes(selected, reviewToolMaxReadBytes)
	}
	return readFileResult{
		Path:      rel,
		StartLine: start,
		EndLine:   start + len(selected) - 1,
		Lines:     selected,
		Truncated: truncated,
	}, nil
}

func (t *fileToolset) searchFiles(_ tool.Context, args searchFilesArgs) (searchFilesResult, error) {
	query := strings.TrimSpace(args.Query)
	if query == "" {
		return searchFilesResult{}, fmt.Errorf("query is required")
	}
	start, _, err := t.resolvePath(args.Path)
	if err != nil {
		return searchFilesResult{}, err
	}
	var expr *regexp.Regexp
	if args.Regex {
		expr, err = regexp.Compile(query)
		if err != nil {
			return searchFilesResult{}, fmt.Errorf("compile regex: %w", err)
		}
	}

	result := searchFilesResult{Matches: []searchMatch{}}
	walkErr := filepath.WalkDir(start, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if shouldSkipToolPath(entry) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil || info.Size() > reviewToolMaxSearchBytes {
			return nil
		}
		matches, err := searchFile(path, t.rel(path), query, expr)
		if err != nil {
			return nil
		}
		for _, match := range matches {
			if len(result.Matches) >= reviewToolMaxSearchResults {
				result.Truncated = true
				return fs.SkipAll
			}
			result.Matches = append(result.Matches, match)
		}
		return nil
	})
	if walkErr != nil && walkErr != fs.SkipAll {
		return searchFilesResult{}, fmt.Errorf("search files: %w", walkErr)
	}
	return result, nil
}

func (t *fileToolset) resolvePath(path string) (string, string, error) {
	cleaned := strings.TrimSpace(path)
	if cleaned == "" {
		cleaned = "."
	}
	if filepath.IsAbs(cleaned) {
		return "", "", fmt.Errorf("absolute paths are not allowed")
	}
	cleaned = filepath.Clean(cleaned)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("path escapes review root")
	}
	full := filepath.Join(t.root, cleaned)
	resolved, err := filepath.EvalSymlinks(full)
	if err != nil {
		return "", "", err
	}
	rel, err := filepath.Rel(t.root, resolved)
	if err != nil {
		return "", "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("path escapes review root")
	}
	return resolved, filepath.ToSlash(rel), nil
}

func (t *fileToolset) rel(path string) string {
	rel, err := filepath.Rel(t.root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func searchFile(path, rel, query string, expr *regexp.Regexp) ([]searchMatch, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), reviewToolMaxSearchBytes)
	matches := []searchMatch{}
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		if !utf8.ValidString(line) {
			return nil, fmt.Errorf("not text")
		}
		if expr != nil {
			if !expr.MatchString(line) {
				continue
			}
		} else if !strings.Contains(line, query) {
			continue
		}
		matches = append(matches, searchMatch{
			Path: rel,
			Line: lineNo,
			Text: strings.TrimSpace(line),
		})
	}
	return matches, scanner.Err()
}

func shouldSkipToolPath(entry fs.DirEntry) bool {
	name := entry.Name()
	if entry.Type()&fs.ModeSymlink != 0 {
		return true
	}
	return entry.IsDir() && (name == ".git" || name == "node_modules" || name == ".artifacts" || name == "dist")
}

func trimLinesToBytes(lines []string, maxBytes int) []string {
	var total int
	var out []string
	for _, line := range lines {
		next := len(line) + 1
		if total+next > maxBytes {
			break
		}
		total += next
		out = append(out, line)
	}
	return out
}
