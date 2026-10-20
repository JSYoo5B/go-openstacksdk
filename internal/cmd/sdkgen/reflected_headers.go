package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
	"strconv"
	"strings"
)

type reflectedHeaderField struct {
	field *types.Var
	tag   string
}

// These native requests reflect their interface argument itself, rather than
// delegating header serialization to the concrete builder's methods.
var reflectedHeaderRequests = map[string]string{
	"network/v2/networks":                      "3b1feeb4106a14f46ed06da981759dda2c6a240a705700127037462d891e3a19",
	"network/v2/ports":                         "cb13b6a51f8bde0be8e8cfae79335153a47d35cf4aa95dafd01681cceffed772",
	"network/v2/subnets":                       "508194c51f7c554f4b2a504c72c8a68089ae3b7bc9f351d0163759491c52a838",
	"network/v2/extensions/trunks":             "2cc12b3910e78e97a94e0ed52bf6fe338f47e034a2fc067a0597590da539f17a",
	"network/v2/extensions/subnetpools":        "65a408d89d92173c5c5c2d24a02f7225e2890fa8c2a37216cf473a8bcfcb8fd2",
	"network/v2/extensions/layer3/routers":     "eadd44f561b905b13a6703b18cd95dd2fc02e96734e085ac2335775bd6750ca7",
	"network/v2/extensions/layer3/floatingips": "a32121fe0e3e708b029b01fa2e585f18c88b3d5b2b4059521de2e6d9d6f5eee0",
	"network/v2/extensions/qos/policies":       "b50c34db64794d08e8626169f24c24f0bb4f49ef219f0bceac0b58a165381652",
	"network/v2/extensions/security/groups":    "538533fb24c1274f806683fbd49369453b8243ca7e1ba5c8c2fe0fd838e85c74",
}

func validateReflectedHeaderRequests(pkg *types.Package, decls map[string]*ast.FuncDecl, files map[*ast.FuncDecl]*ast.File) error {
	wanted, known := reflectedHeaderRequests[sdkPath(pkg.Path())]
	if !known {
		return nil
	}
	mismatch := func(reason string) error {
		return fmt.Errorf("reflected native headers %s.Update: %s; review the native contract before updating the carrier", sdkPath(pkg.Path()), reason)
	}
	decl := decls["Update"]
	hash, err := requestDeclarationHash(decl)
	if err != nil || hash != wanted {
		return mismatch("pinned native declaration changed or is missing")
	}
	fn, ok := pkg.Scope().Lookup("Update").(*types.Func)
	if !ok {
		return mismatch("native function is missing")
	}
	sig := fn.Type().(*types.Signature)
	if sig.Params().Len() != 4 || sig.Variadic() {
		return mismatch("native parameters changed")
	}
	input := sig.Params().At(3)
	iface, ok := ifaceOf(input.Type())
	if !ok {
		return mismatch("native input is not a builder interface")
	}
	base, err := concrete(pkg, input.Type())
	if err != nil {
		return mismatch(err.Error())
	}
	if !directNativeHeaders(pkg, decl, files[decl], input.Name()) {
		return mismatch("owning-file root BuildHeaders call changed")
	}
	b, err := withReflectedHeaders(pkg, builder{name: input.Name(), base: base, iface: iface}, decl, files[decl])
	if err != nil {
		return mismatch(err.Error())
	}
	if len(b.headers) != 1 || b.headers[0].field.Name() != "RevisionNumber" || b.headers[0].tag != `json:"-" h:"If-Match"` {
		return mismatch("native top-level header field or full tag changed")
	}
	pointer, ok := b.headers[0].field.Type().(*types.Pointer)
	if !ok || !types.Identical(pointer.Elem(), types.Typ[types.Int]) {
		return mismatch("native revision field is no longer *int")
	}
	return nil
}

