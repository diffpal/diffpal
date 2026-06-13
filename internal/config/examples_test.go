package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestExampleConfigsLoad(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	matches, err := filepath.Glob(filepath.Join(root, "examples", "configs", "*", "config.yaml"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("no example configs found")
	}

	for _, source := range matches {
		source := source
		t.Run(filepath.Base(filepath.Dir(source)), func(t *testing.T) {
			t.Parallel()

			raw, err := os.ReadFile(source)
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			dir := t.TempDir()
			target := filepath.Join(dir, ".config", "diffpal", "config.yaml")
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}
			if err := os.WriteFile(target, raw, 0o644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			cfg, err := LoadConfig(dir, "", "ci")
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}
			if cfg.ProviderID() == "" {
				t.Fatal("ProviderID() = empty")
			}
			if _, ok := cfg.Providers[cfg.ProviderID()]; !ok {
				t.Fatalf("selected provider %q missing from providers", cfg.ProviderID())
			}
		})
	}
}

func TestCIExamplesAreYAML(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	var files []string
	err := filepath.WalkDir(filepath.Join(root, "examples", "ci"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".yaml") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir() error = %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no CI YAML examples found")
	}

	for _, file := range files {
		file := file
		t.Run(strings.TrimPrefix(file, root+string(os.PathSeparator)), func(t *testing.T) {
			t.Parallel()

			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			var doc any
			if err := yaml.Unmarshal(raw, &doc); err != nil {
				t.Fatalf("YAML parse error = %v", err)
			}
		})
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
