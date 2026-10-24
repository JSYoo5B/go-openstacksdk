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

func TestCloudFirewallRuleUpdateLookupOnlyAndDirtySelection(t *testing.T) {
	for _, tc := range []struct {
		name, body, attrs, want string
		noOp                    bool
	}{
		{"empty", `{"id":"p","name":"old"}`, `{}`, ``, true},
		{"same", `{"id":"p","name":"old","enabled":true}`, `{"name":"old","enabled":1}`, ``, true},
		{"changed-only", `{"id":"p","name":"old","enabled":true}`, `{"name":"new","enabled":true}`, `{"firewall_rule":{"name":"new"}}`, false},
		{"raw-string-bool", `{"id":"p","enabled":"false"}`, `{"enabled":true}`, `{"firewall_rule":{"enabled":true}}`, false},
		{"raw-fields", `{"id":"p"}`, `{"protocol":"ANY","destination_firewall_group_id":null,"enabled":false,"shared":0,"ip_version":"4"}`, `{"firewall_rule":{"protocol":"ANY","destination_firewall_group_id":null,"enabled":false,"shared":0,"ip_version":"4"}}`, false},
		{"null", `{"id":"p","name":"old"}`, `{"name":null}`, `{"firewall_rule":{"name":null}}`, false},
		{"missing-null", `{"id":"p"}`, `{"protocol":null}`, `{"firewall_rule":{"protocol":null}}`, false},
		{"explicit-false-empty", `{"id":"p","enabled":true,"source_firewall_group_id":["sg"]}`, `{"enabled":false,"source_firewall_group_id":[]}`, `{"firewall_rule":{"enabled":false,"source_firewall_group_id":[]}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_rules/p": {{200, `{"firewall_rule":` + tc.body + `}`}}, "PUT /v2.0/fwaas/firewall_rules/p": {{200, `{"firewall_rule":{"id":"p","name":"response"}}`}}})
			result, err := network.New(client).UpdateFirewallRuleRecord(context.Background(), "p", network.WithFirewallRuleUpdateJSON(json.RawMessage(tc.attrs)))
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
				if result.Rule != nil {
					t.Fatal("fabricated PUT response")
				}
				return
			}
			var actual, expected any
			json.Unmarshal([]byte(got[1].body), &actual)
			json.Unmarshal([]byte(tc.want), &expected)
			if !reflect.DeepEqual(actual, expected) || result.Rule == nil || result.Rule.Name != "response" {
				t.Fatal(got, result)
			}
		})
	}
}
func TestCloudFirewallRuleUpdateNameFallbackAndMissing(t *testing.T) {
	for _, found := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "name"}[found], func(t *testing.T) {
			rows := `[]`
			if found {
				rows = `[{"id":"actual","name":"target"}]`
			}
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_rules/target": {{404, `{}`}}, "GET /v2.0/fwaas/firewall_rules": {{200, `{"firewall_rules":` + rows + `}`}}, "PUT /v2.0/fwaas/firewall_rules/actual": {{200, `{"firewall_rule":{"id":"actual"}}`}}})
			result, err := network.New(client).UpdateFirewallRuleRecord(context.Background(), "target", network.WithFirewallRuleUpdateName("new"))
			if found {
				if err != nil || result.Rule == nil || len(calls()) != 3 {
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
func TestCloudFirewallRuleUpdateOwnedOptionsAndPreflight(t *testing.T) {
	raw := json.RawMessage(`{"name":"owned"}`)
	option := network.WithFirewallRuleUpdateJSON(raw)
	raw[2] = 'X'
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_rules/p": {{200, `{"firewall_rule":{"id":"p","name":"owned"}}`}}})
	service := network.New(client)
	result, err := service.UpdateFirewallRuleRecord(context.Background(), "p", option)
	if err != nil || !result.NoOp || len(calls()) != 1 {
		t.Fatal(result, err, calls())
	}
	for _, opts := range [][]network.FirewallRuleUpdateOption{{nil}, {network.WithFirewallRuleUpdateAttribute("is_enabled", true)}, {network.WithFirewallRuleUpdateAttribute("if_revision", 0)}, {network.WithFirewallRuleUpdateAttribute("name", make(chan int))}, {network.WithFirewallRuleUpdateJSON(json.RawMessage(`[]`))}, {network.WithFirewallRuleUpdateJSON(json.RawMessage(`{"x_extension":1}`))}} {
		result, err = service.UpdateFirewallRuleRecord(context.Background(), "p", opts...)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 1 {
			t.Fatal(result, err, calls())
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = service.UpdateFirewallRuleRecord(ctx, "p")
	if result != nil || !errors.Is(err, context.Canceled) || len(calls()) != 1 {
		t.Fatal(result, err)
	}
	result, err = service.UpdateFirewallRuleRecord(context.Background(), "p", func(*network.FirewallRuleUpdateOpts) error { service.API = nil; return nil })
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 1 {
		t.Fatal(result, err)
	}
}
func TestCloudFirewallRuleUpdateConflictKeepsLookupAndSingleAttempt(t *testing.T) {
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_rules/p": {{200, `{"firewall_rule":{"id":"p","name":"old"}}`}}, "PUT /v2.0/fwaas/firewall_rules/p": {{http.StatusConflict, `{"error":"conflict"}`}}})
	result, err := network.New(client).UpdateFirewallRuleRecord(context.Background(), "p", network.WithFirewallRuleUpdateName("new"))
	if err == nil || result == nil || result.Lookup == nil || result.Rule != nil || result.NoOp || len(calls()) != 2 {
		t.Fatal(result, err, calls())
	}
}

func TestCloudFirewallRuleUpdateFiltersStrictBindingAndFalseyDefaults(t *testing.T) {
	for _, raw := range []string{`null`, `false`, `0`, `""`, `[]`, `{}`} {
		t.Run(raw, func(t *testing.T) {
			client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_rules/p": {{200, `{"firewall_rule":{"id":"p"}}`}}})
			result, err := network.New(client).UpdateFirewallRuleRecord(context.Background(), "p", network.WithFirewallRuleUpdateFilters(json.RawMessage(raw)))
			if err != nil || result == nil || !result.NoOp || result.Lookup == nil || len(calls()) != 1 || len(calls()[0].query) != 0 {
				t.Fatal(result, err, calls())
			}
		})
	}
	client, calls := pythonCloudBClient(t, map[string][]pythonCloudBReply{"GET /v2.0/fwaas/firewall_rules/target": {{404, `{}`}}, "GET /v2.0/fwaas/firewall_rules": {{200, `{"firewall_rules":[{"id":"actual","name":"target","action":"allow"}]}`}}})
	raw := json.RawMessage(`{"action":"allow"}`)
	option := network.WithFirewallRuleUpdateFilters(raw)
	raw[2] = 'X'
	result, err := network.New(client).UpdateFirewallRuleRecord(context.Background(), "target", option)
	if err != nil || result == nil || !result.NoOp || len(calls()) != 2 || calls()[0].query.Get("action") != "allow" || calls()[1].query.Get("action") != "allow" {
		t.Fatal(result, err, calls())
	}
	for _, raw := range []string{`"truthy-expression"`, `[1]`, `{"ignore_missing":null}`, `{"name_or_id":null}`} {
		before := len(calls())
		result, err = network.New(client).UpdateFirewallRuleRecord(context.Background(), "target", network.WithFirewallRuleUpdateFilters(json.RawMessage(raw)))
		if err == nil || len(calls()) != before {
			t.Fatal(raw, result, err, calls())
		}
	}
}
