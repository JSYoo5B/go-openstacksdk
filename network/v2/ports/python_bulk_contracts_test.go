package ports_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonBulkPortsSingleEagerFlatPostAndRawAliases(t *testing.T) {
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Method != http.MethodPost || r.URL.Path != "/neutron/v2.0/ports" || r.URL.RawQuery != "" || r.Header.Get("Content-Type") != "application/json" {
			t.Fatal(r.Method, r.URL, r.Header)
		}
		return nativePortWire(201, `{"ports":[{"id":"remote"}]}`)
	})
	data := []ports.BulkPortInput{
		ports.BulkPortJSON(json.RawMessage(`{"binding_profile":{"first":1},"binding:profile":{"last":2},"is_admin_state_up":"false","security_group_ids":"raw","microversion":"99","base_path":"literal","resource_type":false,"__conflicting_attrs":null,"location":{"raw":1}}`)),
		ports.BulkPort(ports.WithBulkPortAttribute("network_id", "n"), ports.WithBulkPortAttribute("name", "two")),
	}
	result, err := api.CreatePorts(context.Background(), data)
	if err != nil || result == nil || result.StatusCode != 201 || len(calls) != 1 {
		t.Fatal(result, err, calls)
	}
	var body struct {
		Ports []map[string]json.RawMessage `json:"ports"`
	}
	if err := json.Unmarshal([]byte(calls[0].body), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Ports) != 2 {
		t.Fatal(body)
	}
	for key, want := range map[string]string{"binding:profile": `{"last":2}`, "admin_state_up": `"false"`, "security_groups": `"raw"`, "base_path": `"literal"`, "resource_type": "false", "__conflicting_attrs": "null", "location": `{"raw":1}`} {
		if string(body.Ports[0][key]) != want {
			t.Fatal(key, string(body.Ports[0][key]), want)
		}
	}
	for _, key := range []string{"port", "binding_profile", "is_admin_state_up", "security_group_ids", "microversion"} {
		if _, bad := body.Ports[0][key]; bad {
			t.Fatal(key, body)
		}
	}
	rows, err := result.All()
	if err != nil || len(rows) != 1 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	pythonFindField(t, rows[0], "id", `"remote"`)
	pythonFindField(t, rows[0], "network_id", "null")
	pythonFindField(t, rows[0], "name", "null")
}

func TestPythonBulkPortsLastRowHeaderAndSelectedVersion(t *testing.T) {
	for _, test := range []struct{ last, want string }{
		{`{}`, "configured"}, {`{"if_match":["a","b"]}`, "a, b"}, {`{"IF-MATCH":"wire"}`, "wire"}, {`{"if_match":null}`, "None"}, {`{"if_match":false}`, "False"}, {`{"if_match":0}`, "0"}, {`{"if_match":[]}`, ""},
	} {
		t.Run(test.last, func(t *testing.T) {
			var calls []nativePortCall
			api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
				if r.Header.Get("If-Match") != test.want || r.Header.Get("OpenStack-API-Version") != "network 2.1" {
					t.Fatal(r.Header, test.want)
				}
				return nativePortWire(201, `{"ports":[]}`)
			})
			api.RawClient().MoreHeaders = map[string]string{"If-Match": "configured"}
			api.RawClient().Microversion = "9.9"
			result, err := api.CreatePorts(context.Background(), []ports.BulkPortInput{ports.BulkPortJSON(json.RawMessage(`{"if_match":"earlier"}`)), ports.BulkPortJSON(json.RawMessage(test.last))}, ports.WithBulkPortMicroversion("2.1"))
			if err != nil || result == nil || len(calls) != 1 || api.RawClient().Microversion != "9.9" {
				t.Fatal(result, err, calls)
			}
			var body map[string]json.RawMessage
			if err := json.Unmarshal([]byte(calls[0].body), &body); err != nil || bytes.Contains(body["ports"], []byte("if_match")) || bytes.Contains(body["ports"], []byte("IF-MATCH")) {
				t.Fatal(string(calls[0].body), err)
			}
		})
	}
}

