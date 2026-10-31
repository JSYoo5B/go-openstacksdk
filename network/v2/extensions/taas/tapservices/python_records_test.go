package tapservices_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapservices"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonTapServiceCall struct{ method, path, query, body, accept string }

const pythonTapServiceBase = "/neutron/v2.0/taas/tap_services"

func pythonTapServiceAPI(t *testing.T, calls *[]pythonTapServiceCall, reply func(*http.Request) (int, string)) *tapservices.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonTapServiceCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Get("Accept")})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return tapservices.New(client)
}

func pythonTapServiceOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "tapservices" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
}

func pythonTapServiceStatus(t *testing.T, err error, code int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

func pythonTapServiceBody(t *testing.T, raw string, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal([]byte(raw), &got); err != nil || !reflect.DeepEqual(got, map[string]any{"tap_service": want}) {
		t.Fatalf("body %s, want %v (%v)", raw, want, err)
	}
}

const pythonTapServiceRow = `{"id":"ts1","name":"collector","description":"d","tenant_id":"p1","port_id":"port1","status":"ACTIVE","x_ext":7}`

func pythonTapServiceCheck(t *testing.T, v *tapservices.TapService) {
	t.Helper()
	if v == nil || v.ID != "ts1" || *v.Name != "collector" || *v.Description != "d" || *v.ProjectID != "p1" || *v.TenantID != "p1" || *v.PortID != "port1" || *v.Status != "ACTIVE" {
		t.Fatalf("decoded tap service: %#v", v)
	}
	if string(v.Resource.Body["x_ext"]) != "7" || string(v.Resource.Body["location"]) != "null" || string(v.Wire.Body["tenant_id"]) != `"p1"` {
		t.Fatalf("raw views: %#v %#v", v.Resource, v.Wire)
	}
}

func TestPythonTapServiceCreatePostsGivenAttrs(t *testing.T) {
	var calls []pythonTapServiceCall
	api := pythonTapServiceAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{"tap_service":` + pythonTapServiceRow + `}` })
	value, err := api.Create(context.Background(),
		tapservices.WithCreateField("name", "collector"),
		tapservices.WithCreateField("port_id", "port1"),
		tapservices.WithCreateField("description", "d"),
		tapservices.WithCreateField("vlan_filter", nil),
		tapservices.WithCreateField("microversion", "ignored"))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != pythonTapServiceBase || calls[0].query != "" || value.StatusCode != 201 {
		t.Fatal(value, err, calls)
	}
	pythonTapServiceBody(t, calls[0].body, map[string]any{"name": "collector", "port_id": "port1", "description": "d", "vlan_filter": nil})
	pythonTapServiceCheck(t, value)

	calls = nil
	if _, err := api.Create(context.Background()); err != nil || len(calls) != 1 {
		t.Fatal(err, calls)
	}
	pythonTapServiceBody(t, calls[0].body, map[string]any{})
}

func TestPythonTapServiceStatusesCarryOperationContext(t *testing.T) {
	for _, code := range []int{200, 202, 302} {
		var calls []pythonTapServiceCall
		api := pythonTapServiceAPI(t, &calls, func(*http.Request) (int, string) { return code, `{"tap_service":` + pythonTapServiceRow + `}` })
		if value, err := api.Get(context.Background(), "ts1"); err != nil || value.StatusCode != code {
			t.Fatal(code, value, err)
		}
	}
	for _, code := range []int{400, 409, 500} {
		var calls []pythonTapServiceCall
		api := pythonTapServiceAPI(t, &calls, func(*http.Request) (int, string) { return code, `{"NeutronError":{}}` })
		_, err := api.Create(context.Background(), tapservices.WithCreateField("name", "x"))
		pythonTapServiceOperation(t, err, "Create")
		pythonTapServiceStatus(t, err, code)
		_, err = api.Get(context.Background(), "ts1")
		pythonTapServiceOperation(t, err, "Get")
		pythonTapServiceStatus(t, err, code)
		_, err = api.Update(context.Background(), "ts1", tapservices.WithUpdateField("name", "x"))
		pythonTapServiceOperation(t, err, "Update")
		pythonTapServiceStatus(t, err, code)
	}
}

func TestPythonTapServiceGetMapsFields(t *testing.T) {
	var calls []pythonTapServiceCall
	api := pythonTapServiceAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"tap_service":` + pythonTapServiceRow + `}` })
	value, err := api.Get(context.Background(), "ts 1", tapservices.WithGetFields("name", "project_id"))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != pythonTapServiceBase+"/ts%201" || calls[0].query != "fields=name&fields=project_id" {
		t.Fatal(value, err, calls)
	}
	pythonTapServiceCheck(t, value)
	calls = nil
	if _, err := api.Get(context.Background(), "ts1", tapservices.WithGetFields()); err != nil || calls[0].query != "" {
		t.Fatal(err, calls)
	}
	_, err = api.Get(context.Background(), "")
	pythonTapServiceOperation(t, err, "Get")
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
		t.Fatal(err, calls)
	}
}

