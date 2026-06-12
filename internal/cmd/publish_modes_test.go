package cmd

import (
	"reflect"
	"testing"
)

func TestDefaultModesMatchProductContract(t *testing.T) {
	t.Parallel()

	cases := []struct {
		platform string
		want     []string
	}{
		{platform: "github", want: []string{"check-run", "comments", "sarif", "summary"}},
		{platform: "gitlab", want: []string{"code-quality", "discussions", "sarif", "summary"}},
		{platform: "azure", want: []string{"threads", "status", "summary"}},
	}
	for _, tc := range cases {
		if got := defaultModes(tc.platform); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("defaultModes(%q) = %v, want %v", tc.platform, got, tc.want)
		}
	}
}

func TestResolvePublishModesUsesFeedbackWhenModeIsEmpty(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		platform string
		feedback string
		want     []string
	}{
		{name: "github summary", platform: "github", feedback: "summary", want: []string{"check-run", "sarif", "summary"}},
		{name: "github balanced", platform: "github", feedback: "balanced", want: []string{"check-run", "comments", "sarif", "summary"}},
		{name: "azure summary", platform: "azure", feedback: "summary", want: []string{"status", "summary"}},
		{name: "azure balanced", platform: "azure", feedback: "balanced", want: []string{"threads", "status", "summary"}},
		{name: "gitlab summary", platform: "gitlab", feedback: "summary", want: []string{"code-quality", "sarif", "summary"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, profile, err := resolvePublishModes(tc.platform, nil, tc.feedback)
			if err != nil {
				t.Fatalf("resolvePublishModes() error = %v", err)
			}
			if profile != FeedbackProfile(tc.feedback) {
				t.Fatalf("profile = %q, want %q", profile, tc.feedback)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("modes = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolvePublishModesExplicitModesOverrideFeedback(t *testing.T) {
	t.Parallel()

	want := []string{"status"}
	got, profile, err := resolvePublishModes("azure", want, "summary")
	if err != nil {
		t.Fatalf("resolvePublishModes() error = %v", err)
	}
	if profile != "" {
		t.Fatalf("profile = %q, want empty for explicit modes", profile)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("modes = %v, want %v", got, want)
	}
}

func TestResolvePublishModesRejectsInvalidFeedback(t *testing.T) {
	t.Parallel()

	if _, _, err := resolvePublishModes("github", nil, "verbose"); err == nil {
		t.Fatal("resolvePublishModes() error = nil, want invalid feedback error")
	}
}
