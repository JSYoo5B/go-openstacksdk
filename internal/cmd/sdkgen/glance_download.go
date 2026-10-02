package main

import (
	"fmt"
	"go/types"
	"reflect"
	"strings"
)

// The writer workflow is manual. Validate its native stream and metadata
// dependencies without inventing a resource binding or replacing raw Download.
func (g *generator) validateGlanceDownload(pkg *types.Package) error {
	if sdkPath(pkg.Path()) != "image/v2/imagedata" {
		return nil
	}
	image, err := g.importer.Import(upstreamModule + "/openstack/image/v2/images")
	if err != nil {
		return err
	}
	model := image.Scope().Lookup("Image")
	if model == nil {
		return fmt.Errorf("Glance download requires native Image")
	}
	return validateGlanceDownloadTypes(pkg, model.Type())
}

func validateGlanceDownloadTypes(pkg *types.Package, model types.Type) error {
	if sdkPath(pkg.Path()) != "image/v2/imagedata" {
		return nil
	}
	download, ok := pkg.Scope().Lookup("Download").(*types.Func)
	if !ok {
		return fmt.Errorf("Glance download requires native Download")
	}
	sig, ok := download.Type().(*types.Signature)
	if !ok || sig.Variadic() || sig.Params().Len() != 3 || sig.Results().Len() != 1 {
		return fmt.Errorf("Glance download requires native Download signature")
	}
	qualified := func(p *types.Package) string { return p.Path() }
	for i, expected := range []string{"context.Context", "*" + upstreamModule + ".ServiceClient", "string"} {
		if types.TypeString(sig.Params().At(i).Type(), qualified) != expected {
			return fmt.Errorf("Glance download requires native Download parameter %d (%s)", i, expected)
		}
	}
	if types.TypeString(sig.Results().At(0).Type(), qualified) != pkg.Path()+".DownloadResult" {
		return fmt.Errorf("Glance download requires native DownloadResult")
	}
	if model == nil {
		return fmt.Errorf("Glance download requires its native Image model")
	}
	structure, ok := model.Underlying().(*types.Struct)
	if !ok {
		return fmt.Errorf("Glance download requires struct Image")
	}
	keys := map[string]string{"ID": "id", "Checksum": "checksum"}
	for i := 0; i < structure.NumFields(); i++ {
		field := structure.Field(i)
		if key, needed := keys[field.Name()]; needed {
			actual, _, _ := strings.Cut(reflect.StructTag(structure.Tag(i)).Get("json"), ",")
			if actual != key || field.Type() != types.Typ[types.String] {
				return fmt.Errorf("Glance download requires canonical string Image.%s (%s)", field.Name(), key)
			}
			delete(keys, field.Name())
		}
	}
	if len(keys) != 0 {
		return fmt.Errorf("Glance download requires native image identity and checksum")
	}
	return nil
}
