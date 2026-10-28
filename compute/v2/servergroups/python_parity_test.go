package servergroups_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// pythonServerGroupCall records the wire request openstacksdk would also send.
type pythonServerGroupCall struct{ method, path, query, body string }

func pythonServerGroupWire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}})
}

func pythonServerGroupAPI(t *testing.T, reply func(*http.Request) *http.Response) (*servergroups.API, *testcloud.Recorder) {
	t.Helper()
	calls := testcloud.NewRecorder(func(req *http.Request) (*http.Response, error) { return reply(req), nil })
	cloud := testcloud.New(t)
	cloud.SetTransport(calls)
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.Microversion = "2.64"
	return servergroups.New(client), calls
}

// pythonServerGroupCalls projects the recorded calls onto the local call fields.
func pythonServerGroupCalls(rec *testcloud.Recorder) []pythonServerGroupCall {
	calls := []pythonServerGroupCall{}
	for _, call := range rec.Calls() {
		calls = append(calls, pythonServerGroupCall{call.Method, call.Path, call.RawQuery, call.Body})
	}
	return calls
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
	if !reflect.DeepEqual(pythonServerGroupCalls(calls), want) {
		t.Fatalf("%+v", pythonServerGroupCalls(calls))
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
	if !reflect.DeepEqual(pythonServerGroupCalls(calls), want) {
		t.Fatalf("%+v", pythonServerGroupCalls(calls))
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
	if !reflect.DeepEqual(pythonServerGroupCalls(calls), want) {
		t.Fatalf("%+v", pythonServerGroupCalls(calls))
	}
}
