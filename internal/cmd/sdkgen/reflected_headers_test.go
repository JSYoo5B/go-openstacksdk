package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func revisionActualMetadata(t *testing.T) (generator, types.Importer) {
	t.Helper()
	path := os.Getenv("GOPHERCLOUD_METADATA")
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
	return snapshotMetadataActualNative(t)
}
func revisionNativeSource(t *testing.T, g generator, imp types.Importer, short string) (*types.Package, map[string]*ast.FuncDecl, map[*ast.FuncDecl]*ast.File) {
	t.Helper()
	path := upstreamModule + "/openstack/" + strings.Replace(short, "network/", "networking/", 1)
	pkg, err := imp.Import(path)
	if err != nil {
		t.Fatal(err)
	}
	entry, present := g.meta[path]
	if !present || len(entry.GoFiles) == 0 {
		t.Fatal("pinned native source missing", path)
	}
	decls, files := map[string]*ast.FuncDecl{}, map[*ast.FuncDecl]*ast.File{}
	for _, name := range entry.GoFiles {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(entry.Dir, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range file.Decls {
			if fn, ok := node.(*ast.FuncDecl); ok && fn.Recv == nil {
				decls[fn.Name.Name], files[fn] = fn, file
			}
		}
	}
	return pkg, decls, files
}
func revisionFormat(t *testing.T, node ast.Node) string {
	t.Helper()
	var out bytes.Buffer
	if err := format.Node(&out, token.NewFileSet(), node); err != nil {
		t.Fatal(err)
	}
	return out.String()
}
func revisionParse(t *testing.T, source string) (*ast.File, *ast.FuncDecl) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "revision.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range file.Decls {
		if fn, ok := node.(*ast.FuncDecl); ok && fn.Name.Name == "Request" {
			return file, fn
		}
	}
	t.Fatal("Request declaration missing")
	return nil, nil
}
func revisionDeclarations(file *ast.File) map[string]ast.Node {
	values := map[string]ast.Node{}
	for _, node := range file.Decls {
		switch decl := node.(type) {
		case *ast.FuncDecl:
			values[identityDeclarationKey(decl)] = decl
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				if typ, ok := spec.(*ast.TypeSpec); ok {
					values[typ.Name.Name] = typ
				}
			}
		}
	}
	return values
}

func TestReflectedHeadersActualNineNativeAdapters(t *testing.T) {
	g, imp := revisionActualMetadata(t)
	paths := []string{"network/v2/networks", "network/v2/ports", "network/v2/subnets", "network/v2/extensions/trunks", "network/v2/extensions/subnetpools", "network/v2/extensions/layer3/routers", "network/v2/extensions/layer3/floatingips", "network/v2/extensions/qos/policies", "network/v2/extensions/security/groups"}
	if len(reflectedHeaderRequests) != len(paths) {
		t.Fatal("reflected request scope changed")
	}
	for _, short := range paths {
		t.Run(short, func(t *testing.T) {
			pkg, decls, files := revisionNativeSource(t, g, imp, short)
			if err := validateReflectedHeaderRequests(pkg, decls, files); err != nil {
				t.Fatal(err)
			}
			fn := pkg.Scope().Lookup("Update").(*types.Func)
			e := emitter{pkg: pkg, imports: map[string]string{}, sourceFiles: files}
			if err := emitOperation(&e, fn, decls["Update"], nil); err != nil {
				t.Fatal(err)
			}
			data, err := e.source()
			if err != nil {
				t.Fatal(err)
			}
			emitted, err := parser.ParseFile(token.NewFileSet(), "emitted.go", data, 0)
			if err != nil {
				t.Fatal(err)
			}
			stored, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", "..", "..", short, "api_generated.go"), nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			actual, generated := revisionDeclarations(stored), revisionDeclarations(emitted)
			for key, declaration := range generated {
				if actual[key] == nil || revisionFormat(t, actual[key]) != revisionFormat(t, declaration) {
					t.Fatalf("generated declaration differs from stored facade: %s", key)
				}
			}
			for _, public := range []string{"UpdateOption", "WithUpdateOptions", "WithUpdateField", "API.Update"} {
				if generated[public] == nil {
					t.Fatal("existing public declaration lost", public)
				}
			}
			if generated["WithUpdateHeader"] != nil || generated["WithUpdateRevisionNumber"] != nil {
				t.Fatal("header carrier expanded public helpers")
			}
			carrier := generated["updateOptsBuilder"].(*ast.TypeSpec).Type.(*ast.StructType)
			if len(carrier.Fields.List) != 3 {
				t.Fatal("unexpected carrier fields", carrier.Fields)
			}
			field := carrier.Fields.List[2]
			tag, err := strconv.Unquote(field.Tag.Value)
			pointer, ok := field.Type.(*ast.StarExpr)
			if err != nil || len(field.Names) != 1 || field.Names[0].Name != "RevisionNumber" || !ok || revisionFormat(t, pointer.X) != "int" || tag != `json:"-" h:"If-Match"` {
				t.Fatal("native field type or tag lost", field, tag, err)
			}
			update := generated["API.Update"].(*ast.FuncDecl)
			initialized, validated := false, false
			for _, statement := range update.Body.List {
				ast.Inspect(statement, func(node ast.Node) bool {
					if call, ok := node.(*ast.CallExpr); ok {
						if selector, ok := call.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "ValidateCapabilities" {
							validated = true
						}
					}
					if literal, ok := node.(*ast.CompositeLit); ok {
						if typ, ok := literal.Type.(*ast.Ident); ok && typ.Name == "updateOptsBuilder" {
							for _, value := range literal.Elts {
								pair := value.(*ast.KeyValueExpr)
								if key := pair.Key.(*ast.Ident); key.Name == "RevisionNumber" {
									initialized = validated && revisionFormat(t, pair.Value) == "cfg.Options.RevisionNumber"
								}
							}
						}
					}
					return true
				})
			}
			if !initialized {
				t.Fatal("carrier is not prepared from final validated options")
			}
		})
	}
}

