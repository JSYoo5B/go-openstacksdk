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

func checkedAddressGroupFilterManifest(t *testing.T) *pythonFilterManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../../..", addressGroupFilterManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := decodePythonFilterManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func addressGroupFilterFixtureSource() string {
	spec := identityCollectionSpec{path: addressGroupSDKPath, model: "AddressGroup", getter: "Get", lister: "List"}
	return strings.Replace(identityQueryFixtureSource(spec), "package fixture", "package addressgroups", 1) + "\nfunc Delete(ctx context.Context, client *gophercloud.ServiceClient, id string) error { return nil }\n"
}

func addressGroupFilterNativeFixture(t *testing.T, source string) (*types.Package, *collectionPlan) {
	t.Helper()
	return identityQueryFixture(t, identityCollectionSpec{path: addressGroupSDKPath, model: "AddressGroup", getter: "Get", lister: "List"}, source)
}

func addressGroupBodyPinnedDeclarations(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	result := map[string]*ast.FuncDecl{}
	for _, source := range []string{pinnedAddressGroupBodySource, pinnedAddressGroupRootSource} {
		file, err := parser.ParseFile(token.NewFileSet(), "native.go", source, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			if fn, ok := declaration.(*ast.FuncDecl); ok {
				key := identityDeclarationKey(fn)
				if strings.Contains(source, "package gophercloud") {
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

func TestAddressGroupPythonFilterManifestKeepsQueryAndLocalTenantIdentity(t *testing.T) {
	m := checkedAddressGroupFilterManifest(t)
	if !addressGroupPythonFilterMetadataValid(m) || len(m.Proof.Files) != 5 || len(m.Proof.Nodes) != 16 || len(m.Reserved) != 9 {
		t.Fatal(m)
	}
	if m.Query["tenant_id"] != "" || m.Query["id"] != "" || m.Query["addresses"] != "" || m.Query["project_id"] != "project_id" || m.Body["tenant_id"].Field != "tenant_id" || m.Body["project_id"].Field != "" || m.Body["addresses"].ResponseType == nil || *m.Body["addresses"].ResponseType != "list" {
		t.Fatal(m.Query, m.Body)
	}
	for key, field := range pythonFilterBodyFields(m) {
		if key != field {
			t.Fatal("local raw field inferred", key, field)
		}
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){
		"tenant-not-query-alias":            func(m *pythonFilterManifest) { m.Query["tenant_id"] = "project_id" },
		"native-id-not-server-query":        func(m *pythonFilterManifest) { m.Query["id"] = "id" },
		"native-addresses-not-server-query": func(m *pythonFilterManifest) { m.Query["addresses"] = "addresses" },
		"project-not-local-fallback":        func(m *pythonFilterManifest) { m.Body["tenant_id"] = pythonFilterField{Field: "project_id"} },
		"addresses-response-type":           func(m *pythonFilterManifest) { f := m.Body["addresses"]; f.ResponseType = nil; m.Body["addresses"] = f },
		"id-accessor-not-used":              func(m *pythonFilterManifest) { f := m.Body["id"]; f.ResponseAccessor = "resource_id"; m.Body["id"] = f },
		"class-base":                        func(m *pythonFilterManifest) { m.ClassBases = []string{"_base.NetworkResource"} },
		"MRO":                               func(m *pythonFilterManifest) { m.MRO = m.MRO[1:] },
		"count":                             func(m *pythonFilterManifest) { m.Counts.LocalBody = 1 },
		"URI":                               func(m *pythonFilterManifest) { m.URI["project_id"] = pythonFilterField{Field: "project_id"} },
	} {
		t.Run(name, func(t *testing.T) {
			copy := clonePythonFilterManifest(t, m)
			mutate(copy)
			if addressGroupPythonFilterMetadataValid(copy) {
				t.Fatal("unaudited metadata accepted")
			}
		})
	}
	for _, symbol := range []string{"AddressGroup.project_id", "AddressGroup.tenant_id", "AddressGroup.addresses", "Resource.id", "Resource.__getattribute__", "QueryParameters.__init__"} {
		t.Run(symbol, func(t *testing.T) {
			copy := clonePythonFilterManifest(t, m)
			for i, node := range copy.Proof.Nodes {
				if node.Symbol == symbol {
					copy.Proof.Nodes = append(copy.Proof.Nodes[:i], copy.Proof.Nodes[i+1:]...)
					break
				}
			}
			if addressGroupPythonFilterMetadataValid(copy) {
				t.Fatal("source anchor absent")
			}
		})
	}
}

func TestAddressGroupBodyFilterNativeContractRejectsModelBuilderPagerAndSourceDrift(t *testing.T) {
	source := addressGroupFilterFixtureSource()
	pkg, plan := addressGroupFilterNativeFixture(t, source)
	if !addressGroupBodyNativeSchema(pkg, plan) || len(bodyFilterCollectionSpecs) != 11 || len(identityCollectionSpecs) != 20 || !identityCollectionEnabled(pkg, plan, 0) {
		t.Fatal(plan)
	}
	if err := validateAddressGroupBodyNativeDeclarations(pkg, addressGroupBodyPinnedDeclarations(t), plan); err != nil {
		t.Fatal(err)
	}
	for name, pair := range map[string][2]string{
		"addresses-model":       {"Addresses []string `json:\"addresses\"`", "Addresses []any `json:\"addresses\"`"},
		"project-tag":           {"ProjectID string `json:\"project_id\"`", "ProjectID string `json:\"tenant_id\"`"},
		"extra-model":           {"type AddressGroup struct{", "type AddressGroup struct{TenantID string;"},
		"builder-receiver":      {"func(ListOpts)ToAddressGroupListQuery", "func(*ListOpts)ToAddressGroupListQuery"},
		"native-fields-absent":  {"type ListOpts struct{", "type ListOpts struct{Fields []string `q:\"fields\"`;"},
		"native-opts-addresses": {"Addresses []string `q:\"addresses\"`", "Addresses string `q:\"addresses\"`"},
		"native-marker":         {"Marker string `q:\"marker\"`", "Marker string `q:\"offset\"`"},
		"page-next":             {"func(AddressGroupPage)NextPageURL()(string,error){return \"\",nil}", ""},
		"page-state":            {"AddressGroupPage struct{pagination.LinkedPageBase}", "AddressGroupPage struct{extra bool;pagination.LinkedPageBase}"},
		"extractor-model":       {"([]AddressGroup,error)", "([]string,error)"},
	} {
		t.Run(name, func(t *testing.T) {
			changed := strings.ReplaceAll(source, pair[0], pair[1])
			if changed == source {
				t.Fatal("mutation absent")
			}
			p, pl := addressGroupFilterNativeFixture(t, changed)
			if _, ok := bodyFilterCollectionContract(p, pl, 0); ok {
				t.Fatal("native drift enabled")
			}
			if err := validateBodyFilterCollectionContracts(p, pl); err == nil {
				t.Fatal("capability silently dropped")
			}
		})
	}
	for name, tail := range map[string]string{"custom-decoder": "\nfunc(*AddressGroup)UnmarshalJSON([]byte)error{return nil}", "custom-page-body": "\nfunc(AddressGroupPage)GetBody()any{return nil}", "extra-builder": "\nfunc(ListOpts)ToOtherListQuery()(string,error){return \"\",nil}"} {
		t.Run(name, func(t *testing.T) {
			p, pl := addressGroupFilterNativeFixture(t, source+tail)
			if addressGroupBodyNativeSchema(p, pl) {
				t.Fatal("unreviewed own method accepted")
			}
		})
	}
	for _, drift := range []string{"Href-type", "Rel-tag", "extra-Link-field", "Link-own-method"} {
		t.Run(drift, func(t *testing.T) {
			p, pl := addressGroupFilterNativeFixture(t, source)
			var link *types.Named
			for _, dependency := range p.Imports() {
				if dependency.Path() == upstreamModule {
					link = dependency.Scope().Lookup("Link").Type().(*types.Named)
				}
			}
			fields := []*types.Var{types.NewVar(token.NoPos, nil, "Href", types.Typ[types.String]), types.NewVar(token.NoPos, nil, "Rel", types.Typ[types.String])}
			tags := []string{`json:"href"`, `json:"rel"`}
			switch drift {
			case "Href-type":
				fields[0] = types.NewVar(token.NoPos, nil, "Href", types.Typ[types.Int])
			case "Rel-tag":
				tags[1] = `json:"relationship"`
			case "extra-Link-field":
				fields = append(fields, types.NewVar(token.NoPos, nil, "Extra", types.Typ[types.String]))
				tags = append(tags, `json:"extra"`)
			case "Link-own-method":
				link.AddMethod(types.NewFunc(token.NoPos, link.Obj().Pkg(), "UnmarshalJSON", types.NewSignatureType(types.NewVar(token.NoPos, link.Obj().Pkg(), "r", types.NewPointer(link)), nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "b", types.NewSlice(types.Typ[types.Uint8]))), types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Universe.Lookup("error").Type())), false)))
			}
			link.SetUnderlying(types.NewStruct(fields, tags))
			if addressGroupBodyNativeSchema(p, pl) {
				t.Fatal("native link schema drift accepted")
			}
		})
	}
	for name := range addressGroupBodyNativeDeclarations {
		t.Run(name, func(t *testing.T) {
			declarations := addressGroupBodyPinnedDeclarations(t)
			declarations[name].Body.List = append(declarations[name].Body.List, &ast.ExprStmt{X: ast.NewIdent("drift")})
			if err := validateAddressGroupBodyNativeDeclarations(pkg, declarations, plan); err == nil || !strings.Contains(err.Error(), name) {
				t.Fatal("source guard bypassed", err)
			}
		})
	}
	spec, _ := bodyFilterCollectionContract(pkg, plan, 0)
	for name, mutate := range map[string]func(*bodyFilterCollectionSpec){"typed-projection": func(s *bodyFilterCollectionSpec) { s.rawRecord = false }, "tenant-fallback": func(s *bodyFilterCollectionSpec) { s.fields[2].member = "ProjectID" }, "project-alias": func(s *bodyFilterCollectionSpec) { s.fields[2].aliases = []string{"project_id"} }} {
		t.Run(name, func(t *testing.T) {
			copy := spec
			copy.fields = append([]bodyFilterCollectionField(nil), spec.fields...)
			mutate(&copy)
			if bodyFilterCollectionMetadataValid(copy) {
				t.Fatal("unreviewed projection metadata accepted")
			}
		})
	}
}

func TestAddressGroupSemanticFilterEmissionPreservesRawFieldsAndExistingBindings(t *testing.T) {
	pkg, plan := addressGroupFilterNativeFixture(t, addressGroupFilterFixtureSource())
	m := checkedAddressGroupFilterManifest(t)
	if err := (&generator{}).validatePythonFilterPlan(pkg, plan); err == nil {
		t.Fatal("unverified source enabled")
	}
	g := generator{addressGroupPythonFilters: m}
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	e := emitter{pkg: pkg, imports: map[string]string{}, pythonFilters: g.pythonFilterFor(pkg, plan)}
	e.printf("func(a *API)newResources()*resource.Collection[AddressGroup]{return ")
	emitCollectionAdapter(&e, plan, "a", nil)
	e.printf("}\n")
	emitBodyFilterList(&e, plan)
	output, err := e.source()
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, wanted := range []string{"addressGroupBodyFilterValue(record, key)", "resource.BodyStreamWithControl(ctx, upstream.List(a.client, _opts)", "upstream.ExtractGroups(page)", `"address_groups", control`, `request.Wrap("List", "addressgroups", err)`, "IdentityFind:", "GetIdentityQuery:", "NameQuery:", "Delete:", "return a.listBodyWithControl(ctx, control, options...)"} {
		if !strings.Contains(text, wanted) {
			t.Fatal("native policy lost", wanted, text)
		}
	}
	if strings.Count(text, "config.Query[key] = append([]string(nil), values...)") != 2 {
		t.Fatal("whole map snapshots lost", text)
	}
	for _, forbidden := range []string{`q.Del("status")`, "WithListQuery(", "BodyFilterValue:", "json.Marshal(v.", "Failed:", `"tenant_id": "project_id"`, `"id": "project_id"`} {
		if strings.Contains(text, forbidden) {
			t.Fatal("unsupported inference", forbidden, text)
		}
	}
	record := collectionRecord{BodyFilterFields: bodyFilterCollectionFields(pkg, plan, 0), SemanticQueryFilters: pythonFilterQueryFields(m), SemanticBodyFilters: pythonFilterBodyFields(m), SemanticReserved: pythonFilterReserved(m)}
	if len(record.BodyFilterFields) != 3 || len(record.SemanticQueryFilters) != 8 || len(record.SemanticBodyFilters) != 3 || record.SemanticQueryFilters["tenant_id"] != "" || record.SemanticQueryFilters["addresses"] != "" || record.SemanticBodyFilters["tenant_id"] != "tenant_id" {
		t.Fatal(record)
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 1); ok {
		t.Fatal("global capability leaked into scope")
	}
	for _, identity := range identityCollectionSpecs {
		if identity.path == addressGroupSDKPath {
			continue
		}
		p, pl := identityQueryFixture(t, identity, identityQueryFixtureSource(identity))
		if g.pythonFilterFor(p, pl) != nil {
			t.Fatal("descriptor leaked", identity.path)
		}
	}
	// Compare the source AST of the four already-published raw bindings. Their
	// real Delete signatures are included so these fixtures exercise callbacks.
	for _, resource := range []string{"secrets", "containers", "orders", "subnets"} {
		var p *types.Package
		var pl *collectionPlan
		var manifest *pythonFilterManifest
		var path string
		switch resource {
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
		case "subnets":
			spec := identityCollectionSpec{path: "network/v2/subnets", model: "Subnet", getter: "Get", lister: "List"}
			source := strings.Replace(identityQueryFixtureSource(spec), "package fixture", "package subnets", 1) + "\nfunc Delete(ctx context.Context, client *gophercloud.ServiceClient, id string) error {return nil}\n"
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
				t.Fatal("prior function changed", resource, name)
			}
		}
	}
}