func TestPythonTapServiceUpdatePutsDirtyAttrs(t *testing.T) {
	var calls []pythonTapServiceCall
	api := pythonTapServiceAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"tap_service":` + pythonTapServiceRow + `}` })
	value, err := api.Update(context.Background(), "ts1", tapservices.WithUpdateField("name", "collector"), tapservices.WithUpdateField("description", ""))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != pythonTapServiceBase+"/ts1" {
		t.Fatal(value, err, calls)
	}
	pythonTapServiceBody(t, calls[0].body, map[string]any{"name": "collector", "description": ""})
	pythonTapServiceCheck(t, value)

	calls = nil
	value, err = api.Update(context.Background(), "ts1")
	if err != nil || len(calls) != 0 || value.ID != "ts1" || value.Name != nil || value.Wire != nil {
		t.Fatal(value, err, calls)
	}
	for _, key := range []string{"id", "tap_service"} {
		_, err = api.Update(context.Background(), "ts1", tapservices.WithUpdateField(key, "x"))
		pythonTapServiceOperation(t, err, "Update")
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(key, err, calls)
		}
	}
}

func TestPythonTapServiceDeleteIgnoresMissingByDefault(t *testing.T) {
	for _, tc := range []struct {
		code    int
		options []tapservices.DeleteOption
		fail    bool
	}{
		{204, nil, false},
		{404, nil, false},
		{404, []tapservices.DeleteOption{tapservices.WithDeleteIgnoreMissing(false)}, true},
		{409, nil, true},
	} {
		var calls []pythonTapServiceCall
		api := pythonTapServiceAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, "" })
		err := api.Delete(context.Background(), "ts1", tc.options...)
		if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != pythonTapServiceBase+"/ts1" || calls[0].body != "" {
			t.Fatal(calls)
		}
		if !tc.fail {
			if err != nil {
				t.Fatal(tc.code, err)
			}
			continue
		}
		pythonTapServiceOperation(t, err, "Delete")
		pythonTapServiceStatus(t, err, tc.code)
	}
}

func TestPythonTapServiceListQueryFiltersAndLinks(t *testing.T) {
	var calls []pythonTapServiceCall
	api := pythonTapServiceAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("marker") == "" {
			return 200, `{"tap_services":[{"id":"a","status":"DOWN"},{"id":"b","status":"ACTIVE"}],"tap_services_links":[{"rel":"next","href":"/v2.0/taas/tap_services?marker=b&name=n"}]}`
		}
		return 200, `{"tap_services":[{"id":"c","status":"ACTIVE"}]}`
	})
	values, err := api.All(context.Background(), tapservices.WithListQuery("name", "n"), tapservices.WithListQuery("status", "ACTIVE"), tapservices.WithListQuery("unknown", "dropped"))
	if err != nil || len(values) != 2 || values[0].ID != "b" || values[1].ID != "c" || len(calls) != 2 {
		t.Fatal(values, err, calls)
	}
	if calls[0].path != pythonTapServiceBase || calls[0].query != "name=n" || calls[0].accept != "application/json" || calls[1].query != "marker=b&name=n" {
		t.Fatal(calls)
	}
}

func TestPythonTapServiceListMarkerContinuation(t *testing.T) {
	var calls []pythonTapServiceCall
	api := pythonTapServiceAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.URL.Query().Get("marker") {
		case "":
			return 200, `{"tap_services":[{"id":"a"},{"id":"b"}]}`
		case "b":
			return 200, `{"tap_services":[{"id":"c"}]}`
		}
		return 200, `{"tap_services":[]}`
	})
	values, err := api.All(context.Background(), tapservices.WithListLimit(2), tapservices.WithListQuery("project_id", "p1"))
	if err != nil || len(values) != 3 || len(calls) != 3 {
		t.Fatal(values, err, calls)
	}
	if calls[0].query != "limit=2&project_id=p1" || calls[1].query != "limit=2&marker=b&project_id=p1" || calls[2].query != "limit=2&marker=c&project_id=p1" {
		t.Fatal(calls)
	}
	calls = nil
	values, err = api.All(context.Background(), tapservices.WithListLimit(2), tapservices.WithListPaginated(false))
	if err != nil || len(values) != 2 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	calls = nil
	api = pythonTapServiceAPI(t, &calls, func(*http.Request) (int, string) { return 503, "" })
	_, err = api.All(context.Background())
	pythonTapServiceOperation(t, err, "List")
	pythonTapServiceStatus(t, err, 503)
}

func TestPythonTapServiceFindFallsBackToNameList(t *testing.T) {
	var calls []pythonTapServiceCall
	api := pythonTapServiceAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == pythonTapServiceBase+"/collector" {
			return 404, `{"NeutronError":{}}`
		}
		return 200, `{"tap_services":[` + pythonTapServiceRow + `]}`
	})
	value, err := api.Find(context.Background(), "collector")
	if err != nil || len(calls) != 2 || calls[0].method != http.MethodGet || calls[1].path != pythonTapServiceBase || calls[1].query != "name=collector" {
		t.Fatal(value, err, calls)
	}
	pythonTapServiceCheck(t, value)

	calls = nil
	api = pythonTapServiceAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == pythonTapServiceBase+"/ts1" {
			return 200, `{"tap_service":` + pythonTapServiceRow + `}`
		}
		return 500, ""
	})
	value, err = api.Find(context.Background(), "ts1", tapservices.WithFindQuery("project_id", "p1"))
	if err != nil || len(calls) != 1 || calls[0].query != "project_id=p1" {
		t.Fatal(value, err, calls)
	}
	pythonTapServiceCheck(t, value)
}

func TestPythonTapServiceFindMissing(t *testing.T) {
	var calls []pythonTapServiceCall
	api := pythonTapServiceAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == pythonTapServiceBase+"/none" {
			return 404, ""
		}
		return 200, `{"tap_services":[]}`
	})
	value, err := api.Find(context.Background(), "none")
	if err != nil || value != nil || len(calls) != 2 {
		t.Fatal(value, err, calls)
	}
	_, err = api.Find(context.Background(), "none", tapservices.WithFindIgnoreMissing(false))
	pythonTapServiceOperation(t, err, "Find")
	var missing *resource.NotFoundError
	if !errors.As(err, &missing) {
		t.Fatal(err)
	}
	calls = nil
	api = pythonTapServiceAPI(t, &calls, func(*http.Request) (int, string) { return 500, "" })
	_, err = api.Find(context.Background(), "x")
	pythonTapServiceOperation(t, err, "Find")
	if len(calls) != 1 {
		t.Fatal(calls)
	}
	pythonTapServiceStatus(t, err, 500)
}
