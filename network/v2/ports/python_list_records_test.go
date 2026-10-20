package ports_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonPortListOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "ports" {
		t.Fatal("ports list context", err, wrapped)
	}
}

func pythonPortListIDs(t *testing.T, values []*ports.PortRecord) []string {
	t.Helper()
	ids := make([]string, 0, len(values))
	for _, value := range values {
		var id string
		if err := json.Unmarshal(value.Resource.Body["id"], &id); err != nil {
			t.Fatal(value.Resource.Body, err)
		}
		ids = append(ids, id)
	}
	return ids
}

// Proxy.ports() sends one GET /ports with only Accept added and returns Port
// Resources whose Body keeps binding/DNS/QoS/port-security extension values.
func TestPythonPortListRecordsDefaultRequestAndExtensionFields(t *testing.T) {
	var calls []nativePortCall
	body := `{"ports":[{"id":"p1","name":"web","binding:host_id":"compute-1","binding:profile":{"vnic":"direct"},"binding:vif_type":"ovs","binding:vnic_type":"normal","dns_name":"web","dns_assignment":[{"hostname":"web"}],"port_security_enabled":true,"qos_policy_id":"qos","qos_network_policy_id":"net-qos","tenant_id":"tenant","security_groups":["sg"],"admin_state_up":true,"x_extension":{"keep":1}},{"id":"p2"}]}`
	api := pythonCorePortAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Method != http.MethodGet || req.URL.Path != "/neutron/v2.0/ports" || req.URL.RawQuery != "" || req.Header.Get("Accept") != "application/json" {
			t.Fatal(req.Method, req.URL, req.Header)
		}
		response := nativePortWire(200, body)
		response.Header.Set("X-Request-Id", "port-list")
		return response
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 2 || len(calls) != 1 || calls[0].body != "" {
		t.Fatal(rows, err, calls)
	}
	for key, want := range map[string]string{"binding_host_id": `"compute-1"`, "binding_profile": `{"vnic":"direct"}`, "binding_vif_type": `"ovs"`, "binding_vnic_type": `"normal"`, "dns_name": `"web"`, "dns_assignment": `[{"hostname":"web"}]`, "is_port_security_enabled": "true", "qos_policy_id": `"qos"`, "qos_network_policy_id": `"net-qos"`, "project_id": `"tenant"`, "tenant_id": `"tenant"`, "security_group_ids": `["sg"]`, "is_admin_state_up": "true", "x_extension": `{"keep":1}`, "location": "null", "if_match": "null"} {
		pythonFindField(t, rows[0], key, want)
	}
	for key, want := range map[string]string{"binding:host_id": `"compute-1"`, "port_security_enabled": "true", "security_groups": `["sg"]`} {
		pythonFindWireField(t, rows[0], key, want)
	}
	for key, want := range map[string]string{"is_port_security_enabled": "false", "tags": "[]", "binding_host_id": "null", "project_id": "null"} {
		pythonFindField(t, rows[1], key, want)
	}
	if string(rows[0].Envelope) != body || rows[0].StatusCode != 200 || rows[0].Header.Get("X-Request-Id") != "port-list" {
		t.Fatal(rows[0])
	}
	if _, exists := rows[0].Resource.Body["binding:host_id"]; exists {
		t.Fatal(rows[0].Resource.Body)
	}
}

// Declared QueryParameters (client or wire names) go to Neutron; declared Body
// attributes outside QueryParameters filter each received row locally.
func TestPythonPortListRecordsQueryAliasesAndLocalBodyFilters(t *testing.T) {
	var calls []nativePortCall
	var query url.Values
	api := pythonCorePortAPI(t, &calls, func(req *http.Request) *http.Response {
		query = req.URL.Query()
		return nativePortWire(200, `{"ports":[{"id":"skip-host","binding:host_id":"other","dns_name":"web","binding:profile":{"vnic":"direct"}},{"id":"keep","binding:host_id":"compute-1","dns_name":"web","binding:profile":{"vnic":"direct","extra":true}},{"id":"skip-profile","binding:host_id":"compute-1","dns_name":"web","binding:profile":{"vnic":"normal"}}]}`)
	})
	rows, err := api.AllRecords(context.Background(),
		ports.WithRecordListQuery("network_id", "net"), ports.WithRecordListQuery("is_admin_state_up", false), ports.WithRecordListQuery("security_group_ids", []string{"sg1", "sg2"}),
		ports.WithRecordListQuery("tenant_id", "tenant"), ports.WithRecordListQuery("binding:vnic_type", "direct"), ports.WithRecordListQuery("any_tags", []string{"red", "blue"}),
		ports.WithRecordListQuery("unknown", make(chan int)), ports.WithRecordListQuery("binding_host_id", "compute-1"), ports.WithRecordListQuery("dns_name", "web"),
		ports.WithRecordListQuery("binding_profile", map[string]any{"vnic": "direct"}))
	if err != nil || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	want := url.Values{"network_id": {"net"}, "admin_state_up": {"False"}, "security_groups": {"sg1", "sg2"}, "project_id": {"tenant"}, "binding:vnic_type": {"direct"}, "tags-any": {"red", "blue"}}
	if !reflect.DeepEqual(query, want) {
		t.Fatal(query, want)
	}
	if ids := pythonPortListIDs(t, rows); !reflect.DeepEqual(ids, []string{"keep"}) {
		t.Fatal(ids)
	}
	if _, err := api.AllRecords(context.Background(), ports.WithRecordListQuery("jmespath_filters", "[0]")); err == nil {
		t.Fatal("jmespath_filters accepted")
	} else {
		pythonPortListOperation(t, err, "ListRecords")
	}
}

