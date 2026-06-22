package azure

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	azgit "github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

type fakeGateVoteClient struct {
	args azgit.CreatePullRequestReviewerArgs
}

func (f *fakeGateVoteClient) CreatePullRequestReviewer(_ context.Context, args azgit.CreatePullRequestReviewerArgs) (*azgit.IdentityRefWithVote, error) {
	f.args = args
	return args.Reviewer, nil
}

func TestApplyGateVoteUsesReviewerUpsert(t *testing.T) {
	t.Parallel()

	client := &fakeGateVoteClient{}
	repositoryID := "repo-1"
	pullRequestID := 55
	project := "proj"
	args := gitClientArgs{
		RepositoryID:  &repositoryID,
		PullRequestID: &pullRequestID,
		Project:       &project,
	}

	err := applyGateVote(context.Background(), client, args, "11111111-1111-1111-1111-111111111111", -5)
	if err != nil {
		t.Fatalf("applyGateVote() error = %v", err)
	}
	if client.args.ReviewerId == nil || *client.args.ReviewerId != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("ReviewerId = %v, want reviewer UUID", client.args.ReviewerId)
	}
	if client.args.Reviewer == nil || client.args.Reviewer.Vote == nil || *client.args.Reviewer.Vote != -5 {
		t.Fatalf("Reviewer vote = %#v, want -5", client.args.Reviewer)
	}
	if client.args.RepositoryId == nil || *client.args.RepositoryId != repositoryID {
		t.Fatalf("RepositoryId = %v, want %q", client.args.RepositoryId, repositoryID)
	}
	if client.args.PullRequestId == nil || *client.args.PullRequestId != pullRequestID {
		t.Fatalf("PullRequestId = %v, want %d", client.args.PullRequestId, pullRequestID)
	}
}

func TestResolveThreadTargetsMatchesCurrentPath(t *testing.T) {
	t.Parallel()

	iterationID := 7
	threadID := "internal/app/service.go:10:correctness:fp-a"
	targets := resolveThreadTargets([]ThreadAction{{
		ThreadID: threadID,
		Path:     "internal/app/service.go",
		Line:     10,
		EndLine:  12,
	}}, iterationID, []pullRequestChangeRef{{
		Path:             "/internal/app/service.go",
		ChangeTrackingID: 41,
	}})

	target, ok := targets[threadID]
	if !ok {
		t.Fatal("resolveThreadTargets() did not return a target")
	}
	if target.FilePath != "/internal/app/service.go" {
		t.Fatalf("FilePath = %q, want /internal/app/service.go", target.FilePath)
	}
	if target.ChangeTrackingID != 41 {
		t.Fatalf("ChangeTrackingID = %d, want 41", target.ChangeTrackingID)
	}
	if target.IterationID != iterationID {
		t.Fatalf("IterationID = %d, want %d", target.IterationID, iterationID)
	}
}

func TestResolveThreadTargetsFallsBackToOriginalPathForRename(t *testing.T) {
	t.Parallel()

	threadID := "internal/old/name.go:8:correctness:fp-b"
	targets := resolveThreadTargets([]ThreadAction{{
		ThreadID: threadID,
		Path:     "internal/old/name.go",
		Line:     8,
		EndLine:  8,
	}}, 9, []pullRequestChangeRef{{
		Path:             "/internal/new/name.go",
		OriginalPath:     "/internal/old/name.go",
		ChangeTrackingID: 73,
	}})

	target, ok := targets[threadID]
	if !ok {
		t.Fatal("resolveThreadTargets() did not return a rename target")
	}
	if target.FilePath != "/internal/new/name.go" {
		t.Fatalf("FilePath = %q, want /internal/new/name.go", target.FilePath)
	}
	if target.ChangeTrackingID != 73 {
		t.Fatalf("ChangeTrackingID = %d, want 73", target.ChangeTrackingID)
	}
}

