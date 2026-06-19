package initcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type InitOptions struct {
	WorkingDir string
	ConfigPath string
	StatePath  string
	Force      bool
}

type InitResult struct {
	ConfigPath  string
	StatePath   string
	IgnorePath  string
	Templates   []string
	ProviderSet []string
	Setup       string
	Platform    string
	Profile     string
	BlockOn     string
}

type WizardOptions struct {
	InitOptions
	Setup    string
	Platform string
	Profile  string
	BlockOn  string
}

const defaultIgnore = `# DiffPal generated ignore file
*.lock
bin/
dist/
.artifacts/
`

const defaultConfigGitignore = `state/
`

func InitWorkspace(opts InitOptions, detectedProviders []string) (InitResult, error) {
	if opts.WorkingDir == "" {
		opts.WorkingDir = "."
	}
	if opts.ConfigPath == "" {
		opts.ConfigPath = filepath.Join(opts.WorkingDir, ".config", "diffpal", "config.yaml")
	}
	if opts.StatePath == "" {
		opts.StatePath = filepath.Join(opts.WorkingDir, ".config", "diffpal", "state")
	}
	result := InitResult{
		ConfigPath: opts.ConfigPath,
		StatePath:  opts.StatePath,
		IgnorePath: filepath.Join(opts.WorkingDir, ".diffpalignore"),
	}
	configIgnorePath := filepath.Join(filepath.Dir(opts.ConfigPath), ".gitignore")
	templateDir := filepath.Join(filepath.Dir(opts.ConfigPath), "templates")

	if err := os.MkdirAll(filepath.Dir(opts.ConfigPath), 0o755); err != nil {
		return result, err
	}
	if err := os.MkdirAll(opts.StatePath, 0o755); err != nil {
		return result, err
	}

	if err := writeIfMissing(opts.ConfigPath, composeConfig(detectedProviders), opts.Force); err != nil {
		return result, err
	}
	if err := writeIfMissing(configIgnorePath, defaultConfigGitignore, opts.Force); err != nil {
		return result, err
	}
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		return result, err
	}
	for _, template := range configTemplates() {
		path := filepath.Join(templateDir, template.name)
		if err := writeIfMissing(path, template.content, opts.Force); err != nil {
			return result, err
		}
		result.Templates = append(result.Templates, path)
	}
	if err := writeIfMissing(result.IgnorePath, defaultIgnore, opts.Force); err != nil {
		return result, err
	}
	result.ProviderSet = detectedProviders

	return result, nil
}

func InitWizardWorkspace(opts WizardOptions) (InitResult, error) {
	initOpts := opts.InitOptions
	if initOpts.WorkingDir == "" {
		initOpts.WorkingDir = "."
	}
	if initOpts.ConfigPath == "" {
		initOpts.ConfigPath = filepath.Join(initOpts.WorkingDir, ".config", "diffpal", "config.yaml")
	}
	if initOpts.StatePath == "" {
		initOpts.StatePath = filepath.Join(initOpts.WorkingDir, ".config", "diffpal", "state")
	}

	setup, providerID, err := resolveWizardSetup(opts.Setup)
	if err != nil {
		return InitResult{}, err
	}
	platform, err := resolveWizardPlatform(initOpts.WorkingDir, opts.Platform)
	if err != nil {
		return InitResult{}, err
	}
	profile := strings.TrimSpace(opts.Profile)
	if profile == "" {
		profile = "ci"
	}
	if !validWizardName(profile) {
		return InitResult{}, fmt.Errorf("invalid profile %q", opts.Profile)
	}
	blockOn := strings.ToLower(strings.TrimSpace(opts.BlockOn))
	if blockOn == "" {
		blockOn = "high"
	}
	if !validWizardBlockOn(blockOn) {
		return InitResult{}, fmt.Errorf("invalid block threshold %q", opts.BlockOn)
	}

	result := InitResult{
		ConfigPath:  initOpts.ConfigPath,
		StatePath:   initOpts.StatePath,
		IgnorePath:  filepath.Join(initOpts.WorkingDir, ".diffpalignore"),
		ProviderSet: []string{providerID},
		Setup:       setup,
		Platform:    platform,
		Profile:     profile,
		BlockOn:     blockOn,
	}
	configIgnorePath := filepath.Join(filepath.Dir(initOpts.ConfigPath), ".gitignore")
	templateDir := filepath.Join(filepath.Dir(initOpts.ConfigPath), "templates")

	if err := os.MkdirAll(filepath.Dir(initOpts.ConfigPath), 0o755); err != nil {
		return result, err
	}
	if err := os.MkdirAll(initOpts.StatePath, 0o755); err != nil {
		return result, err
	}
	if err := writeIfMissing(initOpts.ConfigPath, composeWizardConfig(wizardConfigOptions{
		Setup:      setup,
		ProviderID: providerID,
		Platform:   platform,
		Profile:    profile,
		BlockOn:    blockOn,
	}), initOpts.Force); err != nil {
		return result, err
	}
	if err := writeIfMissing(configIgnorePath, defaultConfigGitignore, initOpts.Force); err != nil {
		return result, err
	}
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		return result, err
	}
	for _, template := range configTemplates() {
		path := filepath.Join(templateDir, template.name)
		if err := writeIfMissing(path, template.content, initOpts.Force); err != nil {
			return result, err
		}
		result.Templates = append(result.Templates, path)
	}
	if err := writeIfMissing(result.IgnorePath, defaultIgnore, initOpts.Force); err != nil {
		return result, err
	}

	return result, nil
}

