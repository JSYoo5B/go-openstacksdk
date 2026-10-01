package gophercloudsdk_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "gophercloudsdk"
	"gophercloudsdk/clustering/v1/nodes"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestConnectionSenlinAsyncResourcesShareVersionTokenAndActionRoutes(t *testing.T) {
	cloud := testcloud.New(t)
	var deletes, creates, actionGets atomic.Int32
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
	action, err := service.Actions.Get(ctx, deleted.ActionID)
	if err != nil || action == nil || action.ID != "delete-action" || action.TargetID != "cluster-id" || action.Status != "RUNNING" || actionGets.Load() != 1 {
		t.Fatalf("explicit action fetch=%+v err=%v", action, err)
	}
}
