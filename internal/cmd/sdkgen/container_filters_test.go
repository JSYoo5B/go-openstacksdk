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

func checkedContainerFilterManifest(t *testing.T) *pythonFilterManifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../../..", containerFilterManifestPath))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := decodePythonFilterManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func containerFilterNativeFixture(t *testing.T, source string) (*types.Package, *collectionPlan) {
	t.Helper()
	return identityQueryFixture(t, identityCollectionSpec{path: "keymanager/v1/containers", model: "Container", getter: "Get", lister: "List"}, source)
}

func containerBodyPinnedDeclarations(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "native_container.go", pinnedContainerBodySource, 0)
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

func TestContainerPythonFilterManifestKeepsInheritedQueriesAndPropertyAccessors(t *testing.T) {
	m := checkedContainerFilterManifest(t)
	if !containerPythonFilterMetadataValid(m) || len(m.Proof.Nodes) != 23 || len(m.Proof.Files) != 7 || len(m.Reserved) != 9 {
		t.Fatal(m)
	}
	if m.Query["name"] != "" || m.Query["offset"] != "" || m.Body["name"].Field != "name" || m.Body["id"].ResponseAccessor != "resource_id" || m.Body["container_id"].Formatter != secretPythonFormatter {
		t.Fatal(m.Query, m.Body)
	}
	for name, canonical := range pythonFilterBodyFields(m) {
		if name != canonical {
			t.Fatal("distinct property accessor collapsed", name, canonical)
		}
	}
	for _, data := range []string{`{"body":{"container_id":{"field":"container_ref","formatter":false}}}`, `{"body":{"id":{"field":"id","extra":true}}}`, `{} {}`} {
		if _, err := decodePythonFilterManifest([]byte(data)); err == nil {
			t.Fatal("unknown/ill-typed schema accepted", data)
		}
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){
		"native-name-query-is-not-python-query":   func(m *pythonFilterManifest) { m.Query["name"] = "name" },
		"native-offset-query-is-not-python-query": func(m *pythonFilterManifest) { m.Query["offset"] = "offset" },
		"default-limit-lost":                      func(m *pythonFilterManifest) { delete(m.Query, "limit") },
		"literal-id-lost":                         func(m *pythonFilterManifest) { f := m.Body["id"]; f.ResponseAccessor = ""; m.Body["id"] = f },
		"formatted-id-collapse":                   func(m *pythonFilterManifest) { m.Body["container_id"] = m.Body["container_ref"] },
		"arrays-untyped":                          func(m *pythonFilterManifest) { f := m.Body["consumers"]; f.ResponseType = nil; m.Body["consumers"] = f },
		"date-wire-key":                           func(m *pythonFilterManifest) { m.Body["created"] = m.Body["created_at"]; delete(m.Body, "created_at") },
		"inherited-mapping-proof-lost": func(m *pythonFilterManifest) {
			for i, a := range m.Proof.Nodes {
				if a.Symbol == "Resource._query_mapping" {
					m.Proof.Nodes = append(m.Proof.Nodes[:i], m.Proof.Nodes[i+1:]...)
					break
				}
			}
		},
		"pagination-default-proof-lost": func(m *pythonFilterManifest) {
			for i, a := range m.Proof.Nodes {
				if a.Symbol == "QueryParameters.__init__" {
					m.Proof.Nodes[i].Symbol = "Container._query_mapping"
					break
				}
			}
		},
		"MRO":            func(m *pythonFilterManifest) { m.MRO = m.MRO[1:] },
		"unknown-policy": func(m *pythonFilterManifest) { m.UnknownFilters = "wire" },
	} {
		t.Run(name, func(t *testing.T) {
			changed := clonePythonFilterManifest(t, m)
			mutate(changed)
			if containerPythonFilterMetadataValid(changed) {
				t.Fatal("unaudited classification accepted")
			}
		})
	}
}

