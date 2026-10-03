package main

import (
	"bytes"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceServiceInfoRegistryAndNativeCompatibility(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	for _, record := range sdkOwnedCollections {
		if record.Package == "gophercloudsdk/image/v2/serviceinfo" {
			g.collections = append(g.collections, record)
		}
	}
	if err := g.generate(upstreamModule + "/openstack/image/v2/imageimport"); err != nil {
		t.Fatal(err)
	}
	path := "image/v2/imageimport/api_generated.go"
	actual, err := os.ReadFile(filepath.Join(g.root, path))
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join("..", "..", "..", path))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatal("rich discovery changed native import API")
	}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	registry, err := os.ReadFile(filepath.Join(g.root, "image/v2/service_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(registry), `"gophercloudsdk/image/v2/serviceinfo"`) != 1 || strings.Count(string(registry), "ServiceInfo ") != 1 || strings.Count(string(registry), "ServiceInfo:") != 1 {
		t.Fatalf("missing or duplicate aggregate: %s", registry)
	}
	docs, err := os.ReadFile(filepath.Join(g.root, "image/v2/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"serviceinfo/README.md", "serviceinfo/api.go", "ListStores/AllStores(ctx)", "GetImportInfo(ctx)", "GetUsageInfo(ctx)", "serviceinfo/usage.go", "serviceinfo/usage.md", "기존 native `ImageImport.Get`", "WithListStoresDetails(true)"} {
		if !strings.Contains(string(docs), want) {
			t.Fatalf("missing actual ServiceInfo contract: %s", want)
		}
	}
	if strings.Contains(string(docs), "ServiceInfo.Resources") || strings.Contains(string(docs), "ServiceInfo.Get(ctx)") {
		t.Fatal("invented aggregate CRUD")
	}
	var records int
	models := map[string]string{"Store": "list_only", "ImportInfo": "service_info", "UsageInfo": "service_info"}
	for _, record := range g.collections {
		if record.Package == "gophercloudsdk/image/v2/serviceinfo" {
			records++
			if record.Source != "sdk_owned" || record.Find || record.Delete || record.Wait || record.Scope != "" || record.Kind != models[record.Model] || models[record.Model] == "" {
				t.Fatalf("invented discovery capability: %+v", record)
			}
			delete(models, record.Model)
		}
	}
	if records != 3 || len(models) != 0 {
		t.Fatalf("records=%d missing models=%v", records, models)
	}
}

func TestGlanceServiceInfoRejectsNativeGetAndCanonicalModelDrift(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	pkg, err := g.importer.Import(upstreamModule + "/openstack/image/v2/imageimport")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateGlanceServiceInfoTypes(pkg); err != nil {
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
	native := pkg.Scope().Lookup("Get").(*types.Func)
	sig := native.Type().(*types.Signature)
	for _, mode := range []string{"missing", "arity", "context", "client", "result"} {
		var object types.Object
		if mode != "missing" {
			var params []*types.Var
			for i := 0; i < sig.Params().Len(); i++ {
				v := sig.Params().At(i)
				if mode == []string{"context", "client"}[i] {
					v = types.NewVar(v.Pos(), v.Pkg(), v.Name(), types.Typ[types.Int])
				}
				if mode != "arity" || i != 1 {
					params = append(params, v)
				}
			}
			results := sig.Results()
			if mode == "result" {
				results = types.NewTuple(types.NewVar(0, pkg, "result", types.Typ[types.String]))
			}
			object = types.NewFunc(native.Pos(), pkg, "Get", types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), results, false))
		}
		if validateGlanceServiceInfoTypes(replace("Get", object)) == nil {
			t.Fatalf("accepted Get %s drift", mode)
		}
	}
	for _, name := range []string{"ImportInfo", "ImportMethods"} {
		structure := pkg.Scope().Lookup(name).Type().Underlying().(*types.Struct)
		if validateGlanceServiceInfoTypes(replace(name, nil)) == nil {
			t.Fatalf("accepted missing %s", name)
		}
		for i := 0; i < structure.NumFields(); i++ {
			for _, mode := range []string{"missing", "type", "tag"} {
				var fields []*types.Var
				var tags []string
				for j := 0; j < structure.NumFields(); j++ {
					field, tag := structure.Field(j), structure.Tag(j)
					if i == j {
						if mode == "missing" {
							continue
						}
						if mode == "type" {
							field = types.NewField(field.Pos(), pkg, field.Name(), types.Typ[types.Int], false)
						}
						if mode == "tag" {
							tag = `json:"wrong"`
						}
					}
					fields = append(fields, field)
					tags = append(tags, tag)
				}
				object := types.NewTypeName(0, pkg, name, types.NewStruct(fields, tags))
				if validateGlanceServiceInfoTypes(replace(name, object)) == nil {
					t.Fatalf("accepted %s.%s %s drift", name, structure.Field(i).Name(), mode)
				}
			}
		}
	}
	if err := validateGlanceServiceInfoTypes(types.NewPackage(upstreamModule+"/openstack/image/v2/images", "images")); err != nil {
		t.Fatal(err)
	}
}
