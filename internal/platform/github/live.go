package github

import (
	"context"
	"fmt"
	"net/http"
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

func githubAPIBaseURL() string {
	if override := strings.TrimSpace(os.Getenv("DIFFPAL_GITHUB_API_URL")); override != "" {
		return override
	}
	return "https://api.github.com"
}
