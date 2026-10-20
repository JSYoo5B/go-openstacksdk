package ports_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonFindField(t *testing.T, value *ports.PortRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatalf("%s: want %s, record=%#v", key, want, value)
	}
}
func pythonFindWireField(t *testing.T, value *ports.PortRecord, key, want string) {
	t.Helper()
	if value == nil || value.Wire == nil || string(value.Wire.Body[key]) != want {
		t.Fatalf("wire %s: want %s, record=%#v", key, want, value)
	}
}

// Source Resource.find first attempts the literal member route with raw kwargs;
// accepted incomplete responses retain constructor state and descriptor defaults.
func TestPythonPortFindRecordMemberFirstSeedAndResponseProjection(t *testing.T) {
	for _, testcase := range []struct {
		name, body string
		checks     map[string]string
	}{
		{"omitted Body defaults", `{"port":{}}`, map[string]string{"id": `"target"`, "name": "null", "tags": "[]", "revision_number": "null", "is_port_security_enabled": "false", "project_id": "null", "tenant_id": "null", "location": "null", "if_match": `["revision_number=7"]`, "x_seed": `{"keep":null}`}},
		{"explicit null overrides seeded identity", `{"port":{"id":null,"name":null,"tags":null,"revision_number":null,"port_security_enabled":null,"project_id":null,"tenant_id":"legacy","x_extension":9007199254740993,"self":"drop"}}`, map[string]string{"id": "null", "name": "null", "tags": "null", "revision_number": "null", "is_port_security_enabled": "null", "project_id": "null", "tenant_id": `"legacy"`, "x_extension": "9007199254740993"}},
		{"descriptor coercion and ordered wire aliases", `{"port":{"id":"wire-id","name":"wire-name","revision_number":"٩٠","tags":"red","admin_state_up":[],"binding_profile":{"old":1},"binding:profile":{"new":2},"resource_request":4,"security_groups":"sg","tenant_id":"legacy","x_extension":{"big":9007199254740993}}}`, map[string]string{"id": `"wire-id"`, "revision_number": "90", "tags": `["red"]`, "is_admin_state_up": "false", "binding_profile": `{"new":2}`, "resource_request": "{}", "security_group_ids": `["sg"]`, "project_id": `"legacy"`, "tenant_id": `"legacy"`, "x_extension": `{"big":9007199254740993}`}},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativePortCall
			api := pythonCorePortAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Query().Get("x_seed") != "keep" || req.URL.Query().Get("status") != "ACTIVE" {
					t.Fatal(req.URL)
				}
				response := nativePortWire(200, testcase.body)
				response.Header.Set("If-Match", "revision_number=7")
				response.Header.Set("X-Request-Id", "member")
				return response
			})
			value, err := api.FindRecord(context.Background(), "target", ports.WithRecordFindQuery("x_seed", map[string]any{"keep": nil}), ports.WithRecordFindQuery("status", "ACTIVE"))
			if err != nil || len(calls) != 1 || calls[0].path != "/neutron/v2.0/ports/target" || calls[0].method != http.MethodGet {
				t.Fatal(value, err, calls)
			}
			for key, want := range testcase.checks {
				pythonFindField(t, value, key, want)
			}
			if string(value.Envelope) != testcase.body || value.StatusCode != 200 || value.Header.Get("X-Request-Id") != "member" {
				t.Fatal(value)
			}
			if value.Wire != nil {
				value.Resource.Body["id"] = json.RawMessage(`"owned"`)
				value.Resource.Header.Set("X-Request-Id", "changed")
				if value.Wire.Header.Get("X-Request-Id") != "member" || value.Header.Get("X-Request-Id") != "member" {
					t.Fatal("aliased evidence")
				}
			}
		})
	}
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(204, "") })
	value, err := api.FindRecord(context.Background(), "literal/id %2F", ports.WithRecordFindQuery("name", "seed"))
	if err != nil || len(calls) != 1 || value.Wire != nil || value.StatusCode != 204 {
		t.Fatal(value, err, calls)
	}
	pythonFindField(t, value, "id", `"literal/id %2F"`)
	pythonFindField(t, value, "name", `"seed"`)
}

