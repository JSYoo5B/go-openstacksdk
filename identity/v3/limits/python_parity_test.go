package limits_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/limits"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonLimitCall struct{ method, path, query, body string }

func pythonLimitAPI(t *testing.T, calls *[]pythonLimitCall, reply func(*http.Request) (int, string)) (*limits.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonLimitCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return limits.New(client), cloud
}

const (
	pythonLimitBase = "/keystone/v3/limits"
	pythonLimitRow  = `{"id":"lim-1","region_id":"RegionOne","project_id":"p","domain_id":null,"service_id":"svc-1","description":"d","resource_name":"cores","resource_limit":10,"links":{"self":"x"}}`
)

var pythonLimitWant = limits.Limit{ID: "lim-1", RegionID: "RegionOne", ProjectID: "p", ServiceID: "svc-1", Description: "d", ResourceName: "cores", ResourceLimit: 10, Links: map[string]any{"self": "x"}}

// Python create_limit wraps the single "limit" body in a one-element "limits"
// array and reads element 0 of the response array.
func TestPythonLimitCreateGetUpdateRequests(t *testing.T) {
	ctx := context.Background()
	var calls []pythonLimitCall
	api, _ := pythonLimitAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.Method {
		case http.MethodPost:
			return 201, `{"limits":[` + pythonLimitRow + `]}`
		case http.MethodPatch:
			return 200, `{"limit":{"id":"lim-1","service_id":"svc-1","resource_name":"cores","resource_limit":0,"description":""}}`
		}
		return 200, `{"limit":` + pythonLimitRow + `}`
	})
	created, err := api.BatchCreate(ctx, limits.BatchCreateOpts{{ServiceID: "svc-1", ProjectID: "p", RegionID: "RegionOne", ResourceName: "cores", ResourceLimit: 10, Description: "d"}})
	if err != nil || len(created) != 1 || !reflect.DeepEqual(created[0], pythonLimitWant) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "lim-1")
	if err != nil || !reflect.DeepEqual(*got, pythonLimitWant) {
		t.Fatal(got, err)
	}
	zero, empty := 0, ""
	updated, err := api.Update(ctx, "lim-1", limits.UpdateOpts{ResourceLimit: &zero, Description: &empty})
	if err != nil || updated.ResourceLimit != 0 || updated.Description != "" {
		t.Fatal(updated, err)
	}
	wantCalls := []pythonLimitCall{
		{http.MethodPost, pythonLimitBase, "", `{"limits":[{"description":"d","project_id":"p","region_id":"RegionOne","resource_limit":10,"resource_name":"cores","service_id":"svc-1"}]}`},
		{http.MethodGet, pythonLimitBase + "/lim-1", "", ""},
		{http.MethodPatch, pythonLimitBase + "/lim-1", "", `{"limit":{"description":"","resource_limit":0}}`},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("%+v", calls)
	}
	// Python sends only the given keys; Go requires service_id and resource_name
	// before HTTP and always sends resource_limit.
	calls = nil
	if _, err := api.BatchCreate(ctx, limits.BatchCreateOpts{{ProjectID: "p", ResourceLimit: 1}}); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonLimitDeleteIgnoreMissing(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		status  int
		options []resource.LookupOption
		check   func(error) bool
	}{
		{"deleted", 204, nil, func(err error) bool { return err == nil }},
		{"missing ignored by default", 404, nil, func(err error) bool { return err == nil }},
		{"ignore_missing=False", 404, []resource.LookupOption{resource.WithMissingError()}, func(err error) bool { return errors.Is(err, resource.ErrNotFound) }},
		{"conflict is returned", 409, nil, func(err error) bool { return gophercloud.ResponseCodeIs(err, http.StatusConflict) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonLimitCall
			api, _ := pythonLimitAPI(t, &calls, func(*http.Request) (int, string) { return tc.status, "" })
			if err := api.Remove(ctx, resource.ID("lim-1"), tc.options...); !tc.check(err) {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []pythonLimitCall{{http.MethodDelete, pythonLimitBase + "/lim-1", "", ""}}) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

// Python limits() sends service_id/region_id/resource_name/project_id/limit/
// marker, filters domain_id locally and ignores the links.next dict; Go sends
// every key as query and follows links.next.
func TestPythonLimitListQueryAndPaging(t *testing.T) {
	ctx := context.Background()
	var calls []pythonLimitCall
	var cloud *testcloud.Cloud
	api, cloud := pythonLimitAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("page") == "2" {
			return 200, `{"limits":[{"id":"lim-2"}],"links":{"next":null}}`
		}
		return 200, `{"limits":[` + pythonLimitRow + `],"links":{"next":"` + cloud.Server.URL + pythonLimitBase + `?page=2"}}`
	})
	rows, err := api.All(ctx, resource.WithQuery("service_id", "svc-1"), resource.WithQuery("project_id", "p"), resource.WithQuery("limit", "1"), resource.WithQuery("domain_id", "d"))
	if err != nil || len(rows) != 2 || rows[0].ID != "lim-1" || rows[1].ID != "lim-2" {
		t.Fatal(rows, err)
	}
	want := []pythonLimitCall{
		{http.MethodGet, pythonLimitBase, "domain_id=d&limit=1&project_id=p&service_id=svc-1", ""},
		{http.MethodGet, pythonLimitBase, "page=2", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}