// Resource.list follows ports_links, an RFC Link header and the limit/marker
// fallback; paginated=False reads exactly one page and max_items caps rows.
func TestPythonPortListRecordsPagingLinksAndLimitMarker(t *testing.T) {
	for _, lane := range []string{"ports_links", "link-header", "limit-marker", "single-page", "max-items"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativePortCall
			api := pythonCorePortAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Query().Get("marker") != "" {
					if lane == "limit-marker" && req.URL.Query().Get("marker") == "second" {
						return nativePortWire(200, `{"ports":[]}`)
					}
					return nativePortWire(200, `{"ports":[{"id":"second"}]}`)
				}
				next := fmt.Sprintf("http://%s/neutron/v2.0/ports?marker=first", req.Host)
				switch lane {
				case "ports_links", "single-page", "max-items":
					return nativePortWire(200, `{"ports":[{"id":"first"}],"ports_links":[{"rel":"next","href":"`+next+`"}]}`)
				case "link-header":
					response := nativePortWire(200, `{"ports":[{"id":"first"}]}`)
					response.Header.Set("Link", "<"+next+`>; rel="next"`)
					return response
				}
				return nativePortWire(200, `{"ports":[{"id":"first"}]}`)
			})
			var options []ports.RecordListOption
			switch lane {
			case "limit-marker":
				options = append(options, ports.WithRecordListLimit(1))
			case "single-page":
				options = append(options, ports.WithRecordListPaginated(false))
			case "max-items":
				options = append(options, ports.WithRecordListMaxItems(1))
			}
			rows, err := api.AllRecords(context.Background(), options...)
			if err != nil {
				t.Fatal(err)
			}
			ids := pythonPortListIDs(t, rows)
			var queries []string
			for _, call := range calls {
				if call.method != http.MethodGet || call.path != "/neutron/v2.0/ports" {
					t.Fatal(calls)
				}
				queries = append(queries, call.query)
			}
			switch lane {
			case "single-page":
				if !reflect.DeepEqual(ids, []string{"first"}) || !reflect.DeepEqual(queries, []string{""}) {
					t.Fatal(ids, queries)
				}
			case "max-items":
				if !reflect.DeepEqual(ids, []string{"first"}) || len(queries) == 0 || queries[0] != "limit=1" {
					t.Fatal(ids, queries)
				}
			case "limit-marker":
				if !reflect.DeepEqual(ids, []string{"first", "second"}) || !reflect.DeepEqual(queries, []string{"limit=1", "limit=1&marker=first", "limit=1&marker=second"}) {
					t.Fatal(ids, queries)
				}
			default:
				if !reflect.DeepEqual(ids, []string{"first", "second"}) || !reflect.DeepEqual(queries, []string{"", "marker=first"}) {
					t.Fatal(ids, queries)
				}
			}
		})
	}
}

// raise_from_response accepts every 2xx page and rejects 4xx/5xx statuses.
func TestPythonPortListRecordsStatusesAndOperationContext(t *testing.T) {
	for _, code := range []int{200, 203} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(code, `{"ports":[{"id":"p1"}]}`) })
		rows, err := api.AllRecords(context.Background())
		if err != nil || len(rows) != 1 || rows[0].StatusCode != code {
			t.Fatal(code, rows, err)
		}
	}
	for _, code := range []int{400, 401, 403, 404, 409, 500, 503} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
			return nativePortWire(code, `{"NeutronError":{"message":"rejected"}}`)
		})
		rows, err := api.AllRecords(context.Background())
		if rows != nil || !gophercloud.ResponseCodeIs(err, code) {
			t.Fatal(code, rows, err)
		}
		pythonPortListOperation(t, err, "ListRecords")
	}
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(200, `{"ports":[{"id":"p1"}]}`) })
	if _, err := api.AllRecords(context.Background(), nil); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	} else {
		pythonPortListOperation(t, err, "ListRecords")
	}
}

func pythonPortSearchRows() string {
	return `{"ports":[{"id":"aaa","name":"web-1","device_id":"dev","status":"ACTIVE","binding:host_id":"compute-1","binding:profile":{"vnic":"direct","x":1}},{"id":"bbb","name":"web-2","device_id":"dev","status":"DOWN","binding:host_id":"compute-1","binding:profile":{"vnic":"normal"}},{"id":"web-9","name":"db","device_id":"dev","status":"ACTIVE","binding:host_id":"compute-2"}]}`
}

// search_ports pushes dictionary filters to list_ports and then runs
// _utils._filter_list; a JMESPath string lists every port without pushdown.
func TestPythonPortSearchRecordsSelectionAndPushdown(t *testing.T) {
	for _, test := range []struct {
		name, nameOrID, query string
		options               []ports.RecordSearchOption
		ids                   []string
		value                 string
	}{
		{"glob name without filters", "web-*", "", nil, []string{"aaa", "bbb", "web-9"}, ""},
		{"exact id", "bbb", "", nil, []string{"bbb"}, ""},
		{"dictionary filters pushdown and nested local match", "web-?", "device_id=dev", []ports.RecordSearchOption{ports.WithRecordSearchFilters(json.RawMessage(`{"device_id":"dev","binding_profile":{"vnic":"direct"}}`))}, []string{"aaa"}, ""},
		{"dictionary filters keep server and Body matches", "", "device_id=dev&status=ACTIVE", []ports.RecordSearchOption{ports.WithRecordSearchFilters(json.RawMessage(`{"device_id":"dev","status":"ACTIVE","binding_host_id":"compute-1"}`))}, []string{"aaa"}, ""},
		{"jmespath expression", "", "", []ports.RecordSearchOption{ports.WithRecordSearchExpression("[?status=='ACTIVE'].name")}, nil, `["web-1","db"]`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []nativePortCall
			api := pythonCorePortAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.Method != http.MethodGet || req.URL.Path != "/neutron/v2.0/ports" {
					t.Fatal(req.Method, req.URL)
				}
				return nativePortWire(200, pythonPortSearchRows())
			})
			result, err := api.SearchRecords(context.Background(), test.nameOrID, test.options...)
			if err != nil || len(calls) != 1 || calls[0].query != test.query {
				t.Fatal(result, err, calls)
			}
			if test.ids == nil {
				if result.Ports != nil || string(result.Value) != test.value {
					t.Fatal(result.Ports, string(result.Value))
				}
			} else if ids := pythonPortListIDs(t, result.Ports); !reflect.DeepEqual(ids, test.ids) {
				t.Fatal(ids, test.ids)
			}
			if len(result.Inventory) == 0 {
				t.Fatal(result)
			}
		})
	}
	// _dict_filter raises AttributeError for a key absent from a Python view.
	var unknown []nativePortCall
	unknownAPI := pythonCorePortAPI(t, &unknown, func(*http.Request) *http.Response { return nativePortWire(200, pythonPortSearchRows()) })
	if result, err := unknownAPI.SearchRecords(context.Background(), "", ports.WithRecordSearchFilters(json.RawMessage(`{"device_id":"dev","unknown_key":null}`))); err == nil || len(unknown) != 1 || unknown[0].query != "device_id=dev" || len(result.Inventory) != 3 || result.Ports != nil {
		t.Fatal(result, err, unknown)
	} else {
		pythonPortListOperation(t, err, "SearchRecords")
	}
	// A truthy filter value that is neither a mapping nor a string cannot be
	// expanded as Python **kwargs, so nothing is sent.
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(200, pythonPortSearchRows()) })
	if _, err := api.SearchRecords(context.Background(), "", ports.WithRecordSearchFilters(json.RawMessage(`[1]`))); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	} else {
		pythonPortListOperation(t, err, "SearchRecords")
	}
	api = pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(404, `{"NeutronError":{}}`) })
	if result, err := api.SearchRecords(context.Background(), "web"); err == nil || !gophercloud.ResponseCodeIs(err, 404) || result == nil || result.Ports != nil {
		t.Fatal(result, err)
	} else {
		pythonPortListOperation(t, err, "SearchRecords")
	}
}

