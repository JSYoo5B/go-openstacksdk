package endpoints_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonEndpointCall struct{ method, path, query, body string }

func pythonEndpointAPI(t *testing.T, calls *[]pythonEndpointCall, reply func(*http.Request) (int, string)) (*endpoints.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonEndpointCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return endpoints.New(client), cloud
}

const (
	pythonEndpointBase = "/keystone/v3/endpoints"
	pythonEndpointRow  = `{"id":"ep-1","interface":"public","region_id":"RegionOne","service_id":"svc-1","url":"https://nova","enabled":false,"links":{"self":"x"}}`
)

// Python create_endpoint/get_endpoint/update_endpoint send only the dirty Body
// attributes in the "endpoint" envelope; region_id is not a typed Go field.
func TestPythonEndpointCreateGetUpdateRequests(t *testing.T) {
	ctx := context.Background()
	var calls []pythonEndpointCall
	api, _ := pythonEndpointAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.Method == http.MethodPost {
			return 201, `{"endpoint":` + pythonEndpointRow + `}`
		}
		if req.Method == http.MethodPatch {
			return 200, `{"endpoint":{"id":"ep-1","interface":"public","region_id":"RegionTwo","service_id":"svc-1","url":"https://nova2","enabled":true}}`
		}
		return 200, `{"endpoint":` + pythonEndpointRow + `}`
	})
	disabled, enabled := false, true
	want := endpoints.Endpoint{ID: "ep-1", Availability: gophercloud.AvailabilityPublic, RegionID: "RegionOne", ServiceID: "svc-1", URL: "https://nova"}
	created, err := api.Create(ctx, endpoints.CreateOpts{Availability: gophercloud.AvailabilityPublic, URL: "https://nova", ServiceID: "svc-1", Enabled: &disabled}, endpoints.WithCreateField("region_id", "RegionOne"))
	if err != nil || !reflect.DeepEqual(*created, want) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "ep-1")
	if err != nil || !reflect.DeepEqual(*got, want) {
		t.Fatal(got, err)
	}
	updated, err := api.Update(ctx, "ep-1", endpoints.UpdateOpts{URL: "https://nova2", Enabled: &enabled}, endpoints.WithUpdateField("region_id", "RegionTwo"))
	if err != nil || updated.RegionID != "RegionTwo" || updated.URL != "https://nova2" || !updated.Enabled {
		t.Fatal(updated, err)
	}
	wantCalls := []pythonEndpointCall{
		{http.MethodPost, pythonEndpointBase, "", `{"endpoint":{"enabled":false,"interface":"public","region_id":"RegionOne","service_id":"svc-1","url":"https://nova"}}`},
		{http.MethodGet, pythonEndpointBase + "/ep-1", "", ""},
		{http.MethodPatch, pythonEndpointBase + "/ep-1", "", `{"endpoint":{"enabled":true,"region_id":"RegionTwo","url":"https://nova2"}}`},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("%+v", calls)
	}
	// Python would POST without url and let Keystone reject it; Go stops first.
	calls = nil
	if _, err := api.Create(ctx, endpoints.CreateOpts{Availability: gophercloud.AvailabilityPublic, ServiceID: "svc-1"}); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonEndpointDeleteIgnoreMissing(t *testing.T) {
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
			var calls []pythonEndpointCall
			api, _ := pythonEndpointAPI(t, &calls, func(*http.Request) (int, string) { return tc.status, "" })
			if err := api.Remove(ctx, resource.ID("ep-1"), tc.options...); !tc.check(err) {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []pythonEndpointCall{{http.MethodDelete, pythonEndpointBase + "/ep-1", "", ""}}) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

// pythonFindEndpoint mirrors Resource.find: GET by ID first, then the list
// matched locally. Endpoint has no name query parameter, so the list is unfiltered.
func pythonFindEndpoint(ctx context.Context, api *endpoints.API, nameOrID string, options ...resource.LookupOption) (*endpoints.Endpoint, error) {
	found, err := api.Find(ctx, resource.ID(nameOrID), resource.WithIgnoreMissing())
	if err != nil || found != nil {
		return found, err
	}
	return api.Find(ctx, resource.Name(nameOrID), options...)
}

func TestPythonEndpointFindIDThenNameFallback(t *testing.T) {
	ctx := context.Background()
	named := `{"id":"ep-1","name":"nova-public","interface":"public"}`
	for _, tc := range []struct {
		name    string
		input   string
		rows    string
		options []resource.LookupOption
		wantID  string
		wantErr error
		calls   int
	}{
		{name: "id hit", input: "ep-1", wantID: "ep-1", calls: 1},
		{name: "name fallback", input: "nova-public", rows: named, options: []resource.LookupOption{resource.WithIgnoreMissing()}, wantID: "ep-1", calls: 2},
		{name: "missing ignored", input: "ghost", rows: named, options: []resource.LookupOption{resource.WithIgnoreMissing()}, calls: 2},
		{name: "missing strict", input: "ghost", rows: named, wantErr: resource.ErrNotFound, calls: 2},
		{name: "duplicate name", input: "nova-public", rows: named + `,{"id":"ep-2","name":"nova-public"}`, wantErr: resource.ErrAmbiguous, calls: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonEndpointCall
			api, _ := pythonEndpointAPI(t, &calls, func(req *http.Request) (int, string) {
				switch req.URL.Path {
				case pythonEndpointBase + "/ep-1":
					return 200, `{"endpoint":` + pythonEndpointRow + `}`
				case pythonEndpointBase:
					return 200, `{"endpoints":[` + tc.rows + `],"links":{"next":null}}`
				}
				return 404, `{"error":{"code":404}}`
			})
			got, err := pythonFindEndpoint(ctx, api, tc.input, tc.options...)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) || got != nil {
					t.Fatal(got, err)
				}
			} else if err != nil || (tc.wantID == "") != (got == nil) || got != nil && got.ID != tc.wantID {
				t.Fatal(got, err)
			}
			want := []pythonEndpointCall{{http.MethodGet, pythonEndpointBase + "/" + tc.input, "", ""}, {http.MethodGet, pythonEndpointBase, "", ""}}[:tc.calls]
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

// Python endpoints() maps interface/region_id/service_id/limit/marker to the
// query and ignores the Keystone links.next dict; Go sends only the three
// filters and follows links.next.
func TestPythonEndpointListQueryAndPaging(t *testing.T) {
	ctx := context.Background()
	var calls []pythonEndpointCall
	var cloud *testcloud.Cloud
	api, cloud := pythonEndpointAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("page") == "2" {
			return 200, `{"endpoints":[{"id":"ep-2","interface":"public"}],"links":{"next":null}}`
		}
		return 200, `{"endpoints":[` + pythonEndpointRow + `],"links":{"next":"` + cloud.Server.URL + pythonEndpointBase + `?page=2"}}`
	})
	var ids []string
	for value, err := range api.List(ctx, endpoints.WithListOptions(endpoints.ListOpts{Availability: gophercloud.AvailabilityPublic, RegionID: "RegionOne", ServiceID: "svc-1"})) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	want := []pythonEndpointCall{
		{http.MethodGet, pythonEndpointBase, "interface=public&region_id=RegionOne&service_id=svc-1", ""},
		{http.MethodGet, pythonEndpointBase, "page=2", ""},
	}
	if !reflect.DeepEqual(ids, []string{"ep-1", "ep-2"}) || !reflect.DeepEqual(calls, want) {
		t.Fatalf("%v %+v", ids, calls)
	}
	calls = nil
	rows, err := api.All(ctx, resource.WithQuery("interface", "public"), resource.WithQuery("service_id", "svc-1"))
	if err != nil || len(rows) != 2 || calls[0].query != "interface=public&service_id=svc-1" {
		t.Fatal(rows, err, calls)
	}
	for _, key := range []string{"limit", "marker"} {
		calls = nil
		if _, err := api.All(ctx, resource.WithQuery(key, "1")); err == nil || len(calls) != 0 {
			t.Fatal(key, err, calls)
		}
	}
}

func TestPythonEndpointWaitHelpers(t *testing.T) {
	ctx := context.Background()
	var calls []pythonEndpointCall
	api, _ := pythonEndpointAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"endpoint":` + pythonEndpointRow + `}` })
	// Endpoint has no status attribute; Python raises AttributeError.
	if _, err := api.WaitFor(ctx, resource.ID("ep-1"), "active"); !errors.Is(err, resource.ErrUnsupported) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}
