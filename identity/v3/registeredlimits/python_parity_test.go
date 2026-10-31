package registeredlimits_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonRegisteredLimitCall struct{ method, path, query, body string }

func pythonRegisteredLimitAPI(t *testing.T, calls *[]pythonRegisteredLimitCall, reply func(*http.Request) (int, string)) (*registeredlimits.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonRegisteredLimitCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return registeredlimits.New(client), cloud
}

const (
	pythonRegisteredLimitBase = "/keystone/v3/registered_limits"
	pythonRegisteredLimitRow  = `{"id":"rl-1","region_id":"RegionOne","service_id":"svc-1","description":"d","resource_name":"cores","default_limit":10,"links":{"self":"x"}}`
)

var pythonRegisteredLimitWant = registeredlimits.RegisteredLimit{ID: "rl-1", RegionID: "RegionOne", ServiceID: "svc-1", Description: "d", ResourceName: "cores", DefaultLimit: 10, Links: map[string]any{"self": "x"}}

// Python create_registered_limit wraps the single body in a one-element
// "registered_limits" array and reads element 0 of the response array.
func TestPythonRegisteredLimitCreateGetUpdateRequests(t *testing.T) {
	ctx := context.Background()
	var calls []pythonRegisteredLimitCall
	api, _ := pythonRegisteredLimitAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.Method {
		case http.MethodPost:
			return 201, `{"registered_limits":[` + pythonRegisteredLimitRow + `]}`
		case http.MethodPatch:
			return 200, `{"registered_limit":{"id":"rl-1","region_id":"RegionTwo","service_id":"svc-1","resource_name":"cores","default_limit":0}}`
		}
		return 200, `{"registered_limit":` + pythonRegisteredLimitRow + `}`
	})
	created, err := api.BatchCreate(ctx, registeredlimits.BatchCreateOpts{{ServiceID: "svc-1", RegionID: "RegionOne", ResourceName: "cores", DefaultLimit: 10, Description: "d"}})
	if err != nil || len(created) != 1 || !reflect.DeepEqual(created[0], pythonRegisteredLimitWant) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "rl-1")
	if err != nil || !reflect.DeepEqual(*got, pythonRegisteredLimitWant) {
		t.Fatal(got, err)
	}
	zero := 0
	updated, err := api.Update(ctx, "rl-1", registeredlimits.UpdateOpts{DefaultLimit: &zero, RegionID: "RegionTwo"})
	if err != nil || updated.DefaultLimit != 0 || updated.RegionID != "RegionTwo" {
		t.Fatal(updated, err)
	}
	wantCalls := []pythonRegisteredLimitCall{
		{http.MethodPost, pythonRegisteredLimitBase, "", `{"registered_limits":[{"default_limit":10,"description":"d","region_id":"RegionOne","resource_name":"cores","service_id":"svc-1"}]}`},
		{http.MethodGet, pythonRegisteredLimitBase + "/rl-1", "", ""},
		{http.MethodPatch, pythonRegisteredLimitBase + "/rl-1", "", `{"registered_limit":{"default_limit":0,"region_id":"RegionTwo"}}`},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("%+v", calls)
	}
	// Python update_registered_limit(region_id=None) sends null; Go omits the
	// empty typed field and rejects the extension key before HTTP.
	calls = nil
	if _, err := api.Update(ctx, "rl-1", registeredlimits.UpdateOpts{}, registeredlimits.WithUpdateField("region_id", nil)); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonRegisteredLimitDeleteIgnoreMissing(t *testing.T) {
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
			var calls []pythonRegisteredLimitCall
			api, _ := pythonRegisteredLimitAPI(t, &calls, func(*http.Request) (int, string) { return tc.status, "" })
			if err := api.Remove(ctx, resource.ID("rl-1"), tc.options...); !tc.check(err) {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []pythonRegisteredLimitCall{{http.MethodDelete, pythonRegisteredLimitBase + "/rl-1", "", ""}}) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

// Python registered_limits() sends service_id/region_id/resource_name/limit/
// marker and ignores the links.next dict; Go forwards the query and follows it.
func TestPythonRegisteredLimitListQueryAndPaging(t *testing.T) {
	ctx := context.Background()
	var calls []pythonRegisteredLimitCall
	var cloud *testcloud.Cloud
	api, cloud := pythonRegisteredLimitAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("page") == "2" {
			return 200, `{"registered_limits":[{"id":"rl-2"}],"links":{"next":null}}`
		}
		return 200, `{"registered_limits":[` + pythonRegisteredLimitRow + `],"links":{"next":"` + cloud.Server.URL + pythonRegisteredLimitBase + `?page=2"}}`
	})
	rows, err := api.All(ctx, resource.WithQuery("service_id", "svc-1"), resource.WithQuery("resource_name", "cores"), resource.WithQuery("marker", "rl-0"))
	if err != nil || len(rows) != 2 || rows[0].ID != "rl-1" || rows[1].ID != "rl-2" {
		t.Fatal(rows, err)
	}
	want := []pythonRegisteredLimitCall{
		{http.MethodGet, pythonRegisteredLimitBase, "marker=rl-0&resource_name=cores&service_id=svc-1", ""},
		{http.MethodGet, pythonRegisteredLimitBase, "page=2", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}
