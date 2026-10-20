package network_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	secgroups "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
	secrules "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonCloudBCall struct {
	method, path string
	query        url.Values
	body         string
}

type pythonCloudBReply struct {
	status int
	body   string
}

// pythonCloudBClient serves scripted Neutron replies keyed by "METHOD path" in
// order and records every request. "{{base}}" in a body becomes the server URL.
func pythonCloudBClient(t *testing.T, replies map[string][]pythonCloudBReply) (*gophercloud.ServiceClient, func() []pythonCloudBCall) {
	t.Helper()
	cloud := testcloud.New(t)
	var mu sync.Mutex
	var calls []pythonCloudBCall
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		key := r.Method + " " + r.URL.Path
		mu.Lock()
		calls = append(calls, pythonCloudBCall{r.Method, r.URL.Path, r.URL.Query(), string(raw)})
		queue := replies[key]
		if len(queue) == 0 {
			mu.Unlock()
			t.Errorf("unexpected request %s?%s", key, r.URL.RawQuery)
			testcloud.JSON(w, 500, `{}`)
			return
		}
		reply := queue[0]
		replies[key] = queue[1:]
		mu.Unlock()
		if reply.body == "" {
			w.WriteHeader(reply.status)
			return
		}
		testcloud.JSON(w, reply.status, strings.ReplaceAll(reply.body, "{{base}}", cloud.Server.URL))
	})
	client := cloud.Client("network", "/")
	client.ResourceBase = cloud.Server.URL + "/v2.0/"
	return client, func() []pythonCloudBCall {
		mu.Lock()
		defer mu.Unlock()
		return append([]pythonCloudBCall(nil), calls...)
	}
}

func pythonCloudBRequests(t *testing.T, got []pythonCloudBCall, want ...string) {
	t.Helper()
	var lines []string
	for _, call := range got {
		line := call.method + " " + call.path
		if len(call.query) > 0 {
			line += "?" + call.query.Encode()
		}
		lines = append(lines, line)
	}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("requests\n got %q\nwant %q", lines, want)
	}
}

func pythonCloudBBody(t *testing.T, raw string, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal([]byte(raw), &got); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("body %s, want %#v (%v)", raw, want, err)
	}
}

const pythonCloudBSubnet = `{"id":"sub-1","name":"private-subnet","network_id":"net-1","cidr":"10.0.0.0/24","ip_version":4,"gateway_ip":"10.0.0.1","enable_dhcp":true,"project_id":"proj"}`

// get_subnet_by_id delegates to Proxy.get_subnet: one member GET whose 404 raises.
func TestPythonCloudNetworkBGetSubnetByID(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
		"GET /v2.0/subnets/sub-1":   {{200, `{"subnet":` + pythonCloudBSubnet + `}`}},
		"GET /v2.0/subnets/missing": {{404, `{"NeutronError":{"type":"SubnetNotFound"}}`}},
	})
	api := subnets.New(client)
	value, err := api.GetWithFields(context.Background(), "sub-1")
	if err != nil || value.ID != "sub-1" || value.Name != "private-subnet" || value.NetworkID != "net-1" || value.CIDR != "10.0.0.0/24" || value.IPVersion != 4 || value.GatewayIP != "10.0.0.1" || !value.EnableDHCP || value.ProjectID != "proj" {
		t.Fatal(value, err)
	}
	if _, err := api.GetWithFields(context.Background(), "missing"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal("404 must remain an error like Python NotFoundException", err)
	}
	pythonCloudBRequests(t, calls(), "GET /v2.0/subnets/sub-1", "GET /v2.0/subnets/missing")
}

const pythonCloudBPool = `{"id":"pool-1","name":"shared-pool","prefixes":["10.0.0.0/8"],"min_prefixlen":8,"max_prefixlen":32,"default_prefixlen":24,"ip_version":4}`

