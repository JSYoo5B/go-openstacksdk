package localips_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localips"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonLocalIPCall struct{ method, path, query, body, accept, ifMatch string }

const pythonLocalIPBase = "/neutron/v2.0/local_ips"

func pythonLocalIPAPI(t *testing.T, calls *[]pythonLocalIPCall, reply func(*http.Request) (int, string)) *localips.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonLocalIPCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Get("Accept"), req.Header.Get("If-Match")})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return localips.New(client)
}

func pythonLocalIPOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "localips" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
}

func pythonLocalIPStatus(t *testing.T, err error, code int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

func pythonLocalIPBody(t *testing.T, raw string, want map[string]any) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal([]byte(raw), &got); err != nil || !reflect.DeepEqual(got, map[string]any{"local_ip": want}) {
		t.Fatalf("body %s, want %v (%v)", raw, want, err)
	}
}

const pythonLocalIPRow = `{"id":"lip1","name":"edge","description":"d","project_id":"p1","network_id":"net1","local_port_id":"port1","local_ip_address":"10.0.0.5","ip_mode":"translate","revision_number":3,"created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-02T00:00:00Z","x_ext":true}`

func pythonLocalIPCheck(t *testing.T, v *localips.LocalIP) {
	t.Helper()
	if v == nil || v.ID != "lip1" || *v.Name != "edge" || *v.Description != "d" || *v.ProjectID != "p1" || *v.NetworkID != "net1" || *v.LocalPortID != "port1" || *v.LocalIPAddress != "10.0.0.5" || *v.IPMode != "translate" || *v.RevisionNumber != 3 || *v.CreatedAt != "2024-01-01T00:00:00Z" || *v.UpdatedAt != "2024-01-02T00:00:00Z" {
		t.Fatalf("decoded local IP: %#v", v)
	}
	if string(v.Resource.Body["x_ext"]) != "true" || string(v.Resource.Body["location"]) != "null" || v.Resource.Body["tenant_id"] != nil {
		t.Fatalf("raw view: %#v", v.Resource)
	}
}

func TestPythonLocalIPCreatePostsGivenAttrs(t *testing.T) {
	var calls []pythonLocalIPCall
	api := pythonLocalIPAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{"local_ip":` + pythonLocalIPRow + `}` })
	value, err := api.Create(context.Background(),
		localips.WithCreateField("name", "edge"),
		localips.WithCreateField("network_id", "net1"),
		localips.WithCreateField("local_ip_address", "10.0.0.5"),
		localips.WithCreateField("ip_mode", "translate"),
		localips.WithCreateField("x_unknown", 0))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != pythonLocalIPBase || calls[0].query != "" || value.StatusCode != 201 {
		t.Fatal(value, err, calls)
	}
	pythonLocalIPBody(t, calls[0].body, map[string]any{"name": "edge", "network_id": "net1", "local_ip_address": "10.0.0.5", "ip_mode": "translate", "x_unknown": float64(0)})
	pythonLocalIPCheck(t, value)

	for _, code := range []int{400, 500} {
		calls = nil
		api = pythonLocalIPAPI(t, &calls, func(*http.Request) (int, string) { return code, `{"NeutronError":{}}` })
		_, err = api.Create(context.Background())
		pythonLocalIPOperation(t, err, "Create")
		pythonLocalIPStatus(t, err, code)
	}
}

func TestPythonLocalIPProjectionKeepsUntypedRevisionAndNoTenantAlias(t *testing.T) {
	var calls []pythonLocalIPCall
	api := pythonLocalIPAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"local_ip":{"id":"lip1","tenant_id":"t1","revision_number":"7"}}`
	})
	value, err := api.Get(context.Background(), "lip1")
	if err != nil || value.ProjectID != nil || value.RevisionNumber != nil || string(value.Resource.Body["revision_number"]) != `"7"` || string(value.Resource.Body["tenant_id"]) != `"t1"` {
		t.Fatal(value, err)
	}
}

func TestPythonLocalIPGetMapsFields(t *testing.T) {
	var calls []pythonLocalIPCall
	api := pythonLocalIPAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"local_ip":` + pythonLocalIPRow + `}` })
	value, err := api.Get(context.Background(), "lip/1", localips.WithGetFields("name", "ip_mode"))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != pythonLocalIPBase+"/lip%2F1" || calls[0].query != "fields=name&fields=ip_mode" {
		t.Fatal(value, err, calls)
	}
	pythonLocalIPCheck(t, value)
	calls = nil
	api = pythonLocalIPAPI(t, &calls, func(*http.Request) (int, string) { return 404, "" })
	_, err = api.Get(context.Background(), "lip1")
	pythonLocalIPOperation(t, err, "Get")
	pythonLocalIPStatus(t, err, 404)
}

func TestPythonLocalIPUpdateIgnoresRevisionPrecondition(t *testing.T) {
	var calls []pythonLocalIPCall
	api := pythonLocalIPAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"local_ip":` + pythonLocalIPRow + `}` })
	value, err := api.Update(context.Background(), "lip1", localips.WithUpdateField("description", "d"))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != pythonLocalIPBase+"/lip1" || calls[0].ifMatch != "" {
		t.Fatal(value, err, calls)
	}
	pythonLocalIPBody(t, calls[0].body, map[string]any{"description": "d"})
	pythonLocalIPCheck(t, value)

	calls = nil
	condition := map[string]string{"If-Match": "revision_number=3"}
	_, err = api.Update(context.Background(), "lip1", localips.WithUpdateField("name", "n"), localips.WithUpdateHeaders(condition))
	if err != nil || len(calls) != 1 || calls[0].ifMatch != "revision_number=3" {
		t.Fatal(err, calls)
	}
	if err = api.Delete(context.Background(), "lip1", localips.WithDeleteHeaders(condition)); err != nil || len(calls) != 2 || calls[1].ifMatch != "revision_number=3" {
		t.Fatal(err, calls)
	}

	calls = nil
	value, err = api.Update(context.Background(), "lip1")
	if err != nil || len(calls) != 0 || value.ID != "lip1" || value.Wire != nil {
		t.Fatal(value, err, calls)
	}
	for _, key := range []string{"id", "local_ip", "if_revision"} {
		_, err = api.Update(context.Background(), "lip1", localips.WithUpdateField(key, 1))
		pythonLocalIPOperation(t, err, "Update")
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(key, err, calls)
		}
	}
}

