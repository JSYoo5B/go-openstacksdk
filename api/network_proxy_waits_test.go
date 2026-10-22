package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	netextensions "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	bgpvpns "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns"
	netflavors "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors"
	conntrackhelpers "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers"
	ndpproxies "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies"
	localipassociations "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localipassociations"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localips"
	meteringlabels "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels"
	meteringrules "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules"
	networkipavailabilities "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges"
	qosrules "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	ruletypes "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
	netquotas "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas"
	defaultrules "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules"
	defaultstatefulness "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness"
	serviceprofiles "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles"
	flowclassifiers "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/flowclassifiers"
	portchains "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portchains"
	portpairgroups "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairgroups"
	portpairs "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/portpairs"
	servicegraphs "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/servicegraphs"
	tapflows "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapflows"
	tapservices "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapservices"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// networkWaitFixture serves one member route: the first `present` GETs
// return body, later GETs return a clean Neutron 404. Any other method or
// path fails the test, so a waiter can never mutate or retarget.
func networkWaitFixture(t *testing.T, path, body string, present int32) (*gophercloud.ServiceClient, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var gets atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != path {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			testcloud.JSON(w, 500, `{}`)
			return
		}
		if gets.Add(1) <= present {
			testcloud.JSON(w, 200, body)
			return
		}
		testcloud.JSON(w, 404, `{"NeutronError":{"type":"NotFound","message":"missing","detail":""}}`)
	})
	client := cloud.Client("network", "/neutron/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return client, &gets
}

func networkFast() []resource.WaitOption {
	return []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithTimeout(5 * time.Second)}
}

