package findings

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	VersionV1 = "v1"
	VersionV2 = "v2"
)

var validSeverities = map[string]struct{}{
	"low":      {},
	"medium":   {},
	"high":     {},
	"critical": {},
}

type FindingsBundle struct {
	Version       string          `json:"version"`
	ReviewID      string          `json:"review_id"`
	BaseSHA       string          `json:"base_sha"`
	HeadSHA       string          `json:"head_sha"`
	Language      string          `json:"language,omitempty"`
	ReviewChecks  []string        `json:"review_checks,omitempty"`
	Prompt        *PromptMetadata `json:"prompt,omitempty"`
	ChangeSummary []string        `json:"change_summary,omitempty"`
	Files         []ReviewedFile  `json:"files,omitempty"`
	Findings      []Finding       `json:"findings"`
}

type PromptMetadata struct {
	PromptID      string `json:"prompt_id,omitempty"`
	PromptVersion string `json:"prompt_version,omitempty"`
	Purpose       string `json:"purpose,omitempty"`
	SchemaVersion string `json:"schema_version,omitempty"`
}

type ReviewedFile struct {
	Path   string `json:"path"`
	Status string `json:"status,omitempty"`
}

type Finding struct {
	ID             string          `json:"id"`
	ReviewID       string          `json:"review_id"`
	Category       string          `json:"category"`
	Severity       string          `json:"severity"`
	Confidence     float64         `json:"confidence"`
	Path           string          `json:"path"`
	StartLine      int             `json:"start_line"`
	EndLine        int             `json:"end_line"`
	ChangedSpan    LineSpan        `json:"changed_span"`
	SupportingSpan *LineSpan       `json:"supporting_span,omitempty"`
	Title          string          `json:"title"`
	Message        string          `json:"message"`
	Evidence       FindingEvidence `json:"evidence"`
	Impact         FindingImpact   `json:"impact"`
	Suggestion     string          `json:"suggestion,omitempty"`
	Blocking       bool            `json:"blocking"`
	Provider       string          `json:"provider"`
}

type LineSpan struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

type FindingEvidence struct {
	Anchor         string `json:"anchor"`
	ReasoningBasis string `json:"reasoning_basis"`
	Source         string `json:"source"`
}

func NewEvidence(text string) FindingEvidence {
	text = strings.TrimSpace(text)
	return FindingEvidence{
		Anchor:         text,
		ReasoningBasis: text,
		Source:         "changed_line",
	}
}

func (e *FindingEvidence) UnmarshalJSON(raw []byte) error {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		*e = FindingEvidence{
			Anchor:         text,
			ReasoningBasis: text,
			Source:         "legacy",
		}
		return nil
	}
	type alias FindingEvidence
	var decoded alias
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	*e = FindingEvidence(decoded)
	return nil
}

type FindingImpact struct {
	Summary string `json:"summary"`
	Scope   string `json:"scope"`
}

func NewImpact(text string) FindingImpact {
	text = strings.TrimSpace(text)
	return FindingImpact{
		Summary: text,
		Scope:   "changed behavior",
	}
}

func (i *FindingImpact) UnmarshalJSON(raw []byte) error {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		*i = FindingImpact{
			Summary: text,
			Scope:   "legacy",
		}
		return nil
	}
	type alias FindingImpact
	var decoded alias
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	*i = FindingImpact(decoded)
	return nil
}

type ValidationError struct {
	Field string
	Msg   string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Msg)
}

