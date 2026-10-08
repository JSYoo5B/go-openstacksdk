// These tests use source fixtures adapted from Gophercloud v2.15.0.
// Copyright 2012-2013 Rackspace, Inc.
// Copyright Gophercloud authors
// SPDX-License-Identifier: Apache-2.0
//
// Local modifications select and combine upstream declarations, add test stubs,
// and create altered source variants to check generator behavior.
// Upstream scope (github.com/gophercloud/gophercloud/v2):
//   results.go
//   openstack/networking/v2/extensions/qos/policies
//   pagination
// See the root THIRD_PARTY_NOTICES.md and licenses/gophercloud-v2.15.0-LICENSE.

package main

import (
	"encoding/json"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func checkedQoSPolicyFilterManifest(t *testing.T) *pythonFilterManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../../..", qosPolicyFilterManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	m, err := decodePythonFilterManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func qosPolicyFilterFixtureSource() string {
	spec := identityCollectionSpec{path: qosPolicySDKPath, model: "Policy", getter: "Get", lister: "List"}
	return strings.Replace(identityQueryFixtureSource(spec), "package fixture", "package policies", 1) + "\nfunc Delete(ctx context.Context, client *gophercloud.ServiceClient,id string)error{return nil}\n"
}

func qosPolicyFilterNativeFixture(t *testing.T, source string) (*types.Package, *collectionPlan) {
	t.Helper()
	return identityQueryFixture(t, identityCollectionSpec{path: qosPolicySDKPath, model: "Policy", getter: "Get", lister: "List"}, source)
}

func qosPolicyBodyPinnedDeclarations(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	result := map[string]*ast.FuncDecl{}
	for _, source := range []string{pinnedQoSIdentitySource, pinnedAddressGroupRootSource + pinnedQoSPolicyAdditionalRootSource} {
		file, err := parser.ParseFile(token.NewFileSet(), "native.go", source, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range file.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok {
				key := identityDeclarationKey(fn)
				if file.Name.Name == "gophercloud" {
					key = "gophercloud." + key
				}
				result[key] = fn
			}
		}
	}
	for name, fn := range pinnedSubnetBodyIdentityDeclarations(t) {
		if strings.HasPrefix(name, "pagination.") {
			result[name] = fn
		}
	}
	return result
}

func TestQoSPolicyPythonFilterManifestKeepsTagQueryAliasesAndLocalTenantRules(t *testing.T) {
	m := checkedQoSPolicyFilterManifest(t)
	if !qosPolicyPythonFilterMetadataValid(m) || len(m.Proof.Files) != 6 || len(m.Proof.Nodes) != 18 || len(m.Reserved) != 9 || m.Query["tenant_id"] != "" || m.Query["rules"] != "" || m.Query["revision_number"] != "" || m.Query["id"] != "id" || m.Query["is_shared"] != "shared" || m.Query["any_tags"] != "tags-any" || m.Body["tenant_id"].Field != "tenant_id" {
		t.Fatal(m)
	}
	for key, field := range pythonFilterBodyFields(m) {
		if key != field {
			t.Fatal("raw local field inference", key, field)
		}
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){
		"tenant-not-query-alias":        func(m *pythonFilterManifest) { m.Query["tenant_id"] = "project_id" },
		"rules-not-server-query":        func(m *pythonFilterManifest) { m.Query["rules"] = "rules" },
		"native-revision-unknown":       func(m *pythonFilterManifest) { m.Query["revision_number"] = "revision_number" },
		"shared-canonical":              func(m *pythonFilterManifest) { delete(m.Query, "is_shared"); m.Query["shared"] = "shared" },
		"tag-canonical":                 func(m *pythonFilterManifest) { delete(m.Query, "any_tags"); m.Query["tags-any"] = "tags-any" },
		"project-not-local-fallback":    func(m *pythonFilterManifest) { m.Body["tenant_id"] = pythonFilterField{Field: "project_id"} },
		"rules-coercion":                func(m *pythonFilterManifest) { f := m.Body["rules"]; f.ResponseType = nil; m.Body["rules"] = f },
		"id-server-only":                func(m *pythonFilterManifest) { m.Body["id"] = pythonFilterField{Field: "id"} },
		"NetworkResource-not-inherited": func(m *pythonFilterManifest) { m.ClassBases = []string{"_base.NetworkResource", "tag.TagMixin"} },
		"protocol-MRO":                  func(m *pythonFilterManifest) { m.MRO = m.MRO[:4] },
		"accepted-count":                func(m *pythonFilterManifest) { m.Counts.AcceptedQuery = 15 },
	} {
		t.Run(name, func(t *testing.T) {
			copy := clonePythonFilterManifest(t, m)
			mutate(copy)
			if qosPolicyPythonFilterMetadataValid(copy) {
				t.Fatal("unreviewed metadata accepted")
			}
		})
	}
	for _, symbol := range []string{"QoSPolicy.project_id", "QoSPolicy.tenant_id", "QoSPolicy.rules", "TagMixin._tag_query_parameters", "TagMixin.tags", "Resource.id", "Resource.__getattribute__", "QueryParameters.__init__"} {
		t.Run(symbol, func(t *testing.T) {
			copy := clonePythonFilterManifest(t, m)
			for i, node := range copy.Proof.Nodes {
				if node.Symbol == symbol {
					copy.Proof.Nodes = append(copy.Proof.Nodes[:i], copy.Proof.Nodes[i+1:]...)
					break
				}
			}
			if qosPolicyPythonFilterMetadataValid(copy) {
				t.Fatal("independent source anchor missing")
			}
		})
	}
}

