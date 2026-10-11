package groups_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonFWGroupTransport func(*http.Request) (*http.Response, error)

func (transport pythonFWGroupTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonFWGroupWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type pythonFWGroupCall struct{ method, path, query, body string }

type pythonFWGroupRecorder struct {
	mu    sync.Mutex
	calls []pythonFWGroupCall
}

func (r *pythonFWGroupRecorder) snapshot() []pythonFWGroupCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]pythonFWGroupCall(nil), r.calls...)
}

func pythonFWGroupAPI(t *testing.T, recorder *pythonFWGroupRecorder, reply func(*http.Request) *http.Response) *groups.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonFWGroupTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		recorder.mu.Lock()
		recorder.calls = append(recorder.calls, pythonFWGroupCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		recorder.mu.Unlock()
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return groups.New(client)
}

// pythonFWGroupExpect compares bodies as JSON so map key order does not matter.
func pythonFWGroupExpect(t *testing.T, got, want []pythonFWGroupCall) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("calls\n got %+v\nwant %+v", got, want)
	}
	for i := range got {
		g, w := got[i], want[i]
		if g.method != w.method || g.path != w.path || g.query != w.query || !pythonFWGroupSameJSON(g.body, w.body) {
			t.Fatalf("call %d\n got %+v\nwant %+v", i, g, w)
		}
	}
}

