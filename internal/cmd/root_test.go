package cmd

import (
	"bytes"
	"strings"
	"testing"
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
	for _, needle := range []string{"init", "review", "doctor", "sarif", "version"} {
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