func TestNetworkProxyWaitsDeleteForCollectionlessResources(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cases := []struct {
		name, path, body string
		wait             func(*gophercloud.ServiceClient) error
	}{
		{"local IP", "/neutron/v2.0/local_ips/lip-1", `{"local_ip":{"id":"lip-1","name":"a"}}`, func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, localips.New(c).Resources(), resource.ID("lip-1"), networkFast()...)
		}},
		{"segment range", "/neutron/v2.0/network_segment_ranges/r-1", `{"network_segment_range":{"id":"r-1","name":"a"}}`, func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, networksegmentranges.New(c).Resources(), resource.ID("r-1"), networkFast()...)
		}},
		{"service profile", "/neutron/v2.0/service_profiles/sp-1", "{\"service_profile\":{\"id\":\"sp-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, serviceprofiles.New(c).Resources(), resource.ID("sp-1"), networkFast()...)
		}},
		{"flavor", "/neutron/v2.0/flavors/f-1", "{\"flavor\":{\"id\":\"f-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, netflavors.New(c).Resources(), resource.ID("f-1"), networkFast()...)
		}},
		{"metering label", "/neutron/v2.0/metering/metering-labels/ml-1", "{\"metering_label\":{\"id\":\"ml-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, meteringlabels.New(c).Resources(), resource.ID("ml-1"), networkFast()...)
		}},
		{"metering label rule", "/neutron/v2.0/metering/metering-label-rules/mr-1", "{\"metering_label_rule\":{\"id\":\"mr-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, meteringrules.New(c).Resources(), resource.ID("mr-1"), networkFast()...)
		}},
		{"tap flow", "/neutron/v2.0/taas/tap_flows/tf-1", "{\"tap_flow\":{\"id\":\"tf-1\",\"status\":\"ACTIVE\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, tapflows.New(c).Resources(), resource.ID("tf-1"), networkFast()...)
		}},
		{"tap service", "/neutron/v2.0/taas/tap_services/ts-1", "{\"tap_service\":{\"id\":\"ts-1\",\"status\":\"ACTIVE\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, tapservices.New(c).Resources(), resource.ID("ts-1"), networkFast()...)
		}},
		{"port pair", "/neutron/v2.0/sfc/port_pairs/pp-1", "{\"port_pair\":{\"id\":\"pp-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, portpairs.New(c).Resources(), resource.ID("pp-1"), networkFast()...)
		}},
		{"port pair group", "/neutron/v2.0/sfc/port_pair_groups/pg-1", "{\"port_pair_group\":{\"id\":\"pg-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, portpairgroups.New(c).Resources(), resource.ID("pg-1"), networkFast()...)
		}},
		{"port chain", "/neutron/v2.0/sfc/port_chains/pc-1", "{\"port_chain\":{\"id\":\"pc-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, portchains.New(c).Resources(), resource.ID("pc-1"), networkFast()...)
		}},
		{"flow classifier", "/neutron/v2.0/sfc/flow_classifiers/fc-1", "{\"flow_classifier\":{\"id\":\"fc-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, flowclassifiers.New(c).Resources(), resource.ID("fc-1"), networkFast()...)
		}},
		{"service graph", "/neutron/v2.0/sfc/service_graphs/sg-1", "{\"service_graph\":{\"id\":\"sg-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, servicegraphs.New(c).Resources(), resource.ID("sg-1"), networkFast()...)
		}},
		{"default security group rule", "/neutron/v2.0/default-security-group-rules/dr-1", "{\"default_security_group_rule\":{\"id\":\"dr-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, defaultrules.New(c).Resources(), resource.ID("dr-1"), networkFast()...)
		}},
		{"default statefulness", "/neutron/v2.0/security-groups-default-statefulness/ds-1", "{\"security_group_default_statefulness\":{\"id\":\"ds-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, defaultstatefulness.New(c).Resources(), resource.ID("ds-1"), networkFast()...)
		}},
		{"NDP proxy", "/neutron/v2.0/ndp_proxies/np-1", "{\"ndp_proxy\":{\"id\":\"np-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, ndpproxies.New(c).Resources(), resource.ID("np-1"), networkFast()...)
		}},
		{"local IP association", "/neutron/v2.0/local_ips/lip-1/port_associations/pa-1", "{\"port_association\":{\"id\":\"pa-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, localipassociations.New(c).Resources("lip-1"), resource.ID("pa-1"), networkFast()...)
		}},
		{"conntrack helper", "/neutron/v2.0/routers/r-1/conntrack_helpers/ch-1", "{\"conntrack_helper\":{\"id\":\"ch-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, conntrackhelpers.New(c).Resources("r-1"), resource.ID("ch-1"), networkFast()...)
		}},
		{"quota", "/neutron/v2.0/quotas/p-1", "{\"quota\":{\"network\":10}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, netquotas.New(c).Resources(), resource.ID("p-1"), networkFast()...)
		}},
		{"network IP availability", "/neutron/v2.0/network-ip-availabilities/n-1", `{"network_ip_availability":{"network_id":"n-1","total_ips":"10","used_ips":"1","subnet_ip_availability":[]}}`, func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, networkipavailabilities.New(c).Resources(), resource.ID("n-1"), networkFast()...)
		}},
		{"QoS rule type", "/neutron/v2.0/qos/rule-types/bandwidth_limit", "{\"rule_type\":{\"type\":\"bandwidth_limit\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, ruletypes.New(c).Resources(), resource.ID("bandwidth_limit"), networkFast()...)
		}},
		{"minimum packet rate rule", "/neutron/v2.0/qos/policies/qp-1/minimum_packet_rate_rules/mp-1", "{\"minimum_packet_rate_rule\":{\"id\":\"mp-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, qosrules.New(c).MinimumPacketRateResources("qp-1"), resource.ID("mp-1"), networkFast()...)
		}},
		{"packet rate limit rule", "/neutron/v2.0/qos/policies/qp-1/packet_rate_limit_rules/pr-1", "{\"packet_rate_limit_rule\":{\"id\":\"pr-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, qosrules.New(c).PacketRateLimitResources("qp-1"), resource.ID("pr-1"), networkFast()...)
		}},
		{"BGP VPN network association", "/neutron/v2.0/bgpvpn/bgpvpns/b-1/network_associations/na-1", "{\"network_association\":{\"id\":\"na-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, bgpvpns.New(c).NetworkAssociationResources("b-1"), resource.ID("na-1"), networkFast()...)
		}},
		{"BGP VPN port association", "/neutron/v2.0/bgpvpn/bgpvpns/b-1/port_associations/pa-1", "{\"port_association\":{\"id\":\"pa-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, bgpvpns.New(c).PortAssociationResources("b-1"), resource.ID("pa-1"), networkFast()...)
		}},
		{"BGP VPN router association", "/neutron/v2.0/bgpvpn/bgpvpns/b-1/router_associations/ra-1", "{\"router_association\":{\"id\":\"ra-1\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, bgpvpns.New(c).RouterAssociationResources("b-1"), resource.ID("ra-1"), networkFast()...)
		}},
		{"extension", "/neutron/v2.0/extensions/dns-integration", "{\"extension\":{\"alias\":\"dns-integration\"}}", func(c *gophercloud.ServiceClient) error {
			return network.WaitForDelete(ctx, netextensions.New(c).Resources(), resource.ID("dns-integration"), networkFast()...)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client, gets := networkWaitFixture(t, tc.path, tc.body, 2)
			if err := tc.wait(client); err != nil || gets.Load() != 3 {
				t.Fatalf("err=%v gets=%d", err, gets.Load())
			}
		})
	}
}

