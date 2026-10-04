package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const troveRootEnabledFixture = `package instances
import "context"
import gophercloud "github.com/gophercloud/gophercloud/v2"
type IsRootEnabledResult struct{gophercloud.Result}
func(IsRootEnabledResult)Extract()(bool,error){return false,nil}
func IsRootEnabled(ctx context.Context,client *gophercloud.ServiceClient,id string)IsRootEnabledResult{return IsRootEnabledResult{}}
func Other(ctx context.Context,client *gophercloud.ServiceClient,id string)IsRootEnabledResult{return IsRootEnabledResult{}}
type Instance struct{ID string}
type GetResult struct{gophercloud.Result}
func(GetResult)Extract()(*Instance,error){return nil,nil}
func Get(ctx context.Context,client *gophercloud.ServiceClient,id string)GetResult{return GetResult{}}
`

func TestTroveRootEnabledAllowlistKeepsGenericExtractionAndOtherOperations(t *testing.T) {
	for _, short := range []string{"db/v1/instances", "db/v2/instances", "db/v1/users", "compute/v2/servers"} {
		pkg, declarations := typedCollectionFixture(t, upstreamModule+"/openstack/"+short, troveRootEnabledFixture)
		fn := pkg.Scope().Lookup("IsRootEnabled").(*types.Func)
		expected := short == "db/v1/instances"
		if troveRootEnabledExtractor(fn) != expected || matchesTroveRootEnabledSignature(pkg, fn) != expected {
			t.Fatal("operation scope changed", short)
		}
		if err := validateTroveRootEnabledTypes(pkg, fn); err != nil {
			t.Fatal(err)
		}
		name, signature := operationExtractor(fn)
		if name != "Extract" || signature == nil || !types.Identical(signature.Results().At(0).Type(), types.Typ[types.Bool]) || operationReturnPolicy(fn) != "extract" || returnPolicy(fn.Type().(*types.Signature)) != "extract" || normalizerFor(fn) != nil {
			t.Fatal("generic extraction policy changed", short, name, signature)
		}
		for _, name := range []string{"Other", "Get"} {
			fn := pkg.Scope().Lookup(name).(*types.Func)
			if troveRootEnabledExtractor(fn) || operationReturnPolicy(fn) != "extract" {
				t.Fatal("unreviewed operation opted in", short, name)
			}
			e := emitter{pkg: pkg, imports: map[string]string{}}
			if err := emitOperation(&e, fn, declarations[name], nil); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(e.body.String(), "result.Extract()") || strings.Contains(e.body.String(), "troveroot") {
				t.Fatal("unreviewed extraction changed", short, name, e.body.String())
			}
		}
		if !expected {
			e := emitter{pkg: pkg, imports: map[string]string{}}
			if err := emitOperation(&e, fn, declarations[fn.Name()], nil); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(e.body.String(), "result.Extract()") || strings.Contains(e.body.String(), "troveroot") {
				t.Fatal("same operation name in another package opted in", short, e.body.String())
			}
		}
	}
	if troveRootEnabledOperation(nil, "IsRootEnabled") || troveRootEnabledExtractor(nil) {
		t.Fatal("nil operation opted in")
	}
}