func TestContainerBodyFilterNativeContractRejectsNestedModelsDecoderPagerAndSourceDrift(t *testing.T) {
	pkg, plan := containerFilterNativeFixture(t, containerFilterNativeFixtureSource)
	if plan == nil || !containerBodyNativeSchema(pkg, plan) || identityCollectionEnabled(pkg, plan, 0) || len(identityCollectionSpecs) != 20 || len(bodyFilterCollectionSpecs) != 9 {
		t.Fatal(plan)
	}
	if err := validateContainerBodyNativeDeclarations(pkg, containerBodyPinnedDeclarations(t), plan); err != nil {
		t.Fatal(err)
	}
	for name, pair := range map[string][2]string{
		"consumer-value-type":           {"URL string `json:\"url\"`", "URL any `json:\"url\"`"},
		"secret-reference-tag":          {"SecretRef string `json:\"secret_ref\"`;Name string", "SecretRef string `json:\"href\"`;Name string"},
		"array-projection":              {"Consumers []ConsumerRef", "Consumers []string"},
		"timestamp-projection":          {"Created time.Time `json:\"-\"`", "Created string `json:\"created\"`"},
		"incidental-creator-field-lost": {"CreatorID string `json:\"creator_id\"`;", ""},
		"extra-raw-id":                  {"type Container struct{", "type Container struct{ID string;"},
		"decoder-removed":               {"func(*Container)UnmarshalJSON([]byte)error{return nil}", ""},
		"decoder-input":                 {"UnmarshalJSON([]byte)", "UnmarshalJSON(string)"},
		"request-builder-missing":       {"func(SecretRef)ToContainerSecretRefMap()(map[string]any,error){return nil,nil}", ""},
		"request-builder-result":        {"ToContainerSecretRefMap()(map[string]any,error)", "ToContainerSecretRefMap()(map[string]string,error)"},
		"request-builder-pointer":       {"func(SecretRef)ToContainerSecretRefMap", "func(*SecretRef)ToContainerSecretRefMap"},
		"request-builder-named-map-key": {"ToContainerSecretRefMap()(map[string]any,error)", "ToContainerSecretRefMap()(map[NamedString]any,error)"},
		"request-builder-named-map":     {"ToContainerSecretRefMap()(map[string]any,error)", "ToContainerSecretRefMap()(NamedMap,error)"},
		"native-name-hint":              {"Name string `q:\"name\"`", "Name string `q:\"pattern\"`"},
		"inherited-next":                {"func(ContainerPage)NextPageURL()(string,error){return \"\",nil}", ""},
		"page-state":                    {"ContainerPage struct{pagination.LinkedPageBase}", "ContainerPage struct{extra bool;pagination.LinkedPageBase}"},
		"wrong-extractor":               {"([]Container,error)", "([]string,error)"},
	} {
		t.Run(name, func(t *testing.T) {
			source := strings.Replace(containerFilterNativeFixtureSource, pair[0], pair[1], 1)
			if name == "request-builder-named-map-key" {
				source += "\ntype NamedString string\n"
			}
			if name == "request-builder-named-map" {
				source += "\ntype NamedMap map[string]any\n"
			}
			if source == containerFilterNativeFixtureSource {
				t.Fatal("mutation not applied")
			}
			p, pl := containerFilterNativeFixture(t, source)
			if _, ok := bodyFilterCollectionContract(p, pl, 0); ok {
				t.Fatal("native drift enabled")
			}
			if err := validateBodyFilterCollectionContracts(p, pl); err == nil {
				t.Fatal("native drift silently dropped capability")
			}
		})
	}
	alias := strings.Replace(containerFilterNativeFixtureSource, "ToContainerSecretRefMap()(map[string]any,error)", "ToContainerSecretRefMap()(BodyAlias,error)", 1) + "\ntype BodyAlias = map[string]interface{}\n"
	aliasPkg, aliasPlan := containerFilterNativeFixture(t, alias)
	if !containerBodyNativeSchema(aliasPkg, aliasPlan) {
		t.Fatal("equivalent map alias rejected")
	}
	for name, tail := range map[string]string{"nested-decoder": "\nfunc(*ConsumerRef)UnmarshalJSON([]byte)error{return nil}", "secret-ref-decoder": "\nfunc(*SecretRef)UnmarshalJSON([]byte)error{return nil}", "body-override": "\nfunc(ContainerPage)GetBody()any{return nil}"} {
		t.Run(name, func(t *testing.T) {
			p, pl := containerFilterNativeFixture(t, containerFilterNativeFixtureSource+tail)
			if containerBodyNativeSchema(p, pl) {
				t.Fatal("native custom override accepted")
			}
		})
	}
	for name := range containerBodyNativeDeclarations {
		t.Run(name, func(t *testing.T) {
			declarations := containerBodyPinnedDeclarations(t)
			declarations[name].Body.List = append(declarations[name].Body.List, &ast.ExprStmt{X: ast.NewIdent("drift")})
			if err := validateContainerBodyNativeDeclarations(pkg, declarations, plan); err == nil || !strings.Contains(err.Error(), name) {
				t.Fatal("source guard bypassed", err)
			}
		})
	}
	spec, _ := bodyFilterCollectionContract(pkg, plan, 0)
	for name, mutate := range map[string]func(*bodyFilterCollectionSpec){"typed-projection": func(s *bodyFilterCollectionSpec) { s.rawRecord = false }, "wire-collapse": func(s *bodyFilterCollectionSpec) { s.fields[1].key = "container_ref" }, "alias": func(s *bodyFilterCollectionSpec) { s.fields[0].aliases = []string{"consumer"} }, "member": func(s *bodyFilterCollectionSpec) { s.fields[0].member = "Consumers" }, "extra": func(s *bodyFilterCollectionSpec) {
		s.fields = append(s.fields, bodyFilterCollectionField{key: "creator_id"})
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

func TestContainerSemanticFilterEmissionKeepsNameLocalNativePoliciesAndQueryOwnership(t *testing.T) {
	pkg, plan := containerFilterNativeFixture(t, containerFilterNativeFixtureSource)
	m := checkedContainerFilterManifest(t)
	if err := (&generator{}).validatePythonFilterPlan(pkg, plan); err == nil {
		t.Fatal("unchecked source enabled")
	}
	g := generator{containerPythonFilters: m}
	if err := g.validatePythonFilterPlan(pkg, plan); err != nil {
		t.Fatal(err)
	}
	e := emitter{pkg: pkg, imports: map[string]string{}, pythonFilters: g.pythonFilterFor(pkg, plan)}
	e.printf("func(a *API)newResources()*resource.Collection[Container]{return ")
	emitCollectionAdapter(&e, plan, "a", nil)
	e.printf("}\n")
	emitBodyFilterList(&e, plan)
	source, err := e.source()
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, wanted := range []string{"containerBodyFilterValue(record, key)", "resource.BodyStreamWithControl(ctx, upstream.List(a.client, _opts)", "upstream.ExtractContainers(page)", `"containers", control`, "return a.listWithControl(ctx, control, options...)", "return a.listBodyWithControl(ctx, control, options...)", "path.Base(parsed.Path)", "NameQuery:", "Status:", "Failed:", "Delete:"} {
		if !strings.Contains(text, wanted) {
			t.Fatal("existing/native policy lost", wanted, text)
		}
	}
	if strings.Count(text, "config.Query[key] = append([]string(nil), values...)") != 2 || strings.Count(text, `q.Del("status")`) != 2 {
		t.Fatal("both lanes must preserve query values/status policy", text)
	}
	for _, forbidden := range []string{"WithListQuery(", "IdentityFind:", "GetIdentityQuery:", "BodyFilterValue:", "reflect.", "json.Marshal(v.", `"container_id": "container_ref"`, `"created_at": "created"`} {
		if strings.Contains(text, forbidden) {
			t.Fatal("unproved capability/projection", forbidden, text)
		}
	}
	file, err := parser.ParseFile(token.NewFileSet(), "emitted.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	ast.Inspect(file, func(n ast.Node) bool {
		if pair, ok := n.(*ast.KeyValueExpr); ok {
			if key, ok := pair.Key.(*ast.Ident); ok {
				counts[key.Name]++
			}
		}
		return true
	})
	if counts["FilterDescriptor"] != 1 || counts["BodyFilterFields"] != 1 || counts["IterateBodyControlled"] != 1 {
		t.Fatal(counts)
	}
	record := collectionRecord{BodyFilterFields: bodyFilterCollectionFields(pkg, plan, 0), SemanticQueryFilters: pythonFilterQueryFields(m), SemanticBodyFilters: pythonFilterBodyFields(m), SemanticReserved: pythonFilterReserved(m)}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if len(record.BodyFilterFields) != 10 || len(record.SemanticQueryFilters) != 2 || len(record.SemanticBodyFilters) != 10 || record.SemanticQueryFilters["name"] != "" || record.SemanticBodyFilters["name"] != "name" || strings.Contains(string(data), `"identity_find":true`) {
		t.Fatal(record, string(data))
	}
	if _, ok := bodyFilterCollectionContract(pkg, plan, 1); ok {
		t.Fatal("global raw capability leaked to scope")
	}
	for _, identity := range identityCollectionSpecs {
		p, pl := identityQueryFixture(t, identity, identityQueryFixtureSource(identity))
		if g.pythonFilterFor(p, pl) != nil {
			t.Fatal("Container descriptor leaked", identity.path)
		}
	}
	// The shared bridge must leave the previously generated Secret functions
	// identical, not merely preserve a few callback names.
	sp, spl := secretFilterNativeFixture(t, secretFilterNativeFixtureSource)
	se := emitter{pkg: sp, imports: map[string]string{}, pythonFilters: checkedSecretFilterManifest(t)}
	se.printf("func(a *API)newResources()*resource.Collection[Secret]{return ")
	emitCollectionAdapter(&se, spl, "a", nil)
	se.printf("}\n")
	emitBodyFilterList(&se, spl)
	emitted, err := se.source()
	if err != nil {
		t.Fatal(err)
	}
	existing, err := os.ReadFile("../../../keymanager/v1/secrets/resources_generated.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"API.newResources", "API.listBodyWithControl"} {
		if emittedFunctionHash(t, emitted, name) != emittedFunctionHash(t, existing, name) {
			t.Fatal("shared bridge changed existing Secret function", name)
		}
	}
}

func emittedFunctionHash(t *testing.T, source []byte, name string) string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "source.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && identityDeclarationKey(fn) == name {
			hash, err := requestDeclarationHash(fn)
			if err != nil {
				t.Fatal(err)
			}
			return hash
		}
	}
	t.Fatal("missing function", name)
	return ""
}

