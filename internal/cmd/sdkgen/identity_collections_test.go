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

func identityCollectionFixture(t *testing.T, spec identityCollectionSpec, fields, idType, query string) (*types.Package, *collectionPlan) {
	t.Helper()
	getArgs, listArgs := "id "+idType, ",opts ListOptsBuilder"
	if spec.parents != 0 {
		getArgs = "parentID string," + getArgs
		listArgs = ",parentID string,opts ListOptsBuilder"
	}
	options := `type ListOpts struct{Name string ` + "`q:\"" + query + "\"`" + `}
type ListOptsBuilder interface{ToListQuery()(string,error)}
func(ListOpts)ToListQuery()(string,error){return "",nil}`
	source := controlledFixtureSource(spec.model, fields, spec.getter, getArgs, spec.lister, listArgs, options)
	path := spec.path
	if strings.HasPrefix(path, "network/") {
		path = "networking/" + strings.TrimPrefix(path, "network/")
	}
	pkg, decls := typedCollectionFixture(t, upstreamModule+"/openstack/"+path, source)
	plan := identifyNamedCollection(pkg, decls, extractorsByPage(pkg, decls), spec.getter, []string{spec.lister}, "Delete", spec.parents)
	if plan == nil {
		t.Fatal("missing fixture plan", spec)
	}
	return pkg, plan
}

func TestIdentityCollectionsOnlyEnableAuditedMemberRoutes(t *testing.T) {
	for _, spec := range identityCollectionSpecs {
		t.Run(spec.path+"/"+spec.model, func(t *testing.T) {
			pkg, plan := identityCollectionFixture(t, spec, "ID string;Name string", "string", "name")
			if !identityCollectionEnabled(pkg, plan, spec.parents) || identityCollectionEnabled(pkg, plan, spec.parents+1) {
				t.Fatal("audited parent arity was not preserved", spec, plan)
			}
			for _, drift := range []string{"numeric-model-id", "numeric-model-name", "missing-model-name", "numeric-request-id", "url-identity", "missing-name-query", "different-name-query", "wrong-model", "wrong-getter", "wrong-lister", "unrelated-package"} {
				t.Run(drift, func(t *testing.T) {
					altered := spec
					fields, idType, query := "ID string;Name string", "string", "name"
					switch drift {
					case "numeric-model-id":
						fields = "ID int;Name string"
					case "numeric-model-name":
						fields = "ID string;Name int"
					case "missing-model-name":
						fields = "ID string"
					case "numeric-request-id":
						idType = "int"
					case "url-identity":
						fields = "SecretRef string;Name string"
					case "missing-name-query":
						query = ""
					case "different-name-query":
						query = "display_name"
					case "wrong-model":
						altered.model = "Other"
					case "wrong-getter":
						altered.getter = "Fetch"
					case "wrong-lister":
						altered.lister = "ListOther"
					case "unrelated-package":
						altered.path = "identity/v3/users"
					}
					pkg, plan := identityCollectionFixture(t, altered, fields, idType, query)
					if identityCollectionEnabled(pkg, plan, altered.parents) {
						t.Fatal("unaudited identity route was enabled", drift, altered, plan)
					}
				})
			}
		})
	}
	// Pools itself has Get/List/Name/ID, but only its fixed-parent member route
	// is audited. Sharing a package cannot enable the global pool binding.
	pools := identityCollectionSpec{"loadbalancer/v2/pools", "Pool", "Get", "List", 0}
	pkg, plan := identityCollectionFixture(t, pools, "ID string;Name string", "string", "name")
	if identityCollectionEnabled(pkg, plan, 0) {
		t.Fatal("member opt-in leaked to pools")
	}
}

func identityAdapterEnabled(t *testing.T, source []byte) bool {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "generated.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		field, ok := node.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		key, ok := field.Key.(*ast.Ident)
		if !ok || key.Name != "IdentityFind" {
			return true
		}
		value, ok := field.Value.(*ast.Ident)
		if !ok || value.Name != "true" || found {
			t.Fatal("invalid/duplicate identity capability", field)
		}
		found = true
		return true
	})
	return found
}

