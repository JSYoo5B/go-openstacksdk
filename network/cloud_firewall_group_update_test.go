package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestCloudFirewallGroupUpdateLookupOnlyAndDirtySelection(t *testing.T) {
	for _, tc := range []struct {
		name, body, attrs, want string
		noOp                    bool
	}{
		{"empty", `{"id":"p","name":"old"}`, `{}`, ``, true},
		{"same", `{"id":"p","name":"old","admin_state_up":true}`, `{"name":"old","admin_state_up":1}`, ``, true},
		{"changed-only", `{"id":"p","name":"old","admin_state_up":true}`, `{"name":"new","admin_state_up":true}`, `{"firewall_group":{"name":"new"}}`, false},
		{"raw-string-bool", `{"id":"p","admin_state_up":"false"}`, `{"admin_state_up":true}`, `{"firewall_group":{"admin_state_up":true}}`, false},

		{"null", `{"id":"p","name":"old"}`, `{"name":null}`, `{"firewall_group":{"name":null}}`, false},
		{"missing-null", `{"id":"p"}`, `{"description":null}`, `{"firewall_group":{"description":null}}`, false},
		{"explicit-false-empty", `{"id":"p","admin_state_up":true,"description":"old"}`, `{"admin_state_up":false,"description":""}`, `{"firewall_group":{"admin_state_up":false,"description":""}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_groups/p": {{200, `{"firewall_group":` + tc.body + `}`}}, "PUT /v2.0/fwaas/firewall_groups/p": {{200, `{"firewall_group":{"id":"p","name":"response"}}`}}})
			result, err := network.New(client).UpdateFirewallGroupRecord(context.Background(), "p", network.WithFirewallGroupUpdateJSON(json.RawMessage(tc.attrs)))
			if err != nil || result == nil || result.Lookup == nil || result.NoOp != tc.noOp {
				t.Fatal(result, err)
			}
			got := calls()
			wantCount := 2
			if tc.noOp {
				wantCount = 1
			}
			if len(got) != wantCount {
				t.Fatal(got)
			}
			if tc.noOp {
				if result.Group != nil {
					t.Fatal("fabricated PUT response")
				}
				return
			}
			var actual, expected any
			json.Unmarshal([]byte(got[1].body), &actual)
			json.Unmarshal([]byte(tc.want), &expected)
			if !reflect.DeepEqual(actual, expected) || result.Group == nil || result.Group.Name != "response" {
				t.Fatal(got, result)
			}
		})
	}
}
func TestCloudFirewallGroupUpdateNameFallbackAndMissing(t *testing.T) {
	for _, found := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "name"}[found], func(t *testing.T) {
			rows := `[]`
			if found {
				rows = `[{"id":"actual","name":"target"}]`
			}
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_groups/target": {{404, `{}`}}, "GET /v2.0/fwaas/firewall_groups": {{200, `{"firewall_groups":` + rows + `}`}}, "PUT /v2.0/fwaas/firewall_groups/actual": {{200, `{"firewall_group":{"id":"actual"}}`}}})
			result, err := network.New(client).UpdateFirewallGroupRecord(context.Background(), "target", network.WithFirewallGroupUpdateName("new"))
			if found {
				if err != nil || result.Group == nil || len(calls()) != 3 {
					t.Fatal(result, err, calls())
				}
			} else {
				if !errors.Is(err, resource.ErrNotFound) || result.Lookup != nil || len(calls()) != 2 {
					t.Fatal(result, err, calls())
				}
			}
		})
	}
}
func TestCloudFirewallGroupUpdateOwnedOptionsAndPreflight(t *testing.T) {
	raw := json.RawMessage(`{"name":"owned"}`)
	option := network.WithFirewallGroupUpdateJSON(raw)
	raw[2] = 'X'
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_groups/p": {{200, `{"firewall_group":{"id":"p","name":"owned"}}`}}})
	service := network.New(client)
	result, err := service.UpdateFirewallGroupRecord(context.Background(), "p", option)
	if err != nil || !result.NoOp || len(calls()) != 1 {
		t.Fatal(result, err, calls())
	}
	for _, opts := range [][]network.FirewallGroupUpdateOption{{nil}, {network.WithFirewallGroupUpdateAttribute("egress_firewall_policy_id", true)}, {network.WithFirewallGroupUpdateAttribute("ingress_firewall_policy_id", nil)}, {network.WithFirewallGroupUpdateAttribute("if_revision", 0)}, {network.WithFirewallGroupUpdateAttribute("name", make(chan int))}, {network.WithFirewallGroupUpdateJSON(json.RawMessage(`[]`))}, {network.WithFirewallGroupUpdateJSON(json.RawMessage(`{"x_extension":1}`))}} {
		result, err = service.UpdateFirewallGroupRecord(context.Background(), "p", opts...)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 1 {
			t.Fatal(result, err, calls())
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = service.UpdateFirewallGroupRecord(ctx, "p")
	if result != nil || !errors.Is(err, context.Canceled) || len(calls()) != 1 {
		t.Fatal(result, err)
	}
	result, err = service.UpdateFirewallGroupRecord(context.Background(), "p", func(*network.FirewallGroupUpdateOpts) error { service.API = nil; return nil })
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 1 {
		t.Fatal(result, err)
	}
}
func TestCloudFirewallGroupUpdateConflictKeepsLookupAndSingleAttempt(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_groups/p": {{200, `{"firewall_group":{"id":"p","name":"old"}}`}}, "PUT /v2.0/fwaas/firewall_groups/p": {{http.StatusConflict, `{"error":"conflict"}`}}})
	result, err := network.New(client).UpdateFirewallGroupRecord(context.Background(), "p", network.WithFirewallGroupUpdateName("new"))
	if err == nil || result == nil || result.Lookup == nil || result.Group != nil || result.NoOp || len(calls()) != 2 {
		t.Fatal(result, err, calls())
	}
}

func TestCloudFirewallGroupUpdateFiltersStrictBindingAndFalseyDefaults(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `0`, `""`, `[]`, `{}`} {
		t.Run(raw, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_groups/p": {{200, `{"firewall_group":{"id":"p"}}`}}})
			result, err := network.New(client).UpdateFirewallGroupRecord(context.Background(), "p", network.WithFirewallGroupUpdateFilters(json.RawMessage(raw)))
			if err != nil || result == nil || !result.NoOp || result.Lookup == nil || len(calls()) != 1 || len(calls()[0].query) != 0 {
				t.Fatal(result, err, calls())
			}
		})
	}
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_groups/target": {{404, `{}`}}, "GET /v2.0/fwaas/firewall_groups": {{200, `{"firewall_groups":[{"id":"actual","name":"target","description":"allow"}]}`}}})
	raw := json.RawMessage(`{"description":"allow"}`)
	option := network.WithFirewallGroupUpdateFilters(raw)
	raw[2] = 'X'
	result, err := network.New(client).UpdateFirewallGroupRecord(context.Background(), "target", option)
	if err != nil || result == nil || !result.NoOp || len(calls()) != 2 || calls()[0].query.Get("description") != "allow" || calls()[1].query.Get("description") != "allow" {
		t.Fatal(result, err, calls())
	}
	for _, raw := range []string{`"truthy-expression"`, `[1]`, `{"ignore_missing":null}`, `{"name_or_id":null}`} {
		before := len(calls())
		result, err = network.New(client).UpdateFirewallGroupRecord(context.Background(), "target", network.WithFirewallGroupUpdateFilters(json.RawMessage(raw)))
		if err == nil || len(calls()) != before {
			t.Fatal(raw, result, err, calls())
		}
	}
}

func TestCloudFirewallGroupUpdateTargetPolicyPortsOrderAndEmptyClear(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
		"GET /v2.0/fwaas/firewall_groups/g":     {{200, `{"firewall_group":{"id":"g","ports":[],"egress_firewall_policy_id":"old","ingress_firewall_policy_id":"old"}}`}},
		"GET /v2.0/fwaas/firewall_policies/out": {{200, `{"firewall_policy":{"id":"egress"}}`}},
		"GET /v2.0/fwaas/firewall_policies/in":  {{200, `{"firewall_policy":{"id":"ingress"}}`}},
		"GET /v2.0/ports/b":                     {{200, `{"port":{"id":"pb"}}`}, {200, `{"port":{"id":"pb"}}`}},
		"GET /v2.0/ports/a":                     {{200, `{"port":{"id":"pa"}}`}},
		"PUT /v2.0/fwaas/firewall_groups/g":     {{200, `{"firewall_group":{"id":"g"}}`}},
	})
	result, err := network.New(client).UpdateFirewallGroupRecord(context.Background(), "g", network.WithFirewallGroupUpdateJSON(json.RawMessage(`{"ingress_firewall_policy":"in","egress_firewall_policy":"out","ports":["b","a","b"]}`)), network.WithFirewallGroupUpdateFilters(json.RawMessage(`{"project_id":"owner"}`)))
	if err != nil || result == nil || result.Group == nil || len(result.PolicyLookups) != 2 || len(result.PortLookups) != 3 || len(calls()) != 7 {
		t.Fatal(result, err, calls())
	}
	got := calls()
	want := []string{"/v2.0/fwaas/firewall_groups/g", "/v2.0/fwaas/firewall_policies/out", "/v2.0/fwaas/firewall_policies/in", "/v2.0/ports/b", "/v2.0/ports/a", "/v2.0/ports/b", "/v2.0/fwaas/firewall_groups/g"}
	for i, path := range want {
		if got[i].path != path {
			t.Fatal(got)
		}
		if i > 0 && len(got[i].query) != 0 {
			t.Fatal(got)
		}
	}
	var body map[string]map[string]json.RawMessage
	json.Unmarshal([]byte(got[6].body), &body)
	if string(body["firewall_group"]["egress_firewall_policy_id"]) != `"egress"` || string(body["firewall_group"]["ingress_firewall_policy_id"]) != `"ingress"` || string(body["firewall_group"]["ports"]) != `["pb","pa","pb"]` {
		t.Fatal(got[6])
	}
}
func TestCloudFirewallGroupUpdateNullPoliciesAndEmptyPorts(t *testing.T) {
	for _, same := range []bool{false, true} {
		t.Run(map[bool]string{false: "clear", true: "equal"}[same], func(t *testing.T) {
			old := `{"id":"g","ports":["p"],"egress_firewall_policy_id":"out","ingress_firewall_policy_id":"in"}`
			if same {
				old = `{"id":"g","ports":[],"egress_firewall_policy_id":null,"ingress_firewall_policy_id":null}`
			}
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_groups/g": {{200, `{"firewall_group":` + old + `}`}}, "PUT /v2.0/fwaas/firewall_groups/g": {{200, `{"firewall_group":{"id":"g"}}`}}})
			result, err := network.New(client).UpdateFirewallGroupRecord(context.Background(), "g", network.WithFirewallGroupUpdateJSON(json.RawMessage(`{"egress_firewall_policy":null,"ingress_firewall_policy":null,"ports":[]}`)))
			if err != nil || result == nil || result.NoOp != same || len(result.PolicyLookups) != 0 || len(result.PortLookups) != 0 {
				t.Fatal(result, err)
			}
			want := 2
			if same {
				want = 1
			}
			if len(calls()) != want {
				t.Fatal(calls())
			}
		})
	}
}
func TestCloudFirewallGroupUpdateReferenceFailuresPreservePhaseOrder(t *testing.T) {
	for _, tc := range []struct {
		name, attrs   string
		policyQueries int
		portQueries   int
	}{
		{"ingress-missing", `{"egress_firewall_policy":"out","ingress_firewall_policy":"missing","ports":["never"]}`, 3, 0},
		{"ports-null-after-policies", `{"egress_firewall_policy":"out","ingress_firewall_policy":null,"ports":null}`, 1, 0},
		{"port-missing-after-policies", `{"egress_firewall_policy":"out","ports":["present","missing","never"]}`, 1, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
				"GET /v2.0/fwaas/firewall_groups/g":         {{200, `{"firewall_group":{"id":"g"}}`}},
				"GET /v2.0/fwaas/firewall_policies/out":     {{200, `{"firewall_policy":{"id":"out-id"}}`}},
				"GET /v2.0/fwaas/firewall_policies/missing": {{404, `{}`}},
				"GET /v2.0/fwaas/firewall_policies":         {{200, `{"firewall_policies":[]}`}},
				"GET /v2.0/ports/present":                   {{200, `{"port":{"id":"p"}}`}},
				"GET /v2.0/ports/missing":                   {{404, `{}`}},
				"GET /v2.0/ports":                           {{200, `{"ports":[]}`}},
			})
			result, err := network.New(client).UpdateFirewallGroupRecord(context.Background(), "g", network.WithFirewallGroupUpdateJSON(json.RawMessage(tc.attrs)))
			if err == nil || result == nil || result.Lookup == nil || len(result.PolicyLookups) < 1 || result.Group != nil || result.NoOp || len(calls()) != 1+tc.policyQueries+tc.portQueries {
				t.Fatal(result, err, calls())
			}
			if tc.portQueries > 0 && len(result.PortLookups) != 1 {
				t.Fatal(result)
			}
		})
	}
}

func TestCloudFirewallGroupUpdateResolvedEqualReferencesStillLookUp(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
		"GET /v2.0/fwaas/firewall_groups/g":     {{200, `{"firewall_group":{"id":"g","ports":["p"],"egress_firewall_policy_id":"out-id"}}`}},
		"GET /v2.0/fwaas/firewall_policies/out": {{200, `{"firewall_policy":{"id":"out-id"}}`}},
		"GET /v2.0/ports/member":                {{200, `{"port":{"id":"p"}}`}},
	})
	result, err := network.New(client).UpdateFirewallGroupRecord(context.Background(), "g", network.WithFirewallGroupUpdateEgressPolicy("out"), network.WithFirewallGroupUpdatePorts("member"))
	if err != nil || result == nil || !result.NoOp || result.Group != nil || len(result.PolicyLookups) != 1 || len(result.PortLookups) != 1 || len(calls()) != 3 {
		t.Fatal(result, err, calls())
	}
}
