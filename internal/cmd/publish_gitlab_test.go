package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
)

func TestPublishBundleToFilesGitLabEmitsCodeQualityAndSARIF(t *testing.T) {
	t.Parallel()

	bundle := findings.FindingsBundle{
		Version:  findings.VersionV1,
		ReviewID: "mr-1",
		BaseSHA:  "base-a",
		HeadSHA:  "head-a",
		Findings: []findings.Finding{
			{
				ID:         "fp-maint",
				ReviewID:   "mr-1",
				RuleID:     "maintainability.deadcode",
				Category:   "maintainability",
				Severity:   "medium",
				Confidence: 0.7,
				Path:       "internal/app/service.go",
				StartLine:  11,
				EndLine:    11,
				Title:      "dead branch",
				Message:    "branch is unreachable",
				Evidence:   "condition is constant false",
			},
			{
				ID:         "fp-sec",
				ReviewID:   "mr-1",
				RuleID:     "security.sql",
				Category:   "security",
				Severity:   "high",
				Confidence: 0.92,
				Path:       "internal/db/query.go",
				StartLine:  20,
				EndLine:    20,
				Title:      "unsafe SQL construction",
				Message:    "query concatenates user input",
				Evidence:   "untrusted input appended into SQL",
			},
		},
	}

	outputs, blocking, err := publishBundleToFiles("gitlab", bundle, "repo-a", "high", []string{"code-quality", "sarif"}, "")
	if err != nil {
		t.Fatalf("publishBundleToFiles() error = %v", err)
	}
	if blocking != 0 {
		t.Fatalf("blocking = %d, want 0 for artifact-only modes", blocking)
	}
	if len(outputs) != 2 {
		t.Fatalf("len(outputs) = %d, want 2", len(outputs))
	}
	for _, item := range outputs {
		raw, err := os.ReadFile(item.Path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", item.Path, err)
		}
		if filepath.Ext(item.Path) == ".sarif" && !strings.Contains(string(raw), "\"runs\"") {
			t.Fatalf("SARIF output missing runs payload:\n%s", string(raw))
		}
		if filepath.Base(item.Path) == "gl-code-quality-report.json" && !strings.Contains(string(raw), "maintainability.deadcode") {
			t.Fatalf("Code Quality output missing maintainability finding:\n%s", string(raw))
		}
	}
}
