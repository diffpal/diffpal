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
		detected = []string{"copilot-acp"}
	}
	defaultProvider := selectedDefaultProvider(detected)
	lines := []string{
		"version: v1",
		"",
		"defaults:",
		fmt.Sprintf("  provider: %s", defaultProvider),
		"  policy: default",
		"",
		"providers:",
	}
	for _, p := range detected {
		lines = append(lines, fmt.Sprintf("  %s:", p))
		switch p {
		case "openai-fast":
			lines = append(lines, "    type: openai")
			lines = append(lines, "    openai:")
			lines = append(lines, "      model: gpt-5.4-mini")
		case "copilot-acp":
			lines = append(lines, "    type: copilot_acp")
			lines = append(lines, "    copilot_acp: {}")
		default:
			blockName := providerTypeForKey(p)
			lines = append(lines, "    type: "+blockName)
			lines = append(lines, "    "+blockName+":")
			lines = append(lines, "      mode: \"\"")
		}
	}
	lines = append(lines, "")
	lines = append(lines, "policies:")
	lines = append(lines, "  default:")
	lines = append(lines, "    block_on: high")
	lines = append(lines, "")
	lines = append(lines, "review:")
	lines = append(lines, "  context_lines: 20")
	lines = append(lines, "  max_files: 200")
	lines = append(lines, "  language: en")
	lines = append(lines, "  checks:")
	lines = append(lines, "    - bugs")
	lines = append(lines, "    - performance")
	lines = append(lines, "    - best-practices")
	lines = append(lines, "  chunking:")
	lines = append(lines, "    max_patch_chars: 12000")
	lines = append(lines, "    max_files_per_chunk: 20")
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func selectedDefaultProvider(detected []string) string {
	for _, provider := range detected {
		if provider == "copilot-acp" {
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
				"providers:",
				"  openai-fast:",
				"    type: openai",
				"    openai:",
				"      model: \"${DIFFPAL_OPENAI_MODEL}\"",
				"      api_key: \"${OPENAI_API_KEY}\"",
				"",
			}, "\n"),
		},
		{
			name: "platform-auth.yaml",
			content: strings.Join([]string{
				"# Copy a platform block into config.yaml when enabling host publish modes.",
				"# These fields use envsubst values. Keep the placeholders quoted.",
				"platforms:",
				"  github:",
				"    auth:",
				"      token: \"${GITHUB_TOKEN}\"",
				"  gitlab:",
				"    auth:",
				"      job_token: \"${CI_JOB_TOKEN}\"",
				"      api_token: \"${GITLAB_TOKEN}\"",
				"  azure:",
				"    auth:",
				"      system_access_token: \"${SYSTEM_ACCESSTOKEN}\"",
				"      pat: \"${AZURE_DEVOPS_EXT_PAT}\"",
				"",
			}, "\n"),
		},
		{
			name: "profiles.yaml",
			content: strings.Join([]string{
				"# Copy profiles into config.yaml when you need scenario-specific overrides.",
				"profiles:",
				"  ci:",
				"    defaults:",
				"      policy: default",
				"    policies:",
				"      default:",
				"        block_on: high",
				"    review:",
				"      context_lines: 20",
				"      max_files: 200",
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
