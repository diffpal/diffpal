package sarif

import (
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
)

func TestToReportMapsStableRulesAndLocations(t *testing.T) {
	t.Parallel()

	bundle := findings.FindingsBundle{
		Findings: []findings.Finding{
			{
				ID:         "fp-1",
				RuleID:     "security.sql",
				Category:   "security",
				Severity:   "critical",
				Confidence: 0.9,
				Path:       "internal/db/query.go",
				StartLine:  12,
				EndLine:    13,
				Title:      "unsafe SQL construction",
				Message:    "query concatenates user input",
				Blocking:   true,
			},
			{
				ID:         "fp-2",
				RuleID:     "correctness.nil",
				Category:   "correctness",
				Severity:   "medium",
				Confidence: 0.6,
				Path:       "internal/app/service.go",
				StartLine:  7,
				EndLine:    7,
				Title:      "possible nil dereference",
				Message:    "service client may be nil",
			},
			{
				ID:         "fp-3",
				RuleID:     "security.sql",
				Category:   "security",
				Severity:   "high",
				Confidence: 0.8,
				Path:       "internal/db/reader.go",
				StartLine:  21,
				EndLine:    21,
				Title:      "unsafe SQL construction",
				Message:    "second SQL sink",
				Blocking:   true,
			},
		},
	}

	report := ToReport(bundle)

	if report.Version != "2.1.0" {
		t.Fatalf("Version = %q, want 2.1.0", report.Version)
	}
	if len(report.Runs) != 1 {
		t.Fatalf("Runs = %d, want 1", len(report.Runs))
	}
	run := report.Runs[0]
	if len(run.Tool.Driver.Rules) != 2 {
		t.Fatalf("Rules = %d, want 2 unique rules", len(run.Tool.Driver.Rules))
	}
	if run.Tool.Driver.Rules[0].ID != "correctness.nil" || run.Tool.Driver.Rules[1].ID != "security.sql" {
		t.Fatalf("unexpected rule ordering: %+v", run.Tool.Driver.Rules)
	}
	if len(run.Results) != 3 {
		t.Fatalf("Results = %d, want 3", len(run.Results))
	}
	first := run.Results[0]
	if first.RuleID != "security.sql" {
		t.Fatalf("first RuleID = %q, want security.sql", first.RuleID)
	}
	if first.Level != "error" {
		t.Fatalf("first Level = %q, want error", first.Level)
	}
	if first.Locations[0].PhysicalLocation.ArtifactLocation.URI != "internal/db/query.go" {
		t.Fatalf("first location URI = %q", first.Locations[0].PhysicalLocation.ArtifactLocation.URI)
	}
	if first.Locations[0].PhysicalLocation.Region.StartLine != 12 || first.Locations[0].PhysicalLocation.Region.EndLine != 13 {
		t.Fatalf("unexpected region: %+v", first.Locations[0].PhysicalLocation.Region)
	}
	if got := first.PartialFingerprints["diffpalFingerprint"]; got != "fp-1" {
		t.Fatalf("partial fingerprint = %q, want fp-1", got)
	}
	if first.Properties.Category != "security" || !first.Properties.Blocking {
		t.Fatalf("unexpected result properties: %+v", first.Properties)
	}
}
