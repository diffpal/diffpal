package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/diffpal/diffpal/internal/findings"
	"github.com/diffpal/diffpal/internal/platformapi"
)

func ActiveReviewThreadState(ctx context.Context, token string, reviewCtx Context, identity ReviewIdentity, items []findings.Finding, client *http.Client) map[string]string {
	ids := activeReviewThreadFindingIDs(ctx, token, reviewCtx, identity, client)
	if len(ids) == 0 {
		return nil
	}
	out := make(map[string]string, len(ids))
	for _, item := range items {
		if _, ok := ids[item.ID]; !ok {
			continue
		}
		out[commentKey(item.Path, item.StartLine, item.Category, item.ID)] = item.ID
	}
	return out
}

func activeReviewThreadFindingIDs(ctx context.Context, token string, reviewCtx Context, identity ReviewIdentity, client *http.Client) map[string]struct{} {
	owner, repo, ok := strings.Cut(strings.TrimSpace(reviewCtx.Repo), "/")
	if !ok || strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" || reviewCtx.PRNumber <= 0 {
		return nil
	}
	out := map[string]struct{}{}
	cursor := ""
	for {
		page, err := queryReviewThreads(ctx, token, owner, repo, reviewCtx.PRNumber, cursor, client)
		if err != nil {
			return out
		}
		for _, thread := range page.Threads {
			if thread.ID == "" || thread.IsResolved {
				continue
			}
			findingID := threadFindingID(thread, identity)
			if findingID == "" {
				continue
			}
			out[findingID] = struct{}{}
		}
		if !page.HasNextPage || page.EndCursor == "" {
			return out
		}
		cursor = page.EndCursor
	}
}

func threadFindingID(thread reviewThread, identity ReviewIdentity) string {
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
		marker := body[idx : idx+end+3]
		return findingIDFromMarker(marker)
	}
	return ""
}

func findingIDFromMarker(marker string) string {
	inner := strings.TrimSuffix(strings.TrimPrefix(marker, "<!--"), "-->")
	for _, part := range strings.Fields(inner) {
		key, value, ok := strings.Cut(part, ":")
		if ok && key == "id" {
			return value
		}
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
