package routers_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonCoreRouterAPI(t *testing.T, calls *[]nativeRouterCall, reply func(*http.Request) *http.Response) *routers.API {
	t.Helper()
	cloud := testcloud.New(t)
	nativeRouterRecorder(cloud, calls, reply)
	return routers.New(nativeRouterClient(cloud))
}

// Expectations come from the pinned Proxy._create/_update, Resource.new and
// NetworkResource mappings. They intentionally differ from native builders.
func TestPythonCoreRouterMutationBodiesDefaultsAndAliases(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for _, test := range []struct {
			name    string
			options []routers.AttributeOption
			want    map[string]any
		}{
			{"omitted", nil, map[string]any{}},
			{"explicit empty Body values", []routers.AttributeOption{routers.WithAttribute("is_admin_state_up", false), routers.WithAttribute("is_distributed", false), routers.WithAttribute("is_ha", false), routers.WithAttribute("external_gateway_info", nil), routers.WithAttribute("revision_number", 0)}, map[string]any{"admin_state_up": false, "distributed": false, "ha": false, "external_gateway_info": nil, "revision": float64(0)}},
			{"null and unknown query-only Body values", []routers.AttributeOption{routers.WithAttribute("x_extension", nil), routers.WithAttribute("fields", []string{"id"})}, map[string]any{"x_extension": nil, "fields": []any{"id"}}},
		} {
			t.Run(method+"/"+test.name, func(t *testing.T) {
				var calls []nativeRouterCall
				api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response { return nativeRouterWire(201, `{"router":`+nativeRouterRow+`}`) })
				var value *routers.Router
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
				path := "/neutron/v2.0/routers"
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
				if want := map[string]any{"router": test.want}; !reflect.DeepEqual(body, want) {
					t.Fatalf("got %#v, want %#v", body, want)
				}
			})
		}
	}
}

func TestPythonCoreRouterRevisionHeaderOnlyPriorityAndClientIsolation(t *testing.T) {
	for _, test := range []struct {
		name, header string
		options      []routers.AttributeOption
	}{
		{"revision zero only commits", "revision_number=0", []routers.AttributeOption{routers.WithRevision(0)}},
		{"revision overrides later raw header", "revision_number=0", []routers.AttributeOption{routers.WithRevision(0), routers.WithIfMatch("raw")}},
		{"revision overrides earlier raw header", "revision_number=0", []routers.AttributeOption{routers.WithAttribute("if_match", "raw"), routers.WithRevision(0)}},
		{"revision discards later invalid raw Header", "revision_number=0", []routers.AttributeOption{routers.WithRevision(0), routers.WithIfMatch("bad\r\nheader")}},
		{"revision discards earlier invalid raw Header list", "revision_number=0", []routers.AttributeOption{routers.WithAttribute("if_match", []int{1}), routers.WithRevision(0)}},
		{"revision discards null raw Header element", "revision_number=0", []routers.AttributeOption{routers.WithAttribute("if_match", []any{"a", nil}), routers.WithRevision(0)}},
		{"revision discards unserializable raw Header", "revision_number=0", []routers.AttributeOption{routers.WithAttribute("if_match", make(chan int)), routers.WithRevision(0)}},
		{"last revision wins", "revision_number=-2", []routers.AttributeOption{routers.WithRevision(0), routers.WithRevision(-2)}},
		{"large revision stays exact", "revision_number=9223372036854775807", []routers.AttributeOption{routers.WithRevision(9223372036854775807)}},
		{"Header list", "a, b", []routers.AttributeOption{routers.WithAttribute("if_match", []string{"a", "b"})}},
		{"wire Header name", "wire", []routers.AttributeOption{routers.WithAttribute("If-Match", "wire")}},
		{"explicit empty list", "", []routers.AttributeOption{routers.WithIfMatch()}},
		{"Source null Header", "None", []routers.AttributeOption{routers.WithAttribute("if_match", nil)}},
		{"bodyless 204 commits", "revision_number=0", []routers.AttributeOption{routers.WithRevision(0)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []nativeRouterCall
			var headers []string
			api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
				headers = append(headers, req.Header.Get("If-Match"))
				if test.name == "bodyless 204 commits" {
					return nativeRouterWire(204, "")
				}
				return nativeRouterWire(200, `{"router":`+nativeRouterRow+`}`)
			})
			configured := map[string]string{"if-match": "old", "If-Match": "other", "X-Trace": "trace"}
			api.RawClient().MoreHeaders = configured
			value, err := api.UpdateWithAttributes(context.Background(), "literal.id", test.options...)
			if err != nil {
				t.Fatal(err)
			}
			if test.name == "bodyless 204 commits" {
				if value != nil {
					t.Fatal(value)
				}
			} else if value == nil || value.ID != "id-1" {
				t.Fatal(value)
			}
			if len(calls) != 1 || calls[0] != (nativeRouterCall{http.MethodPut, "/neutron/v2.0/routers/literal.id", "", `{"router":{}}`}) || !reflect.DeepEqual(headers, []string{test.header}) {
				t.Fatal(calls, headers)
			}
			if !reflect.DeepEqual(configured, map[string]string{"if-match": "old", "If-Match": "other", "X-Trace": "trace"}) {
				t.Fatal(configured)
			}
		})
	}
	t.Run("configured header does not turn empty update into commit", func(t *testing.T) {
		var calls []nativeRouterCall
		api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.Header.Get("If-Match") != "configured" || req.Header.Get("X-Trace") != "trace" {
				t.Fatal(req.Header)
			}
			return nativeRouterWire(200, `{"router":`+nativeRouterRow+`}`)
		})
		api.RawClient().MoreHeaders = map[string]string{"if-match": "configured", "X-Trace": "trace"}
		value, err := api.UpdateWithAttributes(context.Background(), "literal.id")
		if err != nil || value == nil || value.ID != "literal.id" || len(calls) != 0 {
			t.Fatal(value, err, calls)
		}
		if _, err := api.UpdateWithAttributes(context.Background(), "literal.id", routers.WithAttribute("name", "")); err != nil || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
}

