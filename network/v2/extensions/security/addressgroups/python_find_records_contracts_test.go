package addressgroups_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func pythonFindField(t *testing.T, value *addressgroups.AddressGroupFindRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatalf("%s: want %s, record=%#v", key, want, value)
	}
}
func pythonFindWireField(t *testing.T, value *addressgroups.AddressGroupFindRecord, key, want string) {
	t.Helper()
	if value == nil || value.Wire == nil || string(value.Wire.Body[key]) != want {
		t.Fatalf("wire %s: want %s, record=%#v", key, want, value)
	}
}
func pythonFindAPI(t *testing.T, calls *[]nativeGroupCall, reply func(*http.Request) *http.Response) *addressgroups.API {
	t.Helper()
	api, _ := nativeGroupAPI(t, calls, reply)
	return api
}

func TestPythonAddressGroupFindRecordMemberSeedDefaultsAndActualEvidence(t *testing.T) {
	for _, body := range []string{`{"address_group":{}}`, `{"address_group":{"id":null,"name":false,"description":0,"tenant_id":"legacy","project_id":null,"x_new":9007199254740993,"self":"drop"}}`, `not JSON`, ``} {
		var calls []nativeGroupCall
		api := pythonFindAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.URL.EscapedPath() != "/neutron/v2.0/address-groups/literal%2Fid%20%252F" || req.URL.Query().Get("x_seed") != "nested" {
				t.Fatal(req.URL)
			}
			response := nativeGroupWire(203, body)
			response.Header.Set("If-Match", "actual")
			response.Header.Set("X-Request-Id", "receipt")
			return response
		})
		got, err := api.FindRecord(context.Background(), "literal/id %2F", addressgroups.WithRecordFindQuery("x_seed", map[string]any{"nested": nil}), addressgroups.WithRecordFindQuery("name", "seed"), addressgroups.WithRecordFindQuery("if_match", "seed-header"))
		if err != nil || got == nil || len(calls) != 1 || calls[0].body != "" || got.StatusCode != 203 || string(got.Envelope) != body || got.Header.Get("X-Request-Id") != "receipt" {
			t.Fatal(got, err, calls)
		}
		pythonFindField(t, got, "x_seed", `{"nested":null}`)
		pythonFindField(t, got, "if_match", `"seed-header"`)
		pythonFindField(t, got, "location", `null`)
		if strings.Contains(body, `"id":null`) {
			pythonFindField(t, got, "id", `null`)
			pythonFindField(t, got, "project_id", `null`)
			pythonFindField(t, got, "name", `false`)
			pythonFindField(t, got, "description", `0`)
			pythonFindField(t, got, "x_new", `9007199254740993`)
			pythonFindWireField(t, got, "self", `"drop"`)
			if got.Resource.Body["self"] != nil {
				t.Fatal(got.Resource.Body)
			}
		} else {
			pythonFindField(t, got, "id", `"literal/id %2F"`)
			pythonFindField(t, got, "name", `"seed"`)
		}
		pythonFindField(t, got, "addresses", `null`)
		got.Resource.Header.Set("X-Request-Id", "changed")
		got.Header.Set("X-Request-Id", "changed")
		if got.Wire != nil && got.Wire.Header.Get("X-Request-Id") != "receipt" {
			t.Fatal("receipt aliases resource")
		}
	}
	for _, body := range []string{`{"address_group":null}`, `null`, `[]`, `{"address_group":false}`} {
		var calls []nativeGroupCall
		api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(200, body) })
		got, err := api.FindRecord(context.Background(), "target")
		var proof *resource.ResponseError
		if got != nil || !errors.As(err, &proof) || string(proof.Body) != body || len(calls) != 1 {
			t.Fatal(got, err, proof, calls)
		}
	}
}
func TestPythonAddressGroupFindRecordFallbackQueryMappingAndLocalFilters(t *testing.T) {
	var calls []nativeGroupCall
	var listed url.Values
	api := pythonFindAPI(t, &calls, func(req *http.Request) *http.Response {
		if len(calls) == 1 {
			if req.URL.Query().Get("x_unknown") != "keep" || req.URL.Query().Has("all_projects") {
				t.Fatal(req.URL)
			}
			return nativeGroupWire(403, `{"error":"member hidden"}`)
		}
		listed = req.URL.Query()
		return nativeGroupWire(200, `{"address_groups":[{"id":"filtered","name":"target","tenant_id":2},{"id":"match","name":"target","tenant_id":1,"x_extension":null}]}`)
	})
	got, err := api.FindRecord(context.Background(), "target", addressgroups.WithRecordFindQuery("x_unknown", "keep"), addressgroups.WithRecordFindQuery("all_projects", true), addressgroups.WithRecordFindQuery("fields", []string{"id", "name"}), addressgroups.WithRecordFindQuery("tenant_id", true), addressgroups.WithRecordFindQuery("is_admin_state_up", false), addressgroups.WithRecordFindQuery("admin_state_up", true), addressgroups.WithRecordFindQuery("any_tags", []string{"red", "blue"}), addressgroups.WithRecordFindQuery("not_any_tags", "black"), addressgroups.WithRecordFindQuery("sort_key", "name"), addressgroups.WithRecordFindQuery("sort_dir", "asc"))
	if err != nil || got == nil || len(calls) != 2 {
		t.Fatal(got, err, calls)
	}
	pythonFindField(t, got, "id", `"match"`)
	pythonFindField(t, got, "project_id", `1`)
	pythonFindField(t, got, "x_extension", `null`)
	want := url.Values{"name": {"target"}, "fields": {"id", "name"}, "sort_key": {"name"}, "sort_dir": {"asc"}}
	if !reflect.DeepEqual(listed, want) {
		t.Fatal(listed, want)
	}
}
func TestPythonAddressGroupFindRecordNamePresenceMissingAndPassiveResponseID(t *testing.T) {
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
				var calls []nativeGroupCall
				api := pythonFindAPI(t, &calls, func(req *http.Request) *http.Response {
					if len(calls) == 1 {
						return nativeGroupWire(status, `{"error":"missing"}`)
					}
					if !reflect.DeepEqual(req.URL.Query(), testcase.want) {
						t.Fatal(req.URL.Query(), testcase.want)
					}
					return nativeGroupWire(200, `{"address_groups":[]}`)
				})
				options := []addressgroups.RecordFindOption{}
				if testcase.supplied {
					options = append(options, addressgroups.WithRecordFindQuery("name", testcase.value))
				}
				got, err := api.FindRecord(context.Background(), "target", options...)
				if err != nil || got != nil || len(calls) != 2 {
					t.Fatal(got, err, calls)
				}
			})
		}
	}
	for _, body := range []string{`{"address_groups":[{"name":"target"}]}`, `{"address_groups":{"id":17,"name":"target"}}`} {
		var calls []nativeGroupCall
		api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeGroupWire(404, "")
			}
			return nativeGroupWire(200, body)
		})
		got, err := api.FindRecord(context.Background(), "target")
		if err != nil || got == nil || len(calls) != 2 {
			t.Fatal(got, err, calls)
		}
	}
	for _, status := range []int{200, 404} {
		var calls []nativeGroupCall
		api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeGroupWire(404, "")
			}
			body := `{"address_groups":[]}`
			if status != 200 {
				body = `{"error":"list denied"}`
			}
			return nativeGroupWire(status, body)
		})
		got, err := api.FindRecord(context.Background(), "target", addressgroups.WithRecordFindIgnoreMissing(false))
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

