package floatingips_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonCoreFloatingIPAPI(t *testing.T, calls *[]nativeFloatingIPCall, reply func(*http.Request) *http.Response) *floatingips.API {
	t.Helper()
	cloud := testcloud.New(t)
	nativeFloatingIPRecorder(cloud, calls, reply)
	return floatingips.New(nativeFloatingIPClient(cloud))
}

// Expectations come from the pinned Proxy._create/_update, Resource.new and
// NetworkResource mappings. They intentionally differ from native builders.
func TestPythonCoreFloatingIPMutationBodiesDefaultsAndAliases(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for _, test := range []struct {
			name    string
			options []floatingips.AttributeOption
			want    map[string]any
		}{
			{"omitted", nil, map[string]any{}},
			{"explicit empty Body values", []floatingips.AttributeOption{floatingips.WithAttribute("floating_network_id", ""), floatingips.WithAttribute("port_id", nil), floatingips.WithAttribute("floating_ip_address", ""), floatingips.WithAttribute("fixed_ip_address", ""), floatingips.WithAttribute("dns_name", "")}, map[string]any{"floating_network_id": "", "port_id": nil, "floating_ip_address": "", "fixed_ip_address": "", "dns_name": ""}},
			{"null and unknown query-only Body values", []floatingips.AttributeOption{floatingips.WithAttribute("x_extension", nil), floatingips.WithAttribute("fields", []string{"id"})}, map[string]any{"x_extension": nil, "fields": []any{"id"}}},
		} {
			t.Run(method+"/"+test.name, func(t *testing.T) {
				var calls []nativeFloatingIPCall
				api := pythonCoreFloatingIPAPI(t, &calls, func(*http.Request) *http.Response {
					return nativeFloatingIPWire(201, `{"floatingip":`+nativeFloatingIPRow+`}`)
				})
				var value *floatingips.FloatingIP
				var err error
				if method == http.MethodPost {
					value, err = api.CreateWithAttributes(context.Background(), test.options...)
				} else {
					value, err = api.UpdateWithAttributes(context.Background(), "literal.id", test.options...)
				}
				if method == http.MethodPut && test.name == "omitted" {
					if err != nil || value == nil || value.ID != "literal.id" || len(calls) != 0 {
						t.Fatal(value, err, calls)
					}
					return
				}
				if err != nil || value == nil || value.ID != "id-1" {
					t.Fatal(value, err)
				}
				path := "/neutron/v2.0/floatingips"
				if method == http.MethodPut {
					path += "/literal.id"
				}
				if len(calls) != 1 || calls[0].method != method || calls[0].path != path || calls[0].query != "" {
					t.Fatal(calls)
				}
				var body map[string]any
				if err := json.Unmarshal([]byte(calls[0].body), &body); err != nil {
					t.Fatal(err)
				}
				if want := map[string]any{"floatingip": test.want}; !reflect.DeepEqual(body, want) {
					t.Fatalf("got %#v, want %#v", body, want)
				}
			})
		}
	}
}

