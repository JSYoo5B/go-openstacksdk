package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceMetadefResourceTypesRegistryAndConcreteScope(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	var tags int
	for _, record := range sdkOwnedCollections {
		if record.Package == "gophercloudsdk/image/v2/metadefresourcetypes" {
			tags++
			if record.Source != "sdk_owned" || record.Model != "ResourceType" || record.Kind != "catalog_and_association" || !record.Delete || record.Find || record.Wait || record.Scope != "InNamespace" || record.Parent != "gophercloudsdk/image/v2/metadefnamespaces" {
				t.Fatalf("invented resource type capability: %+v", record)
			}
			g.collections = append(g.collections, record)
		}
	}
	if tags != 1 {
		t.Fatalf("resource type records: %d", tags)
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
			t.Fatalf("resource type API changed native %s: %v", name, err)
		}
	}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	registry, err := os.ReadFile(filepath.Join(g.root, "image/v2/service_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"gophercloudsdk/image/v2/metadefresourcetypes"`, "MetadefResourceTypes ", "MetadefResourceTypes:"} {
		if strings.Count(string(registry), want) != 1 {
			t.Fatalf("missing or duplicate resource type aggregate: %s", want)
		}
	}
	docs, err := os.ReadFile(filepath.Join(g.root, "image/v2/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"MetadefResourceTypes", "metadefresourcetypes/api.go", "metadefresourcetypes/README.md", "전역 `List/All(ctx, options...)`", "InNamespace(ctx, namespace)", "Create/Delete(ctx, name, options...)", "두 유한 목록", "공통 concrete ListOpts", "로컬 MaxItems", "query/paging 없음", "Prefix·PropertiesTarget의 생략/빈 값 구별", "기본 clean404 무시", "연결만 해제", "단건 Get/Update·Find·상태 대기 없음"} {
		if !strings.Contains(string(docs), want) {
			t.Fatalf("missing actual resource type policy: %s", want)
		}
	}
	for _, invented := range []string{"MetadefResourceTypes.Resources", "MetadefResourceTypes.Find", "MetadefResourceTypes.Wait", "MetadefResourceTypes.ListLimit", "MetadefResourceTypes.ListMarker", "MetadefResourceTypes.Get", "MetadefResourceTypes.Update", "MetadefResourceTypes.DeleteAll"} {
		if strings.Contains(string(docs), invented) {
			t.Fatalf("invented resource type policy: %s", invented)
		}
	}
}
