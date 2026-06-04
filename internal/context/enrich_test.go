package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/diffpal/diffpal/internal/diff"
)

func TestEnrichBuildsNeighborContextFromChangedSpans(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")
	content := strings.Join([]string{
		"package sample",
		"",
		"func a() {}",
		"func b() {}",
		"func c() {}",
		"func d() {}",
		"func e() {}",
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	contexts, summary, err := Enrich(diff.DiffResult{
		Files: []diff.FileChange{
			{
				ToPath:           path,
				ChangedLineSpans: []diff.LineSpan{{Start: 4, End: 4}},
			},
		},
	}, 1)
	if err != nil {
		t.Fatalf("Enrich() error = %v", err)
	}
	if summary != "no_tests_in_diff" {
		t.Fatalf("summary = %q, want no_tests_in_diff", summary)
	}
	if len(contexts) != 1 {
		t.Fatalf("len(contexts) = %d, want 1", len(contexts))
	}
	snippet := contexts[0].Snippet
	if !strings.Contains(snippet, "@@ lines 3-5 @@") {
		t.Fatalf("Snippet missing expected span header:\n%s", snippet)
	}
	if !strings.Contains(snippet, "4 | func b() {}") {
		t.Fatalf("Snippet missing changed line neighborhood:\n%s", snippet)
	}
	if len(contexts[0].Spans) != 1 || contexts[0].Spans[0].Start != 3 || contexts[0].Spans[0].End != 5 {
		t.Fatalf("Spans = %+v, want 3-5", contexts[0].Spans)
	}
}

func TestChunkByLimitsRespectsFileAndCharacterLimits(t *testing.T) {
	t.Parallel()

	contexts := []FileContext{
		{Path: "a.go", Snippet: strings.Repeat("a", 10)},
		{Path: "b.go", Snippet: strings.Repeat("b", 10)},
		{Path: "c.go", Snippet: strings.Repeat("c", 10)},
	}
	chunks := ChunkByLimits(contexts, 15, 2)
	if len(chunks) != 3 {
		t.Fatalf("len(chunks) = %d, want 3", len(chunks))
	}
	if len(chunks[0].Files) != 1 || len(chunks[1].Files) != 1 || len(chunks[2].Files) != 1 {
		t.Fatalf("chunk file distribution = %+v, want one file per chunk", chunks)
	}
}

func TestChunkByLimitsUsesFileCountWhenCharsAllow(t *testing.T) {
	t.Parallel()

	contexts := []FileContext{
		{Path: "a.go", Snippet: "a"},
		{Path: "b.go", Snippet: "b"},
		{Path: "c.go", Snippet: "c"},
	}
	chunks := ChunkByLimits(contexts, 100, 2)
	if len(chunks) != 2 {
		t.Fatalf("len(chunks) = %d, want 2", len(chunks))
	}
	if len(chunks[0].Files) != 2 || len(chunks[1].Files) != 1 {
		t.Fatalf("chunk file distribution = %+v, want 2 then 1", chunks)
	}
}
