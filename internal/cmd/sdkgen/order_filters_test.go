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

const pinnedOrderBodySource = "package orders\n\nfunc (opts ListOpts) ToOrderListQuery() (string, error) {\n\tq, err := gophercloud.BuildQueryString(opts)\n\treturn q.String(), err\n}\n\nfunc List(client *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {\n\turl := listURL(client)\n\tif opts != nil {\n\t\tquery, err := opts.ToOrderListQuery()\n\t\tif err != nil {\n\t\t\treturn pagination.Pager{Err: err}\n\t\t}\n\t\turl += query\n\t}\n\treturn pagination.NewPager(client, url, func(r pagination.PageResult) pagination.Page {\n\t\treturn OrderPage{pagination.LinkedPageBase{PageResult: r}}\n\t})\n}\n\nfunc Get(ctx context.Context, client *gophercloud.ServiceClient, id string) (r GetResult) {\n\tresp, err := client.Get(ctx, getURL(client, id), &r.Body, nil)\n\t_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)\n\treturn\n}\n\nfunc (r *Order) UnmarshalJSON(b []byte) error {\n\ttype tmp Order\n\tvar s struct {\n\t\ttmp\n\t\tCreated gophercloud.JSONRFC3339NoZ `json:\"created\"`\n\t\tUpdated gophercloud.JSONRFC3339NoZ `json:\"updated\"`\n\t}\n\terr := json.Unmarshal(b, &s)\n\tif err != nil {\n\t\treturn err\n\t}\n\t*r = Order(s.tmp)\n\tr.Created = time.Time(s.Created)\n\tr.Updated = time.Time(s.Updated)\n\treturn nil\n}\n\nfunc (r *Meta) UnmarshalJSON(b []byte) error {\n\ttype tmp Meta\n\tvar s struct {\n\t\ttmp\n\t\tExpiration gophercloud.JSONRFC3339NoZ `json:\"expiration\"`\n\t}\n\terr := json.Unmarshal(b, &s)\n\tif err != nil {\n\t\treturn err\n\t}\n\t*r = Meta(s.tmp)\n\tr.Expiration = time.Time(s.Expiration)\n\treturn nil\n}\n\nfunc (r OrderPage) IsEmpty() (bool, error) {\n\tif r.StatusCode == 204 {\n\t\treturn true, nil\n\t}\n\torders, err := ExtractOrders(r)\n\treturn len(orders) == 0, err\n}\n\nfunc (r OrderPage) NextPageURL() (string, error) {\n\tvar s struct {\n\t\tNext     string `json:\"next\"`\n\t\tPrevious string `json:\"previous\"`\n\t}\n\terr := r.ExtractInto(&s)\n\tif err != nil {\n\t\treturn \"\", err\n\t}\n\treturn s.Next, err\n}\n\nfunc ExtractOrders(r pagination.Page) ([]Order, error) {\n\tvar s struct {\n\t\tOrders []Order `json:\"orders\"`\n\t}\n\terr := (r.(OrderPage)).ExtractInto(&s)\n\treturn s.Orders, err\n}\n\nfunc (r commonResult) Extract() (*Order, error) {\n\tvar s *Order\n\terr := r.ExtractInto(&s)\n\treturn s, err\n}\n\nfunc listURL(client *gophercloud.ServiceClient) string {\n\treturn client.ServiceURL(\"orders\")\n}\n\nfunc getURL(client *gophercloud.ServiceClient, id string) string {\n\treturn client.ServiceURL(\"orders\", id)\n}\n"

func checkedOrderFilterManifest(t *testing.T) *pythonFilterManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../../..", orderFilterManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := decodePythonFilterManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func orderFilterNativeFixture(t *testing.T, source string) (*types.Package, *collectionPlan) {
	t.Helper()
	pkg, decls := typedCollectionFixture(t, upstreamModule+"/openstack/keymanager/v1/orders", source)
	return pkg, identifyNamedCollection(pkg, decls, extractorsByPage(pkg, decls), "Get", []string{"List"}, "Delete", 0, "OrderRef")
}

