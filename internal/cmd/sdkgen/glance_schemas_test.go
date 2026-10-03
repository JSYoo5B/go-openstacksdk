package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceSchemaDiscoveryPreservesNativeImageBindingsAndDocumentsManualGetters(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	if err := g.generate(upstreamModule + "/openstack/image/v2/images"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"image/v2/images/api_generated.go", "image/v2/images/resources_generated.go"} {
		actual, err := os.ReadFile(filepath.Join(g.root, path))
		if err != nil {
			t.Fatal(err)
		}
		expected, err := os.ReadFile(filepath.Join("..", "..", "..", path))
		if err != nil || !bytes.Equal(actual, expected) {
			t.Fatalf("schema discovery changed native API/binding: %s err=%v", path, err)
		}
		for _, name := range []string{"GetImageSchema", "GetImagesSchema", "GetMetadefPropertiesSchema"} {
			if strings.Contains(string(actual), "func (a *API) "+name+"(") {
				t.Fatalf("invented native schema getter %s in %s", name, path)
			}
		}
	}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	docs, err := os.ReadFile(filepath.Join(g.root, "image/v2/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"image.Service.Get*Schema", "16개 메서드", "GetSchemaOption", "image.Schema", "additionalProperties", "../schemas.md", "실제200", "raw/minimal", "중첩 collection 구조"} {
		if !strings.Contains(string(docs), text) {
			t.Fatalf("missing schema discovery policy: %s", text)
		}
	}
}
