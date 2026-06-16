package reviewer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	dpconfig "github.com/diffpal/diffpal/internal/config"
	"github.com/diffpal/diffpal/internal/findings"
	"github.com/diffpal/diffpal/internal/reliability"
	"github.com/diffpal/diffpal/internal/reviewer/promptpack"
	"github.com/normahq/norma/pkg/runtime/agentfactory"
	"github.com/normahq/norma/pkg/runtime/mcpregistry"
	"github.com/normahq/norma/pkg/runtime/structuredagent"
	adkagent "google.golang.org/adk/agent"
	adkrunner "google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

type ADKRuntime struct{}

func (ADKRuntime) ReviewChunk(ctx context.Context, cfg RuntimeConfig, input ChunkInput) (ChunkOutput, RuntimeUsage, error) {
	if strings.TrimSpace(cfg.ProviderID) == "" {
		return ChunkOutput{}, RuntimeUsage{}, wrapError(KindConfig, fmt.Errorf("provider id is required"))
	}
	providerCfg, ok := cfg.Providers[cfg.ProviderID]
	if !ok {
		return ChunkOutput{}, RuntimeUsage{}, wrapError(KindConfig, fmt.Errorf("unknown provider %q", cfg.ProviderID))
	}

	if err := validateHostedProviderConfig(providerCfg); err != nil {
		return ChunkOutput{}, RuntimeUsage{}, wrapError(KindConfig, err)
	}

	factory := agentfactory.New(cfg.Providers, mcpregistry.New(cfg.MCPServers))
	sessionState, err := factory.BuildSessionState(cfg.ProviderID, cfg.WorkingDir)
	if err != nil {
		return ChunkOutput{}, RuntimeUsage{}, wrapError(KindConfig, err)
	}
	reviewTools, inspectionTracker, err := reviewToolsForProvider(providerCfg, cfg)
	if err != nil {
		return ChunkOutput{}, RuntimeUsage{}, wrapError(KindConfig, err)
	}

	agentRuntime, err := factory.Build(ctx, reviewAgentBuildRequest(cfg, reviewTools))
	if err != nil {
		return ChunkOutput{}, RuntimeUsage{}, wrapError(KindConfig, err)
	}
	if closer, ok := agentRuntime.(interface{ Close() error }); ok {
		defer func() { _ = closer.Close() }()
	}

	wrapped, err := structuredagent.NewAgent(agentRuntime,
		structuredagent.WithoutInputSchema(),
		structuredagent.WithOutputSchema(promptpack.OutputSchemaJSON),
		structuredagent.WithSystemInstruction(reviewSystemInstruction(cfg.Instructions)),
		structuredagent.WithOutputValidationRetries(1),
	)
	if err != nil {
		return ChunkOutput{}, RuntimeUsage{}, wrapError(KindInternal, err)
	}

	sessionService := session.InMemoryService()
	runner, err := adkrunner.New(adkrunner.Config{
		AppName:        "diffpal",
		Agent:          wrapped,
		SessionService: sessionService,
	})
	if err != nil {
		return ChunkOutput{}, RuntimeUsage{}, wrapError(KindInternal, fmt.Errorf("create adk runner: %w", err))
	}

	userID := "diffpal-user"
	created, err := sessionService.Create(ctx, &session.CreateRequest{
		AppName: "diffpal",
		UserID:  userID,
		State:   sessionState,
	})
	if err != nil {
		return ChunkOutput{}, RuntimeUsage{}, wrapError(KindInternal, fmt.Errorf("create review session: %w", err))
	}

	userContent := genai.NewContentFromText(renderReviewTaskInput(input), genai.RoleUser)
	events := runner.Run(ctx, userID, created.Session.ID(), userContent, adkagent.RunConfig{})

	var outputText strings.Builder
	var usage RuntimeUsage
	for ev, runErr := range events {
		if runErr != nil {
			if isTransientProviderError(runErr) {
				return ChunkOutput{}, usage, wrapError(KindTransient, runErr)
			}
			return ChunkOutput{}, usage, wrapError(KindInternal, runErr)
		}
		if ev == nil {
			continue
		}
		if ev.UsageMetadata != nil {
			usage.TokenUsage += int64(ev.UsageMetadata.TotalTokenCount)
		}
		appendVisibleText(&outputText, ev.Content)
	}

	trimmed := strings.TrimSpace(outputText.String())
	if trimmed == "" {
		return ChunkOutput{}, usage, wrapError(KindInternal, fmt.Errorf("provider returned no structured output"))
	}

	var output ChunkOutput
	if err := json.Unmarshal([]byte(trimmed), &output); err != nil {
		return ChunkOutput{}, usage, wrapError(KindInternal, fmt.Errorf("parse structured output: %w", err))
	}
	usage.Inspection = inspectionFromTracker(providerCfg.Type, inspectionTracker)
	return output, usage, nil
}