func TestPythonAddressGroupFindRecordLateDuplicateErrorMarkerAndRawCap(t *testing.T) {
	for _, testcase := range []struct {
		name      string
		second    int
		duplicate bool
	}{
		{"late duplicate", 200, true}, {"late page denied", 500, false},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativeGroupCall
			api := pythonFindAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeGroupWire(404, "")
				}
				if req.URL.Query().Get("marker") == "late" {
					if testcase.duplicate {
						return nativeGroupWire(200, `{"address_groups":[{"id":"second","name":"target"}]}`)
					}
					return nativeGroupWire(testcase.second, `{"error":"late"}`)
				}
				return nativeGroupWire(200, `{"address_groups":[{"id":"first","name":"target"}],"address_groups_links":[{"rel":"next","href":"?marker=late"}]}`)
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
		var calls []nativeGroupCall
		api := pythonFindAPI(t, &calls, func(req *http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeGroupWire(404, "")
			}
			q := req.URL.Query()
			if q.Get("limit") != "2" || q.Get("name") != "target" {
				t.Fatal(q)
			}
			if q.Get("marker") == "7" {
				return nativeGroupWire(200, `{"address_groups":[]}`)
			}
			return nativeGroupWire(200, `{"address_groups":[{"id":7,"name":"target"}]}`)
		})
		got, err := api.FindRecord(context.Background(), "target", addressgroups.WithRecordFindQuery("limit", 2), addressgroups.WithRecordFindQuery("tags", []string{"a", "b"}))
		if err != nil || got == nil || len(calls) != 3 {
			t.Fatal(got, err, calls)
		}
		pythonFindField(t, got, "id", "7")
	})
	t.Run("raw cap occurs before local filtering", func(t *testing.T) {
		var calls []nativeGroupCall
		api := pythonFindAPI(t, &calls, func(req *http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeGroupWire(404, "")
			}
			if req.URL.Query().Get("limit") != "1" {
				t.Fatal(req.URL)
			}
			return nativeGroupWire(200, `{"address_groups":[{"id":"filtered","name":"target","tenant_id":2},{"id":"match","name":"target","tenant_id":1}],"address_groups_links":[{"rel":"next","href":"?marker=late"}]}`)
		})
		got, err := api.FindRecord(context.Background(), "target", addressgroups.WithRecordFindQuery("max_items", 1), addressgroups.WithRecordFindQuery("tenant_id", true))
		if err != nil || got != nil || len(calls) != 2 {
			t.Fatal(got, err, calls)
		}
	})
}

