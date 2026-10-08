package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceMetadefObjectsRegistryAndConcreteScope(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	var objects int
	for _, record := range sdkOwnedCollections {
		if record.Package == "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects" {
			objects++
			if record.Source != "sdk_owned" || record.Model != "Object" || record.Kind != "scoped_named_resource" || !record.Delete || record.Find || record.Wait || record.Scope != "InNamespace" || record.Parent != "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces" {
				t.Fatalf("invented object capability: %+v", record)
			}
			g.collections = append(g.collections, record)
		}
	}
	if objects != 1 {
		t.Fatalf("object records: %d", objects)
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
			t.Fatalf("object API changed native %s: %v", name, err)
		}
	}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	registry, err := os.ReadFile(filepath.Join(g.root, "image/v2/service_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects"`, "MetadefObjects ", "MetadefObjects:"} {
		if strings.Count(string(registry), want) != 1 {
			t.Fatalf("missing or duplicate object aggregate: %s", want)
		}
	}
	docs, err := os.ReadFile(filepath.Join(g.root, "image/v2/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"MetadefObjects", "metadefobjects/api.go", "metadefobjects/README.md", "InNamespace(ctx, namespace)", "Create/Get/Update/Delete(ctx, name, options...)", "DeleteAll/List/All(ctx, options...)", "HTTP 없는 고정 literal 범위", "유한 목록", "로컬 MaxItems", "DeleteAll은404 오류", "Resources·Find·상태 대기 없음"} {
		if !strings.Contains(string(docs), want) {
			t.Fatalf("missing actual object policy: %s", want)
		}
	}
	for _, invented := range []string{"MetadefObjects.Resources", "MetadefObjects.Find", "MetadefObjects.Wait", "MetadefObjects.ListLimit", "MetadefObjects.ListMarker"} {
		if strings.Contains(string(docs), invented) {
			t.Fatalf("invented object collection policy: %s", invented)
		}
	}
}
