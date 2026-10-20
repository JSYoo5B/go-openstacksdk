package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const backupUpdateFixture = `package backups
import "context"
import gophercloud "github.com/gophercloud/gophercloud/v2"
type Backup struct{ID string}
type UpdateResult struct{gophercloud.Result}
func(UpdateResult)Extract()(*Backup,error){return nil,nil}
type UpdateOpts struct{
 Name *string ` + "`json:\"name,omitempty\"`" + `
 Description *string ` + "`json:\"description,omitempty\"`" + `
 Metadata map[string]string ` + "`json:\"metadata,omitempty\"`" + `
}
func(UpdateOpts)ToBackupUpdateMap()(map[string]any,error){return nil,nil}
type UpdateOptsBuilder interface{ToBackupUpdateMap()(map[string]any,error)}
func Update(ctx context.Context,client *gophercloud.ServiceClient,id string,opts UpdateOptsBuilder)UpdateResult{return UpdateResult{}}
func Create(ctx context.Context,client *gophercloud.ServiceClient,id string,opts UpdateOptsBuilder)UpdateResult{return UpdateResult{}}
`

func TestBackupUpdateBodyRepairHasExactPackageAndBuilderScope(t *testing.T) {
	for _, path := range []string{backupUpdateNativePath, upstreamModule + "/openstack/blockstorage/v2/backups", upstreamModule + "/openstack/blockstorage/v3/volumes", "fixture/blockstorage/v3/backups"} {
		pkg, decls := typedCollectionFixture(t, path, backupUpdateFixture)
		fn := pkg.Scope().Lookup("Update").(*types.Func)
		e := emitter{pkg: pkg, imports: map[string]string{}}
		if err := emitOperation(&e, fn, decls["Update"], nil); err != nil {
			t.Fatal(err)
		}
		want := path == backupUpdateNativePath
		if matchesBackupUpdateSignature(pkg, fn) != want || strings.Contains(e.body.String(), `value0=map[string]any{"backup":value0}`) != want || strings.Contains(e.body.String(), "maps.Clone(b.base.Metadata)") != want {
			t.Fatalf("wrong repair scope for %s: %s", path, e.body.String())
		}
		method := pkg.Scope().Lookup("UpdateOptsBuilder").Type().Underlying().(*types.Interface).Method(0)
		for _, base := range []types.Type{types.Typ[types.String], types.NewPointer(pkg.Scope().Lookup("UpdateOpts").Type())} {
			if backupUpdateBody(pkg, builder{base: base}, method) {
				t.Fatal("unreviewed builder selected")
			}
		}
		if err := validateBackupUpdateTypes(pkg, pkg.Scope().Lookup("Create").(*types.Func)); err != nil {
			t.Fatal("other operation intercepted", err)
		}
	}
}

func TestBackupUpdateBodyRepairRejectsNativeTypeAndTagDriftBeforeEmission(t *testing.T) {
	for _, tc := range []struct{ name, before, after string }{
		{"request parameter", "id string,opts", "id string,extra string,opts"},
		{"pointer result", ")UpdateResult{return UpdateResult{}", ")*UpdateResult{return &UpdateResult{}"},
		{"concrete input", "opts UpdateOptsBuilder", "opts UpdateOpts"},
		{"pointer builder", "func(UpdateOpts)To", "func(*UpdateOpts)To"},
		{"builder argument", "interface{ToBackupUpdateMap()", "interface{ToBackupUpdateMap(string)"},
		{"builder variadic argument", "interface{ToBackupUpdateMap()", "interface{ToBackupUpdateMap(...string)"},
		{"builder return type", "interface{ToBackupUpdateMap()(map[string]any,error)}", "interface{ToBackupUpdateMap()(map[string]string,error)}"},
		{"extra options method", "type UpdateOptsBuilder", "func(UpdateOpts)Other(){}\ntype UpdateOptsBuilder"},
		{"name member", "Name *string", "Name string"},
		{"description member", "Description *string", "Description string"},
		{"metadata member", "Metadata map[string]string", "Metadata map[string]any"},
		{"metadata tag", "metadata,omitempty", "metadata"},
		{"additional validation tag", "json:\"metadata,omitempty\"", "json:\"metadata,omitempty\" required:\"true\""},
		{"additional member", " Metadata map[string]string", " Extra bool\n Metadata map[string]string"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(backupUpdateFixture, tc.before, tc.after, 1)
			if source == backupUpdateFixture {
				t.Fatal("mutation did not apply")
			}
			pkg, decls := typedCollectionFixture(t, backupUpdateNativePath, source)
			fn := pkg.Scope().Lookup("Update").(*types.Func)
			e := emitter{pkg: pkg, imports: map[string]string{}}
			if err := emitOperation(&e, fn, decls["Update"], nil); err == nil || e.body.Len() != 0 {
				t.Fatalf("drift emitted a partial wrapper: err=%v body=%s", err, e.body.String())
			}
		})
	}
}

