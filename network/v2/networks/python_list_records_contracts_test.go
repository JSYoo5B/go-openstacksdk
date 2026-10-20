package networks_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func networkListResponse(code int, body string) *http.Response {
	response := nativeNetworkWire(code, body)
	response.Header.Set("X-Request-Id", "network-list-actual")
	return response
}
func networkListProof(t *testing.T, err error, code int, body string) {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "network-list-actual" || proof.Cause == nil || !errors.Is(err, proof.Cause) {
		t.Fatal(err, proof)
	}
	pythonNetworkListOperation(t, err)
}

func TestPythonNetworkListRecordsEntireBodyDefaultsAndOrderedAliases(t *testing.T) {
	var calls []nativeNetworkCall
	api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { return networkListResponse(203, `{"networks":[{}]}`) })
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 1 || len(calls) != 1 || len(rows[0].Resource.Body) != 31 {
		t.Fatal(rows, err, calls)
	}
	defaults := map[string]string{
		"id": "null", "revision_number": "null", "tags": "[]", "availability_zone_hints": "null", "availability_zones": "null", "created_at": "null", "description": "null", "dns_domain": "null", "ipv4_address_scope_id": "null", "ipv6_address_scope_id": "null", "is_admin_state_up": "null", "is_default": "null", "is_port_security_enabled": "false", "is_router_external": "false", "is_shared": "null", "mtu": "null", "name": "null", "project_id": "null", "provider_network_type": "null", "provider_physical_network": "null", "provider_segmentation_id": "null", "pvlan": "null", "qos_policy_id": "null", "segments": "null", "status": "null", "subnet_ids": "null", "updated_at": "null", "is_vlan_transparent": "null", "is_vlan_qinq": "null", "if_match": "null", "location": "null",
	}
	for key, want := range defaults {
		pythonFindField(t, rows[0], key, want)
	}
	for _, tc := range []struct{ body, subnets, external string }{
		{`{"subnet_ids":["canonical"],"subnets":"wire","is_router_external":false,"router:external":"false"}`, `["wire"]`, `true`},
		{`{"subnets":"wire","subnet_ids":["canonical"],"router:external":"false","is_router_external":false}`, `["canonical"]`, `false`},
	} {
		body := `{"networks":[` + tc.body + `]}`
		api = pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { return networkListResponse(200, body) })
		rows, err = api.AllRecords(context.Background())
		if err != nil || len(rows) != 1 {
			t.Fatal(rows, err)
		}
		pythonFindField(t, rows[0], "subnet_ids", tc.subnets)
		pythonFindField(t, rows[0], "is_router_external", tc.external)
	}
}