func TestPythonBulkPortsPreflightRejectsEveryInputBeforePost(t *testing.T) {
	cases := [][]ports.BulkPortInput{nil, {}, {ports.BulkPortJSON(json.RawMessage("null"))}, {ports.BulkPortJSON(json.RawMessage("[]"))}, {ports.BulkPortJSON(json.RawMessage("["))}, {ports.BulkPort(nil)}}
	for _, key := range []string{"cls", "self", "connection", "_synchronized"} {
		cases = append(cases, []ports.BulkPortInput{ports.BulkPort(ports.WithBulkPortAttribute(key, nil))})
	}
	for _, raw := range []string{`{"if_match":[null]}`, `{"if_match":[1]}`, `{"if_match":"bad\nheader"}`} {
		cases = append(cases, []ports.BulkPortInput{ports.BulkPortJSON(json.RawMessage(raw)), ports.BulkPort()})
	}
	for i, data := range cases {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("preflight HTTP", i); return nil })
		result, err := api.CreatePorts(context.Background(), data)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(i, result, err, calls)
		}
		pythonPortListOperation(t, err, "CreatePorts")
	}
}

func TestPythonBulkPortsInputsAndOptionCallbacksAreOwned(t *testing.T) {
	fields := map[string]json.RawMessage{"network_id": json.RawMessage(`"n"`), "x_extension": json.RawMessage(`{"keep":[1]}`)}
	input := ports.BulkPortFields(fields)
	fields["network_id"][1] = 'x'
	headers := map[string]string{"X-Trace": "owned"}
	option := ports.WithBulkPortHeaders(headers)
	headers["X-Trace"] = "mutated"
	var escaped *ports.BulkPortOpts
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Trace") != "owned" || r.Header.Get("X-Callback") != "yes" {
			t.Fatal(r.Header)
		}
		escaped.Headers["X-Trace"] = "escaped"
		return nativePortWire(201, `{"ports":[{}]}`)
	})
	result, err := api.CreatePorts(context.Background(), []ports.BulkPortInput{input}, ports.WithBulkPortHeaders(nil), option, func(c *ports.BulkPortOpts) error { c.Headers["X-Callback"] = "yes"; escaped = c; return nil })
	if err != nil || result == nil || len(calls) != 1 {
		t.Fatal(result, err, calls)
	}
	var body map[string][]map[string]json.RawMessage
	if err := json.Unmarshal([]byte(calls[0].body), &body); err != nil || string(body["ports"][0]["network_id"]) != `"n"` {
		t.Fatal(body, err)
	}
	result.Envelope[0] = 'x'
	result.Header.Set("X-Response", "mutated")
	rows, err := result.All()
	if err != nil || len(rows) != 1 || string(rows[0].Envelope) != `{"ports":[{}]}` || rows[0].Header.Get("X-Response") != "" {
		t.Fatal(rows, err)
	}
}

func TestPythonBulkPortsRemoteViewsHaveDefaultsAndNoHttpHeaderSeed(t *testing.T) {
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
		r := nativePortWire(203, `{"ports":[{"id":"one","binding:profile":{"keep":1},"revision_number":"7","admin_state_up":"false","security_groups":"ab","x_remote":{"null":null}},{"id":"two","if_match":"row"}]}`)
		r.Header.Set("If-Match", "server")
		r.Header.Set("X-Request-Id", "bulk-view")
		return r
	})
	result, err := api.CreatePorts(context.Background(), []ports.BulkPortInput{ports.BulkPort(ports.WithBulkPortAttribute("network_id", "input-only"))})
	if err != nil {
		t.Fatal(result, err)
	}
	rows, err := result.All()
	if err != nil || len(rows) != 2 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	for key, want := range map[string]string{"binding_profile": `{"keep":1}`, "revision_number": "7", "is_admin_state_up": "true", "security_group_ids": `["ab"]`, "network_id": "null", "name": "null", "tags": "[]", "is_port_security_enabled": "false", "location": "null", "if_match": "null", "x_remote": `{"null":null}`} {
		pythonFindField(t, rows[0], key, want)
	}
	pythonFindWireField(t, rows[0], "revision_number", `"7"`)
	pythonFindWireField(t, rows[0], "security_groups", `"ab"`)
	pythonFindField(t, rows[1], "if_match", `["row"]`)
	if rows[0].StatusCode != 203 || rows[0].Header.Get("If-Match") != "server" || rows[0].Header.Get("X-Request-Id") != "bulk-view" {
		t.Fatal(rows[0])
	}
}

