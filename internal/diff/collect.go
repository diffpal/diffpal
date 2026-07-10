package diff

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const maxDiffLineBytes = 16 << 20

type ChangeStatus string

const (
	ChangeAdded    ChangeStatus = "added"
	ChangeModified ChangeStatus = "modified"
	ChangeDeleted  ChangeStatus = "deleted"
	ChangeRenamed  ChangeStatus = "renamed"
)

type FileChange struct {
	FromPath         string
	ToPath           string
	Status           ChangeStatus
	IsRename         bool
	RawHeader        string
	ChangedLineSpans []LineSpan
}

type LineSpan struct {
	Start int
	End   int
}

type DiffResult struct {
	BaseSHA      string
	HeadSHA      string
	RawDiff      string
	Files        []FileChange
	ChangedFiles int
}

type Options struct {
	BaseSHA string
	HeadSHA string
	WorkDir string
}

func Collect(ctx context.Context, opts Options) (DiffResult, error) {
	workDir := opts.WorkDir
	if workDir == "" {
		workDir = "."
	}

	baseSHA, err := resolveBaseRevision(ctx, workDir, opts.BaseSHA)
	if err != nil {
		return DiffResult{}, err
	}
	headSHA, err := resolveHeadRevision(ctx, workDir, opts.HeadSHA)
	if err != nil {
		return DiffResult{}, err
	}

	args := []string{"diff", "--find-renames", "--unified=3", "--no-color", "--no-ext-diff"}
	switch {
	case baseSHA != "" && headSHA != "":
		args = append(args, fmt.Sprintf("%s..%s", baseSHA, headSHA))
	case baseSHA != "":
		args = append(args, baseSHA)
	case headSHA != "":
		args = append(args, headSHA)
	}

	raw, err := runGit(ctx, workDir, args...)
	if err != nil {
		return DiffResult{}, fmt.Errorf("git diff failed: %w", err)
	}
	raw = normalizeDiff(raw)
	files, err := normalizeDiffFiles(raw)
	if err != nil {
		return DiffResult{}, fmt.Errorf("parse git diff: %w", err)
	}

	return DiffResult{
		BaseSHA:      baseSHA,
		HeadSHA:      headSHA,
		RawDiff:      raw,
		Files:        files,
		ChangedFiles: len(files),
	}, nil
}

func normalizeDiffFiles(raw string) ([]FileChange, error) {
	out := []FileChange{}
	scanner := bufio.NewScanner(strings.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), maxDiffLineBytes)
	var current *FileChange
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "diff --git ") {
			if current != nil {
				out = append(out, *current)
			}
			left, right, err := parseDiffHeader(line)
			if err != nil {
				return nil, err
			}
			change := FileChange{
				FromPath:  filepath.Clean(left),
				ToPath:    filepath.Clean(right),
				Status:    inferStatus(left, right),
				IsRename:  left != right,
				RawHeader: line,
			}
			if change.IsRename {
				change.Status = ChangeRenamed
			}
			current = &change
			continue
		}
		if current == nil {
			continue
		}
		switch {
		case strings.HasPrefix(line, "new file mode "):
			current.Status = ChangeAdded
			continue
		case strings.HasPrefix(line, "deleted file mode "):
			current.Status = ChangeDeleted
			continue
		case strings.HasPrefix(line, "rename from "):
			current.FromPath = filepath.Clean(decodeGitPath(strings.TrimPrefix(line, "rename from ")))
			current.IsRename = true
			current.Status = ChangeRenamed
			continue
		case strings.HasPrefix(line, "rename to "):
			current.ToPath = filepath.Clean(decodeGitPath(strings.TrimPrefix(line, "rename to ")))
			current.IsRename = true
			current.Status = ChangeRenamed
			continue
		case strings.HasPrefix(line, "--- "):
			current.FromPath = normalizePatchPath(strings.TrimPrefix(line, "--- "))
			if current.FromPath == "/dev/null" {
				current.Status = ChangeAdded
			}
			continue
		case strings.HasPrefix(line, "+++ "):
			current.ToPath = normalizePatchPath(strings.TrimPrefix(line, "+++ "))
			if current.ToPath == "/dev/null" {
				current.Status = ChangeDeleted
			}
			continue
		}
		if span, ok := parseAddedSpan(line); ok {
			current.ChangedLineSpans = append(current.ChangedLineSpans, span)
		}
	}
	if current != nil {
		out = append(out, *current)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan diff: %w", err)
	}
	return out, nil
}