func orderBodyPinnedDeclarations(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "native_order.go", pinnedOrderBodySource, 0)
	if err != nil {
		t.Fatal(err)
	}
	result := map[string]*ast.FuncDecl{}
	for _, declaration := range file.Decls {
		if fn, ok := declaration.(*ast.FuncDecl); ok {
			result[identityDeclarationKey(fn)] = fn
		}
	}
	for name, fn := range pinnedSubnetBodyIdentityDeclarations(t) {
		if strings.HasPrefix(name, "pagination.") {
			result[name] = fn
		}
	}
	return result
}

func TestOrderPythonFilterManifestKeepsInheritedQueriesAndTwoReferenceProperties(t *testing.T) {
	m := checkedOrderFilterManifest(t)
	if !orderPythonFilterMetadataValid(m) || len(m.Proof.Nodes) != 25 || len(m.Proof.Files) != 7 || len(m.Reserved) != 9 {
		t.Fatal(m)
	}
	if m.Query["name"] != "" || m.Query["offset"] != "" || m.Body["name"].Field != "name" || m.Body["id"].ResponseAccessor != "resource_id" || m.Body["order_id"].Field != "order_ref" || m.Body["secret_id"].Field != "secret_ref" {
		t.Fatal(m.Query, m.Body)
	}
	for name, canonical := range pythonFilterBodyFields(m) {
		if name != canonical {
			t.Fatal("property collapsed", name, canonical)
		}
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){
		"native-offset-is-not-semantic": func(m *pythonFilterManifest) { m.Query["offset"] = "offset" },
		"name-is-local":                 func(m *pythonFilterManifest) { m.Query["name"] = "name" },
		"literal-id":                    func(m *pythonFilterManifest) { f := m.Body["id"]; f.ResponseAccessor = ""; m.Body["id"] = f },
		"order-formatter":               func(m *pythonFilterManifest) { m.Body["order_id"] = m.Body["order_ref"] },
		"secret-formatter":              func(m *pythonFilterManifest) { m.Body["secret_id"] = m.Body["secret_ref"] },
		"cross-reference": func(m *pythonFilterManifest) {
			f := m.Body["secret_id"]
			f.Field = "order_ref"
			m.Body["secret_id"] = f
		},
		"meta-untyped": func(m *pythonFilterManifest) { f := m.Body["meta"]; f.ResponseType = nil; m.Body["meta"] = f },
		"nested-name":  func(m *pythonFilterManifest) { f := m.Body["name"]; f.Field = "meta.name"; m.Body["name"] = f },
		"wire-date":    func(m *pythonFilterManifest) { m.Body["created"] = m.Body["created_at"]; delete(m.Body, "created_at") },
		"count":        func(m *pythonFilterManifest) { m.Counts.LocalBody = 13 },
		"MRO":          func(m *pythonFilterManifest) { m.MRO = m.MRO[1:] },
	} {
		t.Run(name, func(t *testing.T) {
			copy := clonePythonFilterManifest(t, m)
			mutate(copy)
			if orderPythonFilterMetadataValid(copy) {
				t.Fatal("unaudited metadata accepted")
			}
		})
	}
	for _, symbol := range []string{"Resource._query_mapping", "QueryParameters.__init__", "Order.order_id", "Order.secret_id", "Order.meta"} {
		t.Run(symbol, func(t *testing.T) {
			copy := clonePythonFilterManifest(t, m)
			for i, node := range copy.Proof.Nodes {
				if node.Symbol == symbol {
					copy.Proof.Nodes = append(copy.Proof.Nodes[:i], copy.Proof.Nodes[i+1:]...)
					break
				}
			}
			if orderPythonFilterMetadataValid(copy) {
				t.Fatal("independent source anchor absent")
			}
		})
	}
}

