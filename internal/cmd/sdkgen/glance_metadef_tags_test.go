package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceMetadefTagsRegistryAndConcreteScope(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	var tags int
	for _, record := range sdkOwnedCollections {
		if record.Package == "gophercloudsdk/image/v2/metadeftags" {
			tags++
			if record.Source != "sdk_owned" || record.Model != "Tag" || record.Kind != "scoped_tag_resource" || !record.Delete || record.Find || record.Wait || record.Scope != "InNamespace" || record.Parent != "gophercloudsdk/image/v2/metadefnamespaces" {
				t.Fatalf("invented tag capability: %+v", record)
			}
			g.collections = append(g.collections, record)
		}
	}
	if tags != 1 {
		t.Fatalf("tag records: %d", tags)
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
			t.Fatalf("tag API changed native %s: %v", name, err)
		}
	}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	registry, err := os.ReadFile(filepath.Join(g.root, "image/v2/service_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"gophercloudsdk/image/v2/metadeftags"`, "MetadefTags ", "MetadefTags:"} {
		if strings.Count(string(registry), want) != 1 {
			t.Fatalf("missing or duplicate tag aggregate: %s", want)
		}
	}
	docs, err := os.ReadFile(filepath.Join(g.root, "image/v2/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"MetadefTags", "metadeftags/api.go", "metadeftags/README.md", "InNamespace(ctx, namespace)", "body 없는 `Create`", "Get/Update/Delete(ctx, name, options...)", "Set(ctx, names, options...)", "DeleteAll/List/All(ctx, options...)", "기본 append=false", "빈 입력은 DB를 지우지 않음", "실제 limit/marker/sort query", "양수 limit의 raw 이름 continuation", "로컬 MaxItems", "두 Delete 모두404 오류", "Resources·Find·상태 대기 없음"} {
		if !strings.Contains(string(docs), want) {
			t.Fatalf("missing actual tag policy: %s", want)
		}
	}
	for _, invented := range []string{"MetadefTags.Resources", "MetadefTags.Find", "MetadefTags.Wait", "MetadefTags.ListLimit", "MetadefTags.ListMarker"} {
		if strings.Contains(string(docs), invented) {
			t.Fatalf("invented tag collection policy: %s", invented)
		}
	}
}
