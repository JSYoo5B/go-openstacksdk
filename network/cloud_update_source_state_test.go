package network_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/gophercloud/gophercloud/v2"
)

// sourceStateClient serves scripted replies keyed by "METHOD path?query" and
// records every request, so tests can prove which PUT (if any) was sent.
func sourceStateClient(t *testing.T, replies map[string]receiptReply) (*gophercloud.ServiceClient, func() []string) {
	t.Helper()
	cloud := testcloud.New(t)
	var mu sync.Mutex
	var calls []string
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		key := r.Method + " " + r.URL.Path
		if r.URL.RawQuery != "" {
			key += "?" + r.URL.RawQuery
		}
		mu.Lock()
		calls = append(calls, key+" "+string(raw))
		mu.Unlock()
		reply, ok := replies[key]
		if !ok {
			t.Errorf("unexpected request %s", key)
			testcloud.JSON(w, http.StatusInternalServerError, `{}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(reply.status)
		_, _ = io.WriteString(w, reply.body)
	})
	client := cloud.Client("network", "/")
	client.ResourceBase = cloud.Server.URL + "/v2.0/"
	return client, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), calls...)
	}
}

func sourceStatePuts(calls []string) []string {
	var puts []string
	for _, call := range calls {
		if strings.HasPrefix(call, "PUT ") {
			puts = append(puts, call)
		}
	}
	return puts
}

// TestCloudUpdateSourceStateCanonicalLookupDirtyComparison covers member GET
// rows whose attribute is named only by its Python attribute name, and
// canonical/wire collisions. Python Resource._consume_attrs stores both under
// the wire key in CPython dict order (first position, last value), and the
// update compares the supplied wire value against that state.
func TestCloudUpdateSourceStateCanonicalLookupDirtyComparison(t *testing.T) {
	cases := []struct {
		name, lookup string
		wantPut      string
	}{
		{"canonical-only equal is no-op", `{"id":"p","name":"r","is_admin_state_up":false}`, ""},
		{"wire then canonical keeps canonical", `{"id":"p","admin_state_up":true,"is_admin_state_up":false}`, ""},
		{"canonical then wire keeps wire", `{"id":"p","is_admin_state_up":false,"admin_state_up":true}`, `PUT /v2.0/routers/p {"router":{"admin_state_up":false}}`},
		// CPython keeps the first position of a duplicate key with its last
		// value, so the later canonical member still wins.
		{"duplicate wire keeps first position", `{"id":"p","admin_state_up":true,"is_admin_state_up":false,"admin_state_up":true}`, ""},
		{"canonical-only differing value", `{"id":"p","is_admin_state_up":true}`, `PUT /v2.0/routers/p {"router":{"admin_state_up":false}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := sourceStateClient(t, map[string]receiptReply{
				"GET /v2.0/routers/p": {status: http.StatusOK, body: `{"router":` + tc.lookup + `}`},
				"PUT /v2.0/routers/p": {status: http.StatusOK, body: `{"router":{"id":"p","admin_state_up":false}}`},
			})
			result, err := network.New(client).UpdateRouterRecord(context.Background(), "p", network.WithRouterUpdateAdminStateUp(false))
			if err != nil {
				t.Fatal(err)
			}
			puts := sourceStatePuts(calls())
			if tc.wantPut == "" {
				if !result.NoOp || len(puts) != 0 || result.Response != nil {
					t.Fatalf("expected Python no-dirty no-op, got NoOp=%v puts=%q", result.NoOp, puts)
				}
				if got := rawText(result.Lookup.SourceBody()["admin_state_up"]); got != "false" {
					t.Fatalf("SourceBody admin_state_up = %s", got)
				}
				if _, exists := result.Lookup.SourceBody()["is_admin_state_up"]; exists {
					t.Fatal("canonical key must be stored under the wire key")
				}
				return
			}
			if len(puts) != 1 || puts[0] != tc.wantPut {
				t.Fatalf("puts = %q, want %q", puts, tc.wantPut)
			}
		})
	}
}

