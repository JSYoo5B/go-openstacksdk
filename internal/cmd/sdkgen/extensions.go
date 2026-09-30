package main

import (
	"fmt"
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
		value := methodCapabilities(pkg, b.iface.Method(i))
		caps.body = caps.body || value.body
		caps.query = caps.query || value.query
		caps.headers = caps.headers || value.headers
	}
	return caps
}

func emitConfiguredBuilderMethod(e *emitter, b builder, method *types.Func, call string) bool {
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
	req := e.use("gophercloudsdk/request")
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