func TestPythonNetworkListRecordsDeclaredCoercionUnknownBodyAndPrivateEvidence(t *testing.T) {
	var calls []nativeNetworkCall
	body := `{"networks":[{"id":null,"name":false,"revision_number":"١٢","tags":"tag","mtu":1450.9,"admin_state_up":"false","is_default":1e-400,"port_security_enabled":null,"router:external":[],"shared":{},"pvlan":[false],"vlan_transparent":0,"vlan_qinq":1,"availability_zone_hints":false,"availability_zones":"zone","segments":{"segmentation_id":9007199254740993},"subnets":null,"provider:segmentation_id":9007199254740993,"created_at":9007199254740993,"updated_at":false,"tenant_id":"extension","IF-MATCH":"row","IF_MATCH":"unknown","self":"wire-only","location":{"spoof":true},"x_extension":{"n":9007199254740993}}]}`
	api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
		response := networkListResponse(203, body)
		response.Header.Set("If-Match", "page")
		return response
	})
	rows, err := api.AllRecords(context.Background())
	if err != nil || len(rows) != 1 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	value := rows[0]
	for key, want := range map[string]string{"id": "null", "name": "false", "revision_number": "12", "tags": `["tag"]`, "mtu": "1450", "is_admin_state_up": "true", "is_default": "false", "is_port_security_enabled": "null", "is_router_external": "false", "is_shared": "false", "pvlan": "true", "is_vlan_transparent": "false", "is_vlan_qinq": "true", "availability_zone_hints": "[false]", "availability_zones": `["zone"]`, "segments": `[{"segmentation_id":9007199254740993}]`, "subnet_ids": "null", "provider_segmentation_id": "9007199254740993", "created_at": "9007199254740993", "updated_at": "false", "project_id": "null", "tenant_id": `"extension"`, "if_match": `["row"]`, "IF_MATCH": `"unknown"`, "location": "null"} {
		pythonFindField(t, value, key, want)
	}
	if value.Resource.Body["self"] != nil || value.Resource.Body["router:external"] != nil || value.StatusCode != 203 || string(value.Envelope) != body || value.Header.Get("If-Match") != "page" {
		t.Fatal(value)
	}
	pythonFindWireField(t, value, "mtu", "1450.9")
	pythonFindWireField(t, value, "IF-MATCH", `"row"`)
	pythonFindWireField(t, value, "self", `"wire-only"`)
	pythonFindWireField(t, value, "location", `{"spoof":true}`)
	value.Resource.Body["x_extension"][0] = 'X'
	value.Envelope[0] = 'X'
	value.Header.Set("X-Request-Id", "consumer")
	if string(value.Wire.Body["x_extension"]) != `{"n":9007199254740993}` || value.Resource.Header.Get("X-Request-Id") != "network-list-actual" || value.Wire.Header.Get("X-Request-Id") != "network-list-actual" {
		t.Fatal(value)
	}
}

func TestPythonNetworkListRecordsAllQueryNamesWireAliasesAndCanonicalPresence(t *testing.T) {
	var calls []nativeNetworkCall
	var query url.Values
	api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
		query = req.URL.Query()
		return networkListResponse(200, `{"networks":[]}`)
	})
	options := []networks.RecordListOption{
		networks.WithRecordListLimit(0), networks.WithRecordListMarker("marker"), networks.WithRecordListQuery("description", "d"), networks.WithRecordListQuery("fields", []string{"id", "router:external"}), networks.WithRecordListQuery("name", "n"), networks.WithRecordListQuery("status", "ACTIVE"), networks.WithRecordListQuery("project_id", "p"), networks.WithRecordListQuery("sort_key", []string{"id", "name"}), networks.WithRecordListQuery("sort_dir", []string{"asc", "desc"}), networks.WithRecordListQuery("id", []string{"n1", "n2"}), networks.WithRecordListQuery("ipv4_address_scope_id", "four"), networks.WithRecordListQuery("ipv4_address_scope", "ignored"), networks.WithRecordListQuery("ipv6_address_scope", "six"), networks.WithRecordListQuery("is_admin_state_up", false), networks.WithRecordListQuery("admin_state_up", true), networks.WithRecordListQuery("is_port_security_enabled", nil), networks.WithRecordListQuery("port_security_enabled", true), networks.WithRecordListQuery("is_router_external", true), networks.WithRecordListQuery("router:external", false), networks.WithRecordListQuery("is_shared", false), networks.WithRecordListQuery("shared", true), networks.WithRecordListQuery("provider_network_type", "vxlan"), networks.WithRecordListQuery("provider:network_type", "vlan"), networks.WithRecordListQuery("provider:physical_network", "phys"), networks.WithRecordListQuery("provider_segmentation_id", 0), networks.WithRecordListQuery("provider:segmentation_id", 1), networks.WithRecordListQuery("tags", []any{"a", nil, "b"}), networks.WithRecordListQuery("any_tags", "any"), networks.WithRecordListQuery("tags-any", "ignored"), networks.WithRecordListQuery("not_tags", "not"), networks.WithRecordListQuery("not-tags", "ignored"), networks.WithRecordListQuery("not_any_tags", "not-any"), networks.WithRecordListQuery("not-tags-any", "ignored"), networks.WithRecordListQuery("tenant_id", make(chan int)), networks.WithRecordListQuery("if_match", make(chan int)),
	}
	rows, err := api.AllRecords(context.Background(), options...)
	want := url.Values{"limit": {"0"}, "marker": {"marker"}, "description": {"d"}, "fields": {"id", "router:external"}, "name": {"n"}, "status": {"ACTIVE"}, "project_id": {"p"}, "sort_key": {"id", "name"}, "sort_dir": {"asc", "desc"}, "id": {"n1", "n2"}, "ipv4_address_scope": {"four"}, "ipv6_address_scope": {"six"}, "admin_state_up": {"False"}, "router:external": {"True"}, "shared": {"False"}, "provider:network_type": {"vxlan"}, "provider:physical_network": {"phys"}, "provider:segmentation_id": {"0"}, "tags": {"a", "b"}, "tags-any": {"any"}, "not-tags": {"not"}, "not-tags-any": {"not-any"}}
	if err != nil || rows == nil || len(rows) != 0 || len(calls) != 1 || !reflect.DeepEqual(query, want) {
		t.Fatal(rows, err, calls, query, want)
	}
}