// Resource.list sends headers and an explicit microversion on every page.
func TestPythonPortListRecordsHeadersAndMicroversionOnEveryPage(t *testing.T) {
	var calls []nativePortCall
	var seen []string
	api := pythonCorePortAPI(t, &calls, func(req *http.Request) *http.Response {
		seen = append(seen, req.Header.Get("X-Trace")+"|"+req.Header.Get("OpenStack-API-Version")+"|"+req.Header.Get("Accept"))
		if req.URL.Query().Get("marker") != "" {
			return nativePortWire(200, `{"ports":[]}`)
		}
		return nativePortWire(200, `{"ports":[{"id":"first"}],"ports_links":[{"rel":"next","href":"`+fmt.Sprintf("http://%s/neutron/v2.0/ports?marker=first", req.Host)+`"}]}`)
	})
	rows, err := api.AllRecords(context.Background(), ports.WithRecordListHeaders(map[string]string{"X-Trace": "owned"}), ports.WithRecordListMicroversion("2.1"))
	want := []string{"owned|network 2.1|application/json", "owned|network 2.1|application/json"}
	if err != nil || len(rows) != 1 || !reflect.DeepEqual(seen, want) {
		t.Fatal(rows, err, seen)
	}
}

func TestPythonPortListRecordsPartialFieldsCoercionAndEvidenceIsolation(t *testing.T) {
	var calls []nativePortCall
	payload := `{"ports":[{"id":"p","binding_profile":{"first":1},"binding:profile":{"last":2},"revision_number":"٩٠","admin_state_up":[],"port_security_enabled":null,"trusted":1e-400,"security_groups":"sg","dns_assignment":{},"resource_request":4,"tenant_id":"legacy","x_extension":{"large":9007199254740993},"self":"drop","location":"server-value"}]}`
	api := pythonCorePortAPI(t, &calls, func(req *http.Request) *http.Response {
		if !reflect.DeepEqual(req.URL.Query(), url.Values{"fields": {"id", "binding:profile", "security_groups"}}) {
			t.Fatal(req.URL)
		}
		r := nativePortWire(203, payload)
		r.Header.Set("X-Request-Id", "partial-fields")
		r.Header.Set("If-Match", "server-header")
		return r
	})
	rows, err := api.AllRecords(context.Background(), ports.WithRecordListQuery("fields", []string{"id", "binding:profile", "security_groups"}))
	if err != nil || len(rows) != 1 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	for key, want := range map[string]string{"binding_profile": `{"last":2}`, "revision_number": "90", "is_admin_state_up": "false", "is_port_security_enabled": "null", "trusted": "false", "security_group_ids": `["sg"]`, "dns_assignment": "[{}]", "resource_request": "{}", "project_id": `"legacy"`, "name": "null", "location": "null", "if_match": "null", "x_extension": `{"large":9007199254740993}`} {
		pythonFindField(t, rows[0], key, want)
	}
	if _, ok := rows[0].Resource.Body["fields"]; ok {
		t.Fatal("list query synthesized as response", rows[0])
	}
	if _, ok := rows[0].Resource.Body["self"]; ok {
		t.Fatal(rows[0])
	}
	pythonFindWireField(t, rows[0], "self", `"drop"`)
	pythonFindWireField(t, rows[0], "location", `"server-value"`)
	rows[0].Resource.Body["binding_profile"][2] = 'x'
	rows[0].Resource.Header.Set("X-Request-Id", "changed")
	rows[0].Header.Set("X-Other", "owned")
	rows[0].Envelope[2] = 'x'
	if string(rows[0].Wire.Body["binding:profile"]) != `{"last":2}` || rows[0].Wire.Header.Get("X-Request-Id") != "partial-fields" || rows[0].Wire.Header.Get("X-Other") != "" {
		t.Fatal(rows[0])
	}
}

type portListUnexpectedMarshaler struct{ calls *int }

func (v portListUnexpectedMarshaler) MarshalJSON() ([]byte, error) {
	*v.calls++
	return nil, errors.New("must not marshal")
}