func TestPythonAddressGroupFindRecordOptionSnapshotsAndPreflightGuards(t *testing.T) {
	fields := []string{"id", "name"}
	query := map[string]json.RawMessage{"fields": json.RawMessage(`["id","name"]`), "x_seed": json.RawMessage(`{"initial":true}`)}
	headers := map[string]string{"X-Trace": "original"}
	ignore := false
	options := addressgroups.WithRecordFindOpts(addressgroups.RecordFindOpts{Query: query, Headers: headers, IgnoreMissing: &ignore})
	query["fields"][2] = 'x'
	delete(query, "x_seed")
	headers["X-Trace"] = "changed"
	ignore = true
	fieldOption := addressgroups.WithRecordFindQuery("fields", fields)
	fields[0] = "changed"
	var escaped *addressgroups.RecordFindOpts
	var calls []nativeGroupCall
	api := pythonFindAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Header.Get("X-Trace") != "original" || !reflect.DeepEqual(req.URL.Query()["fields"], []string{"id", "name"}) || req.URL.Query().Get("x_seed") != "initial" {
			t.Fatal(req.Header, req.URL)
		}
		escaped.Query["fields"][2] = 'z'
		escaped.Headers["X-Trace"] = "callback changed"
		return nativeGroupWire(200, `{"address_group":{}}`)
	})
	got, err := api.FindRecord(context.Background(), "target", options, fieldOption, func(c *addressgroups.RecordFindOpts) error { escaped = c; return nil })
	if err != nil || got == nil || len(calls) != 1 {
		t.Fatal(got, err, calls)
	}
	pythonFindField(t, got, "fields", `["id","name"]`)
	for _, testcase := range []struct {
		name    string
		options []addressgroups.RecordFindOption
		cause   error
	}{
		{"nil", []addressgroups.RecordFindOption{nil}, resource.ErrInvalidOption},
		{"malformed JSON", []addressgroups.RecordFindOption{addressgroups.WithRecordFindOpts(addressgroups.RecordFindOpts{Query: map[string]json.RawMessage{"fields": json.RawMessage(`[`)}})}, resource.ErrInvalidOption},
		{"negative cap", []addressgroups.RecordFindOption{addressgroups.WithRecordFindQuery("max_items", -1)}, resource.ErrInvalidOption},
		{"bad page flag", []addressgroups.RecordFindOption{addressgroups.WithRecordFindOpts(addressgroups.RecordFindOpts{Query: map[string]json.RawMessage{"paginated": json.RawMessage(`bad flag`)}})}, resource.ErrInvalidOption},
		{"identity keyword collision", []addressgroups.RecordFindOption{addressgroups.WithRecordFindQuery("id", "different")}, resource.ErrInvalidOption},
		{"route override", []addressgroups.RecordFindOption{addressgroups.WithRecordFindQuery("base_path", "/other")}, resource.ErrInvalidOption},
		{"fetch requires_id control", []addressgroups.RecordFindOption{addressgroups.WithRecordFindQuery("requires_id", false)}, resource.ErrInvalidOption},
		{"fetch error_message control", []addressgroups.RecordFindOption{addressgroups.WithRecordFindQuery("error_message", "changed")}, resource.ErrInvalidOption},
		{"fetch cache control", []addressgroups.RecordFindOption{addressgroups.WithRecordFindQuery("skip_cache", true)}, resource.ErrInvalidOption},
		{"fetch response envelope control", []addressgroups.RecordFindOption{addressgroups.WithRecordFindQuery("resource_response_key", "other")}, resource.ErrInvalidOption},
		{"protected Header", []addressgroups.RecordFindOption{addressgroups.WithRecordFindHeaders(map[string]string{"X-Auth-Token": "different"})}, resource.ErrInvalidOption},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativeGroupCall
			api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("unexpected HTTP"); return nil })
			got, err := api.FindRecord(context.Background(), "target", testcase.options...)
			if got != nil || !errors.Is(err, testcase.cause) || len(calls) != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
	t.Run("source mutation is sticky", func(t *testing.T) {
		var calls []nativeGroupCall
		api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("unexpected HTTP"); return nil })
		got, err := api.FindRecord(context.Background(), "target", func(*addressgroups.RecordFindOpts) error { api.RawClient().ResourceBase += "changed/"; return nil })
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(got, err, calls)
		}
	})
	t.Run("canceled cause prevents HTTP", func(t *testing.T) {
		var calls []nativeGroupCall
		api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("unexpected HTTP"); return nil })
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

