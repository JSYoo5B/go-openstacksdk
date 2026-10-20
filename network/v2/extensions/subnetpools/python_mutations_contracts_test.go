package subnetpools_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"
	"testing/iotest"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Expectations below come from the pinned Python Proxy._create ->
// Resource.new/_collect_attrs/_prepare_request_body chain, not native builders.
func TestPythonSubnetPoolCreatePreservesOmissionPresenceAliasesAndExtensions(t *testing.T) {
	for _, test := range []struct {
		name    string
		options []subnetpools.AttributeOption
		want    map[string]any
	}{
		{"omitted", nil, map[string]any{}},
		{"explicit empty body fields", []subnetpools.AttributeOption{subnetpools.WithAttribute("default_prefix_length", 0), subnetpools.WithAttribute("minimum_prefix_length", 0), subnetpools.WithAttribute("maximum_prefix_length", 0), subnetpools.WithAttribute("is_shared", false), subnetpools.WithAttribute("prefixes", []string{})}, map[string]any{"default_prefixlen": float64(0), "min_prefixlen": float64(0), "max_prefixlen": float64(0), "shared": false, "prefixes": []any{}}},
		{"unknown and query-only attributes remain Body", []subnetpools.AttributeOption{subnetpools.WithAttribute("x_extension", nil), subnetpools.WithAttribute("fields", []string{"name"})}, map[string]any{"x_extension": nil, "fields": []any{"name"}}},
		{"ID is an ordinary create attribute", []subnetpools.AttributeOption{subnetpools.WithAttribute("id", "chosen")}, map[string]any{"id": "chosen"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []nativePoolCall
			api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(201, `{"subnetpool":`+nativePoolRow+`}`) })
			value, err := api.CreateWithAttributes(context.Background(), test.options...)
			if err != nil || value == nil || value.ID != "sp-1" {
				t.Fatal(value, err)
			}
			if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != "/neutron/v2.0/subnetpools" || calls[0].query != "" {
				t.Fatal(calls)
			}
			var body map[string]any
			if err := json.Unmarshal([]byte(calls[0].body), &body); err != nil {
				t.Fatal(err)
			}
			if want := map[string]any{"subnetpool": test.want}; !reflect.DeepEqual(body, want) {
				t.Fatalf("got %#v, want %#v", body, want)
			}
		})
	}
}

func TestPythonSubnetPoolMutationOptionSnapshotsAndPreflight(t *testing.T) {
	var calls []nativePoolCall
	api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(201, `{"subnetpool":`+nativePoolRow+`}`) })
	values := []string{"original"}
	option := subnetpools.WithAttribute("x_extension", values)
	values[0] = "changed"
	for i := 0; i < 2; i++ {
		if _, err := api.CreateWithAttributes(context.Background(), option, subnetpools.WithAttribute("name", "first"), subnetpools.WithAttribute("name", "last")); err != nil {
			t.Fatal(err)
		}
	}
	for _, call := range calls {
		var body map[string]map[string]any
		if err := json.Unmarshal([]byte(call.body), &body); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(body["subnetpool"], map[string]any{"x_extension": []any{"original"}, "name": "last"}) {
			t.Fatal(body)
		}
	}
	calls = nil
	for _, options := range [][]subnetpools.AttributeOption{
		{nil}, {subnetpools.WithAttribute(" ", 1)}, {subnetpools.WithAttribute("x_extension", make(chan int))},
		{subnetpools.WithAttribute("microversion", "2.0")}, {subnetpools.WithAttribute("base_path", "/other")},
		{request.WithQuery[struct{}]("fields", "id")},
	} {
		_, err := api.CreateWithAttributes(context.Background(), options...)
		nativePoolOperation(t, err, "CreateWithAttributes")
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}

func TestPythonSubnetPoolCreateStatusDecodeAndContextCauses(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 299, 300, 399, 400, 404, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativePoolCall
			body := `{"subnetpool":` + nativePoolRow + `}`
			api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response {
				reply := nativePoolWire(code, body)
				reply.Header.Set("X-Request-Id", "source-mutation")
				return reply
			})
			value, err := api.CreateWithAttributes(context.Background())
			if code == http.StatusNoContent {
				if err != nil || value != nil {
					t.Fatal("native 204 must keep an absent decoded model", value, err)
				}
			} else if code < 400 {
				if err != nil || value == nil || value.ID != "sp-1" {
					t.Fatal(value, err)
				}
			} else {
				nativePoolOperation(t, err, "CreateWithAttributes")
				var status gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &status) || status.Actual != code || string(status.Body) != body || status.ResponseHeader.Get("X-Request-Id") != "source-mutation" {
					t.Fatal(err, status)
				}
			}
			if len(calls) != 1 {
				t.Fatal(calls)
			}
		})
	}
	t.Run("native decode failures are preserved", func(t *testing.T) {
		var calls []nativePoolCall
		api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(201, `{"subnetpool":[]}`) })
		_, err := api.CreateWithAttributes(context.Background())
		nativePoolOperation(t, err, "CreateWithAttributes")
		var decode *json.UnmarshalTypeError
		if !errors.As(err, &decode) {
			t.Fatal(err)
		}
	})
	for _, lane := range []string{"before request", "option callback", "response boundary"} {
		t.Run(lane, func(t *testing.T) {
			cause := errors.New("caller stopped the mutation")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var calls []nativePoolCall
			api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response {
				if lane == "response boundary" {
					cancel(cause)
				}
				return nativePoolWire(201, `{"subnetpool":`+nativePoolRow+`}`)
			})
			var options []subnetpools.AttributeOption
			if lane == "before request" {
				cancel(cause)
			}
			if lane == "option callback" {
				options = []subnetpools.AttributeOption{func(*request.Config[struct{}]) error { cancel(cause); return nil }}
			}
			_, err := api.CreateWithAttributes(ctx, options...)
			nativePoolOperation(t, err, "CreateWithAttributes")
			if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
				t.Fatal(err)
			}
			wantCalls := 0
			if lane == "response boundary" {
				wantCalls = 1
			}
			if len(calls) != wantCalls {
				t.Fatal(calls)
			}
		})
	}
}

