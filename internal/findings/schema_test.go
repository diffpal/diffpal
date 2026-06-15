package findings

import "testing"

func TestNormalizeInheritsReviewIDAndStableFingerprint(t *testing.T) {
	t.Parallel()

	bundle := FindingsBundle{
		Version:  VersionV1,
		ReviewID: "review-1",
		HeadSHA:  "head-a",
		Findings: []Finding{
			{
				Category:   "correctness",
				Severity:   "HIGH",
				Confidence: 0.9,
				Path:       "internal/app/main.go",
				StartLine:  10,
				EndLine:    10,
				Title:      "nil access",
				Message:    "possible nil dereference",
				Evidence:   "cfg.Client may be nil",
			},
		},
	}

	Normalize(&bundle, "repo-a")
	first := bundle.Findings[0]
	if first.ReviewID != "review-1" {
		t.Fatalf("ReviewID = %q, want review-1", first.ReviewID)
	}
	if first.Severity != "high" {
		t.Fatalf("Severity = %q, want high", first.Severity)
	}

	secondBundle := bundle
	secondBundle.Findings = []Finding{{
		Category:   "correctness",
		Severity:   "high",
		Confidence: 0.9,
		Path:       "internal/app/main.go",
		StartLine:  10,
		EndLine:    10,
		Title:      "nil access",
		Message:    "possible nil dereference",
		Evidence:   "cfg.Client may be nil",
	}}
	Normalize(&secondBundle, "repo-a")
	if first.ID != secondBundle.Findings[0].ID {
		t.Fatalf("fingerprint mismatch across reruns: %q != %q", first.ID, secondBundle.Findings[0].ID)
	}

	thirdBundle := secondBundle
	thirdBundle.HeadSHA = "head-b"
	Normalize(&thirdBundle, "repo-a")
	if first.ID == thirdBundle.Findings[0].ID {
		t.Fatal("fingerprint unchanged across head sha change, want different ID")
	}
}

func TestValidateRejectsInvalidFindingShapes(t *testing.T) {
	t.Parallel()

	cases := []FindingsBundle{
		{
			Version:  VersionV1,
			ReviewID: "review",
			Findings: []Finding{{
				Category:  "c",
				Severity:  "urgent",
				Path:      "x.go",
				StartLine: 4,
				EndLine:   4,
				Title:     "t",
				Message:   "m",
				Evidence:  "e",
			}},
		},
		{
			Version:  VersionV1,
			ReviewID: "review",
			Findings: []Finding{{
				Category:  "c",
				Severity:  "high",
				Path:      "   ",
				StartLine: 4,
				EndLine:   4,
				Title:     "t",
				Message:   "m",
				Evidence:  "e",
			}},
		},
		{
			Version:  VersionV1,
			ReviewID: "review",
			Findings: []Finding{{
				Category:  "c",
				Severity:  "high",
				Path:      "x.go",
				StartLine: 0,
				EndLine:   4,
				Title:     "t",
				Message:   "m",
				Evidence:  "e",
			}},
		},
		{
			Version:  VersionV1,
			ReviewID: "review",
			Findings: []Finding{{
				Category:   "c",
				Severity:   "high",
				Confidence: 1.5,
				Path:       "x.go",
				StartLine:  4,
				EndLine:    4,
				Title:      "t",
				Message:    "m",
				Evidence:   "e",
			}},
		},
		{
			Version:  VersionV1,
			ReviewID: "review",
			Findings: []Finding{{
				Category:  "c",
				Severity:  "high",
				Path:      "x.go",
				StartLine: 5,
				EndLine:   4,
				Title:     "t",
				Message:   "m",
				Evidence:  "e",
			}},
		},
	}

	for _, tc := range cases {
		if err := Validate(tc); err == nil {
			t.Fatalf("Validate(%+v) error = nil, want validation error", tc)
		}
	}
}

func TestWriteBundleNormalizesAndValidates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := dir + "/findings.json"
	bundle := FindingsBundle{
		ReviewID: "review-a",
		HeadSHA:  "head-a",
		Prompt: &PromptMetadata{
			PromptID:      "diffpal.review",
			PromptVersion: "v1.0.0",
			Purpose:       "review_changed_diff",
			SchemaVersion: "findings.v1",
		},
		Findings: []Finding{{
			Category:   "security",
			Severity:   "HIGH",
			Confidence: 0.8,
			Path:       "web/app.js",
			StartLine:  7,
			EndLine:    7,
			Title:      "xss risk",
			Message:    "unsafe HTML sink",
			Evidence:   "innerHTML receives tainted input",
			Impact:     "attackers can execute script in another user's browser",
		}},
	}
	if err := WriteBundle(path, bundle, "repo-a"); err != nil {
		t.Fatalf("WriteBundle() error = %v", err)
	}
	readBack, err := ReadBundle(path)
	if err != nil {
		t.Fatalf("ReadBundle() error = %v", err)
	}
	if readBack.Version != VersionV1 {
		t.Fatalf("Version = %q, want %q", readBack.Version, VersionV1)
	}
	if readBack.Findings[0].ID == "" {
		t.Fatal("ID = empty, want fingerprint")
	}
	if readBack.Findings[0].ReviewID != "review-a" {
		t.Fatalf("ReviewID = %q, want review-a", readBack.Findings[0].ReviewID)
	}
	if readBack.Prompt.PromptVersion != "v1.0.0" {
		t.Fatalf("Prompt = %+v, want persisted prompt metadata", readBack.Prompt)
	}
	if readBack.Findings[0].Impact != "attackers can execute script in another user's browser" {
		t.Fatalf("Impact = %q, want persisted impact", readBack.Findings[0].Impact)
	}
}

func TestFingerprintPreservesPathCase(t *testing.T) {
	t.Parallel()

	base := Finding{
		ReviewID:  "review-a",
		Category:  "correctness",
		Path:      "internal/app/service.go",
		StartLine: 12,
		EndLine:   12,
		Message:   "same message",
		Evidence:  "same evidence",
	}
	upper := base
	upper.Path = "internal/app/Service.go"
	if Fingerprint("repo", "head-a", base) == Fingerprint("repo", "head-a", upper) {
		t.Fatal("Fingerprint() matched paths that differ by case")
	}
}