func TestIdentityCollectionWrappersDelegateOwnedOptionsAndKeepScopedParents(t *testing.T) {
	for _, spec := range identityCollectionSpecs {
		t.Run(spec.path+"/"+spec.model, func(t *testing.T) {
			pkg, plan := identityCollectionFixture(t, spec, "ID string;Name string", "string", "name")
			root := t.TempDir()
			dir := filepath.Join(root, spec.path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			g := generator{root: root}
			output, target := "resources_generated.go", "a.Resources"
			if spec.parents == 0 {
				if err := g.emitCollection(pkg, plan); err != nil {
					t.Fatal(err)
				}
			} else {
				method, parent := "InZone", "dns/v2/zones"
				if spec.model == "Member" {
					method, parent = "Members", "loadbalancer/v2/pools"
				}
				scope := scopePlan{scopeSpec{path: spec.path, method: method, parent: parent, get: spec.getter, list: spec.lister, id: "ID"}, plan}
				if err := g.emitScopes(pkg, []scopePlan{scope}, []byte("package fixture\ntype API struct{}\n")); err != nil {
					t.Fatal(err)
				}
				output, target = "scopes_generated.go", "s.Collection"
			}
			emitted, err := os.ReadFile(filepath.Join(dir, output))
			if err != nil {
				t.Fatal(err)
			}
			if !identityAdapterEnabled(t, emitted) {
				t.Fatal("pilot adapter not opted in", string(emitted))
			}
			method := controlledEmittedMethod(t, emitted, "FindIdentity")
			if len(method.Type.Params.List) != 3 || nodeText(method.Type.Params.List[1].Type) != "string" || nodeText(method.Type.Params.List[2].Type) != "...resource.IdentityFindOption" || len(method.Body.List) != 1 {
				t.Fatal("lookup signature gained builders or duplicated logic", string(emitted))
			}
			requireControlledCalls(t, method, target+".FindIdentity(ctx, identity, options...)")
			adapter := controlledEmittedMethod(t, emitted, "newResources")
			if !strings.Contains(string(emitted), "config.Query[key] = append([]string(nil), values...)") || strings.Contains(string(emitted), "With"+spec.lister+"Query(key, value)") {
				t.Fatal("native adapter collapses repeated or nil query values", string(emitted))
			}
			if spec.parents != 0 {
				requireControlledCalls(t, adapter, "s.api."+spec.getter+"(ctx, s.parentID, string(id))", "s.api."+controlledListName(spec.lister)+"(ctx, s.parentID, control, options...)")
			} else {
				requireControlledCalls(t, adapter, "a.Get(ctx, string(id))", "a.listWithControl(ctx, control, options...)")
			}
			if spec.path == "compute/v2/servers" && !strings.Contains(string(emitted), "regexp.QuoteMeta(name)") {
				t.Fatal("literal Nova name query policy was lost")
			}
		})
	}
}

func TestIdentityCollectionsDoNotEmitCapabilitiesForUnrelatedBindings(t *testing.T) {
	for _, spec := range []identityCollectionSpec{
		{"identity/v3/users", "User", "Get", "List", 0},
		{"loadbalancer/v2/pools", "Pool", "Get", "List", 0},
		{"compute/v2/attachinterfaces", "Interface", "Get", "List", 1},
	} {
		t.Run(spec.path, func(t *testing.T) {
			pkg, plan := identityCollectionFixture(t, spec, "ID string;Name string", "string", "name")
			e := emitter{pkg: pkg, imports: map[string]string{}}
			e.printf("func(a *API)newResources()*resource.Collection[%s]{return ", plan.modelName)
			parents := []string(nil)
			if spec.parents != 0 {
				parents = []string{"a.parentID"}
			}
			emitCollectionAdapter(&e, plan, "a", parents)
			e.printf("}\n")
			emitted, err := e.source()
			if err != nil || identityAdapterEnabled(t, emitted) || strings.Contains(string(emitted), "IdentityFindOption") {
				t.Fatal("unsupported binding gained identity behavior", err, string(emitted))
			}
		})
	}
}
