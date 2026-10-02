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

func checkedRouterFilterManifest(t *testing.T) *pythonFilterManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../../..", routerFilterManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	m, err := decodePythonFilterManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func routerFilterFixtureSource() string {
	spec := identityCollectionSpec{path: routerSDKPath, model: "Router", getter: "Get", lister: "List"}
	return strings.Replace(identityQueryFixtureSource(spec), "package fixture", "package routers", 1) + "\nfunc Delete(ctx context.Context, client *gophercloud.ServiceClient,id string)error{return nil}\n"
}

func routerFilterNativeFixture(t *testing.T, source string) (*types.Package, *collectionPlan) {
	t.Helper()
	return identityQueryFixture(t, identityCollectionSpec{path: routerSDKPath, model: "Router", getter: "Get", lister: "List"}, source)
}

func routerBodyPinnedDeclarations(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	result := map[string]*ast.FuncDecl{}
	for _, source := range []string{pinnedRouterFilterLeafSource, pinnedAddressGroupRootSource + pinnedQoSPolicyAdditionalRootSource + pinnedSubnetPoolAdditionalRootSource} {
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
	for key, fn := range pinnedSubnetBodyIdentityDeclarations(t) {
		if strings.HasPrefix(key, "pagination.") {
			result[key] = fn
		}
	}
	return result
}

func TestRouterPythonFilterManifestKeepsRevisionOverrideQueryAliasesAndNetworkResourceMRO(t *testing.T) {
	m := checkedRouterFilterManifest(t)
	if !routerPythonFilterMetadataValid(m) || len(m.Proof.Files) != 7 || len(m.Proof.Nodes) != 41 || len(m.Reserved) != 9 || m.Query["tenant_id"] != "" || m.Query["revision_number"] != "" || m.Query["status"] != "status" || m.Query["fields"] != "fields" || m.Query["is_ha"] != "ha" || m.Query["is_distributed"] != "distributed" || m.Query["is_admin_state_up"] != "admin_state_up" || m.Body["revision_number"].Field != "revision" {
		t.Fatal(m)
	}
	for name, kind := range map[string]string{"enable_ndp_proxy": "bool", "evpn_vni": "int", "revision_number": "int", "availability_zone_hints": "list", "availability_zones": "list", "routes": "list", "external_gateway_info": "dict"} {
		f := m.Body[name]
		if f.ResponseType == nil || *f.ResponseType != kind {
			t.Fatal(name, f)
		}
	}
	for _, name := range []string{"revision", "revision_number_query", "status", "id", "name", "tags", "project_id", "is_ha", "is_distributed"} {
		if _, ok := m.Body[name]; ok {
			t.Fatal("wire/query property inferred local", name)
		}
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){
		"tenant-not-query":   func(m *pythonFilterManifest) { m.Query["tenant_id"] = "project_id" },
		"revision-not-query": func(m *pythonFilterManifest) { m.Query["revision_number"] = "revision_number" },
		"source-override": func(m *pythonFilterManifest) {
			f := m.Body["revision_number"]
			f.Field = "revision_number"
			m.Body["revision_number"] = f
		},
		"bool-coercion": func(m *pythonFilterManifest) {
			f := m.Body["enable_ndp_proxy"]
			f.ResponseType = nil
			m.Body["enable_ndp_proxy"] = f
		},
		"int-coercion": func(m *pythonFilterManifest) { f := m.Body["evpn_vni"]; f.ResponseType = nil; m.Body["evpn_vni"] = f },
		"dict-coercion": func(m *pythonFilterManifest) {
			f := m.Body["external_gateway_info"]
			f.ResponseType = nil
			m.Body["external_gateway_info"] = f
		},
		"ha-canonical":           func(m *pythonFilterManifest) { delete(m.Query, "is_ha"); m.Query["ha"] = "ha" },
		"NetworkResource-parent": func(m *pythonFilterManifest) { m.ClassBases = []string{"resource.Resource", "_base.TagMixinNetwork"} },
		"protocol-MRO":           func(m *pythonFilterManifest) { m.MRO = m.MRO[:6] },
		"accepted-count":         func(m *pythonFilterManifest) { m.Counts.AcceptedQuery = 18 },
	} {
		t.Run(name, func(t *testing.T) {
			c := clonePythonFilterManifest(t, m)
			mutate(c)
			if routerPythonFilterMetadataValid(c) {
				t.Fatal("metadata drift accepted")
			}
		})
	}
	for _, anchor := range m.Proof.Nodes {
		t.Run(anchor.Symbol, func(t *testing.T) {
			c := clonePythonFilterManifest(t, m)
			for i, n := range c.Proof.Nodes {
				if n.Symbol == anchor.Symbol {
					c.Proof.Nodes = append(c.Proof.Nodes[:i], c.Proof.Nodes[i+1:]...)
					break
				}
			}
			if routerPythonFilterMetadataValid(c) {
				t.Fatal("source anchor missing")
			}
		})
	}
}

func TestRouterBodyFilterNativeContractRejectsModelBuilderDecoderAndSourceDrift(t *testing.T) {
	source := routerFilterFixtureSource()
	pkg, plan := routerFilterNativeFixture(t, source)
	if !routerBodyNativeSchema(pkg, plan) || len(bodyFilterCollectionSpecs) != 11 || len(identityCollectionSpecs) != 20 || !identityCollectionEnabled(pkg, plan, 0) {
		t.Fatal(plan)
	}
	decls := routerBodyPinnedDeclarations(t)
	if len(decls) != 23 || len(routerBodyNativeDeclarations) != 23 {
		t.Fatal("reachable list chain changed", len(decls))
	}
	if err := validateRouterBodyNativeDeclarations(pkg, decls, plan); err != nil {
		t.Fatal(err)
	}
	for name, constants := range map[string]map[string]string{"missing": {}, "other-collection": {"resourcePath": "networks"}} {
		t.Run("URL-constant-"+name, func(t *testing.T) {
			if err := validateIdentityCollectionContracts(pkg, routerBodyPinnedDeclarations(t), plan, nil, constants); err == nil || !strings.Contains(err.Error(), "constant resourcePath") {
				t.Fatal("native router route literal not guarded", err)
			}
		})
	}
	if err := validateIdentityCollectionContracts(pkg, routerBodyPinnedDeclarations(t), plan, nil, map[string]string{"resourcePath": "routers"}); err != nil {
		t.Fatal(err)
	}
	for name, pair := range map[string][2]string{"gateway-field": {"GatewayInfo GatewayInfo", "GatewayInfo string"}, "gateway-SNAT-presence": {"EnableSNAT *bool", "EnableSNAT bool"}, "gateway-addresses": {"ExternalFixedIPs []ExternalFixedIP", "ExternalFixedIPs []string"}, "gateway-nested-tag": {"json:\"ip_address,omitempty\"", "json:\"ip_address\""}, "route-type": {"DestinationCIDR string", "DestinationCIDR int"}, "route-tag": {"json:\"nexthop\"", "json:\"next_hop\""}, "tenant-tag": {"TenantID string `json:\"tenant_id\"`", "TenantID string `json:\"project_id\"`"}, "strict-time": {"CreatedAt time.Time", "CreatedAt string"}, "native-revision-distinct": {"RevisionNumber int `json:\"revision_number\"`", "RevisionNumber int `json:\"revision\"`"}, "extra-model": {"type Router struct{", "type Router struct{Extra string;"}, "decoder-value-receiver": {"func(*Router)UnmarshalJSON", "func(Router)UnmarshalJSON"}, "decoder-type": {"UnmarshalJSON([]byte)", "UnmarshalJSON(string)"}, "builder-pointer": {"func(ListOpts)ToRouterListQuery", "func(*ListOpts)ToRouterListQuery"}, "unproved-native-fields": {"type ListOpts struct{", "type ListOpts struct{Fields []string `q:\"fields\"`;"}, "query-bool-presence": {"AdminStateUp *bool `q:\"admin_state_up\"`", "AdminStateUp bool `q:\"admin_state_up\"`"}, "query-revision-presence": {"RevisionNumber *int `q:\"revision_number\"`", "RevisionNumber int `q:\"revision_number\"`"}, "interface-param": {"interface{ToRouterListQuery()(string,error)}", "interface{ToRouterListQuery(string)(string,error)}"}, "interface-variadic": {"interface{ToRouterListQuery()(string,error)}", "interface{ToRouterListQuery(...string)(string,error)}"}, "interface-result": {"interface{ToRouterListQuery()(string,error)}", "interface{ToRouterListQuery()(int,error)}"}, "interface-extra": {"interface{ToRouterListQuery()(string,error)}", "interface{ToRouterListQuery()(string,error);ToOther()string}"}, "page-own-next": {"func(RouterPage)NextPageURL()(string,error){return \"\",nil}", ""}, "page-state": {"RouterPage struct{pagination.LinkedPageBase}", "RouterPage struct{extra bool;pagination.LinkedPageBase}"}, "extractor-model": {"([]Router,error)", "([]string,error)"}, "common-result-pointer": {"func(commonResult)Extract", "func(*commonResult)Extract"}, "raw-root-embed": {"type commonResult struct{gophercloud.Result}", "type commonResult struct{gophercloud.Result;Extra bool}"}} {
		t.Run(name, func(t *testing.T) {
			changed := strings.ReplaceAll(source, pair[0], pair[1])
			if changed == source {
				t.Fatal("mutation absent")
			}
			p, pl := routerFilterNativeFixture(t, changed)
			if _, ok := bodyFilterCollectionContract(p, pl, 0); ok {
				t.Fatal("native drift enabled")
			}
			if err := validateBodyFilterCollectionContracts(p, pl); err == nil {
				t.Fatal("capability silently dropped")
			}
		})
	}
	for name, tail := range map[string]string{"extra-model-method": "\nfunc(Router)Other(){}", "gateway-decoder": "\nfunc(*GatewayInfo)UnmarshalJSON([]byte)error{return nil}", "route-decoder": "\nfunc(*Route)UnmarshalJSON([]byte)error{return nil}", "address-decoder": "\nfunc(*ExternalFixedIP)UnmarshalJSON([]byte)error{return nil}", "page-body": "\nfunc(RouterPage)GetBody()any{return nil}", "extra-builder": "\nfunc(ListOpts)ToOtherListQuery()(string,error){return \"\",nil}", "get-result-own-extractor": "\nfunc(GetResult)Extract()(*Router,error){return nil,nil}"} {
		t.Run(name, func(t *testing.T) {
			p, pl := routerFilterNativeFixture(t, source+tail)
			if routerBodyNativeSchema(p, pl) {
				t.Fatal("new own method accepted")
			}
		})
	}
	for _, name := range []string{"Link", "JSONRFC3339NoZ"} {
		t.Run("root-own-method-"+name, func(t *testing.T) {
			p, pl := routerFilterNativeFixture(t, source)
			for _, dep := range p.Imports() {
				if dep.Path() == upstreamModule {
					n := dep.Scope().Lookup(name).Type().(*types.Named)
					n.AddMethod(types.NewFunc(token.NoPos, dep, "NewMethod", types.NewSignatureType(types.NewVar(token.NoPos, dep, "", n), nil, nil, types.NewTuple(), types.NewTuple(), false)))
				}
			}
			if routerBodyNativeSchema(p, pl) {
				t.Fatal("root own method drift accepted")
			}
		})
	}
	t.Run("time-wrapper-underlying", func(t *testing.T) {
		p, pl := routerFilterNativeFixture(t, source)
		for _, dep := range p.Imports() {
			if dep.Path() == upstreamModule {
				dep.Scope().Lookup("JSONRFC3339NoZ").Type().(*types.Named).SetUnderlying(types.Typ[types.Int])
			}
		}
		if routerBodyNativeSchema(p, pl) {
			t.Fatal("root time wrapper drift accepted")
		}
	})
	for name := range routerBodyNativeDeclarations {
		t.Run(name, func(t *testing.T) {
			decls := routerBodyPinnedDeclarations(t)
			decls[name].Body.List = append(decls[name].Body.List, &ast.ExprStmt{X: ast.NewIdent("drift")})
			if err := validateRouterBodyNativeDeclarations(pkg, decls, plan); err == nil || !strings.Contains(err.Error(), name) {
				t.Fatal("native body chain guard bypassed", err)
			}
		})
	}
	spec, _ := bodyFilterCollectionContract(pkg, plan, 0)
	for name, mutate := range map[string]func(*bodyFilterCollectionSpec){"typed-projection": func(s *bodyFilterCollectionSpec) { s.rawRecord = false }, "revision-alias-removed": func(s *bodyFilterCollectionSpec) { s.fields[6].aliases = nil }, "unproved-revision-member": func(s *bodyFilterCollectionSpec) { s.fields[6].member = "RevisionNumber" }, "extra-name": func(s *bodyFilterCollectionSpec) {
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

func TestRouterSemanticFilterEmissionKeepsPrivateCollectionAndPriorEightBindings(t *testing.T) {
	pkg, plan := routerFilterNativeFixture(t, routerFilterFixtureSource())
	m := checkedRouterFilterManifest(t)
	if err := (&generator{}).validatePythonFilterPlan(pkg, plan); err == nil {
		t.Fatal("source proof optional")
	}
	g := generator{routerPythonFilters: m}
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	g.root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(g.root, routerSDKPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := g.emitCollection(pkg, plan); err != nil {
		t.Fatal(err)
	}
	output, err := os.ReadFile(filepath.Join(g.root, routerSDKPath, "resources_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, want := range []string{"routerBodyFilterValue(record, key)", "resource.BodyStreamWithControl(ctx, upstream.List(a.client, _opts)", "upstream.ExtractRouters(page)", `"routers", control`, `request.Wrap("List", "routers", err)`, "IdentityFind:", "GetIdentityQuery:", "NameQuery:", "Delete:", `[]string{"routers", id}`, "return a.listBodyWithControl(ctx, control, options...)"} {
		if !strings.Contains(text, want) {
			t.Fatal("native policy lost", want, text)
		}
	}
	if strings.Count(text, "config.Query[key] = append([]string(nil), values...)") != 2 {
		t.Fatal("whole query maps lost", text)
	}
	for _, bad := range []string{`q.Del("status")`, "WithListQuery(", "BodyFilterValue:", "json.Marshal(v.", `"tenant_id": "project_id"`, "IdentityAllProjectsQuery:", "IdentityExtraSpecs:"} {
		if strings.Contains(text, bad) {
			t.Fatal("unsupported inference", bad, text)
		}
	}
	if strings.Contains(text, "ResourceAdapter") || !strings.Contains(text, "resource.NewCollection(resource.Adapter[Router]") {
		t.Fatal("Router invented public adapter", text)
	}
	record := collectionRecord{BodyFilterFields: bodyFilterCollectionFields(pkg, plan, 0), SemanticQueryFilters: pythonFilterQueryFields(m), SemanticBodyFilters: pythonFilterBodyFields(m), SemanticReserved: pythonFilterReserved(m)}
	if len(record.BodyFilterFields) != 10 || len(record.SemanticQueryFilters) != 18 || len(record.SemanticBodyFilters) != 10 || record.SemanticQueryFilters["tenant_id"] != "" || record.SemanticBodyFilters["revision_number"] != "revision" {
		t.Fatal(record)
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 1); ok {
		t.Fatal("global contract became scoped")
	}
	for _, identity := range identityCollectionSpecs {
		if identity.path == routerSDKPath {
			continue
		}
		p, pl := identityQueryFixture(t, identity, identityQueryFixtureSource(identity))
		if g.pythonFilterFor(p, pl) != nil {
			t.Fatal("descriptor leaked", identity.path)
		}
	}
	for _, target := range []string{"secrets", "containers", "orders", "subnets", "addressgroups", "policies", "subnetpools", "networks"} {
		var p *types.Package
		var pl *collectionPlan
		var manifest *pythonFilterManifest
		var path string
		switch target {
		case "networks":
			p, pl = networkFilterNativeFixture(t, networkFilterFixtureSource())
			manifest = checkedNetworkFilterManifest(t)
			path = networkSDKPath
		case "secrets":
			p, pl = secretFilterNativeFixture(t, secretFilterNativeFixtureSource)
			manifest = checkedSecretFilterManifest(t)
			path = "keymanager/v1/secrets"
		case "containers":
			p, pl = containerFilterNativeFixture(t, containerFilterNativeFixtureSource)
			manifest = checkedContainerFilterManifest(t)
			path = "keymanager/v1/containers"
		case "subnetpools":
			p, pl = subnetPoolFilterNativeFixture(t, subnetPoolFilterFixtureSource())
			manifest = checkedSubnetPoolFilterManifest(t)
			path = subnetPoolSDKPath
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
		g := generator{root: t.TempDir(), pythonFilters: checkedPythonFilterManifest(t), secretPythonFilters: checkedSecretFilterManifest(t), containerPythonFilters: checkedContainerFilterManifest(t), orderPythonFilters: checkedOrderFilterManifest(t), addressGroupPythonFilters: checkedAddressGroupFilterManifest(t), qosPolicyPythonFilters: checkedQoSPolicyFilterManifest(t), subnetPoolPythonFilters: checkedSubnetPoolFilterManifest(t), networkPythonFilters: checkedNetworkFilterManifest(t)}
		_ = manifest
		if err := os.MkdirAll(filepath.Join(g.root, path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := g.emitCollection(p, pl); err != nil {
			t.Fatal(err)
		}
		emitted, err := os.ReadFile(filepath.Join(g.root, path, "resources_generated.go"))
		if err != nil {
			t.Fatal(err)
		}
		existing, err := os.ReadFile(filepath.Join("../../..", path, "resources_generated.go"))
		if err != nil {
			t.Fatal(err)
		}
		names := []string{"API.newResources", "API.listBodyWithControl"}
		if path == networkSDKPath {
			names = append(names, "API.ResourceAdapter")
		}
		for _, name := range names {
			if emittedFunctionHash(t, emitted, name) != emittedFunctionHash(t, existing, name) {
				t.Fatal("prior generated function changed", target, name)
			}
		}

	}
}

func TestRouterPythonFilterPinnedSourceRequiresRevisionOverrideAndCompleteClassificationProof(t *testing.T) {
	m := checkedRouterFilterManifest(t)
	if err := verifyRouterPythonFilterManifest("", m); err == nil {
		t.Fatal("source optional")
	}
	if err := verifyRouterPythonFilterManifest(t.TempDir(), m); err == nil {
		t.Fatal("missing source accepted")
	}
	source := os.Getenv("OPENSTACKSDK_SOURCE")
	if source == "" {
		source = "/private/tmp/gophercloudsdk-openstacksdk"
	}
	if _, err := os.Stat(filepath.Join(source, "openstack/network/v2/router.py")); err != nil {
		t.Skip("pinned source unavailable")
	}
	if err := verifyRouterPythonFilterManifest(source, m); err != nil {
		t.Fatal(err)
	}
	fresh, err := extractPythonFilterManifestTarget(source, routerPythonResource)
	if err != nil || !reflect.DeepEqual(fresh, m) {
		t.Fatal(fresh, err)
	}
	root := t.TempDir()
	for path := range routerFilterSourceHashes {
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
	for path := range routerFilterSourceHashes {
		t.Run(path, func(t *testing.T) {
			target := filepath.Join(root, path)
			data, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, append(data, []byte("\n# drift\n")...), 0600); err != nil {
				t.Fatal(err)
			}
			if err := verifyRouterPythonFilterManifest(root, m); err == nil || !strings.Contains(err.Error(), path) {
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
			if err := verifyRouterPythonFilterManifest(root, copy); err == nil {
				t.Fatal("manifest replaced live source")
			}
		})
	}
	loaded, err := loadRouterPythonFilterManifest("../../..", source)
	if err != nil || !reflect.DeepEqual(loaded, m) {
		t.Fatal(loaded, err)
	}
	for target, wanted := range map[string]*pythonFilterManifest{"": checkedPythonFilterManifest(t), secretPythonResource: checkedSecretFilterManifest(t), containerPythonResource: checkedContainerFilterManifest(t), orderPythonResource: checkedOrderFilterManifest(t), addressGroupPythonResource: checkedAddressGroupFilterManifest(t), qosPolicyPythonResource: checkedQoSPolicyFilterManifest(t), subnetPoolPythonResource: checkedSubnetPoolFilterManifest(t), networkPythonResource: checkedNetworkFilterManifest(t)} {
		actual, err := extractPythonFilterManifestTarget(source, target)
		if err != nil || !reflect.DeepEqual(actual, wanted) {
			t.Fatal("previous manifest changed", target, err)
		}
	}
}

func TestRouterRawDependencyLoaderPreservesNativeTimeCodesAndNumbers(t *testing.T) {
	pkg, plan := routerFilterNativeFixture(t, routerFilterFixtureSource())
	if _, err := (&generator{}).bodyRecordRootDeclarations(pkg.Path()); err == nil {
		t.Fatal("missing root dependency accepted")
	}
	if _, err := (&generator{}).identityPaginationDeclarations(pkg.Path()); err == nil {
		t.Fatal("missing page dependency accepted")
	}
	dir := t.TempDir()
	page := filepath.Join(dir, "page.go")
	root := filepath.Join(dir, "result.go")
	rootSource := pinnedAddressGroupRootSource + pinnedQoSPolicyAdditionalRootSource + pinnedSubnetPoolAdditionalRootSource + `\nconst RFC3339NoZ="2006-01-02T15:04:05"\n`
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
	if err != nil || len(rd) != 5 {
		t.Fatal(rd, err)
	}
	pd, err := g.identityPaginationDeclarations(pkg.Path())
	if err != nil || len(pd) != 7 {
		t.Fatal(pd, err)
	}
	decls := routerBodyPinnedDeclarations(t)
	for key, fn := range rd {
		decls[key] = fn
	}
	for key, fn := range pd {
		decls[key] = fn
	}
	if err := validateRouterBodyNativeDeclarations(pkg, decls, plan); err != nil {
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
			bad := routerBodyPinnedDeclarations(t)
			for key, fn := range deps {
				bad[key] = fn
			}
			if err := validateRouterBodyNativeDeclarations(pkg, bad, plan); err == nil {
				t.Fatal("pager contract drift ignored")
			}
		})
	}
	for _, missing := range []string{"NewPager", "Request", "Pager.EachPage", "Pager.fetchNextPage"} {
		t.Run("missing-"+missing, func(t *testing.T) {
			bad := routerBodyPinnedDeclarations(t)
			delete(bad, "pagination."+missing)
			if err := validateRouterBodyNativeDeclarations(pkg, bad, plan); err == nil {
				t.Fatal("absent page declaration accepted")
			}
		})
	}
}

func TestRouterBodyFilterAcceptsActualCompiledNativeSchemasAndOwnMethods(t *testing.T) {
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
	path = upstreamModule + "/openstack/networking/v2/extensions/layer3/routers"
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
	g := generator{meta: meta, routerPythonFilters: checkedRouterFilterManifest(t)}
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
	if err := validateRouterBodyNativeDeclarations(pkg, nativeDecls, plan); err != nil {
		t.Fatal(err)
	}
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	for name, count := range map[string]int{"Router": 1, "ListOpts": 1, "GatewayInfo": 0, "ExternalFixedIP": 0, "Route": 0, "RouterPage": 2, "commonResult": 1, "GetResult": 0, "CreateResult": 0, "UpdateResult": 0, "DeleteResult": 0, "CreateOpts": 1, "UpdateOpts": 1} {
		obj := pkg.Scope().Lookup(name)
		if obj == nil {
			t.Fatal("actual type omitted", name)
		}
		n, ok := obj.Type().(*types.Named)
		if !ok || n.NumMethods() != count {
			t.Fatal("actual own method graph", name, n)
		}
	}
	for _, name := range []string{"ListOptsBuilder", "CreateOptsBuilder", "UpdateOptsBuilder"} {
		obj := pkg.Scope().Lookup(name)
		iface, ok := obj.Type().Underlying().(*types.Interface)
		if !ok || iface.NumEmbeddeds() != 0 || iface.NumMethods() != 1 {
			t.Fatal("actual interface graph", name, iface)
		}
	}
	if len(bodyFilterCollectionFields(pkg, plan, 0)) != 10 || !identityCollectionEnabled(pkg, plan, 0) || plan.id != "ID" || plan.name != "Name" || plan.status != "Status" {
		t.Fatal(plan)
	}
}

const pinnedRouterFilterLeafSource = "package routers\nfunc (opts ListOpts) ToRouterListQuery() (string, error) {\n\tq, err := gophercloud.BuildQueryString(&opts)\n\tif err != nil {\n\t\treturn \"\", err\n\t}\n\treturn q.String(), nil\n}\n\nfunc List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {\n\turl := rootURL(c)\n\tif opts != nil {\n\t\tquery, err := opts.ToRouterListQuery()\n\t\tif err != nil {\n\t\t\treturn pagination.Pager{Err: err}\n\t\t}\n\t\turl += query\n\t}\n\treturn pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {\n\t\treturn RouterPage{pagination.LinkedPageBase{PageResult: r}}\n\t})\n}\n\nfunc Get(ctx context.Context, c *gophercloud.ServiceClient, id string) (r GetResult) {\n\tresp, err := c.Get(ctx, resourceURL(c, id), &r.Body, nil)\n\t_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)\n\treturn\n}\n\nfunc (r *Router) UnmarshalJSON(b []byte) error {\n\ttype tmp Router\n\n\t// Support for older neutron time format\n\tvar s1 struct {\n\t\ttmp\n\t\tCreatedAt gophercloud.JSONRFC3339NoZ `json:\"created_at\"`\n\t\tUpdatedAt gophercloud.JSONRFC3339NoZ `json:\"updated_at\"`\n\t}\n\n\terr := json.Unmarshal(b, &s1)\n\tif err == nil {\n\t\t*r = Router(s1.tmp)\n\t\tr.CreatedAt = time.Time(s1.CreatedAt)\n\t\tr.UpdatedAt = time.Time(s1.UpdatedAt)\n\n\t\treturn nil\n\t}\n\n\t// Support for newer neutron time format\n\tvar s2 struct {\n\t\ttmp\n\t\tCreatedAt time.Time `json:\"created_at\"`\n\t\tUpdatedAt time.Time `json:\"updated_at\"`\n\t}\n\n\terr = json.Unmarshal(b, &s2)\n\tif err != nil {\n\t\treturn err\n\t}\n\n\t*r = Router(s2.tmp)\n\tr.CreatedAt = time.Time(s2.CreatedAt)\n\tr.UpdatedAt = time.Time(s2.UpdatedAt)\n\n\treturn nil\n}\n\nfunc (r RouterPage) NextPageURL() (string, error) {\n\tvar s struct {\n\t\tLinks []gophercloud.Link `json:\"routers_links\"`\n\t}\n\terr := r.ExtractInto(&s)\n\tif err != nil {\n\t\treturn \"\", err\n\t}\n\treturn gophercloud.ExtractNextURL(s.Links)\n}\n\nfunc (r RouterPage) IsEmpty() (bool, error) {\n\tif r.StatusCode == 204 {\n\t\treturn true, nil\n\t}\n\n\tis, err := ExtractRouters(r)\n\treturn len(is) == 0, err\n}\n\nfunc ExtractRouters(r pagination.Page) ([]Router, error) {\n\tvar s []Router\n\terr := ExtractRoutersInto(r, &s)\n\treturn s, err\n}\n\nfunc ExtractRoutersInto(r pagination.Page, v any) error {\n\treturn r.(RouterPage).ExtractIntoSlicePtr(v, \"routers\")\n}\n\nfunc (r commonResult) Extract() (*Router, error) {\n\tvar s struct {\n\t\tRouter *Router `json:\"router\"`\n\t}\n\terr := r.ExtractInto(&s)\n\treturn s.Router, err\n}\n\nfunc rootURL(c *gophercloud.ServiceClient) string {\n\treturn c.ServiceURL(resourcePath)\n}\n\nfunc resourceURL(c *gophercloud.ServiceClient, id string) string {\n\treturn c.ServiceURL(resourcePath, id)\n}\n"