// get_subnetpool is find_subnet_pool(name_or_id, ignore_missing=True): member
// GET first, then a name-filtered list, with absence returned as no value.
func TestPythonCloudNetworkBGetSubnetpool(t *testing.T) {
	for _, testcase := range []struct {
		name, identity string
		replies        map[string][]pythonCloudBReply
		wantID         string
		want           []string
	}{
		{"id", "pool-1", map[string][]pythonCloudBReply{"GET /v2.0/subnetpools/pool-1": {{200, `{"subnetpool":` + pythonCloudBPool + `}`}}}, "pool-1", []string{"GET /v2.0/subnetpools/pool-1"}},
		{"name fallback", "shared-pool", map[string][]pythonCloudBReply{
			"GET /v2.0/subnetpools/shared-pool": {{404, `{}`}},
			"GET /v2.0/subnetpools":             {{200, `{"subnetpools":[` + pythonCloudBPool + `]}`}},
		}, "pool-1", []string{"GET /v2.0/subnetpools/shared-pool", "GET /v2.0/subnetpools?name=shared-pool"}},
		{"missing returns nil", "absent", map[string][]pythonCloudBReply{
			"GET /v2.0/subnetpools/absent": {{404, `{}`}},
			"GET /v2.0/subnetpools":        {{200, `{"subnetpools":[]}`}},
		}, "", []string{"GET /v2.0/subnetpools/absent", "GET /v2.0/subnetpools?name=absent"}},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, testcase.replies)
			value, err := subnetpools.New(client).FindIdentity(context.Background(), testcase.identity)
			if err != nil {
				t.Fatal(err)
			}
			if testcase.wantID == "" && value != nil || testcase.wantID != "" && (value == nil || value.ID != testcase.wantID || value.Name != "shared-pool" || value.DefaultPrefixLen != 24) {
				t.Fatal(value)
			}
			pythonCloudBRequests(t, calls(), testcase.want...)
		})
	}
}

func pythonCloudBRouterRow(id string) string {
	return `{"id":"` + id + `","name":"edge","status":"ACTIVE","admin_state_up":true,"project_id":"proj","external_gateway_info":{"network_id":"ext","enable_snat":true}}`
}

// list_routers and search_routers send filters (and name_or_id as the name
// query) to Neutron and consume every advertised page.
func TestPythonCloudNetworkBListAndSearchRouters(t *testing.T) {
	for _, testcase := range []struct {
		name    string
		options []resource.ListOption
		first   url.Values
	}{
		{"list_routers() and search_routers()", nil, url.Values{}},
		{"list_routers(filters)", []resource.ListOption{resource.WithFilters(map[string]any{"project_id": "proj", "name": "edge"})}, url.Values{"name": {"edge"}, "project_id": {"proj"}}},
		{"search_routers(name_or_id, filters)", []resource.ListOption{resource.WithFilter("name", "edge"), resource.WithFilter("project_id", "proj")}, url.Values{"name": {"edge"}, "project_id": {"proj"}}},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			next := url.Values{"marker": {"r-1"}}
			for key, values := range testcase.first {
				next[key] = values
			}
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/routers": {
				{200, `{"routers":[` + pythonCloudBRouterRow("r-1") + `],"routers_links":[{"rel":"next","href":"{{base}}/v2.0/routers?` + next.Encode() + `"}]}`},
				{200, `{"routers":[` + pythonCloudBRouterRow("r-2") + `]}`},
			}})
			values, err := routers.New(client).All(context.Background(), testcase.options...)
			if err != nil || len(values) != 2 || values[0].ID != "r-1" || values[1].ID != "r-2" || values[0].Name != "edge" || !values[0].AdminStateUp || values[0].GatewayInfo.NetworkID != "ext" || values[0].ProjectID != "proj" {
				t.Fatal(values, err)
			}
			got := calls()
			if len(got) != 2 || !reflect.DeepEqual(got[0].query, testcase.first) || !reflect.DeepEqual(got[1].query, next) {
				t.Fatal(got)
			}
		})
	}
}

const pythonCloudBGroup = `{"id":"sg-1","name":"web","description":"web tier","stateful":true,"project_id":"proj","tenant_id":"proj","security_group_rules":[{"id":"rule-1","direction":"ingress","ethertype":"IPv4","security_group_id":"sg-1"}]}`

