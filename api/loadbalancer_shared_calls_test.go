package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/availabilityzoneprofiles"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/availabilityzones"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

// Shared calls that a category admits but an Octavia SDK-owned operation never
// supported fail before HTTP.
func TestLoadBalancerSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	zones := availabilityzones.New(cloud.Client("load-balancer", "/v2"))
	profiles := availabilityzoneprofiles.New(cloud.Client("load-balancer", "/v2"))

	// Lists take Query only; limit and marker are ListOpts members.
	for _, call := range []request.ListCall{request.Limit(1), request.Marker("m"), request.MaxItems(1), request.Paginated(false), request.Header("X-Test", "1"), request.Headers(nil), request.Microversion("2.1")} {
		entry, err := request.Entry(call)
		if err != nil {
			t.Fatal(err)
		}
		for _, err := range zones.List(ctx, call) {
			requireUnsupportedCall(t, err, entry.Kind.String())
		}
		for _, err := range profiles.List(ctx, call) {
			requireUnsupportedCall(t, err, entry.Kind.String())
		}
	}
	// Creates and updates take Field only.
	_, err := zones.Create(ctx, availabilityzones.CreateOpts{Name: "z", AvailabilityZoneProfileID: "p"}, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = profiles.Update(ctx, "p", availabilityzoneprofiles.UpdateOpts{}, request.Microversion("2.1"))
	requireUnsupportedCall(t, err, "Microversion")
	// Quota Reset takes IgnoreMissing only.
	_, err = newOctaviaQuotaScope(t, cloud).Reset(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}
}

// Shared calls apply in caller order: the later call wins.
func TestLoadBalancerSharedCallsApplyInCallerOrder(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var query string
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(404)
			return
		}
		query = r.URL.RawQuery
		testcloud.JSON(w, 200, `{"availability_zones":[]}`)
	})
	for _, err := range availabilityzones.New(cloud.Client("load-balancer", "/v2")).List(ctx, request.Query("vendor", "first"), request.Query("vendor", "second")) {
		t.Fatal(err)
	}
	if query != "vendor=second" {
		t.Fatal(query)
	}
	scope := newOctaviaQuotaScope(t, cloud)
	if _, err := scope.Reset(ctx, request.IgnoreMissing(true), request.IgnoreMissing(false)); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	if value, err := scope.Reset(ctx, request.IgnoreMissing(false), request.IgnoreMissing(true)); value != nil || err != nil {
		t.Fatal(value, err)
	}
}
