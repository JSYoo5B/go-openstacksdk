package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type firewallKwargsCall struct{ line, version, proof string }

// firewallKwargsClient serves queued replies keyed by "METHOD path?query" and
// records the API version and X-Proof request headers of every request.
func firewallKwargsClient(t *testing.T, replies map[string][]receiptReply) (*gophercloud.ServiceClient, func() []firewallKwargsCall) {
	t.Helper()
	cloud := testcloud.New(t)
	var mu sync.Mutex
	var calls []firewallKwargsCall
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		key := r.Method + " " + r.URL.Path
		if r.URL.RawQuery != "" {
			key += "?" + r.URL.RawQuery
		}
		mu.Lock()
		calls = append(calls, firewallKwargsCall{strings.TrimSpace(key + " " + string(raw)), r.Header.Get("OpenStack-API-Version"), r.Header.Get("X-Proof")})
		queue := replies[key]
		if len(queue) == 0 {
			mu.Unlock()
			t.Errorf("unexpected request %s", key)
			testcloud.JSON(w, http.StatusInternalServerError, `{}`)
			return
		}
		reply := queue[0]
		replies[key] = queue[1:]
		mu.Unlock()
		if reply.body == "" {
			w.WriteHeader(reply.status)
			return
		}
		testcloud.JSON(w, reply.status, reply.body)
	})
	client := cloud.Client("network", "/")
	client.ResourceBase = cloud.Server.URL + "/v2.0/"
	return client, func() []firewallKwargsCall {
		mu.Lock()
		defer mu.Unlock()
		return append([]firewallKwargsCall(nil), calls...)
	}
}

func firewallKwargsExpect(t *testing.T, got []firewallKwargsCall, want ...firewallKwargsCall) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("calls = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("call %d = %+v, want %+v (all %+v)", i, got[i], want[i], got)
		}
	}
}

// TestCloudFirewallDeleteFindHeadersAndMicroversionPhases covers raw Find
// kwargs forwarded by delete_firewall_*(filters=...): filters.headers is an
// ordinary initial member GET query value and becomes HTTP headers only on
// the fallback LIST; filters.microversion versions only the initial GET,
// while the fallback LIST and the DELETE return to the adapter default.
func TestCloudFirewallDeleteFindHeadersAndMicroversionPhases(t *testing.T) {
	client, calls := firewallKwargsClient(t, map[string][]receiptReply{
		"GET /v2.0/fwaas/firewall_rules/web?headers=X-Proof": {{status: http.StatusNotFound, body: `{}`}},
		"GET /v2.0/fwaas/firewall_rules?name=web":            {{status: http.StatusOK, body: `{"firewall_rules":[{"id":"r1","name":"web"}]}`}},
		"DELETE /v2.0/fwaas/firewall_rules/r1":               {{status: http.StatusNoContent}},
	})
	result, err := network.New(client).DeleteFirewallRuleRecord(context.Background(), "web", network.WithFirewallDeleteFilters(json.RawMessage(`{"headers":{"X-Proof":"list"},"microversion":"2.5"}`)))
	if err != nil || result == nil || !result.Deleted {
		t.Fatal(result, err)
	}
	firewallKwargsExpect(t, calls(),
		firewallKwargsCall{line: "GET /v2.0/fwaas/firewall_rules/web?headers=X-Proof", version: "network 2.5"},
		firewallKwargsCall{line: "GET /v2.0/fwaas/firewall_rules?name=web", proof: "list"},
		firewallKwargsCall{line: "DELETE /v2.0/fwaas/firewall_rules/r1"},
	)

	// A successful GET keeps the requested version and query and the found
	// member is deleted on the class route without the raw headers.
	client, calls = firewallKwargsClient(t, map[string][]receiptReply{
		"GET /v2.0/fwaas/firewall_groups/g?headers=X-Proof": {{status: http.StatusOK, body: `{"firewall_group":{"id":"g","name":"web"}}`}},
		"DELETE /v2.0/fwaas/firewall_groups/g":              {{status: http.StatusNoContent}},
	})
	result, err = network.New(client).DeleteFirewallGroupRecord(context.Background(), "g", network.WithFirewallDeleteFilters(json.RawMessage(`{"headers":{"X-Proof":"list"},"microversion":"2.7"}`)))
	if err != nil || result == nil || !result.Deleted {
		t.Fatal(result, err)
	}
	firewallKwargsExpect(t, calls(),
		firewallKwargsCall{line: "GET /v2.0/fwaas/firewall_groups/g?headers=X-Proof", version: "network 2.7"},
		firewallKwargsCall{line: "DELETE /v2.0/fwaas/firewall_groups/g"},
	)

	// The Find microversion keyword and the Go transport Microversion option
	// would select two different GET versions, so the pair is rejected.
	client, calls = firewallKwargsClient(t, map[string][]receiptReply{})
	_, err = network.New(client).DeleteFirewallPolicyRecord(context.Background(), "p", network.WithFirewallDeleteFilters(json.RawMessage(`{"microversion":"2.5"}`)), network.WithFirewallDeleteMicroversion("2.1"))
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls()) != 0 {
		t.Fatal(err, calls())
	}
}

