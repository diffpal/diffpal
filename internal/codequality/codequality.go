package codequality

import (
	"encoding/json"
	"fmt"

	"github.com/diffpal/diffpal/internal/findings"
)

type Finding struct {
	Description string       `json:"description"`
	CheckName   string       `json:"check_name"`
	Severity    string       `json:"severity"`
	Fingerprint string       `json:"fingerprint"`
	Location    CodeLocation `json:"location"`
}

type CodeLocation struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

func Convert(bundle findings.FindingsBundle, repo string) ([]Finding, error) {
	out := make([]Finding, 0, len(bundle.Findings))
	for _, f := range bundle.Findings {
		if f.Category != "maintainability" {
			continue
		}
		out = append(out, Finding{
			Description: fmt.Sprintf("[%s] %s", f.Category, f.Message),
			CheckName:   f.RuleID,
			Severity:    mapSeverity(f.Severity),
			Fingerprint: findings.Fingerprint(repo, bundle.HeadSHA, f),
			Location: CodeLocation{
				Path:      f.Path,
				StartLine: f.StartLine,
				EndLine:   f.EndLine,
			},
		})
	}
	return out, nil
}

func ToJSON(bundle findings.FindingsBundle, repo string) ([]byte, error) {
	items, err := Convert(bundle, repo)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(items, "", "  ")
}

func mapSeverity(v string) string {
	switch v {
	case "critical", "high":
		return "high"
	case "medium":
		return "medium"
	default:
		return "low"
	}
}