func TestPythonAddressGroupFindRecordPhysicalFailuresPreserveActualEvidence(t *testing.T) {
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
			var calls []nativeGroupCall
			payload := `{"error":"physical"}`
			if testcase.status == 200 {
				payload = `{"address_group":{"id":"target"}}`
			}
			api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response {
				if testcase.list && len(calls) == 1 {
					return nativeGroupWire(404, "")
				}
				response := nativeGroupWire(testcase.status, payload)
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
				if !errors.As(err, &native) || native.Actual != testcase.status || string(native.Body) != payload || native.ResponseHeader.Get("X-Request-Id") != "physical" || !strings.Contains(native.URL, "/neutron/v2.0/address-groups") {
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
		var calls []nativeGroupCall
		cause := errors.New("late cancellation")
		ctx, cancel := context.WithCancelCause(context.Background())
		api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response {
			response := nativeGroupWire(200, `{"address_group":{}}`)
			response.Body = &pythonFindFaultBody{data: `{"address_group":{}}`, onClose: func() { cancel(cause) }}
			return response
		})
		got, err := api.FindRecord(ctx, "target")
		if got != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || len(calls) != 1 {
			t.Fatal(got, err, calls)
		}
	})
}

func TestPythonAddressGroupFindRecordRetryCallbacksKeepCleanFallbackAndGuards(t *testing.T) {
	for _, fault := range []bool{false, true} {
		t.Run(map[bool]string{false: "clean callback permits fallback", true: "physical rejection blocks fallback"}[fault], func(t *testing.T) {
			var calls []nativeGroupCall
			readCause := errors.New("rejected read")
			api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					response := nativeGroupWire(404, `{"error":"missing"}`)
					if fault {
						response.Body = &pythonFindFaultBody{data: `{"error":"missing"}`, read: readCause}
					}
					return response
				}
				return nativeGroupWire(200, `{"address_groups":[]}`)
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
		var calls []nativeGroupCall
		api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response {
			response := nativeGroupWire(404, `{"error":"actual rejected retry"}`)
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
			if call.path != "/neutron/v2.0/address-groups/target" {
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

func TestPythonAddressGroupFindRecordMalformedPagesAndDetachedExtensionEvidence(t *testing.T) {
	for _, body := range []string{`{"address_groups":[null]}`, `{"address_groups":[4]}`, `{"address_groups":null}`, `{"address_groups":[]`, `{}`, `{"address_groups":[{"id":"match","name":"target","connection":null}]}`, `{"address_groups":[{"id":"match","name":"target","microversion":null}]}`, `{"address_groups":[{"id":"match","name":"target","_synchronized":null}]}`} {
		t.Run(body, func(t *testing.T) {
			var calls []nativeGroupCall
			api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeGroupWire(404, "")
				}
				response := nativeGroupWire(200, body)
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
	var calls []nativeGroupCall
	api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(202, "not JSON") })
	got, err := api.FindRecord(context.Background(), "target", addressgroups.WithRecordFindQuery("x_extension", []any{nil, 9007199254740993}))
	if err != nil || got == nil || got.Wire != nil || string(got.Envelope) != "not JSON" || len(calls) != 1 {
		t.Fatal(got, err, calls)
	}
	pythonFindField(t, got, "id", `"target"`)
	pythonFindField(t, got, "x_extension", `[null,9007199254740993]`)
}

func TestPythonAddressGroupFindRecordContinuationFormatsAndCycleGuards(t *testing.T) {
	for _, testcase := range []struct{ name, extra, header string }{
		{"RFC Link", ``, `<?marker=next>; rel="next"`},
		{"versioned top next", `,"next":"/v2.0/address-groups?marker=next"`, ``},
		{"dictionary links", `,"links":{"next":"?marker=next"}`, ``},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativeGroupCall
			api := pythonFindAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeGroupWire(404, "")
				}
				if req.URL.Query().Get("name") != "target" {
					t.Fatal(req.URL)
				}
				if req.URL.Query().Get("marker") == "next" {
					return nativeGroupWire(200, `{"address_groups":[]}`)
				}
				response := nativeGroupWire(200, `{"address_groups":[{"id":"first","name":"target"}]`+testcase.extra+`}`)
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
		{"changed collection", `https://other.invalid/address-groups?marker=next`, resource.ErrInvalidOption},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativeGroupCall
			api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeGroupWire(404, "")
				}
				raw, _ := json.Marshal(testcase.next)
				return nativeGroupWire(200, `{"address_groups":[{"id":"first","name":"target"}],"next":`+string(raw)+`}`)
			})
			got, err := api.FindRecord(context.Background(), "target")
			var receipt *resource.ResponseError
			if got != nil || !errors.Is(err, testcase.cause) || !errors.As(err, &receipt) || receipt.StatusCode != 200 || len(calls) != 2 {
				t.Fatal(got, err, receipt, calls)
			}
		})
	}
}

