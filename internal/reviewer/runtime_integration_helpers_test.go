//go:build integration

package reviewer

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/diffpal/diffpal/internal/reviewer/promptpack"
)

const providerIntegrationTimeout = 2 * time.Minute

func unsafeHandlerInput() ReviewInput {
	return ReviewInput{
		ReviewID:              "integration-review",
		Repo:                  "diffpal/diffpal",
		BaseSHA:               "base",
		HeadSHA:               "head",
		ReviewTask:            promptpack.ReviewTask([]string{"security"}),
		UntrustedInputWarning: promptpack.UntrustedInputWarning,
		UntrustedInputStart:   promptpack.UntrustedInputStart,
		UntrustedInputEnd:     promptpack.UntrustedInputEnd,
		Language:              "en",
		ReviewChecks:          []string{"security"},
	}
}

func requireCommand(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not found in PATH: %v", name, err)
	}
}

func maybeSkipProviderIntegration(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range []string{
		"401",
		"402",
		"429",
		"api key",
		"authentication",
		"could not determine executable to run",
		"econnrefused",
		"enotfound",
		"etimedout",
		"network",
		"not logged in",
		"npm err!",
		"npm error",
		"openai_api_key",
		"payment required",
		"peer disconnected before response",
		"quota",
		"rate limit",
	} {
		if strings.Contains(msg, marker) {
			t.Skipf("provider integration unavailable in this environment (%s): %v", marker, err)
		}
	}
}
