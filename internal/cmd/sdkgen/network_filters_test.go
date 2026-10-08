// These tests use source fixtures adapted from Gophercloud v2.15.0.
// Copyright 2012-2013 Rackspace, Inc.
// Copyright Gophercloud authors
// SPDX-License-Identifier: Apache-2.0
//
// Local modifications select and combine upstream declarations, add test stubs,
// and create altered source variants to check generator behavior.
// Upstream scope (github.com/gophercloud/gophercloud/v2):
//   openstack/networking/v2/networks
//   results.go
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

func checkedNetworkFilterManifest(t *testing.T) *pythonFilterManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../../..", networkFilterManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	m, err := decodePythonFilterManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func networkFilterFixtureSource() string {
	spec := identityCollectionSpec{path: networkSDKPath, model: "Network", getter: "Get", lister: "List"}
	return strings.Replace(identityQueryFixtureSource(spec), "package fixture", "package networks", 1) + "\nfunc Delete(ctx context.Context, client *gophercloud.ServiceClient,id string)error{return nil}\n"
}

func networkFilterNativeFixture(t *testing.T, source string) (*types.Package, *collectionPlan) {
	t.Helper()
	return identityQueryFixture(t, identityCollectionSpec{path: networkSDKPath, model: "Network", getter: "Get", lister: "List"}, source)
}