// QueryParameters declaration order, explicit null/false, repeated fields/tags,
// nested local equality and unknown-parameter discard are Source contracts.
func TestPythonPortFindRecordFallbackAliasesAndLocalBodyFilters(t *testing.T) {
	var calls []nativePortCall
	var listed url.Values
	api := pythonCorePortAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path == "/neutron/v2.0/ports/target" {
			q := req.URL.Query()
			if q.Get("x_unknown") != "keep" || q.Has("name") || q.Has("all_projects") || q.Get("paginated") != "False" {
				t.Fatal(q)
			}
			return nativePortWire(403, `{"error":"member hidden"}`)
		}
		listed = req.URL.Query()
		if req.Header.Get("X-Trace") != "owned" {
			t.Fatal(req.Header)
		}
		return nativePortWire(200, `{"ports":[{"id":"filtered","name":"target","binding:profile":{"nested":{"enabled":false}}},{"id":"match","name":"target","binding:profile":{"nested":{"enabled":1},"extra":"keep"},"x_extension":null,"self":"drop"}]}`)
	})
	configured := map[string]string{"X-Trace": "old"}
	api.RawClient().MoreHeaders = configured
	value, err := api.FindRecord(context.Background(), "target",
		ports.WithRecordFindQuery("x_unknown", "keep"), ports.WithRecordFindQuery("all_projects", true), ports.WithRecordFindQuery("fields", []string{"id", "name"}),
		ports.WithRecordFindQuery("any_tags", []string{"red", "blue"}), ports.WithRecordFindQuery("not_any_tags", "black"), ports.WithRecordFindQuery("paginated", false),
		ports.WithRecordFindHeaders(map[string]string{"X-Trace": "owned"}), ports.WithRecordFindQuery("project_id", "current"), ports.WithRecordFindQuery("tenant_id", "legacy"), ports.WithRecordFindQuery("security_groups", []string{"wire"}), ports.WithRecordFindQuery("security_group_ids", []string{"sg"}), ports.WithRecordFindQuery("binding_profile", map[string]any{"nested": map[string]any{"enabled": true}}))
	if err != nil || value == nil || len(calls) != 2 {
		t.Fatal(value, err, calls)
	}
	pythonFindField(t, value, "id", `"match"`)
	pythonFindField(t, value, "x_extension", "null")
	if _, exists := value.Resource.Body["self"]; exists {
		t.Fatal(value.Resource.Body)
	}
	pythonFindWireField(t, value, "self", `"drop"`)
	want := url.Values{"name": {"target"}, "fields": {"id", "name"}, "tags-any": {"red", "blue"}, "not-tags-any": {"black"}, "project_id": {"legacy"}, "security_groups": {"sg"}}
	if !reflect.DeepEqual(listed, want) || configured["X-Trace"] != "old" {
		t.Fatal(listed, want, configured)
	}
}

func TestPythonPortFindRecordNamePresenceMissingAndPassiveResponseID(t *testing.T) {
	for _, testcase := range []struct {
		name     string
		supplied bool
		value    any
		want     url.Values
	}{
		{"omitted injects name", false, nil, url.Values{"name": {"target"}}},
		{"null disables name hint", true, nil, url.Values{}},
		{"empty name stays explicit", true, "", url.Values{"name": {""}}},
	} {
		for _, status := range []int{400, 403, 404} {
			t.Run(testcase.name+http.StatusText(status), func(t *testing.T) {
				var calls []nativePortCall
				api := pythonCorePortAPI(t, &calls, func(req *http.Request) *http.Response {
					if len(calls) == 1 {
						return nativePortWire(status, `{"error":"missing"}`)
					}
					if !reflect.DeepEqual(req.URL.Query(), testcase.want) {
						t.Fatal(req.URL.Query(), testcase.want)
					}
					return nativePortWire(200, `{"ports":[]}`)
				})
				options := []ports.RecordFindOption{}
				if testcase.supplied {
					options = append(options, ports.WithRecordFindQuery("name", testcase.value))
				}
				got, err := api.FindRecord(context.Background(), "target", options...)
				if err != nil || got != nil || len(calls) != 2 {
					t.Fatal(got, err, calls)
				}
			})
		}
	}
	for _, body := range []string{`{"ports":[{"name":"target"}]}`, `{"ports":{"id":17,"name":"target"}}`} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
			if len(calls) == 1 {
				return nativePortWire(404, "")
			}
			return nativePortWire(200, body)
		})
		got, err := api.FindRecord(context.Background(), "target")
		if err != nil || got == nil || len(calls) != 2 {
			t.Fatal(got, err, calls)
		}
	}
	for _, status := range []int{200, 404} {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
			if len(calls) == 1 {
				return nativePortWire(404, "")
			}
			body := `{"ports":[]}`
			if status != 200 {
				body = `{"error":"list denied"}`
			}
			return nativePortWire(status, body)
		})
		got, err := api.FindRecord(context.Background(), "target", ports.WithRecordFindIgnoreMissing(false))
		if got != nil || err == nil || (status == 200 && !errors.Is(err, resource.ErrNotFound)) || len(calls) != 2 {
			t.Fatal(got, err, calls)
		}
		if status == 404 {
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Body == nil {
				t.Fatal(err)
			}
		}
	}
}

