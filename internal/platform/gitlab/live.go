package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/diffpal/diffpal/internal/platformapi"
)

func PublishDiscussions(ctx context.Context, tokenMode, token string, reviewCtx Context, plan DiscussionPlan, client *http.Client) error {
	if strings.TrimSpace(reviewCtx.Repo) == "" {
		return fmt.Errorf("missing GitLab repository/project")
	}
	if strings.TrimSpace(reviewCtx.MergeRequestIID) == "" {
		return fmt.Errorf("missing GitLab merge request iid")
	}
	headers := gitLabHeaders(tokenMode, token)
	baseURL := strings.TrimRight(gitLabAPIBaseURL(reviewCtx), "/") + "/projects/" + url.PathEscape(reviewCtx.Repo) + "/merge_requests/" + url.PathEscape(reviewCtx.MergeRequestIID) + "/discussions"
	for _, action := range plan.Actions {
		if action.Type == ActionSkip {
			continue
		}
		req := map[string]any{
			"body": action.Body,
		}
		if err := platformapi.DoJSON(ctx, client, http.MethodPost, baseURL, headers, req); err != nil {
			return err
		}
	}
	if strings.TrimSpace(plan.AdvisorySummary) != "" {
		if err := platformapi.DoJSON(ctx, client, http.MethodPost, baseURL, headers, map[string]any{
			"body": plan.AdvisorySummary,
		}); err != nil {
			return err
		}
	}
	return nil
}

func gitLabHeaders(tokenMode, token string) map[string]string {
	headers := map[string]string{}
	switch tokenMode {
	case "gitlab_token":
		headers["PRIVATE-TOKEN"] = token
	case "ci_job_token":
		headers["JOB-TOKEN"] = token
	}
	return headers
}

func gitLabAPIBaseURL(ctx Context) string {
	if override := strings.TrimSpace(os.Getenv("DIFFPAL_GITLAB_API_URL")); override != "" {
		return override
	}
	if strings.TrimSpace(ctx.WebURL) != "" {
		parsed, err := url.Parse(ctx.WebURL)
		if err == nil && parsed.Scheme != "" && parsed.Host != "" {
			return parsed.Scheme + "://" + parsed.Host + "/api/v4"
		}
	}
	return "https://gitlab.com/api/v4"
}