func TestThreadPayloadForTargetIncludesAzureFileContext(t *testing.T) {
	t.Parallel()

	payload := threadPayloadForTarget(ThreadAction{
		Body:    "body",
		Status:  ThreadStatusActive,
		Path:    "internal/app/service.go",
		Line:    15,
		EndLine: 18,
	}, resolvedThreadTarget{
		FilePath:         "/internal/app/service.go",
		ChangeTrackingID: 55,
		IterationID:      11,
	})

	if payload.ThreadContext == nil || payload.ThreadContext.FilePath == nil {
		t.Fatal("ThreadContext.FilePath is nil")
	}
	if got := *payload.ThreadContext.FilePath; got != "/internal/app/service.go" {
		t.Fatalf("ThreadContext.FilePath = %q, want /internal/app/service.go", got)
	}
	if payload.ThreadContext.RightFileStart == nil || payload.ThreadContext.RightFileStart.Line == nil || *payload.ThreadContext.RightFileStart.Line != 15 {
		t.Fatalf("RightFileStart.Line = %#v, want 15", payload.ThreadContext.RightFileStart)
	}
	if payload.ThreadContext.RightFileStart.Offset == nil || *payload.ThreadContext.RightFileStart.Offset != 1 {
		t.Fatalf("RightFileStart.Offset = %#v, want 1", payload.ThreadContext.RightFileStart)
	}
	if payload.ThreadContext.RightFileEnd == nil || payload.ThreadContext.RightFileEnd.Line == nil || *payload.ThreadContext.RightFileEnd.Line != 18 {
		t.Fatalf("RightFileEnd.Line = %#v, want 18", payload.ThreadContext.RightFileEnd)
	}
	if payload.ThreadContext.RightFileEnd.Offset == nil || *payload.ThreadContext.RightFileEnd.Offset != 1 {
		t.Fatalf("RightFileEnd.Offset = %#v, want 1", payload.ThreadContext.RightFileEnd)
	}
	if payload.PullRequestThreadContext == nil || payload.PullRequestThreadContext.ChangeTrackingId == nil || *payload.PullRequestThreadContext.ChangeTrackingId != 55 {
		t.Fatalf("ChangeTrackingId = %#v, want 55", payload.PullRequestThreadContext)
	}
	if payload.PullRequestThreadContext.IterationContext == nil {
		t.Fatal("IterationContext is nil")
	}
	if got := payload.PullRequestThreadContext.IterationContext.FirstComparingIteration; got == nil || *got != 11 {
		t.Fatalf("FirstComparingIteration = %#v, want 11", got)
	}
	if got := payload.PullRequestThreadContext.IterationContext.SecondComparingIteration; got == nil || *got != 11 {
		t.Fatalf("SecondComparingIteration = %#v, want 11", got)
	}
}

func TestPublishThreadsUsesResolvedAzureFileContext(t *testing.T) {
	t.Parallel()

	repoID := "repo-1"
	project := "proj"
	prID := 19
	client := &fakeThreadGitClient{
		iterations: []azgit.GitPullRequestIteration{{Id: intPtr(3)}},
		changes: []pullRequestChangeRef{{
			Path:             "/internal/app/service.go",
			ChangeTrackingID: 91,
		}},
	}

	err := publishThreadsWithClient(context.Background(), client, gitClientArgs{
		RepositoryID:  &repoID,
		PullRequestID: &prID,
		Project:       &project,
	}, ThreadPlan{
		Actions: []ThreadAction{{
			Type:     ActionCreate,
			ThreadID: "internal/app/service.go:7:correctness:fp-a",
			Status:   ThreadStatusActive,
			Path:     "internal/app/service.go",
			Line:     7,
			EndLine:  9,
			Body:     "body",
		}},
	})
	if err != nil {
		t.Fatalf("publishThreadsWithClient() error = %v", err)
	}
	if client.createThreadCalls != 1 {
		t.Fatalf("CreateThread calls = %d, want 1", client.createThreadCalls)
	}
	if client.lastThread == nil || client.lastThread.ThreadContext == nil || client.lastThread.ThreadContext.FilePath == nil {
		t.Fatal("last thread missing ThreadContext.FilePath")
	}
	if got := *client.lastThread.ThreadContext.FilePath; got != "/internal/app/service.go" {
		t.Fatalf("ThreadContext.FilePath = %q, want /internal/app/service.go", got)
	}
	if got := client.lastThread.ThreadContext.RightFileStart; got == nil || got.Line == nil || *got.Line != 7 || got.Offset == nil || *got.Offset != 1 {
		t.Fatalf("RightFileStart = %#v, want line 7 offset 1", got)
	}
	if got := client.lastThread.ThreadContext.RightFileEnd; got == nil || got.Line == nil || *got.Line != 9 || got.Offset == nil || *got.Offset != 1 {
		t.Fatalf("RightFileEnd = %#v, want line 9 offset 1", got)
	}
	if client.lastThread.PullRequestThreadContext == nil || client.lastThread.PullRequestThreadContext.ChangeTrackingId == nil || *client.lastThread.PullRequestThreadContext.ChangeTrackingId != 91 {
		t.Fatalf("ChangeTrackingId = %#v, want 91", client.lastThread.PullRequestThreadContext)
	}
	if client.lastThread.PullRequestThreadContext.IterationContext == nil {
		t.Fatal("IterationContext is nil")
	}
	if got := client.lastThread.PullRequestThreadContext.IterationContext.FirstComparingIteration; got == nil || *got != 3 {
		t.Fatalf("FirstComparingIteration = %#v, want 3", got)
	}
	if got := client.lastThread.PullRequestThreadContext.IterationContext.SecondComparingIteration; got == nil || *got != 3 {
		t.Fatalf("SecondComparingIteration = %#v, want 3", got)
	}
	raw, err := json.Marshal(client.lastThread)
	if err != nil {
		t.Fatalf("json.Marshal(lastThread) error = %v", err)
	}
	payloadJSON := string(raw)
	if !strings.Contains(payloadJSON, `"offset":1`) {
		t.Fatalf("serialized thread payload does not include offset 1: %s", payloadJSON)
	}
	if strings.Contains(payloadJSON, `"offset":0`) {
		t.Fatalf("serialized thread payload includes invalid offset 0: %s", payloadJSON)
	}
}

