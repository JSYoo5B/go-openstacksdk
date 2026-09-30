package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
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
