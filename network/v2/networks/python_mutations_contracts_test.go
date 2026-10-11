package networks_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonCoreNetworkAPI(t *testing.T, calls *[]nativeNetworkCall, reply func(*http.Request) *http.Response) *networks.API {
	t.Helper()
	cloud := testcloud.New(t)
	nativeNetworkRecorder(cloud, calls, reply)
	return networks.New(nativeNetworkClient(cloud))
}

// Expectations come from the pinned Proxy._create/_update, Resource.new and
// NetworkResource mappings. They intentionally differ from native builders.
func TestPythonCoreNetworkMutationBodiesDefaultsAndAliases(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for _, test := range []struct {
			name    string
			options []networks.AttributeOption
			want    map[string]any
		}{
			{"omitted", nil, map[string]any{}},
			{"explicit empty Body values", []networks.AttributeOption{networks.WithAttribute("name", ""), networks.WithAttribute("is_admin_state_up", false), networks.WithAttribute("is_router_external", false), networks.WithAttribute("is_shared", false), networks.WithAttribute("is_port_security_enabled", false), networks.WithAttribute("subnet_ids", []string{}), networks.WithAttribute("provider_network_type", ""), networks.WithAttribute("provider_segmentation_id", 0), networks.WithAttribute("ipv4_address_scope_id", nil)}, map[string]any{"name": "", "admin_state_up": false, "router:external": false, "shared": false, "port_security_enabled": false, "subnets": []any{}, "provider:network_type": "", "provider:segmentation_id": float64(0), "ipv4_address_scope": nil}},
			{"null and unknown query-only Body values", []networks.AttributeOption{networks.WithAttribute("x_extension", nil), networks.WithAttribute("fields", []string{"id"})}, map[string]any{"x_extension": nil, "fields": []any{"id"}}},
		} {
			t.Run(method+"/"+test.name, func(t *testing.T) {
				var calls []nativeNetworkCall
				api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response { return nativeNetworkWire(201, `{"network":`+nativeNetworkRow+`}`) })
				var value *networks.Network
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
				path := "/neutron/v2.0/networks"
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
				if want := map[string]any{"network": test.want}; !reflect.DeepEqual(body, want) {
					t.Fatalf("got %#v, want %#v", body, want)
				}
			})
		}
	}
}

func TestPythonCoreNetworkRevisionHeaderOnlyPriorityAndClientIsolation(t *testing.T) {
	for _, test := range []struct {
		name, header string
		options      []networks.AttributeOption
	}{
		{"revision zero only commits", "revision_number=0", []networks.AttributeOption{networks.WithRevision(0)}},
		{"revision overrides later raw header", "revision_number=0", []networks.AttributeOption{networks.WithRevision(0), networks.WithIfMatch("raw")}},
		{"revision overrides earlier raw header", "revision_number=0", []networks.AttributeOption{networks.WithAttribute("if_match", "raw"), networks.WithRevision(0)}},
		{"revision discards later invalid raw Header", "revision_number=0", []networks.AttributeOption{networks.WithRevision(0), networks.WithIfMatch("bad\r\nheader")}},
		{"revision discards earlier invalid raw Header list", "revision_number=0", []networks.AttributeOption{networks.WithAttribute("if_match", []int{1}), networks.WithRevision(0)}},
		{"revision discards null raw Header element", "revision_number=0", []networks.AttributeOption{networks.WithAttribute("if_match", []any{"a", nil}), networks.WithRevision(0)}},
		{"revision discards unserializable raw Header", "revision_number=0", []networks.AttributeOption{networks.WithAttribute("if_match", make(chan int)), networks.WithRevision(0)}},
		{"last revision wins", "revision_number=-2", []networks.AttributeOption{networks.WithRevision(0), networks.WithRevision(-2)}},
		{"large revision stays exact", "revision_number=9223372036854775807", []networks.AttributeOption{networks.WithRevision(9223372036854775807)}},
		{"Header list", "a, b", []networks.AttributeOption{networks.WithAttribute("if_match", []string{"a", "b"})}},
		{"wire Header name", "wire", []networks.AttributeOption{networks.WithAttribute("If-Match", "wire")}},
		{"explicit empty list", "", []networks.AttributeOption{networks.WithIfMatch()}},
		{"Source null Header", "None", []networks.AttributeOption{networks.WithAttribute("if_match", nil)}},
		{"bodyless 204 commits", "revision_number=0", []networks.AttributeOption{networks.WithRevision(0)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []nativeNetworkCall
			var headers []string
			api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
				headers = append(headers, req.Header.Get("If-Match"))
				if test.name == "bodyless 204 commits" {
					return nativeNetworkWire(204, "")
				}
				return nativeNetworkWire(200, `{"network":`+nativeNetworkRow+`}`)
			})
			configured := map[string]string{"if-match": "old", "If-Match": "other", "X-Trace": "trace"}
			api.RawClient().MoreHeaders = configured
			value, err := api.UpdateWithAttributes(context.Background(), "literal.id", test.options...)
			if err != nil {
				t.Fatal(err)
			}
			if test.name == "bodyless 204 commits" {
				if !reflect.DeepEqual(value, &networks.Network{}) {
					t.Fatal(value)
				}
			} else if value == nil || value.ID != "id-1" {
				t.Fatal(value)
			}
			if len(calls) != 1 || calls[0] != (nativeNetworkCall{http.MethodPut, "/neutron/v2.0/networks/literal.id", "", `{"network":{}}`}) || !reflect.DeepEqual(headers, []string{test.header}) {
				t.Fatal(calls, headers)
			}
			if !reflect.DeepEqual(configured, map[string]string{"if-match": "old", "If-Match": "other", "X-Trace": "trace"}) {
				t.Fatal(configured)
			}
		})
	}
	t.Run("configured header does not turn empty update into commit", func(t *testing.T) {
		var calls []nativeNetworkCall
		api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.Header.Get("If-Match") != "configured" || req.Header.Get("X-Trace") != "trace" {
				t.Fatal(req.Header)
			}
			return nativeNetworkWire(200, `{"network":`+nativeNetworkRow+`}`)
		})
		api.RawClient().MoreHeaders = map[string]string{"if-match": "configured", "X-Trace": "trace"}
		value, err := api.UpdateWithAttributes(context.Background(), "literal.id")
		if err != nil || value == nil || value.ID != "literal.id" || len(calls) != 0 {
			t.Fatal(value, err, calls)
		}
		if _, err := api.UpdateWithAttributes(context.Background(), "literal.id", networks.WithAttribute("name", "")); err != nil || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
}

