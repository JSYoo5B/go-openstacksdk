package main

import (
	"bytes"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceDeletePreservesNativeAPIAndDocumentsStoreWorkflow(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	if err := g.generate(upstreamModule + "/openstack/image/v2/images"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"image/v2/images/api_generated.go", "image/v2/images/resources_generated.go"} {
		actual, err := os.ReadFile(filepath.Join(g.root, path))
		if err != nil {
			t.Fatal(err)
		}
		expected, err := os.ReadFile(filepath.Join("..", "..", "..", path))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(actual, expected) || strings.Contains(string(actual), "DeleteImage") {
			t.Fatalf("manual deletion changed native image declarations: %s", path)
		}
	}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	docs, err := os.ReadFile(filepath.Join(g.root, "image/v2/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"image.Service.DeleteImage", "../delete.md", "WithDeleteImageStore", "실제204"} {
		if !strings.Contains(string(docs), text) {
			t.Fatalf("missing upper image deletion documentation: %s", text)
		}
	}
}

func TestGlanceDeleteRejectsNativeSignatureResultAndIdentityDrift(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	pkg, err := g.importer.Import(upstreamModule + "/openstack/image/v2/images")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateGlanceDeleteTypes(pkg); err != nil {
		t.Fatal(err)
	}
	replace := func(name string, object types.Object) *types.Package {
		copy := types.NewPackage(pkg.Path(), pkg.Name())
		for _, other := range pkg.Scope().Names() {
			if other != name {
				copy.Scope().Insert(pkg.Scope().Lookup(other))
			}
		}
		if object != nil {
			copy.Scope().Insert(object)
		}
		return copy
	}
	function := pkg.Scope().Lookup("Delete").(*types.Func)
	sig := function.Type().(*types.Signature)
	for _, mode := range []string{"missing", "arity", "context", "client", "id", "result"} {
		var functionCopy types.Object
		if mode != "missing" {
			var params []*types.Var
			for i := 0; i < sig.Params().Len(); i++ {
				v := sig.Params().At(i)
				if mode == []string{"context", "client", "id"}[i] {
					v = types.NewVar(v.Pos(), v.Pkg(), v.Name(), types.Typ[types.Int])
				}
				if mode != "arity" || i != 2 {
					params = append(params, v)
				}
			}
			results := sig.Results()
			if mode == "result" {
				results = types.NewTuple(types.NewVar(0, pkg, "result", types.Typ[types.String]))
			}
			functionCopy = types.NewFunc(function.Pos(), pkg, "Delete", types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), results, false))
		}
		if err := validateGlanceDeleteTypes(replace("Delete", functionCopy)); err == nil {
			t.Fatalf("native Delete %s drift accepted", mode)
		}
	}
	for _, mode := range []string{"missing", "nonstruct", "empty", "embedding"} {
		var object types.Object
		if mode != "missing" {
			var changed types.Type = types.Typ[types.String]
			if mode == "empty" {
				changed = types.NewStruct(nil, nil)
			} else if mode == "embedding" {
				changed = types.NewStruct([]*types.Var{types.NewField(0, pkg, "ErrResult", types.Typ[types.String], true)}, nil)
			}
			object = types.NewTypeName(0, pkg, "DeleteResult", changed)
		}
		if err := validateGlanceDeleteTypes(replace("DeleteResult", object)); err == nil {
			t.Fatalf("native DeleteResult %s drift accepted", mode)
		}
	}
	model := pkg.Scope().Lookup("Image").Type().Underlying().(*types.Struct)
	for _, mode := range []string{"missing-model", "nonstruct", "missing-id", "type", "tag"} {
		var object types.Object
		if mode != "missing-model" {
			var changed types.Type = types.Typ[types.String]
			if mode != "nonstruct" {
				var fields []*types.Var
				var tags []string
				for i := 0; i < model.NumFields(); i++ {
					field, tag := model.Field(i), model.Tag(i)
					if field.Name() == "ID" {
						if mode == "missing-id" {
							continue
						}
						if mode == "type" {
							field = types.NewVar(field.Pos(), field.Pkg(), field.Name(), types.Typ[types.Int])
						} else if mode == "tag" {
							tag = `json:"other"`
						}
					}
					fields, tags = append(fields, field), append(tags, tag)
				}
				changed = types.NewStruct(fields, tags)
			}
			object = types.NewTypeName(0, pkg, "Image", changed)
		}
		if err := validateGlanceDeleteTypes(replace("Image", object)); err == nil {
			t.Fatalf("native Image %s drift accepted", mode)
		}
	}
	for _, path := range []string{"image/v1/images", "image/v2/imagedata", "blockstorage/v3/volumes"} {
		if err := validateGlanceDeleteTypes(types.NewPackage(upstreamModule+"/openstack/"+path, "other")); err != nil {
			t.Fatalf("unrelated native package acquired image deletion policy: %s %v", path, err)
		}
	}
}
