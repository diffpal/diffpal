package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublishSummaryCommentCreatesWhenMissing(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	var postedBody string
	handlerErrs := make(chan error, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/issues/7/comments":
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/diffpal/issues/7/comments":
			var payload struct {
				Body string `json:"body"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				handlerErrs <- fmt.Errorf("decode summary comment: %w", err)
				http.Error(w, "bad payload", http.StatusBadRequest)
				return
			}
			postedBody = payload.Body
			w.WriteHeader(http.StatusCreated)
		default:
			handlerErrs <- fmt.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	err := PublishSummaryComment(context.Background(), "token", Context{Repo: "acme/diffpal", PRNumber: 7}, "# Summary\n\nNo findings.", server.Client())
	if err != nil {
		t.Fatalf("PublishSummaryComment() error = %v", err)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
	if !strings.Contains(postedBody, (ReviewIdentity{}).SummaryMarker()) {
		t.Fatalf("posted body missing marker: %q", postedBody)
	}
	if !strings.Contains(postedBody, "No findings.") {
		t.Fatalf("posted body missing summary: %q", postedBody)
	}
}

func TestPublishSummaryCommentUsesChannelMarker(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	identity, err := NewReviewIdentity("diffpal-dev")
	if err != nil {
		t.Fatalf("NewReviewIdentity() error = %v", err)
	}
	var patched bool
	handlerErrs := make(chan error, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/issues/7/comments":
			_, _ = w.Write([]byte(`[
				{"id":41,"body":"<!-- diffpal:summary -->\nstable"},
				{"id":42,"body":"<!-- diffpal:summary:diffpal-dev -->\ndev"}
			]`))
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/acme/diffpal/issues/comments/42":
			patched = true
			w.WriteHeader(http.StatusOK)
		default:
			handlerErrs <- fmt.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	err = PublishSummaryCommentWithIdentity(context.Background(), "token", Context{Repo: "acme/diffpal", PRNumber: 7}, "# Summary\n\nNo findings.", identity, server.Client())
	if err != nil {
		t.Fatalf("PublishSummaryCommentWithIdentity() error = %v", err)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
	if !patched {
		t.Fatal("channel summary comment was not updated")
	}
}

func TestPublishSummaryCommentFindsMarkerOnNextPage(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	var patched bool
	handlerErrs := make(chan error, 3)
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/issues/7/comments" && r.URL.Query().Get("page") == "":
			w.Header().Set("Link", `<`+server.URL+`/repos/acme/diffpal/issues/7/comments?per_page=100&page=2>; rel="next"`)
			_, _ = w.Write([]byte(`[{"id":41,"body":"not diffpal"}]`))
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/issues/7/comments" && r.URL.Query().Get("page") == "2":
			_, _ = w.Write([]byte(`[{"id":42,"body":"<!-- diffpal:summary -->\nold"}]`))
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/acme/diffpal/issues/comments/42":
			patched = true
			w.WriteHeader(http.StatusOK)
		default:
			handlerErrs <- fmt.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	err := PublishSummaryComment(context.Background(), "token", Context{Repo: "acme/diffpal", PRNumber: 7}, "# Summary\n\nNo findings.", server.Client())
	if err != nil {
		t.Fatalf("PublishSummaryComment() error = %v", err)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
	if !patched {
		t.Fatal("summary comment from second page was not updated")
	}
}

func TestPublishSummaryCommentFollowsRelativeNextPage(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	var patched bool
	handlerErrs := make(chan error, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/issues/7/comments" && r.URL.Query().Get("page") == "":
			w.Header().Set("Link", `</repos/acme/diffpal/issues/7/comments?per_page=100&page=2>; rel="next"`)
			_, _ = w.Write([]byte(`[{"id":41,"body":"not diffpal"}]`))
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/issues/7/comments" && r.URL.Query().Get("page") == "2":
			_, _ = w.Write([]byte(`[{"id":42,"body":"<!-- diffpal:summary -->\nold"}]`))
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/acme/diffpal/issues/comments/42":
			patched = true
			w.WriteHeader(http.StatusOK)
		default:
			handlerErrs <- fmt.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	err := PublishSummaryComment(context.Background(), "token", Context{Repo: "acme/diffpal", PRNumber: 7}, "# Summary\n\nNo findings.", server.Client())
	if err != nil {
		t.Fatalf("PublishSummaryComment() error = %v", err)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
	if !patched {
		t.Fatal("summary comment from relative next page was not updated")
	}
}

func TestPublishSummaryCommentRejectsCrossHostPagination(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	handlerErrs := make(chan error, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/issues/7/comments":
			w.Header().Set("Link", `<https://evil.example/repos/acme/diffpal/issues/7/comments?per_page=100&page=2>; rel="next"`)
			_, _ = w.Write([]byte(`[{"id":41,"body":"not diffpal"}]`))
		default:
			handlerErrs <- fmt.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	err := PublishSummaryComment(context.Background(), "token", Context{Repo: "acme/diffpal", PRNumber: 7}, "# Summary\n\nNo findings.", server.Client())
	if err == nil {
		t.Fatal("PublishSummaryComment() error = nil, want untrusted pagination error")
	}
	if !strings.Contains(err.Error(), "untrusted GitHub pagination URL") {
		t.Fatalf("PublishSummaryComment() error = %v, want untrusted pagination error", err)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
}

func TestPublishSummaryCommentIgnoresQuotedMarker(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	var posted bool
	var patched bool
	handlerErrs := make(chan error, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/issues/7/comments":
			_, _ = w.Write([]byte(`[{"id":42,"body":"user quote: <!-- diffpal:summary --> old"}]`))
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/diffpal/issues/7/comments":
			posted = true
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/acme/diffpal/issues/comments/42":
			patched = true
			w.WriteHeader(http.StatusOK)
		default:
			handlerErrs <- fmt.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	err := PublishSummaryComment(context.Background(), "token", Context{Repo: "acme/diffpal", PRNumber: 7}, "# Summary\n\nNo findings.", server.Client())
	if err != nil {
		t.Fatalf("PublishSummaryComment() error = %v", err)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
	if patched {
		t.Fatal("quoted summary marker was updated")
	}
	if !posted {
		t.Fatal("summary comment was not created")
	}
}

func TestPublishSummaryCommentUpdatesExistingMarker(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	var patched bool
	handlerErrs := make(chan error, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/issues/7/comments":
			_, _ = w.Write([]byte(`[{"id":42,"body":"<!-- diffpal:summary -->\nold"}]`))
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/acme/diffpal/issues/comments/42":
			patched = true
			w.WriteHeader(http.StatusOK)
		default:
			handlerErrs <- fmt.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	err := PublishSummaryComment(context.Background(), "token", Context{Repo: "acme/diffpal", PRNumber: 7}, "# Summary\n\nNo findings.", server.Client())
	if err != nil {
		t.Fatalf("PublishSummaryComment() error = %v", err)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
	if !patched {
		t.Fatal("summary comment was not updated")
	}
}