func TestContainerPythonFilterPinnedSourceRequiresInheritedDefaultProofAndIndependentAST(t *testing.T) {
	m := checkedContainerFilterManifest(t)
	if err := verifyContainerPythonFilterManifest("", m); err == nil {
		t.Fatal("source optional")
	}
	if err := verifyContainerPythonFilterManifest(t.TempDir(), m); err == nil || !strings.Contains(err.Error(), "Python filter source") {
		t.Fatal(err)
	}
	source := os.Getenv("OPENSTACKSDK_SOURCE")
	if source == "" {
		source = "/private/tmp/gophercloudsdk-openstacksdk"
	}
	if _, err := os.Stat(filepath.Join(source, "openstack/key_manager/v1/container.py")); err != nil {
		t.Skip("audited source checkout not present")
	}
	if err := verifyContainerPythonFilterManifest(source, m); err != nil {
		t.Fatal(err)
	}
	fresh, err := extractPythonFilterManifestTarget(source, containerPythonResource)
	if err != nil || !reflect.DeepEqual(fresh, m) {
		t.Fatal(fresh, err)
	}
	root := t.TempDir()
	for path := range containerFilterSourceHashes {
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
	for path := range containerFilterSourceHashes {
		t.Run(path, func(t *testing.T) {
			target := filepath.Join(root, path)
			data, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, append(data, []byte("\n# drift\n")...), 0600); err != nil {
				t.Fatal(err)
			}
			if err := verifyContainerPythonFilterManifest(root, m); err == nil || !strings.Contains(err.Error(), path) {
				t.Fatal("SHA guard bypassed", err)
			}
			if err := os.WriteFile(target, data, 0600); err != nil {
				t.Fatal(err)
			}
		})
	}
	for name, mutate := range map[string]func(*pythonFilterManifest){"proof": func(m *pythonFilterManifest) { m.Proof.Nodes[1].ASTSHA256 = "wrong" }, "parser": func(m *pythonFilterManifest) { m.Proof.PythonParser = "other" }, "reserved": func(m *pythonFilterManifest) { m.Reserved = m.Reserved[:8] }, "controls": func(m *pythonFilterManifest) { m.SourceControls.ResourceList = m.SourceControls.ResourceList[:6] }, "source-path-escape": func(m *pythonFilterManifest) { m.Proof.Files["../outside.py"] = "wrong" }} {
		t.Run(name, func(t *testing.T) {
			copy := clonePythonFilterManifest(t, m)
			mutate(copy)
			if err := verifyContainerPythonFilterManifest(root, copy); err == nil {
				t.Fatal("manifest replaced live proof")
			}
		})
	}
	loaded, err := loadContainerPythonFilterManifest("../../..", source)
	if err != nil || !reflect.DeepEqual(loaded, m) {
		t.Fatal(loaded, err)
	}
	for resource, wanted := range map[string]*pythonFilterManifest{"": checkedPythonFilterManifest(t), secretPythonResource: checkedSecretFilterManifest(t)} {
		actual, err := extractPythonFilterManifestTarget(source, resource)
		if err != nil || !reflect.DeepEqual(actual, wanted) {
			t.Fatal("new target changed existing manifest", resource, err)
		}
	}
}

