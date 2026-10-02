package main

import (
	"bytes"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceCreateImportKeepsNativeImageAPIAndDocumentsUpperService(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	if err := g.generate(upstreamModule + "/openstack/image/v2/images"); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(filepath.Join(g.root, "image/v2/images/api_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join("..", "..", "..", "image/v2/images/api_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected) || strings.Contains(string(actual), "CreateAndImport") {
		t.Fatal("manual workflow changed native image declarations")
	}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	docs, err := os.ReadFile(filepath.Join(g.root, "image/v2/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"image.Service.CreateAndImport", "../create-import.md", "생성201", "staging204", "조회200", "접수202", "conn.Image(ctx)"} {
		if !strings.Contains(string(docs), text) {
			t.Fatalf("missing upper image workflow documentation: %s", text)
		}
	}
}

func TestGlanceCreateImportRejectsNativeMetadataAndResponseFieldDrift(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	pkg, err := g.importer.Import(upstreamModule + "/openstack/image/v2/images")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateGlanceCreateImportTypes(pkg); err != nil {
		t.Fatal(err)
	}
	for model, names := range map[string][]string{
		"Image":      {"ID", "Status", "ContainerFormat", "DiskFormat"},
		"CreateOpts": {"Name", "DiskFormat", "ContainerFormat", "Visibility", "Hidden", "Protected", "MinDisk", "MinRAM", "Tags"},
	} {
		structure := pkg.Scope().Lookup(model).Type().Underlying().(*types.Struct)
		for _, name := range names {
			for _, mode := range []string{"missing", "type", "tag"} {
				t.Run(model+"/"+name+"/"+mode, func(t *testing.T) {
					var fields []*types.Var
					var tags []string
					for i := 0; i < structure.NumFields(); i++ {
						field, tag := structure.Field(i), structure.Tag(i)
						if field.Name() == name {
							switch mode {
							case "missing":
								continue
							case "type":
								field = types.NewVar(field.Pos(), field.Pkg(), field.Name(), types.Typ[types.Float64])
							case "tag":
								tag = `json:"different"`
							}
						}
						fields, tags = append(fields, field), append(tags, tag)
					}
					copy := types.NewPackage(pkg.Path(), pkg.Name())
					for _, other := range pkg.Scope().Names() {
						if other == model {
							copy.Scope().Insert(types.NewTypeName(0, copy, model, types.NewStruct(fields, tags)))
						} else {
							copy.Scope().Insert(pkg.Scope().Lookup(other))
						}
					}
					if err := validateGlanceCreateImportTypes(copy); err == nil {
						t.Fatal("native wire contract drift accepted")
					}
				})
			}
		}
	}
	for _, path := range []string{"image/v1/images", "image/v2/imageimport", "compute/v2/servers"} {
		if err := validateGlanceCreateImportTypes(types.NewPackage(upstreamModule+"/openstack/"+path, "other")); err != nil {
			t.Fatalf("unrelated native package acquired workflow policy: %s %v", path, err)
		}
	}
}
