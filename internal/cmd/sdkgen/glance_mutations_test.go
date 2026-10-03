package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceMutationsPreserveNativeImagesAndDocumentUpperActions(t *testing.T) {
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
			t.Fatalf("manual mutations changed native API/binding: %s err=%v", path, err)
		}
		for _, name := range []string{"AddTag", "RemoveTag", "DeactivateImage", "ReactivateImage"} {
			if strings.Contains(string(actual), "func (a *API) "+name+"(") {
				t.Fatalf("invented dedicated native method %s in %s", name, path)
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
	for _, text := range []string{"image.Service.AddTag/RemoveTag", "DeactivateImage/ReactivateImage", "../mutations.md", "미존재는 오류", "실제204 acknowledgement", "ReplaceImageTags"} {
		if !strings.Contains(string(docs), text) {
			t.Fatalf("missing upper mutation policy: %s", text)
		}
	}
}
