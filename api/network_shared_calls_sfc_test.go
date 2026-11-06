package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/flowclassifiers"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

func networkSharedCallsSFCClient(t *testing.T, reply func(*http.Request) (int, string)) (*gophercloud.ServiceClient, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		code, body := reply(r)
		testcloud.JSON(w, code, body)
	})
	client := cloud.Client("network", "/neutron/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return client, &calls
}

// Shared calls that a category admits but an SFC or Neutron record
// operation never supported fail before HTTP.
func TestNetworkSharedCallsSFCUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	client, calls := networkSharedCallsSFCClient(t, func(*http.Request) (int, string) { return 500, `{}` })

	// SFC records: replacing headers only, no single header or microversion.
	pairs := portpairs.New(client)
	_, err := pairs.Get(ctx, "x", request.Microversion("2.0"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = pairs.Get(ctx, "x", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = pairs.Create(ctx, request.Microversion("2.0"))
	requireUnsupportedCall(t, err, "Microversion")
	err = pairs.Delete(ctx, "x", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = pairs.Find(ctx, "x", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = flowclassifiers.New(client).All(ctx, request.Microversion("2.0"))
	requireUnsupportedCall(t, err, "Microversion")

	// Neutron records built on neutronfind: Headers and Microversion, no
	// single header.
	_, err = defaultrules.New(client).Get(ctx, "x", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = defaultstatefulness.New(client).Update(ctx, "x", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = networksegmentranges.New(client).All(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = conntrackhelpers.New(client).Delete(ctx, "r", "x", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	// ndp_proxies fixes paginated to false.
	_, err = ndpproxies.New(client).All(ctx, request.Paginated(true))
	requireUnsupportedCall(t, err, "Paginated")

	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}
}

// Replacing headers and paging calls apply in caller order: the later call
// wins.
func TestNetworkSharedCallsSFCApplyInCallerOrder(t *testing.T) {
	var header http.Header
	var query string
	client, _ := networkSharedCallsSFCClient(t, func(r *http.Request) (int, string) {
		header, query = r.Header.Clone(), r.URL.RawQuery
		return 200, `{"port_pairs":[],"default_security_group_rules":[]}`
	})
	ctx := context.Background()

	_, err := portpairs.New(client).All(ctx,
		request.Headers(map[string]string{"X-First": "1"}), request.Limit(1), request.Marker("a"),
		request.Headers(map[string]string{"X-Second": "2"}), request.Limit(2), request.Marker("b"))
	if err != nil {
		t.Fatal(err)
	}
	if header.Get("X-First") != "" || header.Get("X-Second") != "2" || query != "limit=2&marker=b" {
		t.Fatal(header, query)
	}

	_, err = defaultrules.New(client).All(ctx,
		request.Headers(map[string]string{"X-First": "1"}), request.Limit(1),
		request.Headers(map[string]string{"X-Second": "2"}), request.Limit(3))
	if err != nil {
		t.Fatal(err)
	}
	if header.Get("X-First") != "" || header.Get("X-Second") != "2" || query != "limit=3" {
		t.Fatal(header, query)
	}
}
