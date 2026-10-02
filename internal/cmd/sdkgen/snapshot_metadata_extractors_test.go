package main

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/format"
	"go/importer"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const snapshotMetadataFixture = `package snapshots
import "context"
import gophercloud "github.com/gophercloud/gophercloud/v2"
type Snapshot struct{ID string}
type commonResult struct{gophercloud.Result}
func(commonResult)Extract()(*Snapshot,error){return nil,nil}
type UpdateMetadataResult struct{commonResult}
func(UpdateMetadataResult)ExtractMetadata()(map[string]any,error){return nil,nil}
type UpdateMetadataOpts struct{Metadata map[string]any ` + "`json:\"metadata,omitempty\"`" + `}
func(UpdateMetadataOpts)ToSnapshotUpdateMetadataMap()(map[string]any,error){return nil,nil}
type UpdateMetadataOptsBuilder interface{ToSnapshotUpdateMetadataMap()(map[string]any,error)}
func UpdateMetadata(ctx context.Context,client *gophercloud.ServiceClient,id string,opts UpdateMetadataOptsBuilder)UpdateMetadataResult{return UpdateMetadataResult{}}
func Get(ctx context.Context,client *gophercloud.ServiceClient,id string)commonResult{return commonResult{}}
func Update(ctx context.Context,client *gophercloud.ServiceClient,id string)commonResult{return commonResult{}}
func Create(ctx context.Context,client *gophercloud.ServiceClient)commonResult{return commonResult{}}
`

func snapshotMetadataFixturePackage(t *testing.T, path, source string) (*types.Package, map[string]*ast.FuncDecl) {
	t.Helper()
	return typedCollectionFixture(t, upstreamModule+"/openstack/"+path, source)
}

