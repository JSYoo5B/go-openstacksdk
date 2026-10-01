package gophercloudsdk_test

import (
	"context"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "gophercloudsdk"
	"gophercloudsdk/clustering/v1/clusters"
	"gophercloudsdk/clustering/v1/receivers"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestConnectionSenlinReceiversAndPolicyCommandsShareSelectedSource(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	check := func(r *http.Request, expected string) {
		calls.Add(1)
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != expected || r.Header.Get("X-Auth-Token") != "refreshed-token" || r.Header.Get("OpenStack-API-Version") != "clustering 1.4" {
			t.Error(string(body), r.Header, err)
		}
	}
	cloud.Mux.HandleFunc("POST /reverse/senlin/v1/receivers", func(w http.ResponseWriter, r *http.Request) {
		check(r, `{"receiver":{"name":"messages","type":"message"}}`)
		w.Header().Set("Location", "https://incidental.invalid/unrelated")
		testcloud.JSON(w, 201, `{"receiver":{"id":"receiver-id","name":"messages","type":"message","cluster_id":null,"action":null,"channel":{"queue_name":"notifications"}}}`)
	})
	cloud.Mux.HandleFunc("GET /reverse/senlin/v1/receivers", func(w http.ResponseWriter, r *http.Request) {
		check(r, "")
		if r.URL.RawQuery != "global_project=false&user=owner-id" {
			t.Error("receiver query", r.URL)
		}
		testcloud.JSON(w, 200, `{"receivers":[{"id":"receiver-id","name":"messages","user":"owner-id"}]}`)
	})
	cloud.Mux.HandleFunc("DELETE /reverse/senlin/v1/receivers/receiver-id", func(w http.ResponseWriter, r *http.Request) {
		check(r, "")
		w.WriteHeader(204)
	})
	var commands atomic.Int32
	cloud.Mux.HandleFunc("POST /reverse/senlin/v1/clusters/cluster-id/actions", func(w http.ResponseWriter, r *http.Request) {
		ordinal := commands.Add(1)
		bodies := []string{`{"policy_attach":{"enabled":false,"policy_id":"policy-name"}}`, `{"policy_update":{"enabled":null,"policy_id":"policy-name"}}`, `{"policy_detach":{"policy_id":"policy-name"}}`}
		if ordinal < 1 || ordinal > int32(len(bodies)) {
			t.Error("command resent", ordinal)
			w.WriteHeader(500)
			return
		}
		check(r, bodies[ordinal-1])
		w.Header().Set("Location", "actions/policy-action")
		testcloud.JSON(w, 202, `{"action":"policy-action"}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("implicit lookup, response follow or resend", r.Method, r.URL)
		w.WriteHeader(500)
	})
	conn, err := sdk.FromProvider(cloud.Provider,
		sdk.WithEndpoint(sdk.Clustering, cloud.Server.URL+"/reverse/senlin"),
		sdk.WithMicroversion(sdk.Clustering, "1.4"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	service, err := conn.Clustering(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if service.Receivers.RawClient() != service.RawClient() || service.Clusters.RawClient() != service.RawClient() {
		t.Fatal("receiver and policy commands do not share selected service client")
	}
	cloud.Provider.SetToken("refreshed-token")
	receiver, err := service.Receivers.Create(ctx, receivers.CreateOpts{Name: "messages", Type: "message"})
	if err != nil || receiver == nil || receiver.ID != "receiver-id" || receiver.ClusterID != nil || receiver.Action != nil || string(receiver.Channel["queue_name"]) != `"notifications"` || receiver.Header.Get("Location") != "https://incidental.invalid/unrelated" {
		t.Fatal(receiver, err)
	}
	listed, err := service.Receivers.All(ctx, receivers.WithListUserID("owner-id"), receivers.WithListGlobalProject(false))
	if err != nil || len(listed) != 1 || listed[0].UserID != "owner-id" {
		t.Fatal(listed, err)
	}
	attached, err := service.Clusters.AttachPolicy(ctx, resource.ID("cluster-id"), clusters.AttachPolicyOpts{PolicyID: "policy-name"}, clusters.WithAttachPolicyEnabled(false))
	if err != nil || attached == nil || attached.ActionID != "policy-action" || attached.StatusCode != 202 {
		t.Fatal(attached, err)
	}
	updated, err := service.Clusters.UpdatePolicy(ctx, resource.ID("cluster-id"), clusters.UpdatePolicyOpts{PolicyID: "policy-name"}, clusters.WithUpdatePolicyEnabledNull())
	if err != nil || updated == nil || updated.ActionID != "policy-action" {
		t.Fatal(updated, err)
	}
	detached, err := service.Clusters.DetachPolicy(ctx, resource.ID("cluster-id"), clusters.DetachPolicyOpts{PolicyID: "policy-name"})
	if err != nil || detached == nil || detached.ActionID != "policy-action" {
		t.Fatal(detached, err)
	}
	if err := service.Receivers.Delete(ctx, resource.ID(receiver.ID)); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 6 || commands.Load() != 3 {
		t.Fatal("unexpected request count", calls.Load(), commands.Load())
	}
}
