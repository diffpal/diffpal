package findings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultBundlePath = ".artifacts/diffpal/findings.json"

func WriteBundle(path string, bundle FindingsBundle, repo string) error {
	if path == "" {
		path = DefaultBundlePath
	}
	if err := EnsurePath(path); err != nil {
		return err
	}
	bundle.Version = ensureWriteVersion(bundle.Version)
	Normalize(&bundle, repo)
	if err := Validate(bundle); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func ReadBundle(path string) (FindingsBundle, error) {
	if path == "" {
		path = DefaultBundlePath
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return FindingsBundle{}, err
	}
	var out FindingsBundle
	if err := json.Unmarshal(raw, &out); err != nil {
		return FindingsBundle{}, err
	}
	out.Version = ensureVersion(out.Version)
	if err := Validate(out); err != nil {
		return FindingsBundle{}, err
	}
	return out, nil
}

func EnsurePath(path string) error {
	dir := filepath.Dir(path)
	if dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func ensureVersion(v string) string {
	if v == "" {
		return VersionV1
	}
	return v
}

func WriteStringBundle(path string, payload string) error {
	if path == "" {
		path = DefaultBundlePath
	}
	if err := EnsurePath(path); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(payload), 0o644)
}

func FormatBundle(bundle FindingsBundle, repo string) ([]byte, error) {
	bundle.Version = ensureWriteVersion(bundle.Version)
	Normalize(&bundle, repo)
	if err := Validate(bundle); err != nil {
		return nil, fmt.Errorf("invalid bundle: %w", err)
	}
	return json.MarshalIndent(bundle, "", "  ")
}

func ensureWriteVersion(v string) string {
	if v == "" {
		return VersionV2
	}
	return v
}