func parseDiffHeader(line string) (string, string, error) {
	rest := strings.TrimPrefix(line, "diff --git ")
	left, rest, err := nextGitPath(rest)
	if err != nil {
		return "", "", fmt.Errorf("invalid diff header %q: %w", line, err)
	}
	right, trailing, err := nextGitPath(rest)
	if err != nil {
		return "", "", fmt.Errorf("invalid diff header %q: %w", line, err)
	}
	if strings.TrimSpace(trailing) != "" {
		return "", "", fmt.Errorf("invalid diff header %q: unexpected trailing data", line)
	}
	return trimPrefix(left, "a/"), trimPrefix(right, "b/"), nil
}

func nextGitPath(value string) (string, string, error) {
	value = strings.TrimLeft(value, " \t")
	if value == "" {
		return "", "", fmt.Errorf("missing path")
	}
	if value[0] != '"' {
		pathValue, rest, _ := strings.Cut(value, " ")
		return pathValue, rest, nil
	}
	escaped := false
	for i := 1; i < len(value); i++ {
		switch {
		case escaped:
			escaped = false
		case value[i] == '\\':
			escaped = true
		case value[i] == '"':
			decoded, err := strconv.Unquote(value[:i+1])
			if err != nil {
				return "", "", err
			}
			return decoded, value[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("unterminated quoted path")
}

func resolveBaseRevision(ctx context.Context, workDir string, rev string) (string, error) {
	if strings.TrimSpace(rev) == "" {
		return "", nil
	}
	return resolveRevision(ctx, workDir, rev)
}

func resolveHeadRevision(ctx context.Context, workDir string, rev string) (string, error) {
	if strings.TrimSpace(rev) == "" {
		if !insideWorkTree(ctx, workDir) {
			return "", nil
		}
		return resolveRevision(ctx, workDir, "HEAD")
	}
	return resolveRevision(ctx, workDir, rev)
}

func resolveRevision(ctx context.Context, workDir string, rev string) (string, error) {
	resolved, err := runGit(ctx, workDir, "rev-parse", "--verify", rev+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", rev, err)
	}
	return strings.TrimSpace(resolved), nil
}

func insideWorkTree(ctx context.Context, workDir string) bool {
	out, err := runGit(ctx, workDir, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == "true"
}

func inferStatus(left string, right string) ChangeStatus {
	switch {
	case left == "/dev/null":
		return ChangeAdded
	case right == "/dev/null":
		return ChangeDeleted
	default:
		return ChangeModified
	}
}

func parseAddedSpan(line string) (LineSpan, bool) {
	if !strings.HasPrefix(line, "@@ ") && !strings.HasPrefix(line, "@@-") && !strings.HasPrefix(line, "@@ -") {
		return LineSpan{}, false
	}
	parts := strings.Split(line, " ")
	for _, part := range parts {
		if !strings.HasPrefix(part, "+") || len(part) < 2 {
			continue
		}
		start, count, ok := parseSpanToken(strings.TrimPrefix(part, "+"))
		if !ok {
			continue
		}
		if count == 0 {
			return LineSpan{Start: start, End: start}, true
		}
		return LineSpan{Start: start, End: start + count - 1}, true
	}
	return LineSpan{}, false
}

func parseSpanToken(token string) (int, int, bool) {
	token = strings.TrimSuffix(token, "@@")
	parts := strings.SplitN(token, ",", 2)
	start, err := parsePositiveInt(parts[0])
	if err != nil {
		return 0, 0, false
	}
	if len(parts) == 1 {
		return start, 1, true
	}
	count, err := parsePositiveInt(parts[1])
	if err != nil {
		return 0, 0, false
	}
	return start, count, true
}

func parsePositiveInt(v string) (int, error) {
	n := 0
	for _, r := range v {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid int")
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

func normalizeDiff(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	return strings.TrimSuffix(raw, "\n") + "\n"
}

func trimPrefix(v, prefix string) string {
	return strings.TrimPrefix(v, prefix)
}

func normalizePatchPath(v string) string {
	v = decodeGitPath(v)
	switch v {
	case "/dev/null":
		return v
	default:
		return filepath.Clean(trimPrefix(trimPrefix(v, "a/"), "b/"))
	}
}

func decodeGitPath(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, `"`) {
		if decoded, err := strconv.Unquote(value); err == nil {
			return decoded
		}
	}
	return value
}

func runGit(ctx context.Context, workDir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workDir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return stdout.String(), nil
}
