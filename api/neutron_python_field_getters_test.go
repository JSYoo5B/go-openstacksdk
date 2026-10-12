package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns"
	fwgroups "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups"
	fwpolicies "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies"
	fwrules "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type neutronPythonFieldGetterFixture struct {
	name, path, envelope, operation, kind string
	// Python Body attribute names and the wire names Python sends for them.
	fields, wireFields []string
	// A response attribute and the typed model field that must carry it.
	wireKey, wireValue, goField string
	goValue                     any
	get                         func(context.Context, *gophercloud.ServiceClient, string, ...string) (any, error)
}

// Expectations are the pinned Resource.Body aliases of each Python resource.
func neutronPythonFieldGetterFixtures() []neutronPythonFieldGetterFixture {
	return []neutronPythonFieldGetterFixture{
		{"agents", "agents", "agent", "GetWithFields", "agents",
			[]string{"is_alive", "configuration", "last_heartbeat_at"}, []string{"alive", "configurations", "heartbeat_timestamp"},
			"alive", "true", "Alive", true,
			func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
				return agents.New(c).GetWithFields(ctx, id, fields...)
			}},
		{"segments", "segments", "segment", "GetWithFields", "segments",
			[]string{"network_type", "segmentation_id", "physical_network"}, []string{"network_type", "segmentation_id", "physical_network"},
			"network_type", `"vlan"`, "NetworkType", "vlan",
			func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
				return segments.New(c).GetWithFields(ctx, id, fields...)
			}},
		{"network-ip-availabilities", "network-ip-availabilities", "network_ip_availability", "GetWithFields", "networkipavailabilities",
			[]string{"network_name", "subnet_ip_availability", "total_ips"}, []string{"network_name", "subnet_ip_availability", "total_ips"},
			"network_name", `"net"`, "NetworkName", "net",
			func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
				return networkipavailabilities.New(c).GetWithFields(ctx, id, fields...)
			}},
		{"tap-mirrors", "taas/tap_mirrors", "tap_mirror", "GetWithFields", "tapmirrors",
			[]string{"port_id", "mirror_type", "directions"}, []string{"port_id", "mirror_type", "directions"},
			"mirror_type", `"erspanv1"`, "MirrorType", "erspanv1",
			func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
				return tapmirrors.New(c).GetWithFields(ctx, id, fields...)
			}},
		{"firewall-groups", "fwaas/firewall_groups", "firewall_group", "GetWithFields", "groups",
			[]string{"ingress_firewall_policy_id", "admin_state_up", "ports"}, []string{"ingress_firewall_policy_id", "admin_state_up", "ports"},
			"ingress_firewall_policy_id", `"fwp-1"`, "IngressFirewallPolicyID", "fwp-1",
			func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
				return fwgroups.New(c).GetWithFields(ctx, id, fields...)
			}},
		{"firewall-policies", "fwaas/firewall_policies", "firewall_policy", "GetWithFields", "policies",
			[]string{"firewall_rules", "audited", "shared"}, []string{"firewall_rules", "audited", "shared"},
			"audited", "true", "Audited", true,
			func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
				return fwpolicies.New(c).GetWithFields(ctx, id, fields...)
			}},
		{"firewall-rules", "fwaas/firewall_rules", "firewall_rule", "GetWithFields", "rules",
			[]string{"destination_port", "action", "ip_version"}, []string{"destination_port", "action", "ip_version"},
			"action", `"deny"`, "Action", "deny",
			func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
				return fwrules.New(c).GetWithFields(ctx, id, fields...)
			}},
		{"vpn-services", "vpn/vpnservices", "vpnservice", "GetWithFields", "services",
			[]string{"is_admin_state_up", "external_v4_ip", "router_id"}, []string{"admin_state_up", "external_v4_ip", "router_id"},
			"external_v4_ip", `"192.0.2.1"`, "ExternalV4IP", "192.0.2.1",
			func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
				return services.New(c).GetWithFields(ctx, id, fields...)
			}},
		{"vpn-endpoint-groups", "vpn/endpoint-groups", "endpoint_group", "GetWithFields", "endpointgroups",
			[]string{"endpoints", "type", "project_id"}, []string{"endpoints", "type", "project_id"},
			"type", `"cidr"`, "Type", "cidr",
			func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
				return endpointgroups.New(c).GetWithFields(ctx, id, fields...)
			}},
		{"vpn-ike-policies", "vpn/ikepolicies", "ikepolicy", "GetWithFields", "ikepolicies",
			[]string{"ike_version", "lifetime", "units"}, []string{"ike_version", "lifetime", "units"},
			"ike_version", `"v2"`, "IKEVersion", "v2",
			func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
				return ikepolicies.New(c).GetWithFields(ctx, id, fields...)
			}},
		{"vpn-ipsec-policies", "vpn/ipsecpolicies", "ipsecpolicy", "GetWithFields", "ipsecpolicies",
			[]string{"transform_protocol", "encapsulation_mode", "value"}, []string{"transform_protocol", "encapsulation_mode", "value"},
			"transform_protocol", `"esp"`, "TransformProtocol", "esp",
			func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
				return ipsecpolicies.New(c).GetWithFields(ctx, id, fields...)
			}},
		{"vpn-ipsec-site-connections", "vpn/ipsec-site-connections", "ipsec_site_connection", "GetWithFields", "siteconnections",
			[]string{"is_admin_state_up", "peer_cidrs", "psk"}, []string{"admin_state_up", "peer_cidrs", "psk"},
			"psk", `"secret"`, "PSK", "secret",
			func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
				return siteconnections.New(c).GetWithFields(ctx, id, fields...)
			}},
		{"bgpvpns", "bgpvpn/bgpvpns", "bgpvpn", "GetWithFields", "bgpvpns",
			[]string{"route_targets", "vni", "local_pref"}, []string{"route_targets", "vni", "local_pref"},
			"vni", "1000", "VNI", 1000,
			func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
				return bgpvpns.New(c).GetWithFields(ctx, id, fields...)
			}},
		{"bgpvpn-network-associations", "bgpvpn/bgpvpns/bgpvpn-1/network_associations", "network_association", "GetNetworkAssociationWithFields", "bgpvpns",
			[]string{"network_id", "project_id", "name"}, []string{"network_id", "project_id", "name"},
			"network_id", `"net-1"`, "NetworkID", "net-1",
			func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
				return bgpvpns.New(c).GetNetworkAssociationWithFields(ctx, "bgpvpn-1", id, fields...)
			}},
		{"bgpvpn-port-associations", "bgpvpn/bgpvpns/bgpvpn-1/port_associations", "port_association", "GetPortAssociationWithFields", "bgpvpns",
			[]string{"advertise_fixed_ips", "routes", "port_id"}, []string{"advertise_fixed_ips", "routes", "port_id"},
			"port_id", `"port-1"`, "PortID", "port-1",
			func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
				return bgpvpns.New(c).GetPortAssociationWithFields(ctx, "bgpvpn-1", id, fields...)
			}},
		{"bgpvpn-router-associations", "bgpvpn/bgpvpns/bgpvpn-1/router_associations", "router_association", "GetRouterAssociationWithFields", "bgpvpns",
			[]string{"advertise_extra_routes", "router_id", "tenant_id"}, []string{"advertise_extra_routes", "router_id", "tenant_id"},
			"router_id", `"router-1"`, "RouterID", "router-1",
			func(ctx context.Context, c *gophercloud.ServiceClient, id string, fields ...string) (any, error) {
				return bgpvpns.New(c).GetRouterAssociationWithFields(ctx, "bgpvpn-1", id, fields...)
			}},
	}
}

