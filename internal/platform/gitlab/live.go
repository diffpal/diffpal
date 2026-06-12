package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	gl "gitlab.com/gitlab-org/api/client-go"
)

func PublishDiscussions(ctx context.Context, tokenMode, token string, reviewCtx Context, plan DiscussionPlan, client *http.Client) error {
	gitlabClient, mrIID, err := newClient(tokenMode, token, reviewCtx, client)
	if err != nil {
		return err
	}
	for _, action := range plan.Actions {
		if action.Type == ActionSkip {
			continue
		}
		body := action.Body
		if _, _, err := gitlabClient.Discussions.CreateMergeRequestDiscussion(reviewCtx.Repo, mrIID, &gl.CreateMergeRequestDiscussionOptions{
			Body: &body,
		}, gl.WithContext(ctx)); err != nil {
			return err
		}
	}
	if strings.TrimSpace(plan.AdvisorySummary) != "" {
		if err := publishSummary(ctx, gitlabClient, reviewCtx, mrIID, plan.AdvisorySummary); err != nil {
			return err
		}
	}
	return nil
}

func PublishSummaryDiscussion(ctx context.Context, tokenMode, token string, reviewCtx Context, summary string, client *http.Client) error {
	gitlabClient, mrIID, err := newClient(tokenMode, token, reviewCtx, client)
	if err != nil {
		return err
	}
	return publishSummary(ctx, gitlabClient, reviewCtx, mrIID, summary)
}

func publishSummary(ctx context.Context, gitlabClient *gl.Client, reviewCtx Context, mrIID int64, summary string) error {
	body := strings.TrimSpace(summary)
	if body == "" {
		return nil
	}
	_, _, err := gitlabClient.Discussions.CreateMergeRequestDiscussion(reviewCtx.Repo, mrIID, &gl.CreateMergeRequestDiscussionOptions{
		Body: &body,
	}, gl.WithContext(ctx))
	return err
}

func newClient(tokenMode, token string, reviewCtx Context, client *http.Client) (*gl.Client, int64, error) {
	if strings.TrimSpace(reviewCtx.Repo) == "" {
		return nil, 0, fmt.Errorf("missing GitLab repository/project")
	}
	if strings.TrimSpace(reviewCtx.MergeRequestIID) == "" {
		return nil, 0, fmt.Errorf("missing GitLab merge request iid")
	}
	mrIID, err := strconv.ParseInt(reviewCtx.MergeRequestIID, 10, 64)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid GitLab merge request iid %q: %w", reviewCtx.MergeRequestIID, err)
	}
	options := []gl.ClientOptionFunc{
		gl.WithBaseURL(gitLabAPIBaseURL(reviewCtx)),
	}
	if client != nil {
		options = append(options, gl.WithHTTPClient(client))
	}
	switch tokenMode {
	case "ci_job_token":
		gitlabClient, err := gl.NewJobClient(token, options...)
		return gitlabClient, mrIID, err
	default:
		gitlabClient, err := gl.NewClient(token, options...)
		return gitlabClient, mrIID, err
	}
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
