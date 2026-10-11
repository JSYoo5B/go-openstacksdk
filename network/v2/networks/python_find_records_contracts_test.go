package networks_test

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

	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonFindField(t *testing.T, value *networks.NetworkRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatalf("%s: want %s, record=%#v", key, want, value)
	}
}
func pythonFindWireField(t *testing.T, value *networks.NetworkRecord, key, want string) {
	t.Helper()
	if value == nil || value.Wire == nil || string(value.Wire.Body[key]) != want {
		t.Fatalf("wire %s: want %s, record=%#v", key, want, value)
	}
}

// Source Resource.find first attempts the literal member route with raw kwargs;
// accepted incomplete responses retain constructor state and descriptor defaults.
func TestPythonNetworkFindRecordMemberFirstSeedAndResponseProjection(t *testing.T) {
	for _, testcase := range []struct {
		name, body string
		checks     map[string]string
	}{
		{"omitted Body defaults", `{"network":{}}`, map[string]string{"id": `"target"`, "name": "null", "tags": "[]", "revision_number": "null", "is_port_security_enabled": "false", "is_router_external": "false", "is_shared": "null", "location": "null", "if_match": `["revision_number=7"]`, "x_seed": `{"keep":null}`}},
		{"explicit null overrides seeded identity", `{"network":{"id":null,"name":null,"tags":null,"revision_number":null,"port_security_enabled":null,"router:external":null,"x_extension":9007199254740993,"self":"drop"}}`, map[string]string{"id": "null", "name": "null", "tags": "null", "revision_number": "null", "is_port_security_enabled": "null", "is_router_external": "null", "x_extension": "9007199254740993"}},
		{"descriptor coercion and ordered wire aliases", `{"network":{"id":"wire-id","name":"wire-name","revision_number":"٩٠","tags":"red","admin_state_up":"False","mtu":1450.9,"provider:segmentation_id":9007199254740993,"subnet_ids":["old"],"subnets":"sn","x_extension":{"big":9007199254740993}}}`, map[string]string{"id": `"wire-id"`, "revision_number": "90", "tags": `["red"]`, "is_admin_state_up": "true", "mtu": "1450", "provider_segmentation_id": "9007199254740993", "subnet_ids": `["sn"]`, "x_extension": `{"big":9007199254740993}`}},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativeNetworkCall
			api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Query().Get("x_seed") != "keep" || req.URL.Query().Get("status") != "ACTIVE" {
					t.Fatal(req.URL)
				}
				response := nativeNetworkWire(200, testcase.body)
				response.Header.Set("If-Match", "revision_number=7")
				response.Header.Set("X-Request-Id", "member")
				return response
			})
			value, err := api.FindRecord(context.Background(), "target", networks.WithRecordFindQuery("x_seed", map[string]any{"keep": nil}), networks.WithRecordFindQuery("status", "ACTIVE"))
			if err != nil || len(calls) != 1 || calls[0].path != "/neutron/v2.0/networks/target" || calls[0].method != http.MethodGet {
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
	var calls []nativeNetworkCall
	api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { return nativeNetworkWire(204, "") })
	value, err := api.FindRecord(context.Background(), "literal/id %2F", networks.WithRecordFindQuery("name", "seed"))
	if err != nil || len(calls) != 1 || value.Wire != nil || value.StatusCode != 204 {
		t.Fatal(value, err, calls)
	}
	pythonFindField(t, value, "id", `"literal/id %2F"`)
	pythonFindField(t, value, "name", `"seed"`)
}

