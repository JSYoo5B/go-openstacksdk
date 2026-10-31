package rules_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonFWRuleWire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}})
}

type pythonFWRuleCall struct{ method, path, query, body string }

type pythonFWRuleRecorder struct {
	mu    sync.Mutex
	calls []pythonFWRuleCall
}

func (r *pythonFWRuleRecorder) snapshot() []pythonFWRuleCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]pythonFWRuleCall(nil), r.calls...)
}

func pythonFWRuleAPI(t *testing.T, recorder *pythonFWRuleRecorder, reply func(*http.Request) *http.Response) *rules.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		recorder.mu.Lock()
		recorder.calls = append(recorder.calls, pythonFWRuleCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		recorder.mu.Unlock()
		return reply(req), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return rules.New(client)
}

// pythonFWRuleExpect compares bodies as JSON so map key order does not matter.
func pythonFWRuleExpect(t *testing.T, got, want []pythonFWRuleCall) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("calls\n got %+v\nwant %+v", got, want)
	}
	for i := range got {
		g, w := got[i], want[i]
		if g.method != w.method || g.path != w.path || g.query != w.query || !pythonFWRuleSameJSON(g.body, w.body) {
			t.Fatalf("call %d\n got %+v\nwant %+v", i, g, w)
		}
	}
}

