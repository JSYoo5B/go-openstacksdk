package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndependentSDKOwnedServiceRegistryAndCapabilityDocs(t *testing.T) {
	root := t.TempDir()
	g := generator{root: root, collections: []collectionRecord{
		{Package: "gophercloudsdk/clustering/v1/buildinfo", Source: "sdk_owned", Model: "BuildInfo", Kind: "service_info"},
		{Package: "gophercloudsdk/clustering/v1/profiletypes", Source: "sdk_owned", Model: "ProfileType", Find: true},
		{Package: "gophercloudsdk/clustering/v1/profiles", Source: "sdk_owned", Model: "Profile", Find: true, Delete: true},
		{Package: "gophercloudsdk/clustering/v1/policies", Source: "sdk_owned", Model: "Policy", Find: true, Delete: true},
		{Package: "gophercloudsdk/clustering/v1/clusters", Source: "sdk_owned", Model: "Cluster", Kind: "async_resource", Find: true, Wait: true},
		{Package: "gophercloudsdk/clustering/v1/nodes", Source: "sdk_owned", Model: "Node", Kind: "async_resource", Find: true, Wait: true},
		{Package: "gophercloudsdk/clustering/v1/services", Source: "sdk_owned", Model: "Service", Kind: "list_only"},
		{Package: "gophercloudsdk/instanceha/v1/segments", Source: "sdk_owned", Model: "Segment", Find: true, Delete: true},
	}}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"connection_services_generated.go", "clustering/v1/service_generated.go", "instanceha/v1/service_generated.go"} {
		if _, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, path), nil, 0); err != nil {
			t.Fatal(err)
		}
	}
	read := func(path string) string {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	connection := read("connection_services_generated.go")
	for _, method := range []string{"ClusteringV1", "Clustering(ctx", "InstanceHAV1", "InstanceHA(ctx"} {
		if !strings.Contains(connection, method) {
			t.Fatalf("missing Connection.%s", method)
		}
	}
	registry := read("clustering/v1/service_generated.go")
	if !strings.Contains(registry, "BuildInfo") || !strings.Contains(registry, "ProfileTypes") || !strings.Contains(registry, "Profiles") || !strings.Contains(registry, "Policies") || !strings.Contains(registry, "Clusters") || !strings.Contains(registry, "Nodes") {
		t.Fatalf("registry field names: %s", registry)
	}
	docs := read("clustering/v1/README.md")
	for _, part := range []string{"pinned Gophercloud에 없어", "서비스 build 정보 singleton", "목록만 제공", "service.ProfileTypes.Resources.List(ctx)", "service.ProfileTypes.Resources.Find(ctx", "profiles/api.go", "policies/api.go", "clusters/api.go", "nodes/api.go", "profiles/README.md", "policies/README.md", "clusters/README.md", "nodes/README.md", "객체 PATCH 갱신", "47개 직접 선언", "Collection.Delete는 미지원"} {
		if !strings.Contains(docs, part) {
			t.Fatalf("missing actual SDK capability %q: %s", part, docs)
		}
	}
	if strings.Contains(docs, "service.BuildInfo.Resources") || strings.Contains(docs, "service.ProfileTypes.Find(ctx") {
		t.Fatal("documentation invented Collection or API method")
	}
	if !strings.Contains(read("instanceha/v1/README.md"), "conn.instance_ha") {
		t.Fatal("Python service name missing")
	}
}
