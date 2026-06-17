package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"

	"github.com/diffpal/diffpal/internal/config"
	"github.com/diffpal/diffpal/internal/platformauth"
	"github.com/diffpal/diffpal/internal/provider"
	"github.com/normahq/norma/pkg/runtime/agentfactory"
	"github.com/normahq/norma/pkg/runtime/mcpregistry"
	"github.com/spf13/cobra"
)

func newDoctorCommand() *cobra.Command {
	doctor := &cobra.Command{
		Use:   "doctor",
		Short: "Validate local runtime and environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			var issues []string
			errorIssues := []string{}
			mode, err := cmd.Flags().GetString("mode")
			if err != nil {
				return err
			}
			workingDir, err := currentWorkingDir()
			if err != nil {
				return err
			}
			configPath, configExists, err := config.ResolveConfigPath(workingDir, rootConfigDir)
			if err != nil {
				return err
			}

			cfg := config.Config{}
			if configExists {
				cfg, err = config.LoadConfig(workingDir, rootConfigDir, rootProfile)
			}
			if err != nil && configExists {
				issues = append(issues, "error: config validation failed: "+err.Error())
				errorIssues = append(errorIssues, err.Error())
				return printDoctorResult(cmd, issues, errorIssues)
			}

			issues = append(issues, provider.Diagnostics()...)
			if len(provider.AutoPoolDefaults()) == 0 {
				issues = append(issues, "warn: no default provider definitions available")
			}
			issues = append(issues, diagnoseProviderConfig(cfg)...)
			selectedProviderIssues, selectedProviderError := diagnoseSelectedProvider(cfg, workingDir, requiresProviderAuth(mode))
			issues = append(issues, selectedProviderIssues...)
			if selectedProviderError != "" {
				errorIssues = append(errorIssues, selectedProviderError)
			}
			platformIssues, platformError := diagnosePlatformAuth(cfg, mode)
			issues = append(issues, platformIssues...)
			if platformError != "" {
				errorIssues = append(errorIssues, platformError)
			}
			issues = append(issues, diagnoseWorkspace(configPath)...)

			return printDoctorResult(cmd, issues, errorIssues)
		},
	}
	doctor.Flags().String("mode", "local", "Validate mode-specific platform authorization for local, github, gitlab, or ado")
	return doctor
}

func printDoctorResult(cmd *cobra.Command, issues []string, errorIssues []string) error {
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "doctor runtime: goos=%s\n", runtime.GOOS); err != nil {
		return err
	}
	for _, issue := range issues {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), issue); err != nil {
			return err
		}
	}
	if len(errorIssues) > 0 {
		return withExitCode(2, errors.New("doctor failures: "+strings.Join(errorIssues, "; ")))
	}
	return nil
}

func diagnoseProviderConfig(cfg config.Config) []string {
	if len(cfg.Providers) == 0 {
		return nil
	}
	keys := make([]string, 0, len(cfg.Providers))
	for key := range cfg.Providers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	issues := make([]string, 0, len(keys))
	for _, key := range keys {
		providerCfg := cfg.Providers[key]
		switch providerCfg.Type {
		case "openai", "aistudio":
			if tokenEnv := hostedAPIKeyEnv(providerCfg.Type); tokenEnv != "" {
				if os.Getenv(tokenEnv) == "" {
					issues = append(issues, fmt.Sprintf("warn: hosted provider %s expects %s", key, tokenEnv))
				} else {
					issues = append(issues, fmt.Sprintf("ok: hosted provider %s auth via %s", key, tokenEnv))
				}
			}
		case "pool":
			count := 0
			if providerCfg.PoolConfig != nil {
				count = len(providerCfg.PoolConfig.Members)
			}
			issues = append(issues, fmt.Sprintf("ok: pool provider %s configured with %d entries", key, count))
		default:
			bin := providerBinary(providerCfg)
			if bin == "" {
				issues = append(issues, fmt.Sprintf("warn: provider %s has no executable configured", key))
				continue
			}
			if provider.HasExecutable(bin) {
				issues = append(issues, fmt.Sprintf("ok: provider %s executable available: %s", key, bin))
			} else {
				issues = append(issues, fmt.Sprintf("warn: provider %s executable missing: %s", key, bin))
			}
		}
	}
	return issues
}

