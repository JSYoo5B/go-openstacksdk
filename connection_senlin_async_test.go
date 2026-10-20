package openstack_test

import (
	"context"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestConnectionSenlinAsyncResourcesShareVersionTokenAndActionRoutes(t *testing.T) {
	cloud := testcloud.New(t)
	var deletes, creates, actionGets atomic.Int32
	var scales, checks, cancellations atomic.Int32
	check := func(r *http.Request) {
		if r.Header.Get("X-Auth-Token") != "fresh-token" || r.Header.Get("OpenStack-API-Version") != "clustering 1.13" {
			t.Errorf("shared source lost: %#v", r.Header)
		}
	}
	cloud.Mux.HandleFunc("DELETE /reverse/senlin/v1/clusters/cluster-id", func(w http.ResponseWriter, r *http.Request) {
		check(r)
		deletes.Add(1)
		w.Header().Set("Location", "/reverse/senlin/v1/actions/delete-action")
		w.Header().Set("X-Request-Id", "delete-request")
		w.WriteHeader(http.StatusAccepted)
	})
	cloud.Mux.HandleFunc("POST /reverse/senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		check(r)
		creates.Add(1)
		w.Header().Set("Location", "actions/create-action")
		testcloud.JSON(w, http.StatusAccepted, `{"node":{"id":"senlin-node","physical_id":"nova-server","name":"worker","index":9007199254740993,"status":"INIT"}}`)
	})
	cloud.Mux.HandleFunc("GET /reverse/senlin/v1/actions/delete-action", func(w http.ResponseWriter, r *http.Request) {
		check(r)
		actionGets.Add(1)
		testcloud.JSON(w, http.StatusOK, `{"action":{"id":"delete-action","target":"cluster-id","status":"RUNNING"}}`)
	})
	cloud.Mux.HandleFunc("POST /reverse/senlin/v1/clusters/cluster-id/actions", func(w http.ResponseWriter, r *http.Request) {
		check(r)
		scales.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"scale_out":{"count":null}}` {
			t.Error(string(body))
		}
		w.Header().Set("Location", "actions/scale-action")
		testcloud.JSON(w, 202, `{"action":"scale-action"}`)
	})
	cloud.Mux.HandleFunc("POST /reverse/senlin/v1/nodes/senlin-node/actions", func(w http.ResponseWriter, r *http.Request) {
		check(r)
		checks.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"check":{}}` {
			t.Error(string(body))
		}
		w.Header().Set("Location", "/reverse/senlin/v1/actions/check-action")
		testcloud.JSON(w, 202, `{"action":"check-action"}`)
	})
	cloud.Mux.HandleFunc("PATCH /reverse/senlin/v1/actions/delete-action", func(w http.ResponseWriter, r *http.Request) {
		check(r)
		cancellations.Add(1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"action":{"status":"CANCELLED"}}` || r.URL.Query().Get("force") != "false" {
			t.Error(string(body), r.URL)
		}
		w.WriteHeader(202)
	})
	conn, err := sdk.FromProvider(cloud.Provider,
		sdk.WithEndpoint(sdk.Clustering, cloud.Server.URL+"/reverse/senlin"),
		sdk.WithMicroversion(sdk.Clustering, "1.13"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	service, err := conn.Clustering(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if service.Clusters.RawClient() != service.RawClient() || service.Nodes.RawClient() != service.RawClient() || service.Actions.RawClient() != service.RawClient() || service.RawClient().ProviderClient != cloud.Provider {
		t.Fatal("asynchronous resource APIs do not share the selected client")
	}
	cloud.Provider.SetToken("fresh-token")
	deleted, err := service.Clusters.Delete(ctx, resource.ID("cluster-id"))
	if err != nil || deleted == nil || deleted.ActionID != "delete-action" || deleted.StatusCode != 202 || len(deleted.Body) != 0 || deleted.Header.Get("X-Request-Id") != "delete-request" {
		t.Fatalf("delete=%+v err=%v", deleted, err)
	}
	node, err := service.Nodes.Create(ctx, nodes.CreateOpts{Name: "worker", ProfileID: "profile-id"})
	if err != nil || node == nil || node.ID != "senlin-node" || node.PhysicalID != "nova-server" || node.Index == nil || node.Index.String() != "9007199254740993" || node.Operation == nil || node.Operation.ActionID != "create-action" {
		t.Fatalf("node=%+v err=%v", node, err)
	}
	if deletes.Load() != 1 || creates.Load() != 1 || actionGets.Load() != 0 {
		t.Fatal("submission implicitly fetched or resent", deletes.Load(), creates.Load(), actionGets.Load())
	}
	scaled, err := service.Clusters.ScaleOut(ctx, resource.ID("cluster-id"), clusters.ScaleOutOpts{})
	if err != nil || scaled == nil || scaled.ActionID != "scale-action" {
		t.Fatal(scaled, err)
	}
	checked, err := service.Nodes.Check(ctx, resource.ID(node.ID))
	if err != nil || checked == nil || checked.ActionID != "check-action" {
		t.Fatal(checked, err)
	}
	accepted, err := service.Actions.Cancel(ctx, resource.ID(deleted.ActionID), actions.WithUpdateForce(false))
	if err != nil || accepted == nil || accepted.StatusCode != 202 || len(accepted.Body) != 0 {
		t.Fatal(accepted, err)
	}
	if scales.Load() != 1 || checks.Load() != 1 || cancellations.Load() != 1 || actionGets.Load() != 0 {
		t.Fatal("command/cancel fetched or resent", scales.Load(), checks.Load(), cancellations.Load(), actionGets.Load())
	}
	action, err := service.Actions.Get(ctx, deleted.ActionID)
	if err != nil || action == nil || action.ID != "delete-action" || action.TargetID != "cluster-id" || action.Status != "RUNNING" || actionGets.Load() != 1 {
		t.Fatalf("explicit action fetch=%+v err=%v", action, err)
	}
}

func TestConnectionSenlinAdoptionAndClusterRecoveryShareSelectedSource(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	check := func(r *http.Request, expected string) {
		calls.Add(1)
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != expected || r.Header.Get("X-Auth-Token") != "refreshed-token" || r.Header.Get("OpenStack-API-Version") != "clustering 1.7" {
			t.Error(string(body), r.Header, err)
		}
	}
	cloud.Mux.HandleFunc("POST /reverse/senlin/v1/nodes/adopt", func(w http.ResponseWriter, r *http.Request) {
		check(r, `{"identity":"physical/id","type":"os.nova.server-1.0"}`)
		w.Header().Set("Location", "https://incidental.invalid/unrelated")
		testcloud.JSON(w, 200, `{"node":{"id":"adopted-node","physical_id":"physical/id"}}`)
	})
	cloud.Mux.HandleFunc("POST /reverse/senlin/v1/nodes/adopt-preview", func(w http.ResponseWriter, r *http.Request) {
		check(r, `{"identity":"physical/id","overrides":null,"snapshot":null,"type":"os.nova.server-1.0"}`)
		testcloud.JSON(w, 200, `{"node_preview":{"type":"os.nova.server","version":1.0,"properties":{}}}`)
	})
	cloud.Mux.HandleFunc("POST /reverse/senlin/v1/clusters/cluster-id/actions", func(w http.ResponseWriter, r *http.Request) {
		check(r, `{"recover":{"check_capacity":false}}`)
		w.Header().Set("Location", "actions/recovery-action")
		testcloud.JSON(w, 202, `{"action":"recovery-action"}`)
	})
	cloud.Mux.HandleFunc("POST /reverse/senlin/v1/clusters/cluster-id/ops", func(w http.ResponseWriter, r *http.Request) {
		check(r, `{"reboot":{"filters":{"role":"worker"},"params":{"type":"SOFT"}}}`)
		w.Header().Set("Location", "/reverse/senlin/v1/actions/reboot-action")
		testcloud.JSON(w, 202, `{"action":"reboot-action"}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("adoption/recovery implicitly fetched or followed a response", r.Method, r.URL)
		w.WriteHeader(500)
	})
	conn, err := sdk.FromProvider(cloud.Provider,
		sdk.WithEndpoint(sdk.Clustering, cloud.Server.URL+"/reverse/senlin"),
		sdk.WithMicroversion(sdk.Clustering, "1.7"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	service, err := conn.Clustering(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cloud.Provider.SetToken("refreshed-token")
	adopted, err := service.Nodes.Adopt(ctx, nodes.AdoptOpts{Identity: "physical/id", Type: "os.nova.server-1.0"})
	if err != nil || adopted == nil || adopted.ID != "adopted-node" || adopted.Operation != nil || adopted.Header.Get("Location") != "https://incidental.invalid/unrelated" {
		t.Fatal(adopted, err)
	}
	preview, err := service.Nodes.AdoptPreview(ctx, nodes.AdoptPreviewOpts{Identity: "physical/id", Type: "os.nova.server-1.0"})
	if err != nil || preview == nil || string(preview.Spec.Version) != "1.0" {
		t.Fatal(preview, err)
	}
	recovered, err := service.Clusters.Recover(ctx, resource.ID("cluster-id"), clusters.RecoverOpts{}, clusters.WithRecoverCheckCapacity(false))
	if err != nil || recovered == nil || recovered.ActionID != "recovery-action" {
		t.Fatal(recovered, err)
	}
	operated, err := service.Clusters.PerformOperation(ctx, resource.ID("cluster-id"), "reboot", clusters.PerformOperationOpts{},
		clusters.WithPerformOperationFilters(map[string]string{"role": "worker"}),
		clusters.WithPerformOperationParams(map[string]string{"type": "SOFT"}))
	if err != nil || operated == nil || operated.ActionID != "reboot-action" || calls.Load() != 4 {
		t.Fatal(operated, err, calls.Load())
	}
}
