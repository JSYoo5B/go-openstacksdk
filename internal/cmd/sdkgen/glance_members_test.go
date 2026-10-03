package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	if err := g.generate(upstreamModule + "/openstack/image/v2/members"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"image/v2/members/api_generated.go", "image/v2/members/scopes_generated.go"} {
		actual, err := os.ReadFile(filepath.Join(g.root, path))
		if err != nil {
			t.Fatal(err)
		}
		expected, err := os.ReadFile(filepath.Join("..", "..", "..", path))
		if err != nil || !bytes.Equal(actual, expected) {
			t.Fatalf("upper member workflows changed native API/scope: %s err=%v", path, err)
		}
	}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	docs, err := os.ReadFile(filepath.Join(g.root, "image/v2/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"AddImageMember", "GetImageMember", "UpdateImageMember", "RemoveImageMember", "FindImageMember", "ListImageMembers", "AllImageMembers", "../members.md", "nil IgnoreMissing", "GET404", "DELETE404", "lazy finite", "MaxItems", "timestamp 문자열", "Members.InImage"} {
		if !strings.Contains(string(docs), text) {
			t.Fatalf("missing member contract: %s", text)
		}
	}
}