func TestPythonPortFindRecordLateDuplicateErrorMarkerAndRawCap(t *testing.T) {
	for _, testcase := range []struct {
		name      string
		second    int
		duplicate bool
	}{
		{"late duplicate", 200, true}, {"late page denied", 500, false},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativePortCall
			api := pythonCorePortAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) == 1 {
					return nativePortWire(404, "")
				}
				if req.URL.Query().Get("marker") == "late" {
					if testcase.duplicate {
						return nativePortWire(200, `{"ports":[{"id":"second","name":"target"}]}`)
					}
					return nativePortWire(testcase.second, `{"error":"late"}`)
				}
				return nativePortWire(200, `{"ports":[{"id":"first","name":"target"}],"ports_links":[{"rel":"next","href":"?marker=late"}]}`)
			})
			got, err := api.FindRecord(context.Background(), "target")
			if got != nil || err == nil || len(calls) != 3 {
				t.Fatal(got, err, calls)
			}
			if testcase.duplicate {
				if !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 500 || string(native.Body) != `{"error":"late"}` {
					t.Fatal(err)
				}
			}
		})
	}
	t.Run("short page derives marker and keeps filters", func(t *testing.T) {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(req *http.Request) *http.Response {
			if len(calls) == 1 {
				return nativePortWire(404, "")
			}
			q := req.URL.Query()
			if q.Get("limit") != "2" || q.Get("name") != "target" || !reflect.DeepEqual(q["tags"], []string{"a", "b"}) {
				t.Fatal(q)
			}
			if q.Get("marker") == "7" {
				return nativePortWire(200, `{"ports":[]}`)
			}
			return nativePortWire(200, `{"ports":[{"id":7,"name":"target"}]}`)
		})
		got, err := api.FindRecord(context.Background(), "target", ports.WithRecordFindQuery("limit", 2), ports.WithRecordFindQuery("tags", []string{"a", "b"}))
		if err != nil || got == nil || len(calls) != 3 {
			t.Fatal(got, err, calls)
		}
		pythonFindField(t, got, "id", "7")
	})
	t.Run("raw cap occurs before local filtering", func(t *testing.T) {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(req *http.Request) *http.Response {
			if len(calls) == 1 {
				return nativePortWire(404, "")
			}
			if req.URL.Query().Get("limit") != "1" {
				t.Fatal(req.URL)
			}
			return nativePortWire(200, `{"ports":[{"id":"filtered","name":"target","binding:profile":{"nested":{"enabled":false}}},{"id":"match","name":"target","binding:profile":{"nested":{"enabled":1},"extra":"keep"}}],"ports_links":[{"rel":"next","href":"?marker=late"}]}`)
		})
		got, err := api.FindRecord(context.Background(), "target", ports.WithRecordFindQuery("max_items", 1), ports.WithRecordFindQuery("binding_profile", map[string]any{"nested": map[string]any{"enabled": true}}))
		if err != nil || got != nil || len(calls) != 2 {
			t.Fatal(got, err, calls)
		}
	})
}

