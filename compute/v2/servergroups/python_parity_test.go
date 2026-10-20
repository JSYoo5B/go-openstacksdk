package servergroups_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// pythonServerGroupCall records the wire request openstacksdk would also send.
type pythonServerGroupCall struct{ method, path, query, body string }

type pythonServerGroupTransport func(*http.Request) (*http.Response, error)

func (transport pythonServerGroupTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonServerGroupWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonServerGroupAPI(t *testing.T, reply func(*http.Request) *http.Response) (*servergroups.API, *[]pythonServerGroupCall) {
	t.Helper()
	calls := &[]pythonServerGroupCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonServerGroupTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonServerGroupCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.Microversion = "2.64"
	return servergroups.New(client), calls
}

const pythonServerGroupBase = "/nova/v2.1/os-server-groups"

const pythonServerGroupRow = `{"id":"g1","name":"web","policy":"anti-affinity","rules":{"max_server_per_host":2},"members":["s1"],"project_id":"p","user_id":"u","metadata":{}}`

func TestPythonServerGroupCreateGetDeleteMatchProxyRequests(t *testing.T) {
	ctx := context.Background()
	deleteStatus := http.StatusNotFound
	api, calls := pythonServerGroupAPI(t, func(req *http.Request) *http.Response {
		if req.Method == http.MethodDelete {
			return pythonServerGroupWire(deleteStatus, `{"itemNotFound":{"message":"gone"}}`)
		}
		return pythonServerGroupWire(200, `{"server_group":`+pythonServerGroupRow+`}`)
	})
	// create_server_group(name="web", policy="anti-affinity", rules={...}) at 2.64.
	maxPerHost := 2
	created, err := api.Create(ctx, servergroups.CreateOpts{Name: "web", Policy: "anti-affinity", Rules: &servergroups.Rules{MaxServerPerHost: maxPerHost}})
	if err != nil || created.ID != "g1" || created.Policy == nil || *created.Policy != "anti-affinity" || created.Rules.MaxServerPerHost != 2 || !reflect.DeepEqual(created.Members, []string{"s1"}) {
		t.Fatal(created, err)
	}
	// Below 2.64 Python rewrites policy to policies; the Go caller sends policies itself.
	if _, err := api.Create(ctx, servergroups.CreateOpts{Name: "web", Policies: []string{"anti-affinity"}}); err != nil {
		t.Fatal(err)
	}
	// get_server_group(server_group)
	got, err := api.Get(ctx, "g1")
	if err != nil || got.Name != "web" || got.ProjectID != "p" || got.UserID != "u" {
		t.Fatal(got, err)
	}
	// delete_server_group(server_group) ignores a missing group by default.
	if err := api.Remove(ctx, resource.ID("g1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("g1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	deleteStatus = http.StatusNoContent
	if err := api.Remove(ctx, resource.ID("g1")); err != nil {
		t.Fatal(err)
	}
	want := []pythonServerGroupCall{
		{http.MethodPost, pythonServerGroupBase, "", `{"server_group":{"name":"web","policy":"anti-affinity","rules":{"max_server_per_host":2}}}`},
		{http.MethodPost, pythonServerGroupBase, "", `{"server_group":{"name":"web","policies":["anti-affinity"]}}`},
		{http.MethodGet, pythonServerGroupBase + "/g1", "", ""},
		{http.MethodDelete, pythonServerGroupBase + "/g1", "", ""},
		{http.MethodDelete, pythonServerGroupBase + "/g1", "", ""},
		{http.MethodDelete, pythonServerGroupBase + "/g1", "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonServerGroupListAllProjectsAndQuery(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonServerGroupAPI(t, func(*http.Request) *http.Response {
		return pythonServerGroupWire(200, `{"server_groups":[`+pythonServerGroupRow+`]}`)
	})
	collect := func(options ...servergroups.ListOption) []string {
		var ids []string
		for value, err := range api.List(ctx, options...) {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, value.ID)
		}
		return ids
	}
	// server_groups()
	if ids := collect(); !reflect.DeepEqual(ids, []string{"g1"}) {
		t.Fatal(ids)
	}
	// server_groups(all_projects=True, limit=1, marker="g0")
	if ids := collect(servergroups.WithListQuery("all_projects", "True"), servergroups.WithListOptions(servergroups.ListOpts{Limit: 1}), servergroups.WithListQuery("marker", "g0")); !reflect.DeepEqual(ids, []string{"g1"}) {
		t.Fatal(ids)
	}
	want := []pythonServerGroupCall{
		{http.MethodGet, pythonServerGroupBase, "", ""},
		{http.MethodGet, pythonServerGroupBase, "all_projects=True&limit=1&marker=g0", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonServerGroupFindHasNoIDThenNameFallback(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonServerGroupAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Path == pythonServerGroupBase {
			return pythonServerGroupWire(200, `{"server_groups":[`+pythonServerGroupRow+`]}`)
		}
		return pythonServerGroupWire(404, `{"itemNotFound":{"message":"gone"}}`)
	})
	// find_server_group("web") first GETs by ID, then lists and matches ID or name.
	// Go needs two separate calls, and Find(Name) cannot carry all_projects.
	if got, err := api.Find(ctx, resource.ID("web"), resource.WithIgnoreMissing()); err != nil || got != nil {
		t.Fatal(got, err)
	}
	got, err := api.Find(ctx, resource.Name("web"))
	if err != nil || got == nil || got.ID != "g1" {
		t.Fatal(got, err)
	}
	if missing, err := api.Find(ctx, resource.Name("db"), resource.WithIgnoreMissing()); err != nil || missing != nil {
		t.Fatal(missing, err)
	}
	want := []pythonServerGroupCall{
		{http.MethodGet, pythonServerGroupBase + "/web", "", ""},
		{http.MethodGet, pythonServerGroupBase, "", ""},
		{http.MethodGet, pythonServerGroupBase, "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}