func TestPythonNetworkListRecordsCompleteLocalDescriptorInventory(t *testing.T) {
	var calls []nativeNetworkCall
	body := `{"networks":[{"id":"match","revision_number":"١٢","availability_zone_hints":"az","availability_zones":["zone"],"created_at":{"nested":{"enabled":true,"extra":"keep"}},"dns_domain":"domain.","is_default":0,"mtu":1500.9,"pvlan":"false","qos_policy_id":"q","segments":{"physical_network":"phys","segmentation_id":9007199254740993},"subnets":["s"],"updated_at":"literal","vlan_transparent":0,"vlan_qinq":1e-400},{"id":"skip","revision_number":"١٢","mtu":1500.9}]}`
	api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.RawQuery != "" {
			t.Fatal(req.URL)
		}
		return networkListResponse(200, body)
	})
	options := []networks.RecordListOption{
		networks.WithRecordListQuery("revision_number", 12), networks.WithRecordListQuery("availability_zone_hints", []string{"az"}), networks.WithRecordListQuery("availability_zones", []string{"zone"}), networks.WithRecordListQuery("created_at", map[string]any{"nested": map[string]any{"enabled": 1}}), networks.WithRecordListQuery("dns_domain", "domain."), networks.WithRecordListQuery("is_default", false), networks.WithRecordListQuery("mtu", 1500), networks.WithRecordListQuery("pvlan", true), networks.WithRecordListQuery("qos_policy_id", "q"), networks.WithRecordListQuery("segments", []map[string]any{{"physical_network": "phys", "segmentation_id": json.Number("9007199254740993")}}), networks.WithRecordListQuery("subnet_ids", []string{"s"}), networks.WithRecordListQuery("updated_at", "literal"), networks.WithRecordListQuery("is_vlan_transparent", false), networks.WithRecordListQuery("is_vlan_qinq", false),
	}
	rows, err := api.AllRecords(context.Background(), options...)
	if err != nil || !reflect.DeepEqual(pythonNetworkListIDs(t, rows), []string{"match"}) || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	rows, err = api.AllRecords(context.Background(), networks.WithRecordListQuery("availability_zone_hints", nil), networks.WithRecordListQuery("is_default", nil))
	if err != nil || !reflect.DeepEqual(pythonNetworkListIDs(t, rows), []string{"skip"}) {
		t.Fatal(rows, err)
	}
}

func TestPythonNetworkListRecordsLocalOrderShortCircuitAndDictionaryFailureProof(t *testing.T) {
	for _, lane := range []string{"false-first", "mapping-first", "empty-dictionary", "underflow"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeNetworkCall
			body := `{"networks":[{"id":"one","created_at":"truthy","updated_at":1e-400}]}`
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { return networkListResponse(203, body) })
			var options []networks.RecordListOption
			switch lane {
			case "false-first":
				options = []networks.RecordListOption{networks.WithRecordListQuery("updated_at", true), networks.WithRecordListQuery("created_at", map[string]any{"x": true})}
			case "mapping-first":
				options = []networks.RecordListOption{networks.WithRecordListQuery("created_at", map[string]any{"x": true}), networks.WithRecordListQuery("updated_at", true)}
			case "empty-dictionary":
				options = []networks.RecordListOption{networks.WithRecordListQuery("created_at", map[string]any{})}
			case "underflow":
				options = []networks.RecordListOption{networks.WithRecordListQuery("updated_at", map[string]any{})}
			}
			rows, err := api.AllRecords(context.Background(), options...)
			if lane == "mapping-first" {
				if rows != nil || err == nil {
					t.Fatal(rows, err)
				}
				networkListProof(t, err, 203, body)
			} else if err != nil || rows == nil || (len(rows) == 1) != (lane == "empty-dictionary") {
				t.Fatal(rows, err)
			}
		})
	}
}