func TestPythonPortListRecordsControlsDiscardAndValidationBeforeEncoding(t *testing.T) {
	for _, key := range []string{"cls", "self", "resource_type", "session", "", "bad\nname"} {
		t.Run(key, func(t *testing.T) {
			encoded := 0
			var calls []nativePortCall
			api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("unexpected HTTP"); return nil })
			rows, err := api.AllRecords(context.Background(), ports.WithRecordListQuery(key, portListUnexpectedMarshaler{&encoded}))
			if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || encoded != 0 || len(calls) != 0 {
				t.Fatal(rows, err, encoded, calls)
			}
			pythonPortListOperation(t, err, "ListRecords")
		})
	}
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(200, `{"ports":[]}`) })
	encoded := 0
	rows, err := api.AllRecords(context.Background(), ports.WithRecordListQuery("unknown_extension", portListUnexpectedMarshaler{&encoded}), ports.WithRecordListQuery("all_projects", portListUnexpectedMarshaler{&encoded}))
	if err != nil || rows == nil || len(rows) != 0 || encoded != 0 || len(calls) != 1 || calls[0].query != "" {
		t.Fatal(rows, err, encoded, calls)
	}
	for _, query := range []map[string]json.RawMessage{
		{"headers": json.RawMessage(`1`)}, {"base_path": json.RawMessage(`1`)}, {"jmespath_filters": json.RawMessage(`"[*].id"`)},
		{"fields": json.RawMessage(`[`)}, {"fields": json.RawMessage{'"', 0xff, '"'}}, {"max_items": json.RawMessage(`-1`)},
	} {
		n := len(calls)
		rows, err = api.AllRecords(context.Background(), ports.WithRecordListOpts(ports.RecordListOpts{Query: query}))
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != n {
			t.Fatal(rows, err, calls, query)
		}
	}
}

func TestPythonPortListRecordsOptionsOwnMapsAndNilHeaders(t *testing.T) {
	headers := map[string]string{"X-Trace": "original"}
	fields := []string{"id", "name"}
	option := ports.WithRecordListHeaders(headers)
	qoption := ports.WithRecordListQuery("fields", fields)
	headers["X-Trace"] = "mutated"
	fields[0] = "mutated"
	var escaped *ports.RecordListOpts
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Trace") != "original" || r.Header.Get("X-Callback") != "owned" || !reflect.DeepEqual(r.URL.Query()["fields"], []string{"id", "name"}) {
			t.Fatal(r.Header, r.URL)
		}
		escaped.Headers["X-Trace"] = "escaped"
		escaped.Query["fields"][2] = 'x'
		return nativePortWire(200, `{"ports":[{"id":"p"}]}`)
	})
	rows, err := api.AllRecords(context.Background(), ports.WithRecordListHeaders(nil), ports.WithRecordListOpts(ports.RecordListOpts{}), option, qoption, func(c *ports.RecordListOpts) error { c.Headers["X-Callback"] = "owned"; escaped = c; return nil })
	if err != nil || len(rows) != 1 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	if _, ok := rows[0].Resource.Body["fields"]; ok {
		t.Fatal(rows[0])
	}
}

func TestPythonPortListAndSearchRecordOptionsGuardCancellationAndSource(t *testing.T) {
	for _, search := range []bool{false, true} {
		for _, cancelOption := range []bool{false, true} {
			t.Run(fmt.Sprint(search, cancelOption), func(t *testing.T) {
				var calls []nativePortCall
				api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("unexpected HTTP"); return nil })
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				supplied, cause := errors.New("option failure"), errors.New("option cancellation")
				mutate := func() {
					if cancelOption {
						cancel(cause)
					} else {
						api.RawClient().ResourceBase += "changed/"
					}
				}
				var err error
				op := "ListRecords"
				if search {
					_, err = api.SearchRecords(ctx, "", func(c *ports.RecordSearchOpts) error { c.Headers["X-Owned"] = "yes"; mutate(); return supplied })
					op = "SearchRecords"
				} else {
					_, err = api.AllRecords(ctx, func(c *ports.RecordListOpts) error { c.Headers["X-Owned"] = "yes"; mutate(); return supplied })
				}
				if !errors.Is(err, supplied) || len(calls) != 0 {
					t.Fatal(err, calls)
				}
				if cancelOption && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) || !cancelOption && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				pythonPortListOperation(t, err, op)
			})
		}
	}
}

func TestPythonPortListRecordsCapCountsFilteredRowsAndLazyConsumer(t *testing.T) {
	for _, mode := range []string{"cap filters scanned rows", "stop after first row", "single page"} {
		t.Run(mode, func(t *testing.T) {
			var calls []nativePortCall
			api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
				if r.URL.Query().Get("marker") == "next" {
					return nativePortWire(200, `{"ports":[{"id":"third","dns_name":"keep"}]}`)
				}
				return nativePortWire(200, `{"ports":[{"id":"first","dns_name":"drop"},{"id":"second","dns_name":"keep"}],"ports_links":[{"rel":"next","href":"?marker=next"}]}`)
			})
			if mode == "stop after first row" {
				n := 0
				for row, err := range api.ListRecords(context.Background()) {
					if err != nil || row == nil {
						t.Fatal(row, err)
					}
					n++
					break
				}
				if n != 1 || len(calls) != 1 {
					t.Fatal(n, calls)
				}
				return
			}
			options := []ports.RecordListOption{ports.WithRecordListQuery("dns_name", "keep")}
			if mode == "single page" {
				options = append(options, ports.WithRecordListPaginated(false))
			} else {
				options = append(options, ports.WithRecordListMaxItems(1))
			}
			rows, err := api.AllRecords(context.Background(), options...)
			if err != nil || len(calls) != 1 {
				t.Fatal(rows, err, calls)
			}
			if mode == "single page" && !reflect.DeepEqual(pythonPortListIDs(t, rows), []string{"second"}) || mode != "single page" && len(rows) != 0 {
				t.Fatal(rows)
			}
		})
	}
}

