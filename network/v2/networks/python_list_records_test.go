package networks_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonNetworkListOperation(t *testing.T, err error) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != "ListRecords" || wrapped.Resource != "networks" {
		t.Fatal("networks list context", err, wrapped)
	}
}

func pythonNetworkListIDs(t *testing.T, values []*networks.NetworkRecord) []string {
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

// Proxy.networks() sends one GET /networks with only Accept added and returns
// Network Resources whose Body keeps router/provider/MTU/DNS extension values.
func TestPythonNetworkListRecordsDefaultRequestAndExtensionFields(t *testing.T) {
	var calls []nativeNetworkCall
	body := `{"networks":[{"id":"n1","name":"public","router:external":true,"provider:network_type":"vlan","provider:physical_network":"physnet1","provider:segmentation_id":100,"mtu":1500,"dns_domain":"example.org.","port_security_enabled":true,"shared":false,"subnets":["s1"],"ipv4_address_scope":"scope","x_extension":{"keep":1}},{"id":"n2"}]}`
	api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Method != http.MethodGet || req.URL.Path != "/neutron/v2.0/networks" || req.URL.RawQuery != "" || req.Header.Get("Accept") != "application/json" {
			t.Fatal(req.Method, req.URL, req.Header)
		}
		response := nativeNetworkWire(200, body)
		response.Header.Set("X-Request-Id", "network-list")
		return response
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 2 || len(calls) != 1 || calls[0].body != "" {
		t.Fatal(rows, err, calls)
	}
	for key, want := range map[string]string{"is_router_external": "true", "provider_network_type": `"vlan"`, "provider_physical_network": `"physnet1"`, "provider_segmentation_id": "100", "mtu": "1500", "dns_domain": `"example.org."`, "is_port_security_enabled": "true", "is_shared": "false", "subnet_ids": `["s1"]`, "ipv4_address_scope_id": `"scope"`, "x_extension": `{"keep":1}`, "location": "null"} {
		pythonFindField(t, rows[0], key, want)
	}
	for key, want := range map[string]string{"router:external": "true", "provider:network_type": `"vlan"`, "mtu": "1500", "subnets": `["s1"]`} {
		pythonFindWireField(t, rows[0], key, want)
	}
	for key, want := range map[string]string{"is_router_external": "false", "is_port_security_enabled": "false", "tags": "[]", "mtu": "null"} {
		pythonFindField(t, rows[1], key, want)
	}
	if string(rows[0].Envelope) != body || rows[0].StatusCode != 200 || rows[0].Header.Get("X-Request-Id") != "network-list" {
		t.Fatal(rows[0])
	}
}

// Declared QueryParameters (client or wire names) go to Neutron; declared Body
// attributes outside QueryParameters filter each received row locally.
func TestPythonNetworkListRecordsQueryAliasesAndLocalBodyFilters(t *testing.T) {
	var calls []nativeNetworkCall
	var query url.Values
	api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
		query = req.URL.Query()
		return nativeNetworkWire(200, `{"networks":[{"id":"skip-mtu","mtu":1450,"subnets":["s1"]},{"id":"keep","mtu":1500,"subnets":["s1"]},{"id":"skip-subnets","mtu":1500,"subnets":["s2"]}]}`)
	})
	rows, err := api.AllRecords(context.Background(),
		networks.WithRecordListQuery("is_router_external", true), networks.WithRecordListQuery("provider_network_type", "vlan"), networks.WithRecordListQuery("is_shared", false),
		networks.WithRecordListQuery("project_id", "project"), networks.WithRecordListQuery("not_tags", []string{"a", "b"}), networks.WithRecordListQuery("unknown", make(chan int)),
		networks.WithRecordListQuery("mtu", 1500), networks.WithRecordListQuery("subnet_ids", []string{"s1"}))
	if err != nil || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	want := url.Values{"router:external": {"True"}, "provider:network_type": {"vlan"}, "shared": {"False"}, "project_id": {"project"}, "not-tags": {"a", "b"}}
	if !reflect.DeepEqual(query, want) {
		t.Fatal(query, want)
	}
	if ids := pythonNetworkListIDs(t, rows); !reflect.DeepEqual(ids, []string{"keep"}) {
		t.Fatal(ids)
	}
}

