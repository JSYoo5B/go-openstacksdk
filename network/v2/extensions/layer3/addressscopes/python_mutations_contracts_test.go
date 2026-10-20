package addressscopes_test

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

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Expectations below come from the pinned Python Proxy._create ->
// Resource.new/_collect_attrs/_prepare_request_body chain, not native builders.
func TestPythonAddressScopeCreatePreservesOmissionPresenceAliasesAndExtensions(t *testing.T) {
	for _, test := range []struct {
		name    string
		options []addressscopes.AttributeOption
		want    map[string]any
	}{
		{"omitted", nil, map[string]any{}},
		{"explicit empty body fields", []addressscopes.AttributeOption{addressscopes.WithAttribute("ip_version", 0), addressscopes.WithAttribute("is_shared", false), addressscopes.WithAttribute("name", "")}, map[string]any{"ip_version": float64(0), "shared": false, "name": ""}},
		{"unknown and query-only attributes remain Body", []addressscopes.AttributeOption{addressscopes.WithAttribute("x_extension", nil), addressscopes.WithAttribute("fields", []string{"name"})}, map[string]any{"x_extension": nil, "fields": []any{"name"}}},
		{"ID is an ordinary create attribute", []addressscopes.AttributeOption{addressscopes.WithAttribute("id", "chosen")}, map[string]any{"id": "chosen"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []nativeScopeCall
			api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response {
				return nativeScopeWire(201, `{"address_scope":`+nativeScopeRow+`}`)
			})
			value, err := api.CreateWithAttributes(context.Background(), test.options...)
			if err != nil || value == nil || value.ID != "as-1" {
				t.Fatal(value, err)
			}
			if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != "/neutron/v2.0/address-scopes" || calls[0].query != "" {
				t.Fatal(calls)
			}
			var body map[string]any
			if err := json.Unmarshal([]byte(calls[0].body), &body); err != nil {
				t.Fatal(err)
			}
			if want := map[string]any{"address_scope": test.want}; !reflect.DeepEqual(body, want) {
				t.Fatalf("got %#v, want %#v", body, want)
			}
		})
	}
}

func TestPythonAddressScopeMutationOptionSnapshotsAndPreflight(t *testing.T) {
	var calls []nativeScopeCall
	api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeScopeWire(201, `{"address_scope":`+nativeScopeRow+`}`)
	})
	values := []string{"original"}
	option := addressscopes.WithAttribute("x_extension", values)
	values[0] = "changed"
	for i := 0; i < 2; i++ {
		if _, err := api.CreateWithAttributes(context.Background(), option, addressscopes.WithAttribute("name", "first"), addressscopes.WithAttribute("name", "last")); err != nil {
			t.Fatal(err)
		}
	}
	for _, call := range calls {
		var body map[string]map[string]any
		if err := json.Unmarshal([]byte(call.body), &body); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(body["address_scope"], map[string]any{"x_extension": []any{"original"}, "name": "last"}) {
			t.Fatal(body)
		}
	}
	calls = nil
	for _, options := range [][]addressscopes.AttributeOption{
		{nil}, {addressscopes.WithAttribute(" ", 1)}, {addressscopes.WithAttribute("x_extension", make(chan int))},
		{addressscopes.WithAttribute("microversion", "2.0")}, {addressscopes.WithAttribute("base_path", "/other")},
		{request.WithQuery[struct{}]("fields", "id")},
	} {
		_, err := api.CreateWithAttributes(context.Background(), options...)
		nativeScopeOperation(t, err, "CreateWithAttributes")
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}