func TestOrderBodyFilterNativeContractRejectsModelMetaDecoderPagerAndSourceDrift(t *testing.T) {
	pkg, plan := orderFilterNativeFixture(t, orderFilterNativeFixtureSource)
	if !orderBodyNativeSchema(pkg, plan) || len(bodyFilterCollectionSpecs) != 11 || len(identityCollectionSpecs) != 20 || identityCollectionEnabled(pkg, plan, 0) {
		t.Fatal(plan)
	}
	if err := validateOrderBodyNativeDeclarations(pkg, orderBodyPinnedDeclarations(t), plan); err != nil {
		t.Fatal(err)
	}
	for name, pair := range map[string][2]string{
		"meta-map":                {"Meta Meta `json:\"meta\"`", "Meta map[string]any `json:\"meta\"`"},
		"meta-bit-string":         {"BitLength int `json:\"bit_length\"`", "BitLength string `json:\"bit_length\"`"},
		"meta-missing-name":       {";Name string `json:\"name\"`", ""},
		"meta-date":               {"Expiration time.Time `json:\"-\"`", "Expiration string `json:\"expiration\"`"},
		"order-date":              {"Created time.Time `json:\"-\"`", "Created string `json:\"created\"`"},
		"incidental-native-field": {"ErrorReason string `json:\"error_reason\"`;", ""},
		"new-top-name":            {"type Order struct{", "type Order struct{Name string;"},
		"order-decoder-missing":   {"func(*Order)UnmarshalJSON([]byte)error{return nil}", ""},
		"meta-decoder-missing":    {"func(*Meta)UnmarshalJSON([]byte)error{return nil}", ""},
		"order-value-decoder":     {"func(*Order)UnmarshalJSON", "func(Order)UnmarshalJSON"},
		"meta-value-decoder":      {"func(*Meta)UnmarshalJSON", "func(Meta)UnmarshalJSON"},
		"decode-input":            {"UnmarshalJSON([]byte)", "UnmarshalJSON(string)"},
		"offset-tag":              {"Offset int `q:\"offset\"`", "Offset int `q:\"marker\"`"},
		"builder-receiver":        {"func(ListOpts)ToOrderListQuery", "func(*ListOpts)ToOrderListQuery"},
		"page-next":               {"func(OrderPage)NextPageURL()(string,error){return \"\",nil}", ""},
		"page-extra-state":        {"OrderPage struct{pagination.LinkedPageBase}", "OrderPage struct{extra bool;pagination.LinkedPageBase}"},
		"extractor-model":         {"([]Order,error)", "([]string,error)"},
	} {
		t.Run(name, func(t *testing.T) {
			source := strings.ReplaceAll(orderFilterNativeFixtureSource, pair[0], pair[1])
			if source == orderFilterNativeFixtureSource {
				t.Fatal("mutation absent")
			}
			p, pl := orderFilterNativeFixture(t, source)
			if _, ok := bodyFilterCollectionContract(p, pl, 0); ok {
				t.Fatal("native drift enabled")
			}
			if err := validateBodyFilterCollectionContracts(p, pl); err == nil {
				t.Fatal("capability silently dropped")
			}
		})
	}
	for name, tail := range map[string]string{"order-extra-method": "\nfunc(*Order)MarshalJSON()([]byte,error){return nil,nil}", "meta-extra-method": "\nfunc(*Meta)MarshalJSON()([]byte,error){return nil,nil}", "page-body-override": "\nfunc(OrderPage)GetBody()any{return nil}", "list-extra-method": "\nfunc(ListOpts)ToOtherListQuery()(string,error){return \"\",nil}"} {
		t.Run(name, func(t *testing.T) {
			p, pl := orderFilterNativeFixture(t, orderFilterNativeFixtureSource+tail)
			if orderBodyNativeSchema(p, pl) {
				t.Fatal("custom method accepted")
			}
		})
	}
	for name := range orderBodyNativeDeclarations {
		t.Run(name, func(t *testing.T) {
			decls := orderBodyPinnedDeclarations(t)
			decls[name].Body.List = append(decls[name].Body.List, &ast.ExprStmt{X: ast.NewIdent("drift")})
			if err := validateOrderBodyNativeDeclarations(pkg, decls, plan); err == nil || !strings.Contains(err.Error(), name) {
				t.Fatal("source drift guard bypassed", err)
			}
		})
	}
	spec, _ := bodyFilterCollectionContract(pkg, plan, 0)
	for name, mutate := range map[string]func(*bodyFilterCollectionSpec){"typed-lane": func(s *bodyFilterCollectionSpec) { s.rawRecord = false }, "raw-ref-collapse": func(s *bodyFilterCollectionSpec) { s.fields[5].key = "order_ref" }, "alias": func(s *bodyFilterCollectionSpec) { s.fields[0].aliases = []string{"created"} }, "typed-member": func(s *bodyFilterCollectionSpec) { s.fields[0].member = "Created" }, "extra": func(s *bodyFilterCollectionSpec) {
		s.fields = append(s.fields, bodyFilterCollectionField{key: "container_ref"})
	}} {
		t.Run(name, func(t *testing.T) {
			copy := spec
			copy.fields = append([]bodyFilterCollectionField(nil), spec.fields...)
			mutate(&copy)
			if bodyFilterCollectionMetadataValid(copy) {
				t.Fatal("unreviewed field metadata accepted")
			}
		})
	}
}

