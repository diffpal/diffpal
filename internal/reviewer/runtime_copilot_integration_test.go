//go:build integration && copilot

package reviewer

import (
	"context"
	"errors"
	"strings"
	"testing"

	dpconfig "github.com/diffpal/diffpal/internal/config"
	"github.com/normahq/norma/pkg/runtime/agentconfig"
)

func TestADKRuntimeCopilotACPProviderErrorPath(t *testing.T) {
	requireCommand(t, "copilot")

	ctx, cancel := context.WithTimeout(context.Background(), providerIntegrationTimeout)
	defer cancel()

	_, _, err := ADKRuntime{}.ReviewChunk(ctx, RuntimeConfig{
		ProviderID: "copilot-acp",
		Providers: map[string]dpconfig.ProviderConfig{
			"copilot-acp": {
				Type:       "copilot_acp",
				CopilotACP: &agentconfig.ACPConfig{Model: "definitely-invalid-model"},
			},
		},
		WorkingDir: ".",
	}, unsafeHandlerInput())
	if err == nil {
		t.Fatal("ReviewChunk(copilot_acp invalid model) error = nil, want provider error")
	}

	var reviewErr *Error
	if !errors.As(err, &reviewErr) {
		t.Fatalf("ReviewChunk(copilot_acp) error type = %T, want *reviewer.Error: %v", err, err)
	}
	if reviewErr.Kind != KindTransient && reviewErr.Kind != KindInternal {
		t.Fatalf("ReviewChunk(copilot_acp) error kind = %q, want transient or internal: %v", reviewErr.Kind, err)
	}
	if strings.TrimSpace(reviewErr.Error()) == "" {
		t.Fatal("ReviewChunk(copilot_acp) returned empty provider error")
	}
}
