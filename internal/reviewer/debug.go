package reviewer

import (
	"context"

	dpconfig "github.com/diffpal/diffpal/internal/config"
	"github.com/diffpal/diffpal/internal/findings"
)

type DebugResult struct {
	SystemPrompt string
	TaskSnapshot string
	Bundle       findings.FindingsBundle
}

func DebugPrompt(ctx context.Context, cfg dpconfig.Config, opts Options) (DebugResult, error) {
	runtime := &debugRuntime{}
	result, err := RunWithRuntime(ctx, cfg, opts, runtime)
	if err != nil {
		return DebugResult{}, err
	}
	instructions := opts.Instructions
	if instructions == "" {
		instructions = cfg.ReviewInstructions()
	}
	return DebugResult{
		SystemPrompt: reviewSystemInstruction(instructions),
		TaskSnapshot: runtime.taskSnapshot,
		Bundle:       result.Bundle,
	}, nil
}

type debugRuntime struct {
	taskSnapshot string
}

func (r *debugRuntime) ReviewChunk(_ context.Context, cfg RuntimeConfig, input ChunkInput) (ChunkOutput, RuntimeUsage, error) {
	r.taskSnapshot = renderReviewTaskInput(input)
	return ChunkOutput{
			ChangeSummary: []string{"Debug harness rendered the review task without contacting a provider."},
			Findings:      []ChunkFinding{},
		},
		RuntimeUsage{Inspection: debugInspection(cfg)},
		nil
}

func debugInspection(cfg RuntimeConfig) *findings.Inspection {
	inspection := inspectionForProvider(cfg.ProviderID, cfg.Providers)
	if inspection == nil || !inspection.Required {
		return inspection
	}
	inspection.ToolCalls = []string{"git_diff"}
	inspection.DiffInspected = true
	return inspection
}