func TestPythonCoreFloatingIPRevisionHeaderOnlyPriorityAndClientIsolation(t *testing.T) {
	for _, test := range []struct {
		name, header string
		options      []floatingips.AttributeOption
	}{
		{"revision zero only commits", "revision_number=0", []floatingips.AttributeOption{floatingips.WithRevision(0)}},
		{"revision overrides later raw header", "revision_number=0", []floatingips.AttributeOption{floatingips.WithRevision(0), floatingips.WithIfMatch("raw")}},
		{"revision overrides earlier raw header", "revision_number=0", []floatingips.AttributeOption{floatingips.WithAttribute("if_match", "raw"), floatingips.WithRevision(0)}},
		{"revision discards later invalid raw Header", "revision_number=0", []floatingips.AttributeOption{floatingips.WithRevision(0), floatingips.WithIfMatch("bad\r\nheader")}},
		{"revision discards earlier invalid raw Header list", "revision_number=0", []floatingips.AttributeOption{floatingips.WithAttribute("if_match", []int{1}), floatingips.WithRevision(0)}},
		{"revision discards null raw Header element", "revision_number=0", []floatingips.AttributeOption{floatingips.WithAttribute("if_match", []any{"a", nil}), floatingips.WithRevision(0)}},
		{"revision discards unserializable raw Header", "revision_number=0", []floatingips.AttributeOption{floatingips.WithAttribute("if_match", make(chan int)), floatingips.WithRevision(0)}},
		{"last revision wins", "revision_number=-2", []floatingips.AttributeOption{floatingips.WithRevision(0), floatingips.WithRevision(-2)}},
		{"large revision stays exact", "revision_number=9223372036854775807", []floatingips.AttributeOption{floatingips.WithRevision(9223372036854775807)}},
		{"Header list", "a, b", []floatingips.AttributeOption{floatingips.WithAttribute("if_match", []string{"a", "b"})}},
		{"wire Header name", "wire", []floatingips.AttributeOption{floatingips.WithAttribute("If-Match", "wire")}},
		{"explicit empty list", "", []floatingips.AttributeOption{floatingips.WithIfMatch()}},
		{"Source null Header", "None", []floatingips.AttributeOption{floatingips.WithAttribute("if_match", nil)}},
		{"bodyless 204 commits", "revision_number=0", []floatingips.AttributeOption{floatingips.WithRevision(0)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []nativeFloatingIPCall
			var headers []string
			api := pythonCoreFloatingIPAPI(t, &calls, func(req *http.Request) *http.Response {
				headers = append(headers, req.Header.Get("If-Match"))
				if test.name == "bodyless 204 commits" {
					return nativeFloatingIPWire(204, "")
				}
				return nativeFloatingIPWire(200, `{"floatingip":`+nativeFloatingIPRow+`}`)
			})
			configured := map[string]string{"if-match": "old", "If-Match": "other", "X-Trace": "trace"}
			api.RawClient().MoreHeaders = configured
			value, err := api.UpdateWithAttributes(context.Background(), "literal.id", test.options...)
			if err != nil {
				t.Fatal(err)
			}
			if test.name == "bodyless 204 commits" {
				if !reflect.DeepEqual(value, &floatingips.FloatingIP{}) {
					t.Fatal(value)
				}
			} else if value == nil || value.ID != "id-1" {
				t.Fatal(value)
			}
			if len(calls) != 1 || calls[0] != (nativeFloatingIPCall{http.MethodPut, "/neutron/v2.0/floatingips/literal.id", "", `{"floatingip":{}}`}) || !reflect.DeepEqual(headers, []string{test.header}) {
				t.Fatal(calls, headers)
			}
			if !reflect.DeepEqual(configured, map[string]string{"if-match": "old", "If-Match": "other", "X-Trace": "trace"}) {
				t.Fatal(configured)
			}
		})
	}
	t.Run("configured header does not turn empty update into commit", func(t *testing.T) {
		var calls []nativeFloatingIPCall
		api := pythonCoreFloatingIPAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.Header.Get("If-Match") != "configured" || req.Header.Get("X-Trace") != "trace" {
				t.Fatal(req.Header)
			}
			return nativeFloatingIPWire(200, `{"floatingip":`+nativeFloatingIPRow+`}`)
		})
		api.RawClient().MoreHeaders = map[string]string{"if-match": "configured", "X-Trace": "trace"}
		value, err := api.UpdateWithAttributes(context.Background(), "literal.id")
		if err != nil || value == nil || value.ID != "literal.id" || len(calls) != 0 {
			t.Fatal(value, err, calls)
		}
		if _, err := api.UpdateWithAttributes(context.Background(), "literal.id", floatingips.WithAttribute("name", "")); err != nil || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
}