func TestPythonAddressGroupFindRecordLiveReauthenticationAndMicroversionOwnership(t *testing.T) {
	var calls []nativeGroupCall
	var tokens, versions []string
	api := pythonFindAPI(t, &calls, func(req *http.Request) *http.Response {
		tokens = append(tokens, req.Header.Get("X-Auth-Token"))
		versions = append(versions, req.Header.Get("OpenStack-API-Version"))
		if req.Header.Get("X-Trace") != "owned" || req.URL.Query().Get("fields") != "id" {
			t.Fatal(req.Header, req.URL)
		}
		if len(calls) == 1 {
			return nativeGroupWire(401, `{"error":"expired"}`)
		}
		return nativeGroupWire(200, `{"address_group":{"id":"target"}}`)
	})
	source := api.RawClient()
	source.Microversion = "2.0"
	source.MoreHeaders = map[string]string{"X-Trace": "source"}
	reauths := 0
	source.ProviderClient.ReauthFunc = func(context.Context) error { reauths++; source.ProviderClient.SetToken("fresh-token"); return nil }
	version := "2.1"
	option := addressgroups.WithRecordFindOpts(addressgroups.RecordFindOpts{Microversion: &version, Headers: map[string]string{"X-Trace": "owned"}, Query: map[string]json.RawMessage{"fields": json.RawMessage(`"id"`)}})
	version = "changed"
	got, err := api.FindRecord(context.Background(), "target", option)
	if err != nil || got == nil || len(calls) != 2 || reauths != 1 || !reflect.DeepEqual(tokens, []string{"test-token", "fresh-token"}) || !reflect.DeepEqual(versions, []string{"network 2.1", "network 2.1"}) || source.Microversion != "2.0" || source.MoreHeaders["X-Trace"] != "source" {
		t.Fatal(got, err, calls, reauths, tokens, versions, source)
	}
	var none []nativeGroupCall
	invalid := pythonFindAPI(t, &none, func(*http.Request) *http.Response { t.Fatal("invalid version HTTP"); return nil })
	got, err = invalid.FindRecord(context.Background(), "target", addressgroups.WithRecordFindMicroversion("2.1\r\nchanged"))
	if got != nil || !errors.Is(err, resource.ErrInvalidOption) || len(none) != 0 {
		t.Fatal(got, err, none)
	}
}

