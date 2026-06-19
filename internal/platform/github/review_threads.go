package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/diffpal/diffpal/internal/platformapi"
)

func resolveSupersededFindingThreads(ctx context.Context, token string, reviewCtx Context, identity ReviewIdentity, plan CommentPlan, client *http.Client) {
	owner, repo, ok := strings.Cut(strings.TrimSpace(reviewCtx.Repo), "/")
	if !ok || strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" || reviewCtx.PRNumber <= 0 {
		return
	}
	current := currentFindingMarkers(plan, identity)
	cursor := ""
	for {
		page, err := queryReviewThreads(ctx, token, owner, repo, reviewCtx.PRNumber, cursor, client)
		if err != nil {
			return
		}
		for _, thread := range page.Threads {
			if thread.ID == "" || thread.IsResolved {
				continue
			}
			marker := threadFindingMarker(thread, identity)
			if marker == "" {
				continue
			}
			if _, ok := current[marker]; ok {
				continue
			}
			_ = resolveReviewThread(ctx, token, thread.ID, client)
		}
		if !page.HasNextPage || page.EndCursor == "" {
			return
		}
		cursor = page.EndCursor
	}
}

func currentFindingMarkers(plan CommentPlan, identity ReviewIdentity) map[string]struct{} {
	out := map[string]struct{}{}
	for _, state := range plan.State {
		if marker := findingMarker(identity, state.FindingID); marker != "" {
			out[marker] = struct{}{}
		}
	}
	for _, action := range plan.Actions {
		if marker := findingMarker(identity, action.FindingID); marker != "" {
			out[marker] = struct{}{}
		}
	}
	return out
}

func threadFindingMarker(thread reviewThread, identity ReviewIdentity) string {
	prefix := "<!-- diffpal:finding:" + identity.channel() + " "
	for _, comment := range thread.Comments {
		body := strings.TrimSpace(comment.Body)
		idx := strings.Index(body, prefix)
		if idx < 0 {
			continue
		}
		end := strings.Index(body[idx:], "-->")
		if end < 0 {
			continue
		}
		return body[idx : idx+end+3]
	}
	return ""
}

type reviewThreadsPage struct {
	Threads     []reviewThread
	HasNextPage bool
	EndCursor   string
}

type reviewThread struct {
	ID         string
	IsResolved bool
	Comments   []reviewThreadComment
}

type reviewThreadComment struct {
	Body string
}

func queryReviewThreads(ctx context.Context, token, owner, repo string, prNumber int, cursor string, client *http.Client) (reviewThreadsPage, error) {
	const query = `query($owner:String!, $name:String!, $number:Int!, $cursor:String) {
  repository(owner:$owner, name:$name) {
    pullRequest(number:$number) {
      reviewThreads(first:100, after:$cursor) {
        nodes {
          id
          isResolved
          comments(first:20) {
            nodes { body }
          }
        }
        pageInfo { hasNextPage endCursor }
      }
    }
  }
}`
	var resp struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					ReviewThreads struct {
						Nodes []struct {
							ID         string `json:"id"`
							IsResolved bool   `json:"isResolved"`
							Comments   struct {
								Nodes []struct {
									Body string `json:"body"`
								} `json:"nodes"`
							} `json:"comments"`
						} `json:"nodes"`
						PageInfo struct {
							HasNextPage bool   `json:"hasNextPage"`
							EndCursor   string `json:"endCursor"`
						} `json:"pageInfo"`
					} `json:"reviewThreads"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := doGraphQL(ctx, token, query, map[string]any{
		"owner":  owner,
		"name":   repo,
		"number": prNumber,
		"cursor": emptyStringAsNil(cursor),
	}, &resp, client); err != nil {
		return reviewThreadsPage{}, err
	}
	if len(resp.Errors) > 0 {
		return reviewThreadsPage{}, fmt.Errorf("github graphql: %s", resp.Errors[0].Message)
	}
	threads := resp.Data.Repository.PullRequest.ReviewThreads
	out := reviewThreadsPage{
		HasNextPage: threads.PageInfo.HasNextPage,
		EndCursor:   threads.PageInfo.EndCursor,
		Threads:     make([]reviewThread, 0, len(threads.Nodes)),
	}
	for _, node := range threads.Nodes {
		thread := reviewThread{
			ID:         node.ID,
			IsResolved: node.IsResolved,
			Comments:   make([]reviewThreadComment, 0, len(node.Comments.Nodes)),
		}
		for _, comment := range node.Comments.Nodes {
			thread.Comments = append(thread.Comments, reviewThreadComment{Body: comment.Body})
		}
		out.Threads = append(out.Threads, thread)
	}
	return out, nil
}

func resolveReviewThread(ctx context.Context, token, threadID string, client *http.Client) error {
	const mutation = `mutation($threadId:ID!) {
  resolveReviewThread(input:{threadId:$threadId}) {
    thread { id }
  }
}`
	var resp struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := doGraphQL(ctx, token, mutation, map[string]any{"threadId": threadID}, &resp, client); err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return fmt.Errorf("github graphql: %s", resp.Errors[0].Message)
	}
	return nil
}

func doGraphQL(ctx context.Context, token, query string, variables map[string]any, out any, client *http.Client) error {
	payload, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubGraphQLURL(), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range githubHeaders(token) {
		req.Header.Set(key, value)
	}
	resp, err := platformapi.DefaultClient(client).Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return fmt.Errorf("platform api %s %s failed: status=%d body=%s", http.MethodPost, githubGraphQLURL(), resp.StatusCode, msg)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func githubGraphQLURL() string {
	return strings.TrimRight(githubAPIBaseURL(), "/") + "/graphql"
}

func emptyStringAsNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
