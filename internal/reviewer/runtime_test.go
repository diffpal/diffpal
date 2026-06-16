package reviewer

import (
	"strings"
	"testing"
)

func TestReviewSystemInstructionIsAppliedByStructuredWrapperOnly(t *testing.T) {
	t.Parallel()

	system := reviewSystemInstruction("Prefer auth findings.")
	if !strings.Contains(system, "Prefer auth findings.") {
		t.Fatalf("reviewSystemInstruction() = %q, want custom instructions", system)
	}

	req := reviewAgentBuildRequest(RuntimeConfig{
		ProviderID: "codex-acp",
		WorkingDir: "/repo",
	}, nil)
	if req.GlobalInstruction != "" {
		t.Fatalf("GlobalInstruction = %q, want empty; structured wrapper owns review system prompt", req.GlobalInstruction)
	}
	if req.Instruction != "" {
		t.Fatalf("Instruction = %q, want empty; structured wrapper owns review system prompt", req.Instruction)
	}
}