func TestPythonNetworkListRecordsContinuationRawCapsBreakAndAtomicAll(t *testing.T) {
	var calls []nativeNetworkCall
	body := `{"networks":[{"id":"skip","mtu":1},{"id":"keep","mtu":2},{"cls":null}],"next":"https://foreign.invalid/networks"}`
	api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { return networkListResponse(203, body) })
	rows, err := api.AllRecords(context.Background(), networks.WithRecordListMaxItems(2), networks.WithRecordListLimit(0), networks.WithRecordListQuery("mtu", 2))
	if err != nil || len(rows) != 1 || len(calls) != 1 || calls[0].query != "limit=2" {
		t.Fatal(rows, err, calls)
	}
	calls = nil
	rows, err = api.AllRecords(context.Background(), networks.WithRecordListMaxItems(1), networks.WithRecordListQuery("mtu", 2))
	if err != nil || rows == nil || len(rows) != 0 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	calls = nil
	for _, err := range api.ListRecords(context.Background(), networks.WithRecordListQuery("mtu", 2)) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if len(calls) != 1 {
		t.Fatal(calls)
	}
	calls = nil
	api = pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") == "later" {
			return networkListResponse(403, `{"error":"late"}`)
		}
		return networkListResponse(200, `{"networks":[{"id":"first"}],"next":"/v2.0/networks?marker=later"}`)
	})
	rows, err = api.AllRecords(context.Background())
	if rows != nil || !gophercloud.ResponseCodeIs(err, 403) || len(calls) != 2 {
		t.Fatal(rows, err, calls)
	}
	calls = nil
	yielded := 0
	for value, err := range api.ListRecords(context.Background()) {
		if err != nil {
			if yielded != 1 || !gophercloud.ResponseCodeIs(err, 403) {
				t.Fatal(yielded, err)
			}
			continue
		}
		if value == nil {
			t.Fatal(value)
		}
		yielded++
	}
	if yielded != 1 || len(calls) != 2 {
		t.Fatal(yielded, calls)
	}
}

func TestPythonNetworkListRecordsMalformedRowsConstructorsAndActualPage(t *testing.T) {
	for _, body := range []string{`{"networks":null}`, `{"networks":false}`, `{"networks":[null]}`, `{}`, `[]`, "", "{bad", `{"networks":[{"cls":null}]}`, `{"networks":[{"connection":null}]}`, `{"networks":[{"microversion":null}]}`, `{"networks":[{"_synchronized":null}]}`, `{"networks":[{"mtu":"²"}]}`, "{\"networks\":[{\"name\":\"\xff\"}]}"} {
		t.Run(fmt.Sprintf("%q", body), func(t *testing.T) {
			var calls []nativeNetworkCall
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { return networkListResponse(203, body) })
			rows, err := api.AllRecords(context.Background())
			if rows != nil || err == nil || len(calls) != 1 {
				t.Fatal(rows, err, calls)
			}
			networkListProof(t, err, 203, body)
		})
	}
	for _, body := range []string{`{"networks":[]}`, `{"networks":{}}`} {
		var calls []nativeNetworkCall
		api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { return networkListResponse(200, body) })
		rows, err := api.AllRecords(context.Background())
		if err != nil || rows == nil || len(calls) != 1 || (len(rows) == 1) != (body == `{"networks":{}}`) {
			t.Fatal(rows, err, calls)
		}
	}
}

