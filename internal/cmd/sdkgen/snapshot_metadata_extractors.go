package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
)

// Only these two native operations have an unrelated promoted Extract method
// and an unsafe own ExtractMetadata method in the pinned release.
func snapshotMetadataOperation(pkg *types.Package, name string) bool {
	if pkg == nil || name != "UpdateMetadata" {
		return false
	}
	path := sdkPath(pkg.Path())
	return path == "blockstorage/v2/snapshots" || path == "blockstorage/v3/snapshots"
}

func snapshotMetadataExtractor(fn *types.Func) bool {
	if fn == nil || !snapshotMetadataOperation(fn.Pkg(), fn.Name()) {
		return false
	}
	sig := fn.Type().(*types.Signature)
	if sig.Results().Len() != 1 {
		return false
	}
	result, ok := sig.Results().At(0).Type().(*types.Named)
	return ok && result.Obj().Pkg() == fn.Pkg() && result.Obj().Name() == "UpdateMetadataResult"
}

func snapshotMetadataMap(t types.Type) bool {
	return types.Identical(t, types.NewMap(types.Typ[types.String], types.NewInterfaceType(nil, nil).Complete()))
}

func snapshotMetadataMapSignature(sig *types.Signature) bool {
	return !sig.Variadic() && sig.Params().Len() == 0 && sig.Results().Len() == 2 && snapshotMetadataMap(sig.Results().At(0).Type()) && isError(sig.Results().At(1).Type())
}

func matchesSnapshotMetadataSignature(pkg *types.Package, fn *types.Func) bool {
	if !snapshotMetadataExtractor(fn) {
		return false
	}
	sig := fn.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 4 || !isContext(sig.Params().At(0).Type()) || types.TypeString(sig.Params().At(1).Type(), func(p *types.Package) string { return p.Path() }) != "*"+upstreamModule+".ServiceClient" || !types.Identical(sig.Params().At(2).Type(), types.Typ[types.String]) {
		return false
	}
	localNamed := func(t types.Type, name string) *types.Named {
		n, ok := t.(*types.Named)
		if !ok || n.Obj().Pkg() != pkg || n.Obj().Name() != name {
			return nil
		}
		return n
	}
	builder := localNamed(sig.Params().At(3).Type(), "UpdateMetadataOptsBuilder")
	if builder == nil {
		return false
	}
	iface, ok := builder.Underlying().(*types.Interface)
	if !ok || iface.NumEmbeddeds() != 0 || iface.NumMethods() != 1 || iface.Method(0).Name() != "ToSnapshotUpdateMetadataMap" || !snapshotMetadataMapSignature(iface.Method(0).Type().(*types.Signature)) {
		return false
	}
	optsObject := pkg.Scope().Lookup("UpdateMetadataOpts")
	if optsObject == nil {
		return false
	}
	opts := localNamed(optsObject.Type(), "UpdateMetadataOpts")
	if opts == nil || opts.NumMethods() != 1 || opts.Method(0).Name() != "ToSnapshotUpdateMetadataMap" || !types.Implements(opts, iface) {
		return false
	}
	fields, ok := opts.Underlying().(*types.Struct)
	if !ok || fields.NumFields() != 1 || fields.Field(0).Embedded() || fields.Field(0).Name() != "Metadata" || !snapshotMetadataMap(fields.Field(0).Type()) || fields.Tag(0) != `json:"metadata,omitempty"` {
		return false
	}
	optSig := opts.Method(0).Type().(*types.Signature)
	if !types.Identical(optSig.Recv().Type(), opts) || !snapshotMetadataMapSignature(optSig) {
		return false
	}
	result := localNamed(sig.Results().At(0).Type(), "UpdateMetadataResult")
	if result == nil || result.NumMethods() != 1 || result.Method(0).Name() != "ExtractMetadata" {
		return false
	}
	ex := result.Method(0).Type().(*types.Signature)
	if !types.Identical(ex.Recv().Type(), result) || !snapshotMetadataMapSignature(ex) {
		return false
	}
	embedded := func(n *types.Named) types.Type {
		st, ok := n.Underlying().(*types.Struct)
		if !ok || st.NumFields() != 1 || !st.Field(0).Embedded() {
			return nil
		}
		return st.Field(0).Type()
	}
	common := localNamed(embedded(result), "commonResult")
	if common == nil || common.NumMethods() != 1 || common.Method(0).Name() != "Extract" {
		return false
	}
	base := embedded(common)
	if base == nil || types.TypeString(base, func(p *types.Package) string { return p.Path() }) != upstreamModule+".Result" {
		return false
	}
	commonSig := common.Method(0).Type().(*types.Signature)
	if !types.Identical(commonSig.Recv().Type(), common) || commonSig.Variadic() || commonSig.Params().Len() != 0 || commonSig.Results().Len() != 2 || !isError(commonSig.Results().At(1).Type()) {
		return false
	}
	modelObject := pkg.Scope().Lookup("Snapshot")
	if modelObject == nil || localNamed(modelObject.Type(), "Snapshot") == nil || !types.Identical(commonSig.Results().At(0).Type(), types.NewPointer(modelObject.Type())) {
		return false
	}
	return true
}

