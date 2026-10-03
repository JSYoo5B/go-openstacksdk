package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceMetadefNamespaceNestedCreatePreservesRegistry(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	for _, name := range []string{"imagedata", "imageimport", "images", "members", "tasks"} {
		if err := g.generate(upstreamModule + "/openstack/image/v2/" + name); err != nil {
			t.Fatal(err)
		}
	}
	g.collections = append(g.collections, sdkOwnedCollections...)
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"image/v2/service_generated.go"} {
		got, err := os.ReadFile(filepath.Join(g.root, name))
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join("..", "..", "..", name))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("nested Create changed %s: %v", name, err)
		}
	}
	docs, err := os.ReadFile(filepath.Join(g.root, "image/v2/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"Create의 Properties·Objects·Tags·ResourceTypeAssociations를 한 POST로 전달", "명시 PUT 교체", "Resources·Find·상태 대기 없음"} {
		if !strings.Contains(string(docs), value) {
			t.Fatalf("missing namespace Create contract: %s", value)
		}
	}
}
