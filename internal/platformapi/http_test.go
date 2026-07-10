package platformapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDefaultClientHasBoundedTimeout(t *testing.T) {
	t.Parallel()

	client := DefaultClient(nil)
	if client.Timeout != defaultTimeout {
		t.Fatalf("Timeout = %s, want %s", client.Timeout, defaultTimeout)
	}
}

func TestDoJSONReturnsStructuredHTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(strings.Repeat("x", 9000)))
	}))
	t.Cleanup(server.Close)

	err := DoJSON(context.Background(), nil, http.MethodPost, server.URL, nil, map[string]string{"value": "x"})
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("DoJSON() error = %T, want *Error", err)
	}
	if apiErr.StatusCode != http.StatusBadGateway || apiErr.Method != http.MethodPost || apiErr.URL != server.URL {
		t.Fatalf("structured error = %#v", apiErr)
	}
	if len(apiErr.Body) != 8192 {
		t.Fatalf("body length = %d, want 8192", len(apiErr.Body))
	}
}

func TestDoJSONRespectsContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := DoJSON(ctx, nil, http.MethodPost, server.URL, nil, struct{}{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("DoJSON() error = %v, want context deadline", err)
	}
}
