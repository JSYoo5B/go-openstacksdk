package projectendpoints_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonProjectEndpointCall struct{ method, path, query, body string }

func pythonProjectEndpointAPI(t *testing.T, calls *[]pythonProjectEndpointCall, reply func(*http.Request) (int, string)) *projectendpoints.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonProjectEndpointCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return projectendpoints.New(client)
}

const pythonProjectEndpointPath = "/keystone/v3/OS-EP-FILTER/projects/p-1/endpoints"

// Python associate_endpoint_with_project and disassociate_endpoint_from_project
// send a bodyless PUT/DELETE and raise on any error status, including 404.
func TestPythonProjectEndpointAssociation(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		status int
		ok     bool
	}{
		{"no content", 204, true},
		{"missing", 404, false},
		{"conflict", 409, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonProjectEndpointCall
			api := pythonProjectEndpointAPI(t, &calls, func(*http.Request) (int, string) { return tc.status, "" })
			createErr := api.Create(ctx, "p-1", "ep-1")
			deleteErr := api.Delete(ctx, "p-1", "ep-1")
			for _, err := range []error{createErr, deleteErr} {
				if tc.ok != (err == nil) || !tc.ok && !gophercloud.ResponseCodeIs(err, tc.status) {
					t.Fatal(err)
				}
			}
			want := []pythonProjectEndpointCall{{http.MethodPut, pythonProjectEndpointPath + "/ep-1", "", ""}, {http.MethodDelete, pythonProjectEndpointPath + "/ep-1", "", ""}}
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

// Python project_endpoints(project) without query sends the same GET; the Go
// List takes no options, so interface/region_id/service_id/limit/marker cannot
// be added.
func TestPythonProjectEndpointListDefault(t *testing.T) {
	ctx := context.Background()
	var calls []pythonProjectEndpointCall
	api := pythonProjectEndpointAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"endpoints":[{"id":"ep-1","interface":"public","region":"RegionOne","service_id":"svc-1","url":"https://nova","enabled":true}],"links":{"next":null}}`
	})
	var rows []projectendpoints.Endpoint
	for value, err := range api.List(ctx, "p-1") {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, *value)
	}
	want := []projectendpoints.Endpoint{{ID: "ep-1", Availability: gophercloud.AvailabilityPublic, Region: "RegionOne", ServiceID: "svc-1", URL: "https://nova"}}
	if !reflect.DeepEqual(rows, want) || !reflect.DeepEqual(calls, []pythonProjectEndpointCall{{http.MethodGet, pythonProjectEndpointPath, "", ""}}) {
		t.Fatalf("%+v %+v", rows, calls)
	}
}
