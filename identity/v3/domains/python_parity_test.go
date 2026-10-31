package domains_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonDomainCall struct{ method, path, query, body string }

func pythonDomainAPI(t *testing.T, calls *[]pythonDomainCall, reply func(*http.Request) (int, string)) (*domains.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonDomainCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return domains.New(client), cloud
}

const (
	pythonDomainBase = "/keystone/v3/domains"
	pythonDomainRow  = `{"id":"d-1","name":"dom","description":"desc","enabled":true,"options":{"immutable":true},"links":{"self":"https://keystone.invalid/v3/domains/d-1"}}`
)

// Native request fixtures use the "domain" resource_key and explicit PATCH
// fields. WithUpdateField("id", ...) adds the ID only when requested by Go;
// pinned Source Resource.commit removes the ID from its dirty body.
func TestPythonDomainCreateGetUpdateRequests(t *testing.T) {
	ctx := context.Background()
	var calls []pythonDomainCall
	api, _ := pythonDomainAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.Method {
		case http.MethodPost:
			return 201, `{"domain":` + pythonDomainRow + `}`
		case http.MethodPatch:
			return 200, `{"domain":{"id":"d-1","name":"dom","description":"new","enabled":false,"options":{"immutable":true}}}`
		}
		return 200, `{"domain":` + pythonDomainRow + `}`
	})
	disabled, description := false, "new"
	// The native Domain has no Options field, so the options object Python
	// exposes as Domain.options is dropped.
	want := domains.Domain{ID: "d-1", Name: "dom", Description: "desc", Enabled: true, Links: map[string]any{"self": "https://keystone.invalid/v3/domains/d-1"}}
	created, err := api.Create(ctx, domains.CreateOpts{Name: "dom", Description: "desc", Enabled: &disabled}, domains.WithCreateField("options", map[string]any{"immutable": true}))
	if err != nil || !reflect.DeepEqual(*created, want) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "d-1")
	if err != nil || !reflect.DeepEqual(*got, want) {
		t.Fatal(got, err)
	}
	updated, err := api.Update(ctx, "d-1", domains.UpdateOpts{Description: &description, Enabled: &disabled}, domains.WithUpdateField("id", "d-1"))
	if err != nil || !reflect.DeepEqual(*updated, domains.Domain{ID: "d-1", Name: "dom", Description: "new"}) {
		t.Fatal(updated, err)
	}
	wantCalls := []pythonDomainCall{
		{http.MethodPost, pythonDomainBase, "", `{"domain":{"description":"desc","enabled":false,"name":"dom","options":{"immutable":true}}}`},
		{http.MethodGet, pythonDomainBase + "/d-1", "", ""},
		{http.MethodPatch, pythonDomainBase + "/d-1", "", `{"domain":{"description":"new","enabled":false,"id":"d-1"}}`},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("%+v", calls)
	}

	// Python would POST {"domain": {"description": ...}} and PATCH
	// {"domain": {"description": null}}; Go needs a name and cannot send null
	// for a typed key.
	calls = nil
	if _, err := api.Create(ctx, domains.CreateOpts{Description: "desc"}); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	if _, err := api.Update(ctx, "d-1", domains.UpdateOpts{}, domains.WithUpdateField("description", nil)); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonDomainDeleteIgnoreMissing(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name     string
		status   int
		options  []resource.LookupOption
		missing  bool
		conflict bool
	}{
		{name: "deleted", status: 204},
		{name: "missing ignored by default", status: 404},
		{name: "ignore_missing=False", status: 404, options: []resource.LookupOption{resource.WithMissingError()}, missing: true},
		{name: "conflict is not missing", status: 409, conflict: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonDomainCall
			api, _ := pythonDomainAPI(t, &calls, func(*http.Request) (int, string) { return tc.status, "" })
			err := api.Remove(ctx, resource.ID("d-1"), tc.options...)
			if tc.missing != errors.Is(err, resource.ErrNotFound) || tc.conflict != gophercloud.ResponseCodeIs(err, http.StatusConflict) || !tc.missing && !tc.conflict && err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []pythonDomainCall{{http.MethodDelete, pythonDomainBase + "/d-1", "", ""}}) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

func TestPythonDomainListRequests(t *testing.T) {
	ctx := context.Background()
	collect := func(t *testing.T, api *domains.API, options ...domains.ListOption) []string {
		t.Helper()
		var ids []string
		for value, err := range api.List(ctx, options...) {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, value.ID)
		}
		return ids
	}
	t.Run("default single request", func(t *testing.T) {
		var calls []pythonDomainCall
		api, _ := pythonDomainAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"domains":[` + pythonDomainRow + `],"links":{"self":"x","next":null,"previous":null}}`
		})
		if ids := collect(t, api); !reflect.DeepEqual(ids, []string{"d-1"}) || !reflect.DeepEqual(calls, []pythonDomainCall{{http.MethodGet, pythonDomainBase, "", ""}}) {
			t.Fatalf("%v %+v", ids, calls)
		}
	})
	t.Run("query mapping and Python bool text", func(t *testing.T) {
		var calls []pythonDomainCall
		api, _ := pythonDomainAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"domains":[]}` })
		enabled := true
		collect(t, api, domains.WithListOptions(domains.ListOpts{Name: "dom", Enabled: &enabled}))
		// Python domains(name="dom", is_enabled=True) sends enabled=True.
		collect(t, api, domains.WithListOptions(domains.ListOpts{Name: "dom"}), domains.WithListQuery("enabled", "True"))
		if calls[0].query != "enabled=true&name=dom" || calls[1].query != "enabled=True&name=dom" {
			t.Fatalf("%+v", calls)
		}
	})
	t.Run("limit sends no marker request", func(t *testing.T) {
		var calls []pythonDomainCall
		api, _ := pythonDomainAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"domains":[` + pythonDomainRow + `],"links":{"next":null}}`
		})
		// Python follows with GET ?limit=1&marker=d-1 until an empty page.
		collect(t, api, domains.WithListOptions(domains.ListOpts{Limit: 1}))
		if !reflect.DeepEqual(calls, []pythonDomainCall{{http.MethodGet, pythonDomainBase, "limit=1", ""}}) {
			t.Fatalf("%+v", calls)
		}
	})
	t.Run("links.next is followed", func(t *testing.T) {
		var calls []pythonDomainCall
		var cloud *testcloud.Cloud
		api, cloud := pythonDomainAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.RawQuery == "page=2" {
				return 200, `{"domains":[{"id":"d-2"}],"links":{"next":null}}`
			}
			return 200, `{"domains":[` + pythonDomainRow + `],"links":{"next":"` + cloud.Server.URL + pythonDomainBase + `?page=2"}}`
		})
		// Python reads the Keystone links dict without a rel key and stops.
		if ids := collect(t, api); !reflect.DeepEqual(ids, []string{"d-1", "d-2"}) || len(calls) != 2 {
			t.Fatalf("%v %+v", ids, calls)
		}
	})
	t.Run("no local Body filter", func(t *testing.T) {
		var calls []pythonDomainCall
		api, _ := pythonDomainAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"domains":[]}` })
		if _, err := api.All(ctx, resource.WithBodyFilter("description", "desc")); !errors.Is(err, resource.ErrUnsupported) || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	})
}
