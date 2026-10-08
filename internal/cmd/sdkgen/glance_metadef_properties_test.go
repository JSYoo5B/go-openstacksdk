package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceMetadefPropertiesRegistryAndConcreteScope(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	var properties int
	for _, record := range sdkOwnedCollections {
		if record.Package == "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties" {
			properties++
			if record.Source != "sdk_owned" || record.Model != "Property" || record.Kind != "scoped_definition" || !record.Delete || record.Find || record.Wait || record.Scope != "InNamespace" || record.Parent != "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces" {
				t.Fatalf("invented property capability: %+v", record)
			}
			g.collections = append(g.collections, record)
		}
	}
	if properties != 1 {
		t.Fatalf("property records: %d", properties)
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
			t.Fatalf("property API changed native %s: %v", name, err)
		}
	}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	registry, err := os.ReadFile(filepath.Join(g.root, "image/v2/service_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties"`, "MetadefProperties ", "MetadefProperties:"} {
		if strings.Count(string(registry), want) != 1 {
			t.Fatalf("missing or duplicate property aggregate: %s", want)
		}
	}
	docs, err := os.ReadFile(filepath.Join(g.root, "image/v2/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"MetadefProperties", "metadefproperties/api.go", "metadefproperties/README.md", "InNamespace(ctx, namespace)", "Create/Get/Update/Delete(ctx, name, options...)", "DeleteAll/List/All(ctx, options...)", "HTTP 없는 고정 literal 범위", "Type·Title 필수 flat JSONSchema", "유한 dictionary 목록", "독립 Key", "선택 resource_type", "로컬 MaxItems", "DeleteAll은404 오류", "Resources·Find·상태 대기 없음"} {
		if !strings.Contains(string(docs), want) {
			t.Fatalf("missing actual property policy: %s", want)
		}
	}
	for _, invented := range []string{"MetadefProperties.Resources", "MetadefProperties.Find", "MetadefProperties.Wait", "MetadefProperties.ListLimit", "MetadefProperties.ListMarker"} {
		if strings.Contains(string(docs), invented) {
			t.Fatalf("invented property collection policy: %s", invented)
		}
	}
}
