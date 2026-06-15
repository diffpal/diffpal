package promptpack

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestRenderReviewSystemMatchesGolden(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("testdata/review_system.golden")
	if err != nil {
		t.Fatalf("ReadFile(golden) error = %v", err)
	}
	got := RenderReviewSystem(ReviewOptions{
		Instructions: "Focus on auth boundary changes.",
	})
	if got != strings.TrimRight(string(raw), "\n") {
		t.Fatalf("RenderReviewSystem() mismatch\nwant:\n%s\n\ngot:\n%s", string(raw), got)
	}
}

func TestReviewSchemasAreValidJSON(t *testing.T) {
	t.Parallel()

	for name, raw := range map[string]string{
		"input":  InputSchemaJSON,
		"output": OutputSchemaJSON,
	} {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			t.Fatalf("%s schema is invalid JSON: %v", name, err)
		}
	}
}

func TestReviewMetadataIsStable(t *testing.T) {
	t.Parallel()

	got := ReviewMetadata()
	if got.PromptID != "diffpal.review" {
		t.Fatalf("PromptID = %q, want diffpal.review", got.PromptID)
	}
	if got.PromptVersion != "v1.0.0" {
		t.Fatalf("PromptVersion = %q, want v1.0.0", got.PromptVersion)
	}
	if got.Purpose != "review_changed_diff" {
		t.Fatalf("Purpose = %q, want review_changed_diff", got.Purpose)
	}
	if got.SchemaVersion != "findings.v1" {
		t.Fatalf("SchemaVersion = %q, want findings.v1", got.SchemaVersion)
	}
}

func TestReviewTaskNamesRequestedChecks(t *testing.T) {
	t.Parallel()

	got := ReviewTask([]string{"security", "bugs"})
	if !strings.Contains(got, "security, bugs") {
		t.Fatalf("ReviewTask() = %q, want requested checks", got)
	}
}