func TestPythonAddressGroupFindRecordDictionaryTruthinessShortCircuitAndFailureReceipts(t *testing.T) {
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
			var calls []nativeGroupCall
			body := `{"address_groups":[{"id":"match","name":"target","tenant_id":` + testcase.actual + `}]}`
			api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeGroupWire(404, "")
				}
				response := nativeGroupWire(203, body)
				response.Header.Set("X-Request-Id", "dict-filter")
				return response
			})
			got, err := api.FindRecord(context.Background(), "target", addressgroups.WithRecordFindQuery("tenant_id", json.RawMessage(testcase.filter)))
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
}

func TestPythonAddressGroupFindRecordPaginationTruthiness(t *testing.T) {
	for _, testcase := range []struct {
		name  string
		flag  any
		pages int
	}{
		{"null", nil, 1}, {"zero", 0, 1}, {"empty list", []any{}, 1}, {"nonzero", 1, 2}, {"nonempty string", "False", 2},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			var calls []nativeGroupCall
			api := pythonFindAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeGroupWire(404, "")
				}
				if req.URL.Query().Get("marker") == "next" {
					return nativeGroupWire(200, `{"address_groups":[]}`)
				}
				return nativeGroupWire(200, `{"address_groups":[{"id":"match","name":"target"}],"next":"?marker=next"}`)
			})
			got, err := api.FindRecord(context.Background(), "target", addressgroups.WithRecordFindQuery("paginated", testcase.flag))
			if err != nil || got == nil || len(calls) != testcase.pages+1 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestPythonAddressGroupFindRecordListAcceptAndRetryVersionPolicy(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(map[bool]string{false: "Source LIST default Accept overrides configured Header", true: "explicit caller LIST Accept wins"}[explicit], func(t *testing.T) {
			var calls []nativeGroupCall
			var accepts []string
			api := pythonFindAPI(t, &calls, func(req *http.Request) *http.Response {
				accepts = append(accepts, req.Header.Get("Accept"))
				if len(calls) == 1 {
					return nativeGroupWire(404, "")
				}
				return nativeGroupWire(200, `{"address_groups":[]}`)
			})
			api.RawClient().MoreHeaders = map[string]string{"Accept": "text/xml"}
			options := []addressgroups.RecordFindOption{}
			want := []string{"text/xml", "application/json"}
			if explicit {
				options = append(options, addressgroups.WithRecordFindHeaders(map[string]string{"Accept": "text/plain"}))
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
			var calls []nativeGroupCall
			body := `{"error":"original"}`
			api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response {
				response := nativeGroupWire(404, body)
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
			options := []addressgroups.RecordFindOption{}
			if version != "" {
				options = append(options, addressgroups.WithRecordFindMicroversion(version))
			}
			got, err := api.FindRecord(context.Background(), "target", options...)
			var native gophercloud.ErrUnexpectedResponseCode
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "version-guard" || len(calls) != 1 {
				t.Fatal(got, err, native, calls)
			}
		})
	}
}

func TestPythonAddressGroupFindRecordAllStatusAndPhysicalRejectionContracts(t *testing.T) {
	for status := 200; status < 600; status++ {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls []nativeGroupCall
			api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeGroupWire(status, `{"address_group":{"id":"target"}}`)
				}
				return nativeGroupWire(200, `{"address_groups":[]}`)
			})
			got, err := api.FindRecord(context.Background(), "target")
			if status < 400 {
				if err != nil || got == nil || len(calls) != 1 {
					t.Fatal(got, err, calls)
				}
			} else if status == 400 || status == 403 || status == 404 {
				if got != nil || err != nil || len(calls) != 2 {
					t.Fatal(got, err, calls)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if got != nil || !errors.As(err, &native) || native.Actual != status || len(calls) != 1 {
					t.Fatal(got, err, native, calls)
				}
			}
		})
	}
	for status := 400; status < 600; status++ {
		t.Run("physical/"+fmt.Sprint(status), func(t *testing.T) {
			var calls []nativeGroupCall
			readCause, closeCause := errors.New("read cause"), errors.New("close cause")
			body := `{"error":"actual rejected body"}`
			api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response {
				r := nativeGroupWire(status, body)
				r.Header.Set("X-Request-Id", "rejected-proof")
				r.Body = &pythonFindFaultBody{data: body, read: readCause, close: closeCause}
				return r
			})
			got, err := api.FindRecord(context.Background(), "target")
			var native gophercloud.ErrUnexpectedResponseCode
			if got != nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.As(err, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "rejected-proof" || len(calls) != 1 {
				t.Fatal(got, err, native, calls)
			}
		})
	}
}
func TestPythonAddressGroupFindRecordLocalFilterDescriptorAndAliasOrder(t *testing.T) {
	for _, testcase := range []struct {
		actual, filter string
		match          bool
	}{{`1`, `[true]`, true}, {`false`, `[0]`, true}, {`"one"`, `"one"`, false}, {`null`, `null`, true}, {`[]`, `[]`, true}} {
		var calls []nativeGroupCall
		api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response {
			if len(calls) == 1 {
				return nativeGroupWire(404, "")
			}
			return nativeGroupWire(200, `{"address_groups":[{"id":"match","name":"target","addresses":`+testcase.actual+`}]} `)
		})
		got, err := api.FindRecord(context.Background(), "target", addressgroups.WithRecordFindQuery("addresses", json.RawMessage(testcase.filter)))
		if err != nil || (got != nil) != testcase.match || len(calls) != 2 {
			t.Fatal(testcase, got, err, calls)
		}
	}
}