// TestCloudFirewallUpdateFindFetchControls covers base_path,
// resource_response_key and skip_cache Find kwargs: they route and select the
// initial GET only (skip_cache has no Go cache to bypass), seed the found
// Resource as unknown constructor attributes, and never change the PUT route
// or the sent dirty body.
func TestCloudFirewallUpdateFindFetchControls(t *testing.T) {
	client, calls := firewallKwargsClient(t, map[string][]receiptReply{
		"GET /v2.0/fwaas/custom_rules/r":   {{status: http.StatusOK, body: `{"rule":{"id":"r","name":"old"},"firewall_rule":{"id":"wrong"}}`}},
		"PUT /v2.0/fwaas/firewall_rules/r": {{status: http.StatusOK, body: `{"firewall_rule":{"id":"r","name":"new"}}`}},
	})
	filters := json.RawMessage(`{"base_path":"fwaas/custom_rules","resource_response_key":"rule","skip_cache":true}`)
	result, err := network.New(client).UpdateFirewallRuleRecord(context.Background(), "r", network.WithFirewallRuleUpdateFilters(filters), network.WithFirewallRuleUpdateName("new"))
	if err != nil || result == nil || result.Response == nil {
		t.Fatal(result, err)
	}
	firewallKwargsExpect(t, calls(),
		firewallKwargsCall{line: "GET /v2.0/fwaas/custom_rules/r"},
		firewallKwargsCall{line: `PUT /v2.0/fwaas/firewall_rules/r {"firewall_rule":{"name":"new"}}`},
	)
	merged := result.Response.Merged.Body
	for key, want := range map[string]string{"name": `"new"`, "base_path": `"fwaas/custom_rules"`, "resource_response_key": `"rule"`, "skip_cache": "true"} {
		if got := rawText(merged[key]); got != want {
			t.Fatalf("Merged[%s] = %s, want %s", key, got, want)
		}
	}

	// list_base_path changes only the fallback LIST route.
	client, calls = firewallKwargsClient(t, map[string][]receiptReply{
		"GET /v2.0/fwaas/firewall_policies/web":    {{status: http.StatusNotFound, body: `{}`}},
		"GET /v2.0/fwaas/custom_policies?name=web": {{status: http.StatusOK, body: `{"firewall_policies":[{"id":"p1","name":"web"}]}`}},
		"PUT /v2.0/fwaas/firewall_policies/p1":     {{status: http.StatusOK, body: `{"firewall_policy":{"id":"p1","name":"web","description":"d"}}`}},
	})
	policy, err := network.New(client).UpdateFirewallPolicyRecord(context.Background(), "web", network.WithFirewallPolicyUpdateFilters(json.RawMessage(`{"list_base_path":"fwaas/custom_policies"}`)), network.WithFirewallPolicyUpdateJSON(json.RawMessage(`{"description":"d"}`)))
	if err != nil || policy == nil || policy.Response == nil {
		t.Fatal(policy, err, calls())
	}
	if got := calls(); len(got) != 3 || got[1].line != "GET /v2.0/fwaas/custom_policies?name=web" || got[2].line != `PUT /v2.0/fwaas/firewall_policies/p1 {"firewall_policy":{"description":"d"}}` {
		t.Fatalf("calls = %+v", got)
	}
}

// TestCloudFirewallDeleteFindCapEqualsWholePage covers a raw max_items cap
// equal to a full fallback page. Python compares the cap before each row, so
// it performs the next marker GET and stops at its first row; a failure of
// that GET is observed and no DELETE begins.
func TestCloudFirewallDeleteFindCapEqualsWholePage(t *testing.T) {
	client, calls := firewallKwargsClient(t, map[string][]receiptReply{
		"GET /v2.0/fwaas/firewall_rules/web?max_items=1":            {{status: http.StatusNotFound, body: `{}`}},
		"GET /v2.0/fwaas/firewall_rules?limit=1&name=web":           {{status: http.StatusOK, body: `{"firewall_rules":[{"id":"r1","name":"web"}]}`}},
		"GET /v2.0/fwaas/firewall_rules?limit=1&marker=r1&name=web": {{status: http.StatusOK, body: `{"firewall_rules":[{"id":"r2","name":"web"}]}`}},
		"DELETE /v2.0/fwaas/firewall_rules/r1":                      {{status: http.StatusNoContent}},
	})
	result, err := network.New(client).DeleteFirewallRuleRecord(context.Background(), "web", network.WithFirewallDeleteFilters(json.RawMessage(`{"max_items":1}`)))
	if err != nil || result == nil || !result.Deleted {
		t.Fatal(result, err, calls())
	}
	firewallKwargsExpect(t, calls(),
		firewallKwargsCall{line: "GET /v2.0/fwaas/firewall_rules/web?max_items=1"},
		firewallKwargsCall{line: "GET /v2.0/fwaas/firewall_rules?limit=1&name=web"},
		firewallKwargsCall{line: "GET /v2.0/fwaas/firewall_rules?limit=1&marker=r1&name=web"},
		firewallKwargsCall{line: "DELETE /v2.0/fwaas/firewall_rules/r1"},
	)

	client, calls = firewallKwargsClient(t, map[string][]receiptReply{
		"GET /v2.0/fwaas/firewall_rules/web?max_items=1":            {{status: http.StatusNotFound, body: `{}`}},
		"GET /v2.0/fwaas/firewall_rules?limit=1&name=web":           {{status: http.StatusOK, body: `{"firewall_rules":[{"id":"r1","name":"web"}]}`}},
		"GET /v2.0/fwaas/firewall_rules?limit=1&marker=r1&name=web": {{status: http.StatusServiceUnavailable, body: `{}`}},
	})
	result, err = network.New(client).DeleteFirewallRuleRecord(context.Background(), "web", network.WithFirewallDeleteFilters(json.RawMessage(`{"max_items":1}`)))
	if err == nil || result == nil || result.Deleted || len(calls()) != 3 {
		t.Fatal(result, err, calls())
	}
}