func TestPythonCoreFloatingIPOptionSnapshotsHeaderRoutingAndPreflight(t *testing.T) {
	var calls []nativeFloatingIPCall
	var headers []string
	api := pythonCoreFloatingIPAPI(t, &calls, func(req *http.Request) *http.Response {
		headers = append(headers, req.Header.Get("If-Match"))
		return nativeFloatingIPWire(201, `{"floatingip":`+nativeFloatingIPRow+`}`)
	})
	values, conditions := []string{"original"}, []string{"first", "second"}
	bodyOption, headerOption := floatingips.WithAttribute("x_extension", values), floatingips.WithAttribute("if_match", conditions)
	values[0], conditions[0] = "mutated", "mutated"
	for i := 0; i < 2; i++ {
		if _, err := api.CreateWithAttributes(context.Background(), bodyOption, headerOption, floatingips.WithAttribute("name", "first"), floatingips.WithAttribute("name", "last")); err != nil {
			t.Fatal(err)
		}
	}
	for _, call := range calls {
		var body map[string]map[string]any
		if err := json.Unmarshal([]byte(call.body), &body); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(body["floatingip"], map[string]any{"x_extension": []any{"original"}, "name": "last"}) {
			t.Fatal(body)
		}
	}
	if !reflect.DeepEqual(headers, []string{"first, second", "first, second"}) {
		t.Fatal(headers)
	}
	calls = nil
	for _, options := range [][]floatingips.AttributeOption{
		{nil}, {floatingips.WithAttribute(" ", 1)}, {floatingips.WithAttribute("x_extension", make(chan int))},
		{floatingips.WithAttribute("microversion", "2.0")}, {floatingips.WithAttribute("base_path", "/other")},
		{floatingips.WithAttribute("if_match", []int{1})}, {floatingips.WithAttribute("if_match", []any{"a", nil})}, {floatingips.WithIfMatch("bad\r\nheader")},
		{request.WithHeader[struct{}]("Authorization", "replaced")},
		{request.WithField[struct{}]("If-Match", "body")},
		{request.WithQuery[struct{}]("fields", "id")}, {floatingips.WithRevision(0)},
	} {
		_, err := api.CreateWithAttributes(context.Background(), options...)
		nativeFloatingIPOperation(t, err, "CreateWithAttributes")
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	for _, options := range [][]floatingips.AttributeOption{
		{floatingips.WithAttribute("id", "other")}, {floatingips.WithAttribute("if_revision", 0)},
		{request.WithArgument[struct{}]("if_revision", "0")},
	} {
		_, err := api.UpdateWithAttributes(context.Background(), "literal.id", options...)
		nativeFloatingIPOperation(t, err, "UpdateWithAttributes")
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}

func TestPythonCoreFloatingIPMutationStatusesDecodeAndContextCauses(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for _, code := range []int{200, 201, 202, 204, 299, 300, 399, 400, 404, 409, 500} {
			t.Run(fmt.Sprintf("%s/%d", method, code), func(t *testing.T) {
				var calls []nativeFloatingIPCall
				body := `{"floatingip":` + nativeFloatingIPRow + `}`
				api := pythonCoreFloatingIPAPI(t, &calls, func(*http.Request) *http.Response {
					reply := nativeFloatingIPWire(code, body)
					reply.Header.Set("X-Request-Id", "core-mutation")
					return reply
				})
				var value *floatingips.FloatingIP
				var err error
				operation := "CreateWithAttributes"
				if method == http.MethodPost {
					value, err = api.CreateWithAttributes(context.Background())
				} else {
					operation = "UpdateWithAttributes"
					value, err = api.UpdateWithAttributes(context.Background(), "literal.id", floatingips.WithRevision(0))
				}
				switch {
				case code == 204:
					if err != nil || !reflect.DeepEqual(value, &floatingips.FloatingIP{}) {
						t.Fatal(value, err)
					}
				case code < 400:
					if err != nil || value == nil || value.ID != "id-1" {
						t.Fatal(value, err)
					}
				default:
					nativeFloatingIPOperation(t, err, operation)
					var status gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &status) || status.Actual != code || string(status.Body) != body || status.ResponseHeader.Get("X-Request-Id") != "core-mutation" {
						t.Fatal(err, status)
					}
				}
				if len(calls) != 1 {
					t.Fatal(calls)
				}
			})
		}
		for _, lane := range []string{"pre-canceled", "empty option callback", "response boundary", "decode type", "bodyless JSON"} {
			t.Run(method+"/"+lane, func(t *testing.T) {
				cause := errors.New("caller stopped mutation")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				var calls []nativeFloatingIPCall
				api := pythonCoreFloatingIPAPI(t, &calls, func(*http.Request) *http.Response {
					if lane == "response boundary" {
						cancel(cause)
					}
					if lane == "decode type" {
						return nativeFloatingIPWire(200, `{"floatingip":[]}`)
					}
					if lane == "bodyless JSON" {
						return nativeFloatingIPWire(200, "")
					}
					return nativeFloatingIPWire(200, `{"floatingip":`+nativeFloatingIPRow+`}`)
				})
				var options []floatingips.AttributeOption
				switch lane {
				case "pre-canceled":
					cancel(cause)
				case "empty option callback":
					options = []floatingips.AttributeOption{func(*request.Config[struct{}]) error { cancel(cause); return nil }}
				default:
					options = []floatingips.AttributeOption{floatingips.WithAttribute("name", "")}
				}
				var err error
				operation := "CreateWithAttributes"
				if method == http.MethodPost {
					_, err = api.CreateWithAttributes(ctx, options...)
				} else {
					operation = "UpdateWithAttributes"
					_, err = api.UpdateWithAttributes(ctx, "literal.id", options...)
				}
				nativeFloatingIPOperation(t, err, operation)
				wantCalls := 1
				switch lane {
				case "decode type":
					var decode *json.UnmarshalTypeError
					if !errors.As(err, &decode) {
						t.Fatal(err)
					}
				case "bodyless JSON":
					if !errors.Is(err, io.EOF) {
						t.Fatal(err)
					}
				default:
					if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
						t.Fatal(err)
					}
					if lane != "response boundary" {
						wantCalls = 0
					}
				}
				if len(calls) != wantCalls {
					t.Fatal(calls)
				}
			})
		}
	}
}
