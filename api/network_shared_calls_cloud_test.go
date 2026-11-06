package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
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
