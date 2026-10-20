package endpointgroups_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpointgroups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonCloudEndpointGroupCreate(t *testing.T) {
	var calls []pythonEndpointGroupCall
	api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) {
		return 201, `{"endpoint_group":{"id":"eg1","name":"web","filters":{"service_id":"s1"}}}`
	})
	v, err := api.CloudCreate(context.Background(), "web", map[string]string{"service_id": "s1"}, "")
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != pythonEndpointGroupBase || calls[0].body != `{"endpoint_group":{"filters":{"service_id":"s1"},"name":"web"}}` {
		t.Fatal("empty description is not sent", err, calls)
	}
	pythonEndpointGroupField(t, v, "id", `"eg1"`)
	calls = nil
	if _, err := api.CloudCreate(context.Background(), "web", map[string]string{}, "d"); err != nil || calls[0].body != `{"endpoint_group":{"description":"d","filters":{},"name":"web"}}` {
		t.Fatal(err, calls)
	}
	api = pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) { return 400, `bad` })
	_, err = api.CloudCreate(context.Background(), "web", nil, "")
	pythonEndpointGroupOperation(t, err, "CloudCreate")
}

func TestPythonCloudEndpointGroupDelete(t *testing.T) {
	t.Run("missing returns false without DELETE", func(t *testing.T) {
		var calls []pythonEndpointGroupCall
		api := pythonEndpointGroupAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/eg1") {
				return 404, "missing"
			}
			return 200, `{"endpoint_groups":[]}`
		})
		deleted, err := api.CloudDelete(context.Background(), "eg1")
		if err != nil || deleted || len(calls) != 2 || calls[1].query != "name=eg1" {
			t.Fatal(deleted, err, calls)
		}
	})
	t.Run("found deletes the literal argument", func(t *testing.T) {
		var calls []pythonEndpointGroupCall
		api := pythonEndpointGroupAPI(t, &calls, func(req *http.Request) (int, string) {
			switch {
			case req.Method == http.MethodDelete:
				return 404, "gone"
			case strings.HasSuffix(req.URL.Path, "/web"):
				return 404, "missing"
			}
			return 200, `{"endpoint_groups":[{"id":"eg1","name":"web"}]}`
		})
		deleted, err := api.CloudDelete(context.Background(), "web")
		if err != nil || !deleted || len(calls) != 3 || calls[2].method != http.MethodDelete || calls[2].path != pythonEndpointGroupBase+"/web" {
			t.Fatal("Python passes id, not the found ID, and ignores the 404", deleted, err, calls)
		}
	})
	t.Run("delete failure returns false and the error", func(t *testing.T) {
		var calls []pythonEndpointGroupCall
		api := pythonEndpointGroupAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.Method == http.MethodDelete {
				return 409, "conflict"
			}
			return 200, `{"endpoint_group":{"id":"eg1"}}`
		})
		deleted, err := api.CloudDelete(context.Background(), "eg1")
		var native gophercloud.ErrUnexpectedResponseCode
		if deleted || !errors.As(err, &native) || native.Actual != 409 || len(calls) != 2 {
			t.Fatal(deleted, err, calls)
		}
		pythonEndpointGroupOperation(t, err, "CloudDelete")
	})
}

func TestPythonCloudEndpointGroupSearch(t *testing.T) {
	var calls []pythonEndpointGroupCall
	api := pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"endpoint_groups":[{"id":"a1","name":"web-public","filters":{"interface":"public"}},{"id":"b2","name":"web-admin","filters":{"interface":"admin"}},{"id":"c3","name":"db","filters":{"interface":"public"}}]}`
	})
	ids := func(result *endpointgroups.SearchResult) []string {
		var out []string
		for _, v := range result.EndpointGroups {
			out = append(out, v.Text("id"))
		}
		return out
	}
	result, err := api.CloudSearch(context.Background(), "web-*")
	if err != nil || !reflect.DeepEqual(ids(result), []string{"a1", "b2"}) || len(result.Inventory) != 3 || len(calls) != 1 || calls[0].query != "" {
		t.Fatal("glob on name with an unfiltered list", result, err, calls)
	}
	result, err = api.CloudSearch(context.Background(), "", endpointgroups.WithSearchFilters(json.RawMessage(`{"filters":{"interface":"public"}}`)))
	if err != nil || !reflect.DeepEqual(ids(result), []string{"a1", "c3"}) {
		t.Fatal("nested dictionary filter", result, err)
	}
	result, err = api.CloudSearch(context.Background(), "b2")
	if err != nil || !reflect.DeepEqual(ids(result), []string{"b2"}) {
		t.Fatal("exact ID", result, err)
	}
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(result.Value, &rows); err != nil || len(rows) != 1 || string(rows[0]["location"]) != "null" || len(rows[0]) != 6 {
		t.Fatal("Value holds the Python views", string(result.Value), err)
	}
	result, err = api.CloudSearch(context.Background(), "", endpointgroups.WithSearchExpression("[?filters.interface=='admin'].name"))
	if err != nil || result.EndpointGroups != nil || string(result.Value) != `["web-admin"]` {
		t.Fatal("JMESPath expression", result, err)
	}
	_, err = api.CloudSearch(context.Background(), "", endpointgroups.WithSearchFilters(json.RawMessage(`{"missing":1}`)))
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("_filter_list raises on a missing key", err)
	}
	pythonEndpointGroupOperation(t, err, "CloudSearch")
	api = pythonEndpointGroupAPI(t, &calls, func(*http.Request) (int, string) { return 500, `boom` })
	_, err = api.CloudSearch(context.Background(), "web")
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != 500 {
		t.Fatal(err)
	}
}
