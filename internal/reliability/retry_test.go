package reliability

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryWithPolicyRetriesTransientErrors(t *testing.T) {
	t.Parallel()

	attempts := 0
	err := RetryWithPolicy(context.Background(), Policy{
		Attempts:  3,
		BaseDelay: time.Millisecond,
		Timeout:   50 * time.Millisecond,
	}, func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("rate limit exceeded")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RetryWithPolicy() error = %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestRetryWithPolicyStopsOnNonTransientError(t *testing.T) {
	t.Parallel()

	attempts := 0
	err := RetryWithPolicy(context.Background(), Policy{
		Attempts:  3,
		BaseDelay: time.Millisecond,
	}, func(ctx context.Context) error {
		attempts++
		return errors.New("validation failed")
	})
	if err == nil {
		t.Fatal("RetryWithPolicy() error = nil, want non-transient failure")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestRetryWithPolicyRespectsAttemptTimeout(t *testing.T) {
	t.Parallel()

	attempts := 0
	err := RetryWithPolicy(context.Background(), Policy{
		Attempts:  2,
		BaseDelay: time.Millisecond,
		Timeout:   5 * time.Millisecond,
	}, func(ctx context.Context) error {
		attempts++
		<-ctx.Done()
		return ctx.Err()
	})
	if err == nil {
		t.Fatal("RetryWithPolicy() error = nil, want deadline exceeded")
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestBatchSplitsValuesBySize(t *testing.T) {
	t.Parallel()

	batches := Batch([]int{1, 2, 3, 4, 5}, 2)
	if len(batches) != 3 {
		t.Fatalf("len(batches) = %d, want 3", len(batches))
	}
	if len(batches[0]) != 2 || len(batches[1]) != 2 || len(batches[2]) != 1 {
		t.Fatalf("unexpected batch sizes: %#v", batches)
	}
}
