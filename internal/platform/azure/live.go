package azure

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/diffpal/diffpal/internal/platformapi"
)

func PublishThreads(ctx context.Context, tokenMode, token string, reviewCtx Context, plan ThreadPlan, client *http.Client) error {
	baseURL, err := adoPullRequestBaseURL(reviewCtx)
	if err != nil {
		return err
	}
	headers := adoHeaders(tokenMode, token)
	url := baseURL + "/threads?api-version=7.1-preview.1"
	for _, action := range plan.Actions {
		if action.Type == ActionSkip {
			continue
		}
		req := map[string]any{
			"comments": []map[string]any{{
				"parentCommentId": 0,
				"content":         action.Body,
				"commentType":     1,
			}},
			"status": 1,
		}
		if err := platformapi.DoJSON(ctx, client, http.MethodPost, url, headers, req); err != nil {
			return err
		}
	}
	return nil
}

func PublishStatus(ctx context.Context, tokenMode, token string, reviewCtx Context, payload StatusPayload, client *http.Client) error {
	baseURL, err := adoPullRequestBaseURL(reviewCtx)
	if err != nil {
		return err
	}
	headers := adoHeaders(tokenMode, token)
	url := baseURL + "/statuses?api-version=7.1-preview.1"
	req := map[string]any{
		"state":       string(payload.State),
		"description": payload.Description,
		"context": map[string]any{
			"name":  payload.Name,
			"genre": payload.Context,
		},
	}
	return platformapi.DoJSON(ctx, client, http.MethodPost, url, headers, req)
}

func adoHeaders(tokenMode, token string) map[string]string {
	headers := map[string]string{}
	switch tokenMode {
	case "pat":
		headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(":"+token))
	default:
		headers["Authorization"] = "Bearer " + token
	}
	return headers
}

func adoPullRequestBaseURL(ctx Context) (string, error) {
	if override := strings.TrimSpace(os.Getenv("DIFFPAL_ADO_API_URL")); override != "" {
		return strings.TrimRight(override, "/"), nil
	}
	if strings.TrimSpace(ctx.CollectionURI) == "" || strings.TrimSpace(ctx.ProjectName) == "" || strings.TrimSpace(ctx.RepositoryID) == "" || strings.TrimSpace(ctx.PullRequestID) == "" {
		return "", fmt.Errorf("missing Azure DevOps collection/project/repository/pull request context")
	}
	base, err := url.Parse(ctx.CollectionURI)
	if err != nil {
		return "", err
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/" + strings.TrimLeft(ctx.ProjectName, "/") + "/_apis/git/repositories/" + url.PathEscape(ctx.RepositoryID) + "/pullRequests/" + url.PathEscape(ctx.PullRequestID)
	return strings.TrimRight(base.String(), "/"), nil
}
