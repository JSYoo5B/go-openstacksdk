package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestCloudFirewallPolicyRemoveRuleMembershipAndAction(t *testing.T) {
	for _, tc := range []struct {
		name, rules       string
		action, wantError bool
	}{
		{"member", `["r"]`, true, false}, {"absent", `["other"]`, false, false}, {"empty", `[]`, false, false},
		{"mixed", `[true,1,null,{"id":"r"},"r"]`, true, false}, {"scalar-list-descriptor", `"r"`, true, false},
		{"number-list-descriptor", `1`, false, false}, {"null", `null`, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
				"GET /v2.0/fwaas/firewall_policies/p":             {{200, `{"firewall_policy":{"id":"p","firewall_rules":` + tc.rules + `}}`}},
				"GET /v2.0/fwaas/firewall_rules/r":                {{200, `{"firewall_rule":{"id":"r"}}`}},
				"PUT /v2.0/fwaas/firewall_policies/p/remove_rule": {{200, `{"id":"p","name":"changed"}`}},
			})
			result, err := network.New(client).RemoveRuleFromPolicyRecord(context.Background(), "p", "r")
			if (err != nil) != tc.wantError || result == nil || result.Lookup == nil || result.RuleLookup == nil {
				t.Fatal(result, err)
			}
			count := 2
			if tc.action {
				count++
			}
			got := calls()
			if len(got) != count {
				t.Fatal(got)
			}
			if tc.action {
				if result.NoOp || result.Policy == nil || result.Policy.Name != "changed" || strings.TrimSpace(got[2].body) != `{"firewall_rule_id":"r"}` {
					t.Fatal(result, got)
				}
			} else if !tc.wantError && (!result.NoOp || result.Policy != nil) {
				t.Fatal(result)
			}
		})
	}
}
func TestCloudFirewallPolicyRemoveRuleMissingTargetAndRule(t *testing.T) {
	for _, policyFound := range []bool{false, true} {
		t.Run(map[bool]string{false: "policy-missing", true: "rule-missing"}[policyFound], func(t *testing.T) {
			replies := map[string][]pythonCloudBReply{
				"GET /v2.0/fwaas/firewall_policies/p": {{404, `{}`}}, "GET /v2.0/fwaas/firewall_policies": {{200, `{"firewall_policies":[]}`}},
				"GET /v2.0/fwaas/firewall_rules/r": {{404, `{}`}}, "GET /v2.0/fwaas/firewall_rules": {{200, `{"firewall_rules":[]}`}},
			}
			if policyFound {
				replies["GET /v2.0/fwaas/firewall_policies/p"] = []pythonCloudBReply{{200, `{"firewall_policy":{"id":"p"}}`}}
			}
			client, calls := pythonCloudBClient(t, replies)
			result, err := network.New(client).RemoveRuleFromPolicyRecord(context.Background(), "p", "r")
			if policyFound {
				if err != nil || result == nil || !result.NoOp || result.Lookup == nil || result.RuleLookup != nil || len(calls()) != 3 {
					t.Fatal(result, err, calls())
				}
			} else if !errors.Is(err, resource.ErrNotFound) || result.Lookup != nil || len(calls()) != 2 {
				t.Fatal(result, err, calls())
			}
		})
	}
}
func TestCloudFirewallPolicyRemoveRuleOwnedFiltersAndFallback(t *testing.T) {
	raw := json.RawMessage(`{"description":"allow"}`)
	option := network.WithFirewallPolicyRemoveRuleFilters(raw)
	raw[2] = 'X'
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
		"GET /v2.0/fwaas/firewall_policies/target": {{404, `{}`}}, "GET /v2.0/fwaas/firewall_policies": {{200, `{"firewall_policies":[{"id":"p","name":"target","description":"allow","firewall_rules":["r"]}]}`}},
		"GET /v2.0/fwaas/firewall_rules/rule": {{404, `{}`}}, "GET /v2.0/fwaas/firewall_rules": {{200, `{"firewall_rules":[{"id":"r","name":"rule"}]}`}},
		"PUT /v2.0/fwaas/firewall_policies/p/remove_rule": {{200, `{"id":"p"}`}},
	})
	result, err := network.New(client).RemoveRuleFromPolicyRecord(context.Background(), "target", "rule", option)
	if err != nil || result == nil || result.Policy == nil || len(calls()) != 5 {
		t.Fatal(result, err, calls())
	}
	for i, call := range calls() {
		if i < 2 {
			if call.query.Get("description") != "allow" {
				t.Fatal(calls())
			}
		} else if call.query.Get("description") != "" {
			t.Fatal(calls())
		}
	}
}
func TestCloudFirewallPolicyRemoveRulePreflightAndFalseyFilters(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `0`, `""`, `[]`, `{}`} {
		t.Run(raw, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_policies/p": {{200, `{"firewall_policy":{"id":"p","firewall_rules":[]}}`}}, "GET /v2.0/fwaas/firewall_rules/r": {{200, `{"firewall_rule":{"id":"r"}}`}}})
			result, err := network.New(client).RemoveRuleFromPolicyRecord(context.Background(), "p", "r", network.WithFirewallPolicyRemoveRuleFilters(json.RawMessage(raw)))
			if err != nil || !result.NoOp || len(calls()) != 2 {
				t.Fatal(result, err, calls())
			}
		})
	}
	for _, raw := range []string{`"truthy"`, `[1]`, `{"ignore_missing":null}`, `{"name_or_id":null}`, `{"headers":{}}`} {
		t.Run("invalid"+raw, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, nil)
			result, err := network.New(client).RemoveRuleFromPolicyRecord(context.Background(), "p", "r", network.WithFirewallPolicyRemoveRuleFilters(json.RawMessage(raw)))
			if !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 0 {
				t.Fatal(result, err, calls())
			}
		})
	}
	client, calls := pythonCloudBClient(t, nil)
	s := network.New(client)
	for _, opts := range [][]network.FirewallPolicyRemoveRuleOption{{nil}, {func(*network.FirewallPolicyRemoveRuleOpts) error { s.API = nil; return nil }}} {
		result, err := s.RemoveRuleFromPolicyRecord(context.Background(), "p", "r", opts...)
		if result != nil || err == nil || len(calls()) != 0 {
			t.Fatal(result, err, calls())
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := network.New(client).RemoveRuleFromPolicyRecord(ctx, "p", "r")
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestCloudFirewallPolicyRemoveRuleFailureKeepsLookupProof(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		ruleCode, actionCode int
	}{
		{"rule-denied", 403, 200}, {"rule-physical-not-clean-missing", 500, 200}, {"action-conflict", 200, http.StatusConflict}, {"action-source-status-gap", 200, 202},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
				"GET /v2.0/fwaas/firewall_policies/p":             {{200, `{"firewall_policy":{"id":"p","firewall_rules":["r"]}}`}},
				"GET /v2.0/fwaas/firewall_rules/r":                {{tc.ruleCode, `{"firewall_rule":{"id":"r"}}`}},
				"GET /v2.0/fwaas/firewall_rules":                  {{http.StatusForbidden, `{"error":"list denied"}`}},
				"PUT /v2.0/fwaas/firewall_policies/p/remove_rule": {{tc.actionCode, `{"id":"p"}`}},
			})
			result, err := network.New(client).RemoveRuleFromPolicyRecord(context.Background(), "p", "r")
			if err == nil || result == nil || result.Lookup == nil || result.RuleLookup == nil || result.Policy != nil || result.NoOp {
				t.Fatal(result, err, calls())
			}
			want := 2
			if tc.ruleCode == 200 || tc.ruleCode == http.StatusForbidden {
				want = 3
			}
			if len(calls()) != want {
				t.Fatal(calls())
			}
		})
	}
}
