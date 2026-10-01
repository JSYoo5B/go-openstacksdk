package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

func emittedBodyFilterAdapter(t *testing.T, spec identityCollectionSpec) ([]byte, map[string]ast.Expr) {
	t.Helper()
	pkg, plan := identityQueryFixture(t, spec, identityQueryFixtureSource(spec))
	e := emitter{pkg: pkg, imports: map[string]string{}}
	e.printf("func(a *API)newResources()*resource.Collection[%s]{return ", spec.model)
	parents := []string(nil)
	if spec.parents != 0 {
		parents = []string{"fixedParent"}
	}
	emitCollectionAdapter(&e, plan, "a", parents)
	e.printf("}\n")
	source, err := e.source()
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "emitted.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]ast.Expr{}
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		index, ok := literal.Type.(*ast.IndexExpr)
		if !ok {
			return true
		}
		selector, ok := index.X.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Adapter" {
			return true
		}
		for _, entry := range literal.Elts {
			pair, ok := entry.(*ast.KeyValueExpr)
			if !ok {
				t.Fatal("non-keyed adapter field")
			}
			fields[pair.Key.(*ast.Ident).Name] = pair.Value
		}
		return true
	})
	return source, fields
}

func TestBodyFilterCollectionGateRejectsNativeShapeAndUnauditedFields(t *testing.T) {
	if len(bodyFilterCollectionSpecs) != 2 || len(identityCollectionSpecs) != 20 {
		t.Fatal("body-filter or identity inventory broadened")
	}
	enabled := 0
	for _, identity := range identityCollectionSpecs {
		pkg, plan := identityQueryFixture(t, identity, identityQueryFixtureSource(identity))
		spec, ok := bodyFilterCollectionContract(pkg, plan, identity.parents)
		wanted := identity.path == "network/v2/extensions/qos/policies" || identity.path == "network/v2/extensions/security/addressgroups"
		if ok != wanted {
			t.Fatal("body filter capability leaked or disappeared", identity.path, ok)
		}
		if !ok {
			continue
		}
		enabled++
		if _, ok := bodyFilterCollectionContract(pkg, plan, 1); ok {
			t.Fatal("global descriptor leaked into parent scope")
		}
		if err := validateBodyFilterCollectionContracts(pkg, plan); err != nil {
			t.Fatal(err)
		}
		base := identityQueryFixtureSource(identity)
		field := spec.fields[0]
		mutations := map[string]string{
			"wire field tag":         strings.Replace(base, `json:"`+field.key+`"`, `json:"different_body_key"`, 1),
			"missing selected field": strings.Replace(base, field.member+" ", "OtherField ", 1),
		}
		if identity.model == "Policy" {
			mutations["numeric rule decoder"] = strings.Replace(base, "Rules []map[string]any", "Rules []map[string]float64", 1)
		} else {
			mutations["arbitrary address decoder"] = strings.Replace(base, "Addresses []string", "Addresses []any", 1)
		}
		for name, changed := range mutations {
			t.Run(identity.model+"/"+name, func(t *testing.T) {
				if changed == base {
					t.Fatal("native field mutation did not apply")
				}
				badPkg, badPlan := identityQueryFixture(t, identity, changed)
				if _, ok := bodyFilterCollectionContract(badPkg, badPlan, 0); ok {
					t.Fatal("invalid native selected field enabled")
				}
				if err := validateBodyFilterCollectionContracts(badPkg, badPlan); err == nil || !strings.Contains(err.Error(), "schema changed") {
					t.Fatal("native field drift silently dropped capability", err)
				}
				if err := validateIdentityCollectionContracts(badPkg, pinnedIdentityDeclarations(t, identity), badPlan, nil, identityQuerySourceConstants(t, changed)); err == nil {
					t.Fatal("shared native model/schema gate bypassed")
				}
			})
		}
		for name, mutate := range map[string]func(*bodyFilterCollectionSpec){
			"unreviewed path":  func(value *bodyFilterCollectionSpec) { value.path = "network/v2/ports" },
			"wrong model":      func(value *bodyFilterCollectionSpec) { value.model = "Port" },
			"public alias":     func(value *bodyFilterCollectionSpec) { value.fields[0].key = "local_alias" },
			"different member": func(value *bodyFilterCollectionSpec) { value.fields[0].member = "Name" },
			"additional field": func(value *bodyFilterCollectionSpec) {
				value.fields = append(value.fields, bodyFilterCollectionField{key: "name", member: "Name"})
			},
		} {
			t.Run(identity.model+"/"+name, func(t *testing.T) {
				copy := spec
				copy.fields = append([]bodyFilterCollectionField(nil), spec.fields...)
				mutate(&copy)
				if bodyFilterCollectionMetadataValid(copy) {
					t.Fatal("unreviewed field metadata accepted", copy)
				}
			})
		}
	}
	if enabled != 2 {
		t.Fatal("missing audited body selectors", enabled)
	}
}