func TestPythonPortFindRecordOptionSnapshotsAndPreflightGuards(t *testing.T) {
	fields := []string{"id", "name"}
	query := map[string]json.RawMessage{"fields": json.RawMessage(`["id","name"]`), "x_seed": json.RawMessage(`{"initial":true}`)}
	headers := map[string]string{"X-Trace": "original"}
	ignore := false
	options := ports.WithRecordFindOpts(ports.RecordFindOpts{Query: query, Headers: headers, IgnoreMissing: &ignore})
	query["fields"][2] = 'x'
	delete(query, "x_seed")
	headers["X-Trace"] = "changed"
	ignore = true
	fieldOption := ports.WithRecordFindQuery("fields", fields)
	fields[0] = "changed"
	var escaped *ports.RecordFindOpts
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Header.Get("X-Trace") != "original" || !reflect.DeepEqual(req.URL.Query()["fields"], []string{"id", "name"}) || req.URL.Query().Get("x_seed") != "initial" {
			t.Fatal(req.Header, req.URL)
		}
		escaped.Query["fields"][2] = 'z'
		escaped.Headers["X-Trace"] = "callback changed"
		return nativePortWire(200, `{"port":{}}`)
	})
	got, err := api.FindRecord(context.Background(), "target", options, fieldOption, func(c *ports.RecordFindOpts) error { escaped = c; return nil })
	if err != nil || got == nil || len(calls) != 1 {
		t.Fatal(got, err, calls)
	}
	pythonFindField(t, got, "fields", `["id","name"]`)
	for _, testcase := range []struct {
		name    string
		options []ports.RecordFindOption
		cause   error
	}{
		{"nil", []ports.RecordFindOption{nil}, resource.ErrInvalidOption},
		{"malformed JSON", []ports.RecordFindOption{ports.WithRecordFindOpts(ports.RecordFindOpts{Query: map[string]json.RawMessage{"fields": json.RawMessage(`[`)}})}, resource.ErrInvalidOption},
		{"negative cap", []ports.RecordFindOption{ports.WithRecordFindQuery("max_items", -1)}, resource.ErrInvalidOption},
		{"bad page flag", []ports.RecordFindOption{ports.WithRecordFindOpts(ports.RecordFindOpts{Query: map[string]json.RawMessage{"paginated": json.RawMessage(`bad flag`)}})}, resource.ErrInvalidOption},
		{"identity keyword collision", []ports.RecordFindOption{ports.WithRecordFindQuery("id", "different")}, resource.ErrInvalidOption},
		{"route override", []ports.RecordFindOption{ports.WithRecordFindQuery("base_path", "/other")}, resource.ErrInvalidOption},
		{"fetch requires_id control", []ports.RecordFindOption{ports.WithRecordFindQuery("requires_id", false)}, resource.ErrInvalidOption},
		{"fetch error_message control", []ports.RecordFindOption{ports.WithRecordFindQuery("error_message", "changed")}, resource.ErrInvalidOption},
		{"fetch cache control", []ports.RecordFindOption{ports.WithRecordFindQuery("skip_cache", true)}, resource.ErrInvalidOption},
		{"fetch response envelope control", []ports.RecordFindOption{ports.WithRecordFindQuery("resource_response_key", "other")}, resource.ErrInvalidOption},
		{"protected Header", []ports.RecordFindOption{ports.WithRecordFindHeaders(map[string]string{"X-Auth-Token": "different"})}, resource.ErrInvalidOption},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativePortCall
			api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("unexpected HTTP"); return nil })
			got, err := api.FindRecord(context.Background(), "target", testcase.options...)
			if got != nil || !errors.Is(err, testcase.cause) || len(calls) != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
	t.Run("source mutation is sticky", func(t *testing.T) {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("unexpected HTTP"); return nil })
		got, err := api.FindRecord(context.Background(), "target", func(*ports.RecordFindOpts) error { api.RawClient().ResourceBase += "changed/"; return nil })
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(got, err, calls)
		}
	})
	t.Run("canceled cause prevents HTTP", func(t *testing.T) {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("unexpected HTTP"); return nil })
		cause := errors.New("caller canceled")
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(cause)
		got, err := api.FindRecord(ctx, "target")
		if got != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || len(calls) != 0 {
			t.Fatal(got, err, calls)
		}
	})
}

type pythonFindFaultBody struct {
	data        string
	read, close error
	used        bool
	onClose     func()
}

func (b *pythonFindFaultBody) Read(p []byte) (int, error) {
	if b.used {
		return 0, io.EOF
	}
	b.used = true
	return copy(p, b.data), b.read
}
func (b *pythonFindFaultBody) Close() error {
	if b.onClose != nil {
		b.onClose()
	}
	return b.close
}

