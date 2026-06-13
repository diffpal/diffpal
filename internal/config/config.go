package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	Provider   string                                 `json:"provider"              yaml:"provider"    mapstructure:"provider"`
	Gate       GateConfig                             `json:"gate"                  yaml:"gate"        mapstructure:"gate"`
	Providers  map[string]ProviderConfig              `json:"providers"             yaml:"providers"   mapstructure:"providers"`
	Review     ReviewConfig                           `json:"review"                yaml:"review"      mapstructure:"review"`
	Platforms  PlatformConfigs                        `json:"platforms,omitempty"   yaml:"platforms"   mapstructure:"platforms"`
	MCPServers map[string]agentconfig.MCPServerConfig `json:"mcp_servers,omitempty" yaml:"mcp_servers" mapstructure:"mcp_servers"`
}

type configDocument struct {
	Version string                  `mapstructure:"version"`
	Runtime appconfig.RuntimeConfig `mapstructure:"runtime"`
	DiffPal DiffPalConfig           `mapstructure:"diffpal"`
}

type DiffPalConfig struct {
	Provider  string          `json:"provider"            yaml:"provider"            mapstructure:"provider"`
	Gate      GateConfig      `json:"gate"                yaml:"gate"                mapstructure:"gate"`
	Review    ReviewConfig    `json:"review"              yaml:"review"              mapstructure:"review"`
	Platforms PlatformConfigs `json:"platforms,omitempty" yaml:"platforms,omitempty" mapstructure:"platforms"`
}

type GateConfig struct {
	BlockOn string `json:"block_on" yaml:"block_on" mapstructure:"block_on"`
}