func TestBodyFilterCollectionSelectorsMarshalOnlyTheSelectedNativeField(t *testing.T) {
	for _, identity := range identityCollectionSpecs {
		t.Run(identity.model, func(t *testing.T) {
			source, fields := emittedBodyFilterAdapter(t, identity)
			pkg, plan := identityQueryFixture(t, identity, identityQueryFixtureSource(identity))
			spec, enabled := bodyFilterCollectionContract(pkg, plan, identity.parents)
			if !enabled {
				if fields["BodyFilterFields"] != nil || fields["BodyFilterValue"] != nil {
					t.Fatal("unsupported binding acquired body selectors", string(source))
				}
				return
			}
			field := spec.fields[0]
			descriptor, ok := fields["BodyFilterFields"].(*ast.CompositeLit)
			if !ok || len(descriptor.Elts) != 1 {
				t.Fatal("canonical one-field descriptor missing", string(source))
			}
			pair := descriptor.Elts[0].(*ast.KeyValueExpr)
			key, err := strconv.Unquote(pair.Key.(*ast.BasicLit).Value)
			if err != nil {
				t.Fatal(err)
			}
			value, err := strconv.Unquote(pair.Value.(*ast.BasicLit).Value)
			if err != nil || key != field.key || value != field.key {
				t.Fatal("body descriptor exposed alias or model name", key, value, err)
			}
			callback, ok := fields["BodyFilterValue"].(*ast.FuncLit)
			if !ok || len(callback.Type.Params.List) != 2 || len(callback.Type.Results.List) != 2 {
				t.Fatal("typed body selector callback missing")
			}
			if nodeText(callback.Type.Params.List[0].Type) != "*"+identity.model || nodeText(callback.Type.Params.List[1].Type) != "string" || nodeText(callback.Type.Results.List[0].Type) != "json.RawMessage" {
				t.Fatal("selector lost native model or raw JSON result", nodeText(callback))
			}
			var selection *ast.SwitchStmt
			marshals := 0
			ast.Inspect(callback.Body, func(node ast.Node) bool {
				if branch, ok := node.(*ast.SwitchStmt); ok {
					selection = branch
				}
				call, ok := node.(*ast.CallExpr)
				if ok && nodeText(call.Fun) == "json.Marshal" {
					marshals++
					if len(call.Args) != 1 || nodeText(call.Args[0]) != "v."+field.member {
						t.Fatal("selector projected a whole model or wrong body field", nodeText(call))
					}
				}
				return true
			})
			if marshals != 1 || selection == nil || nodeText(selection.Tag) != "key" || len(selection.Body.List) != 2 {
				t.Fatal("selector did not preserve a single direct marshal per canonical key", nodeText(callback))
			}
			selected := selection.Body.List[0].(*ast.CaseClause)
			if len(selected.List) != 1 || nodeText(selected.List[0]) != strconv.Quote(field.key) || len(selected.Body) != 1 {
				t.Fatal("selected field case changed", nodeText(callback))
			}
			// Direct marshal preserves native nil slices as null and nonnil
			// empty slices as []; no model omitempty/projection can erase them.
			for _, required := range []string{"if v == nil", "resource.ErrInvalidOption", "config.Query[key] = append([]string(nil), values...)", "return result.Extract()"} {
				if !strings.Contains(string(source), required) {
					t.Fatal("nil/error or unchanged wire query behavior lost", required)
				}
			}
			for _, forbidden := range []string{"reflect.", `q.Del("` + field.key + `")`, "json.Marshal(v)", "WithBodyFilter", "IdentityExtraSpecs:"} {
				if strings.Contains(string(source), forbidden) {
					t.Fatal("body selector changed request policy or used model reflection", forbidden)
				}
			}
		})
	}
}

func TestBodyFilterCollectionStillRequiresPinnedNativeContracts(t *testing.T) {
	for _, identity := range qosAddressIdentitySpecs() {
		t.Run(identity.model, func(t *testing.T) {
			source := identityQueryFixtureSource(identity)
			pkg, plan := identityQueryFixture(t, identity, source)
			if err := validateBodyFilterCollectionContracts(pkg, plan); err != nil {
				t.Fatal(err)
			}
			decls := pinnedIdentityDeclarations(t, identity)
			decls["commonResult.Extract"].Body.List = append(decls["commonResult.Extract"].Body.List, &ast.ExprStmt{X: &ast.CallExpr{Fun: ast.NewIdent("differentBodyDecode")}})
			if err := validateIdentityCollectionContracts(pkg, decls, plan, nil, identityQuerySourceConstants(t, source)); err == nil || !strings.Contains(err.Error(), "declaration commonResult.Extract changed") {
				t.Fatal("body-filter binding bypassed pinned native extraction gate", err)
			}
		})
	}
}

func TestBodyFilterCollectionInventoryHasOnlyOwnedCanonicalFields(t *testing.T) {
	enabled := 0
	for _, identity := range identityCollectionSpecs {
		pkg, plan := identityQueryFixture(t, identity, identityQueryFixtureSource(identity))
		fields := bodyFilterCollectionFields(pkg, plan, identity.parents)
		record := collectionRecord{BodyFilterFields: fields}
		encoded, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		if len(fields) == 0 {
			if strings.Contains(string(encoded), "body_filter_fields") {
				t.Fatal("unsupported body capability recorded", string(encoded))
			}
			continue
		}
		enabled++
		if len(fields) != 1 || !strings.Contains(string(encoded), `"body_filter_fields":["`+fields[0]+`"]`) {
			t.Fatal("canonical body field descriptor missing", string(encoded))
		}
		fields[0] = "mutated"
		fresh := bodyFilterCollectionFields(pkg, plan, identity.parents)
		if len(fresh) != 1 || fresh[0] == "mutated" {
			t.Fatal("inventory caller mutated generator-owned schema")
		}
	}
	if enabled != 2 {
		t.Fatal("audited inventory body capability count changed", enabled)
	}
}