func networkBodyPinnedDeclarations(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	result := map[string]*ast.FuncDecl{}
	for _, source := range []string{pinnedNetworkFilterLeafSource, pinnedAddressGroupRootSource + pinnedQoSPolicyAdditionalRootSource + pinnedNetworkFilterStructPtrSource + pinnedSubnetPoolAdditionalRootSource} {
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

func TestNetworkPythonFilterManifestKeepsQueryAliasesLocalTypesAndNetworkResourceMRO(t *testing.T) {
	m := checkedNetworkFilterManifest(t)
	if !networkPythonFilterMetadataValid(m) || len(m.Proof.Files) != 7 || len(m.Proof.Nodes) != 43 || len(m.Reserved) != 9 || m.Query["tenant_id"] != "" || m.Query["subnet_ids"] != "" || m.Query["revision_number"] != "" || m.Query["status"] != "status" || m.Query["id"] != "id" || m.Query["fields"] != "fields" || m.Query["is_router_external"] != "router:external" || m.Query["provider_network_type"] != "provider:network_type" {
		t.Fatal(m)
	}
	for name, kind := range map[string]string{"is_default": "bool", "pvlan": "bool", "is_vlan_qinq": "bool", "is_vlan_transparent": "bool", "mtu": "int", "revision_number": "int", "availability_zone_hints": "list", "availability_zones": "list", "segments": "list", "subnet_ids": "list"} {
		field := m.Body[name]
		if field.ResponseType == nil || *field.ResponseType != kind {
			t.Fatal(name, field)
		}
	}
	for _, name := range []string{"subnets", "vlan_qinq", "vlan_transparent", "tenant_id", "status", "id", "name", "tags", "is_port_security_enabled", "is_router_external"} {
		if _, ok := m.Body[name]; ok {
			t.Fatal("wire/query/inherited property inferred local", name)
		}
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){
		"tenant-is-not-query":      func(m *pythonFilterManifest) { m.Query["tenant_id"] = "project_id" },
		"subnets-local":            func(m *pythonFilterManifest) { m.Query["subnet_ids"] = "subnets" },
		"no-native-revision-query": func(m *pythonFilterManifest) { m.Query["revision_number"] = "revision_number" },
		"external-client-canonical": func(m *pythonFilterManifest) {
			delete(m.Query, "is_router_external")
			m.Query["router:external"] = "router:external"
		},
		"provider-client-canonical": func(m *pythonFilterManifest) {
			delete(m.Query, "provider_network_type")
			m.Query["provider:network_type"] = "provider:network_type"
		},
		"bool-coercion": func(m *pythonFilterManifest) { f := m.Body["pvlan"]; f.ResponseType = nil; m.Body["pvlan"] = f },
		"int-coercion":  func(m *pythonFilterManifest) { f := m.Body["mtu"]; f.ResponseType = nil; m.Body["mtu"] = f },
		"list-coercion": func(m *pythonFilterManifest) { f := m.Body["segments"]; f.ResponseType = nil; m.Body["segments"] = f },
		"wire-spelling": func(m *pythonFilterManifest) {
			f := m.Body["subnet_ids"]
			delete(m.Body, "subnet_ids")
			m.Body["subnets"] = f
		},
		"NetworkResource-parent": func(m *pythonFilterManifest) { m.ClassBases = []string{"resource.Resource", "_base.TagMixinNetwork"} },
		"protocol-MRO":           func(m *pythonFilterManifest) { m.MRO = m.MRO[:6] },
		"accepted-count":         func(m *pythonFilterManifest) { m.Counts.AcceptedQuery = 23 },
	} {
		t.Run(name, func(t *testing.T) {
			copy := clonePythonFilterManifest(t, m)
			mutate(copy)
			if networkPythonFilterMetadataValid(copy) {
				t.Fatal("unreviewed metadata accepted")
			}
		})
	}
	for _, anchor := range m.Proof.Nodes {
		t.Run(anchor.Symbol, func(t *testing.T) {
			copy := clonePythonFilterManifest(t, m)
			removed := false
			for i, node := range copy.Proof.Nodes {
				if node.Symbol == anchor.Symbol {
					copy.Proof.Nodes = append(copy.Proof.Nodes[:i], copy.Proof.Nodes[i+1:]...)
					removed = true
					break
				}
			}
			if !removed || networkPythonFilterMetadataValid(copy) {
				t.Fatal("independent source anchor missing", removed)
			}
		})
	}
}

func TestNetworkBodyFilterNativeContractRejectsModelBuilderDecoderAndSourceDrift(t *testing.T) {
	source := networkFilterFixtureSource()
	pkg, plan := networkFilterNativeFixture(t, source)
	if !networkBodyNativeSchema(pkg, plan) || len(bodyFilterCollectionSpecs) != 11 || len(identityCollectionSpecs) != 20 || !identityCollectionEnabled(pkg, plan, 0) {
		t.Fatal(plan)
	}
	decls := networkBodyPinnedDeclarations(t)
	if len(decls) != 28 || len(networkBodyNativeDeclarations) != 28 {
		t.Fatal("reachable list chain changed", len(decls))
	}
	if err := validateNetworkBodyNativeDeclarations(pkg, decls, plan); err != nil {
		t.Fatal(err)
	}
	for name, pair := range map[string][2]string{
		"subnets-type": {"Subnets []string", "Subnets []any"}, "tenant-tag": {"TenantID string `json:\"tenant_id\"`", "TenantID string `json:\"project_id\"`"}, "strict-time": {"CreatedAt time.Time", "CreatedAt string"}, "strict-revision": {"RevisionNumber int `json:\"revision_number\"`", "RevisionNumber string `json:\"revision_number\"`"}, "extra-model": {"type Network struct{", "type Network struct{Extra string;"}, "decoder-value-receiver": {"func(*Network)UnmarshalJSON", "func(Network)UnmarshalJSON"}, "decoder-type": {"UnmarshalJSON([]byte)", "UnmarshalJSON(string)"}, "builder-pointer": {"func(ListOpts)ToNetworkListQuery", "func(*ListOpts)ToNetworkListQuery"}, "unproved-native-fields": {"type ListOpts struct{", "type ListOpts struct{Fields []string `q:\"fields\"`;"}, "query-bool-presence": {"Shared *bool `q:\"shared\"`", "Shared bool `q:\"shared\"`"}, "query-revision-presence": {"RevisionNumber *int `q:\"revision_number\"`", "RevisionNumber int `q:\"revision_number\"`"}, "interface-param": {"interface{ToNetworkListQuery()(string,error)}", "interface{ToNetworkListQuery(string)(string,error)}"}, "interface-variadic": {"interface{ToNetworkListQuery()(string,error)}", "interface{ToNetworkListQuery(...string)(string,error)}"}, "interface-result": {"interface{ToNetworkListQuery()(string,error)}", "interface{ToNetworkListQuery()(int,error)}"}, "interface-extra": {"interface{ToNetworkListQuery()(string,error)}", "interface{ToNetworkListQuery()(string,error);ToOther()string}"}, "page-own-next": {"func(NetworkPage)NextPageURL()(string,error){return \"\",nil}", ""}, "page-state": {"NetworkPage struct{pagination.LinkedPageBase}", "NetworkPage struct{extra bool;pagination.LinkedPageBase}"}, "page-resource-key": {`func(NetworkPage)ResourceKey()string{return "networks"}`, `func(NetworkPage)ResourceKey()int{return 0}`}, "extractor-model": {"([]Network,error)", "([]string,error)"}, "common-result-pointer": {"func(commonResult)ExtractInto", "func(*commonResult)ExtractInto"}, "raw-root-embed": {"type commonResult struct{gophercloud.Result}", "type commonResult struct{gophercloud.Result;Extra bool}"},
	} {
		t.Run(name, func(t *testing.T) {
			changed := strings.ReplaceAll(source, pair[0], pair[1])
			if changed == source {
				t.Fatal("mutation absent")
			}
			p, pl := networkFilterNativeFixture(t, changed)
			if _, ok := bodyFilterCollectionContract(p, pl, 0); ok {
				t.Fatal("native drift enabled")
			}
			if err := validateBodyFilterCollectionContracts(p, pl); err == nil {
				t.Fatal("capability silently dropped")
			}
		})
	}
	for name, tail := range map[string]string{"extra-model-method": "\nfunc(Network)Other(){}", "page-body": "\nfunc(NetworkPage)GetBody()any{return nil}", "extra-builder": "\nfunc(ListOpts)ToOtherListQuery()(string,error){return \"\",nil}", "get-result-own-extractor": "\nfunc(GetResult)Extract()(*Network,error){return nil,nil}"} {
		t.Run(name, func(t *testing.T) {
			p, pl := networkFilterNativeFixture(t, source+tail)
			if networkBodyNativeSchema(p, pl) {
				t.Fatal("new own method accepted")
			}
		})
	}
	for _, name := range []string{"Link", "JSONRFC3339NoZ"} {
		t.Run("root-own-method-"+name, func(t *testing.T) {
			p, pl := networkFilterNativeFixture(t, source)
			for _, dep := range p.Imports() {
				if dep.Path() == upstreamModule {
					n := dep.Scope().Lookup(name).Type().(*types.Named)
					n.AddMethod(types.NewFunc(token.NoPos, dep, "NewMethod", types.NewSignatureType(types.NewVar(token.NoPos, dep, "", n), nil, nil, types.NewTuple(), types.NewTuple(), false)))
				}
			}
			if networkBodyNativeSchema(p, pl) {
				t.Fatal("root own method drift accepted")
			}
		})
	}
	t.Run("time-wrapper-underlying", func(t *testing.T) {
		p, pl := networkFilterNativeFixture(t, source)
		for _, dep := range p.Imports() {
			if dep.Path() == upstreamModule {
				dep.Scope().Lookup("JSONRFC3339NoZ").Type().(*types.Named).SetUnderlying(types.Typ[types.Int])
			}
		}
		if networkBodyNativeSchema(p, pl) {
			t.Fatal("root time wrapper drift accepted")
		}
	})
	for name := range networkBodyNativeDeclarations {
		t.Run(name, func(t *testing.T) {
			decls := networkBodyPinnedDeclarations(t)
			decls[name].Body.List = append(decls[name].Body.List, &ast.ExprStmt{X: ast.NewIdent("drift")})
			if err := validateNetworkBodyNativeDeclarations(pkg, decls, plan); err == nil || !strings.Contains(err.Error(), name) {
				t.Fatal("native body chain guard bypassed", err)
			}
		})
	}
	spec, _ := bodyFilterCollectionContract(pkg, plan, 0)
	for name, mutate := range map[string]func(*bodyFilterCollectionSpec){"typed-projection": func(s *bodyFilterCollectionSpec) { s.rawRecord = false }, "subnet-alias-removed": func(s *bodyFilterCollectionSpec) { s.fields[10].aliases = nil }, "unproved-mtu-member": func(s *bodyFilterCollectionSpec) { s.fields[5].member = "ID" }, "extra-name": func(s *bodyFilterCollectionSpec) {
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

func TestNetworkSemanticFilterEmissionUsesFreshResourceAdapterAndPreservesPriorSevenBindings(t *testing.T) {
	pkg, plan := networkFilterNativeFixture(t, networkFilterFixtureSource())
	m := checkedNetworkFilterManifest(t)
	if err := (&generator{}).validatePythonFilterPlan(pkg, plan); err == nil {
		t.Fatal("source proof optional")
	}
	g := generator{networkPythonFilters: m}
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	g.root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(g.root, networkSDKPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := g.emitCollection(pkg, plan); err != nil {
		t.Fatal(err)
	}
	output, err := os.ReadFile(filepath.Join(g.root, networkSDKPath, "resources_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(output)
	for _, want := range []string{"networkBodyFilterValue(record, key)", "resource.BodyStreamWithControl(ctx, upstream.List(a.client, _opts)", "upstream.ExtractNetworks(page)", `"networks", control`, `request.Wrap("List", "networks", err)`, "IdentityFind:", "GetIdentityQuery:", "NameQuery:", "Delete:", `[]string{"networks", id}`, "return a.listBodyWithControl(ctx, control, options...)"} {
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
	file, err := parser.ParseFile(token.NewFileSet(), "output.go", output, 0)
	if err != nil {
		t.Fatal(err)
	}
	adapterFound := false
	freshMaps := 0
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "ResourceAdapter" {
			continue
		}
		adapterFound = true
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if literal, ok := n.(*ast.CompositeLit); ok {
				if _, ok := literal.Type.(*ast.MapType); ok {
					freshMaps++
				}
			}
			return true
		})
	}
	if !adapterFound || freshMaps < 3 || !strings.Contains(text, "return resource.NewCollection(a.ResourceAdapter())") {
		t.Fatal("adapter maps not constructed per call", freshMaps, text)
	}
	record := collectionRecord{BodyFilterFields: bodyFilterCollectionFields(pkg, plan, 0), SemanticQueryFilters: pythonFilterQueryFields(m), SemanticBodyFilters: pythonFilterBodyFields(m), SemanticReserved: pythonFilterReserved(m)}
	if len(record.BodyFilterFields) != 14 || len(record.SemanticQueryFilters) != 23 || len(record.SemanticBodyFilters) != 14 || record.SemanticQueryFilters["tenant_id"] != "" || record.SemanticBodyFilters["subnet_ids"] != "subnets" {
		t.Fatal(record)
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 1); ok {
		t.Fatal("global contract became scoped")
	}
	for _, identity := range identityCollectionSpecs {
		if identity.path == networkSDKPath {
			continue
		}
		p, pl := identityQueryFixture(t, identity, identityQueryFixtureSource(identity))
		if g.pythonFilterFor(p, pl) != nil {
			t.Fatal("descriptor leaked", identity.path)
		}
	}
	for _, target := range []string{"secrets", "containers", "orders", "subnets", "addressgroups", "policies", "subnetpools"} {
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

func TestNetworkPythonFilterPinnedSourceRequiresBooleanDefaultsAndCompleteClassificationProof(t *testing.T) {
	m := checkedNetworkFilterManifest(t)
	if err := verifyNetworkPythonFilterManifest("", m); err == nil {
		t.Fatal("source optional")
	}
	if err := verifyNetworkPythonFilterManifest(t.TempDir(), m); err == nil {
		t.Fatal("missing source accepted")
	}
	source := os.Getenv("OPENSTACKSDK_SOURCE")
	if source == "" {
		source = "/private/tmp/go-openstacksdk-openstacksdk"
	}
	if _, err := os.Stat(filepath.Join(source, "openstack/network/v2/network.py")); err != nil {
		t.Skip("pinned source unavailable")
	}
	if err := verifyNetworkPythonFilterManifest(source, m); err != nil {
		t.Fatal(err)
	}
	fresh, err := extractPythonFilterManifestTarget(source, networkPythonResource)
	if err != nil || !reflect.DeepEqual(fresh, m) {
		t.Fatal(fresh, err)
	}
	root := t.TempDir()
	for path := range networkFilterSourceHashes {
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
	for path := range networkFilterSourceHashes {
		t.Run(path, func(t *testing.T) {
			target := filepath.Join(root, path)
			data, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, append(data, []byte("\n# drift\n")...), 0600); err != nil {
				t.Fatal(err)
			}
			if err := verifyNetworkPythonFilterManifest(root, m); err == nil || !strings.Contains(err.Error(), path) {
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
			if err := verifyNetworkPythonFilterManifest(root, copy); err == nil {
				t.Fatal("manifest replaced live source")
			}
		})
	}
	loaded, err := loadNetworkPythonFilterManifest("../../..", source)
	if err != nil || !reflect.DeepEqual(loaded, m) {
		t.Fatal(loaded, err)
	}
	for target, wanted := range map[string]*pythonFilterManifest{"": checkedPythonFilterManifest(t), secretPythonResource: checkedSecretFilterManifest(t), containerPythonResource: checkedContainerFilterManifest(t), orderPythonResource: checkedOrderFilterManifest(t), addressGroupPythonResource: checkedAddressGroupFilterManifest(t), qosPolicyPythonResource: checkedQoSPolicyFilterManifest(t), subnetPoolPythonResource: checkedSubnetPoolFilterManifest(t)} {
		actual, err := extractPythonFilterManifestTarget(source, target)
		if err != nil || !reflect.DeepEqual(actual, wanted) {
			t.Fatal("previous manifest changed", target, err)
		}
	}
}

func TestNetworkRawDependencyLoaderPreservesNativeTimeCodesAndNumbers(t *testing.T) {
	pkg, plan := networkFilterNativeFixture(t, networkFilterFixtureSource())
	if _, err := (&generator{}).bodyRecordRootDeclarations(pkg.Path()); err == nil {
		t.Fatal("missing root dependency accepted")
	}
	if _, err := (&generator{}).identityPaginationDeclarations(pkg.Path()); err == nil {
		t.Fatal("missing page dependency accepted")
	}
	dir := t.TempDir()
	page := filepath.Join(dir, "page.go")
	root := filepath.Join(dir, "result.go")
	rootSource := pinnedAddressGroupRootSource + pinnedQoSPolicyAdditionalRootSource + pinnedNetworkFilterStructPtrSource + pinnedSubnetPoolAdditionalRootSource + `\nconst RFC3339NoZ="2006-01-02T15:04:05"\n`
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
	if err != nil || len(rd) != 6 {
		t.Fatal(rd, err)
	}
	pd, err := g.identityPaginationDeclarations(pkg.Path())
	if err != nil || len(pd) != 7 {
		t.Fatal(pd, err)
	}
	decls := networkBodyPinnedDeclarations(t)
	for key, fn := range rd {
		decls[key] = fn
	}
	for key, fn := range pd {
		decls[key] = fn
	}
	if err := validateNetworkBodyNativeDeclarations(pkg, decls, plan); err != nil {
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
			bad := networkBodyPinnedDeclarations(t)
			for key, fn := range deps {
				bad[key] = fn
			}
			if err := validateNetworkBodyNativeDeclarations(pkg, bad, plan); err == nil {
				t.Fatal("pager contract drift ignored")
			}
		})
	}
	for _, missing := range []string{"NewPager", "Request", "Pager.EachPage", "Pager.fetchNextPage"} {
		t.Run("missing-"+missing, func(t *testing.T) {
			bad := networkBodyPinnedDeclarations(t)
			delete(bad, "pagination."+missing)
			if err := validateNetworkBodyNativeDeclarations(pkg, bad, plan); err == nil {
				t.Fatal("absent page declaration accepted")
			}
		})
	}
}

func TestNetworkBodyFilterAcceptsActualCompiledNativeSchemasAndOwnMethods(t *testing.T) {
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
	path = upstreamModule + "/openstack/networking/v2/networks"
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
	g := generator{meta: meta, networkPythonFilters: checkedNetworkFilterManifest(t)}
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
	if err := validateNetworkBodyNativeDeclarations(pkg, nativeDecls, plan); err != nil {
		t.Fatal(err)
	}
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	for name, count := range map[string]int{"Network": 1, "ListOpts": 1, "NetworkPage": 3, "commonResult": 2, "GetResult": 0, "CreateResult": 0, "UpdateResult": 0, "DeleteResult": 0, "CreateOpts": 1, "UpdateOpts": 1} {
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
	if len(bodyFilterCollectionFields(pkg, plan, 0)) != 14 || !identityCollectionEnabled(pkg, plan, 0) || plan.id != "ID" || plan.name != "Name" || plan.status != "Status" {
		t.Fatal(plan)
	}
}

const pinnedNetworkFilterLeafSource = "package networks\nfunc (opts ListOpts) ToNetworkListQuery() (string, error) {\n\tq, err := gophercloud.BuildQueryString(opts)\n\treturn q.String(), err\n}\n\nfunc List(c *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {\n\turl := listURL(c)\n\tif opts != nil {\n\t\tquery, err := opts.ToNetworkListQuery()\n\t\tif err != nil {\n\t\t\treturn pagination.Pager{Err: err}\n\t\t}\n\t\turl += query\n\t}\n\treturn pagination.NewPager(c, url, func(r pagination.PageResult) pagination.Page {\n\t\treturn NetworkPage{pagination.LinkedPageBase{PageResult: r}}\n\t})\n}\n\nfunc Get(ctx context.Context, c *gophercloud.ServiceClient, id string) (r GetResult) {\n\tresp, err := c.Get(ctx, getURL(c, id), &r.Body, nil)\n\t_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)\n\treturn\n}\n\nfunc (r commonResult) Extract() (*Network, error) {\n\tvar s Network\n\terr := r.ExtractInto(&s)\n\treturn &s, err\n}\n\nfunc (r commonResult) ExtractInto(v any) error {\n\treturn r.ExtractIntoStructPtr(v, \"network\")\n}\n\nfunc (r *Network) UnmarshalJSON(b []byte) error {\n\ttype tmp Network\n\n\t// Support for older neutron time format\n\tvar s1 struct {\n\t\ttmp\n\t\tCreatedAt gophercloud.JSONRFC3339NoZ `json:\"created_at\"`\n\t\tUpdatedAt gophercloud.JSONRFC3339NoZ `json:\"updated_at\"`\n\t}\n\n\terr := json.Unmarshal(b, &s1)\n\tif err == nil {\n\t\t*r = Network(s1.tmp)\n\t\tr.CreatedAt = time.Time(s1.CreatedAt)\n\t\tr.UpdatedAt = time.Time(s1.UpdatedAt)\n\n\t\treturn nil\n\t}\n\n\t// Support for newer neutron time format\n\tvar s2 struct {\n\t\ttmp\n\t\tCreatedAt time.Time `json:\"created_at\"`\n\t\tUpdatedAt time.Time `json:\"updated_at\"`\n\t}\n\n\terr = json.Unmarshal(b, &s2)\n\tif err != nil {\n\t\treturn err\n\t}\n\n\t*r = Network(s2.tmp)\n\tr.CreatedAt = time.Time(s2.CreatedAt)\n\tr.UpdatedAt = time.Time(s2.UpdatedAt)\n\n\treturn nil\n}\n\nfunc (r NetworkPage) ResourceKey() string {\n\treturn \"networks\"\n}\n\nfunc (r NetworkPage) NextPageURL() (string, error) {\n\tvar s struct {\n\t\tLinks []gophercloud.Link `json:\"networks_links\"`\n\t}\n\terr := r.ExtractInto(&s)\n\tif err != nil {\n\t\treturn \"\", err\n\t}\n\treturn gophercloud.ExtractNextURL(s.Links)\n}\n\nfunc (r NetworkPage) IsEmpty() (bool, error) {\n\tif r.StatusCode == 204 {\n\t\treturn true, nil\n\t}\n\n\tis, err := ExtractNetworks(r)\n\treturn len(is) == 0, err\n}\n\nfunc ExtractNetworks(r pagination.Page) ([]Network, error) {\n\tvar s []Network\n\terr := ExtractNetworksInto(r, &s)\n\treturn s, err\n}\n\nfunc ExtractNetworksInto(r pagination.Page, v any) error {\n\treturn r.(NetworkPage).ExtractIntoSlicePtr(v, \"networks\")\n}\n\nfunc resourceURL(c *gophercloud.ServiceClient, id string) string {\n\treturn c.ServiceURL(\"networks\", id)\n}\n\nfunc rootURL(c *gophercloud.ServiceClient) string {\n\treturn c.ServiceURL(\"networks\")\n}\n\nfunc getURL(c *gophercloud.ServiceClient, id string) string {\n\treturn resourceURL(c, id)\n}\n\nfunc listURL(c *gophercloud.ServiceClient) string {\n\treturn rootURL(c)\n}\n"
const pinnedNetworkFilterStructPtrSource = "\nfunc (r Result) ExtractIntoStructPtr(to any, label string) error {\n\tif r.Err != nil {\n\t\treturn r.Err\n\t}\n\n\tif to == nil {\n\t\treturn fmt.Errorf(\"expected pointer, got %T\", to)\n\t}\n\n\tt := reflect.TypeOf(to)\n\tif k := t.Kind(); k != reflect.Pointer {\n\t\treturn fmt.Errorf(\"expected pointer, got %v\", k)\n\t}\n\n\tif reflect.ValueOf(to).IsNil() {\n\t\treturn fmt.Errorf(\"expected pointer, got %T\", to)\n\t}\n\n\tswitch t.Elem().Kind() {\n\tcase reflect.Struct:\n\t\treturn r.extractIntoPtr(to, label)\n\tdefault:\n\t\treturn fmt.Errorf(\"expected pointer to struct, got: %v\", t)\n\t}\n}\n"