// search_networks builds {"name": name_or_id} and then updates it with the
// filters, so a filter name replaces the value but keeps the first position.
// list_networks pushes the filters unchanged. Neither adds local selection.
func TestPythonNetworkListRecordsCloudListAndSearchQueries(t *testing.T) {
	for _, test := range []struct {
		name    string
		options []networks.RecordListOption
		query   string
	}{
		{"list_networks without filters", nil, ""},
		{"list_networks filters", []networks.RecordListOption{networks.WithRecordListQuery("router:external", true), networks.WithRecordListQuery("status", "ACTIVE")}, "router%3Aexternal=True&status=ACTIVE"},
		{"search_networks name_or_id", []networks.RecordListOption{networks.WithRecordListQuery("name", "public")}, "name=public"},
		{"search_networks merged filters", []networks.RecordListOption{networks.WithRecordListQuery("name", "public"), networks.WithRecordListQuery("is_shared", true), networks.WithRecordListQuery("name", "override")}, "name=override&shared=True"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []nativeNetworkCall
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
				return nativeNetworkWire(200, `{"networks":[{"id":"n1","name":"other"}]}`)
			})
			rows, err := api.AllRecords(context.Background(), test.options...)
			got := ""
			if len(calls) == 1 {
				parsed, _ := url.ParseQuery(calls[0].query)
				got = parsed.Encode()
			}
			if err != nil || len(calls) != 1 || calls[0].path != "/neutron/v2.0/networks" || got != test.query || len(rows) != 1 {
				t.Fatal(rows, err, calls, got)
			}
		})
	}
}

// Resource.list follows networks_links, an RFC Link header and the
// limit/marker fallback; paginated=False reads exactly one page.
func TestPythonNetworkListRecordsPagingLinksAndLimitMarker(t *testing.T) {
	for _, lane := range []string{"networks_links", "link-header", "limit-marker", "single-page"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeNetworkCall
			api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Query().Get("marker") != "" {
					if lane == "limit-marker" && req.URL.Query().Get("marker") == "second" {
						return nativeNetworkWire(200, `{"networks":[]}`)
					}
					return nativeNetworkWire(200, `{"networks":[{"id":"second"}]}`)
				}
				next := fmt.Sprintf("http://%s/neutron/v2.0/networks?marker=first", req.Host)
				switch lane {
				case "networks_links", "single-page":
					return nativeNetworkWire(200, `{"networks":[{"id":"first"}],"networks_links":[{"rel":"next","href":"`+next+`"}]}`)
				case "link-header":
					response := nativeNetworkWire(200, `{"networks":[{"id":"first"}]}`)
					response.Header.Set("Link", "<"+next+`>; rel="next"`)
					return response
				}
				return nativeNetworkWire(200, `{"networks":[{"id":"first"}]}`)
			})
			var options []networks.RecordListOption
			switch lane {
			case "limit-marker":
				options = append(options, networks.WithRecordListLimit(1))
			case "single-page":
				options = append(options, networks.WithRecordListPaginated(false))
			}
			rows, err := api.AllRecords(context.Background(), options...)
			if err != nil {
				t.Fatal(err)
			}
			ids := pythonNetworkListIDs(t, rows)
			var queries []string
			for _, call := range calls {
				queries = append(queries, call.query)
			}
			switch lane {
			case "single-page":
				if !reflect.DeepEqual(ids, []string{"first"}) || !reflect.DeepEqual(queries, []string{""}) {
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

// The owned completion profile accepts final 200..399 and rejects 4xx/5xx.
func TestPythonNetworkListRecordsStatusesAndOperationContext(t *testing.T) {
	for _, code := range []int{200, 203, 299, 300, 399} {
		var calls []nativeNetworkCall
		api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeNetworkWire(code, `{"networks":[{"id":"n1"}]}`)
		})
		rows, err := api.AllRecords(context.Background())
		if err != nil || len(rows) != 1 || rows[0].StatusCode != code {
			t.Fatal(code, rows, err)
		}
	}
	for _, code := range []int{400, 401, 403, 404, 409, 500, 503} {
		var calls []nativeNetworkCall
		api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeNetworkWire(code, `{"NeutronError":{"message":"rejected"}}`)
		})
		rows, err := api.AllRecords(context.Background())
		if rows != nil || !gophercloud.ResponseCodeIs(err, code) {
			t.Fatal(code, rows, err)
		}
		pythonNetworkListOperation(t, err)
	}
	var calls []nativeNetworkCall
	api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { return nativeNetworkWire(200, `{"networks":[]}`) })
	if _, err := api.AllRecords(context.Background(), networks.WithRecordListQuery("jmespath_filters", "[0]")); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	} else {
		pythonNetworkListOperation(t, err)
	}
}
