package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	netflavors "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors"
	localipassociations "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localipassociations"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localips"
	meteringrules "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules"
	tapflows "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapflows"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

func networkOwnedRecordClient(t *testing.T, reply func(*http.Request) (int, string)) (*gophercloud.ServiceClient, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var requests atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		code, body := reply(r)
		testcloud.JSON(w, code, body)
	})
	client := cloud.Client("network", "/neutron/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return client, &requests
}

// Shared calls that a category admits but an owned-record network operation
// never supported (a single merged request.Header) fail before HTTP.
func TestNetworkOwnedRecordSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	client, requests := networkOwnedRecordClient(t, func(*http.Request) (int, string) { return 500, `{}` })
	header := request.Header("X-Test", "1")

	lips := localips.New(client)
	_, err := lips.Create(ctx, header)
	requireUnsupportedCall(t, err, "Header")
	_, err = lips.Get(ctx, "x", header)
	requireUnsupportedCall(t, err, "Header")
	_, err = lips.Update(ctx, "x", header)
	requireUnsupportedCall(t, err, "Header")
	err = lips.Delete(ctx, "x", header)
	requireUnsupportedCall(t, err, "Header")
	_, err = lips.Find(ctx, "x", header)
	requireUnsupportedCall(t, err, "Header")
	_, err = lips.All(ctx, header)
	requireUnsupportedCall(t, err, "Header")

	_, err = localipassociations.New(client).All(ctx, "lip", header)
	requireUnsupportedCall(t, err, "Header")
	_, err = tapflows.New(client).Get(ctx, "x", header)
	requireUnsupportedCall(t, err, "Header")

	flavors := netflavors.New(client)
	_, err = flavors.Create(ctx, header)
	requireUnsupportedCall(t, err, "Header")
	_, err = flavors.Find(ctx, "x", header)
	requireUnsupportedCall(t, err, "Header")
	_, err = flavors.All(ctx, header)
	requireUnsupportedCall(t, err, "Header")
	_, err = flavors.AssociateServiceProfile(ctx, "f", "sp", header)
	requireUnsupportedCall(t, err, "Header")
	err = flavors.DisassociateServiceProfile(ctx, "f", "sp", header)
	requireUnsupportedCall(t, err, "Header")
	_, err = meteringrules.New(client).Delete(ctx, "x", header)
	requireUnsupportedCall(t, err, "Header")

	if n := requests.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}
}

// Replacing headers, microversions, query values and paging calls apply in
// caller order: the later call wins.
func TestNetworkOwnedRecordSharedCallsApplyInCallerOrder(t *testing.T) {
	ctx := context.Background()
	var header http.Header
	var query string
	client, _ := networkOwnedRecordClient(t, func(r *http.Request) (int, string) {
		header, query = r.Header.Clone(), r.URL.RawQuery
		if r.URL.Path == "/neutron/v2.0/flavors" {
			return 200, `{"flavors":[]}`
		}
		return 200, `{"local_ips":[]}`
	})

	_, err := localips.New(client).All(ctx,
		request.Headers(map[string]string{"X-First": "1"}), request.Limit(1), request.Query("name", "a"),
		request.Headers(map[string]string{"X-Second": "2"}), request.Limit(2), request.Query("name", "b"))
	if err != nil {
		t.Fatal(err)
	}
	if header.Get("X-First") != "" || header.Get("X-Second") != "2" || query != "limit=2&name=b" {
		t.Fatal(header, query)
	}

	_, err = netflavors.New(client).All(ctx,
		request.Microversion("2.1"), request.Marker("m1"),
		request.Microversion("2.2"), request.Marker("m2"))
	if err != nil {
		t.Fatal(err)
	}
	if header.Get("OpenStack-API-Version") != "network 2.2" || query != "marker=m2" {
		t.Fatal(header, query)
	}
}
