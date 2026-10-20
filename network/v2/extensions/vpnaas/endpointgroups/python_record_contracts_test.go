package endpointgroups_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"net/http"
	"testing"
)

func TestVPNEndpointGroupRecordsAllPagesServerQueryAndLocalFilter(t *testing.T) {
	var base string
	api, calls, cloud := pythonEndpointGroupAPI(t, func(r *http.Request) *http.Response {
		if r.URL.Query().Get("marker") == "next" {
			return pythonEndpointGroupWire(200, `{"endpoint_groups":[{"id":"last","endpoints":["a"]}]}`)
		}
		return pythonEndpointGroupWire(200, `{"endpoint_groups":[{"id":"skip","endpoints":["b"]},{"id":"first","endpoints":["a"]}],"next":"`+base+`?marker=next"}`)
	})
	base = cloud.Server.URL + pythonEndpointGroupBase
	rows, err := api.AllRecords(context.Background(), endpointgroups.WithRecordListQuery("endpoints", json.RawMessage(`["a"]`)), endpointgroups.WithRecordListQuery("type", json.RawMessage(`"cidr"`)), endpointgroups.WithRecordListQuery("tenant_id", "tenant"))
	if err != nil || len(rows) != 2 || len(*calls) != 2 || string(rows[0].Resource.Body["id"]) != `"first"` || string(rows[1].Resource.Body["id"]) != `"last"` {
		t.Fatal(rows, err, *calls)
	}
	for _, call := range *calls {
		if call.method != "GET" || call.path != pythonEndpointGroupBase {
			t.Fatal(*calls)
		}
	}
	if (*calls)[0].query == "" {
		t.Fatal(*calls)
	}
	// Declared aliases are server-side; the local Body field must stay out of query.
	api, calls, _ = pythonEndpointGroupAPI(t, func(r *http.Request) *http.Response {
		if r.URL.Query().Get("endpoint_type") == "" || r.URL.Query().Get("endpoints") != "" || r.URL.Query().Get("tenant_id") != "tenant" {
			t.Error(r.URL)
		}
		return pythonEndpointGroupWire(200, `{"endpoint_groups":[]}`)
	})
	_, err = api.AllRecords(context.Background(), endpointgroups.WithRecordListQuery("endpoints", json.RawMessage(`["a"]`)), endpointgroups.WithRecordListQuery("type", json.RawMessage(`"cidr"`)), endpointgroups.WithRecordListQuery("tenant_id", "tenant"))
	if err != nil || len(*calls) != 1 {
		t.Fatal(err, *calls)
	}
}
func TestVPNEndpointGroupRecordsProjectionDefaultsAndUnknownFields(t *testing.T) {
	api, _, _ := pythonEndpointGroupAPI(t, func(*http.Request) *http.Response {
		return pythonEndpointGroupWire(203, `{"endpoint_groups":[{"id":"r","tenant_id":"owner","x_extension":{"n":9007199254740993},"endpoints":"a"}]}`)
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	row := rows[0]
	if row.StatusCode != 203 || string(row.Resource.Body["project_id"]) != `"owner"` || string(row.Resource.Body["name"]) != "null" || string(row.Wire.Body["x_extension"]) != `{"n":9007199254740993}` {
		t.Fatal(row)
	}
	if string(row.Resource.Body["endpoints"]) != `["a"]` {
		t.Fatal(row.Resource.Body)
	}
	row.Resource.Body["x_extension"][0] = 'X'
	if !json.Valid(row.Wire.Body["x_extension"]) || !json.Valid(row.Envelope) {
		t.Fatal("shared record evidence")
	}
}
func TestVPNEndpointGroupProxyJMESReturnsOpaqueGeneratorWithoutHTTP(t *testing.T) {
	api, calls, _ := pythonEndpointGroupAPI(t, func(*http.Request) *http.Response { return pythonEndpointGroupWire(200, `{"endpoint_groups":[]}`) })
	for _, tc := range []struct{ expression, json string }{{"[*].id", "null"}, {"type(@)", "null"}} {
		value, err := api.ProxyListRecords(context.Background(), endpointgroups.WithRecordListJMESPath(tc.expression))
		if err != nil || string(value.JSON) != tc.json || len(*calls) != 0 {
			t.Fatal(value, err, *calls)
		}
	}
	value, err := api.ProxyListRecords(context.Background(), endpointgroups.WithRecordListJMESPath("@"))
	if err != nil || len(*calls) != 0 {
		t.Fatal(value, err, *calls)
	}
	generator, ok := value.Value.(*endpointgroups.RecordListGenerator)
	if !ok {
		t.Fatal(value)
	}
	for _, err := range generator.Records {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(*calls) != 1 {
		t.Fatal(*calls)
	}
}
func TestVPNEndpointGroupRecordsOptionsCancellationAndFailures(t *testing.T) {
	count := 0
	api, calls, _ := pythonEndpointGroupAPI(t, func(*http.Request) *http.Response {
		count++
		return pythonEndpointGroupWire(200, `{"endpoint_groups":[{"id":"r","endpoints":["a"]}]}`)
	})
	raw := json.RawMessage(`["a"]`)
	option := endpointgroups.WithRecordListQuery("endpoints", raw)
	raw[0] = 'X'
	rows, err := api.AllRecords(context.Background(), option, endpointgroups.WithRecordListQuery("ignored_extension", make(chan int)))
	if err != nil || len(rows) != 1 || count != 1 {
		t.Fatal(rows, err)
	}
	for _, options := range [][]endpointgroups.RecordListOption{{nil}, {endpointgroups.WithRecordListQuery("session", make(chan int))}, {endpointgroups.WithRecordListMicroversion("2.1\r\nbad")}} {
		rows, err = api.AllRecords(context.Background(), options...)
		if err == nil || rows != nil || len(*calls) != 1 {
			t.Fatal(rows, err, *calls)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	rows, err = api.AllRecords(ctx, func(*endpointgroups.RecordListOpts) error { cancel(); return nil })
	if !errors.Is(err, context.Canceled) || rows != nil || len(*calls) != 1 {
		t.Fatal(rows, err, *calls)
	}
	api, _, _ = pythonEndpointGroupAPI(t, func(*http.Request) *http.Response { return pythonEndpointGroupWire(500, `{"error":"failed"}`) })
	rows, err = api.AllRecords(context.Background())
	if err == nil || rows != nil {
		t.Fatal(rows, err)
	}
	var operation *resource.OperationError
	if !errors.As(err, &operation) {
		t.Fatal(err)
	}
}
