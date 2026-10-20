package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// These resources share the same AST-only verification protocol. Each caller
// still owns its exact descriptor, inheritance and fixed source hashes.
func verifyKeyManagerPythonFilterManifest(source string, manifest *pythonFilterManifest, label, resource string, hashes map[string]string, valid func(*pythonFilterManifest) bool) error {
	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("-openstacksdk-source is required for audited %s semantic filters", label)
	}
	if !valid(manifest) {
		return fmt.Errorf("audited %s Python filter identity or descriptor changed", label)
	}
	if !reflect.DeepEqual(manifest.Proof.Files, hashes) {
		return fmt.Errorf("audited %s Python source proof changed", label)
	}
	for path, expected := range hashes {
		data, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(path)))
		if err != nil {
			return fmt.Errorf("Python filter source %s: %w", path, err)
		}
		digest := sha256.Sum256(data)
		if hex.EncodeToString(digest[:]) != expected {
			return fmt.Errorf("audited Python filter source %s changed", path)
		}
	}
	fresh, err := extractPythonFilterManifestTarget(source, resource)
	if err != nil {
		return err
	}
	if manifest.Proof.PythonParser != fresh.Proof.PythonParser {
		return fmt.Errorf("Python filter AST parser version %s differs from audited %s", fresh.Proof.PythonParser, manifest.Proof.PythonParser)
	}
	if !reflect.DeepEqual(manifest, fresh) {
		return fmt.Errorf("%s Python filter manifest differs from independent live-source extraction", label)
	}
	return nil
}

func loadKeyManagerPythonFilterManifest(root, source, label, path string, verify func(string, *pythonFilterManifest) error) (*pythonFilterManifest, error) {
	if strings.TrimSpace(source) == "" {
		return nil, fmt.Errorf("-openstacksdk-source is required for audited %s semantic filters", label)
	}
	data, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		return nil, fmt.Errorf("%s Python filter manifest: %w", label, err)
	}
	manifest, err := decodePythonFilterManifest(data)
	if err != nil {
		return nil, err
	}
	if err := verify(source, manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}
