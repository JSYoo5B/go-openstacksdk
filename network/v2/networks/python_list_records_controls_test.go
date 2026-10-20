package networks_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestPythonNetworkListRecordsConflictingAttrsFormalOverridesAndOriginalJMES(t *testing.T) {
	var calls []nativeNetworkCall
	var versions, headers []string
	api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
		versions = append(versions, req.Header.Get("OpenStack-API-Version"))
		headers = append(headers, req.Header.Get("X-Trace"))
		if req.URL.Path != "/neutron/v2.0/networks" || req.URL.Query().Get("name") != "restored" || req.URL.Query().Get("jmespath_filters") != "" || req.URL.Query().Get("mtu") != "" {
			t.Fatal(req.URL)
		}
		return networkListResponse(200, `{"networks":[{"id":"one","mtu":2}]}`)
	})
	conflict := json.RawMessage(`{"name":"restored","mtu":2,"headers":{"X-Trace":"restored"},"microversion":"2.1","jmespath_filters":"[?","__conflicting_attrs":{"name":"never recurse"}}`)
	result, err := api.ProxyListRecords(context.Background(), networks.WithRecordListQuery("name", "original"), networks.WithRecordListHeaders(map[string]string{"X-Trace": "original"}), networks.WithRecordListMicroversion("2.0"), networks.WithRecordListConflictingAttrs(conflict), networks.WithRecordListJMESPath("@"))
	conflict[0] = 'X'
	if err != nil || result == nil || result.JSON != nil || len(calls) != 0 {
		t.Fatal(result, err, calls)
	}
	generator, ok := result.Value.(*networks.RecordListGenerator)
	if !ok {
		t.Fatal(result.Value)
	}
	var ids []string
	for value, err := range generator.Records {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, pythonNetworkListIDs(t, []*networks.NetworkRecord{value})[0])
	}
	if !reflect.DeepEqual(ids, []string{"one"}) || !reflect.DeepEqual(versions, []string{"network 2.1"}) || !reflect.DeepEqual(headers, []string{"restored"}) || api.RawClient().Microversion != "" || api.RawClient().MoreHeaders != nil {
		t.Fatal(ids, versions, headers, api.RawClient())
	}
	for _, raw := range []json.RawMessage{json.RawMessage(`null`), json.RawMessage(`false`), json.RawMessage(`0`), json.RawMessage(`1e-400`), json.RawMessage(`[]`), json.RawMessage(`{}`)} {
		var untouched []nativeNetworkCall
		empty := pythonCoreNetworkAPI(t, &untouched, func(*http.Request) *http.Response { return networkListResponse(200, `{"networks":[]}`) })
		rows, err := empty.AllRecords(context.Background(), networks.WithRecordListConflictingAttrs(raw))
		if err != nil || rows == nil || len(rows) != 0 || len(untouched) != 1 {
			t.Fatal(string(raw), rows, err, untouched)
		}
	}
	for _, raw := range []string{`{"paginated":false}`, `{"base_path":null}`, `{"cls":null}`, `{"session":null}`, `{"resource_type":null}`, `true`, `[1]`} {
		var untouched []nativeNetworkCall
		invalid := pythonCoreNetworkAPI(t, &untouched, func(*http.Request) *http.Response { t.Fatal("invalid restored control reached HTTP"); return nil })
		rows, err := invalid.AllRecords(context.Background(), networks.WithRecordListConflictingAttrs(json.RawMessage(raw)))
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || len(untouched) != 0 {
			t.Fatal(raw, rows, err, untouched)
		}
	}
}

