package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/softwareconfigs"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/softwaredeployments"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func orchestrationSharedCallsClient(t *testing.T, reply func(*http.Request) (int, string)) (*gophercloud.ServiceClient, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	return cloud.Client("orchestration", "/heat/v1"), &calls
}

// Shared calls that a category admits but a Heat operation never supported
// fail before HTTP.
func TestOrchestrationSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	client, calls := orchestrationSharedCallsClient(t, func(*http.Request) (int, string) { return 500, `{}` })

	// Software lists take Limit, Marker and Query only.
	for _, call := range []request.ListCall{request.MaxItems(1), request.Paginated(false), request.Header("X-Test", "1"), request.Headers(nil), request.Microversion("1.1")} {
		entry, err := request.Entry(call)
		if err != nil {
			t.Fatal(err)
		}
		_, err = softwareconfigs.New(client).All(ctx, call)
		requireUnsupportedCall(t, err, entry.Kind.String())
		_, err = softwaredeployments.New(client).All(ctx, call)
		requireUnsupportedCall(t, err, entry.Kind.String())
	}

	// Software writes take Field and Header only.
	_, err := softwareconfigs.New(client).Create(ctx, softwareconfigs.CreateOpts{}, request.Microversion("1.1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = softwaredeployments.New(client).Update(ctx, "d", softwaredeployments.UpdateOpts{Status: request.Present("COMPLETE")}, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")

	// Event Get takes Header only; stack FindNameOrID takes IgnoreMissing only.
	scope, err := stackevents.New(client).ForStack(stacks.StackIdentity{Name: "app", ID: "fixed"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = scope.Get(ctx, "r", "e", request.Microversion("1.1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = stacks.New(client).FindNameOrID(ctx, "app", request.Query("resolve_outputs", "False"))
	requireUnsupportedCall(t, err, "Query")
	_, err = stacks.New(client).FindNameOrID(ctx, "app", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}
}

// Shared calls apply in caller order: the later call wins.
func TestOrchestrationSharedCallsApplyInCallerOrder(t *testing.T) {
	var query string
	client, calls := orchestrationSharedCallsClient(t, func(r *http.Request) (int, string) {
		query = r.URL.RawQuery
		return 200, `{"software_configs":[]}`
	})
	ctx := context.Background()
	if _, err := softwareconfigs.New(client).All(ctx, request.Limit(5), request.Marker("a"), request.Query("vendor", "first"),
		request.Limit(2), request.Marker("b"), request.Query("vendor", "second")); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 || query != "limit=2&marker=b&vendor=second" {
		t.Fatal(calls.Load(), query)
	}

	// IgnoreMissing: a later false turns the nil result into an error.
	client, _ = orchestrationSharedCallsClient(t, func(*http.Request) (int, string) { return 404, `{}` })
	value, err := stacks.New(client).FindNameOrID(ctx, "missing", request.IgnoreMissing(true), request.IgnoreMissing(false))
	if value != nil || !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(value, err)
	}
	value, err = stacks.New(client).FindNameOrID(ctx, "missing", request.IgnoreMissing(false), request.IgnoreMissing(true))
	if value != nil || err != nil {
		t.Fatal(value, err)
	}
}
