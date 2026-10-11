package projects_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonProjectTransport func(*http.Request) (*http.Response, error)

func (transport pythonProjectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonProjectCall struct{ method, path, query, body string }

func pythonProjectAPI(t *testing.T, calls *[]pythonProjectCall, reply func(*http.Request) (int, string)) (*projects.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonProjectTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonProjectCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return projects.New(client), cloud
}

const (
	pythonProjectBase = "/keystone/v3/projects"
	pythonProjectRow  = `{"id":"p-1","name":"proj","description":"desc","domain_id":"dom-1","enabled":true,"is_domain":false,"parent_id":"dom-1","tags":["a"],"options":{"immutable":false},"links":{"self":"https://keystone.invalid/v3/projects/p-1"},"x_vendor":"v"}`
)

// Python create_project/get_project/update_project use the "project"
// resource_key and PATCH; Project allows unknown attributes in the body, and
// update carries the ID in the dirty body.
func TestPythonProjectCreateGetUpdateRequests(t *testing.T) {
	ctx := context.Background()
	var calls []pythonProjectCall
	api, _ := pythonProjectAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.Method {
		case http.MethodPost:
			return 201, `{"project":` + pythonProjectRow + `}`
		case http.MethodPatch:
			return 200, `{"project":{"id":"p-1","name":"proj","description":null,"enabled":false,"tags":[]}}`
		}
		return 200, `{"project":` + pythonProjectRow + `}`
	})
	enabled, disabled := true, false
	want := projects.Project{
		ID: "p-1", Name: "proj", Description: "desc", DomainID: "dom-1", Enabled: true, ParentID: "dom-1", Tags: []string{"a"},
		Options: map[projects.Option]any{projects.Immutable: false},
		Extra:   map[string]any{"links": map[string]any{"self": "https://keystone.invalid/v3/projects/p-1"}, "x_vendor": "v"},
	}
	created, err := api.Create(ctx, projects.CreateOpts{
		Name: "proj", Description: "desc", DomainID: "dom-1", Enabled: &enabled, ParentID: "dom-1", Tags: []string{"a"},
		Extra: map[string]any{"x_vendor": "v"},
	})
	if err != nil || !reflect.DeepEqual(*created, want) {
		t.Fatalf("%#v %v", created, err)
	}
	got, err := api.Get(ctx, "p-1")
	if err != nil || !reflect.DeepEqual(*got, want) {
		t.Fatalf("%#v %v", got, err)
	}
	empty := []string{}
	// Python update_project(p, description=None, is_enabled=False, tags=[]).
	updated, err := api.Update(ctx, "p-1", projects.UpdateOpts{Enabled: &disabled, Tags: &empty, Extra: map[string]any{"description": nil}}, projects.WithUpdateField("id", "p-1"))
	if err != nil || updated.Enabled || len(updated.Tags) != 0 {
		t.Fatalf("%#v %v", updated, err)
	}
	wantCalls := []pythonProjectCall{
		{http.MethodPost, pythonProjectBase, "", `{"project":{"description":"desc","domain_id":"dom-1","enabled":true,"name":"proj","parent_id":"dom-1","tags":["a"],"x_vendor":"v"}}`},
		{http.MethodGet, pythonProjectBase + "/p-1", "", ""},
		{http.MethodPatch, pythonProjectBase + "/p-1", "", `{"project":{"description":null,"enabled":false,"id":"p-1","tags":[]}}`},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("%+v", calls)
	}
	calls = nil
	if _, err := api.Create(ctx, projects.CreateOpts{Description: "desc"}); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonProjectDeleteIgnoreMissing(t *testing.T) {
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
			var calls []pythonProjectCall
			api, _ := pythonProjectAPI(t, &calls, func(*http.Request) (int, string) { return tc.status, "" })
			err := api.Remove(ctx, resource.ID("p-1"), tc.options...)
			if tc.missing != errors.Is(err, resource.ErrNotFound) || tc.conflict != gophercloud.ResponseCodeIs(err, http.StatusConflict) || !tc.missing && !tc.conflict && err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []pythonProjectCall{{http.MethodDelete, pythonProjectBase + "/p-1", "", ""}}) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

func TestPythonProjectListRequests(t *testing.T) {
	ctx := context.Background()
	collect := func(t *testing.T, api *projects.API, options ...projects.ListOption) []string {
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
		var calls []pythonProjectCall
		api, _ := pythonProjectAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"projects":[` + pythonProjectRow + `],"links":{"self":"x","next":null,"previous":null}}`
		})
		if ids := collect(t, api); !reflect.DeepEqual(ids, []string{"p-1"}) || !reflect.DeepEqual(calls, []pythonProjectCall{{http.MethodGet, pythonProjectBase, "", ""}}) {
			t.Fatalf("%v %+v", ids, calls)
		}
	})
	t.Run("query mapping and Python bool text", func(t *testing.T) {
		var calls []pythonProjectCall
		api, _ := pythonProjectAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"projects":[]}` })
		// Python projects(domain_id=..., name=..., parent_id=..., any_tags=...,
		// not_tags=..., is_domain=False, is_enabled=True).
		collect(t, api,
			projects.WithListOptions(projects.ListOpts{DomainID: "dom-1", Name: "proj", ParentID: "dom-1", TagsAny: "a,b", NotTags: "c"}),
			projects.WithListQuery("is_domain", "False"), projects.WithListQuery("enabled", "True"))
		want := "domain_id=dom-1&enabled=True&is_domain=False&name=proj&not-tags=c&parent_id=dom-1&tags-any=a%2Cb"
		if len(calls) != 1 || calls[0].query != want {
			t.Fatalf("%+v", calls)
		}
	})
	t.Run("limit sends no marker request", func(t *testing.T) {
		var calls []pythonProjectCall
		api, _ := pythonProjectAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"projects":[` + pythonProjectRow + `],"links":{"next":null}}`
		})
		// Python follows with GET ?limit=1&marker=p-1 until an empty page.
		collect(t, api, projects.WithListOptions(projects.ListOpts{Limit: 1}))
		if !reflect.DeepEqual(calls, []pythonProjectCall{{http.MethodGet, pythonProjectBase, "limit=1", ""}}) {
			t.Fatalf("%+v", calls)
		}
	})
	t.Run("no local Body filter", func(t *testing.T) {
		var calls []pythonProjectCall
		api, _ := pythonProjectAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"projects":[]}` })
		if _, err := api.All(ctx, resource.WithBodyFilter("description", "desc")); !errors.Is(err, resource.ErrUnsupported) || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	})
}
