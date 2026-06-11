package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/normahq/norma/pkg/runtime/agentconfig"
	"github.com/normahq/norma/pkg/runtime/appconfig"
)

const (
	AppName        = "diffpal"
	configDirName  = ".config/diffpal"
	configFileName = "config.yaml"
)

//go:embed defaults.yaml
var defaultConfigYAML []byte

type Config struct {
	Version    string                                 `json:"version"               yaml:"version"     mapstructure:"version"`
	Defaults   DefaultsConfig                         `json:"defaults"              yaml:"defaults"    mapstructure:"defaults"`
	Providers  map[string]ProviderConfig              `json:"providers"             yaml:"providers"   mapstructure:"providers"`
	Policies   map[string]PolicyConfig                `json:"policies"              yaml:"policies"    mapstructure:"policies"`
	Review     ReviewConfig                           `json:"review"                yaml:"review"      mapstructure:"review"`
	Platforms  PlatformConfigs                        `json:"platforms,omitempty"   yaml:"platforms"   mapstructure:"platforms"`
	MCPServers map[string]agentconfig.MCPServerConfig `json:"mcp_servers,omitempty" yaml:"mcp_servers" mapstructure:"mcp_servers"`
}

type DefaultsConfig struct {
	Provider string `json:"provider" yaml:"provider" mapstructure:"provider"`
	Policy   string `json:"policy"   yaml:"policy"   mapstructure:"policy"`
}

type PolicyConfig struct {
	BlockOn string `json:"block_on" yaml:"block_on" mapstructure:"block_on"`
}

type ReviewConfig struct {
	ContextLines int            `json:"context_lines" yaml:"context_lines" mapstructure:"context_lines"`
	MaxFiles     int            `json:"max_files"     yaml:"max_files"     mapstructure:"max_files"`
	Chunking     ChunkingConfig `json:"chunking"      yaml:"chunking"      mapstructure:"chunking"`
}

type ChunkingConfig struct {
	MaxPatchChars    int `json:"max_patch_chars"     yaml:"max_patch_chars"     mapstructure:"max_patch_chars"`
	MaxFilesPerChunk int `json:"max_files_per_chunk" yaml:"max_files_per_chunk" mapstructure:"max_files_per_chunk"`
}

type PlatformConfigs struct {
	GitHub GitHubPlatformConfig `json:"github,omitempty" yaml:"github,omitempty" mapstructure:"github"`
	GitLab GitLabPlatformConfig `json:"gitlab,omitempty" yaml:"gitlab,omitempty" mapstructure:"gitlab"`
	Azure  AzurePlatformConfig  `json:"azure,omitempty"  yaml:"azure,omitempty"  mapstructure:"azure"`
}

type GitHubPlatformConfig struct {
	Auth           GitHubAuthConfig           `json:"auth,omitempty"            yaml:"auth,omitempty"            mapstructure:"auth"`
	SummaryComment GitHubSummaryCommentConfig `json:"summary_comment,omitempty" yaml:"summary_comment,omitempty" mapstructure:"summary_comment"`
}

type GitHubSummaryCommentConfig struct {
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty" mapstructure:"enabled"`
}

type GitHubAuthConfig struct {
	Token string `json:"token,omitempty" yaml:"token,omitempty" mapstructure:"token"`
}

type GitLabPlatformConfig struct {
	Auth GitLabAuthConfig `json:"auth,omitempty" yaml:"auth,omitempty" mapstructure:"auth"`
}

type GitLabAuthConfig struct {
	JobToken string `json:"job_token,omitempty" yaml:"job_token,omitempty" mapstructure:"job_token"`
	APIToken string `json:"api_token,omitempty" yaml:"api_token,omitempty" mapstructure:"api_token"`
}

type AzurePlatformConfig struct {
	Auth AzureAuthConfig `json:"auth,omitempty" yaml:"auth,omitempty" mapstructure:"auth"`
}

type AzureAuthConfig struct {
	SystemAccessToken string `json:"system_access_token,omitempty" yaml:"system_access_token,omitempty" mapstructure:"system_access_token"`
	PAT               string `json:"pat,omitempty" yaml:"pat,omitempty" mapstructure:"pat"`
}

type LoadedConfig struct {
	Config  Config
	Profile string
	Path    string
}

type ProviderConfig = agentconfig.Config

var validSeverityThresholds = map[string]struct{}{
	"low":      {},
	"medium":   {},
	"high":     {},
	"critical": {},
}