func TestPythonAddressGroupFindRecordFallbackUsesSelectedVersionAndSourceReceipt(t *testing.T) {
	for _, selected := range []string{"", "2.0"} {
		var calls []nativeGroupCall
		var versions []string
		api := pythonFindAPI(t, &calls, func(req *http.Request) *http.Response {
			versions = append(versions, req.Header.Get("OpenStack-API-Version"))
			if len(calls) == 1 {
				return nativeGroupWire(404, `{"error":"fallback"}`)
			}
			return nativeGroupWire(200, `{"address_groups":[{"id":"match","name":"target"}]}`)
		})
		source := api.RawClient()
		source.Microversion = selected
		source.MoreHeaders = map[string]string{"Accept": "text/xml"}
		if selected != "" {
			source.MoreHeaders["OpenStack-API-Version"] = "network " + selected
		}
		got, err := api.FindRecord(context.Background(), "target", addressgroups.WithRecordFindMicroversion("2.1"))
		wantList := ""
		if selected != "" {
			wantList = "network " + selected
		}
		if err != nil || got == nil || len(calls) != 2 || !reflect.DeepEqual(versions, []string{"network 2.1", wantList}) || source.Microversion != selected || source.MoreHeaders["Accept"] != "text/xml" {
			t.Fatal(got, err, calls, versions, source)
		}
	}
}
func TestPythonAddressGroupFindRecordSourceCancellationAndConstructorPreflight(t *testing.T) {
	for _, status := range []int{200, 400, 403, 404, 500} {
		for _, fault := range []string{"cancel", "source"} {
			t.Run(fmt.Sprintf("%d/%s", status, fault), func(t *testing.T) {
				var calls []nativeGroupCall
				var source *gophercloud.ServiceClient
				body := `{"address_group":{}}`
				cause := errors.New("caller physical cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response {
					r := nativeGroupWire(status, body)
					r.Header.Set("X-Request-Id", "actual-cause")
					r.Body = &pythonFindFaultBody{data: body, onClose: func() {
						if fault == "cancel" {
							cancel(cause)
						} else {
							source.ResourceBase += "changed/"
						}
					}}
					return r
				})
				source = api.RawClient()
				original := source.ResourceBase
				defer func() { source.ResourceBase = original }()
				got, err := api.FindRecord(ctx, "target")
				if got != nil || len(calls) != 1 {
					t.Fatal(got, err, calls)
				}
				if fault == "cancel" {
					if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if status >= 400 {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "actual-cause" {
						t.Fatal(err, native)
					}
				} else {
					var receipt *resource.ResponseError
					if !errors.As(err, &receipt) || receipt.StatusCode != status || string(receipt.Body) != body || receipt.Header.Get("X-Request-Id") != "actual-cause" {
						t.Fatal(err, receipt)
					}
				}
			})
		}
	}
	var calls []nativeGroupCall
	api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("preflight made HTTP"); return nil })
	for _, key := range []string{"connection", "session", "list_base_path", "microversion", "_synchronized", "resource_type", "self", "location", "__conflicting_attrs"} {
		got, err := api.FindRecord(context.Background(), "target", addressgroups.WithRecordFindQuery(key, nil))
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(key, got, err, calls)
		}
	}
	for _, identity := range []string{"", " ", ".", "..", "bad\n"} {
		got, err := api.FindRecord(context.Background(), identity)
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(identity, got, err, calls)
		}
	}
	got, err := api.FindRecord(nil, "target")
	if got != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(got, err)
	}
	var nilAPI *addressgroups.API
	got, err = nilAPI.FindRecord(context.Background(), "target")
	if got != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(got, err)
	}
}

