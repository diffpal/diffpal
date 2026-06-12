package findings

import (
	"strings"
	"testing"
)

func TestSemanticChangeSummaryGroupsPathsByPurpose(t *testing.T) {
	t.Parallel()

	got := SemanticChangeSummary([]ReviewedFile{
		{Path: ".config/diffpal/config.yaml"},
		{Path: ".github/workflows/ci.yml"},
		{Path: "action.yml"},
		{Path: "docs/ci-examples.md"},
		{Path: "go.mod"},
		{Path: "internal/reviewer/engine.go"},
		{Path: "tasks/azure-devops/package.json"},
	})

	joined := strings.Join(got, "\n")
	for _, want := range []string{
		"Updated DiffPal configuration defaults and examples.",
		"Updated the GitHub Action integration for installing and running DiffPal.",
		"Updated CI workflow automation for testing, review, or release packaging.",
		"Updated Azure DevOps task packaging or pipeline integration.",
		"Updated review output generation and findings reporting behavior.",
		"Updated user-facing documentation and setup guidance.",
		"Updated Go module dependencies used by DiffPal.",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("SemanticChangeSummary() missing %q in %v", want, got)
		}
	}
	for _, unwanted := range []string{
		"Updated `action.yml`.",
		"Updated `.github/workflows/ci.yml`.",
		"Updated `docs/ci-examples.md`.",
	} {
		if strings.Contains(joined, unwanted) {
			t.Fatalf("SemanticChangeSummary() contains file-list item %q in %v", unwanted, got)
		}
	}
}
