package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceImageUploadPreservesNativeAndRegistry(t *testing.T) {
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
	for _, path := range []string{"image/v2/imagedata/api_generated.go", "image/v2/service_generated.go", "image/v2/images/api_generated.go", "image/v2/images/resources_generated.go", "image/v2/tasks/api_generated.go", "image/v2/tasks/resources_generated.go"} {
		got, err := os.ReadFile(filepath.Join(g.root, path))
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join("..", "..", "..", path))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("image upload facade changed %s: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(g.root, "image/v2/imagedata/resources_generated.go")); !os.IsNotExist(err) {
		t.Fatalf("imagedata unexpectedly gained a generated resource binding: %v", err)
	}
	docs, err := os.ReadFile(filepath.Join(g.root, "image/v2/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"image.Service.UploadImage", "ImageUploadOpts", "POST201", "PUT204", "../upload-image.md"} {
		if !strings.Contains(string(docs), text) {
			t.Fatalf("missing image upload facade policy: %s", text)
		}
	}
}