func TestReflectedHeadersRejectNativeSourceAndRevisionDrift(t *testing.T) {
	g, imp := revisionActualMetadata(t)
	pkg, decls, files := revisionNativeSource(t, g, imp, "network/v2/networks")
	original := decls["Update"]
	source := revisionFormat(t, original)
	for _, mutation := range [][2]string{{"revision_number=%s", "revision=%s"}, {"[]int{200, 201}", "[]int{202}"}, {"opts.ToNetworkUpdateMap()", "opts.ToChangedMap()"}} {
		changed := strings.Replace(source, mutation[0], mutation[1], 1)
		if changed == source {
			t.Fatal("source mutation did not apply", mutation[0])
		}
		file, err := parser.ParseFile(token.NewFileSet(), "changed.go", "package networks\n"+changed, 0)
		if err != nil {
			t.Fatal(err)
		}
		fn := file.Decls[0].(*ast.FuncDecl)
		if err := validateReflectedHeaderRequests(pkg, map[string]*ast.FuncDecl{"Update": fn}, map[*ast.FuncDecl]*ast.File{fn: files[original]}); err == nil {
			t.Fatal("changed native Update accepted", mutation[0])
		}
	}
	if err := validateReflectedHeaderRequests(pkg, map[string]*ast.FuncDecl{}, files); err == nil {
		t.Fatal("missing native Update accepted")
	}
	changedFile := *files[original]
	changedFile.Imports = append([]*ast.ImportSpec(nil), changedFile.Imports...)
	for i, spec := range changedFile.Imports {
		path, _ := strconv.Unquote(spec.Path.Value)
		if path == upstreamModule {
			copy := *spec
			copy.Path = &ast.BasicLit{Kind: token.STRING, Value: `"example.invalid/gophercloud"`}
			changedFile.Imports[i] = &copy
		}
	}
	if err := validateReflectedHeaderRequests(pkg, decls, map[*ast.FuncDecl]*ast.File{original: &changedFile}); err == nil {
		t.Fatal("wrong owning-file import accepted")
	}
	for _, field := range []string{`RevisionNumber *int64 ` + "`json:\"-\" h:\"If-Match\"`", `RevisionNumber *int ` + "`json:\"-\" h:\"Other\"`", `RevisionNumber *int ` + "`json:\"-\" h:\"If-Match\" required:\"true\"`", `Other *int ` + "`json:\"-\" h:\"If-Match\"`", `RevisionNumber *int ` + "`json:\"-\"`"} {
		fixtureSource := "package networks\nimport gophercloud \"" + upstreamModule + "\"\ntype UpdateOpts struct{" + field + "}\ntype UpdateOptsBuilder interface{ToNetworkUpdateMap()(map[string]any,error)}\nfunc(UpdateOpts)ToNetworkUpdateMap()(map[string]any,error){return nil,nil}\nfunc Update(ctx any,c *gophercloud.ServiceClient,id string,opts UpdateOptsBuilder)error{return nil}\n"
		fixturePackage, _ := typedCollectionFixture(t, pkg.Path(), fixtureSource)
		if err := validateReflectedHeaderRequests(fixturePackage, decls, files); err == nil {
			t.Fatal("changed revision field accepted", field)
		}
	}
}

