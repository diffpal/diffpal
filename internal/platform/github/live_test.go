package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
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
		case r.Method == http.MethodPost && r.URL.Path == "/graphql":
			_, _ = w.Write([]byte(`{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}}`))
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
		Type:      ActionCreate,
		FindingID: "fp-1",
		Body:      "finding body",
		Path:      "internal/app.go",
		Line:      12,
	}}}, server.Client())
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
	if posted.Event != "COMMENT" {
		t.Fatalf("event = %q, want COMMENT", posted.Event)
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
	if !strings.Contains(posted.Comments[0].Body, "<!-- diffpal:finding:diffpal-dev id:fp-1 -->") {
		t.Fatalf("review comment missing finding marker:\n%s", posted.Comments[0].Body)
	}
}

func TestPublishPullRequestReviewCreatesMultilineInlineComment(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	var posted struct {
		Comments []struct {
			Path      string `json:"path"`
			Line      int    `json:"line"`
			Side      string `json:"side"`
			StartLine int    `json:"start_line"`
			StartSide string `json:"start_side"`
			Body      string `json:"body"`
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
		case r.Method == http.MethodPost && r.URL.Path == "/graphql":
			_, _ = w.Write([]byte(`{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}}`))
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
	}, "# Summary\n\nFinding.", ReviewIdentity{}, CommentPlan{Actions: []CommentAction{{
		Type:    ActionCreate,
		Body:    "finding body",
		Path:    "internal/cmd/review.go",
		Line:    473,
		EndLine: 475,
	}}}, server.Client())
	if err != nil {
		t.Fatalf("PublishPullRequestReviewWithIdentity() error = %v", err)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
	if len(posted.Comments) != 1 {
		t.Fatalf("comments = %d, want 1", len(posted.Comments))
	}
	got := posted.Comments[0]
	if got.Path != "internal/cmd/review.go" || got.StartLine != 473 || got.Line != 475 || got.StartSide != "RIGHT" || got.Side != "RIGHT" {
		t.Fatalf("comment location = %#v, want internal/cmd/review.go lines 473-475 on RIGHT", got)
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
		case r.Method == http.MethodPost && r.URL.Path == "/graphql":
			_, _ = w.Write([]byte(`{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}}`))
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
	}, "# Summary\n\nUpdated.", ReviewIdentity{}, CommentPlan{}, server.Client())
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
		case r.Method == http.MethodPost && r.URL.Path == "/graphql":
			_, _ = w.Write([]byte(`{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}}`))
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
	}}}, server.Client())
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

