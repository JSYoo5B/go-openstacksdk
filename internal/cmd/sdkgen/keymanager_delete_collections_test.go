package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Reuse the existing model fixtures and their inspected DeleteResult signatures.
func keyManagerDeleteFixture(t *testing.T, kind string) (*types.Package, map[string]*ast.FuncDecl, *collectionPlan) {
	t.Helper()
	source := orderIdentityFixtureSource
	switch kind {
	case "containers":
		source = containerFilterNativeFixtureSource
	case "secrets":
		source = secretFilterNativeFixtureSource
	}
	pkg, declarations := typedCollectionFixture(t, upstreamModule+"/openstack/keymanager/v1/"+kind, source)
	plan, err := identifyCollectionBinding(pkg, declarations, extractorsByPage(pkg, declarations))
	if err != nil || plan == nil {
		t.Fatal(plan, err)
	}
	return pkg, declarations, plan
}

func keyManagerDeleteDeclarations(t *testing.T, kind string) map[string]*ast.FuncDecl {
	t.Helper()
	source := `package fixture
func Delete(ctx context.Context, client *gophercloud.ServiceClient, id string) (r DeleteResult) {
    resp, err := client.Delete(ctx, deleteURL(client, id), nil)
    _, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
    return
}
func deleteURL(client *gophercloud.ServiceClient, id string) string {
    return client.ServiceURL("` + kind + `", id)
}`
	file, err := parser.ParseFile(token.NewFileSet(), "native_delete.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	declarations := make(map[string]*ast.FuncDecl)
	for _, declaration := range file.Decls {
		if fn, ok := declaration.(*ast.FuncDecl); ok {
			declarations[identityDeclarationKey(fn)] = fn
		}
	}
	return declarations
}

func TestKeyManagerDeleteCollectionsUseOnlyInspectedOwnedHooks(t *testing.T) {
	nativeHashes := map[string]string{
		"containers": "319a58d3024b40360683c34886debfee582dbf25127acdd686b2bdbf625fbf86",
		"orders":     "a4e2bdc01bfbf0b0f5bd89d834a29187652f05424d8a05218ec8c1c72744f1b1",
		"secrets":    "82fa04d28c09362dc998eeb6a171608ad10e87200c5bfc6e1eeff9a45729b319",
	}
	for _, kind := range []string{"containers", "orders", "secrets"} {
		t.Run(kind, func(t *testing.T) {
			pkg, _, plan := keyManagerDeleteFixture(t, kind)
			if !keyManagerOwnedDeleteCollection(pkg, plan, 0) || keyManagerOwnedDeleteCollection(pkg, plan, 1) || keyManagerOwnedDeleteCollection(types.NewPackage("example/keymanager/v1/"+kind, kind), plan, 0) {
				t.Fatal("owned delete leaked from its exact unscoped package")
			}
			e := emitter{pkg: pkg, imports: map[string]string{}}
			emitCollectionAdapter(&e, plan, "a", nil)
			source := e.body.String()
			for _, want := range []string{"a.deleteOwned(ctx,id)", "a.Get(ctx,string(id))", "a.listWithControl(ctx,control,options...)"} {
				if !strings.Contains(source, want) {
					t.Fatal("missing selected binding", want, source)
				}
			}
			if strings.Contains(source, "a.Delete(ctx,string(id))") || (kind == "orders" && strings.Contains(source, "Name:")) || (kind != "orders" && !strings.Contains(source, "NameQuery:func(name string)string{return name}")) {
				t.Fatal("native mutation or name policy changed", source)
			}
			g := generator{root: t.TempDir()}
			dir := filepath.Join(g.root, "keymanager/v1", kind)
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			if err := g.emitCollection(pkg, plan); err != nil {
				t.Fatal(err)
			}
			generated, err := os.ReadFile(filepath.Join(dir, "resources_generated.go"))
			if err != nil || !strings.Contains(string(generated), "return a.removeOwned(ctx, ref, options...)") {
				t.Fatal(string(generated), err)
			}
			// Preserve the public native method separately from collection policy.
			native, err := os.ReadFile(filepath.Join("../../..", "keymanager/v1", kind, "api_generated.go"))
			if err != nil || emittedFunctionHash(t, native, "API.Delete") != nativeHashes[kind] {
				t.Fatal("native Delete compatibility changed", err)
			}
		})
	}
}

func TestKeyManagerDeleteCollectionsRejectPinnedDeleteSourceAndSignatureDrift(t *testing.T) {
	for _, kind := range []string{"containers", "orders", "secrets"} {
		t.Run(kind, func(t *testing.T) {
			pkg, _, plan := keyManagerDeleteFixture(t, kind)
			original := keyManagerDeleteDeclarations(t, kind)
			if err := validateKeyManagerDeleteDeclarations(pkg, original, plan); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"Delete", "deleteURL"} {
				t.Run(name, func(t *testing.T) {
					declarations := keyManagerDeleteDeclarations(t, kind)
					declarations[name].Body.List = nil
					if err := validateKeyManagerDeleteDeclarations(pkg, declarations, plan); err == nil || !strings.Contains(err.Error(), name) {
						t.Fatal("native body drift did not block generation", err)
					}
				})
			}
			for _, name := range []string{"missing delete", "scoped ID", "missing context", "numeric ID", "wrong result"} {
				t.Run(name, func(t *testing.T) {
					copy := *plan
					if name == "missing delete" {
						copy.deleter = nil
					} else {
						source := `package fixture
import "context"
import gophercloud "github.com/gophercloud/gophercloud/v2"
var _ context.Context
var _ *gophercloud.ServiceClient
type DeleteResult struct{}
func(DeleteResult)ExtractErr()error{return nil}
type WrongResult struct{}
func(WrongResult)ExtractErr()error{return nil}
func Delete(ctx context.Context,client *gophercloud.ServiceClient,id string)DeleteResult{return DeleteResult{}}
`
						switch name {
						case "scoped ID":
							source = strings.Replace(source, "id string)", "parentID,id string)", 1)
						case "missing context":
							source = strings.Replace(source, "ctx context.Context,", "", 1)
						case "numeric ID":
							source = strings.Replace(source, "id string)", "id int)", 1)
						case "wrong result":
							source = strings.Replace(source, ")DeleteResult{return DeleteResult{}}", ")WrongResult{return WrongResult{}}", 1)
						}
						changed, _ := typedCollectionFixture(t, pkg.Path(), source)
						copy.deleter = changed.Scope().Lookup("Delete").(*types.Func)
					}
					if err := validateKeyManagerDeleteDeclarations(pkg, original, &copy); err == nil {
						t.Fatal("native signature drift did not block generation", name)
					}
				})
			}
		})
	}
}
