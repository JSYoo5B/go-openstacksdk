package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestCloudFirewallPolicyInsertOrderBothPositionsAndDefaultNulls(t *testing.T) {
	for _, both := range []bool{false, true} {
		t.Run(map[bool]string{false: "neither", true: "both"}[both], func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
				"GET /v2.0/fwaas/firewall_policies/p":             {{200, `{"firewall_policy":{"id":"p","firewall_rules":[]}}`}},
				"GET /v2.0/fwaas/firewall_rules/r":                {{200, `{"firewall_rule":{"id":"inserted"}}`}},
				"GET /v2.0/fwaas/firewall_rules/a":                {{200, `{"firewall_rule":{"id":"after"}}`}},
				"GET /v2.0/fwaas/firewall_rules/b":                {{200, `{"firewall_rule":{"id":"before"}}`}},
				"PUT /v2.0/fwaas/firewall_policies/p/insert_rule": {{202, `{"id":"p","firewall_rules":["inserted"],"x_extension":true}`}},
			})
			opts := []network.FirewallPolicyInsertOption{network.WithFirewallPolicyInsertFilters(json.RawMessage(`{"project_id":"owner"}`))}
			if both {
				opts = append(opts, network.WithFirewallPolicyInsertBefore("b"), network.WithFirewallPolicyInsertAfter("a"))
			}
			result, err := network.New(client).InsertFirewallRuleIntoPolicyRecord(context.Background(), "p", "r", opts...)
			if err != nil || result == nil || result.NoOp || result.Lookup == nil || result.Rule == nil || result.Action == nil || result.Action.Policy == nil || result.Action.StatusCode != 202 {
				t.Fatal(result, err)
			}
			got := calls()
			want := 3
			if both {
				want = 5
			}
			if len(got) != want {
				t.Fatal(got)
			}
			if got[0].query.Get("project_id") != "owner" {
				t.Fatal(got)
			}
			for _, call := range got[1:] {
				if len(call.query) != 0 {
					t.Fatal(got)
				}
			}
			var body map[string]json.RawMessage
			json.Unmarshal([]byte(got[len(got)-1].body), &body)
			if len(body) != 3 || string(body["firewall_rule_id"]) != `"inserted"` {
				t.Fatal(got)
			}
			if both {
				if got[2].path != "/v2.0/fwaas/firewall_rules/a" || got[3].path != "/v2.0/fwaas/firewall_rules/b" || string(body["insert_after"]) != `"after"` || string(body["insert_before"]) != `"before"` {
					t.Fatal(got)
				}
			} else if string(body["insert_after"]) != "null" || string(body["insert_before"]) != "null" {
				t.Fatal(got)
			}
			if string(result.Action.Fields["x_extension"]) != "true" {
				t.Fatal(result.Action)
			}
		})
	}
}
func TestCloudFirewallPolicyInsertAlreadyPresentSkipsInvalidPositions(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
		"GET /v2.0/fwaas/firewall_policies/p": {{200, `{"firewall_policy":{"id":"p","firewall_rules":["r"]}}`}},
		"GET /v2.0/fwaas/firewall_rules/r":    {{200, `{"firewall_rule":{"id":"r"}}`}},
	})
	result, err := network.New(client).InsertFirewallRuleIntoPolicyRecord(context.Background(), "p", "r", network.WithFirewallPolicyInsertAfterJSON(json.RawMessage(`[]`)), network.WithFirewallPolicyInsertBeforeJSON(json.RawMessage(`{broken`)))
	if err != nil || result == nil || !result.NoOp || result.Action != nil || len(result.Positions) != 0 || len(calls()) != 2 {
		t.Fatal(result, err, calls())
	}
}
func TestCloudFirewallPolicyInsertMissingMembershipFailsAfterRuleLookup(t *testing.T) {
	for _, body := range []string{`{"id":"p"}`, `{"id":"p","firewall_rules":null}`} {
		t.Run(body, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_policies/p": {{200, `{"firewall_policy":` + body + `}`}}, "GET /v2.0/fwaas/firewall_rules/r": {{200, `{"firewall_rule":{"id":"r"}}`}}})
			result, err := network.New(client).InsertFirewallRuleIntoPolicyRecord(context.Background(), "p", "r", network.WithFirewallPolicyInsertAfter("never"))
			if err == nil || result == nil || result.Rule == nil || result.Action != nil || len(result.Positions) != 0 || len(calls()) != 2 {
				t.Fatal(result, err, calls())
			}
		})
	}
}
func TestCloudFirewallPolicyInsertLaterPositionFailureKeepsReceipts(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{
		"GET /v2.0/fwaas/firewall_policies/p":    {{200, `{"firewall_policy":{"id":"p","firewall_rules":[]}}`}},
		"GET /v2.0/fwaas/firewall_rules/r":       {{200, `{"firewall_rule":{"id":"r"}}`}},
		"GET /v2.0/fwaas/firewall_rules/a":       {{200, `{"firewall_rule":{"id":"a"}}`}},
		"GET /v2.0/fwaas/firewall_rules/missing": {{404, `{}`}},
		"GET /v2.0/fwaas/firewall_rules":         {{200, `{"firewall_rules":[]}`}},
	})
	result, err := network.New(client).InsertFirewallRuleIntoPolicyRecord(context.Background(), "p", "r", network.WithFirewallPolicyInsertAfter("a"), network.WithFirewallPolicyInsertBefore("missing"))
	if !errors.Is(err, resource.ErrNotFound) || result == nil || result.Lookup == nil || result.Rule == nil || len(result.Positions) < 1 || result.Action != nil || len(calls()) != 5 {
		t.Fatal(result, err, calls())
	}
}
func TestCloudFirewallPolicyInsertOwnedOptionsAndPreflight(t *testing.T) {
	raw := json.RawMessage(`null`)
	opt := network.WithFirewallPolicyInsertAfterJSON(raw)
	raw[0] = 'X'
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_policies/p": {{200, `{"firewall_policy":{"id":"p","firewall_rules":["r"]}}`}}, "GET /v2.0/fwaas/firewall_rules/r": {{200, `{"firewall_rule":{"id":"r"}}`}}})
	service := network.New(client)
	result, err := service.InsertFirewallRuleIntoPolicyRecord(context.Background(), "p", "r", opt)
	if err != nil || result == nil || !result.NoOp || len(calls()) != 2 {
		t.Fatal(result, err, calls())
	}
	for _, opts := range [][]network.FirewallPolicyInsertOption{{nil}, {network.WithFirewallPolicyInsertFilters(json.RawMessage(`{"ignore_missing":null}`))}, {network.WithFirewallPolicyInsertFilters(json.RawMessage(`"expression"`))}} {
		result, err = service.InsertFirewallRuleIntoPolicyRecord(context.Background(), "p", "r", opts...)
		if err == nil || len(calls()) != 2 {
			t.Fatal(result, err, calls())
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = service.InsertFirewallRuleIntoPolicyRecord(ctx, "p", "r")
	if !errors.Is(err, context.Canceled) || result != nil || len(calls()) != 2 {
		t.Fatal(result, err, calls())
	}
}
