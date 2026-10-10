package tenants_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pythonTenantTransport func(*http.Request) (*http.Response, error)

func (transport pythonTenantTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonTenantCall struct{ method, path, query, body string }

func pythonTenantAPI(t *testing.T, calls *[]pythonTenantCall, reply func(*http.Request) (int, string)) (*tenants.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonTenantTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonTenantCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v2.0/"
	return tenants.New(client), cloud
}

const (
	pythonTenantBase = "/keystone/v2.0/tenants"
	pythonTenantRow  = `{"id":"t-1","name":"demo","description":"d","enabled":true}`
)

// Python create_tenant/get_tenant/update_tenant send Resource.create, fetch and
// commit requests wrapped in the "tenant" resource_key. Proxy._update builds
// the Resource from the ID, so the dirty body also carries "id".
func TestPythonTenantCreateGetUpdateRequests(t *testing.T) {
	ctx := context.Background()
	var calls []pythonTenantCall
	api, _ := pythonTenantAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.Method == http.MethodPut {
			return 200, `{"tenant":{"id":"t-1","name":"renamed","description":"d","enabled":false}}`
		}
		return 200, `{"tenant":` + pythonTenantRow + `}`
	})
	enabled, disabled := true, false
	created, err := api.Create(ctx, tenants.CreateOpts{Name: "demo", Description: "d", Enabled: &enabled})
	want := tenants.Tenant{ID: "t-1", Name: "demo", Description: "d", Enabled: true}
	if err != nil || !reflect.DeepEqual(*created, want) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "t-1")
	if err != nil || !reflect.DeepEqual(*got, want) {
		t.Fatal(got, err)
	}
	updated, err := api.Update(ctx, "t-1", tenants.UpdateOpts{Name: "renamed", Enabled: &disabled}, tenants.WithUpdateField("id", "t-1"))
	if err != nil || updated.Name != "renamed" || updated.Enabled {
		t.Fatal(updated, err)
	}
	wantCalls := []pythonTenantCall{
		{http.MethodPost, pythonTenantBase, "", `{"tenant":{"description":"d","enabled":true,"name":"demo"}}`},
		{http.MethodGet, pythonTenantBase + "/t-1", "", ""},
		{http.MethodPut, pythonTenantBase + "/t-1", "", `{"tenant":{"enabled":false,"id":"t-1","name":"renamed"}}`},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("%+v", calls)
	}
	// Python would POST {"tenant": {}} without a name; Go rejects it before HTTP.
	calls = nil
	if _, err := api.Create(ctx, tenants.CreateOpts{}); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonTenantDeleteIgnoreMissing(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		status  int
		options []resource.LookupOption
		missing bool
	}{
		{"deleted", 204, nil, false},
		{"missing ignored by default", 404, nil, false},
		{"ignore_missing=False", 404, []resource.LookupOption{resource.WithMissingError()}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonTenantCall
			api, _ := pythonTenantAPI(t, &calls, func(*http.Request) (int, string) { return tc.status, "" })
			err := api.Remove(ctx, resource.ID("t-1"), tc.options...)
			if tc.missing != errors.Is(err, resource.ErrNotFound) || !tc.missing && err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []pythonTenantCall{{http.MethodDelete, pythonTenantBase + "/t-1", "", ""}}) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

// pythonFindTenant mirrors Resource.find: GET by ID first, then the full list
// matched locally by name.
func pythonFindTenant(ctx context.Context, api *tenants.API, nameOrID string, options ...resource.LookupOption) (*tenants.Tenant, error) {
	found, err := api.Find(ctx, resource.ID(nameOrID), resource.WithIgnoreMissing())
	if err != nil || found != nil {
		return found, err
	}
	return api.Find(ctx, resource.Name(nameOrID), options...)
}

func TestPythonTenantFindIDThenNameFallback(t *testing.T) {
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
		{name: "id hit", input: "t-1", wantID: "t-1", calls: 1},
		{name: "name fallback across tenants_links", input: "demo", rows: `{"id":"t-1","name":"demo"}`, options: []resource.LookupOption{resource.WithIgnoreMissing()}, wantID: "t-1", calls: 3},
		{name: "missing ignored", input: "ghost", rows: `{"id":"t-1","name":"demo"}`, options: []resource.LookupOption{resource.WithIgnoreMissing()}, calls: 3},
		{name: "missing strict", input: "ghost", rows: `{"id":"t-1","name":"demo"}`, wantErr: resource.ErrNotFound, calls: 3},
		{name: "duplicate name", input: "demo", rows: `{"id":"t-1","name":"demo"},{"id":"t-3","name":"demo"}`, wantErr: resource.ErrAmbiguous, calls: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonTenantCall
			var cloud *testcloud.Cloud
			api, cloud := pythonTenantAPI(t, &calls, func(req *http.Request) (int, string) {
				switch req.URL.Path {
				case pythonTenantBase + "/t-1":
					return 200, `{"tenant":` + pythonTenantRow + `}`
				case pythonTenantBase:
					return 200, `{"tenants":[` + tc.rows + `],"tenants_links":[{"rel":"next","href":"` + cloud.Server.URL + `/keystone/v2.0/next"}]}`
				case "/keystone/v2.0/next":
					return 200, `{"tenants":[{"id":"t-2","name":"other"}],"tenants_links":[]}`
				}
				return 404, `{"error":{"code":404}}`
			})
			got, err := pythonFindTenant(ctx, api, tc.input, tc.options...)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) || got != nil {
					t.Fatal(got, err)
				}
			} else if err != nil || (tc.wantID == "") != (got == nil) || got != nil && got.ID != tc.wantID {
				t.Fatal(got, err)
			}
			if len(calls) != tc.calls || calls[0] != (pythonTenantCall{http.MethodGet, pythonTenantBase + "/" + tc.input, "", ""}) {
				t.Fatalf("%+v", calls)
			}
			if tc.calls >= 2 && calls[1] != (pythonTenantCall{http.MethodGet, pythonTenantBase, "", ""}) || tc.calls == 3 && calls[2].path != "/keystone/v2.0/next" {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

// Python tenants(limit=1) keeps requesting marker pages when no next link is
// returned. The native pager only follows tenants_links.
func TestPythonTenantListRequests(t *testing.T) {
	ctx := context.Background()
	var calls []pythonTenantCall
	var cloud *testcloud.Cloud
	api, cloud := pythonTenantAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == "/keystone/v2.0/next" {
			return 200, `{"tenants":[{"id":"t-2","name":"other","enabled":false}],"tenants_links":[]}`
		}
		if req.URL.RawQuery == "" {
			return 200, `{"tenants":[` + pythonTenantRow + `],"tenants_links":[{"rel":"next","href":"` + cloud.Server.URL + `/keystone/v2.0/next?marker=t-1"}]}`
		}
		return 200, `{"tenants":[` + pythonTenantRow + `],"tenants_links":[]}`
	})
	var ids []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	for value, err := range api.List(ctx, tenants.WithListOptions(tenants.ListOpts{Limit: 1, Marker: "t-0"})) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	want := []pythonTenantCall{
		{http.MethodGet, pythonTenantBase, "", ""},
		{http.MethodGet, "/keystone/v2.0/next", "marker=t-1", ""},
		{http.MethodGet, pythonTenantBase, "limit=1&marker=t-0", ""},
	}
	if !reflect.DeepEqual(ids, []string{"t-1", "t-2", "t-1"}) || !reflect.DeepEqual(calls, want) {
		t.Fatalf("%v %+v", ids, calls)
	}
}