func diagnoseSelectedProvider(cfg config.Config, workingDir string, requireAuth bool) ([]string, string) {
	providerID := strings.TrimSpace(cfg.ProviderID())
	if providerID == "" {
		return nil, ""
	}

	issues := make([]string, 0, 2)
	providerCfg, ok := cfg.Providers[providerID]
	if !ok {
		msg := fmt.Sprintf("selected provider %s is not defined", providerID)
		return []string{"error: " + msg}, msg
	}

	providers := providerConfigsWithEnv(cfg.Providers)
	selected := providers[providerID]
	if missing := hostedProviderMissing(selected); missing != "" {
		if !requireAuth && strings.Contains(missing, "api_key") {
			issues = append(issues, "warn: "+missing)
			return issues, ""
		}
		issues = append(issues, "error: "+missing)
		return issues, missing
	}

	factory := agentfactory.New(providers, mcpregistry.New(cfg.MCPServers))
	if err := factory.ValidateAgent(providerID); err != nil {
		msg := fmt.Sprintf("selected provider %s failed validation", providerID)
		return append(issues, "error: "+msg), msg
	}
	if _, err := factory.BuildSessionState(providerID, workingDir); err != nil {
		msg := fmt.Sprintf("selected provider %s session state invalid", providerID)
		return append(issues, "error: "+msg), msg
	}

	issues = append(issues, fmt.Sprintf("ok: selected provider %s validated (%s)", providerID, providerCfg.Type))
	return issues, ""
}

func requiresProviderAuth(mode string) bool {
	selected := strings.ToLower(strings.TrimSpace(mode))
	return selected != "" && selected != "local"
}

func diagnoseWorkspace(configPath string) []string {
	issues := []string{}
	if _, err := os.Stat(configPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			issues = append(issues, fmt.Sprintf("warn: %s not found; run `diffpal init`", configPath))
		} else {
			issues = append(issues, fmt.Sprintf("warn: cannot inspect %s: %v", configPath, err))
		}
	}
	if !provider.HasExecutable("git") {
		issues = append(issues, "warn: git is not available; diff collection and SCM context will fail")
		return issues
	}
	if !insideGitWorkTree() {
		issues = append(issues, "warn: not inside a git work tree; review and SCM-aware commands are unsupported")
	} else {
		issues = append(issues, "ok: inside git work tree")
	}
	return issues
}

func diagnosePlatformAuth(cfg config.Config, mode string) ([]string, string) {
	selected := strings.ToLower(strings.TrimSpace(mode))
	if selected == "" || selected == "local" {
		return []string{"ok: local mode does not require platform authorization"}, ""
	}
	auth, err := platformauth.Resolve(cfg, selected)
	if err != nil {
		return []string{"error: " + err.Error()}, err.Error()
	}
	return []string{fmt.Sprintf("ok: %s auth resolved via %s (%s)", selected, auth.Source, auth.Mode)}, ""
}

func insideGitWorkTree() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}

func providerBinary(cfg config.ProviderConfig) string {
	switch cfg.Type {
	case "copilot_acp":
		return "copilot"
	case "gemini_acp":
		return "gemini"
	case "claude_code_acp":
		return "claude"
	case "codex_acp":
		return "codex"
	case "generic_acp":
		if cfg.GenericACP != nil && len(cfg.GenericACP.Cmd) > 0 {
			return cfg.GenericACP.Cmd[0]
		}
	}
	return ""
}

func hostedAPIKeyEnv(providerType string) string {
	switch strings.ToLower(strings.TrimSpace(providerType)) {
	case "openai":
		return "OPENAI_API_KEY"
	case "aistudio":
		return "GEMINI_API_KEY"
	default:
		return ""
	}
}

func providerConfigsWithEnv(in map[string]config.ProviderConfig) map[string]config.ProviderConfig {
	out := make(map[string]config.ProviderConfig, len(in))
	for key, cfg := range in {
		copied := cfg
		if cfg.OpenAI != nil {
			block := *cfg.OpenAI
			if strings.TrimSpace(block.APIKey) == "" {
				block.APIKey = os.Getenv("OPENAI_API_KEY")
			}
			copied.OpenAI = &block
		}
		if cfg.AIStudio != nil {
			block := *cfg.AIStudio
			if strings.TrimSpace(block.APIKey) == "" {
				block.APIKey = os.Getenv("GEMINI_API_KEY")
			}
			copied.AIStudio = &block
		}
		out[key] = copied
	}
	return out
}

func hostedProviderMissing(cfg config.ProviderConfig) string {
	switch strings.ToLower(strings.TrimSpace(cfg.Type)) {
	case "openai":
		if cfg.OpenAI == nil {
			return "selected provider openai block is required"
		}
		if strings.TrimSpace(cfg.OpenAI.Model) == "" {
			return "selected provider openai.model is required"
		}
		if strings.TrimSpace(cfg.OpenAI.APIKey) == "" {
			return "selected provider openai.api_key is required or OPENAI_API_KEY must be set"
		}
	case "aistudio":
		if cfg.AIStudio == nil {
			return "selected provider aistudio block is required"
		}
		if strings.TrimSpace(cfg.AIStudio.Model) == "" {
			return "selected provider aistudio.model is required"
		}
		if strings.TrimSpace(cfg.AIStudio.APIKey) == "" {
			return "selected provider aistudio.api_key is required or GEMINI_API_KEY must be set"
		}
	}
	return ""
}
