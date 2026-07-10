package platformapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultTimeout = 60 * time.Second

// Error describes a failed platform API operation.
type Error struct {
	Method     string
	URL        string
	StatusCode int
	Body       string
	Err        error
}

func (e *Error) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("platform api %s %s failed: status=%d body=%s", e.Method, e.URL, e.StatusCode, e.Body)
	}
	return fmt.Sprintf("platform api %s %s failed: %v", e.Method, e.URL, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

func DefaultClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: defaultTimeout}
}

func DoJSON(ctx context.Context, client *http.Client, method, url string, headers map[string]string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		if strings.TrimSpace(value) == "" {
			continue
		}
		req.Header.Set(key, value)
	}
	resp, err := DefaultClient(client).Do(req)
	if err != nil {
		return &Error{Method: method, URL: url, Err: err}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = resp.Status
	}
	return &Error{Method: method, URL: url, StatusCode: resp.StatusCode, Body: msg}
}