func TestTroveRootEnabledRejectsRequestResultAndOwnMethodGraphDrift(t *testing.T) {
	for _, tc := range []struct{ name, before, after string }{
		{"request context", "ctx context.Context", "ctx any"},
		{"request client", "client *gophercloud.ServiceClient", "client gophercloud.ServiceClient"},
		{"request identifier", "id string", "id *string"},
		{"request extra parameter", "id string", "id string,extra string"},
		{"request variadic", "id string", "id ...string"},
		{"request pointer result", ")IsRootEnabledResult{return IsRootEnabledResult{}", ")*IsRootEnabledResult{return &IsRootEnabledResult{}"},
		{"request wrong result", ")IsRootEnabledResult{return IsRootEnabledResult{}", ")GetResult{return GetResult{}"},
		{"result pointer base", "struct{gophercloud.Result}", "struct{*gophercloud.Result}"},
		{"result different base", "struct{gophercloud.Result}", "struct{gophercloud.ServiceClient}"},
		{"result named base", "struct{gophercloud.Result}", "struct{Base gophercloud.Result}"},
		{"result extra field", "struct{gophercloud.Result}", "struct{gophercloud.Result; Extra bool}"},
		{"result embedded tag", "struct{gophercloud.Result}", "struct{gophercloud.Result `json:\"result\"`}"},
		{"extract pointer receiver", "func(IsRootEnabledResult)Extract", "func(*IsRootEnabledResult)Extract"},
		{"extract parameter", "Extract()(bool,error)", "Extract(any)(bool,error)"},
		{"extract variadic", "Extract()(bool,error)", "Extract(...any)(bool,error)"},
		{"extract non-bool result", "Extract()(bool,error)", "Extract()(any,error)"},
		{"extract non-error result", "Extract()(bool,error)", "Extract()(bool,any)"},
		{"result extra method", "func IsRootEnabled(", "func(IsRootEnabledResult)OtherMethod(){}\nfunc IsRootEnabled("},
		{"promoted extractor", "type IsRootEnabledResult struct{gophercloud.Result}\nfunc(IsRootEnabledResult)Extract()(bool,error){return false,nil}", "type Parent struct{gophercloud.Result}\nfunc(Parent)Extract()(bool,error){return false,nil}\ntype IsRootEnabledResult struct{Parent}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := strings.Replace(troveRootEnabledFixture, tc.before, tc.after, 1)
			if changed == troveRootEnabledFixture {
				t.Fatal("typed drift fixture did not change")
			}
			pkg, declarations := typedCollectionFixture(t, upstreamModule+"/openstack/db/v1/instances", changed)
			fn := pkg.Scope().Lookup("IsRootEnabled").(*types.Func)
			if matchesTroveRootEnabledSignature(pkg, fn) || validateTroveRootEnabledTypes(pkg, fn) == nil || validateTroveRootEnabledDeclarations(pkg, nil) == nil {
				t.Fatal("changed graph accepted")
			}
			e := emitter{pkg: pkg, imports: map[string]string{}}
			if err := emitOperation(&e, fn, declarations[fn.Name()], nil); err == nil || e.body.Len() != 0 {
				t.Fatal("invalid graph reached emitter", err, e.body.String())
			}
		})
	}
}

func TestTroveRootEnabledGuardsActualThreePinnedDeclarations(t *testing.T) {
	g, imp := bufferedPayloadMetadata(t)
	path := upstreamModule + "/openstack/db/v1/instances"
	pkg, err := imp.Import(path)
	if err != nil {
		t.Fatal(err)
	}
	declarations, err := g.troveRootEnabledDeclarations(path)
	if err != nil || len(declarations) != 3 || len(troveRootEnabledNativeDeclarations) != 3 {
		t.Fatal("reviewed source scope changed", len(declarations), err)
	}
	if err := validateTroveRootEnabledDeclarations(pkg, declarations); err != nil {
		t.Fatal(err)
	}
	fn := pkg.Scope().Lookup("IsRootEnabled").(*types.Func)
	if !matchesTroveRootEnabledSignature(pkg, fn) || operationReturnPolicy(fn) != "extract" {
		t.Fatal("actual compiled request/result graph rejected")
	}
	for name, expected := range troveRootEnabledNativeDeclarations {
		hash, err := requestDeclarationHash(declarations[name])
		if err != nil || hash != expected {
			t.Fatal("actual source hash differs", name, hash, err)
		}
		missing := make(map[string]*ast.FuncDecl, len(declarations))
		for key, declaration := range declarations {
			if key != name {
				missing[key] = declaration
			}
		}
		if err := validateTroveRootEnabledDeclarations(pkg, missing); err == nil {
			t.Fatal("missing reviewed declaration accepted", name)
		}
	}
	for _, tc := range []struct{ name, before, after string }{
		{"IsRootEnabled", "client.Get(ctx, userRootURL(client, id), &r.Body, nil)", "client.Get(ctx, userRootURL(client, id), &r.Body, &gophercloud.RequestOpts{OkCodes: []int{200, 204}})"},
		{"IsRootEnabled", "gophercloud.ParseResponse(resp, err)", "gophercloud.ParseResponse(resp, nil)"},
		{"IsRootEnabledResult.Extract", `"rootEnabled"`, `"root_enabled"`},
		{"IsRootEnabledResult.Extract", "== true", "== false"},
		{"userRootURL", `"root"`, `"roots"`},
	} {
		source := bufferedPayloadFormat(t, declarations[tc.name])
		changed := strings.Replace(source, tc.before, tc.after, 1)
		if changed == source {
			t.Fatal("source drift fixture did not change", tc.name, tc.before)
		}
		file, err := parser.ParseFile(token.NewFileSet(), "changed.go", "package instances\n"+changed, 0)
		if err != nil {
			t.Fatal(err)
		}
		own := make(map[string]*ast.FuncDecl, len(declarations))
		for name, declaration := range declarations {
			own[name] = declaration
		}
		own[tc.name] = file.Decls[0].(*ast.FuncDecl)
		if err := validateTroveRootEnabledDeclarations(pkg, own); err == nil || !strings.Contains(err.Error(), tc.name) {
			t.Fatal("changed reviewed declaration accepted", tc.name, err)
		}
	}
	if source, err := g.troveRootEnabledDeclarations(upstreamModule + "/openstack/db/v1/users"); err != nil || source != nil {
		t.Fatal("unreviewed source was loaded", err)
	}
}

