package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
)

const backupUpdateNativePath = upstreamModule + "/openstack/blockstorage/v3/backups"

func backupUpdateOperation(pkg *types.Package, name string) bool {
	return pkg != nil && pkg.Path() == backupUpdateNativePath && name == "Update"
}

// The pinned builder emits a flat object and omits explicit empty metadata.
// Only this audited value builder receives the Cinder backup envelope repair.
func backupUpdateBody(pkg *types.Package, b builder, method *types.Func) bool {
	if !backupUpdateOperation(pkg, "Update") || method == nil || method.Name() != "ToBackupUpdateMap" {
		return false
	}
	named, ok := b.base.(*types.Named)
	return ok && named.Obj().Pkg() == pkg && named.Obj().Name() == "UpdateOpts"
}

func matchesBackupUpdateSignature(pkg *types.Package, fn *types.Func) bool {
	if fn == nil || fn.Pkg() != pkg || !backupUpdateOperation(pkg, fn.Name()) {
		return false
	}
	sig := fn.Type().(*types.Signature)
	if sig.Recv() != nil || sig.Variadic() || sig.TypeParams().Len() != 0 || sig.Params().Len() != 4 || sig.Results().Len() != 1 ||
		!isContext(sig.Params().At(0).Type()) || !types.Identical(sig.Params().At(2).Type(), types.Typ[types.String]) ||
		types.TypeString(sig.Params().At(1).Type(), func(p *types.Package) string { return p.Path() }) != "*"+upstreamModule+".ServiceClient" {
		return false
	}
	local := func(t types.Type, name string) *types.Named {
		n, ok := t.(*types.Named)
		if !ok || n.Obj().Pkg() != pkg || n.Obj().Name() != name || n.TypeParams().Len() != 0 {
			return nil
		}
		return n
	}
	if local(sig.Results().At(0).Type(), "UpdateResult") == nil {
		return false
	}
	b := local(sig.Params().At(3).Type(), "UpdateOptsBuilder")
	if b == nil {
		return false
	}
	iface, ok := b.Underlying().(*types.Interface)
	if !ok || iface.NumEmbeddeds() != 0 || iface.NumMethods() != 1 || iface.Method(0).Name() != "ToBackupUpdateMap" || !snapshotMetadataMapSignature(iface.Method(0).Type().(*types.Signature)) {
		return false
	}
	obj := pkg.Scope().Lookup("UpdateOpts")
	if obj == nil {
		return false
	}
	opts := local(obj.Type(), "UpdateOpts")
	if opts == nil || opts.NumMethods() != 1 || opts.Method(0).Name() != "ToBackupUpdateMap" || !types.Implements(opts, iface) {
		return false
	}
	method := opts.Method(0).Type().(*types.Signature)
	if !types.Identical(method.Recv().Type(), opts) || !snapshotMetadataMapSignature(method) {
		return false
	}
	fields, ok := opts.Underlying().(*types.Struct)
	if !ok || fields.NumFields() != 3 {
		return false
	}
	for i, name := range []string{"Name", "Description", "Metadata"} {
		field := fields.Field(i)
		want := types.Type(types.NewPointer(types.Typ[types.String]))
		tag := []string{`json:"name,omitempty"`, `json:"description,omitempty"`, `json:"metadata,omitempty"`}[i]
		if i == 2 {
			want = types.NewMap(types.Typ[types.String], types.Typ[types.String])
		}
		if field.Embedded() || field.Name() != name || !types.Identical(field.Type(), want) || fields.Tag(i) != tag {
			return false
		}
	}
	return true
}

func validateBackupUpdateTypes(pkg *types.Package, fn *types.Func) error {
	if fn == nil || !backupUpdateOperation(pkg, fn.Name()) {
		return nil
	}
	if !matchesBackupUpdateSignature(pkg, fn) {
		return fmt.Errorf("audited Backup update body: native request/options/builder graph changed")
	}
	return nil
}

var backupUpdateNativeDeclarations = map[string]string{
	"Update":                       "9af2e6df5ab05457578eaa1c2bf79eeb1aa2b9ca7e40a6a5e2659ec3ddb70acb",
	"UpdateOpts.ToBackupUpdateMap": "05c926a834f467fe4c12a8de035f4ecb6b73fe2ce06880356b45eb22ad4534cc",
}

func validateBackupUpdateDeclarations(pkg *types.Package, decls map[string]*ast.FuncDecl) error {
	if !backupUpdateOperation(pkg, "Update") {
		return nil
	}
	fn, ok := pkg.Scope().Lookup("Update").(*types.Func)
	if !ok || !matchesBackupUpdateSignature(pkg, fn) {
		return fmt.Errorf("audited Backup update body: native request/options/builder graph changed")
	}
	for name, expected := range backupUpdateNativeDeclarations {
		hash, err := requestDeclarationHash(decls[name])
		if err != nil || hash != expected {
			return fmt.Errorf("audited Backup update body: pinned native declaration %s changed", name)
		}
	}
	return nil
}

// Reject missing or duplicate qualified declarations before the general
// source map can overwrite them; an upstream fix must be reviewed explicitly.
func (g *generator) backupUpdateDeclarations(path string) (map[string]*ast.FuncDecl, error) {
	if path != backupUpdateNativePath {
		return nil, nil
	}
	m, ok := g.meta[path]
	if !ok || m.Dir == "" || len(m.GoFiles) == 0 {
		return nil, fmt.Errorf("audited Backup update body: native source metadata missing")
	}
	result := map[string]*ast.FuncDecl{}
	for _, name := range m.GoFiles {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(m.Dir, name), nil, 0)
		if err != nil {
			return nil, err
		}
		for _, raw := range file.Decls {
			fn, ok := raw.(*ast.FuncDecl)
			if !ok {
				continue
			}
			key := identityDeclarationKey(fn)
			if _, ok := backupUpdateNativeDeclarations[key]; !ok {
				continue
			}
			if result[key] != nil {
				return nil, fmt.Errorf("audited Backup update body: duplicate native declaration %s", key)
			}
			result[key] = fn
		}
	}
	if len(result) != len(backupUpdateNativeDeclarations) {
		return nil, fmt.Errorf("audited Backup update body: native declarations missing")
	}
	return result, nil
}