func pythonFWRuleSameJSON(a, b string) bool {
	if a == "" || b == "" {
		return a == b
	}
	var x, y any
	if json.Unmarshal([]byte(a), &x) != nil || json.Unmarshal([]byte(b), &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

func pythonFWRuleString(v string) *string { return &v }
func pythonFWRuleBool(v bool) *bool       { return &v }

const pythonFWRuleBase = "/neutron/v2.0/fwaas/firewall_rules"

const pythonFWRuleRow = `{"id":"fr-1","name":"ssh","description":"","protocol":"tcp","action":"allow","ip_version":4,"source_ip_address":"10.0.0.0/24","destination_port":"22","enabled":true,"shared":false,"firewall_policy_id":["fp-1"],"project_id":"proj"}`

// FirewallRule is a plain Resource at /fwaas/firewall_rules with resource_key firewall_rule.
func TestPythonFirewallRuleProxyParity(t *testing.T) {
	ctx := context.Background()
	t.Run("create_firewall_rule", func(t *testing.T) {
		var recorder pythonFWRuleRecorder
		api := pythonFWRuleAPI(t, &recorder, func(*http.Request) *http.Response {
			return pythonFWRuleWire(201, `{"firewall_rule":`+pythonFWRuleRow+`}`)
		})
		// create_firewall_rule(name="ssh", protocol="tcp", action="allow", ip_version=4,
		//     source_ip_address="10.0.0.0/24", destination_port="22", enabled=True, x_extension=1)
		created, err := api.Create(ctx, rules.CreateOpts{Name: "ssh", Protocol: rules.ProtocolTCP, Action: rules.ActionAllow, IPVersion: gophercloud.IPv4, SourceIPAddress: "10.0.0.0/24", DestinationPort: "22", Enabled: pythonFWRuleBool(true)}, rules.WithCreateField("x_extension", 1))
		if err != nil || created.ID != "fr-1" || created.Protocol != "tcp" || created.IPVersion != 4 || !created.Enabled || !reflect.DeepEqual(created.FirewallPolicyID, []string{"fp-1"}) {
			t.Fatal(created, err)
		}
		// create_firewall_rule(name="any") relies on server defaults; Go requires protocol and action,
		// and ProtocolAny with ActionDeny states those defaults explicitly (protocol null, action deny).
		if _, err := api.Create(ctx, rules.CreateOpts{Name: "any"}); err == nil {
			t.Fatal("protocol and action are required in Go")
		}
		if _, err := api.Create(ctx, rules.CreateOpts{Name: "any", Protocol: rules.ProtocolAny, Action: rules.ActionDeny}); err != nil {
			t.Fatal(err)
		}
		pythonFWRuleExpect(t, recorder.snapshot(), []pythonFWRuleCall{
			{http.MethodPost, pythonFWRuleBase, "", `{"firewall_rule":{"name":"ssh","protocol":"tcp","action":"allow","ip_version":4,"source_ip_address":"10.0.0.0/24","destination_port":"22","enabled":true,"x_extension":1}}`},
			{http.MethodPost, pythonFWRuleBase, "", `{"firewall_rule":{"name":"any","protocol":null,"action":"deny"}}`},
		})
	})
	t.Run("get_firewall_rule", func(t *testing.T) {
		var recorder pythonFWRuleRecorder
		api := pythonFWRuleAPI(t, &recorder, func(req *http.Request) *http.Response {
			if strings.HasSuffix(req.URL.Path, "/missing") {
				return pythonFWRuleWire(404, `{}`)
			}
			return pythonFWRuleWire(200, `{"firewall_rule":`+pythonFWRuleRow+`}`)
		})
		// get_firewall_rule("fr-1") with fields=None sends no query.
		got, err := api.Get(ctx, "fr-1")
		if err != nil || got.Action != "allow" || got.SourceIPAddress != "10.0.0.0/24" || got.DestinationPort != "22" {
			t.Fatal(got, err)
		}
		if _, err := api.Get(ctx, "missing"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			t.Fatal(err)
		}
		pythonFWRuleExpect(t, recorder.snapshot(), []pythonFWRuleCall{
			{http.MethodGet, pythonFWRuleBase + "/fr-1", "", ""},
			{http.MethodGet, pythonFWRuleBase + "/missing", "", ""},
		})
	})
	t.Run("update_firewall_rule", func(t *testing.T) {
		var recorder pythonFWRuleRecorder
		api := pythonFWRuleAPI(t, &recorder, func(*http.Request) *http.Response {
			return pythonFWRuleWire(200, `{"firewall_rule":`+pythonFWRuleRow+`}`)
		})
		// update_firewall_rule("fr-1", action="deny", enabled=False, destination_port="")
		action := rules.ActionDeny
		updated, err := api.Update(ctx, "fr-1", rules.UpdateOpts{Action: &action, Enabled: pythonFWRuleBool(false), DestinationPort: pythonFWRuleString("")})
		if err != nil || updated.ID != "fr-1" {
			t.Fatal(updated, err)
		}
		pythonFWRuleExpect(t, recorder.snapshot(), []pythonFWRuleCall{
			{http.MethodPut, pythonFWRuleBase + "/fr-1", "", `{"firewall_rule":{"action":"deny","enabled":false,"destination_port":""}}`},
		})
	})
	t.Run("delete_firewall_rule", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			code    int
			options []resource.LookupOption
			check   func(error) bool
		}{
			{"deleted", 204, nil, func(err error) bool { return err == nil }},
			{"ignore_missing=True default", 404, nil, func(err error) bool { return err == nil }},
			{"ignore_missing=False", 404, []resource.LookupOption{resource.WithMissingError()}, func(err error) bool { return errors.Is(err, resource.ErrNotFound) }},
			{"in use conflict is not hidden", 409, nil, func(err error) bool { return gophercloud.ResponseCodeIs(err, http.StatusConflict) }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var recorder pythonFWRuleRecorder
				api := pythonFWRuleAPI(t, &recorder, func(*http.Request) *http.Response { return pythonFWRuleWire(tc.code, `{}`) })
				if err := api.Remove(ctx, resource.ID("fr-1"), tc.options...); !tc.check(err) {
					t.Fatal(err)
				}
				pythonFWRuleExpect(t, recorder.snapshot(), []pythonFWRuleCall{{http.MethodDelete, pythonFWRuleBase + "/fr-1", "", ""}})
			})
		}
	})
	t.Run("firewall_rules", func(t *testing.T) {
		var recorder pythonFWRuleRecorder
		api := pythonFWRuleAPI(t, &recorder, func(*http.Request) *http.Response {
			return pythonFWRuleWire(200, `{"firewall_rules":[`+pythonFWRuleRow+`]}`)
		})
		// firewall_rules(protocol="tcp", action="allow", ip_version=4, firewall_policy_id="fp-1", enabled=True)
		var ids []string
		for value, err := range api.List(ctx, rules.WithListOptions(rules.ListOpts{Protocol: rules.ProtocolTCP, Action: rules.ActionAllow, IPVersion: 4, FirewallPolicyID: "fp-1"}), rules.WithListQuery("enabled", "True")) {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, value.ID)
		}
		if !reflect.DeepEqual(ids, []string{"fr-1"}) {
			t.Fatal(ids)
		}
		// firewall_rules() sends no query.
		if all, err := api.All(ctx); err != nil || len(all) != 1 {
			t.Fatal(all, err)
		}
		pythonFWRuleExpect(t, recorder.snapshot(), []pythonFWRuleCall{
			{http.MethodGet, pythonFWRuleBase, "action=allow&enabled=True&firewall_policy_id=fp-1&ip_version=4&protocol=tcp", ""},
			{http.MethodGet, pythonFWRuleBase, "", ""},
		})
	})
}

