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

func TestCloudFirewallPolicyUpdateLookupOnlyAndDirtySelection(t *testing.T) {
	for _, tc := range []struct {
		name, body, attrs, want string
		noOp                    bool
	}{
		{"empty", `{"id":"p","name":"old"}`, `{}`, ``, true},
		{"same", `{"id":"p","name":"old","audited":true}`, `{"name":"old","audited":1}`, ``, true},
		{"changed-only", `{"id":"p","name":"old","audited":true}`, `{"name":"new","audited":true}`, `{"firewall_policy":{"name":"new"}}`, false},
		{"raw-string-bool", `{"id":"p","audited":"false"}`, `{"audited":true}`, `{"firewall_policy":{"audited":true}}`, false},

		{"null", `{"id":"p","name":"old"}`, `{"name":null}`, `{"firewall_policy":{"name":null}}`, false},
		{"missing-null", `{"id":"p"}`, `{"description":null}`, `{"firewall_policy":{"description":null}}`, false},
		{"explicit-false-empty", `{"id":"p","audited":true,"description":"old"}`, `{"audited":false,"description":""}`, `{"firewall_policy":{"audited":false,"description":""}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_policies/p": {{200, `{"firewall_policy":` + tc.body + `}`}}, "PUT /v2.0/fwaas/firewall_policies/p": {{200, `{"firewall_policy":{"id":"p","name":"response"}}`}}})
			result, err := network.New(client).UpdateFirewallPolicyRecord(context.Background(), "p", network.WithFirewallPolicyUpdateJSON(json.RawMessage(tc.attrs)))
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
				if result.Policy != nil {
					t.Fatal("fabricated PUT response")
				}
				return
			}
			var actual, expected any
			json.Unmarshal([]byte(got[1].body), &actual)
			json.Unmarshal([]byte(tc.want), &expected)
			if !reflect.DeepEqual(actual, expected) || result.Policy == nil || result.Policy.Name != "response" {
				t.Fatal(got, result)
			}
		})
	}
}
func TestCloudFirewallPolicyUpdateNameFallbackAndMissing(t *testing.T) {
	for _, found := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "name"}[found], func(t *testing.T) {
			rows := `[]`
			if found {
				rows = `[{"id":"actual","name":"target"}]`
			}
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_policies/target": {{404, `{}`}}, "GET /v2.0/fwaas/firewall_policies": {{200, `{"firewall_policies":` + rows + `}`}}, "PUT /v2.0/fwaas/firewall_policies/actual": {{200, `{"firewall_policy":{"id":"actual"}}`}}})
			result, err := network.New(client).UpdateFirewallPolicyRecord(context.Background(), "target", network.WithFirewallPolicyUpdateName("new"))
			if found {
				if err != nil || result.Policy == nil || len(calls()) != 3 {
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
func TestCloudFirewallPolicyUpdateOwnedOptionsAndPreflight(t *testing.T) {
	raw := json.RawMessage(`{"name":"owned"}`)
	option := network.WithFirewallPolicyUpdateJSON(raw)
	raw[2] = 'X'
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_policies/p": {{200, `{"firewall_policy":{"id":"p","name":"owned"}}`}}})
	service := network.New(client)
	result, err := service.UpdateFirewallPolicyRecord(context.Background(), "p", option)
	if err != nil || !result.NoOp || len(calls()) != 1 {
		t.Fatal(result, err, calls())
	}
	for _, opts := range [][]network.FirewallPolicyUpdateOption{{nil}, {network.WithFirewallPolicyUpdateAttribute("is_enabled", true)}, {network.WithFirewallPolicyUpdateAttribute("if_revision", 0)}, {network.WithFirewallPolicyUpdateAttribute("name", make(chan int))}, {network.WithFirewallPolicyUpdateJSON(json.RawMessage(`[]`))}, {network.WithFirewallPolicyUpdateJSON(json.RawMessage(`{"x_extension":1}`))}} {
		result, err = service.UpdateFirewallPolicyRecord(context.Background(), "p", opts...)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 1 {
			t.Fatal(result, err, calls())
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = service.UpdateFirewallPolicyRecord(ctx, "p")
	if result != nil || !errors.Is(err, context.Canceled) || len(calls()) != 1 {
		t.Fatal(result, err)
	}
	result, err = service.UpdateFirewallPolicyRecord(context.Background(), "p", func(*network.FirewallPolicyUpdateOpts) error { service.API = nil; return nil })
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 1 {
		t.Fatal(result, err)
	}
}
func TestCloudFirewallPolicyUpdateConflictKeepsLookupAndSingleAttempt(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_policies/p": {{200, `{"firewall_policy":{"id":"p","name":"old"}}`}}, "PUT /v2.0/fwaas/firewall_policies/p": {{http.StatusConflict, `{"error":"conflict"}`}}})
	result, err := network.New(client).UpdateFirewallPolicyRecord(context.Background(), "p", network.WithFirewallPolicyUpdateName("new"))
	if err == nil || result == nil || result.Lookup == nil || result.Policy != nil || result.NoOp || len(calls()) != 2 {
		t.Fatal(result, err, calls())
	}
}

func TestCloudFirewallPolicyUpdateFiltersStrictBindingAndFalseyDefaults(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `0`, `""`, `[]`, `{}`} {
		t.Run(raw, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_policies/p": {{200, `{"firewall_policy":{"id":"p"}}`}}})
			result, err := network.New(client).UpdateFirewallPolicyRecord(context.Background(), "p", network.WithFirewallPolicyUpdateFilters(json.RawMessage(raw)))
			if err != nil || result == nil || !result.NoOp || result.Lookup == nil || len(calls()) != 1 || len(calls()[0].query) != 0 {
				t.Fatal(result, err, calls())
			}
		})
	}
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_policies/target": {{404, `{}`}}, "GET /v2.0/fwaas/firewall_policies": {{200, `{"firewall_policies":[{"id":"actual","name":"target","description":"allow"}]}`}}})
	raw := json.RawMessage(`{"description":"allow"}`)
	option := network.WithFirewallPolicyUpdateFilters(raw)
	raw[2] = 'X'
	result, err := network.New(client).UpdateFirewallPolicyRecord(context.Background(), "target", option)
	if err != nil || result == nil || !result.NoOp || len(calls()) != 2 || calls()[0].query.Get("description") != "allow" || calls()[1].query.Get("description") != "allow" {
		t.Fatal(result, err, calls())
	}
	for _, raw := range []string{`"truthy-expression"`, `[1]`, `{"ignore_missing":null}`, `{"name_or_id":null}`, `{"headers":{}}`, `{"microversion":"2.1"}`} {
		before := len(calls())
		result, err = network.New(client).UpdateFirewallPolicyRecord(context.Background(), "target", network.WithFirewallPolicyUpdateFilters(json.RawMessage(raw)))
		if err == nil || len(calls()) != before {
			t.Fatal(raw, result, err, calls())
		}
	}
}

func TestCloudFirewallPolicyUpdateOrderedRulesTargetFirstAndEmptyClear(t *testing.T) {
	for _, tc := range []struct {
		name, old, refs string
		ids             []string
		put             bool
	}{
		{"ordered-duplicates", `[]`, `["second","first","second"]`, []string{"r2", "r1", "r2"}, true},
		{"equal-after-resolution", `["r1"]`, `["first"]`, []string{"r1"}, false},
		{"empty-clear", `["r1"]`, `[]`, []string{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
				"GET /v2.0/fwaas/firewall_policies/p":   {{200, `{"firewall_policy":{"id":"p","firewall_rules":` + tc.old + `}}`}},
				"GET /v2.0/fwaas/firewall_rules/first":  {{200, `{"firewall_rule":{"id":"r1"}}`}},
				"GET /v2.0/fwaas/firewall_rules/second": {{200, `{"firewall_rule":{"id":"r2"}}`}, {200, `{"firewall_rule":{"id":"r2"}}`}},
				"PUT /v2.0/fwaas/firewall_policies/p":   {{200, `{"firewall_policy":{"id":"p"}}`}},
			})
			result, err := network.New(client).UpdateFirewallPolicyRecord(context.Background(), "p", network.WithFirewallPolicyUpdateJSON(json.RawMessage(`{"firewall_rules":`+tc.refs+`}`)), network.WithFirewallPolicyUpdateFilters(json.RawMessage(`{"project_id":"owner"}`)))
			if err != nil || result == nil || result.NoOp == tc.put || len(result.RuleLookups) != len(tc.ids) {
				t.Fatal(result, err)
			}
			got := calls()
			want := 1 + len(tc.ids)
			if tc.put {
				want++
			}
			if len(got) != want {
				t.Fatal(got)
			}
			if got[0].path != "/v2.0/fwaas/firewall_policies/p" || got[0].query.Get("project_id") != "owner" {
				t.Fatal(got)
			}
			for i, id := range tc.ids {
				var actual string
				json.Unmarshal(result.RuleLookups[i].Resource.Body["id"], &actual)
				if actual != id || len(got[i+1].query) != 0 {
					t.Fatal(i, id, result, got)
				}
			}
			if tc.put {
				var body struct {
					Policy struct {
						Rules []string `json:"firewall_rules"`
					} `json:"firewall_policy"`
				}
				json.Unmarshal([]byte(got[len(got)-1].body), &body)
				if !reflect.DeepEqual(body.Policy.Rules, tc.ids) {
					t.Fatal(got)
				}
			}
		})
	}
}
func TestCloudFirewallPolicyUpdateRuleFailureKeepsPriorEvidence(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
		"GET /v2.0/fwaas/firewall_policies/p":    {{200, `{"firewall_policy":{"id":"p"}}`}},
		"GET /v2.0/fwaas/firewall_rules/first":   {{200, `{"firewall_rule":{"id":"r1"}}`}},
		"GET /v2.0/fwaas/firewall_rules/missing": {{404, `{}`}},
		"GET /v2.0/fwaas/firewall_rules":         {{200, `{"firewall_rules":[]}`}},
	})
	result, err := network.New(client).UpdateFirewallPolicyRecord(context.Background(), "p", network.WithFirewallPolicyUpdateRules("first", "missing", "never"))
	if !errors.Is(err, resource.ErrNotFound) || result == nil || result.Lookup == nil || len(result.RuleLookups) < 1 || result.Policy != nil || result.NoOp || len(calls()) != 4 {
		t.Fatal(result, err, calls())
	}
}
func TestCloudFirewallPolicyUpdateUnsupportedRuleShapesAfterTargetLookup(t *testing.T) {
	for _, raw := range []string{`null`, `"first"`, `{}`, `[null]`, `[1]`} {
		t.Run(raw, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_policies/p": {{200, `{"firewall_policy":{"id":"p"}}`}}})
			result, err := network.New(client).UpdateFirewallPolicyRecord(context.Background(), "p", network.WithFirewallPolicyUpdateJSON(json.RawMessage(`{"firewall_rules":`+raw+`}`)))
			if err == nil || result == nil || result.Lookup == nil || result.Policy != nil || len(calls()) != 1 {
				t.Fatal(result, err, calls())
			}
		})
	}
}
