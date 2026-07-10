package provider

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSuggestProviderPoolReturnsSortedKeys(t *testing.T) {
	t.Parallel()

	keys, warnings := SuggestProviderPool([]DetectedProvider{
		{Key: "copilot-acp"},
		{Key: "openai-fast"},
	})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if len(keys) != 2 || keys[0] != "copilot-acp" || keys[1] != "openai-fast" {
		t.Fatalf("keys = %v, want sorted provider keys", keys)
	}
}

func TestSuggestProviderPoolWarnsWhenNoDetectedProviders(t *testing.T) {
	t.Parallel()

	keys, warnings := SuggestProviderPool(nil)
	if len(keys) != 0 {
		t.Fatalf("keys = %v, want none", keys)
	}
	if len(warnings) == 0 {
		t.Fatal("warnings = nil, want advisory warning")
	}
}

func TestDiscoveryAndDiagnosticsUseAvailableExecutables(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable fixture uses Unix permission bits")
	}
	dir := t.TempDir()
	for _, name := range []string{"git", "codex"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", path, err)
		}
	}
	t.Setenv("PATH", dir)

	if !HasExecutable("git") || HasExecutable("missing") {
		t.Fatal("HasExecutable() did not reflect PATH")
	}
	detected := AutodetectProviders()
	keys := make([]string, 0, len(detected))
	for _, item := range detected {
		keys = append(keys, item.Key)
	}
	if got := strings.Join(keys, ","); got != "openai-fast,codex-acp" {
		t.Fatalf("detected providers = %q, want openai-fast,codex-acp", got)
	}
	if got := strings.Join(AutoPoolDefaults(), ","); got != "openai-fast,codex-acp" {
		t.Fatalf("auto pool = %q, want openai-fast,codex-acp", got)
	}
	diagnostics := strings.Join(Diagnostics(), "\n")
	if !strings.Contains(diagnostics, "git detected") || !strings.Contains(diagnostics, "OpenAI Codex ACP") {
		t.Fatalf("Diagnostics() = %q", diagnostics)
	}
}
