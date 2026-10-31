package localipassociations_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localipassociations"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonLocalIPAssociationCall struct{ method, path, query, body, accept string }

const pythonLocalIPAssociationBase = "/neutron/v2.0/local_ips/lip1/port_associations"

func pythonLocalIPAssociationAPI(t *testing.T, calls *[]pythonLocalIPAssociationCall, reply func(*http.Request) (int, string)) *localipassociations.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonLocalIPAssociationCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Get("Accept")})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return localipassociations.New(client)
}

func pythonLocalIPAssociationOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "localipassociations" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
}

func pythonLocalIPAssociationStatus(t *testing.T, err error, code int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

const pythonLocalIPAssociationRow = `{"local_ip_id":"server-lip","local_ip_address":"10.0.0.5","fixed_port_id":"port1","fixed_ip":"192.168.0.4","host":"compute-1"}`

func pythonLocalIPAssociationCheck(t *testing.T, v *localipassociations.LocalIPAssociation) {
	t.Helper()
	if v == nil || v.LocalIPID != "lip1" || v.ID != "" || v.Name != nil || *v.FixedPortID != "port1" || *v.FixedIP != "192.168.0.4" || *v.Host != "compute-1" || *v.LocalIPAddress != "10.0.0.5" {
		t.Fatalf("decoded association: %#v", v)
	}
	if string(v.Resource.Body["id"]) != "null" || string(v.Resource.Body["location"]) != "null" || string(v.Wire.Body["local_ip_id"]) != `"server-lip"` {
		t.Fatalf("raw views: %#v %#v", v.Resource, v.Wire)
	}
}

func TestPythonLocalIPAssociationCreatePostsUnderParent(t *testing.T) {
	var calls []pythonLocalIPAssociationCall
	api := pythonLocalIPAssociationAPI(t, &calls, func(*http.Request) (int, string) {
		return 201, `{"port_association":` + pythonLocalIPAssociationRow + `}`
	})
	value, err := api.Create(context.Background(), "lip1", localipassociations.WithCreateField("fixed_port_id", "port1"), localipassociations.WithCreateField("fixed_ip", nil))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != pythonLocalIPAssociationBase || calls[0].query != "" || value.StatusCode != 201 {
		t.Fatal(value, err, calls)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(calls[0].body), &body); err != nil || !reflect.DeepEqual(body, map[string]any{"port_association": map[string]any{"fixed_port_id": "port1", "fixed_ip": nil}}) {
		t.Fatal(calls[0].body, err)
	}
	pythonLocalIPAssociationCheck(t, value)
	// A member response naming the URI attribute reads back as the bound parent.
	if string(value.Resource.Body["local_ip_id"]) != `"lip1"` {
		t.Fatal(value.Resource.Body)
	}

	calls = nil
	if _, err := api.Create(context.Background(), "lip /1"); err != nil || calls[0].path != "/neutron/v2.0/local_ips/lip%20%2F1/port_associations" {
		t.Fatal(err, calls)
	}
	calls = nil
	for _, key := range []string{"local_ip", "local_ip_id"} {
		_, err = api.Create(context.Background(), "lip1", localipassociations.WithCreateField(key, "x"))
		pythonLocalIPAssociationOperation(t, err, "Create")
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(key, err, calls)
		}
	}
	for _, parent := range []string{"", ".", "bad\n"} {
		_, err = api.Create(context.Background(), parent)
		pythonLocalIPAssociationOperation(t, err, "Create")
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(parent, err, calls)
		}
	}
	api = pythonLocalIPAssociationAPI(t, &calls, func(*http.Request) (int, string) { return 409, `{"NeutronError":{}}` })
	_, err = api.Create(context.Background(), "lip1")
	pythonLocalIPAssociationOperation(t, err, "Create")
	pythonLocalIPAssociationStatus(t, err, 409)
}