func TestTroveRootEnabledRejectsMissingAndDuplicateNativeSource(t *testing.T) {
	g, _ := bufferedPayloadMetadata(t)
	path := upstreamModule + "/openstack/db/v1/instances"
	entry := g.meta[path]
	for _, mode := range []string{"missing operation", "missing extractor", "missing URL", "duplicate operation", "duplicate extractor", "duplicate URL", "missing metadata", "missing directory", "missing files"} {
		t.Run(mode, func(t *testing.T) {
			own := generator{meta: map[string]metadata{}}
			copy := entry
			copy.Dir = t.TempDir()
			copy.GoFiles = append([]string(nil), entry.GoFiles...)
			for _, name := range entry.GoFiles {
				data, err := os.ReadFile(filepath.Join(entry.Dir, name))
				if err != nil {
					t.Fatal(err)
				}
				source := string(data)
				switch mode {
				case "missing operation":
					source = strings.Replace(source, "func IsRootEnabled(", "func ChangedIsRootEnabled(", 1)
				case "missing extractor":
					source = strings.Replace(source, "func (r IsRootEnabledResult) Extract(", "func (r IsRootEnabledResult) ChangedExtract(", 1)
				case "missing URL":
					source = strings.Replace(source, "func userRootURL(", "func changedUserRootURL(", 1)
				}
				if err := os.WriteFile(filepath.Join(copy.Dir, name), []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			duplicate := ""
			switch mode {
			case "duplicate operation":
				duplicate = "func IsRootEnabled(){}"
			case "duplicate extractor":
				duplicate = "func (r IsRootEnabledResult) Extract(){}"
			case "duplicate URL":
				duplicate = "func userRootURL(){}"
			case "missing directory":
				copy.Dir = ""
			case "missing files":
				copy.GoFiles = nil
			}
			if duplicate != "" {
				copy.GoFiles = append(copy.GoFiles, "duplicate.go")
				if err := os.WriteFile(filepath.Join(copy.Dir, "duplicate.go"), []byte("package instances\n"+duplicate+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if mode != "missing metadata" {
				own.meta[path] = copy
			}
			if _, err := own.troveRootEnabledDeclarations(path); err == nil {
				t.Fatal("invalid source accepted", mode)
			}
		})
	}
}

func TestTroveRootEnabledActualEmissionStoredFacadeAndBoundedInventory(t *testing.T) {
	g, imp := bufferedPayloadMetadata(t)
	pkg, native, files := bufferedPayloadNativeSource(t, g, imp, "db/v1/instances")
	for _, name := range []string{"IsRootEnabled", "EnableRootUser", "Get"} {
		t.Run(name, func(t *testing.T) {
			fn := pkg.Scope().Lookup(name).(*types.Func)
			e := emitter{pkg: pkg, imports: map[string]string{}, sourceFiles: files}
			if err := emitOperation(&e, fn, native[name], nil); err != nil {
				t.Fatal(err)
			}
			data, err := e.source()
			if err != nil {
				t.Fatal(err)
			}
			emitted, err := parser.ParseFile(token.NewFileSet(), "generated.go", data, 0)
			if err != nil {
				t.Fatal(err)
			}
			stored, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", "..", "..", "db", "v1", "instances", "api_generated.go"), nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			actual, generated := bufferedPayloadFacadeDeclarations(stored), bufferedPayloadFacadeDeclarations(emitted)
			for key, declaration := range generated {
				if actual[key] == nil || bufferedPayloadFormat(t, declaration) != bufferedPayloadFormat(t, actual[key]) {
					t.Fatal("stored facade differs from actual native emission", key)
				}
			}
			out := e.body.String()
			if name == "IsRootEnabled" {
				for _, wanted := range []string{"(bool,error)", "upstream.IsRootEnabled(ctx,a.client,id)", "troveroot.Extract(result.Result)", `request.Wrap("IsRootEnabled","instances",err)`} {
					if !strings.Contains(out, wanted) {
						t.Fatal("safe bool extraction missing", wanted, out)
					}
				}
				if strings.Contains(out, "result.Extract()") || e.imports["gophercloudsdk/internal/troveroot"] != "troveroot" || operationReturnPolicy(fn) != "extract" {
					t.Fatal("unsafe native assertion emitted", out)
				}
			} else if troveRootEnabledExtractor(fn) || strings.Contains(out, "troveroot") || !strings.Contains(out, "result.Extract()") {
				t.Fatal("sibling extraction changed", name, out)
			}
		})
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "api", "gophercloud_inventory.json"))
	if err != nil {
		t.Fatal(err)
	}
	var stored inventory
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, op := range stored.Operations {
		if op.ResultPolicy != "sdk_trove_root_enabled_boolean" {
			continue
		}
		count++
		if op.Source != upstreamModule+"/openstack/db/v1/instances" || op.Package != "db/v1/instances" || op.Name != "IsRootEnabled" || op.SDKPackage != "gophercloudsdk/db/v1/instances" || !op.BuilderFree || op.ReturnPolicy != "extract" || op.Issue != "" {
			t.Fatal("unreviewed inventory operation opted in", op)
		}
	}
	if count != 1 {
		t.Fatal("exactly one operation must record the root-enabled result policy", count)
	}
}

func TestTroveRootEnabledKeepsExistingAuditedResultFamilies(t *testing.T) {
	g, imp := bufferedPayloadMetadata(t)
	for _, tc := range []struct{ short, name, policy string }{
		{"keymanager/v1/secrets", "GetPayload", "extract"},
		{"blockstorage/v2/snapshots", "UpdateMetadata", "extract"},
		{"blockstorage/v3/snapshots", "UpdateMetadata", "extract"},
		{"objectstorage/v1/objects", "Download", "download"},
		{"image/v2/imagedata", "Download", "download"},
		{"compute/v2/servers", "GetPassword", "normalize"},
		{"identity/v2/tokens", "Get", "normalize"},
		{"baremetal/v1/nodes", "GetVirtualMedia", "normalize"},
	} {
		t.Run(tc.short+"/"+tc.name, func(t *testing.T) {
			pkg, native, files := bufferedPayloadNativeSource(t, g, imp, tc.short)
			fn := pkg.Scope().Lookup(tc.name).(*types.Func)
			if troveRootEnabledExtractor(fn) || operationReturnPolicy(fn) != tc.policy {
				t.Fatal("existing result family was reclassified", tc.short, tc.name)
			}
			e := emitter{pkg: pkg, imports: map[string]string{}, sourceFiles: files}
			if err := emitOperation(&e, fn, native[tc.name], nil); err != nil {
				t.Fatal(err)
			}
			data, err := e.source()
			if err != nil {
				t.Fatal(err)
			}
			emitted, err := parser.ParseFile(token.NewFileSet(), "generated.go", data, 0)
			if err != nil {
				t.Fatal(err)
			}
			stored, err := parser.ParseFile(token.NewFileSet(), filepath.Join("..", "..", "..", filepath.FromSlash(tc.short), "api_generated.go"), nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			actual, generated := bufferedPayloadFacadeDeclarations(stored), bufferedPayloadFacadeDeclarations(emitted)
			for key, declaration := range generated {
				if actual[key] == nil || bufferedPayloadFormat(t, declaration) != bufferedPayloadFormat(t, actual[key]) {
					t.Fatal("existing stored result family differs", tc.short, tc.name, key)
				}
			}
			if strings.Contains(e.body.String(), "troveroot") {
				t.Fatal("unreviewed result family opted in", tc.short, tc.name)
			}
		})
	}
}