func TestPythonCoreNetworkOptionSnapshotsHeaderRoutingAndPreflight(t *testing.T) {
	var calls []nativeNetworkCall
	var headers []string
	api := pythonCoreNetworkAPI(t, &calls, func(req *http.Request) *http.Response {
		headers = append(headers, req.Header.Get("If-Match"))
		return nativeNetworkWire(201, `{"network":`+nativeNetworkRow+`}`)
	})
	values, conditions := []string{"original"}, []string{"first", "second"}
	bodyOption, headerOption := networks.WithAttribute("x_extension", values), networks.WithAttribute("if_match", conditions)
	values[0], conditions[0] = "mutated", "mutated"
	for i := 0; i < 2; i++ {
		if _, err := api.CreateWithAttributes(context.Background(), bodyOption, headerOption, networks.WithAttribute("name", "first"), networks.WithAttribute("name", "last")); err != nil {
			t.Fatal(err)
		}
	}
	for _, call := range calls {
		var body map[string]map[string]any
		if err := json.Unmarshal([]byte(call.body), &body); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(body["network"], map[string]any{"x_extension": []any{"original"}, "name": "last"}) {
			t.Fatal(body)
		}
	}
	if !reflect.DeepEqual(headers, []string{"first, second", "first, second"}) {
		t.Fatal(headers)
	}
	calls = nil
	for _, options := range [][]networks.AttributeOption{
		{nil}, {networks.WithAttribute(" ", 1)}, {networks.WithAttribute("x_extension", make(chan int))},
		{networks.WithAttribute("microversion", "2.0")}, {networks.WithAttribute("base_path", "/other")},
		{networks.WithAttribute("if_match", []int{1})}, {networks.WithAttribute("if_match", []any{"a", nil})}, {networks.WithIfMatch("bad\r\nheader")},
		{request.WithHeader[struct{}]("Authorization", "replaced")},
		{request.WithField[struct{}]("If-Match", "body")},
		{request.WithQuery[struct{}]("fields", "id")}, {networks.WithRevision(0)},
	} {
		_, err := api.CreateWithAttributes(context.Background(), options...)
		nativeNetworkOperation(t, err, "CreateWithAttributes")
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	for _, options := range [][]networks.AttributeOption{
		{networks.WithAttribute("id", "other")}, {networks.WithAttribute("if_revision", 0)},
		{request.WithArgument[struct{}]("if_revision", "0")},
	} {
		_, err := api.UpdateWithAttributes(context.Background(), "literal.id", options...)
		nativeNetworkOperation(t, err, "UpdateWithAttributes")
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}

func TestPythonCoreNetworkMutationStatusesDecodeAndContextCauses(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for _, code := range []int{200, 201, 202, 204, 299, 300, 399, 400, 404, 409, 500} {
			t.Run(fmt.Sprintf("%s/%d", method, code), func(t *testing.T) {
				var calls []nativeNetworkCall
				body := `{"network":` + nativeNetworkRow + `}`
				api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
					reply := nativeNetworkWire(code, body)
					reply.Header.Set("X-Request-Id", "core-mutation")
					return reply
				})
				var value *networks.Network
				var err error
				operation := "CreateWithAttributes"
				if method == http.MethodPost {
					value, err = api.CreateWithAttributes(context.Background())
				} else {
					operation = "UpdateWithAttributes"
					value, err = api.UpdateWithAttributes(context.Background(), "literal.id", networks.WithRevision(0))
				}
				switch {
				case code == 204:
					if err != nil || !reflect.DeepEqual(value, &networks.Network{}) {
						t.Fatal(value, err)
					}
				case code < 400:
					if err != nil || value == nil || value.ID != "id-1" {
						t.Fatal(value, err)
					}
				default:
					nativeNetworkOperation(t, err, operation)
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
				var calls []nativeNetworkCall
				api := pythonCoreNetworkAPI(t, &calls, func(*http.Request) *http.Response {
					if lane == "response boundary" {
						cancel(cause)
					}
					if lane == "decode type" {
						return nativeNetworkWire(200, `{"network":[]}`)
					}
					if lane == "bodyless JSON" {
						return nativeNetworkWire(200, "")
					}
					return nativeNetworkWire(200, `{"network":`+nativeNetworkRow+`}`)
				})
				var options []networks.AttributeOption
				switch lane {
				case "pre-canceled":
					cancel(cause)
				case "empty option callback":
					options = []networks.AttributeOption{func(*request.Config[struct{}]) error { cancel(cause); return nil }}
				default:
					options = []networks.AttributeOption{networks.WithAttribute("name", "")}
				}
				var err error
				operation := "CreateWithAttributes"
				if method == http.MethodPost {
					_, err = api.CreateWithAttributes(ctx, options...)
				} else {
					operation = "UpdateWithAttributes"
					_, err = api.UpdateWithAttributes(ctx, "literal.id", options...)
				}
				nativeNetworkOperation(t, err, operation)
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
