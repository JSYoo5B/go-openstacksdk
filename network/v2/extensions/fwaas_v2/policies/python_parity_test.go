package policies_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonFWPolicyTransport func(*http.Request) (*http.Response, error)

func (transport pythonFWPolicyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonFWPolicyWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type pythonFWPolicyCall struct{ method, path, query, body string }

type pythonFWPolicyRecorder struct {
	mu    sync.Mutex
	calls []pythonFWPolicyCall
}

func (r *pythonFWPolicyRecorder) snapshot() []pythonFWPolicyCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]pythonFWPolicyCall(nil), r.calls...)
}

func pythonFWPolicyAPI(t *testing.T, recorder *pythonFWPolicyRecorder, reply func(*http.Request) *http.Response) *policies.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonFWPolicyTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		recorder.mu.Lock()
		recorder.calls = append(recorder.calls, pythonFWPolicyCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		recorder.mu.Unlock()
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return policies.New(client)
}

// pythonFWPolicyExpect compares bodies as JSON so map key order does not matter.
func pythonFWPolicyExpect(t *testing.T, got, want []pythonFWPolicyCall) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("calls\n got %+v\nwant %+v", got, want)
	}
	for i := range got {
		g, w := got[i], want[i]
		if g.method != w.method || g.path != w.path || g.query != w.query || !pythonFWPolicySameJSON(g.body, w.body) {
			t.Fatalf("call %d\n got %+v\nwant %+v", i, g, w)
		}
	}
}

