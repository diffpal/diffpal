package initcmd

import (
	"path/filepath"
	"strings"
	"testing"

	dc "github.com/diffpal/diffpal/internal/config"
)

func TestInitWorkspaceWritesRunnableConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	result, err := InitWorkspace(InitOptions{
		WorkingDir: dir,
		ConfigPath: filepath.Join(dir, ".config", "diffpal", "config.yaml"),
		StatePath:  filepath.Join(dir, ".config", "diffpal", "state"),
	}, []string{"openai-fast", "copilot-acp"})
	if err != nil {
		t.Fatalf("InitWorkspace() error = %v", err)
	}
	if result.ConfigPath == "" || result.StatePath == "" || result.IgnorePath == "" {
		t.Fatalf("InitWorkspace() returned incomplete paths: %+v", result)
	}
	if len(result.Templates) != 3 {
		t.Fatalf("len(Templates) = %d, want 3", len(result.Templates))
	}

	cfg, err := dc.LoadConfig(dir, "", "")
	if err != nil {
		t.Fatalf("generated config failed to load: %v", err)
	}
	if cfg.ProviderID() != "copilot-acp" {
		t.Fatalf("ProviderID() = %q, want copilot-acp", cfg.ProviderID())
	}
	if _, ok := cfg.Providers["openai-fast"]; !ok {
		t.Fatalf("generated config missing openai-fast provider: %+v", cfg.Providers)
	}
	if _, ok := cfg.Providers["copilot-acp"]; !ok {
		t.Fatalf("generated config missing copilot-acp provider: %+v", cfg.Providers)
	}
	if cfg.Review.Language != "en" {
		t.Fatalf("Review.Language = %q, want en", cfg.Review.Language)
	}
	if strings.Join(cfg.Review.Checks, ",") != "security,bugs,performance,best-practices" {
		t.Fatalf("Review.Checks = %v, want default review checks", cfg.Review.Checks)
	}
}

func TestComposeConfigUsesSelectedProviderRoot(t *testing.T) {
	t.Parallel()

	rendered := composeConfig([]string{"openai-fast", "copilot-acp"})
	if !strings.Contains(rendered, "diffpal:\n  provider: copilot-acp") {
		t.Fatalf("composeConfig() missing copilot diffpal provider:\n%s", rendered)
	}
	if !strings.Contains(rendered, "  gate:\n    block_on: high") {
		t.Fatalf("composeConfig() missing gate.block_on:\n%s", rendered)
	}
	if strings.Contains(rendered, "profiles:") || strings.Contains(rendered, "platforms:") {
		t.Fatalf("composeConfig() should keep profiles/platform auth in templates only:\n%s", rendered)
	}
	for _, needle := range []string{
		"      model: gpt-5-mini",
		"    language: en",
		"      - security",
		"      - bugs",
		"      - performance",
		"      - best-practices",
	} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("composeConfig() missing %q:\n%s", needle, rendered)
		}
	}
}

func TestComposeConfigDoesNotEmitOldSchemaRoots(t *testing.T) {
	t.Parallel()

	rendered := composeConfig([]string{"openai-fast"})
	for _, legacy := range []string{"\ndefaults:\n", "\npolicies:\n", "\nproviders:\n"} {
		if strings.Contains(rendered, legacy) {
			t.Fatalf("composeConfig() unexpectedly emitted legacy root %q:\n%s", legacy, rendered)
		}
	}
}

func TestConfigTemplatesUseDirectEnvsubstValues(t *testing.T) {
	t.Parallel()

	rendered := strings.Builder{}
	for _, template := range configTemplates() {
		rendered.WriteString(template.content)
	}
	text := rendered.String()
	for _, needle := range []string{
		`api_key: "${OPENAI_API_KEY}"`,
		`token: "${GITHUB_TOKEN}"`,
		`job_token: "${CI_JOB_TOKEN}"`,
		`api_token: "${GITLAB_TOKEN}"`,
		`system_access_token: "${SYSTEM_ACCESSTOKEN}"`,
		`pat: "${AZURE_DEVOPS_EXT_PAT}"`,
		"block_on: high",
	} {
		if !strings.Contains(text, needle) {
			t.Fatalf("config templates missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "_env:") {
		t.Fatalf("config templates must not use *_env fields:\n%s", text)
	}
}

func TestComposeConfigWritesDiffpalArtifactsIgnore(t *testing.T) {
	t.Parallel()

	if !strings.Contains(defaultIgnore, ".artifacts/") {
		t.Fatalf("defaultIgnore missing .artifacts entry:\n%s", defaultIgnore)
	}
	if strings.TrimSpace(defaultConfigGitignore) != "state/" {
		t.Fatalf("defaultConfigGitignore = %q, want only state/", defaultConfigGitignore)
	}
}

func TestProviderTypeForKey(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"copilot-acp":     "copilot_acp",
		"gemini-acp":      "gemini_acp",
		"claude-code-acp": "claude_code_acp",
		"codex-acp":       "codex_acp",
		"unknown-acp":     "generic_acp",
	}
	for key, want := range cases {
		if got := providerTypeForKey(key); got != want {
			t.Fatalf("providerTypeForKey(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestSelectedDefaultProviderFallsBackToFirstDetected(t *testing.T) {
	t.Parallel()

	if got := selectedDefaultProvider([]string{"openai-fast"}); got != "openai-fast" {
		t.Fatalf("selectedDefaultProvider() = %q, want openai-fast", got)
	}
}