func TestContainerRawPageDependencyLoaderRequiresExactNumbersAndOriginalBody(t *testing.T) {
	pkg, plan := containerFilterNativeFixture(t, containerFilterNativeFixtureSource)
	if _, err := (&generator{}).identityPaginationDeclarations(pkg.Path()); err == nil || !strings.Contains(err.Error(), "Container body collection") {
		t.Fatal("missing dependency metadata accepted", err)
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
	declarations := containerBodyPinnedDeclarations(t)
	for name, fn := range dependencies {
		declarations[name] = fn
	}
	if err := validateContainerBodyNativeDeclarations(pkg, declarations, plan); err != nil {
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
		declarations[name] = fn
	}
	if err := validateContainerBodyNativeDeclarations(pkg, declarations, plan); err == nil || !strings.Contains(err.Error(), "pagination.PageResultFrom changed") {
		t.Fatal("native precision loss guard bypassed", err)
	}
}

// The generator consumes compiled native exports. This check uses the same
// audited source metadata as generation, so declarations outside results.go
// cannot be accidentally omitted from a convenient model fixture.
func TestContainerBodyFilterAcceptsRealPinnedNativeTypeGraphAndBuilderMethod(t *testing.T) {
	path := os.Getenv("GOPHERCLOUD_METADATA")
	if path == "" {
		path = "/private/tmp/gophercloudsdk-upstream-packages.json"
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		t.Skip("compiled pinned native metadata not present; pass GOPHERCLOUD_METADATA")
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
	path = upstreamModule + "/openstack/keymanager/v1/containers"
	m, ok := meta[path]
	if !ok || m.Dir == "" || m.Export == "" {
		t.Fatal("native Container source/export metadata missing")
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
	decls, native := map[string]*ast.FuncDecl{}, map[string]*ast.FuncDecl{}
	for _, name := range m.GoFiles {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(m.Dir, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			if fn, ok := declaration.(*ast.FuncDecl); ok {
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
	if err := validateBodyFilterCollectionContracts(pkg, plan); err != nil {
		t.Fatal(err)
	}
	g := generator{meta: meta}
	dependencies, err := g.identityPaginationDeclarations(path)
	if err != nil {
		t.Fatal(err)
	}
	for name, declaration := range dependencies {
		native[name] = declaration
	}
	if err := validateContainerBodyNativeDeclarations(pkg, native, plan); err != nil {
		t.Fatal(err)
	}
	nested, ok := pkg.Scope().Lookup("SecretRef").Type().(*types.Named)
	if !ok || nested.NumMethods() != 1 || nested.Method(0).Name() != "ToContainerSecretRefMap" {
		t.Fatal("actual nested request builder was omitted", nested)
	}
}

const containerFilterNativeFixtureSource = `package containers
import "context"
import "time"
import gophercloud "github.com/gophercloud/gophercloud/v2"
import "github.com/gophercloud/gophercloud/v2/pagination"
type ConsumerRef struct{Name string ` + "`" + `json:"name"` + "`" + `;URL string ` + "`" + `json:"url"` + "`" + `}
type SecretRef struct{SecretRef string ` + "`" + `json:"secret_ref"` + "`" + `;Name string ` + "`" + `json:"name"` + "`" + `}
func(SecretRef)ToContainerSecretRefMap()(map[string]any,error){return nil,nil}
type Container struct{Consumers []ConsumerRef ` + "`" + `json:"consumers"` + "`" + `;ContainerRef string ` + "`" + `json:"container_ref"` + "`" + `;Created time.Time ` + "`" + `json:"-"` + "`" + `;CreatorID string ` + "`" + `json:"creator_id"` + "`" + `;Name string ` + "`" + `json:"name"` + "`" + `;SecretRefs []SecretRef ` + "`" + `json:"secret_refs"` + "`" + `;Status string ` + "`" + `json:"status"` + "`" + `;Type string ` + "`" + `json:"type"` + "`" + `;Updated time.Time ` + "`" + `json:"-"` + "`" + `}
func(*Container)UnmarshalJSON([]byte)error{return nil}
type ListOpts struct{Limit int ` + "`" + `q:"limit"` + "`" + `;Name string ` + "`" + `q:"name"` + "`" + `;Offset int ` + "`" + `q:"offset"` + "`" + `}
type ListOptsBuilder interface{ToContainerListQuery()(string,error)}
func(ListOpts)ToContainerListQuery()(string,error){return "",nil}
type GetResult struct{}
func(GetResult)Extract()(*Container,error){return nil,nil}
func Get(context.Context,*gophercloud.ServiceClient,string)GetResult{return GetResult{}}
func Delete(context.Context,*gophercloud.ServiceClient,string)error{return nil}
type ContainerPage struct{pagination.LinkedPageBase}
func(ContainerPage)IsEmpty()(bool,error){return false,nil}
func(ContainerPage)NextPageURL()(string,error){return "",nil}
func List(*gophercloud.ServiceClient,ListOptsBuilder)pagination.Pager{_ = ContainerPage{};return pagination.Pager{}}
func ExtractContainers(p pagination.Page)([]Container,error){_ = p.(ContainerPage);return nil,nil}
`

const pinnedContainerBodySource = `package containers
func (opts ListOpts) ToContainerListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	return q.String(), err
}

func List(client *gophercloud.ServiceClient, opts ListOptsBuilder) pagination.Pager {
	url := listURL(client)
	if opts != nil {
		query, err := opts.ToContainerListQuery()
		if err != nil {
			return pagination.Pager{Err: err}
		}
		url += query
	}
	return pagination.NewPager(client, url, func(r pagination.PageResult) pagination.Page {
		return ContainerPage{pagination.LinkedPageBase{PageResult: r}}
	})
}

func Get(ctx context.Context, client *gophercloud.ServiceClient, id string) (r GetResult) {
	resp, err := client.Get(ctx, getURL(client, id), &r.Body, nil)
	_, r.Header, r.Err = gophercloud.ParseResponse(resp, err)
	return
}

func (r *Container) UnmarshalJSON(b []byte) error {
	type tmp Container
	var s struct {
		tmp
		Created gophercloud.JSONRFC3339NoZ ` + "`" + `json:"created"` + "`" + `
		Updated gophercloud.JSONRFC3339NoZ ` + "`" + `json:"updated"` + "`" + `
	}
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	*r = Container(s.tmp)

	r.Created = time.Time(s.Created)
	r.Updated = time.Time(s.Updated)

	return nil
}

func (r commonResult) Extract() (*Container, error) {
	var s *Container
	err := r.ExtractInto(&s)
	return s, err
}

func (r ContainerPage) IsEmpty() (bool, error) {
	if r.StatusCode == 204 {
		return true, nil
	}

	containers, err := ExtractContainers(r)
	return len(containers) == 0, err
}

func (r ContainerPage) NextPageURL() (string, error) {
	var s struct {
		Next     string ` + "`" + `json:"next"` + "`" + `
		Previous string ` + "`" + `json:"previous"` + "`" + `
	}
	err := r.ExtractInto(&s)
	if err != nil {
		return "", err
	}
	return s.Next, err
}

func ExtractContainers(r pagination.Page) ([]Container, error) {
	var s struct {
		Containers []Container ` + "`" + `json:"containers"` + "`" + `
	}
	err := (r.(ContainerPage)).ExtractInto(&s)
	return s.Containers, err
}

func listURL(client *gophercloud.ServiceClient) string {
	return client.ServiceURL("containers")
}

func getURL(client *gophercloud.ServiceClient, id string) string {
	return client.ServiceURL("containers", id)
}
`
