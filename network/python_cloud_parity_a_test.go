package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	fwgroups "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups"
	fwpolicies "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies"
	fwrules "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	qospolicies "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	qosrules "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonCloudACall struct{ method, path, query, body string }

type pythonCloudATransport func(*http.Request) (*http.Response, error)

func (f pythonCloudATransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type pythonCloudAReply struct {
	code int
	body string
}

func pythonCloudAWire(code int, body string) pythonCloudAReply { return pythonCloudAReply{code, body} }

// pythonCloudAClient records every request and answers from route, keyed by
// "METHOD path?query" relative to /neutron/v2.0.
func pythonCloudAClient(t *testing.T, calls *[]pythonCloudACall, route map[string]pythonCloudAReply) *gophercloud.ServiceClient {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonCloudATransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		path := strings.TrimPrefix(req.URL.Path, "/neutron/v2.0")
		*calls = append(*calls, pythonCloudACall{req.Method, path, req.URL.RawQuery, raw})
		key := req.Method + " " + path
		if req.URL.RawQuery != "" {
			key += "?" + req.URL.RawQuery
		}
		reply, ok := route[key]
		if !ok {
			reply = pythonCloudAWire(404, `{"NeutronError":{"type":"NotFound","message":"`+key+`"}}`)
		}
		return &http.Response{StatusCode: reply.code, Body: io.NopCloser(strings.NewReader(reply.body)), Header: http.Header{"Content-Type": {"application/json"}}, Request: req}, nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return client
}

func pythonCloudARequests(calls []pythonCloudACall) []string {
	out := make([]string, 0, len(calls))
	for _, call := range calls {
		value := call.method + " " + call.path
		if call.query != "" {
			value += "?" + call.query
		}
		out = append(out, value)
	}
	return out
}

func pythonCloudAExpect(t *testing.T, calls []pythonCloudACall, want ...string) {
	t.Helper()
	if got := pythonCloudARequests(calls); !reflect.DeepEqual(got, want) {
		t.Fatalf("requests:\n got %q\nwant %q", got, want)
	}
}

func pythonCloudABody(t *testing.T, raw string, want string) {
	t.Helper()
	var got, expected any
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("body %q: %v", raw, err)
	}
	if err := json.Unmarshal([]byte(want), &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("body:\n got %s\nwant %s", raw, want)
	}
}

func pythonCloudAID(t *testing.T, value *resource.RawResource) string {
	t.Helper()
	var id string
	if value == nil || json.Unmarshal(value.Body["id"], &id) != nil {
		t.Fatalf("record id missing: %#v", value)
	}
	return id
}

