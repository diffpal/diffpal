//go:build integration && codex

package reviewer

import (
	"context"
	"testing"

	dpconfig "github.com/diffpal/diffpal/internal/config"
	"github.com/normahq/norma/pkg/runtime/agentconfig"
)

func TestADKRuntimeCodexACPReviewFindsUnsafeHandler(t *testing.T) {
	requireCommand(t, "npx")

	ctx, cancel := context.WithTimeout(context.Background(), providerIntegrationTimeout)
	defer cancel()

	input := unsafeHandlerInput()
	output, _, err := ADKRuntime{}.ReviewChunk(ctx, RuntimeConfig{
		ProviderID: "codex-acp",
		Providers: map[string]dpconfig.ProviderConfig{
			"codex-acp": {
				Type: "codex_acp",
				CodexACP: &agentconfig.ACPConfig{
					ReasoningEffort: "low",
				},
			},
		},
		WorkingDir: ".",
		Instructions: "Report directly exploitable security flaws in the provided handler. " +
			"Keep findings tied to the visible changed lines.",
	}, input)
	if err != nil {
		maybeSkipProviderIntegration(t, err)
		t.Fatalf("ReviewChunk(codex_acp) error = %v", err)
	}

	if len(output.Findings) == 0 {
		t.Fatalf("ReviewChunk(codex_acp) returned no findings; summary=%v", output.ChangeSummary)
	}
	valid := validateChunkFindings(output.Findings, input.Files, "codex-acp", categoriesForReviewChecks([]string{"security"}))
	if len(valid) == 0 {
		t.Fatalf("ReviewChunk(codex_acp) returned no valid security findings: %+v", output.Findings)
	}
}
