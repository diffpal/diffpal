package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPublishStatusUsesBoundedGitLabClient(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost || !strings.Contains(r.URL.EscapedPath(), "/projects/group%2Fproject/statuses/head-a") {
			http.Error(w, fmt.Sprintf("unexpected request: %s %s", r.Method, r.URL.String()), http.StatusBadRequest)
			return
		}
		if got := r.Header.Get("PRIVATE-TOKEN"); got != "token" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if payload["state"] != "success" || payload["name"] != "DiffPal Review" {
			http.Error(w, fmt.Sprintf("unexpected payload: %#v", payload), http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITLAB_API_URL", server.URL+"/api/v4")

	err := PublishStatus(context.Background(), "api_token", "token", Context{
		Repo:            "group/project",
		MergeRequestIID: "7",
		HeadSHA:         "head-a",
	}, StatusPayload{State: "success", Name: "DiffPal Review", Context: "diffpal/review", Description: "complete"}, server.Client())
	if err != nil {
		t.Fatalf("PublishStatus() error = %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("requests = %d, want 1", calls.Load())
	}
}

func TestPublishSummaryDiscussionUpdatesExistingSummary(t *testing.T) {
	var updateCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/discussions"):
			_, _ = w.Write([]byte(`[{"id":"summary-thread","notes":[{"id":9,"body":"<!-- diffpal:summary -->\nold"}]}]`))
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/discussions/summary-thread/notes/9"):
			updateCalls.Add(1)
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if body, _ := payload["body"].(string); !strings.Contains(body, summaryDiscussionMarker) || !strings.Contains(body, "new summary") {
				http.Error(w, fmt.Sprintf("unexpected body: %#v", payload), http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{"id":9}`))
		default:
			http.Error(w, fmt.Sprintf("unexpected request: %s %s", r.Method, r.URL.String()), http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITLAB_API_URL", server.URL+"/api/v4")

	err := PublishSummaryDiscussion(context.Background(), "api_token", "token", Context{
		Repo:            "group/project",
		MergeRequestIID: "7",
	}, "new summary", server.Client())
	if err != nil {
		t.Fatalf("PublishSummaryDiscussion() error = %v", err)
	}
	if updateCalls.Load() != 1 {
		t.Fatalf("update calls = %d, want 1", updateCalls.Load())
	}
}

func TestPublishDiscussionsCreatesInlineFinding(t *testing.T) {
	var createCalls atomic.Int32
	var resolveCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/versions"):
			_, _ = w.Write([]byte(`[{"id":1,"base_commit_sha":"base-a","start_commit_sha":"base-a","head_commit_sha":"head-a"}]`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/versions/1"):
			_, _ = w.Write([]byte(`{"id":1,"base_commit_sha":"base-a","start_commit_sha":"base-a","head_commit_sha":"head-a","diffs":[{"old_path":"main.go","new_path":"main.go","diff":"@@ -1 +1 @@\n-old\n+new"}]}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/discussions"):
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/discussions"):
			createCalls.Add(1)
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if body, _ := payload["body"].(string); !strings.Contains(body, "id:fp-a") || !strings.Contains(body, "loc:") {
				http.Error(w, fmt.Sprintf("missing finding marker: %#v", payload), http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{"id":"discussion-a"}`))
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/discussions/discussion-a"):
			resolveCalls.Add(1)
			_, _ = w.Write([]byte(`{"id":"discussion-a"}`))
		default:
			http.Error(w, fmt.Sprintf("unexpected request: %s %s", r.Method, r.URL.String()), http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("DIFFPAL_GITLAB_API_URL", server.URL+"/api/v4")

	plan := DiscussionPlan{Actions: []DiscussionAction{{
		Type:       ActionCreate,
		FindingID:  "fp-a",
		Body:       "finding body",
		Path:       "main.go",
		Line:       1,
		EndLine:    1,
		Resolved:   false,
		ThreadHash: discussionKey("main.go", 1, "correctness", "fp-a"),
	}}}
	err := PublishDiscussions(context.Background(), "api_token", "token", Context{
		Repo:            "group/project",
		MergeRequestIID: "7",
	}, plan, server.Client())
	if err != nil {
		t.Fatalf("PublishDiscussions() error = %v", err)
	}
	if createCalls.Load() != 1 || resolveCalls.Load() != 1 {
		t.Fatalf("calls = create:%d resolve:%d, want 1/1", createCalls.Load(), resolveCalls.Load())
	}
}
