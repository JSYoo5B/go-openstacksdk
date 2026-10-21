package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type r2CloudNetServer struct {
	mu    sync.Mutex
	calls []string
	reply func(path string, q url.Values) (int, string)
	host  string
}

func (s *r2CloudNetServer) service(t *testing.T) *network.Service {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.calls = append(s.calls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
		s.mu.Unlock()
		code, body := s.reply(r.URL.Path, r.URL.Query())
		if code == 0 {
			t.Errorf("unexpected request %s", r.URL)
			code, body = 500, `{}`
		}
		testcloud.JSON(w, code, body)
	})
	s.host = cloud.Server.URL
	client := cloud.Client("network", "/")
	client.ResourceBase = cloud.Server.URL + "/v2.0/"
	return network.New(client)
}

func r2CloudNetIDs(t *testing.T, rows any) string {
	t.Helper()
	raw, _ := json.Marshal(rows)
	var list []map[string]any
	_ = json.Unmarshal(raw, &list)
	ids := []string{}
	for _, row := range list {
		ids = append(ids, row["id"].(string))
	}
	return strings.Join(ids, ",")
}

const r2CloudNetGroups = `{"firewall_groups":[{"id":"g1","name":"a","admin_state_up":true},{"id":"g2","name":"b","admin_state_up":false},{"id":"g3","name":"c","admin_state_up":true}]}`

func TestR2CloudNetFirewallGroupListQueryAndLocalFilters(t *testing.T) {
	srv := &r2CloudNetServer{reply: func(p string, q url.Values) (int, string) {
		if p == "/v2.0/fwaas/firewall_groups" {
			return 200, r2CloudNetGroups
		}
		return 0, ""
	}}
	svc := srv.service(t)
	cases := []struct{ filters, ids, query string }{
		{``, "g1,g2,g3", ""},
		{`{}`, "g1,g2,g3", ""},
		{`null`, "g1,g2,g3", ""},
		{`{"name":"a","unknown":1}`, "g1,g2,g3", "name=a"},
		{`{"admin_state_up":true}`, "g1,g3", ""},
		{`{"admin_state_up":1}`, "g1,g3", ""},
		{`{"admin_state_up":false,"id":"g2"}`, "g2", ""},
		{`{"id":"g3","ports":["p1","p2"]}`, "g3", "ports=p1&ports=p2"},
		{`{"id":"nope"}`, "", ""},
		{`{"admin_state_up":"true"}`, "", ""},
		{`{"shared":true}`, "g1,g2,g3", "shared=True"},
	}
	for _, c := range cases {
		srv.calls = nil
		var opts []network.FirewallListOption
		if c.filters != "" {
			opts = append(opts, network.WithFirewallListFilters(json.RawMessage(c.filters)))
		}
		rows, err := svc.ListFirewallGroups(context.Background(), opts...)
		if err != nil || r2CloudNetIDs(t, rows) != c.ids {
			t.Fatal(c, rows, err)
		}
		if len(srv.calls) != 1 || srv.calls[0] != "GET /v2.0/fwaas/firewall_groups?"+c.query {
			t.Fatal(c, srv.calls)
		}
	}
}

