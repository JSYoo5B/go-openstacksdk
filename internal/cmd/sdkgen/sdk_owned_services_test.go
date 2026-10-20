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
		{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/buildinfo", Source: "sdk_owned", Model: "BuildInfo", Kind: "service_info"},
		{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes", Source: "sdk_owned", Model: "ProfileType", Find: true},
		{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles", Source: "sdk_owned", Model: "Profile", Find: true, Delete: true},
		{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies", Source: "sdk_owned", Model: "Policy", Find: true, Delete: true},
		{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters", Source: "sdk_owned", Model: "Cluster", Kind: "async_resource", Find: true, Wait: true},
		{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes", Source: "sdk_owned", Model: "Node", Kind: "async_resource", Find: true, Wait: true},
		{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers", Source: "sdk_owned", Model: "Receiver", Find: true, Delete: true},
		{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/services", Source: "sdk_owned", Model: "Service", Kind: "list_only"},
		{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies", Source: "sdk_owned", Model: "ClusterPolicy", Kind: "policy_binding", Find: true, Scope: "InCluster", Parent: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"},
		{Package: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterattributes", Source: "sdk_owned", Model: "ClusterAttribute", Kind: "scoped_list_only", Scope: "InCluster", Parent: "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"},
		{Package: "github.com/JSYoo5B/go-openstacksdk/instanceha/v1/segments", Source: "sdk_owned", Model: "Segment", Find: true, Delete: true},
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
	if !strings.Contains(registry, "BuildInfo") || !strings.Contains(registry, "ProfileTypes") || !strings.Contains(registry, "Profiles") || !strings.Contains(registry, "Policies") || !strings.Contains(registry, "Clusters") || !strings.Contains(registry, "Nodes") || !strings.Contains(registry, "Receivers") || !strings.Contains(registry, "ClusterPolicies") || !strings.Contains(registry, "ClusterAttributes") {
		t.Fatalf("registry field names: %s", registry)
	}
	docs := read("clustering/v1/README.md")
	for _, part := range []string{"pinned Gophercloud does not provide it", "service build-info singleton", "list only", "service.ProfileTypes.Resources.List(ctx)", "service.ProfileTypes.Resources.Find(ctx", "profiles/api.go", "policies/api.go", "clusters/api.go", "nodes/api.go", "receivers/api.go", "profiles/README.md", "policies/README.md", "clusters/README.md", "nodes/README.md", "receivers/README.md", "object PATCH update", "69 directly declared operations", "Collection.Delete is unsupported", "policy attach/detach/update", "wire `user`", "waiting/README.md", "unlimited duration, ERROR failure, and a two-second interval", "Common Resources.Wait has a separate five-minute policy", "clusterpolicies/api.go", "clusterattributes/api.go", "separate binding IDs", "fixed-cluster/JSONPath", "tracking/README.md", "clusters/metadata/README.md", "Previous/Requested", "accepted Operation"} {
		if !strings.Contains(docs, part) {
			t.Fatalf("missing actual SDK capability %q: %s", part, docs)
		}
	}
	if strings.Contains(docs, "service.BuildInfo.Resources") || strings.Contains(docs, "service.ProfileTypes.Find(ctx") {
		t.Fatal("documentation invented Collection or API method")
	}
	for _, part := range []string{"listing/README.md", "WithListMaxItems", "WithListPaginated", "resource.WithMaxItems", "resource.WithPaginated", "raw rows before local filtering", "ClusterPolicies apply the cap locally", "deployment behavior determines support for type catalog and Services hints", "do not establish full list parity"} {
		if !strings.Contains(docs, part) {
			t.Fatalf("missing bounded Senlin list control contract %q: %s", part, docs)
		}
	}
	if !strings.Contains(read("instanceha/v1/README.md"), "conn.instance_ha") {
		t.Fatal("Python service name missing")
	}
	for _, part := range []string{"waiting/README.md", "17 direct declarations", "WaitForStatus/WaitForDelete", "120-second deletion waits", "fixed parents"} {
		if !strings.Contains(read("instanceha/v1/README.md"), part) {
			t.Fatalf("missing Masakari wait capability %q", part)
		}
	}
}
