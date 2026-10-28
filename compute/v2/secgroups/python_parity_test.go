package secgroups_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

// pythonSecGroupCall records the wire request openstacksdk would also send.
type pythonSecGroupCall struct{ method, path, body, version string }

func pythonSecGroupAPI(t *testing.T, reply func(*http.Request) (int, string)) (*secgroups.API, *testcloud.Recorder) {
	t.Helper()
	calls := testcloud.NewRecorder(func(req *http.Request) (*http.Response, error) {
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	})
	cloud := testcloud.New(t)
	cloud.SetTransport(calls)
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.Microversion = "2.1"
	return secgroups.New(client), calls
}

// pythonSecGroupCalls projects the recorded calls onto the local call fields.
func pythonSecGroupCalls(rec *testcloud.Recorder) []pythonSecGroupCall {
	calls := []pythonSecGroupCall{}
	for _, call := range rec.Calls() {
		calls = append(calls, pythonSecGroupCall{call.Method, call.Path, call.Body, call.Header.Get("X-OpenStack-Nova-API-Version")})
	}
	return calls
}

func TestPythonSecGroupServerFetchAddRemoveMatchProxyRequests(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonSecGroupAPI(t, func(req *http.Request) (int, string) {
		if req.Method == http.MethodPost {
			return 202, ""
		}
		return 200, `{"security_groups":[{"id":"sg-1","name":"web","description":"d","tenant_id":"p","rules":[]}]}`
	})
	// fetch_server_security_groups(server)
	var names []string
	for group, err := range api.ListByServer(ctx, "s1") {
		if err != nil {
			t.Fatal(err)
		}
		if group.ID != "sg-1" || group.Description != "d" || group.TenantID != "p" {
			t.Fatalf("%+v", group)
		}
		names = append(names, group.Name)
	}
	if !reflect.DeepEqual(names, []string{"web"}) {
		t.Fatal(names)
	}
	// add_security_group_to_server(server, "web") and remove_security_group_from_server(server, "web")
	// send the given string as the group name without a lookup.
	if err := api.AddServer(ctx, "s1", "web"); err != nil {
		t.Fatal(err)
	}
	if err := api.RemoveServer(ctx, "s1", "web"); err != nil {
		t.Fatal(err)
	}
	want := []pythonSecGroupCall{
		{http.MethodGet, "/nova/v2.1/servers/s1/os-security-groups", "", "2.1"},
		{http.MethodPost, "/nova/v2.1/servers/s1/action", `{"addSecurityGroup":{"name":"web"}}`, "2.1"},
		{http.MethodPost, "/nova/v2.1/servers/s1/action", `{"removeSecurityGroup":{"name":"web"}}`, "2.1"},
	}
	if !reflect.DeepEqual(pythonSecGroupCalls(calls), want) {
		t.Fatalf("%+v", pythonSecGroupCalls(calls))
	}
}
