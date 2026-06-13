package codequality

import (
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
	if items[0].Severity != "medium" {
		t.Fatalf("Severity = %q, want medium", items[0].Severity)
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
