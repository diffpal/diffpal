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

func TestResolveGitHubUsesEnvToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "env-token")

	resolved, err := Resolve(config.Config{}, "github")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Source != "GITHUB_TOKEN" {
		t.Fatalf("Source = %q, want GITHUB_TOKEN", resolved.Source)
	}
	assertResolvedToken(t, resolved, "env-token")
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
	assertResolvedToken(t, resolved, "api-token")
}

func TestResolveGitLabUsesEnvTokens(t *testing.T) {
	t.Setenv("CI_JOB_TOKEN", "job-env-token")

	resolved, err := Resolve(config.Config{}, "gitlab")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Source != "CI_JOB_TOKEN" {
		t.Fatalf("Source = %q, want CI_JOB_TOKEN", resolved.Source)
	}
	assertResolvedToken(t, resolved, "job-env-token")
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
	assertResolvedToken(t, resolved, "system-token")
}

func TestResolveADOUsesEnvToken(t *testing.T) {
	t.Setenv("SYSTEM_ACCESSTOKEN", "system-env-token")

	resolved, err := Resolve(config.Config{}, "ado")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Source != "SYSTEM_ACCESSTOKEN" {
		t.Fatalf("Source = %q, want SYSTEM_ACCESSTOKEN", resolved.Source)
	}
	assertResolvedToken(t, resolved, "system-env-token")
}

func TestResolveGitLabFailsWhenTokensMissing(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("CI_JOB_TOKEN", "")

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
	if !strings.Contains(err.Error(), "platforms.gitlab.auth.api_token, GITLAB_TOKEN") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertResolvedToken(t *testing.T, resolved Resolved, want string) {
	t.Helper()
	var got string
	err := resolved.WithToken(func(token string) error {
		got = token
		return nil
	})
	if err != nil {
		t.Fatalf("WithToken() error = %v", err)
	}
	if got != want {
		t.Fatalf("token passed to callback = %q, want %q", got, want)
	}
}
