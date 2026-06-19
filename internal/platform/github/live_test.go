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

func TestPublishPullRequestReviewCreatesReviewWithInlineComments(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	identity, err := NewReviewIdentity("diffpal-dev")
	if err != nil {
		t.Fatalf("NewReviewIdentity() error = %v", err)
	}
	var posted struct {
		CommitID string `json:"commit_id"`
		Event    string `json:"event"`
		Body     string `json:"body"`
		Comments []struct {
			Path string `json:"path"`
			Line int    `json:"line"`
			Side string `json:"side"`
			Body string `json:"body"`
		} `json:"comments"`
	}
	handlerErrs := make(chan error, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/pulls/7/reviews":
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/diffpal/pulls/7/reviews":
			if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
				handlerErrs <- fmt.Errorf("decode review: %w", err)
				http.Error(w, "bad payload", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
		default:
			handlerErrs <- fmt.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	err = PublishPullRequestReviewWithIdentity(context.Background(), "token", Context{
		Repo:     "acme/diffpal",
		PRNumber: 7,
		HeadSHA:  "head-a",
	}, "# Summary\n\nNo findings.", identity, CommentPlan{Actions: []CommentAction{{
		Type: ActionCreate,
		Body: "finding body",
		Path: "internal/app.go",
		Line: 12,
	}}}, ReviewEventRequestChanges, server.Client())
	if err != nil {
		t.Fatalf("PublishPullRequestReviewWithIdentity() error = %v", err)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
	if posted.CommitID != "head-a" {
		t.Fatalf("commit_id = %q, want head-a", posted.CommitID)
	}
	if posted.Event != "REQUEST_CHANGES" {
		t.Fatalf("event = %q, want REQUEST_CHANGES", posted.Event)
	}
	if !strings.Contains(posted.Body, "<!-- diffpal:review:diffpal-dev head_sha:head-a -->") {
		t.Fatalf("body missing review marker:\n%s", posted.Body)
	}
	if len(posted.Comments) != 1 {
		t.Fatalf("comments = %d, want 1", len(posted.Comments))
	}
	if posted.Comments[0].Path != "internal/app.go" || posted.Comments[0].Line != 12 || posted.Comments[0].Side != "RIGHT" {
		t.Fatalf("unexpected review comment: %#v", posted.Comments[0])
	}
}

func TestPublishPullRequestReviewUpdatesExistingHeadReview(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	var patchedBody string
	handlerErrs := make(chan error, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/pulls/7/reviews":
			_, _ = w.Write([]byte(`[
				{"id":41,"state":"COMMENTED","body":"<!-- diffpal:review:diffpal head_sha:old-head -->\nold"},
				{"id":42,"state":"COMMENTED","body":"<!-- diffpal:review:diffpal head_sha:head-a -->\ncurrent"}
			]`))
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/acme/diffpal/pulls/7/reviews/42":
			var payload struct {
				Body string `json:"body"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				handlerErrs <- fmt.Errorf("decode review update: %w", err)
				http.Error(w, "bad payload", http.StatusBadRequest)
				return
			}
			patchedBody = payload.Body
			w.WriteHeader(http.StatusOK)
		default:
			handlerErrs <- fmt.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	err := PublishPullRequestReviewWithIdentity(context.Background(), "token", Context{
		Repo:     "acme/diffpal",
		PRNumber: 7,
		HeadSHA:  "head-a",
	}, "# Summary\n\nUpdated.", ReviewIdentity{}, CommentPlan{}, ReviewEventComment, server.Client())
	if err != nil {
		t.Fatalf("PublishPullRequestReviewWithIdentity() error = %v", err)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
	if !strings.Contains(patchedBody, "Updated.") {
		t.Fatalf("patched body missing summary:\n%s", patchedBody)
	}
}

func TestPublishPullRequestReviewCreatesNewReviewWithCommentsWhenExistingHeadReviewExists(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	var posted struct {
		Body     string           `json:"body"`
		Comments []map[string]any `json:"comments"`
	}
	handlerErrs := make(chan error, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/pulls/7/reviews":
			_, _ = w.Write([]byte(`[
				{"id":42,"state":"COMMENTED","body":"<!-- diffpal:review:diffpal head_sha:head-a -->\ncurrent"}
			]`))
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/diffpal/pulls/7/reviews":
			if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
				handlerErrs <- fmt.Errorf("decode review create: %w", err)
				http.Error(w, "bad payload", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
		default:
			handlerErrs <- fmt.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	err := PublishPullRequestReviewWithIdentity(context.Background(), "token", Context{
		Repo:     "acme/diffpal",
		PRNumber: 7,
		HeadSHA:  "head-a",
	}, "# Summary\n\nUpdated.", ReviewIdentity{}, CommentPlan{Actions: []CommentAction{{
		Type: ActionCreate,
		Body: "new finding",
		Path: "internal/app.go",
		Line: 12,
	}}}, ReviewEventComment, server.Client())
	if err != nil {
		t.Fatalf("PublishPullRequestReviewWithIdentity() error = %v", err)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
	if !strings.Contains(posted.Body, "Updated.") {
		t.Fatalf("posted body missing summary:\n%s", posted.Body)
	}
	if len(posted.Comments) != 1 {
		t.Fatalf("comments = %d, want 1", len(posted.Comments))
	}
}

func TestPublishPullRequestReviewCreatesWhenExistingStateDoesNotMatchEvent(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	var postedEvent string
	handlerErrs := make(chan error, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/pulls/7/reviews":
			_, _ = w.Write([]byte(`[
				{"id":42,"state":"COMMENTED","body":"<!-- diffpal:review:diffpal head_sha:head-a -->\ncurrent"}
			]`))
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/diffpal/pulls/7/reviews":
			var payload struct {
				Event string `json:"event"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				handlerErrs <- fmt.Errorf("decode review create: %w", err)
				http.Error(w, "bad payload", http.StatusBadRequest)
				return
			}
			postedEvent = payload.Event
			w.WriteHeader(http.StatusCreated)
		default:
			handlerErrs <- fmt.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	err := PublishPullRequestReviewWithIdentity(context.Background(), "token", Context{
		Repo:     "acme/diffpal",
		PRNumber: 7,
		HeadSHA:  "head-a",
	}, "# Summary\n\nBlocking.", ReviewIdentity{}, CommentPlan{}, ReviewEventRequestChanges, server.Client())
	if err != nil {
		t.Fatalf("PublishPullRequestReviewWithIdentity() error = %v", err)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
	if postedEvent != "REQUEST_CHANGES" {
		t.Fatalf("posted event = %q, want REQUEST_CHANGES", postedEvent)
	}
}

func TestPublishPullRequestReviewRejectsCrossHostPagination(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	handlerErrs := make(chan error, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/pulls/7/reviews":
			w.Header().Set("Link", `<https://evil.example/repos/acme/diffpal/pulls/7/reviews?per_page=100&page=2>; rel="next"`)
			_, _ = w.Write([]byte(`[{"id":41,"body":"not diffpal"}]`))
		default:
			handlerErrs <- fmt.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	err := PublishPullRequestReviewWithIdentity(context.Background(), "token", Context{
		Repo:     "acme/diffpal",
		PRNumber: 7,
		HeadSHA:  "head-a",
	}, "# Summary\n\nNo findings.", ReviewIdentity{}, CommentPlan{}, ReviewEventComment, server.Client())
	if err == nil {
		t.Fatal("PublishPullRequestReviewWithIdentity() error = nil, want untrusted pagination error")
	}
	if !strings.Contains(err.Error(), "untrusted GitHub pagination URL") {
		t.Fatalf("PublishPullRequestReviewWithIdentity() error = %v, want untrusted pagination error", err)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
}
