package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceCacheManualMethodsPreserveNativeBindingsAndDocumentDefaults(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	if err := g.generate(upstreamModule + "/openstack/image/v2/images"); err != nil {
		t.Fatal(err)
	}
	names := []string{"GetImageCache", "QueueImage", "CacheDeleteImage", "ClearCache", "CachedImageNodes", "CleanCache", "PruneCache"}
	for _, path := range []string{"image/v2/images/api_generated.go", "image/v2/images/resources_generated.go"} {
		actual, err := os.ReadFile(filepath.Join(g.root, path))
		if err != nil {
			t.Fatal(err)
		}
		expected, err := os.ReadFile(filepath.Join("..", "..", "..", path))
		if err != nil || !bytes.Equal(actual, expected) {
			t.Fatalf("cache methods changed native API/binding: %s err=%v", path, err)
		}
		for _, name := range names {
			if strings.Contains(string(actual), "func (a *API) "+name+"(") {
				t.Fatalf("invented native cache method %s in %s", name, path)
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
	for _, text := range append(names, "../cache.md", "nil IgnoreMissing", "zero target", "DELETE404", "Clean은 실제200", "Prune은 실제200", "passive node URL", "Python 버전 cap") {
		if !strings.Contains(string(docs), text) {
			t.Fatalf("missing cache contract: %s", text)
		}
	}
}