func reviewSystemInstruction(instructions string) string {
	return promptpack.RenderReviewSystem(promptpack.ReviewOptions{Instructions: instructions})
}

func reviewAgentBuildRequest(cfg RuntimeConfig, reviewTools []tool.Tool) agentfactory.BuildRequest {
	return agentfactory.BuildRequest{
		AgentID:          cfg.ProviderID,
		Name:             "DiffPalReviewerAgent",
		Description:      "DiffPal provider-backed review agent",
		WorkingDirectory: cfg.WorkingDir,
		Tools:            reviewTools,
	}
}

func validateHostedProviderConfig(cfg dpconfig.ProviderConfig) error {
	switch strings.ToLower(strings.TrimSpace(cfg.Type)) {
	case "openai":
		if cfg.OpenAI == nil {
			return fmt.Errorf("openai block is required")
		}
		if strings.TrimSpace(cfg.OpenAI.Model) == "" {
			return fmt.Errorf("openai.model is required")
		}
		if strings.TrimSpace(cfg.OpenAI.APIKey) == "" {
			return fmt.Errorf("openai.api_key is required or OPENAI_API_KEY must be set")
		}
	case "aistudio":
		if cfg.AIStudio == nil {
			return fmt.Errorf("aistudio block is required")
		}
		if strings.TrimSpace(cfg.AIStudio.Model) == "" {
			return fmt.Errorf("aistudio.model is required")
		}
		if strings.TrimSpace(cfg.AIStudio.APIKey) == "" {
			return fmt.Errorf("aistudio.api_key is required or GEMINI_API_KEY must be set")
		}
	}
	return nil
}

func reviewToolsForProvider(providerCfg dpconfig.ProviderConfig, cfg RuntimeConfig) ([]tool.Tool, *inspectionTracker, error) {
	switch strings.ToLower(strings.TrimSpace(providerCfg.Type)) {
	case "openai", "aistudio":
		tracker := newInspectionTracker()
		tools, err := newReviewTools(reviewToolOptions{
			Root:         cfg.WorkingDir,
			BaseSHA:      cfg.BaseSHA,
			HeadSHA:      cfg.HeadSHA,
			ChangedFiles: cfg.ChangedFiles,
			Inspection:   tracker,
		})
		if err != nil {
			return nil, nil, err
		}
		return tools, tracker, nil
	default:
		return nil, nil, nil
	}
}

func inspectionFromTracker(providerType string, tracker *inspectionTracker) *findings.Inspection {
	providerType = strings.ToLower(strings.TrimSpace(providerType))
	required := providerType == "openai" || providerType == "aistudio"
	if !required {
		return &findings.Inspection{ProviderType: providerType, Required: false}
	}
	if tracker == nil {
		return &findings.Inspection{ProviderType: providerType, Required: true}
	}
	return &findings.Inspection{
		ProviderType:     providerType,
		Required:         true,
		ToolCalls:        tracker.callsList(),
		DiffInspected:    tracker.called("git_diff"),
		ContextInspected: tracker.called("read_file") || tracker.called("list_files") || tracker.called("search_files") || tracker.called("git_changed_files"),
	}
}

