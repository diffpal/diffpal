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
defaults:
  provider: openai-fast
  policy: default
providers:
  openai-fast:
    type: openai
    openai:
      model: gpt-5.4-mini
policies:
  default:
    block_on: high
  strict:
    block_on: critical
review:
  context_lines: 10
  max_files: 100
  chunking:
    max_patch_chars: 12000
    max_files_per_chunk: 20
profiles:
  ci:
    defaults:
      policy: strict
    review:
      context_lines: 20
      max_files: 200
`)

	cfg, err := LoadConfig(dir, "", "ci")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.ProviderID() != "openai-fast" {
		t.Fatalf("ProviderID() = %q, want openai-fast", cfg.ProviderID())
	}
	if cfg.Review.ContextLines != 20 {
		t.Fatalf("Review.ContextLines = %d, want 20", cfg.Review.ContextLines)
	}
	if cfg.Review.MaxFiles != 200 {
		t.Fatalf("Review.MaxFiles = %d, want 200", cfg.Review.MaxFiles)
	}
	if cfg.BlockOn() != "critical" {
		t.Fatalf("BlockOn() = %q, want critical", cfg.BlockOn())
	}
}

func TestLoadConfigEnvProfileOverridesDefaultSelection(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".config", "diffpal", "config.yaml"), `
version: v1
defaults:
  provider: openai-fast
  policy: default
providers:
  openai-fast:
    type: openai
    openai:
      model: gpt-5.4-mini
  copilot-acp:
    type: copilot_acp
    copilot_acp:
      mode: ""
policies:
  default:
    block_on: high
review:
  context_lines: 20
  max_files: 200
profiles:
  enterprise:
    defaults:
      provider: copilot-acp
    review:
      context_lines: 7
      max_files: 42
`)

	t.Setenv("DIFFPAL_PROFILE", "enterprise")
	cfg, err := LoadConfig(dir, "", "")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.ProviderID() != "copilot-acp" {
		t.Fatalf("ProviderID() = %q, want copilot-acp", cfg.ProviderID())
	}
	if cfg.Review.ContextLines != 7 {
		t.Fatalf("Review.ContextLines = %d, want 7", cfg.Review.ContextLines)
	}
	if cfg.Review.MaxFiles != 42 {
		t.Fatalf("Review.MaxFiles = %d, want 42", cfg.Review.MaxFiles)
	}
}

func TestLoadConfigEnvLeafOverridesApply(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".config", "diffpal", "config.yaml"), minimalConfig("openai-fast"))

	t.Setenv("DIFFPAL_REVIEW_CONTEXT_LINES", "33")
	t.Setenv("DIFFPAL_REVIEW_MAX_FILES", "55")
	t.Setenv("DIFFPAL_BLOCK_ON", "critical")
	cfg, err := LoadConfig(dir, "", "")
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Review.ContextLines != 33 {
		t.Fatalf("Review.ContextLines = %d, want 33", cfg.Review.ContextLines)
	}
	if cfg.Review.MaxFiles != 55 {
		t.Fatalf("Review.MaxFiles = %d, want 55", cfg.Review.MaxFiles)
	}
	if cfg.BlockOn() != "critical" {
		t.Fatalf("BlockOn() = %q, want critical", cfg.BlockOn())
	}
}

func TestLoadConfigExpandsEnvsubstValuesBeforeYAMLDecode(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".config", "diffpal", "config.yaml"), `
version: v1
defaults:
  provider: openai-fast
  policy: default
providers:
  openai-fast:
    type: openai
    openai:
      model: "${DIFFPAL_MODEL_TEST}"
      api_key: "${OPENAI_API_KEY_TEST}"
policies:
  default:
    block_on: high
review:
  context_lines: 20
  max_files: 200
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
	if err == nil || !strings.Contains(err.Error(), `unknown defaults.provider "missing-provider"`) {
		t.Fatalf("LoadConfig() error = %v, want unknown provider error", err)
	}
}

func TestLoadConfigRejectsUnknownPolicy(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".config", "diffpal", "config.yaml"), `
version: v1
defaults:
  provider: openai-fast
  policy: missing-policy
providers:
  openai-fast:
    type: openai
    openai:
      model: gpt-5.4-mini
policies:
  default:
    block_on: high
review:
  context_lines: 20
  max_files: 200
`)

	_, err := LoadConfig(dir, "", "")
	if err == nil || !strings.Contains(err.Error(), `unknown defaults.policy "missing-policy"`) {
		t.Fatalf("LoadConfig() error = %v, want unknown policy error", err)
	}
}

func TestLoadConfigRejectsInvalidBlockOn(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, ".config", "diffpal", "config.yaml"), `
version: v1
defaults:
  provider: openai-fast
  policy: default
providers:
  openai-fast:
    type: openai
    openai:
      model: gpt-5.4-mini
policies:
  default:
    block_on: severe
review:
  context_lines: 20
  max_files: 200
`)

	_, err := LoadConfig(dir, "", "")
	if err == nil || !strings.Contains(err.Error(), `invalid policies.default.block_on "severe"`) {
		t.Fatalf("LoadConfig() error = %v, want invalid block_on error", err)
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
defaults:
  provider: ` + provider + `
  policy: default
providers:
  openai-fast:
    type: openai
    openai:
      model: gpt-5.4-mini
policies:
  default:
    block_on: high
review:
  context_lines: 20
  max_files: 200
  chunking:
    max_patch_chars: 12000
    max_files_per_chunk: 20
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