func TestPythonCoreRouterOptionSnapshotsHeaderRoutingAndPreflight(t *testing.T) {
	var calls []nativeRouterCall
	var headers []string
	api := pythonCoreRouterAPI(t, &calls, func(req *http.Request) *http.Response {
		headers = append(headers, req.Header.Get("If-Match"))
		return nativeRouterWire(201, `{"router":`+nativeRouterRow+`}`)
	})
	values, conditions := []string{"original"}, []string{"first", "second"}
	bodyOption, headerOption := routers.WithAttribute("x_extension", values), routers.WithAttribute("if_match", conditions)
	values[0], conditions[0] = "mutated", "mutated"
	for i := 0; i < 2; i++ {
		if _, err := api.CreateWithAttributes(context.Background(), bodyOption, headerOption, routers.WithAttribute("name", "first"), routers.WithAttribute("name", "last")); err != nil {
			t.Fatal(err)
		}
	}
	for _, call := range calls {
		var body map[string]map[string]any
		if err := json.Unmarshal([]byte(call.body), &body); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(body["router"], map[string]any{"x_extension": []any{"original"}, "name": "last"}) {
			t.Fatal(body)
		}
	}
	if !reflect.DeepEqual(headers, []string{"first, second", "first, second"}) {
		t.Fatal(headers)
	}
	calls = nil
	for _, options := range [][]routers.AttributeOption{
		{nil}, {routers.WithAttribute(" ", 1)}, {routers.WithAttribute("x_extension", make(chan int))},
		{routers.WithAttribute("microversion", "2.0")}, {routers.WithAttribute("base_path", "/other")},
		{routers.WithAttribute("if_match", []int{1})}, {routers.WithAttribute("if_match", []any{"a", nil})}, {routers.WithIfMatch("bad\r\nheader")},
		{request.WithHeader[struct{}]("Authorization", "replaced")},
		{request.WithField[struct{}]("If-Match", "body")},
		{request.WithQuery[struct{}]("fields", "id")}, {routers.WithRevision(0)},
	} {
		_, err := api.CreateWithAttributes(context.Background(), options...)
		nativeRouterOperation(t, err, "CreateWithAttributes")
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	for _, options := range [][]routers.AttributeOption{
		{routers.WithAttribute("id", "other")}, {routers.WithAttribute("if_revision", 0)},
		{request.WithArgument[struct{}]("if_revision", "0")},
	} {
		_, err := api.UpdateWithAttributes(context.Background(), "literal.id", options...)
		nativeRouterOperation(t, err, "UpdateWithAttributes")
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}

func TestPythonCoreRouterMutationStatusesDecodeAndContextCauses(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for _, code := range []int{200, 201, 202, 204, 299, 300, 399, 400, 404, 409, 500} {
			t.Run(fmt.Sprintf("%s/%d", method, code), func(t *testing.T) {
				var calls []nativeRouterCall
				body := `{"router":` + nativeRouterRow + `}`
				api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
					reply := nativeRouterWire(code, body)
					reply.Header.Set("X-Request-Id", "core-mutation")
					return reply
				})
				var value *routers.Router
				var err error
				operation := "CreateWithAttributes"
				if method == http.MethodPost {
					value, err = api.CreateWithAttributes(context.Background())
				} else {
					operation = "UpdateWithAttributes"
					value, err = api.UpdateWithAttributes(context.Background(), "literal.id", routers.WithRevision(0))
				}
				switch {
				case code == 204:
					if err != nil || value != nil {
						t.Fatal(value, err)
					}
				case code < 400:
					if err != nil || value == nil || value.ID != "id-1" {
						t.Fatal(value, err)
					}
				default:
					nativeRouterOperation(t, err, operation)
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
				var calls []nativeRouterCall
				api := pythonCoreRouterAPI(t, &calls, func(*http.Request) *http.Response {
					if lane == "response boundary" {
						cancel(cause)
					}
					if lane == "decode type" {
						return nativeRouterWire(200, `{"router":[]}`)
					}
					if lane == "bodyless JSON" {
						return nativeRouterWire(200, "")
					}
					return nativeRouterWire(200, `{"router":`+nativeRouterRow+`}`)
				})
				var options []routers.AttributeOption
				switch lane {
				case "pre-canceled":
					cancel(cause)
				case "empty option callback":
					options = []routers.AttributeOption{func(*request.Config[struct{}]) error { cancel(cause); return nil }}
				default:
					options = []routers.AttributeOption{routers.WithAttribute("name", "")}
				}
				var err error
				operation := "CreateWithAttributes"
				if method == http.MethodPost {
					_, err = api.CreateWithAttributes(ctx, options...)
				} else {
					operation = "UpdateWithAttributes"
					_, err = api.UpdateWithAttributes(ctx, "literal.id", options...)
				}
				nativeRouterOperation(t, err, operation)
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
