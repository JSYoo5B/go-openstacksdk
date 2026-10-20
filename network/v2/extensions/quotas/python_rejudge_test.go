package quotas_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonRejudgeTransport func(*http.Request) (*http.Response, error)

func (transport pythonRejudgeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonRejudgeCall struct{ method, path, query, body string }

type pythonRejudgeReply struct {
	code int
	body string
}

const pythonRejudgeRoot = "/neutron/v2.0"

func pythonRejudgeAPI(t *testing.T, calls *[]pythonRejudgeCall, replies map[string]pythonRejudgeReply) *quotas.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonRejudgeTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		path := strings.TrimPrefix(req.URL.Path, pythonRejudgeRoot)
		*calls = append(*calls, pythonRejudgeCall{req.Method, path, req.URL.RawQuery, raw})
		reply, ok := replies[req.Method+" "+path]
		if !ok {
			reply = pythonRejudgeReply{404, `{"NeutronError":{"type":"NotFound","message":"not found"}}`}
		}
		return &http.Response{StatusCode: reply.code, Body: io.NopCloser(strings.NewReader(reply.body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + pythonRejudgeRoot + "/"
	return quotas.New(client)
}

func pythonRejudgeScope(t *testing.T, api *quotas.API) *quotas.ProjectQuotaScope {
	t.Helper()
	scope, err := api.InProject(context.Background(), resource.ID("proj"))
	if err != nil || scope.ProjectID() != "proj" {
		t.Fatal(scope, err)
	}
	return scope
}

func pythonRejudgeExpect(t *testing.T, calls []pythonRejudgeCall, want ...pythonRejudgeCall) {
	t.Helper()
	if len(calls) != len(want) {
		t.Fatalf("calls\n got %+v\nwant %+v", calls, want)
	}
	for i := range want {
		got := calls[i]
		if got.method != want[i].method || got.path != want[i].path || got.query != want[i].query {
			t.Fatalf("call %d\n got %+v\nwant %+v", i, got, want[i])
		}
		if want[i].body == "" {
			if got.body != "" {
				t.Fatalf("call %d body %q", i, got.body)
			}
			continue
		}
		var gotBody, wantBody any
		if err := json.Unmarshal([]byte(got.body), &gotBody); err != nil {
			t.Fatal(got.body, err)
		}
		if err := json.Unmarshal([]byte(want[i].body), &wantBody); err != nil {
			t.Fatal(want[i].body, err)
		}
		if !reflect.DeepEqual(gotBody, wantBody) {
			t.Fatalf("call %d body\n got %s\nwant %s", i, got.body, want[i].body)
		}
	}
}

const pythonRejudgeQuota = `{"network":10,"port":50,"router":-1,"subnet":0,"floatingip":5,"security_group":10,"security_group_rule":100,"subnetpool":-1,"rbac_policy":10,"trunk":-1,"healthmonitor":7,"project_id":"proj"}`

// get_quota(project) and get_quota(project, details=True).
func TestPythonRejudgeQuotaGet(t *testing.T) {
	ctx := context.Background()
	var calls []pythonRejudgeCall
	api := pythonRejudgeAPI(t, &calls, map[string]pythonRejudgeReply{
		"GET /quotas/proj":              {200, `{"quota":` + pythonRejudgeQuota + `}`},
		"GET /quotas/proj/details.json": {200, `{"quota":{"network":{"used":2,"reserved":0,"limit":10},"healthmonitor":{"used":1,"reserved":0,"limit":7}}}`},
	})
	scope := pythonRejudgeScope(t, api)
	got, err := scope.Get(ctx)
	if err != nil || got.ProjectID != "proj" || got.Network != 10 || got.Port != 50 || got.Router != -1 || got.Subnet != 0 || string(got.Body["healthmonitor"]) != "7" {
		t.Fatal(got, err)
	}
	detail, err := scope.Detail(ctx)
	if err != nil || detail.Network != (quotas.QuotaDetail{Used: 2, Reserved: 0, Limit: 10}) || string(detail.Body["healthmonitor"]) != `{"used":1,"reserved":0,"limit":7}` {
		t.Fatal(detail, err)
	}
	// Python QuotaDetails uses /details; the native route adds the .json format suffix.
	pythonRejudgeExpect(t, calls, pythonRejudgeCall{"GET", "/quotas/proj", "", ""}, pythonRejudgeCall{"GET", "/quotas/proj/details.json", "", ""})

	calls = nil
	if _, err := scope.Get(ctx); err != nil {
		t.Fatal(err)
	}
	other, err := api.InProject(ctx, resource.ID("ghost"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Get(ctx); !errors.Is(err, resource.ErrNotFound) || !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	pythonRejudgeExpect(t, calls, pythonRejudgeCall{"GET", "/quotas/proj", "", ""}, pythonRejudgeCall{"GET", "/quotas/ghost", "", ""})
}

// get_quota_default(project).
func TestPythonRejudgeQuotaDefault(t *testing.T) {
	var calls []pythonRejudgeCall
	api := pythonRejudgeAPI(t, &calls, map[string]pythonRejudgeReply{
		"GET /quotas/proj/default": {200, `{"quota":{"network":100,"port":500,"router":10,"subnet":100,"healthmonitor":-1}}`},
	})
	got, err := pythonRejudgeScope(t, api).Defaults(context.Background())
	if err != nil || got.ProjectID != "proj" || got.Network != 100 || got.Port != 500 || got.Router != 10 || string(got.Body["healthmonitor"]) != "-1" {
		t.Fatal(got, err)
	}
	pythonRejudgeExpect(t, calls, pythonRejudgeCall{"GET", "/quotas/proj/default", "", ""})
}

// update_quota(project, networks=10, ports=0, health_monitors=7, check_limit=False)
// PUTs only the given attributes under their wire names.
func TestPythonRejudgeQuotaUpdate(t *testing.T) {
	var calls []pythonRejudgeCall
	api := pythonRejudgeAPI(t, &calls, map[string]pythonRejudgeReply{
		"PUT /quotas/proj": {200, `{"quota":` + pythonRejudgeQuota + `}`},
	})
	scope := pythonRejudgeScope(t, api)
	ten, zero := 10, 0
	got, err := scope.Update(context.Background(), quotas.UpdateOpts{Network: &ten, Port: &zero}, quotas.WithUpdateField("healthmonitor", 7), quotas.WithUpdateCheckLimit(false))
	if err != nil || got.ProjectID != "proj" || got.Network != 10 || string(got.Body["healthmonitor"]) != "7" {
		t.Fatal(got, err)
	}
	if _, err := scope.Update(context.Background(), quotas.UpdateOpts{Network: &ten}); err != nil {
		t.Fatal(err)
	}
	pythonRejudgeExpect(t, calls,
		pythonRejudgeCall{"PUT", "/quotas/proj", "", `{"quota":{"network":10,"port":0,"healthmonitor":7,"check_limit":false}}`},
		pythonRejudgeCall{"PUT", "/quotas/proj", "", `{"quota":{"network":10}}`},
	)
}

// delete_quota(project) defaults to ignore_missing=True.
func TestPythonRejudgeQuotaDelete(t *testing.T) {
	for _, tc := range []struct {
		name    string
		code    int
		options []quotas.DeleteOption
		check   func(*quotas.DeleteResponse, error) bool
	}{
		{"deleted", 204, []quotas.DeleteOption{quotas.WithDeleteIgnoreMissing(true)}, func(r *quotas.DeleteResponse, err error) bool {
			return err == nil && r != nil && r.StatusCode == 204
		}},
		{"404 ignored", 404, []quotas.DeleteOption{quotas.WithDeleteIgnoreMissing(true)}, func(r *quotas.DeleteResponse, err error) bool { return err == nil && r == nil }},
		{"409 returned", 409, []quotas.DeleteOption{quotas.WithDeleteIgnoreMissing(true)}, func(_ *quotas.DeleteResponse, err error) bool {
			return gophercloud.ResponseCodeIs(err, http.StatusConflict)
		}},
		{"ignore_missing=False raises 404", 404, nil, func(_ *quotas.DeleteResponse, err error) bool { return errors.Is(err, resource.ErrNotFound) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonRejudgeCall
			api := pythonRejudgeAPI(t, &calls, map[string]pythonRejudgeReply{"DELETE /quotas/proj": {tc.code, ""}})
			response, err := pythonRejudgeScope(t, api).Delete(context.Background(), tc.options...)
			if !tc.check(response, err) {
				t.Fatal(response, err)
			}
			pythonRejudgeExpect(t, calls, pythonRejudgeCall{"DELETE", "/quotas/proj", "", ""})
		})
	}
}

// quotas() lists GET /quotas once without query.
func TestPythonRejudgeQuotaList(t *testing.T) {
	var calls []pythonRejudgeCall
	api := pythonRejudgeAPI(t, &calls, map[string]pythonRejudgeReply{
		"GET /quotas": {200, `{"quotas":[` + pythonRejudgeQuota + `,{"network":3,"port":4,"project_id":"p2","tenant_id":"p2"}]}`},
	})
	rows, err := api.AllProjects(context.Background())
	if err != nil || len(rows) != 2 || rows[0].ProjectID != "proj" || rows[0].Network != 10 || rows[1].ProjectID != "p2" || rows[1].Port != 4 {
		t.Fatal(rows, err)
	}
	var seen []string
	for row, err := range api.ListProjects(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		seen = append(seen, row.ProjectID)
	}
	if !reflect.DeepEqual(seen, []string{"proj", "p2"}) {
		t.Fatal(seen)
	}
	pythonRejudgeExpect(t, calls, pythonRejudgeCall{"GET", "/quotas", "", ""}, pythonRejudgeCall{"GET", "/quotas", "", ""})
}