func TestPythonPortFindRecordPhysicalFailuresPreserveActualEvidence(t *testing.T) {
	for _, testcase := range []struct {
		name   string
		status int
		list   bool
		read   bool
	}{
		{"rejected member 404", 404, false, true}, {"rejected member 500", 500, false, true}, {"rejected late list", 403, true, true}, {"accepted member", 200, false, true}, {"accepted close", 200, false, false},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			readCause, closeCause := errors.New("physical read"), errors.New("physical close")
			var calls []nativePortCall
			payload := `{"error":"physical"}`
			if testcase.status == 200 {
				payload = `{"port":{"id":"target"}}`
			}
			api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
				if testcase.list && len(calls) == 1 {
					return nativePortWire(404, "")
				}
				response := nativePortWire(testcase.status, payload)
				read := error(nil)
				if testcase.read {
					read = readCause
				}
				response.Body = &pythonFindFaultBody{data: payload, read: read, close: closeCause}
				response.Header.Set("X-Request-Id", "physical")
				return response
			})
			got, err := api.FindRecord(context.Background(), "target")
			wantCalls := 1
			if testcase.list {
				wantCalls = 2
			}
			if got != nil || !errors.Is(err, closeCause) || (testcase.read && !errors.Is(err, readCause)) || len(calls) != wantCalls {
				t.Fatal(got, err, calls)
			}
			if testcase.status >= 400 {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != testcase.status || string(native.Body) != payload || native.ResponseHeader.Get("X-Request-Id") != "physical" || !strings.Contains(native.URL, "/neutron/v2.0/ports") {
					t.Fatal(err, native)
				}
			} else {
				var response *resource.ResponseError
				if !errors.As(err, &response) || response.StatusCode != 200 || string(response.Body) != payload || response.Header.Get("X-Request-Id") != "physical" {
					t.Fatal(err, response)
				}
			}
		})
	}
	t.Run("accepted late close cancellation retains original cause", func(t *testing.T) {
		var calls []nativePortCall
		cause := errors.New("late cancellation")
		ctx, cancel := context.WithCancelCause(context.Background())
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
			response := nativePortWire(200, `{"port":{}}`)
			response.Body = &pythonFindFaultBody{data: `{"port":{}}`, onClose: func() { cancel(cause) }}
			return response
		})
		got, err := api.FindRecord(ctx, "target")
		if got != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || len(calls) != 1 {
			t.Fatal(got, err, calls)
		}
	})
}

func TestPythonPortFindRecordRetryCallbacksKeepCleanFallbackAndGuards(t *testing.T) {
	for _, fault := range []bool{false, true} {
		t.Run(map[bool]string{false: "clean callback permits fallback", true: "physical rejection blocks fallback"}[fault], func(t *testing.T) {
			var calls []nativePortCall
			readCause := errors.New("rejected read")
			api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					response := nativePortWire(404, `{"error":"missing"}`)
					if fault {
						response.Body = &pythonFindFaultBody{data: `{"error":"missing"}`, read: readCause}
					}
					return response
				}
				return nativePortWire(200, `{"ports":[]}`)
			})
			callbacks := 0
			api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				callbacks++
				return original
			}
			got, err := api.FindRecord(context.Background(), "target")
			if callbacks != 1 || got != nil {
				t.Fatal(got, err, calls, callbacks)
			}
			if fault {
				if !errors.Is(err, readCause) || len(calls) != 1 {
					t.Fatal(err, calls)
				}
			} else if err != nil || len(calls) != 2 {
				t.Fatal(err, calls)
			}
		})
	}
	t.Run("retry cannot admit rejected status", func(t *testing.T) {
		var calls []nativePortCall
		api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
			response := nativePortWire(404, `{"error":"actual rejected retry"}`)
			response.Header.Set("X-Request-Id", "status-policy")
			return response
		})
		callbacks := 0
		api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, options *gophercloud.RequestOpts, original error, _ uint) error {
			callbacks++
			options.OkCodes = append(options.OkCodes, 404)
			return nil
		}
		got, err := api.FindRecord(context.Background(), "target")
		var native gophercloud.ErrUnexpectedResponseCode
		if got != nil || err == nil || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != `{"error":"actual rejected retry"}` || native.ResponseHeader.Get("X-Request-Id") != "status-policy" || callbacks != 1 || len(calls) != 2 {
			t.Fatal(got, err, native, calls, callbacks)
		}
		for _, call := range calls {
			if call.path != "/neutron/v2.0/ports/target" {
				t.Fatal("rejected retry triggered LIST", calls)
			}
		}
		for _, code := range native.Expected {
			if code == 404 {
				t.Fatal("SDK success policy expanded", native)
			}
		}
	})
}

