package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/diffpal/diffpal/internal/platformapi"
)

func PublishCheckRun(ctx context.Context, token string, reviewCtx Context, payload CheckRunPayload, client *http.Client) error {
	if strings.TrimSpace(reviewCtx.Repo) == "" {
		return fmt.Errorf("missing GitHub repository")
	}
	req := map[string]any{
		"name":       payload.Name,
		"head_sha":   payload.HeadSHA,
		"status":     payload.Status,
		"conclusion": payload.Conclusion,
		"output": map[string]any{
			"title":       payload.Name,
			"summary":     payload.Summary,
			"annotations": payload.Annotations,
		},
	}
	url := strings.TrimRight(githubAPIBaseURL(), "/") + "/repos/" + reviewCtx.Repo + "/check-runs"
	return platformapi.DoJSON(ctx, client, http.MethodPost, url, map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/vnd.github+json",
	}, req)
}

func PublishInlineComments(ctx context.Context, token string, reviewCtx Context, plan CommentPlan, client *http.Client) error {
	if reviewCtx.PRNumber <= 0 {
		return fmt.Errorf("missing GitHub pull request number")
	}
	if strings.TrimSpace(reviewCtx.Repo) == "" {
		return fmt.Errorf("missing GitHub repository")
	}
	baseURL := strings.TrimRight(githubAPIBaseURL(), "/") + "/repos/" + reviewCtx.Repo + "/pulls/" + fmt.Sprint(reviewCtx.PRNumber) + "/comments"
	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/vnd.github+json",
	}
	for _, action := range plan.Actions {
		if action.Type == ActionSkip {
			continue
		}
		req := map[string]any{
			"body":      action.Body,
			"commit_id": reviewCtx.HeadSHA,
			"path":      action.Path,
			"line":      action.Line,
			"side":      "RIGHT",
		}
		if err := platformapi.DoJSON(ctx, client, http.MethodPost, baseURL, headers, req); err != nil {
			return err
		}
	}
	return nil
}

type issueComment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

func PublishSummaryComment(ctx context.Context, token string, reviewCtx Context, summary string, client *http.Client) error {
	return PublishSummaryCommentWithIdentity(ctx, token, reviewCtx, summary, ReviewIdentity{}, client)
}

func PublishSummaryCommentWithIdentity(ctx context.Context, token string, reviewCtx Context, summary string, identity ReviewIdentity, client *http.Client) error {
	if reviewCtx.PRNumber <= 0 {
		return fmt.Errorf("missing GitHub pull request number")
	}
	if strings.TrimSpace(reviewCtx.Repo) == "" {
		return fmt.Errorf("missing GitHub repository")
	}
	body := summaryCommentBody(summary, identity)
	baseURL := strings.TrimRight(githubAPIBaseURL(), "/") + "/repos/" + reviewCtx.Repo + "/issues/" + fmt.Sprint(reviewCtx.PRNumber) + "/comments"
	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/vnd.github+json",
	}
	existingID, err := findSummaryComment(ctx, token, baseURL, identity, client)
	if err != nil {
		return err
	}
	if existingID > 0 {
		updateURL := strings.TrimRight(githubAPIBaseURL(), "/") + "/repos/" + reviewCtx.Repo + "/issues/comments/" + fmt.Sprint(existingID)
		return platformapi.DoJSON(ctx, client, http.MethodPatch, updateURL, headers, map[string]any{"body": body})
	}
	return platformapi.DoJSON(ctx, client, http.MethodPost, baseURL, headers, map[string]any{"body": body})
}

func summaryCommentBody(summary string, identity ReviewIdentity) string {
	return identity.SummaryMarker() + "\n" + strings.TrimSpace(summary) + "\n"
}

func findSummaryComment(ctx context.Context, token, url string, identity ReviewIdentity, client *http.Client) (int64, error) {
	marker := identity.SummaryMarker()
	nextURL := url + "?per_page=100"
	for nextURL != "" {
		resp, err := getGitHubIssueCommentsPage(ctx, token, nextURL, client)
		if err != nil {
			return 0, err
		}
		for _, comment := range resp.comments {
			if hasSummaryMarker(comment.Body, marker) {
				return comment.ID, nil
			}
		}
		nextURL = resp.nextURL
	}
	return 0, nil
}

func hasSummaryMarker(body, marker string) bool {
	body = strings.TrimLeft(body, " \t\r\n")
	return body == marker || strings.HasPrefix(body, marker+"\n")
}

type issueCommentsPage struct {
	comments []issueComment
	nextURL  string
}

func getGitHubIssueCommentsPage(ctx context.Context, token, pageURL string, client *http.Client) (issueCommentsPage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return issueCommentsPage{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := platformapi.DefaultClient(client).Do(req)
	if err != nil {
		return issueCommentsPage{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return issueCommentsPage{}, fmt.Errorf("platform api %s %s failed: status=%d body=%s", http.MethodGet, pageURL, resp.StatusCode, msg)
	}
	var comments []issueComment
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return issueCommentsPage{}, err
	}
	return issueCommentsPage{
		comments: comments,
		nextURL:  nextLinkURL(resp.Header.Get("Link")),
	}, nil
}

func nextLinkURL(header string) string {
	for _, part := range strings.Split(header, ",") {
		link, rel, ok := parseLinkHeaderPart(part)
		if ok && rel == "next" {
			return link
		}
	}
	return ""
}

func parseLinkHeaderPart(part string) (string, string, bool) {
	part = strings.TrimSpace(part)
	if !strings.HasPrefix(part, "<") {
		return "", "", false
	}
	end := strings.Index(part, ">")
	if end <= 1 {
		return "", "", false
	}
	link := part[1:end]
	if _, err := url.ParseRequestURI(link); err != nil {
		return "", "", false
	}
	for _, param := range strings.Split(part[end+1:], ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(param), "=")
		if !ok || name != "rel" {
			continue
		}
		return link, strings.Trim(value, `"`), true
	}
	return "", "", false
}

func githubAPIBaseURL() string {
	if override := strings.TrimSpace(os.Getenv("DIFFPAL_GITHUB_API_URL")); override != "" {
		return override
	}
	return "https://api.github.com"
}
