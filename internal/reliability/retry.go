package reliability

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"
)

type Policy struct {
	Attempts    int
	BaseDelay   time.Duration
	Timeout     time.Duration
	IsTransient func(error) bool
}

func Retry(ctx context.Context, attempts int, baseDelay time.Duration, fn func() error) error {
	return RetryWithPolicy(ctx, Policy{
		Attempts:  attempts,
		BaseDelay: baseDelay,
	}, func(context.Context) error {
		return fn()
	})
}

func RetryWithPolicy(ctx context.Context, policy Policy, fn func(context.Context) error) error {
	if policy.Attempts <= 0 {
		policy.Attempts = 1
	}
	if policy.IsTransient == nil {
		policy.IsTransient = IsTransient
	}

	var lastErr error
	for n := 1; n <= policy.Attempts; n++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		attemptCtx := ctx
		cancel := func() {}
		if policy.Timeout > 0 {
			attemptCtx, cancel = context.WithTimeout(ctx, policy.Timeout)
		}
		lastErr = fn(attemptCtx)
		cancel()
		if lastErr == nil {
			return nil
		}
		if n == policy.Attempts || !policy.IsTransient(lastErr) {
			return lastErr
		}
		jitter := policy.BaseDelay * time.Duration(n)
		select {
		case <-time.After(jitter):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return lastErr
}

func IsTransient(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range []string{
		"timeout",
		"temporarily unavailable",
		"temporary failure",
		"rate limit",
		"too many requests",
		"429",
		"502",
		"503",
		"504",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func Batch[T any](values []T, size int) [][]T {
	if size <= 0 {
		return [][]T{values}
	}
	out := [][]T{}
	for i := 0; i < len(values); i += size {
		end := i + size
		if end > len(values) {
			end = len(values)
		}
		out = append(out, values[i:end])
	}
	return out
}