func TestPythonPortFindRecordMalformedPagesAndDetachedExtensionEvidence(t *testing.T) {
	for _, body := range []string{`{"ports":[null]}`, `{"ports":[4]}`, `{"ports":null}`, `{"ports":[]`, `{}`, `{"ports":[{"id":"match","name":"target","connection":null}]}`, `{"ports":[{"id":"match","name":"target","microversion":null}]}`, `{"ports":[{"id":"match","name":"target","_synchronized":null}]}`} {
		t.Run(body, func(t *testing.T) {
			var calls []nativePortCall
			api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					return nativePortWire(404, "")
				}
				response := nativePortWire(200, body)
				response.Header.Set("X-Request-Id", "bad-list")
				return response
			})
			got, err := api.FindRecord(context.Background(), "target")
			var response *resource.ResponseError
			if got != nil || !errors.As(err, &response) || response.StatusCode != 200 || string(response.Body) != body || response.Header.Get("X-Request-Id") != "bad-list" || len(calls) != 2 {
				t.Fatal(got, err, response, calls)
			}
		})
	}
	var calls []nativePortCall
	api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response { return nativePortWire(202, "not JSON") })
	got, err := api.FindRecord(context.Background(), "target", ports.WithRecordFindQuery("x_extension", []any{nil, 9007199254740993}))
	if err != nil || got == nil || got.Wire != nil || string(got.Envelope) != "not JSON" || len(calls) != 1 {
		t.Fatal(got, err, calls)
	}
	pythonFindField(t, got, "id", `"target"`)
	pythonFindField(t, got, "x_extension", `[null,9007199254740993]`)
}

func TestPythonPortFindRecordContinuationFormatsAndCycleGuards(t *testing.T) {
	for _, testcase := range []struct{ name, extra, header string }{
		{"RFC Link", ``, `<?marker=next>; rel="next"`},
		{"versioned top next", `,"next":"/v2.0/ports?marker=next"`, ``},
		{"dictionary links", `,"links":{"next":"?marker=next"}`, ``},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativePortCall
			api := pythonCorePortAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) == 1 {
					return nativePortWire(404, "")
				}
				if req.URL.Query().Get("name") != "target" {
					t.Fatal(req.URL)
				}
				if req.URL.Query().Get("marker") == "next" {
					return nativePortWire(200, `{"ports":[]}`)
				}
				response := nativePortWire(200, `{"ports":[{"id":"first","name":"target"}]`+testcase.extra+`}`)
				if testcase.header != "" {
					response.Header.Set("Link", testcase.header)
				}
				return response
			})
			got, err := api.FindRecord(context.Background(), "target")
			if err != nil || got == nil || len(calls) != 3 {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, testcase := range []struct {
		name, next string
		cause      error
	}{
		{"cycle", `?name=target`, resource.ErrPaginationCycle},
		{"changed filter", `?marker=next&name=other`, resource.ErrInvalidOption},
		{"changed collection", `https://other.invalid/ports?marker=next`, resource.ErrInvalidOption},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativePortCall
			api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					return nativePortWire(404, "")
				}
				raw, _ := json.Marshal(testcase.next)
				return nativePortWire(200, `{"ports":[{"id":"first","name":"target"}],"next":`+string(raw)+`}`)
			})
			got, err := api.FindRecord(context.Background(), "target")
			var receipt *resource.ResponseError
			if got != nil || !errors.Is(err, testcase.cause) || !errors.As(err, &receipt) || receipt.StatusCode != 200 || len(calls) != 2 {
				t.Fatal(got, err, receipt, calls)
			}
		})
	}
}

func TestPythonPortFindRecordLiveReauthenticationAndMicroversionOwnership(t *testing.T) {
	var calls []nativePortCall
	var tokens, versions []string
	api := pythonCorePortAPI(t, &calls, func(req *http.Request) *http.Response {
		tokens = append(tokens, req.Header.Get("X-Auth-Token"))
		versions = append(versions, req.Header.Get("OpenStack-API-Version"))
		if req.Header.Get("X-Trace") != "owned" || req.URL.Query().Get("fields") != "id" {
			t.Fatal(req.Header, req.URL)
		}
		if len(calls) == 1 {
			return nativePortWire(401, `{"error":"expired"}`)
		}
		return nativePortWire(200, `{"port":{"id":"target"}}`)
	})
	source := api.RawClient()
	source.Microversion = "2.0"
	source.MoreHeaders = map[string]string{"X-Trace": "source"}
	reauths := 0
	source.ProviderClient.ReauthFunc = func(context.Context) error { reauths++; source.ProviderClient.SetToken("fresh-token"); return nil }
	version := "2.1"
	option := ports.WithRecordFindOpts(ports.RecordFindOpts{Microversion: &version, Headers: map[string]string{"X-Trace": "owned"}, Query: map[string]json.RawMessage{"fields": json.RawMessage(`"id"`)}})
	version = "changed"
	got, err := api.FindRecord(context.Background(), "target", option)
	if err != nil || got == nil || len(calls) != 2 || reauths != 1 || !reflect.DeepEqual(tokens, []string{"test-token", "fresh-token"}) || !reflect.DeepEqual(versions, []string{"network 2.1", "network 2.1"}) || source.Microversion != "2.0" || source.MoreHeaders["X-Trace"] != "source" {
		t.Fatal(got, err, calls, reauths, tokens, versions, source)
	}
	var none []nativePortCall
	invalid := pythonCorePortAPI(t, &none, func(*http.Request) *http.Response { t.Fatal("invalid version HTTP"); return nil })
	got, err = invalid.FindRecord(context.Background(), "target", ports.WithRecordFindMicroversion("2.1\r\nchanged"))
	if got != nil || !errors.Is(err, resource.ErrInvalidOption) || len(none) != 0 {
		t.Fatal(got, err, none)
	}
}