// QueryParameters declaration order, explicit null/false, repeated fields/tags,
// nested local equality and unknown-parameter discard are Source contracts.
func TestPythonNetworkFindRecordFallbackAliasesAndLocalBodyFilters(t *testing.T) {
	var calls []nativeNetworkCall
	var listed url.Values
	api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path == "/neutron/v2.0/networks/target" {
			q := req.URL.Query()
			if q.Get("x_unknown") != "keep" || q.Has("name") || q.Has("all_projects") || q.Get("paginated") != "False" {
				t.Fatal(q)
			}
			return nativeNetworkWire(403, `{"error":"member hidden"}`)
		}
		listed = req.URL.Query()
		if req.Header.Get("X-Trace") != "owned" {
			t.Fatal(req.Header)
		}
		return nativeNetworkWire(200, `{"networks":[{"id":"filtered","name":"target","mtu":2},{"id":"match","name":"target","mtu":true,"x_extension":null,"self":"drop"}]}`)
	})
	configured := map[string]string{"X-Trace": "old"}
	api.RawClient().MoreHeaders = configured
	value, err := api.FindRecord(context.Background(), "target",
		networks.WithRecordFindQuery("x_unknown", "keep"), networks.WithRecordFindQuery("all_projects", true), networks.WithRecordFindQuery("fields", []string{"id", "name"}),
		networks.WithRecordFindQuery("any_tags", []string{"red", "blue"}), networks.WithRecordFindQuery("not_any_tags", "black"), networks.WithRecordFindQuery("paginated", false),
		networks.WithRecordFindHeaders(map[string]string{"X-Trace": "owned"}), networks.WithRecordFindQuery("is_shared", false), networks.WithRecordFindQuery("shared", true), networks.WithRecordFindQuery("provider_network_type", "vxlan"), networks.WithRecordFindQuery("provider:network_type", "vlan"), networks.WithRecordFindQuery("mtu", true))
	if err != nil || value == nil || len(calls) != 2 {
		t.Fatal(value, err, calls)
	}
	pythonFindField(t, value, "id", `"match"`)
	pythonFindField(t, value, "x_extension", "null")
	if _, exists := value.Resource.Body["self"]; exists {
		t.Fatal(value.Resource.Body)
	}
	pythonFindWireField(t, value, "self", `"drop"`)
	want := url.Values{"name": {"target"}, "fields": {"id", "name"}, "tags-any": {"red", "blue"}, "not-tags-any": {"black"}, "shared": {"False"}, "provider:network_type": {"vxlan"}}
	if !reflect.DeepEqual(listed, want) || configured["X-Trace"] != "old" {
		t.Fatal(listed, want, configured)
	}
}

func TestPythonNetworkFindRecordNamePresenceMissingAndPassiveResponseID(t *testing.T) {
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
				var calls []nativeNetworkCall
				api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
					if len(calls) == 1 {
						return nativeNetworkWire(status, `{"error":"missing"}`)
					}
					if !reflect.DeepEqual(req.URL.Query(), testcase.want) {
						t.Fatal(req.URL.Query(), testcase.want)
					}
					return nativeNetworkWire(200, `{"networks":[]}`)
				})
				options := []networks.RecordFindOption{}
				if testcase.supplied {
					options = append(options, networks.WithRecordFindQuery("name", testcase.value))
				}
				got, err := api.FindRecord(context.Background(), "target", options...)
				if err != nil || got != nil || len(calls) != 2 {
					t.Fatal(got, err, calls)
				}
			})
		}
	}
	for _, body := range []string{`{"networks":[{"name":"target"}]}`, `{"networks":{"id":17,"name":"target"}}`} {
		var calls []nativeNetworkCall
		api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeNetworkWire(404, "")
			}
			return nativeNetworkWire(200, body)
		})
		got, err := api.FindRecord(context.Background(), "target")
		if err != nil || got == nil || len(calls) != 2 {
			t.Fatal(got, err, calls)
		}
	}
	for _, status := range []int{200, 404} {
		var calls []nativeNetworkCall
		api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeNetworkWire(404, "")
			}
			body := `{"networks":[]}`
			if status != 200 {
				body = `{"error":"list denied"}`
			}
			return nativeNetworkWire(status, body)
		})
		got, err := api.FindRecord(context.Background(), "target", networks.WithRecordFindIgnoreMissing(false))
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

