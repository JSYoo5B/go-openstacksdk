package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// The Connection-owned cloud create path resolves group and data volume names
// before one POST, then applies the explicit ACTIVE-without-addresses policy.
func TestConnectionCreateLifecycleResolvesGroupVolumesAndDeletesOnlyWhenSelected(t *testing.T) {
	cloud := testcloud.New(t)
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute"), sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/volume/v3"))
	if err != nil {
		t.Fatal(err)
	}
	var posts, deletes, gets atomic.Int32
	var body map[string]json.RawMessage
	cloud.Mux.HandleFunc("GET /compute/os-server-groups", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"server_groups":[{"id":"group-id","name":"anti"}]}`)
	})
	cloud.Mux.HandleFunc("GET /volume/v3/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"volumes":[{"id":"data-id","name":"data"}]}`)
	})
	cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		body = nil
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		testcloud.JSON(w, 202, `{"server":{"id":"server","status":"BUILD","adminPass":"generated"}}`)
	})
	cloud.Mux.HandleFunc("GET /compute/servers/server", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if deletes.Load() > 0 {
			testcloud.JSON(w, 404, `{"itemNotFound":{"message":"gone"}}`)
			return
		}
		testcloud.JSON(w, 200, `{"server":{"id":"server","status":"ACTIVE","addresses":{}}}`)
	})
	cloud.Mux.HandleFunc("DELETE /compute/servers/server", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		w.WriteHeader(204)
	})
	for _, policy := range []compute.ServerWithoutAddressesPolicy{compute.KeepServerWithoutAddresses, compute.DeleteServerWithoutAddresses} {
		deletes.Store(0)
		o := connectionAutomaticCreateOptions()
		o.Server = append(o.Server, compute.WithServerGroup(resource.Name("anti")), compute.WithSchedulerHints(map[string]any{"different_host": []string{"other"}}), compute.WithDataVolumes(resource.Name("data")))
		o.ServerWithoutAddresses = policy
		result, err := conn.CreateWithAutomaticFloatingIP(context.Background(), compute.CreateServerRequest{Name: "web", Image: resource.ID("image"), Flavor: resource.ID("flavor")}, o)
		var unavailable *compute.ServerAddressesUnavailableError
		if !errors.As(err, &unavailable) || result == nil || result.Creation.AdminPass != "generated" || result.Server.Status != "ACTIVE" {
			t.Fatal(result, err)
		}
		var hints map[string]any
		var server map[string]any
		if json.Unmarshal(body["OS-SCH-HNT:scheduler_hints"], &hints) != nil || json.Unmarshal(body["server"], &server) != nil {
			t.Fatal(string(body["OS-SCH-HNT:scheduler_hints"]))
		}
		if !reflect.DeepEqual(hints, map[string]any{"group": "group-id", "different_host": []any{"other"}}) {
			t.Fatal(hints)
		}
		mappings, _ := server["block_device_mapping_v2"].([]any)
		if len(mappings) != 2 || mappings[0].(map[string]any)["destination_type"] != "local" || mappings[1].(map[string]any)["uuid"] != "data-id" {
			t.Fatal(server["block_device_mapping_v2"])
		}
		switch policy {
		case compute.KeepServerWithoutAddresses:
			if unavailable.Deletion != nil || deletes.Load() != 0 {
				t.Fatal(unavailable.Deletion, deletes.Load())
			}
		default:
			if unavailable.Deletion == nil || !unavailable.Deletion.Waited || unavailable.Deletion.Err != nil || deletes.Load() != 1 {
				t.Fatal(unavailable.Deletion, deletes.Load())
			}
		}
	}
	if posts.Load() != 2 {
		t.Fatal(posts.Load(), gets.Load())
	}
}
