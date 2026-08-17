package findings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadBundleDefaultPath(t *testing.T) {
	dir := t.TempDir()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir(%q) error = %v", dir, err)
	}
	defer func() {
		if err := os.Chdir(prev); err != nil {
			t.Fatalf("restore cwd error = %v", err)
		}
	}()

	bundle := FindingsBundle{
		ReviewID: "review-default",
		HeadSHA:  "head-default",
		Findings: []Finding{{
			Category:   "maintainability",
			Severity:   "MEDIUM",
			Confidence: 0.7,
			Path:       "pkg/example/example.go",
			StartLine:  12,
			EndLine:    12,
			Title:      "dead branch",
			Message:    "conditional can never be true",
			Evidence:   NewEvidence("constant comparison folds to false"),
			Impact:     NewImpact("maintainers may spend time on unreachable logic"),
		}},
	}

	if err := WriteBundle("", bundle, "repo-default"); err != nil {
		t.Fatalf("WriteBundle(default path) error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, DefaultBundlePath)); err != nil {
		t.Fatalf("Stat(default bundle path) error = %v", err)
	}

	readBack, err := ReadBundle("")
	if err != nil {
		t.Fatalf("ReadBundle(default path) error = %v", err)
	}
	if readBack.Version != VersionV4 {
		t.Fatalf("Version = %q, want %q", readBack.Version, VersionV4)
	}
	if got := readBack.Findings[0].Severity; got != "medium" {
		t.Fatalf("Severity = %q, want medium", got)
	}
}

func TestReadBundleRejectsUnsupportedVersion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "findings.json")
	raw := []byte(`{"version":"v5","review_id":"review","findings":[]}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := ReadBundle(path); err == nil {
		t.Fatal("ReadBundle() error = nil, want version validation failure")
	}
}

func TestReadBundleDefaultsLegacyChangedSpansToRight(t *testing.T) {
	t.Parallel()

	for _, version := range []string{VersionV1, VersionV2, VersionV3} {
		t.Run(version, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "findings.json")
			raw := []byte(`{"version":"` + version + `","review_id":"review","findings":[{"category":"correctness","severity":"medium","confidence":0.8,"path":"app.go","start_line":4,"end_line":4,"changed_span":{"path":"app.go","start_line":4,"end_line":4},"title":"wrong result","message":"the changed line returns the wrong value","evidence":{"anchor":"L4","reasoning_basis":"the return value changed","source":"changed_line"},"impact":{"summary":"callers receive the wrong value","scope":"API callers"},"blocking":false,"provider":"test"}]}`)
			if err := os.WriteFile(path, raw, 0o644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			bundle, err := ReadBundle(path)
			if err != nil {
				t.Fatalf("ReadBundle(%s) error = %v", version, err)
			}
			if got := bundle.Findings[0].ChangedSpan.Side; got != SideRight {
				t.Fatalf("ChangedSpan.Side = %q, want %q", got, SideRight)
			}
		})
	}
}

func TestFormatBundleProducesCanonicalJSON(t *testing.T) {
	t.Parallel()

	bundle := FindingsBundle{
		ReviewID: "review-format",
		HeadSHA:  "head-format",
		Findings: []Finding{{
			Category:   "security",
			Severity:   "CRITICAL",
			Confidence: 1,
			Path:       "internal/db/query.go",
			StartLine:  22,
			EndLine:    23,
			Title:      "unsafe query construction",
			Message:    "query concatenates untrusted input",
			Evidence:   NewEvidence("user input is appended into SQL text"),
			Impact:     NewImpact("attackers can run unintended queries"),
		}},
	}

	raw, err := FormatBundle(bundle, "repo-format")
	if err != nil {
		t.Fatalf("FormatBundle() error = %v", err)
	}

	var readBack FindingsBundle
	if err := json.Unmarshal(raw, &readBack); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if readBack.Version != VersionV4 {
		t.Fatalf("Version = %q, want %q", readBack.Version, VersionV4)
	}
	if readBack.Findings[0].ID == "" {
		t.Fatal("ID = empty, want fingerprint")
	}
	if got := readBack.Findings[0].Severity; got != "critical" {
		t.Fatalf("Severity = %q, want critical", got)
	}
}
