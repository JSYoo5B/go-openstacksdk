package gophercloudsdk_test

import (
	"context"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "gophercloudsdk"
	"gophercloudsdk/clustering/v1/actions"
	"gophercloudsdk/clustering/v1/clusters"
	"gophercloudsdk/clustering/v1/nodes"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
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