type ReviewConfig struct {
	Language     string   `json:"language"     yaml:"language"     mapstructure:"language"`
	Checks       []string `json:"checks"       yaml:"checks"       mapstructure:"checks"`
	Instructions string   `json:"instructions" yaml:"instructions" mapstructure:"instructions"`
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

var defaultReviewChecks = []string{"security", "bugs", "performance", "best-practices"}

const (
	DefaultReviewMaxFiles         = 200
	DefaultReviewContextLines     = 20
	DefaultReviewMaxPatchChars    = 12000
	DefaultReviewMaxFilesPerChunk = 20
	defaultBlockOn                = "high"
)

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

	var doc configDocument
	resolvedProfile, err := appconfig.LoadConfigDocument(
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
		&doc,
	)
	if err != nil {
		return LoadedConfig{}, err
	}
	cfg := configFromDocument(doc)
	if err := cfg.ApplyEnvOverrides(); err != nil {
		return LoadedConfig{}, err
	}
	if err := cfg.Normalize(); err != nil {
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

func configFromDocument(doc configDocument) Config {
	return Config{
		Version:    doc.Version,
		Provider:   doc.DiffPal.Provider,
		Gate:       doc.DiffPal.Gate,
		Providers:  doc.Runtime.Providers,
		Review:     doc.DiffPal.Review,
		Platforms:  doc.DiffPal.Platforms,
		MCPServers: doc.Runtime.MCPServers,
	}
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
		return fmt.Errorf("diffpal.provider is required")
	}
	if _, ok := cfg.Providers[providerID]; !ok {
		return fmt.Errorf("unknown diffpal.provider %q", providerID)
	}
	blockOn := cfg.BlockOn()
	if _, ok := validSeverityThresholds[blockOn]; !ok {
		return fmt.Errorf("invalid diffpal.gate.block_on %q", blockOn)
	}
	if _, err := NormalizeReviewLanguage(cfg.Review.Language); err != nil {
		return err
	}
	if _, err := NormalizeReviewChecks(cfg.Review.Checks); err != nil {
		return err
	}
	if err := cfg.Platforms.Validate(); err != nil {
		return err
	}
	return nil
}

func (cfg *Config) Normalize() error {
	language, err := NormalizeReviewLanguage(cfg.Review.Language)
	if err != nil {
		return err
	}
	checks, err := NormalizeReviewChecks(cfg.Review.Checks)
	if err != nil {
		return err
	}
	cfg.Review.Language = language
	cfg.Review.Checks = checks
	cfg.Review.Instructions = strings.TrimSpace(cfg.Review.Instructions)
	cfg.Provider = strings.TrimSpace(cfg.Provider)
	cfg.Gate.BlockOn = strings.ToLower(strings.TrimSpace(cfg.Gate.BlockOn))
	if cfg.Gate.BlockOn == "" {
		cfg.Gate.BlockOn = defaultBlockOn
	}
	return nil
}

func (cfg Config) ProviderID() string {
	return strings.TrimSpace(cfg.Provider)
}

func (cfg Config) BlockOn() string {
	blockOn := strings.ToLower(strings.TrimSpace(cfg.Gate.BlockOn))
	if blockOn == "" {
		return defaultBlockOn
	}
	return blockOn
}

func (cfg Config) ReviewLanguage() string {
	language, err := NormalizeReviewLanguage(cfg.Review.Language)
	if err != nil {
		return "en"
	}
	return language
}

func (cfg Config) ReviewChecks() []string {
	checks, err := NormalizeReviewChecks(cfg.Review.Checks)
	if err != nil {
		return append([]string(nil), defaultReviewChecks...)
	}
	return checks
}

func (cfg Config) ReviewInstructions() string {
	return strings.TrimSpace(cfg.Review.Instructions)
}

func (cfg *Config) ApplyEnvOverrides() error {
	if value := strings.TrimSpace(os.Getenv("DIFFPAL_PROVIDER")); value != "" {
		cfg.Provider = value
	}
	if value := strings.TrimSpace(os.Getenv("DIFFPAL_BLOCK_ON")); value != "" {
		cfg.Gate.BlockOn = value
	}
	if value := strings.TrimSpace(os.Getenv("DIFFPAL_OPENAI_MODEL")); value != "" {
		cfg.setOpenAIModel(value)
	}
	if value := strings.TrimSpace(os.Getenv("DIFFPAL_REVIEW_LANGUAGE")); value != "" {
		cfg.Review.Language = value
	}
	if value := strings.TrimSpace(os.Getenv("DIFFPAL_REVIEW_CHECKS")); value != "" {
		cfg.Review.Checks = splitCommaList(value)
	}
	if value := strings.TrimSpace(os.Getenv("DIFFPAL_REVIEW_INSTRUCTIONS")); value != "" {
		cfg.Review.Instructions = value
	}
	return nil
}

func NormalizeReviewLanguage(value string) (string, error) {
	language := strings.TrimSpace(value)
	if language == "" {
		return "en", nil
	}
	if strings.ContainsAny(language, "\r\n") {
		return "", fmt.Errorf("review.language must be a single line")
	}
	return language, nil
}

func NormalizeReviewChecks(values []string) ([]string, error) {
	if len(values) == 0 {
		return append([]string(nil), defaultReviewChecks...), nil
	}
	selected := map[string]struct{}{}
	for _, raw := range values {
		for _, part := range splitCommaList(raw) {
			check, ok := canonicalReviewCheck(part)
			if !ok {
				return nil, fmt.Errorf("invalid review.checks value %q; supported values are security, bugs, performance, best-practices", part)
			}
			selected[check] = struct{}{}
		}
	}
	if len(selected) == 0 {
		return append([]string(nil), defaultReviewChecks...), nil
	}
	out := make([]string, 0, len(defaultReviewChecks))
	for _, check := range defaultReviewChecks {
		if _, ok := selected[check]; ok {
			out = append(out, check)
		}
	}
	return out, nil
}

func canonicalReviewCheck(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "security", "sec", "secure":
		return "security", true
	case "bug", "bugs":
		return "bugs", true
	case "perf", "performance":
		return "performance", true
	case "best-practice", "best-practices", "best_practice", "best_practices", "practices":
		return "best-practices", true
	default:
		return "", false
	}
}

func splitCommaList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
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
		errs = append(errs, "diffpal.platforms.github.auth.token must not be blank")
	}
	if cfg.GitLab.Auth.JobToken != "" && strings.TrimSpace(cfg.GitLab.Auth.JobToken) == "" {
		errs = append(errs, "diffpal.platforms.gitlab.auth.job_token must not be blank")
	}
	if cfg.GitLab.Auth.APIToken != "" && strings.TrimSpace(cfg.GitLab.Auth.APIToken) == "" {
		errs = append(errs, "diffpal.platforms.gitlab.auth.api_token must not be blank")
	}
	if cfg.Azure.Auth.SystemAccessToken != "" && strings.TrimSpace(cfg.Azure.Auth.SystemAccessToken) == "" {
		errs = append(errs, "diffpal.platforms.azure.auth.system_access_token must not be blank")
	}
	if cfg.Azure.Auth.PAT != "" && strings.TrimSpace(cfg.Azure.Auth.PAT) == "" {
		errs = append(errs, "diffpal.platforms.azure.auth.pat must not be blank")
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

func ProviderKeys(cfg Config) []string {
	keys := make([]string, 0, len(cfg.Providers))
	for key := range cfg.Providers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