func LoadConfig(workingDir, configDir, profile string) (Config, error) {
	loaded, err := LoadConfigWithMetadata(workingDir, configDir, profile)
	if err != nil {
		return Config{}, err
	}
	return loaded.Config, nil
}

func LoadConfigWithMetadata(workingDir, configDir, profile string) (LoadedConfig, error) {
	resolvedWorkingDir, err := resolveWorkingDir(workingDir)
	if err != nil {
		return LoadedConfig{}, err
	}
	selectedPath, _, err := ResolveConfigPath(resolvedWorkingDir, configDir)
	if err != nil {
		return LoadedConfig{}, err
	}

	selectedProfile := strings.TrimSpace(profile)
	if selectedProfile == "" {
		selectedProfile = strings.TrimSpace(os.Getenv("DIFFPAL_PROFILE"))
	}

	settings, resolvedProfile, err := appconfig.LoadResolvedSettings(
		appconfig.RuntimeLoadOptions{
			WorkingDir: resolvedWorkingDir,
			ConfigDir:  strings.TrimSpace(configDir),
			Profile:    selectedProfile,
		},
		appconfig.AppLoadOptions{
			AppName:            AppName,
			EnvPrefix:          "DIFFPAL",
			DefaultsYAML:       defaultConfigYAML,
			UseDotConfigAppDir: true,
		},
	)
	if err != nil {
		return LoadedConfig{}, err
	}
	if err := validateRawSettings(settings); err != nil {
		return LoadedConfig{}, err
	}

	var cfg Config
	if err := appconfig.DecodeSettings(settings, &cfg); err != nil {
		return LoadedConfig{}, fmt.Errorf("decode config: %w", err)
	}
	if err := cfg.ApplyEnvOverrides(); err != nil {
		return LoadedConfig{}, err
	}
	if err := cfg.Validate(); err != nil {
		return LoadedConfig{}, err
	}

	return LoadedConfig{
		Config:  cfg,
		Profile: resolvedProfile,
		Path:    selectedPath,
	}, nil
}

func (cfg Config) Validate() error {
	if cfg.Version != "" && cfg.Version != "v1" {
		return fmt.Errorf("unsupported config version %q", cfg.Version)
	}
	runtimeConfig := appconfig.RuntimeConfig{
		Providers:  cfg.Providers,
		MCPServers: cfg.MCPServers,
	}
	if err := runtimeConfig.Validate(); err != nil {
		return fmt.Errorf("validate runtime config: %w", err)
	}
	providerID := cfg.ProviderID()
	if providerID == "" {
		return fmt.Errorf("defaults.provider is required")
	}
	if _, ok := cfg.Providers[providerID]; !ok {
		return fmt.Errorf("unknown defaults.provider %q", providerID)
	}
	policyName := cfg.PolicyName()
	policyCfg, ok := cfg.Policies[policyName]
	if !ok {
		return fmt.Errorf("unknown defaults.policy %q", policyName)
	}
	if policyCfg.BlockOn == "" {
		return fmt.Errorf("policies.%s.block_on is required", policyName)
	}
	if _, ok := validSeverityThresholds[policyCfg.BlockOn]; !ok {
		return fmt.Errorf("invalid policies.%s.block_on %q", policyName, policyCfg.BlockOn)
	}
	if err := cfg.Platforms.Validate(); err != nil {
		return err
	}
	return nil
}

func (cfg Config) ProviderID() string {
	return strings.TrimSpace(cfg.Defaults.Provider)
}

func (cfg Config) PolicyName() string {
	name := strings.TrimSpace(cfg.Defaults.Policy)
	if name == "" {
		return "default"
	}
	return name
}

func (cfg Config) BlockOn() string {
	policyCfg, ok := cfg.Policies[cfg.PolicyName()]
	if !ok {
		return ""
	}
	return strings.TrimSpace(policyCfg.BlockOn)
}

func (cfg *Config) ApplyEnvOverrides() error {
	if value := strings.TrimSpace(os.Getenv("DIFFPAL_PROVIDER")); value != "" {
		cfg.Defaults.Provider = value
	}
	if value := strings.TrimSpace(os.Getenv("DIFFPAL_POLICY")); value != "" {
		cfg.Defaults.Policy = value
	}
	if value := strings.TrimSpace(os.Getenv("DIFFPAL_BLOCK_ON")); value != "" {
		cfg.setSelectedBlockOn(value)
	}
	if value := strings.TrimSpace(os.Getenv("DIFFPAL_OPENAI_MODEL")); value != "" {
		cfg.setOpenAIModel(value)
	}
	if value := strings.TrimSpace(os.Getenv("DIFFPAL_REVIEW_MAX_FILES")); value != "" {
		parsed, err := parseNonNegativeEnvInt("DIFFPAL_REVIEW_MAX_FILES", value)
		if err != nil {
			return err
		}
		cfg.Review.MaxFiles = parsed
	}
	if value := strings.TrimSpace(os.Getenv("DIFFPAL_REVIEW_CONTEXT_LINES")); value != "" {
		parsed, err := parseNonNegativeEnvInt("DIFFPAL_REVIEW_CONTEXT_LINES", value)
		if err != nil {
			return err
		}
		cfg.Review.ContextLines = parsed
	}
	return nil
}

