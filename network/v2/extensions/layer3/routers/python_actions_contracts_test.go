package routers_test

import (
	"bytes"
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

type pythonRouterActionCall struct {
	operation, route string
	call             func(*routers.API, context.Context, string, any, ...routers.RouterActionOption) (*routers.RouterActionRecord, error)
}

var pythonRouterActionCalls = []pythonRouterActionCall{
	{"AddExtraRoutesRecord", "add_extraroutes", (*routers.API).AddExtraRoutesRecord},
	{"RemoveExtraRoutesRecord", "remove_extraroutes", (*routers.API).RemoveExtraRoutesRecord},
	{"AddExternalGatewaysRecord", "add_external_gateways", (*routers.API).AddExternalGatewaysRecord},
	{"UpdateExternalGatewaysRecord", "update_external_gateways", (*routers.API).UpdateExternalGatewaysRecord},
	{"RemoveExternalGatewaysRecord", "remove_external_gateways", (*routers.API).RemoveExternalGatewaysRecord},
}

func pythonRouterActionValue(t *testing.T, value *resource.RawResource, key, want string) {
	t.Helper()
	if value == nil || string(value.Body[key]) != want {
		t.Fatalf("%s: got %+v, want %s", key, value, want)
	}
}

func TestPythonRouterActionsPreserveRawBodyFixedRoutesAndActualAcknowledgement(t *testing.T) {
	const body = `{"router":{"routes":[],"external_gateways":[{"network_id":"ext","x_extension":{"null":null,"false":false,"zero":0,"empty":[],"integer":9007199254740993}}],"is_admin_state_up":false,"if_match":null,"if_revision":0},"outside":{}}`
	for _, action := range pythonRouterActionCalls {
		t.Run(action.operation, func(t *testing.T) {
			var calls []nativeRouterCall
			var escaped, header string
			api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
				escaped, header = req.URL.EscapedPath(), req.Header.Get("If-Match")
				return nativeRouterWire(200, `{"router":{"id":"actual-id","routes":[],"x_ack":null},"response_extension":false}`)
			})
			value, err := action.call(api, context.Background(), "router/id?#", json.RawMessage(body))
			if err != nil || len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].query != "" || strings.TrimSpace(calls[0].body) != body || escaped != "/neutron/v2.0/routers/router%2Fid%3F%23/"+action.route || header != "" {
				t.Fatal(value, err, calls, escaped, header)
			}
			pythonRouterActionValue(t, value.Resource, "id", `"actual-id"`)
			pythonRouterActionValue(t, value.Resource, "routes", `[]`)
			pythonRouterActionValue(t, value.Resource, "x_ack", `null`)
			if _, synthetic := value.Wire.Body["tags"]; synthetic {
				t.Fatal(value.Wire)
			}
			if _, requestSeed := value.Resource.Body["outside"]; requestSeed {
				t.Fatal(value.Resource)
			}
			if !bytes.Contains(value.Envelope, []byte(`"response_extension":false`)) || value.StatusCode != 200 {
				t.Fatal(value)
			}
		})
	}
	for _, body := range []any{map[string]any{}, nil, json.RawMessage(`null`), json.RawMessage(`[]`)} {
		var calls []nativeRouterCall
		api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return nativeRouterWire(204, "") })
		value, err := api.AddExtraRoutesRecord(context.Background(), "literal-id", body)
		want := ""
		if body != nil {
			raw, _ := json.Marshal(body)
			want = string(raw)
		}
		if err != nil || len(calls) != 1 || strings.TrimSpace(calls[0].body) != want || value.Wire != nil || value.StatusCode != 204 {
			t.Fatal(body, value, err, calls)
		}
		pythonRouterActionValue(t, value.Resource, "id", `"literal-id"`)
		pythonRouterActionValue(t, value.Resource, "routes", `null`)
		pythonRouterActionValue(t, value.Resource, "tags", `[]`)
	}
}