func pythonFWPolicySameJSON(a, b string) bool {
	if a == "" || b == "" {
		return a == b
	}
	var x, y any
	if json.Unmarshal([]byte(a), &x) != nil || json.Unmarshal([]byte(b), &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

func pythonFWPolicyString(v string) *string { return &v }
func pythonFWPolicyBool(v bool) *bool       { return &v }

const pythonFWPolicyBase = "/neutron/v2.0/fwaas/firewall_policies"

const pythonFWPolicyRow = `{"id":"fp-1","name":"web","description":"d","audited":false,"shared":true,"firewall_rules":["r1","r2"],"project_id":"proj"}`

// FirewallPolicy is a plain Resource at /fwaas/firewall_policies with resource_key firewall_policy.
func TestPythonFirewallPolicyProxyParity(t *testing.T) {
	ctx := context.Background()
	t.Run("create_firewall_policy", func(t *testing.T) {
		var recorder pythonFWPolicyRecorder
		api := pythonFWPolicyAPI(t, &recorder, func(*http.Request) *http.Response {
			return pythonFWPolicyWire(201, `{"firewall_policy":`+pythonFWPolicyRow+`}`)
		})
		// create_firewall_policy(name="web", shared=True, audited=False, firewall_rules=["r1","r2"], x_extension=1)
		created, err := api.Create(ctx, policies.CreateOpts{Name: "web", Shared: pythonFWPolicyBool(true), Audited: pythonFWPolicyBool(false), FirewallRules: []string{"r1", "r2"}}, policies.WithCreateField("x_extension", 1))
		if err != nil || created.ID != "fp-1" || !created.Shared || created.Audited || !reflect.DeepEqual(created.Rules, []string{"r1", "r2"}) {
			t.Fatal(created, err)
		}
		pythonFWPolicyExpect(t, recorder.snapshot(), []pythonFWPolicyCall{
			{http.MethodPost, pythonFWPolicyBase, "", `{"firewall_policy":{"name":"web","shared":true,"audited":false,"firewall_rules":["r1","r2"],"x_extension":1}}`},
		})
	})
	t.Run("get_firewall_policy", func(t *testing.T) {
		var recorder pythonFWPolicyRecorder
		api := pythonFWPolicyAPI(t, &recorder, func(req *http.Request) *http.Response {
			if strings.HasSuffix(req.URL.Path, "/missing") {
				return pythonFWPolicyWire(404, `{}`)
			}
			return pythonFWPolicyWire(200, `{"firewall_policy":`+pythonFWPolicyRow+`}`)
		})
		// get_firewall_policy("fp-1") with fields=None sends no query.
		got, err := api.Get(ctx, "fp-1")
		if err != nil || got.Name != "web" || got.ProjectID != "proj" || len(got.Rules) != 2 {
			t.Fatal(got, err)
		}
		if _, err := api.Get(ctx, "missing"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			t.Fatal(err)
		}
		pythonFWPolicyExpect(t, recorder.snapshot(), []pythonFWPolicyCall{
			{http.MethodGet, pythonFWPolicyBase + "/fp-1", "", ""},
			{http.MethodGet, pythonFWPolicyBase + "/missing", "", ""},
		})
	})
	t.Run("update_firewall_policy", func(t *testing.T) {
		var recorder pythonFWPolicyRecorder
		api := pythonFWPolicyAPI(t, &recorder, func(*http.Request) *http.Response {
			return pythonFWPolicyWire(200, `{"firewall_policy":`+pythonFWPolicyRow+`}`)
		})
		// update_firewall_policy("fp-1", description="", audited=True, firewall_rules=[])
		rules := []string{}
		updated, err := api.Update(ctx, "fp-1", policies.UpdateOpts{Description: pythonFWPolicyString(""), Audited: pythonFWPolicyBool(true), FirewallRules: &rules})
		if err != nil || updated.ID != "fp-1" {
			t.Fatal(updated, err)
		}
		pythonFWPolicyExpect(t, recorder.snapshot(), []pythonFWPolicyCall{
			{http.MethodPut, pythonFWPolicyBase + "/fp-1", "", `{"firewall_policy":{"description":"","audited":true,"firewall_rules":[]}}`},
		})
	})
	t.Run("delete_firewall_policy", func(t *testing.T) {
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
				var recorder pythonFWPolicyRecorder
				api := pythonFWPolicyAPI(t, &recorder, func(*http.Request) *http.Response { return pythonFWPolicyWire(tc.code, `{}`) })
				if err := api.Remove(ctx, resource.ID("fp-1"), tc.options...); !tc.check(err) {
					t.Fatal(err)
				}
				pythonFWPolicyExpect(t, recorder.snapshot(), []pythonFWPolicyCall{{http.MethodDelete, pythonFWPolicyBase + "/fp-1", "", ""}})
			})
		}
	})
	t.Run("firewall_policies", func(t *testing.T) {
		var recorder pythonFWPolicyRecorder
		api := pythonFWPolicyAPI(t, &recorder, func(*http.Request) *http.Response {
			return pythonFWPolicyWire(200, `{"firewall_policies":[`+pythonFWPolicyRow+`]}`)
		})
		// firewall_policies(name="web", project_id="proj", shared=True, firewall_rules="r1", fields="id")
		var ids []string
		for value, err := range api.List(ctx, policies.WithListOptions(policies.ListOpts{Name: "web", ProjectID: "proj"}), policies.WithListQuery("shared", "True"), policies.WithListQuery("firewall_rules", "r1"), policies.WithListQuery("fields", "id")) {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, value.ID)
		}
		if !reflect.DeepEqual(ids, []string{"fp-1"}) {
			t.Fatal(ids)
		}
		// firewall_policies() sends no query.
		if all, err := api.All(ctx); err != nil || len(all) != 1 {
			t.Fatal(all, err)
		}
		pythonFWPolicyExpect(t, recorder.snapshot(), []pythonFWPolicyCall{
			{http.MethodGet, pythonFWPolicyBase, "fields=id&firewall_rules=r1&name=web&project_id=proj&shared=True", ""},
			{http.MethodGet, pythonFWPolicyBase, "", ""},
		})
	})
}