func TestPythonSubnetPoolDeleteDefaultsStrictOptionsAndSourceStatuses(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 299, 300, 399, 404, 409, 500} {
		for _, strict := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/strict=%v", code, strict), func(t *testing.T) {
				var calls []nativePoolCall
				api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response {
					response := nativePoolWire(code, "opaque response body")
					response.Header.Set("X-Request-Id", "source-delete")
					return response
				})
				var options []subnetpools.DeleteOption
				if strict {
					options = []subnetpools.DeleteOption{subnetpools.WithDeleteIgnoreMissing(false)}
				}
				err := api.DeleteWithOptions(context.Background(), "literal.id", options...)
				if code < 400 || code == 404 && !strict {
					if err != nil {
						t.Fatal(err)
					}
				} else {
					nativePoolOperation(t, err, "DeleteWithOptions")
					var status gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &status) || status.Actual != code || string(status.Body) != "opaque response body" || status.ResponseHeader.Get("X-Request-Id") != "source-delete" {
						t.Fatal(err, status)
					}
				}
				if len(calls) != 1 || calls[0] != (nativePoolCall{http.MethodDelete, "/neutron/v2.0/subnetpools/literal.id", "", ""}) {
					t.Fatal(calls)
				}
			})
		}
	}
	t.Run("constructor snapshot, reuse and last option wins", func(t *testing.T) {
		var calls []nativePoolCall
		api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(404, "missing") })
		strict := false
		option := subnetpools.WithDeleteOptions(subnetpools.DeleteOpts{IgnoreMissing: &strict})
		strict = true
		for i := 0; i < 2; i++ {
			if err := api.DeleteWithOptions(context.Background(), "gone", option); !gophercloud.ResponseCodeIs(err, 404) {
				t.Fatal(err)
			}
		}
		if err := api.DeleteWithOptions(context.Background(), "gone", option, subnetpools.WithDeleteIgnoreMissing(true)); err != nil {
			t.Fatal(err)
		}
		if err := api.DeleteWithOptions(context.Background(), "gone", subnetpools.WithDeleteIgnoreMissing(false), subnetpools.WithDeleteOptions(subnetpools.DeleteOpts{})); err != nil {
			t.Fatal(err)
		}
		if len(calls) != 4 {
			t.Fatal(calls)
		}
	})
}

func TestPythonSubnetPoolDeletePreservesTerminalFailuresAndPreflight(t *testing.T) {
	t.Run("physical 404 read failure is not missing success", func(t *testing.T) {
		wireCause := errors.New("response read failed")
		var calls []nativePoolCall
		api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response {
			response := nativePoolWire(404, "")
			response.Body = io.NopCloser(iotest.ErrReader(wireCause))
			return response
		})
		err := api.DeleteWithOptions(context.Background(), "missing")
		nativePoolOperation(t, err, "DeleteWithOptions")
		if !errors.Is(err, wireCause) || !gophercloud.ResponseCodeIs(err, 404) || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
	for _, lane := range []string{"before request", "response boundary"} {
		t.Run(lane, func(t *testing.T) {
			cause := errors.New("caller stopped deletion")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var calls []nativePoolCall
			api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response {
				if lane == "response boundary" {
					cancel(cause)
				}
				return nativePoolWire(200, "opaque")
			})
			if lane == "before request" {
				cancel(cause)
			}
			err := api.DeleteWithOptions(ctx, "missing")
			nativePoolOperation(t, err, "DeleteWithOptions")
			if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
				t.Fatal(err)
			}
			wantCalls := 0
			if lane == "response boundary" {
				wantCalls = 1
			}
			if len(calls) != wantCalls {
				t.Fatal(calls)
			}
		})
	}
	t.Run("invalid option makes no request", func(t *testing.T) {
		var calls []nativePoolCall
		api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(204, "") })
		err := api.DeleteWithOptions(context.Background(), "literal", nil)
		nativePoolOperation(t, err, "DeleteWithOptions")
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	})
}
