package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestKeyManagerSDKOwnedRegistryAndActualCapabilities(t *testing.T) {
	want := []collectionRecord{
		{Package: "gophercloudsdk/keymanager/v1/secretstores", Source: "sdk_owned", Model: "SecretStore", Kind: "store_defaults"},
		{Package: "gophercloudsdk/keymanager/v1/quotas", Source: "sdk_owned", Model: "Quota", Kind: "effective_project_quota", Scope: "InProject", Parent: "gophercloudsdk/identity/v3/projects"},
		{Package: "gophercloudsdk/keymanager/v1/secretconsumers", Source: "sdk_owned", Model: "Consumer", Kind: "secret_consumer", Scope: "InSecret", Parent: "gophercloudsdk/keymanager/v1/secrets"},
	}
	var actual []collectionRecord
	for _, record := range sdkOwnedCollections {
		if strings.HasPrefix(record.Package, "gophercloudsdk/keymanager/") {
			actual = append(actual, record)
		}
	}
	if !reflect.DeepEqual(actual, want) {
		t.Fatalf("manual registry invented a native or common capability: %#v", actual)
	}
	root := t.TempDir()
	g := generator{root: root, inventory: inventory{Operations: []operation{{SDKPackage: "gophercloudsdk/keymanager/v1/secrets", Name: "List"}}}, collections: append([]collectionRecord{
		{Package: "gophercloudsdk/keymanager/v1/secrets", Model: "Secret", Find: true, Delete: true, Wait: true},
	}, actual...)}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	read := func(path string) string {
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	for _, path := range []string{"keymanager/v1/service_generated.go", "connection_services_generated.go"} {
		if _, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, path), nil, 0); err != nil {
			t.Fatal(err)
		}
	}
	registry := read("keymanager/v1/service_generated.go")
	for _, part := range []string{"SecretStores", "Quotas", "SecretConsumers", "Secrets", "gophercloudsdk/keymanager/v1/secretstores", "gophercloudsdk/keymanager/v1/quotas", "gophercloudsdk/keymanager/v1/secretconsumers", "New(client)"} {
		if !strings.Contains(registry, part) {
			t.Fatalf("missing shared-client registry %q: %s", part, registry)
		}
	}
	connection := read("connection_services_generated.go")
	for _, part := range []string{"KeyManagerV1", "KeyManager(ctx", "cachedService(ctx, c, KeyManager"} {
		if !strings.Contains(connection, part) {
			t.Fatalf("missing cached authenticated facade %q: %s", part, connection)
		}
	}
	docs := read("keymanager/v1/README.md")
	for _, part := range []string{"SecretStores.List/All", "GetGlobalDefault", "GetPreferred", "Quotas.Get", "Quotas.InProject", "secretstores/api.go", "quotas/api.go", "secretconsumers/api.go", "secretconsumers/README.md", "SecretConsumers.InSecret", "consumer ID·Resources·Find·Wait 없음", "advertised offset next", "교체 PUT204", "초기화 DELETE204", "seeded Resource", "Resources·CRUD·Find·Wait 없음", "인증된 프로젝트 effective quota", "service.Secrets.Resources.List(ctx)", "Containers.Fetch(ctx", "Orders.Fetch(ctx", "metadata-fetch.md", "Containers.Remove", "Orders.Remove", "Secrets.Remove", "metadata-delete.md", "WithListFilter", "WithListFilters", "Containers.CreateRecord", "Orders.CreateRecord", "Secrets.CreateRecord", "WithCreateRecordAttribute", "metadata-create.md"} {
		if !strings.Contains(docs, part) {
			t.Fatalf("missing documented actual capability %q: %s", part, docs)
		}
	}
	for _, invented := range []string{"SecretStores.Resources", "Quotas.Resources", "SecretConsumers.Resources", "SecretStores.Find(", "Quotas.Find(", "SecretConsumers.Find("} {
		if strings.Contains(docs, invented) {
			t.Fatalf("documentation invented an unsupported abstraction %q", invented)
		}
	}
}
