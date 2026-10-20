package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"net/http"
	"testing"
)

func TestVPNServiceRecordsAllPagesServerQueryAndLocalFilter(t *testing.T) {
	var base string
	api, calls, cloud := pythonVPNServiceAPI(t, func(r *http.Request) *http.Response {
		if r.URL.Query().Get("marker") == "next" {
			return pythonVPNServiceWire(200, `{"vpnservices":[{"id":"last","status":"ACTIVE"}]}`)
		}
		return pythonVPNServiceWire(200, `{"vpnservices":[{"id":"skip","status":"DOWN"},{"id":"first","status":"ACTIVE"}],"next":"`+base+`?marker=next"}`)
	})
	base = cloud.Server.URL + pythonVPNServiceBase
	rows, err := api.AllRecords(context.Background(), services.WithRecordListQuery("status", json.RawMessage(`"ACTIVE"`)), services.WithRecordListQuery("is_admin_state_up", json.RawMessage(`false`)), services.WithRecordListQuery("tenant_id", "tenant"))
	if err != nil || len(rows) != 2 || len(*calls) != 2 || string(rows[0].Resource.Body["id"]) != `"first"` || string(rows[1].Resource.Body["id"]) != `"last"` {
		t.Fatal(rows, err, *calls)
	}
	for _, call := range *calls {
		if call.method != "GET" || call.path != pythonVPNServiceBase {
			t.Fatal(*calls)
		}
	}
	if (*calls)[0].query == "" {
		t.Fatal(*calls)
	}
	// Declared aliases are server-side; the local Body field must stay out of query.
	api, calls, _ = pythonVPNServiceAPI(t, func(r *http.Request) *http.Response {
		if r.URL.Query().Get("admin_state_up") == "" || r.URL.Query().Get("status") != "" || r.URL.Query().Get("tenant_id") != "tenant" {
			t.Error(r.URL)
		}
		return pythonVPNServiceWire(200, `{"vpnservices":[]}`)
	})
	_, err = api.AllRecords(context.Background(), services.WithRecordListQuery("status", json.RawMessage(`"ACTIVE"`)), services.WithRecordListQuery("is_admin_state_up", json.RawMessage(`false`)), services.WithRecordListQuery("tenant_id", "tenant"))
	if err != nil || len(*calls) != 1 {
		t.Fatal(err, *calls)
	}
}
func TestVPNServiceRecordsProjectionDefaultsAndUnknownFields(t *testing.T) {
	api, _, _ := pythonVPNServiceAPI(t, func(*http.Request) *http.Response {
		return pythonVPNServiceWire(203, `{"vpnservices":[{"id":"r","tenant_id":"owner","x_extension":{"n":9007199254740993},"admin_state_up":"false"}]}`)
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	row := rows[0]
	if row.StatusCode != 203 || string(row.Resource.Body["project_id"]) != `"owner"` || string(row.Resource.Body["name"]) != "null" || string(row.Wire.Body["x_extension"]) != `{"n":9007199254740993}` {
		t.Fatal(row)
	}
	if string(row.Resource.Body["is_admin_state_up"]) != "true" {
		t.Fatal(row.Resource.Body)
	}
	row.Resource.Body["x_extension"][0] = 'X'
	if !json.Valid(row.Wire.Body["x_extension"]) || !json.Valid(row.Envelope) {
		t.Fatal("shared record evidence")
	}
}
func TestVPNServiceProxyJMESReturnsOpaqueGeneratorWithoutHTTP(t *testing.T) {
	api, calls, _ := pythonVPNServiceAPI(t, func(*http.Request) *http.Response { return pythonVPNServiceWire(200, `{"vpnservices":[]}`) })
	for _, tc := range []struct{ expression, json string }{{"[*].id", "null"}, {"type(@)", "null"}} {
		value, err := api.ProxyListRecords(context.Background(), services.WithRecordListJMESPath(tc.expression))
		if err != nil || string(value.JSON) != tc.json || len(*calls) != 0 {
			t.Fatal(value, err, *calls)
		}
	}
	value, err := api.ProxyListRecords(context.Background(), services.WithRecordListJMESPath("@"))
	if err != nil || len(*calls) != 0 {
		t.Fatal(value, err, *calls)
	}
	generator, ok := value.Value.(*services.RecordListGenerator)
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
func TestVPNServiceRecordsOptionsCancellationAndFailures(t *testing.T) {
	count := 0
	api, calls, _ := pythonVPNServiceAPI(t, func(*http.Request) *http.Response {
		count++
		return pythonVPNServiceWire(200, `{"vpnservices":[{"id":"r","status":"ACTIVE"}]}`)
	})
	raw := json.RawMessage(`"ACTIVE"`)
	option := services.WithRecordListQuery("status", raw)
	raw[0] = 'X'
	rows, err := api.AllRecords(context.Background(), option, services.WithRecordListQuery("ignored_extension", make(chan int)))
	if err != nil || len(rows) != 1 || count != 1 {
		t.Fatal(rows, err)
	}
	for _, options := range [][]services.RecordListOption{{nil}, {services.WithRecordListQuery("session", make(chan int))}, {services.WithRecordListMicroversion("2.1\r\nbad")}} {
		rows, err = api.AllRecords(context.Background(), options...)
		if err == nil || rows != nil || len(*calls) != 1 {
			t.Fatal(rows, err, *calls)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	rows, err = api.AllRecords(ctx, func(*services.RecordListOpts) error { cancel(); return nil })
	if !errors.Is(err, context.Canceled) || rows != nil || len(*calls) != 1 {
		t.Fatal(rows, err, *calls)
	}
	api, _, _ = pythonVPNServiceAPI(t, func(*http.Request) *http.Response { return pythonVPNServiceWire(500, `{"error":"failed"}`) })
	rows, err = api.AllRecords(context.Background())
	if err == nil || rows != nil {
		t.Fatal(rows, err)
	}
	var operation *resource.OperationError
	if !errors.As(err, &operation) {
		t.Fatal(err)
	}
}
