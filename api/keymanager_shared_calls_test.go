package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/quotas"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Shared calls that a category admits but a key manager operation never
// supported fail before HTTP.
func TestKeyManagerSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	client := secretFetchClient(cloud)

	_, err := secrets.New(client).Fetch(ctx, resource.ID("s1"), request.Microversion("1.1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = secrets.New(client).FindRecord(ctx, "s1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = containers.New(client).Fetch(ctx, resource.ID("c1"), request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = containers.New(client).CreateRecord(ctx, request.Microversion("1.1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = containers.New(client).UpdateRecord(ctx, "c1", request.Field("name", "x"))
	requireUnsupportedCall(t, err, "Field")
	for _, err := range containers.New(client).ListRecords(ctx, request.Limit(1)) {
		requireUnsupportedCall(t, err, "Limit")
	}
	for _, err := range containers.New(client).ListRecords(ctx, request.Header("X-Test", "1")) {
		requireUnsupportedCall(t, err, "Header")
	}
	for _, err := range secretstores.New(client).List(ctx, request.Marker("m")) {
		requireUnsupportedCall(t, err, "Marker")
	}
	_, err = quotas.New(client).Get(ctx, request.Microversion("1.1"))
	requireUnsupportedCall(t, err, "Microversion")
	scope, err := quotas.New(client).InProject(ctx, resource.ID("p1"))
	if err != nil {
		t.Fatal(err)
	}
	requireUnsupportedCall(t, scope.Delete(ctx, request.Headers(nil)), "Headers")
	_, err = scope.GetRecord(ctx, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	if n := calls.Load(); n != 0 {
		t.Fatalf("calls = %d", n)
	}
}

// request.Paginated keeps caller order: a later Paginated(false) stops after
// the first page even after an earlier Paginated(true).
func TestKeyManagerSecretStoresPaginatedCallerOrder(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	body := secretStorePage(secretStoreRow(t, "https://foreign.invalid/first", nil), "https://foreign.invalid/not-read")
	cloud.Mux.HandleFunc("GET "+secretStoresPath, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, body)
	})
	values, err := secretstores.New(secretStoresClient(cloud)).All(context.Background(), request.Paginated(true), request.Paginated(false))
	if err != nil || len(values) != 1 || calls.Load() != 1 {
		t.Fatal(values, err, calls.Load())
	}
}
