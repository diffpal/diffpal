package provider

import (
	"os"
	"os/exec"
	"sort"
)

type DetectedProvider struct {
	Key         string
	Source      string
	Type        string
	Description string
}

type Candidate struct {
	Key         string
	Bin         string
	Type        string
	Description string
}

var defaultCandidates = []Candidate{
	{Key: "openai-fast", Bin: "", Type: "hosted", Description: "hosted provider placeholder"},
	{Key: "copilot-acp", Bin: "copilot", Type: "copilot_acp", Description: "GitHub Copilot ACP"},
	{Key: "gemini-acp", Bin: "gemini", Type: "gemini_acp", Description: "Google Gemini ACP"},
	{Key: "claude-code-acp", Bin: "claude", Type: "claude_code_acp", Description: "Anthropic Claude Code ACP"},
	{Key: "codex-acp", Bin: "codex", Type: "codex_acp", Description: "OpenAI Codex ACP"},
}

func AutodetectProviders() []DetectedProvider {
	out := []DetectedProvider{}
	for _, candidate := range defaultCandidates {
		if candidate.Bin != "" {
			if _, err := exec.LookPath(candidate.Bin); err != nil {
				continue
			}
		}
		out = append(out, DetectedProvider{
			Key:         candidate.Key,
			Source:      candidate.Bin,
			Type:        candidate.Type,
			Description: candidate.Description,
		})
	}
	return out
}

func SuggestProviderPool(detected []DetectedProvider) ([]string, []string) {
	keys := make([]string, 0, len(detected))
	warnings := []string{}
	for _, d := range detected {
		keys = append(keys, d.Key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		warnings = append(warnings, "no ACP providers detected; add providers entries in .config/diffpal/config.yaml manually")
	}
	return keys, warnings
}

func HasExecutable(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

func Diagnostics() []string {
	out := []string{}
	if HasExecutable("git") {
		out = append(out, "provider_hint: git detected on PATH")
	} else {
		out = append(out, "provider_warning: git missing from PATH")
	}
	for _, candidate := range defaultCandidates {
		if candidate.Bin == "" {
			continue
		}
		name := candidate.Description
		if !HasExecutable(candidate.Bin) {
			continue
		}
		out = append(out, "provider_detected: "+name+" via "+candidate.Bin)
	}
	if len(out) == 1 {
		out = append(out, "provider_warning: no optional ACP providers detected in PATH")
	}
	if len(AutoPoolDefaults()) == 0 {
		out = append(out, "provider_warning: no default providers configured")
	}
	return out
}

func AutoPoolDefaults() []string {
	keys := []string{}
	for _, candidate := range defaultCandidates {
		if candidate.Bin != "" && !HasExecutable(candidate.Bin) {
			continue
		}
		keys = append(keys, candidate.Key)
	}
	return keys
}

func EnsureStatePath(path string) error {
	return os.MkdirAll(path, 0o755)
}