func TestPythonPortFindRecordDictionaryTruthinessShortCircuitAndFailureReceipts(t *testing.T) {
	for _, testcase := range []struct {
		name, actual, filter string
		match, failure       bool
	}{
		{"empty dict accepts truthy scalar", `"now"`, `{}`, true, false},
		{"empty dict accepts truthy list", `[1]`, `{}`, true, false},
		{"empty dict rejects empty actual", `{}`, `{}`, false, false},
		{"nonempty dict cannot use scalar get", `"now"`, `{"nested":1}`, false, true},
		{"nonempty dict cannot use list get", `[1]`, `{"nested":1}`, false, true},
		{"nested scalar and empty dict", `{"nested":1}`, `{"nested":{}}`, true, false},
		{"nested scalar and nonempty dict fails", `{"nested":1}`, `{"nested":{"value":1}}`, false, true},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativePortCall
			body := `{"ports":[{"id":"match","name":"target","created_at":` + testcase.actual + `}]}`
			api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					return nativePortWire(404, "")
				}
				response := nativePortWire(203, body)
				response.Header.Set("X-Request-Id", "dict-filter")
				return response
			})
			got, err := api.FindRecord(context.Background(), "target", ports.WithRecordFindQuery("created_at", json.RawMessage(testcase.filter)))
			if len(calls) != 2 {
				t.Fatal(got, err, calls)
			}
			if testcase.failure {
				var receipt *resource.ResponseError
				if got != nil || !errors.As(err, &receipt) || receipt.StatusCode != 203 || string(receipt.Body) != body || receipt.Header.Get("X-Request-Id") != "dict-filter" {
					t.Fatal(got, err, receipt)
				}
			} else if err != nil || (got != nil) != testcase.match {
				t.Fatal(got, err)
			}
		})
	}
	for _, firstMatches := range []bool{false, true} {
		t.Run(map[bool]string{false: "first false stops before scalar dict error", true: "first true observes scalar dict error"}[firstMatches], func(t *testing.T) {
			var calls []nativePortCall
			body := `{"ports":[{"id":"match","name":"target","created_at":"observed","revision_number":1}]}`
			api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					return nativePortWire(404, "")
				}
				response := nativePortWire(200, body)
				response.Header.Set("X-Request-Id", "short-circuit")
				return response
			})
			first := "different"
			if firstMatches {
				first = "observed"
			}
			got, err := api.FindRecord(context.Background(), "target", ports.WithRecordFindQuery("created_at", first), ports.WithRecordFindQuery("revision_number", map[string]any{"value": 1}))
			if got != nil || len(calls) != 2 {
				t.Fatal(got, err, calls)
			}
			if firstMatches {
				var receipt *resource.ResponseError
				if !errors.As(err, &receipt) || string(receipt.Body) != body || receipt.Header.Get("X-Request-Id") != "short-circuit" {
					t.Fatal(err, receipt)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPythonPortFindRecordPaginationTruthinessAndHeaderConstructorProjection(t *testing.T) {
	for _, testcase := range []struct {
		name  string
		flag  any
		pages int
	}{
		{"null", nil, 1}, {"zero", 0, 1}, {"empty list", []any{}, 1}, {"nonzero", 1, 2}, {"nonempty string", "False", 2},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativePortCall
			api := pythonCorePortAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) == 1 {
					return nativePortWire(404, "")
				}
				if req.URL.Query().Get("marker") == "next" {
					return nativePortWire(200, `{"ports":[]}`)
				}
				return nativePortWire(200, `{"ports":[{"id":"match","name":"target"}],"next":"?marker=next"}`)
			})
			got, err := api.FindRecord(context.Background(), "target", ports.WithRecordFindQuery("paginated", testcase.flag))
			if err != nil || got == nil || len(calls) != testcase.pages+1 {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, testcase := range []struct {
		name, body, seed, header, want string
		list                           bool
	}{
		{"GET actual header wins Body", `{"port":{"id":"target","if_match":"body","if-match":"wire"}}`, "", "actual", `["actual"]`, false},
		{"GET seed header survives unknown Body", `{"port":{"id":"target","if_match":"body"}}`, "seed", "", `["seed"]`, false},
		{"LIST canonical header constructor", `{"ports":[{"id":"match","name":"target","if_match":"row"}]}`, "", "actual", `["row"]`, true},
		{"LIST wire header constructor", `{"ports":[{"id":"match","name":"target","If-Match":"row"}]}`, "", "actual", `["row"]`, true},
		{"LIST null header constructor", `{"ports":[{"id":"match","name":"target","if_match":null}]}`, "", "actual", `null`, true},
		{"LIST does not inflate page Header into row", `{"ports":[{"id":"match","name":"target"}]}`, "", "actual", `null`, true},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativePortCall
			api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
				if testcase.list && len(calls) == 1 {
					return nativePortWire(404, "")
				}
				response := nativePortWire(200, testcase.body)
				if testcase.header != "" {
					response.Header.Set("If-Match", testcase.header)
				}
				return response
			})
			options := []ports.RecordFindOption{}
			if testcase.seed != "" {
				options = append(options, ports.WithRecordFindQuery("if_match", testcase.seed))
			}
			got, err := api.FindRecord(context.Background(), "target", options...)
			if err != nil || got == nil {
				t.Fatal(got, err, calls)
			}
			pythonFindField(t, got, "if_match", testcase.want)
			if testcase.header != "" && got.Header.Get("If-Match") != testcase.header {
				t.Fatal("lost actual HTTP header", got.Header)
			}
		})
	}
}

