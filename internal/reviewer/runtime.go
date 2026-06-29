package reviewer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	acp "github.com/coder/acp-go-sdk"
	dpconfig "github.com/diffpal/diffpal/internal/config"
	"github.com/diffpal/diffpal/internal/logging"
	"github.com/diffpal/diffpal/internal/reliability"
	"github.com/diffpal/diffpal/internal/reviewer/promptpack"
	"github.com/normahq/norma/pkg/runtime/agentfactory"
	"github.com/normahq/norma/pkg/runtime/mcpregistry"
	"github.com/normahq/norma/pkg/runtime/providererror"
	"github.com/normahq/norma/pkg/runtime/structuredagent"
	adkagent "google.golang.org/adk/agent"
	adkrunner "google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

type ADKRuntime struct{}

func (ADKRuntime) Review(ctx context.Context, cfg RuntimeConfig, input ReviewInput) (ReviewOutput, RuntimeUsage, error) {
	if strings.TrimSpace(cfg.ProviderID) == "" {
		return ReviewOutput{}, RuntimeUsage{}, wrapError(KindConfig, fmt.Errorf("provider id is required"))
	}
	providerCfg, ok := cfg.Providers[cfg.ProviderID]
	if !ok {
		return ReviewOutput{}, RuntimeUsage{}, wrapError(KindConfig, fmt.Errorf("unknown provider %q", cfg.ProviderID))
	}

	if err := validateHostedProviderConfig(providerCfg); err != nil {
		return ReviewOutput{}, RuntimeUsage{}, wrapError(KindConfig, err)
	}

	factory := agentfactory.New(
		cfg.Providers,
		mcpregistry.New(cfg.MCPServers),
		agentfactory.WithPermissionHandler(reviewPermissionHandler),
	)
	sessionState, err := factory.BuildSessionState(cfg.ProviderID, cfg.WorkingDir)
	if err != nil {
		return ReviewOutput{}, RuntimeUsage{}, wrapError(KindConfig, err)
	}
	agentRuntime, err := factory.Build(ctx, reviewAgentBuildRequest(cfg))
	if err != nil {
		return ReviewOutput{}, RuntimeUsage{}, wrapError(KindConfig, err)
	}
	if closer, ok := agentRuntime.(interface{ Close() error }); ok {
		defer func() { _ = closer.Close() }()
	}

	prompt := promptpack.DefaultReviewPrompt()
	wrapped, err := structuredagent.NewAgent(agentRuntime,
		structuredagent.WithoutInputSchema(),
		structuredagent.WithOutputSchema(prompt.OutputSchema),
		structuredagent.WithSystemInstruction(prompt.RenderReviewSystem(promptpack.ReviewOptions{Instructions: cfg.Instructions})),
		structuredagent.WithOutputValidationRetries(3),
	)
	if err != nil {
		return ReviewOutput{}, RuntimeUsage{}, wrapError(KindInternal, err)
	}

	sessionService := session.InMemoryService()
	runner, err := adkrunner.New(adkrunner.Config{
		AppName:        "diffpal",
		Agent:          wrapped,
		SessionService: sessionService,
	})
	if err != nil {
		return ReviewOutput{}, RuntimeUsage{}, wrapError(KindInternal, fmt.Errorf("create adk runner: %w", err))
	}

	userID := "diffpal-user"
	created, err := sessionService.Create(ctx, &session.CreateRequest{
		AppName: "diffpal",
		UserID:  userID,
		State:   sessionState,
	})
	if err != nil {
		return ReviewOutput{}, RuntimeUsage{}, wrapError(KindInternal, fmt.Errorf("create review session: %w", err))
	}

	userContent := genai.NewContentFromText(renderReviewTaskInput(input), genai.RoleUser)
	events := runner.Run(ctx, userID, created.Session.ID(), userContent, adkagent.RunConfig{})

	var outputText strings.Builder
	var usage RuntimeUsage
	for ev, runErr := range events {
		if runErr != nil {
			if providerErr, ok := providerErrorFromRuntimeError(runErr); ok {
				return ReviewOutput{}, usage, wrapError(KindTransient, providerErr)
			}
			if isTransientProviderError(runErr) {
				return ReviewOutput{}, usage, wrapError(KindTransient, runErr)
			}
			return ReviewOutput{}, usage, wrapError(KindInternal, runErr)
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
		return ReviewOutput{}, usage, wrapError(KindInternal, fmt.Errorf("provider returned no structured output"))
	}

	var output ReviewOutput
	if err := json.Unmarshal([]byte(trimmed), &output); err != nil {
		logging.DebugProviderResponse(ctx, trimmed)
		return ReviewOutput{}, usage, wrapError(KindInternal, fmt.Errorf("parse structured output: %w", err))
	}
	if len(nonEmptyChangeSummary(output.ChangeSummary)) == 0 {
		return ReviewOutput{}, usage, wrapError(KindInternal, fmt.Errorf("provider returned no change_summary"))
	}
	output.ReviewResult = strings.TrimSpace(output.ReviewResult)
	return output, usage, nil
}

func nonEmptyChangeSummary(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func reviewPermissionHandler(_ context.Context, req acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	// Agent security is delegated to the provider configuration. DiffPal does not
	// layer a second tool policy on top of ACP; it selects provider-offered allow
	// options so provider-specific sandbox and approval settings remain the source
	// of truth.
	if option, ok := firstPermissionOption(req.Options, acp.PermissionOptionKindAllowOnce, acp.PermissionOptionKindAllowAlways); ok {
		return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeSelected(option.OptionId)}, nil
	}
	return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeCancelled()}, nil
}