func TestPublishThreadsFailsUnmatchedInlineThread(t *testing.T) {
	t.Parallel()

	repoID := "repo-1"
	project := "proj"
	prID := 19
	client := &fakeThreadGitClient{
		iterations: []azgit.GitPullRequestIteration{{Id: intPtr(3)}},
		changes:    []pullRequestChangeRef{},
	}

	err := publishThreadsWithClient(context.Background(), client, gitClientArgs{
		RepositoryID:  &repoID,
		PullRequestID: &prID,
		Project:       &project,
	}, ThreadPlan{
		Actions: []ThreadAction{{
			Type:     ActionCreate,
			ThreadID: "missing.go:7:correctness:fp-a",
			Status:   ThreadStatusActive,
			Path:     "missing.go",
			Line:     7,
			EndLine:  7,
			Body:     "body",
		}},
	})
	if err == nil {
		t.Fatal("publishThreadsWithClient() error = nil, want unmatched inline target error")
	}
	if !strings.Contains(err.Error(), "azure inline thread target not found for missing.go:7") {
		t.Fatalf("publishThreadsWithClient() error = %v, want unmatched inline target", err)
	}
	if client.createThreadCalls != 0 {
		t.Fatalf("CreateThread calls = %d, want 0", client.createThreadCalls)
	}
}

func TestPublishThreadsDoesNotFetchChangesForFallbackOnlyPlan(t *testing.T) {
	t.Parallel()

	repoID := "repo-1"
	project := "proj"
	prID := 19
	client := &fakeThreadGitClient{
		iterationErr: errors.New("iteration lookup should not run"),
	}

	err := publishThreadsWithClient(context.Background(), client, gitClientArgs{
		RepositoryID:  &repoID,
		PullRequestID: &prID,
		Project:       &project,
	}, ThreadPlan{
		Actions: []ThreadAction{
			{
				Type:     ActionSkip,
				ThreadID: "internal/app/service.go:7:correctness:fp-a",
				Status:   ThreadStatusActive,
				Path:     "internal/app/service.go",
				Line:     7,
				EndLine:  7,
				Body:     "body",
			},
			{
				Type:     ActionCreate,
				ThreadID: fallbackAdvisoryThreadID,
				Status:   ThreadStatusClosed,
				Body:     "fallback body",
			},
		},
	})
	if err != nil {
		t.Fatalf("publishThreadsWithClient() error = %v", err)
	}
	if client.iterationCalls != 0 {
		t.Fatalf("GetPullRequestIterations calls = %d, want 0", client.iterationCalls)
	}
	if client.changeCalls != 0 {
		t.Fatalf("GetPullRequestIterationChanges calls = %d, want 0", client.changeCalls)
	}
	if client.createThreadCalls != 1 {
		t.Fatalf("CreateThread calls = %d, want 1", client.createThreadCalls)
	}
}

type fakeThreadGitClient struct {
	iterations        []azgit.GitPullRequestIteration
	changes           []pullRequestChangeRef
	iterationErr      error
	changeErr         error
	iterationCalls    int
	changeCalls       int
	createThreadCalls int
	lastThread        *azgit.GitPullRequestCommentThread
}

func (f *fakeThreadGitClient) CreateThread(_ context.Context, args azgit.CreateThreadArgs) (*azgit.GitPullRequestCommentThread, error) {
	f.createThreadCalls++
	f.lastThread = args.CommentThread
	return args.CommentThread, nil
}

func (f *fakeThreadGitClient) GetPullRequestIterations(_ context.Context, _ azgit.GetPullRequestIterationsArgs) (*[]azgit.GitPullRequestIteration, error) {
	f.iterationCalls++
	if f.iterationErr != nil {
		return nil, f.iterationErr
	}
	items := append([]azgit.GitPullRequestIteration(nil), f.iterations...)
	return &items, nil
}

func (f *fakeThreadGitClient) GetPullRequestIterationChanges(_ context.Context, _ azgit.GetPullRequestIterationChangesArgs) (*azgit.GitPullRequestIterationChanges, error) {
	f.changeCalls++
	if f.changeErr != nil {
		return nil, f.changeErr
	}
	entries := make([]azgit.GitPullRequestChange, 0, len(f.changes))
	for _, change := range f.changes {
		path := change.Path
		originalPath := change.OriginalPath
		changeTrackingID := change.ChangeTrackingID
		item := azgit.GitItem{Path: &path}
		entry := azgit.GitPullRequestChange{
			Item:             item,
			ChangeTrackingId: &changeTrackingID,
		}
		if originalPath != "" {
			entry.OriginalPath = &originalPath
		}
		entries = append(entries, entry)
	}
	return &azgit.GitPullRequestIterationChanges{ChangeEntries: &entries}, nil
}

func intPtr(v int) *int {
	return &v
}