// insert_rule_into_policy and remove_rule_from_policy PUT an unwrapped body and read an unwrapped policy.
func TestPythonFirewallPolicyRuleActions(t *testing.T) {
	ctx := context.Background()
	t.Run("insert_rule_into_policy", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			opts policies.InsertRuleOpts
			body string
		}{
			// Python always sends insert_after/insert_before, as null when omitted; Go omits the unused one.
			{"insert_after", policies.InsertRuleOpts{ID: "r3", InsertAfter: "r1"}, `{"firewall_rule_id":"r3","insert_after":"r1"}`},
			{"insert_before", policies.InsertRuleOpts{ID: "r3", InsertBefore: "r2"}, `{"firewall_rule_id":"r3","insert_before":"r2"}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var recorder pythonFWPolicyRecorder
				api := pythonFWPolicyAPI(t, &recorder, func(*http.Request) *http.Response {
					return pythonFWPolicyWire(200, `{"id":"fp-1","name":"web","firewall_rules":["r1","r3","r2"]}`)
				})
				policy, err := api.InsertRule(ctx, "fp-1", tc.opts)
				if err != nil || policy.ID != "fp-1" || !reflect.DeepEqual(policy.Rules, []string{"r1", "r3", "r2"}) {
					t.Fatal(policy, err)
				}
				pythonFWPolicyExpect(t, recorder.snapshot(), []pythonFWPolicyCall{{http.MethodPut, pythonFWPolicyBase + "/fp-1/insert_rule", "", tc.body}})
			})
		}
		// Python's default call sends both positions as null; Go requires exactly one position
		// and rejects a null extension for the core insert_before key.
		for name, opts := range map[string]policies.InsertRuleOpts{
			"no position rejected before HTTP":    {ID: "r3"},
			"both positions rejected before HTTP": {ID: "r3", InsertAfter: "r1", InsertBefore: "r2"},
		} {
			t.Run(name, func(t *testing.T) {
				var recorder pythonFWPolicyRecorder
				api := pythonFWPolicyAPI(t, &recorder, func(*http.Request) *http.Response { return pythonFWPolicyWire(200, `{}`) })
				if _, err := api.InsertRule(ctx, "fp-1", opts); err == nil || len(recorder.snapshot()) != 0 {
					t.Fatal(err, recorder.snapshot())
				}
				if _, err := api.InsertRule(ctx, "fp-1", opts, policies.WithInsertRuleField("insert_before", nil)); err == nil || len(recorder.snapshot()) != 0 {
					t.Fatal(err, recorder.snapshot())
				}
			})
		}
		t.Run("error raises", func(t *testing.T) {
			var recorder pythonFWPolicyRecorder
			api := pythonFWPolicyAPI(t, &recorder, func(*http.Request) *http.Response {
				return pythonFWPolicyWire(404, `{"NeutronError":{"message":"Firewall rule r9 could not be found."}}`)
			})
			if _, err := api.InsertRule(ctx, "fp-1", policies.InsertRuleOpts{ID: "r9", InsertAfter: "r1"}); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
				t.Fatal(err)
			}
		})
	})
	t.Run("remove_rule_from_policy", func(t *testing.T) {
		var recorder pythonFWPolicyRecorder
		api := pythonFWPolicyAPI(t, &recorder, func(req *http.Request) *http.Response {
			if strings.Contains(req.URL.Path, "fp-x") {
				return pythonFWPolicyWire(400, `{"NeutronError":{"message":"not associated"}}`)
			}
			return pythonFWPolicyWire(200, `{"id":"fp-1","firewall_rules":["r2"]}`)
		})
		policy, err := api.RemoveRule(ctx, "fp-1", "r1")
		if err != nil || !reflect.DeepEqual(policy.Rules, []string{"r2"}) {
			t.Fatal(policy, err)
		}
		if _, err := api.RemoveRule(ctx, "fp-x", "r1"); !gophercloud.ResponseCodeIs(err, http.StatusBadRequest) {
			t.Fatal(err)
		}
		pythonFWPolicyExpect(t, recorder.snapshot(), []pythonFWPolicyCall{
			{http.MethodPut, pythonFWPolicyBase + "/fp-1/remove_rule", "", `{"firewall_rule_id":"r1"}`},
			{http.MethodPut, pythonFWPolicyBase + "/fp-x/remove_rule", "", `{"firewall_rule_id":"r1"}`},
		})
	})
}

// find_firewall_policy tries GET by ID, then lists with name=<value>; ignore_missing defaults to True.
func TestPythonFirewallPolicyFindIDThenName(t *testing.T) {
	ctx := context.Background()
	find := func(api *policies.API, value string, options ...resource.LookupOption) (*policies.Policy, error) {
		found, err := api.Find(ctx, resource.ID(value), resource.WithIgnoreMissing())
		if err != nil && !gophercloud.ResponseCodeIs(err, http.StatusBadRequest) && !gophercloud.ResponseCodeIs(err, http.StatusForbidden) {
			return nil, err
		}
		if found != nil {
			return found, nil
		}
		return api.Find(ctx, resource.Name(value), options...)
	}
	listed := `{"firewall_policies":[` + pythonFWPolicyRow + `]}`
	for _, tc := range []struct {
		name     string
		getCode  int
		listBody string
		options  []resource.LookupOption
		wantID   string
		check    func(error) bool
		calls    int
	}{
		{"id hit", 200, "", nil, "fp-1", nil, 1},
		{"name fallback after 404", 404, listed, []resource.LookupOption{resource.WithIgnoreMissing()}, "fp-1", nil, 2},
		// Python also falls back after 400 and 403; Go Find(ID) reports them, so the caller continues explicitly.
		{"name fallback after 400", 400, listed, []resource.LookupOption{resource.WithIgnoreMissing()}, "fp-1", nil, 2},
		{"name fallback after 403", 403, listed, []resource.LookupOption{resource.WithIgnoreMissing()}, "fp-1", nil, 2},
		{"ignore_missing=True returns nil", 404, `{"firewall_policies":[]}`, []resource.LookupOption{resource.WithIgnoreMissing()}, "", nil, 2},
		{"ignore_missing=False raises", 404, `{"firewall_policies":[]}`, nil, "", func(err error) bool { return errors.Is(err, resource.ErrNotFound) }, 2},
		{"duplicate names", 404, `{"firewall_policies":[{"id":"a","name":"web"},{"id":"b","name":"web"}]}`, nil, "", func(err error) bool { return errors.Is(err, resource.ErrAmbiguous) }, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var recorder pythonFWPolicyRecorder
			api := pythonFWPolicyAPI(t, &recorder, func(req *http.Request) *http.Response {
				if req.URL.Path == pythonFWPolicyBase {
					return pythonFWPolicyWire(200, tc.listBody)
				}
				if tc.getCode != 200 {
					return pythonFWPolicyWire(tc.getCode, `{}`)
				}
				return pythonFWPolicyWire(200, `{"firewall_policy":`+pythonFWPolicyRow+`}`)
			})
			value := "fp-1"
			if tc.getCode != 200 {
				value = "web"
			}
			found, err := find(api, value, tc.options...)
			if tc.check != nil {
				if !tc.check(err) || found != nil {
					t.Fatal(found, err)
				}
			} else if err != nil || (tc.wantID == "") != (found == nil) || (found != nil && found.ID != tc.wantID) {
				t.Fatal(found, err)
			}
			want := []pythonFWPolicyCall{{http.MethodGet, pythonFWPolicyBase + "/" + value, "", ""}}
			if tc.calls == 2 {
				want = append(want, pythonFWPolicyCall{http.MethodGet, pythonFWPolicyBase, "name=" + value, ""})
			}
			pythonFWPolicyExpect(t, recorder.snapshot(), want)
		})
	}
}