func TestReflectedHeadersOwningImportsAndLexicalScope(t *testing.T) {
	cloud := types.NewPackage(upstreamModule, "gophercloud")
	cloud.MarkComplete()
	pkg := types.NewPackage("fixture", "fixture")
	pkg.SetImports([]*types.Package{cloud})
	for _, tc := range []struct {
		name, importLine, body string
		want                   bool
	}{
		{"default", `import "` + upstreamModule + `"`, `_,_=gophercloud.BuildHeaders(opts)`, true},
		{"renamed", `import gc "` + upstreamModule + `"`, `_,_=gc.BuildHeaders(opts)`, true},
		{"dot", `import . "` + upstreamModule + `"`, `_,_=BuildHeaders(opts)`, true},
		{"unrelated", `import gophercloud "example.invalid/other"`, `_,_=gophercloud.BuildHeaders(opts)`, false},
		{"other argument", `import "` + upstreamModule + `"`, `_,_=gophercloud.BuildHeaders(other)`, false},
		{"parameter shadow", `import "` + upstreamModule + `"`, `{opts:=other;_,_=gophercloud.BuildHeaders(opts)}`, false},
		{"qualifier shadow", `import "` + upstreamModule + `"`, `{gophercloud:=other;_=gophercloud.BuildHeaders(opts)}`, false},
		{"nested function", `import "` + upstreamModule + `"`, `_=func(){_,_=gophercloud.BuildHeaders(opts)}`, false},
		{"different function", `import "` + upstreamModule + `"`, `_,_=gophercloud.BuildRequestBody(opts)`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, fn := revisionParse(t, "package fixture\n"+tc.importLine+"\nfunc Request(opts,other any){"+tc.body+"}\n")
			if got := directNativeHeaders(pkg, fn, file, "opts"); got != tc.want {
				t.Fatalf("direct=%v want=%v", got, tc.want)
			}
		})
	}
	file, fn := revisionParse(t, "package fixture\nimport gc \"example.invalid/other\"\nfunc Request(opts any){_,_=gc.BuildHeaders(opts)}")
	other, _ := revisionParse(t, "package fixture\nimport gc \""+upstreamModule+"\"\nfunc Request(opts any){}")
	_ = other
	if directNativeHeaders(pkg, fn, file, "opts") {
		t.Fatal("another file's root alias was used")
	}
	if directNativeHeaders(pkg, nil, file, "opts") || directNativeHeaders(pkg, fn, nil, "opts") {
		t.Fatal("missing owning declaration accepted")
	}
}

func TestReflectedHeadersPreserveOptionalBuildersAndSecondaryInputs(t *testing.T) {
	for _, pointer := range []bool{false, true} {
		for _, tagged := range []bool{false, true} {
			methodReceiver, tag := "Options", `json:"-"`
			if pointer {
				methodReceiver = "*Options"
			}
			if tagged {
				tag += ` h:"If-Match" required:"true"`
			}
			source := "package fixture\nimport gophercloud \"" + upstreamModule + "\"\ntype Options struct{RevisionNumber *int " + strconv.Quote(tag) + "}\ntype SecondaryOpts struct{RevisionNumber *int " + strconv.Quote(tag) + "}\ntype OptionsBuilder interface{ToBody()(map[string]any,error)}\ntype HeadersBuilder interface{ToHeaders()(map[string]string,error)}\ntype SecondaryOptsBuilder interface{ToSecondaryMap()(map[string]any,error)}\nfunc(" + methodReceiver + ")ToBody()(map[string]any,error){return nil,nil}\nfunc(" + methodReceiver + ")ToHeaders()(map[string]string,error){return nil,nil}\nfunc(SecondaryOpts)ToSecondaryMap()(map[string]any,error){return nil,nil}\nfunc Request(client *gophercloud.ServiceClient,opts OptionsBuilder,other SecondaryOptsBuilder)error{return nil}\n"
			pkg, _ := typedCollectionFixture(t, "fixture", source)
			file, decl := revisionParse(t, strings.Replace(source, "error{return nil}", "error{_,_=gophercloud.BuildHeaders(opts);if h,ok:=opts.(HeadersBuilder);ok{_,_=h.ToHeaders()};if other!=nil{_,_=gophercloud.BuildHeaders(other)};return nil}", 1))
			base, err := concrete(pkg, pkg.Scope().Lookup("OptionsBuilder").Type())
			if err != nil {
				t.Fatal(err)
			}
			iface, _ := ifaceOf(pkg.Scope().Lookup("OptionsBuilder").Type())
			b := withOptionalBuilders(pkg, builder{name: "opts", base: base, iface: iface, adapter: "requestOptsBuilder"}, decl, nil)
			b, err = withReflectedHeaders(pkg, b, decl, file)
			if err != nil || b.iface.NumMethods() != 2 || !types.Implements(base, b.iface) {
				t.Fatal("optional native methods lost", b, err)
			}
			if _, gotPointer := base.(*types.Pointer); gotPointer != pointer {
				t.Fatal("concrete pointer shape changed")
			}
			wantFields := 0
			if tagged {
				wantFields = 1
			}
			if len(b.headers) != wantFields || tagged && b.headers[0].tag != tag {
				t.Fatal("top-level native tag not retained", b.headers)
			}
			e := emitter{pkg: pkg, imports: map[string]string{}, sourceFiles: map[*ast.FuncDecl]*ast.File{decl: file}}
			if err := emitOperation(&e, pkg.Scope().Lookup("Request").(*types.Func), decl, nil); err != nil {
				t.Fatal(err)
			}
			data, err := e.source()
			if err != nil {
				t.Fatal(err)
			}
			generated, err := parser.ParseFile(token.NewFileSet(), "generated.go", data, 0)
			if err != nil {
				t.Fatal(err)
			}
			declarations := revisionDeclarations(generated)
			if declarations["requestOptsBuilder.ToHeaders"] == nil || declarations["WithRequestOther"] == nil {
				t.Fatal("optional method or secondary helper lost")
			}
			request := declarations["API.Request"].(*ast.FuncDecl)
			var hasAbsentInterface, provided bool
			constructors := map[string]string{}
			ast.Inspect(request.Body, func(node ast.Node) bool {
				if group, ok := node.(*ast.GenDecl); ok && group.Tok == token.VAR {
					for _, raw := range group.Specs {
						value := raw.(*ast.ValueSpec)
						for _, name := range value.Names {
							if name.Name == "_other" {
								hasAbsentInterface = true
							}
						}
					}
				}
				if branch, ok := node.(*ast.IfStmt); ok {
					if name, ok := branch.Cond.(*ast.Ident); ok && name.Name == "provided" {
						provided = true
					}
				}
				if literal, ok := node.(*ast.CompositeLit); ok {
					if typ, ok := literal.Type.(*ast.Ident); ok && (typ.Name == "requestOptsBuilder" || typ.Name == "requestOtherBuilder") {
						for _, raw := range literal.Elts {
							pair := raw.(*ast.KeyValueExpr)
							if key := pair.Key.(*ast.Ident); key.Name == "RevisionNumber" {
								constructors[typ.Name] = revisionFormat(t, pair.Value)
							}
						}
					}
				}
				return true
			})
			if !hasAbsentInterface || !provided {
				t.Fatal("secondary nil/provided branch changed")
			}
			if tagged && (constructors["requestOptsBuilder"] != "cfg.Options.RevisionNumber" || constructors["requestOtherBuilder"] != "base.RevisionNumber") || !tagged && len(constructors) != 0 {
				t.Fatal("final primary/secondary header sources changed", constructors)
			}
		}
	}
}

