package platformauth

import (
	"strings"
	"testing"

	"github.com/diffpal/diffpal/internal/config"
)

func TestResolveGitHubUsesConfiguredToken(t *testing.T) {
	cfg := config.Config{
		Platforms: config.PlatformConfigs{
			GitHub: config.GitHubPlatformConfig{
				Auth: config.GitHubAuthConfig{Token: "token"},
			},
		},
	}

	resolved, err := Resolve(cfg, "github")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Mode != "github_token" {
		t.Fatalf("Mode = %q, want github_token", resolved.Mode)
	}
	if resolved.Source != "platforms.github.auth.token" {
		t.Fatalf("Source = %q, want platforms.github.auth.token", resolved.Source)
	}
}

func TestResolveGitLabPrefersAPITokenOverJobToken(t *testing.T) {
	cfg := config.Config{
		Platforms: config.PlatformConfigs{
			GitLab: config.GitLabPlatformConfig{
				Auth: config.GitLabAuthConfig{
					APIToken: "api-token",
					JobToken: "job-token",
				},
			},
		},
	}

	resolved, err := Resolve(cfg, "gitlab")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Mode != "gitlab_token" {
		t.Fatalf("Mode = %q, want gitlab_token", resolved.Mode)
	}
	if resolved.Token != "api-token" {
		t.Fatalf("Token = %q, want api-token", resolved.Token)
	}
}

func TestResolveADOPrefersSystemAccessTokenOverPAT(t *testing.T) {
	cfg := config.Config{
		Platforms: config.PlatformConfigs{
			Azure: config.AzurePlatformConfig{
				Auth: config.AzureAuthConfig{
					SystemAccessToken: "system-token",
					PAT:               "pat-token",
				},
			},
		},
	}

	resolved, err := Resolve(cfg, "ado")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Mode != "system_access_token" {
		t.Fatalf("Mode = %q, want system_access_token", resolved.Mode)
	}
	if resolved.Token != "system-token" {
		t.Fatalf("Token = %q, want system-token", resolved.Token)
	}
}

func TestResolveGitLabFailsWhenTokensMissing(t *testing.T) {
	cfg := config.Config{
		Platforms: config.PlatformConfigs{
			GitLab: config.GitLabPlatformConfig{
				Auth: config.GitLabAuthConfig{},
			},
		},
	}

	_, err := Resolve(cfg, "gitlab")
	if err == nil {
		t.Fatal("Resolve() error = nil, want missing token error")
	}
	if !strings.Contains(err.Error(), "platforms.gitlab.auth.api_token or platforms.gitlab.auth.job_token") {
		t.Fatalf("unexpected error: %v", err)
	}
}
