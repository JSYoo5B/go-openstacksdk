package tapflows_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapflows"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonTapFlowCall struct{ method, path, query, body, accept string }

const pythonTapFlowBase = "/neutron/v2.0/taas/tap_flows"

func pythonTapFlowAPI(t *testing.T, calls *[]pythonTapFlowCall, reply func(*http.Request) (int, string)) *tapflows.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonTapFlowCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Get("Accept")})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return tapflows.New(client)
}

func pythonTapFlowOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "tapflows" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
}

func pythonTapFlowStatus(t *testing.T, err error, code int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

func pythonTapFlowBody(t *testing.T, raw string, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal([]byte(raw), &got); err != nil || !reflect.DeepEqual(got, map[string]any{"tap_flow": want}) {
		t.Fatalf("body %s, want %v (%v)", raw, want, err)
	}
}

const pythonTapFlowRow = `{"id":"tf1","name":"mirror","description":"d","tenant_id":"p1","tap_service_id":"ts1","direction":"BOTH","status":"ACTIVE","source_port":"port1","x_ext":7}`

func pythonTapFlowCheck(t *testing.T, v *tapflows.TapFlow) {
	t.Helper()
	if v == nil || v.ID != "tf1" || *v.Name != "mirror" || *v.Description != "d" || *v.ProjectID != "p1" || *v.TenantID != "p1" || *v.TapServiceID != "ts1" || *v.Direction != "BOTH" || *v.Status != "ACTIVE" || *v.SourcePort != "port1" {
		t.Fatalf("decoded tap flow: %#v", v)
	}
	if string(v.Resource.Body["x_ext"]) != "7" || string(v.Resource.Body["location"]) != "null" || string(v.Wire.Body["tenant_id"]) != `"p1"` {
		t.Fatalf("raw views: %#v %#v", v.Resource, v.Wire)
	}
}

func TestPythonTapFlowCreatePostsGivenAttrs(t *testing.T) {
	var calls []pythonTapFlowCall
	api := pythonTapFlowAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{"tap_flow":` + pythonTapFlowRow + `}` })
	value, err := api.Create(context.Background(),
		tapflows.WithCreateField("name", "mirror"),
		tapflows.WithCreateField("tap_service_id", "ts1"),
		tapflows.WithCreateField("source_port", "port1"),
		tapflows.WithCreateField("direction", "BOTH"),
		tapflows.WithCreateField("vlan_filter", nil),
		tapflows.WithCreateField("microversion", "ignored"))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != pythonTapFlowBase || calls[0].query != "" || value.StatusCode != 201 {
		t.Fatal(value, err, calls)
	}
	pythonTapFlowBody(t, calls[0].body, map[string]any{"name": "mirror", "tap_service_id": "ts1", "source_port": "port1", "direction": "BOTH", "vlan_filter": nil})
	pythonTapFlowCheck(t, value)

	calls = nil
	if _, err := api.Create(context.Background()); err != nil || len(calls) != 1 {
		t.Fatal(err, calls)
	}
	pythonTapFlowBody(t, calls[0].body, map[string]any{})
}

func TestPythonTapFlowStatusesCarryOperationContext(t *testing.T) {
	for _, code := range []int{200, 202, 302} {
		var calls []pythonTapFlowCall
		api := pythonTapFlowAPI(t, &calls, func(*http.Request) (int, string) { return code, `{"tap_flow":` + pythonTapFlowRow + `}` })
		if value, err := api.Get(context.Background(), "tf1"); err != nil || value.StatusCode != code {
			t.Fatal(code, value, err)
		}
	}
	for _, code := range []int{400, 409, 500} {
		var calls []pythonTapFlowCall
		api := pythonTapFlowAPI(t, &calls, func(*http.Request) (int, string) { return code, `{"NeutronError":{}}` })
		_, err := api.Create(context.Background(), tapflows.WithCreateField("name", "x"))
		pythonTapFlowOperation(t, err, "Create")
		pythonTapFlowStatus(t, err, code)
		_, err = api.Get(context.Background(), "tf1")
		pythonTapFlowOperation(t, err, "Get")
		pythonTapFlowStatus(t, err, code)
		_, err = api.Update(context.Background(), "tf1", tapflows.WithUpdateField("name", "x"))
		pythonTapFlowOperation(t, err, "Update")
		pythonTapFlowStatus(t, err, code)
	}
}

func TestPythonTapFlowGetMapsFields(t *testing.T) {
	var calls []pythonTapFlowCall
	api := pythonTapFlowAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"tap_flow":` + pythonTapFlowRow + `}` })
	value, err := api.Get(context.Background(), "tf 1", tapflows.WithGetFields("name", "project_id"))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != pythonTapFlowBase+"/tf%201" || calls[0].query != "fields=name&fields=project_id" {
		t.Fatal(value, err, calls)
	}
	pythonTapFlowCheck(t, value)
	calls = nil
	if _, err := api.Get(context.Background(), "tf1", tapflows.WithGetFields()); err != nil || calls[0].query != "" {
		t.Fatal(err, calls)
	}
	_, err = api.Get(context.Background(), "")
	pythonTapFlowOperation(t, err, "Get")
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
		t.Fatal(err, calls)
	}
}

