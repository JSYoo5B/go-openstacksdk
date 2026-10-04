package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
)

// The pinned Trove extractor asserts the decoded Body before checking its Err.
// Only this reviewed operation receives SDK-owned error-first extraction.
func troveRootEnabledOperation(pkg *types.Package, name string) bool {
	return pkg != nil && pkg.Path() == upstreamModule+"/openstack/db/v1/instances" && name == "IsRootEnabled"
}

func troveRootEnabledExtractor(fn *types.Func) bool {
	if fn == nil || !troveRootEnabledOperation(fn.Pkg(), fn.Name()) {
		return false
	}
	sig := fn.Type().(*types.Signature)
	if sig.Results().Len() != 1 {
		return false
	}
	result, ok := sig.Results().At(0).Type().(*types.Named)
	return ok && result.Obj().Pkg() == fn.Pkg() && result.Obj().Name() == "IsRootEnabledResult"
}

func matchesTroveRootEnabledSignature(pkg *types.Package, fn *types.Func) bool {
	if !troveRootEnabledExtractor(fn) || fn.Pkg() != pkg {
		return false
	}
	sig := fn.Type().(*types.Signature)
	if sig.Recv() != nil || sig.Variadic() || sig.TypeParams().Len() != 0 || sig.Params().Len() != 3 ||
		!isContext(sig.Params().At(0).Type()) || !types.Identical(sig.Params().At(2).Type(), types.Typ[types.String]) ||
		types.TypeString(sig.Params().At(1).Type(), func(p *types.Package) string { return p.Path() }) != "*"+upstreamModule+".ServiceClient" {
		return false
	}
	result := sig.Results().At(0).Type().(*types.Named)
	if result.TypeParams().Len() != 0 || result.NumMethods() != 1 || result.Method(0).Name() != "Extract" {
		return false
	}
	fields, ok := result.Underlying().(*types.Struct)
	if !ok || fields.NumFields() != 1 || !fields.Field(0).Embedded() || fields.Field(0).Name() != "Result" || fields.Tag(0) != "" {
		return false
	}
	base, ok := fields.Field(0).Type().(*types.Named)
	if !ok || base.Obj().Pkg() == nil || base.Obj().Pkg().Path() != upstreamModule || base.Obj().Name() != "Result" {
		return false
	}
	extract := result.Method(0).Type().(*types.Signature)
	return types.Identical(extract.Recv().Type(), result) && !extract.Variadic() && extract.Params().Len() == 0 &&
		extract.Results().Len() == 2 && types.Identical(extract.Results().At(0).Type(), types.Typ[types.Bool]) && isError(extract.Results().At(1).Type())
}

func validateTroveRootEnabledTypes(pkg *types.Package, fn *types.Func) error {
	if fn == nil || !troveRootEnabledOperation(pkg, fn.Name()) {
		return nil
	}
	if !matchesTroveRootEnabledSignature(pkg, fn) {
		return fmt.Errorf("audited Trove root-enabled extractor: native request/result/method graph changed")
	}
	return nil
}

var troveRootEnabledNativeDeclarations = map[string]string{
	"IsRootEnabled":               "ddbdbd12aa8f1555a2d3f9607c2e8406226c0e29ca8905520cba5ea7bb2759c3",
	"IsRootEnabledResult.Extract": "20ed1471b961b5f36cf7d2e791119eb457dc39c93e16f073d0e401e91142aaa7",
	"userRootURL":                 "c92b04277aa6d748a17355e1f24dedf467f4ec90585645c1a74769ee85e4ff57",
}

func validateTroveRootEnabledDeclarations(pkg *types.Package, decls map[string]*ast.FuncDecl) error {
	if !troveRootEnabledOperation(pkg, "IsRootEnabled") {
		return nil
	}
	fn, ok := pkg.Scope().Lookup("IsRootEnabled").(*types.Func)
	if !ok || !matchesTroveRootEnabledSignature(pkg, fn) {
		return fmt.Errorf("audited Trove root-enabled extractor: native request/result/method graph changed")
	}
	for name, expected := range troveRootEnabledNativeDeclarations {
		hash, err := requestDeclarationHash(decls[name])
		if err != nil || hash != expected {
			return fmt.Errorf("audited Trove root-enabled extractor: pinned native declaration %s changed", name)
		}
	}
	return nil
}

// Read only this reviewed leaf package and reject duplicate declarations before
// the general receiver-qualified source map can overwrite them.
func (g *generator) troveRootEnabledDeclarations(path string) (map[string]*ast.FuncDecl, error) {
	if path != upstreamModule+"/openstack/db/v1/instances" {
		return nil, nil
	}
	m, ok := g.meta[path]
	if !ok || m.Dir == "" || len(m.GoFiles) == 0 {
		return nil, fmt.Errorf("audited Trove root-enabled extractor: native source metadata missing")
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
			if _, ok := troveRootEnabledNativeDeclarations[key]; !ok {
				continue
			}
			if result[key] != nil {
				return nil, fmt.Errorf("audited Trove root-enabled extractor: duplicate native declaration %s", key)
			}
			result[key] = fn
		}
	}
	if len(result) != len(troveRootEnabledNativeDeclarations) {
		return nil, fmt.Errorf("audited Trove root-enabled extractor: native declarations missing")
	}
	return result, nil
}