func TestReflectedQueryFieldsAreCopiedOnlyForReviewedLists(t *testing.T) {
	source := "package fixture\nimport gophercloud \"" + upstreamModule + "\"\ntype ListOpts struct{Kind string `q:\"interface\"`;Body string `json:\"body\"`}\ntype ListOptsBuilder interface{ToListParams()(string,error)}\nfunc(ListOpts)ToListParams()(string,error){return \"\",nil}\nfunc Request(client *gophercloud.ServiceClient,opts ListOptsBuilder)error{return nil}\n"
	pkg, _ := typedCollectionFixture(t, "fixture", source)
	file, decl := revisionParse(t, strings.Replace(source, "error{return nil}", "error{_,_=gophercloud.BuildQueryString(opts);return nil}", 1))
	base, err := concrete(pkg, pkg.Scope().Lookup("ListOptsBuilder").Type())
	if err != nil {
		t.Fatal(err)
	}
	iface, _ := ifaceOf(pkg.Scope().Lookup("ListOptsBuilder").Type())
	b := builder{name: "opts", base: base, iface: iface, adapter: "listOptsBuilder"}
	if _, err := withReflectedHeaders(pkg, b, decl, file); err == nil || !strings.Contains(err.Error(), "fixture.Request needs review") {
		t.Fatal("unreviewed query reflection accepted", err)
	}
	if !capabilities(pkg, b).query {
		t.Fatal("fixture builder should expose query extensions before review")
	}
	reflectedQueryRequests["fixture.Request"] = true
	defer delete(reflectedQueryRequests, "fixture.Request")
	b, err = withReflectedHeaders(pkg, b, decl, file)
	if err != nil || !b.reflectedQuery || len(b.headers) != 1 || b.headers[0].field.Name() != "Kind" || b.headers[0].tag != `q:"interface"` {
		t.Fatal("reviewed query fields not copied", b.headers, err)
	}
	if capabilities(pkg, b).query {
		t.Fatal("query extensions remain enabled although native List never reads them")
	}
	if literal := builderLiteral(b, "cfg.Options", "cfg"); literal != "listOptsBuilder{base:cfg.Options,config:cfg,Kind:cfg.Options.Kind}" {
		t.Fatal(literal)
	}
}