// find_firewall_rule tries GET by ID, then lists with name=<value>; ignore_missing defaults to True.
func TestPythonFirewallRuleFindIDThenName(t *testing.T) {
	ctx := context.Background()
	find := func(api *rules.API, value string, options ...resource.LookupOption) (*rules.Rule, error) {
		found, err := api.Find(ctx, resource.ID(value), resource.WithIgnoreMissing())
		if err != nil && !gophercloud.ResponseCodeIs(err, http.StatusBadRequest) && !gophercloud.ResponseCodeIs(err, http.StatusForbidden) {
			return nil, err
		}
		if found != nil {
			return found, nil
		}
		return api.Find(ctx, resource.Name(value), options...)
	}
	listed := `{"firewall_rules":[` + pythonFWRuleRow + `]}`
	for _, tc := range []struct {
		name     string
		getCode  int
		listBody string
		options  []resource.LookupOption
		wantID   string
		check    func(error) bool
		calls    int
	}{
		{"id hit", 200, "", nil, "fr-1", nil, 1},
		{"name fallback after 404", 404, listed, []resource.LookupOption{resource.WithIgnoreMissing()}, "fr-1", nil, 2},
		// Python also falls back after 400 and 403; Go Find(ID) reports them, so the caller continues explicitly.
		{"name fallback after 400", 400, listed, []resource.LookupOption{resource.WithIgnoreMissing()}, "fr-1", nil, 2},
		{"name fallback after 403", 403, listed, []resource.LookupOption{resource.WithIgnoreMissing()}, "fr-1", nil, 2},
		{"ignore_missing=True returns nil", 404, `{"firewall_rules":[]}`, []resource.LookupOption{resource.WithIgnoreMissing()}, "", nil, 2},
		{"ignore_missing=False raises", 404, `{"firewall_rules":[]}`, nil, "", func(err error) bool { return errors.Is(err, resource.ErrNotFound) }, 2},
		{"duplicate names", 404, `{"firewall_rules":[{"id":"a","name":"ssh"},{"id":"b","name":"ssh"}]}`, nil, "", func(err error) bool { return errors.Is(err, resource.ErrAmbiguous) }, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var recorder pythonFWRuleRecorder
			api := pythonFWRuleAPI(t, &recorder, func(req *http.Request) *http.Response {
				if req.URL.Path == pythonFWRuleBase {
					return pythonFWRuleWire(200, tc.listBody)
				}
				if tc.getCode != 200 {
					return pythonFWRuleWire(tc.getCode, `{}`)
				}
				return pythonFWRuleWire(200, `{"firewall_rule":`+pythonFWRuleRow+`}`)
			})
			value := "fr-1"
			if tc.getCode != 200 {
				value = "ssh"
			}
			found, err := find(api, value, tc.options...)
			if tc.check != nil {
				if !tc.check(err) || found != nil {
					t.Fatal(found, err)
				}
			} else if err != nil || (tc.wantID == "") != (found == nil) || (found != nil && found.ID != tc.wantID) {
				t.Fatal(found, err)
			}
			want := []pythonFWRuleCall{{http.MethodGet, pythonFWRuleBase + "/" + value, "", ""}}
			if tc.calls == 2 {
				want = append(want, pythonFWRuleCall{http.MethodGet, pythonFWRuleBase, "name=" + value, ""})
			}
			pythonFWRuleExpect(t, recorder.snapshot(), want)
		})
	}
}
