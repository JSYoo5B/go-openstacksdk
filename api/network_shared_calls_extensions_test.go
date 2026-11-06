package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers"
	fwpolicies "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/portbindings"
	qospolicies "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func networkExtensionSharedCallsClient(t *testing.T, reply func(*http.Request) (int, string)) (*gophercloud.ServiceClient, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return client, &calls
}

// Shared calls that a category admits but a network extension operation
// never supported fail before HTTP.
func TestNetworkExtensionSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	client, calls := networkExtensionSharedCallsClient(t, func(*http.Request) (int, string) { return 500, `{}` })

	// Source record lists and finds replace headers; no single header.
	_, err := extensions.New(client).AllRecords(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = extensions.New(client).FindRecord(ctx, "x", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = services.New(client).AllRecords(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	// Agent operations.
	_, err = agents.New(client).UpdateRouterInAgent(ctx, "a", "r", request.Field("priority", 1))
	requireUnsupportedCall(t, err, "Field")
	_, err = agents.New(client).UpdateRouterInAgent(ctx, "a", "r", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	for _, err := range agents.New(client).ListDHCPNetworkRecords(ctx, "a", request.Limit(1)) {
		requireUnsupportedCall(t, err, "Limit")
	}
	for _, err := range agents.New(client).ListDHCPNetworkRecords(ctx, "a", request.Headers(nil)) {
		requireUnsupportedCall(t, err, "Headers")
	}
	_, err = agents.New(client).AllDHCPHostRecords(ctx, "n", request.Marker("m"))
	requireUnsupportedCall(t, err, "Marker")

	// Config-based writes take one header and fields only.
	_, err = peers.New(client).UpdateRecord(ctx, "p", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = fwpolicies.New(client).InsertRuleRecord(ctx, "p", "r", request.Field("x", 1))
	requireUnsupportedCall(t, err, "Field")
	_, err = attributestags.New(client).GetTags(ctx, attributestags.NewTaggedResource(attributestags.NetworkResource, "n"), request.Microversion("2.0"))
	requireUnsupportedCall(t, err, "Microversion")

	// Records with one option type for several operations.
	_, err = autoallocatedtopologies.New(client).GetRecord(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = portbindings.New(client).Create(ctx, "port", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	// QoS policies, rule types and RBAC policies.
	_, err = qospolicies.New(client).FindRecord(ctx, "x", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	requireUnsupportedCall(t, qospolicies.New(client).DeleteWithOptions(ctx, "id", request.Headers(nil)), "Headers")
	_, err = ruletypes.New(client).FindRecord(ctx, "x", request.Query("type", "x"))
	requireUnsupportedCall(t, err, "Query")
	_, err = rbacpolicies.New(client).AllRecords(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = rbacpolicies.New(client).DeleteRecord(ctx, "id", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	// Quotas consume locally and reset with ignore-missing only.
	for _, err := range quotas.New(client).ListProjects(ctx, request.Limit(1)) {
		requireUnsupportedCall(t, err, "Limit")
	}
	scope, err := quotas.New(client).InProject(ctx, resource.ID("p"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = scope.Delete(ctx, request.Microversion("2.0"))
	requireUnsupportedCall(t, err, "Microversion")

	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}
}

// Replacing headers and paging calls apply in caller order: the later call
// wins.
func TestNetworkExtensionSharedCallsApplyInCallerOrder(t *testing.T) {
	var header http.Header
	var query string
	client, _ := networkExtensionSharedCallsClient(t, func(r *http.Request) (int, string) {
		header, query = r.Header.Clone(), r.URL.RawQuery
		return 200, `{"extensions":[]}`
	})
	_, err := extensions.New(client).AllRecords(context.Background(),
		request.Headers(map[string]string{"X-First": "1"}), request.Limit(1), request.Marker("a"),
		request.Headers(map[string]string{"X-Second": "2"}), request.Limit(2), request.Marker("b"))
	if err != nil {
		t.Fatal(err)
	}
	if header.Get("X-First") != "" || header.Get("X-Second") != "2" || query != "limit=2&marker=b" {
		t.Fatal(header, query)
	}
}