func TestPythonAddressGroupFindRecordFloatTruthinessKeepsWireAndExactComparisonBoundary(t *testing.T) {
	for _, testcase := range []struct {
		name, raw string
		truth     bool
	}{
		{"positive underflow", `1e-4000`, false}, {"negative underflow", `-1e-4000`, false},
		{"signed floating zero", `-0.0`, false}, {"integer zero", `0`, false},
		{"positive overflow", `1e4000`, true}, {"negative overflow", `-1e4000`, true},
		{"representable subnormal", `5e-324`, true}, {"integer beyond float precision", `9007199254740993`, true},
	} {
		t.Run(testcase.name+" local empty dictionary", func(t *testing.T) {
			var calls []nativeGroupCall
			body := `{"address_groups":[{"id":"match","name":"target","tenant_id":` + testcase.raw + `}]}`
			api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeGroupWire(404, "")
				}
				return nativeGroupWire(200, body)
			})
			got, err := api.FindRecord(context.Background(), "target", addressgroups.WithRecordFindQuery("tenant_id", map[string]any{}))
			if err != nil || (got != nil) != testcase.truth || len(calls) != 2 {
				t.Fatal(testcase, got, err, calls)
			}
			if got != nil {
				pythonFindField(t, got, "tenant_id", testcase.raw)
				pythonFindWireField(t, got, "tenant_id", testcase.raw)
				if string(got.Envelope) != body {
					t.Fatal(got)
				}
			}
		})
		t.Run(testcase.name+" paginated scalar", func(t *testing.T) {
			var calls []nativeGroupCall
			api := pythonFindAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeGroupWire(404, "")
				}
				if req.URL.Query().Get("marker") == "next" {
					return nativeGroupWire(200, `{"address_groups":[]}`)
				}
				return nativeGroupWire(200, `{"address_groups":[{"id":"match","name":"target"}],"next":"?marker=next"}`)
			})
			got, err := api.FindRecord(context.Background(), "target", addressgroups.WithRecordFindQuery("paginated", json.RawMessage(testcase.raw)))
			expected := 2
			if testcase.truth {
				expected = 3
			}
			if err != nil || got == nil || len(calls) != expected {
				t.Fatal(testcase, got, err, calls)
			}
		})
	}
	// Numeric equality deliberately preserves exact decimal JSON instead of
	// applying CPython's float conversion. This difference is documented.
	for _, testcase := range []struct {
		actual, filter string
		match          bool
	}{
		{`1e-4000`, `0`, false}, {`1e-4000`, `1e-4000`, true}, {`9007199254740993`, `9007199254740992.0`, false}, {`9007199254740993.0`, `9007199254740993`, true}, {`1e4000`, `2e4000`, false}, {`1`, `true`, true},
	} {
		t.Run("exact comparison "+testcase.actual+" vs "+testcase.filter, func(t *testing.T) {
			var calls []nativeGroupCall
			api := pythonFindAPI(t, &calls, func(*http.Request) *http.Response {
				if len(calls) == 1 {
					return nativeGroupWire(404, "")
				}
				return nativeGroupWire(200, `{"address_groups":[{"id":"match","name":"target","tenant_id":`+testcase.actual+`}]}`)
			})
			got, err := api.FindRecord(context.Background(), "target", addressgroups.WithRecordFindQuery("tenant_id", json.RawMessage(testcase.filter)))
			if err != nil || (got != nil) != testcase.match || len(calls) != 2 {
				t.Fatal(testcase, got, err, calls)
			}
		})
	}
}
