package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
)

func TestPublishBundleToFilesGitLabEmitsCodeQualityAndSARIF(t *testing.T) {
	bundle := findings.FindingsBundle{
		Version:  findings.VersionV1,
		ReviewID: "mr-1",
		BaseSHA:  "base-a",
		HeadSHA:  "head-a",
		Findings: []findings.Finding{
			{
				ID:         "fp-maint",
				ReviewID:   "mr-1",
				Category:   "maintainability",
				Severity:   "medium",
				Confidence: 0.7,
				Path:       "internal/app/service.go",
				StartLine:  11,
				EndLine:    11,
				Title:      "dead branch",
				Message:    "branch is unreachable",
				Evidence:   findings.NewEvidence("condition is constant false"),
			},
			{
				ID:         "fp-sec",
				ReviewID:   "mr-1",
				Category:   "security",
				Severity:   "high",
				Confidence: 0.92,
				Path:       "internal/db/query.go",
				StartLine:  20,
				EndLine:    20,
				Title:      "unsafe SQL construction",
				Message:    "query concatenates user input",
				Evidence:   findings.NewEvidence("untrusted input appended into SQL"),
			},
		},
	}

	outputs, blocking, err := publishBundleToFiles("gitlab", bundle, "repo-a", "high", false, "summary", true, "", "")
	if err != nil {
		t.Fatalf("publishBundleToFiles() error = %v", err)
	}
	if blocking != 1 {
		t.Fatalf("blocking = %d, want 1 with GitLab status surface", blocking)
	}
	if len(outputs) != 4 {
		t.Fatalf("len(outputs) = %d, want 4", len(outputs))
	}
	for _, item := range outputs {
		raw, err := os.ReadFile(item.Path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", item.Path, err)
		}
		if filepath.Ext(item.Path) == ".sarif" && !strings.Contains(string(raw), "\"runs\"") {
			t.Fatalf("SARIF output missing runs payload:\n%s", string(raw))
		}
		if filepath.Base(item.Path) == "gl-code-quality-report.json" && !strings.Contains(string(raw), "maintainability") {
			t.Fatalf("Code Quality output missing maintainability finding:\n%s", string(raw))
		}
	}
}

func TestPublishBundleToFilesGitHubEmbedsPermanentLinks(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("GITHUB_REPOSITORY", "acme/diffpal")
	sourcePath := filepath.Join(dir, "internal", "db", "query.go")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	source := strings.Join([]string{
		"package db",
		"",
		"func deleteSessions(user string) {",
		"    _, _ = db.Exec(\"DELETE FROM sessions WHERE user = '\" + user + \"'\")",
		"}",
	}, "\n")
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	bundle := findings.FindingsBundle{
		Version:  findings.VersionV1,
		ReviewID: "github-pr-1",
		BaseSHA:  "base-a",
		HeadSHA:  "head-a",
		Files: []findings.ReviewedFile{
			{Path: "internal/db/query.go"},
		},
		Findings: []findings.Finding{
			{
				ID:         "fp-sec",
				ReviewID:   "github-pr-1",
				Category:   "security",
				Severity:   "high",
				Confidence: 0.96,
				Path:       "internal/db/query.go",
				StartLine:  3,
				EndLine:    5,
				Title:      "unsafe SQL construction",
				Message:    "query concatenates user input",
				Evidence:   findings.NewEvidence("user is appended into SQL"),
				Suggestion: "Use a parameterized statement.",
				Blocking:   true,
			},
		},
	}

	outputs, blocking, err := publishBundleToFiles("github", bundle, "repo-a", "high", false, "review", true, "", "")
	if err != nil {
		t.Fatalf("publishBundleToFiles() error = %v", err)
	}
	if blocking != 1 {
		t.Fatalf("blocking = %d, want 1", blocking)
	}
	if len(outputs) != 3 {
		t.Fatalf("outputs = %d, want 3", len(outputs))
	}
	path := filepath.Join(".artifacts", "diffpal", "github-comments.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	text := string(raw)
	if !strings.Contains(text, "https://github.com/acme/diffpal/blob/head-a/internal/db/query.go#L3-L5") {
		t.Fatalf("%s missing GitHub permanent link:\n%s", path, text)
	}
	if strings.Contains(text, "```") || strings.Contains(text, "func deleteSessions(user string)") {
		t.Fatalf("%s contains fenced or embedded code snippet:\n%s", path, text)
	}
	if !strings.Contains(text, "Use a parameterized statement.") {
		t.Fatalf("%s missing suggestion:\n%s", path, text)
	}
}

func TestPublishBundleToFilesGitHubCommentsReportsBlocking(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("GITHUB_EVENT_PATH", "")
	t.Setenv("GITHUB_REPOSITORY", "acme/diffpal")
	t.Setenv("GITHUB_BASE_SHA", "base-a")
	t.Setenv("GITHUB_HEAD_SHA", "head-a")

	bundle := findings.FindingsBundle{
		Version:  findings.VersionV1,
		ReviewID: "github-pr-1",
		BaseSHA:  "base-a",
		HeadSHA:  "head-a",
		Files: []findings.ReviewedFile{
			{Path: "internal/db/query.go"},
		},
		Findings: []findings.Finding{
			{
				ID:         "fp-sec",
				ReviewID:   "github-pr-1",
				Category:   "security",
				Severity:   "high",
				Confidence: 0.96,
				Path:       "internal/db/query.go",
				StartLine:  3,
				EndLine:    5,
				Title:      "unsafe SQL construction",
				Message:    "query concatenates user input",
				Evidence:   findings.NewEvidence("user is appended into SQL"),
				Suggestion: "Use a parameterized statement.",
				Blocking:   true,
			},
		},
	}

	_, blocking, err := publishBundleToFiles("github", bundle, "repo-a", "high", false, "review", true, "", "")
	if err != nil {
		t.Fatalf("publishBundleToFiles() error = %v", err)
	}
	if blocking != 1 {
		t.Fatalf("blocking = %d, want 1", blocking)
	}
}

func TestPublishBundleToFilesGitHubCommentsIncludeAdvisoryFindings(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("GITHUB_EVENT_PATH", "")
	t.Setenv("GITHUB_REPOSITORY", "acme/diffpal")
	t.Setenv("GITHUB_BASE_SHA", "base-a")
	t.Setenv("GITHUB_HEAD_SHA", "head-a")

	bundle := findings.FindingsBundle{
		Version:  findings.VersionV1,
		ReviewID: "github-pr-1",
		BaseSHA:  "base-a",
		HeadSHA:  "head-a",
		Findings: []findings.Finding{{
			ID:         "fp-medium",
			ReviewID:   "github-pr-1",
			Category:   "correctness",
			Severity:   "medium",
			Confidence: 0.95,
			Path:       "internal/db/query.go",
			StartLine:  3,
			EndLine:    3,
			Title:      "advisory",
			Message:    "medium advisory",
			Evidence:   findings.NewEvidence("medium evidence"),
		}, {
			ID:         "fp-high",
			ReviewID:   "github-pr-1",
			Category:   "security",
			Severity:   "high",
			Confidence: 0.95,
			Path:       "internal/db/query.go",
			StartLine:  5,
			EndLine:    5,
			Title:      "blocking",
			Message:    "high finding",
			Evidence:   findings.NewEvidence("high evidence"),
		}},
	}

	_, blocking, err := publishBundleToFiles("github", bundle, "repo-a", "high", false, "review", true, "", "")
	if err != nil {
		t.Fatalf("publishBundleToFiles() error = %v", err)
	}
	if blocking != 1 {
		t.Fatalf("blocking = %d, want 1", blocking)
	}
	raw, err := os.ReadFile(filepath.Join(".artifacts", "diffpal", "github-comments.json"))
	if err != nil {
		t.Fatalf("ReadFile(github-comments.json) error = %v", err)
	}
	text := string(raw)
	for _, needle := range []string{"fp-medium", "medium advisory", "fp-high", "high finding"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("github comments missing %q:\n%s", needle, text)
		}
	}
}

