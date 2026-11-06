package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/blacklists"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/ptr/floatingips"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/tlds"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/zoneexports"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/zoneimports"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func dnsSharedCallsClient(t *testing.T, reply func(*http.Request) (int, string)) (*gophercloud.ServiceClient, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	return cloud.Client("dns", "/dns/v2"), &calls
}

func drainDNS[T any](seq func(func(*T, error) bool)) error {
	var first error
	seq(func(_ *T, err error) bool { first = err; return false })
	return first
}

// Shared calls that a category admits but a Designate operation never
// supported fail before HTTP.
func TestDNSSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	client, calls := dnsSharedCallsClient(t, func(*http.Request) (int, string) { return 500, `{}` })

	// Lists take Query only; limit and marker are ListOpts members.
	for _, call := range []request.ListCall{request.Limit(1), request.Marker("m"), request.MaxItems(1), request.Paginated(false), request.Header("X-Test", "1"), request.Headers(nil), request.Microversion("2.1")} {
		entry, err := request.Entry(call)
		if err != nil {
			t.Fatal(err)
		}
		requireUnsupportedCall(t, drainDNS(blacklists.New(client).List(ctx, call)), entry.Kind.String())
		requireUnsupportedCall(t, drainDNS(tlds.New(client).List(ctx, call)), entry.Kind.String())
		requireUnsupportedCall(t, drainDNS(floatingips.New(client).List(ctx, call)), entry.Kind.String())
		requireUnsupportedCall(t, drainDNS(zoneexports.New(client).List(ctx, call)), entry.Kind.String())
		requireUnsupportedCall(t, drainDNS(zoneimports.New(client).List(ctx, call)), entry.Kind.String())
	}

	// Blacklist and TLD writes take Field only; the PTR set also takes Header.
	_, err := blacklists.New(client).Create(ctx, blacklists.CreateOpts{Pattern: "p"}, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = tlds.New(client).Update(ctx, "t", tlds.UpdateOpts{}, request.Microversion("2.1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = floatingips.New(client).Set(ctx, "RegionOne:fip", floatingips.SetOpts{}, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")

	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}
}

// Shared calls apply in caller order: the later call wins.
func TestDNSSharedCallsApplyInCallerOrder(t *testing.T) {
	var query string
	client, _ := dnsSharedCallsClient(t, func(r *http.Request) (int, string) {
		query = r.URL.RawQuery
		return 200, `{"blacklists":[],"links":{}}`
	})
	ctx := context.Background()
	if err := drainDNS(blacklists.New(client).List(ctx, request.Query("pattern", "first"), request.Query("pattern", ""))); err != nil {
		t.Fatal(err)
	}
	if query != "pattern=" {
		t.Fatal(query)
	}

	// Reset takes IgnoreMissing only; a later false keeps 404 an error.
	client, calls := dnsSharedCallsClient(t, func(*http.Request) (int, string) { return 404, `{}` })
	scope, err := quotas.New(client).InProject(ctx, resource.ID("project-fixed"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scope.Reset(ctx, request.IgnoreMissing(true), request.IgnoreMissing(false)); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	if value, err := scope.Reset(ctx, request.IgnoreMissing(false), request.IgnoreMissing(true)); value != nil || err != nil {
		t.Fatal(value, err)
	}
	if n := calls.Load(); n != 2 {
		t.Fatalf("HTTP requests = %d", n)
	}
}
