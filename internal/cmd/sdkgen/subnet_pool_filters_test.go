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

func checkedSubnetPoolFilterManifest(t *testing.T) *pythonFilterManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../../..", subnetPoolFilterManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	m, err := decodePythonFilterManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func subnetPoolFilterFixtureSource() string {
	spec := identityCollectionSpec{path: subnetPoolSDKPath, model: "SubnetPool", getter: "Get", lister: "List"}
	return strings.Replace(identityQueryFixtureSource(spec), "package fixture", "package subnetpools", 1) + "\nfunc Delete(ctx context.Context, client *gophercloud.ServiceClient,id string)error{return nil}\n"
}

func subnetPoolFilterNativeFixture(t *testing.T, source string) (*types.Package, *collectionPlan) {
	t.Helper()
	return identityQueryFixture(t, identityCollectionSpec{path: subnetPoolSDKPath, model: "SubnetPool", getter: "Get", lister: "List"}, source)
}

func subnetPoolBodyPinnedDeclarations(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	result := map[string]*ast.FuncDecl{}
	for _, source := range []string{pinnedSubnetPoolIdentitySource, pinnedAddressGroupRootSource + pinnedSubnetPoolAdditionalRootSource + `
const RFC3339NoZ = "2006-01-02T15:04:05"
`} {
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
	file, err := parser.ParseFile(token.NewFileSet(), "pager.go", "package pagination\n"+pinnedSubnetPoolAdditionalPagerSource, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range file.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			result["pagination."+identityDeclarationKey(fn)] = fn
		}
	}
	for name, fn := range pinnedSubnetBodyIdentityDeclarations(t) {
		if strings.HasPrefix(name, "pagination.") {
			result[name] = fn
		}
	}
	return result
}

func TestSubnetPoolPythonFilterManifestKeepsTagAliasesLocalIntegersAndInheritedMRO(t *testing.T) {
	m := checkedSubnetPoolFilterManifest(t)
	if !subnetPoolPythonFilterMetadataValid(m) || len(m.Proof.Files) != 7 || len(m.Proof.Nodes) != 33 || len(m.Reserved) != 9 || m.Query["tenant_id"] != "" || m.Query["prefixes"] != "" || m.Query["revision_number"] != "" || m.Query["fields"] != "fields" || m.Query["is_shared"] != "shared" || m.Query["any_tags"] != "tags-any" {
		t.Fatal(m)
	}
	for _, attr := range []string{"default_prefix_length", "default_quota", "maximum_prefix_length", "minimum_prefix_length", "revision_number"} {
		field := m.Body[attr]
		if field.ResponseType == nil || *field.ResponseType != "int" {
			t.Fatal(attr, field)
		}
	}
	for _, key := range []string{"default_prefixlen", "min_prefixlen", "max_prefixlen"} {
		if _, ok := m.Body[key]; ok {
			t.Fatal("wire spelling inferred as Python property", key)
		}
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){
		"tenant-is-local":          func(m *pythonFilterManifest) { m.Query["tenant_id"] = "project_id" },
		"prefixes-local":           func(m *pythonFilterManifest) { m.Query["prefixes"] = "prefixes" },
		"no-native-revision-query": func(m *pythonFilterManifest) { m.Query["revision_number"] = "revision_number" },
		"shared-canonical":         func(m *pythonFilterManifest) { delete(m.Query, "is_shared"); m.Query["shared"] = "shared" },
		"tag-canonical":            func(m *pythonFilterManifest) { delete(m.Query, "any_tags"); m.Query["tags-any"] = "tags-any" },
		"prefix-property": func(m *pythonFilterManifest) {
			f := m.Body["minimum_prefix_length"]
			delete(m.Body, "minimum_prefix_length")
			m.Body["min_prefixlen"] = f
		},
		"integer-coercion": func(m *pythonFilterManifest) {
			f := m.Body["default_prefix_length"]
			f.ResponseType = nil
			m.Body["default_prefix_length"] = f
		},
		"list-coercion":       func(m *pythonFilterManifest) { f := m.Body["prefixes"]; f.ResponseType = nil; m.Body["prefixes"] = f },
		"no-project-fallback": func(m *pythonFilterManifest) { m.Body["tenant_id"] = pythonFilterField{Field: "project_id"} },
		"NetworkResource-not-parent": func(m *pythonFilterManifest) {
			m.ClassBases = []string{"_base.NetworkResource", "_base.TagMixinNetwork"}
		},
		"protocol-MRO":   func(m *pythonFilterManifest) { m.MRO = m.MRO[:5] },
		"accepted-count": func(m *pythonFilterManifest) { m.Counts.AcceptedQuery = 16 },
	} {
		t.Run(name, func(t *testing.T) {
			copy := clonePythonFilterManifest(t, m)
			mutate(copy)
			if subnetPoolPythonFilterMetadataValid(copy) {
				t.Fatal("unreviewed metadata accepted")
			}
		})
	}
	for _, symbol := range []string{"SubnetPool.project_id", "SubnetPool.tenant_id", "SubnetPool.prefixes", "SubnetPool.default_prefix_length", "SubnetPool.default_quota", "SubnetPool.maximum_prefix_length", "SubnetPool.minimum_prefix_length", "SubnetPool.revision_number", "SubnetPool.created_at", "SubnetPool.updated_at", "TagMixinNetwork", "TagMixin._tag_query_parameters", "TagMixin.tags", "Resource.id", "Resource.__getattribute__", "QueryParameters.__init__", "_BaseComponent.__init__", "_convert_type", "Resource.list._dict_filter"} {
		t.Run(symbol, func(t *testing.T) {
			copy := clonePythonFilterManifest(t, m)
			removed := false
			for i, node := range copy.Proof.Nodes {
				if node.Symbol == symbol {
					copy.Proof.Nodes = append(copy.Proof.Nodes[:i], copy.Proof.Nodes[i+1:]...)
					removed = true
					break
				}
			}
			if !removed || subnetPoolPythonFilterMetadataValid(copy) {
				t.Fatal("independent source anchor missing", removed)
			}
		})
	}
}