// TestCloudUpdateSourceStateListRowAndResponseMapping covers the fallback LIST
// constructor (Resource.existing/_collect_attrs) and the accepted response
// translation. A list row named by attribute names compares by wire key; the
// response's canonical members are merged under their wire keys while Body
// keeps the actual member names, and "self" never enters the state.
func TestCloudUpdateSourceStateListRowAndResponseMapping(t *testing.T) {
	client, calls := sourceStateClient(t, map[string]receiptReply{
		"GET /v2.0/ports/web":                {status: http.StatusNotFound, body: `{"NeutronError":{"type":"PortNotFound"}}`},
		"GET /v2.0/ports?name=web":           {status: http.StatusOK, body: `{"ports":[{"id":"p1","name":"web","self":"x","is_port_security_enabled":true,"is_admin_state_up":true,"binding_host_id":"h1","x_ext":1}]}`},
		"GET /v2.0/ports?marker=p1&name=web": {status: http.StatusOK, body: `{"ports":[]}`},
		"PUT /v2.0/ports/p1":                 {status: http.StatusOK, body: `{"port":{"id":"p1","name":"web","is_admin_state_up":false,"binding_host_id":"h2","admin_state_up":false,"self":"y"}}`},
	})
	result, err := network.New(client).UpdatePortRecord(context.Background(), "web",
		network.WithPortUpdatePortSecurityEnabled(true),
		network.WithPortUpdateAdminStateUp(false),
		network.WithPortUpdateAttribute("binding:host_id", "h1"),
	)
	if err != nil {
		t.Fatal(err)
	}
	puts := sourceStatePuts(calls())
	// port_security_enabled and binding:host_id equal the Python state, so
	// only admin_state_up is dirty.
	if len(puts) != 1 || puts[0] != `PUT /v2.0/ports/p1 {"port":{"admin_state_up":false}}` {
		t.Fatalf("puts = %q (all %q)", puts, calls())
	}
	state := result.Lookup.SourceBody()
	for key, want := range map[string]string{"port_security_enabled": "true", "admin_state_up": "true", "binding:host_id": `"h1"`, "x_ext": "1"} {
		if got := rawText(state[key]); got != want {
			t.Fatalf("SourceBody[%s] = %s, want %s", key, got, want)
		}
	}
	for _, key := range []string{"self", "is_port_security_enabled", "binding_host_id"} {
		if _, exists := state[key]; exists {
			t.Fatalf("SourceBody kept %q", key)
		}
	}
	merged := result.Response.Merged.Body
	for key, want := range map[string]string{"admin_state_up": "false", "binding:host_id": `"h2"`, "port_security_enabled": "true", "x_ext": "1"} {
		if got := rawText(merged[key]); got != want {
			t.Fatalf("Merged[%s] = %s, want %s", key, got, want)
		}
	}
	for _, key := range []string{"self", "is_admin_state_up", "binding_host_id"} {
		if _, exists := merged[key]; exists {
			t.Fatalf("Merged kept %q", key)
		}
	}
	if _, exists := result.Response.Body["is_admin_state_up"]; !exists {
		t.Fatal("Body must keep the actual response member names")
	}
	if result.Port == nil || result.Port.AdminStateUp || result.Port.Name != "web" {
		t.Fatalf("native projection = %+v", result.Port)
	}
}

// TestCloudUpdateSourceStateRouterRevisionAndSubnetAlias covers Router's own
// revision_number declaration (wire "revision"), so a Neutron
// "revision_number" member is the attribute name and maps to "revision", and
// Subnet's is_dhcp_enabled/enable_dhcp alias.
func TestCloudUpdateSourceStateRouterRevisionAndSubnetAlias(t *testing.T) {
	client, _ := sourceStateClient(t, map[string]receiptReply{
		"GET /v2.0/routers/p": {status: http.StatusOK, body: `{"router":{"id":"p","name":"old","revision_number":3}}`},
		"PUT /v2.0/routers/p": {status: http.StatusOK, body: `{"router":{"id":"p","name":"new","revision_number":4,"is_ha":true}}`},
	})
	router, err := network.New(client).UpdateRouterRecord(context.Background(), "p", network.WithRouterUpdateName("new"))
	if err != nil {
		t.Fatal(err)
	}
	if got := rawText(router.Lookup.SourceBody()["revision"]); got != "3" {
		t.Fatalf("lookup revision = %s", got)
	}
	merged := router.Response.Merged.Body
	if rawText(merged["revision"]) != "4" || rawText(merged["ha"]) != "true" || merged["revision_number"] != nil || merged["is_ha"] != nil {
		t.Fatalf("router merged = %v", merged)
	}

	client, calls := sourceStateClient(t, map[string]receiptReply{
		"GET /v2.0/subnets/s": {status: http.StatusOK, body: `{"subnet":{"id":"s","name":"n","is_dhcp_enabled":false}}`},
	})
	subnet, err := network.New(client).UpdateSubnetRecord(context.Background(), "s", network.WithSubnetUpdateJSON(json.RawMessage(`{"enable_dhcp":false}`)))
	if err != nil {
		t.Fatal(err)
	}
	if !subnet.NoOp || len(sourceStatePuts(calls())) != 0 {
		t.Fatalf("subnet canonical-only enable_dhcp must be a Python no-op: %q", calls())
	}
}

// TestCloudUpdateSourceStateFirewallIdentityMapping records that the three
// firewall resources declare no attribute aliases (every Body name equals its
// wire key), so their state is the actual row without "self", and a
// case-variant member stays an unknown member rather than a declared one.
func TestCloudUpdateSourceStateFirewallIdentityMapping(t *testing.T) {
	client, calls := sourceStateClient(t, map[string]receiptReply{
		"GET /v2.0/fwaas/firewall_rules/r": {status: http.StatusOK, body: `{"firewall_rule":{"id":"r","name":"old","Name":"case","self":"x","enabled":true}}`},
		"PUT /v2.0/fwaas/firewall_rules/r": {status: http.StatusOK, body: `{"firewall_rule":{"id":"r","name":"new","Enabled":false}}`},
	})
	result, err := network.New(client).UpdateFirewallRuleRecord(context.Background(), "r", network.WithFirewallRuleUpdateJSON(json.RawMessage(`{"name":"new","enabled":true}`)))
	if err != nil {
		t.Fatal(err)
	}
	puts := sourceStatePuts(calls())
	if len(puts) != 1 || puts[0] != `PUT /v2.0/fwaas/firewall_rules/r {"firewall_rule":{"name":"new"}}` {
		t.Fatalf("puts = %q", puts)
	}
	merged := result.Response.Merged.Body
	if rawText(merged["name"]) != `"new"` || rawText(merged["Name"]) != `"case"` || rawText(merged["enabled"]) != "true" || rawText(merged["Enabled"]) != "false" || merged["self"] != nil {
		t.Fatalf("firewall merged = %v", merged)
	}
}
