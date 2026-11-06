package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/quotas"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

// Shared calls that a category admits but a Magnum quota operation never
// supported fail before HTTP.
func TestContainerInfraSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	scope := newMagnumQuotaScope(t, cloud)
	api := quotas.New(cloud.Client("container-infrastructure-management", "/magnum/v1"))

	// The quota list takes Query only; limit and marker are ListOpts members.
	for _, call := range []request.ListCall{request.Limit(1), request.Marker("m"), request.MaxItems(1), request.Paginated(false), request.Header("X-Test", "1"), request.Headers(nil), request.Microversion("1.1")} {
		entry, err := request.Entry(call)
		if err != nil {
			t.Fatal(err)
		}
		for _, err := range api.List(ctx, call) {
			requireUnsupportedCall(t, err, entry.Kind.String())
		}
	}
	// Create and Update take Field only; Delete takes IgnoreMissing only.
	_, err := scope.Create(ctx, quotas.WithHardLimit(1), request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = scope.Update(ctx, quotas.WithHardLimit(1), request.Microversion("1.1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = scope.Delete(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}
}

// Shared calls apply in caller order: the later call wins.
func TestContainerInfraSharedCallsApplyInCallerOrder(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(404) })
	scope := newMagnumQuotaScope(t, cloud)
	if _, err := scope.Delete(ctx, request.IgnoreMissing(true), request.IgnoreMissing(false)); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	if value, err := scope.Delete(ctx, request.IgnoreMissing(false), request.IgnoreMissing(true)); value != nil || err != nil {
		t.Fatal(value, err)
	}
	if n := calls.Load(); n != 2 {
		t.Fatalf("HTTP requests = %d", n)
	}
}