type networkListJSONCall struct{ calls *atomic.Int32 }

func (v networkListJSONCall) MarshalJSON() ([]byte, error) {
	v.calls.Add(1)
	return nil, errors.New("caller marshaler must not run")
}
func TestPythonNetworkListRecordsPreflightControlPresenceAndUnknownDiscard(t *testing.T) {
	var calls []nativeNetworkCall
	api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("preflight reached HTTP"); return nil })
	options := [][]networks.RecordListOption{{nil}, {networks.WithRecordListLimit(-1)}, {networks.WithRecordListMaxItems(-1)}, {networks.WithRecordListMarker("")}, {networks.WithRecordListQuery("limit", []int{1, 2})}, {networks.WithRecordListQuery("limit", 1.5)}, {networks.WithRecordListQuery("created_at", json.RawMessage(`bad`))}, {networks.WithRecordListHeaders(map[string]string{"X-Auth-Token": "bad"})}, {networks.WithRecordListHeaders(map[string]string{"OpenStack-API-Version": "network 2.0"})}}
	var marshals atomic.Int32
	for _, key := range []string{"cls", "resource_type", "session", "self"} {
		options = append(options, []networks.RecordListOption{networks.WithRecordListQuery(key, networkListJSONCall{&marshals})}, []networks.RecordListOption{networks.WithRecordListOpts(networks.RecordListOpts{Query: map[string]json.RawMessage{key: json.RawMessage(`null`)}})})
	}
	for _, opts := range options {
		rows, err := api.AllRecords(context.Background(), opts...)
		if rows != nil || err == nil || len(calls) != 0 {
			t.Fatal(rows, err, calls)
		}
		pythonNetworkListOperation(t, err)
	}
	if marshals.Load() != 0 {
		t.Fatal("reserved control marshaled", marshals.Load())
	}
	var nilAPI *networks.API
	if _, err := nilAPI.AllRecords(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.AllRecords(nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	api.RawClient().Type = "image"
	if _, err := api.AllRecords(context.Background()); !errors.Is(err, resource.ErrUnsupported) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	api.RawClient().Type = "network"
	api = pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { return networkListResponse(200, `{"networks":[]}`) })
	rows, err := api.AllRecords(context.Background(), networks.WithRecordListQuery("unknown", networkListJSONCall{&marshals}), networks.WithRecordListQuery("if_match", networkListJSONCall{&marshals}), networks.WithRecordListQuery("all_projects", networkListJSONCall{&marshals}), networks.WithRecordListOpts(networks.RecordListOpts{Query: map[string]json.RawMessage{"another_unknown": json.RawMessage(`invalid`)}}))
	if err != nil || rows == nil || len(calls) != 1 || marshals.Load() != 0 {
		t.Fatal(rows, err, calls, marshals.Load())
	}
}

