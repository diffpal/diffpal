package azure

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"

	azuredevops "github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	azgit "github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/identity"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/location"
)

type gateVoteClient interface {
	CreatePullRequestReviewer(context.Context, azgit.CreatePullRequestReviewerArgs) (*azgit.IdentityRefWithVote, error)
}

type threadGitClient interface {
	CreateThread(context.Context, azgit.CreateThreadArgs) (*azgit.GitPullRequestCommentThread, error)
	GetPullRequestIterations(context.Context, azgit.GetPullRequestIterationsArgs) (*[]azgit.GitPullRequestIteration, error)
	GetPullRequestIterationChanges(context.Context, azgit.GetPullRequestIterationChangesArgs) (*azgit.GitPullRequestIterationChanges, error)
}

type pullRequestChangeRef struct {
	Path             string
	OriginalPath     string
	ChangeTrackingID int
}

type resolvedThreadTarget struct {
	FilePath         string
	ChangeTrackingID int
	IterationID      int
}

func PublishThreads(ctx context.Context, tokenMode, token string, reviewCtx Context, plan ThreadPlan, client *http.Client) error {
	gitClient, args, err := newGitClient(ctx, tokenMode, token, reviewCtx, client)
	if err != nil {
		return err
	}
	return publishThreadsWithClient(ctx, gitClient, args, plan)
}