func TestPythonNetworkFindRecordLateDuplicateErrorMarkerAndRawCap(t *testing.T) {
	for _, testcase := range []struct {
		name      string
		second    int
		duplicate bool
	}{
		{"late duplicate", 200, true}, {"late page denied", 500, false},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativeNetworkCall
			api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeNetworkWire(404, "")
				}
				if req.URL.Query().Get("marker") == "late" {
					if testcase.duplicate {
						return nativeNetworkWire(200, `{"networks":[{"id":"second","name":"target"}]}`)
					}
					return nativeNetworkWire(testcase.second, `{"error":"late"}`)
				}
				return nativeNetworkWire(200, `{"networks":[{"id":"first","name":"target"}],"networks_links":[{"rel":"next","href":"?marker=late"}]}`)
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
		var calls []nativeNetworkCall
		api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeNetworkWire(404, "")
			}
			q := req.URL.Query()
			if q.Get("limit") != "2" || q.Get("name") != "target" || !reflect.DeepEqual(q["tags"], []string{"a", "b"}) {
				t.Fatal(q)
			}
			if q.Get("marker") == "7" {
				return nativeNetworkWire(200, `{"networks":[]}`)
			}
			return nativeNetworkWire(200, `{"networks":[{"id":7,"name":"target"}]}`)
		})
		got, err := api.FindRecord(context.Background(), "target", networks.WithRecordFindQuery("limit", 2), networks.WithRecordFindQuery("tags", []string{"a", "b"}))
		if err != nil || got == nil || len(calls) != 3 {
			t.Fatal(got, err, calls)
		}
		pythonFindField(t, got, "id", "7")
	})
	t.Run("raw cap occurs before local filtering", func(t *testing.T) {
		var calls []nativeNetworkCall
		api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeNetworkWire(404, "")
			}
			if req.URL.Query().Get("limit") != "1" {
				t.Fatal(req.URL)
			}
			return nativeNetworkWire(200, `{"networks":[{"id":"filtered","name":"target","mtu":2},{"id":"match","name":"target","mtu":true}],"networks_links":[{"rel":"next","href":"?marker=late"}]}`)
		})
		got, err := api.FindRecord(context.Background(), "target", networks.WithRecordFindQuery("max_items", 1), networks.WithRecordFindQuery("mtu", true))
		if err != nil || got != nil || len(calls) != 2 {
			t.Fatal(got, err, calls)
		}
	})
}

func TestPythonNetworkFindRecordOptionSnapshotsAndPreflightGuards(t *testing.T) {
	fields := []string{"id", "name"}
	query := map[string]json.RawMessage{"fields": json.RawMessage(`["id","name"]`), "x_seed": json.RawMessage(`{"initial":true}`)}
	headers := map[string]string{"X-Trace": "original"}
	ignore := false
	options := networks.WithRecordFindOpts(networks.RecordFindOpts{Query: query, Headers: headers, IgnoreMissing: &ignore})
	query["fields"][2] = 'x'
	delete(query, "x_seed")
	headers["X-Trace"] = "changed"
	ignore = true
	fieldOption := networks.WithRecordFindQuery("fields", fields)
	fields[0] = "changed"
	var escaped *networks.RecordFindOpts
	var calls []nativeNetworkCall
	api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Header.Get("X-Trace") != "original" || !reflect.DeepEqual(req.URL.Query()["fields"], []string{"id", "name"}) || req.URL.Query().Get("x_seed") != "initial" {
			t.Fatal(req.Header, req.URL)
		}
		escaped.Query["fields"][2] = 'z'
		escaped.Headers["X-Trace"] = "callback changed"
		return nativeNetworkWire(200, `{"network":{}}`)
	})
	got, err := api.FindRecord(context.Background(), "target", options, fieldOption, func(c *networks.RecordFindOpts) error { escaped = c; return nil })
	if err != nil || got == nil || len(calls) != 1 {
		t.Fatal(got, err, calls)
	}
	pythonFindField(t, got, "fields", `["id","name"]`)
	for _, testcase := range []struct {
		name    string
		options []networks.RecordFindOption
		cause   error
	}{
		{"nil", []networks.RecordFindOption{nil}, resource.ErrInvalidOption},
		{"malformed JSON", []networks.RecordFindOption{networks.WithRecordFindOpts(networks.RecordFindOpts{Query: map[string]json.RawMessage{"fields": json.RawMessage(`[`)}})}, resource.ErrInvalidOption},
		{"negative cap", []networks.RecordFindOption{networks.WithRecordFindQuery("max_items", -1)}, resource.ErrInvalidOption},
		{"bad page flag", []networks.RecordFindOption{networks.WithRecordFindOpts(networks.RecordFindOpts{Query: map[string]json.RawMessage{"paginated": json.RawMessage(`bad flag`)}})}, resource.ErrInvalidOption},
		{"identity keyword collision", []networks.RecordFindOption{networks.WithRecordFindQuery("id", "different")}, resource.ErrInvalidOption},
		{"route override", []networks.RecordFindOption{networks.WithRecordFindQuery("base_path", "/other")}, resource.ErrInvalidOption},
		{"fetch requires_id control", []networks.RecordFindOption{networks.WithRecordFindQuery("requires_id", false)}, resource.ErrInvalidOption},
		{"fetch error_message control", []networks.RecordFindOption{networks.WithRecordFindQuery("error_message", "changed")}, resource.ErrInvalidOption},
		{"fetch cache control", []networks.RecordFindOption{networks.WithRecordFindQuery("skip_cache", true)}, resource.ErrInvalidOption},
		{"fetch response envelope control", []networks.RecordFindOption{networks.WithRecordFindQuery("resource_response_key", "other")}, resource.ErrInvalidOption},
		{"protected Header", []networks.RecordFindOption{networks.WithRecordFindHeaders(map[string]string{"X-Auth-Token": "different"})}, resource.ErrInvalidOption},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativeNetworkCall
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("unexpected HTTP"); return nil })
			got, err := api.FindRecord(context.Background(), "target", testcase.options...)
			if got != nil || !errors.Is(err, testcase.cause) || len(calls) != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
	t.Run("source mutation is sticky", func(t *testing.T) {
		var calls []nativeNetworkCall
		api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("unexpected HTTP"); return nil })
		got, err := api.FindRecord(context.Background(), "target", func(*networks.RecordFindOpts) error { api.RawClient().ResourceBase += "changed/"; return nil })
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(got, err, calls)
		}
	})
	t.Run("canceled cause prevents HTTP", func(t *testing.T) {
		var calls []nativeNetworkCall
		api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("unexpected HTTP"); return nil })
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