func TestOrderSemanticFilterEmissionKeepsRawPropertiesQuerySnapshotsAndPriorBindings(t *testing.T) {
	pkg, plan := orderFilterNativeFixture(t, orderFilterNativeFixtureSource)
	m := checkedOrderFilterManifest(t)
	if err := (&generator{}).validatePythonFilterPlan(pkg, plan); err == nil {
		t.Fatal("unverified source enabled")
	}
	g := generator{orderPythonFilters: m}
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	e := emitter{pkg: pkg, imports: map[string]string{}, pythonFilters: g.pythonFilterFor(pkg, plan)}
	e.printf("func(a *API)newResources()*resource.Collection[Order]{return ")
	emitCollectionAdapter(&e, plan, "a", nil)
	e.printf("}\n")
	emitBodyFilterList(&e, plan)
	source, err := e.source()
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, wanted := range []string{"orderBodyFilterValue(record, key)", "resource.BodyStreamWithControl(ctx, upstream.List(a.client, _opts)", "upstream.ExtractOrders(page)", `"orders", control`, "url.Parse(v.OrderRef)", "Status:", "Failed:", "Delete:", "return a.listWithControl(ctx, control, options...)", "return a.listBodyWithControl(ctx, control, options...)"} {
		if !strings.Contains(text, wanted) {
			t.Fatal("native policy lost", wanted, text)
		}
	}
	if strings.Count(text, "config.Query[key] = append([]string(nil), values...)") != 2 || strings.Count(text, `q.Del("status")`) != 2 {
		t.Fatal("both query lanes not preserved", text)
	}
	for _, forbidden := range []string{"Name:", "NameQuery:", "IdentityFind:", "GetIdentityQuery:", "WithListQuery(", "BodyFilterValue:", "reflect.", "json.Marshal(v.", `"order_id": "order_ref"`, `"secret_id": "secret_ref"`, `"created_at": "created"`} {
		if strings.Contains(text, forbidden) {
			t.Fatal("unsupported inference", forbidden, text)
		}
	}
	record := collectionRecord{BodyFilterFields: bodyFilterCollectionFields(pkg, plan, 0), SemanticQueryFilters: pythonFilterQueryFields(m), SemanticBodyFilters: pythonFilterBodyFields(m), SemanticReserved: pythonFilterReserved(m)}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if len(record.BodyFilterFields) != 14 || len(record.SemanticQueryFilters) != 2 || len(record.SemanticBodyFilters) != 14 || record.SemanticBodyFilters["name"] != "name" || strings.Contains(string(data), `"identity_find":true`) {
		t.Fatal(record, string(data))
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 1); ok {
		t.Fatal("global Body contract leaked into scope")
	}
	for _, identity := range identityCollectionSpecs {
		p, pl := identityQueryFixture(t, identity, identityQueryFixtureSource(identity))
		if g.pythonFilterFor(p, pl) != nil {
			t.Fatal("Order descriptor leaked", identity.path)
		}
	}
	for _, resource := range []string{"secrets", "containers", "subnets"} {
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
		case "subnets":
			spec := identityCollectionSpec{path: "network/v2/subnets", model: "Subnet", getter: "Get", lister: "List"}
			source := strings.Replace(identityQueryFixtureSource(spec), "package fixture", "package subnets", 1)
			source += "\nfunc Delete(ctx context.Context, client *gophercloud.ServiceClient, id string) error { return nil }\n"
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
				t.Fatal("existing generated function changed", resource, name)
			}
		}
	}
}