func publishThreadsWithClient(ctx context.Context, gitClient threadGitClient, args gitClientArgs, plan ThreadPlan) error {
	iterationID, changes, err := listPullRequestIterationChanges(ctx, gitClient, args)
	if err != nil {
		return err
	}
	targets := resolveThreadTargets(plan.Actions, iterationID, changes)
	for _, action := range plan.Actions {
		if action.Type == ActionSkip {
			continue
		}
		payload := threadPayload(action.Body, action.Status, action.Path, action.Line, action.EndLine)
		if hasCanonicalAzureThreadLocation(action.Path, action.Line) {
			target, ok := targets[action.ThreadID]
			if !ok {
				log.Printf("diffpal: skipped Azure inline thread for %s:%d: no current PR change mapping found", action.Path, action.Line)
				continue
			}
			payload = threadPayloadForTarget(action, target)
		}
		if _, err := gitClient.CreateThread(ctx, azgit.CreateThreadArgs{
			CommentThread: payload,
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
		if _, err := gitClient.UpdateThread(ctx, azgit.UpdateThreadArgs{
			CommentThread: threadStatusPayload(ThreadStatusClosed),
			RepositoryId:  args.RepositoryID,
			PullRequestId: args.PullRequestID,
			ThreadId:      &threadID,
			Project:       args.Project,
		}); err != nil {
			return err
		}
		return nil
	}
	_, err = gitClient.CreateThread(ctx, azgit.CreateThreadArgs{
		CommentThread: threadPayload(body, ThreadStatusClosed, "", 0, 0),
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

func PublishGateVote(ctx context.Context, tokenMode, token string, reviewCtx Context, vote int, client *http.Client) error {
	gitClient, args, err := newGitClient(ctx, tokenMode, token, reviewCtx, client)
	if err != nil {
		return err
	}
	locationClient := newLocationClient(ctx, tokenMode, token, reviewCtx, client)
	connectionData, err := locationClient.GetConnectionData(ctx, location.GetConnectionDataArgs{})
	if err != nil {
		return err
	}
	reviewerID, err := reviewerIDFromConnectionData(connectionData)
	if err != nil {
		return err
	}
	return applyGateVote(ctx, gitClient, args, reviewerID, vote)
}

func applyGateVote(ctx context.Context, client gateVoteClient, args gitClientArgs, reviewerID string, vote int) error {
	reviewer := azgit.IdentityRefWithVote{
		Id:   &reviewerID,
		Vote: &vote,
	}
	_, err := client.CreatePullRequestReviewer(ctx, azgit.CreatePullRequestReviewerArgs{
		Reviewer:      &reviewer,
		RepositoryId:  args.RepositoryID,
		PullRequestId: args.PullRequestID,
		ReviewerId:    &reviewerID,
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

func newLocationClient(ctx context.Context, tokenMode, token string, reviewCtx Context, client *http.Client) location.Client {
	connection := azureConnection(reviewCtx.CollectionURI, tokenMode, token)
	if client == nil {
		return location.NewClient(ctx, connection)
	}
	sdkClient := azuredevops.NewClientWithOptions(connection, connection.BaseUrl, azuredevops.WithHTTPClient(client))
	return &location.ClientImpl{
		Client: *sdkClient,
	}
}

func azureConnection(collectionURI, tokenMode, token string) *azuredevops.Connection {
	if tokenMode == "pat" {
		return azuredevops.NewPatConnection(collectionURI, token)
	}
	connection := azuredevops.NewAnonymousConnection(collectionURI)
	connection.AuthorizationString = "Bearer " + token
	return connection
}

func threadPayload(content string, status ThreadStatus, path string, line int, endLine int) *azgit.GitPullRequestCommentThread {
	commentType := azgit.CommentTypeValues.Text
	parentID := 0
	comments := []azgit.Comment{{
		ParentCommentId: &parentID,
		Content:         &content,
		CommentType:     &commentType,
	}}
	payload := threadStatusPayload(status)
	payload.Comments = &comments
	if strings.TrimSpace(path) != "" && line > 0 {
		if endLine < line {
			endLine = line
		}
		offset := 1
		start := azgit.CommentPosition{
			Line:   &line,
			Offset: &offset,
		}
		end := azgit.CommentPosition{
			Line:   &endLine,
			Offset: &offset,
		}
		payload.ThreadContext = &azgit.CommentThreadContext{
			FilePath:       &path,
			RightFileStart: &start,
			RightFileEnd:   &end,
		}
	}
	return payload
}

func threadPayloadForTarget(action ThreadAction, target resolvedThreadTarget) *azgit.GitPullRequestCommentThread {
	payload := threadPayload(action.Body, action.Status, "", 0, 0)
	startLine := action.Line
	endLine := action.EndLine
	if endLine <= 0 || endLine < startLine {
		endLine = startLine
	}
	offset := 1
	firstIteration := target.IterationID
	secondIteration := target.IterationID
	payload.ThreadContext = &azgit.CommentThreadContext{
		FilePath: &target.FilePath,
		RightFileStart: &azgit.CommentPosition{
			Line:   &startLine,
			Offset: &offset,
		},
		RightFileEnd: &azgit.CommentPosition{
			Line:   &endLine,
			Offset: &offset,
		},
	}
	payload.PullRequestThreadContext = &azgit.GitPullRequestCommentThreadContext{
		ChangeTrackingId: &target.ChangeTrackingID,
		IterationContext: &azgit.CommentIterationContext{
			FirstComparingIteration:  &firstIteration,
			SecondComparingIteration: &secondIteration,
		},
	}
	return payload
}

func hasCanonicalAzureThreadLocation(pathValue string, line int) bool {
	return strings.TrimSpace(pathValue) != "" && line > 0
}

func threadStatusPayload(status ThreadStatus) *azgit.GitPullRequestCommentThread {
	azureStatus := azgit.CommentThreadStatusValues.Active
	if status == ThreadStatusClosed {
		azureStatus = azgit.CommentThreadStatusValues.Closed
	}
	return &azgit.GitPullRequestCommentThread{
		Status: &azureStatus,
	}
}

func reviewerIDFromConnectionData(data *location.ConnectionData) (string, error) {
	if id := identityID(data, true); id != "" {
		return id, nil
	}
	if id := identityID(data, false); id != "" {
		return id, nil
	}
	return "", fmt.Errorf("could not resolve Azure DevOps reviewer identity from connection data")
}

func identityID(data *location.ConnectionData, authorized bool) string {
	if data == nil {
		return ""
	}
	var item *identity.Identity
	if authorized {
		item = data.AuthorizedUser
	} else {
		item = data.AuthenticatedUser
	}
	if item == nil || item.Id == nil {
		return ""
	}
	return item.Id.String()
}

func listPullRequestIterationChanges(ctx context.Context, gitClient threadGitClient, args gitClientArgs) (int, []pullRequestChangeRef, error) {
	iterations, err := gitClient.GetPullRequestIterations(ctx, azgit.GetPullRequestIterationsArgs{
		RepositoryId:  args.RepositoryID,
		PullRequestId: args.PullRequestID,
		Project:       args.Project,
	})
	if err != nil {
		return 0, nil, err
	}
	iterationID := latestIterationID(iterations)
	if iterationID <= 0 {
		return 0, nil, fmt.Errorf("no Azure pull request iterations found")
	}

	top := 2000
	skip := 0
	changes := make([]pullRequestChangeRef, 0)
	for {
		page, err := gitClient.GetPullRequestIterationChanges(ctx, azgit.GetPullRequestIterationChangesArgs{
			RepositoryId:  args.RepositoryID,
			PullRequestId: args.PullRequestID,
			IterationId:   &iterationID,
			Project:       args.Project,
			Top:           &top,
			Skip:          &skip,
		})
		if err != nil {
			return 0, nil, err
		}
		if page != nil && page.ChangeEntries != nil {
			for _, change := range *page.ChangeEntries {
				ref := pullRequestChangeRef{
					Path:             normalizeAzureRepoPath(changePath(change)),
					OriginalPath:     normalizeAzureRepoPath(derefString(change.OriginalPath)),
					ChangeTrackingID: derefInt(change.ChangeTrackingId),
				}
				if ref.Path == "" || ref.ChangeTrackingID <= 0 {
					continue
				}
				changes = append(changes, ref)
			}
		}
		if page == nil || page.NextTop == nil || page.NextSkip == nil || *page.NextTop <= 0 {
			break
		}
		top = *page.NextTop
		skip = *page.NextSkip
	}
	return iterationID, changes, nil
}

func latestIterationID(items *[]azgit.GitPullRequestIteration) int {
	if items == nil {
		return 0
	}
	latest := 0
	for _, item := range *items {
		if item.Id != nil && *item.Id > latest {
			latest = *item.Id
		}
	}
	return latest
}

func resolveThreadTargets(actions []ThreadAction, iterationID int, changes []pullRequestChangeRef) map[string]resolvedThreadTarget {
	current := map[string]pullRequestChangeRef{}
	original := map[string]pullRequestChangeRef{}
	ambiguousCurrent := map[string]struct{}{}
	ambiguousOriginal := map[string]struct{}{}
	for _, change := range changes {
		addThreadChangeIndex(current, ambiguousCurrent, trimAzureRepoPath(change.Path), change)
		addThreadChangeIndex(original, ambiguousOriginal, trimAzureRepoPath(change.OriginalPath), change)
	}

	targets := make(map[string]resolvedThreadTarget, len(actions))
	for _, action := range actions {
		if !hasCanonicalAzureThreadLocation(action.Path, action.Line) {
			continue
		}
		normalized := trimAzureRepoPath(normalizeAzureRepoPath(action.Path))
		if normalized == "" {
			continue
		}
		change, ok := lookupThreadChange(current, ambiguousCurrent, normalized)
		if !ok {
			change, ok = lookupThreadChange(original, ambiguousOriginal, normalized)
		}
		if !ok {
			continue
		}
		targets[action.ThreadID] = resolvedThreadTarget{
			FilePath:         change.Path,
			ChangeTrackingID: change.ChangeTrackingID,
			IterationID:      iterationID,
		}
	}
	return targets
}

func addThreadChangeIndex(index map[string]pullRequestChangeRef, ambiguous map[string]struct{}, key string, value pullRequestChangeRef) {
	if key == "" {
		return
	}
	if _, exists := ambiguous[key]; exists {
		return
	}
	if _, exists := index[key]; exists {
		delete(index, key)
		ambiguous[key] = struct{}{}
		return
	}
	index[key] = value
}

func lookupThreadChange(index map[string]pullRequestChangeRef, ambiguous map[string]struct{}, key string) (pullRequestChangeRef, bool) {
	if _, exists := ambiguous[key]; exists {
		return pullRequestChangeRef{}, false
	}
	value, ok := index[key]
	return value, ok
}

func normalizeAzureRepoPath(raw string) string {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	if raw == "" || raw == "/dev/null" {
		return ""
	}
	cleaned := path.Clean("/" + strings.TrimPrefix(raw, "/"))
	if cleaned == "/." {
		return ""
	}
	return cleaned
}

func trimAzureRepoPath(raw string) string {
	return strings.TrimPrefix(normalizeAzureRepoPath(raw), "/")
}

func changePath(change azgit.GitPullRequestChange) string {
	if item := change.Item; item != nil {
		switch typed := item.(type) {
		case azgit.GitItem:
			return derefString(typed.Path)
		case *azgit.GitItem:
			if typed != nil {
				return derefString(typed.Path)
			}
		case map[string]any:
			if value, ok := typed["path"].(string); ok {
				return value
			}
		}
	}
	return derefString(change.SourceServerItem)
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
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
