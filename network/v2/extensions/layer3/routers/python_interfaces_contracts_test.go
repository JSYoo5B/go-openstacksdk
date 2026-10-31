package routers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonRouterInterfacesSelectionDefaultsRenamedAliasesAndAdvertiseHost(t *testing.T) {
	cachedNull := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`null`)}}}
	cachedID := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"cached-port"`)}}}
	for _, sample := range []struct {
		name, body string
		options    []routers.RouterInterfaceOption
	}{
		{"default", `{"subnet_id":null}`, nil},
		{"port wins", `{"port_id":"port"}`, []routers.RouterInterfaceOption{routers.WithRouterInterfaceSubnet("subnet"), routers.WithRouterInterfacePort("port")}},
		{"empty port", `{"subnet_id":"subnet"}`, []routers.RouterInterfaceOption{routers.WithRouterInterfacePort(""), routers.WithRouterInterfaceSubnet("subnet")}},
		{"false port", `{"subnet_id":false}`, []routers.RouterInterfaceOption{routers.WithRouterInterfacePort(false), routers.WithRouterInterfaceSubnet(false)}},
		{"zero port", `{"subnet_id":0}`, []routers.RouterInterfaceOption{routers.WithRouterInterfacePort(0), routers.WithRouterInterfaceSubnet(0)}},
		{"null port", `{"subnet_id":null}`, []routers.RouterInterfaceOption{routers.WithRouterInterfacePort(nil)}},
		{"empty list port", `{"subnet_id":[]}`, []routers.RouterInterfaceOption{routers.WithRouterInterfacePort([]any{}), routers.WithRouterInterfaceSubnet([]any{})}},
		{"empty object port", `{"subnet_id":{}}`, []routers.RouterInterfaceOption{routers.WithRouterInterfacePort(map[string]any{}), routers.WithRouterInterfaceSubnet(map[string]any{})}},
		{"decimal port underflows", `{"subnet_id":"subnet"}`, []routers.RouterInterfaceOption{routers.WithRouterInterfacePort(json.RawMessage(`1e-4000`)), routers.WithRouterInterfaceSubnet("subnet")}},
		{"JSON object port is passthrough", `{"port_id":{"id":"not a Resource"}}`, []routers.RouterInterfaceOption{routers.WithRouterInterfacePort(map[string]any{"id": "not a Resource"})}},
		{"cached port ID", `{"port_id":"cached-port"}`, []routers.RouterInterfaceOption{routers.WithRouterInterfacePort(cachedID)}},
		{"cached null port wins", `{"port_id":null}`, []routers.RouterInterfaceOption{routers.WithRouterInterfacePort(cachedNull), routers.WithRouterInterfaceSubnet("unused")}},
		{"old port overrides new", `{"port_id":"old-port"}`, []routers.RouterInterfaceOption{routers.WithRouterInterfacePortID("old-port"), routers.WithRouterInterfacePort("new-port")}},
		{"old null overrides new", `{"subnet_id":"subnet"}`, []routers.RouterInterfaceOption{routers.WithRouterInterfacePort("new-port"), routers.WithRouterInterfacePortID(nil), routers.WithRouterInterfaceSubnet("subnet")}},
		{"old subnet overrides new", `{"subnet_id":"old-subnet"}`, []routers.RouterInterfaceOption{routers.WithRouterInterfaceSubnetID("old-subnet"), routers.WithRouterInterfaceSubnet("new-subnet")}},
		{"unselected malformed subnet", `{"port_id":"port"}`, []routers.RouterInterfaceOption{routers.WithRouterInterfaceOpts(routers.RouterInterfaceOpts{Port: &routers.RouterInterfaceRef{Value: json.RawMessage(`"port"`)}, Subnet: &routers.RouterInterfaceRef{Value: json.RawMessage(`{`)}})}},
	} {
		for _, add := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/add=%t", sample.name, add), func(t *testing.T) {
				var calls []nativeRouterCall
				api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
					return nativeRouterWire(200, `{"port_id":"observed","x_extension":null}`)
				})
				var value *routers.RouterJSONResult
				var err error
				route := "remove_router_interface"
				if add {
					value, err = api.AddInterfaceRecord(context.Background(), "literal/id?", sample.options...)
					route = "add_router_interface"
				} else {
					value, err = api.RemoveInterfaceRecord(context.Background(), "literal/id?", sample.options...)
				}
				if err != nil || len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != "/neutron/v2.0/routers/literal/id?/"+route || calls[0].query != "" || strings.TrimSpace(calls[0].body) != sample.body || value.StatusCode != 200 {
					t.Fatal(value, err, calls)
				}
			})
		}
	}
	for _, sample := range []struct {
		host     any
		included bool
	}{
		{false, false}, {nil, false}, {0, false}, {"", false}, {[]any{}, false}, {map[string]any{}, false},
		{true, true}, {1, true}, {"false", true}, {[]any{false}, true},
	} {
		var calls []nativeRouterCall
		api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return nativeRouterWire(200, `null`) })
		value, err := api.AddInterfaceRecord(context.Background(), "router", routers.WithRouterInterfacePort("port"), routers.WithRouterInterfaceAdvertiseHost(sample.host))
		if err != nil || len(calls) != 1 || string(value.Value) != "null" {
			t.Fatal(value, err, calls)
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal([]byte(calls[0].body), &body); err != nil {
			t.Fatal(err)
		}
		raw, present := body["advertise_host"]
		encoded, _ := json.Marshal(sample.host)
		if present != sample.included || present && string(raw) != string(encoded) {
			t.Fatal(sample, body)
		}
	}
}

func TestPythonRouterGatewayStockBridgeAndExplicitLegacyBodies(t *testing.T) {
	for _, sample := range []struct {
		name, route, body string
		add, legacy       bool
		fields            map[string]json.RawMessage
	}{
		{"stock add", "", `{"router":{"external_gateway_info":{"network_id":"external","x_extension":{"null":null,"false":false,"zero":0,"empty":[],"integer":9007199254740993}}}}`, true, false, map[string]json.RawMessage{"network_id": json.RawMessage(`"external"`), "x_extension": json.RawMessage(`{"null":null,"false":false,"zero":0,"empty":[],"integer":9007199254740993}`)}},
		{"stock remove", "", `{"router":{"external_gateway_info":null}}`, false, false, nil},
		{"legacy add", "/add_gateway_router", `{"if_match":null,"if_revision":0,"network_id":"external","x_extension":false}`, true, true, map[string]json.RawMessage{"network_id": json.RawMessage(`"external"`), "if_match": json.RawMessage(`null`), "if_revision": json.RawMessage(`0`), "x_extension": json.RawMessage(`false`)}},
		{"legacy remove", "/remove_gateway_router", `{"network_id":"external","x_extension":null}`, false, true, map[string]json.RawMessage{"network_id": json.RawMessage(`"external"`), "x_extension": json.RawMessage(`null`)}},
		{"legacy add no defaults", "/add_gateway_router", `{}`, true, true, nil},
		{"legacy remove no defaults", "/remove_gateway_router", `{}`, false, true, nil},
	} {
		t.Run(sample.name, func(t *testing.T) {
			var calls []nativeRouterCall
			var escaped, match string
			api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
				escaped, match = req.URL.EscapedPath(), req.Header.Get("If-Match")
				return nativeRouterWire(200, `{"router":{"id":"actual","external_gateway_info":null},"x_response":false}`)
			})
			options := []routers.RouterGatewayOption{routers.WithRouterGatewayOpts(routers.RouterGatewayOpts{Fields: sample.fields}), routers.WithSourceLegacyGatewayRoutes(sample.legacy)}
			var value *routers.RouterJSONResult
			var err error
			if sample.add {
				value, err = api.AddGatewayRecord(context.Background(), "literal/id?", options...)
			} else {
				value, err = api.RemoveGatewayRecord(context.Background(), "literal/id?", options...)
			}
			if err != nil || len(calls) != 1 || calls[0].method != http.MethodPut || strings.TrimSpace(calls[0].body) != sample.body || escaped != "/neutron/v2.0/routers/literal%2Fid%3F"+sample.route || match != "" {
				t.Fatal(value, err, calls, escaped, match)
			}
			if value.Wire == nil || string(value.Wire.Body["x_response"]) != "false" || value.Wire.Body["router"] == nil || value.Wire.Body["id"] != nil {
				t.Fatal(value)
			}
		})
	}
	for _, sample := range []struct {
		add     bool
		options []routers.RouterGatewayOption
	}{
		{true, nil}, {true, []routers.RouterGatewayOption{routers.WithRouterGatewayField("network_id", nil)}},
		{true, []routers.RouterGatewayOption{routers.WithRouterGatewayField("network_id", "")}},
		{false, []routers.RouterGatewayOption{routers.WithRouterGatewayField("network_id", "external")}},
		{true, []routers.RouterGatewayOption{routers.WithSourceLegacyGatewayRoutes(true), routers.WithRouterGatewayField("router", map[string]any{})}},
		{false, []routers.RouterGatewayOption{routers.WithSourceLegacyGatewayRoutes(true), routers.WithRouterGatewayField("session", false)}},
	} {
		var calls []nativeRouterCall
		api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return nativeRouterWire(200, `{}`) })
		var err error
		if sample.add {
			_, err = api.AddGatewayRecord(context.Background(), "router", sample.options...)
		} else {
			_, err = api.RemoveGatewayRecord(context.Background(), "router", sample.options...)
		}
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	}
}

type pythonRouterJSONCall struct {
	operation string
	invoke    func(*routers.API, context.Context, string) (*routers.RouterJSONResult, error)
}

var pythonRouterJSONCalls = []pythonRouterJSONCall{
	{"AddInterfaceRecord", func(a *routers.API, ctx context.Context, id string) (*routers.RouterJSONResult, error) {
		return a.AddInterfaceRecord(ctx, id, routers.WithRouterInterfaceSubnet("subnet"))
	}},
	{"RemoveInterfaceRecord", func(a *routers.API, ctx context.Context, id string) (*routers.RouterJSONResult, error) {
		return a.RemoveInterfaceRecord(ctx, id, routers.WithRouterInterfacePort("port"))
	}},
	{"AddGatewayRecord", func(a *routers.API, ctx context.Context, id string) (*routers.RouterJSONResult, error) {
		return a.AddGatewayRecord(ctx, id, routers.WithRouterGatewayField("network_id", "external"))
	}},
	{"RemoveGatewayRecord", func(a *routers.API, ctx context.Context, id string) (*routers.RouterJSONResult, error) {
		return a.RemoveGatewayRecord(ctx, id)
	}},
}

func TestPythonRouterInterfacesWholeJSONValuesStrictStatusesAndDecodeEvidence(t *testing.T) {
	for _, call := range pythonRouterJSONCalls {
		for _, body := range []string{`{"id":"actual","x_extension":9007199254740993}`, `{"router":{"id":"nested"}}`, `[]`, `"text"`, `false`, `0`, `null`} {
			var calls []nativeRouterCall
			reply := nativeRouterWire(200, body)
			reply.Header.Set("X-Request-Id", "whole-JSON")
			api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return reply })
			value, err := call.invoke(api, context.Background(), "router")
			if err != nil || len(calls) != 1 || string(value.Value) != body || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "whole-JSON" {
				t.Fatal(call.operation, value, err, calls)
			}
			if strings.HasPrefix(body, "{") != (value.Wire != nil) {
				t.Fatal(value)
			}
			value.Value[0] = 'X'
			reply.Header.Set("X-Request-Id", "changed")
			if string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "whole-JSON" {
				t.Fatal(value)
			}
		}
		for _, status := range []int{200, 201, 202, 204, 299, 300, 399, 400, 403, 404, 409, 429, 500, 599} {
			t.Run(fmt.Sprintf("%s/%d", call.operation, status), func(t *testing.T) {
				const body = `{"error":"actual JSON"}`
				var calls []nativeRouterCall
				api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
					reply := nativeRouterWire(status, body)
					reply.Header.Set("X-Request-Id", "actual-status")
					return reply
				})
				value, err := call.invoke(api, context.Background(), "router")
				if len(calls) != 1 {
					t.Fatal(calls)
				}
				if status < 400 {
					if err != nil || value.StatusCode != status || string(value.Value) != body {
						t.Fatal(value, err)
					}
				} else {
					nativeRouterOperation(t, err, call.operation)
					var native gophercloud.ErrUnexpectedResponseCode
					if value != nil || !errors.As(err, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "actual-status" {
						t.Fatal(value, err, native)
					}
				}
			})
		}
		for _, body := range []string{"", `{"incomplete":`, `{} {}`, "not JSON"} {
			var calls []nativeRouterCall
			api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return nativeRouterWire(204, body) })
			value, err := call.invoke(api, context.Background(), "router")
			nativeRouterOperation(t, err, call.operation)
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || value == nil || value.Value != nil || string(value.Envelope) != body || proof.StatusCode != 204 || string(proof.Body) != body || len(calls) != 1 {
				t.Fatal(value, err, proof, calls)
			}
			if body == "" && !errors.Is(err, io.EOF) {
				t.Fatal(err)
			}
		}
	}
	// Route compatibility is explicit; legacy Source's unchecked HTTP JSON is
	// intentionally repaired even when the supporting legacy backend is used.
	for _, add := range []bool{true, false} {
		var calls []nativeRouterCall
		api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return nativeRouterWire(404, `{"error":"legacy path absent"}`) })
		var err error
		if add {
			_, err = api.AddGatewayRecord(context.Background(), "router", routers.WithSourceLegacyGatewayRoutes(true))
		} else {
			_, err = api.RemoveGatewayRecord(context.Background(), "router", routers.WithSourceLegacyGatewayRoutes(true))
		}
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 404 || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	}
}

func TestPythonRouterInterfacesPhysicalReadCloseCancelAndSourceFaultsRemainTerminal(t *testing.T) {
	for _, call := range pythonRouterJSONCalls {
		for _, status := range []int{200, 204, 400, 404, 429, 500} {
			for _, lane := range []string{"read and close", "cancel during read", "source restored on Close"} {
				t.Run(fmt.Sprintf("%s/%d/%s", call.operation, status, lane), func(t *testing.T) {
					cloud := testcloud.New(t)
					client := nativeRouterClient(cloud)
					original := client.ResourceBase
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					readCause, closeCause, cancelCause := errors.New("actual read"), errors.New("actual Close"), errors.New("caller stopped JSON action")
					physical := &pythonRouterActionObservedBody{body: []byte(`{"wire":"actual"}`)}
					switch lane {
					case "read and close":
						physical.readErr, physical.closeErr = readCause, closeCause
					case "cancel during read":
						physical.onRead = func() { cancel(cancelCause) }
					default:
						physical.onRead = func() { client.ResourceBase = cloud.Server.URL + "/changed/" }
						physical.onClose = func() { client.ResourceBase = original }
					}
					var calls []nativeRouterCall
					nativeRouterRecorder(cloud, &calls, func(*http.Request) *http.Response {
						reply := nativeRouterWire(status, "")
						reply.Body = physical
						reply.Header.Set("X-Request-Id", "physical-JSON")
						return reply
					})
					value, err := call.invoke(routers.New(client), ctx, "router")
					nativeRouterOperation(t, err, call.operation)
					if len(calls) != 1 || physical.closed != 1 {
						t.Fatal(calls, physical.closed, err)
					}
					switch lane {
					case "read and close":
						if !errors.Is(err, readCause) || !errors.Is(err, closeCause) {
							t.Fatal(err)
						}
					case "cancel during read":
						if !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) {
							t.Fatal(err)
						}
					default:
						if !errors.Is(err, resource.ErrInvalidOption) || client.ResourceBase != original {
							t.Fatal(err)
						}
					}
					if status < 400 {
						var proof *resource.ResponseError
						if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != string(physical.body) || proof.Header.Get("X-Request-Id") != "physical-JSON" || value == nil || value.Value != nil {
							t.Fatal(value, err, proof)
						}
					} else {
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(err, &native) || native.Actual != status || string(native.Body) != string(physical.body) || native.ResponseHeader.Get("X-Request-Id") != "physical-JSON" || value != nil {
							t.Fatal(value, err, native)
						}
					}
				})
			}
		}
	}
}

func TestPythonRouterInterfacesOptionsOwnSnapshotsAndContextSourcePreflight(t *testing.T) {
	ref := map[string]any{"x_extension": []any{false, 0, nil}}
	headers := map[string]string{"X-Interface": "owned"}
	portOption, headerOption := routers.WithRouterInterfacePort(ref), routers.WithRouterInterfaceHeaders(headers)
	ref["x_extension"] = nil
	headers["X-Interface"] = "changed"
	var retained *routers.RouterInterfaceOpts
	var calls []nativeRouterCall
	api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Header.Get("X-Interface") != "owned" {
			t.Fatal(req.Header)
		}
		retained.Port.Value[0] = 'X'
		retained.Headers["X-Interface"] = "retained mutation"
		return nativeRouterWire(200, `{}`)
	})
	_, err := api.AddInterfaceRecord(context.Background(), "router", portOption, headerOption, func(config *routers.RouterInterfaceOpts) error { retained = config; return nil })
	if err != nil || len(calls) != 1 || strings.TrimSpace(calls[0].body) != `{"port_id":{"x_extension":[false,0,null]}}` {
		t.Fatal(err, calls)
	}
	for _, gateway := range []bool{false, true} {
		for _, lane := range []string{"nil context", "canceled callback", "source callback", "protected header", "nil option", "invalid body", "remove advertise_host"} {
			if gateway && lane == "remove advertise_host" {
				continue
			}
			t.Run(fmt.Sprintf("gateway=%t/%s", gateway, lane), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := nativeRouterClient(cloud)
				api := routers.New(client)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				var actual context.Context = ctx
				cause := errors.New("option cause")
				calls := 0
				cloud.SetTransport(testcloud.LinkRequest(func(*http.Request) (*http.Response, error) { calls++; return nativeRouterWire(200, `{}`), nil }))
				var err error
				if gateway {
					options := []routers.RouterGatewayOption{routers.WithRouterGatewayField("network_id", "external")}
					switch lane {
					case "nil context":
						actual = nil
					case "canceled callback":
						options = append(options, func(*routers.RouterGatewayOpts) error { cancel(cause); return cause })
					case "source callback":
						options = append(options, func(*routers.RouterGatewayOpts) error { client.Microversion = "changed"; return nil })
					case "protected header":
						options = append(options, routers.WithRouterGatewayHeaders(map[string]string{"X-Auth-Token": "override"}))
					case "nil option":
						options = append(options, nil)
					case "invalid body":
						options = append(options, routers.WithRouterGatewayOpts(routers.RouterGatewayOpts{Fields: map[string]json.RawMessage{"network_id": json.RawMessage(`{`)}}))
					}
					_, err = api.AddGatewayRecord(actual, "router", options...)
					nativeRouterOperation(t, err, "AddGatewayRecord")
				} else {
					var options []routers.RouterInterfaceOption
					switch lane {
					case "nil context":
						actual = nil
					case "canceled callback":
						options = []routers.RouterInterfaceOption{func(*routers.RouterInterfaceOpts) error { cancel(cause); return cause }}
					case "source callback":
						options = []routers.RouterInterfaceOption{func(*routers.RouterInterfaceOpts) error {
							client.ResourceBase = cloud.Server.URL + "/changed/"
							return nil
						}}
					case "protected header":
						options = []routers.RouterInterfaceOption{routers.WithRouterInterfaceHeaders(map[string]string{"Authorization": "override"})}
					case "nil option":
						options = []routers.RouterInterfaceOption{nil}
					case "invalid body":
						options = []routers.RouterInterfaceOption{routers.WithRouterInterfaceOpts(routers.RouterInterfaceOpts{Subnet: &routers.RouterInterfaceRef{Value: json.RawMessage(`{`)}})}
					case "remove advertise_host":
						options = []routers.RouterInterfaceOption{routers.WithRouterInterfaceAdvertiseHost(false)}
					}
					_, err = api.RemoveInterfaceRecord(actual, "router", options...)
					nativeRouterOperation(t, err, "RemoveInterfaceRecord")
				}
				if err == nil || calls != 0 {
					t.Fatal(err, calls)
				}
				if lane == "canceled callback" {
					if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestPythonRouterInterfacesRetryReauthAndConcurrentInputsKeepOwnership(t *testing.T) {
	for _, call := range pythonRouterJSONCalls {
		for _, lane := range []string{"clean retry", "body mutation", "source mutation", "reauthentication"} {
			cloud := testcloud.New(t)
			client := nativeRouterClient(cloud)
			var calls []nativeRouterCall
			var tokens []string
			nativeRouterRecorder(cloud, &calls, func(req *http.Request) *http.Response {
				tokens = append(tokens, req.Header.Get("X-Auth-Token"))
				if len(calls) == 1 {
					if lane == "reauthentication" {
						return nativeRouterWire(401, `{"error":"expired"}`)
					}
					return nativeRouterWire(503, `{"error":"retryable"}`)
				}
				return nativeRouterWire(200, `{"ack":true}`)
			})
			if lane == "reauthentication" {
				cloud.Provider.ReauthFunc = func(context.Context) error { cloud.Provider.SetToken("refreshed"); return nil }
			} else {
				cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					if lane == "body mutation" {
						opts.JSONBody = map[string]any{"changed": true}
					} else if lane == "source mutation" {
						client.Type = "compute"
					}
					return nil
				}
			}
			value, err := call.invoke(routers.New(client), context.Background(), "router")
			if lane == "clean retry" || lane == "reauthentication" {
				if err != nil || value == nil || len(calls) != 2 || calls[0] != calls[1] {
					t.Fatal(call.operation, lane, value, err, calls)
				}
				if lane == "reauthentication" && !reflect.DeepEqual(tokens, []string{"test-token", "refreshed"}) {
					t.Fatal(tokens)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || len(calls) != 1 {
					t.Fatal(value, err, native, calls)
				}
			}
		}
	}
	cloud := testcloud.New(t)
	api := routers.New(nativeRouterClient(cloud))
	option := routers.WithRouterInterfacePort("shared-port")
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		return nativeRouterWire(200, string(raw)), nil
	}))
	values := make([]*routers.RouterJSONResult, 8)
	failures := make([]error, 8)
	var workers sync.WaitGroup
	for index := range values {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			values[index], failures[index] = api.AddInterfaceRecord(context.Background(), fmt.Sprint(index), option)
		}(index)
	}
	workers.Wait()
	for index, value := range values {
		if failures[index] != nil || string(value.Value) != `{"port_id":"shared-port"}` {
			t.Fatal(value, failures[index])
		}
		value.Value[0] = 'X'
		if string(value.Envelope) != `{"port_id":"shared-port"}` || string(value.Wire.Body["port_id"]) != `"shared-port"` {
			t.Fatal(value)
		}
	}
}

func TestPythonRouterInterfacesRetryCannotChangeProtocolOrJSONHeaders(t *testing.T) {
	cases := []struct {
		name, version string
		change        func(*gophercloud.RequestOpts)
	}{
		{"selected_changed", "2.0", func(o *gophercloud.RequestOpts) { o.MoreHeaders["OpenStack-API-Version"] = "network changed" }},
		{"selected_deleted", "2.0", func(o *gophercloud.RequestOpts) { delete(o.MoreHeaders, "OpenStack-API-Version") }},
		{"selected_case_alias", "2.0", func(o *gophercloud.RequestOpts) { o.MoreHeaders["openstack-api-version"] = "network changed" }},
		{"selected_omitted", "2.0", func(o *gophercloud.RequestOpts) { o.OmitHeaders = []string{"oPeNsTaCk-ApI-vErSiOn"} }},
		{"absent_added", "", func(o *gophercloud.RequestOpts) { o.MoreHeaders["OpenStack-API-Version"] = "network 2.0" }},
		{"nova_added", "2.0", func(o *gophercloud.RequestOpts) { o.MoreHeaders["X-OpenStack-Nova-API-Version"] = "2.99" }},
		{"volume_added", "", func(o *gophercloud.RequestOpts) { o.MoreHeaders["x-openstack-volume-api-version"] = "3.99" }},
		{"authorization_added", "", func(o *gophercloud.RequestOpts) { o.MoreHeaders["Authorization"] = "foreign" }},
		{"framing_added", "2.0", func(o *gophercloud.RequestOpts) { o.MoreHeaders["Transfer-Encoding"] = "chunked" }},
		{"content_type_changed", "", func(o *gophercloud.RequestOpts) { o.MoreHeaders["Content-Type"] = "application/octet-stream" }},
		{"content_type_omitted", "2.0", func(o *gophercloud.RequestOpts) { o.OmitHeaders = []string{"content-type"} }},
	}
	for _, call := range pythonRouterJSONCalls {
		for _, sample := range cases {
			for _, callbackFailure := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/callbackFailure=%t", call.operation, sample.name, callbackFailure), func(t *testing.T) {
					const body = `{"error":"original physical 503"}`
					var calls []nativeRouterCall
					api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
						want := ""
						if sample.version != "" {
							want = "network " + sample.version
						}
						if req.Header.Get("OpenStack-API-Version") != want || req.Header.Get("Content-Type") != "application/json" {
							t.Error(req.Header, want)
						}
						response := nativeRouterWire(503, body)
						response.Header.Set("X-Request-Id", "original-header")
						return response
					})
					client := api.RawClient()
					client.Microversion = sample.version
					callbacks := 0
					cause := errors.New("native callback cause")
					client.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
						callbacks++
						sample.change(opts)
						if callbackFailure {
							return cause
						}
						return nil
					}
					value, err := call.invoke(api, context.Background(), "router")
					nativeRouterOperation(t, err, call.operation)
					var native gophercloud.ErrUnexpectedResponseCode
					if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "original-header" || native.Method != http.MethodPut || native.URL == "" || callbacks != 1 || len(calls) != 1 || client.Microversion != sample.version {
						t.Fatal(value, err, native, callbacks, calls)
					}
					if callbackFailure && !errors.Is(err, cause) {
						t.Fatal(err)
					}
				})
			}
		}
	}
}

func TestPythonRouterInterfacesOrdinaryRetryHeadersRemainNative(t *testing.T) {
	for _, call := range pythonRouterJSONCalls {
		t.Run(call.operation, func(t *testing.T) {
			var calls []nativeRouterCall
			api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.Header.Get("OpenStack-API-Version") != "network 2.0" || req.Header.Get("Content-Type") != "application/json" {
					t.Error(req.Header)
				}
				if len(calls) == 1 {
					return nativeRouterWire(503, `{"error":"retryable"}`)
				}
				if req.Header.Get("X-Retry") != "ordinary change" {
					t.Error(req.Header)
				}
				return nativeRouterWire(200, `{"ack":true}`)
			})
			client := api.RawClient()
			client.Microversion = "2.0"
			callbacks := 0
			client.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				callbacks++
				opts.MoreHeaders["X-Retry"] = "ordinary change"
				return nil
			}
			value, err := call.invoke(api, context.Background(), "router")
			if err != nil || value == nil || string(value.Value) != `{"ack":true}` || callbacks != 1 || len(calls) != 2 || calls[0] != calls[1] {
				t.Fatal(value, err, callbacks, calls)
			}
		})
	}
}
