package codequality

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

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
			Fingerprint: codeQualityFingerprint(repo, f),
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

func codeQualityFingerprint(repo string, f findings.Finding) string {
	type payload struct {
		Repo      string `json:"repo"`
		Path      string `json:"path"`
		LineStart int    `json:"line_start"`
		LineEnd   int    `json:"line_end"`
		Category  string `json:"category"`
	}
	canonical := payload{
		Repo:      repo,
		Path:      strings.TrimSpace(f.Path),
		LineStart: f.StartLine,
		LineEnd:   f.EndLine,
		Category:  strings.TrimSpace(strings.ToLower(f.Category)),
	}
	raw, _ := json.Marshal(canonical)
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum[:])
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