func TestPythonRouterActionsProjectDescriptorsAliasesDefaultsAndOwnedWire(t *testing.T) {
	const row = `{"id":null,"admin_state_up":false,"is_admin_state_up":"false","distributed":0e-4000,"ha":[],"enable_ndp_proxy":{},"revision":7.8,"revision_number":"009","evpn_vni":true,"external_gateway_info":[],"routes":{},"availability_zones":"az1","availability_zone_hints":null,"tenant_id":"tenant","tags":null,"x_extension":9007199254740993,"Name":"case-sensitive extension","self":"reserved","if_match":"body header","if-match":"body wire header","location":"body location"}`
	var calls []nativeRouterCall
	original := nativeRouterWire(200, `{"router":`+row+`}`)
	original.Header["if_match"] = []string{"revision_number=9", "extension=one"}
	original.Header.Set("X-Request-Id", "source-descriptors")
	api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return original })
	value, err := api.UpdateExternalGatewaysRecord(context.Background(), "seed-id", map[string]any{"router": map[string]any{"name": "request-name"}})
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"id": "null", "is_admin_state_up": "true", "is_distributed": "false", "is_ha": "false", "enable_ndp_proxy": "false",
		"revision_number": "9", "evpn_vni": "true", "external_gateway_info": "{}", "routes": "[{}]",
		"availability_zones": `["az1"]`, "availability_zone_hints": "null", "project_id": `"tenant"`, "tenant_id": `"tenant"`, "tags": "null",
		"x_extension": "9007199254740993", "Name": `"case-sensitive extension"`, "name": "null", "if_match": `["revision_number=9, extension=one"]`,
		"if-match": `"body wire header"`, "location": "null",
	} {
		pythonRouterActionValue(t, value.Resource, key, want)
	}
	if _, reserved := value.Resource.Body["self"]; reserved {
		t.Fatal(value.Resource)
	}
	if _, wireAlias := value.Resource.Body["admin_state_up"]; wireAlias {
		t.Fatal(value.Resource)
	}
	pythonRouterActionValue(t, value.Wire, "self", `"reserved"`)
	pythonRouterActionValue(t, value.Wire, "admin_state_up", `false`)
	pythonRouterActionValue(t, value.Wire, "revision", `7.8`)
	if value.Header.Get("X-Request-Id") != "source-descriptors" || value.Resource.StatusCode != 200 || value.Wire.StatusCode != 200 {
		t.Fatal(value)
	}
	original.Header.Set("X-Request-Id", "mutated original")
	value.Resource.Body["x_extension"][0] = '0'
	value.Resource.Header.Set("X-Request-Id", "mutated view")
	value.Wire.Header.Set("X-Request-Id", "mutated wire")
	pythonRouterActionValue(t, value.Wire, "x_extension", "9007199254740993")
	if value.Header.Get("X-Request-Id") != "source-descriptors" || !bytes.Contains(value.Envelope, []byte("9007199254740993")) {
		t.Fatal(value)
	}

	for _, sample := range []struct{ row, admin, revision, project, tags string }{
		{`{"is_admin_state_up":false,"admin_state_up":"false","revision_number":2,"revision":3,"tenant_id":"tenant","project_id":null}`, "true", "3", "null", "[]"},
		{`{"admin_state_up":1e-4000,"revision":"-2","tenant_id":false}`, "false", "0", "false", "[]"},
		{`{"admin_state_up":1e4000,"revision":-2.9,"tags":{}}`, "true", "-2", "null", "[{}]"},
		{`{}`, "null", "null", "null", "[]"},
	} {
		var calls []nativeRouterCall
		api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return nativeRouterWire(200, sample.row) })
		value, err := api.RemoveExtraRoutesRecord(context.Background(), "seed", map[string]any{})
		if err != nil {
			t.Fatal(sample, err)
		}
		for key, want := range map[string]string{"is_admin_state_up": sample.admin, "revision_number": sample.revision, "project_id": sample.project, "tags": sample.tags, "id": `"seed"`, "if_match": "null"} {
			pythonRouterActionValue(t, value.Resource, key, want)
		}
	}
}

