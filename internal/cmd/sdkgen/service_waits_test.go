package main

import (
	"encoding/json"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceWaitPoliciesBindOnlyActualStatusCollections(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	policies := map[string]string{
		"compute/v2/servers": "compute", "image/v2/images": "image",
		"blockstorage/v2/volumes": "cinder", "blockstorage/v3/volumes": "cinder",
		"blockstorage/v2/snapshots": "cinder", "blockstorage/v3/snapshots": "cinder",
	}
	for path, policy := range policies {
		if err := g.generate(upstreamModule + "/openstack/" + path); err != nil {
			t.Fatalf("actual native %s: %v", path, err)
		}
		record := g.collections[len(g.collections)-1]
		if record.ServiceWait != policy || !record.Find || !record.Wait || !record.Delete {
			t.Fatalf("lost actual parent capabilities: %+v", record)
		}
		body, err := os.ReadFile(filepath.Join(g.root, path, "resources_generated.go"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(body), "WaitForState") || !strings.Contains(string(body), "return a.Resources.Wait(ctx, ref, status, options...)") || !strings.Contains(string(body), "return a.Resources.WaitDeleted(ctx, ref, options...)") {
			t.Fatalf("manual service policy replaced shared generated wait: %s", path)
		}
	}
	if len(g.collections) != 6 {
		t.Fatalf("unexpected invented collection: %d", len(g.collections))
	}
	if err := os.MkdirAll(filepath.Join(g.root, "api"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := g.writeCollectionInventory(); err != nil {
		t.Fatal(err)
	}
	if err := g.generateServices(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(g.root, "api/resource_inventory.json"))
	if err != nil {
		t.Fatal(err)
	}
	var records []map[string]any
	if err := json.Unmarshal(data, &records); err != nil || len(records) != 6 {
		t.Fatalf("service wait inventory: %v", err)
	}
	for _, record := range records {
		path := strings.TrimPrefix(record["package"].(string), "gophercloudsdk/")
		if record["service_wait"] != policies[path] {
			t.Fatalf("missing exact owned policy: %v", record)
		}
	}
	for _, service := range []string{"compute/v2", "blockstorage/v2", "blockstorage/v3", "image/v2"} {
		body, err := os.ReadFile(filepath.Join(g.root, service, "README.md"))
		if err != nil {
			t.Fatal(err)
		}
		for _, text := range []string{"WaitForState/WaitForDelete", "120초", "../../docs/service-waits.md", "기존 공통 `WaitFor/WaitForDeletion`의 5분 기본"} {
			if !strings.Contains(string(body), text) {
				t.Fatalf("%s missing %q", service, text)
			}
		}
	}
}

func TestServiceWaitPolicyRejectsParentDriftAndDoesNotInferOtherModels(t *testing.T) {
	fn := types.NewFunc(0, nil, "Get", types.NewSignatureType(nil, nil, nil, nil, nil, false))
	valid := collectionPlan{modelName: "Volume", id: "ID", name: "Name", status: "Status", getter: fn, lister: fn, deleter: fn}
	pkg := types.NewPackage(upstreamModule+"/openstack/blockstorage/v3/volumes", "volumes")
	for _, modify := range []func(*collectionPlan){
		func(p *collectionPlan) { p.modelName = "Backup" }, func(p *collectionPlan) { p.id = "OtherID" },
		func(p *collectionPlan) { p.name = "" }, func(p *collectionPlan) { p.status = "State" },
		func(p *collectionPlan) { p.getter = nil }, func(p *collectionPlan) { p.lister = nil }, func(p *collectionPlan) { p.deleter = nil },
	} {
		plan := valid
		modify(&plan)
		if policy, err := serviceWaitCollectionPolicy(pkg, &plan); err == nil || policy != "" {
			t.Fatalf("parent drift accepted: %+v policy=%s error=%v", plan, policy, err)
		}
	}
	if _, err := serviceWaitCollectionPolicy(pkg, nil); err == nil {
		t.Fatal("missing audited native collection accepted")
	}
	for _, path := range []string{"blockstorage/v3/backups", "compute/v2/flavors", "image/v2/tasks", "image/v1/images", "network/v2/ports"} {
		pkg := types.NewPackage(upstreamModule+"/openstack/"+path, "unrelated")
		if policy, err := serviceWaitCollectionPolicy(pkg, &valid); err != nil || policy != "" {
			t.Fatalf("invented service-default wait for %s: %s %v", path, policy, err)
		}
	}
}
