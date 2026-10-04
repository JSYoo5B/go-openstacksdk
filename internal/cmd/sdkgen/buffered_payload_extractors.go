package main

import (
	"fmt"
	"go/ast"
	"go/types"
)

// A result's streaming Body does not imply that Extract leaves it open.
// Barbican's pinned payload extractor consumes and closes its response body.
func bufferedPayloadOperation(pkg *types.Package, name string) bool {
	return pkg != nil && sdkPath(pkg.Path()) == "keymanager/v1/secrets" && name == "GetPayload"
}

func bufferedPayloadSignature(sig *types.Signature) bool {
	return !sig.Variadic() && sig.Params().Len() == 0 && sig.Results().Len() == 2 &&
		types.Identical(sig.Results().At(0).Type(), types.NewMap(types.Typ[types.String], types.Typ[types.String])) && isError(sig.Results().At(1).Type())
}

func matchesBufferedPayloadSignature(pkg *types.Package, fn *types.Func) bool {
	if fn == nil || !bufferedPayloadOperation(pkg, fn.Name()) {
		return false
	}
	sig := fn.Type().(*types.Signature)
	if sig.Variadic() || sig.Params().Len() != 4 || sig.Results().Len() != 1 ||
		!isContext(sig.Params().At(0).Type()) || !types.Identical(sig.Params().At(2).Type(), types.Typ[types.String]) ||
		types.TypeString(sig.Params().At(1).Type(), func(p *types.Package) string { return p.Path() }) != "*"+upstreamModule+".ServiceClient" {
		return false
	}
	localNamed := func(value types.Type, name string) *types.Named {
		named, ok := value.(*types.Named)
		if !ok || named.Obj().Pkg() != pkg || named.Obj().Name() != name {
			return nil
		}
		return named
	}
	builder := localNamed(sig.Params().At(3).Type(), "GetPayloadOptsBuilder")
	if builder == nil {
		return false
	}
	iface, ok := builder.Underlying().(*types.Interface)
	if !ok || iface.NumEmbeddeds() != 0 || iface.NumMethods() != 1 || iface.Method(0).Name() != "ToSecretPayloadGetParams" ||
		!bufferedPayloadSignature(iface.Method(0).Type().(*types.Signature)) {
		return false
	}
	optsObject := pkg.Scope().Lookup("GetPayloadOpts")
	if optsObject == nil {
		return false
	}
	opts := localNamed(optsObject.Type(), "GetPayloadOpts")
	if opts == nil || opts.NumMethods() != 1 || opts.Method(0).Name() != "ToSecretPayloadGetParams" || !types.Implements(opts, iface) {
		return false
	}
	fields, ok := opts.Underlying().(*types.Struct)
	if !ok || fields.NumFields() != 1 || fields.Field(0).Embedded() || fields.Field(0).Name() != "PayloadContentType" ||
		!types.Identical(fields.Field(0).Type(), types.Typ[types.String]) || fields.Tag(0) != `h:"Accept"` {
		return false
	}
	method := opts.Method(0).Type().(*types.Signature)
	if !types.Identical(method.Recv().Type(), opts) || !bufferedPayloadSignature(method) {
		return false
	}
	result := localNamed(sig.Results().At(0).Type(), "PayloadResult")
	if result == nil || result.NumMethods() != 1 || result.Method(0).Name() != "Extract" {
		return false
	}
	fields, ok = result.Underlying().(*types.Struct)
	if !ok || fields.NumFields() != 2 || !fields.Field(0).Embedded() || fields.Field(0).Name() != "Result" ||
		fields.Field(1).Embedded() || fields.Field(1).Name() != "Body" || fields.Tag(0) != "" || fields.Tag(1) != "" {
		return false
	}
	qualified := func(value types.Type) string {
		return types.TypeString(value, func(p *types.Package) string { return p.Path() })
	}
	if qualified(fields.Field(0).Type()) != upstreamModule+".Result" || qualified(fields.Field(1).Type()) != "io.ReadCloser" {
		return false
	}
	extract := result.Method(0).Type().(*types.Signature)
	return types.Identical(extract.Recv().Type(), result) && !extract.Variadic() && extract.Params().Len() == 0 && extract.Results().Len() == 2 &&
		types.Identical(extract.Results().At(0).Type(), types.NewSlice(types.Typ[types.Uint8])) && isError(extract.Results().At(1).Type())
}

func validateBufferedPayloadTypes(pkg *types.Package, fn *types.Func) error {
	if fn == nil || !bufferedPayloadOperation(pkg, fn.Name()) {
		return nil
	}
	if !matchesBufferedPayloadSignature(pkg, fn) {
		return fmt.Errorf("audited buffered payload %s.GetPayload: native request/options/result graph changed", sdkPath(pkg.Path()))
	}
	return nil
}

var bufferedPayloadNativeDeclarations = map[string]string{
	"GetPayload": "62d34e7ae45503815ad00d51d16f7e9ee161232560f7d9afee956a7fbad20fc7",
	"GetPayloadOpts.ToSecretPayloadGetParams": "4cf7c149670929a94f9c59672998d0d473620ef284af175f9192535d8c9e958c",
	"payloadURL":            "1b9946ce8c64b352f92dea9b6e33dc836d42246b88c66655f03e829d2653be74",
	"PayloadResult.Extract": "1b1d1a5de3a3d747544d67ecc7382f678c2a95889e2e054312ec486f3d77af23",
}

func validateBufferedPayloadDeclarations(pkg *types.Package, decls map[string]*ast.FuncDecl) error {
	if !bufferedPayloadOperation(pkg, "GetPayload") {
		return nil
	}
	fn, ok := pkg.Scope().Lookup("GetPayload").(*types.Func)
	if !ok || !matchesBufferedPayloadSignature(pkg, fn) {
		return fmt.Errorf("audited buffered payload %s.GetPayload: native request/options/result graph changed", sdkPath(pkg.Path()))
	}
	for name, wanted := range bufferedPayloadNativeDeclarations {
		hash, err := requestDeclarationHash(decls[name])
		if err != nil || hash != wanted {
			return fmt.Errorf("audited buffered payload %s.GetPayload: pinned native declaration %s changed", sdkPath(pkg.Path()), name)
		}
	}
	return nil
}