func pythonFWGroupSameJSON(a, b string) bool {
	if a == "" || b == "" {
		return a == b
	}
	var x, y any
	if json.Unmarshal([]byte(a), &x) != nil || json.Unmarshal([]byte(b), &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

func pythonFWGroupString(v string) *string { return &v }
func pythonFWGroupBool(v bool) *bool       { return &v }

const pythonFWGroupBase = "/neutron/v2.0/fwaas/firewall_groups"

const pythonFWGroupRow = `{"id":"fg-1","name":"edge","description":"d","ingress_firewall_policy_id":"in","egress_firewall_policy_id":null,"admin_state_up":false,"ports":["p1"],"status":"ACTIVE","shared":false,"project_id":"proj"}`

// FirewallGroup is a plain Resource at /fwaas/firewall_groups with resource_key firewall_group.
func TestPythonFirewallGroupProxyParity(t *testing.T) {
	ctx := context.Background()
	t.Run("create_firewall_group", func(t *testing.T) {
		var recorder pythonFWGroupRecorder
		api := pythonFWGroupAPI(t, &recorder, func(*http.Request) *http.Response {
			return pythonFWGroupWire(201, `{"firewall_group":`+pythonFWGroupRow+`}`)
		})
		// create_firewall_group(name="edge", admin_state_up=False, ports=["p1"], ingress_firewall_policy_id="in", x_extension=1)
		created, err := api.Create(ctx, groups.CreateOpts{Name: "edge", AdminStateUp: pythonFWGroupBool(false), Ports: []string{"p1"}, IngressFirewallPolicyID: "in"}, groups.WithCreateField("x_extension", 1))
		if err != nil || created.ID != "fg-1" || created.Status != "ACTIVE" || created.AdminStateUp || created.ProjectID != "proj" || !reflect.DeepEqual(created.Ports, []string{"p1"}) {
			t.Fatal(created, err)
		}
		pythonFWGroupExpect(t, recorder.snapshot(), []pythonFWGroupCall{
			{http.MethodPost, pythonFWGroupBase, "", `{"firewall_group":{"name":"edge","admin_state_up":false,"ports":["p1"],"ingress_firewall_policy_id":"in","x_extension":1}}`},
		})
	})
	t.Run("get_firewall_group", func(t *testing.T) {
		var recorder pythonFWGroupRecorder
		api := pythonFWGroupAPI(t, &recorder, func(req *http.Request) *http.Response {
			if strings.HasSuffix(req.URL.Path, "/missing") {
				return pythonFWGroupWire(404, `{"NeutronError":{"type":"FirewallGroupNotFound"}}`)
			}
			return pythonFWGroupWire(200, `{"firewall_group":`+pythonFWGroupRow+`}`)
		})
		// get_firewall_group("fg-1") with fields=None sends no query.
		got, err := api.Get(ctx, "fg-1")
		if err != nil || got.Name != "edge" || got.IngressFirewallPolicyID != "in" || got.EgressFirewallPolicyID != "" {
			t.Fatal(got, err)
		}
		// Python raises NotFoundException; Go returns the 404.
		if _, err := api.Get(ctx, "missing"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			t.Fatal(err)
		}
		pythonFWGroupExpect(t, recorder.snapshot(), []pythonFWGroupCall{
			{http.MethodGet, pythonFWGroupBase + "/fg-1", "", ""},
			{http.MethodGet, pythonFWGroupBase + "/missing", "", ""},
		})
	})
	t.Run("update_firewall_group", func(t *testing.T) {
		var recorder pythonFWGroupRecorder
		api := pythonFWGroupAPI(t, &recorder, func(*http.Request) *http.Response {
			return pythonFWGroupWire(200, `{"firewall_group":`+pythonFWGroupRow+`}`)
		})
		// update_firewall_group("fg-1", name="edge", description="", admin_state_up=True, ports=[])
		ports := []string{}
		updated, err := api.Update(ctx, "fg-1", groups.UpdateOpts{Name: pythonFWGroupString("edge"), Description: pythonFWGroupString(""), AdminStateUp: pythonFWGroupBool(true), Ports: &ports})
		if err != nil || updated.ID != "fg-1" {
			t.Fatal(updated, err)
		}
		// update_firewall_group("fg-1", ingress_firewall_policy_id=None) and the egress twin send JSON null.
		if _, err := api.RemoveIngressPolicy(ctx, "fg-1"); err != nil {
			t.Fatal(err)
		}
		if _, err := api.RemoveEgressPolicy(ctx, "fg-1"); err != nil {
			t.Fatal(err)
		}
		pythonFWGroupExpect(t, recorder.snapshot(), []pythonFWGroupCall{
			{http.MethodPut, pythonFWGroupBase + "/fg-1", "", `{"firewall_group":{"name":"edge","description":"","admin_state_up":true,"ports":[]}}`},
			{http.MethodPut, pythonFWGroupBase + "/fg-1", "", `{"firewall_group":{"ingress_firewall_policy_id":null}}`},
			{http.MethodPut, pythonFWGroupBase + "/fg-1", "", `{"firewall_group":{"egress_firewall_policy_id":null}}`},
		})
	})
	t.Run("delete_firewall_group", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			code    int
			options []resource.LookupOption
			check   func(error) bool
		}{
			{"deleted", 204, nil, func(err error) bool { return err == nil }},
			{"ignore_missing=True default", 404, nil, func(err error) bool { return err == nil }},
			{"ignore_missing=False", 404, []resource.LookupOption{resource.WithMissingError()}, func(err error) bool { return errors.Is(err, resource.ErrNotFound) }},
			{"conflict is not hidden", 409, nil, func(err error) bool { return gophercloud.ResponseCodeIs(err, http.StatusConflict) }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var recorder pythonFWGroupRecorder
				api := pythonFWGroupAPI(t, &recorder, func(*http.Request) *http.Response { return pythonFWGroupWire(tc.code, `{}`) })
				if err := api.Remove(ctx, resource.ID("fg-1"), tc.options...); !tc.check(err) {
					t.Fatal(err)
				}
				pythonFWGroupExpect(t, recorder.snapshot(), []pythonFWGroupCall{{http.MethodDelete, pythonFWGroupBase + "/fg-1", "", ""}})
			})
		}
	})
	t.Run("firewall_groups", func(t *testing.T) {
		var recorder pythonFWGroupRecorder
		var api *groups.API
		api = pythonFWGroupAPI(t, &recorder, func(req *http.Request) *http.Response {
			if req.URL.Query().Get("marker") == "fg-1" {
				return pythonFWGroupWire(200, `{"firewall_groups":[{"id":"fg-2","name":"edge"}]}`)
			}
			next := api.RawClient().ResourceBase + "fwaas/firewall_groups?limit=1&marker=fg-1&name=edge"
			return pythonFWGroupWire(200, `{"firewall_groups":[`+pythonFWGroupRow+`],"firewall_groups_links":[{"rel":"next","href":"`+next+`"}]}`)
		})
		// firewall_groups(name="edge", status="ACTIVE", project_id="proj", shared=False, ports=["p1","p2"], fields="id", limit=1)
		ports := []string{"p1", "p2"}
		var ids []string
		for value, err := range api.List(ctx, groups.WithListOptions(groups.ListOpts{Name: "edge", Status: "ACTIVE", ProjectID: "proj", Ports: &ports, Limit: 1}), groups.WithListQuery("shared", "False"), groups.WithListQuery("fields", "id")) {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, value.ID)
		}
		if !reflect.DeepEqual(ids, []string{"fg-1", "fg-2"}) {
			t.Fatal(ids)
		}
		// firewall_groups() sends no query.
		all, err := api.All(ctx)
		if err != nil || len(all) != 2 {
			t.Fatal(all, err)
		}
		pythonFWGroupExpect(t, recorder.snapshot(), []pythonFWGroupCall{
			{http.MethodGet, pythonFWGroupBase, "fields=id&limit=1&name=edge&ports=p1&ports=p2&project_id=proj&shared=False&status=ACTIVE", ""},
			{http.MethodGet, pythonFWGroupBase, "limit=1&marker=fg-1&name=edge", ""},
			{http.MethodGet, pythonFWGroupBase, "", ""},
			{http.MethodGet, pythonFWGroupBase, "limit=1&marker=fg-1&name=edge", ""},
		})
	})
}

