package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublishSummaryCommentCreatesWhenMissing(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	var postedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/issues/7/comments":
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/diffpal/issues/7/comments":
			var payload struct {
				Body string `json:"body"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			postedBody = payload.Body
			w.WriteHeader(http.StatusCreated)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	err := PublishSummaryComment(context.Background(), "token", Context{Repo: "acme/diffpal", PRNumber: 7}, "# Summary\n\nNo findings.", server.Client())
	if err != nil {
		t.Fatalf("PublishSummaryComment() error = %v", err)
	}
	if !strings.Contains(postedBody, summaryCommentMarker) {
		t.Fatalf("posted body missing marker: %q", postedBody)
	}
	if !strings.Contains(postedBody, "No findings.") {
		t.Fatalf("posted body missing summary: %q", postedBody)
	}
}

func TestPublishSummaryCommentUpdatesExistingMarker(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	var patched bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/issues/7/comments":
			_, _ = w.Write([]byte(`[{"id":42,"body":"<!-- diffpal:summary --> old"}]`))
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/acme/diffpal/issues/7/comments/42":
			patched = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	err := PublishSummaryComment(context.Background(), "token", Context{Repo: "acme/diffpal", PRNumber: 7}, "# Summary\n\nNo findings.", server.Client())
	if err != nil {
		t.Fatalf("PublishSummaryComment() error = %v", err)
	}
	if !patched {
		t.Fatal("summary comment was not updated")
	}
}
