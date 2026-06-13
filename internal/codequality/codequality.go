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
	occurrences := map[string]int{}
	for _, f := range bundle.Findings {
		if f.Category != "maintainability" {
			continue
		}
		identity := codeQualityIdentity(f)
		occurrences[identity]++
		out = append(out, Finding{
			Description: fmt.Sprintf("[%s] %s", f.Category, f.Message),
			CheckName:   f.Category,
			Severity:    mapSeverity(f.Severity),
			Fingerprint: codeQualityFingerprint(repo, f, occurrences[identity]),
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

func codeQualityFingerprint(repo string, f findings.Finding, occurrence int) string {
	type payload struct {
		Repo      string `json:"repo"`
		FindingID string `json:"finding_id,omitempty"`
		Path      string `json:"path"`
		LineStart int    `json:"line_start"`
		LineEnd   int    `json:"line_end"`
		Category  string `json:"category"`
		Slot      int    `json:"slot"`
	}
	canonical := payload{
		Repo:      repo,
		FindingID: strings.TrimSpace(f.ID),
		Path:      strings.TrimSpace(f.Path),
		LineStart: f.StartLine,
		LineEnd:   f.EndLine,
		Category:  strings.TrimSpace(strings.ToLower(f.Category)),
		Slot:      occurrence,
	}
	raw, _ := json.Marshal(canonical)
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum[:])
}

func codeQualityIdentity(f findings.Finding) string {
	if strings.TrimSpace(f.ID) != "" {
		return strings.TrimSpace(f.ID)
	}
	return fmt.Sprintf("%s:%d:%d:%s",
		strings.TrimSpace(f.Path),
		f.StartLine,
		f.EndLine,
		strings.TrimSpace(strings.ToLower(f.Category)),
	)
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