func TestSubnetPoolBodyFilterNativeContractRejectsModelBuilderDecoderAndSourceDrift(t *testing.T) {
	source := subnetPoolFilterFixtureSource()
	pkg, plan := subnetPoolFilterNativeFixture(t, source)
	if !subnetPoolBodyNativeSchema(pkg, plan) || len(bodyFilterCollectionSpecs) != 9 || len(identityCollectionSpecs) != 20 || !identityCollectionEnabled(pkg, plan, 0) {
		t.Fatal(plan)
	}
	decls := subnetPoolBodyPinnedDeclarations(t)
	if len(decls) != 22 || len(subnetPoolBodyNativeDeclarations) != 22 {
		t.Fatal("reachable list chain changed", len(decls))
	}
	if err := validateSubnetPoolBodyNativeDeclarations(pkg, decls, plan); err != nil {
		t.Fatal(err)
	}
	for name, pair := range map[string][2]string{
		"prefixes-type": {"Prefixes []string", "Prefixes []any"}, "quota-type": {"DefaultQuota int `json:", "DefaultQuota int64 `json:"}, "tenant-tag": {"TenantID string `json:\"tenant_id\"`", "TenantID string `json:\"project_id\"`"}, "strict-time": {"CreatedAt time.Time", "CreatedAt string"}, "prefix-tag": {"MinPrefixLen int `json:\"-\"`", "MinPrefixLen int `json:\"min_prefixlen\"`"}, "extra-model": {"type SubnetPool struct{", "type SubnetPool struct{Extra string;"}, "decoder-value-receiver": {"func(*SubnetPool)UnmarshalJSON", "func(SubnetPool)UnmarshalJSON"}, "decoder-type": {"UnmarshalJSON([]byte)", "UnmarshalJSON(string)"}, "builder-pointer": {"func(ListOpts)ToSubnetPoolListQuery", "func(*ListOpts)ToSubnetPoolListQuery"}, "unproved-native-fields": {"type ListOpts struct{", "type ListOpts struct{Fields []string `q:\"fields\"`;"}, "query-bool-presence": {"Shared *bool `q:\"shared\"`", "Shared bool `q:\"shared\"`"}, "interface-param": {"interface{ToSubnetPoolListQuery()(string,error)}", "interface{ToSubnetPoolListQuery(string)(string,error)}"}, "interface-variadic": {"interface{ToSubnetPoolListQuery()(string,error)}", "interface{ToSubnetPoolListQuery(...string)(string,error)}"}, "interface-result": {"interface{ToSubnetPoolListQuery()(string,error)}", "interface{ToSubnetPoolListQuery()(int,error)}"}, "interface-extra": {"interface{ToSubnetPoolListQuery()(string,error)}", "interface{ToSubnetPoolListQuery()(string,error);ToOther()string}"}, "page-own-next": {"func(SubnetPoolPage)NextPageURL()(string,error){return \"\",nil}", ""}, "page-state": {"SubnetPoolPage struct{pagination.LinkedPageBase}", "SubnetPoolPage struct{extra bool;pagination.LinkedPageBase}"}, "extractor-model": {"([]SubnetPool,error)", "([]string,error)"},
	} {
		t.Run(name, func(t *testing.T) {
			changed := strings.ReplaceAll(source, pair[0], pair[1])
			if changed == source {
				t.Fatal("mutation absent")
			}
			p, pl := subnetPoolFilterNativeFixture(t, changed)
			if _, ok := bodyFilterCollectionContract(p, pl, 0); ok {
				t.Fatal("native drift enabled")
			}
			if err := validateBodyFilterCollectionContracts(p, pl); err == nil {
				t.Fatal("capability silently dropped")
			}
		})
	}
	for name, tail := range map[string]string{"extra-model-method": "\nfunc(SubnetPool)Other(){}", "page-body": "\nfunc(SubnetPoolPage)GetBody()any{return nil}", "extra-builder": "\nfunc(ListOpts)ToOtherListQuery()(string,error){return \"\",nil}", "get-result-own-extractor": "\nfunc(GetResult)Extract()(*SubnetPool,error){return nil,nil}"} {
		t.Run(name, func(t *testing.T) {
			p, pl := subnetPoolFilterNativeFixture(t, source+tail)
			if subnetPoolBodyNativeSchema(p, pl) {
				t.Fatal("new own method accepted")
			}
		})
	}
	for _, name := range []string{"Link", "JSONRFC3339NoZ"} {
		t.Run("root-own-method-"+name, func(t *testing.T) {
			p, pl := subnetPoolFilterNativeFixture(t, source)
			for _, dep := range p.Imports() {
				if dep.Path() == upstreamModule {
					n := dep.Scope().Lookup(name).Type().(*types.Named)
					n.AddMethod(types.NewFunc(token.NoPos, dep, "NewMethod", types.NewSignatureType(types.NewVar(token.NoPos, dep, "", n), nil, nil, types.NewTuple(), types.NewTuple(), false)))
				}
			}
			if subnetPoolBodyNativeSchema(p, pl) {
				t.Fatal("root own method drift accepted")
			}
		})
	}
	t.Run("time-wrapper-underlying", func(t *testing.T) {
		p, pl := subnetPoolFilterNativeFixture(t, source)
		for _, dep := range p.Imports() {
			if dep.Path() == upstreamModule {
				dep.Scope().Lookup("JSONRFC3339NoZ").Type().(*types.Named).SetUnderlying(types.Typ[types.Int])
			}
		}
		if subnetPoolBodyNativeSchema(p, pl) {
			t.Fatal("root time wrapper drift accepted")
		}
	})
	for name := range subnetPoolBodyNativeDeclarations {
		t.Run(name, func(t *testing.T) {
			decls := subnetPoolBodyPinnedDeclarations(t)
			decls[name].Body.List = append(decls[name].Body.List, &ast.ExprStmt{X: ast.NewIdent("drift")})
			if err := validateSubnetPoolBodyNativeDeclarations(pkg, decls, plan); err == nil || !strings.Contains(err.Error(), name) {
				t.Fatal("native body chain guard bypassed", err)
			}
		})
	}
	spec, _ := bodyFilterCollectionContract(pkg, plan, 0)
	for name, mutate := range map[string]func(*bodyFilterCollectionSpec){"typed-projection": func(s *bodyFilterCollectionSpec) { s.rawRecord = false }, "prefix-alias-removed": func(s *bodyFilterCollectionSpec) { s.fields[1].aliases = nil }, "project-fallback": func(s *bodyFilterCollectionSpec) { s.fields[8].member = "ProjectID" }, "extra-name": func(s *bodyFilterCollectionSpec) {
		s.fields = append(s.fields, bodyFilterCollectionField{key: "name", member: "Name"})
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

func TestSubnetPoolSemanticFilterEmissionPreservesQueryMapsStatusAndPriorSixBindings(t *testing.T) {
	pkg, plan := subnetPoolFilterNativeFixture(t, subnetPoolFilterFixtureSource())
	m := checkedSubnetPoolFilterManifest(t)
	if err := (&generator{}).validatePythonFilterPlan(pkg, plan); err == nil {
		t.Fatal("source proof optional")
	}
	g := generator{subnetPoolPythonFilters: m}
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	e := emitter{pkg: pkg, imports: map[string]string{}, pythonFilters: g.pythonFilterFor(pkg, plan)}
	e.printf("func(a *API)newResources()*resource.Collection[SubnetPool]{return ")
	emitCollectionAdapter(&e, plan, "a", nil)
	e.printf("}\n")
	emitBodyFilterList(&e, plan)
	output, err := e.source()
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, want := range []string{"subnetPoolBodyFilterValue(record, key)", "resource.BodyStreamWithControl(ctx, upstream.List(a.client, _opts)", "upstream.ExtractSubnetPools(page)", `"subnetpools", control`, `request.Wrap("List", "subnetpools", err)`, "IdentityFind:", "GetIdentityQuery:", "NameQuery:", "Delete:", `[]string{"subnetpools", id}`, "return a.listBodyWithControl(ctx, control, options...)"} {
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
	if len(record.BodyFilterFields) != 10 || len(record.SemanticQueryFilters) != 16 || len(record.SemanticBodyFilters) != 10 || record.SemanticQueryFilters["tenant_id"] != "" || record.SemanticBodyFilters["tenant_id"] != "tenant_id" {
		t.Fatal(record)
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 1); ok {
		t.Fatal("global contract became scoped")
	}
	for _, identity := range identityCollectionSpecs {
		if identity.path == subnetPoolSDKPath {
			continue
		}
		p, pl := identityQueryFixture(t, identity, identityQueryFixtureSource(identity))
		if g.pythonFilterFor(p, pl) != nil {
			t.Fatal("descriptor leaked", identity.path)
		}
	}
	for _, target := range []string{"secrets", "containers", "orders", "subnets", "addressgroups", "policies"} {
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
		case "policies":
			p, pl = qosPolicyFilterNativeFixture(t, qosPolicyFilterFixtureSource())
			manifest = checkedQoSPolicyFilterManifest(t)
			path = qosPolicySDKPath
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

func TestSubnetPoolPythonFilterPinnedSourceRequiresIntegerCoercionAndClassificationProof(t *testing.T) {
	m := checkedSubnetPoolFilterManifest(t)
	if err := verifySubnetPoolPythonFilterManifest("", m); err == nil {
		t.Fatal("source optional")
	}
	if err := verifySubnetPoolPythonFilterManifest(t.TempDir(), m); err == nil {
		t.Fatal("missing source accepted")
	}
	source := os.Getenv("OPENSTACKSDK_SOURCE")
	if source == "" {
		source = "/private/tmp/gophercloudsdk-openstacksdk"
	}
	if _, err := os.Stat(filepath.Join(source, "openstack/network/v2/subnet_pool.py")); err != nil {
		t.Skip("pinned source unavailable")
	}
	if err := verifySubnetPoolPythonFilterManifest(source, m); err != nil {
		t.Fatal(err)
	}
	fresh, err := extractPythonFilterManifestTarget(source, subnetPoolPythonResource)
	if err != nil || !reflect.DeepEqual(fresh, m) {
		t.Fatal(fresh, err)
	}
	root := t.TempDir()
	for path := range subnetPoolFilterSourceHashes {
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
	for path := range subnetPoolFilterSourceHashes {
		t.Run(path, func(t *testing.T) {
			target := filepath.Join(root, path)
			data, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, append(data, []byte("\n# drift\n")...), 0600); err != nil {
				t.Fatal(err)
			}
			if err := verifySubnetPoolPythonFilterManifest(root, m); err == nil || !strings.Contains(err.Error(), path) {
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
			if err := verifySubnetPoolPythonFilterManifest(root, copy); err == nil {
				t.Fatal("manifest replaced live source")
			}
		})
	}
	loaded, err := loadSubnetPoolPythonFilterManifest("../../..", source)
	if err != nil || !reflect.DeepEqual(loaded, m) {
		t.Fatal(loaded, err)
	}
	for target, wanted := range map[string]*pythonFilterManifest{"": checkedPythonFilterManifest(t), secretPythonResource: checkedSecretFilterManifest(t), containerPythonResource: checkedContainerFilterManifest(t), orderPythonResource: checkedOrderFilterManifest(t), addressGroupPythonResource: checkedAddressGroupFilterManifest(t), qosPolicyPythonResource: checkedQoSPolicyFilterManifest(t)} {
		actual, err := extractPythonFilterManifestTarget(source, target)
		if err != nil || !reflect.DeepEqual(actual, wanted) {
			t.Fatal("previous manifest changed", target, err)
		}
	}
}

func TestSubnetPoolRawDependencyLoaderPreservesNativeTimeCodesAndNumbers(t *testing.T) {
	pkg, plan := subnetPoolFilterNativeFixture(t, subnetPoolFilterFixtureSource())
	if _, err := (&generator{}).bodyRecordRootDeclarations(pkg.Path()); err == nil {
		t.Fatal("missing root dependency accepted")
	}
	if _, err := (&generator{}).identityPaginationDeclarations(pkg.Path()); err == nil {
		t.Fatal("missing page dependency accepted")
	}
	dir := t.TempDir()
	page := filepath.Join(dir, "page.go")
	root := filepath.Join(dir, "result.go")
	rootSource := pinnedAddressGroupRootSource + pinnedSubnetPoolAdditionalRootSource + `\nconst RFC3339NoZ="2006-01-02T15:04:05"\n`
	rootSource = strings.ReplaceAll(rootSource, `\n`, "\n")
	pageSource := pinnedSubnetBodyIdentitySource + pinnedSubnetPoolAdditionalPagerSource
	if err := os.WriteFile(page, []byte(pageSource), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte(rootSource), 0600); err != nil {
		t.Fatal(err)
	}
	g := generator{meta: map[string]metadata{upstreamModule + "/pagination": {Dir: dir, GoFiles: []string{"page.go"}}, upstreamModule: {Dir: dir, GoFiles: []string{"result.go"}}}}
	rd, err := g.bodyRecordRootDeclarations(pkg.Path())
	if err != nil || len(rd) != 3 {
		t.Fatal(rd, err)
	}
	pd, err := g.identityPaginationDeclarations(pkg.Path())
	if err != nil || len(pd) != 7 {
		t.Fatal(pd, err)
	}
	decls := subnetPoolBodyPinnedDeclarations(t)
	for key, fn := range rd {
		decls[key] = fn
	}
	for key, fn := range pd {
		decls[key] = fn
	}
	if err := validateSubnetPoolBodyNativeDeclarations(pkg, decls, plan); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{upstreamModule + "/openstack/networking/v2/ports", upstreamModule + "/openstack/keymanager/v1/secrets"} {
		deps, err := g.bodyRecordRootDeclarations(path)
		if err != nil || deps != nil {
			t.Fatal("dependencies leaked", deps, err)
		}
	}
	old, err := g.addressGroupBodyRootDeclarations(upstreamModule + "/openstack/networking/v2/extensions/security/addressgroups")
	if err != nil || len(old) != 2 {
		t.Fatal("Address dependency widened", old, err)
	}
	for name, code := range map[string]string{"missing": "package gophercloud", "duplicate": rootSource + "\nfunc(jt *JSONRFC3339NoZ)UnmarshalJSON(data []byte)error{return nil}", "syntax": "package gophercloud\nfunc {", "changed-time-layout": strings.Replace(rootSource, "2006-01-02T15:04:05", "2006-01-02", 1), "missing-time-layout": strings.Replace(rootSource, `const RFC3339NoZ="2006-01-02T15:04:05"`, "", 1), "duplicate-time-layout": rootSource + `\nconst RFC3339NoZ="2006-01-02T15:04:05"`} {
		t.Run(name, func(t *testing.T) {
			code = strings.ReplaceAll(code, `\n`, "\n")
			if err := os.WriteFile(root, []byte(code), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := g.bodyRecordRootDeclarations(pkg.Path()); err == nil {
				t.Fatal("invalid extraction dependency accepted")
			}
		})
	}
	if err := os.WriteFile(root, []byte(rootSource), 0600); err != nil {
		t.Fatal(err)
	}
	for name, pair := range map[string][2]string{"UseNumber": {"dec.UseNumber()", ""}, "native-codes": {"[]int{200, 204, 300}", "[]int{200}"}, "IsEmpty-before-handler": {"empty, err := currentPage.IsEmpty()", "empty, err := false, error(nil)"}, "next-after-handler": {"currentURL, err = currentPage.NextPageURL()", `currentURL, err = "", error(nil)`}} {
		t.Run(name, func(t *testing.T) {
			changed := strings.Replace(pageSource, pair[0], pair[1], 1)
			if changed == pageSource {
				t.Fatal("mutation absent")
			}
			if err := os.WriteFile(page, []byte(changed), 0600); err != nil {
				t.Fatal(err)
			}
			deps, err := g.identityPaginationDeclarations(pkg.Path())
			if err != nil {
				t.Fatal(err)
			}
			bad := subnetPoolBodyPinnedDeclarations(t)
			for key, fn := range deps {
				bad[key] = fn
			}
			if err := validateSubnetPoolBodyNativeDeclarations(pkg, bad, plan); err == nil {
				t.Fatal("pager contract drift ignored")
			}
		})
	}
	for _, missing := range []string{"NewPager", "Request", "Pager.EachPage", "Pager.fetchNextPage"} {
		t.Run("missing-"+missing, func(t *testing.T) {
			bad := subnetPoolBodyPinnedDeclarations(t)
			delete(bad, "pagination."+missing)
			if err := validateSubnetPoolBodyNativeDeclarations(pkg, bad, plan); err == nil {
				t.Fatal("absent page declaration accepted")
			}
		})
	}
}

func TestSubnetPoolBodyFilterAcceptsActualCompiledNativeSchemasAndOwnMethods(t *testing.T) {
	path := os.Getenv("GOPHERCLOUD_METADATA")
	if path == "" {
		path = "/private/tmp/gophercloudsdk-upstream-packages.json"
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
	path = upstreamModule + "/openstack/networking/v2/extensions/subnetpools"
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
	g := generator{meta: meta, subnetPoolPythonFilters: checkedSubnetPoolFilterManifest(t)}
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
	if err := validateSubnetPoolBodyNativeDeclarations(pkg, nativeDecls, plan); err != nil {
		t.Fatal(err)
	}
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	for name, count := range map[string]int{"SubnetPool": 1, "ListOpts": 1, "SubnetPoolPage": 2, "commonResult": 1, "GetResult": 0, "CreateResult": 0, "UpdateResult": 0, "DeleteResult": 0, "CreateOpts": 1, "UpdateOpts": 1, "PrefixesOpsOpts": 1, "PrefixesOpsResult": 1} {
		obj := pkg.Scope().Lookup(name)
		if obj == nil {
			t.Fatal("actual type omitted", name)
		}
		n, ok := obj.Type().(*types.Named)
		if !ok || n.NumMethods() != count {
			t.Fatal("actual own method graph", name, n)
		}
	}
	for _, name := range []string{"ListOptsBuilder", "CreateOptsBuilder", "UpdateOptsBuilder", "PrefixesOpsOptsBuilder"} {
		obj := pkg.Scope().Lookup(name)
		iface, ok := obj.Type().Underlying().(*types.Interface)
		if !ok || iface.NumEmbeddeds() != 0 || iface.NumMethods() != 1 {
			t.Fatal("actual interface graph", name, iface)
		}
	}
	if len(bodyFilterCollectionFields(pkg, plan, 0)) != 10 || !identityCollectionEnabled(pkg, plan, 0) || plan.id != "ID" || plan.name != "Name" || plan.status != "" {
		t.Fatal(plan)
	}
}

const pinnedSubnetPoolAdditionalRootSource = "\nfunc (jt *JSONRFC3339NoZ) UnmarshalJSON(data []byte) error {\n\tvar s string\n\tif err := json.Unmarshal(data, &s); err != nil {\n\t\treturn err\n\t}\n\tif s == \"\" {\n\t\treturn nil\n\t}\n\tt, err := time.Parse(RFC3339NoZ, s)\n\tif err != nil {\n\t\treturn err\n\t}\n\t*jt = JSONRFC3339NoZ(t)\n\treturn nil\n}\n"
const pinnedSubnetPoolAdditionalPagerSource = "\nfunc Request(ctx context.Context, client *gophercloud.ServiceClient, headers map[string]string, url string) (*http.Response, error) {\n\treturn client.Get(ctx, url, nil, &gophercloud.RequestOpts{\n\t\tMoreHeaders:      headers,\n\t\tOkCodes:          []int{200, 204, 300},\n\t\tKeepResponseBody: true,\n\t})\n}\n\nfunc NewPager(client *gophercloud.ServiceClient, initialURL string, createPage func(r PageResult) Page) Pager {\n\treturn Pager{\n\t\tclient:     client,\n\t\tinitialURL: initialURL,\n\t\tcreatePage: createPage,\n\t}\n}\n\nfunc (p Pager) fetchNextPage(ctx context.Context, url string) (Page, error) {\n\tresp, err := Request(ctx, p.client, p.Headers, url)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\tremembered, err := PageResultFrom(resp)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\treturn p.createPage(remembered), nil\n}\n\nfunc (p Pager) EachPage(ctx context.Context, handler func(context.Context, Page) (bool, error)) error {\n\tif p.Err != nil {\n\t\treturn p.Err\n\t}\n\tcurrentURL := p.initialURL\n\tfor {\n\t\tvar currentPage Page\n\n\t\t// if first page has already been fetched, no need to fetch it again\n\t\tif p.firstPage != nil {\n\t\t\tcurrentPage = p.firstPage\n\t\t\tp.firstPage = nil\n\t\t} else {\n\t\t\tvar err error\n\t\t\tcurrentPage, err = p.fetchNextPage(ctx, currentURL)\n\t\t\tif err != nil {\n\t\t\t\treturn err\n\t\t\t}\n\t\t}\n\n\t\tempty, err := currentPage.IsEmpty()\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\tif empty {\n\t\t\treturn nil\n\t\t}\n\n\t\tok, err := handler(ctx, currentPage)\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\tif !ok {\n\t\t\treturn nil\n\t\t}\n\n\t\tcurrentURL, err = currentPage.NextPageURL()\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\tif currentURL == \"\" {\n\t\t\treturn nil\n\t\t}\n\t}\n}\n"
