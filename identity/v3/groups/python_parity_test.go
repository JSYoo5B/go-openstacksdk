package groups_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonGroupTransport func(*http.Request) (*http.Response, error)

func (transport pythonGroupTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonGroupCall struct{ method, path, query, body string }

func pythonGroupAPI(t *testing.T, calls *[]pythonGroupCall, reply func(*http.Request) (int, string)) (*groups.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonGroupTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonGroupCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return groups.New(client), cloud
}

const (
	pythonGroupBase = "/keystone/v3/groups"
	pythonGroupRow  = `{"id":"g-1","name":"grp","description":"desc","domain_id":"dom-1","links":{"self":"https://keystone.invalid/v3/groups/g-1"}}`
)

// Python create_group/get_group/update_group use the "group" resource_key and
// PATCH; update carries the ID in the dirty body because Proxy._update builds
// the Resource from it.
func TestPythonGroupCreateGetUpdateRequests(t *testing.T) {
	ctx := context.Background()
	var calls []pythonGroupCall
	api, _ := pythonGroupAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.Method {
		case http.MethodPost:
			return 201, `{"group":` + pythonGroupRow + `}`
		case http.MethodPatch:
			return 200, `{"group":{"id":"g-1","name":"renamed","description":null,"domain_id":"dom-1"}}`
		}
		return 200, `{"group":` + pythonGroupRow + `}`
	})
	want := groups.Group{ID: "g-1", Name: "grp", Description: "desc", DomainID: "dom-1", Links: map[string]any{"self": "https://keystone.invalid/v3/groups/g-1"}, Extra: map[string]any{}}
	created, err := api.Create(ctx, groups.CreateOpts{Name: "grp", Description: "desc", DomainID: "dom-1"})
	if err != nil || !reflect.DeepEqual(*created, want) {
		t.Fatalf("%#v %v", created, err)
	}
	got, err := api.Get(ctx, "g-1")
	if err != nil || !reflect.DeepEqual(*got, want) {
		t.Fatalf("%#v %v", got, err)
	}
	// Python update_group(g, name="renamed", description=None) sends null.
	updated, err := api.Update(ctx, "g-1", groups.UpdateOpts{Name: "renamed", Extra: map[string]any{"description": nil}}, groups.WithUpdateField("id", "g-1"))
	if err != nil || updated.Name != "renamed" || updated.Description != "" {
		t.Fatalf("%#v %v", updated, err)
	}
	wantCalls := []pythonGroupCall{
		{http.MethodPost, pythonGroupBase, "", `{"group":{"description":"desc","domain_id":"dom-1","name":"grp"}}`},
		{http.MethodGet, pythonGroupBase + "/g-1", "", ""},
		{http.MethodPatch, pythonGroupBase + "/g-1", "", `{"group":{"description":null,"id":"g-1","name":"renamed"}}`},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("%+v", calls)
	}
	calls = nil
	if _, err := api.Create(ctx, groups.CreateOpts{Description: "desc"}); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonGroupDeleteIgnoreMissing(t *testing.T) {
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
			var calls []pythonGroupCall
			api, _ := pythonGroupAPI(t, &calls, func(*http.Request) (int, string) { return tc.status, "" })
			err := api.Remove(ctx, resource.ID("g-1"), tc.options...)
			if tc.missing != errors.Is(err, resource.ErrNotFound) || tc.conflict != gophercloud.ResponseCodeIs(err, http.StatusConflict) || !tc.missing && !tc.conflict && err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []pythonGroupCall{{http.MethodDelete, pythonGroupBase + "/g-1", "", ""}}) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

func TestPythonGroupListRequests(t *testing.T) {
	ctx := context.Background()
	collect := func(t *testing.T, api *groups.API, options ...groups.ListOption) []string {
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
		var calls []pythonGroupCall
		api, _ := pythonGroupAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"groups":[` + pythonGroupRow + `],"links":{"self":"x","next":null,"previous":null}}`
		})
		if ids := collect(t, api); !reflect.DeepEqual(ids, []string{"g-1"}) || !reflect.DeepEqual(calls, []pythonGroupCall{{http.MethodGet, pythonGroupBase, "", ""}}) {
			t.Fatalf("%v %+v", ids, calls)
		}
	})
	t.Run("query mapping", func(t *testing.T) {
		var calls []pythonGroupCall
		api, _ := pythonGroupAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"groups":[]}` })
		collect(t, api, groups.WithListOptions(groups.ListOpts{DomainID: "dom-1", Name: "grp"}))
		if !reflect.DeepEqual(calls, []pythonGroupCall{{http.MethodGet, pythonGroupBase, "domain_id=dom-1&name=grp", ""}}) {
			t.Fatalf("%+v", calls)
		}
	})
	t.Run("limit sends no marker request", func(t *testing.T) {
		var calls []pythonGroupCall
		api, _ := pythonGroupAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"groups":[` + pythonGroupRow + `],"links":{"next":null}}`
		})
		// Python follows with GET ?limit=1&marker=g-1 until an empty page.
		collect(t, api, groups.WithListQuery("limit", "1"))
		if !reflect.DeepEqual(calls, []pythonGroupCall{{http.MethodGet, pythonGroupBase, "limit=1", ""}}) {
			t.Fatalf("%+v", calls)
		}
	})
	t.Run("no local Body filter", func(t *testing.T) {
		var calls []pythonGroupCall
		api, _ := pythonGroupAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"groups":[]}` })
		if _, err := api.All(ctx, resource.WithBodyFilter("description", "desc")); !errors.Is(err, resource.ErrUnsupported) || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	})
}
