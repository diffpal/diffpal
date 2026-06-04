package cache

import (
	"context"
	"path/filepath"
	"testing"
)

func TestStateStoreNamespacesByRepoReviewAndHeadSHA(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := context.Background()

	first := ReviewState{
		Repo:        "acme/diffpal",
		ReviewID:    "pr-101",
		BaseSHA:     "base-1",
		HeadSHA:     "head-1",
		Fingerprint: "fp-1",
	}
	second := ReviewState{
		Repo:        "acme/diffpal",
		ReviewID:    "pr-101",
		BaseSHA:     "base-2",
		HeadSHA:     "head-2",
		Fingerprint: "fp-2",
	}
	third := ReviewState{
		Repo:        "acme/other",
		ReviewID:    "pr-101",
		BaseSHA:     "base-3",
		HeadSHA:     "head-2",
		Fingerprint: "fp-3",
	}

	for _, item := range []ReviewState{first, second, third} {
		if err := store.Upsert(ctx, item); err != nil {
			t.Fatalf("Upsert(%+v) error = %v", item, err)
		}
	}

	got, err := store.Get(ctx, "acme/diffpal", "pr-101", "head-2")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Fingerprint != "fp-2" {
		t.Fatalf("Fingerprint = %q, want fp-2", got.Fingerprint)
	}

	otherRepo, err := store.Get(ctx, "acme/other", "pr-101", "head-2")
	if err != nil {
		t.Fatalf("Get(other repo) error = %v", err)
	}
	if otherRepo.Fingerprint != "fp-3" {
		t.Fatalf("Fingerprint(other repo) = %q, want fp-3", otherRepo.Fingerprint)
	}
}

func openTestStore(t *testing.T) *StateStore {
	t.Helper()

	path := filepath.Join(t.TempDir(), "state.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = store.DB.Close() })
	return store
}
