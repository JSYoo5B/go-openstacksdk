package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/hosts"
	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/notifications"
	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/segments"
	"github.com/JSYoo5B/go-openstacksdk/instanceha/v1/vmoves"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func instanceHASharedCallsClient(t *testing.T, reply func(*http.Request) (int, string)) (*gophercloud.ServiceClient, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("instance-ha", "/instance-ha/v1")
	client.Microversion = "1.3"
	return client, &calls
}

// Shared calls that a category admits but a Masakari operation never
// supported fail before HTTP.
func TestInstanceHASharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	client, calls := instanceHASharedCallsClient(t, func(*http.Request) (int, string) { return 500, `{}` })

	// Lists take Query only; limit and marker are ListOpts members.
	for _, call := range []request.ListCall{request.Limit(1), request.Marker("m"), request.MaxItems(1), request.Paginated(false), request.Header("X-Test", "1"), request.Headers(nil), request.Microversion("1.1")} {
		entry, err := request.Entry(call)
		if err != nil {
			t.Fatal(err)
		}
		_, err = segments.New(client).All(ctx, call)
		requireUnsupportedCall(t, err, entry.Kind.String())
		_, err = notifications.New(client).AllMarkerFallback(ctx, call)
		requireUnsupportedCall(t, err, entry.Kind.String())
	}
	hostScope, err := hosts.New(client).InSegment(ctx, resource.ID(segmentUUID))
	if err != nil {
		t.Fatal(err)
	}
	_, err = hostScope.All(ctx, request.Limit(1))
	requireUnsupportedCall(t, err, "Limit")
	moveScope, err := vmoves.New(client).InNotification(ctx, resource.ID(segmentUUID))
	if err != nil {
		t.Fatal(err)
	}
	_, err = moveScope.All(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	// Creates, updates and attribute writes take Field and Header only.
	_, err = segments.New(client).Create(ctx, segments.CreateOpts{Name: "s", RecoveryMethod: "auto", ServiceType: "COMPUTE"}, request.Microversion("1.1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = hostScope.Create(ctx, hosts.CreateOpts{Name: "h", Type: "COMPUTE", ControlAttributes: "SSH"}, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = notifications.New(client).Create(ctx, notifications.CreateOpts{Type: "VM", Hostname: "h", GeneratedTime: "t"}, request.Microversion("1.1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = segments.New(client).CreateAttributes(ctx, segments.Attributes{Name: request.Present("s")}, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")

	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}
}

// Shared calls apply in caller order: the later call wins.
func TestInstanceHASharedCallsApplyInCallerOrder(t *testing.T) {
	var header http.Header
	var query string
	client, _ := instanceHASharedCallsClient(t, func(r *http.Request) (int, string) {
		header, query = r.Header.Clone(), r.URL.RawQuery
		if r.Method == "GET" {
			return 200, `{"segments":[]}`
		}
		return 202, `{"segment":{"uuid":"` + segmentUUID + `","name":"s"}}`
	})
	ctx := context.Background()
	if _, err := segments.New(client).All(ctx, request.Query("vendor", "first"), request.Query("vendor", "second")); err != nil {
		t.Fatal(err)
	}
	if query != "vendor=second" {
		t.Fatal(query)
	}
	if _, err := segments.New(client).Create(ctx, segments.CreateOpts{Name: "s", RecoveryMethod: "auto", ServiceType: "COMPUTE"},
		request.Header("X-Vendor", "first"), request.Header("X-Vendor", "second")); err != nil {
		t.Fatal(err)
	}
	if header.Get("X-Vendor") != "second" {
		t.Fatal(header)
	}
}