func TestOrderPythonFilterPinnedSourceRequiresDualFormatterDictAndInheritedDefaultProof(t *testing.T) {
	m := checkedOrderFilterManifest(t)
	if err := verifyOrderPythonFilterManifest("", m); err == nil {
		t.Fatal("source optional")
	}
	if err := verifyOrderPythonFilterManifest(t.TempDir(), m); err == nil {
		t.Fatal("missing source accepted")
	}
	source := os.Getenv("OPENSTACKSDK_SOURCE")
	if source == "" {
		source = "/private/tmp/go-openstacksdk-openstacksdk"
	}
	if _, err := os.Stat(filepath.Join(source, "openstack/key_manager/v1/order.py")); err != nil {
		t.Skip("audited source checkout unavailable")
	}
	if err := verifyOrderPythonFilterManifest(source, m); err != nil {
		t.Fatal(err)
	}
	fresh, err := extractPythonFilterManifestTarget(source, orderPythonResource)
	if err != nil || !reflect.DeepEqual(fresh, m) {
		t.Fatal(fresh, err)
	}
	root := t.TempDir()
	for path := range orderFilterSourceHashes {
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
	for path := range orderFilterSourceHashes {
		t.Run(path, func(t *testing.T) {
			target := filepath.Join(root, path)
			data, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, append(data, []byte("\n# drift\n")...), 0600); err != nil {
				t.Fatal(err)
			}
			if err := verifyOrderPythonFilterManifest(root, m); err == nil || !strings.Contains(err.Error(), path) {
				t.Fatal("SHA guard bypassed", err)
			}
			if err := os.WriteFile(target, data, 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){"proof": func(m *pythonFilterManifest) { m.Proof.Nodes[2].ASTSHA256 = "wrong" }, "parser": func(m *pythonFilterManifest) { m.Proof.PythonParser = "other" }, "reserved": func(m *pythonFilterManifest) { m.Reserved = m.Reserved[:8] }, "controls": func(m *pythonFilterManifest) { m.SourceControls.ResourceList = m.SourceControls.ResourceList[:6] }, "source-path-escape": func(m *pythonFilterManifest) { m.Proof.Files["../outside.py"] = "wrong" }} {
		t.Run(name, func(t *testing.T) {
			copy := clonePythonFilterManifest(t, m)
			mutate(copy)
			if err := verifyOrderPythonFilterManifest(root, copy); err == nil {
				t.Fatal("manifest replaced live proof")
			}
		})
	}
	loaded, err := loadOrderPythonFilterManifest("../../..", source)
	if err != nil || !reflect.DeepEqual(loaded, m) {
		t.Fatal(loaded, err)
	}
	for resource, wanted := range map[string]*pythonFilterManifest{"": checkedPythonFilterManifest(t), secretPythonResource: checkedSecretFilterManifest(t), containerPythonResource: checkedContainerFilterManifest(t)} {
		actual, err := extractPythonFilterManifestTarget(source, resource)
		if err != nil || !reflect.DeepEqual(actual, wanted) {
			t.Fatal("existing manifest changed", resource, err)
		}
	}
}

func TestOrderRawPageDependencyLoaderPreservesExactNumbersAndOriginalRows(t *testing.T) {
	pkg, plan := orderFilterNativeFixture(t, orderFilterNativeFixtureSource)
	if _, err := (&generator{}).identityPaginationDeclarations(pkg.Path()); err == nil || !strings.Contains(err.Error(), "Order body collection") {
		t.Fatal("missing dependency accepted", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "page.go")
	if err := os.WriteFile(path, []byte(pinnedSubnetBodyIdentitySource), 0600); err != nil {
		t.Fatal(err)
	}
	g := generator{meta: map[string]metadata{upstreamModule + "/pagination": {Dir: dir, GoFiles: []string{"page.go"}}}}
	dependencies, err := g.identityPaginationDeclarations(pkg.Path())
	if err != nil || len(dependencies) != 3 {
		t.Fatal(dependencies, err)
	}
	decls := orderBodyPinnedDeclarations(t)
	for name, fn := range dependencies {
		decls[name] = fn
	}
	if err := validateOrderBodyNativeDeclarations(pkg, decls, plan); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(pinnedSubnetBodyIdentitySource, "dec.UseNumber()", "", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	dependencies, err = g.identityPaginationDeclarations(pkg.Path())
	if err != nil {
		t.Fatal(err)
	}
	for name, fn := range dependencies {
		decls[name] = fn
	}
	if err := validateOrderBodyNativeDeclarations(pkg, decls, plan); err == nil || !strings.Contains(err.Error(), "pagination.PageResultFrom changed") {
		t.Fatal("precision guard bypassed", err)
	}
}

func TestOrderBodyFilterAcceptsActualCompiledNativeSchemasAndOwnMethods(t *testing.T) {
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
	path = upstreamModule + "/openstack/keymanager/v1/orders"
	native, ok := meta[path]
	if !ok || native.Dir == "" || native.Export == "" {
		t.Fatal("Order source/export missing")
	}
	compiled := importer.ForCompiler(token.NewFileSet(), "gc", func(path string) (io.ReadCloser, error) {
		entry, ok := meta[path]
		if !ok || entry.Export == "" {
			return nil, os.ErrNotExist
		}
		return os.Open(entry.Export)
	})
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
		t.Fatal(err)
	}
	g := generator{meta: meta, orderPythonFilters: checkedOrderFilterManifest(t)}
	dependencies, err := g.identityPaginationDeclarations(path)
	if err != nil {
		t.Fatal(err)
	}
	for name, fn := range dependencies {
		nativeDecls[name] = fn
	}
	if err := validateOrderBodyNativeDeclarations(pkg, nativeDecls, plan); err != nil {
		t.Fatal(err)
	}
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	for name, count := range map[string]int{"Order": 1, "Meta": 1, "ListOpts": 1, "MetaOpts": 0, "CreateOpts": 1, "OrderPage": 2, "commonResult": 1, "GetResult": 0, "CreateResult": 0, "DeleteResult": 0} {
		named, ok := pkg.Scope().Lookup(name).Type().(*types.Named)
		if !ok || named.NumMethods() != count {
			t.Fatal("actual own methods omitted", name, named)
		}
	}
	if plan.id != "OrderRef" || plan.name != "" || len(bodyFilterCollectionFields(pkg, plan, 0)) != 14 || identityCollectionEnabled(pkg, plan, 0) {
		t.Fatal(plan)
	}
}

const orderFilterNativeFixtureSource = `package orders
import "context"
import "time"
import gophercloud "github.com/gophercloud/gophercloud/v2"
import "github.com/gophercloud/gophercloud/v2/pagination"
type Meta struct{Algorithm string ` + "`json:\"algorithm\"`" + `;BitLength int ` + "`json:\"bit_length\"`" + `;Expiration time.Time ` + "`json:\"-\"`" + `;Mode string ` + "`json:\"mode\"`" + `;Name string ` + "`json:\"name\"`" + `;PayloadContentType string ` + "`json:\"payload_content_type\"`" + `}
func(*Meta)UnmarshalJSON([]byte)error{return nil}
type Order struct{ContainerRef string ` + "`json:\"container_ref\"`" + `;Created time.Time ` + "`json:\"-\"`" + `;CreatorID string ` + "`json:\"creator_id\"`" + `;ErrorReason string ` + "`json:\"error_reason\"`" + `;ErrorStatusCode string ` + "`json:\"error_status_code\"`" + `;OrderRef string ` + "`json:\"order_ref\"`" + `;Meta Meta ` + "`json:\"meta\"`" + `;SecretRef string ` + "`json:\"secret_ref\"`" + `;Status string ` + "`json:\"status\"`" + `;SubStatus string ` + "`json:\"sub_status\"`" + `;SubStatusMessage string ` + "`json:\"sub_status_message\"`" + `;Type string ` + "`json:\"type\"`" + `;Updated time.Time ` + "`json:\"-\"`" + `}
func(*Order)UnmarshalJSON([]byte)error{return nil}
type ListOpts struct{Limit int ` + "`q:\"limit\"`" + `;Offset int ` + "`q:\"offset\"`" + `}
type ListOptsBuilder interface{ToOrderListQuery()(string,error)}
func(ListOpts)ToOrderListQuery()(string,error){return "",nil}
type GetResult struct{}
func(GetResult)Extract()(*Order,error){return nil,nil}
type DeleteResult struct{}
func(DeleteResult)ExtractErr()error{return nil}
func Get(context.Context,*gophercloud.ServiceClient,string)GetResult{return GetResult{}}
func Delete(context.Context,*gophercloud.ServiceClient,string)DeleteResult{return DeleteResult{}}
type OrderPage struct{pagination.LinkedPageBase}
func(OrderPage)IsEmpty()(bool,error){return false,nil}
func(OrderPage)NextPageURL()(string,error){return "",nil}
func List(*gophercloud.ServiceClient,ListOptsBuilder)pagination.Pager{_ = OrderPage{};return pagination.Pager{}}
func ExtractOrders(p pagination.Page)([]Order,error){_ = p.(OrderPage);return nil,nil}
`