func TestPythonPortListRecordsContinuationTruthinessAndTerminalEvidence(t *testing.T) {
	for _, extra := range []string{`,"next":1e-400`, `,"next":false`, `,"next":null`, `,"next":[]`} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(203, `{"ports":[{"id":"p"}]`+extra+`}`) })
		rows, err := api.AllRecords(context.Background())
		if err != nil || len(rows) != 1 || len(calls) != 1 {
			t.Fatal(extra, rows, err, calls)
		}
	}
	for _, payload := range []string{`{"ports":[null]}`, `{"ports":[4]}`, `{"ports":null}`, `{}`, `{"ports":[{"id":"p","cls":null}]}`, `{"ports":[{"id":"p","microversion":null}]}`, `{"ports":[{"id":"p"}],"next":true}`} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativePortWire(203, payload)
			r.Header.Set("X-Request-Id", "malformed-page")
			return r
		})
		rows, err := api.AllRecords(context.Background())
		var proof *resource.ResponseError
		if rows != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != payload || proof.Header.Get("X-Request-Id") != "malformed-page" || len(calls) != 1 {
			t.Fatal(rows, err, proof, calls)
		}
	}
}

func TestPythonPortListRecordsPhysicalFailuresPreserveNative401AndStopCallbacks(t *testing.T) {
	for _, status := range []int{200, 203, 401, 404, 429, 503} {
		for _, fault := range []string{"read", "close", "mixed"} {
			t.Run(fmt.Sprint(status, fault), func(t *testing.T) {
				readCause, closeCause, hookCause := errors.New("physical read"), errors.New("physical close"), errors.New("callback must not run")
				raw := `{"ports":[{"id":"p"}]}`
				if status >= 400 {
					raw = `{"NeutronError":{"message":"physical rejection"}}`
				}
				closed := 0
				body := &pythonFindFaultBody{data: raw, onClose: func() { closed++ }}
				if fault != "close" {
					body.read = readCause
				}
				if fault != "read" {
					body.close = closeCause
				}
				var calls []nativePortCall
				api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
					r := nativePortWire(status, raw)
					r.Body = body
					r.Header.Set("X-Request-Id", "actual-port-list")
					return r
				})
				retries, reauth := 0, 0
				api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return hookCause
				}
				api.RawClient().ReauthFunc = func(context.Context) error { reauth++; return nil }
				rows, err := api.AllRecords(context.Background())
				if rows != nil || err == nil || len(calls) != 1 || closed != 1 || retries != 0 || reauth != 0 || errors.Is(err, hookCause) {
					t.Fatal(rows, err, calls, closed, retries, reauth)
				}
				assertCauses := func(e error) {
					t.Helper()
					if fault != "close" && !errors.Is(e, readCause) || fault != "read" && !errors.Is(e, closeCause) {
						t.Fatal(e)
					}
				}
				assertCauses(err)
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != raw || proof.Header.Get("X-Request-Id") != "actual-port-list" {
						t.Fatal(err, proof)
					}
					return
				}
				original := err
				if status == 401 {
					var native *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &native) || native.ErrOriginal == nil || native.ErrReauth == nil {
						t.Fatal(err, native)
					}
					assertCauses(native.ErrReauth)
					original = native.ErrOriginal
				}
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(original, &proof) || proof.Actual != status || proof.Method != http.MethodGet || proof.URL != api.RawClient().ServiceURL("ports") || string(proof.Body) != raw || proof.ResponseHeader.Get("X-Request-Id") != "actual-port-list" {
					t.Fatal(err, proof)
				}
			})
		}
	}
}

func TestPythonPortListRecordsCleanReauthenticationAndRetryKeepPolicy(t *testing.T) {
	for _, status := range []int{401, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls []nativePortCall
			var tokens []string
			api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
				tokens = append(tokens, r.Header.Get("X-Auth-Token"))
				if r.Header.Get("If-Match") != "configured" || r.Header.Get("Accept") != "application/json" || r.Header.Get("OpenStack-API-Version") != "network 2.1" || r.URL.RawQuery != "network_id=n" {
					t.Fatal(r.Header, r.URL)
				}
				if len(calls) == 1 {
					return nativePortWire(status, `{"NeutronError":{"message":"retry"}}`)
				}
				return nativePortWire(200, `{"ports":[{"id":"p"}]}`)
			})
			api.RawClient().Microversion = "9.9"
			api.RawClient().MoreHeaders = map[string]string{"Accept": "configured", "If-Match": "configured"}
			retries, reauth := 0, 0
			api.RawClient().ReauthFunc = func(context.Context) error { reauth++; api.RawClient().SetToken("renewed"); return nil }
			api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, c *gophercloud.RequestOpts, _ error, _ uint) error {
				retries++
				c.OkCodes = append(c.OkCodes, 503)
				c.MoreHeaders["X-Retry-Probe"] = "ordinary"
				return nil
			}
			rows, err := api.AllRecords(context.Background(), ports.WithRecordListMicroversion("2.1"), ports.WithRecordListQuery("network_id", "n"))
			if err != nil || len(rows) != 1 || len(calls) != 2 || status == 401 && (reauth != 1 || retries != 0 || !reflect.DeepEqual(tokens, []string{"test-token", "renewed"})) || status == 503 && (retries != 1 || reauth != 0) {
				t.Fatal(rows, err, calls, tokens, retries, reauth)
			}
			if api.RawClient().Microversion != "9.9" || api.RawClient().MoreHeaders["Accept"] != "configured" {
				t.Fatal(api.RawClient())
			}
		})
	}
}

func TestPythonPortListRecordValuesArbitraryJSONAndInventory(t *testing.T) {
	for _, test := range []struct{ expression, want string }{
		{"[?status=='ACTIVE'].id", `["aaa","web-9"]`}, {"length(@)", "3"}, {"[0].binding_profile", `{"vnic":"direct","x":1}`}, {"`true`", "true"}, {"type(@)", `"array"`}, {"[?id=='missing']", "[]"},
	} {
		t.Run(test.expression, func(t *testing.T) {
			var calls []nativePortCall
			api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(200, pythonPortSearchRows()) })
			result, err := api.ListRecordValues(context.Background(), test.expression)
			if err != nil || result == nil || string(result.Value) != test.want || len(result.Inventory) != 3 || len(calls) != 1 {
				t.Fatal(result, err, calls)
			}
		})
	}
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(200, pythonPortSearchRows()) })
	result, err := api.ListRecordValues(context.Background(), "[")
	if result == nil || len(result.Inventory) != 3 || result.Value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
		t.Fatal(result, err, calls)
	}
	pythonPortListOperation(t, err, "ListRecordValues")
}