func TestPythonTenantWaitHelpers(t *testing.T) {
	ctx := context.Background()
	t.Run("status attribute is required", func(t *testing.T) {
		var calls []pythonTenantCall
		api, _ := pythonTenantAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"tenant":` + pythonTenantRow + `}` })
		// Tenant has no status attribute; Python raises AttributeError before HTTP.
		if _, err := api.WaitFor(ctx, resource.ID("t-1"), "active"); !errors.Is(err, resource.ErrUnsupported) || len(calls) != 0 {
			t.Fatal(err, calls)
		}
		got, err := api.WaitFor(ctx, resource.ID("t-1"), "DEMO", resource.WithStatusAttribute("name"), resource.WithUnlimitedWait())
		if err != nil || got.ID != "t-1" || len(calls) != 1 {
			t.Fatal(got, err, calls)
		}
	})
	t.Run("deletion polls until 404", func(t *testing.T) {
		var calls []pythonTenantCall
		api, _ := pythonTenantAPI(t, &calls, func(*http.Request) (int, string) {
			if len(calls) <= 2 {
				return 200, `{"tenant":` + pythonTenantRow + `}`
			}
			return 404, ""
		})
		var progress []int
		err := api.WaitForDeletion(ctx, resource.ID("t-1"), resource.WithTimeout(2*time.Minute), resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(value int) { progress = append(progress, value) }))
		get := pythonTenantCall{http.MethodGet, pythonTenantBase + "/t-1", "", ""}
		if err != nil || !reflect.DeepEqual(calls, []pythonTenantCall{get, get, get}) || !reflect.DeepEqual(progress, []int{0, 0}) {
			t.Fatal(err, calls, progress)
		}
	})
}
