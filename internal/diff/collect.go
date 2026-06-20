package diff

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

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

func Collect(opts Options) (DiffResult, error) {
	workDir := opts.WorkDir
	if workDir == "" {
		workDir = "."
	}

	baseSHA, err := resolveBaseRevision(workDir, opts.BaseSHA)
	if err != nil {
		return DiffResult{}, err
	}
	headSHA, err := resolveHeadRevision(workDir, opts.HeadSHA)
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

	raw, err := runGit(workDir, args...)
	if err != nil {
		return DiffResult{}, fmt.Errorf("git diff failed: %w", err)
	}
	raw = normalizeDiff(raw)
	files := normalizeDiffFiles(raw)

	return DiffResult{
		BaseSHA:      baseSHA,
		HeadSHA:      headSHA,
		RawDiff:      raw,
		Files:        files,
		ChangedFiles: len(files),
	}, nil
}

func normalizeDiffFiles(raw string) []FileChange {
	out := []FileChange{}
	scanner := bufio.NewScanner(strings.NewReader(raw))
	var current *FileChange
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "diff --git a/") {
			if current != nil {
				out = append(out, *current)
			}
			parts := strings.Split(line, " ")
			if len(parts) < 4 {
				current = nil
				continue
			}
			left := trimPrefix(parts[2], "a/")
			right := trimPrefix(parts[3], "b/")
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
			current.FromPath = filepath.Clean(strings.TrimPrefix(line, "rename from "))
			current.IsRename = true
			current.Status = ChangeRenamed
			continue
		case strings.HasPrefix(line, "rename to "):
			current.ToPath = filepath.Clean(strings.TrimPrefix(line, "rename to "))
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
	return out
}

func resolveBaseRevision(workDir string, rev string) (string, error) {
	if strings.TrimSpace(rev) == "" {
		return "", nil
	}
	return resolveRevision(workDir, rev)
}

func resolveHeadRevision(workDir string, rev string) (string, error) {
	if strings.TrimSpace(rev) == "" {
		if !insideWorkTree(workDir) {
			return "", nil
		}
		return resolveRevision(workDir, "HEAD")
	}
	return resolveRevision(workDir, rev)
}

func resolveRevision(workDir string, rev string) (string, error) {
	resolved, err := runGit(workDir, "rev-parse", "--verify", rev+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", rev, err)
	}
	return strings.TrimSpace(resolved), nil
}

func insideWorkTree(workDir string) bool {
	out, err := runGit(workDir, "rev-parse", "--is-inside-work-tree")
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
	switch v {
	case "/dev/null":
		return v
	default:
		return filepath.Clean(trimPrefix(trimPrefix(v, "a/"), "b/"))
	}
}

func runGit(workDir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
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
