package main

import (
	"fmt"
	"go/types"
	"reflect"
	"strings"
)

// Staging owns streaming and the follow-up metadata observation. Validate only
// its actual native inputs; do not infer a Collection or rewrite native Stage.
func (g *generator) validateGlanceStage(pkg *types.Package) error {
	if sdkPath(pkg.Path()) != "image/v2/imagedata" {
		return nil
	}
	image, err := g.importer.Import(upstreamModule + "/openstack/image/v2/images")
	if err != nil {
		return err
	}
	model := image.Scope().Lookup("Image")
	if model == nil {
		return fmt.Errorf("Glance staging requires native Image")
	}
	return validateGlanceStageTypes(pkg, model.Type())
}

func validateGlanceStageTypes(pkg *types.Package, model types.Type) error {
	if sdkPath(pkg.Path()) != "image/v2/imagedata" {
		return nil
	}
	stage, ok := pkg.Scope().Lookup("Stage").(*types.Func)
	if !ok {
		return fmt.Errorf("Glance staging requires native Stage")
	}
	sig, ok := stage.Type().(*types.Signature)
	if !ok || sig.Variadic() || sig.Params().Len() != 4 || sig.Results().Len() != 1 {
		return fmt.Errorf("Glance staging requires the native Stage signature")
	}
	qualified := func(p *types.Package) string { return p.Path() }
	inputs := []string{"context.Context", "*" + upstreamModule + ".ServiceClient", "string", "io.Reader"}
	for i, expected := range inputs {
		if types.TypeString(sig.Params().At(i).Type(), qualified) != expected {
			return fmt.Errorf("Glance staging requires native Stage parameter %d (%s)", i, expected)
		}
	}
	if types.TypeString(sig.Results().At(0).Type(), qualified) != pkg.Path()+".StageResult" {
		return fmt.Errorf("Glance staging requires native StageResult")
	}
	if model == nil {
		return fmt.Errorf("Glance staging requires its native Image model")
	}
	structure, ok := model.Underlying().(*types.Struct)
	if !ok {
		return fmt.Errorf("Glance staging requires a struct Image model")
	}
	keys := map[string]string{"ID": "id", "Status": "status"}
	for i := 0; i < structure.NumFields(); i++ {
		field := structure.Field(i)
		if key, needed := keys[field.Name()]; needed {
			actual, _, _ := strings.Cut(reflect.StructTag(structure.Tag(i)).Get("json"), ",")
			basic, isBasic := field.Type().Underlying().(*types.Basic)
			if actual != key || !isBasic || basic.Info()&types.IsString == 0 {
				return fmt.Errorf("Glance staging requires canonical string Image.%s (%s)", field.Name(), key)
			}
			delete(keys, field.Name())
		}
	}
	if len(keys) != 0 {
		return fmt.Errorf("Glance staging requires Image identity and status")
	}
	return nil
}