func TestPythonTapFlowUpdatePutsDirtyAttrs(t *testing.T) {
	var calls []pythonTapFlowCall
	api := pythonTapFlowAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"tap_flow":` + pythonTapFlowRow + `}` })
	value, err := api.Update(context.Background(), "tf1", tapflows.WithUpdateField("name", "mirror"), tapflows.WithUpdateField("description", ""))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != pythonTapFlowBase+"/tf1" {
		t.Fatal(value, err, calls)
	}
	pythonTapFlowBody(t, calls[0].body, map[string]any{"name": "mirror", "description": ""})
	pythonTapFlowCheck(t, value)

	calls = nil
	value, err = api.Update(context.Background(), "tf1")
	if err != nil || len(calls) != 0 || value.ID != "tf1" || value.Name != nil || value.Wire != nil {
		t.Fatal(value, err, calls)
	}
	for _, key := range []string{"id", "tap_flow"} {
		_, err = api.Update(context.Background(), "tf1", tapflows.WithUpdateField(key, "x"))
		pythonTapFlowOperation(t, err, "Update")
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(key, err, calls)
		}
	}
}

func TestPythonTapFlowDeleteIgnoresMissingByDefault(t *testing.T) {
	for _, tc := range []struct {
		code    int
		options []tapflows.DeleteOption
		fail    bool
	}{
		{204, nil, false},
		{404, nil, false},
		{404, []tapflows.DeleteOption{tapflows.WithDeleteIgnoreMissing(false)}, true},
		{409, nil, true},
	} {
		var calls []pythonTapFlowCall
		api := pythonTapFlowAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, "" })
		err := api.Delete(context.Background(), "tf1", tc.options...)
		if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != pythonTapFlowBase+"/tf1" || calls[0].body != "" {
			t.Fatal(calls)
		}
		if !tc.fail {
			if err != nil {
				t.Fatal(tc.code, err)
			}
			continue
		}
		pythonTapFlowOperation(t, err, "Delete")
		pythonTapFlowStatus(t, err, tc.code)
	}
}

func TestPythonTapFlowListQueryFiltersAndLinks(t *testing.T) {
	var calls []pythonTapFlowCall
	api := pythonTapFlowAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("marker") == "" {
			return 200, `{"tap_flows":[{"id":"a","direction":"IN"},{"id":"b","direction":"OUT"}],"tap_flows_links":[{"rel":"next","href":"/v2.0/taas/tap_flows?marker=b&name=n"}]}`
		}
		return 200, `{"tap_flows":[{"id":"c","direction":"OUT"}]}`
	})
	values, err := api.All(context.Background(), tapflows.WithListQuery("name", "n"), tapflows.WithListQuery("direction", "OUT"), tapflows.WithListQuery("unknown", "dropped"))
	if err != nil || len(values) != 2 || values[0].ID != "b" || values[1].ID != "c" || len(calls) != 2 {
		t.Fatal(values, err, calls)
	}
	if calls[0].path != pythonTapFlowBase || calls[0].query != "name=n" || calls[0].accept != "application/json" || calls[1].query != "marker=b&name=n" {
		t.Fatal(calls)
	}
}

func TestPythonTapFlowListMarkerContinuation(t *testing.T) {
	var calls []pythonTapFlowCall
	api := pythonTapFlowAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.URL.Query().Get("marker") {
		case "":
			return 200, `{"tap_flows":[{"id":"a"},{"id":"b"}]}`
		case "b":
			return 200, `{"tap_flows":[{"id":"c"}]}`
		}
		return 200, `{"tap_flows":[]}`
	})
	values, err := api.All(context.Background(), tapflows.WithListLimit(2), tapflows.WithListQuery("project_id", "p1"))
	if err != nil || len(values) != 3 || len(calls) != 3 {
		t.Fatal(values, err, calls)
	}
	if calls[0].query != "limit=2&project_id=p1" || calls[1].query != "limit=2&marker=b&project_id=p1" || calls[2].query != "limit=2&marker=c&project_id=p1" {
		t.Fatal(calls)
	}
	calls = nil
	values, err = api.All(context.Background(), tapflows.WithListLimit(2), tapflows.WithListPaginated(false))
	if err != nil || len(values) != 2 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	calls = nil
	api = pythonTapFlowAPI(t, &calls, func(*http.Request) (int, string) { return 503, "" })
	_, err = api.All(context.Background())
	pythonTapFlowOperation(t, err, "List")
	pythonTapFlowStatus(t, err, 503)
}

func TestPythonTapFlowFindFallsBackToNameList(t *testing.T) {
	var calls []pythonTapFlowCall
	api := pythonTapFlowAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == pythonTapFlowBase+"/mirror" {
			return 404, `{"NeutronError":{}}`
		}
		return 200, `{"tap_flows":[` + pythonTapFlowRow + `]}`
	})
	value, err := api.Find(context.Background(), "mirror")
	if err != nil || len(calls) != 2 || calls[0].method != http.MethodGet || calls[1].path != pythonTapFlowBase || calls[1].query != "name=mirror" {
		t.Fatal(value, err, calls)
	}
	pythonTapFlowCheck(t, value)

	calls = nil
	api = pythonTapFlowAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == pythonTapFlowBase+"/tf1" {
			return 200, `{"tap_flow":` + pythonTapFlowRow + `}`
		}
		return 500, ""
	})
	value, err = api.Find(context.Background(), "tf1", tapflows.WithFindQuery("project_id", "p1"))
	if err != nil || len(calls) != 1 || calls[0].query != "project_id=p1" {
		t.Fatal(value, err, calls)
	}
	pythonTapFlowCheck(t, value)
}

func TestPythonTapFlowFindMissing(t *testing.T) {
	var calls []pythonTapFlowCall
	api := pythonTapFlowAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == pythonTapFlowBase+"/none" {
			return 404, ""
		}
		return 200, `{"tap_flows":[]}`
	})
	value, err := api.Find(context.Background(), "none")
	if err != nil || value != nil || len(calls) != 2 {
		t.Fatal(value, err, calls)
	}
	_, err = api.Find(context.Background(), "none", tapflows.WithFindIgnoreMissing(false))
	pythonTapFlowOperation(t, err, "Find")
	var missing *resource.NotFoundError
	if !errors.As(err, &missing) {
		t.Fatal(err)
	}
	calls = nil
	api = pythonTapFlowAPI(t, &calls, func(*http.Request) (int, string) { return 500, "" })
	_, err = api.Find(context.Background(), "x")
	pythonTapFlowOperation(t, err, "Find")
	if len(calls) != 1 {
		t.Fatal(calls)
	}
	pythonTapFlowStatus(t, err, 500)
}
