package findings

import "strings"

// SemanticChangeSummary derives a concise purpose-oriented overview from reviewed paths.
// It is intentionally coarser than a file list so summary reports remain useful when
// the model does not return change_summary.
func SemanticChangeSummary(files []ReviewedFile) []string {
	if len(files) == 0 {
		return nil
	}
	areas := map[string]bool{}
	for _, file := range files {
		path := normalizeOverviewPath(file.Path)
		if path == "" {
			continue
		}
		switch {
		case path == "action.yml" || strings.HasPrefix(path, "tasks/github/"):
			areas["github-action"] = true
		case strings.HasPrefix(path, ".github/workflows/"):
			areas["ci"] = true
		case strings.HasPrefix(path, "tasks/azure-devops/"):
			areas["azure"] = true
		case path == ".config/diffpal/config.yaml" || strings.HasPrefix(path, "internal/config/"):
			areas["config"] = true
		case path == "go.mod" || path == "go.sum":
			areas["dependencies"] = true
		case path == "README.md" || strings.HasPrefix(path, "docs/"):
			areas["docs"] = true
		case strings.HasPrefix(path, "internal/reviewer/") ||
			strings.HasPrefix(path, "internal/markdown/") ||
			strings.HasPrefix(path, "internal/findings/"):
			areas["review-output"] = true
		case strings.HasPrefix(path, "internal/cmd/") || strings.HasPrefix(path, "cmd/"):
			areas["cli"] = true
		case path == ".gitignore":
			areas["repo-maintenance"] = true
		default:
			areas["implementation"] = true
		}
	}

	order := []string{
		"config",
		"github-action",
		"ci",
		"azure",
		"review-output",
		"cli",
		"docs",
		"dependencies",
		"repo-maintenance",
		"implementation",
	}
	text := map[string]string{
		"config":           "Updated DiffPal configuration defaults and examples.",
		"github-action":    "Updated the GitHub Action integration for installing and running DiffPal.",
		"ci":               "Updated CI workflow automation for testing, review, or release packaging.",
		"azure":            "Updated Azure DevOps task packaging or pipeline integration.",
		"review-output":    "Updated review output generation and findings reporting behavior.",
		"cli":              "Updated CLI review or publish command behavior.",
		"docs":             "Updated user-facing documentation and setup guidance.",
		"dependencies":     "Updated Go module dependencies used by DiffPal.",
		"repo-maintenance": "Updated repository housekeeping for generated or local artifacts.",
		"implementation":   "Updated DiffPal implementation files.",
	}

	out := make([]string, 0, len(areas))
	for _, key := range order {
		if areas[key] {
			out = append(out, text[key])
		}
	}
	if len(out) == 0 {
		return nil
	}
	const maxSummaryItems = 8
	if len(out) > maxSummaryItems {
		out = out[:maxSummaryItems]
	}
	return out
}

func normalizeOverviewPath(path string) string {
	path = strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	path = strings.TrimPrefix(path, "./")
	return path
}