func TestPythonPortFindRecordListAcceptAndRetryVersionPolicy(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(map[bool]string{false: "Source LIST default Accept overrides configured Header", true: "explicit caller LIST Accept wins"}[explicit], func(t *testing.T) {
			var calls []nativePortCall
			var accepts []string
			api := pythonCorePortAPI(t, &calls, func(req *http.Request) *http.Response {
				accepts = append(accepts, req.Header.Get("Accept"))
				if len(calls) == 1 {
					return nativePortWire(404, "")
				}
				return nativePortWire(200, `{"ports":[]}`)
			})
			api.RawClient().MoreHeaders = map[string]string{"Accept": "text/xml"}
			options := []ports.RecordFindOption{}
			want := []string{"text/xml", "application/json"}
			if explicit {
				options = append(options, ports.WithRecordFindHeaders(map[string]string{"Accept": "text/plain"}))
				want = []string{"text/plain", "text/plain"}
			}
			got, err := api.FindRecord(context.Background(), "target", options...)
			if got != nil || err != nil || len(calls) != 2 || !reflect.DeepEqual(accepts, want) || api.RawClient().MoreHeaders["Accept"] != "text/xml" {
				t.Fatal(got, err, calls, accepts, want)
			}
		})
	}
	for _, version := range []string{"", "2.1"} {
		t.Run("retry cannot change version "+version, func(t *testing.T) {
			var calls []nativePortCall
			body := `{"error":"original"}`
			api := pythonCorePortAPI(t, &calls, func(*http.Request) *http.Response {
				response := nativePortWire(404, body)
				response.Header.Set("X-Request-Id", "version-guard")
				return response
			})
			api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, options *gophercloud.RequestOpts, original error, _ uint) error {
				if options.MoreHeaders == nil {
					options.MoreHeaders = map[string]string{}
				}
				options.MoreHeaders["OpenStack-API-Version"] = "network 2.9"
				return nil
			}
			options := []ports.RecordFindOption{}
			if version != "" {
				options = append(options, ports.WithRecordFindMicroversion(version))
			}
			got, err := api.FindRecord(context.Background(), "target", options...)
			var native gophercloud.ErrUnexpectedResponseCode
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "version-guard" || len(calls) != 1 {
				t.Fatal(got, err, native, calls)
			}
		})
	}
}