// create_security_group (Neutron source) posts name and description, adding
// stateful and tenant_id only when supplied.
func TestPythonCloudNetworkBCreateSecurityGroup(t *testing.T) {
	for _, testcase := range []struct {
		name    string
		options []secgroups.AttributeOption
		want    map[string]any
	}{
		{"defaults", []secgroups.AttributeOption{secgroups.WithAttribute("name", "web"), secgroups.WithAttribute("description", "web tier")}, map[string]any{"name": "web", "description": "web tier"}},
		{"project and stateful", []secgroups.AttributeOption{secgroups.WithAttribute("name", "web"), secgroups.WithAttribute("description", "web tier"), secgroups.WithAttribute("stateful", false), secgroups.WithAttribute("tenant_id", "proj")}, map[string]any{"name": "web", "description": "web tier", "stateful": false, "tenant_id": "proj"}},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"POST /v2.0/security-groups": {{201, `{"security_group":` + pythonCloudBGroup + `}`}}})
			value, err := secgroups.New(client).CreateWithAttributes(context.Background(), testcase.options...)
			if err != nil || value.ID != "sg-1" || value.Name != "web" || value.Description != "web tier" || !value.Stateful || len(value.Rules) != 1 || value.Rules[0].ID != "rule-1" {
				t.Fatal(value, err)
			}
			got := calls()
			pythonCloudBRequests(t, got, "POST /v2.0/security-groups")
			pythonCloudBBody(t, got[0].body, map[string]any{"security_group": testcase.want})
		})
	}
}

// get_security_group_by_id and list_security_groups (Neutron source) map to the
// member GET and the filtered, fully paged collection list.
func TestPythonCloudNetworkBSecurityGroupReads(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
		"GET /v2.0/security-groups/sg-1": {{200, `{"security_group":` + pythonCloudBGroup + `}`}, {404, `{}`}},
		"GET /v2.0/security-groups": {
			{200, `{"security_groups":[` + pythonCloudBGroup + `],"security_groups_links":[{"rel":"next","href":"{{base}}/v2.0/security-groups?marker=sg-1"}]}`},
			{200, `{"security_groups":[]}`},
			{200, `{"security_groups":[` + pythonCloudBGroup + `]}`},
		},
	})
	api := secgroups.New(client)
	value, err := api.GetWithFields(context.Background(), "sg-1")
	if err != nil || value.ID != "sg-1" || value.ProjectID != "proj" || len(value.Rules) != 1 {
		t.Fatal(value, err)
	}
	if _, err := api.GetWithFields(context.Background(), "sg-1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	all, err := api.All(context.Background())
	if err != nil || len(all) != 1 || all[0].Name != "web" {
		t.Fatal(all, err)
	}
	filtered, err := api.All(context.Background(), resource.WithFilters(map[string]any{"name": "web", "project_id": "proj"}))
	if err != nil || len(filtered) != 1 {
		t.Fatal(filtered, err)
	}
	pythonCloudBRequests(t, calls(),
		"GET /v2.0/security-groups/sg-1", "GET /v2.0/security-groups/sg-1",
		"GET /v2.0/security-groups", "GET /v2.0/security-groups?marker=sg-1",
		"GET /v2.0/security-groups?name=web&project_id=proj")
}

// delete_security_group_rule (Neutron source) uses ignore_missing=False: 204 is
// success while 404 and 409 stay errors.
func TestPythonCloudNetworkBDeleteSecurityGroupRule(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"DELETE /v2.0/security-group-rules/rule-1": {{204, ""}, {404, `{}`}, {409, `{}`}}})
	api := secrules.New(client)
	strict := secrules.WithDeleteIgnoreMissing(false)
	if err := api.DeleteWithOptions(context.Background(), "rule-1", strict); err != nil {
		t.Fatal(err)
	}
	if err := api.DeleteWithOptions(context.Background(), "rule-1", strict); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	if err := api.DeleteWithOptions(context.Background(), "rule-1", strict); !gophercloud.ResponseCodeIs(err, http.StatusConflict) {
		t.Fatal(err)
	}
	pythonCloudBRequests(t, calls(), "DELETE /v2.0/security-group-rules/rule-1", "DELETE /v2.0/security-group-rules/rule-1", "DELETE /v2.0/security-group-rules/rule-1")
	for _, call := range calls() {
		if call.body != "" {
			t.Fatal(call)
		}
	}
}

