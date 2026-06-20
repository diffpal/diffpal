package azure

import (
	"context"
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
