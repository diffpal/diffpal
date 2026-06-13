package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigAppliesProfileOverlay(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".config", "diffpal", "config.yaml"), `
version: v1
runtime:
  providers:
    openai-fast:
      type: openai
      openai:
        model: gpt-5.4-mini
diffpal:
  provider: openai-fast
  gate:
    block_on: high
  review:
    language: en
    checks:
      - bugs
profiles:
  ci:
    diffpal:
      gate:
        block_on: critical
      review:
        language: Spanish
        checks:
          - performance
`)

	cfg, err := LoadConfig(dir, "", "ci")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.ProviderID() != "openai-fast" {
		t.Fatalf("ProviderID() = %q, want openai-fast", cfg.ProviderID())
	}
	if cfg.BlockOn() != "critical" {
		t.Fatalf("BlockOn() = %q, want critical", cfg.BlockOn())
	}
	if cfg.Review.Language != "Spanish" {
		t.Fatalf("Review.Language = %q, want Spanish", cfg.Review.Language)
	}
	if strings.Join(cfg.Review.Checks, ",") != "performance" {
		t.Fatalf("Review.Checks = %v, want [performance]", cfg.Review.Checks)
	}
}

func TestLoadConfigEnvProfileOverridesDefaultSelection(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".config", "diffpal", "config.yaml"), `
version: v1
runtime:
  providers:
    openai-fast:
      type: openai
      openai:
        model: gpt-5.4-mini
    copilot-acp:
      type: copilot_acp
      copilot_acp:
        mode: ""
diffpal:
  provider: openai-fast
  gate:
    block_on: high
profiles:
  enterprise:
    diffpal:
      provider: copilot-acp
`)

	t.Setenv("DIFFPAL_PROFILE", "enterprise")
	cfg, err := LoadConfig(dir, "", "")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.ProviderID() != "copilot-acp" {
		t.Fatalf("ProviderID() = %q, want copilot-acp", cfg.ProviderID())
	}
}

func TestLoadConfigEnvLeafOverridesApply(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".config", "diffpal", "config.yaml"), minimalConfig("openai-fast"))

	t.Setenv("DIFFPAL_REVIEW_LANGUAGE", "Portuguese")
	t.Setenv("DIFFPAL_REVIEW_CHECKS", "security,best_practices")
	t.Setenv("DIFFPAL_REVIEW_INSTRUCTIONS", "Focus on auth-sensitive paths.")
	t.Setenv("DIFFPAL_BLOCK_ON", "critical")
	t.Setenv("DIFFPAL_OPENAI_MODEL", "gpt-env")
	cfg, err := LoadConfig(dir, "", "")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Review.Language != "Portuguese" {
		t.Fatalf("Review.Language = %q, want Portuguese", cfg.Review.Language)
	}
	if strings.Join(cfg.Review.Checks, ",") != "security,best-practices" {
		t.Fatalf("Review.Checks = %v, want [security best-practices]", cfg.Review.Checks)
	}
	if cfg.ReviewInstructions() != "Focus on auth-sensitive paths." {
		t.Fatalf("ReviewInstructions() = %q, want env override", cfg.ReviewInstructions())
	}
	if cfg.BlockOn() != "critical" {
		t.Fatalf("BlockOn() = %q, want critical", cfg.BlockOn())
	}
	if cfg.Providers["openai-fast"].OpenAI.Model != "gpt-env" {
		t.Fatalf("OpenAI.Model = %q, want gpt-env", cfg.Providers["openai-fast"].OpenAI.Model)
	}
}

func TestGitHubSummaryCommentDefaultsEnabled(t *testing.T) {
	var cfg GitHubPlatformConfig
	if !cfg.SummaryCommentEnabled() {
		t.Fatal("SummaryCommentEnabled() = false, want default true")
	}

	enabled := false
	cfg.SummaryComment.Enabled = &enabled
	if cfg.SummaryCommentEnabled() {
		t.Fatal("SummaryCommentEnabled() = true, want configured false")
	}
}

func TestLoadConfigExpandsEnvsubstValuesBeforeYAMLDecode(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".config", "diffpal", "config.yaml"), `
version: v1
runtime:
  providers:
    openai-fast:
      type: openai
      openai:
        model: "${DIFFPAL_MODEL_TEST}"
        api_key: "${OPENAI_API_KEY_TEST}"
diffpal:
  provider: openai-fast
  gate:
    block_on: high
  platforms:
    github:
      auth:
        token: "${GITHUB_TOKEN_TEST}"
    azure:
      auth:
        pat: "${AZURE_DEVOPS_PAT_TEST}"
`)

	t.Setenv("DIFFPAL_MODEL_TEST", "gpt-test")
	t.Setenv("OPENAI_API_KEY_TEST", "openai-token")
	t.Setenv("GITHUB_TOKEN_TEST", "github-token")
	t.Setenv("AZURE_DEVOPS_PAT_TEST", "pat-token")
	cfg, err := LoadConfig(dir, "", "")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Providers["openai-fast"].OpenAI.Model != "gpt-test" {
		t.Fatalf("model = %q, want gpt-test", cfg.Providers["openai-fast"].OpenAI.Model)
	}
	if cfg.Providers["openai-fast"].OpenAI.APIKey != "openai-token" {
		t.Fatalf("api key = %q, want openai-token", cfg.Providers["openai-fast"].OpenAI.APIKey)
	}
	if cfg.Platforms.GitHub.Auth.Token != "github-token" {
		t.Fatalf("github token = %q, want github-token", cfg.Platforms.GitHub.Auth.Token)
	}
	if cfg.Platforms.Azure.Auth.PAT != "pat-token" {
		t.Fatalf("azure pat = %q, want pat-token", cfg.Platforms.Azure.Auth.PAT)
	}
}