func snapshotMetadataSource(t *testing.T, declaration *ast.FuncDecl) string {
	t.Helper()
	var out bytes.Buffer
	if err := format.Node(&out, token.NewFileSet(), declaration); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestSnapshotMetadataExtractorAllowlistChoosesOwnMapWithoutChangingSnapshotCRUD(t *testing.T) {
	for _, path := range []string{"blockstorage/v2/snapshots", "blockstorage/v3/snapshots", "blockstorage/v1/snapshots", "blockstorage/v3/volumes", "compute/v2/servers"} {
		pkg, _ := snapshotMetadataFixturePackage(t, path, snapshotMetadataFixture)
		fn := pkg.Scope().Lookup("UpdateMetadata").(*types.Func)
		name, signature := operationExtractor(fn)
		expected := path == "blockstorage/v2/snapshots" || path == "blockstorage/v3/snapshots"
		if expected {
			if name != "ExtractMetadata" || !snapshotMetadataMapSignature(signature) || !matchesSnapshotMetadataSignature(pkg, fn) {
				t.Fatalf("%s selected %s %v", path, name, signature)
			}
		} else if name != "Extract" || snapshotMetadataExtractor(fn) {
			t.Fatalf("unreviewed package %s selected %s", path, name)
		}
		for _, op := range []string{"Get", "Create", "Update"} {
			name, sig := operationExtractor(pkg.Scope().Lookup(op).(*types.Func))
			if name != "Extract" || snapshotMetadataMap(sig.Results().At(0).Type()) {
				t.Fatalf("%s.%s changed: %s %v", path, op, name, sig)
			}
		}
	}
}

func TestSnapshotMetadataExtractorRejectsRequestResultAndBuilderGraphDrift(t *testing.T) {
	for _, tc := range []struct{ name, before, after string }{
		{"pointer result", ")UpdateMetadataResult{return", ")*UpdateMetadataResult{return &"},
		{"wrong result", ")UpdateMetadataResult{return UpdateMetadataResult{}", ")commonResult{return commonResult{}"},
		{"extra request arg", "id string,opts", "id string,extra string,opts"},
		{"named identifier", "type Snapshot struct", "type Identifier string\ntype Snapshot struct"},
		{"concrete builder", "opts UpdateMetadataOptsBuilder", "opts UpdateMetadataOpts"},
		{"pointer builder method", "func(UpdateMetadataOpts)To", "func(*UpdateMetadataOpts)To"},
		{"builder interface parameter", "interface{ToSnapshotUpdateMetadataMap()", "interface{ToSnapshotUpdateMetadataMap(string)"},
		{"builder interface variadic", "interface{ToSnapshotUpdateMetadataMap()", "interface{ToSnapshotUpdateMetadataMap(...string)"},
		{"builder interface result", "interface{ToSnapshotUpdateMetadataMap()(map[string]any,error)}", "interface{ToSnapshotUpdateMetadataMap()(map[string]string,error)}"},
		{"metadata tag", "metadata,omitempty", "metadata"},
		{"required metadata", "json:\"metadata,omitempty\"", "json:\"metadata,omitempty\" required:\"true\""},
		{"extra metadata tag", "json:\"metadata,omitempty\"", "json:\"metadata,omitempty\" q:\"metadata\""},
		{"metadata member type", "Metadata map[string]any", "Metadata map[string]string"},
		{"additional options method", "type UpdateMetadataOptsBuilder", "func(UpdateMetadataOpts)Other(){}\ntype UpdateMetadataOptsBuilder"},
		{"pointer extractor", "func(UpdateMetadataResult)ExtractMetadata", "func(*UpdateMetadataResult)ExtractMetadata"},
		{"extractor result", "ExtractMetadata()(map[string]any,error)", "ExtractMetadata()(map[string]string,error)"},
		{"extra result method", "type UpdateMetadataOpts struct", "func(UpdateMetadataResult)Other(){}\ntype UpdateMetadataOpts struct"},
		{"pointer embedded common", "struct{commonResult}", "struct{*commonResult}"},
		{"extra embedded result field", "struct{commonResult}", "struct{commonResult;Other string}"},
		{"pointer common extractor", "func(commonResult)Extract", "func(*commonResult)Extract"},
		{"wrong common model", "Extract()(*Snapshot,error)", "Extract()(string,error)"},
		{"pointer native base", "struct{gophercloud.Result}", "struct{*gophercloud.Result}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(snapshotMetadataFixture, tc.before, tc.after, 1)
			if source == snapshotMetadataFixture {
				t.Fatal("mutation did not apply")
			}
			if tc.name == "named identifier" {
				source = strings.Replace(source, "id string,opts", "id Identifier,opts", 1)
			}
			if tc.name == "wrong common model" {
				source = strings.Replace(source, "Extract()(string,error){return nil,nil}", "Extract()(string,error){return \"\",nil}", 1)
			}
			pkg, decls := snapshotMetadataFixturePackage(t, "blockstorage/v3/snapshots", source)
			fn := pkg.Scope().Lookup("UpdateMetadata").(*types.Func)
			if err := validateSnapshotMetadataTypes(pkg, fn); err == nil {
				t.Fatal("request/result drift accepted")
			}
			e := emitter{pkg: pkg, imports: map[string]string{}}
			if err := emitOperation(&e, fn, decls[fn.Name()], nil); err == nil || e.body.Len() != 0 {
				t.Fatalf("drift silently emitted body: err=%v body=%s", err, e.body.String())
			}
		})
	}
}

func snapshotMetadataActualNative(t *testing.T) (generator, types.Importer) {
	t.Helper()
	path := os.Getenv("GOPHERCLOUD_METADATA")
	if path == "" {
		path = "/private/tmp/gophercloudsdk-upstream-packages.json"
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
		entry, ok := meta[path]
		if !ok || entry.Export == "" {
			return nil, os.ErrNotExist
		}
		return os.Open(entry.Export)
	})
	if _, err := imp.Import(upstreamModule); err != nil {
		t.Fatal(err)
	}
	return generator{meta: meta, importer: imp}, imp
}

