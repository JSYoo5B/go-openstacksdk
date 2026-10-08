package main

import (
	"bytes"
	"go/constant"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceImportKeepsActualNativeCallsAndNoInventedCollection(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	if err := g.generate(upstreamModule + "/openstack/image/v2/imageimport"); err != nil {
		t.Fatal(err)
	}
	// The inventory already records an unsupported placeholder, rather
	// than a usable Collection. Preserve that boundary for submission.
	if len(g.collections) != 1 {
		t.Fatalf("invented import inventory rows: %+v", g.collections)
	}
	r := g.collections[0]
	if r.Package != "github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport" || r.Model != "" || r.Find || r.Delete || r.Wait || r.Scope != "" || r.Kind != "" || r.Issue != "requires a scoped or specialized resource binding" {
		t.Fatalf("import submission became a collection: %+v", r)
	}
	actual, err := os.ReadFile(filepath.Join(g.root, "image/v2/imageimport/api_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join("..", "..", "..", "image/v2/imageimport/api_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatal("SDK import changed the native Create/Get declarations")
	}
	if strings.Contains(string(actual), "ImportImage") {
		t.Fatal("manual SDK submission was counted as a native operation")
	}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	docs, err := os.ReadFile(filepath.Join(g.root, "image/v2/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"ImageImport.ImportImage", "ImportKnownImage", "imageimport/README.md", "실제 202 응답"} {
		if !strings.Contains(string(docs), text) {
			t.Fatalf("missing import contract documentation: %s", text)
		}
	}
}

func TestGlanceImportRejectsNativeIdentityFormatsAndConstantsDrift(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	pkg, err := g.importer.Import(upstreamModule + "/openstack/image/v2/imageimport")
	if err != nil {
		t.Fatal(err)
	}
	image, err := g.importer.Import(upstreamModule + "/openstack/image/v2/images")
	if err != nil {
		t.Fatal(err)
	}
	model := image.Scope().Lookup("Image").Type()
	if err := validateGlanceImportTypes(pkg, model); err != nil {
		t.Fatal(err)
	}
	for _, missing := range []string{"Create", "Get", "GlanceDirectMethod", "WebDownloadMethod"} {
		copy := types.NewPackage(pkg.Path(), pkg.Name())
		for _, name := range pkg.Scope().Names() {
			if name != missing {
				copy.Scope().Insert(pkg.Scope().Lookup(name))
			}
		}
		if err := validateGlanceImportTypes(copy, model); err == nil {
			t.Fatalf("missing native %s accepted", missing)
		}
	}
	for _, name := range []string{"GlanceDirectMethod", "WebDownloadMethod"} {
		copy := types.NewPackage(pkg.Path(), pkg.Name())
		for _, existing := range pkg.Scope().Names() {
			if existing == name {
				copy.Scope().Insert(types.NewConst(0, copy, name, types.Typ[types.String], constant.MakeString("changed")))
			} else {
				copy.Scope().Insert(pkg.Scope().Lookup(existing))
			}
		}
		if err := validateGlanceImportTypes(copy, model); err == nil {
			t.Fatalf("native method constant drift accepted: %s", name)
		}
	}
	structure := model.Underlying().(*types.Struct)
	for _, name := range []string{"ID", "ContainerFormat", "DiskFormat"} {
		for _, mode := range []string{"missing", "type", "tag"} {
			var fields []*types.Var
			var tags []string
			for i := 0; i < structure.NumFields(); i++ {
				f, tag := structure.Field(i), structure.Tag(i)
				if f.Name() == name {
					switch mode {
					case "missing":
						continue
					case "type":
						f = types.NewVar(f.Pos(), f.Pkg(), f.Name(), types.Typ[types.Int])
					case "tag":
						tag = `json:"changed"`
					}
				}
				fields, tags = append(fields, f), append(tags, tag)
			}
			if err := validateGlanceImportTypes(pkg, types.NewStruct(fields, tags)); err == nil {
				t.Fatalf("Image.%s %s drift accepted", name, mode)
			}
		}
	}
	for _, invalid := range []types.Type{nil, types.Typ[types.String]} {
		if err := validateGlanceImportTypes(pkg, invalid); err == nil {
			t.Fatal("invalid Image model accepted")
		}
	}
	for _, path := range []string{"image/v2/tasks", "image/v1/imageimport", "blockstorage/v3/volumes"} {
		other := types.NewPackage(upstreamModule+"/openstack/"+path, "other")
		if err := validateGlanceImportTypes(other, nil); err != nil {
			t.Fatalf("unrelated package acquired import policy: %s %v", path, err)
		}
	}
}
