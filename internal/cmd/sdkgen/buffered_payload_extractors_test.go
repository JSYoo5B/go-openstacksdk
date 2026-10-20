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
	"strings"
	"testing"
)

func bufferedPayloadMetadata(t *testing.T) (generator, types.Importer) {
	t.Helper()
	// An explicitly supplied metadata file must never turn into a skip.
	if path := os.Getenv("GOPHERCLOUD_METADATA"); path != "" {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
	return snapshotMetadataActualNative(t)
}

func bufferedPayloadNativeSource(t *testing.T, g generator, imp types.Importer, short string) (*types.Package, map[string]*ast.FuncDecl, map[*ast.FuncDecl]*ast.File) {
	t.Helper()
	path := upstreamModule + "/openstack/" + short
	pkg, err := imp.Import(path)
	if err != nil {
		t.Fatal(err)
	}
	entry, present := g.meta[path]
	if !present || len(entry.GoFiles) == 0 {
		t.Fatal("native source missing", path)
	}
	decls, files := map[string]*ast.FuncDecl{}, map[*ast.FuncDecl]*ast.File{}
	for _, name := range entry.GoFiles {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(entry.Dir, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, raw := range file.Decls {
			if fn, ok := raw.(*ast.FuncDecl); ok {
				key := identityDeclarationKey(fn)
				if key == "" {
					continue
				}
				if decls[key] != nil {
					t.Fatal("duplicate native declaration", key)
				}
				decls[key], files[fn] = fn, file
			}
		}
	}
	return pkg, decls, files
}

func bufferedPayloadFormat(t *testing.T, node ast.Node) string {
	t.Helper()
	var out bytes.Buffer
	if err := format.Node(&out, token.NewFileSet(), node); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func bufferedPayloadFacadeDeclarations(file *ast.File) map[string]ast.Node {
	decls := map[string]ast.Node{}
	for _, raw := range file.Decls {
		switch decl := raw.(type) {
		case *ast.FuncDecl:
			decls[identityDeclarationKey(decl)] = decl
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				if typ, ok := spec.(*ast.TypeSpec); ok {
					decls[typ.Name.Name] = typ
				}
			}
		}
	}
	return decls
}

func TestBufferedPayloadActualNativeFamiliesAndFacades(t *testing.T) {
	g, imp := bufferedPayloadMetadata(t)
	for _, tc := range []struct{ path, operation, policy, extracted string }{
		{"keymanager/v1/secrets", "GetPayload", "extract", "[]byte"},
		{"objectstorage/v1/objects", "Download", "download", "*DownloadHeader"},
		{"image/v2/imagedata", "Download", "download", "io.ReadCloser"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			pkg, native, files := bufferedPayloadNativeSource(t, g, imp, tc.path)
			fn := pkg.Scope().Lookup(tc.operation).(*types.Func)
			if err := validateBufferedPayloadTypes(pkg, fn); err != nil {
				t.Fatal(err)
			}
			if err := validateBufferedPayloadDeclarations(pkg, native); err != nil {
				t.Fatal(err)
			}
			if got := operationReturnPolicy(fn); got != tc.policy {
				t.Fatalf("policy=%s want=%s", got, tc.policy)
			}
			if got := returnPolicy(fn.Type().(*types.Signature)); got != "download" {
				t.Fatal("generic stream classifier changed", got)
			}
			name, extractor := operationExtractor(fn)
			if name != "Extract" || extractor == nil || types.TypeString(extractor.Results().At(0).Type(), func(p *types.Package) string {
				if p == pkg {
					return ""
				}
				return p.Name()
			}) != tc.extracted {
				t.Fatal("native extractor shape changed", name, extractor)
			}
			e := emitter{pkg: pkg, imports: map[string]string{}, sourceFiles: files}
			if err := emitOperation(&e, fn, native[tc.operation], nil); err != nil {
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
			stored, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", "..", "..", tc.path, "api_generated.go"), nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			actual, emitted := bufferedPayloadFacadeDeclarations(stored), bufferedPayloadFacadeDeclarations(generated)
			for name, declaration := range emitted {
				if actual[name] == nil || bufferedPayloadFormat(t, actual[name]) != bufferedPayloadFormat(t, declaration) {
					t.Fatal("stored facade differs from native emission", name)
				}
			}
			method := emitted["API."+tc.operation].(*ast.FuncDecl)
			var extracts, opens int
			ast.Inspect(method.Body, func(node ast.Node) bool {
				if call, ok := node.(*ast.CallExpr); ok {
					if name, ok := call.Fun.(*ast.SelectorExpr); ok {
						switch name.Sel.Name {
						case "Extract":
							extracts++
						case "OpenDownload":
							opens++
						}
					}
				}
				return true
			})
			if extracts != 1 {
				t.Fatal("native extractor must run once", extracts)
			}
			if tc.policy == "extract" {
				returns := method.Type.Results.List
				if opens != 0 || len(returns) != 2 || bufferedPayloadFormat(t, returns[0].Type) != "[]byte" || bufferedPayloadFormat(t, returns[1].Type) != "error" {
					t.Fatal("payload was wrapped as a stream", method.Type.Results, opens)
				}
				for _, name := range []string{"GetPayloadOption", "WithGetPayloadOptions", "WithGetPayloadHeader"} {
					if emitted[name] == nil {
						t.Fatal("existing option helper lost", name)
					}
				}
			} else if opens != 1 {
				t.Fatal("working native download wrapper changed", opens)
			}
		})
	}
}

func TestBufferedPayloadRejectsPinnedDeclarationDrift(t *testing.T) {
	g, imp := bufferedPayloadMetadata(t)
	pkg, native, _ := bufferedPayloadNativeSource(t, g, imp, "keymanager/v1/secrets")
	if len(bufferedPayloadNativeDeclarations) != 4 {
		t.Fatal("audited declaration scope changed")
	}
	for name, expected := range bufferedPayloadNativeDeclarations {
		hash, err := requestDeclarationHash(native[name])
		if err != nil || hash != expected {
			t.Fatal("actual native declaration differs", name, hash, err)
		}
		missing := make(map[string]*ast.FuncDecl, len(native))
		for key, declaration := range native {
			if key != name {
				missing[key] = declaration
			}
		}
		if err := validateBufferedPayloadDeclarations(pkg, missing); err == nil {
			t.Fatal("missing source accepted", name)
		}
	}
	for _, tc := range []struct{ name, before, after string }{
		{"GetPayload", `"text/plain"`, `"application/octet-stream"`},
		{"GetPayload", "[]int{200}", "[]int{201}"},
		{"GetPayload", "KeepResponseBody: true", "KeepResponseBody: false"},
		{"GetPayloadOpts.ToSecretPayloadGetParams", "gophercloud.BuildHeaders(opts)", "gophercloud.BuildQueryString(opts)"},
		{"payloadURL", `"payload"`, `"different"`},
		{"PayloadResult.Extract", "io.ReadAll(r.Body)", "io.ReadAll(io.LimitReader(r.Body, 1))"},
		{"PayloadResult.Extract", "defer r.Body.Close()", "defer func() {}()"},
	} {
		source := bufferedPayloadFormat(t, native[tc.name])
		changed := strings.Replace(source, tc.before, tc.after, 1)
		if changed == source {
			t.Fatal("drift fixture did not change source", tc.name, tc.before)
		}
		file, err := parser.ParseFile(token.NewFileSet(), "changed.go", "package secrets\n"+changed, 0)
		if err != nil {
			t.Fatal(err)
		}
		decls := make(map[string]*ast.FuncDecl, len(native))
		for key, declaration := range native {
			decls[key] = declaration
		}
		decls[tc.name] = file.Decls[0].(*ast.FuncDecl)
		if err := validateBufferedPayloadDeclarations(pkg, decls); err == nil {
			t.Fatal("changed native source accepted", tc.name, tc.before)
		}
	}
}

func TestBufferedPayloadRejectsTypedGraphDriftWithoutBroadeningPolicy(t *testing.T) {
	_, imp := bufferedPayloadMetadata(t)
	source := `package secrets
import "context"
import "io"
import gophercloud "github.com/gophercloud/gophercloud/v2"
var _ context.Context
type GetPayloadOpts struct { PayloadContentType string ` + "`h:\"Accept\"`" + ` }
type GetPayloadOptsBuilder interface { ToSecretPayloadGetParams() (map[string]string,error) }
func(GetPayloadOpts) ToSecretPayloadGetParams()(map[string]string,error){return nil,nil}
type PayloadResult struct { gophercloud.Result; Body io.ReadCloser }
func(PayloadResult) Extract()([]byte,error){return nil,nil}
func GetPayload(ctx context.Context,client *gophercloud.ServiceClient,id string,opts GetPayloadOptsBuilder)PayloadResult{return PayloadResult{}}
func Download(ctx context.Context,client *gophercloud.ServiceClient,id string)PayloadResult{return PayloadResult{}}
`
	check := func(t *testing.T, path, source string) (*types.Package, map[string]*ast.FuncDecl) {
		t.Helper()
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "payload.go", source, 0)
		if err != nil {
			t.Fatal(err)
		}
		config := types.Config{Importer: imp}
		pkg, err := config.Check(path, fset, []*ast.File{file}, nil)
		if err != nil {
			t.Fatal(err)
		}
		decls := map[string]*ast.FuncDecl{}
		for _, raw := range file.Decls {
			if fn, ok := raw.(*ast.FuncDecl); ok {
				decls[identityDeclarationKey(fn)] = fn
			}
		}
		return pkg, decls
	}
	path := upstreamModule + "/openstack/keymanager/v1/secrets"
	pkg, _ := check(t, path, source)
	if !matchesBufferedPayloadSignature(pkg, pkg.Scope().Lookup("GetPayload").(*types.Func)) {
		t.Fatal("valid native typed graph rejected")
	}
	if got := operationReturnPolicy(pkg.Scope().Lookup("Download").(*types.Func)); got != "download" {
		t.Fatal("another operation's consuming result was reclassified", got)
	}
	unrelated, _ := check(t, "fixture", source)
	if got := operationReturnPolicy(unrelated.Scope().Lookup("GetPayload").(*types.Func)); got != "download" {
		t.Fatal("unrelated GetPayload was reclassified", got)
	}
	for _, tc := range []struct{ name, before, after string }{
		{"content type", "PayloadContentType string", "PayloadContentType *string"},
		{"full options tag", `h:"Accept"`, `h:"Accept" required:"true"`},
		{"options extra field", "PayloadContentType string", "Extra bool; PayloadContentType string"},
		{"builder return", "map[string]string", "map[string]any"},
		{"options receiver", "func(GetPayloadOpts)", "func(*GetPayloadOpts)"},
		{"result base", "gophercloud.Result;", "gophercloud.HeaderResult;"},
		{"body type", "Body io.ReadCloser", "Body io.Reader"},
		{"result extra field", "Body io.ReadCloser", "Extra bool; Body io.ReadCloser"},
		{"extract receiver", "func(PayloadResult)", "func(*PayloadResult)"},
		{"extract bytes", "([]byte,error)", "([]uint16,error)"},
		{"request context", "ctx context.Context", "ctx any"},
		{"request client", "client *gophercloud.ServiceClient", "client *gophercloud.ProviderClient"},
		{"request id", "id string,opts", "id *string,opts"},
		{"variadic options", "opts GetPayloadOptsBuilder", "opts ...GetPayloadOptsBuilder"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := strings.ReplaceAll(source, tc.before, tc.after)
			if changed == source {
				t.Fatal("typed drift fixture did not change")
			}
			pkg, decls := check(t, path, changed)
			fn := pkg.Scope().Lookup("GetPayload").(*types.Func)
			if matchesBufferedPayloadSignature(pkg, fn) || validateBufferedPayloadTypes(pkg, fn) == nil || validateBufferedPayloadDeclarations(pkg, decls) == nil {
				t.Fatal("changed typed graph accepted")
			}
			e := emitter{pkg: pkg, imports: map[string]string{}}
			if err := emitOperation(&e, fn, decls["GetPayload"], nil); err == nil {
				t.Fatal("invalid graph reached emitter")
			}
		})
	}
}
