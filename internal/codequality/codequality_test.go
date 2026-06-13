package codequality

import (
	"encoding/json"
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
)

func TestConvertExportsOnlyMaintainabilityFindingsWithDeterministicFingerprint(t *testing.T) {
	t.Parallel()

	bundle := findings.FindingsBundle{
		HeadSHA: "head-1",
		Findings: []findings.Finding{
			{
				ReviewID:  "review-1",
				Category:  "maintainability",
				Severity:  "medium",
				Path:      "internal/app/service.go",
				StartLine: 14,
				EndLine:   14,
				Message:   "branch is unreachable",
			},
			{
				ReviewID:  "review-1",
				Category:  "security",
				Severity:  "high",
				Path:      "internal/db/query.go",
				StartLine: 20,
				EndLine:   20,
				Message:   "query concatenates input",
			},
		},
	}

	items, err := Convert(bundle, "repo-a")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Convert() items = %d, want 1", len(items))
	}
	if items[0].CheckName != "maintainability" {
		t.Fatalf("CheckName = %q, want maintainability", items[0].CheckName)
	}
	if items[0].Severity != "minor" {
		t.Fatalf("Severity = %q, want minor", items[0].Severity)
	}
	if items[0].Location.Path != "internal/app/service.go" {
		t.Fatalf("Location.Path = %q", items[0].Location.Path)
	}
	if items[0].Location.Lines.Begin != 14 {
		t.Fatalf("Location.Lines.Begin = %d, want 14", items[0].Location.Lines.Begin)
	}
	if items[0].Fingerprint == "" {
		t.Fatal("Fingerprint = empty, want deterministic fingerprint")
	}

	again, err := Convert(bundle, "repo-a")
	if err != nil {
		t.Fatalf("Convert() second call error = %v", err)
	}
	if items[0].Fingerprint != again[0].Fingerprint {
		t.Fatalf("Fingerprint mismatch across reruns: %q != %q", items[0].Fingerprint, again[0].Fingerprint)
	}
}

func TestToJSONUsesGitLabCodeQualityLocationShape(t *testing.T) {
	t.Parallel()

	raw, err := ToJSON(findings.FindingsBundle{
		HeadSHA: "head-1",
		Findings: []findings.Finding{{
			ReviewID:  "review-1",
			Category:  "maintainability",
			Severity:  "high",
			Path:      "internal/app/service.go",
			StartLine: 14,
			EndLine:   20,
			Message:   "branch is unreachable",
		}},
	}, "repo-a")
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	var payload []map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if len(payload) != 1 {
		t.Fatalf("len(payload) = %d, want 1: %s", len(payload), raw)
	}
	location, ok := payload[0]["location"].(map[string]any)
	if !ok {
		t.Fatalf("location missing or wrong type: %s", raw)
	}
	if _, ok := location["start_line"]; ok {
		t.Fatalf("location contains legacy start_line: %s", raw)
	}
	lines, ok := location["lines"].(map[string]any)
	if !ok {
		t.Fatalf("location.lines missing or wrong type: %s", raw)
	}
	if lines["begin"] != float64(14) {
		t.Fatalf("location.lines.begin = %v, want 14", lines["begin"])
	}
	if payload[0]["severity"] != "major" {
		t.Fatalf("severity = %v, want major", payload[0]["severity"])
	}
}

func TestConvertFingerprintDoesNotDependOnHeadSHA(t *testing.T) {
	t.Parallel()

	finding := findings.Finding{
		Category:  "maintainability",
		Severity:  "medium",
		Path:      "internal/app/service.go",
		StartLine: 14,
		EndLine:   16,
		Title:     "Long helper",
		Message:   "helper is difficult to follow",
		Evidence:  "nested branches",
	}
	first, err := Convert(findings.FindingsBundle{HeadSHA: "head-a", Findings: []findings.Finding{finding}}, "repo")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	second, err := Convert(findings.FindingsBundle{HeadSHA: "head-b", Findings: []findings.Finding{finding}}, "repo")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if first[0].Fingerprint != second[0].Fingerprint {
		t.Fatalf("fingerprint changed with head SHA: %q vs %q", first[0].Fingerprint, second[0].Fingerprint)
	}
}

func TestConvertFingerprintDoesNotDependOnFindingWording(t *testing.T) {
	t.Parallel()

	firstFinding := findings.Finding{
		Category:  "maintainability",
		Severity:  "medium",
		Path:      "internal/app/service.go",
		StartLine: 14,
		EndLine:   16,
		Title:     "Long helper",
		Message:   "helper is difficult to follow",
		Evidence:  "nested branches",
	}
	secondFinding := firstFinding
	secondFinding.Title = "Complex helper"
	secondFinding.Message = "helper has too much branching"
	secondFinding.Evidence = "different wording"

	first, err := Convert(findings.FindingsBundle{Findings: []findings.Finding{firstFinding}}, "repo")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	second, err := Convert(findings.FindingsBundle{Findings: []findings.Finding{secondFinding}}, "repo")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if first[0].Fingerprint != second[0].Fingerprint {
		t.Fatalf("fingerprint changed with wording: %q vs %q", first[0].Fingerprint, second[0].Fingerprint)
	}
}

func TestConvertFingerprintDistinguishesSameLineFindings(t *testing.T) {
	t.Parallel()

	items, err := Convert(findings.FindingsBundle{Findings: []findings.Finding{
		{
			Category:  "maintainability",
			Severity:  "medium",
			Path:      "internal/app/service.go",
			StartLine: 14,
			EndLine:   16,
			Message:   "first issue",
		},
		{
			Category:  "maintainability",
			Severity:  "medium",
			Path:      "internal/app/service.go",
			StartLine: 14,
			EndLine:   16,
			Message:   "second issue",
		},
	}}, "repo")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	if items[0].Fingerprint == items[1].Fingerprint {
		t.Fatalf("same-line findings share fingerprint %q", items[0].Fingerprint)
	}
}