func neutronPythonFieldGetterValue(t *testing.T, value any, name string) any {
	t.Helper()
	v := reflect.ValueOf(value)
	if !v.IsValid() || v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		t.Fatalf("expected the native model pointer, got %#v", value)
	}
	field := v.Elem().FieldByName(name)
	if !field.IsValid() {
		t.Fatalf("model %T has no field %s", value, name)
	}
	return field.Interface()
}

func TestNeutronPythonFieldGettersSendAliasedRepeatedFields(t *testing.T) {
	for _, fixture := range neutronPythonFieldGetterFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			fields := append([]string{"id", fixture.fields[0], "id"}, fixture.fields[1:]...)
			fields = append(fields, "unknown vendor:field", "")
			wire := append([]string{"id", fixture.wireFields[0], "id"}, fixture.wireFields[1:]...)
			wire = append(wire, "unknown vendor:field", "")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), url.Values{"fields": wire}) || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != "" {
					t.Errorf("member GET lost ordered fields or headers: %s %#v", r.URL, r.Header)
				}
				testcloud.JSON(w, 200, `{"`+fixture.envelope+`":{"id":"wire-id","network_id":"wire-id","`+fixture.wireKey+`":`+fixture.wireValue+`,"vendor":{"ignored":true}}}`)
			})
			original := append([]string(nil), fields...)
			value, err := fixture.get(context.Background(), neutronPythonMemberClient(cloud), "target", fields...)
			if err != nil || calls.Load() != 1 || !reflect.DeepEqual(fields, original) {
				t.Fatalf("value=%#v calls=%d fields=%q err=%v", value, calls.Load(), fields, err)
			}
			if got := neutronPythonFieldGetterValue(t, value, fixture.goField); !reflect.DeepEqual(got, fixture.goValue) {
				t.Fatalf("%s decoded %#v, want %#v", fixture.goField, got, fixture.goValue)
			}
			idField := "ID"
			if fixture.envelope == "network_ip_availability" {
				idField = "NetworkID"
			}
			if got := neutronPythonFieldGetterValue(t, value, idField); got != "wire-id" {
				t.Fatalf("%s decoded %#v", idField, got)
			}
		})
	}
}

