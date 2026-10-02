package main

import (
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlanceTaskWaitBindsActualIDOnlyCollectionWithoutReplacingNativeCalls(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	if err := g.generate(upstreamModule + "/openstack/image/v2/tasks"); err != nil {
		t.Fatal(err)
	}
	if len(g.collections) != 1 {
		t.Fatalf("invented Task collections: %d", len(g.collections))
	}
	r := g.collections[0]
	if r.TaskWait != "WaitForTask" || r.ServiceWait != "" || r.Model != "Task" || r.Find || r.Delete || !r.Wait {
		t.Fatalf("lost native Task collection boundary: %+v", r)
	}
	api, err := os.ReadFile(filepath.Join(g.root, "image/v2/tasks/api_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, signature := range []string{"func (a *API) Get(", "func (a *API) Create(", "func (a *API) List("} {
		if !strings.Contains(string(api), signature) {
			t.Fatalf("lost native Task API: %s", signature)
		}
	}
	if strings.Contains(string(api), "WaitForTask") {
		t.Fatal("SDK Task workflow was emitted as a native operation")
	}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	docs, err := os.ReadFile(filepath.Join(g.root, "image/v2/README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Tasks.WaitForTask", "WaitForTask/WaitForTaskState", "tasks/README.md", "새 ID도 같은 context 시간 제한"} {
		if !strings.Contains(string(docs), text) {
			t.Fatalf("missing Task wait reference: %s", text)
		}
	}
}

func TestGlanceTaskWaitRejectsParentAndPayloadModelDrift(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	pkg, err := g.importer.Import(upstreamModule + "/openstack/image/v2/tasks")
	if err != nil {
		t.Fatal(err)
	}
	plan := &collectionPlan{model: pkg.Scope().Lookup("Task").Type(), modelName: "Task", id: "ID", status: "Status", getter: pkg.Scope().Lookup("Get").(*types.Func), lister: pkg.Scope().Lookup("List").(*types.Func)}
	for _, mutate := range []func(*collectionPlan){
		func(p *collectionPlan) { p.model = nil }, func(p *collectionPlan) { p.modelName = "Image" },
		func(p *collectionPlan) { p.id = "OtherID" }, func(p *collectionPlan) { p.status = "State" },
		func(p *collectionPlan) { p.name = "Name" }, func(p *collectionPlan) { p.getter = nil }, func(p *collectionPlan) { p.lister = nil },
	} {
		copy := *plan
		mutate(&copy)
		if binding, err := glanceTaskWaitCollectionBinding(pkg, &copy); err == nil || binding != "" {
			t.Fatalf("invalid Task parent accepted: %+v %s %v", copy, binding, err)
		}
	}
	withoutCreate := types.NewPackage(pkg.Path(), "tasks")
	if _, err := glanceTaskWaitCollectionBinding(withoutCreate, plan); err == nil {
		t.Fatal("missing native Task Create accepted")
	}
	model := plan.model.Underlying().(*types.Struct)
	for _, changed := range []string{"ID", "Status", "Type", "Message", "Input", "Result"} {
		var fields []*types.Var
		var tags []string
		for i := 0; i < model.NumFields(); i++ {
			f := model.Field(i)
			typ := f.Type()
			if f.Name() == changed {
				typ = types.Typ[types.Int]
				if changed == "Input" || changed == "Result" {
					typ = types.NewMap(types.Typ[types.String], types.Typ[types.Int])
				}
			}
			fields = append(fields, types.NewVar(f.Pos(), f.Pkg(), f.Name(), typ))
			tags = append(tags, model.Tag(i))
		}
		copy := *plan
		copy.model = types.NewStruct(fields, tags)
		if binding, err := glanceTaskWaitCollectionBinding(pkg, &copy); err == nil || binding != "" {
			t.Fatalf("Task payload type drift accepted for %s: %s %v", changed, binding, err)
		}
	}
	fields := make([]*types.Var, model.NumFields())
	tags := make([]string, model.NumFields())
	for i := 0; i < model.NumFields(); i++ {
		fields[i], tags[i] = model.Field(i), model.Tag(i)
		if fields[i].Name() == "Input" {
			tags[i] = `json:"changed_input"`
		}
	}
	copy := *plan
	copy.model = types.NewStruct(fields, tags)
	if binding, err := glanceTaskWaitCollectionBinding(pkg, &copy); err == nil || binding != "" {
		t.Fatalf("Task mutation input JSON tag drift accepted: %s %v", binding, err)
	}
	for _, path := range []string{"image/v2/images", "image/v1/tasks", "workflow/v2/tasks", "compute/v2/servers"} {
		pkg := types.NewPackage(upstreamModule+"/openstack/"+path, "unrelated")
		if binding, err := glanceTaskWaitCollectionBinding(pkg, plan); err != nil || binding != "" {
			t.Fatalf("invented Task workflow for %s: %s %v", path, binding, err)
		}
	}
}
