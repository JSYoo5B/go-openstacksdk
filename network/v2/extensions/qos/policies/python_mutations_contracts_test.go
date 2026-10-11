package policies_test

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

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Expectations below come from the pinned Python Proxy._create ->
// Resource.new/_collect_attrs/_prepare_request_body chain, not native builders.
func TestPythonQoSPolicyCreatePreservesOmissionPresenceAliasesAndExtensions(t *testing.T) {
	for _, test := range []struct {
		name    string
		options []policies.AttributeOption
		want    map[string]any
	}{
		{"omitted", nil, map[string]any{}},
		{"explicit empty body fields", []policies.AttributeOption{policies.WithAttribute("is_shared", false), policies.WithAttribute("is_default", false), policies.WithAttribute("name", "")}, map[string]any{"shared": false, "is_default": false, "name": ""}},
		{"unknown and query-only attributes remain Body", []policies.AttributeOption{policies.WithAttribute("x_extension", nil), policies.WithAttribute("fields", []string{"name"})}, map[string]any{"x_extension": nil, "fields": []any{"name"}}},
		{"ID is an ordinary create attribute", []policies.AttributeOption{policies.WithAttribute("id", "chosen")}, map[string]any{"id": "chosen"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []nativePolicyCall
			api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response { return nativePolicyWire(201, `{"policy":`+nativePolicyRow+`}`) })
			value, err := api.CreateWithAttributes(context.Background(), test.options...)
			if err != nil || value == nil || value.ID != "qp-1" {
				t.Fatal(value, err)
			}
			if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != "/neutron/v2.0/qos/policies" || calls[0].query != "" {
				t.Fatal(calls)
			}
			var body map[string]any
			if err := json.Unmarshal([]byte(calls[0].body), &body); err != nil {
				t.Fatal(err)
			}
			if want := map[string]any{"policy": test.want}; !reflect.DeepEqual(body, want) {
				t.Fatalf("got %#v, want %#v", body, want)
			}
		})
	}
}

func TestPythonQoSPolicyMutationOptionSnapshotsAndPreflight(t *testing.T) {
	var calls []nativePolicyCall
	api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response { return nativePolicyWire(201, `{"policy":`+nativePolicyRow+`}`) })
	values := []string{"original"}
	option := policies.WithAttribute("x_extension", values)
	values[0] = "changed"
	for i := 0; i < 2; i++ {
		if _, err := api.CreateWithAttributes(context.Background(), option, policies.WithAttribute("name", "first"), policies.WithAttribute("name", "last")); err != nil {
			t.Fatal(err)
		}
	}
	for _, call := range calls {
		var body map[string]map[string]any
		if err := json.Unmarshal([]byte(call.body), &body); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(body["policy"], map[string]any{"x_extension": []any{"original"}, "name": "last"}) {
			t.Fatal(body)
		}
	}
	calls = nil
	for _, options := range [][]policies.AttributeOption{
		{nil}, {policies.WithAttribute(" ", 1)}, {policies.WithAttribute("x_extension", make(chan int))},
		{policies.WithAttribute("microversion", "2.0")}, {policies.WithAttribute("base_path", "/other")},
		{request.WithQuery[struct{}]("fields", "id")},
	} {
		_, err := api.CreateWithAttributes(context.Background(), options...)
		nativePolicyOperation(t, err, "CreateWithAttributes")
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}

func TestPythonQoSPolicyCreateStatusDecodeAndContextCauses(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 299, 300, 399, 400, 404, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativePolicyCall
			body := `{"policy":` + nativePolicyRow + `}`
			api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response {
				reply := nativePolicyWire(code, body)
				reply.Header.Set("X-Request-Id", "source-mutation")
				return reply
			})
			value, err := api.CreateWithAttributes(context.Background())
			if code == http.StatusNoContent {
				if err != nil || value != nil {
					t.Fatal("native 204 must keep an absent decoded model", value, err)
				}
			} else if code < 400 {
				if err != nil || value == nil || value.ID != "qp-1" {
					t.Fatal(value, err)
				}
			} else {
				nativePolicyOperation(t, err, "CreateWithAttributes")
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
		var calls []nativePolicyCall
		api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response { return nativePolicyWire(201, `{"policy":[]}`) })
		_, err := api.CreateWithAttributes(context.Background())
		nativePolicyOperation(t, err, "CreateWithAttributes")
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
			var calls []nativePolicyCall
			api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response {
				if lane == "response boundary" {
					cancel(cause)
				}
				return nativePolicyWire(201, `{"policy":`+nativePolicyRow+`}`)
			})
			var options []policies.AttributeOption
			if lane == "before request" {
				cancel(cause)
			}
			if lane == "option callback" {
				options = []policies.AttributeOption{func(*request.Config[struct{}]) error { cancel(cause); return nil }}
			}
			_, err := api.CreateWithAttributes(ctx, options...)
			nativePolicyOperation(t, err, "CreateWithAttributes")
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

