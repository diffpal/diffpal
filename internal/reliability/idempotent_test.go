package reliability

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/diffpal/diffpal/internal/cache"
)

func TestAttemptDeciderReconcileForSameHeadSHA(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	decider := AttemptDecider{Store: store}
	ctx := context.Background()

	decision, err := decider.Reconcile(ctx, "acme/diffpal", "pr-1", "head-a", "fp-a")
	if err != nil {
		t.Fatalf("Reconcile(create) error = %v", err)
	}
	if decision.Action != ActionCreate {
		t.Fatalf("Action = %q, want create", decision.Action)
	}

	if err := decider.MarkPublished(ctx, "pr-1", "acme/diffpal", "base-a", "head-a", "fp-a"); err != nil {
		t.Fatalf("MarkPublished() error = %v", err)
	}

	decision, err = decider.Reconcile(ctx, "acme/diffpal", "pr-1", "head-a", "fp-a")
	if err != nil {
		t.Fatalf("Reconcile(skip) error = %v", err)
	}
	if decision.Action != ActionSkip {
		t.Fatalf("Action = %q, want skip", decision.Action)
	}

	decision, err = decider.Reconcile(ctx, "acme/diffpal", "pr-1", "head-a", "fp-b")
	if err != nil {
		t.Fatalf("Reconcile(update) error = %v", err)
	}
	if decision.Action != ActionUpdate {
		t.Fatalf("Action = %q, want update", decision.Action)
	}
	if decision.PreviousFingerprint != "fp-a" {
		t.Fatalf("PreviousFingerprint = %q, want fp-a", decision.PreviousFingerprint)
	}
}

func TestAttemptDeciderDifferentHeadSHACreatesNewAttempt(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	decider := AttemptDecider{Store: store}
	ctx := context.Background()

	if err := decider.MarkPublished(ctx, "pr-1", "acme/diffpal", "base-a", "head-a", "fp-a"); err != nil {
		t.Fatalf("MarkPublished() error = %v", err)
	}

	decision, err := decider.Reconcile(ctx, "acme/diffpal", "pr-1", "head-b", "fp-b")
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if decision.Action != ActionCreate {
		t.Fatalf("Action = %q, want create", decision.Action)
	}
}

func openStore(t *testing.T) *cache.StateStore {
	t.Helper()

	store, err := cache.Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatalf("cache.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = store.DB.Close() })
	return store
}