// get_network/get_router/get_subnet/get_port call find_* with
// ignore_missing=True and pass filters to both the member GET and the list.
func TestPythonCloudNetworkGetHelpersFindByIDThenNameWithFilters(t *testing.T) {
	for _, kind := range []string{"networks", "routers", "subnets", "ports"} {
		t.Run(kind, func(t *testing.T) {
			singular := strings.TrimSuffix(kind, "s")
			var calls []pythonCloudACall
			client := pythonCloudAClient(t, &calls, map[string]pythonCloudAReply{
				"GET /" + kind + "?name=app&project_id=p":  pythonCloudAWire(200, `{"`+kind+`":[{"id":"found","name":"app","project_id":"p"}]}`),
				"GET /" + kind + "/direct?project_id=p":    pythonCloudAWire(200, `{"`+singular+`":{"id":"direct","name":"other"}}`),
				"GET /" + kind + "?name=gone&project_id=p": pythonCloudAWire(200, `{"`+kind+`":[]}`),
			})
			find := func(identity string) (*resource.RawResource, error) {
				ctx := context.Background()
				switch kind {
				case "networks":
					value, err := networks.New(client).FindRecord(ctx, identity, networks.WithRecordFindIgnoreMissing(true), networks.WithRecordFindQuery("project_id", "p"))
					if value == nil {
						return nil, err
					}
					return value.Resource, err
				case "routers":
					value, err := routers.New(client).FindRecord(ctx, identity, routers.WithRecordFindIgnoreMissing(true), routers.WithRecordFindQuery("project_id", "p"))
					if value == nil {
						return nil, err
					}
					return value.Resource, err
				case "subnets":
					value, err := subnets.New(client).FindRecord(ctx, identity, subnets.WithRecordFindIgnoreMissing(true), subnets.WithRecordFindQuery("project_id", "p"))
					if value == nil {
						return nil, err
					}
					return value.Resource, err
				default:
					value, err := ports.New(client).FindRecord(ctx, identity, ports.WithRecordFindIgnoreMissing(true), ports.WithRecordFindQuery("project_id", "p"))
					if value == nil {
						return nil, err
					}
					return value.Resource, err
				}
			}
			value, err := find("app")
			if err != nil || pythonCloudAID(t, value) != "found" {
				t.Fatal(value, err)
			}
			value, err = find("direct")
			if err != nil || pythonCloudAID(t, value) != "direct" {
				t.Fatal(value, err)
			}
			value, err = find("gone")
			if err != nil || value != nil {
				t.Fatal("missing must be nil, nil", value, err)
			}
			pythonCloudAExpect(t, calls,
				"GET /"+kind+"/app?project_id=p", "GET /"+kind+"?name=app&project_id=p",
				"GET /"+kind+"/direct?project_id=p",
				"GET /"+kind+"/gone?project_id=p", "GET /"+kind+"?name=gone&project_id=p")
		})
	}
}

// get_network_by_id and get_port_by_id are a plain GET that raises on 404.
func TestPythonCloudNetworkGetByIDIsPlainGet(t *testing.T) {
	var calls []pythonCloudACall
	client := pythonCloudAClient(t, &calls, map[string]pythonCloudAReply{
		"GET /networks/n1": pythonCloudAWire(200, `{"network":{"id":"n1","name":"net","status":"ACTIVE"}}`),
		"GET /ports/p1":    pythonCloudAWire(200, `{"port":{"id":"p1","network_id":"n1","name":"port"}}`),
	})
	ctx := context.Background()
	network, err := networks.New(client).Get(ctx, "n1")
	if err != nil || network.ID != "n1" || network.Name != "net" || network.Status != "ACTIVE" {
		t.Fatal(network, err)
	}
	port, err := ports.New(client).Get(ctx, "p1")
	if err != nil || port.ID != "p1" || port.NetworkID != "n1" {
		t.Fatal(port, err)
	}
	if _, err := networks.New(client).Get(ctx, "missing"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal("network 404 must be an error", err)
	}
	if _, err := ports.New(client).Get(ctx, "missing"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal("port 404 must be an error", err)
	}
	pythonCloudAExpect(t, calls, "GET /networks/n1", "GET /ports/p1", "GET /networks/missing", "GET /ports/missing")
}

// get_network_extensions lists /extensions once and keeps the aliases.
func TestPythonCloudNetworkExtensionsListAliases(t *testing.T) {
	var calls []pythonCloudACall
	client := pythonCloudAClient(t, &calls, map[string]pythonCloudAReply{
		"GET /extensions": pythonCloudAWire(200, `{"extensions":[{"alias":"qos","name":"QoS"},{"alias":"router","name":"L3"},{"alias":"qos","name":"dup"}]}`),
	})
	aliases := map[string]bool{}
	for extension, err := range extensions.New(client).List(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		aliases[extension.Alias] = true
	}
	got := make([]string, 0, len(aliases))
	for alias := range aliases {
		got = append(got, alias)
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, []string{"qos", "router"}) {
		t.Fatal(got)
	}
	pythonCloudAExpect(t, calls, "GET /extensions")
}