func TestPythonNetworkFindRecordPhysicalFailuresPreserveActualEvidence(t *testing.T) {
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
			var calls []nativeNetworkCall
			payload := `{"error":"physical"}`
			if testcase.status == 200 {
				payload = `{"network":{"id":"target"}}`
			}
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
				if testcase.list && len(calls) == 1 {
					return nativeNetworkWire(404, "")
				}
				response := nativeNetworkWire(testcase.status, payload)
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
				if !errors.As(err, &native) || native.Actual != testcase.status || string(native.Body) != payload || native.ResponseHeader.Get("X-Request-Id") != "physical" || !strings.Contains(native.URL, "/neutron/v2.0/networks") {
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
		var calls []nativeNetworkCall
		cause := errors.New("late cancellation")
		ctx, cancel := context.WithCancelCause(context.Background())
		api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
			response := nativeNetworkWire(200, `{"network":{}}`)
			response.Body = &pythonFindFaultBody{data: `{"network":{}}`, onClose: func() { cancel(cause) }}
			return response
		})
		got, err := api.FindRecord(ctx, "target")
		if got != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || len(calls) != 1 {
			t.Fatal(got, err, calls)
		}
	})
}

func TestPythonNetworkFindRecordRetryCallbacksKeepCleanFallbackAndGuards(t *testing.T) {
	for _, fault := range []bool{false, true} {
		t.Run(map[bool]string{false: "clean callback permits fallback", true: "physical rejection blocks fallback"}[fault], func(t *testing.T) {
			var calls []nativeNetworkCall
			readCause := errors.New("rejected read")
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					response := nativeNetworkWire(404, `{"error":"missing"}`)
					if fault {
						response.Body = &pythonFindFaultBody{data: `{"error":"missing"}`, read: readCause}
					}
					return response
				}
				return nativeNetworkWire(200, `{"networks":[]}`)
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
		var calls []nativeNetworkCall
		api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
			response := nativeNetworkWire(404, `{"error":"actual rejected retry"}`)
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
			if call.path != "/neutron/v2.0/networks/target" {
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

func TestPythonNetworkFindRecordMalformedPagesAndDetachedExtensionEvidence(t *testing.T) {
	for _, body := range []string{`{"networks":[null]}`, `{"networks":[4]}`, `{"networks":null}`, `{"networks":[]`, `{}`, `{"networks":[{"id":"match","name":"target","connection":null}]}`, `{"networks":[{"id":"match","name":"target","microversion":null}]}`, `{"networks":[{"id":"match","name":"target","_synchronized":null}]}`} {
		t.Run(body, func(t *testing.T) {
			var calls []nativeNetworkCall
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeNetworkWire(404, "")
				}
				response := nativeNetworkWire(200, body)
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
	var calls []nativeNetworkCall
	api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { return nativeNetworkWire(202, "not JSON") })
	got, err := api.FindRecord(context.Background(), "target", networks.WithRecordFindQuery("x_extension", []any{nil, 9007199254740993}))
	if err != nil || got == nil || got.Wire != nil || string(got.Envelope) != "not JSON" || len(calls) != 1 {
		t.Fatal(got, err, calls)
	}
	pythonFindField(t, got, "id", `"target"`)
	pythonFindField(t, got, "x_extension", `[null,9007199254740993]`)
}

func TestPythonNetworkFindRecordContinuationFormatsAndCycleGuards(t *testing.T) {
	for _, testcase := range []struct{ name, extra, header string }{
		{"RFC Link", ``, `<?marker=next>; rel="next"`},
		{"versioned top next", `,"next":"/v2.0/networks?marker=next"`, ``},
		{"dictionary links", `,"links":{"next":"?marker=next"}`, ``},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativeNetworkCall
			api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeNetworkWire(404, "")
				}
				if req.URL.Query().Get("name") != "target" {
					t.Fatal(req.URL)
				}
				if req.URL.Query().Get("marker") == "next" {
					return nativeNetworkWire(200, `{"networks":[]}`)
				}
				response := nativeNetworkWire(200, `{"networks":[{"id":"first","name":"target"}]`+testcase.extra+`}`)
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
		{"changed collection", `https://other.invalid/networks?marker=next`, resource.ErrInvalidOption},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativeNetworkCall
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeNetworkWire(404, "")
				}
				raw, _ := json.Marshal(testcase.next)
				return nativeNetworkWire(200, `{"networks":[{"id":"first","name":"target"}],"next":`+string(raw)+`}`)
			})
			got, err := api.FindRecord(context.Background(), "target")
			var receipt *resource.ResponseError
			if got != nil || !errors.Is(err, testcase.cause) || !errors.As(err, &receipt) || receipt.StatusCode != 200 || len(calls) != 2 {
				t.Fatal(got, err, receipt, calls)
			}
		})
	}
}

