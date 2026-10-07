package main

import (
	"bytes"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceStageKeepsActualNativeStreamingAndNoInventedCollection(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	if err := g.generate(upstreamModule + "/openstack/image/v2/imagedata"); err != nil {
		t.Fatal(err)
	}
	if len(g.collections) != 1 {
		t.Fatalf("invented staging inventory rows: %+v", g.collections)
	}
	r := g.collections[0]
	if r.Package != "github.com/JSYoo5B/gophercloudsdk/image/v2/imagedata" || r.Model != "" || r.Find || r.Delete || r.Wait || r.Scope != "" || r.Kind != "" || r.Issue != "requires a scoped or specialized resource binding" {
		t.Fatalf("staging workflow became a collection: %+v", r)
	}
	actual, err := os.ReadFile(filepath.Join(g.root, "image/v2/imagedata/api_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join("..", "..", "..", "image/v2/imagedata/api_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatal("SDK staging changed native Stage/Upload/Download declarations")
	}
	if strings.Contains(string(actual), "StageImage") {
		t.Fatal("manual SDK staging was counted as a native operation")
	}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	docs, err := os.ReadFile(filepath.Join(g.root, "image/v2/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"ImageData.StageImage", "StageKnownImage", "imagedata/README.md", "PUT204", "GET200", "HTTP Content-Length"} {
		if !strings.Contains(string(docs), text) {
			t.Fatalf("missing staging contract documentation: %s", text)
		}
	}
}

func TestGlanceStageRejectsNativeSignatureIdentityAndStatusDrift(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	pkg, err := g.importer.Import(upstreamModule + "/openstack/image/v2/imagedata")
	if err != nil {
		t.Fatal(err)
	}
	image, err := g.importer.Import(upstreamModule + "/openstack/image/v2/images")
	if err != nil {
		t.Fatal(err)
	}
	model := image.Scope().Lookup("Image").Type()
	if err := validateGlanceStageTypes(pkg, model); err != nil {
		t.Fatal(err)
	}
	stage := pkg.Scope().Lookup("Stage").(*types.Func)
	sig := stage.Type().(*types.Signature)
	for _, mode := range []string{"missing", "arity", "context", "client", "id", "reader", "result"} {
		copy := types.NewPackage(pkg.Path(), pkg.Name())
		for _, name := range pkg.Scope().Names() {
			if name != "Stage" {
				copy.Scope().Insert(pkg.Scope().Lookup(name))
			}
		}
		if mode != "missing" {
			var params []*types.Var
			for i := 0; i < sig.Params().Len(); i++ {
				v := sig.Params().At(i)
				changed := mode == []string{"context", "client", "id", "reader"}[i]
				if changed {
					v = types.NewVar(v.Pos(), v.Pkg(), v.Name(), types.Typ[types.Int])
				}
				if mode != "arity" || i != 3 {
					params = append(params, v)
				}
			}
			results := sig.Results()
			if mode == "result" {
				results = types.NewTuple(types.NewVar(0, copy, "r", types.Typ[types.String]))
			}
			copy.Scope().Insert(types.NewFunc(stage.Pos(), copy, "Stage", types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), results, false)))
		}
		if err := validateGlanceStageTypes(copy, model); err == nil {
			t.Fatalf("native Stage %s drift accepted", mode)
		}
	}
	structure := model.Underlying().(*types.Struct)
	for _, name := range []string{"ID", "Status"} {
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
			if err := validateGlanceStageTypes(pkg, types.NewStruct(fields, tags)); err == nil {
				t.Fatalf("Image.%s %s drift accepted", name, mode)
			}
		}
	}
	for _, invalid := range []types.Type{nil, types.Typ[types.String]} {
		if err := validateGlanceStageTypes(pkg, invalid); err == nil {
			t.Fatal("invalid Image model accepted")
		}
	}
	for _, path := range []string{"image/v2/imageimport", "image/v1/imagedata", "blockstorage/v3/volumes"} {
		other := types.NewPackage(upstreamModule+"/openstack/"+path, "other")
		if err := validateGlanceStageTypes(other, nil); err != nil {
			t.Fatalf("unrelated package acquired staging policy: %s %v", path, err)
		}
	}
}
