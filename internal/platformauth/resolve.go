package platformauth

import (
	"fmt"
	"strings"

	"github.com/diffpal/diffpal/internal/config"
)

type Resolved struct {
	Platform string
	Mode     string
	Token    string
	Source   string
}

func Resolve(cfg config.Config, platform string) (Resolved, error) {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "github":
		return resolveGitHub(cfg)
	case "gitlab":
		return resolveGitLab(cfg)
	case "ado", "azure":
		return resolveADO(cfg)
	default:
		return Resolved{}, fmt.Errorf("unsupported platform %q", platform)
	}
}

func resolveGitHub(cfg config.Config) (Resolved, error) {
	token := strings.TrimSpace(cfg.Platforms.GitHub.Auth.Token)
	if token == "" {
		return Resolved{}, fmt.Errorf("platforms.github.auth.token is required for review github")
	}
	return Resolved{
		Platform: "github",
		Mode:     "github_token",
		Token:    token,
		Source:   "platforms.github.auth.token",
	}, nil
}

func resolveGitLab(cfg config.Config) (Resolved, error) {
	apiToken := strings.TrimSpace(cfg.Platforms.GitLab.Auth.APIToken)
	jobToken := strings.TrimSpace(cfg.Platforms.GitLab.Auth.JobToken)
	if apiToken == "" && jobToken == "" {
		return Resolved{}, fmt.Errorf("platforms.gitlab.auth.api_token or platforms.gitlab.auth.job_token is required for review gitlab")
	}
	if apiToken != "" {
		return Resolved{
			Platform: "gitlab",
			Mode:     "gitlab_token",
			Token:    apiToken,
			Source:   "platforms.gitlab.auth.api_token",
		}, nil
	}
	return Resolved{
		Platform: "gitlab",
		Mode:     "ci_job_token",
		Token:    jobToken,
		Source:   "platforms.gitlab.auth.job_token",
	}, nil
}

func resolveADO(cfg config.Config) (Resolved, error) {
	systemToken := strings.TrimSpace(cfg.Platforms.Azure.Auth.SystemAccessToken)
	pat := strings.TrimSpace(cfg.Platforms.Azure.Auth.PAT)
	if systemToken == "" && pat == "" {
		return Resolved{}, fmt.Errorf("platforms.azure.auth.system_access_token or platforms.azure.auth.pat is required for review ado")
	}
	if systemToken != "" {
		return Resolved{
			Platform: "ado",
			Mode:     "system_access_token",
			Token:    systemToken,
			Source:   "platforms.azure.auth.system_access_token",
		}, nil
	}
	return Resolved{
		Platform: "ado",
		Mode:     "pat",
		Token:    pat,
		Source:   "platforms.azure.auth.pat",
	}, nil
}