func TestPythonRouterActionsSourceStatusTranslationAndErrorEvidence(t *testing.T) {
	for _, action := range pythonRouterActionCalls {
		for _, status := range []int{200, 201, 202, 204, 299, 300, 399, 400, 403, 404, 409, 429, 500, 599} {
			t.Run(fmt.Sprintf("%s/%d", action.operation, status), func(t *testing.T) {
				const body = `{"router":{"id":"returned","x_observed":false}}`
				var calls []nativeRouterCall
				api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
					response := nativeRouterWire(status, body)
					response.Header.Set("X-Request-Id", "actual-action")
					return response
				})
				value, err := action.call(api, context.Background(), "seed", map[string]any{})
				if len(calls) != 1 {
					t.Fatal(calls)
				}
				if status < 400 {
					if err != nil || value.StatusCode != status {
						t.Fatal(value, err)
					}
					pythonRouterActionValue(t, value.Resource, "id", `"returned"`)
				} else {
					nativeRouterOperation(t, err, action.operation)
					var native gophercloud.ErrUnexpectedResponseCode
					if value != nil || !errors.As(err, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "actual-action" || native.Method != http.MethodPut {
						t.Fatal(value, err, native)
					}
				}
			})
		}
	}
	for _, sample := range []struct {
		body  string
		fails bool
	}{
		{"", false}, {"not JSON", false}, {`{"router":`, false},
		{`null`, true}, {`[]`, true}, {`1`, true}, {`{"router":null}`, true}, {`{"router":[]}`, true},
		{`{"router":{"revision":"²"}}`, true},
	} {
		t.Run("translation/"+sample.body, func(t *testing.T) {
			var calls []nativeRouterCall
			api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
				response := nativeRouterWire(202, sample.body)
				response.Header.Set("X-Request-Id", "translation")
				return response
			})
			value, err := api.AddExternalGatewaysRecord(context.Background(), "seed", map[string]any{"request_only": true})
			if value == nil || value.StatusCode != 202 || string(value.Envelope) != sample.body || len(calls) != 1 {
				t.Fatal(value, err, calls)
			}
			if !sample.fails {
				if err != nil {
					t.Fatal(err)
				}
				pythonRouterActionValue(t, value.Resource, "id", `"seed"`)
				pythonRouterActionValue(t, value.Resource, "routes", "null")
			} else {
				nativeRouterOperation(t, err, "AddExternalGatewaysRecord")
				var evidence *resource.ResponseError
				if !errors.As(err, &evidence) || !errors.Is(err, resource.ErrInvalidOption) || string(evidence.Body) != sample.body || evidence.StatusCode != 202 || evidence.Header.Get("X-Request-Id") != "translation" || value.Resource != nil {
					t.Fatal(value, err, evidence)
				}
			}
		})
	}
}

type pythonRouterActionObservedBody struct {
	body              []byte
	readErr, closeErr error
	onRead, onClose   func()
	read              bool
	closed            int
}

func (body *pythonRouterActionObservedBody) Read(target []byte) (int, error) {
	if body.read {
		return 0, io.EOF
	}
	body.read = true
	if body.onRead != nil {
		body.onRead()
	}
	n := copy(target, body.body)
	if body.readErr != nil {
		return n, body.readErr
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}
func (body *pythonRouterActionObservedBody) Close() error {
	body.closed++
	if body.onClose != nil {
		body.onClose()
	}
	return body.closeErr
}

func TestPythonRouterActionsPreservePhysicalReadCloseCancelAndSourceFaults(t *testing.T) {
	for _, status := range []int{200, 204, 399, 400, 403, 404, 409, 429, 500, 599} {
		for _, lane := range []string{"read and close", "cancel during read", "source during read then restored on close"} {
			t.Run(fmt.Sprintf("%d/%s", status, lane), func(t *testing.T) {
				readCause, closeCause, cancelCause := errors.New("physical read failed"), errors.New("physical close failed"), errors.New("caller stopped action")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cloud := testcloud.New(t)
				client := nativeRouterClient(cloud)
				originalBase := client.ResourceBase
				physical := &pythonRouterActionObservedBody{body: []byte(`{"router":{"id":"wire-id"}}`)}
				switch lane {
				case "read and close":
					physical.readErr, physical.closeErr = readCause, closeCause
				case "cancel during read":
					physical.onRead = func() { cancel(cancelCause) }
				default:
					physical.onRead = func() { client.ResourceBase = cloud.Server.URL + "/changed/v2.0/" }
					physical.onClose = func() { client.ResourceBase = originalBase }
				}
				var calls []nativeRouterCall
				nativeRouterRecorder(cloud, &calls, func(*http.Request) *http.Response {
					response := nativeRouterWire(status, "")
					response.Body = physical
					response.Header.Set("X-Request-Id", "physical-action")
					return response
				})
				value, err := routers.New(client).RemoveExternalGatewaysRecord(ctx, "seed", map[string]any{})
				nativeRouterOperation(t, err, "RemoveExternalGatewaysRecord")
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
					if !errors.Is(err, resource.ErrInvalidOption) || client.ResourceBase != originalBase {
						t.Fatal(err, client.ResourceBase)
					}
				}
				if status < 400 {
					var evidence *resource.ResponseError
					if !errors.As(err, &evidence) || evidence.StatusCode != status || string(evidence.Body) != string(physical.body) || evidence.Header.Get("X-Request-Id") != "physical-action" || value == nil || value.Resource != nil || string(value.Envelope) != string(physical.body) {
						t.Fatal(value, err, evidence)
					}
					pythonRouterActionValue(t, value.Wire, "id", `"wire-id"`)
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != status || string(native.Body) != string(physical.body) || native.ResponseHeader.Get("X-Request-Id") != "physical-action" || value != nil {
						t.Fatal(value, err, native)
					}
				}
			})
		}
	}
}

