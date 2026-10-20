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
	signature                                          func(*types.Package, *types.Func) bool
}

var auditedRequestCalls = []auditedRequestCall{
	{
		path: "baremetalintrospection/v1/introspection", operation: "StartIntrospection",
		helper: "startIntrospection", policy: "sdk_query_preserving_start",
		declarationSHA256: "cf3afa01f5c61e99f1713b6f22bbdf6818982efa44fc7791cf30f4610e71f1dd",
		signature:         matchesIntrospectionStartSignature,
	},
	// Native dns zones Share reflects over its ShareOptsBuilder argument instead
	// of calling ToShareMap, so any wrapper builder serializes as an empty body.
	{
		path: "dns/v2/zones", operation: "Share",
		helper: "shareZone", policy: "sdk_builder_body_share",
		declarationSHA256: "8a61e0064fe7922e1e70f20cdb6f7079023b64f6c03983e564ba21e85e3cce13",
		signature:         matchesZoneShareSignature,
	},
	// Native messaging queues Update passes its UpdateOptsBuilder argument
	// directly to the request body, which reflects over the value, so any
	// wrapper builder serializes as an empty object instead of the patch list.
	{
		path: "messaging/v2/queues", operation: "Update",
		helper: "updateQueue", policy: "sdk_builder_body_update",
		declarationSHA256: "9e1844f9565bc6f5166cb8a8edb697ded38eb0392edd490ba806f9f6f7a75c79",
		signature:         matchesQueueUpdateSignature,
	},
	// Native shareaccessrules List concatenates the raw share ID into the
	// share_id query value, so reserved characters inject query parameters.
	{
		path: "sharedfilesystems/v2/shareaccessrules", operation: "List",
		helper: "listAccessRules", policy: "sdk_escaped_query_list",
		declarationSHA256: "ea934b66db7f3a0727f84ec86054bb4ff88f6fbbde7decf31f62c986fcaa4c5a",
		signature:         matchesAccessRuleListSignature,
	},
	// Native snapshot WaitForStatus dereferences the result of Get without a
	// nil check, so a response without the snapshot envelope panics.
	{
		path: "blockstorage/v2/snapshots", operation: "WaitForStatus",
		helper: "waitForStatus", policy: "sdk_nil_guarded_wait",
		declarationSHA256: "a69c9b5988c69c5e5fb1038366e4757042777580d904f4a62a52af5cc4945e84",
		signature:         matchesStatusWaitSignature,
	},
	{
		path: "blockstorage/v3/snapshots", operation: "WaitForStatus",
		helper: "waitForStatus", policy: "sdk_nil_guarded_wait",
		declarationSHA256: "a69c9b5988c69c5e5fb1038366e4757042777580d904f4a62a52af5cc4945e84",
		signature:         matchesStatusWaitSignature,
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
	if !ok || spec.signature == nil || !spec.signature(pkg, fn) {
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

// matchesStatusWaitSignature accepts WaitForStatus(ctx, client, id, status) error.
func matchesStatusWaitSignature(pkg *types.Package, fn *types.Func) bool {
	sig := fn.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 4 || sig.Results().Len() != 1 || !isContext(sig.Params().At(0).Type()) || !isString(sig.Params().At(2).Type()) || !isString(sig.Params().At(3).Type()) || !isError(sig.Results().At(0).Type()) {
		return false
	}
	client := types.TypeString(sig.Params().At(1).Type(), func(p *types.Package) string { return p.Path() })
	return client == "*"+upstreamModule+".ServiceClient"
}

// matchesZoneShareSignature accepts Share(ctx, client, zoneID, opts) ZoneShareResult
// where opts is a single-method ToShareMap builder implemented by ShareZoneOpts.
func matchesZoneShareSignature(pkg *types.Package, fn *types.Func) bool {
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
	if !localNamed(sig.Params().At(3).Type(), "ShareOptsBuilder") || !localNamed(sig.Results().At(0).Type(), "ZoneShareResult") {
		return false
	}
	builder, ok := sig.Params().At(3).Type().Underlying().(*types.Interface)
	if !ok || builder.NumMethods() != 1 || builder.Method(0).Name() != "ToShareMap" {
		return false
	}
	body := builder.Method(0).Type().(*types.Signature)
	if body.Params().Len() != 0 || body.Results().Len() != 2 || !isError(body.Results().At(1).Type()) {
		return false
	}
	if m, ok := body.Results().At(0).Type().(*types.Map); !ok || !isString(m.Key()) {
		return false
	}
	options := pkg.Scope().Lookup("ShareZoneOpts")
	if options == nil || !types.Implements(options.Type(), builder) {
		return false
	}
	fields, ok := options.Type().Underlying().(*types.Struct)
	if !ok || fields.NumFields() != 1 || fields.Field(0).Name() != "TargetProjectID" || !isString(fields.Field(0).Type()) || reflect.StructTag(fields.Tag(0)).Get("json") != "target_project_id" {
		return false
	}
	result, ok := sig.Results().At(0).Type().Underlying().(*types.Struct)
	if !ok {
		return false
	}
	extractor, _, _ := types.LookupFieldOrMethod(sig.Results().At(0).Type(), true, pkg, "Extract")
	return extractor != nil && result.NumFields() > 0
}

// matchesQueueUpdateSignature accepts Update(ctx, client, queueName, opts) UpdateResult
// where opts is a single-method ToQueueUpdateMap builder implemented by
// BatchUpdateOpts, a slice of UpdateOpts{Op, Path, Value}.
func matchesQueueUpdateSignature(pkg *types.Package, fn *types.Func) bool {
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
	if !localNamed(sig.Params().At(3).Type(), "UpdateOptsBuilder") || !localNamed(sig.Results().At(0).Type(), "UpdateResult") {
		return false
	}
	builder, ok := sig.Params().At(3).Type().Underlying().(*types.Interface)
	if !ok || builder.NumMethods() != 1 || builder.Method(0).Name() != "ToQueueUpdateMap" {
		return false
	}
	body := builder.Method(0).Type().(*types.Signature)
	if body.Params().Len() != 0 || body.Results().Len() != 2 || !isError(body.Results().At(1).Type()) {
		return false
	}
	list, ok := body.Results().At(0).Type().(*types.Slice)
	if !ok {
		return false
	}
	if m, ok := list.Elem().(*types.Map); !ok || !isString(m.Key()) {
		return false
	}
	batch := pkg.Scope().Lookup("BatchUpdateOpts")
	if batch == nil || !types.Implements(batch.Type(), builder) {
		return false
	}
	batchSlice, ok := batch.Type().Underlying().(*types.Slice)
	if !ok || !localNamed(batchSlice.Elem(), "UpdateOpts") {
		return false
	}
	fields, ok := batchSlice.Elem().Underlying().(*types.Struct)
	if !ok || fields.NumFields() != 3 {
		return false
	}
	for i, want := range []struct{ name, tag string }{{"Op", "op"}, {"Path", "path"}, {"Value", "value"}} {
		if fields.Field(i).Name() != want.name || reflect.StructTag(fields.Tag(i)).Get("json") != want.tag {
			return false
		}
	}
	if !localNamed(fields.Field(0).Type(), "UpdateOp") || !isString(fields.Field(0).Type().Underlying()) || !isString(fields.Field(1).Type()) {
		return false
	}
	result, ok := sig.Results().At(0).Type().Underlying().(*types.Struct)
	if !ok {
		return false
	}
	extractor, _, _ := types.LookupFieldOrMethod(sig.Results().At(0).Type(), true, pkg, "Extract")
	return extractor != nil && result.NumFields() > 0
}

// matchesAccessRuleListSignature accepts List(ctx, client, shareID) ListResult
// where ListResult is a local struct with an Extract method.
func matchesAccessRuleListSignature(pkg *types.Package, fn *types.Func) bool {
	sig := fn.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 3 || sig.Results().Len() != 1 || !isContext(sig.Params().At(0).Type()) || !isString(sig.Params().At(2).Type()) {
		return false
	}
	client := types.TypeString(sig.Params().At(1).Type(), func(p *types.Package) string { return p.Path() })
	if client != "*"+upstreamModule+".ServiceClient" {
		return false
	}
	named, ok := sig.Results().At(0).Type().(*types.Named)
	if !ok || named.Obj().Pkg() != pkg || named.Obj().Name() != "ListResult" {
		return false
	}
	result, ok := named.Underlying().(*types.Struct)
	if !ok || result.NumFields() == 0 {
		return false
	}
	extractor, _, _ := types.LookupFieldOrMethod(named, true, pkg, "Extract")
	return extractor != nil
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