func TestPythonAddressScopeCreateStatusDecodeAndContextCauses(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 299, 300, 399, 400, 404, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativeScopeCall
			body := `{"address_scope":` + nativeScopeRow + `}`
			api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response {
				reply := nativeScopeWire(code, body)
				reply.Header.Set("X-Request-Id", "source-mutation")
				return reply
			})
			value, err := api.CreateWithAttributes(context.Background())
			if code == http.StatusNoContent {
				if err != nil || value != nil {
					t.Fatal("native 204 must keep an absent decoded model", value, err)
				}
			} else if code < 400 {
				if err != nil || value == nil || value.ID != "as-1" {
					t.Fatal(value, err)
				}
			} else {
				nativeScopeOperation(t, err, "CreateWithAttributes")
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
		var calls []nativeScopeCall
		api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response { return nativeScopeWire(201, `{"address_scope":[]}`) })
		_, err := api.CreateWithAttributes(context.Background())
		nativeScopeOperation(t, err, "CreateWithAttributes")
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
			var calls []nativeScopeCall
			api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response {
				if lane == "response boundary" {
					cancel(cause)
				}
				return nativeScopeWire(201, `{"address_scope":`+nativeScopeRow+`}`)
			})
			var options []addressscopes.AttributeOption
			if lane == "before request" {
				cancel(cause)
			}
			if lane == "option callback" {
				options = []addressscopes.AttributeOption{func(*request.Config[struct{}]) error { cancel(cause); return nil }}
			}
			_, err := api.CreateWithAttributes(ctx, options...)
			nativeScopeOperation(t, err, "CreateWithAttributes")
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

func TestPythonAddressScopeDeleteDefaultsStrictOptionsAndSourceStatuses(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 299, 300, 399, 404, 409, 500} {
		for _, strict := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/strict=%v", code, strict), func(t *testing.T) {
				var calls []nativeScopeCall
				api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response {
					response := nativeScopeWire(code, "opaque response body")
					response.Header.Set("X-Request-Id", "source-delete")
					return response
				})
				var options []addressscopes.DeleteOption
				if strict {
					options = []addressscopes.DeleteOption{addressscopes.WithDeleteIgnoreMissing(false)}
				}
				err := api.DeleteWithOptions(context.Background(), "literal.id", options...)
				if code < 400 || code == 404 && !strict {
					if err != nil {
						t.Fatal(err)
					}
				} else {
					nativeScopeOperation(t, err, "DeleteWithOptions")
					var status gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &status) || status.Actual != code || string(status.Body) != "opaque response body" || status.ResponseHeader.Get("X-Request-Id") != "source-delete" {
						t.Fatal(err, status)
					}
				}
				if len(calls) != 1 || calls[0] != (nativeScopeCall{http.MethodDelete, "/neutron/v2.0/address-scopes/literal.id", "", ""}) {
					t.Fatal(calls)
				}
			})
		}
	}
	t.Run("constructor snapshot, reuse and last option wins", func(t *testing.T) {
		var calls []nativeScopeCall
		api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response { return nativeScopeWire(404, "missing") })
		strict := false
		option := addressscopes.WithDeleteOptions(addressscopes.DeleteOpts{IgnoreMissing: &strict})
		strict = true
		for i := 0; i < 2; i++ {
			if err := api.DeleteWithOptions(context.Background(), "gone", option); !gophercloud.ResponseCodeIs(err, 404) {
				t.Fatal(err)
			}
		}
		if err := api.DeleteWithOptions(context.Background(), "gone", option, addressscopes.WithDeleteIgnoreMissing(true)); err != nil {
			t.Fatal(err)
		}
		if err := api.DeleteWithOptions(context.Background(), "gone", addressscopes.WithDeleteIgnoreMissing(false), addressscopes.WithDeleteOptions(addressscopes.DeleteOpts{})); err != nil {
			t.Fatal(err)
		}
		if len(calls) != 4 {
			t.Fatal(calls)
		}
	})
}

func TestPythonAddressScopeDeletePreservesTerminalFailuresAndPreflight(t *testing.T) {
	t.Run("physical 404 read failure is not missing success", func(t *testing.T) {
		wireCause := errors.New("response read failed")
		var calls []nativeScopeCall
		api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response {
			response := nativeScopeWire(404, "")
			response.Body = io.NopCloser(iotest.ErrReader(wireCause))
			return response
		})
		err := api.DeleteWithOptions(context.Background(), "missing")
		nativeScopeOperation(t, err, "DeleteWithOptions")
		if !errors.Is(err, wireCause) || !gophercloud.ResponseCodeIs(err, 404) || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
	for _, lane := range []string{"before request", "response boundary"} {
		t.Run(lane, func(t *testing.T) {
			cause := errors.New("caller stopped deletion")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var calls []nativeScopeCall
			api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response {
				if lane == "response boundary" {
					cancel(cause)
				}
				return nativeScopeWire(200, "opaque")
			})
			if lane == "before request" {
				cancel(cause)
			}
			err := api.DeleteWithOptions(ctx, "missing")
			nativeScopeOperation(t, err, "DeleteWithOptions")
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
		var calls []nativeScopeCall
		api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response { return nativeScopeWire(204, "") })
		err := api.DeleteWithOptions(context.Background(), "literal", nil)
		nativeScopeOperation(t, err, "DeleteWithOptions")
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	})
}