func TestPythonNetworkFindRecordLiveReauthenticationAndMicroversionOwnership(t *testing.T) {
	var calls []nativeNetworkCall
	var tokens, versions []string
	api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
		tokens = append(tokens, req.Header.Get("X-Auth-Token"))
		versions = append(versions, req.Header.Get("OpenStack-API-Version"))
		if req.Header.Get("X-Trace") != "owned" || req.URL.Query().Get("fields") != "id" {
			t.Fatal(req.Header, req.URL)
		}
		if len(calls) == 1 {
			return nativeNetworkWire(401, `{"error":"expired"}`)
		}
		return nativeNetworkWire(200, `{"network":{"id":"target"}}`)
	})
	source := api.RawClient()
	source.Microversion = "2.0"
	source.MoreHeaders = map[string]string{"X-Trace": "source"}
	reauths := 0
	source.ProviderClient.ReauthFunc = func(context.Context) error { reauths++; source.ProviderClient.SetToken("fresh-token"); return nil }
	version := "2.1"
	option := networks.WithRecordFindOpts(networks.RecordFindOpts{Microversion: &version, Headers: map[string]string{"X-Trace": "owned"}, Query: map[string]json.RawMessage{"fields": json.RawMessage(`"id"`)}})
	version = "changed"
	got, err := api.FindRecord(context.Background(), "target", option)
	if err != nil || got == nil || len(calls) != 2 || reauths != 1 || !reflect.DeepEqual(tokens, []string{"test-token", "fresh-token"}) || !reflect.DeepEqual(versions, []string{"network 2.1", "network 2.1"}) || source.Microversion != "2.0" || source.MoreHeaders["X-Trace"] != "source" {
		t.Fatal(got, err, calls, reauths, tokens, versions, source)
	}
	var none []nativeNetworkCall
	invalid := pythonCoreNetworkAPI(t, &none, func(*http.Request) *http.Response { t.Fatal("invalid version HTTP"); return nil })
	got, err = invalid.FindRecord(context.Background(), "target", networks.WithRecordFindMicroversion("2.1\r\nchanged"))
	if got != nil || !errors.Is(err, resource.ErrInvalidOption) || len(none) != 0 {
		t.Fatal(got, err, none)
	}
}

