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
	return writeArtifact(path, raw)
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
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.Chmod(dir, 0o700)
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
	return writeArtifact(path, []byte(payload))
}

func writeArtifact(path string, payload []byte) (err error) {
	dir := filepath.Dir(path)
	if dir == "" {
		dir = "."
	}
	tmp, err := os.CreateTemp(dir, ".diffpal-artifact-*")
	if err != nil {
		return fmt.Errorf("create temporary artifact: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		if err != nil {
			_ = os.Remove(tmpPath)
		}
	}()
	if err = tmp.Chmod(0o600); err != nil {
		return fmt.Errorf("set artifact permissions: %w", err)
	}
	if _, err = tmp.Write(payload); err != nil {
		return fmt.Errorf("write artifact: %w", err)
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("sync artifact: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close artifact: %w", err)
	}
	if err = os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace artifact: %w", err)
	}
	return nil
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
		return VersionV3
	}
	return v
}