func TestBackupUpdateBodyRepairGuardsPinnedSourceAndRegeneratesExactWrapper(t *testing.T) {
	g, imp := snapshotMetadataActualNative(t)
	pkg, err := imp.Import(backupUpdateNativePath)
	if err != nil {
		t.Fatal(err)
	}
	decls, err := g.backupUpdateDeclarations(backupUpdateNativePath)
	if err != nil || len(decls) != 2 || len(backupUpdateNativeDeclarations) != 2 {
		t.Fatalf("declarations=%d err=%v", len(decls), err)
	}
	if err := validateBackupUpdateDeclarations(pkg, decls); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, before, after string }{
		{"UpdateOpts.ToBackupUpdateMap", `BuildRequestBody(opts, "")`, `BuildRequestBody(opts, "backup")`},
		{"Update", "OkCodes: []int{200}", "OkCodes: []int{202}"},
	} {
		source := snapshotMetadataSource(t, decls[tc.name])
		changed := strings.Replace(source, tc.before, tc.after, 1)
		if changed == source {
			t.Fatalf("native mutation did not apply: %s", tc.name)
		}
		file, err := parser.ParseFile(token.NewFileSet(), "changed.go", "package backups\n"+changed, 0)
		if err != nil {
			t.Fatal(err)
		}
		own := map[string]*ast.FuncDecl{}
		for name, decl := range decls {
			own[name] = decl
		}
		own[tc.name] = file.Decls[0].(*ast.FuncDecl)
		if err := validateBackupUpdateDeclarations(pkg, own); err == nil || !strings.Contains(err.Error(), tc.name) {
			t.Fatalf("changed native contract accepted: %s %v", tc.name, err)
		}
		delete(own, tc.name)
		if err := validateBackupUpdateDeclarations(pkg, own); err == nil {
			t.Fatalf("missing declaration accepted: %s", tc.name)
		}
	}
	entry := g.meta[backupUpdateNativePath]
	duplicate := ""
	for _, name := range entry.GoFiles {
		data, err := os.ReadFile(filepath.Join(entry.Dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "func Update(") {
			duplicate = name
			break
		}
	}
	if duplicate == "" {
		t.Fatal("native Update source not found")
	}
	for _, files := range [][]string{nil, {"missing.go"}, append(append([]string(nil), entry.GoFiles...), duplicate)} {
		own := generator{meta: map[string]metadata{}}
		copy := entry
		copy.GoFiles = files
		own.meta[backupUpdateNativePath] = copy
		if _, err := own.backupUpdateDeclarations(backupUpdateNativePath); err == nil {
			t.Fatal("missing or duplicate native source accepted", files)
		}
	}
	if values, err := g.backupUpdateDeclarations(upstreamModule + "/openstack/blockstorage/v2/backups"); err != nil || values != nil {
		t.Fatalf("unreviewed package source read: %v %v", values, err)
	}
	g.root = t.TempDir()
	if err := g.generate(backupUpdateNativePath); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(filepath.Join(g.root, "blockstorage/v3/backups/api_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	tracked, err := os.ReadFile(filepath.Join("../../..", "blockstorage/v3/backups/api_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated, tracked) {
		t.Fatal("tracked backup wrapper differs from audited native regeneration")
	}
	for _, operation := range g.inventory.Operations {
		if operation.Issue != "" {
			t.Fatalf("regeneration dropped operation %s: %s", operation.Name, operation.Issue)
		}
	}
}
