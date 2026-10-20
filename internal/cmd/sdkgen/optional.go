package main

import (
	"go/ast"
	"go/types"
)

// Native operations sometimes assert optional builder interfaces not present
// in their parameter type. Preserve those implemented by our concrete input.
func withOptionalBuilders(pkg *types.Package, b builder, decl *ast.FuncDecl, imports map[string]*types.Package) builder {
	if b.iface == nil || decl == nil {
		return b
	}
	methods := []*types.Func{}
	seen := map[string]bool{}
	add := func(iface *types.Interface) {
		for i := 0; i < iface.NumMethods(); i++ {
			method := iface.Method(i)
			if !seen[method.Name()] {
				methods = append(methods, method)
				seen[method.Name()] = true
			}
		}
	}
	add(b.iface)
	ast.Inspect(decl.Body, func(node ast.Node) bool {
		assertion, ok := node.(*ast.TypeAssertExpr)
		if !ok {
			return true
		}
		input, ok := assertion.X.(*ast.Ident)
		if !ok || input.Name != b.name {
			return true
		}
		var object types.Object
		switch typ := assertion.Type.(type) {
		case *ast.Ident:
			object = pkg.Scope().Lookup(typ.Name)
		case *ast.SelectorExpr:
			if alias, ok := typ.X.(*ast.Ident); ok && imports[alias.Name] != nil {
				object = imports[alias.Name].Scope().Lookup(typ.Sel.Name)
			}
		}
		if object != nil {
			if iface, ok := ifaceOf(object.Type()); ok && types.Implements(b.base, iface) {
				add(iface)
			}
		}
		return true
	})
	b.iface = types.NewInterfaceType(methods, nil).Complete()
	return b
}
