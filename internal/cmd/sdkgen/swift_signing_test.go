package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSwiftSigningPreservesNativeAndRegistry(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	for _, name := range []string{"accounts", "containers", "objects", "swauth"} {
		if err := g.generate(upstreamModule + "/openstack/objectstorage/v1/" + name); err != nil {
			t.Fatal(err)
		}
	}
	g.collections = append(g.collections, sdkOwnedCollections...)
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"objectstorage/v1/accounts/api_generated.go", "objectstorage/v1/service_generated.go", "objectstorage/v1/containers/api_generated.go", "objectstorage/v1/objects/api_generated.go", "objectstorage/v1/swauth/api_generated.go"} {
		got, err := os.ReadFile(filepath.Join(g.root, path))
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join("..", "..", "..", path))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("Swift signing workflow changed %s: %v", path, err)
		}
	}
	for _, name := range []string{"accounts", "containers", "objects"} {
		if _, err := os.Stat(filepath.Join(g.root, "objectstorage/v1", name, "resources_generated.go")); !os.IsNotExist(err) {
			t.Fatalf("%s unexpectedly gained a generated Collection: %v", name, err)
		}
	}
	docs, err := os.ReadFile(filepath.Join(g.root, "objectstorage/v1/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"GenerateFormSignature", "GenerateTempURL", "FormSignatureInput", "GenerateFormSignatureOpts", "GenerateTempURLOpts", "SHA1", "signing.md"} {
		if !strings.Contains(string(docs), text) {
			t.Fatalf("missing Swift signing policy: %s", text)
		}
	}
}