func renderReviewTaskInput(input ChunkInput) string {
	var out strings.Builder
	fmt.Fprintf(&out, "DiffPal review task snapshot\n\n")
	fmt.Fprintf(&out, "%s\n", promptpack.TrustedControlStart)
	fmt.Fprintf(&out, "Review ID: %s\n", promptpack.EscapeUntrustedField(input.ReviewID))
	fmt.Fprintf(&out, "Repository: %s\n", promptpack.EscapeUntrustedField(input.Repo))
	fmt.Fprintf(&out, "Base: %s\n", promptpack.EscapeUntrustedField(input.BaseSHA))
	fmt.Fprintf(&out, "Head: %s\n", promptpack.EscapeUntrustedField(input.HeadSHA))
	fmt.Fprintf(&out, "Chunk: %d of %d\n", input.ChunkIndex+1, input.ChunkCount)
	fmt.Fprintf(&out, "Language: %s\n", promptpack.EscapeUntrustedField(input.Language))
	fmt.Fprintf(&out, "Review checks: %s\n", promptpack.EscapeUntrustedField(strings.Join(input.ReviewChecks, ", ")))
	fmt.Fprintf(&out, "Test summary: %s\n", promptpack.EscapeUntrustedField(input.TestSummary))
	if trimmed := strings.TrimSpace(input.Instructions); trimmed != "" {
		fmt.Fprintf(&out, "\nRepository-local instructions:\n%s\n", promptpack.EscapeUntrusted(trimmed))
	}
	fmt.Fprintf(&out, "\n%s\n", input.UntrustedInputWarning)
	fmt.Fprintf(&out, "\nTask:\n%s\n", input.ReviewTask)
	fmt.Fprintf(&out, "%s\n", promptpack.TrustedControlEnd)
	fmt.Fprintf(&out, "\n%s\n", promptpack.UntrustedInputStart)
	if len(input.CommitMessages) > 0 {
		fmt.Fprintf(&out, "\nCommit messages, untrusted:\n")
		for _, message := range input.CommitMessages {
			fmt.Fprintf(&out, "- %s\n", promptpack.EscapeUntrustedField(message))
		}
	}
	fmt.Fprintf(&out, "\nChanged files in this task:\n")
	for _, file := range input.Files {
		fmt.Fprintf(&out, "- %s", promptpack.EscapeUntrustedField(file.Path))
		if file.Status != "" {
			fmt.Fprintf(&out, " [%s]", promptpack.EscapeUntrustedField(file.Status))
		}
		if file.PreviousPath != "" {
			fmt.Fprintf(&out, " from %s", promptpack.EscapeUntrustedField(file.PreviousPath))
		}
		if len(file.Spans) > 0 {
			fmt.Fprintf(&out, " changed lines ")
			for i, span := range file.Spans {
				if i > 0 {
					out.WriteString(", ")
				}
				fmt.Fprintf(&out, "L%d-L%d", span.Start, span.End)
			}
		}
		out.WriteString("\n")
	}
	fmt.Fprintf(&out, "%s\n", promptpack.UntrustedInputEnd)
	return out.String()
}

func appendVisibleText(out *strings.Builder, content *genai.Content) {
	if out == nil || content == nil {
		return
	}
	for _, part := range content.Parts {
		if part == nil || part.Thought || strings.TrimSpace(part.Text) == "" {
			continue
		}
		out.WriteString(part.Text)
	}
}

func isTransientProviderError(err error) bool {
	if err == nil {
		return false
	}
	if reliability.IsTransient(err) {
		return true
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return isStructuredOutputProviderMessage(msg) ||
		strings.Contains(msg, "authentication required") ||
		strings.Contains(msg, "exceeded your monthly quota") ||
		strings.Contains(msg, "payment required") ||
		strings.Contains(msg, "rate limit") ||
		(strings.Contains(msg, "generate content") && strings.Contains(msg, "request"))
}

func isStructuredOutputProviderError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return isStructuredOutputProviderMessage(msg)
}

func isStructuredOutputProviderMessage(msg string) bool {
	return strings.Contains(msg, "structured output schema validation") ||
		strings.Contains(msg, "no json object found")
}
