package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/diffpal/diffpal/internal/findings"
)

func TestInitCommandGeneratesWorkspaceWithoutState(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	var out bytes.Buffer
	cmd := newInitCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".config", "diffpal", "config.yaml")); err != nil {
		t.Fatalf("generated config: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".config", "diffpal", "state")); !os.IsNotExist(err) {
		t.Fatalf("obsolete state directory exists: %v", err)
	}
	if !strings.Contains(out.String(), "initialized diffpal workspace") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestInitCommandWizardReportsSelectedSetup(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	var out bytes.Buffer
	cmd := newInitCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--wizard", "--setup", "opencode-acp", "--platform", "none", "--profile", "local", "--block-on", "medium"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, value := range []string{"setup: opencode-acp", "platform: none", "profile: local", "block_on: medium"} {
		if !strings.Contains(out.String(), value) {
			t.Fatalf("output missing %q: %s", value, out.String())
		}
	}
}

func TestDoctorCommandPrintsLocalDiagnostics(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	var out bytes.Buffer
	cmd := newDoctorCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "doctor runtime:") || !strings.Contains(out.String(), "local mode does not require platform authorization") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestSARIFCommandConvertsFindingsBundle(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "findings.json")
	output := filepath.Join(dir, "result.sarif")
	bundle := findings.FindingsBundle{
		ReviewID: "review",
		HeadSHA:  "head-a",
		Findings: []findings.Finding{{
			ReviewID:   "review",
			Category:   "correctness",
			Severity:   "high",
			Confidence: 0.9,
			Path:       "main.go",
			StartLine:  2,
			EndLine:    2,
			Title:      "bug",
			Message:    "bug",
			Evidence:   findings.NewEvidence("line 2"),
			Impact:     findings.NewImpact("failure"),
		}},
	}
	if err := findings.WriteBundle(input, bundle, "acme/repo"); err != nil {
		t.Fatalf("WriteBundle() error = %v", err)
	}

	var out bytes.Buffer
	cmd := newSARIFCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--input", input, "--out", output})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(raw), `"version": "2.1.0"`) || !strings.Contains(out.String(), "findings=1") {
		t.Fatalf("SARIF output = %s\ncommand output = %s", raw, out.String())
	}
}
