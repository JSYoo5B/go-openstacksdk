package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"
)

type extensionCapabilities struct{ body, query, headers bool }

func isStringMap(t types.Type) bool {
	value, ok := t.Underlying().(*types.Map)
	return ok && isString(value.Key()) && isString(value.Elem())
}

// A map of strings may be HTTP headers or an extra-spec JSON body, and a
// string may be a query or a URL key. Use the upstream serialization contract.
func methodCapabilities(pkg *types.Package, method *types.Func) extensionCapabilities {
	name := method.Name()
	sig := method.Type().(*types.Signature)
	caps := extensionCapabilities{}
	for i := 0; i < sig.Results().Len()-1; i++ {
		t := sig.Results().At(i).Type()
		if isAnyMap(t) {
			caps.body = true
		}
		if isStringMap(t) && (strings.Contains(pkg.Path(), "/objectstorage/") || strings.Contains(name, "Headers") || name == "ToSecretPayloadGetParams" || name == "ToSecretUpdateRequest") {
			caps.headers = true
		}
	}
	if sig.Results().Len() >= 2 && isError(sig.Results().At(sig.Results().Len()-1).Type()) {
		last := sig.Results().At(sig.Results().Len() - 2).Type()
		caps.query = isString(last) && (sig.Results().Len() == 2 || strings.HasSuffix(name, "Query") || strings.HasSuffix(name, "Params") || name == "ToClaimCreateRequest")
	}
	return caps
}

func capabilities(pkg *types.Package, b builder) extensionCapabilities {
	caps := extensionCapabilities{}
	if b.iface == nil {
		return caps
	}
	for i := 0; i < b.iface.NumMethods(); i++ {
		if b.unread[b.iface.Method(i).Name()] {
			continue
		}
		value := methodCapabilities(pkg, b.iface.Method(i))
		caps.body = caps.body || value.body
		caps.query = caps.query || value.query
		caps.headers = caps.headers || value.headers
	}
	if b.reflectedQuery {
		caps.query = false
	}
	return caps
}

func emitConfiguredBuilderMethod(e *emitter, b builder, method *types.Func, call string) bool {
	// The native token builders nest the scope map inside the create map, which
	// already receives extension fields. Merging them into the scope as well
	// duplicated each field there and invented a scope when none was requested.
	if method.Name() == "ToTokenV3ScopeMap" {
		return false
	}
	sig := method.Type().(*types.Signature)
	caps := methodCapabilities(e.pkg, method)
	if sig.Results().Len() < 2 || !isError(sig.Results().At(sig.Results().Len()-1).Type()) || !(caps.body || caps.query || caps.headers) {
		return false
	}
	values := []string{}
	for i := 0; i < sig.Results().Len()-1; i++ {
		values = append(values, fmt.Sprintf("value%d", i))
	}
	e.printf("%s,err:=%s\n", strings.Join(values, ","), call)
	errReturn := func() {
		e.printf("if err!=nil{\n")
		zeros := []string{}
		for i := 0; i < sig.Results().Len()-1; i++ {
			name := fmt.Sprintf("zero%d", i)
			e.printf("var %s %s\n", name, e.typ(sig.Results().At(i).Type()))
			zeros = append(zeros, name)
		}
		e.printf("return %s,err}\n", strings.Join(zeros, ","))
	}
	errReturn()
	if backupUpdateBody(e.pkg, b, method) {
		maps := e.use("maps")
		e.printf("if b.base.Metadata!=nil{value0[\"metadata\"]=%s.Clone(b.base.Metadata)}\n", maps)
		e.printf("value0=map[string]any{\"backup\":value0}\n")
	}
	req := e.use("github.com/JSYoo5B/go-openstacksdk/request")
	for i := 0; i < sig.Results().Len()-1; i++ {
		t := sig.Results().At(i).Type()
		if caps.body && isAnyMap(t) {
			e.printf("%s,err=%s.MergeFieldsFor(%s,b.config.Fields,b.base)\n", values[i], req, values[i])
			errReturn()
		}
		if caps.headers && isStringMap(t) {
			e.printf("%s,err=%s.MergeHeadersFor(%s,b.config.Headers,b.base)\n", values[i], req, values[i])
			errReturn()
		}
		if caps.query && i == sig.Results().Len()-2 {
			e.printf("%s,err=%s.ExtendQuery(%s,b.config.Query)\n", values[i], req, values[i])
			errReturn()
		}
	}
	e.printf("return %s,nil\n", strings.Join(values, ","))
	return true
}

// unreadBuilderMethods names builder methods the native function never calls.
// Extensions merged by such a method would be dropped silently, so they must
// not be offered. Any other use of the parameter, such as passing it to a
// helper or asserting another interface, may read every method.
func unreadBuilderMethods(decl *ast.FuncDecl, parameter string, iface *types.Interface) map[string]bool {
	if decl == nil || decl.Body == nil || iface == nil {
		return nil
	}
	called, escaped := map[string]bool{}, false
	selected := map[*ast.Ident]bool{}
	ast.Inspect(decl.Body, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
				if id, ok := selector.X.(*ast.Ident); ok && id.Name == parameter {
					called[selector.Sel.Name] = true
					selected[id] = true
				}
			}
		}
		return true
	})
	ast.Inspect(decl.Body, func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok && id.Name == parameter && !selected[id] {
			escaped = true
		}
		return !escaped
	})
	if escaped {
		return nil
	}
	unread := map[string]bool{}
	for i := 0; i < iface.NumMethods(); i++ {
		if name := iface.Method(i).Name(); !called[name] {
			unread[name] = true
		}
	}
	return unread
}
