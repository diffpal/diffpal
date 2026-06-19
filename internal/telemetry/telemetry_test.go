package telemetry

import (
	"strings"
	"testing"
)

func TestRecorderTracksProviderUsageAndPublishOutcomes(t *testing.T) {
	t.Parallel()

	rec := NewRecorder()
	rec.Record(Event{
		Name:       "publish",
		Provider:   "github",
		Success:    true,
		TokenUsage: 0,
		LatencyMs:  12,
		Outcome:    "publish:review",
	})
	rec.Record(Event{
		Name:       "publish",
		Provider:   "github",
		Success:    false,
		TokenUsage: 0,
		LatencyMs:  5,
		Outcome:    "publish:comments",
	})
	rec.Record(Event{
		Name:       "review",
		Provider:   "openai",
		Success:    true,
		TokenUsage: 321,
		LatencyMs:  44,
		Outcome:    "review:findings",
	})

	snap := rec.Snapshot()
	github := snap["publish:github"]
	if github.Provider != "github" {
		t.Fatalf("Provider = %q, want github", github.Provider)
	}
	if github.PublishSuccess != 1 || github.PublishFailure != 1 {
		t.Fatalf("unexpected publish counters: %+v", github)
	}
	openai := snap["review:openai"]
	if openai.TokenUsage != 321 {
		t.Fatalf("TokenUsage = %d, want 321", openai.TokenUsage)
	}
}

func TestLogEntryRedactsSensitivePromptPreview(t *testing.T) {
	t.Parallel()

	entry := LogEntry(Event{
		Name:          "review",
		Provider:      "OpenAI",
		Success:       true,
		TokenUsage:    42,
		LatencyMs:     9,
		Outcome:       "review:findings",
		PromptPreview: "Authorization: Bearer abc123\napi_key=shh-secret\nnormal line",
	})

	if strings.Contains(entry, "abc123") || strings.Contains(entry, "shh-secret") {
		t.Fatalf("entry leaked secret material: %s", entry)
	}
	if !strings.Contains(entry, "provider=openai") {
		t.Fatalf("entry missing normalized provider: %s", entry)
	}
}