func TestLoadConfigRejectsUnknownProvider(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".config", "diffpal", "config.yaml"), minimalConfig("missing-provider"))

	_, err := LoadConfig(dir, "", "")
	if err == nil || !strings.Contains(err.Error(), `unknown diffpal.provider "missing-provider"`) {
		t.Fatalf("LoadConfig() error = %v, want unknown provider error", err)
	}
}

func TestLoadConfigRequiresProvider(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".config", "diffpal", "config.yaml"), `
version: v1
runtime:
  providers:
    openai-fast:
      type: openai
      openai:
        model: gpt-5.4-mini
diffpal:
  gate:
    block_on: high
`)

	_, err := LoadConfig(dir, "", "")
	if err == nil || !strings.Contains(err.Error(), `diffpal.provider is required`) {
		t.Fatalf("LoadConfig() error = %v, want missing provider error", err)
	}
}

func TestLoadConfigRejectsInvalidBlockOn(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".config", "diffpal", "config.yaml"), `
version: v1
runtime:
  providers:
    openai-fast:
      type: openai
      openai:
        model: gpt-5.4-mini
diffpal:
  provider: openai-fast
  gate:
    block_on: severe
`)

	_, err := LoadConfig(dir, "", "")
	if err == nil || !strings.Contains(err.Error(), `invalid diffpal.gate.block_on "severe"`) {
		t.Fatalf("LoadConfig() error = %v, want invalid block_on error", err)
	}
}

func TestLoadConfigRejectsInvalidReviewChecks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".config", "diffpal", "config.yaml"), `
version: v1
runtime:
  providers:
    openai-fast:
      type: openai
      openai:
        model: gpt-5.4-mini
diffpal:
  provider: openai-fast
  gate:
    block_on: high
  review:
    checks:
      - architecture
`)

	_, err := LoadConfig(dir, "", "")
	if err == nil || !strings.Contains(err.Error(), `invalid review.checks value "architecture"`) {
		t.Fatalf("LoadConfig() error = %v, want invalid review.checks error", err)
	}
}

func TestNormalizeReviewDefaults(t *testing.T) {
	t.Parallel()

	cfg := Config{}
	if err := cfg.Normalize(); err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if cfg.Review.Language != "en" {
		t.Fatalf("Review.Language = %q, want en", cfg.Review.Language)
	}
	if strings.Join(cfg.Review.Checks, ",") != "security,bugs,performance,best-practices" {
		t.Fatalf("Review.Checks = %v, want default checks", cfg.Review.Checks)
	}
}

func TestProviderKeysReturnsStableOrder(t *testing.T) {
	t.Parallel()

	cfg := Config{Providers: map[string]ProviderConfig{
		"z": {},
		"a": {},
	}}
	got := ProviderKeys(cfg)
	if strings.Join(got, ",") != "a,z" {
		t.Fatalf("ProviderKeys() = %v, want [a z]", got)
	}
}

func TestResolveConfigPathPrefersConfiguredRoot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	overrideRoot := filepath.Join(dir, "override")
	overridePath := filepath.Join(overrideRoot, "diffpal", "config.yaml")
	repoConfigPath := filepath.Join(dir, ".config", "diffpal", "config.yaml")
	writeTestFile(t, overridePath, minimalConfig("openai-fast"))
	writeTestFile(t, repoConfigPath, minimalConfig("openai-fast"))

	got, found, err := ResolveConfigPath(dir, overrideRoot)
	if err != nil {
		t.Fatalf("ResolveConfigPath() error = %v", err)
	}
	if !found {
		t.Fatal("ResolveConfigPath() found = false, want true")
	}
	if got != overridePath {
		t.Fatalf("ResolveConfigPath() = %q, want %q", got, overridePath)
	}
}

func TestResolveConfigPathFallsBackToRepositoryConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	overrideRoot := filepath.Join(dir, "override")
	repoConfigPath := filepath.Join(dir, ".config", "diffpal", "config.yaml")
	writeTestFile(t, repoConfigPath, minimalConfig("openai-fast"))

	got, found, err := ResolveConfigPath(dir, overrideRoot)
	if err != nil {
		t.Fatalf("ResolveConfigPath() error = %v", err)
	}
	if !found {
		t.Fatal("ResolveConfigPath() found = false, want true")
	}
	if got != repoConfigPath {
		t.Fatalf("ResolveConfigPath() = %q, want %q", got, repoConfigPath)
	}
}

func minimalConfig(provider string) string {
	return `
version: v1
runtime:
  providers:
    openai-fast:
      type: openai
      openai:
        model: gpt-5.4-mini
diffpal:
  provider: ` + provider + `
  gate:
    block_on: high
`
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimLeft(content, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", path, err)
	}
}
