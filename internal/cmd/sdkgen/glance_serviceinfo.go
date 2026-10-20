package main

import (
	"fmt"
	"go/types"
	"reflect"
	"strings"
)

// Rich discovery is SDK-owned. Keep the existing native import-info API and
// its canonical model intact; store endpoints do not become native functions.
func validateGlanceServiceInfoTypes(pkg *types.Package) error {
	if sdkPath(pkg.Path()) != "image/v2/imageimport" {
		return nil
	}
	qualified := func(p *types.Package) string { return p.Path() }
	function, ok := pkg.Scope().Lookup("Get").(*types.Func)
	if !ok {
		return fmt.Errorf("Glance discovery requires native Get")
	}
	sig, ok := function.Type().(*types.Signature)
	if !ok || sig.Variadic() || sig.Params().Len() != 2 || sig.Results().Len() != 1 {
		return fmt.Errorf("Glance discovery requires native Get signature")
	}
	for i, want := range []string{"context.Context", "*" + upstreamModule + ".ServiceClient"} {
		if types.TypeString(sig.Params().At(i).Type(), qualified) != want {
			return fmt.Errorf("Glance discovery requires Get parameter %d (%s)", i, want)
		}
	}
	if types.TypeString(sig.Results().At(0).Type(), qualified) != pkg.Path()+".GetResult" {
		return fmt.Errorf("Glance discovery requires native GetResult return")
	}
	for name, fields := range map[string]map[string][2]string{
		"ImportInfo":    {"ImportMethods": {pkg.Path() + ".ImportMethods", "import-methods"}},
		"ImportMethods": {"Description": {"string", "description"}, "Type": {"string", "type"}, "Value": {"[]string", "value"}},
	} {
		object := pkg.Scope().Lookup(name)
		if object == nil {
			return fmt.Errorf("Glance discovery requires native %s", name)
		}
		structure, ok := object.Type().Underlying().(*types.Struct)
		if !ok {
			return fmt.Errorf("Glance discovery requires struct %s", name)
		}
		for field, want := range fields {
			found := false
			for i := 0; i < structure.NumFields(); i++ {
				value := structure.Field(i)
				if value.Name() != field {
					continue
				}
				key, _, _ := strings.Cut(reflect.StructTag(structure.Tag(i)).Get("json"), ",")
				if types.TypeString(value.Type(), qualified) != want[0] || key != want[1] {
					return fmt.Errorf("Glance discovery requires canonical %s.%s", name, field)
				}
				found = true
			}
			if !found {
				return fmt.Errorf("Glance discovery requires native %s.%s", name, field)
			}
		}
	}
	return nil
}
