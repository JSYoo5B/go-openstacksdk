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

func TestServiceRegistriesSeparateVersionsAndBindReadableResources(t *testing.T) {
	root := t.TempDir()
	g := generator{root: root, inventory: inventory{Operations: []operation{
		{SDKPackage: "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"},
		{SDKPackage: "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"},
		{SDKPackage: "github.com/JSYoo5B/go-openstacksdk/identity/v2/users"},
		{SDKPackage: "github.com/JSYoo5B/go-openstacksdk/identity/v3/users"},
		{SDKPackage: "github.com/JSYoo5B/go-openstacksdk/utils"},
	}}}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"connection_services_generated.go", "network/v2/service_generated.go", "identity/v2/service_generated.go", "identity/v3/service_generated.go"} {
		if _, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, path), nil, 0); err != nil {
			t.Fatal(err)
		}
	}
	connection, err := os.ReadFile(filepath.Join(root, "connection_services_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{"IdentityV2", "IdentityV3", "Identity(ctx", "NetworkV2"} {
		if !strings.Contains(string(connection), method) {
			t.Fatalf("missing %s", method)
		}
	}
	registry, err := os.ReadFile(filepath.Join(root, "network/v2/service_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(registry), "SecurityGroups") || !strings.Contains(string(registry), "Routers") {
		t.Fatalf("registry=%s", registry)
	}
}

func TestSDKOwnedAPIsExtendRegisteredServiceWithoutNativeDeclarations(t *testing.T) {
	root := t.TempDir()
	native := inventory{Version: "test-pin", Operations: []operation{{SDKPackage: "github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shares", Name: "List"}}}
	g := generator{root: root, inventory: native, collections: []collectionRecord{
		{Package: "github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/quotasets", Source: "sdk_owned", Model: "QuotaResource", Kind: "singleton", Scope: "InProject"},
		{Package: "github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shares", Source: "sdk_owned", Model: "Share"},
		{Package: "github.com/JSYoo5B/go-openstacksdk/accelerator/v2/devices", Source: "sdk_owned", Model: "Device"},
	}}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.inventory, native) {
		t.Fatalf("native inventory changed: %+v", g.inventory)
	}
	registry, err := os.ReadFile(filepath.Join(root, "sharedfilesystems/v2/service_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(registry), "QuotaSets *resource") != 1 || strings.Count(string(registry), "Shares    *resource") != 1 {
		t.Fatalf("manual quota API missing or native resource duplicated: %s", registry)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "registry.go", registry, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "accelerator/v2/service_generated.go")); !os.IsNotExist(err) {
		t.Fatalf("manual service registry was invented: %v", err)
	}
}

func TestNovaImageProxyExtendsComputeRegistryWithoutNativeInventory(t *testing.T) {
	root := t.TempDir()
	native := inventory{Version: "test-pin", Operations: []operation{{SDKPackage: "github.com/JSYoo5B/go-openstacksdk/compute/v2/servers", Name: "Get"}}}
	g := generator{root: root, inventory: native}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.inventory, native) {
		t.Fatal("owned proxy changed native inventory")
	}
	registry, err := os.ReadFile(filepath.Join(root, "compute/v2/service_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "registry.go", registry, 0); err != nil {
		t.Fatal(err)
	}
	// Field alignment changes as SDK-owned resources are added. Assert the
	// declarations and shared-client constructors independently of spacing.
	contents := strings.Join(strings.Fields(string(registry)), " ")
	for _, want := range []string{"github.com/JSYoo5B/go-openstacksdk/compute/v2/images", "Images *resource0.API", "Images: resource0.New(client)", "Servers *resource2.API", "Servers: resource2.New(client)", "github.com/JSYoo5B/go-openstacksdk/compute/v2/shareattachments", "ShareAttachments *resource3.API", "ShareAttachments: resource3.New(client)", "ServerExternalEvents *resource1.API", "ServerExternalEvents: resource1.New(client)"} {
		if !strings.Contains(contents, want) {
			t.Fatalf("missing %q: %s", want, registry)
		}
	}
	if len(g.collections) != 0 {
		t.Fatal("owned proxy invented a Resource collection")
	}
}

func TestCinderOwnedManagementRegistriesPreserveNativeInventory(t *testing.T) {
	root := t.TempDir()
	native := inventory{Version: "test-pin", Operations: []operation{{SDKPackage: "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes", Name: "Get"}}}
	g := generator{root: root, inventory: native}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.inventory, native) {
		t.Fatal("owned registry changed native inventory")
	}
	source, err := os.ReadFile(filepath.Join(root, "blockstorage/v3/service_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), "registry.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Capabilities", "ManageableSnapshots"} {
		if parsed.Scope.Lookup(name) != nil {
			t.Fatal("resource should be a service member")
		}
		if !strings.Contains(string(source), name+":") || !strings.Contains(string(source), "/"+strings.ToLower(name)+"\"") {
			t.Fatal(name, string(source))
		}
	}
}

func TestOwnedPortBindingsRegistryUsesExistingGuide(t *testing.T) {
	root := t.TempDir()
	guide := filepath.Join(root, "network/v2/extensions/portbindings/README.md")
	if err := os.MkdirAll(filepath.Dir(guide), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(guide, []byte("# Port bindings\n"), 0644); err != nil {
		t.Fatal(err)
	}
	native := inventory{Version: "test-pin", Operations: []operation{{SDKPackage: "github.com/JSYoo5B/go-openstacksdk/network/v2/ports", Name: "Get"}}}
	g := generator{root: root, inventory: native}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.inventory, native) {
		t.Fatal("owned facade changed native declarations")
	}
	registry, err := os.ReadFile(filepath.Join(root, "network/v2/service_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(registry), "PortBindings") || !strings.Contains(string(registry), "extensions/portbindings") {
		t.Fatal(string(registry))
	}
	docs, err := os.ReadFile(filepath.Join(root, "network/v2/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(docs), "extensions/portbindings/README.md") {
		t.Fatal(string(docs))
	}
}