func TestNeutronPythonFieldGettersWithoutFieldsSendPlainGet(t *testing.T) {
	for _, fixture := range neutronPythonFieldGetterFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Errorf("nil/empty fields created a query: %s", r.URL)
				}
				// A sparse row: Python returns a Resource whose other attributes are None.
				testcloud.JSON(w, 200, `{"`+fixture.envelope+`":{}}`)
			})
			for _, fields := range [][]string{nil, {}} {
				if _, err := fixture.get(context.Background(), neutronPythonMemberClient(cloud), "target", fields...); err != nil {
					t.Fatal("sparse native response rejected", err)
				}
			}
			if calls.Load() != 2 {
				t.Fatal("empty field calls were not independent", calls.Load())
			}
		})
	}
}

func TestNeutronPythonFieldGettersAcceptPythonStatusesAndWrapFailures(t *testing.T) {
	for _, fixture := range neutronPythonFieldGetterFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			for _, code := range []int{200, 203, 299, 399} {
				t.Run(fmt.Sprint(code), func(t *testing.T) {
					cloud := testcloud.New(t)
					cloud.Mux.HandleFunc("GET "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) {
						testcloud.JSON(w, code, `{"`+fixture.envelope+`":{"`+fixture.wireKey+`":`+fixture.wireValue+`}}`)
					})
					value, err := fixture.get(context.Background(), neutronPythonMemberClient(cloud), "target", fixture.fields[0])
					if err != nil || !reflect.DeepEqual(neutronPythonFieldGetterValue(t, value, fixture.goField), fixture.goValue) {
						t.Fatal("Python-profile GET rejected a success status", code, value, err)
					}
				})
			}
			for _, code := range []int{404, 500} {
				t.Run(fmt.Sprint(code), func(t *testing.T) {
					cloud := testcloud.New(t)
					cloud.Mux.HandleFunc("GET "+neutronPythonMemberBase+fixture.path+"/target", func(w http.ResponseWriter, r *http.Request) {
						testcloud.JSON(w, code, `{"error":"get-denied"}`)
					})
					_, err := fixture.get(context.Background(), neutronPythonMemberClient(cloud), "target", "id")
					var operation *resource.OperationError
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &operation) || operation.Operation != fixture.operation || operation.Resource != fixture.kind || !errors.As(err, &native) || native.Actual != code || !strings.Contains(native.URL, "/target?fields=id") || !gophercloud.ResponseCodeIs(err, code) {
						t.Fatalf("failure lost operation context or native evidence: %v", err)
					}
				})
			}
		})
	}
}

func TestNeutronPythonNetworkIPAvailabilityFieldsKeepOmittedCountersEmpty(t *testing.T) {
	for _, check := range []struct {
		name, row   string
		total, used string
		bad         bool
	}{
		{"name-only", `{"network_name":"net"}`, "", "", false},
		{"total-only", `{"total_ips":256}`, "256", "", false},
		{"both", `{"total_ips":256,"used_ips":3}`, "256", "3", false},
		{"explicit-null", `{"total_ips":null}`, "", "", true},
		{"invalid-type", `{"used_ips":false}`, "", "", true},
	} {
		t.Run(check.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("GET "+neutronPythonMemberBase+"network-ip-availabilities/net-1", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"network_ip_availability":`+check.row+`}`)
			})
			value, err := networkipavailabilities.New(neutronPythonMemberClient(cloud)).GetWithFields(context.Background(), "net-1", "network_name")
			if check.bad {
				if err == nil {
					t.Fatal("explicit null or invalid counter became a default", value)
				}
				return
			}
			if err != nil || value.TotalIPs != check.total || value.UsedIPs != check.used {
				t.Fatalf("selected counters changed: %#v %v", value, err)
			}
		})
	}
}
