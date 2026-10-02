package main

import (
	"encoding/json"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCinderMetadataScopesUseActualCollectionsAndPreserveNativeAPIs(t *testing.T) {
	g, _ := snapshotMetadataActualNative(t)
	g.root = t.TempDir()
	for _, path := range []string{"blockstorage/v2/volumes", "blockstorage/v2/snapshots", "blockstorage/v3/volumes", "blockstorage/v3/snapshots"} {
		if err := g.generate(upstreamModule + "/openstack/" + path); err != nil {
			t.Fatalf("actual pinned %s: %v", path, err)
		}
		body, err := os.ReadFile(filepath.Join(g.root, path, "api_generated.go"))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(body), "MetadataIn") {
			t.Fatal("manual metadata scope was emitted as a native operation")
		}
		if strings.HasSuffix(path, "/snapshots") && !strings.Contains(string(body), "snapshotmetadata.Extract(result.Result)") {
			t.Fatal("existing native Snapshot metadata result policy was lost")
		}
		if strings.HasSuffix(path, "/volumes") && !strings.Contains(string(body), "func (a *API) SetImageMetadata(") {
			t.Fatal("volume image metadata API was replaced by the child metadata scope")
		}
	}
	if len(g.collections) != 4 {
		t.Fatalf("invented metadata collections: %d", len(g.collections))
	}
	for _, record := range g.collections {
		if record.MetadataScope != "MetadataIn" || record.Scope != "" || record.Kind != "" || !record.Find || !record.Delete || !record.Wait || record.Source != "" {
			t.Fatalf("native collection capabilities changed: %+v", record)
		}
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
	if err := json.Unmarshal(data, &records); err != nil || len(records) != 4 {
		t.Fatalf("metadata inventory: %v", err)
	}
	for _, record := range records {
		if record["metadata_scope"] != "MetadataIn" || record["scope"] != nil {
			t.Fatalf("metadata scope replaced the parent collection: %v", record)
		}
	}
	for _, version := range []string{"v2", "v3"} {
		data, err := os.ReadFile(filepath.Join(g.root, "blockstorage", version, "README.md"))
		if err != nil {
			t.Fatal(err)
		}
		docs := string(data)
		for _, fragment := range []string{"Volumes.MetadataIn(ctx, ref)", "Snapshots.MetadataIn(ctx, ref)", "POST 병합", "PUT 전체 교체", "../metadata/README.md", "volumes/README.md", "snapshots/README.md", "SDK 소유 요청·실제 metadata 응답"} {
			if !strings.Contains(docs, fragment) {
				t.Fatalf("%s omits %q", version, fragment)
			}
		}
		if version == "v2" && strings.Contains(docs, "conn.BlockStorageProjectQuotas") {
			t.Fatal("v3 project quota convenience was advertised for v2")
		}
	}
}

func TestCinderMetadataScopeRejectsMissingParentBindingAndOtherResources(t *testing.T) {
	fn := types.NewFunc(0, nil, "Get", types.NewSignatureType(nil, nil, nil, nil, nil, false))
	valid := collectionPlan{modelName: "Volume", id: "ID", name: "Name", getter: fn, lister: fn}
	pkg := types.NewPackage(upstreamModule+"/openstack/blockstorage/v3/volumes", "volumes")
	for _, modify := range []func(*collectionPlan){
		func(p *collectionPlan) { p.modelName = "Backup" },
		func(p *collectionPlan) { p.getter = nil },
		func(p *collectionPlan) { p.lister = nil },
		func(p *collectionPlan) { p.id = "OtherID" },
		func(p *collectionPlan) { p.name = "" },
	} {
		changed := valid
		modify(&changed)
		if _, err := cinderMetadataCollectionScope(pkg, &changed); err == nil {
			t.Fatal("metadata scope retained a changed parent binding")
		}
	}
	if _, err := cinderMetadataCollectionScope(pkg, nil); err == nil {
		t.Fatal("metadata scope accepted a missing collection")
	}
	for _, path := range []string{"blockstorage/v3/backups", "blockstorage/v1/volumes", "blockstorage/v4/snapshots", "compute/v2/servers", "sharedfilesystems/v2/shares"} {
		pkg := types.NewPackage(upstreamModule+"/openstack/"+path, "other")
		if scope, err := cinderMetadataCollectionScope(pkg, &valid); err != nil || scope != "" {
			t.Fatalf("invented metadata capability for %s: %q %v", path, scope, err)
		}
	}
}
