package main

import (
	"fmt"
	"go/types"
	"reflect"
	"strings"
)

// Store deletion belongs to the manual upper service. Preserve native deletion
// and the canonical identity used by exact name resolution without inventing
// a native store operation or a resource inventory capability.
func validateGlanceDeleteTypes(pkg *types.Package) error {
	if sdkPath(pkg.Path()) != "image/v2/images" {
		return nil
	}
	native, ok := pkg.Scope().Lookup("Delete").(*types.Func)
	if !ok {
		return fmt.Errorf("Glance deletion requires native Delete")
	}
	sig, ok := native.Type().(*types.Signature)
	if !ok || sig.Variadic() || sig.Params().Len() != 3 || sig.Results().Len() != 1 {
		return fmt.Errorf("Glance deletion requires native Delete signature")
	}
	qualified := func(p *types.Package) string { return p.Path() }
	for i, expected := range []string{"context.Context", "*" + upstreamModule + ".ServiceClient", "string"} {
		if types.TypeString(sig.Params().At(i).Type(), qualified) != expected {
			return fmt.Errorf("Glance deletion requires native Delete parameter %d (%s)", i, expected)
		}
	}
	if types.TypeString(sig.Results().At(0).Type(), qualified) != pkg.Path()+".DeleteResult" {
		return fmt.Errorf("Glance deletion requires native DeleteResult return")
	}
	result := pkg.Scope().Lookup("DeleteResult")
	if result == nil {
		return fmt.Errorf("Glance deletion requires native DeleteResult")
	}
	structure, ok := result.Type().Underlying().(*types.Struct)
	if !ok || structure.NumFields() != 1 || !structure.Field(0).Embedded() || types.TypeString(structure.Field(0).Type(), qualified) != upstreamModule+".ErrResult" {
		return fmt.Errorf("Glance deletion requires native DeleteResult embedding ErrResult")
	}
	model := pkg.Scope().Lookup("Image")
	if model == nil {
		return fmt.Errorf("Glance deletion requires native Image")
	}
	structure, ok = model.Type().Underlying().(*types.Struct)
	if !ok {
		return fmt.Errorf("Glance deletion requires struct Image")
	}
	for i := 0; i < structure.NumFields(); i++ {
		field := structure.Field(i)
		if field.Name() != "ID" {
			continue
		}
		key, _, _ := strings.Cut(reflect.StructTag(structure.Tag(i)).Get("json"), ",")
		if key != "id" || field.Type() != types.Typ[types.String] {
			return fmt.Errorf("Glance deletion requires canonical string Image.ID (id)")
		}
		return nil
	}
	return fmt.Errorf("Glance deletion requires native image identity")
}