// Evidence for unresolved update_port/update_subnet: the Go lookup plus update
// chain sends Python's requests once the caller supplies attributes.
func TestPythonCloudNetworkBUpdatePortAndSubnetChains(t *testing.T) {
	t.Run("update_port by name", func(t *testing.T) {
		client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
			"GET /v2.0/ports/web-port": {{404, `{}`}},
			"GET /v2.0/ports":          {{200, `{"ports":[{"id":"port-1","name":"web-port","network_id":"net-1"}]}`}},
			"PUT /v2.0/ports/port-1":   {{200, `{"port":{"id":"port-1","name":"renamed","network_id":"net-1","admin_state_up":false}}`}},
		})
		api := ports.New(client)
		found, err := api.FindRecord(context.Background(), "web-port", ports.WithRecordFindIgnoreMissing(false))
		if err != nil || string(found.Resource.Body["id"]) != `"port-1"` {
			t.Fatal(found, err)
		}
		updated, err := api.UpdateWithAttributes(context.Background(), "port-1", ports.WithAttribute("name", "renamed"), ports.WithAttribute("admin_state_up", false), ports.WithAttribute("binding:host_id", "host-a"))
		if err != nil || updated.Name != "renamed" || updated.AdminStateUp {
			t.Fatal(updated, err)
		}
		got := calls()
		pythonCloudBRequests(t, got, "GET /v2.0/ports/web-port", "GET /v2.0/ports?name=web-port", "PUT /v2.0/ports/port-1")
		pythonCloudBBody(t, got[2].body, map[string]any{"port": map[string]any{"name": "renamed", "admin_state_up": false, "binding:host_id": "host-a"}})
	})
	t.Run("update_port missing raises before PUT", func(t *testing.T) {
		client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
			"GET /v2.0/ports/absent": {{404, `{}`}},
			"GET /v2.0/ports":        {{200, `{"ports":[]}`}},
		})
		if value, err := ports.New(client).FindRecord(context.Background(), "absent", ports.WithRecordFindIgnoreMissing(false)); err == nil {
			t.Fatal(value)
		}
		pythonCloudBRequests(t, calls(), "GET /v2.0/ports/absent", "GET /v2.0/ports?name=absent")
	})
	t.Run("update_subnet disable_gateway_ip", func(t *testing.T) {
		client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
			"GET /v2.0/subnets/sub-1": {{200, `{"subnet":` + pythonCloudBSubnet + `}`}},
			"PUT /v2.0/subnets/sub-1": {{200, `{"subnet":{"id":"sub-1","name":"renamed","network_id":"net-1","cidr":"10.0.0.0/24","ip_version":4,"gateway_ip":null,"enable_dhcp":false}}`}},
		})
		api := subnets.New(client)
		found, err := api.FindRecord(context.Background(), "sub-1", subnets.WithRecordFindIgnoreMissing(false))
		if err != nil || string(found.Resource.Body["id"]) != `"sub-1"` {
			t.Fatal(found, err)
		}
		updated, err := api.UpdateWithAttributes(context.Background(), "sub-1", subnets.WithAttribute("name", "renamed"), subnets.WithAttribute("enable_dhcp", false), subnets.WithAttribute("gateway_ip", nil))
		if err != nil || updated.Name != "renamed" || updated.GatewayIP != "" || updated.EnableDHCP {
			t.Fatal(updated, err)
		}
		got := calls()
		pythonCloudBRequests(t, got, "GET /v2.0/subnets/sub-1", "PUT /v2.0/subnets/sub-1")
		pythonCloudBBody(t, got[1].body, map[string]any{"subnet": map[string]any{"name": "renamed", "enable_dhcp": false, "gateway_ip": nil}})
	})
}