func TestPythonBulkPortsEagerEnvelopeAndIterableFailuresKeepActualPost(t *testing.T) {
	for _, test := range []struct {
		status int
		body   string
	}{{204, ""}, {201, "not json"}, {201, "{}"}, {201, `{"ports":null}`}, {201, `{"ports":true}`}, {201, `{"ports":1}`}, {201, "null"}} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativePortWire(test.status, test.body)
			r.Header.Set("X-Request-Id", "bulk-envelope")
			return r
		})
		result, err := api.CreatePorts(context.Background(), []ports.BulkPortInput{ports.BulkPort()})
		var proof *resource.ResponseError
		if result == nil || result.Records != nil || !errors.As(err, &proof) || proof.StatusCode != test.status || string(proof.Body) != test.body || proof.Header.Get("X-Request-Id") != "bulk-envelope" || len(calls) != 1 {
			t.Fatal(test, result, err, proof, calls)
		}
	}
	for _, body := range []string{`{"ports":[]}`, `{"ports":{}}`, `{"ports":""}`} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(201, body) })
		result, err := api.CreatePorts(context.Background(), []ports.BulkPortInput{{}})
		if err != nil || result == nil {
			t.Fatal(body, result, err)
		}
		rows, err := result.All()
		if err != nil || rows == nil || len(rows) != 0 || len(calls) != 1 {
			t.Fatal(body, rows, err, calls)
		}
	}
}

func TestPythonBulkPortsResponseRowsFailLazilyAfterEarlierActualRows(t *testing.T) {
	invalid := []string{"null", "42", `"x"`, `[]`}
	for _, key := range []string{"cls", "self", "connection", "_synchronized", "microversion"} {
		invalid = append(invalid, `{"`+key+`":null}`)
	}
	for _, bad := range invalid {
		payload := `{"ports":[{"id":"valid"},` + bad + `]}`
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativePortWire(201, payload)
			r.Header.Set("X-Request-Id", "lazy-bulk")
			return r
		})
		result, err := api.CreatePorts(context.Background(), []ports.BulkPortInput{ports.BulkPort()})
		if err != nil || result == nil || len(calls) != 1 {
			t.Fatal(bad, result, err, calls)
		}
		valid, fail := 0, 0
		for record, err := range result.Records {
			if err == nil {
				if record == nil {
					t.Fatal("nil successful record")
				}
				valid++
				continue
			}
			var proof *resource.ResponseError
			if record != nil || !errors.As(err, &proof) || string(proof.Body) != payload || proof.StatusCode != 201 || proof.Header.Get("X-Request-Id") != "lazy-bulk" {
				t.Fatal(bad, record, err, proof)
			}
			fail++
		}
		if valid != 1 || fail != 1 || len(calls) != 1 {
			t.Fatal(bad, valid, fail, calls)
		}
	}
	for _, body := range []string{`{"ports":{"key":{}}}`, `{"ports":"한"}`} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(201, body) })
		result, err := api.CreatePorts(context.Background(), []ports.BulkPortInput{ports.BulkPort()})
		if err != nil || result == nil {
			t.Fatal(body, result, err)
		}
		rows, err := result.All()
		if rows != nil || err == nil || len(calls) != 1 {
			t.Fatal(body, rows, err, calls)
		}
	}
}

func TestPythonBulkPortsCancellationAndSourceMutationRetainCauses(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("callback changed captured source"); return nil })
		ctx, cancel := context.WithCancelCause(context.Background())
		cause, optionCause := errors.New("cancelled bulk"), errors.New("failed bulk option")
		result, err := api.CreatePorts(ctx, []ports.BulkPortInput{ports.BulkPort()}, func(*ports.BulkPortOpts) error {
			if cancelled {
				cancel(cause)
			} else {
				api.RawClient().ResourceBase += "changed/"
			}
			return optionCause
		})
		cancel(nil)
		if result != nil || len(calls) != 0 || !errors.Is(err, optionCause) || cancelled && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) || !cancelled && !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(result, err, calls)
		}
	}
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(201, `{"ports":[{"id":"actual"}]}`) })
	result, err := api.CreatePorts(context.Background(), []ports.BulkPortInput{ports.BulkPort()})
	if err != nil {
		t.Fatal(result, err)
	}
	api.RawClient().ResourceBase += "changed/"
	rows, err := result.All()
	var proof *resource.ResponseError
	if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 201 || len(calls) != 1 {
		t.Fatal(rows, err, proof, calls)
	}
}

