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
		{SDKPackage: "gophercloudsdk/network/v2/extensions/security/groups"},
		{SDKPackage: "gophercloudsdk/network/v2/extensions/layer3/routers"},
		{SDKPackage: "gophercloudsdk/identity/v2/users"},
		{SDKPackage: "gophercloudsdk/identity/v3/users"},
		{SDKPackage: "gophercloudsdk/utils"},
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
	native := inventory{Version: "test-pin", Operations: []operation{{SDKPackage: "gophercloudsdk/sharedfilesystems/v2/shares", Name: "List"}}}
	g := generator{root: root, inventory: native, collections: []collectionRecord{
		{Package: "gophercloudsdk/sharedfilesystems/v2/quotasets", Source: "sdk_owned", Model: "QuotaResource", Kind: "singleton", Scope: "InProject"},
		{Package: "gophercloudsdk/sharedfilesystems/v2/shares", Source: "sdk_owned", Model: "Share"},
		{Package: "gophercloudsdk/accelerator/v2/devices", Source: "sdk_owned", Model: "Device"},
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
