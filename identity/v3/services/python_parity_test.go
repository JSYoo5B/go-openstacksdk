package services_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/services"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonServiceCall struct{ method, path, query, body string }

func pythonServiceAPI(t *testing.T, calls *[]pythonServiceCall, reply func(*http.Request) (int, string)) (*services.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonServiceCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return services.New(client), cloud
}

const (
	pythonServiceBase = "/keystone/v3/services"
	pythonServiceRow  = `{"id":"svc-1","name":"nova","type":"compute","description":"d","enabled":true,"links":{"self":"x"}}`
)

func TestPythonServiceCreateGetUpdateRequests(t *testing.T) {
	ctx := context.Background()
	var calls []pythonServiceCall
	api, _ := pythonServiceAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.Method {
		case http.MethodPost:
			return 201, `{"service":` + pythonServiceRow + `}`
		case http.MethodPatch:
			return 200, `{"service":{"id":"svc-1","name":"nova","type":"compute","description":"","enabled":false}}`
		}
		return 200, `{"service":` + pythonServiceRow + `}`
	})
	check := func(got *services.Service) bool {
		return got != nil && got.ID == "svc-1" && got.Name == "nova" && got.Type == "compute" && got.Description == "d" && got.Enabled && reflect.DeepEqual(got.Links, map[string]any{"self": "x"})
	}
	enabled, disabled, empty := true, false, ""
	created, err := api.Create(ctx, services.CreateOpts{Name: "nova", Type: "compute", Description: "d", Enabled: &enabled})
	if err != nil || !check(created) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "svc-1")
	if err != nil || !check(got) {
		t.Fatal(got, err)
	}
	updated, err := api.Update(ctx, "svc-1", services.UpdateOpts{Description: &empty, Enabled: &disabled})
	if err != nil || updated.Enabled || updated.Description != "" {
		t.Fatal(updated, err)
	}
	wantCalls := []pythonServiceCall{
		{http.MethodPost, pythonServiceBase, "", `{"service":{"description":"d","enabled":true,"name":"nova","type":"compute"}}`},
		{http.MethodGet, pythonServiceBase + "/svc-1", "", ""},
		{http.MethodPatch, pythonServiceBase + "/svc-1", "", `{"service":{"description":"","enabled":false}}`},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("%+v", calls)
	}
	// Python create_service(name="nova") omits type; the Go body always has it.
	calls = nil
	if _, err := api.Create(ctx, services.CreateOpts{Name: "nova"}); err != nil || calls[0].body != `{"service":{"name":"nova","type":""}}` {
		t.Fatal(err, calls)
	}
}

func TestPythonServiceDeleteIgnoreMissing(t *testing.T) {
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
			var calls []pythonServiceCall
			api, _ := pythonServiceAPI(t, &calls, func(*http.Request) (int, string) { return tc.status, "" })
			if err := api.Remove(ctx, resource.ID("svc-1"), tc.options...); !tc.check(err) {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []pythonServiceCall{{http.MethodDelete, pythonServiceBase + "/svc-1", "", ""}}) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

// pythonFindService mirrors Resource.find: GET by ID, then the list with the
// name query because Service maps name in its QueryParameters.
func pythonFindService(ctx context.Context, api *services.API, nameOrID string, options ...resource.LookupOption) (*services.Service, error) {
	found, err := api.Find(ctx, resource.ID(nameOrID), resource.WithIgnoreMissing())
	if err != nil || found != nil {
		return found, err
	}
	return api.Find(ctx, resource.Name(nameOrID), options...)
}

func TestPythonServiceFindIDThenNameFallback(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		input   string
		rows    string
		options []resource.LookupOption
		wantID  string
		wantErr error
		calls   int
	}{
		{name: "id hit", input: "svc-1", wantID: "svc-1", calls: 1},
		{name: "name fallback", input: "nova", rows: pythonServiceRow, options: []resource.LookupOption{resource.WithIgnoreMissing()}, wantID: "svc-1", calls: 2},
		{name: "missing ignored", input: "ghost", options: []resource.LookupOption{resource.WithIgnoreMissing()}, calls: 2},
		{name: "missing strict", input: "ghost", wantErr: resource.ErrNotFound, calls: 2},
		{name: "duplicate name", input: "nova", rows: pythonServiceRow + `,{"id":"svc-2","name":"nova"}`, wantErr: resource.ErrAmbiguous, calls: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonServiceCall
			api, _ := pythonServiceAPI(t, &calls, func(req *http.Request) (int, string) {
				switch req.URL.Path {
				case pythonServiceBase + "/svc-1":
					return 200, `{"service":` + pythonServiceRow + `}`
				case pythonServiceBase:
					return 200, `{"services":[` + tc.rows + `],"links":{"next":null}}`
				}
				return 404, `{"error":{"code":404}}`
			})
			got, err := pythonFindService(ctx, api, tc.input, tc.options...)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) || got != nil {
					t.Fatal(got, err)
				}
			} else if err != nil || (tc.wantID == "") != (got == nil) || got != nil && got.ID != tc.wantID {
				t.Fatal(got, err)
			}
			want := []pythonServiceCall{{http.MethodGet, pythonServiceBase + "/" + tc.input, "", ""}, {http.MethodGet, pythonServiceBase, "name=" + tc.input, ""}}[:tc.calls]
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

// Python services() sends name/type/limit/marker and ignores the Keystone
// links.next dict; Go forwards the query and follows links.next.
func TestPythonServiceListQueryAndPaging(t *testing.T) {
	ctx := context.Background()
	var calls []pythonServiceCall
	var cloud *testcloud.Cloud
	api, cloud := pythonServiceAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("page") == "2" {
			return 200, `{"services":[{"id":"svc-2","type":"compute"}],"links":{"next":null}}`
		}
		return 200, `{"services":[` + pythonServiceRow + `],"links":{"next":"` + cloud.Server.URL + pythonServiceBase + `?page=2"}}`
	})
	rows, err := api.All(ctx, resource.WithQuery("type", "compute"), resource.WithQuery("name", "nova"), resource.WithQuery("limit", "1"))
	if err != nil || len(rows) != 2 || rows[0].ID != "svc-1" || rows[1].ID != "svc-2" {
		t.Fatal(rows, err)
	}
	want := []pythonServiceCall{
		{http.MethodGet, pythonServiceBase, "limit=1&name=nova&type=compute", ""},
		{http.MethodGet, pythonServiceBase, "page=2", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}