func TestPythonAddressScopeUpdateEmptyAttributesReturnsIDWithoutHTTP(t *testing.T) {
	var calls []nativeScopeCall
	api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeScopeWire(200, `{"address_scope":`+nativeScopeRow+`}`)
	})
	value, err := api.UpdateWithAttributes(context.Background(), "literal.id")
	if err != nil || value == nil || value.ID != "literal.id" || value.Name != "" || len(calls) != 0 {
		t.Fatal(value, err, calls)
	}
	if _, err := api.UpdateWithAttributes(context.Background(), "literal.id", addressscopes.WithAttribute("id", "other")); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonAddressScopeUpdatePreservesChangedBodyAndResponseModel(t *testing.T) {
	var calls []nativeScopeCall
	api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeScopeWire(200, `{"address_scope":`+nativeScopeRow+`}`)
	})
	value, err := api.UpdateWithAttributes(context.Background(), "literal.id", addressscopes.WithAttribute("name", ""), addressscopes.WithAttribute("is_shared", false), addressscopes.WithAttribute("x_extension", nil))
	if err != nil || value == nil || value.ID != "as-1" || value.Name == "" {
		t.Fatal(value, err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != "/neutron/v2.0/address-scopes/literal.id" || calls[0].query != "" {
		t.Fatal(calls)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(calls[0].body), &body); err != nil {
		t.Fatal(err)
	}
	if want := map[string]any{"address_scope": map[string]any{"name": "", "shared": false, "x_extension": nil}}; !reflect.DeepEqual(body, want) {
		t.Fatal(body, want)
	}
}

func TestPythonAddressScopeUpdateStatusCancellationAndDecodePolicy(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 300, 399, 404, 409} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativeScopeCall
			body := `{"address_scope":` + nativeScopeRow + `}`
			api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response { return nativeScopeWire(code, body) })
			value, err := api.UpdateWithAttributes(context.Background(), "literal.id", addressscopes.WithAttribute("name", ""))
			if code == http.StatusNoContent {
				if err != nil || value != nil {
					t.Fatal("native 204 must keep an absent decoded model", value, err)
				}
			} else if code < 400 {
				if err != nil || value == nil || value.ID != "as-1" {
					t.Fatal(value, err)
				}
			} else {
				nativeScopeOperation(t, err, "UpdateWithAttributes")
				var status gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &status) || status.Actual != code || string(status.Body) != body {
					t.Fatal(err, status)
				}
			}
			if len(calls) != 1 {
				t.Fatal(calls)
			}
		})
	}
	for _, lane := range []string{"empty before request", "empty option callback", "response boundary", "decode"} {
		t.Run(lane, func(t *testing.T) {
			cause := errors.New("caller stopped update")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var calls []nativeScopeCall
			api, _ := nativeScopeAPI(t, &calls, func(*http.Request) *http.Response {
				if lane == "response boundary" {
					cancel(cause)
				}
				if lane == "decode" {
					return nativeScopeWire(200, `{"address_scope":[]}`)
				}
				return nativeScopeWire(200, `{"address_scope":`+nativeScopeRow+`}`)
			})
			var options []addressscopes.AttributeOption
			switch lane {
			case "empty before request":
				cancel(cause)
			case "empty option callback":
				options = []addressscopes.AttributeOption{func(*request.Config[struct{}]) error { cancel(cause); return nil }}
			default:
				options = []addressscopes.AttributeOption{addressscopes.WithAttribute("name", "")}
			}
			_, err := api.UpdateWithAttributes(ctx, "literal.id", options...)
			nativeScopeOperation(t, err, "UpdateWithAttributes")
			wantCalls := 0
			if lane == "decode" {
				var decode *json.UnmarshalTypeError
				if !errors.As(err, &decode) {
					t.Fatal(err)
				}
				wantCalls = 1
			} else {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
					t.Fatal(err)
				}
				if lane == "response boundary" {
					wantCalls = 1
				}
			}
			if len(calls) != wantCalls {
				t.Fatal(calls)
			}
		})
	}
}
