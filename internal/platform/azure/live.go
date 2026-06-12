package azure

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	azuredevops "github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	azgit "github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

func PublishThreads(ctx context.Context, tokenMode, token string, reviewCtx Context, plan ThreadPlan, client *http.Client) error {
	gitClient, args, err := newGitClient(ctx, tokenMode, token, reviewCtx, client)
	if err != nil {
		return err
	}
	for _, action := range plan.Actions {
		if action.Type == ActionSkip {
			continue
		}
		if _, err := gitClient.CreateThread(ctx, azgit.CreateThreadArgs{
			CommentThread: threadPayload(action.Body),
			RepositoryId:  args.RepositoryID,
			PullRequestId: args.PullRequestID,
			Project:       args.Project,
		}); err != nil {
			return err
		}
	}
	return nil
}

const summaryThreadMarker = "<!-- diffpal:summary -->"

func PublishSummaryThread(ctx context.Context, tokenMode, token string, reviewCtx Context, summary string, client *http.Client) error {
	gitClient, args, err := newGitClient(ctx, tokenMode, token, reviewCtx, client)
	if err != nil {
		return err
	}
	body := summaryThreadMarker + "\n" + strings.TrimSpace(summary) + "\n"
	threadID, commentID, err := findSummaryThread(ctx, gitClient, args)
	if err != nil {
		return err
	}
	if threadID > 0 && commentID > 0 {
		if _, err := gitClient.UpdateComment(ctx, azgit.UpdateCommentArgs{
			Comment:       &azgit.Comment{Content: &body},
			RepositoryId:  args.RepositoryID,
			PullRequestId: args.PullRequestID,
			ThreadId:      &threadID,
			CommentId:     &commentID,
			Project:       args.Project,
		}); err != nil {
			return err
		}
		return nil
	}
	_, err = gitClient.CreateThread(ctx, azgit.CreateThreadArgs{
		CommentThread: threadPayload(body),
		RepositoryId:  args.RepositoryID,
		PullRequestId: args.PullRequestID,
		Project:       args.Project,
	})
	return err
}

func PublishStatus(ctx context.Context, tokenMode, token string, reviewCtx Context, payload StatusPayload, client *http.Client) error {
	gitClient, args, err := newGitClient(ctx, tokenMode, token, reviewCtx, client)
	if err != nil {
		return err
	}
	state := azgit.GitStatusState(payload.State)
	status := azgit.GitPullRequestStatus{
		State:       &state,
		Description: &payload.Description,
		Context: &azgit.GitStatusContext{
			Name:  &payload.Name,
			Genre: &payload.Context,
		},
	}
	_, err = gitClient.CreatePullRequestStatus(ctx, azgit.CreatePullRequestStatusArgs{
		Status:        &status,
		RepositoryId:  args.RepositoryID,
		PullRequestId: args.PullRequestID,
		Project:       args.Project,
	})
	return err
}

type gitClientArgs struct {
	RepositoryID  *string
	PullRequestID *int
	Project       *string
}

func newGitClient(ctx context.Context, tokenMode, token string, reviewCtx Context, client *http.Client) (azgit.Client, gitClientArgs, error) {
	if strings.TrimSpace(reviewCtx.CollectionURI) == "" || strings.TrimSpace(reviewCtx.ProjectName) == "" || strings.TrimSpace(reviewCtx.RepositoryID) == "" || strings.TrimSpace(reviewCtx.PullRequestID) == "" {
		return nil, gitClientArgs{}, fmt.Errorf("missing Azure DevOps collection/project/repository/pull request context")
	}
	prID, err := strconv.Atoi(reviewCtx.PullRequestID)
	if err != nil {
		return nil, gitClientArgs{}, fmt.Errorf("invalid Azure DevOps pull request id %q: %w", reviewCtx.PullRequestID, err)
	}
	connection := azureConnection(reviewCtx.CollectionURI, tokenMode, token)
	if client == nil {
		gitClient, err := azgit.NewClient(ctx, connection)
		if err != nil {
			return nil, gitClientArgs{}, err
		}
		return gitClient, gitClientArgs{
			RepositoryID:  &reviewCtx.RepositoryID,
			PullRequestID: &prID,
			Project:       &reviewCtx.ProjectName,
		}, nil
	}
	sdkClient := azuredevops.NewClientWithOptions(connection, connection.BaseUrl, azuredevops.WithHTTPClient(client))
	gitClient := &azgit.ClientImpl{
		Client: *sdkClient,
	}
	return gitClient, gitClientArgs{
		RepositoryID:  &reviewCtx.RepositoryID,
		PullRequestID: &prID,
		Project:       &reviewCtx.ProjectName,
	}, nil
}

func azureConnection(collectionURI, tokenMode, token string) *azuredevops.Connection {
	if tokenMode == "pat" {
		return azuredevops.NewPatConnection(collectionURI, token)
	}
	connection := azuredevops.NewAnonymousConnection(collectionURI)
	connection.AuthorizationString = "Bearer " + token
	return connection
}

func threadPayload(content string) *azgit.GitPullRequestCommentThread {
	commentType := azgit.CommentTypeValues.Text
	status := azgit.CommentThreadStatusValues.Active
	parentID := 0
	comments := []azgit.Comment{{
		ParentCommentId: &parentID,
		Content:         &content,
		CommentType:     &commentType,
	}}
	return &azgit.GitPullRequestCommentThread{
		Comments: &comments,
		Status:   &status,
	}
}

func findSummaryThread(ctx context.Context, gitClient azgit.Client, args gitClientArgs) (int, int, error) {
	threads, err := gitClient.GetThreads(ctx, azgit.GetThreadsArgs{
		RepositoryId:  args.RepositoryID,
		PullRequestId: args.PullRequestID,
		Project:       args.Project,
	})
	if err != nil {
		return 0, 0, err
	}
	if threads == nil {
		return 0, 0, nil
	}
	for _, thread := range *threads {
		if thread.Id == nil || thread.Comments == nil {
			continue
		}
		for _, comment := range *thread.Comments {
			if comment.Id != nil && comment.Content != nil && strings.Contains(*comment.Content, summaryThreadMarker) {
				return *thread.Id, *comment.Id, nil
			}
		}
	}
	return 0, 0, nil
}
