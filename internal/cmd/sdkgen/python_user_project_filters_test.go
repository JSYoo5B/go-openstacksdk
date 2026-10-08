package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func checkedUserProjectFilterManifest(t *testing.T) *pythonFilterManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../../..", userProjectFilterManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := decodePythonFilterManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func TestUserProjectPythonFilterManifestKeepsOwnedParentAndInheritedDescriptors(t *testing.T) {
	manifest := checkedUserProjectFilterManifest(t)
	if !userProjectPythonFilterMetadataValid(manifest) || !reflect.DeepEqual(manifest.Proof.Files, userProjectFilterSourceHashes) {
		t.Fatal("pinned UserProject descriptor differs", manifest)
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){
		"native-user-model":         func(m *pythonFilterManifest) { m.Resource = "openstack.identity.v3.user.User" },
		"native-project-collection": func(m *pythonFilterManifest) { m.BasePath = "/projects" },
		"wrong-package": func(m *pythonFilterManifest) {
			m.SDKPackage = "github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
		},
		"parent-not-query":           func(m *pythonFilterManifest) { m.Query["user_id"] = "user_id" },
		"parent-uri":                 func(m *pythonFilterManifest) { delete(m.URI, "user_id") },
		"enabled-alias":              func(m *pythonFilterManifest) { m.Query["is_enabled"] = "is_enabled" },
		"computed-location-not-body": func(m *pythonFilterManifest) { m.Body["location"] = pythonFilterField{Field: "location"} },
		"options-conversion":         func(m *pythonFilterManifest) { m.Body["options"] = pythonFilterField{Field: "options"} },
		"local-links":                func(m *pythonFilterManifest) { delete(m.Body, "links") },
		"inheritance":                func(m *pythonFilterManifest) { m.MRO = m.MRO[1:] },
		"list-controls":              func(m *pythonFilterManifest) { m.Reserved = m.Reserved[1:] },
		"source-anchor":              func(m *pythonFilterManifest) { m.Proof.Nodes = m.Proof.Nodes[1:] },
	} {
		t.Run(name, func(t *testing.T) {
			changed := clonePythonFilterManifest(t, manifest)
			mutate(changed)
			if userProjectPythonFilterMetadataValid(changed) {
				t.Fatal("unaudited owned descriptor accepted")
			}
		})
	}
}

func TestUserProjectPythonFilterVerificationReadsFreshSourceAndRejectsProofDrift(t *testing.T) {
	manifest := checkedUserProjectFilterManifest(t)
	if err := verifyUserProjectPythonFilterManifest("", manifest); err == nil {
		t.Fatal("missing source accepted")
	}
	if err := verifyUserProjectPythonFilterManifest(t.TempDir(), manifest); err == nil || !strings.Contains(err.Error(), "Python filter source") {
		t.Fatalf("missing live source = %v", err)
	}
	source := os.Getenv("OPENSTACKSDK_SOURCE")
	if source == "" {
		source = "/private/tmp/go-openstacksdk-openstacksdk-pin-zqsdOs"
	}
	if _, err := os.Stat(filepath.Join(source, "openstack/identity/v3/project.py")); err != nil {
		t.Skip("pinned Python source unavailable; pass OPENSTACKSDK_SOURCE")
	}
	loaded, err := loadUserProjectPythonFilterManifest("../../..", source)
	if err != nil || !reflect.DeepEqual(loaded, manifest) {
		t.Fatalf("fresh owned UserProject verification = %v, %v", loaded, err)
	}
	changed := clonePythonFilterManifest(t, manifest)
	changed.Proof.Files["openstack/identity/v3/project.py"] = strings.Repeat("0", 64)
	if err := verifyUserProjectPythonFilterManifest(source, changed); err == nil {
		t.Fatal("changed checked-in source hashes accepted")
	}
	changed = clonePythonFilterManifest(t, manifest)
	changed.Proof.Nodes[0].ASTSHA256 = strings.Repeat("0", 64)
	if err := verifyUserProjectPythonFilterManifest(source, changed); err == nil {
		t.Fatal("changed AST proof accepted")
	}

	// Copy only the audited source files and corrupt a live class declaration.
	// A matching checked-in descriptor cannot grant filters for a changed source.
	drifted := t.TempDir()
	for path := range userProjectFilterSourceHashes {
		data, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		if path == "openstack/identity/v3/project.py" {
			data = append(data, []byte("\n# source drift\n")...)
		}
		target := filepath.Join(drifted, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := verifyUserProjectPythonFilterManifest(drifted, manifest); err == nil || !strings.Contains(err.Error(), "openstack/identity/v3/project.py") {
		t.Fatalf("changed live source accepted: %v", err)
	}
}