func Validate(bundle FindingsBundle) error {
	version := ensureVersion(bundle.Version)
	if version != VersionV1 && version != VersionV2 {
		return ValidationError{Field: "version", Msg: "must be v1 or v2"}
	}
	if bundle.ReviewID == "" {
		return ValidationError{Field: "review_id", Msg: "review_id is required"}
	}
	for _, file := range bundle.Files {
		if strings.TrimSpace(file.Path) == "" {
			return ValidationError{Field: "file.path", Msg: "path is required"}
		}
	}
	for _, f := range bundle.Findings {
		if strings.TrimSpace(f.Path) == "" {
			return ValidationError{Field: "finding.path", Msg: "path is required"}
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
		if version == VersionV2 {
			if err := validateLineSpan("finding.changed_span", f.ChangedSpan, true); err != nil {
				return err
			}
			if f.SupportingSpan != nil {
				if err := validateLineSpan("finding.supporting_span", *f.SupportingSpan, false); err != nil {
					return err
				}
			}
			if strings.TrimSpace(f.Evidence.Anchor) == "" {
				return ValidationError{Field: "finding.evidence.anchor", Msg: "anchor is required"}
			}
			if strings.TrimSpace(f.Evidence.ReasoningBasis) == "" {
				return ValidationError{Field: "finding.evidence.reasoning_basis", Msg: "reasoning_basis is required"}
			}
			if strings.TrimSpace(f.Evidence.Source) == "" {
				return ValidationError{Field: "finding.evidence.source", Msg: "source is required"}
			}
			if strings.TrimSpace(f.Impact.Summary) == "" {
				return ValidationError{Field: "finding.impact.summary", Msg: "summary is required"}
			}
			if strings.TrimSpace(f.Impact.Scope) == "" {
				return ValidationError{Field: "finding.impact.scope", Msg: "scope is required"}
			}
		} else {
			if strings.TrimSpace(f.EvidenceText()) == "" {
				return ValidationError{Field: "finding.evidence", Msg: "evidence is required"}
			}
		}
		if f.StartLine <= 0 || f.EndLine <= 0 {
			return ValidationError{Field: "finding.line", Msg: "line numbers must be positive"}
		}
		if f.StartLine > f.EndLine {
			return ValidationError{Field: "finding.line", Msg: "start_line must be <= end_line"}
		}
		if f.Confidence < 0 || f.Confidence > 1 {
			return ValidationError{Field: "finding.confidence", Msg: "confidence must be between 0 and 1"}
		}
	}
	return nil
}

func validateLineSpan(field string, span LineSpan, requirePath bool) error {
	if requirePath && strings.TrimSpace(span.Path) == "" {
		return ValidationError{Field: field + ".path", Msg: "path is required"}
	}
	if span.StartLine <= 0 || span.EndLine <= 0 {
		return ValidationError{Field: field, Msg: "line numbers must be positive"}
	}
	if span.StartLine > span.EndLine {
		return ValidationError{Field: field, Msg: "start_line must be <= end_line"}
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
		if f.ChangedSpan.Path == "" && f.StartLine > 0 && f.EndLine > 0 {
			f.ChangedSpan = LineSpan{Path: f.Path, StartLine: f.StartLine, EndLine: f.EndLine}
		}
		if f.StartLine == 0 && f.ChangedSpan.StartLine > 0 {
			f.StartLine = f.ChangedSpan.StartLine
		}
		if f.EndLine == 0 && f.ChangedSpan.EndLine > 0 {
			f.EndLine = f.ChangedSpan.EndLine
		}
		if f.Path == "" && f.ChangedSpan.Path != "" {
			f.Path = f.ChangedSpan.Path
		}
		f.ID = Fingerprint(repo, bundle.HeadSHA, *f)
	}
}

func (f Finding) EvidenceText() string {
	parts := make([]string, 0, 3)
	if text := strings.TrimSpace(f.Evidence.Anchor); text != "" {
		parts = append(parts, text)
	}
	if text := strings.TrimSpace(f.Evidence.ReasoningBasis); text != "" {
		parts = append(parts, text)
	}
	if text := strings.TrimSpace(f.Evidence.Source); text != "" {
		parts = append(parts, text)
	}
	return strings.Join(parts, " ")
}

func (f Finding) ImpactText() string {
	summary := strings.TrimSpace(f.Impact.Summary)
	scope := strings.TrimSpace(f.Impact.Scope)
	if scope == "legacy" || scope == "changed behavior" {
		scope = ""
	}
	switch {
	case summary != "" && scope != "":
		return summary + " Scope: " + scope
	case summary != "":
		return summary
	default:
		return scope
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
		Category    string `json:"category"`
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
		Category:    strings.TrimSpace(strings.ToLower(f.Category)),
		MessageNorm: normalizeMessage(f.Message),
		Evidence:    shaText(f.EvidenceText()),
	}
	jsonBytes, _ := json.Marshal(canonical)
	sum := sha256.Sum256(jsonBytes)
	return fmt.Sprintf("%x", sum[:])
}

func normalizePath(v string) string {
	return strings.TrimSpace(v)
}

func normalizeMessage(msg string) string {
	return strings.TrimSpace(strings.ToLower(msg))
}

func shaText(v string) string {
	sum := sha256.Sum256([]byte(v))
	return fmt.Sprintf("%x", sum[:])
}
