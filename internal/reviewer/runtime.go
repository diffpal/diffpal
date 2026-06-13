package reviewer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	dpconfig "github.com/diffpal/diffpal/internal/config"
	"github.com/diffpal/diffpal/internal/reliability"
	"github.com/normahq/norma/pkg/runtime/agentfactory"
	"github.com/normahq/norma/pkg/runtime/mcpregistry"
	"github.com/normahq/norma/pkg/runtime/structuredagent"
	adkagent "google.golang.org/adk/agent"
	adkrunner "google.golang.org/adk/runner"
	"google.golang.org/adk/session"
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

	agentRuntime, err := factory.Build(ctx, agentfactory.BuildRequest{
		AgentID:           cfg.ProviderID,
		Name:              "DiffPalReviewerAgent",
		Description:       "DiffPal provider-backed review agent",
		GlobalInstruction: reviewInstruction(cfg.Instructions),
		WorkingDirectory:  cfg.WorkingDir,
	})
	if err != nil {
		return ChunkOutput{}, RuntimeUsage{}, wrapError(KindConfig, err)
	}
	if closer, ok := agentRuntime.(interface{ Close() error }); ok {
		defer func() { _ = closer.Close() }()
	}

	wrapped, err := structuredagent.NewAgent(agentRuntime,
		structuredagent.WithInputSchema(inputSchemaJSON),
		structuredagent.WithOutputSchema(outputSchemaJSON),
		structuredagent.WithSystemInstruction(reviewInstruction(cfg.Instructions)),
		structuredagent.WithOutputValidationRetries(1),
	)
	if err != nil {
		return ChunkOutput{}, RuntimeUsage{}, wrapError(KindInternal, err)
	}

	rawInput, err := json.Marshal(input)
	if err != nil {
		return ChunkOutput{}, RuntimeUsage{}, wrapError(KindInternal, fmt.Errorf("marshal chunk input: %w", err))
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

	userContent := genai.NewContentFromText(string(rawInput), genai.RoleUser)
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
	return output, usage, nil
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