func TestPythonLocalIPAssociationGetUsesMemberRoute(t *testing.T) {
	var calls []pythonLocalIPAssociationCall
	api := pythonLocalIPAssociationAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"port_association":` + pythonLocalIPAssociationRow + `}`
	})
	value, err := api.Get(context.Background(), "lip1", "port1", localipassociations.WithGetFields("host", "fixed_ip"))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != pythonLocalIPAssociationBase+"/port1" || calls[0].query != "fields=host&fields=fixed_ip" {
		t.Fatal(value, err, calls)
	}
	// Resource.fetch keeps the requested id when the response omits it.
	if value.ID != "port1" || *value.FixedPortID != "port1" || *value.Host != "compute-1" || value.LocalIPID != "lip1" {
		t.Fatal(value)
	}
	api = pythonLocalIPAssociationAPI(t, &calls, func(*http.Request) (int, string) { return 404, "" })
	_, err = api.Get(context.Background(), "lip1", "port1")
	pythonLocalIPAssociationOperation(t, err, "Get")
	pythonLocalIPAssociationStatus(t, err, 404)
}

func TestPythonLocalIPAssociationDeleteIgnoresMissingByDefault(t *testing.T) {
	for _, tc := range []struct {
		code    int
		options []localipassociations.DeleteOption
		fail    bool
	}{
		{204, nil, false},
		{404, nil, false},
		{404, []localipassociations.DeleteOption{localipassociations.WithDeleteIgnoreMissing(false)}, true},
		{409, nil, true},
	} {
		var calls []pythonLocalIPAssociationCall
		api := pythonLocalIPAssociationAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, "" })
		err := api.Delete(context.Background(), "lip1", "port1", tc.options...)
		if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != pythonLocalIPAssociationBase+"/port1" || calls[0].body != "" {
			t.Fatal(calls)
		}
		if !tc.fail {
			if err != nil {
				t.Fatal(tc.code, err)
			}
			continue
		}
		pythonLocalIPAssociationOperation(t, err, "Delete")
		pythonLocalIPAssociationStatus(t, err, tc.code)
	}
}

func TestPythonLocalIPAssociationListQueryFiltersAndLinks(t *testing.T) {
	var calls []pythonLocalIPAssociationCall
	api := pythonLocalIPAssociationAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("marker") == "" {
			return 200, `{"port_associations":[{"fixed_port_id":"a","local_ip_address":"10.0.0.1"},{"fixed_port_id":"b","local_ip_address":"10.0.0.5"}],"port_associations_links":[{"rel":"next","href":"/v2.0/local_ips/lip1/port_associations?host=h&marker=b"}]}`
		}
		return 200, `{"port_associations":[{"fixed_port_id":"c","local_ip_address":"10.0.0.5"}]}`
	})
	values, err := api.All(context.Background(), "lip1", localipassociations.WithListQuery("host", "h"), localipassociations.WithListQuery("local_ip_address", "10.0.0.5"), localipassociations.WithListQuery("unknown", 1))
	if err != nil || len(values) != 2 || *values[0].FixedPortID != "b" || *values[1].FixedPortID != "c" || values[0].LocalIPID != "lip1" || len(calls) != 2 {
		t.Fatal(values, err, calls)
	}
	if values[0].Resource.Body["local_ip_id"] != nil {
		t.Fatal(values[0].Resource.Body)
	}
	if calls[0].path != pythonLocalIPAssociationBase || calls[0].query != "host=h" || calls[0].accept != "application/json" || calls[1].query != "host=h&marker=b" {
		t.Fatal(calls)
	}
	calls = nil
	for _, key := range []string{"local_ip", "local_ip_id"} {
		_, err = api.All(context.Background(), "lip1", localipassociations.WithListQuery(key, "x"))
		pythonLocalIPAssociationOperation(t, err, "List")
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(key, err, calls)
		}
	}
}

func TestPythonLocalIPAssociationListMarkerNeedsID(t *testing.T) {
	var calls []pythonLocalIPAssociationCall
	api := pythonLocalIPAssociationAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"port_associations":[{"fixed_port_id":"a"}]}`
	})
	// Python derives the marker from Resource.id, which port associations lack.
	_, err := api.All(context.Background(), "lip1", localipassociations.WithListLimit(1))
	pythonLocalIPAssociationOperation(t, err, "List")
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 || calls[0].query != "limit=1" {
		t.Fatal(err, calls)
	}
	calls = nil
	values, err := api.All(context.Background(), "lip1", localipassociations.WithListLimit(1), localipassociations.WithListPaginated(false))
	if err != nil || len(values) != 1 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
}

func TestPythonLocalIPAssociationFindForwardsParent(t *testing.T) {
	var calls []pythonLocalIPAssociationCall
	api := pythonLocalIPAssociationAPI(t, &calls, func(req *http.Request) (int, string) {
		if strings.HasPrefix(req.URL.Path, pythonLocalIPAssociationBase+"/") {
			return 404, ""
		}
		return 200, `{"port_associations":[{"id":"assoc","fixed_port_id":"port1"},{"fixed_port_id":"port2"}]}`
	})
	value, err := api.Find(context.Background(), "lip1", "assoc")
	if err != nil || value == nil || value.ID != "assoc" || value.LocalIPID != "lip1" || len(calls) != 2 {
		t.Fatal(value, err, calls)
	}
	if calls[0].query != "local_ip_id=lip1" || calls[1].path != pythonLocalIPAssociationBase || calls[1].query != "" {
		t.Fatal(calls)
	}

	calls = nil
	value, err = api.Find(context.Background(), "lip1", "port2", localipassociations.WithFindQuery("host", "h"))
	if err != nil || value != nil || len(calls) != 2 || calls[0].query != "host=h&local_ip_id=lip1" || calls[1].query != "host=h" {
		t.Fatal(value, err, calls)
	}
	_, err = api.Find(context.Background(), "lip1", "port2", localipassociations.WithFindIgnoreMissing(false))
	var missing *resource.NotFoundError
	if !errors.As(err, &missing) {
		t.Fatal(err)
	}
	pythonLocalIPAssociationOperation(t, err, "Find")
	calls = nil
	_, err = api.Find(context.Background(), "lip1", "x", localipassociations.WithFindQuery("local_ip", "y"))
	pythonLocalIPAssociationOperation(t, err, "Find")
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}

	api = pythonLocalIPAssociationAPI(t, &calls, func(req *http.Request) (int, string) {
		return 200, `{"port_association":` + pythonLocalIPAssociationRow + `}`
	})
	value, err = api.Find(context.Background(), "lip1", "port1")
	if err != nil || value == nil || len(calls) != 1 {
		t.Fatal(value, err, calls)
	}
}
