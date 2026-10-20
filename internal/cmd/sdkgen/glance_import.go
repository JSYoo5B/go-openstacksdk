package main

import (
	"fmt"
	"go/constant"
	"go/types"
	"reflect"
	"strings"
)

// Import submission is SDK-owned. Validate the native types it uses without
// inferring a new collection or changing native Create/Get declarations.
func (g *generator) validateGlanceImport(pkg *types.Package) error {
	if sdkPath(pkg.Path()) != "image/v2/imageimport" {
		return nil
	}
	image, err := g.importer.Import(upstreamModule + "/openstack/image/v2/images")
	if err != nil {
		return err
	}
	model := image.Scope().Lookup("Image")
	if model == nil {
		return fmt.Errorf("Glance import requires native Image")
	}
	return validateGlanceImportTypes(pkg, model.Type())
}

func validateGlanceImportTypes(pkg *types.Package, model types.Type) error {
	if sdkPath(pkg.Path()) != "image/v2/imageimport" {
		return nil
	}
	for _, name := range []string{"Create", "Get"} {
		if _, ok := pkg.Scope().Lookup(name).(*types.Func); !ok {
			return fmt.Errorf("Glance import requires native %s", name)
		}
	}
	for name, expected := range map[string]string{"GlanceDirectMethod": "glance-direct", "WebDownloadMethod": "web-download"} {
		value, ok := pkg.Scope().Lookup(name).(*types.Const)
		if !ok || value.Val().Kind() != constant.String || constant.StringVal(value.Val()) != expected {
			return fmt.Errorf("Glance import requires native %s=%q", name, expected)
		}
	}
	if model == nil {
		return fmt.Errorf("Glance import requires its native Image model")
	}
	structure, ok := model.Underlying().(*types.Struct)
	if !ok {
		return fmt.Errorf("Glance import requires a struct Image model")
	}
	keys := map[string]string{"ID": "id", "ContainerFormat": "container_format", "DiskFormat": "disk_format"}
	for i := 0; i < structure.NumFields(); i++ {
		field := structure.Field(i)
		if key, needed := keys[field.Name()]; needed {
			actual, _, _ := strings.Cut(reflect.StructTag(structure.Tag(i)).Get("json"), ",")
			basic, isBasic := field.Type().Underlying().(*types.Basic)
			if actual != key || !isBasic || basic.Info()&types.IsString == 0 {
				return fmt.Errorf("Glance import requires canonical string Image.%s (%s)", field.Name(), key)
			}
			delete(keys, field.Name())
		}
	}
	if len(keys) != 0 {
		return fmt.Errorf("Glance import requires Image identity and both formats")
	}
	return nil
}
