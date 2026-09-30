package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"reflect"
)

// Native request defects require an explicit, inspected call override. A pinned
// declaration and its dependent input/result types must still match before the
// generator can redirect the operation to an SDK-owned helper.
type auditedRequestCall struct {
	path, operation, helper, policy, declarationSHA256 string
}

var auditedRequestCalls = []auditedRequestCall{
	{
		path: "baremetalintrospection/v1/introspection", operation: "StartIntrospection",
		helper: "startIntrospection", policy: "sdk_query_preserving_start",
		declarationSHA256: "cf3afa01f5c61e99f1713b6f22bbdf6818982efa44fc7791cf30f4610e71f1dd",
	},
}

func requestCallOverride(pkg *types.Package, name string) *auditedRequestCall {
	for i := range auditedRequestCalls {
		spec := &auditedRequestCalls[i]
		if sdkPath(pkg.Path()) == spec.path && name == spec.operation {
			return spec
		}
	}
	return nil
}

func validateAuditedRequestCalls(pkg *types.Package, decls map[string]*ast.FuncDecl) error {
	for _, spec := range auditedRequestCalls {
		if sdkPath(pkg.Path()) == spec.path {
			if err := validateAuditedRequestCall(pkg, decls, spec); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateAuditedRequestCall(pkg *types.Package, decls map[string]*ast.FuncDecl, spec auditedRequestCall) error {
	mismatch := func(reason string) error {
		return fmt.Errorf("audited request call %s.%s: %s; review the native contract before updating the override", spec.path, spec.operation, reason)
	}
	fn, ok := pkg.Scope().Lookup(spec.operation).(*types.Func)
	if !ok || !matchesIntrospectionStartSignature(pkg, fn) {
		return mismatch("native signature or input/result types changed")
	}
	hash, err := requestDeclarationHash(decls[spec.operation])
	if err != nil {
		return mismatch(err.Error())
	}
	if hash != spec.declarationSHA256 {
		return mismatch("pinned native declaration changed")
	}
	return nil
}

// The override is deliberately specific to v2.15.0's StartIntrospection rather
// than a general replacement for native request execution.
func matchesIntrospectionStartSignature(pkg *types.Package, fn *types.Func) bool {
	sig := fn.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 4 || sig.Results().Len() != 1 || !isContext(sig.Params().At(0).Type()) || !isString(sig.Params().At(2).Type()) {
		return false
	}
	client := types.TypeString(sig.Params().At(1).Type(), func(p *types.Package) string { return p.Path() })
	if client != "*"+upstreamModule+".ServiceClient" {
		return false
	}
	localNamed := func(t types.Type, name string) bool {
		named, ok := t.(*types.Named)
		return ok && named.Obj().Pkg() == pkg && named.Obj().Name() == name
	}
	if !localNamed(sig.Params().At(3).Type(), "StartOptsBuilder") || !localNamed(sig.Results().At(0).Type(), "StartResult") {
		return false
	}
	builder, ok := sig.Params().At(3).Type().Underlying().(*types.Interface)
	if !ok || builder.NumMethods() != 1 || builder.Method(0).Name() != "ToStartIntrospectionQuery" {
		return false
	}
	query := builder.Method(0).Type().(*types.Signature)
	if query.Params().Len() != 0 || query.Results().Len() != 2 || !isString(query.Results().At(0).Type()) || !isError(query.Results().At(1).Type()) {
		return false
	}
	options := pkg.Scope().Lookup("StartOpts")
	if options == nil {
		return false
	}
	fields, ok := options.Type().Underlying().(*types.Struct)
	if !ok || fields.NumFields() != 1 || fields.Field(0).Name() != "ManageBoot" || reflect.StructTag(fields.Tag(0)).Get("q") != "manage_boot" {
		return false
	}
	manageBoot, ok := fields.Field(0).Type().(*types.Pointer)
	if !ok || !types.Identical(manageBoot.Elem(), types.Typ[types.Bool]) || !types.Implements(options.Type(), builder) {
		return false
	}
	result, ok := sig.Results().At(0).Type().Underlying().(*types.Struct)
	if !ok || result.NumFields() != 1 || !result.Field(0).Embedded() {
		return false
	}
	embedded := types.TypeString(result.Field(0).Type(), func(p *types.Package) string { return p.Path() })
	if embedded != upstreamModule+".ErrResult" {
		return false
	}
	extractor, _, _ := types.LookupFieldOrMethod(sig.Results().At(0).Type(), true, pkg, "ExtractErr")
	if extractor == nil {
		return false
	}
	extract, ok := extractor.Type().(*types.Signature)
	return ok && extract.Params().Len() == 0 && extract.Results().Len() == 1 && isError(extract.Results().At(0).Type())
}

// ParseFile omits comments, and formatting discards whitespace-only changes.
// Hashing the entire declaration also detects a native fix to the lost query.
func requestDeclarationHash(decl *ast.FuncDecl) (string, error) {
	if decl == nil || decl.Body == nil {
		return "", fmt.Errorf("native declaration is missing")
	}
	var source bytes.Buffer
	if err := format.Node(&source, token.NewFileSet(), decl); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(source.Bytes())), nil
}
