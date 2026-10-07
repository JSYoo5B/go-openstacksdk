package gophercloudsdk_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestConnectionSenlinWaitersShareLiveTokenVersionAndFixedRoutes(t *testing.T) {
	cloud := testcloud.New(t)
	var clusterGets, receiverGets atomic.Int32
	cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters/cluster-id", func(w http.ResponseWriter, r *http.Request) {
		ordinal := clusterGets.Add(1)
		token := "start-token"
		if ordinal > 1 {
			token = "renewed-token"
		}
		if r.Header.Get("X-Auth-Token") != token || r.Header.Get("OpenStack-API-Version") != "clustering 1.13" {
			t.Error(r.Header)
		}
		if ordinal == 1 {
			cloud.Provider.SetToken("renewed-token")
			testcloud.JSON(w, 200, `{"cluster":{"id":"incidental-id","status":"INIT"}}`)
			return
		}
		testcloud.JSON(w, 200, `{"cluster":{"id":"incidental-id","status":"ACTIVE"}}`)
	})
	cloud.Mux.HandleFunc("GET /reverse/senlin/v1/receivers/receiver-id", func(w http.ResponseWriter, r *http.Request) {
		ordinal := receiverGets.Add(1)
		if r.Header.Get("X-Auth-Token") != "renewed-token" || r.Header.Get("OpenStack-API-Version") != "clustering 1.13" {
			t.Error(r.Header)
		}
		if ordinal == 1 {
			testcloud.JSON(w, 200, `{"receiver":{"id":"incidental-receiver","name":"messages","type":"message","channel":{"url":"https://incidental.invalid/trigger"}}}`)
			return
		}
		testcloud.JSON(w, 404, `{"error":"receiver gone"}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("wait changed identity, followed a channel or submitted a mutation", r.Method, r.URL)
		w.WriteHeader(500)
	})
	cloud.Provider.SetToken("start-token")
	conn, err := sdk.FromProvider(cloud.Provider,
		sdk.WithEndpoint(sdk.Clustering, cloud.Server.URL+"/reverse/senlin"),
		sdk.WithMicroversion(sdk.Clustering, "1.13"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	service, err := conn.Clustering(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := service.Clusters.WaitForStatus(ctx, resource.ID("cluster-id"), "ACTIVE", resource.WithPollInterval(time.Millisecond))
	if err != nil || cluster == nil || cluster.ID != "incidental-id" || cluster.Status != "ACTIVE" || cluster.StatusCode != 200 {
		t.Fatal(cluster, err)
	}
	if err := service.Receivers.WaitForDelete(ctx, resource.ID("receiver-id"), resource.WithPollInterval(time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if clusterGets.Load() != 2 || receiverGets.Load() != 2 {
		t.Fatal("unexpected polling", clusterGets.Load(), receiverGets.Load())
	}
}