func TestPythonPortSearchRecordsFalseyFiltersHeaderSnapshotAndUnknownKeys(t *testing.T) {
	for _, filter := range []string{"null", "false", "0", "1e-400", "[]", "{}", `""`} {
		var calls []nativePortCall
		headers := map[string]string{"X-Trace": "original"}
		option := ports.WithRecordSearchHeaders(headers)
		headers["X-Trace"] = "mutated"
		api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
			if r.URL.RawQuery != "" || r.Header.Get("X-Trace") != "original" {
				t.Fatal(r.URL, r.Header)
			}
			return nativePortWire(200, pythonPortSearchRows())
		})
		result, err := api.SearchRecords(context.Background(), "", ports.WithRecordSearchFilters(json.RawMessage(filter)), ports.WithRecordSearchHeaders(nil), option, func(c *ports.RecordSearchOpts) error { c.Headers["X-Own"] = "yes"; return nil })
		if err != nil || result == nil || len(result.Ports) != 3 || len(calls) != 1 {
			t.Fatal(filter, result, err, calls)
		}
	}
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(200, pythonPortSearchRows()) })
	result, err := api.SearchRecords(context.Background(), "", ports.WithRecordSearchFilters(json.RawMessage(`{"unexpected":null}`)))
	if result == nil || len(result.Inventory) != 3 || result.Value != nil || err == nil || len(calls) != 1 || calls[0].query != "" || !strings.Contains(err.Error(), "unexpected") {
		t.Fatal(result, err, calls)
	}
	pythonPortListOperation(t, err, "SearchRecords")
}

func TestPythonPortProxyListRecordsSourceGeneratorExpressionsDoNoHTTP(t *testing.T) {
	for _, test := range []struct{ expression, want string }{
		{"[*].id", "null"}, {"[?status=='ACTIVE']", "null"}, {"[0]", "null"}, {"type(@)", "null"}, {"`true`", "true"}, {"@ == @", "true"}, {"[ @ ] == [ @ ]", "true"},
	} {
		t.Run(test.expression, func(t *testing.T) {
			var calls []nativePortCall
			api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("expression consumed generator"); return nil })
			result, err := api.ProxyListRecords(context.Background(), ports.WithRecordListQuery("jmespath_filters", test.expression))
			if err != nil || result == nil || string(result.JSON) != test.want || len(calls) != 0 {
				t.Fatal(result, err, calls)
			}
		})
	}
	for _, expression := range []string{"length(@)", "contains(@,'x')", "sort_by(@,&id)", "to_string(@)", "["} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("bad expression consumed generator"); return nil })
		result, err := api.ProxyListRecords(context.Background(), ports.WithRecordListQuery("jmespath_filters", expression))
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(expression, result, err, calls)
		}
		pythonPortListOperation(t, err, "ProxyListRecords")
	}
}

func TestPythonPortProxyListRecordsPreserveReturnedAndNestedGenerators(t *testing.T) {
	for _, filter := range []any{nil, false, 0, 1e-300, []string{}, map[string]any{}, true, 1, []string{"x"}, map[string]any{"x": true}, "", "@", "not_null(@,`[]`)", "to_array(@)"} {
		t.Run(fmt.Sprint(filter), func(t *testing.T) {
			var calls []nativePortCall
			api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
				if r.URL.RawQuery != "network_id=n" {
					t.Fatal(r.URL)
				}
				return nativePortWire(200, `{"ports":[{"id":"p"}]}`)
			})
			result, err := api.ProxyListRecords(context.Background(), ports.WithRecordListQuery("jmespath_filters", filter), ports.WithRecordListQuery("network_id", "n"))
			if err != nil || result == nil || result.JSON != nil || len(calls) != 0 {
				t.Fatal(result, err, calls)
			}
			value := result.Value
			if array, ok := value.([]any); ok {
				if len(array) != 1 {
					t.Fatal(value)
				}
				value = array[0]
			}
			generator, ok := value.(*ports.RecordListGenerator)
			if !ok || generator == nil {
				t.Fatal(value)
			}
			n := 0
			for row, err := range generator.Records {
				if err != nil || row == nil {
					t.Fatal(row, err)
				}
				n++
			}
			if n != 1 || len(calls) != 1 {
				t.Fatal(n, calls)
			}
		})
	}
}

func TestPythonPortListRecordsNormalizeSourceFormalsAndConflictingAttrs(t *testing.T) {
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Trace") != "restored" || r.Header.Get("OpenStack-API-Version") != "network 2.4" || r.URL.Query().Get("network_id") != "new" || r.URL.Query().Has("__conflicting_attrs") || r.URL.Query().Has("jmespath_filters") || r.URL.Query().Has("location") {
			t.Fatal(r.Header, r.URL)
		}
		return nativePortWire(200, `{"ports":[{"id":"p","binding:profile":{"role":"keep"}}]}`)
	})
	raw := json.RawMessage(`{"network_id":"new","binding_profile":{"role":"keep"},"headers":{"X-Trace":"restored"},"microversion":"2.4","jmespath_filters":"length(@)","__conflicting_attrs":{"network_id":"must-not-recurse"},"location":null}`)
	rows, err := api.AllRecords(context.Background(), ports.WithRecordListQuery("network_id", "old"), ports.WithRecordListQuery("__conflicting_attrs", raw))
	if err != nil || len(rows) != 1 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	n := len(calls)
	result, err := api.ProxyListRecords(context.Background(), ports.WithRecordListJMESPath("[*].id"), ports.WithRecordListQuery("__conflicting_attrs", json.RawMessage(`{"jmespath_filters":"length(@)"}`)))
	if err != nil || result == nil || string(result.JSON) != "null" || len(calls) != n {
		t.Fatal(result, err, calls)
	}
	for _, raw := range []json.RawMessage{json.RawMessage(`{"paginated":false}`), json.RawMessage(`{"base_path":"/ports"}`), json.RawMessage(`{"cls":null}`), json.RawMessage(`true`), json.RawMessage(`[1]`)} {
		rows, err = api.AllRecords(context.Background(), ports.WithRecordListQuery("__conflicting_attrs", raw))
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != n {
			t.Fatal(raw, rows, err, calls)
		}
	}
}

