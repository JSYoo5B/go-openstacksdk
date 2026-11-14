package main

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestKeyManagerSDKOwnedRegistryAndActualCapabilities(t *testing.T) {
	want := []collectionRecord{
		{Package: "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores", Source: "sdk_owned", Model: "SecretStore", Kind: "store_defaults"},
		{Package: "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas", Source: "sdk_owned", Model: "Quota", Kind: "effective_project_quota", Scope: "InProject", Parent: "github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"},
		{Package: "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretconsumers", Source: "sdk_owned", Model: "Consumer", Kind: "secret_consumer", Scope: "InSecret", Parent: "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"},
		{Package: "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretacls", Source: "sdk_owned", Model: "SecretACL", Kind: "secret_acl", Scope: "InSecret", Parent: "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"},
	}
	var actual []collectionRecord
	for _, record := range sdkOwnedCollections {
		if strings.HasPrefix(record.Package, "github.com/JSYoo5B/go-openstacksdk/keymanager/") {
			actual = append(actual, record)
		}
	}
	if !reflect.DeepEqual(actual, want) {
		t.Fatalf("manual registry invented a native or common capability: %#v", actual)
	}
	root := t.TempDir()
	g := generator{root: root, inventory: inventory{Operations: []operation{{SDKPackage: "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets", Name: "List"}}}, collections: append([]collectionRecord{
		{Package: "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets", Model: "Secret", Find: true, Delete: true, Wait: true},
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
	for _, part := range []string{"SecretStores", "Quotas", "SecretConsumers", "SecretACLs", "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretacls", "Secrets", "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores", "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas", "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretconsumers", "New(client)"} {
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
	for _, part := range []string{"GetGlobalDefault", "GetPreferred", "secretstores/api.go", "quotas/api.go", "secretconsumers/api.go", "secretacls/api.go", "PUT Set/Update", "absent attributes cause no HTTP", "no consumer ID/Resources/Find/Wait", "no Resources/CRUD/Find/Wait", "authenticated-project effective quota", "service.Secrets.Resources.List(ctx)", "go doc ./keymanager/v1/..."} {
		if !strings.Contains(docs, part) {
			t.Fatalf("missing documented actual capability %q: %s", part, docs)
		}
	}
	packageDocs := keyManagerPackageDocs(t)
	for _, part := range []string{"List and All, GetGlobalDefault and GetPreferred", "WithListFilter and WithListFilters", "InProject fixes a project ID", "replaces configuration with PUT204", "resets overrides with DELETE204", "seeded Resource", "InSecret fixes the parent secret", "advertised offset next links", "Secrets.Fetch, Containers.Fetch and Orders.Fetch", "Secrets.CreateRecord, Containers.CreateRecord and Orders.CreateRecord", "WithCreateRecordAttribute", "Secrets.Remove, Containers.Remove and Orders.Remove"} {
		if !strings.Contains(packageDocs, part) {
			t.Fatalf("missing package doc capability %q: %s", part, packageDocs)
		}
	}
	for _, invented := range []string{"SecretStores.Resources", "Quotas.Resources", "SecretConsumers.Resources", "SecretStores.Find(", "Quotas.Find(", "SecretConsumers.Find("} {
		if strings.Contains(docs, invented) {
			t.Fatalf("documentation invented an unsupported abstraction %q", invented)
		}
	}
}

func keyManagerPackageDocs(t *testing.T) string {
	t.Helper()
	var text []string
	err := filepath.WalkDir(filepath.Join("..", "..", "..", "keymanager", "v1"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "_generated.go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly|parser.ParseComments)
		if err != nil {
			return err
		}
		if file.Doc != nil {
			text = append(text, file.Doc.Text())
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(strings.Fields(strings.Join(text, " ")), " ")
}