func TestPythonBulkPortsPhysicalFailuresPreserveNativeEvidenceAndStopCallbacks(t *testing.T) {
	for _, status := range []int{201, 203, 400, 401, 429, 503} {
		for _, fault := range []string{"read", "close", "mixed"} {
			t.Run(fmt.Sprint(status, fault), func(t *testing.T) {
				readCause, closeCause, hookCause := errors.New("bulk read"), errors.New("bulk close"), errors.New("hook must not run")
				raw := `{"ports":[{"id":"actual"}]}`
				if status >= 400 {
					raw = `{"NeutronError":{"message":"actual rejection"}}`
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
					r.Header.Set("X-Request-Id", "actual-bulk-fault")
					return r
				})
				retries, reauth := 0, 0
				api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return hookCause
				}
				api.RawClient().ReauthFunc = func(context.Context) error { reauth++; return nil }
				result, err := api.CreatePorts(context.Background(), []ports.BulkPortInput{ports.BulkPort()})
				if err == nil || result != nil && result.Records != nil || len(calls) != 1 || closed != 1 || retries != 0 || reauth != 0 || errors.Is(err, hookCause) {
					t.Fatal(result, err, calls, closed, retries, reauth)
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
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != raw || proof.Header.Get("X-Request-Id") != "actual-bulk-fault" {
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
				if !errors.As(original, &proof) || proof.Actual != status || proof.Method != http.MethodPost || proof.URL != api.RawClient().ServiceURL("ports") || string(proof.Body) != raw || proof.ResponseHeader.Get("X-Request-Id") != "actual-bulk-fault" {
					t.Fatal(err, proof)
				}
			})
		}
	}
}

func TestPythonBulkPortsCleanReauthenticationAndRetryKeepOneBulkBodyPolicy(t *testing.T) {
	for _, status := range []int{401, 503} {
		var calls []nativePortCall
		var tokens []string
		api := pythonCorePortAPI(t, &calls, func(r *http.Request) *http.Response {
			tokens = append(tokens, r.Header.Get("X-Auth-Token"))
			if r.Header.Get("If-Match") != "last" || r.Header.Get("OpenStack-API-Version") != "network 2.1" || r.URL.RawQuery != "" || r.Method != http.MethodPost {
				t.Fatal(r.Header, r.URL, r.Method)
			}
			if len(calls) == 1 {
				return nativePortWire(status, `{"NeutronError":{"message":"retry"}}`)
			}
			return nativePortWire(201, `{"ports":[{"id":"actual"}]}`)
		})
		retries, reauth := 0, 0
		api.RawClient().ReauthFunc = func(context.Context) error { reauth++; api.RawClient().SetToken("renewed"); return nil }
		api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, c *gophercloud.RequestOpts, _ error, _ uint) error {
			retries++
			c.OkCodes = append(c.OkCodes, 503)
			c.MoreHeaders["X-Retry-Probe"] = "ordinary"
			return nil
		}
		result, err := api.CreatePorts(context.Background(), []ports.BulkPortInput{ports.BulkPort(ports.WithBulkPortAttribute("network_id", "n"), ports.WithBulkPortAttribute("if_match", "last"))}, ports.WithBulkPortMicroversion("2.1"))
		if err != nil || result == nil {
			t.Fatal(result, err, calls)
		}
		rows, err := result.All()
		if err != nil || len(rows) != 1 || len(calls) != 2 || calls[0].body != calls[1].body || status == 401 && (reauth != 1 || retries != 0 || !reflect.DeepEqual(tokens, []string{"test-token", "renewed"})) || status == 503 && (retries != 1 || reauth != 0) {
			t.Fatal(rows, err, calls, tokens, retries, reauth)
		}
	}
}
