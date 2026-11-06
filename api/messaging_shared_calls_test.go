package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/messaging/v2/subscriptions"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// Shared calls that a category admits but a subscription operation never
// supported fail before HTTP.
func TestMessagingSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	scope := subscriptionScope(t, subscriptionClient(cloud), "queue")

	// Lists take Header, Query, MaxItems and Paginated; limit and marker are
	// ListOpts members.
	for _, call := range []request.ListCall{request.Limit(1), request.Marker("m"), request.Headers(nil), request.Microversion("2")} {
		entry, err := request.Entry(call)
		if err != nil {
			t.Fatal(err)
		}
		_, err = scope.All(ctx, call)
		requireUnsupportedCall(t, err, entry.Kind.String())
	}
	_, err := scope.Create(ctx, subscriptions.CreateOpts{Subscriber: "http://example.invalid"}, request.Microversion("2"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = scope.Get(ctx, "s", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	err = scope.Delete(ctx, "s", request.Microversion("2"))
	requireUnsupportedCall(t, err, "Microversion")

	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}
}

// Shared calls apply in caller order: the later call wins.
func TestMessagingSharedCallsApplyInCallerOrder(t *testing.T) {
	cloud := testcloud.New(t)
	var header http.Header
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		header = r.Header.Clone()
		if r.Method == http.MethodGet {
			testcloud.JSON(w, 200, `{"subscriptions":[{"id":"a","subscriber":"http://a"},{"id":"b","subscriber":"http://b"}]}`)
			return
		}
		w.WriteHeader(404)
	})
	scope := subscriptionScope(t, subscriptionClient(cloud), "queue")
	ctx := context.Background()
	if err := scope.Delete(ctx, "s", request.Header("X-Vendor", "first"), request.IgnoreMissing(false), request.Header("X-Vendor", "second"), request.IgnoreMissing(true)); err != nil {
		t.Fatal(err)
	}
	if header.Get("X-Vendor") != "second" {
		t.Fatal(header)
	}
	if err := scope.Delete(ctx, "s", request.IgnoreMissing(true), request.IgnoreMissing(false)); err == nil {
		t.Fatal("later IgnoreMissing(false) ignored a 404")
	}
	values, err := scope.All(ctx, request.Paginated(true), request.Paginated(false), request.MaxItems(5), request.MaxItems(1))
	if err != nil || len(values) != 1 || calls.Load() != 3 {
		t.Fatal(len(values), err, calls.Load())
	}
}