func TestAddressGroupPythonFilterPinnedSourceRequiresLiveAliasAndCoercionProof(t *testing.T) {
	m := checkedAddressGroupFilterManifest(t)
	if err := verifyAddressGroupPythonFilterManifest("", m); err == nil {
		t.Fatal("source optional")
	}
	if err := verifyAddressGroupPythonFilterManifest(t.TempDir(), m); err == nil {
		t.Fatal("missing source accepted")
	}
	source := os.Getenv("OPENSTACKSDK_SOURCE")
	if source == "" {
		source = "/private/tmp/go-openstacksdk-openstacksdk"
	}
	if _, err := os.Stat(filepath.Join(source, "openstack/network/v2/address_group.py")); err != nil {
		t.Skip("audited checkout unavailable")
	}
	if err := verifyAddressGroupPythonFilterManifest(source, m); err != nil {
		t.Fatal(err)
	}
	fresh, err := extractPythonFilterManifestTarget(source, addressGroupPythonResource)
	if err != nil || !reflect.DeepEqual(fresh, m) {
		t.Fatal(fresh, err)
	}
	root := t.TempDir()
	for path := range addressGroupFilterSourceHashes {
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
	for path := range addressGroupFilterSourceHashes {
		t.Run(path, func(t *testing.T) {
			target := filepath.Join(root, path)
			data, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, append(data, []byte("\n# drift\n")...), 0600); err != nil {
				t.Fatal(err)
			}
			if err := verifyAddressGroupPythonFilterManifest(root, m); err == nil || !strings.Contains(err.Error(), path) {
				t.Fatal("SHA guard bypassed", err)
			}
			if err := os.WriteFile(target, data, 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){"parser": func(m *pythonFilterManifest) { m.Proof.PythonParser = "other" }, "reserved": func(m *pythonFilterManifest) { m.Reserved = m.Reserved[:8] }, "AST": func(m *pythonFilterManifest) { m.Proof.Nodes[0].ASTSHA256 = "wrong" }, "controls": func(m *pythonFilterManifest) { m.SourceControls.ResourceList = m.SourceControls.ResourceList[:6] }, "source-proof": func(m *pythonFilterManifest) { m.Proof.Files["../outside.py"] = "wrong" }} {
		t.Run(name, func(t *testing.T) {
			copy := clonePythonFilterManifest(t, m)
			mutate(copy)
			if err := verifyAddressGroupPythonFilterManifest(root, copy); err == nil {
				t.Fatal("manifest replaced live proof")
			}
		})
	}
	loaded, err := loadAddressGroupPythonFilterManifest("../../..", source)
	if err != nil || !reflect.DeepEqual(loaded, m) {
		t.Fatal(loaded, err)
	}
	for target, wanted := range map[string]*pythonFilterManifest{"": checkedPythonFilterManifest(t), secretPythonResource: checkedSecretFilterManifest(t), containerPythonResource: checkedContainerFilterManifest(t), orderPythonResource: checkedOrderFilterManifest(t)} {
		actual, err := extractPythonFilterManifestTarget(source, target)
		if err != nil || !reflect.DeepEqual(actual, wanted) {
			t.Fatal("prior manifest changed", target, err)
		}
	}
}

func TestAddressGroupRawDependencyLoaderPreservesNumbersAndNativeLinkHelpers(t *testing.T) {
	pkg, plan := addressGroupFilterNativeFixture(t, addressGroupFilterFixtureSource())
	for _, other := range []string{upstreamModule + "/openstack/keymanager/v1/secrets", upstreamModule + "/openstack/networking/v2/extensions/qos/policies"} {
		deps, err := (&generator{}).addressGroupBodyRootDeclarations(other)
		if err != nil || deps != nil {
			t.Fatal("root dependency leaked", deps, err)
		}
	}
	if _, err := (&generator{}).addressGroupBodyRootDeclarations(pkg.Path()); err == nil {
		t.Fatal("missing root dependency accepted")
	}
	if _, err := (&generator{}).identityPaginationDeclarations(pkg.Path()); err == nil {
		t.Fatal("missing page dependency accepted")
	}
	dir := t.TempDir()
	page := filepath.Join(dir, "page.go")
	root := filepath.Join(dir, "result.go")
	if err := os.WriteFile(page, []byte(pinnedSubnetBodyIdentitySource), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte(pinnedAddressGroupRootSource), 0600); err != nil {
		t.Fatal(err)
	}
	g := generator{meta: map[string]metadata{upstreamModule + "/pagination": {Dir: dir, GoFiles: []string{"page.go"}}, upstreamModule: {Dir: dir, GoFiles: []string{"result.go"}}}}
	decls := addressGroupBodyPinnedDeclarations(t)
	dependencies, err := g.identityPaginationDeclarations(pkg.Path())
	if err != nil || len(dependencies) != 3 {
		t.Fatal(dependencies, err)
	}
	rootDeps, err := g.addressGroupBodyRootDeclarations(pkg.Path())
	if err != nil || len(rootDeps) != 2 {
		t.Fatal(rootDeps, err)
	}
	for key, fn := range dependencies {
		decls[key] = fn
	}
	for key, fn := range rootDeps {
		decls[key] = fn
	}
	if err := validateAddressGroupBodyNativeDeclarations(pkg, decls, plan); err != nil {
		t.Fatal(err)
	}
	for name, pair := range map[string][2]string{"exact-number": {"dec.UseNumber()", ""}, "root-rel": {`l.Rel=="next"`, `l.Rel=="prev"`}, "root-decoder": {"json.Unmarshal(b,to)", "json.Unmarshal(b,&to)"}} {
		t.Run(name, func(t *testing.T) {
			psource, rsource := pinnedSubnetBodyIdentitySource, pinnedAddressGroupRootSource
			if name == "exact-number" {
				psource = strings.Replace(psource, pair[0], pair[1], 1)
			} else {
				rsource = strings.Replace(rsource, pair[0], pair[1], 1)
			}
			if psource == pinnedSubnetBodyIdentitySource && rsource == pinnedAddressGroupRootSource {
				t.Fatal("dependency mutation absent")
			}
			if err := os.WriteFile(page, []byte(psource), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(root, []byte(rsource), 0600); err != nil {
				t.Fatal(err)
			}
			pd, err := g.identityPaginationDeclarations(pkg.Path())
			if err != nil {
				t.Fatal(err)
			}
			rd, err := g.addressGroupBodyRootDeclarations(pkg.Path())
			if err != nil {
				t.Fatal(err)
			}
			copy := addressGroupBodyPinnedDeclarations(t)
			for key, fn := range pd {
				copy[key] = fn
			}
			for key, fn := range rd {
				copy[key] = fn
			}
			if err := validateAddressGroupBodyNativeDeclarations(pkg, copy, plan); err == nil {
				t.Fatal("dependency drift bypassed")
			}
		})
	}
	for name, code := range map[string]string{"missing": "package gophercloud", "duplicate": pinnedAddressGroupRootSource + "\nfunc ExtractNextURL(links []Link)(string,error){return \"\",nil}", "syntax": "package gophercloud\nfunc {"} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(root, []byte(code), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := g.addressGroupBodyRootDeclarations(pkg.Path()); err == nil {
				t.Fatal("invalid dependency accepted")
			}
		})
	}
}

func TestAddressGroupBodyFilterAcceptsActualCompiledNativeSchemasAndOwnMethods(t *testing.T) {
	path := os.Getenv("GOPHERCLOUD_METADATA")
	if path == "" {
		path = "/private/tmp/go-openstacksdk-upstream-packages.json"
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		t.Skip("compiled pinned metadata unavailable")
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
	path = upstreamModule + "/openstack/networking/v2/extensions/security/addressgroups"
	native, ok := meta[path]
	if !ok || native.Dir == "" || native.Export == "" {
		t.Fatal("source/export missing")
	}
	compiled := importer.ForCompiler(token.NewFileSet(), "gc", func(path string) (io.ReadCloser, error) {
		entry, ok := meta[path]
		if !ok || entry.Export == "" {
			return nil, os.ErrNotExist
		}
		return os.Open(entry.Export)
	})
	// sdkgen imports the root package before leaf packages. Link is used only
	// in the pager body, so importing the leaf's signature graph alone need
	// not populate that exported root-package object in gc export data.
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
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
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
		t.Fatalf("%v; plan=%+v model=%v list=%v imports=%v", err, plan, plan.model, plan.listInput, pkg.Imports())
	}
	g := generator{meta: meta, addressGroupPythonFilters: checkedAddressGroupFilterManifest(t)}
	pd, err := g.identityPaginationDeclarations(path)
	if err != nil {
		t.Fatal(err)
	}
	rd, err := g.addressGroupBodyRootDeclarations(path)
	if err != nil {
		t.Fatal(err)
	}
	for key, fn := range pd {
		nativeDecls[key] = fn
	}
	for key, fn := range rd {
		nativeDecls[key] = fn
	}
	if err := validateAddressGroupBodyNativeDeclarations(pkg, nativeDecls, plan); err != nil {
		t.Fatal(err)
	}
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	for name, count := range map[string]int{"AddressGroup": 0, "ListOpts": 1, "AddressGroupPage": 2, "commonResult": 1, "GetResult": 0, "CreateOpts": 2, "UpdateOpts": 1, "UpdateAddressesOpts": 1, "CreateResult": 0, "DeleteResult": 0, "UpdateResult": 0, "AddAddressesResult": 0, "RemoveAddressesResult": 0} {
		obj := pkg.Scope().Lookup(name)
		if obj == nil {
			t.Fatal("own type omitted", name)
		}
		named, ok := obj.Type().(*types.Named)
		if !ok || named.NumMethods() != count {
			t.Fatal("actual own method count", name, named)
		}
	}
	if plan.id != "ID" || plan.name != "Name" || plan.status != "" || len(bodyFilterCollectionFields(pkg, plan, 0)) != 3 || !identityCollectionEnabled(pkg, plan, 0) {
		t.Fatal(plan)
	}
}

const pinnedAddressGroupBodySource = `package addressgroups
func (opts ListOpts) ToAddressGroupListQuery() (string,error) {q,err:=gophercloud.BuildQueryString(opts);return q.String(),err}
func List(c *gophercloud.ServiceClient,opts ListOptsBuilder) pagination.Pager {url:=rootURL(c);if opts!=nil{query,err:=opts.ToAddressGroupListQuery();if err!=nil{return pagination.Pager{Err:err}};url+=query};return pagination.NewPager(c,url,func(r pagination.PageResult) pagination.Page{return AddressGroupPage{pagination.LinkedPageBase{PageResult:r}}})}
func Get(ctx context.Context,c *gophercloud.ServiceClient,id string)(r GetResult){resp,err:=c.Get(ctx,resourceURL(c,id),&r.Body,nil);_,r.Header,r.Err=gophercloud.ParseResponse(resp,err);return}
func rootURL(c *gophercloud.ServiceClient)string{return c.ServiceURL(rootPath)}
func resourceURL(c *gophercloud.ServiceClient,id string)string{return c.ServiceURL(rootPath,id)}
func(r AddressGroupPage)NextPageURL()(string,error){var s struct{Links []gophercloud.Link ` + "`json:\"address_groups_links\"`" + `};err:=r.ExtractInto(&s);if err!=nil{return "",err};return gophercloud.ExtractNextURL(s.Links)}
func(r AddressGroupPage)IsEmpty()(bool,error){if r.StatusCode==204{return true,nil};is,err:=ExtractGroups(r);return len(is)==0,err}
func ExtractGroups(r pagination.Page)([]AddressGroup,error){var s struct{AddressGroups []AddressGroup ` + "`json:\"address_groups\"`" + `};err:=(r.(AddressGroupPage)).ExtractInto(&s);return s.AddressGroups,err}
func(r commonResult)Extract()(*AddressGroup,error){var s struct{AddressGroup *AddressGroup ` + "`json:\"address_group\"`" + `};err:=r.ExtractInto(&s);return s.AddressGroup,err}
`

const pinnedAddressGroupRootSource = `package gophercloud
func ExtractNextURL(links []Link)(string,error){var url string;for _,l:=range links{if l.Rel=="next"{url=l.Href}};if url==""{return "",nil};return url,nil}
func(r Result)ExtractInto(to any)error{if r.Err!=nil{return r.Err};if reader,ok:=r.Body.(io.Reader);ok{if readCloser,ok:=reader.(io.Closer);ok{defer readCloser.Close()};return json.NewDecoder(reader).Decode(to)};b,err:=json.Marshal(r.Body);if err!=nil{return err};err=json.Unmarshal(b,to);return err}
`