func parseNonNegativeEnvInt(name, value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", name, value, err)
	}
	if parsed < 0 {
		return 0, fmt.Errorf("invalid %s %q: must be non-negative", name, value)
	}
	return parsed, nil
}

func (cfg *Config) setSelectedBlockOn(value string) {
	name := cfg.PolicyName()
	if cfg.Policies == nil {
		cfg.Policies = map[string]PolicyConfig{}
	}
	policyCfg := cfg.Policies[name]
	policyCfg.BlockOn = value
	cfg.Policies[name] = policyCfg
}

func (cfg *Config) setOpenAIModel(value string) {
	for name, providerCfg := range cfg.Providers {
		if !strings.EqualFold(strings.TrimSpace(providerCfg.Type), "openai") || providerCfg.OpenAI == nil {
			continue
		}
		block := *providerCfg.OpenAI
		block.Model = value
		providerCfg.OpenAI = &block
		cfg.Providers[name] = providerCfg
	}
}

func (cfg PlatformConfigs) Validate() error {
	var errs []string
	if cfg.GitHub.Auth.Token != "" && strings.TrimSpace(cfg.GitHub.Auth.Token) == "" {
		errs = append(errs, "platforms.github.auth.token must not be blank")
	}
	if cfg.GitLab.Auth.JobToken != "" && strings.TrimSpace(cfg.GitLab.Auth.JobToken) == "" {
		errs = append(errs, "platforms.gitlab.auth.job_token must not be blank")
	}
	if cfg.GitLab.Auth.APIToken != "" && strings.TrimSpace(cfg.GitLab.Auth.APIToken) == "" {
		errs = append(errs, "platforms.gitlab.auth.api_token must not be blank")
	}
	if cfg.Azure.Auth.SystemAccessToken != "" && strings.TrimSpace(cfg.Azure.Auth.SystemAccessToken) == "" {
		errs = append(errs, "platforms.azure.auth.system_access_token must not be blank")
	}
	if cfg.Azure.Auth.PAT != "" && strings.TrimSpace(cfg.Azure.Auth.PAT) == "" {
		errs = append(errs, "platforms.azure.auth.pat must not be blank")
	}
	if len(errs) == 0 {
		return nil
	}
	sort.Strings(errs)
	return fmt.Errorf("platform config validation failed: %s", strings.Join(errs, "; "))
}

func (cfg GitHubPlatformConfig) SummaryCommentEnabled() bool {
	return cfg.SummaryComment.Enabled == nil || *cfg.SummaryComment.Enabled
}

func ConfigDir(workingDir string) string {
	trimmed := strings.TrimSpace(workingDir)
	if trimmed == "" {
		return configDirName
	}
	return filepath.Join(trimmed, ".config", AppName)
}

func ConfigPath(workingDir, configuredRoot string) string {
	trimmedRoot := strings.TrimSpace(configuredRoot)
	if trimmedRoot != "" {
		if workingDir != "" && !filepath.IsAbs(trimmedRoot) {
			trimmedRoot = filepath.Join(workingDir, trimmedRoot)
		}
		return filepath.Join(trimmedRoot, AppName, configFileName)
	}
	return filepath.Join(ConfigDir(workingDir), configFileName)
}

func StatePath(workingDir string) string {
	return filepath.Join(ConfigDir(workingDir), "state")
}

func ConfigExists(workingDir, configuredRoot string) (bool, string, error) {
	selectedPath, found, err := ResolveConfigPath(workingDir, configuredRoot)
	if err != nil {
		return false, selectedPath, err
	}
	return found, selectedPath, nil
}

func ResolveConfigPath(workingDir, configuredRoot string) (string, bool, error) {
	searchPaths := configSearchPaths(workingDir, configuredRoot)
	for _, path := range searchPaths {
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", false, fmt.Errorf("stat config file %q: %w", path, err)
		}
		return path, true, nil
	}
	if len(searchPaths) == 0 {
		return "", false, nil
	}
	return searchPaths[len(searchPaths)-1], false, nil
}

