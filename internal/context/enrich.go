package context

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/diffpal/diffpal/internal/diff"
)

type FileContext struct {
	Path      string
	Signature string
	Snippet   string
	Spans     []diff.LineSpan
}

type Chunk struct {
	Files []FileContext
	Chars int
}

func Enrich(diffResult diff.DiffResult, contextLines int) ([]FileContext, string, error) {
	return EnrichWithWorkingDir(diffResult, contextLines, "")
}

func EnrichWithWorkingDir(diffResult diff.DiffResult, contextLines int, workingDir string) ([]FileContext, string, error) {
	out := make([]FileContext, 0, len(diffResult.Files))
	testCount := 0
	for _, changed := range diffResult.Files {
		if strings.HasSuffix(changed.ToPath, "_test.go") {
			testCount++
		}
		signature, snippet, spans, err := fileSignatureAndSnippet(resolveContextPath(workingDir, changed.ToPath), changed.ChangedLineSpans, contextLines)
		if err != nil {
			return nil, "", err
		}
		out = append(out, FileContext{
			Path:      changed.ToPath,
			Signature: signature,
			Snippet:   snippet,
			Spans:     spans,
		})
	}
	return out, summaryFromTests(testCount), nil
}

func resolveContextPath(workingDir, path string) string {
	if filepath.IsAbs(path) || strings.TrimSpace(workingDir) == "" {
		return path
	}
	return filepath.Join(workingDir, path)
}

func ChunkByLimits(contexts []FileContext, maxPatchChars int, maxFilesPerChunk int) []Chunk {
	if maxPatchChars <= 0 {
		maxPatchChars = 12000
	}
	if maxFilesPerChunk <= 0 {
		maxFilesPerChunk = 20
	}

	chunks := []Chunk{}
	current := Chunk{}
	for _, ctx := range contexts {
		size := len(ctx.Snippet)
		needsSplit := len(current.Files) > 0 && (current.Chars+size > maxPatchChars || len(current.Files) >= maxFilesPerChunk)
		if needsSplit {
			chunks = append(chunks, current)
			current = Chunk{}
		}
		current.Files = append(current.Files, ctx)
		current.Chars += size
	}
	if len(current.Files) > 0 {
		chunks = append(chunks, current)
	}
	return chunks
}

func fileSignatureAndSnippet(path string, spans []diff.LineSpan, contextLines int) (string, string, []diff.LineSpan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", nil, err
	}
	sum := sha1.Sum(data)
	signature := fmt.Sprintf("%x", sum[:])
	lines := splitLines(string(data))
	merged := expandAndMergeSpans(spans, contextLines, len(lines))
	if len(merged) == 0 && len(lines) > 0 {
		merged = []diff.LineSpan{{Start: 1, End: min(lineCountLimit(contextLines, len(lines)), len(lines))}}
	}

	builder := strings.Builder{}
	for i, span := range merged {
		if i > 0 {
			builder.WriteString("\n")
		}
		fmt.Fprintf(&builder, "@@ lines %d-%d @@\n", span.Start, span.End)
		for lineNo := span.Start; lineNo <= span.End && lineNo <= len(lines); lineNo++ {
			fmt.Fprintf(&builder, "%4d | %s\n", lineNo, lines[lineNo-1])
		}
	}
	return signature, builder.String(), merged, nil
}

func expandAndMergeSpans(spans []diff.LineSpan, contextLines int, totalLines int) []diff.LineSpan {
	if len(spans) == 0 {
		return nil
	}
	expanded := make([]diff.LineSpan, 0, len(spans))
	for _, span := range spans {
		start := span.Start - contextLines
		if start < 1 {
			start = 1
		}
		end := span.End + contextLines
		if end > totalLines {
			end = totalLines
		}
		if span.Start == 0 && span.End == 0 {
			continue
		}
		expanded = append(expanded, diff.LineSpan{Start: start, End: end})
	}
	if len(expanded) == 0 {
		return nil
	}

	merged := []diff.LineSpan{expanded[0]}
	for _, span := range expanded[1:] {
		last := &merged[len(merged)-1]
		if span.Start <= last.End+1 {
			if span.End > last.End {
				last.End = span.End
			}
			continue
		}
		merged = append(merged, span)
	}
	return merged
}

func splitLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return nil
	}
	return strings.Split(content, "\n")
}

func lineCountLimit(contextLines int, total int) int {
	if contextLines <= 0 {
		if total < 20 {
			return total
		}
		return 20
	}
	limit := contextLines * 2
	if limit < 1 {
		return 1
	}
	if limit > total {
		return total
	}
	return limit
}

func summaryFromTests(testCount int) string {
	if testCount == 0 {
		return "no_tests_in_diff"
	}
	return fmt.Sprintf("tests_in_diff=%d", testCount)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