func validateSnapshotMetadataTypes(pkg *types.Package, fn *types.Func) error {
	if !snapshotMetadataOperation(pkg, fn.Name()) {
		return nil
	}
	if !matchesSnapshotMetadataSignature(pkg, fn) {
		return fmt.Errorf("audited Snapshot metadata extractor %s: native request/result/method graph changed", sdkPath(pkg.Path()))
	}
	return nil
}

func validateSnapshotMetadataDeclarations(pkg *types.Package, decls map[string]*ast.FuncDecl) error {
	wanted := snapshotMetadataNativeDeclarations[sdkPath(pkg.Path())]
	if wanted == nil {
		return nil
	}
	fn, ok := pkg.Scope().Lookup("UpdateMetadata").(*types.Func)
	if !ok || !matchesSnapshotMetadataSignature(pkg, fn) {
		return fmt.Errorf("audited Snapshot metadata extractor %s: native request/result/method graph changed", sdkPath(pkg.Path()))
	}
	for name, expected := range wanted {
		hash, err := requestDeclarationHash(decls[name])
		if err != nil || hash != expected {
			return fmt.Errorf("audited Snapshot metadata extractor %s: pinned native declaration %s changed", sdkPath(pkg.Path()), name)
		}
	}
	return nil
}

// Read only the two reviewed leaf packages and reject duplicate declarations
// before the general receiver-qualified source map can overwrite them.
func (g *generator) snapshotMetadataDeclarations(path string) (map[string]*ast.FuncDecl, error) {
	wanted := snapshotMetadataNativeDeclarations[sdkPath(path)]
	if wanted == nil {
		return nil, nil
	}
	m, ok := g.meta[path]
	if !ok || m.Dir == "" || len(m.GoFiles) == 0 {
		return nil, fmt.Errorf("audited Snapshot metadata extractor: native source metadata missing")
	}
	result := map[string]*ast.FuncDecl{}
	for _, name := range m.GoFiles {
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(m.Dir, name), nil, 0)
		if err != nil {
			return nil, err
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			key := identityDeclarationKey(fn)
			if _, ok := wanted[key]; !ok {
				continue
			}
			if result[key] != nil {
				return nil, fmt.Errorf("audited Snapshot metadata extractor: duplicate native declaration %s", key)
			}
			result[key] = fn
		}
	}
	if len(result) != len(wanted) {
		return nil, fmt.Errorf("audited Snapshot metadata extractor: native declarations missing")
	}
	return result, nil
}

var snapshotMetadataNativeDeclarations = map[string]map[string]string{
	"blockstorage/v2/snapshots": {
		"UpdateMetadata": "8c693d895fcb9c0ae4269f572bf2d8044578c681c49e8c2932535a8450d21178",
		"UpdateMetadataOpts.ToSnapshotUpdateMetadataMap": "8099ebc4809989e76066eec6fcce34992705d8ed9b11e7570c36fb7d01d4a08b",
		"UpdateMetadataResult.ExtractMetadata":           "73275b34eaba32849bb7119b457cd417d364b0a3901116fec95a5ca6b4c98e24",
		"commonResult.Extract":                           "655d33d33c2d3e6a0312f8edcd8e24337d7a2ebe61bdc865c74adcac4f73a66e",
		"updateMetadataURL":                              "75a8e67c8a0cae282222cf29f7cf4f73c52ae6d2ef172720cf173a85c394723d",
		"metadataURL":                                    "a6e835d82ad27f0325694e72cb73bd46227667ad1f6925d24de550acdb0c9085",
	},
	"blockstorage/v3/snapshots": {
		"UpdateMetadata": "8c693d895fcb9c0ae4269f572bf2d8044578c681c49e8c2932535a8450d21178",
		"UpdateMetadataOpts.ToSnapshotUpdateMetadataMap": "8099ebc4809989e76066eec6fcce34992705d8ed9b11e7570c36fb7d01d4a08b",
		"UpdateMetadataResult.ExtractMetadata":           "73275b34eaba32849bb7119b457cd417d364b0a3901116fec95a5ca6b4c98e24",
		"commonResult.Extract":                           "655d33d33c2d3e6a0312f8edcd8e24337d7a2ebe61bdc865c74adcac4f73a66e",
		"updateMetadataURL":                              "75a8e67c8a0cae282222cf29f7cf4f73c52ae6d2ef172720cf173a85c394723d",
		"metadataURL":                                    "a6e835d82ad27f0325694e72cb73bd46227667ad1f6925d24de550acdb0c9085",
	},
}
