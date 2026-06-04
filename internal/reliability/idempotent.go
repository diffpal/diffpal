package reliability

import (
	"context"
	"fmt"

	"github.com/diffpal/diffpal/internal/cache"
)

type AttemptDecider struct {
	Store *cache.StateStore
}

type PublishAction string

const (
	ActionCreate PublishAction = "create"
	ActionUpdate PublishAction = "update"
	ActionSkip   PublishAction = "skip"
)

type ReconcileDecision struct {
	Action              PublishAction
	CurrentFingerprint  string
	PreviousFingerprint string
}

func (a AttemptDecider) IsAlreadyProcessed(ctx context.Context, repo, reviewID, headSHA, currentFingerprint string) (bool, error) {
	state, err := a.Store.Get(ctx, repo, reviewID, headSHA)
	if err != nil {
		return false, nil
	}
	if state.Fingerprint == currentFingerprint {
		return true, nil
	}
	return false, nil
}

func (a AttemptDecider) Reconcile(ctx context.Context, repo, reviewID, headSHA, currentFingerprint string) (ReconcileDecision, error) {
	state, err := a.Store.Get(ctx, repo, reviewID, headSHA)
	if err != nil {
		return ReconcileDecision{
			Action:             ActionCreate,
			CurrentFingerprint: currentFingerprint,
		}, nil
	}
	if state.Fingerprint == currentFingerprint {
		return ReconcileDecision{
			Action:              ActionSkip,
			CurrentFingerprint:  currentFingerprint,
			PreviousFingerprint: state.Fingerprint,
		}, nil
	}
	return ReconcileDecision{
		Action:              ActionUpdate,
		CurrentFingerprint:  currentFingerprint,
		PreviousFingerprint: state.Fingerprint,
	}, nil
}

func (a AttemptDecider) MarkPublished(ctx context.Context, reviewID, repo, baseSHA, headSHA, fingerprint string) error {
	return a.Store.Upsert(ctx, cache.ReviewState{
		ReviewID:    reviewID,
		Repo:        repo,
		BaseSHA:     baseSHA,
		HeadSHA:     headSHA,
		Fingerprint: fingerprint,
	})
}

func (a AttemptDecider) DiffFingerprint(reviewID, repo, baseSHA, headSHA, path string) (string, error) {
	if reviewID == "" || repo == "" || headSHA == "" {
		return "", fmt.Errorf("missing namespace fields")
	}
	return repo + "|" + reviewID + "|" + baseSHA + "|" + headSHA + "|" + path, nil
}
