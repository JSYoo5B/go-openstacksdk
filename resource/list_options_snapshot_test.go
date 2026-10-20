package resource_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestCollectionListRetainsCallerOptionsAcrossIteration(t *testing.T) {
	cloud := testcloud.New(t)
	var nativeCalls, ownedCalls atomic.Int32
	handler := func(counter *atomic.Int32, name, body string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			counter.Add(1)
			if r.URL.Query().Get("name") != name || r.URL.Query().Get("status") != "ACTIVE" || r.URL.Query().Get("vendor") != "retained" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, body)
		}
	}
	cloud.Mux.HandleFunc("GET /v2.1/project/servers/detail", handler(&nativeCalls, "^original$", `{"servers":[{"id":"matched","name":"original","status":"ACTIVE"},{"id":"changed","name":"changed","status":"ERROR"}]}`))
	cloud.Mux.HandleFunc("GET /v1/nodes", handler(&ownedCalls, "original", `{"nodes":[{"id":"matched","name":"original","status":"active"},{"id":"changed","name":"changed","status":"ERROR"}]}`))
	t.Run("native-pager", func(t *testing.T) {
		collection := compute.New(cloud.Client("compute", "/v2.1/project"), compute.Dependencies{}).Servers.Collection
		assertListOptionSnapshot(t, collection, func(value *compute.Server) string { return value.ID }, &nativeCalls)
	})
	t.Run("owned-iterator", func(t *testing.T) {
		collection := nodes.New(cloud.Client("clustering", "/v1")).Resources
		assertListOptionSnapshot(t, collection, func(value *nodes.Node) string { return value.ID }, &ownedCalls)
	})
}

func assertListOptionSnapshot[T any](t *testing.T, collection *resource.Collection[T], identity func(*T) string, calls *atomic.Int32) {
	t.Helper()
	options := []resource.ListOption{resource.WithName("original"), resource.WithStatus("ACTIVE"), resource.WithQuery("vendor", "retained")}
	stream := collection.List(context.Background(), options...)
	if calls.Load() != 0 {
		t.Fatal("List eagerly requested", calls.Load())
	}
	options[0], options[1], options[2] = resource.WithName("changed"), resource.WithStatus("ERROR"), nil
	for iteration := range 2 {
		var found int
		for value, err := range stream {
			if err != nil || value == nil || identity(value) != "matched" {
				t.Fatal(value, err)
			}
			found++
		}
		if found != 1 || calls.Load() != int32(iteration+1) {
			t.Fatal(found, calls.Load())
		}
	}
	invalid := []resource.ListOption{nil}
	errorsOnly := collection.List(context.Background(), invalid...)
	invalid[0] = resource.WithQuery("vendor", "replacement")
	var failures int
	for value, err := range errorsOnly {
		failures++
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
	}
	if failures != 1 || calls.Load() != 2 {
		t.Fatal("invalid lazy options reached HTTP", failures, calls.Load())
	}
}