func TestPythonPortListRecordsSourceUnknownControlNamesDiscardBeforeMarshaler(t *testing.T) {
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(200, `{"ports":[]}`) })
	encoded := 0
	var opts []ports.RecordListOption
	for _, key := range []string{"list_base_path", "connection", "_synchronized", "requires_id", "error_message", "skip_cache", "resource_response_key", "location", "if_match", "allow_unknown_params"} {
		opts = append(opts, ports.WithRecordListQuery(key, portListUnexpectedMarshaler{&encoded}))
	}
	rows, err := api.AllRecords(context.Background(), opts...)
	if err != nil || rows == nil || len(rows) != 0 || encoded != 0 || len(calls) != 1 || calls[0].query != "" {
		t.Fatal(rows, err, encoded, calls)
	}
}

func TestPythonPortListRecordsBasePathOwnsRouteAndLocksContinuation(t *testing.T) {
	for _, mode := range []string{"typed", "raw formal", "snapshot"} {
		t.Run(mode, func(t *testing.T) {
			var calls []nativePortCall
			api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
				if r.URL.Path != "/neutron/v2.0/custom/ports" || r.URL.Query().Get("network_id") != "n" {
					t.Fatal(r.URL)
				}
				if r.URL.Query().Get("marker") != "" {
					return nativePortWire(200, `{"ports":[]}`)
				}
				return nativePortWire(200, `{"ports":[{"id":"p"}],"ports_links":[{"rel":"next","href":"?marker=next"}]}`)
			})
			options := []ports.RecordListOption{ports.WithRecordListQuery("network_id", "n")}
			switch mode {
			case "typed":
				options = append(options, ports.WithRecordListBasePath("/custom/ports"))
			case "raw formal":
				options = append(options, ports.WithRecordListQuery("base_path", "/custom/ports"))
			case "snapshot":
				path := "/custom/ports"
				option := ports.WithRecordListOpts(ports.RecordListOpts{BasePath: &path})
				path = "changed"
				options = []ports.RecordListOption{option, ports.WithRecordListQuery("network_id", "n")}
			}
			rows, err := api.AllRecords(context.Background(), options...)
			if err != nil || len(rows) != 1 || len(calls) != 2 {
				t.Fatal(rows, err, calls)
			}
		})
	}
	for _, path := range []string{"https://foreign.invalid/ports", "//foreign.invalid/ports", "../ports", "/ports/../other", "/ports?query=x", "/ports#fragment"} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("unsafe route HTTP"); return nil })
		rows, err := api.AllRecords(context.Background(), ports.WithRecordListBasePath(path))
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(path, rows, err, calls)
		}
	}
}

func TestPythonPortListRecordsLocalEqualityUsesSourceLoadedFloats(t *testing.T) {
	for _, test := range []struct {
		row, filter string
		count       int
	}{
		{`{"id":"p","binding:profile":{"number":9007199254740992}}`, `{"number":9007199254740993.0}`, 1},
		{`{"id":"p","binding:profile":{"number":9007199254740993}}`, `{"number":9007199254740993.0}`, 0},
		{`{"id":"p","binding:profile":{"nested":[true,1e-400,9007199254740993.0]}}`, `{"nested":[1,0,9007199254740992]}`, 1},
		{`{"id":"p","binding:profile":{"number":1e400}}`, `{"number":1e500}`, 1},
	} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(200, `{"ports":[`+test.row+`]}`) })
		rows, err := api.AllRecords(context.Background(), ports.WithRecordListQuery("binding_profile", json.RawMessage(test.filter)))
		if err != nil || len(rows) != test.count || len(calls) != 1 || calls[0].query != "" {
			t.Fatal(test, rows, err, calls)
		}
	}
}

func TestPythonPortSearchRecordsHonorsProxyJMESBeforeCloudLocalSelection(t *testing.T) {
	for _, expression := range []string{"[*].id", "length(@)", "`true`", "`42`"} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("Source non-iterable result performed HTTP"); return nil })
		filter, _ := json.Marshal(map[string]any{"jmespath_filters": expression})
		result, err := api.SearchRecords(context.Background(), "", ports.WithRecordSearchFilters(filter))
		if result == nil || err == nil || len(result.Inventory) != 0 || len(calls) != 0 {
			t.Fatal(expression, result, err, calls)
		}
		pythonPortListOperation(t, err, "SearchRecords")
	}
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(200, `{"ports":[{"id":"p"}]}`) })
	result, err := api.SearchRecords(context.Background(), "", ports.WithRecordSearchFilters(json.RawMessage(`{"jmespath_filters":"@"}`)))
	if result == nil || err == nil || len(result.Inventory) != 1 || len(calls) != 1 {
		t.Fatal(result, err, calls)
	}
	for _, literal := range []string{"`[]`", "`{}`", "' '", "`[null,false,0,{},[]]`"} {
		filter, _ := json.Marshal(map[string]any{"jmespath_filters": literal})
		n := len(calls)
		result, err = api.SearchRecords(context.Background(), "", ports.WithRecordSearchFilters(filter))
		if literal == "' '" {
			if err == nil {
				t.Fatal(literal, result)
			}
		} else if err != nil || result == nil || string(result.Value) != "[]" || len(result.Inventory) != 0 {
			t.Fatal(literal, result, err)
		}
		if len(calls) != n {
			t.Fatal(literal, calls)
		}
	}
}