func resolveWorkingDir(workingDir string) (string, error) {
	if strings.TrimSpace(workingDir) != "" {
		return filepath.Abs(workingDir)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current working directory: %w", err)
	}
	return filepath.Abs(cwd)
}

func validateRawSettings(settings map[string]any) error {
	if settings == nil {
		return fmt.Errorf("config settings are empty")
	}
	for _, key := range []string{"diffpal", "runtime"} {
		if _, ok := settings[key]; ok {
			return fmt.Errorf("top-level key %q is not supported; use defaults, providers, policies, review, and platforms", key)
		}
	}
	for _, key := range []string{"defaults", "providers", "policies", "review"} {
		if _, ok := settings[key]; !ok {
			return fmt.Errorf("config key %q is required", key)
		}
	}
	if err := validatePlatformSettings(settings["platforms"], "platforms"); err != nil {
		return err
	}
	rawProfiles, ok := settings["profiles"]
	if !ok || rawProfiles == nil {
		return nil
	}
	profiles, ok := toStringAnyMap(rawProfiles)
	if !ok {
		return fmt.Errorf("top-level key %q must be an object", "profiles")
	}
	for name, rawProfile := range profiles {
		profileMap, ok := toStringAnyMap(rawProfile)
		if !ok {
			return fmt.Errorf("profiles.%s must be an object", name)
		}
		if err := validatePlatformSettings(profileMap["platforms"], "profiles."+name+".platforms"); err != nil {
			return err
		}
	}
	return nil
}

func validatePlatformSettings(rawPlatforms any, path string) error {
	if rawPlatforms == nil {
		return nil
	}
	platformsMap, ok := toStringAnyMap(rawPlatforms)
	if !ok {
		return fmt.Errorf("%s must be an object", path)
	}
	for host, rawPlatform := range platformsMap {
		switch host {
		case "github", "gitlab", "azure":
		default:
			return fmt.Errorf("%s.%s is not supported", path, host)
		}
		platformMap, ok := toStringAnyMap(rawPlatform)
		if !ok {
			return fmt.Errorf("%s.%s must be an object", path, host)
		}
		if _, exists := platformMap["enabled"]; exists {
			return fmt.Errorf("%s.%s.enabled is not supported", path, host)
		}
		if rawAuth, exists := platformMap["auth"]; exists && rawAuth != nil {
			authMap, ok := toStringAnyMap(rawAuth)
			if !ok {
				return fmt.Errorf("%s.%s.auth must be an object", path, host)
			}
			if err := validatePlatformAuthSettings(authMap, path, host); err != nil {
				return err
			}
		}
	}
	return nil
}

func validatePlatformAuthSettings(authMap map[string]any, path string, host string) error {
	allowed := map[string]struct{}{}
	switch host {
	case "github":
		allowed["token"] = struct{}{}
	case "gitlab":
		allowed["api_token"] = struct{}{}
		allowed["job_token"] = struct{}{}
	case "azure":
		allowed["system_access_token"] = struct{}{}
		allowed["pat"] = struct{}{}
	}
	for key := range authMap {
		if _, ok := allowed[key]; ok {
			continue
		}
		return fmt.Errorf("%s.%s.auth.%s is not supported", path, host, key)
	}
	return nil
}

func configSearchPaths(workingDir, configuredRoot string) []string {
	paths := make([]string, 0, 3)

	if extra := strings.TrimSpace(configuredRoot); extra != "" {
		if !filepath.IsAbs(extra) && workingDir != "" {
			extra = filepath.Join(workingDir, extra)
		}
		paths = append(paths,
			filepath.Join(extra, AppName, configFileName),
			filepath.Join(extra, configFileName),
		)
	}
	if workingDir != "" {
		paths = append(paths, filepath.Join(workingDir, ".config", AppName, configFileName))
	}

	return dedupePaths(paths)
}

func dedupePaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		cleaned := filepath.Clean(strings.TrimSpace(p))
		if cleaned == "." || cleaned == "" {
			continue
		}
		if _, exists := seen[cleaned]; exists {
			continue
		}
		seen[cleaned] = struct{}{}
		out = append(out, cleaned)
	}
	return out
}

func toStringAnyMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, raw := range typed {
			keyString, ok := key.(string)
			if !ok {
				return nil, false
			}
			out[keyString] = raw
		}
		return out, true
	default:
		return nil, false
	}
}

func ProviderKeys(cfg Config) []string {
	keys := make([]string, 0, len(cfg.Providers))
	for key := range cfg.Providers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
