package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceMetadefNamespacesRegistryAndConcretePolicy(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	var namespaces int
	for _, record := range sdkOwnedCollections {
		if record.Package == "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces" {
			namespaces++
			if record.Source != "sdk_owned" || record.Model != "Namespace" || record.Kind != "named_resource" || !record.Delete || record.Find || record.Wait || record.Scope != "" || record.Parent != "" {
				t.Fatalf("invented namespace capability: %+v", record)
			}
			g.collections = append(g.collections, record)
		}
	}
	if namespaces != 1 {
		t.Fatalf("namespace records: %d", namespaces)
	}
	for _, name := range []string{"images", "members", "imageimport"} {
		if err := g.generate(upstreamModule + "/openstack/image/v2/" + name); err != nil {
			t.Fatal(err)
		}
		path := "image/v2/" + name + "/api_generated.go"
		actual, err := os.ReadFile(filepath.Join(g.root, path))
		if err != nil {
			t.Fatal(err)
		}
		expected, err := os.ReadFile(filepath.Join("..", "..", "..", path))
		if err != nil || !bytes.Equal(actual, expected) {
			t.Fatalf("namespace API changed native %s: %v", name, err)
		}
	}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	registry, err := os.ReadFile(filepath.Join(g.root, "image/v2/service_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"`, "MetadefNamespaces ", "MetadefNamespaces:"} {
		if strings.Count(string(registry), want) != 1 {
			t.Fatalf("missing or duplicate namespace aggregate: %s", want)
		}
	}
	docs, err := os.ReadFile(filepath.Join(g.root, "image/v2/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"MetadefNamespaces", "metadefnamespaces/api.go", "metadefnamespaces/README.md", "Create/Get/Update/Delete(ctx, namespace, options...)", "List/All(ctx, options...)", "concrete 기본값", "명시 PUT 교체", "Resources·Find·상태 대기 없음"} {
		if !strings.Contains(string(docs), want) {
			t.Fatalf("missing actual namespace policy: %s", want)
		}
	}
	if strings.Contains(string(docs), "MetadefNamespaces.Resources") || strings.Contains(string(docs), "MetadefNamespaces.Find") || strings.Contains(string(docs), "MetadefNamespaces.Wait") {
		t.Fatal("invented generic namespace collection")
	}
}