type pythonRouterActionMarshaler func() ([]byte, error)

func (callback pythonRouterActionMarshaler) MarshalJSON() ([]byte, error) { return callback() }

func TestPythonRouterActionsCaptureBeforeCallbacksAndRejectInvalidInputs(t *testing.T) {
	cloud := testcloud.New(t)
	client := nativeRouterClient(cloud)
	client.MoreHeaders = map[string]string{"X-Selected": "before"}
	input := map[string]any{"router": map[string]any{"routes": []any{}, "x_extension": false}}
	headers := map[string]string{"X-Action": "owned"}
	owned := routers.WithRouterActionHeaders(headers)
	headers["X-Action"] = "later"
	var retained *routers.RouterActionOpts
	var seen http.Header
	var calls []nativeRouterCall
	nativeRouterRecorder(cloud, &calls, func(req *http.Request) *http.Response {
		seen = req.Header.Clone()
		// Caller callback storage cannot affect already captured options.
		retained.Headers["X-Action"] = "late option mutation"
		return nativeRouterWire(204, "")
	})
	value, err := routers.New(client).AddExtraRoutesRecord(context.Background(), "seed", input, owned, func(config *routers.RouterActionOpts) error {
		retained = config
		client.MoreHeaders["X-Selected"] = "after"
		input["router"] = nil
		return nil
	})
	if err != nil || value.StatusCode != 204 || len(calls) != 1 || strings.TrimSpace(calls[0].body) != `{"router":{"routes":[],"x_extension":false}}` || seen.Get("X-Selected") != "before" || seen.Get("X-Action") != "owned" {
		t.Fatal(value, err, calls, seen)
	}

	for _, lane := range []string{"nil context", "pre-canceled", "nil client", "wrong role", "invalid endpoint", "invalid base", "invalid ID", "nil option", "invalid header", "auth header", "version header", "option cause", "marshal cause", "marshal source", "option source"} {
		t.Run(lane, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := nativeRouterClient(cloud)
			api := routers.New(client)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var actualCtx context.Context = ctx
			cause := errors.New("callback interrupted")
			body, id := any(map[string]any{}), "seed"
			var options []routers.RouterActionOption
			switch lane {
			case "nil context":
				actualCtx = nil
			case "pre-canceled":
				cancel(cause)
			case "nil client":
				api = routers.New(nil)
			case "wrong role":
				client.Type = "compute"
			case "invalid endpoint":
				client.Endpoint += "?route=changed"
			case "invalid base":
				client.ResourceBase = "https://unrelated.example/v2.0/"
			case "invalid ID":
				id = ".."
			case "nil option":
				options = []routers.RouterActionOption{nil}
			case "invalid header":
				options = []routers.RouterActionOption{routers.WithRouterActionHeaders(map[string]string{"X-Action": "bad\nvalue"})}
			case "auth header":
				options = []routers.RouterActionOption{routers.WithRouterActionHeaders(map[string]string{"X-Auth-Token": "external"})}
			case "version header":
				options = []routers.RouterActionOption{routers.WithRouterActionHeaders(map[string]string{"OpenStack-API-Version": "network 2.99"})}
			case "option cause":
				options = []routers.RouterActionOption{func(*routers.RouterActionOpts) error { cancel(cause); return cause }}
			case "marshal cause":
				body = pythonRouterActionMarshaler(func() ([]byte, error) { cancel(cause); return nil, cause })
			case "marshal source":
				body = pythonRouterActionMarshaler(func() ([]byte, error) { client.Microversion = "changed"; return []byte(`{}`), nil })
			case "option source":
				options = []routers.RouterActionOption{func(*routers.RouterActionOpts) error {
					client.ResourceBase = cloud.Server.URL + "/changed/"
					return nil
				}}
			}
			calls := 0
			cloud.Provider.HTTPClient.Transport = nativeRouterTransport(func(*http.Request) (*http.Response, error) { calls++; return nativeRouterWire(200, `{}`), nil })
			value, err := api.AddExternalGatewaysRecord(actualCtx, id, body, options...)
			nativeRouterOperation(t, err, "AddExternalGatewaysRecord")
			if err == nil || value != nil || calls != 0 {
				t.Fatal(value, err, calls)
			}
			switch lane {
			case "wrong role":
				if !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal(err)
				}
			case "pre-canceled", "option cause", "marshal cause":
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
					t.Fatal(err)
				}
			default:
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPythonRouterActionsRetryAndReauthenticationKeepRequestOwnership(t *testing.T) {
	for _, lane := range []string{"clean retry", "body mutation", "accepted code mutation", "response decoder mutation", "source mutation", "custom retry cause", "reauthentication"} {
		t.Run(lane, func(t *testing.T) {
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
				if lane == "accepted code mutation" {
					return nativeRouterWire(400, `{"error":"original-policy"}`)
				}
				return nativeRouterWire(200, `{"router":{"id":"actual-id"}}`)
			})
			callbackCause := errors.New("retry refused")
			reauth := 0
			if lane == "reauthentication" {
				cloud.Provider.ReauthFunc = func(context.Context) error { reauth++; cloud.Provider.SetToken("refreshed"); return nil }
			} else {
				cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, original error, _ uint) error {
					switch lane {
					case "body mutation":
						opts.JSONBody = map[string]any{"changed": true}
					case "accepted code mutation":
						opts.OkCodes = append(opts.OkCodes, 400)
					case "response decoder mutation":
						opts.JSONResponse = &map[string]any{}
					case "source mutation":
						client.ResourceBase = cloud.Server.URL + "/changed/"
					case "custom retry cause":
						return callbackCause
					}
					return nil
				}
			}
			value, err := routers.New(client).UpdateExternalGatewaysRecord(context.Background(), "literal-id", json.RawMessage(`{"router":{"external_gateways":[{"network_id":"ext"}]}}`))
			if lane == "clean retry" || lane == "reauthentication" {
				if err != nil || value == nil || len(calls) != 2 || calls[0] != calls[1] {
					t.Fatal(value, err, calls)
				}
				pythonRouterActionValue(t, value.Resource, "id", `"actual-id"`)
				if lane == "reauthentication" && (!reflect.DeepEqual(tokens, []string{"test-token", "refreshed"}) || reauth != 1) {
					t.Fatal(tokens, reauth)
				}
			} else {
				nativeRouterOperation(t, err, "UpdateExternalGatewaysRecord")
				wantCalls := 1
				if lane == "accepted code mutation" {
					wantCalls = 2
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if err == nil || value != nil || len(calls) != wantCalls || !errors.As(err, &native) {
					t.Fatal(value, err, calls, native)
				}
				if lane == "custom retry cause" {
					if !errors.Is(err, callbackCause) {
						t.Fatal(err)
					}
				} else if lane != "accepted code mutation" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if lane == "accepted code mutation" && native.Actual != 400 {
					t.Fatal(native, err)
				}
			}
		})
	}
}