func TestPythonNetworkListRecordsBasePathAndNamedHeadersVersionsUseOwnedFinalValues(t *testing.T) {
	for _, lane := range []string{"typed-final", "source-final", "nullable", "default-empty"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeNetworkCall
			var received http.Header
			target, version, trace := "/neutron/v2.0/alternate-networks", "network 2.2", "final"
			var options []networks.RecordListOption
			switch lane {
			case "typed-final":
				options = []networks.RecordListOption{networks.WithRecordListQuery("base_path", "/wrong"), networks.WithRecordListQuery("headers", map[string]string{"X-Trace": "wrong"}), networks.WithRecordListQuery("microversion", "2.1"), networks.WithRecordListBasePath("/alternate-networks"), networks.WithRecordListHeaders(map[string]string{"X-Trace": "final"}), networks.WithRecordListMicroversion("2.2")}
			case "source-final":
				options = []networks.RecordListOption{networks.WithRecordListBasePath("/wrong"), networks.WithRecordListHeaders(map[string]string{"X-Trace": "wrong"}), networks.WithRecordListMicroversion("2.1"), networks.WithRecordListQuery("base_path", "/alternate-networks"), networks.WithRecordListQuery("headers", map[string]string{"X-Trace": "final"}), networks.WithRecordListQuery("microversion", "2.2")}
			case "nullable":
				target, version, trace = "/neutron/v2.0/networks", "", ""
				options = []networks.RecordListOption{networks.WithRecordListBasePath("/wrong"), networks.WithRecordListHeaders(map[string]string{"X-Trace": "wrong"}), networks.WithRecordListMicroversion("2.1"), networks.WithRecordListQuery("base_path", nil), networks.WithRecordListQuery("headers", nil), networks.WithRecordListQuery("microversion", nil)}
			case "default-empty":
				target, version, trace = "/neutron/v2.0/networks", "", ""
				options = []networks.RecordListOption{networks.WithRecordListBasePath("")}
			}
			api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
				received = req.Header.Clone()
				if req.URL.Path != target {
					t.Fatal(req.URL, target)
				}
				return networkListResponse(200, `{"networks":[]}`)
			})
			rows, err := api.AllRecords(context.Background(), options...)
			if err != nil || rows == nil || len(calls) != 1 || received.Get("OpenStack-API-Version") != version || received.Get("X-Trace") != trace || api.RawClient().Microversion != "" || api.RawClient().MoreHeaders != nil {
				t.Fatal(rows, err, calls, received)
			}
		})
	}
	for _, path := range []string{"https://foreign.invalid/networks", "../networks", "/networks?foreign=true", "/networks#fragment", "//foreign.invalid/networks", "/a/../networks"} {
		var calls []nativeNetworkCall
		api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("bad path reached HTTP"); return nil })
		rows, err := api.AllRecords(context.Background(), networks.WithRecordListBasePath(path))
		if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(path, rows, err, calls)
		}
	}
	var calls []nativeNetworkCall
	api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path != "/neutron/v2.0/alternate-networks" {
			t.Fatal(req.URL)
		}
		if req.URL.Query().Get("marker") == "next" {
			return networkListResponse(200, `{"networks":[]}`)
		}
		return networkListResponse(200, `{"networks":[{"id":"one"}],"next":"/neutron/v2.0/alternate-networks?marker=next"}`)
	})
	rows, err := api.AllRecords(context.Background(), networks.WithRecordListBasePath("/alternate-networks"))
	if err != nil || len(rows) != 1 || len(calls) != 2 {
		t.Fatal(rows, err, calls)
	}
}

func TestPythonNetworkListRecordsUnknownConstructorNamesDiscardBeforeMarshal(t *testing.T) {
	var calls []nativeNetworkCall
	var marshals atomic.Int32
	api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.RawQuery != "" {
			t.Fatal(req.URL)
		}
		return networkListResponse(200, `{"networks":[]}`)
	})
	var options []networks.RecordListOption
	for _, key := range []string{"allow_unknown_params", "list_base_path", "connection", "_synchronized", "requires_id", "error_message", "skip_cache", "resource_response_key", "location"} {
		options = append(options, networks.WithRecordListQuery(key, networkListJSONCall{&marshals}))
	}
	rows, err := api.AllRecords(context.Background(), options...)
	if err != nil || rows == nil || len(calls) != 1 || marshals.Load() != 0 {
		t.Fatal(rows, err, calls, marshals.Load())
	}
}

func TestPythonNetworkListRecordsUsesLoadedCPythonFloatEqualityWithoutChangingRawEvidence(t *testing.T) {
	for _, tc := range []struct {
		actual, filter string
		matches        bool
	}{
		{`9007199254740992`, `9007199254740993.0`, true}, {`9007199254740993`, `9007199254740993.0`, false}, {`9007199254740993`, `9007199254740992`, false}, {`1e-400`, `0`, true}, {`[1e-400,true]`, `[false,1.0]`, true}, {`{"nested":{"n":9007199254740992,"extra":true}}`, `{"nested":{"n":9007199254740993.0}}`, true}, {`1e400`, `1e500`, true},
	} {
		t.Run(tc.actual+"/"+tc.filter, func(t *testing.T) {
			var calls []nativeNetworkCall
			body := `{"networks":[{"id":"one","created_at":` + tc.actual + `}]}`
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { return networkListResponse(200, body) })
			rows, err := api.AllRecords(context.Background(), networks.WithRecordListQuery("created_at", json.RawMessage(tc.filter)))
			if err != nil || rows == nil || (len(rows) == 1) != tc.matches || len(calls) != 1 {
				t.Fatal(rows, err, calls, tc)
			}
			if tc.matches && (string(rows[0].Resource.Body["created_at"]) != tc.actual || string(rows[0].Wire.Body["created_at"]) != tc.actual || string(rows[0].Envelope) != body) {
				t.Fatal(rows[0], tc)
			}
		})
	}
}