func firstPermissionOption(options []acp.PermissionOption, kinds ...acp.PermissionOptionKind) (acp.PermissionOption, bool) {
	for _, kind := range kinds {
		for _, option := range options {
			if option.Kind == kind {
				return option, true
			}
		}
	}
	return acp.PermissionOption{}, false
}

func reviewSystemInstruction(instructions string) string {
	return promptpack.DefaultReviewPrompt().RenderReviewSystem(promptpack.ReviewOptions{Instructions: instructions})
}

func reviewAgentBuildRequest(cfg RuntimeConfig) agentfactory.BuildRequest {
	return agentfactory.BuildRequest{
		AgentID:          cfg.ProviderID,
		Name:             "DiffPalReviewerAgent",
		Description:      "DiffPal provider-backed review agent",
		WorkingDirectory: cfg.WorkingDir,
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

func renderReviewTaskInput(input ReviewInput) string {
	var out strings.Builder
	fmt.Fprintf(&out, "DiffPal review task snapshot\n\n")
	fmt.Fprintf(&out, "%s\n", promptpack.TrustedControlStart)
	fmt.Fprintf(&out, "Review ID: %s\n", promptpack.EscapeUntrustedField(input.ReviewID))
	fmt.Fprintf(&out, "Repository: %s\n", promptpack.EscapeUntrustedField(input.Repo))
	fmt.Fprintf(&out, "Base: %s\n", promptpack.EscapeUntrustedField(input.BaseSHA))
	fmt.Fprintf(&out, "Head: %s\n", promptpack.EscapeUntrustedField(input.HeadSHA))
	fmt.Fprintf(&out, "Block on: %s\n", promptpack.EscapeUntrustedField(input.BlockOn))
	fmt.Fprintf(&out, "Language: %s\n", promptpack.EscapeUntrustedField(input.Language))
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
	return isStructuredOutputProviderMessage(msg)
}

func isStructuredOutputProviderMessage(msg string) bool {
	return strings.Contains(msg, "structured output schema validation") ||
		strings.Contains(msg, "no json object found")
}

func providerErrorFromRuntimeError(err error) (*providererror.ProviderError, bool) {
	if err == nil {
		return nil, false
	}
	var validationErr *structuredagent.OutputValidationError
	if errors.As(err, &validationErr) && validationErr.ProviderError != nil {
		return validationErr.ProviderError, true
	}
	var reqErr *acp.RequestError
	if errors.As(err, &reqErr) {
		if providerErr, ok := providererror.FromWireData(reqErr.Data); ok {
			return providerErr, true
		}
		if reqErr.Code == -32000 {
			return &providererror.ProviderError{
				Kind:    providererror.KindAuthenticationRequired,
				Message: strings.TrimSpace(reqErr.Message),
			}, true
		}
	}
	return nil, false
}