func TestPythonRouterActionsReusableOptionsOwnIndependentConcurrentRecords(t *testing.T) {
	cloud := testcloud.New(t)
	api := routers.New(nativeRouterClient(cloud))
	headers := map[string]string{"X-Action": "owned"}
	seed := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"x_cached": json.RawMessage(`{"owned":true}`)}}}
	option := routers.WithRouterActionOpts(routers.RouterActionOpts{Headers: headers, Seed: seed})
	headers["X-Action"] = "changed"
	seed.Body["x_cached"] = json.RawMessage(`null`)
	cloud.Provider.HTTPClient.Transport = nativeRouterTransport(func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("X-Action") != "owned" {
			return nil, errors.New("lost option snapshot")
		}
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		return nativeRouterWire(200, string(raw)), nil
	})
	const count = 8
	values := make([]*routers.RouterActionRecord, count)
	failures := make([]error, count)
	var workers sync.WaitGroup
	for index := range values {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			values[index], failures[index] = api.AddExtraRoutesRecord(context.Background(), fmt.Sprintf("router-%d", index), map[string]any{"router": map[string]any{"id": fmt.Sprintf("ack-%d", index), "x_extension": index}}, option)
		}(index)
	}
	workers.Wait()
	for index, value := range values {
		if failures[index] != nil {
			t.Fatal(failures[index])
		}
		pythonRouterActionValue(t, value.Resource, "id", fmt.Sprintf(`"ack-%d"`, index))
		pythonRouterActionValue(t, value.Resource, "x_extension", fmt.Sprint(index))
		value.Resource.Body["id"][1] = 'X'
		pythonRouterActionValue(t, value.Resource, "x_cached", `{"owned":true}`)
		value.Resource.Body["x_cached"][1] = 'X'
		pythonRouterActionValue(t, value.Wire, "id", fmt.Sprintf(`"ack-%d"`, index))
		if _, fabricated := value.Wire.Body["x_cached"]; fabricated {
			t.Fatal(value.Wire)
		}
	}
}