func writeIfMissing(path, content string, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func composeConfig(detected []string) string {
	if len(detected) == 0 {
		detected = []string{"codex-acp"}
	}
	defaultProvider := selectedDefaultProvider(detected)
	lines := []string{
		"version: v1",
		"",
		"runtime:",
		"  providers:",
	}
	for _, p := range detected {
		lines = append(lines, fmt.Sprintf("    %s:", p))
		switch p {
		case "openai-fast":
			lines = append(lines, "      type: openai")
			lines = append(lines, "      openai:")
			lines = append(lines, "        model: gpt-5.4-mini")
		case "copilot-acp":
			lines = append(lines, "      type: copilot_acp")
			lines = append(lines, "      copilot_acp:")
			lines = append(lines, "        model: auto")
		case "codex-acp":
			lines = append(lines, "      type: codex_acp")
			lines = append(lines, "      codex_acp:")
			lines = append(lines, "        reasoning_effort: low")
		default:
			blockName := providerTypeForKey(p)
			lines = append(lines, "      type: "+blockName)
			lines = append(lines, "      "+blockName+":")
			lines = append(lines, "        mode: \"\"")
		}
	}
	lines = append(lines, "")
	lines = append(lines, "diffpal:")
	lines = append(lines, fmt.Sprintf("  provider: %s", defaultProvider))
	lines = append(lines, "  gate:")
	lines = append(lines, "    block_on: high")
	lines = append(lines, "  review:")
	lines = append(lines, "    language: en")
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

type wizardConfigOptions struct {
	Setup      string
	ProviderID string
	Platform   string
	Profile    string
	BlockOn    string
}

func composeWizardConfig(opts wizardConfigOptions) string {
	lines := []string{
		"version: v1",
		"",
		"runtime:",
		"  providers:",
		fmt.Sprintf("    %s:", opts.ProviderID),
	}
	lines = append(lines, providerConfigLinesForSetup(opts.Setup)...)
	lines = append(lines, "")
	lines = append(lines, "diffpal:")
	lines = append(lines, fmt.Sprintf("  provider: %s", opts.ProviderID))
	lines = append(lines, "  gate:")
	lines = append(lines, "    block_on: "+opts.BlockOn)
	lines = append(lines, "  review:")
	lines = append(lines, "    language: en")
	lines = append(lines, "    instructions: |")
	lines = append(lines, "      Prefer actionable findings that are directly supported by the diff.")
	if opts.Platform != "" && opts.Platform != "none" {
		lines = append(lines, "  platforms:")
		lines = append(lines, "    "+opts.Platform+": {}")
	}
	lines = append(lines, "")
	lines = append(lines, "profiles:")
	lines = append(lines, "  "+opts.Profile+":")
	lines = append(lines, "    diffpal:")
	lines = append(lines, "      gate:")
	lines = append(lines, "        block_on: "+opts.BlockOn)
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func providerConfigLinesForSetup(setup string) []string {
	switch setup {
	case "copilot-github-token":
		return []string{
			"      type: copilot_acp",
			"      copilot_acp:",
			"        model: auto",
		}
	case "generic-acp":
		return []string{
			"      type: generic_acp",
			"      generic_acp:",
			"        cmd: [\"your-acp-cli\", \"acp\", \"--stdio\"]",
		}
	default:
		return []string{
			"      type: codex_acp",
			"      codex_acp:",
			"        reasoning_effort: low",
		}
	}
}

func resolveWizardSetup(value string) (string, string, error) {
	setup := strings.ToLower(strings.TrimSpace(value))
	if setup == "" {
		setup = "codex-api-key"
	}
	switch setup {
	case "codex-api-key", "codex-subscription":
		return setup, "codex-acp", nil
	case "copilot-github-token":
		return setup, "copilot-acp", nil
	case "generic-acp":
		return setup, "generic-acp", nil
	default:
		return "", "", fmt.Errorf("invalid setup %q", value)
	}
}

func resolveWizardPlatform(workingDir, value string) (string, error) {
	platform := strings.ToLower(strings.TrimSpace(value))
	if platform == "" {
		platform = "auto"
	}
	switch platform {
	case "github", "gitlab", "azure", "none":
		return platform, nil
	case "auto":
		return detectCIPlatform(workingDir), nil
	default:
		return "", fmt.Errorf("invalid platform %q", value)
	}
}

func detectCIPlatform(workingDir string) string {
	if workingDir == "" {
		workingDir = "."
	}
	if matchesAny(filepath.Join(workingDir, ".github", "workflows", "*.yml"), filepath.Join(workingDir, ".github", "workflows", "*.yaml")) {
		return "github"
	}
	if fileExists(filepath.Join(workingDir, ".gitlab-ci.yml")) {
		return "gitlab"
	}
	if fileExists(filepath.Join(workingDir, "azure-pipelines.yml")) || fileExists(filepath.Join(workingDir, "azure-pipelines.yaml")) {
		return "azure"
	}
	if matchesAny(filepath.Join(workingDir, ".azure-pipelines", "*.yml"), filepath.Join(workingDir, ".azure-pipelines", "*.yaml")) {
		return "azure"
	}
	return "github"
}

func matchesAny(patterns ...string) bool {
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err == nil && len(matches) > 0 {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func validWizardBlockOn(value string) bool {
	switch value {
	case "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func validWizardName(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		switch r {
		case '-', '_', '.':
			continue
		default:
			return false
		}
	}
	return true
}

func selectedDefaultProvider(detected []string) string {
	for _, provider := range detected {
		if provider == "codex-acp" {
			return provider
		}
	}
	return detected[0]
}

type configTemplate struct {
	name    string
	content string
}

func configTemplates() []configTemplate {
	return []configTemplate{
		{
			name: "hosted-openai-envsubst.yaml",
			content: strings.Join([]string{
				"# Copy the relevant fields into config.yaml when you want envsubst-backed hosted config.",
				"# Config loading expands $VAR and ${VAR} before YAML parsing; missing vars fail config load.",
				"runtime:",
				"  providers:",
				"    openai-fast:",
				"      type: openai",
				"      openai:",
				"        model: \"${DIFFPAL_OPENAI_MODEL}\"",
				"        api_key: \"${OPENAI_API_KEY}\"",
				"",
			}, "\n"),
		},
		{
			name: "platform-auth.yaml",
			content: strings.Join([]string{
				"# Copy a platform block into config.yaml when enabling host publish modes.",
				"# These fields use envsubst values. Keep the placeholders quoted.",
				"diffpal:",
				"  platforms:",
				"    github:",
				"      auth:",
				"        token: \"${GITHUB_TOKEN}\"",
				"    gitlab:",
				"      auth:",
				"        job_token: \"${CI_JOB_TOKEN}\"",
				"        api_token: \"${GITLAB_TOKEN}\"",
				"    azure:",
				"      auth:",
				"        system_access_token: \"${SYSTEM_ACCESSTOKEN}\"",
				"        pat: \"${AZURE_DEVOPS_EXT_PAT}\"",
				"",
			}, "\n"),
		},
		{
			name: "profiles.yaml",
			content: strings.Join([]string{
				"# Copy profiles into config.yaml when you need scenario-specific overrides.",
				"profiles:",
				"  ci:",
				"    diffpal:",
				"      gate:",
				"        block_on: high",
				"",
			}, "\n"),
		},
	}
}

func providerTypeForKey(key string) string {
	switch key {
	case "copilot-acp":
		return "copilot_acp"
	case "gemini-acp":
		return "gemini_acp"
	case "claude-code-acp":
		return "claude_code_acp"
	case "codex-acp":
		return "codex_acp"
	default:
		return "generic_acp"
	}
}