func TestPythonNetworkListRecordsOwnedOptionsNilMapsLazinessAndConcurrentReuse(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/neutron/", func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		if req.URL.Query().Get("name") != "seed" || req.Header.Get("X-Trace") != "seed" || req.Header.Get("OpenStack-API-Version") != "network 2.1" || req.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(req.URL, req.Header)
		}
		testcloud.JSON(w, 200, `{"networks":[{"id":"one"}]}`)
	})
	api := networks.New(nativeNetworkClient(cloud))
	raw := json.RawMessage(`"seed"`)
	headers := map[string]string{"X-Trace": "seed"}
	version := "2.1"
	option := networks.WithRecordListOpts(networks.RecordListOpts{Query: map[string]json.RawMessage{"name": raw}, Headers: headers, Microversion: &version})
	raw[1] = 'X'
	headers["X-Trace"] = "changed"
	version = "2.2"
	input := []networks.RecordListOption{option}
	sequence := api.ListRecords(context.Background(), input...)
	input[0] = nil
	if calls.Load() != 0 {
		t.Fatal("eager request", calls.Load())
	}
	for _, err := range sequence {
		if err != nil {
			t.Fatal(err)
		}
	}
	var workers sync.WaitGroup
	for i := 0; i < 6; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			rows, err := api.AllRecords(context.Background(), option)
			if err != nil || len(rows) != 1 {
				t.Error(rows, err)
			}
		}()
	}
	workers.Wait()
	if calls.Load() != 7 || api.RawClient().Microversion != "" || api.RawClient().MoreHeaders != nil {
		t.Fatal(calls.Load(), api.RawClient())
	}
	var kept *networks.RecordListOpts
	callback := func(c *networks.RecordListOpts) error {
		c.Query["name"] = json.RawMessage(`"seed"`)
		c.Headers["X-Trace"] = "seed"
		v := "2.1"
		c.Microversion = &v
		kept = c
		return nil
	}
	after := func(c *networks.RecordListOpts) error {
		kept.Query["name"] = json.RawMessage(`"changed"`)
		kept.Headers["X-Trace"] = "changed"
		return nil
	}
	for _, head := range []networks.RecordListOption{networks.WithRecordListHeaders(nil), func(c *networks.RecordListOpts) error { c.Headers = nil; c.Query = nil; return nil }} {
		rows, err := api.AllRecords(context.Background(), head, callback, after)
		if err != nil || len(rows) != 1 {
			t.Fatal(rows, err)
		}
	}
}

func TestPythonNetworkListRecordsSourceMutationAndCanceledOptionAreSticky(t *testing.T) {
	var calls []nativeNetworkCall
	api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("guard reached HTTP"); return nil })
	original := api.RawClient().ResourceBase
	after := false
	mutate := func(c *networks.RecordListOpts) error { api.RawClient().ResourceBase += "other/"; return nil }
	restore := func(c *networks.RecordListOpts) error {
		after = true
		api.RawClient().ResourceBase = original
		return nil
	}
	rows, err := api.AllRecords(context.Background(), mutate, restore)
	api.RawClient().ResourceBase = original
	if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || after || len(calls) != 0 {
		t.Fatal(rows, err, after, calls)
	}
	cause := errors.New("cancel option")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	rows, err = api.AllRecords(ctx, func(*networks.RecordListOpts) error { cancel(cause); return nil })
	if rows != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || len(calls) != 0 {
		t.Fatal(rows, err, calls)
	}
}

type networkListCountedBody struct {
	*pythonFindFaultBody
	closes int
}

func (b *networkListCountedBody) Close() error { b.closes++; return b.pythonFindFaultBody.Close() }
func TestPythonNetworkListRecordsPhysicalFaultsPreserveNative401AndNeverInvokeCallbacks(t *testing.T) {
	for _, status := range []int{200, 203, 299, 401, 403, 404, 503, 599} {
		for _, lane := range []string{"read", "close", "read-close", "cancel", "source"} {
			t.Run(fmt.Sprintf("%d/%s", status, lane), func(t *testing.T) {
				var calls []nativeNetworkCall
				body := `{"networks":[{"id":"one"}]}`
				readCause, closeCause, cancelCause := errors.New("read"), errors.New("close"), errors.New("cancel")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				physical := &networkListCountedBody{pythonFindFaultBody: &pythonFindFaultBody{data: body}}
				api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
					response := networkListResponse(status, body)
					response.Body = physical
					return response
				})
				originalBase := api.RawClient().ResourceBase
				switch lane {
				case "read":
					physical.read = readCause
				case "close":
					physical.close = closeCause
				case "read-close":
					physical.read, physical.close = readCause, closeCause
				case "cancel":
					physical.onClose = func() { cancel(cancelCause) }
				case "source":
					physical.onClose = func() { api.RawClient().ResourceBase += "changed/" }
				}
				retries, reauths := 0, 0
				api.RawClient().ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				api.RawClient().ProviderClient.ReauthFunc = func(context.Context) error { reauths++; return nil }
				rows, err := api.AllRecords(ctx)
				api.RawClient().ResourceBase = originalBase
				if rows != nil || err == nil || len(calls) != 1 || physical.closes != 1 || retries != 0 || reauths != 0 {
					t.Fatal(rows, err, calls, physical.closes, retries, reauths)
				}
				pythonNetworkListOperation(t, err)
				if (lane == "read" || lane == "read-close") && !errors.Is(err, readCause) || (lane == "close" || lane == "read-close") && !errors.Is(err, closeCause) || lane == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) || lane == "source" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if status < 400 {
					networkListProof(t, err, status, body)
				} else {
					receipt := err
					if status == 401 {
						var nativeAuth *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &nativeAuth) || nativeAuth.ErrOriginal == nil || nativeAuth.ErrReauth == nil {
							t.Fatal(err, nativeAuth)
						}
						receipt = nativeAuth.ErrOriginal
						if (lane == "read" || lane == "read-close") && !errors.Is(nativeAuth.ErrReauth, readCause) || (lane == "close" || lane == "read-close") && !errors.Is(nativeAuth.ErrReauth, closeCause) || lane == "cancel" && (!errors.Is(nativeAuth.ErrReauth, context.Canceled) || !errors.Is(nativeAuth.ErrReauth, cancelCause)) || lane == "source" && !errors.Is(nativeAuth.ErrReauth, resource.ErrInvalidOption) {
							t.Fatal(err, nativeAuth)
						}
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(receipt, &native) || native.Actual != status || native.Method != http.MethodGet || native.URL != originalBase+"networks" || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "network-list-actual" {
						t.Fatal(err, native)
					}
				}
			})
		}
	}
}