func TestPythonRouterActionsCachedSeedSurvivesAcknowledgementWithoutInventingWire(t *testing.T) {
	for _, action := range pythonRouterActionCalls {
		for _, body := range []string{"", `{"router":{"name":null,"routes":null,"revision":6}}`} {
			t.Run(action.operation+"/"+body, func(t *testing.T) {
				cached := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{
					"id": json.RawMessage(`"cached-other-id"`), "name": json.RawMessage(`"cached-name"`),
					"routes":         json.RawMessage(`[{"destination":"192.0.2.0/24","nexthop":"10.0.0.9"}]`),
					"admin_state_up": json.RawMessage(`false`), "revision_number": json.RawMessage(`0`),
					"tenant_id": json.RawMessage(`"tenant"`), "x_cached": json.RawMessage(`{"zero":0}`),
					"if_match": json.RawMessage(`["cached-header"]`),
				}, Header: http.Header{"X-Request-Id": {"cached-request"}}, StatusCode: 200}}
				option := routers.WithRouterActionSeed(cached)
				cached.Body["name"] = json.RawMessage(`"mutated caller"`)
				var calls []nativeRouterCall
				api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
					if req.Header.Get("If-Match") != "" {
						t.Fatal("cached Header was submitted", req.Header)
					}
					return nativeRouterWire(202, body)
				})
				value, err := action.call(api, context.Background(), "literal-id", map[string]any{}, option)
				if err != nil || len(calls) != 1 || strings.TrimSpace(calls[0].body) != `{}` || value.StatusCode != 202 || value.Header.Get("X-Request-Id") != "" {
					t.Fatal(value, err, calls)
				}
				pythonRouterActionValue(t, value.Resource, "id", `"literal-id"`)
				pythonRouterActionValue(t, value.Resource, "is_admin_state_up", `false`)
				pythonRouterActionValue(t, value.Resource, "project_id", `"tenant"`)
				pythonRouterActionValue(t, value.Resource, "x_cached", `{"zero":0}`)
				pythonRouterActionValue(t, value.Resource, "if_match", `["cached-header"]`)
				if body == "" {
					if value.Wire != nil {
						t.Fatal(value.Wire)
					}
					pythonRouterActionValue(t, value.Resource, "name", `"cached-name"`)
					pythonRouterActionValue(t, value.Resource, "revision_number", `0`)
				} else {
					pythonRouterActionValue(t, value.Resource, "name", `null`)
					pythonRouterActionValue(t, value.Resource, "routes", `null`)
					pythonRouterActionValue(t, value.Resource, "revision_number", `6`)
					if _, fabricated := value.Wire.Body["x_cached"]; fabricated {
						t.Fatal(value.Wire)
					}
				}
				value.Resource.Body["x_cached"][1] = 'X'
				pythonRouterActionValue(t, cached, "x_cached", `{"zero":0}`)
			})
		}
	}
	var calls []nativeRouterCall
	api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
		response := nativeRouterWire(204, "")
		response.Header.Set("If-Match", "actual-header")
		return response
	})
	cached := &resource.RawResource{Metadata: resource.Metadata{Header: http.Header{"If-Match": {"cached-wire"}}}}
	value, err := api.AddExternalGatewaysRecord(context.Background(), "literal-id", nil, routers.WithRouterActionSeed(cached))
	if err != nil {
		t.Fatal(err)
	}
	pythonRouterActionValue(t, value.Resource, "if_match", `["actual-header"]`)
	for _, fields := range []map[string]json.RawMessage{
		{"admin_state_up": json.RawMessage(`false`), "is_admin_state_up": json.RawMessage(`true`)},
		{"if_match": json.RawMessage(`null`), "if-match": json.RawMessage(`[]`)},
		{"base_path": json.RawMessage(`"/other"`)},
		{"x_extension": json.RawMessage(`{`)},
		{"revision_number": json.RawMessage(`"²"`)},
	} {
		before := len(calls)
		_, err := api.AddExtraRoutesRecord(context.Background(), "literal-id", nil, routers.WithRouterActionSeed(&resource.RawResource{Metadata: resource.Metadata{Body: fields}}))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
			t.Fatal(err, calls)
		}
	}
}
