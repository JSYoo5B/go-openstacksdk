package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers"
	netflavors "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors"
	fwpolicies "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	localipassociations "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localipassociations"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localips"
	meteringrules "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/portbindings"
	qospolicies "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/flowclassifiers"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs"
	tapflows "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapflows"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func networkCloudSharedCallsService(t *testing.T, reply func(*http.Request) (int, string)) (*network.Service, *atomic.Int32, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		code, body := reply(r)
		testcloud.JSON(w, code, body)
	})
	client := cloud.Client("network", "/")
	client.ResourceBase = cloud.Server.URL + "/v2.0/"
	return network.New(client), &calls, cloud
}

// Shared calls that a category admits but a cloud-layer network operation
// never supported fail before HTTP.
func TestNetworkCloudSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	service, calls, _ := networkCloudSharedCallsService(t, func(*http.Request) (int, string) { return 500, `{}` })

	// Reads and deletes (GetCall): replacing headers and microversion only.
	_, err := service.DeleteFirewallRuleRecord(ctx, "r", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = service.ListNetworkRecords(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = service.ListRouterInterfaces(ctx, network.RouterInterfaceRouterID("router"), request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = service.SearchSecurityGroups(ctx, "", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	// Router interface removal takes replacing headers only.
	_, err = service.RemoveRouterInterface(ctx, "r", request.Microversion("2.1"))
	requireUnsupportedCall(t, err, "Microversion")

	// Subnet reads (ListBasePathCall): no query values or paging.
	_, err = service.ListSubnetRecords(ctx, request.Query("name", "x"))
	requireUnsupportedCall(t, err, "Query")
	_, err = service.SearchSubnetRecords(ctx, "s", request.Limit(1))
	requireUnsupportedCall(t, err, "Limit")

	// Network and floating IP creation (WriteCall): request.Field only.
	_, err = service.CreateNetwork(ctx, network.CreateNetworkRequest{Name: "n"}, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = service.UpdateNetwork(ctx, resource.ID("n"), request.Microversion("2.1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = service.FloatingIPs.Create(ctx, network.CreateFloatingIPRequest{Network: resource.ID("external")}, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")

	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}

	// QoS cloud reads and deletes check the QoS extension (and rule
	// operations their parent policy) first, then reject the call before any
	// rule request.
	qos, _, _ := networkCloudSharedCallsService(t, func(r *http.Request) (int, string) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2.0/extensions":
			return 200, `{"extensions":[{"alias":"qos","name":"QoS"}]}`
		case r.Method == http.MethodGet && r.URL.Path == "/v2.0/qos/policies/p":
			return 200, `{"policy":{"id":"p","name":"p"}}`
		}
		t.Errorf("unexpected %s %s", r.Method, r.URL)
		return 500, `{}`
	})
	_, err = qos.ListQoSPolicyRecords(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = qos.ListQoSBandwidthLimitRules(ctx, "p", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = qos.DeleteQoSBandwidthLimitRule(ctx, "p", "r", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
}

// Replacing headers and microversions apply in caller order: the later call
// wins.
func TestNetworkCloudSharedCallsApplyInCallerOrder(t *testing.T) {
	var header http.Header
	service, _, _ := networkCloudSharedCallsService(t, func(r *http.Request) (int, string) {
		header = r.Header.Clone()
		return 200, `{"ports":[]}`
	})
	_, err := service.ListRouterInterfaces(context.Background(), json.RawMessage(`{"id":"router"}`),
		request.Headers(map[string]string{"X-First": "1"}), request.Microversion("2.1"),
		request.Headers(map[string]string{"X-Second": "2"}), request.Microversion("2.5"))
	if err != nil {
		t.Fatal(err)
	}
	if header.Get("X-First") != "" || header.Get("X-Second") != "2" || header.Get("OpenStack-API-Version") != "network 2.5" {
		t.Fatal(header)
	}
}

// request.Field keeps the extension rules of network creation: later values
// win and reserved fields are rejected.
func TestNetworkCloudSharedFieldCalls(t *testing.T) {
	var body map[string]any
	service, calls, _ := networkCloudSharedCallsService(t, func(r *http.Request) (int, string) {
		var envelope struct{ Network map[string]any }
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Error(err)
		}
		body = envelope.Network
		return 201, `{"network":{"id":"created"}}`
	})
	_, err := service.CreateNetwork(context.Background(), network.CreateNetworkRequest{Name: "n"},
		request.Field("vendor:value", 1), request.Field("vendor:value", 2))
	if err != nil || body["vendor:value"] != float64(2) {
		t.Fatal(body, err)
	}
	calls.Store(0)
	_, err = service.CreateNetwork(context.Background(), network.CreateNetworkRequest{Name: "n"}, request.Field("router:external", true))
	if err == nil || calls.Load() != 0 {
		t.Fatal(err, calls.Load())
	}
}

// Shared calls that a category admits but a core network record operation
// never supported fail before HTTP.
func TestNetworkCoreSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	client, calls := networkExtensionSharedCallsClient(t, func(*http.Request) (int, string) { return 500, `{}` })

	// Record lists keep WithRecordListQuery, which classifies the key before
	// it marshals the value, so request.Query is rejected; no single header.
	_, err := networks.New(client).AllRecords(ctx, request.Query("name", "x"))
	requireUnsupportedCall(t, err, "Query")
	_, err = networks.New(client).AllRecords(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = subnets.New(client).AllRecords(ctx, request.Query("name", "x"))
	requireUnsupportedCall(t, err, "Query")
	_, err = floatingips.New(client).AllRecords(ctx, request.Query("dns_name", "x"))
	requireUnsupportedCall(t, err, "Query")
	_, err = routers.New(client).AllRecords(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = trunks.New(client).AllRecords(ctx, request.Query("name", "x"))
	requireUnsupportedCall(t, err, "Query")

	// Record finds replace headers; routers keep WithRecordFindQuery.
	_, err = networks.New(client).FindRecord(ctx, "x", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = routers.New(client).FindRecord(ctx, "x", request.Query("project_id", "p"))
	requireUnsupportedCall(t, err, "Query")

	// Deletes select ignore_missing only.
	err = addressscopes.New(client).DeleteWithOptions(ctx, "x", request.Microversion("2.0"))
	requireUnsupportedCall(t, err, "Microversion")
	err = trunks.New(client).DeleteWithOptions(ctx, "x", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")

	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}
}

// Replacing headers and paging calls apply in caller order on core record
// lists: the later call wins.
func TestNetworkCoreSharedCallsApplyInCallerOrder(t *testing.T) {
	var header http.Header
	var query string
	client, _ := networkExtensionSharedCallsClient(t, func(r *http.Request) (int, string) {
		header, query = r.Header.Clone(), r.URL.RawQuery
		return 200, `{"networks":[]}`
	})
	_, err := networks.New(client).AllRecords(context.Background(),
		request.Headers(map[string]string{"X-First": "1"}), request.Limit(1), request.Marker("a"),
		request.Headers(map[string]string{"X-Second": "2"}), request.Limit(2), request.Marker("b"))
	if err != nil {
		t.Fatal(err)
	}
	if header.Get("X-First") != "" || header.Get("X-Second") != "2" || query != "limit=2&marker=b" {
		t.Fatal(header, query)
	}
}

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

// Shared calls that a category admits but an SFC or Neutron record
// operation never supported fail before HTTP.
func TestNetworkSharedCallsSFCUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	client, calls := networkOwnedRecordClient(t, func(*http.Request) (int, string) { return 500, `{}` })

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
	client, _ := networkOwnedRecordClient(t, func(r *http.Request) (int, string) {
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