func TestPythonNetworkListRecordsCleanRetryReauthenticationAndForbiddenMutation(t *testing.T) {
	for _, lane := range []string{"clean", "reauth", "body", "version", "auth", "condition", "codes", "callback"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativeNetworkCall
			var tokens, versions, conditions []string
			api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
				tokens = append(tokens, req.Header.Get("X-Auth-Token"))
				versions = append(versions, req.Header.Get("OpenStack-API-Version"))
				conditions = append(conditions, req.Header.Get("If-Match"))
				if len(calls) == 1 {
					code := 503
					if lane == "reauth" {
						code = 401
					}
					return networkListResponse(code, `{"error":"original"}`)
				}
				if lane == "codes" {
					return networkListResponse(400, `{"error":"expanded"}`)
				}
				return networkListResponse(200, `{"networks":[{"id":"one"}]}`)
			})
			callbackCause := errors.New("callback")
			retries, reauths := 0, 0
			if lane == "reauth" {
				api.RawClient().ProviderClient.ReauthFunc = func(context.Context) error { reauths++; api.RawClient().ProviderClient.SetToken("renewed"); return nil }
			} else {
				api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					retries++
					switch lane {
					case "body":
						opts.JSONBody = map[string]any{"changed": true}
					case "version":
						opts.MoreHeaders["OpenStack-API-Version"] = "network 2.2"
					case "auth":
						opts.MoreHeaders["Authorization"] = "foreign"
					case "condition":
						opts.MoreHeaders["If-Match"] = "new"
					case "codes":
						opts.OkCodes = append(opts.OkCodes, 400)
					case "callback":
						return callbackCause
					}
					return nil
				}
			}
			rows, err := api.AllRecords(context.Background())
			if lane == "clean" || lane == "reauth" {
				if err != nil || len(rows) != 1 || len(calls) != 2 || calls[0] != calls[1] || !reflect.DeepEqual(versions, []string{"", ""}) || !reflect.DeepEqual(conditions, []string{"", ""}) {
					t.Fatal(rows, err, calls, tokens, versions, conditions)
				}
				if lane == "reauth" && (!reflect.DeepEqual(tokens, []string{"test-token", "renewed"}) || reauths != 1) {
					t.Fatal(tokens, reauths)
				}
			} else {
				want, count := 503, 1
				if lane == "codes" {
					want, count = 400, 2
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if rows != nil || !errors.As(err, &native) || native.Actual != want || len(calls) != count || retries != 1 || lane == "callback" && !errors.Is(err, callbackCause) || lane != "callback" && lane != "codes" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(rows, err, native, calls, retries)
				}
			}
		})
	}
}