func TestPublishBundleToFilesNormalizesBlockingFromBlockOn(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("GITHUB_EVENT_PATH", "")
	t.Setenv("GITHUB_REPOSITORY", "acme/diffpal")
	t.Setenv("GITHUB_BASE_SHA", "base-a")
	t.Setenv("GITHUB_HEAD_SHA", "head-a")

	bundle := findings.FindingsBundle{
		Version:  findings.VersionV1,
		ReviewID: "github-pr-1",
		BaseSHA:  "base-a",
		HeadSHA:  "head-a",
		Findings: []findings.Finding{{
			ID:         "fp-medium-stale-blocking",
			ReviewID:   "github-pr-1",
			Category:   "correctness",
			Severity:   "medium",
			Confidence: 0.95,
			Path:       "internal/db/query.go",
			StartLine:  3,
			EndLine:    3,
			Title:      "advisory",
			Message:    "medium advisory",
			Evidence:   findings.NewEvidence("medium evidence"),
			Blocking:   true,
		}, {
			ID:         "fp-high-stale-advisory",
			ReviewID:   "github-pr-1",
			Category:   "security",
			Severity:   "high",
			Confidence: 0.95,
			Path:       "internal/db/query.go",
			StartLine:  5,
			EndLine:    5,
			Title:      "blocking",
			Message:    "high finding",
			Evidence:   findings.NewEvidence("high evidence"),
			Blocking:   false,
		}},
	}

	_, blocking, err := publishBundleToFiles("github", bundle, "repo-a", "high", false, "review", true, "", "")
	if err != nil {
		t.Fatalf("publishBundleToFiles() error = %v", err)
	}
	if blocking != 1 {
		t.Fatalf("blocking = %d, want 1 after block_on normalization", blocking)
	}
	raw, err := os.ReadFile(filepath.Join(".artifacts", "diffpal", "github-comments.json"))
	if err != nil {
		t.Fatalf("ReadFile(github-comments.json) error = %v", err)
	}
	text := string(raw)
	for _, needle := range []string{"fp-medium-stale-blocking", "fp-high-stale-advisory"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("github comments missing %q after normalization:\n%s", needle, text)
		}
	}
}

func TestPublishBundleToFilesRejectsSingleOutputForMultipleSurfaces(t *testing.T) {
	_, _, err := publishBundleToFiles("github", findings.FindingsBundle{ReviewID: "github-pr-1"}, "repo-a", "high", false, "review", true, "review.out", "")
	if err == nil {
		t.Fatal("publishBundleToFiles() error = nil, want single-output multi-surface error")
	}
	if !strings.Contains(err.Error(), "--out cannot be used when feedback publishes multiple surfaces") {
		t.Fatalf("unexpected error: %v", err)
	}
}