func TestQoSPolicyBodyFilterNativeContractRejectsModelBuilderInterfacePagerAndSourceDrift(t *testing.T) {
	source := qosPolicyFilterFixtureSource()
	pkg, plan := qosPolicyFilterNativeFixture(t, source)
	if !qosPolicyBodyNativeSchema(pkg, plan) || len(bodyFilterCollectionSpecs) != 11 || len(identityCollectionSpecs) != 20 || !identityCollectionEnabled(pkg, plan, 0) {
		t.Fatal(plan)
	}
	if err := validateQoSPolicyBodyNativeDeclarations(pkg, qosPolicyBodyPinnedDeclarations(t), plan); err != nil {
		t.Fatal(err)
	}
	for name, pair := range map[string][2]string{
		"rules-type":              {"Rules []map[string]any", "Rules []map[string]float64"},
		"tenant-tag":              {"TenantID string `json:\"tenant_id\"`", "TenantID string `json:\"project_id\"`"},
		"strict-time":             {"CreatedAt time.Time", "CreatedAt string"},
		"strict-revision":         {"RevisionNumber int `json:\"revision_number\"`", "RevisionNumber string `json:\"revision_number\"`"},
		"extra-model":             {"type Policy struct{", "type Policy struct{Extra string;"},
		"builder-pointer":         {"func(ListOpts)ToPolicyListQuery", "func(*ListOpts)ToPolicyListQuery"},
		"extra-list-field":        {"type ListOpts struct{", "type ListOpts struct{Fields []string `q:\"fields\"`;"},
		"query-bool-presence":     {"Shared *bool `q:\"shared\"`", "Shared bool `q:\"shared\"`"},
		"query-revision-presence": {"RevisionNumber *int `q:\"revision_number\"`", "RevisionNumber int `q:\"revision_number\"`"},
		"interface-param":         {"interface{ToPolicyListQuery()(string,error)}", "interface{ToPolicyListQuery(string)(string,error)}"},
		"interface-variadic":      {"interface{ToPolicyListQuery()(string,error)}", "interface{ToPolicyListQuery(...string)(string,error)}"},
		"interface-result":        {"interface{ToPolicyListQuery()(string,error)}", "interface{ToPolicyListQuery()(int,error)}"},
		"interface-extra":         {"interface{ToPolicyListQuery()(string,error)}", "interface{ToPolicyListQuery()(string,error);ToOther()string}"},
		"page-own-next":           {"func(PolicyPage)NextPageURL()(string,error){return \"\",nil}", ""},
		"page-state":              {"PolicyPage struct{pagination.LinkedPageBase}", "PolicyPage struct{extra bool;pagination.LinkedPageBase}"},
		"extractor-model":         {"([]Policy,error)", "([]string,error)"},
	} {
		t.Run(name, func(t *testing.T) {
			changed := strings.ReplaceAll(source, pair[0], pair[1])
			if changed == source {
				t.Fatal("mutation absent")
			}
			p, pl := qosPolicyFilterNativeFixture(t, changed)
			if _, ok := bodyFilterCollectionContract(p, pl, 0); ok {
				t.Fatal("native drift enabled")
			}
			if err := validateBodyFilterCollectionContracts(p, pl); err == nil {
				t.Fatal("capability silently dropped")
			}
		})
	}
	for name, tail := range map[string]string{"model-decoder": "\nfunc(*Policy)UnmarshalJSON([]byte)error{return nil}", "page-body": "\nfunc(PolicyPage)GetBody()any{return nil}", "extra-builder": "\nfunc(ListOpts)ToOtherListQuery()(string,error){return \"\",nil}"} {
		t.Run(name, func(t *testing.T) {
			p, pl := qosPolicyFilterNativeFixture(t, source+tail)
			if qosPolicyBodyNativeSchema(p, pl) {
				t.Fatal("new own method accepted")
			}
		})
	}
	for name := range qosPolicyBodyNativeDeclarations {
		t.Run(name, func(t *testing.T) {
			decls := qosPolicyBodyPinnedDeclarations(t)
			decls[name].Body.List = append(decls[name].Body.List, &ast.ExprStmt{X: ast.NewIdent("drift")})
			if err := validateQoSPolicyBodyNativeDeclarations(pkg, decls, plan); err == nil || !strings.Contains(err.Error(), name) {
				t.Fatal("native body chain guard bypassed", err)
			}
		})
	}
	spec, _ := bodyFilterCollectionContract(pkg, plan, 0)
	for name, mutate := range map[string]func(*bodyFilterCollectionSpec){"typed-projection": func(s *bodyFilterCollectionSpec) { s.rawRecord = false }, "project-fallback": func(s *bodyFilterCollectionSpec) { s.fields[1].member = "ProjectID" }, "alias": func(s *bodyFilterCollectionSpec) { s.fields[1].aliases = []string{"project_id"} }, "extra-ID": func(s *bodyFilterCollectionSpec) {
		s.fields = append(s.fields, bodyFilterCollectionField{key: "id", member: "ID"})
	}} {
		t.Run(name, func(t *testing.T) {
			copy := spec
			copy.fields = append([]bodyFilterCollectionField(nil), spec.fields...)
			mutate(&copy)
			if bodyFilterCollectionMetadataValid(copy) {
				t.Fatal("unreviewed metadata accepted")
			}
		})
	}
}