// find_firewall_group tries GET by ID, then lists with name=<value>; ignore_missing defaults to True.
func TestPythonFirewallGroupFindIDThenName(t *testing.T) {
	ctx := context.Background()
	find := func(api *groups.API, value string, options ...resource.LookupOption) (*groups.Group, error) {
		found, err := api.Find(ctx, resource.ID(value), resource.WithIgnoreMissing())
		if err != nil && !gophercloud.ResponseCodeIs(err, http.StatusBadRequest) && !gophercloud.ResponseCodeIs(err, http.StatusForbidden) {
			return nil, err
		}
		if found != nil {
			return found, nil
		}
		return api.Find(ctx, resource.Name(value), options...)
	}
	listed := `{"firewall_groups":[` + pythonFWGroupRow + `]}`
	for _, tc := range []struct {
		name     string
		getCode  int
		listBody string
		options  []resource.LookupOption
		wantID   string
		check    func(error) bool
		calls    int
	}{
		{"id hit", 200, "", nil, "fg-1", nil, 1},
		{"name fallback after 404", 404, listed, []resource.LookupOption{resource.WithIgnoreMissing()}, "fg-1", nil, 2},
		// Python also falls back after 400 and 403; Go Find(ID) reports them, so the caller continues explicitly.
		{"name fallback after 400", 400, listed, []resource.LookupOption{resource.WithIgnoreMissing()}, "fg-1", nil, 2},
		{"name fallback after 403", 403, listed, []resource.LookupOption{resource.WithIgnoreMissing()}, "fg-1", nil, 2},
		{"ignore_missing=True returns nil", 404, `{"firewall_groups":[]}`, []resource.LookupOption{resource.WithIgnoreMissing()}, "", nil, 2},
		{"ignore_missing=False raises", 404, `{"firewall_groups":[]}`, nil, "", func(err error) bool { return errors.Is(err, resource.ErrNotFound) }, 2},
		{"duplicate names", 404, `{"firewall_groups":[{"id":"a","name":"edge"},{"id":"b","name":"edge"}]}`, nil, "", func(err error) bool { return errors.Is(err, resource.ErrAmbiguous) }, 2},
		{"server error stops", 500, "", nil, "", func(err error) bool { return gophercloud.ResponseCodeIs(err, http.StatusInternalServerError) }, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var recorder pythonFWGroupRecorder
			api := pythonFWGroupAPI(t, &recorder, func(req *http.Request) *http.Response {
				if req.URL.Path == pythonFWGroupBase {
					return pythonFWGroupWire(200, tc.listBody)
				}
				if tc.getCode != 200 {
					return pythonFWGroupWire(tc.getCode, `{}`)
				}
				return pythonFWGroupWire(200, `{"firewall_group":`+pythonFWGroupRow+`}`)
			})
			value := "fg-1"
			if tc.getCode != 200 {
				value = "edge"
			}
			found, err := find(api, value, tc.options...)
			if tc.check != nil {
				if !tc.check(err) || found != nil {
					t.Fatal(found, err)
				}
			} else if err != nil || (tc.wantID == "") != (found == nil) || (found != nil && found.ID != tc.wantID) {
				t.Fatal(found, err)
			}
			want := []pythonFWGroupCall{{http.MethodGet, pythonFWGroupBase + "/" + value, "", ""}}
			if tc.calls == 2 {
				want = append(want, pythonFWGroupCall{http.MethodGet, pythonFWGroupBase, "name=" + value, ""})
			}
			pythonFWGroupExpect(t, recorder.snapshot(), want)
		})
	}
}