func TestPythonQoSPolicyDeleteDefaultsStrictOptionsAndSourceStatuses(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 299, 300, 399, 404, 409, 500} {
		for _, strict := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/strict=%v", code, strict), func(t *testing.T) {
				var calls []nativePolicyCall
				api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response {
					response := nativePolicyWire(code, "opaque response body")
					response.Header.Set("X-Request-Id", "source-delete")
					return response
				})
				var options []policies.DeleteOption
				if strict {
					options = []policies.DeleteOption{policies.WithDeleteIgnoreMissing(false)}
				}
				err := api.DeleteWithOptions(context.Background(), "literal.id", options...)
				if code < 400 || code == 404 && !strict {
					if err != nil {
						t.Fatal(err)
					}
				} else {
					nativePolicyOperation(t, err, "DeleteWithOptions")
					var status gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &status) || status.Actual != code || string(status.Body) != "opaque response body" || status.ResponseHeader.Get("X-Request-Id") != "source-delete" {
						t.Fatal(err, status)
					}
				}
				if len(calls) != 1 || calls[0] != (nativePolicyCall{http.MethodDelete, "/neutron/v2.0/qos/policies/literal.id", "", ""}) {
					t.Fatal(calls)
				}
			})
		}
	}
	t.Run("constructor snapshot, reuse and last option wins", func(t *testing.T) {
		var calls []nativePolicyCall
		api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response { return nativePolicyWire(404, "missing") })
		strict := false
		option := policies.WithDeleteOptions(policies.DeleteOpts{IgnoreMissing: &strict})
		strict = true
		for i := 0; i < 2; i++ {
			if err := api.DeleteWithOptions(context.Background(), "gone", option); !gophercloud.ResponseCodeIs(err, 404) {
				t.Fatal(err)
			}
		}
		if err := api.DeleteWithOptions(context.Background(), "gone", option, policies.WithDeleteIgnoreMissing(true)); err != nil {
			t.Fatal(err)
		}
		if err := api.DeleteWithOptions(context.Background(), "gone", policies.WithDeleteIgnoreMissing(false), policies.WithDeleteOptions(policies.DeleteOpts{})); err != nil {
			t.Fatal(err)
		}
		if len(calls) != 4 {
			t.Fatal(calls)
		}
	})
}

func TestPythonQoSPolicyDeletePreservesTerminalFailuresAndPreflight(t *testing.T) {
	t.Run("physical 404 read failure is not missing success", func(t *testing.T) {
		wireCause := errors.New("response read failed")
		var calls []nativePolicyCall
		api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response {
			response := nativePolicyWire(404, "")
			response.Body = io.NopCloser(iotest.ErrReader(wireCause))
			return response
		})
		err := api.DeleteWithOptions(context.Background(), "missing")
		nativePolicyOperation(t, err, "DeleteWithOptions")
		if !errors.Is(err, wireCause) || !gophercloud.ResponseCodeIs(err, 404) || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
	for _, lane := range []string{"before request", "response boundary"} {
		t.Run(lane, func(t *testing.T) {
			cause := errors.New("caller stopped deletion")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var calls []nativePolicyCall
			api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response {
				if lane == "response boundary" {
					cancel(cause)
				}
				return nativePolicyWire(200, "opaque")
			})
			if lane == "before request" {
				cancel(cause)
			}
			err := api.DeleteWithOptions(ctx, "missing")
			nativePolicyOperation(t, err, "DeleteWithOptions")
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
		var calls []nativePolicyCall
		api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response { return nativePolicyWire(204, "") })
		err := api.DeleteWithOptions(context.Background(), "literal", nil)
		nativePolicyOperation(t, err, "DeleteWithOptions")
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	})
}
