package findings

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

const VersionV1 = "v1"

var validSeverities = map[string]struct{}{
	"low":      {},
	"medium":   {},
	"high":     {},
	"critical": {},
}

type FindingsBundle struct {
	Version  string    `json:"version"`
	ReviewID string    `json:"review_id"`
	BaseSHA  string    `json:"base_sha"`
	HeadSHA  string    `json:"head_sha"`
	Findings []Finding `json:"findings"`
}

type Finding struct {
	ID         string  `json:"id"`
	ReviewID   string  `json:"review_id"`
	RuleID     string  `json:"rule_id"`
	Category   string  `json:"category"`
	Severity   string  `json:"severity"`
	Confidence float64 `json:"confidence"`
	Path       string  `json:"path"`
	StartLine  int     `json:"start_line"`
	EndLine    int     `json:"end_line"`
	Title      string  `json:"title"`
	Message    string  `json:"message"`
	Evidence   string  `json:"evidence"`
	Suggestion string  `json:"suggestion,omitempty"`
	Blocking   bool    `json:"blocking"`
	Provider   string  `json:"provider"`
}

type ValidationError struct {
	Field string
	Msg   string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Msg)
}

func Validate(bundle FindingsBundle) error {
	if bundle.Version != "" && bundle.Version != VersionV1 {
		return ValidationError{Field: "version", Msg: "must be v1"}
	}
	if bundle.ReviewID == "" {
		return ValidationError{Field: "review_id", Msg: "review_id is required"}
	}
	for _, f := range bundle.Findings {
		if f.Path == "" {
			return ValidationError{Field: "finding.path", Msg: "path is required"}
		}
		if f.RuleID == "" {
			return ValidationError{Field: "finding.rule_id", Msg: "rule_id is required"}
		}
		if f.Category == "" {
			return ValidationError{Field: "finding.category", Msg: "category is required"}
		}
		if f.Severity == "" {
			return ValidationError{Field: "finding.severity", Msg: "severity is required"}
		}
		if _, ok := validSeverities[strings.ToLower(strings.TrimSpace(f.Severity))]; !ok {
			return ValidationError{Field: "finding.severity", Msg: "severity must be low|medium|high|critical"}
		}
		if f.Title == "" {
			return ValidationError{Field: "finding.title", Msg: "title is required"}
		}
		if f.Message == "" {
			return ValidationError{Field: "finding.message", Msg: "message is required"}
		}
		if f.Evidence == "" {
			return ValidationError{Field: "finding.evidence", Msg: "evidence is required"}
		}
		if f.StartLine < 0 || f.EndLine < 0 {
			return ValidationError{Field: "finding.line", Msg: "line numbers must be non-negative"}
		}
		if f.EndLine > 0 && f.StartLine > f.EndLine {
			return ValidationError{Field: "finding.line", Msg: "start_line must be <= end_line"}
		}
		if f.Confidence < 0 || f.Confidence > 1 {
			return ValidationError{Field: "finding.confidence", Msg: "confidence must be between 0 and 1"}
		}
	}
	return nil
}

func Normalize(bundle *FindingsBundle, repo string) {
	for i := range bundle.Findings {
		f := &bundle.Findings[i]
		if f.ReviewID == "" {
			f.ReviewID = bundle.ReviewID
		}
		f.Severity = strings.ToLower(strings.TrimSpace(f.Severity))
		f.ID = Fingerprint(repo, bundle.HeadSHA, *f)
	}
}

func Fingerprint(repo string, headSHA string, f Finding) string {
	type payload struct {
		Platform    string `json:"platform"`
		Repo        string `json:"repo"`
		ReviewID    string `json:"review_id"`
		HeadSHA     string `json:"head_sha"`
		Path        string `json:"path"`
		LineStart   int    `json:"line_start"`
		LineEnd     int    `json:"line_end"`
		RuleID      string `json:"rule_id"`
		MessageNorm string `json:"message_norm"`
		Evidence    string `json:"evidence"`
	}
	canonical := payload{
		Platform:    "diffpal",
		Repo:        repo,
		ReviewID:    f.ReviewID,
		HeadSHA:     headSHA,
		Path:        normalizePath(f.Path),
		LineStart:   f.StartLine,
		LineEnd:     f.EndLine,
		RuleID:      f.RuleID,
		MessageNorm: normalizeMessage(f.Message),
		Evidence:    shaText(f.Evidence),
	}
	jsonBytes, _ := json.Marshal(canonical)
	sum := sha256.Sum256(jsonBytes)
	return fmt.Sprintf("%x", sum[:])
}

func normalizePath(v string) string {
	return strings.TrimSpace(strings.ToLower(v))
}

func normalizeMessage(msg string) string {
	return strings.TrimSpace(strings.ToLower(msg))
}

func shaText(v string) string {
	sum := sha256.Sum256([]byte(v))
	return fmt.Sprintf("%x", sum[:])
}
