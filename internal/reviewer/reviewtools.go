package reviewer

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/diffpal/diffpal/internal/reviewer/promptpack"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type inspectionTracker struct {
	mu    sync.Mutex
	calls map[string]int
}

func newInspectionTracker() *inspectionTracker {
	return &inspectionTracker{calls: map[string]int{}}
}

func (t *inspectionTracker) record(name string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.calls[name]++
}

func (t *inspectionTracker) callsList() []string {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, 0, len(t.calls))
	for name := range t.calls {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (t *inspectionTracker) called(name string) bool {
	if t == nil {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.calls[name] > 0
}

func (t *reviewToolset) recordToolCall(name string) {
	if t == nil {
		return
	}
	t.inspection.record(name)
}

func (t *fileToolset) recordToolCall(name string) {
	if t == nil {
		return
	}
	t.inspection.record(name)
}

const (
	reviewToolMaxEntries       = 200
	reviewToolMaxReadBytes     = 64 * 1024
	reviewToolMaxDiffBytes     = 128 * 1024
	reviewToolMaxSearchResults = 50
	reviewToolMaxSearchBytes   = 1024 * 1024
)

type reviewToolOptions struct {
	Root         string
	BaseSHA      string
	HeadSHA      string
	ChangedFiles []ChunkFile
	Inspection   *inspectionTracker
}

type reviewToolset struct {
	*fileToolset
	baseSHA      string
	headSHA      string
	changedFiles []ChunkFile
	changedPaths map[string]struct{}
	inspection   *inspectionTracker
}

type fileToolset struct {
	root       string
	inspection *inspectionTracker
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

type gitChangedFilesArgs struct{}

type gitChangedFilesResult struct {
	BaseSHA   string      `json:"base_sha"`
	HeadSHA   string      `json:"head_sha"`
	Files     []ChunkFile `json:"files"`
	Untrusted bool        `json:"untrusted"`
	Warning   string      `json:"warning"`
}

type gitDiffArgs struct {
	Path    string `json:"path,omitempty"`
	Unified int    `json:"unified,omitempty"`
}

type gitDiffResult struct {
	Path      string `json:"path,omitempty"`
	BaseSHA   string `json:"base_sha"`
	HeadSHA   string `json:"head_sha"`
	Diff      string `json:"diff"`
	Truncated bool   `json:"truncated"`
	Untrusted bool   `json:"untrusted"`
	Warning   string `json:"warning"`
}

func newReviewTools(opts reviewToolOptions) ([]tool.Tool, error) {
	ts, err := newReviewToolset(opts)
	if err != nil {
		return nil, err
	}

	gitChangedFiles, err := functiontool.New(functiontool.Config{
		Name:        "git_changed_files",
		Description: "Return the changed files and changed line spans for this DiffPal review task.",
	}, ts.gitChangedFiles)
	if err != nil {
		return nil, fmt.Errorf("create git_changed_files tool: %w", err)
	}
	gitDiff, err := functiontool.New(functiontool.Config{
		Name:        "git_diff",
		Description: "Read the untrusted Git diff for this review task, optionally scoped to one changed path.",
	}, ts.gitDiff)
	if err != nil {
		return nil, fmt.Errorf("create git_diff tool: %w", err)
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

	return []tool.Tool{gitChangedFiles, gitDiff, listFiles, readFile, searchFiles}, nil
}

func newReviewToolset(opts reviewToolOptions) (*reviewToolset, error) {
	files, err := newFileToolset(opts.Root)
	if err != nil {
		return nil, err
	}
	files.inspection = opts.Inspection
	changed := append([]ChunkFile(nil), opts.ChangedFiles...)
	sort.Slice(changed, func(i, j int) bool {
		return changed[i].Path < changed[j].Path
	})
	paths := make(map[string]struct{}, len(changed))
	for _, file := range changed {
		path, err := cleanReviewPath(file.Path)
		if err != nil {
			return nil, fmt.Errorf("changed file path %q: %w", file.Path, err)
		}
		paths[path] = struct{}{}
	}
	return &reviewToolset{
		fileToolset:  files,
		baseSHA:      strings.TrimSpace(opts.BaseSHA),
		headSHA:      strings.TrimSpace(opts.HeadSHA),
		changedFiles: changed,
		changedPaths: paths,
		inspection:   opts.Inspection,
	}, nil
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

func (t *reviewToolset) gitChangedFiles(_ tool.Context, _ gitChangedFilesArgs) (gitChangedFilesResult, error) {
	t.recordToolCall("git_changed_files")
	return gitChangedFilesResult{
		BaseSHA:   t.baseSHA,
		HeadSHA:   t.headSHA,
		Files:     append([]ChunkFile(nil), t.changedFiles...),
		Untrusted: true,
		Warning:   promptpack.UntrustedInputWarning,
	}, nil
}

func (t *reviewToolset) gitDiff(_ tool.Context, args gitDiffArgs) (gitDiffResult, error) {
	t.recordToolCall("git_diff")
	path := strings.TrimSpace(args.Path)
	if path != "" {
		cleaned, err := cleanReviewPath(path)
		if err != nil {
			return gitDiffResult{}, err
		}
		if _, ok := t.changedPaths[cleaned]; !ok {
			return gitDiffResult{}, fmt.Errorf("%q is not a changed path in this review task", cleaned)
		}
		path = cleaned
	}
	unified := args.Unified
	if unified <= 0 {
		unified = 3
	}
	if unified > 50 {
		unified = 50
	}
	gitArgs := []string{"diff", "--find-renames", fmt.Sprintf("--unified=%d", unified), "--no-color", "--no-ext-diff"}
	switch {
	case t.baseSHA != "" && t.headSHA != "":
		gitArgs = append(gitArgs, fmt.Sprintf("%s..%s", t.baseSHA, t.headSHA))
	case t.baseSHA != "":
		gitArgs = append(gitArgs, t.baseSHA)
	case t.headSHA != "":
		gitArgs = append(gitArgs, t.headSHA)
	}
	if path != "" {
		gitArgs = append(gitArgs, "--", path)
	} else if len(t.changedFiles) > 0 {
		gitArgs = append(gitArgs, "--")
		for _, file := range t.changedFiles {
			gitArgs = append(gitArgs, file.Path)
		}
	}
	out, err := runReviewGit(t.root, reviewToolMaxDiffBytes+1, gitArgs...)
	if err != nil {
		return gitDiffResult{}, fmt.Errorf("git diff: %w", err)
	}
	truncated := len(out) > reviewToolMaxDiffBytes
	if truncated {
		out = out[:reviewToolMaxDiffBytes]
	}
	return gitDiffResult{
		Path:      path,
		BaseSHA:   t.baseSHA,
		HeadSHA:   t.headSHA,
		Diff:      promptpack.UntrustedInputStart + "\n" + promptpack.EscapeUntrusted(out) + "\n" + promptpack.UntrustedInputEnd,
		Truncated: truncated,
		Untrusted: true,
		Warning:   promptpack.UntrustedInputWarning,
	}, nil
}

func (t *fileToolset) listFiles(_ tool.Context, args listFilesArgs) (listFilesResult, error) {
	t.recordToolCall("list_files")
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
	t.recordToolCall("read_file")
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
	t.recordToolCall("search_files")
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
	cleaned, err := cleanReviewPath(path)
	if err != nil {
		return "", "", err
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

func cleanReviewPath(path string) (string, error) {
	cleaned := strings.TrimSpace(path)
	if cleaned == "" {
		cleaned = "."
	}
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}
	cleaned = filepath.Clean(cleaned)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes review root")
	}
	return filepath.ToSlash(cleaned), nil
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

func runReviewGit(dir string, maxBytes int, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	if len(out) > maxBytes {
		out = out[:maxBytes]
	}
	return strings.ReplaceAll(string(out), "\r\n", "\n"), nil
}
