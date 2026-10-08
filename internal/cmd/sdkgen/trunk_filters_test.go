package main

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/format"
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

func checkedTrunkFilterManifest(t *testing.T) *pythonFilterManifest {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("../../..", trunkFilterManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	m, err := decodePythonFilterManifest(b)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func trunkFilterFixtureSource() string {
	return strings.Replace(identityQueryFixtureSource(identityCollectionSpec{path: trunkSDKPath, model: "Trunk", getter: "Get", lister: "List"}), "package fixture", "package trunks", 1) + "\nfunc Delete(ctx context.Context,client *gophercloud.ServiceClient,id string)error{return nil}\n"
}
func trunkFilterFixture(t *testing.T, source string) (*types.Package, *collectionPlan) {
	t.Helper()
	return identityQueryFixture(t, identityCollectionSpec{path: trunkSDKPath, model: "Trunk", getter: "Get", lister: "List"}, source)
}
func TestTrunkPythonManifestSeparatesQuerySubportsFromIndependentJSONIdentity(t *testing.T) {
	m := checkedTrunkFilterManifest(t)
	if !trunkPythonFilterMetadataValid(m) || len(m.Proof.Files) != 6 || len(m.Proof.Nodes) != 35 || len(m.Reserved) != 9 || m.Query["sub_ports"] != "sub_ports" || m.Query["is_admin_state_up"] != "admin_state_up" || m.Query["project_id"] != "project_id" || m.Query["status"] != "status" || m.Query["any_tags"] != "tags-any" || m.Query["fields"] != "fields" {
		t.Fatal(m)
	}
	for _, key := range []string{"id", "tenant_id"} {
		if m.Body[key].Field != key || m.Body[key].ResponseType != nil {
			t.Fatal(key, m.Body[key])
		}
		if _, ok := m.Query[key]; ok {
			t.Fatal("native-only query inferred", key)
		}
	}
	for _, key := range []string{"sub_ports", "created_at", "updated_at", "revision_number", "name", "status", "project_id", "tags"} {
		if _, ok := m.Body[key]; ok {
			t.Fatal("undeclared non-query Body invented", key)
		}
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){
		"tenant-alias": func(c *pythonFilterManifest) { c.Body["tenant_id"] = pythonFilterField{Field: "project_id"} },
		"subports-local": func(c *pythonFilterManifest) {
			delete(c.Query, "sub_ports")
			c.Body["sub_ports"] = pythonFilterField{Field: "sub_ports"}
		},
		"native-id-query": func(c *pythonFilterManifest) { c.Query["id"] = "id" },
		"admin-wire-canonical": func(c *pythonFilterManifest) {
			delete(c.Query, "is_admin_state_up")
			c.Query["admin_state_up"] = "admin_state_up"
		},
		"NetworkResource-base": func(c *pythonFilterManifest) { c.ClassBases = []string{"resource.Resource", "_base.TagMixinNetwork"} },
		"MRO":                  func(c *pythonFilterManifest) { c.MRO = c.MRO[:5] },
		"accepted-count":       func(c *pythonFilterManifest) { c.Counts.AcceptedQuery = 14 },
		"coercion": func(c *pythonFilterManifest) {
			f := c.Body["id"]
			kind := "str"
			f.ResponseType = &kind
			c.Body["id"] = f
		},
	} {
		t.Run(name, func(t *testing.T) {
			c := clonePythonFilterManifest(t, m)
			mutate(c)
			if trunkPythonFilterMetadataValid(c) {
				t.Fatal("classification drift accepted")
			}
		})
	}
	for _, node := range m.Proof.Nodes {
		t.Run(node.Symbol, func(t *testing.T) {
			c := clonePythonFilterManifest(t, m)
			for i, n := range c.Proof.Nodes {
				if n.Symbol == node.Symbol {
					c.Proof.Nodes = append(c.Proof.Nodes[:i], c.Proof.Nodes[i+1:]...)
					break
				}
			}
			if trunkPythonFilterMetadataValid(c) {
				t.Fatal("source anchor missing")
			}
		})
	}
}
func TestTrunkBodyNativeSchemaGuardsPlainModelsBuilderAndInheritedContinuation(t *testing.T) {
	source := trunkFilterFixtureSource()
	pkg, plan := trunkFilterFixture(t, source)
	if !trunkBodyNativeSchema(pkg, plan) || len(bodyFilterCollectionSpecs) != 11 || len(identityCollectionSpecs) != 20 || !identityCollectionEnabled(pkg, plan, 0) {
		t.Fatal(plan)
	}
	for name, pair := range map[string][2]string{
		"model-id-tag":            {"json:\"id\"", "json:\"trunk_id\""},
		"native-subports":         {"Subports []Subport", "Subports []string"},
		"timestamp-type":          {"CreatedAt time.Time", "CreatedAt string"},
		"revision-native-int":     {"RevisionNumber int `json:", "RevisionNumber string `json:"},
		"subport-segmentation":    {"SegmentationID int", "SegmentationID string"},
		"subport-required":        {"required:\"true\"", "required:\"false\""},
		"query-boolean-presence":  {"AdminStateUp *bool `q:", "AdminStateUp bool `q:"},
		"query-revision-string":   {"RevisionNumber string `q:", "RevisionNumber *int `q:"},
		"native-fields-absent":    {"type ListOpts struct{", "type ListOpts struct{Fields []string `q:\"fields\"`;"},
		"list-value-builder":      {"func(ListOpts)ToTrunkListQuery", "func(*ListOpts)ToTrunkListQuery"},
		"interface-only-result":   {"interface{ToTrunkListQuery()(string,error)}", "interface{ToTrunkListQuery()(int,error)}"},
		"interface-only-params":   {"interface{ToTrunkListQuery()(string,error)}", "interface{ToTrunkListQuery(string)(string,error)}"},
		"interface-only-variadic": {"interface{ToTrunkListQuery()(string,error)}", "interface{ToTrunkListQuery(...string)(string,error)}"},
		"concrete-list":           {"opts ListOptsBuilder)", "opts ListOpts)"},
		"page-extra-state":        {"TrunkPage struct{pagination.LinkedPageBase}", "TrunkPage struct{Extra bool;pagination.LinkedPageBase}"},
		"page-pointer":            {"func(TrunkPage)IsEmpty", "func(*TrunkPage)IsEmpty"},
		"extractor-element":       {"([]Trunk,error)", "([]string,error)"},
		"common-pointer":          {"func(commonResult)Extract", "func(*commonResult)Extract"},
		"common-extra-state":      {"commonResult struct{gophercloud.Result}", "commonResult struct{gophercloud.Result;Extra bool}"},
	} {
		t.Run(name, func(t *testing.T) {
			changed := strings.ReplaceAll(source, pair[0], pair[1])
			if changed == source {
				t.Fatal("mutation absent")
			}
			p, pl := trunkFilterFixture(t, changed)
			if trunkBodyNativeSchema(p, pl) {
				t.Fatal("native graph drift accepted")
			}
			if validateBodyFilterCollectionContracts(p, pl) == nil {
				t.Fatal("audited capability silently dropped")
			}
		})
	}
	for name, tail := range map[string]string{
		"model-decoder":     "\nfunc(*Trunk)UnmarshalJSON([]byte)error{return nil}",
		"nested-decoder":    "\nfunc(*Subport)UnmarshalJSON([]byte)error{return nil}",
		"list-extra-method": "\nfunc(ListOpts)Other(){}",
		"own-next":          "\nfunc(TrunkPage)NextPageURL()(string,error){return \"\",nil}",
		"own-body":          "\nfunc(TrunkPage)GetBody()any{return nil}",
		"get-own-extract":   "\nfunc(GetResult)Extract()(*Trunk,error){return nil,nil}",
	} {
		t.Run(name, func(t *testing.T) {
			p, pl := trunkFilterFixture(t, source+tail)
			if trunkBodyNativeSchema(p, pl) {
				t.Fatal("new own method changed extraction boundary")
			}
		})
	}
}
func trunkAllManifests(t *testing.T) generator {
	t.Helper()
	g := securityGroupAllManifests(t)
	g.trunkPythonFilters = checkedTrunkFilterManifest(t)
	return g
}
func trunkPriorManifests(t *testing.T) map[string]*pythonFilterManifest {
	t.Helper()
	m := securityGroupPriorManifests(t)
	m[securityGroupPythonResource] = checkedSecurityGroupFilterManifest(t)
	return m
}
func trunkPriorFixture(t *testing.T, path string) (*types.Package, *collectionPlan) {
	t.Helper()
	if path == securityGroupSDKPath {
		return securityGroupFilterFixture(t, securityGroupFilterFixtureSource())
	}
	return securityGroupPriorFixture(t, path)
}
func TestTrunkFilterEmissionOwnsBothQueryLanesAndPreservesPriorTenBindings(t *testing.T) {
	pkg, plan := trunkFilterFixture(t, trunkFilterFixtureSource())
	if (&generator{}).validatePythonFilterPlan(pkg, plan) == nil {
		t.Fatal("source proof optional")
	}
	g := trunkAllManifests(t)
	g.root = t.TempDir()
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(g.root, trunkSDKPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := g.emitCollection(pkg, plan); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(g.root, trunkSDKPath, "resources_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, want := range []string{"trunkBodyFilterValue(record, key)", "BodyStreamWithControl(ctx, upstream.List(a.client, _opts)", "upstream.ExtractTrunks", `"trunks", control)`, "config.Query[key] = append([]string(nil), values...)", "IdentityFind:", "GetIdentityQuery:", "Delete:", `[]string{"trunks", id}`, "Status:", "Failed:"} {
		if !strings.Contains(text, want) {
			t.Fatal("owned path missing", want, text)
		}
	}
	if strings.Count(text, "config.Query[key] = append([]string(nil), values...)") != 2 {
		t.Fatal("both iterator snapshots required", text)
	}
	for _, bad := range []string{`q.Del("status")`, "WithListQuery(", "json.Marshal(v.", "ResourceAdapter", "nativefind.IterateSecurity", "IdentityAllProjectsQuery:", "IdentityExtraSpecs:", `"tenant_id": "project_id"`, "BodyFieldInteger", "BodyFieldBoolean"} {
		if strings.Contains(text, bad) {
			t.Fatal("unproved policy emitted", bad, text)
		}
	}
	record := collectionRecord{BodyFilterFields: bodyFilterCollectionFields(pkg, plan, 0), SemanticQueryFilters: pythonFilterQueryFields(g.trunkPythonFilters), SemanticBodyFilters: pythonFilterBodyFields(g.trunkPythonFilters), SemanticReserved: pythonFilterReserved(g.trunkPythonFilters)}
	if len(record.BodyFilterFields) != 2 || len(record.SemanticQueryFilters) != 14 || len(record.SemanticBodyFilters) != 2 || len(record.SemanticReserved) != 9 || record.SemanticBodyFilters["tenant_id"] != "tenant_id" || record.SemanticQueryFilters["sub_ports"] != "sub_ports" {
		t.Fatal(record)
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 1); ok {
		t.Fatal("descriptor escaped unscoped binding")
	}
	for _, spec := range identityCollectionSpecs {
		if spec.path == trunkSDKPath {
			continue
		}
		p, pl := identityQueryFixture(t, spec, identityQueryFixtureSource(spec))
		if (&generator{trunkPythonFilters: g.trunkPythonFilters}).pythonFilterFor(p, pl) != nil {
			t.Fatal("descriptor leaked", spec.path)
		}
	}
	for _, path := range []string{"network/v2/subnets", "keymanager/v1/secrets", "keymanager/v1/containers", "keymanager/v1/orders", addressGroupSDKPath, qosPolicySDKPath, subnetPoolSDKPath, networkSDKPath, routerSDKPath, securityGroupSDKPath} {
		t.Run(path, func(t *testing.T) {
			p, pl := trunkPriorFixture(t, path)
			copy := trunkAllManifests(t)
			copy.root = t.TempDir()
			if err := os.MkdirAll(filepath.Join(copy.root, path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := copy.emitCollection(p, pl); err != nil {
				t.Fatal(err)
			}
			actual, err := os.ReadFile(filepath.Join(copy.root, path, "resources_generated.go"))
			if err != nil {
				t.Fatal(err)
			}
			old, err := os.ReadFile(filepath.Join("../../..", path, "resources_generated.go"))
			if err != nil {
				t.Fatal(err)
			}
			names := []string{"API.newResources", "API.listBodyWithControl"}
			if path == networkSDKPath {
				names = append(names, "API.ResourceAdapter")
			}
			for _, name := range names {
				if emittedFunctionHash(t, actual, name) != emittedFunctionHash(t, old, name) {
					t.Fatal("prior generated AST changed", name)
				}
			}
		})
	}
}
func TestTrunkPythonPinnedSourceRequiresLiveSixSHAAndThirtyFiveASTProof(t *testing.T) {
	m := checkedTrunkFilterManifest(t)
	if verifyTrunkPythonFilterManifest("", m) == nil || verifyTrunkPythonFilterManifest(t.TempDir(), m) == nil {
		t.Fatal("source optional")
	}
	source := os.Getenv("OPENSTACKSDK_SOURCE")
	if source == "" {
		source = "/private/tmp/go-openstacksdk-openstacksdk"
	}
	if _, err := os.Stat(filepath.Join(source, "openstack/network/v2/trunk.py")); err != nil {
		t.Skip("pinned source unavailable")
	}
	if err := verifyTrunkPythonFilterManifest(source, m); err != nil {
		t.Fatal(err)
	}
	fresh, err := extractPythonFilterManifestTarget(source, trunkPythonResource)
	if err != nil || !reflect.DeepEqual(fresh, m) {
		t.Fatal(fresh, err)
	}
	temp := t.TempDir()
	for path := range trunkFilterSourceHashes {
		b, err := os.ReadFile(filepath.Join(source, path))
		if err != nil {
			t.Fatal(err)
		}
		dest := filepath.Join(temp, path)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dest, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for path := range trunkFilterSourceHashes {
		t.Run(path, func(t *testing.T) {
			dest := filepath.Join(temp, path)
			b, err := os.ReadFile(dest)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(dest, append(b, []byte("\n# source drift\n")...), 0600); err != nil {
				t.Fatal(err)
			}
			if err := verifyTrunkPythonFilterManifest(temp, m); err == nil || !strings.Contains(err.Error(), path) {
				t.Fatal("source SHA drift ignored", err)
			}
			if err := os.WriteFile(dest, b, 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){"AST": func(c *pythonFilterManifest) { c.Proof.Nodes[0].ASTSHA256 = "wrong" }, "parser": func(c *pythonFilterManifest) { c.Proof.PythonParser = "wrong" }, "reserved": func(c *pythonFilterManifest) { c.Reserved = c.Reserved[:8] }, "controls": func(c *pythonFilterManifest) { c.SourceControls.ResourceList = c.SourceControls.ResourceList[:6] }, "foreign-proof": func(c *pythonFilterManifest) { c.Proof.Files["../foreign.py"] = "wrong" }} {
		t.Run(name, func(t *testing.T) {
			c := clonePythonFilterManifest(t, m)
			mutate(c)
			if verifyTrunkPythonFilterManifest(temp, c) == nil {
				t.Fatal("source proof mutation accepted")
			}
		})
	}
	loaded, err := loadTrunkPythonFilterManifest("../../..", source)
	if err != nil || !reflect.DeepEqual(loaded, m) {
		t.Fatal(loaded, err)
	}
	for target, expected := range trunkPriorManifests(t) {
		actual, err := extractPythonFilterManifestTarget(source, target)
		if err != nil || !reflect.DeepEqual(actual, expected) {
			t.Fatal("prior manifest changed", target, err)
		}
	}
}
func trunkActualNative(t *testing.T) (generator, *types.Package, *collectionPlan, map[string]*ast.FuncDecl) {
	t.Helper()
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
	imp := importer.ForCompiler(token.NewFileSet(), "gc", func(path string) (io.ReadCloser, error) {
		m, ok := meta[path]
		if !ok || m.Export == "" {
			return nil, os.ErrNotExist
		}
		return os.Open(m.Export)
	})
	if _, err := imp.Import(upstreamModule); err != nil {
		t.Fatal(err)
	}
	path = upstreamModule + "/openstack/networking/v2/extensions/trunks"
	pkg, err := imp.Import(path)
	if err != nil {
		t.Fatal(err)
	}
	decls, native := map[string]*ast.FuncDecl{}, map[string]*ast.FuncDecl{}
	for _, name := range meta[path].GoFiles {
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(meta[path].Dir, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok {
				native[identityDeclarationKey(fn)] = fn
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
	g := generator{meta: meta, importer: imp, trunkPythonFilters: checkedTrunkFilterManifest(t)}
	for _, load := range []func(string) (map[string]*ast.FuncDecl, error){g.bodyRecordRootDeclarations, g.identityPaginationDeclarations} {
		extra, err := load(path)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range extra {
			native[k] = v
		}
	}
	if err := g.trunkBodyLeafDeclarations(path); err != nil {
		t.Fatal(err)
	}
	return g, pkg, plan, native
}
func TestTrunkDependenciesGuardInheritedLinksWholePageNumbersCodesAndRoute(t *testing.T) {
	_, pkg, plan, decls := trunkActualNative(t)
	if len(trunkBodyNativeDeclarations) != 20 {
		t.Fatal("dependency union count")
	}
	if err := validateTrunkBodyNativeDeclarations(pkg, decls, plan); err != nil {
		t.Fatal(err)
	}
	for name := range trunkBodyNativeDeclarations {
		t.Run("hash-"+name, func(t *testing.T) {
			copy := map[string]*ast.FuncDecl{}
			for k, v := range decls {
				copy[k] = v
			}
			fn := *copy[name]
			block := *fn.Body
			block.List = append(append([]ast.Stmt(nil), block.List...), &ast.ExprStmt{X: ast.NewIdent("changed")})
			fn.Body = &block
			copy[name] = &fn
			if validateTrunkBodyNativeDeclarations(pkg, copy, plan) == nil {
				t.Fatal("native hash drift accepted")
			}
			delete(copy, name)
			if validateTrunkBodyNativeDeclarations(pkg, copy, plan) == nil {
				t.Fatal("missing dependency accepted")
			}
		})
	}
	dir := t.TempDir()
	sources := map[string]string{"root.go": "package gophercloud\n" + securityGroupFunctionSource(t, decls, "gophercloud."), "page.go": "package pagination\n" + securityGroupFunctionSource(t, decls, "pagination.")}
	leaf := "package trunks\nconst resourcePath=\"trunks\"\n"
	for key, fn := range decls {
		if strings.HasPrefix(key, "gophercloud.") || strings.HasPrefix(key, "pagination.") {
			continue
		}
		var b bytes.Buffer
		if err := format.Node(&b, token.NewFileSet(), fn); err != nil {
			t.Fatal(err)
		}
		leaf += b.String() + "\n"
	}
	sources["leaf.go"] = leaf
	for name, source := range sources {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	path := pkg.Path()
	g := generator{meta: map[string]metadata{upstreamModule: {Dir: dir, GoFiles: []string{"root.go"}}, upstreamModule + "/pagination": {Dir: dir, GoFiles: []string{"page.go"}}, path: {Dir: dir, GoFiles: []string{"leaf.go"}}}}
	for label, load := range map[string]func(string) (map[string]*ast.FuncDecl, error){"root": g.bodyRecordRootDeclarations, "page": g.identityPaginationDeclarations} {
		got, err := load(path)
		count := map[string]int{"root": 2, "page": 8}[label]
		if err != nil || len(got) != count {
			t.Fatal(label, got, err)
		}
	}
	if err := g.trunkBodyLeafDeclarations(path); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{upstreamModule + "/openstack/networking/v2/ports", upstreamModule + "/openstack/compute/v2/servers"} {
		d, err := g.bodyRecordRootDeclarations(path)
		if err != nil || d != nil {
			t.Fatal("bounded root dependency leaked", path, d, err)
		}
	}
	for name, entry := range map[string]struct {
		file, code string
		load       func() error
	}{
		"route-wrong":     {"leaf.go", strings.Replace(leaf, `resourcePath="trunks"`, `resourcePath="ports"`, 1), func() error { return g.trunkBodyLeafDeclarations(path) }},
		"route-missing":   {"leaf.go", strings.Replace(leaf, `const resourcePath="trunks"`, "", 1), func() error { return g.trunkBodyLeafDeclarations(path) }},
		"route-duplicate": {"leaf.go", leaf + "\nconst resourcePath=\"trunks\"", func() error { return g.trunkBodyLeafDeclarations(path) }},
		"leaf-duplicate":  {"leaf.go", leaf + "\nfunc ExtractTrunks(v any)error{return nil}", func() error { return g.trunkBodyLeafDeclarations(path) }},
		"root-duplicate":  {"root.go", sources["root.go"] + "\nfunc BuildQueryString(v any)string{return \"\"}", func() error { _, err := g.bodyRecordRootDeclarations(path); return err }},
		"pager-duplicate": {"page.go", sources["page.go"] + "\nfunc NewPager(){}", func() error { _, err := g.identityPaginationDeclarations(path); return err }},
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(dir, entry.file), []byte(entry.code), 0600); err != nil {
				t.Fatal(err)
			}
			if err := entry.load(); err == nil {
				t.Fatal("duplicate/missing source accepted")
			}
			if err := os.WriteFile(filepath.Join(dir, entry.file), []byte(sources[entry.file]), 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
	// Trunk has standard time.Time decoding and uses the inherited default links.next.
	for name, pair := range map[string][2]string{"UseNumber": {"dec.UseNumber()", ""}, "native-codes": {"[]int{200, 204, 300}", "[]int{200}"}, "IsEmpty-before-handler": {"empty, err := currentPage.IsEmpty()", "empty, err := false, error(nil)"}, "continuation-after-handler": {"currentURL, err = currentPage.NextPageURL()", "currentURL, err = \"\", error(nil)"}, "inherited-links": {"[]string{\"links\", \"next\"}", "[]string{\"trunks_links\", \"next\"}"}} {
		t.Run(name, func(t *testing.T) {
			source := strings.Replace(sources["page.go"], pair[0], pair[1], 1)
			if source == sources["page.go"] {
				t.Fatal("native mutation absent")
			}
			if err := os.WriteFile(filepath.Join(dir, "page.go"), []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			pd, err := g.identityPaginationDeclarations(path)
			if err != nil {
				t.Fatal(err)
			}
			copy := map[string]*ast.FuncDecl{}
			for k, v := range decls {
				copy[k] = v
			}
			for k, v := range pd {
				copy[k] = v
			}
			if validateTrunkBodyNativeDeclarations(pkg, copy, plan) == nil {
				t.Fatal("native behavior drift ignored")
			}
			if err := os.WriteFile(filepath.Join(dir, "page.go"), []byte(sources["page.go"]), 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, name := range []string{"gophercloud.ExtractNextURL", "gophercloud.Result.ExtractIntoSlicePtr", "gophercloud.Result.ExtractIntoStructPtr", "gophercloud.JSONRFC3339NoZ.UnmarshalJSON"} {
		if _, ok := trunkBodyNativeDeclarations[name]; ok {
			t.Fatal("unreachable dependency invented", name)
		}
	}
}
func TestTrunkBodyFiltersAcceptActualCompiledPlainNestedAndRequestMethodGraph(t *testing.T) {
	g, pkg, plan, decls := trunkActualNative(t)
	if err := validateBodyFilterCollectionContracts(pkg, plan); err != nil {
		t.Fatal(err)
	}
	if err := validateTrunkBodyNativeDeclarations(pkg, decls, plan); err != nil {
		t.Fatal(err)
	}
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	if !plan.listQueryBuilder || plan.id != "ID" || plan.status != "Status" || plan.statusQuery != "status" || len(bodyFilterCollectionFields(pkg, plan, 0)) != 2 || !identityCollectionEnabled(pkg, plan, 0) {
		t.Fatal(plan)
	}
	for name, count := range map[string]int{"Trunk": 0, "Subport": 0, "ListOpts": 1, "TrunkPage": 1, "commonResult": 1, "GetResult": 0, "CreateResult": 0, "UpdateResult": 0, "DeleteResult": 0, "GetSubportsResult": 1, "UpdateSubportsResult": 1, "CreateOpts": 1, "UpdateOpts": 1, "AddSubportsOpts": 1, "RemoveSubportsOpts": 1, "RemoveSubport": 0} {
		obj := pkg.Scope().Lookup(name)
		if obj == nil {
			t.Fatal("compiled type missing", name)
		}
		n, ok := obj.Type().(*types.Named)
		if !ok || n.NumMethods() != count {
			t.Fatal("actual own method graph", name, n)
		}
	}
	for _, name := range []string{"ListOptsBuilder", "CreateOptsBuilder", "UpdateOptsBuilder", "AddSubportsOptsBuilder", "RemoveSubportsOptsBuilder"} {
		obj := pkg.Scope().Lookup(name)
		if obj == nil {
			t.Fatal(name)
		}
		iface, ok := obj.Type().Underlying().(*types.Interface)
		if !ok || iface.NumEmbeddeds() != 0 || iface.NumMethods() != 1 {
			t.Fatal("actual request-only graph", name, iface)
		}
	}
	page := pkg.Scope().Lookup("TrunkPage").Type()
	obj, index, indirect := types.LookupFieldOrMethod(page, false, pkg, "NextPageURL")
	if obj == nil || len(index) < 2 || indirect {
		t.Fatal("inherited value continuation missing", obj, index, indirect)
	}
	for _, name := range []string{"Trunk", "Subport"} {
		obj, _, _ := types.LookupFieldOrMethod(types.NewPointer(pkg.Scope().Lookup(name).Type()), false, pkg, "UnmarshalJSON")
		if obj != nil {
			t.Fatal("plain native model unexpectedly decoder-controlled", name)
		}
	}
}