// delete_port/delete_router/delete_subnet: find with ignore_missing=True,
// return false without DELETE when absent, otherwise DELETE the found ID with
// the proxy default ignore_missing=True and return true.
func TestPythonCloudNetworkDeleteHelpersFindThenDelete(t *testing.T) {
	for _, kind := range []string{"ports", "routers", "subnets"} {
		for _, test := range []struct {
			name, identity string
			deleteCode     int
			wantFound      bool
			wantErr        bool
			want           []string
		}{
			{"found by name, 204", "app", 204, true, false, []string{"GET /K/app", "GET /K?name=app", "DELETE /K/id-1"}},
			{"found by name, delete race 404", "app", 404, true, false, []string{"GET /K/app", "GET /K?name=app", "DELETE /K/id-1"}},
			{"found by name, 409", "app", 409, true, true, []string{"GET /K/app", "GET /K?name=app", "DELETE /K/id-1"}},
			{"absent", "gone", 0, false, false, []string{"GET /K/gone", "GET /K?name=gone"}},
		} {
			t.Run(kind+"/"+test.name, func(t *testing.T) {
				var calls []pythonCloudACall
				route := map[string]pythonCloudAReply{
					"GET /" + kind + "?name=app":  pythonCloudAWire(200, `{"`+kind+`":[{"id":"id-1","name":"app"}]}`),
					"GET /" + kind + "?name=gone": pythonCloudAWire(200, `{"`+kind+`":[]}`),
				}
				if test.deleteCode != 0 {
					route["DELETE /"+kind+"/id-1"] = pythonCloudAWire(test.deleteCode, "")
				}
				client := pythonCloudAClient(t, &calls, route)
				ctx := context.Background()
				var found *resource.RawResource
				var err error
				switch kind {
				case "ports":
					value, findErr := ports.New(client).FindRecord(ctx, test.identity, ports.WithRecordFindIgnoreMissing(true))
					if err = findErr; value != nil {
						found = value.Resource
						err = ports.New(client).DeleteWithOptions(ctx, pythonCloudAID(t, found))
					}
				case "routers":
					value, findErr := routers.New(client).FindRecord(ctx, test.identity, routers.WithRecordFindIgnoreMissing(true))
					if err = findErr; value != nil {
						found = value.Resource
						err = routers.New(client).DeleteWithOptions(ctx, pythonCloudAID(t, found))
					}
				default:
					value, findErr := subnets.New(client).FindRecord(ctx, test.identity, subnets.WithRecordFindIgnoreMissing(true))
					if err = findErr; value != nil {
						found = value.Resource
						err = subnets.New(client).DeleteWithOptions(ctx, pythonCloudAID(t, found))
					}
				}
				if (found != nil) != test.wantFound || (err != nil) != test.wantErr {
					t.Fatal(found, err)
				}
				want := make([]string, 0, len(test.want))
				for _, value := range test.want {
					want = append(want, strings.ReplaceAll(value, "/K", "/"+kind))
				}
				pythonCloudAExpect(t, calls, want...)
				for _, call := range calls {
					if call.method == http.MethodDelete && call.body != "" {
						t.Fatal("DELETE must not carry a body", call)
					}
				}
			})
		}
	}
}