func TestQoSPolicySemanticFilterEmissionPreservesQueryMapsStatusAndPriorFiveBindings(t *testing.T) {
	pkg, plan := qosPolicyFilterNativeFixture(t, qosPolicyFilterFixtureSource())
	m := checkedQoSPolicyFilterManifest(t)
	if err := (&generator{}).validatePythonFilterPlan(pkg, plan); err == nil {
		t.Fatal("source proof optional")
	}
	g := generator{qosPolicyPythonFilters: m}
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	e := emitter{pkg: pkg, imports: map[string]string{}, pythonFilters: g.pythonFilterFor(pkg, plan)}
	e.printf("func(a *API)newResources()*resource.Collection[Policy]{return ")
	emitCollectionAdapter(&e, plan, "a", nil)
	e.printf("}\n")
	emitBodyFilterList(&e, plan)
	output, err := e.source()
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, want := range []string{"qosPolicyBodyFilterValue(record, key)", "resource.BodyStreamWithControl(ctx, upstream.List(a.client, _opts)", "upstream.ExtractPolicies(page)", `"policies", control`, `request.Wrap("List", "policies", err)`, "IdentityFind:", "GetIdentityQuery:", "NameQuery:", "Delete:", `[]string{"qos", "policies", id}`, "return a.listBodyWithControl(ctx, control, options...)"} {
		if !strings.Contains(text, want) {
			t.Fatal("native policy lost", want, text)
		}
	}
	if strings.Count(text, "config.Query[key] = append([]string(nil), values...)") != 2 {
		t.Fatal("whole query maps lost", text)
	}
	for _, bad := range []string{`q.Del("status")`, "WithListQuery(", "BodyFilterValue:", "json.Marshal(v.", "Failed:", `"tenant_id": "project_id"`, "IdentityAllProjectsQuery:", "IdentityExtraSpecs:"} {
		if strings.Contains(text, bad) {
			t.Fatal("unsupported inference", bad, text)
		}
	}
	record := collectionRecord{BodyFilterFields: bodyFilterCollectionFields(pkg, plan, 0), SemanticQueryFilters: pythonFilterQueryFields(m), SemanticBodyFilters: pythonFilterBodyFields(m), SemanticReserved: pythonFilterReserved(m)}
	if len(record.BodyFilterFields) != 2 || len(record.SemanticQueryFilters) != 15 || len(record.SemanticBodyFilters) != 2 || record.SemanticQueryFilters["tenant_id"] != "" || record.SemanticBodyFilters["tenant_id"] != "tenant_id" {
		t.Fatal(record)
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 1); ok {
		t.Fatal("global contract became scoped")
	}
	for _, identity := range identityCollectionSpecs {
		if identity.path == qosPolicySDKPath {
			continue
		}
		p, pl := identityQueryFixture(t, identity, identityQueryFixtureSource(identity))
		if g.pythonFilterFor(p, pl) != nil {
			t.Fatal("descriptor leaked", identity.path)
		}
	}
	for _, target := range []string{"secrets", "containers", "orders", "subnets", "addressgroups"} {
		var p *types.Package
		var pl *collectionPlan
		var manifest *pythonFilterManifest
		var path string
		switch target {
		case "secrets":
			p, pl = secretFilterNativeFixture(t, secretFilterNativeFixtureSource)
			manifest = checkedSecretFilterManifest(t)
			path = "keymanager/v1/secrets"
		case "containers":
			p, pl = containerFilterNativeFixture(t, containerFilterNativeFixtureSource)
			manifest = checkedContainerFilterManifest(t)
			path = "keymanager/v1/containers"
		case "orders":
			p, pl = orderFilterNativeFixture(t, orderFilterNativeFixtureSource)
			manifest = checkedOrderFilterManifest(t)
			path = "keymanager/v1/orders"
		case "addressgroups":
			p, pl = addressGroupFilterNativeFixture(t, addressGroupFilterFixtureSource())
			manifest = checkedAddressGroupFilterManifest(t)
			path = addressGroupSDKPath
		case "subnets":
			spec := identityCollectionSpec{path: "network/v2/subnets", model: "Subnet", getter: "Get", lister: "List"}
			source := strings.Replace(identityQueryFixtureSource(spec), "package fixture", "package subnets", 1) + "\nfunc Delete(ctx context.Context,client *gophercloud.ServiceClient,id string)error{return nil}\n"
			p, pl = identityQueryFixture(t, spec, source)
			manifest = checkedPythonFilterManifest(t)
			path = "network/v2/subnets"
		}
		old := emitter{pkg: p, imports: map[string]string{}, pythonFilters: manifest}
		old.printf("func(a *API)newResources()*resource.Collection[%s]{return ", pl.modelName)
		emitCollectionAdapter(&old, pl, "a", nil)
		old.printf("}\n")
		emitBodyFilterList(&old, pl)
		emitted, err := old.source()
		if err != nil {
			t.Fatal(err)
		}
		existing, err := os.ReadFile(filepath.Join("../../..", path, "resources_generated.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"API.newResources", "API.listBodyWithControl"} {
			if emittedFunctionHash(t, emitted, name) != emittedFunctionHash(t, existing, name) {
				t.Fatal("prior generated function changed", target, name)
			}
		}
	}
}

func TestQoSPolicyPythonFilterPinnedSourceRequiresTagMROAliasAndListCoercionProof(t *testing.T) {
	m := checkedQoSPolicyFilterManifest(t)
	if err := verifyQoSPolicyPythonFilterManifest("", m); err == nil {
		t.Fatal("source optional")
	}
	if err := verifyQoSPolicyPythonFilterManifest(t.TempDir(), m); err == nil {
		t.Fatal("missing source accepted")
	}
	source := os.Getenv("OPENSTACKSDK_SOURCE")
	if source == "" {
		source = "/private/tmp/go-openstacksdk-openstacksdk"
	}
	if _, err := os.Stat(filepath.Join(source, "openstack/network/v2/qos_policy.py")); err != nil {
		t.Skip("pinned source unavailable")
	}
	if err := verifyQoSPolicyPythonFilterManifest(source, m); err != nil {
		t.Fatal(err)
	}
	fresh, err := extractPythonFilterManifestTarget(source, qosPolicyPythonResource)
	if err != nil || !reflect.DeepEqual(fresh, m) {
		t.Fatal(fresh, err)
	}
	root := t.TempDir()
	for path := range qosPolicyFilterSourceHashes {
		data, err := os.ReadFile(filepath.Join(source, path))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for path := range qosPolicyFilterSourceHashes {
		t.Run(path, func(t *testing.T) {
			target := filepath.Join(root, path)
			data, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, append(data, []byte("\n# drift\n")...), 0600); err != nil {
				t.Fatal(err)
			}
			if err := verifyQoSPolicyPythonFilterManifest(root, m); err == nil || !strings.Contains(err.Error(), path) {
				t.Fatal("SHA guard bypassed", err)
			}
			if err := os.WriteFile(target, data, 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){"AST": func(m *pythonFilterManifest) { m.Proof.Nodes[0].ASTSHA256 = "wrong" }, "parser": func(m *pythonFilterManifest) { m.Proof.PythonParser = "other" }, "reserved": func(m *pythonFilterManifest) { m.Reserved = m.Reserved[:8] }, "controls": func(m *pythonFilterManifest) { m.SourceControls.ResourceList = m.SourceControls.ResourceList[:6] }, "proof-path": func(m *pythonFilterManifest) { m.Proof.Files["../outside.py"] = "wrong" }} {
		t.Run(name, func(t *testing.T) {
			copy := clonePythonFilterManifest(t, m)
			mutate(copy)
			if err := verifyQoSPolicyPythonFilterManifest(root, copy); err == nil {
				t.Fatal("manifest replaced live source")
			}
		})
	}
	loaded, err := loadQoSPolicyPythonFilterManifest("../../..", source)
	if err != nil || !reflect.DeepEqual(loaded, m) {
		t.Fatal(loaded, err)
	}
	for target, wanted := range map[string]*pythonFilterManifest{"": checkedPythonFilterManifest(t), secretPythonResource: checkedSecretFilterManifest(t), containerPythonResource: checkedContainerFilterManifest(t), orderPythonResource: checkedOrderFilterManifest(t), addressGroupPythonResource: checkedAddressGroupFilterManifest(t)} {
		actual, err := extractPythonFilterManifestTarget(source, target)
		if err != nil || !reflect.DeepEqual(actual, wanted) {
			t.Fatal("previous manifest changed", target, err)
		}
	}
}

func TestQoSPolicyRawDependencyLoaderPreservesSliceDecoderNumbersAndNativeLinks(t *testing.T) {
	pkg, plan := qosPolicyFilterNativeFixture(t, qosPolicyFilterFixtureSource())
	if _, err := (&generator{}).bodyRecordRootDeclarations(pkg.Path()); err == nil {
		t.Fatal("missing root dependency accepted")
	}
	if _, err := (&generator{}).identityPaginationDeclarations(pkg.Path()); err == nil {
		t.Fatal("missing page dependency accepted")
	}
	dir := t.TempDir()
	page := filepath.Join(dir, "page.go")
	root := filepath.Join(dir, "result.go")
	rootSource := pinnedAddressGroupRootSource + pinnedQoSPolicyAdditionalRootSource
	if err := os.WriteFile(page, []byte(pinnedSubnetBodyIdentitySource), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte(rootSource), 0600); err != nil {
		t.Fatal(err)
	}
	g := generator{meta: map[string]metadata{upstreamModule + "/pagination": {Dir: dir, GoFiles: []string{"page.go"}}, upstreamModule: {Dir: dir, GoFiles: []string{"result.go"}}}}
	rootDeps, err := g.bodyRecordRootDeclarations(pkg.Path())
	if err != nil || len(rootDeps) != 4 {
		t.Fatal(rootDeps, err)
	}
	pageDeps, err := g.identityPaginationDeclarations(pkg.Path())
	if err != nil || len(pageDeps) != 3 {
		t.Fatal(pageDeps, err)
	}
	decls := qosPolicyBodyPinnedDeclarations(t)
	for key, fn := range rootDeps {
		decls[key] = fn
	}
	for key, fn := range pageDeps {
		decls[key] = fn
	}
	if err := validateQoSPolicyBodyNativeDeclarations(pkg, decls, plan); err != nil {
		t.Fatal(err)
	}
	old, err := g.addressGroupBodyRootDeclarations(upstreamModule + "/openstack/networking/v2/extensions/security/addressgroups")
	if err != nil || len(old) != 2 || old["gophercloud.Result.extractIntoPtr"] != nil {
		t.Fatal("AddressGroup dependencies widened", old, err)
	}
	for _, path := range []string{upstreamModule + "/openstack/networking/v2/ports", upstreamModule + "/openstack/keymanager/v1/secrets"} {
		deps, err := g.bodyRecordRootDeclarations(path)
		if err != nil || deps != nil {
			t.Fatal("dependencies leaked", deps, err)
		}
	}
	for name, fn := range rootDeps {
		t.Run(name, func(t *testing.T) {
			bad := qosPolicyBodyPinnedDeclarations(t)
			fncopy := *fn
			bodycopy := *fn.Body
			bodycopy.List = append(append([]ast.Stmt(nil), fn.Body.List...), &ast.ExprStmt{X: ast.NewIdent("drift")})
			fncopy.Body = &bodycopy
			bad[name] = &fncopy
			if err := validateQoSPolicyBodyNativeDeclarations(pkg, bad, plan); err == nil || !strings.Contains(err.Error(), name) {
				t.Fatal("root decoder guard bypassed", err)
			}
		})
	}
	if err := os.WriteFile(page, []byte(strings.Replace(pinnedSubnetBodyIdentitySource, "dec.UseNumber()", "", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	pd, err := g.identityPaginationDeclarations(pkg.Path())
	if err != nil {
		t.Fatal(err)
	}
	for key, fn := range pd {
		decls[key] = fn
	}
	if err := validateQoSPolicyBodyNativeDeclarations(pkg, decls, plan); err == nil || !strings.Contains(err.Error(), "pagination.PageResultFrom") {
		t.Fatal("raw precision guard bypassed", err)
	}
	for name, code := range map[string]string{"missing": "package gophercloud", "duplicate": rootSource + "\nfunc(r Result)extractIntoPtr(v any,label string)error{return nil}", "syntax": "package gophercloud\nfunc {"} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(root, []byte(code), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := g.bodyRecordRootDeclarations(pkg.Path()); err == nil {
				t.Fatal("invalid root dependency accepted")
			}
		})
	}
}

func TestQoSPolicyBodyFilterAcceptsActualCompiledNativeSchemasAndOwnMethods(t *testing.T) {
	path := os.Getenv("GOPHERCLOUD_METADATA")
	if path == "" {
		path = "/private/tmp/go-openstacksdk-upstream-packages.json"
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		t.Skip("actual compiled metadata unavailable")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	meta := map[string]metadata{}
	decoder := json.NewDecoder(file)
	for {
		var entry metadata
		if err := decoder.Decode(&entry); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		meta[entry.ImportPath] = entry
	}
	path = upstreamModule + "/openstack/networking/v2/extensions/qos/policies"
	native, ok := meta[path]
	if !ok || native.Dir == "" || native.Export == "" {
		t.Fatal("native source/export missing")
	}
	compiled := importer.ForCompiler(token.NewFileSet(), "gc", func(path string) (io.ReadCloser, error) {
		entry, ok := meta[path]
		if !ok || entry.Export == "" {
			return nil, os.ErrNotExist
		}
		return os.Open(entry.Export)
	})
	if _, err := compiled.Import(upstreamModule); err != nil {
		t.Fatal(err)
	}
	pkg, err := compiled.Import(path)
	if err != nil {
		t.Fatal(err)
	}
	decls, nativeDecls := map[string]*ast.FuncDecl{}, map[string]*ast.FuncDecl{}
	for _, name := range native.GoFiles {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(native.Dir, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range file.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok {
				nativeDecls[identityDeclarationKey(fn)] = fn
				if fn.Recv == nil && ast.IsExported(fn.Name.Name) {
					decls[fn.Name.Name] = fn
				}
			}
		}
	}
	plan, err := identifyCollectionBinding(pkg, decls, extractorsByPage(pkg, decls))
	if err != nil {
		t.Fatal(err)
	}
	if err := validateBodyFilterCollectionContracts(pkg, plan); err != nil {
		t.Fatal(err)
	}
	g := generator{meta: meta, qosPolicyPythonFilters: checkedQoSPolicyFilterManifest(t)}
	pd, err := g.identityPaginationDeclarations(path)
	if err != nil {
		t.Fatal(err)
	}
	rd, err := g.bodyRecordRootDeclarations(path)
	if err != nil {
		t.Fatal(err)
	}
	for key, fn := range pd {
		nativeDecls[key] = fn
	}
	for key, fn := range rd {
		nativeDecls[key] = fn
	}
	if err := validateQoSPolicyBodyNativeDeclarations(pkg, nativeDecls, plan); err != nil {
		t.Fatal(err)
	}
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	for name, count := range map[string]int{"Policy": 0, "ListOpts": 1, "PolicyPage": 2, "commonResult": 1, "GetResult": 0, "CreateResult": 0, "UpdateResult": 0, "DeleteResult": 0, "QoSPolicyExt": 0, "CreateOpts": 1, "UpdateOpts": 1, "PortCreateOptsExt": 1, "PortUpdateOptsExt": 1, "NetworkCreateOptsExt": 1, "NetworkUpdateOptsExt": 1} {
		obj := pkg.Scope().Lookup(name)
		if obj == nil {
			t.Fatal("actual type omitted", name)
		}
		n, ok := obj.Type().(*types.Named)
		if !ok || n.NumMethods() != count {
			t.Fatal("actual own method graph", name, n)
		}
	}
	for _, name := range []string{"PolicyListOptsBuilder", "CreateOptsBuilder", "UpdateOptsBuilder"} {
		obj := pkg.Scope().Lookup(name)
		iface, ok := obj.Type().Underlying().(*types.Interface)
		if !ok || iface.NumEmbeddeds() != 0 || iface.NumMethods() != 1 {
			t.Fatal("actual interface graph", name, iface)
		}
	}
	if len(bodyFilterCollectionFields(pkg, plan, 0)) != 2 || !identityCollectionEnabled(pkg, plan, 0) || plan.id != "ID" || plan.name != "Name" || plan.status != "" {
		t.Fatal(plan)
	}
}

const pinnedQoSPolicyAdditionalRootSource = "\nfunc (r Result) extractIntoPtr(to any, label string) error {\n\tif label == \"\" {\n\t\treturn r.ExtractInto(&to)\n\t}\n\n\t// Decode into map[string]any with UseNumber so integers that do not fit\n\t// in float64 survive the extra marshal/unmarshal below. ExtractInto() is\n\t// not used here because it would decode numbers into any as float64.\n\tvar m map[string]any\n\tif reader, ok := r.Body.(io.Reader); ok {\n\t\tif readCloser, ok := reader.(io.Closer); ok {\n\t\t\tdefer readCloser.Close()\n\t\t}\n\t\tdec := json.NewDecoder(reader)\n\t\tdec.UseNumber()\n\t\tif err := dec.Decode(&m); err != nil {\n\t\t\treturn err\n\t\t}\n\t} else {\n\t\tb, err := json.Marshal(r.Body)\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\tdec := json.NewDecoder(bytes.NewReader(b))\n\t\tdec.UseNumber()\n\t\tif err := dec.Decode(&m); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\n\t// Check if the expected label exists in the response\n\tvalue, exists := m[label]\n\tif !exists && len(m) > 0 {\n\t\t// Key doesn't exist but response has other data - this is an error\n\t\t// If len(m) == 0, we allow empty responses (e.g., tokens API where data is in headers)\n\t\treturn fmt.Errorf(\"expected response key %q not found in response\", label)\n\t}\n\n\tb, err := json.Marshal(value)\n\tif err != nil {\n\t\treturn err\n\t}\n\n\ttoValue := reflect.ValueOf(to)\n\tif toValue.Kind() == reflect.Pointer {\n\t\ttoValue = toValue.Elem()\n\t}\n\n\tswitch toValue.Kind() {\n\tcase reflect.Slice:\n\t\ttypeOfV := toValue.Type().Elem()\n\t\tif typeOfV.Kind() == reflect.Struct {\n\t\t\tif typeOfV.NumField() > 0 && typeOfV.Field(0).Anonymous {\n\t\t\t\tnewSlice := reflect.MakeSlice(reflect.SliceOf(typeOfV), 0, 0)\n\n\t\t\t\tif mSlice, ok := m[label].([]any); ok {\n\t\t\t\t\tfor _, v := range mSlice {\n\t\t\t\t\t\t// For each iteration of the slice, we create a new struct.\n\t\t\t\t\t\t// This is to work around a bug where elements of a slice\n\t\t\t\t\t\t// are reused and not overwritten when the same copy of the\n\t\t\t\t\t\t// struct is used:\n\t\t\t\t\t\t//\n\t\t\t\t\t\t// https://github.com/golang/go/issues/21092\n\t\t\t\t\t\t// https://github.com/golang/go/issues/24155\n\t\t\t\t\t\t// https://play.golang.org/p/NHo3ywlPZli\n\t\t\t\t\t\tnewType := reflect.New(typeOfV).Elem()\n\n\t\t\t\t\t\tb, err := json.Marshal(v)\n\t\t\t\t\t\tif err != nil {\n\t\t\t\t\t\t\treturn err\n\t\t\t\t\t\t}\n\n\t\t\t\t\t\t// This is needed for structs with an UnmarshalJSON method.\n\t\t\t\t\t\t// Technically this is just unmarshalling the response into\n\t\t\t\t\t\t// a struct that is never used, but it's good enough to\n\t\t\t\t\t\t// trigger the UnmarshalJSON method.\n\t\t\t\t\t\tfor i := 0; i < newType.NumField(); i++ {\n\t\t\t\t\t\t\tif newType.Field(i).Kind() != reflect.Struct {\n\t\t\t\t\t\t\t\tcontinue\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\ts := newType.Field(i).Addr().Interface()\n\n\t\t\t\t\t\t\t// Unmarshal is used rather than NewDecoder to also work\n\t\t\t\t\t\t\t// around the above-mentioned bug.\n\t\t\t\t\t\t\terr = json.Unmarshal(b, s)\n\t\t\t\t\t\t\tif err != nil {\n\t\t\t\t\t\t\t\treturn err\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t}\n\n\t\t\t\t\t\tnewSlice = reflect.Append(newSlice, newType)\n\t\t\t\t\t}\n\t\t\t\t}\n\n\t\t\t\t// \"to\" should now be properly modeled to receive the\n\t\t\t\t// JSON response body and unmarshal into all the correct\n\t\t\t\t// fields of the struct or composed extension struct\n\t\t\t\t// at the end of this method.\n\t\t\t\ttoValue.Set(newSlice)\n\n\t\t\t\t// jtopjian: This was put into place to resolve the issue\n\t\t\t\t// described at\n\t\t\t\t// https://github.com/gophercloud/gophercloud/issues/1963\n\t\t\t\t//\n\t\t\t\t// This probably isn't the best fix, but it appears to\n\t\t\t\t// be resolving the issue, so I'm going to implement it\n\t\t\t\t// for now.\n\t\t\t\t//\n\t\t\t\t// For future readers, this entire case statement could\n\t\t\t\t// use a review.\n\t\t\t\treturn nil\n\t\t\t}\n\t\t}\n\tcase reflect.Struct:\n\t\ttypeOfV := toValue.Type()\n\t\tif typeOfV.NumField() > 0 && typeOfV.Field(0).Anonymous {\n\t\t\tfor i := 0; i < toValue.NumField(); i++ {\n\t\t\t\ttoField := toValue.Field(i)\n\t\t\t\tif toField.Kind() == reflect.Struct {\n\t\t\t\t\ts := toField.Addr().Interface()\n\t\t\t\t\terr = json.NewDecoder(bytes.NewReader(b)).Decode(s)\n\t\t\t\t\tif err != nil {\n\t\t\t\t\t\treturn err\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t}\n\n\terr = json.Unmarshal(b, &to)\n\treturn err\n}\n\nfunc (r Result) ExtractIntoSlicePtr(to any, label string) error {\n\tif r.Err != nil {\n\t\treturn r.Err\n\t}\n\n\tif to == nil {\n\t\treturn fmt.Errorf(\"expected pointer, got %T\", to)\n\t}\n\n\tt := reflect.TypeOf(to)\n\tif k := t.Kind(); k != reflect.Pointer {\n\t\treturn fmt.Errorf(\"expected pointer, got %v\", k)\n\t}\n\n\tif reflect.ValueOf(to).IsNil() {\n\t\treturn fmt.Errorf(\"expected pointer, got %T\", to)\n\t}\n\n\tswitch t.Elem().Kind() {\n\tcase reflect.Slice:\n\t\treturn r.extractIntoPtr(to, label)\n\tdefault:\n\t\treturn fmt.Errorf(\"expected pointer to slice, got: %v\", t)\n\t}\n}\n"
