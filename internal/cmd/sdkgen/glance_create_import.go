package main

import (
	"fmt"
	"go/types"
	"reflect"
	"strings"
)

// The manual workflow uses the native creation builder and Image model. Check
// their wire fields without turning the workflow into a native declaration.
func validateGlanceCreateImportTypes(pkg *types.Package) error {
	if sdkPath(pkg.Path()) != "image/v2/images" {
		return nil
	}
	qualified := func(p *types.Package) string { return p.Path() }
	for name, fields := range map[string]map[string][2]string{
		"Image": {
			"ID": {"id", "string"}, "Status": {"status", pkg.Path() + ".ImageStatus"},
			"ContainerFormat": {"container_format", "string"}, "DiskFormat": {"disk_format", "string"},
		},
		"CreateOpts": {
			"Name": {"name", "string"}, "DiskFormat": {"disk_format", "string"},
			"ContainerFormat": {"container_format", "string"}, "Visibility": {"visibility", "*" + pkg.Path() + ".ImageVisibility"},
			"Hidden": {"os_hidden", "*bool"}, "Protected": {"protected", "*bool"},
			"MinDisk": {"min_disk", "int"}, "MinRAM": {"min_ram", "int"}, "Tags": {"tags", "[]string"},
		},
	} {
		object := pkg.Scope().Lookup(name)
		if object == nil {
			return fmt.Errorf("Glance creation workflow requires native %s", name)
		}
		structure, ok := object.Type().Underlying().(*types.Struct)
		if !ok {
			return fmt.Errorf("Glance creation workflow requires struct %s", name)
		}
		for i := 0; i < structure.NumFields(); i++ {
			field := structure.Field(i)
			expected, needed := fields[field.Name()]
			if !needed {
				continue
			}
			key, _, _ := strings.Cut(reflect.StructTag(structure.Tag(i)).Get("json"), ",")
			if key != expected[0] || types.TypeString(field.Type(), qualified) != expected[1] {
				return fmt.Errorf("Glance creation workflow requires %s.%s with canonical JSON %s and type %s", name, field.Name(), expected[0], expected[1])
			}
			delete(fields, field.Name())
		}
		if len(fields) != 0 {
			return fmt.Errorf("Glance creation workflow requires native %s fields", name)
		}
	}
	return nil
}
