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
	Path  string            `json:"path"`
	Lines CodeLocationLines `json:"lines"`
}

type CodeLocationLines struct {
	Begin int `json:"begin"`
}

func Convert(bundle findings.FindingsBundle, repo string) ([]Finding, error) {
	out := make([]Finding, 0, len(bundle.Findings))
	for _, f := range bundle.Findings {
		if f.Category != "maintainability" {
			continue
		}
		out = append(out, Finding{
			Description: fmt.Sprintf("[%s] %s", f.Category, f.Message),
			CheckName:   f.Category,
			Severity:    mapSeverity(f.Severity),
			Fingerprint: findings.Fingerprint(repo, bundle.HeadSHA, f),
			Location: CodeLocation{
				Path: f.Path,
				Lines: CodeLocationLines{
					Begin: f.StartLine,
				},
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
	case "critical":
		return "critical"
	case "high":
		return "major"
	case "medium":
		return "minor"
	default:
		return "info"
	}
}
