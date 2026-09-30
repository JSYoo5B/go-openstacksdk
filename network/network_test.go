package network_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func TestExtraQueryAndDeleteByExactName(t *testing.T) {
	cloud := testcloud.New(t)
	service := network.New(cloud.Client("network", "/v2.0"))
	cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("name") == "" && (r.URL.Query().Get("router:external") != "true" || r.URL.Query().Get("description") != "a&b") {
			t.Errorf("query=%v", r.URL.Query())
		}
		testcloud.JSON(w, 200, `{"networks":[{"id":"id","name":"private"},{"id":"other","name":"private-copy"}]}`)
	})
	var deletes atomic.Int32
	cloud.Mux.HandleFunc("DELETE /v2.0/networks/id", func(w http.ResponseWriter, r *http.Request) { deletes.Add(1); w.WriteHeader(204) })
	ctx := context.Background()
	if all, err := service.Networks.All(ctx, resource.WithQuery("router:external", "true"), resource.WithQuery("description", "a&b")); err != nil || len(all) != 2 {
		t.Fatalf("all=%v err=%v", all, err)
	}
	if err := service.Networks.Delete(ctx, resource.Name("private")); err != nil {
		t.Fatal(err)
	}
	if deletes.Load() != 1 {
		t.Fatal("resolved wrong network")
	}
}