func TestPythonPortListRecordsFormalFactoriesAndTypedOptionsUseLastValue(t *testing.T) {
	for _, rawLast := range []bool{false, true} {
		var calls []nativePortCall
		var path, version, trace string
		api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
			path, version, trace = r.URL.Path, r.Header.Get("OpenStack-API-Version"), r.Header.Get("X-Trace")
			return nativePortWire(200, `{"ports":[]}`)
		})
		raw := []ports.RecordListOption{ports.WithRecordListQuery("headers", map[string]string{"X-Trace": "raw"}), ports.WithRecordListQuery("microversion", "2.1"), ports.WithRecordListQuery("base_path", "/raw-ports")}
		typed := []ports.RecordListOption{ports.WithRecordListHeaders(map[string]string{"X-Trace": "typed"}), ports.WithRecordListMicroversion("2.2"), ports.WithRecordListBasePath("/typed-ports")}
		options := append(raw, typed...)
		wantPath, wantVersion, wantTrace := "/neutron/v2.0/typed-ports", "network 2.2", "typed"
		if rawLast {
			options = append(typed, raw...)
			wantPath, wantVersion, wantTrace = "/neutron/v2.0/raw-ports", "network 2.1", "raw"
		}
		rows, err := api.AllRecords(context.Background(), options...)
		if err != nil || rows == nil || len(calls) != 1 || path != wantPath || version != wantVersion || trace != wantTrace {
			t.Fatal(rows, err, calls, path, version, trace)
		}
	}
}

func TestPythonPortSearchRecordsSourceWireFiltersAndConnectionLocationBeforeSelection(t *testing.T) {
	location := json.RawMessage(`{"cloud":"fixture","project":{"id":"project"}}`)
	option := ports.WithRecordSearchLocation(location)
	location[2] = 'x'
	for _, filter := range []string{`{"binding_profile":{"vnic":"direct"}}`, `{"security_group_ids":["sg"]}`, `{"is_admin_state_up":true}`, `{"location":{"project":{"id":"project"}}}`} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
			return nativePortWire(200, `{"ports":[{"id":"p","binding:profile":{"vnic":"direct"},"security_groups":["sg"],"admin_state_up":true}]}`)
		})
		result, err := api.SearchRecords(context.Background(), "", ports.WithRecordSearchFilters(json.RawMessage(filter)), option)
		if err != nil || result == nil || len(result.Ports) != 1 || len(result.Inventory) != 1 || len(calls) != 1 {
			t.Fatal(filter, result, err, calls)
		}
		pythonFindField(t, result.Ports[0], "location", `{"cloud":"fixture","project":{"id":"project"}}`)
		if _, present := result.Ports[0].Wire.Body["location"]; present {
			t.Fatal("Connection view fabricated in Wire", result.Ports[0])
		}
	}
	// Resource is a dict subclass filled with canonical to_dict keys. Its wire
	// __getitem__ fallback does not change builtin dict.get or membership used
	// by cloud._utils._filter_list, so these second-phase filters are unknown.
	for _, filter := range []string{`{"binding:profile":{"vnic":"direct"}}`, `{"security_groups":["sg"]}`, `{"admin_state_up":true}`} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
			return nativePortWire(200, `{"ports":[{"id":"p","binding:profile":{"vnic":"direct"},"security_groups":["sg"],"admin_state_up":true}]}`)
		})
		result, err := api.SearchRecords(context.Background(), "", ports.WithRecordSearchFilters(json.RawMessage(filter)))
		if result == nil || err == nil || len(result.Inventory) != 1 || result.Ports != nil || len(calls) != 1 {
			t.Fatal(filter, result, err, calls)
		}
	}
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("invalid location performed HTTP"); return nil })
	result, err := api.SearchRecords(context.Background(), "", ports.WithRecordSearchLocation(json.RawMessage(`false`)))
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(result, err, calls)
	}
}

func TestPythonPortSearchRecordsSecondFilterKeepsSourceLoadedNumericMatch(t *testing.T) {
	for _, test := range []struct {
		row, filter string
		count       int
	}{
		{`{"id":"p","binding:profile":{"number":9007199254740992}}`, `{"binding_profile":{"number":9007199254740993.0}}`, 1},
		{`{"id":"p","binding:profile":{"number":9007199254740993}}`, `{"binding_profile":{"number":9007199254740993.0}}`, 0},
		{`{"id":"p","binding:profile":{"nested":[true,1e-400,9007199254740993.0]}}`, `{"binding_profile":{"nested":[1,0,9007199254740992]}}`, 1},
	} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(200, `{"ports":[`+test.row+`]}`) })
		result, err := api.SearchRecords(context.Background(), "", ports.WithRecordSearchFilters(json.RawMessage(test.filter)))
		if err != nil || result == nil || len(result.Ports) != test.count || len(calls) != 1 {
			t.Fatal(test, result, err, calls)
		}
	}
}

func TestPythonPortSearchRecordsCloudLocationOwnsForeignProjectAndZone(t *testing.T) {
	name, cloud, domain := "current-project", "fixture", "domain"
	location := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"current"`), Name: &name, DomainID: &domain}}
	option := ports.WithRecordSearchCloudLocation(location)
	name = "mutated"
	location.Project.ID[2] = 'x'
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
		return nativePortWire(200, `{"ports":[{"id":"same","project_id":"current","availability_zone":"zone-a"},{"id":"foreign","project_id":"other","availability_zone":"zone-b"}]}`)
	})
	result, err := api.SearchRecords(context.Background(), "", option, ports.WithRecordSearchFilters(json.RawMessage(`{"location":{"project":{"id":"other"}}}`)))
	if err != nil || result == nil || len(result.Ports) != 1 || len(result.Inventory) != 2 || len(calls) != 1 || calls[0].query != "" {
		t.Fatal(result, err, calls)
	}
	var current, foreign resource.CloudLocation
	if err := json.Unmarshal(result.Inventory[0].Resource.Body["location"], &current); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(result.Ports[0].Resource.Body["location"], &foreign); err != nil {
		t.Fatal(err)
	}
	if current.Project.Name == nil || *current.Project.Name != "current-project" || string(current.Zone) != `"zone-a"` || string(foreign.Project.ID) != `"other"` || foreign.Project.Name != nil || foreign.Project.DomainID != nil || string(foreign.Zone) != `"zone-b"` {
		t.Fatal(current, foreign)
	}
	for _, row := range result.Inventory {
		if _, ok := row.Wire.Body["location"]; ok {
			t.Fatal("computed location fabricated in Wire", row)
		}
	}
}
