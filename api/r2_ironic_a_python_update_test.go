package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/chassis"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/portgroups"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/volumeconnectors"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/volumetargets"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

// Python update_*(**attrs) diffs the attributes against an empty remote state
// and sends one add per top-level attribute, sorted by path.
func TestR2IronicAFieldPatchesMatchPythonUpdateAttributes(t *testing.T) {
	cloud := testcloud.New(t)
	k := r2ironicAKinds()[0]
	rec := r2ironicAServe(t, cloud, k, func(w http.ResponseWriter, req r2ironicAReq) { testcloud.JSON(w, 200, k.item("c-1", "")) })
	fields := map[string]any{"extra": map[string]any{"k": false}, "description": "", "zeta": nil}
	if _, err := chassis.New(ironicClient(cloud, "baremetal", "1.8")).Update(context.Background(), "c-1", chassis.UpdateOpts{Operations: chassis.FieldPatches(fields)}); err != nil {
		t.Fatal(err)
	}
	r2ironicAJSONEqual(t, rec.all()[0].Body, `[{"op":"add","path":"/description","value":""},{"op":"add","path":"/extra","value":{"k":false}},{"op":"add","path":"/zeta","value":null}]`)
	for name, got := range map[string][]any{
		"portgroups":       {len(portgroups.FieldPatches(fields))},
		"volumeconnectors": {len(volumeconnectors.FieldPatches(fields))},
		"volumetargets":    {len(volumetargets.FieldPatches(fields))},
	} {
		if got[0] != 3 {
			t.Fatalf("%s patches=%v", name, got)
		}
	}
	// The same JSON Patch endpoint serves Python patch_*: WithUpdateOptions replaces,
	// WithUpdateOperations appends.
	rec.mu.Lock()
	rec.requests = nil
	rec.mu.Unlock()
	api := chassis.New(ironicClient(cloud, "baremetal", ""))
	_, err := api.Update(context.Background(), "c-1", chassis.UpdateOpts{Operations: []chassis.PatchOperation{chassis.Add("/a", 1)}},
		chassis.WithUpdateOperations(chassis.Remove("/b")), chassis.WithUpdateOptions(chassis.UpdateOpts{Operations: []chassis.PatchOperation{chassis.Replace("/c", false)}}),
		chassis.WithUpdateOperations(chassis.Remove("/d")))
	if err != nil {
		t.Fatal(err)
	}
	r2ironicAJSONEqual(t, rec.all()[0].Body, `[{"op":"replace","path":"/c","value":false},{"op":"remove","path":"/d"}]`)
}