func TestR2CloudNetFirewallListRejectsBadFiltersBeforeHTTP(t *testing.T) {
	srv := &r2CloudNetServer{reply: func(string, url.Values) (int, string) { return 0, "" }}
	svc := srv.service(t)
	for _, f := range []string{`[1]`, `"x"`, `{bad`, `{"max_items":1}`, `{"base_path":"/x"}`} {
		_, err := svc.ListFirewallPolicies(context.Background(), network.WithFirewallListFilters(json.RawMessage(f)))
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(f, err)
		}
	}
	if _, err := svc.ListFirewallRules(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestR2CloudNetFirewallPoliciesAndRulesLocalFiltersAndPaging(t *testing.T) {
	var srv *r2CloudNetServer
	srv = &r2CloudNetServer{reply: func(p string, q url.Values) (int, string) {
		switch p {
		case "/v2.0/fwaas/firewall_policies":
			return 200, `{"firewall_policies":[{"id":"p1","audited":true},{"id":"p2","audited":false}]}`
		case "/v2.0/fwaas/firewall_rules":
			if q.Get("marker") == "r1" {
				return 200, `{"firewall_rules":[{"id":"r2","action":"deny"}]}`
			}
			return 200, `{"firewall_rules":[{"id":"r1","action":"allow"}],"firewall_rules_links":[{"rel":"next","href":"` + srv.host + `/v2.0/fwaas/firewall_rules?marker=r1"}]}`
		}
		return 0, ""
	}}
	svc := srv.service(t)
	rows, err := svc.ListFirewallPolicies(context.Background(), network.WithFirewallListFilters(json.RawMessage(`{"audited":false}`)))
	if err != nil || r2CloudNetIDs(t, rows) != "p2" {
		t.Fatal(rows, err)
	}
	// Without a usable next link the first page is the whole result.
	rules, err := svc.ListFirewallRules(context.Background(), network.WithFirewallListFilters(json.RawMessage(`{"action":"deny","id":"r2"}`)))
	if err != nil || len(rules) != 1 || len(srv.calls) != 3 || !strings.Contains(srv.calls[2], "marker=r1") {
		t.Fatal(rules, err, srv.calls)
	}
}

func TestR2CloudNetFirewallListCancelledAndServerError(t *testing.T) {
	srv := &r2CloudNetServer{reply: func(string, url.Values) (int, string) { return 500, `{"NeutronError":{}}` }}
	svc := srv.service(t)
	if rows, err := svc.ListFirewallGroups(context.Background()); err == nil || rows != nil {
		t.Fatal(rows, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	srv.calls = nil
	if _, err := svc.ListFirewallGroups(ctx); err == nil || len(srv.calls) != 0 {
		t.Fatal(err, srv.calls)
	}
}

func TestR2CloudNetQoSRuleTypeDetailsGatesThenGet(t *testing.T) {
	ext := func(aliases ...string) string {
		rows := []string{}
		for _, a := range aliases {
			rows = append(rows, `{"alias":"`+a+`"}`)
		}
		return `{"extensions":[` + strings.Join(rows, ",") + `]}`
	}
	exts := 0
	srv := &r2CloudNetServer{reply: func(p string, q url.Values) (int, string) {
		switch p {
		case "/v2.0/extensions":
			exts++
			if exts == 1 {
				return 200, ext("qos")
			}
			return 200, ext("x", "qos-rule-type-details")
		case "/v2.0/qos/rule-types/bandwidth_limit":
			return 200, `{"rule_type":{"type":"bandwidth_limit","drivers":[{"name":"ovn"}]}}`
		}
		return 0, ""
	}}
	svc := srv.service(t)
	res, err := svc.GetQoSRuleTypeDetails(context.Background(), "bandwidth_limit")
	if err != nil || res == nil || res.RuleType == nil || len(res.Extensions) != 3 || len(srv.calls) != 3 || !strings.HasPrefix(srv.calls[2], "GET /v2.0/qos/rule-types/bandwidth_limit") {
		t.Fatal(res, err, srv.calls)
	}
}

func TestR2CloudNetQoSRuleTypeDetailsMissingExtensionAndErrors(t *testing.T) {
	for name, second := range map[string]string{"missing details": `{"extensions":[{"alias":"qos"}]}`} {
		n := 0
		srv := &r2CloudNetServer{reply: func(p string, q url.Values) (int, string) {
			if p == "/v2.0/extensions" {
				n++
				if n == 1 {
					return 200, `{"extensions":[{"alias":"qos"}]}`
				}
				return 200, second
			}
			return 0, ""
		}}
		svc := srv.service(t)
		_, err := svc.GetQoSRuleTypeDetails(context.Background(), "bandwidth_limit")
		var unavailable *network.UnavailableExtensionError
		if !errors.As(err, &unavailable) || unavailable.Alias != "qos-rule-type-details" || len(srv.calls) != 2 {
			t.Fatal(name, err, srv.calls)
		}
	}
	srv := &r2CloudNetServer{reply: func(p string, q url.Values) (int, string) {
		switch p {
		case "/v2.0/extensions":
			return 200, `{"extensions":[{"alias":"qos"},{"alias":"qos-rule-type-details"}]}`
		case "/v2.0/qos/rule-types/nope":
			return 404, `{"NeutronError":{"message":"nf"}}`
		}
		return 0, ""
	}}
	svc := srv.service(t)
	if res, err := svc.GetQoSRuleTypeDetails(context.Background(), "nope"); err == nil || res == nil || res.RuleType != nil {
		t.Fatal(res, err)
	}
	if _, err := svc.GetQoSRuleTypeDetails(context.Background(), ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