func TestSnapshotMetadataExtractorGuardsAllSixPinnedDeclarationsAndSourceOwnership(t *testing.T) {
	g, imp := snapshotMetadataActualNative(t)
	if len(snapshotMetadataNativeDeclarations) != 2 {
		t.Fatal("allowlist widened")
	}
	for _, short := range []string{"blockstorage/v2/snapshots", "blockstorage/v3/snapshots"} {
		path := upstreamModule + "/openstack/" + short
		pkg, err := imp.Import(path)
		if err != nil {
			t.Fatal(err)
		}
		decls, err := g.snapshotMetadataDeclarations(path)
		if err != nil || len(decls) != 6 || len(snapshotMetadataNativeDeclarations[short]) != 6 {
			t.Fatalf("declarations=%d err=%v", len(decls), err)
		}
		if err := validateSnapshotMetadataDeclarations(pkg, decls); err != nil {
			t.Fatal(err)
		}
		for key, declaration := range decls {
			t.Run(short+"/"+key, func(t *testing.T) {
				copy := make(map[string]*ast.FuncDecl, len(decls))
				for name, decl := range decls {
					copy[name] = decl
				}
				changed := *declaration
				body := *declaration.Body
				body.List = append(append([]ast.Stmt(nil), body.List...), &ast.ExprStmt{X: &ast.CallExpr{Fun: ast.NewIdent("panic"), Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"drift"`}}}})
				changed.Body = &body
				copy[key] = &changed
				if err := validateSnapshotMetadataDeclarations(pkg, copy); err == nil || !strings.Contains(err.Error(), key) {
					t.Fatalf("changed %s accepted: %v", key, err)
				}
				delete(copy, key)
				if err := validateSnapshotMetadataDeclarations(pkg, copy); err == nil {
					t.Fatalf("missing %s accepted", key)
				}
			})
		}
		if !strings.Contains(snapshotMetadataSource(t, decls["UpdateMetadata"]), ".Put(ctx, updateMetadataURL(client, id), b, &r.Body") || !strings.Contains(snapshotMetadataSource(t, decls["UpdateMetadata"]), "OkCodes: []int{200}") || !strings.Contains(snapshotMetadataSource(t, decls["metadataURL"]), `ServiceURL("snapshots", id, "metadata")`) {
			t.Fatal("pinned PUT/200/route contract changed")
		}
	}
	if decls, err := g.snapshotMetadataDeclarations(upstreamModule + "/openstack/compute/v2/servers"); err != nil || decls != nil {
		t.Fatalf("unreviewed source loaded: %v %v", decls, err)
	}
}

func TestSnapshotMetadataExtractorRejectsMissingAndDuplicateQualifiedSource(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	path := upstreamModule + "/openstack/blockstorage/v3/snapshots"
	entry := g.meta[path]
	for _, mode := range []string{"missing", "duplicate", "wrong receiver", "missing metadata"} {
		t.Run(mode, func(t *testing.T) {
			own := generator{meta: map[string]metadata{}}
			copy := entry
			copy.Dir = t.TempDir()
			for _, name := range entry.GoFiles {
				data, err := os.ReadFile(filepath.Join(entry.Dir, name))
				if err != nil {
					t.Fatal(err)
				}
				if mode == "missing" && strings.Contains(string(data), "func updateMetadataURL(") {
					data = []byte(strings.Replace(string(data), "func updateMetadataURL(", "func noLongerUpdateMetadataURL(", 1))
				}
				if mode == "wrong receiver" && strings.Contains(string(data), "func (r UpdateMetadataResult) ExtractMetadata(") {
					data = []byte(strings.Replace(string(data), "func (r UpdateMetadataResult) ExtractMetadata(", "func (r commonResult) ExtractMetadata(", 1))
				}
				if err := os.WriteFile(filepath.Join(copy.Dir, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			copy.GoFiles = append([]string(nil), entry.GoFiles...)
			if mode == "duplicate" {
				copy.GoFiles = append(copy.GoFiles, "duplicate.go")
				if err := os.WriteFile(filepath.Join(copy.Dir, "duplicate.go"), []byte("package snapshots\nfunc (r UpdateMetadataResult) ExtractMetadata(){}\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "missing metadata" {
				copy.GoFiles = nil
			}
			own.meta[path] = copy
			if _, err := own.snapshotMetadataDeclarations(path); err == nil {
				t.Fatalf("%s source accepted", mode)
			}
		})
	}
}

func TestSnapshotMetadataExtractorEmitsSafeMapAndPreservesBuildersAndInventoryPolicy(t *testing.T) {
	g, imp := snapshotMetadataActualNative(t)
	for _, short := range []string{"blockstorage/v2/snapshots", "blockstorage/v3/snapshots"} {
		path := upstreamModule + "/openstack/" + short
		pkg, err := imp.Import(path)
		if err != nil {
			t.Fatal(err)
		}
		decls, err := g.snapshotMetadataDeclarations(path)
		if err != nil {
			t.Fatal(err)
		}
		fn := pkg.Scope().Lookup("UpdateMetadata").(*types.Func)
		e := emitter{pkg: pkg, imports: map[string]string{}}
		if err := emitOperation(&e, fn, decls[fn.Name()], nil); err != nil {
			t.Fatal(err)
		}
		out := e.body.String()
		for _, want := range []string{"(map[string]any,error)", "snapshotmetadata.Extract(result.Result)", "request.Apply(", "request.ValidateCapabilities", "ToSnapshotUpdateMetadataMap", "Metadata", "upstream.UpdateMetadata("} {
			if !strings.Contains(out, want) {
				t.Fatalf("missing %q: %s", want, out)
			}
		}
		for _, forbidden := range []string{"result.ExtractMetadata(", "result.Extract(", "json.Marshal", "ResponseError", "upstream.Get("} {
			if strings.Contains(out, forbidden) {
				t.Fatalf("unsafe/misleading %q: %s", forbidden, out)
			}
		}
		if e.imports["gophercloudsdk/internal/snapshotmetadata"] != "snapshotmetadata" || operationReturnPolicy(fn) != "extract" {
			t.Fatal("safe helper import/return policy missing")
		}
		// The optional inventory marker describes this correction without changing
		// existing return_policy or making the other extract operations opt in.
		encoded, err := json.Marshal(operation{Package: short, Name: fn.Name(), ReturnPolicy: operationReturnPolicy(fn), ResultPolicy: "sdk_snapshot_metadata_object"})
		if err != nil || !strings.Contains(string(encoded), `"result_policy":"sdk_snapshot_metadata_object"`) {
			t.Fatalf("inventory=%s err=%v", encoded, err)
		}
		plain, err := json.Marshal(operation{ReturnPolicy: "extract"})
		if err != nil || strings.Contains(string(plain), "result_policy") {
			t.Fatalf("ordinary inventory=%s err=%v", plain, err)
		}
	}
}

func TestSnapshotMetadataExtractorAcceptsActualPromotedGraphAndKeepsOtherMetadataOperations(t *testing.T) {
	_, imp := snapshotMetadataActualNative(t)
	for _, short := range []string{"blockstorage/v2/snapshots", "blockstorage/v3/snapshots"} {
		pkg, err := imp.Import(upstreamModule + "/openstack/" + short)
		if err != nil {
			t.Fatal(err)
		}
		fn := pkg.Scope().Lookup("UpdateMetadata").(*types.Func)
		if !matchesSnapshotMetadataSignature(pkg, fn) {
			t.Fatalf("compiled native graph rejected: %s", short)
		}
		result := fn.Type().(*types.Signature).Results().At(0).Type()
		methods := types.NewMethodSet(types.NewPointer(result))
		generic, own := methods.Lookup(pkg, "Extract"), methods.Lookup(pkg, "ExtractMetadata")
		if generic == nil || own == nil || len(generic.Index()) != 2 || len(own.Index()) != 1 || snapshotMetadataMap(generic.Obj().Type().(*types.Signature).Results().At(0).Type()) || !snapshotMetadataMap(own.Obj().Type().(*types.Signature).Results().At(0).Type()) {
			t.Fatalf("promoted/own extraction graph changed: %v %v", generic, own)
		}
	}
	other := map[string]map[string]string{
		"blockstorage/v2/volumes": {"SetImageMetadata": "error"}, "blockstorage/v3/volumes": {"SetImageMetadata": "error"},
		"compute/v2/aggregates":           {"SetMetadata": "extract"},
		"compute/v2/servers":              {"CreateMetadatum": "extract", "DeleteMetadatum": "error", "Metadata": "extract", "Metadatum": "extract", "ResetMetadata": "extract", "UpdateMetadata": "extract"},
		"keymanager/v1/secrets":           {"CreateMetadata": "extract", "CreateMetadatum": "error", "DeleteMetadatum": "error", "GetMetadata": "extract", "GetMetadatum": "extract", "UpdateMetadatum": "extract"},
		"orchestration/v1/stackresources": {"Metadata": "extract"},
		"sharedfilesystems/v2/shares":     {"DeleteMetadatum": "error", "GetMetadata": "extract", "GetMetadatum": "extract", "SetMetadata": "extract", "UpdateMetadata": "extract"},
	}
	count := 0
	for short, operations := range other {
		pkg, err := imp.Import(upstreamModule + "/openstack/" + short)
		if err != nil {
			t.Fatal(err)
		}
		for name, policy := range operations {
			fn := pkg.Scope().Lookup(name).(*types.Func)
			if snapshotMetadataExtractor(fn) || operationReturnPolicy(fn) != policy {
				t.Fatalf("unreviewed operation changed %s.%s", short, name)
			}
			selected, sig := operationExtractor(fn)
			if policy == "extract" && (selected != "Extract" || sig == nil) {
				t.Fatalf("native own Extract changed %s.%s", short, name)
			}
			count++
		}
	}
	if count != 21 {
		t.Fatalf("bounded native sibling coverage=%d want 21", count)
	}
}