func TestPythonLocalIPDeleteIgnoresMissingByDefault(t *testing.T) {
	for _, tc := range []struct {
		code    int
		options []localips.DeleteOption
		fail    bool
	}{
		{204, nil, false},
		{404, nil, false},
		{404, []localips.DeleteOption{localips.WithDeleteIgnoreMissing(false)}, true},
		{409, nil, true},
	} {
		var calls []pythonLocalIPCall
		api := pythonLocalIPAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, "" })
		err := api.Delete(context.Background(), "lip1", tc.options...)
		if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != pythonLocalIPBase+"/lip1" || calls[0].ifMatch != "" {
			t.Fatal(calls)
		}
		if !tc.fail {
			if err != nil {
				t.Fatal(tc.code, err)
			}
			continue
		}
		pythonLocalIPOperation(t, err, "Delete")
		pythonLocalIPStatus(t, err, tc.code)
	}
}

func TestPythonLocalIPListQueryFiltersAndLinks(t *testing.T) {
	var calls []pythonLocalIPCall
	api := pythonLocalIPAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("marker") == "" {
			return 200, `{"local_ips":[{"id":"a","revision_number":1},{"id":"b","revision_number":2}],"local_ips_links":[{"rel":"next","href":"/v2.0/local_ips?marker=b&network_id=net1"}]}`
		}
		return 200, `{"local_ips":[{"id":"c","revision_number":2}]}`
	})
	values, err := api.All(context.Background(), localips.WithListQuery("network_id", "net1"), localips.WithListQuery("revision_number", 2), localips.WithListQuery("if_revision", 1))
	if err != nil || len(values) != 2 || values[0].ID != "b" || values[1].ID != "c" || len(calls) != 2 {
		t.Fatal(values, err, calls)
	}
	if calls[0].path != pythonLocalIPBase || calls[0].query != "network_id=net1" || calls[0].accept != "application/json" || calls[1].query != "marker=b&network_id=net1" {
		t.Fatal(calls)
	}
	calls = nil
	api = pythonLocalIPAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("marker") == "" {
			return 200, `{"local_ips":[{"id":"a"}]}`
		}
		return 200, `{"local_ips":[]}`
	})
	values, err = api.All(context.Background(), localips.WithListLimit(1), localips.WithListQuery("ip_mode", "passthrough"), localips.WithListQuery("local_ip_address", "10.0.0.5"))
	if err != nil || len(values) != 1 || len(calls) != 2 || calls[0].query != "ip_mode=passthrough&limit=1&local_ip_address=10.0.0.5" || calls[1].query != "ip_mode=passthrough&limit=1&local_ip_address=10.0.0.5&marker=a" {
		t.Fatal(values, err, calls)
	}
	calls = nil
	api = pythonLocalIPAPI(t, &calls, func(*http.Request) (int, string) { return 500, "" })
	_, err = api.All(context.Background())
	pythonLocalIPOperation(t, err, "List")
	pythonLocalIPStatus(t, err, 500)
}

func TestPythonLocalIPFindFallsBackToNameList(t *testing.T) {
	var calls []pythonLocalIPCall
	api := pythonLocalIPAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == pythonLocalIPBase+"/edge" {
			return 400, ""
		}
		return 200, `{"local_ips":[` + pythonLocalIPRow + `]}`
	})
	value, err := api.Find(context.Background(), "edge")
	if err != nil || len(calls) != 2 || calls[1].path != pythonLocalIPBase || calls[1].query != "name=edge" {
		t.Fatal(value, err, calls)
	}
	pythonLocalIPCheck(t, value)

	calls = nil
	api = pythonLocalIPAPI(t, &calls, func(req *http.Request) (int, string) {
		if strings.HasPrefix(req.URL.Path, pythonLocalIPBase+"/") {
			return 404, ""
		}
		return 200, `{"local_ips":[{"id":"x","name":"dup"},{"id":"y","name":"dup"}]}`
	})
	_, err = api.Find(context.Background(), "dup")
	pythonLocalIPOperation(t, err, "Find")
	value, err = api.Find(context.Background(), "none")
	if err != nil || value != nil {
		t.Fatal(value, err)
	}
	_, err = api.Find(context.Background(), "none", localips.WithFindIgnoreMissing(false))
	var missing *resource.NotFoundError
	if !errors.As(err, &missing) {
		t.Fatal(err)
	}
	pythonLocalIPOperation(t, err, "Find")
}
