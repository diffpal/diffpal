package telemetry

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Usage struct {
	Provider       string
	ProviderCalls  int64
	ProviderErrors int64
	PublishSuccess int64
	PublishFailure int64
	TokenUsage     int64
	DurationMs     int64
}

type Recorder struct {
	mu      sync.Mutex
	byEvent map[string]Usage
}

type Event struct {
	Name          string
	Provider      string
	Success       bool
	TokenUsage    int64
	LatencyMs     int64
	Outcome       string
	PromptPreview string
}

func NewRecorder() *Recorder {
	return &Recorder{
		byEvent: map[string]Usage{},
	}
}

func (r *Recorder) Record(event Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := strings.ToLower(event.Name + ":" + event.Provider)
	cur := r.byEvent[key]
	cur.Provider = event.Provider
	cur.ProviderCalls++
	cur.DurationMs += event.LatencyMs
	cur.TokenUsage += event.TokenUsage
	if !event.Success {
		cur.ProviderErrors++
	}
	if strings.HasPrefix(strings.ToLower(event.Outcome), "publish:") {
		if event.Success {
			cur.PublishSuccess++
		} else {
			cur.PublishFailure++
		}
	}
	r.byEvent[key] = cur
}

func (r *Recorder) Snapshot() map[string]Usage {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]Usage{}
	for k, v := range r.byEvent {
		out[k] = v
	}
	return out
}

func (r *Recorder) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	keys := make([]string, 0, len(r.byEvent))
	for k := range r.byEvent {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		u := r.byEvent[key]
		lines = append(lines, fmt.Sprintf("%s calls=%d errors=%d publish_success=%d publish_failure=%d tokens=%d duration_ms=%d", key, u.ProviderCalls, u.ProviderErrors, u.PublishSuccess, u.PublishFailure, u.TokenUsage, u.DurationMs))
	}
	return strings.Join(lines, "\n")
}

func RedactPath(path string) string {
	clean := strings.ToLower(path)
	if strings.Contains(clean, "token") || strings.Contains(clean, "secret") || strings.Contains(clean, "key") {
		return "***redacted***"
	}
	return path
}

func RedactText(raw string) string {
	clean := raw
	replacements := []string{
		"authorization: bearer ", "authorization: bearer ***redacted***",
		"api_key=", "api_key=***redacted***",
		"token=", "token=***redacted***",
		"secret=", "secret=***redacted***",
	}
	for i := 0; i < len(replacements); i += 2 {
		lower := strings.ToLower(clean)
		target := replacements[i]
		if idx := strings.Index(lower, target); idx >= 0 {
			end := strings.Index(clean[idx:], "\n")
			if end < 0 {
				end = len(clean) - idx
			}
			clean = clean[:idx] + replacements[i+1] + clean[idx+end:]
		}
	}
	if len(clean) > 160 {
		clean = clean[:160] + "..."
	}
	return clean
}

func LogEntry(event Event) string {
	return fmt.Sprintf(
		"name=%s provider=%s success=%t outcome=%s latency_ms=%d tokens=%d preview=%q",
		strings.ToLower(event.Name),
		strings.ToLower(event.Provider),
		event.Success,
		strings.ToLower(event.Outcome),
		event.LatencyMs,
		event.TokenUsage,
		RedactText(event.PromptPreview),
	)
}

func RecordDuration(start time.Time, success bool, name string, provider string, tokens int64) Event {
	return Event{
		Name:       name + ":" + provider,
		Success:    success,
		TokenUsage: tokens,
		LatencyMs:  time.Since(start).Milliseconds(),
	}
}
