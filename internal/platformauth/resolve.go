package platformauth

import (
	"fmt"
	"os"
	"strings"

	"github.com/diffpal/diffpal/internal/config"
)

type Resolved struct {
	Platform string
	Mode     string
	Source   string
	token    string
}

func (r Resolved) WithToken(use func(string) error) error {
	return use(r.token)
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
	source := "diffpal.platforms.github.auth.token"
	if token == "" {
		token = strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
		source = "GITHUB_TOKEN"
	}
	if token == "" {
		return Resolved{}, fmt.Errorf("diffpal.platforms.github.auth.token or GITHUB_TOKEN is required for review github")
	}
	return Resolved{
		Platform: "github",
		Mode:     "github_token",
		Source:   source,
		token:    token,
	}, nil
}

func resolveGitLab(cfg config.Config) (Resolved, error) {
	apiToken := strings.TrimSpace(cfg.Platforms.GitLab.Auth.APIToken)
	jobToken := strings.TrimSpace(cfg.Platforms.GitLab.Auth.JobToken)
	apiSource := "diffpal.platforms.gitlab.auth.api_token"
	jobSource := "diffpal.platforms.gitlab.auth.job_token"
	if apiToken == "" {
		apiToken = strings.TrimSpace(os.Getenv("GITLAB_TOKEN"))
		apiSource = "GITLAB_TOKEN"
	}
	if jobToken == "" {
		jobToken = strings.TrimSpace(os.Getenv("CI_JOB_TOKEN"))
		jobSource = "CI_JOB_TOKEN"
	}
	if apiToken == "" && jobToken == "" {
		return Resolved{}, fmt.Errorf("diffpal.platforms.gitlab.auth.api_token, GITLAB_TOKEN, diffpal.platforms.gitlab.auth.job_token, or CI_JOB_TOKEN is required for review gitlab")
	}
	if apiToken != "" {
		return Resolved{
			Platform: "gitlab",
			Mode:     "gitlab_token",
			Source:   apiSource,
			token:    apiToken,
		}, nil
	}
	return Resolved{
		Platform: "gitlab",
		Mode:     "ci_job_token",
		Source:   jobSource,
		token:    jobToken,
	}, nil
}

func resolveADO(cfg config.Config) (Resolved, error) {
	systemToken := strings.TrimSpace(cfg.Platforms.Azure.Auth.SystemAccessToken)
	pat := strings.TrimSpace(cfg.Platforms.Azure.Auth.PAT)
	systemSource := "diffpal.platforms.azure.auth.system_access_token"
	patSource := "diffpal.platforms.azure.auth.pat"
	if systemToken == "" {
		systemToken = strings.TrimSpace(os.Getenv("SYSTEM_ACCESSTOKEN"))
		systemSource = "SYSTEM_ACCESSTOKEN"
	}
	if pat == "" {
		pat = strings.TrimSpace(os.Getenv("AZURE_DEVOPS_EXT_PAT"))
		patSource = "AZURE_DEVOPS_EXT_PAT"
	}
	if systemToken == "" && pat == "" {
		return Resolved{}, fmt.Errorf("diffpal.platforms.azure.auth.system_access_token, SYSTEM_ACCESSTOKEN, diffpal.platforms.azure.auth.pat, or AZURE_DEVOPS_EXT_PAT is required for review ado")
	}
	if systemToken != "" {
		return Resolved{
			Platform: "ado",
			Mode:     "system_access_token",
			Source:   systemSource,
			token:    systemToken,
		}, nil
	}
	return Resolved{
		Platform: "ado",
		Mode:     "pat",
		Source:   patSource,
		token:    pat,
	}, nil
}
