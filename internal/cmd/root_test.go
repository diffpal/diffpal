package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestRootHelpShowsCanonicalCommands(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	text := out.String()
	for _, needle := range []string{"init", "review", "doctor", "debug", "sarif", "version"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("root help missing %q:\n%s", needle, text)
		}
	}
	for _, hidden := range []string{"\n  ci", "\n  publish", "\n  config"} {
		if strings.Contains(text, hidden) {
			t.Fatalf("root help should not show %q:\n%s", hidden, text)
		}
	}
}

func TestRootSilencesUsageAndErrorsForRunErrors(t *testing.T) {
	cmd := NewRootCommand()
	if !cmd.SilenceUsage {
		t.Fatal("root command should silence usage for runtime errors")
	}
	if !cmd.SilenceErrors {
		t.Fatal("root command should let main print runtime errors")
	}
}

func TestRootDebugFlagEnablesZerologDebugLevel(t *testing.T) {
	prevLevel := zerolog.GlobalLevel()
	defer zerolog.SetGlobalLevel(prevLevel)

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--debug", "version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := zerolog.GlobalLevel(); got != zerolog.DebugLevel {
		t.Fatalf("zerolog global level = %s, want %s", got, zerolog.DebugLevel)
	}
}

func TestRootDefaultLogLevelIsInfo(t *testing.T) {
	prevLevel := zerolog.GlobalLevel()
	defer zerolog.SetGlobalLevel(prevLevel)

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := zerolog.GlobalLevel(); got != zerolog.InfoLevel {
		t.Fatalf("zerolog global level = %s, want %s", got, zerolog.InfoLevel)
	}
}

func TestReviewHelpShowsModeSubcommands(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"review", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	text := out.String()
	for _, needle := range []string{"local", "github", "gitlab", "ado"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("review help missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "publish") || strings.Contains(text, "ci-oriented") {
		t.Fatalf("review help still references removed flow:\n%s", text)
	}
}

func TestInitHelpShowsWizardFlags(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"init", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	text := out.String()
	for _, needle := range []string{"--wizard", "--setup", "--platform", "--profile", "--block-on"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("init help missing %q:\n%s", needle, text)
		}
	}
	if strings.Contains(text, "openai-fast") {
		t.Fatalf("init help exposed internal provider name:\n%s", text)
	}
}
