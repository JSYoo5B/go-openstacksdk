package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/allocationcandidates"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// Shared calls that a category admits but a placement SDK-owned operation
// never supported fail before HTTP.
func TestPlacementSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	client := cloud.Client("placement", "/placement")

	// CreateRecord and UpdateRecord take Header only.
	_, err := allocations.New(client).CreateRecord(ctx, nil, request.Field("vendor", true))
	requireUnsupportedCall(t, err, "Field")
	_, err = resourceclasses.New(client).UpdateRecord(ctx, "CUSTOM_A", request.Microversion("1.2"))
	requireUnsupportedCall(t, err, "Microversion")
	// CreateInventory takes Field and Header only.
	_, err = resourceproviders.New(client).CreateInventory(ctx, "rp", "VCPU", resourceproviders.CreateInventoryOpts{Total: 1}, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")

	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}
}

// A []string request.Query repeats the key, and the later call wins.
func TestPlacementSharedCallsApplyInCallerOrder(t *testing.T) {
	cloud := testcloud.New(t)
	var query string
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		testcloud.JSON(w, 200, `{"allocation_requests":[],"provider_summaries":{}}`)
	})
	api := allocationcandidates.New(cloud.Client("placement", "/placement"))
	for _, err := range api.List(context.Background(), request.Query("member_of1", "agg-1"), request.Query("member_of1", []string{"agg-4", "agg-5"})) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if query != "member_of1=agg-4&member_of1=agg-5" {
		t.Fatal(query)
	}
}