// These native requests reflect their builder argument with BuildQueryString
// instead of calling its query method, so the adapter must carry the q fields.
var reflectedQueryRequests = map[string]bool{
	"identity/v3/endpoints.List": true,
}

func directNativeHeaders(pkg *types.Package, decl *ast.FuncDecl, file *ast.File, parameter string) bool {
	return directNativeReflection(pkg, decl, file, parameter, "BuildHeaders")
}

// Resolve imports in the declaration's file. A package-wide alias map cannot
// distinguish different imports using the same spelling in separate files.
func directNativeReflection(pkg *types.Package, decl *ast.FuncDecl, file *ast.File, parameter, reflector string) bool {
	if decl == nil || decl.Body == nil || file == nil {
		return false
	}
	alias, dot := "", false
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || path != upstreamModule {
			continue
		}
		for _, imported := range pkg.Imports() {
			if imported.Path() == path {
				alias = imported.Name()
			}
		}
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		dot = alias == "."
	}
	if alias == "" || alias == "_" {
		return false
	}
	var object *ast.Object
	for _, field := range decl.Type.Params.List {
		for _, name := range field.Names {
			if name.Name == parameter {
				object = name.Obj
			}
		}
	}
	found := false
	ast.Inspect(decl.Body, func(node ast.Node) bool {
		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}
		argument, ok := call.Args[0].(*ast.Ident)
		if !ok || argument.Name != parameter || (object != nil && argument.Obj != object) {
			return true
		}
		switch function := call.Fun.(type) {
		case *ast.SelectorExpr:
			qualifier, ok := function.X.(*ast.Ident)
			if ok && !dot && qualifier.Name == alias && qualifier.Obj == nil && function.Sel.Name == reflector {
				found = true
			}
		case *ast.Ident:
			if dot && function.Name == reflector && function.Obj == nil && pkg.Scope().Lookup(reflector) == nil {
				found = true
			}
		}
		return true
	})
	return found
}

func withReflectedHeaders(pkg *types.Package, b builder, decl *ast.FuncDecl, file *ast.File) (builder, error) {
	if b.iface == nil {
		return b, nil
	}
	headers := directNativeHeaders(pkg, decl, file, b.name)
	query := directNativeReflection(pkg, decl, file, b.name, "BuildQueryString")
	if !headers && !query {
		return b, nil
	}
	if query {
		name := sdkPath(pkg.Path()) + "." + decl.Name.Name
		if !reflectedQueryRequests[name] {
			return b, fmt.Errorf("direct native query reflection in %s needs review", name)
		}
		b.reflectedQuery = true
	}
	base := types.Unalias(b.base)
	if pointer, ok := base.(*types.Pointer); ok {
		base = types.Unalias(pointer.Elem())
	}
	fields, ok := base.Underlying().(*types.Struct)
	if !ok {
		return b, fmt.Errorf("direct reflected input %s is not a concrete struct", b.name)
	}
	for i := 0; i < fields.NumFields(); i++ {
		tag := reflect.StructTag(fields.Tag(i))
		if !(headers && tag.Get("h") != "") && !(query && tag.Get("q") != "") {
			continue
		}
		field := fields.Field(i)
		if !field.Exported() {
			return b, fmt.Errorf("direct reflected field %s is not exported", field.Name())
		}
		for j := 0; j < b.iface.NumMethods(); j++ {
			if b.iface.Method(j).Name() == field.Name() {
				return b, fmt.Errorf("direct reflected field %s conflicts with a builder method", field.Name())
			}
		}
		b.headers = append(b.headers, reflectedHeaderField{field: field, tag: fields.Tag(i)})
	}
	return b, nil
}

func builderLiteral(b builder, base, config string) string {
	fields := []string{"base:" + base, "config:" + config}
	for _, header := range b.headers {
		name := header.field.Name()
		fields = append(fields, name+":"+base+"."+name)
	}
	return b.adapter + "{" + strings.Join(fields, ",") + "}"
}