// Evidence for unresolved remove_router_interface: the port reference wins and a
// subnet-only call sends subnet_id, but Go has no local both-missing rejection.
func TestPythonCloudNetworkBRemoveRouterInterfaceBodies(t *testing.T) {
	for _, testcase := range []struct {
		name    string
		options []routers.RouterInterfaceOption
		want    map[string]any
	}{
		{"subnet_id", []routers.RouterInterfaceOption{routers.WithRouterInterfaceSubnet("sub-1")}, map[string]any{"subnet_id": "sub-1"}},
		{"port wins", []routers.RouterInterfaceOption{routers.WithRouterInterfaceSubnet("sub-1"), routers.WithRouterInterfacePort("port-1")}, map[string]any{"port_id": "port-1"}},
		{"neither still sends", nil, map[string]any{"subnet_id": nil}},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"PUT /v2.0/routers/r-1/remove_router_interface": {{200, `{"id":"r-1","subnet_id":"sub-1","port_id":"port-1"}`}}})
			if _, err := routers.New(client).RemoveInterfaceRecord(context.Background(), "r-1", testcase.options...); err != nil {
				t.Fatal(err)
			}
			got := calls()
			pythonCloudBRequests(t, got, "PUT /v2.0/routers/r-1/remove_router_interface")
			pythonCloudBBody(t, got[0].body, testcase.want)
		})
	}
}

// Evidence for unresolved list_firewall_*: default arguments list every page of
// the FWaaS v2 collections and decode every Python-declared field.
func TestPythonCloudNetworkBFirewallDefaultLists(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
		"GET /v2.0/fwaas/firewall_groups": {
			{200, `{"firewall_groups":[{"id":"g-1","name":"edge","admin_state_up":true,"ingress_firewall_policy_id":"p-1","egress_firewall_policy_id":"p-2","ports":["port-1"],"status":"ACTIVE","shared":false,"project_id":"proj"}],"firewall_groups_links":[{"rel":"next","href":"{{base}}/v2.0/fwaas/firewall_groups?marker=g-1"}]}`},
			{200, `{"firewall_groups":[]}`},
		},
		"GET /v2.0/fwaas/firewall_policies": {{200, `{"firewall_policies":[{"id":"p-1","name":"in","audited":true,"shared":false,"firewall_rules":["fr-1","fr-2"],"project_id":"proj"}]}`}},
		"GET /v2.0/fwaas/firewall_rules":    {{200, `{"firewall_rules":[{"id":"fr-1","name":"ssh","action":"allow","protocol":"tcp","ip_version":4,"destination_port":"22","enabled":true,"shared":false,"firewall_policy_id":["p-1"],"project_id":"proj"}]}`}},
	})
	allGroups, err := groups.New(client).All(context.Background())
	if err != nil || len(allGroups) != 1 || allGroups[0].IngressFirewallPolicyID != "p-1" || allGroups[0].EgressFirewallPolicyID != "p-2" || !allGroups[0].AdminStateUp || allGroups[0].Ports[0] != "port-1" {
		t.Fatal(allGroups, err)
	}
	allPolicies, err := policies.New(client).All(context.Background())
	if err != nil || len(allPolicies) != 1 || !allPolicies[0].Audited || !reflect.DeepEqual(allPolicies[0].Rules, []string{"fr-1", "fr-2"}) {
		t.Fatal(allPolicies, err)
	}
	allRules, err := rules.New(client).All(context.Background())
	if err != nil || len(allRules) != 1 || allRules[0].Action != "allow" || allRules[0].DestinationPort != "22" || !reflect.DeepEqual(allRules[0].FirewallPolicyID, []string{"p-1"}) {
		t.Fatal(allRules, err)
	}
	pythonCloudBRequests(t, calls(), "GET /v2.0/fwaas/firewall_groups", "GET /v2.0/fwaas/firewall_groups?marker=g-1", "GET /v2.0/fwaas/firewall_policies", "GET /v2.0/fwaas/firewall_rules")
}