func TestNetworkProxyWaitsStatusDefaultsAttributesAndTerminalErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// TapFlow declares status: the default attribute is selected, the target is
	// case-insensitive and ERROR fails.
	cloud := testcloud.New(t)
	var flowGets atomic.Int32
	cloud.Mux.HandleFunc("/neutron/v2.0/taas/tap_flows/tf-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected %s", r.Method)
		}
		status := "PENDING_CREATE"
		if flowGets.Add(1) >= 2 {
			status = "active"
		}
		testcloud.JSON(w, 200, `{"tap_flow":{"id":"tf-1","status":"`+status+`"}}`)
	})
	cloud.Mux.HandleFunc("/neutron/v2.0/taas/tap_flows/tf-2", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"tap_flow":{"id":"tf-2","status":"ERROR"}}`)
	})
	cloud.Mux.HandleFunc("/neutron/v2.0/taas/tap_flows/tf-3", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"tap_flow":{"id":"tf-3","status":"DELETED"}}`)
	})
	var lipGets atomic.Int32
	cloud.Mux.HandleFunc("/neutron/v2.0/local_ips/lip-1", func(w http.ResponseWriter, r *http.Request) {
		mode := "translate"
		if lipGets.Add(1) >= 2 {
			mode = "passthrough"
		}
		testcloud.JSON(w, 200, `{"local_ip":{"id":"lip-1","ip_mode":"`+mode+`"}}`)
	})
	var deniedGets atomic.Int32
	cloud.Mux.HandleFunc("/neutron/v2.0/local_ips/lip-2", func(w http.ResponseWriter, r *http.Request) {
		deniedGets.Add(1)
		testcloud.JSON(w, 403, `{"NeutronError":{"type":"Forbidden","message":"denied","detail":""}}`)
	})
	client := cloud.Client("network", "/neutron/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	fast := resource.WithPollInterval(time.Millisecond)

	flows := tapflows.New(client).Resources()
	flow, err := network.WaitForStatus(ctx, flows, resource.ID("tf-1"), "ACTIVE", fast)
	if err != nil || flow.Status == nil || *flow.Status != "active" || flowGets.Load() != 2 {
		t.Fatalf("flow=%v err=%v gets=%d", flow, err, flowGets.Load())
	}
	if _, err := network.WaitForStatus(ctx, flows, resource.ID("tf-2"), "ACTIVE", fast); !errors.Is(err, resource.ErrFailedState) {
		t.Fatalf("ERROR err=%v", err)
	}
	if err := network.WaitForDelete(ctx, flows, resource.ID("tf-3"), fast); err != nil {
		t.Fatalf("deleted status err=%v", err)
	}

	// LocalIP has no status attribute: the default fails before HTTP and an
	// explicit exported string attribute is compared case-insensitively.
	lips := localips.New(client).Resources()
	if _, err := network.WaitForStatus(ctx, lips, resource.ID("lip-1"), "PASSTHROUGH", fast); !errors.Is(err, resource.ErrUnsupported) || lipGets.Load() != 0 {
		t.Fatalf("statusless err=%v gets=%d", err, lipGets.Load())
	}
	lip, err := network.WaitForStatus(ctx, lips, resource.ID("lip-1"), "PASSTHROUGH", fast, resource.WithStatusAttribute("IPMode"))
	if err != nil || lip.IPMode == nil || *lip.IPMode != "passthrough" || lipGets.Load() != 2 {
		t.Fatalf("lip=%v err=%v gets=%d", lip, err, lipGets.Load())
	}

	// A 403 is not absence: deletion waiting fails after one GET.
	if err := network.WaitForDelete(ctx, lips, resource.ID("lip-2"), fast); err == nil || errors.Is(err, resource.ErrNotFound) || deniedGets.Load() != 1 {
		t.Fatalf("403 err=%v gets=%d", err, deniedGets.Load())
	}
}