// add_router_interface forwards subnet_id/port_id as subnet/port: a truthy
// port wins, otherwise subnet_id (even null) is sent; the raw JSON is returned.
func TestPythonCloudNetworkAddRouterInterfaceBodies(t *testing.T) {
	for _, test := range []struct {
		name    string
		options []routers.RouterInterfaceOption
		body    string
	}{
		{"subnet", []routers.RouterInterfaceOption{routers.WithRouterInterfaceSubnet("s1")}, `{"subnet_id":"s1"}`},
		{"port wins over subnet", []routers.RouterInterfaceOption{routers.WithRouterInterfaceSubnet("s1"), routers.WithRouterInterfacePort("p1")}, `{"port_id":"p1"}`},
		{"no arguments", nil, `{"subnet_id":null}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []pythonCloudACall
			client := pythonCloudAClient(t, &calls, map[string]pythonCloudAReply{
				"PUT /routers/r1/add_router_interface": pythonCloudAWire(200, `{"id":"r1","subnet_id":"s1","port_id":"p9","tenant_id":"t"}`),
			})
			result, err := routers.New(client).AddInterfaceRecord(context.Background(), "r1", test.options...)
			if err != nil || len(calls) != 1 {
				t.Fatal(result, err, calls)
			}
			pythonCloudAExpect(t, calls, "PUT /routers/r1/add_router_interface")
			pythonCloudABody(t, calls[0].body, test.body)
			pythonCloudABody(t, string(result.Value), `{"id":"r1","subnet_id":"s1","port_id":"p9","tenant_id":"t"}`)
		})
	}
}

// create_port sets network_id and forwards the cloud kwargs unchanged.
func TestPythonCloudNetworkCreatePortForwardsKwargs(t *testing.T) {
	var calls []pythonCloudACall
	client := pythonCloudAClient(t, &calls, map[string]pythonCloudAReply{
		"POST /ports": pythonCloudAWire(201, `{"port":{"id":"p1","network_id":"n1","name":"web","admin_state_up":false}}`),
	})
	port, err := ports.New(client).CreateWithAttributes(context.Background(),
		ports.WithAttribute("network_id", "n1"),
		ports.WithAttribute("name", "web"),
		ports.WithAttribute("admin_state_up", false),
		ports.WithAttribute("binding:vnic_type", "direct"),
		ports.WithAttribute("fixed_ips", []map[string]string{{"subnet_id": "s1", "ip_address": "10.0.0.5"}}),
	)
	if err != nil || port.ID != "p1" || port.NetworkID != "n1" || port.AdminStateUp {
		t.Fatal(port, err)
	}
	pythonCloudAExpect(t, calls, "POST /ports")
	pythonCloudABody(t, calls[0].body, `{"port":{"network_id":"n1","name":"web","admin_state_up":false,"binding:vnic_type":"direct","fixed_ips":[{"subnet_id":"s1","ip_address":"10.0.0.5"}]}}`)

	calls = nil
	if _, err := ports.New(client).CreateWithAttributes(context.Background(), ports.WithAttribute("network_id", "n1")); err != nil {
		t.Fatal(err)
	}
	pythonCloudABody(t, calls[0].body, `{"port":{"network_id":"n1"}}`)
}

func pythonCloudAFirewallRuleIDs(t *testing.T, api *fwrules.API, identities ...string) ([]string, error) {
	t.Helper()
	ctx := context.Background()
	ids := make([]string, 0, len(identities))
	for _, identity := range identities {
		rule, err := api.Find(ctx, resource.ID(identity), resource.WithIgnoreMissing())
		if err == nil && rule == nil {
			rule, err = api.Find(ctx, resource.Name(identity))
		}
		if err != nil {
			return nil, err
		}
		ids = append(ids, rule.ID)
	}
	return ids, nil
}

// create_firewall_policy resolves every firewall_rules entry with
// find_firewall_rule(ignore_missing=False) before one POST.
func TestPythonCloudFirewallPolicyCreateResolvesRuleNames(t *testing.T) {
	var calls []pythonCloudACall
	client := pythonCloudAClient(t, &calls, map[string]pythonCloudAReply{
		"GET /fwaas/firewall_rules/r1":        pythonCloudAWire(200, `{"firewall_rule":{"id":"r1","name":"ssh"}}`),
		"GET /fwaas/firewall_rules?name=web":  pythonCloudAWire(200, `{"firewall_rules":[{"id":"w1","name":"web"}]}`),
		"GET /fwaas/firewall_rules?name=nope": pythonCloudAWire(200, `{"firewall_rules":[]}`),
		"POST /fwaas/firewall_policies":       pythonCloudAWire(201, `{"firewall_policy":{"id":"fp","name":"pol","firewall_rules":["r1","w1"],"audited":true}}`),
	})
	ids, err := pythonCloudAFirewallRuleIDs(t, fwrules.New(client), "r1", "web")
	if err != nil || !reflect.DeepEqual(ids, []string{"r1", "w1"}) {
		t.Fatal(ids, err)
	}
	audited := true
	policy, err := fwpolicies.New(client).Create(context.Background(), fwpolicies.CreateOpts{Name: "pol", FirewallRules: ids, Audited: &audited})
	if err != nil || policy.ID != "fp" || !reflect.DeepEqual(policy.Rules, []string{"r1", "w1"}) {
		t.Fatal(policy, err)
	}
	pythonCloudAExpect(t, calls, "GET /fwaas/firewall_rules/r1", "GET /fwaas/firewall_rules/web", "GET /fwaas/firewall_rules?name=web", "POST /fwaas/firewall_policies")
	pythonCloudABody(t, calls[3].body, `{"firewall_policy":{"name":"pol","firewall_rules":["r1","w1"],"audited":true}}`)

	calls = nil
	_, err = pythonCloudAFirewallRuleIDs(t, fwrules.New(client), "nope")
	if !errors.Is(err, resource.ErrNotFound) {
		t.Fatal("missing rule must be NotFound before POST", err)
	}
	pythonCloudAExpect(t, calls, "GET /fwaas/firewall_rules/nope", "GET /fwaas/firewall_rules?name=nope")
}

// create_firewall_group maps ingress/egress policy names to *_policy_id via
// find_firewall_policy(ignore_missing=False) and port names to IDs via
// get_port, then POSTs once.
func TestPythonCloudFirewallGroupCreateResolvesPoliciesAndPorts(t *testing.T) {
	var calls []pythonCloudACall
	client := pythonCloudAClient(t, &calls, map[string]pythonCloudAReply{
		"GET /fwaas/firewall_policies?name=in": pythonCloudAWire(200, `{"firewall_policies":[{"id":"in-id","name":"in"}]}`),
		"GET /fwaas/firewall_policies/out-id":  pythonCloudAWire(200, `{"firewall_policy":{"id":"out-id","name":"out"}}`),
		"GET /ports?name=web":                  pythonCloudAWire(200, `{"ports":[{"id":"port-1","name":"web"}]}`),
		"GET /ports?name=gone":                 pythonCloudAWire(200, `{"ports":[]}`),
		"POST /fwaas/firewall_groups":          pythonCloudAWire(201, `{"firewall_group":{"id":"g1","name":"grp","ingress_firewall_policy_id":"in-id","egress_firewall_policy_id":"out-id","ports":["port-1"],"status":"INACTIVE"}}`),
	})
	ctx := context.Background()
	policyID := func(identity string) string {
		policy, err := fwpolicies.New(client).Find(ctx, resource.ID(identity), resource.WithIgnoreMissing())
		if err == nil && policy == nil {
			policy, err = fwpolicies.New(client).Find(ctx, resource.Name(identity))
		}
		if err != nil {
			t.Fatal(err)
		}
		return policy.ID
	}
	ingress, egress := policyID("in"), policyID("out-id")
	port, err := ports.New(client).FindRecord(ctx, "web", ports.WithRecordFindIgnoreMissing(false))
	if err != nil {
		t.Fatal(err)
	}
	adminUp := true
	group, err := fwgroups.New(client).Create(ctx, fwgroups.CreateOpts{Name: "grp", IngressFirewallPolicyID: ingress, EgressFirewallPolicyID: egress, Ports: []string{pythonCloudAID(t, port.Resource)}, AdminStateUp: &adminUp})
	if err != nil || group.ID != "g1" || group.IngressFirewallPolicyID != "in-id" || group.EgressFirewallPolicyID != "out-id" || !reflect.DeepEqual(group.Ports, []string{"port-1"}) {
		t.Fatal(group, err)
	}
	pythonCloudAExpect(t, calls,
		"GET /fwaas/firewall_policies/in", "GET /fwaas/firewall_policies?name=in",
		"GET /fwaas/firewall_policies/out-id",
		"GET /ports/web", "GET /ports?name=web",
		"POST /fwaas/firewall_groups")
	pythonCloudABody(t, calls[5].body, `{"firewall_group":{"name":"grp","ingress_firewall_policy_id":"in-id","egress_firewall_policy_id":"out-id","ports":["port-1"],"admin_state_up":true}}`)

	calls = nil
	if _, err := ports.New(client).FindRecord(ctx, "gone", ports.WithRecordFindIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal("missing port must be NotFound before POST", err)
	}
	pythonCloudAExpect(t, calls, "GET /ports/gone", "GET /ports?name=gone")
}

// Evidence for unresolved fwaas rows: rules.Create cannot send Python's
// create_firewall_rule() body without protocol/action, and the generic
// Find(ID) does not fall back to the name list after 400/403.
func TestPythonCloudFirewallGapsCreateRuleAndFindFallback(t *testing.T) {
	var calls []pythonCloudACall
	client := pythonCloudAClient(t, &calls, map[string]pythonCloudAReply{
		"POST /fwaas/firewall_rules":        pythonCloudAWire(201, `{"firewall_rule":{"id":"r1","protocol":"tcp","action":"allow","name":"ssh"}}`),
		"GET /fwaas/firewall_rules/bad-req": pythonCloudAWire(400, `{"NeutronError":{"type":"BadRequest"}}`),
		"GET /fwaas/firewall_groups/denied": pythonCloudAWire(403, `{"NeutronError":{"type":"Forbidden"}}`),
	})
	ctx := context.Background()
	if _, err := fwrules.New(client).Create(ctx, fwrules.CreateOpts{Name: "ssh"}); err == nil {
		t.Fatal("rules.Create must reject a body without protocol and action")
	}
	if len(calls) != 0 {
		t.Fatal("no HTTP expected", calls)
	}
	rule, err := fwrules.New(client).Create(ctx, fwrules.CreateOpts{Name: "ssh", Protocol: fwrules.ProtocolTCP, Action: fwrules.ActionAllow}, fwrules.WithCreateField("source_firewall_group_id", "g1"))
	if err != nil || rule.ID != "r1" {
		t.Fatal(rule, err)
	}
	pythonCloudABody(t, calls[0].body, `{"firewall_rule":{"name":"ssh","protocol":"tcp","action":"allow","source_firewall_group_id":"g1"}}`)

	calls = nil
	if value, err := fwrules.New(client).Find(ctx, resource.ID("bad-req"), resource.WithIgnoreMissing()); err == nil || value != nil {
		t.Fatal("400 must stay an error", value, err)
	}
	if value, err := fwgroups.New(client).Find(ctx, resource.ID("denied"), resource.WithIgnoreMissing()); err == nil || value != nil {
		t.Fatal("403 must stay an error", value, err)
	}
	pythonCloudAExpect(t, calls, "GET /fwaas/firewall_rules/bad-req", "GET /fwaas/firewall_groups/denied")
}

// Evidence for unresolved create_router/create_subnet: the record mutations
// send only explicit attributes, so cloud defaults are not applied.
func TestPythonCloudNetworkCreateRouterSubnetSendOnlyExplicitAttributes(t *testing.T) {
	var calls []pythonCloudACall
	client := pythonCloudAClient(t, &calls, map[string]pythonCloudAReply{
		"POST /routers": pythonCloudAWire(201, `{"router":{"id":"r1","admin_state_up":true}}`),
		"POST /subnets": pythonCloudAWire(201, `{"subnet":{"id":"s1","network_id":"n1","ip_version":4,"cidr":"10.0.0.0/24"}}`),
	})
	ctx := context.Background()
	if _, err := routers.New(client).CreateWithAttributes(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := routers.New(client).CreateWithAttributes(ctx, routers.WithAttribute("admin_state_up", true), routers.WithAttribute("external_gateway_info", map[string]any{"network_id": "ext", "enable_snat": false})); err != nil {
		t.Fatal(err)
	}
	if _, err := subnets.New(client).CreateWithAttributes(ctx, subnets.WithAttribute("network_id", "n1"), subnets.WithAttribute("cidr", "10.0.0.0/24")); err != nil {
		t.Fatal(err)
	}
	if _, err := subnets.New(client).CreateWithAttributes(ctx, subnets.WithAttribute("network_id", "n1"), subnets.WithAttribute("ip_version", 4), subnets.WithAttribute("enable_dhcp", false), subnets.WithAttribute("cidr", "10.0.0.0/24"), subnets.WithAttribute("gateway_ip", nil)); err != nil {
		t.Fatal(err)
	}
	pythonCloudAExpect(t, calls, "POST /routers", "POST /routers", "POST /subnets", "POST /subnets")
	pythonCloudABody(t, calls[0].body, `{"router":{}}`)
	pythonCloudABody(t, calls[1].body, `{"router":{"admin_state_up":true,"external_gateway_info":{"network_id":"ext","enable_snat":false}}}`)
	pythonCloudABody(t, calls[2].body, `{"subnet":{"network_id":"n1","cidr":"10.0.0.0/24"}}`)
	pythonCloudABody(t, calls[3].body, `{"subnet":{"network_id":"n1","ip_version":4,"enable_dhcp":false,"cidr":"10.0.0.0/24","gateway_ip":null}}`)
}

// Evidence for unresolved QoS rows: each request after the cloud
// extension gate is available, but the gate itself and the rule delete 404
// to false conversion are not SDK behavior.
func TestPythonCloudQoSRequestsAfterExtensionGate(t *testing.T) {
	var calls []pythonCloudACall
	client := pythonCloudAClient(t, &calls, map[string]pythonCloudAReply{
		"GET /extensions":                                  pythonCloudAWire(200, `{"extensions":[{"alias":"qos"},{"alias":"qos-rule-type-details"}]}`),
		"GET /qos/policies?name=gold":                      pythonCloudAWire(200, `{"policies":[{"id":"qp","name":"gold"}]}`),
		"GET /qos/policies/qp/bandwidth_limit_rules/b1":    pythonCloudAWire(200, `{"bandwidth_limit_rule":{"id":"b1","max_kbps":100,"direction":"egress"}}`),
		"GET /qos/policies/qp/dscp_marking_rules/d1":       pythonCloudAWire(200, `{"dscp_marking_rule":{"id":"d1","dscp_mark":26}}`),
		"GET /qos/policies/qp/minimum_bandwidth_rules/m1":  pythonCloudAWire(200, `{"minimum_bandwidth_rule":{"id":"m1","min_kbps":10,"direction":"egress"}}`),
		"POST /qos/policies/qp/bandwidth_limit_rules":      pythonCloudAWire(201, `{"bandwidth_limit_rule":{"id":"b2","max_kbps":100}}`),
		"POST /qos/policies/qp/dscp_marking_rules":         pythonCloudAWire(201, `{"dscp_marking_rule":{"id":"d2","dscp_mark":26}}`),
		"POST /qos/policies/qp/minimum_bandwidth_rules":    pythonCloudAWire(201, `{"minimum_bandwidth_rule":{"id":"m2","min_kbps":10}}`),
		"DELETE /qos/policies/qp/bandwidth_limit_rules/b1": pythonCloudAWire(204, ""),
		"GET /qos/rule-types/bandwidth_limit":              pythonCloudAWire(200, `{"rule_type":{"type":"bandwidth_limit","drivers":[{"name":"ovs"}]}}`),
		"POST /qos/policies":                               pythonCloudAWire(201, `{"policy":{"id":"qp2","name":"silver","is_default":true}}`),
		"DELETE /qos/policies/qp":                          pythonCloudAWire(204, ""),
	})
	ctx := context.Background()
	for _, err := range extensions.New(client).List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
	}
	policy, err := qospolicies.New(client).FindRecord(ctx, "gold", qospolicies.WithRecordFindIgnoreMissing(true))
	if err != nil || pythonCloudAID(t, policy.Resource) != "qp" {
		t.Fatal(policy, err)
	}
	rules := qosrules.New(client)
	if value, err := rules.GetBandwidthLimitRecord(ctx, "qp", "b1"); err != nil || string(value.Resource.Body["max_kbps"]) != "100" {
		t.Fatal(value, err)
	}
	if value, err := rules.GetDSCPMarkingRecord(ctx, "qp", "d1"); err != nil || string(value.Resource.Body["dscp_mark"]) != "26" {
		t.Fatal(value, err)
	}
	if value, err := rules.GetMinimumBandwidthRecord(ctx, "qp", "m1"); err != nil || string(value.Resource.Body["min_kbps"]) != "10" {
		t.Fatal(value, err)
	}
	if _, err := rules.CreateBandwidthLimitRule(ctx, "qp", qosrules.CreateBandwidthLimitRuleOpts{MaxKBps: 100, Direction: "egress"}); err != nil {
		t.Fatal(err)
	}
	if _, err := rules.CreateDSCPMarkingRule(ctx, "qp", qosrules.CreateDSCPMarkingRuleOpts{DSCPMark: 26}); err != nil {
		t.Fatal(err)
	}
	if _, err := rules.CreateMinimumBandwidthRule(ctx, "qp", qosrules.CreateMinimumBandwidthRuleOpts{MinKBps: 10}); err != nil {
		t.Fatal(err)
	}
	if err := rules.DeleteBandwidthLimitRule(ctx, "qp", "b1"); err != nil {
		t.Fatal(err)
	}
	if err := rules.DeleteDSCPMarkingRule(ctx, "qp", "gone"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal("rule delete 404 stays an error", err)
	}
	if value, err := ruletypes.New(client).GetRecord(ctx, "bandwidth_limit"); err != nil || value == nil {
		t.Fatal(value, err)
	}
	if _, err := qospolicies.New(client).CreateWithAttributes(ctx, qospolicies.WithAttribute("name", "silver"), qospolicies.WithAttribute("is_default", true)); err != nil {
		t.Fatal(err)
	}
	if err := qospolicies.New(client).DeleteWithOptions(ctx, "qp"); err != nil {
		t.Fatal(err)
	}
	pythonCloudAExpect(t, calls,
		"GET /extensions",
		"GET /qos/policies/gold", "GET /qos/policies?name=gold",
		"GET /qos/policies/qp/bandwidth_limit_rules/b1",
		"GET /qos/policies/qp/dscp_marking_rules/d1",
		"GET /qos/policies/qp/minimum_bandwidth_rules/m1",
		"POST /qos/policies/qp/bandwidth_limit_rules",
		"POST /qos/policies/qp/dscp_marking_rules",
		"POST /qos/policies/qp/minimum_bandwidth_rules",
		"DELETE /qos/policies/qp/bandwidth_limit_rules/b1",
		"DELETE /qos/policies/qp/dscp_marking_rules/gone",
		"GET /qos/rule-types/bandwidth_limit",
		"POST /qos/policies",
		"DELETE /qos/policies/qp")
	pythonCloudABody(t, calls[6].body, `{"bandwidth_limit_rule":{"max_kbps":100,"direction":"egress"}}`)
	pythonCloudABody(t, calls[7].body, `{"dscp_marking_rule":{"dscp_mark":26}}`)
	pythonCloudABody(t, calls[8].body, `{"minimum_bandwidth_rule":{"min_kbps":10}}`)
	pythonCloudABody(t, calls[12].body, `{"policy":{"name":"silver","is_default":true}}`)
}
