package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceImageAssociatedTasksPreservesNativeAndRegistry(t *testing.T) {
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
	for _, path := range []string{"image/v2/service_generated.go", "image/v2/images/api_generated.go", "image/v2/images/resources_generated.go", "image/v2/tasks/api_generated.go", "image/v2/tasks/resources_generated.go"} {
		got, err := os.ReadFile(filepath.Join(g.root, path))
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join("..", "..", "..", path))
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("associated Task facade changed %s: %v", path, err)
		}
	}
	docs, err := os.ReadFile(filepath.Join(g.root, "image/v2/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"image.Service.ImageTasks/AllImageTasks", "GET 한 번", "nullable deleted·deleted_at", "../image-tasks.md"} {
		if !strings.Contains(string(docs), text) {
			t.Fatalf("missing associated Task facade policy: %s", text)
		}
	}
}