func TestPublishPullRequestReviewResolvesSupersededFindingThreads(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	var resolved []string
	handlerErrs := make(chan error, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/pulls/7/reviews":
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/repos/acme/diffpal/pulls/7/reviews":
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && r.URL.Path == "/graphql":
			var payload struct {
				Query     string         `json:"query"`
				Variables map[string]any `json:"variables"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				handlerErrs <- fmt.Errorf("decode graphql: %w", err)
				http.Error(w, "bad payload", http.StatusBadRequest)
				return
			}
			if threadID, _ := payload.Variables["threadId"].(string); threadID != "" {
				resolved = append(resolved, threadID)
				_, _ = w.Write([]byte(`{"data":{"resolveReviewThread":{"thread":{"id":"` + threadID + `"}}}}`))
				return
			}
			_, _ = w.Write([]byte(`{
				"data": {"repository": {"pullRequest": {"reviewThreads": {
					"nodes": [
						{"id":"old-thread","isResolved":false,"comments":{"nodes":[{"body":"old\n<!-- diffpal:finding:diffpal id:old-finding -->"}]}},
						{"id":"current-thread","isResolved":false,"comments":{"nodes":[{"body":"current\n<!-- diffpal:finding:diffpal id:current-finding -->"}]}},
						{"id":"prior-review-thread","isResolved":false,"comments":{"nodes":[{"body":"legacy","pullRequestReview":{"body":"<!-- diffpal:review:diffpal head_sha:old-head -->\n# DiffPal Review Summary"}}]}},
						{"id":"other-channel","isResolved":false,"comments":{"nodes":[{"body":"other\n<!-- diffpal:finding:diffpal-dev id:old-finding -->"}]}},
						{"id":"already-resolved","isResolved":true,"comments":{"nodes":[{"body":"old\n<!-- diffpal:finding:diffpal id:resolved-finding -->"}]}}
					],
					"pageInfo": {"hasNextPage":false,"endCursor":""}
				}}}}
			}`))
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
	}, "# Summary\n\nUpdated.", ReviewIdentity{}, CommentPlan{
		Actions: []CommentAction{{
			Type:      ActionCreate,
			FindingID: "current-finding",
			Body:      "current",
			Path:      "internal/app.go",
			Line:      12,
		}},
		State: []CommentState{{FindingID: "current-finding"}},
	}, server.Client())
	if err != nil {
		t.Fatalf("PublishPullRequestReviewWithIdentity() error = %v", err)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
	wantResolved := map[string]bool{"old-thread": true, "prior-review-thread": true}
	if len(resolved) != len(wantResolved) {
		t.Fatalf("resolved threads = %#v, want old-thread and prior-review-thread", resolved)
	}
	for _, id := range resolved {
		if !wantResolved[id] {
			t.Fatalf("resolved threads = %#v, want old-thread and prior-review-thread", resolved)
		}
	}
}

func TestActiveReviewThreadStateUsesUnresolvedFindingMarkers(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	handlerErrs := make(chan error, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/graphql" {
			handlerErrs <- fmt.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{
			"data": {"repository": {"pullRequest": {"reviewThreads": {
				"nodes": [
					{"id":"active-thread","isResolved":false,"comments":{"nodes":[{"body":"active\n<!-- diffpal:finding:diffpal-dev id:current-finding -->"}]}},
					{"id":"resolved-thread","isResolved":true,"comments":{"nodes":[{"body":"resolved\n<!-- diffpal:finding:diffpal-dev id:resolved-finding -->"}]}},
					{"id":"other-channel","isResolved":false,"comments":{"nodes":[{"body":"other\n<!-- diffpal:finding:diffpal id:other-finding -->"}]}}
				],
				"pageInfo": {"hasNextPage":false,"endCursor":""}
			}}}}
		}`))
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITHUB_API_URL", server.URL)

	state := ActiveReviewThreadState(context.Background(), "token", Context{
		Repo:     "acme/diffpal",
		PRNumber: 7,
	}, ReviewIdentity{Channel: "diffpal-dev"}, []findings.Finding{{
		ID:        "current-finding",
		Path:      "internal/app.go",
		StartLine: 12,
		Category:  "security",
	}, {
		ID:        "resolved-finding",
		Path:      "internal/app.go",
		StartLine: 24,
		Category:  "security",
	}}, server.Client())
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
	wantKey := commentKey("internal/app.go", 12, "security", "current-finding")
	if len(state) != 1 || state[wantKey] != "current-finding" {
		t.Fatalf("state = %#v, want only active current finding", state)
	}
}

func TestPublishPullRequestReviewCreatesWhenExistingReviewIsNotCommented(t *testing.T) {
	t.Setenv("DIFFPAL_GITHUB_API_URL", "")
	var postedEvent string
	handlerErrs := make(chan error, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/acme/diffpal/pulls/7/reviews":
			_, _ = w.Write([]byte(`[
				{"id":42,"state":"CHANGES_REQUESTED","body":"<!-- diffpal:review:diffpal head_sha:head-a -->\ncurrent"}
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
		case r.Method == http.MethodPost && r.URL.Path == "/graphql":
			_, _ = w.Write([]byte(`{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}}`))
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
	}, "# Summary\n\nUpdated.", ReviewIdentity{}, CommentPlan{}, server.Client())
	if err != nil {
		t.Fatalf("PublishPullRequestReviewWithIdentity() error = %v", err)
	}
	select {
	case err := <-handlerErrs:
		t.Fatal(err)
	default:
	}
	if postedEvent != "COMMENT" {
		t.Fatalf("posted event = %q, want COMMENT", postedEvent)
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
	}, "# Summary\n\nNo findings.", ReviewIdentity{}, CommentPlan{}, server.Client())
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
