package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/diffpal/diffpal/internal/config"
	"github.com/normahq/norma/pkg/runtime/agentconfig"
)

func TestHostedAPIKeyEnv(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"openai":    "OPENAI_API_KEY",
		"aistudio":  "GEMINI_API_KEY",
		"something": "",
		"":          "",
	}
	for providerType, want := range cases {
		if got := hostedAPIKeyEnv(providerType); got != want {
			t.Fatalf("hostedAPIKeyEnv(%q) = %q, want %q", providerType, got, want)
		}
	}
}

func TestProviderBinary(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  config.ProviderConfig
		want string
	}{
		{name: "command wins", cfg: config.ProviderConfig{Type: "generic_acp", GenericACP: &agentconfig.ACPConfig{Cmd: []string{"custom-acp"}}}, want: "custom-acp"},
		{name: "copilot", cfg: config.ProviderConfig{Type: "copilot_acp", CopilotACP: &agentconfig.ACPConfig{}}, want: "copilot"},
		{name: "gemini", cfg: config.ProviderConfig{Type: "gemini_acp", GeminiACP: &agentconfig.ACPConfig{}}, want: "gemini"},
		{name: "claude", cfg: config.ProviderConfig{Type: "claude_code_acp", ClaudeCodeACP: &agentconfig.ACPConfig{}}, want: "claude"},
		{name: "codex", cfg: config.ProviderConfig{Type: "codex_acp", CodexACP: &agentconfig.ACPConfig{}}, want: "codex"},
		{name: "empty", cfg: config.ProviderConfig{Type: "openai", OpenAI: &agentconfig.LocalAPIConfig{}}, want: ""},
	}
	for _, tc := range cases {
		if got := providerBinary(tc.cfg); got != tc.want {
			t.Fatalf("%s: providerBinary() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestDiagnoseProviderConfigHostedAndPool(t *testing.T) {
	cfg := config.Config{
		Providers: map[string]config.ProviderConfig{
			"openai-fast": {
				Type:   "openai",
				OpenAI: &agentconfig.LocalAPIConfig{Model: "gpt-5.4-mini"},
			},
			"pool": {
				Type:       "pool",
				PoolConfig: &agentconfig.PoolConfig{Members: []string{"openai-fast"}},
			},
		},
	}

	t.Setenv("OPENAI_API_KEY", "")
	issues := diagnoseProviderConfig(cfg)
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "hosted provider openai-fast expects OPENAI_API_KEY") {
		t.Fatalf("diagnoseProviderConfig() missing hosted auth warning:\n%s", joined)
	}
	if !strings.Contains(joined, "hosted provider openai-fast review tools: git_changed_files, git_diff, list_files, read_file, search_files") {
		t.Fatalf("diagnoseProviderConfig() missing hosted tool capabilities:\n%s", joined)
	}
	if !strings.Contains(joined, "pool provider pool configured with 1 entries") {
		t.Fatalf("diagnoseProviderConfig() missing pool summary:\n%s", joined)
	}
}

func TestDiagnosePlatformAuthLocalMode(t *testing.T) {
	issues, fatal := diagnosePlatformAuth(config.Config{}, "local")
	if fatal != "" {
		t.Fatalf("fatal = %q, want empty", fatal)
	}
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "local mode does not require platform authorization") {
		t.Fatalf("diagnosePlatformAuth() missing local-mode message:\n%s", joined)
	}
}

func TestDiagnosePlatformAuthGitHubResolvesConfiguredEnv(t *testing.T) {
	cfg := config.Config{
		Platforms: config.PlatformConfigs{
			GitHub: config.GitHubPlatformConfig{
				Auth: config.GitHubAuthConfig{Token: "token"},
			},
		},
	}

	issues, fatal := diagnosePlatformAuth(cfg, "github")
	if fatal != "" {
		t.Fatalf("fatal = %q, want empty", fatal)
	}
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "github auth resolved via diffpal.platforms.github.auth.token") {
		t.Fatalf("diagnosePlatformAuth() missing GitHub auth success:\n%s", joined)
	}
}

func TestDiagnoseSelectedProviderHostedSuccess(t *testing.T) {
	cfg := config.Config{
		Provider: "openai-fast",
		Providers: map[string]config.ProviderConfig{
			"openai-fast": {
				Type:   "openai",
				OpenAI: &agentconfig.LocalAPIConfig{Model: "gpt-5"},
			},
		},
	}

	t.Setenv("OPENAI_API_KEY", "test-key")
	issues, fatal := diagnoseSelectedProvider(cfg, t.TempDir(), true)
	if fatal != "" {
		t.Fatalf("fatal = %q, want empty", fatal)
	}
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "selected provider openai-fast validated") {
		t.Fatalf("diagnoseSelectedProvider() missing success message:\n%s", joined)
	}
}

func TestDiagnoseSelectedProviderHostedMissingAuth(t *testing.T) {
	cfg := config.Config{
		Provider: "openai-fast",
		Providers: map[string]config.ProviderConfig{
			"openai-fast": {
				Type:   "openai",
				OpenAI: &agentconfig.LocalAPIConfig{Model: "gpt-5"},
			},
		},
	}

	t.Setenv("OPENAI_API_KEY", "")
	issues, fatal := diagnoseSelectedProvider(cfg, t.TempDir(), false)
	if fatal != "" {
		t.Fatalf("fatal = %q, want empty for local-mode auth warning", fatal)
	}
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "warn: selected provider openai.api_key is required") {
		t.Fatalf("diagnoseSelectedProvider() missing auth warning:\n%s", joined)
	}
}

func TestDiagnoseSelectedProviderHostedMissingAuthRequired(t *testing.T) {
	cfg := config.Config{
		Provider: "openai-fast",
		Providers: map[string]config.ProviderConfig{
			"openai-fast": {
				Type:   "openai",
				OpenAI: &agentconfig.LocalAPIConfig{Model: "gpt-5"},
			},
		},
	}

	t.Setenv("OPENAI_API_KEY", "")
	issues, fatal := diagnoseSelectedProvider(cfg, t.TempDir(), true)
	if fatal == "" {
		t.Fatal("fatal = empty, want missing auth error")
	}
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "error: selected provider openai.api_key is required") {
		t.Fatalf("diagnoseSelectedProvider() missing auth error:\n%s", joined)
	}
}

func TestDiagnoseWorkspaceReportsMissingConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".config", "diffpal", "config.yaml")

	issues := diagnoseWorkspace(configPath)
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "not found; run `diffpal init`") {
		t.Fatalf("diagnoseWorkspace() missing not-found warning:\n%s", joined)
	}
}

func TestDiagnoseWorkspaceReportsStatErrors(t *testing.T) {
	dir := t.TempDir()
	parentFile := filepath.Join(dir, ".config")
	if err := os.WriteFile(parentFile, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	configPath := filepath.Join(parentFile, "diffpal", "config.yaml")

	issues := diagnoseWorkspace(configPath)
	joined := strings.Join(issues, "\n")
	if !strings.Contains(joined, "cannot inspect") {
		t.Fatalf("diagnoseWorkspace() missing stat error warning:\n%s", joined)
	}
	if strings.Contains(joined, "run `diffpal init`") {
		t.Fatalf("diagnoseWorkspace() misreported stat error as missing config:\n%s", joined)
	}
}