func TestPythonNetworkFindRecordDictionaryTruthinessShortCircuitAndFailureReceipts(t *testing.T) {
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
			var calls []nativeNetworkCall
			body := `{"networks":[{"id":"match","name":"target","created_at":` + testcase.actual + `}]}`
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeNetworkWire(404, "")
				}
				response := nativeNetworkWire(203, body)
				response.Header.Set("X-Request-Id", "dict-filter")
				return response
			})
			got, err := api.FindRecord(context.Background(), "target", networks.WithRecordFindQuery("created_at", json.RawMessage(testcase.filter)))
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
			var calls []nativeNetworkCall
			body := `{"networks":[{"id":"match","name":"target","created_at":"observed","revision_number":1}]}`
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeNetworkWire(404, "")
				}
				response := nativeNetworkWire(200, body)
				response.Header.Set("X-Request-Id", "short-circuit")
				return response
			})
			first := "different"
			if firstMatches {
				first = "observed"
			}
			got, err := api.FindRecord(context.Background(), "target", networks.WithRecordFindQuery("created_at", first), networks.WithRecordFindQuery("revision_number", map[string]any{"value": 1}))
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

func TestPythonNetworkFindRecordPaginationTruthinessAndHeaderConstructorProjection(t *testing.T) {
	for _, testcase := range []struct {
		name  string
		flag  any
		pages int
	}{
		{"null", nil, 1}, {"zero", 0, 1}, {"empty list", []any{}, 1}, {"nonzero", 1, 2}, {"nonempty string", "False", 2},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativeNetworkCall
			api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeNetworkWire(404, "")
				}
				if req.URL.Query().Get("marker") == "next" {
					return nativeNetworkWire(200, `{"networks":[]}`)
				}
				return nativeNetworkWire(200, `{"networks":[{"id":"match","name":"target"}],"next":"?marker=next"}`)
			})
			got, err := api.FindRecord(context.Background(), "target", networks.WithRecordFindQuery("paginated", testcase.flag))
			if err != nil || got == nil || len(calls) != testcase.pages+1 {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, testcase := range []struct {
		name, body, seed, header, want string
		list                           bool
	}{
		{"GET actual header wins Body", `{"network":{"id":"target","if_match":"body","if-match":"wire"}}`, "", "actual", `["actual"]`, false},
		{"GET seed header survives unknown Body", `{"network":{"id":"target","if_match":"body"}}`, "seed", "", `["seed"]`, false},
		{"LIST canonical header constructor", `{"networks":[{"id":"match","name":"target","if_match":"row"}]}`, "", "actual", `["row"]`, true},
		{"LIST wire header constructor", `{"networks":[{"id":"match","name":"target","If-Match":"row"}]}`, "", "actual", `["row"]`, true},
		{"LIST null header constructor", `{"networks":[{"id":"match","name":"target","if_match":null}]}`, "", "actual", `null`, true},
		{"LIST does not inflate page Header into row", `{"networks":[{"id":"match","name":"target"}]}`, "", "actual", `null`, true},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativeNetworkCall
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
				if testcase.list && len(calls) == 1 {
					return nativeNetworkWire(404, "")
				}
				response := nativeNetworkWire(200, testcase.body)
				if testcase.header != "" {
					response.Header.Set("If-Match", testcase.header)
				}
				return response
			})
			options := []networks.RecordFindOption{}
			if testcase.seed != "" {
				options = append(options, networks.WithRecordFindQuery("if_match", testcase.seed))
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

func TestPythonNetworkFindRecordListAcceptAndRetryVersionPolicy(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(map[bool]string{false: "Source LIST default Accept overrides configured Header", true: "explicit caller LIST Accept wins"}[explicit], func(t *testing.T) {
			var calls []nativeNetworkCall
			var accepts []string
			api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
				accepts = append(accepts, req.Header.Get("Accept"))
				if len(calls) == 1 {
					return nativeNetworkWire(404, "")
				}
				return nativeNetworkWire(200, `{"networks":[]}`)
			})
			api.RawClient().MoreHeaders = map[string]string{"Accept": "text/xml"}
			options := []networks.RecordFindOption{}
			want := []string{"text/xml", "application/json"}
			if explicit {
				options = append(options, networks.WithRecordFindHeaders(map[string]string{"Accept": "text/plain"}))
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
			var calls []nativeNetworkCall
			body := `{"error":"original"}`
			api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
				response := nativeNetworkWire(404, body)
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
			options := []networks.RecordFindOption{}
			if version != "" {
				options = append(options, networks.WithRecordFindMicroversion(version))
			}
			got, err := api.FindRecord(context.Background(), "target", options...)
			var native gophercloud.ErrUnexpectedResponseCode
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "version-guard" || len(calls) != 1 {
				t.Fatal(got, err, native, calls)
			}
		})
	}
}
