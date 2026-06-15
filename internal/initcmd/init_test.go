package initcmd

import (
	"os"
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
	}, []string{"openai-fast", "codex-acp"})
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
	if cfg.ProviderID() != "codex-acp" {
		t.Fatalf("ProviderID() = %q, want codex-acp", cfg.ProviderID())
	}
	if _, ok := cfg.Providers["openai-fast"]; !ok {
		t.Fatalf("generated config missing openai-fast provider: %+v", cfg.Providers)
	}
	if _, ok := cfg.Providers["codex-acp"]; !ok {
		t.Fatalf("generated config missing codex-acp provider: %+v", cfg.Providers)
	}
	if cfg.Review.Language != "en" {
		t.Fatalf("Review.Language = %q, want en", cfg.Review.Language)
	}
	if strings.Join(cfg.Review.Checks, ",") != "security,bugs,performance,best-practices" {
		t.Fatalf("Review.Checks = %v, want default review checks", cfg.Review.Checks)
	}
}

func TestInitWizardWorkspaceWritesGitHubCIProfileConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".github", "workflows", "review.yml"), []byte("name: review\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := InitWizardWorkspace(WizardOptions{
		InitOptions: InitOptions{
			WorkingDir: dir,
			ConfigPath: filepath.Join(dir, ".config", "diffpal", "config.yaml"),
			StatePath:  filepath.Join(dir, ".config", "diffpal", "state"),
		},
		Setup:    "codex-api-key",
		Platform: "auto",
		Profile:  "ci",
		BlockOn:  "high",
	})
	if err != nil {
		t.Fatalf("InitWizardWorkspace() error = %v", err)
	}
	if result.Setup != "codex-api-key" || result.Platform != "github" || result.Profile != "ci" || result.BlockOn != "high" {
		t.Fatalf("InitWizardWorkspace() result = %+v, want codex GitHub ci high", result)
	}

	renderedBytes, err := os.ReadFile(result.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	rendered := string(renderedBytes)
	for _, needle := range []string{
		"    codex-acp:",
		"  provider: codex-acp",
		"    github: {}",
		"profiles:",
		"  ci:",
		"        block_on: high",
	} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("wizard config missing %q:\n%s", needle, rendered)
		}
	}
	if strings.Contains(rendered, "openai-fast") {
		t.Fatalf("wizard config exposed openai-fast:\n%s", rendered)
	}

	cfg, err := dc.LoadConfig(dir, "", "ci")
	if err != nil {
		t.Fatalf("wizard config failed to load with ci profile: %v", err)
	}
	if cfg.ProviderID() != "codex-acp" {
		t.Fatalf("ProviderID() = %q, want codex-acp", cfg.ProviderID())
	}
	if cfg.BlockOn() != "high" {
		t.Fatalf("BlockOn() = %q, want high", cfg.BlockOn())
	}
}

func TestInitWizardWorkspacePreservesExistingConfigWithoutForce(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, ".config", "diffpal", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	original := "version: v1\n# keep me\n"
	if err := os.WriteFile(configPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InitWizardWorkspace(WizardOptions{
		InitOptions: InitOptions{
			WorkingDir: dir,
			ConfigPath: configPath,
		},
		Setup:    "copilot-github-token",
		Platform: "github",
	}); err != nil {
		t.Fatalf("InitWizardWorkspace() error = %v", err)
	}
	gotBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotBytes) != original {
		t.Fatalf("wizard overwrote existing config without force:\n%s", gotBytes)
	}
}

func TestComposeWizardConfigUsesSetupRecipes(t *testing.T) {
	t.Parallel()

	rendered := composeWizardConfig(wizardConfigOptions{
		Setup:      "copilot-github-token",
		ProviderID: "copilot-acp",
		Platform:   "gitlab",
		Profile:    "ci",
		BlockOn:    "critical",
	})
	for _, needle := range []string{
		"    copilot-acp:",
		"      type: copilot_acp",
		"        model: auto",
		"    gitlab: {}",
		"        block_on: critical",
	} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("composeWizardConfig() missing %q:\n%s", needle, rendered)
		}
	}
	if strings.Contains(rendered, "openai-fast") || strings.Contains(rendered, "gpt-5-mini") {
		t.Fatalf("composeWizardConfig() exposed obsolete provider details:\n%s", rendered)
	}
}

func TestDetectCIPlatform(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		path string
		want string
	}{
		{name: "github", path: filepath.Join(".github", "workflows", "diffpal.yml"), want: "github"},
		{name: "gitlab", path: ".gitlab-ci.yml", want: "gitlab"},
		{name: "azure-root", path: "azure-pipelines.yml", want: "azure"},
		{name: "azure-dir", path: filepath.Join(".azure-pipelines", "review.yaml"), want: "azure"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, tc.path)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("name: ci\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if got := detectCIPlatform(dir); got != tc.want {
				t.Fatalf("detectCIPlatform() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestComposeConfigUsesSelectedProviderRoot(t *testing.T) {
	t.Parallel()

	rendered := composeConfig([]string{"openai-fast", "codex-acp"})
	if !strings.Contains(rendered, "diffpal:\n  provider: codex-acp") {
		t.Fatalf("composeConfig() missing codex diffpal provider:\n%s", rendered)
	}
	if !strings.Contains(rendered, "  gate:\n    block_on: high") {
		t.Fatalf("composeConfig() missing gate.block_on:\n%s", rendered)
	}
	if strings.Contains(rendered, "profiles:") || strings.Contains(rendered, "platforms:") {
		t.Fatalf("composeConfig() should keep profiles/platform auth in templates only:\n%s", rendered)
	}
	for _, needle := range []string{
		"      reasoning_effort: low",
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

func TestSelectedDefaultProviderPrefersCodex(t *testing.T) {
	t.Parallel()

	got := selectedDefaultProvider([]string{"openai-fast", "copilot-acp", "codex-acp"})
	if got != "codex-acp" {
		t.Fatalf("selectedDefaultProvider() = %q, want codex-acp", got)
	}
}
