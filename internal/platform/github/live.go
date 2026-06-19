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

type pullReview struct {
	ID       int64  `json:"id"`
	Body     string `json:"body"`
	CommitID string `json:"commit_id"`
	State    string `json:"state"`
}

func PublishPullRequestReviewWithIdentity(ctx context.Context, token string, reviewCtx Context, summary string, identity ReviewIdentity, plan CommentPlan, client *http.Client) error {
	if reviewCtx.PRNumber <= 0 {
		return fmt.Errorf("missing GitHub pull request number")
	}
	if strings.TrimSpace(reviewCtx.Repo) == "" {
		return fmt.Errorf("missing GitHub repository")
	}
	if strings.TrimSpace(reviewCtx.HeadSHA) == "" {
		return fmt.Errorf("missing GitHub head SHA")
	}
	repoURL := strings.TrimRight(githubAPIBaseURL(), "/") + "/repos/" + reviewCtx.Repo
	baseURL := repoURL + "/pulls/" + fmt.Sprint(reviewCtx.PRNumber) + "/reviews"
	headers := githubHeaders(token)
	body := pullRequestReviewBody(summary, identity, reviewCtx.HeadSHA)
	existingID, err := findPullRequestReview(ctx, token, baseURL, identity, reviewCtx.HeadSHA, client)
	if err != nil {
		return err
	}
	comments := pullRequestReviewComments(plan, identity)
	if existingID > 0 && len(comments) == 0 {
		updateURL := baseURL + "/" + fmt.Sprint(existingID)
		if err := platformapi.DoJSON(ctx, client, http.MethodPatch, updateURL, headers, map[string]any{"body": body}); err != nil {
			return err
		}
		return nil
	}
	req := map[string]any{
		"commit_id": reviewCtx.HeadSHA,
		"event":     "COMMENT",
		"body":      body,
	}
	if len(comments) > 0 {
		req["comments"] = comments
	}
	if err := platformapi.DoJSON(ctx, client, http.MethodPost, baseURL, headers, req); err != nil {
		return err
	}
	return nil
}

func pullRequestReviewBody(summary string, identity ReviewIdentity, headSHA string) string {
	return identity.ReviewMarker(headSHA) + "\n" + strings.TrimSpace(summary) + "\n"
}

func pullRequestReviewComments(plan CommentPlan, identity ReviewIdentity) []map[string]any {
	out := make([]map[string]any, 0, len(plan.Actions))
	for _, action := range plan.Actions {
		if action.Type == ActionSkip || strings.TrimSpace(action.Body) == "" || strings.TrimSpace(action.Path) == "" || action.Line <= 0 {
			continue
		}
		body := action.Body
		if marker := findingMarker(identity, action.FindingID); marker != "" {
			body += "\n" + marker
		}
		out = append(out, map[string]any{
			"path": action.Path,
			"line": action.Line,
			"side": "RIGHT",
			"body": body,
		})
		if action.EndLine > action.Line {
			out[len(out)-1]["start_line"] = action.Line
			out[len(out)-1]["start_side"] = "RIGHT"
			out[len(out)-1]["line"] = action.EndLine
		}
	}
	return out
}

func findPullRequestReview(ctx context.Context, token, url string, identity ReviewIdentity, headSHA string, client *http.Client) (int64, error) {
	marker := identity.ReviewMarker(headSHA)
	nextURL := url + "?per_page=100"
	var existingID int64
	for nextURL != "" {
		resp, err := getGitHubPullReviewsPage(ctx, token, nextURL, client)
		if err != nil {
			return 0, err
		}
		for _, review := range resp.reviews {
			if hasReviewMarker(review.Body, marker) && reviewStateIsCommented(review.State) {
				existingID = review.ID
			}
		}
		nextURL = resp.nextURL
	}
	return existingID, nil
}

type pullReviewsPage struct {
	reviews []pullReview
	nextURL string
}

func githubHeaders(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/vnd.github+json",
	}
}

func getGitHubPullReviewsPage(ctx context.Context, token, pageURL string, client *http.Client) (pullReviewsPage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return pullReviewsPage{}, err
	}
	for key, value := range githubHeaders(token) {
		req.Header.Set(key, value)
	}
	resp, err := platformapi.DefaultClient(client).Do(req)
	if err != nil {
		return pullReviewsPage{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return pullReviewsPage{}, fmt.Errorf("platform api %s %s failed: status=%d body=%s", http.MethodGet, pageURL, resp.StatusCode, msg)
	}
	var reviews []pullReview
	if err := json.NewDecoder(resp.Body).Decode(&reviews); err != nil {
		return pullReviewsPage{}, err
	}
	nextURL, err := nextLinkURL(resp.Header.Get("Link"), pageURL)
	if err != nil {
		return pullReviewsPage{}, err
	}
	return pullReviewsPage{
		reviews: reviews,
		nextURL: nextURL,
	}, nil
}

func hasReviewMarker(body, marker string) bool {
	body = strings.TrimLeft(body, " \t\r\n")
	return body == marker || strings.HasPrefix(body, marker+"\n")
}

func reviewStateIsCommented(state string) bool {
	return strings.EqualFold(strings.TrimSpace(state), "COMMENTED")
}

func nextLinkURL(header, currentPageURL string) (string, error) {
	currentURL, err := url.Parse(currentPageURL)
	if err != nil {
		return "", err
	}
	for _, part := range strings.Split(header, ",") {
		link, rel, ok := parseLinkHeaderPart(part)
		if ok && rel == "next" {
			nextURL, trusted, err := trustedPaginationURL(link, currentURL)
			if err != nil {
				return "", err
			}
			if !trusted {
				return "", fmt.Errorf("untrusted GitHub pagination URL: %s", link)
			}
			return nextURL, nil
		}
	}
	return "", nil
}

func trustedPaginationURL(link string, currentURL *url.URL) (string, bool, error) {
	nextURL, err := url.Parse(link)
	if err != nil {
		return "", false, err
	}
	if !nextURL.IsAbs() {
		nextURL = currentURL.ResolveReference(nextURL)
	}
	trusted := strings.EqualFold(nextURL.Scheme, currentURL.Scheme) &&
		strings.EqualFold(nextURL.Host, currentURL.Host)
	return nextURL.String(), trusted, nil
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
