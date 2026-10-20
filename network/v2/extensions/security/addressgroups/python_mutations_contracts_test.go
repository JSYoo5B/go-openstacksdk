package addressgroups_test

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

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Expectations below come from the pinned Python Proxy._create ->
// Resource.new/_collect_attrs/_prepare_request_body chain, not native builders.
func TestPythonAddressGroupCreatePreservesOmissionPresenceAliasesAndExtensions(t *testing.T) {
	for _, test := range []struct {
		name    string
		options []addressgroups.AttributeOption
		want    map[string]any
	}{
		{"omitted", nil, map[string]any{}},
		{"explicit empty body fields", []addressgroups.AttributeOption{addressgroups.WithAttribute("addresses", []string{}), addressgroups.WithAttribute("name", ""), addressgroups.WithAttribute("description", nil)}, map[string]any{"addresses": []any{}, "name": "", "description": nil}},
		{"unknown and query-only attributes remain Body", []addressgroups.AttributeOption{addressgroups.WithAttribute("x_extension", nil), addressgroups.WithAttribute("fields", []string{"name"})}, map[string]any{"x_extension": nil, "fields": []any{"name"}}},
		{"ID is an ordinary create attribute", []addressgroups.AttributeOption{addressgroups.WithAttribute("id", "chosen")}, map[string]any{"id": "chosen"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []nativeGroupCall
			api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
				return nativeGroupWire(201, `{"address_group":`+nativeGroupRow+`}`)
			})
			value, err := api.CreateWithAttributes(context.Background(), test.options...)
			if err != nil || value == nil || value.ID != "ag-1" {
				t.Fatal(value, err)
			}
			if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != "/neutron/v2.0/address-groups" || calls[0].query != "" {
				t.Fatal(calls)
			}
			var body map[string]any
			if err := json.Unmarshal([]byte(calls[0].body), &body); err != nil {
				t.Fatal(err)
			}
			if want := map[string]any{"address_group": test.want}; !reflect.DeepEqual(body, want) {
				t.Fatalf("got %#v, want %#v", body, want)
			}
		})
	}
}

func TestPythonAddressGroupMutationOptionSnapshotsAndPreflight(t *testing.T) {
	var calls []nativeGroupCall
	api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeGroupWire(201, `{"address_group":`+nativeGroupRow+`}`)
	})
	values := []string{"original"}
	option := addressgroups.WithAttribute("x_extension", values)
	values[0] = "changed"
	for i := 0; i < 2; i++ {
		if _, err := api.CreateWithAttributes(context.Background(), option, addressgroups.WithAttribute("name", "first"), addressgroups.WithAttribute("name", "last")); err != nil {
			t.Fatal(err)
		}
	}
	for _, call := range calls {
		var body map[string]map[string]any
		if err := json.Unmarshal([]byte(call.body), &body); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(body["address_group"], map[string]any{"x_extension": []any{"original"}, "name": "last"}) {
			t.Fatal(body)
		}
	}
	calls = nil
	for _, options := range [][]addressgroups.AttributeOption{
		{nil}, {addressgroups.WithAttribute(" ", 1)}, {addressgroups.WithAttribute("x_extension", make(chan int))},
		{addressgroups.WithAttribute("microversion", "2.0")}, {addressgroups.WithAttribute("base_path", "/other")},
		{request.WithQuery[struct{}]("fields", "id")},
	} {
		_, err := api.CreateWithAttributes(context.Background(), options...)
		nativeGroupOperation(t, err, "CreateWithAttributes")
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}

func TestPythonAddressGroupCreateStatusDecodeAndContextCauses(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 299, 300, 399, 400, 404, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativeGroupCall
			body := `{"address_group":` + nativeGroupRow + `}`
			api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
				reply := nativeGroupWire(code, body)
				reply.Header.Set("X-Request-Id", "source-mutation")
				return reply
			})
			value, err := api.CreateWithAttributes(context.Background())
			if code == http.StatusNoContent {
				if err != nil || value != nil {
					t.Fatal("native 204 must keep an absent decoded model", value, err)
				}
			} else if code < 400 {
				if err != nil || value == nil || value.ID != "ag-1" {
					t.Fatal(value, err)
				}
			} else {
				nativeGroupOperation(t, err, "CreateWithAttributes")
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
		var calls []nativeGroupCall
		api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(201, `{"address_group":[]}`) })
		_, err := api.CreateWithAttributes(context.Background())
		nativeGroupOperation(t, err, "CreateWithAttributes")
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
			var calls []nativeGroupCall
			api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
				if lane == "response boundary" {
					cancel(cause)
				}
				return nativeGroupWire(201, `{"address_group":`+nativeGroupRow+`}`)
			})
			var options []addressgroups.AttributeOption
			if lane == "before request" {
				cancel(cause)
			}
			if lane == "option callback" {
				options = []addressgroups.AttributeOption{func(*request.Config[struct{}]) error { cancel(cause); return nil }}
			}
			_, err := api.CreateWithAttributes(ctx, options...)
			nativeGroupOperation(t, err, "CreateWithAttributes")
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

func TestPythonAddressGroupDeleteDefaultsStrictOptionsAndSourceStatuses(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 299, 300, 399, 404, 409, 500} {
		for _, strict := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/strict=%v", code, strict), func(t *testing.T) {
				var calls []nativeGroupCall
				api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
					response := nativeGroupWire(code, "opaque response body")
					response.Header.Set("X-Request-Id", "source-delete")
					return response
				})
				var options []addressgroups.DeleteOption
				if strict {
					options = []addressgroups.DeleteOption{addressgroups.WithDeleteIgnoreMissing(false)}
				}
				err := api.DeleteWithOptions(context.Background(), "literal.id", options...)
				if code < 400 || code == 404 && !strict {
					if err != nil {
						t.Fatal(err)
					}
				} else {
					nativeGroupOperation(t, err, "DeleteWithOptions")
					var status gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &status) || status.Actual != code || string(status.Body) != "opaque response body" || status.ResponseHeader.Get("X-Request-Id") != "source-delete" {
						t.Fatal(err, status)
					}
				}
				if len(calls) != 1 || calls[0] != (nativeGroupCall{http.MethodDelete, "/neutron/v2.0/address-groups/literal.id", "", ""}) {
					t.Fatal(calls)
				}
			})
		}
	}
	t.Run("constructor snapshot, reuse and last option wins", func(t *testing.T) {
		var calls []nativeGroupCall
		api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(404, "missing") })
		strict := false
		option := addressgroups.WithDeleteOptions(addressgroups.DeleteOpts{IgnoreMissing: &strict})
		strict = true
		for i := 0; i < 2; i++ {
			if err := api.DeleteWithOptions(context.Background(), "gone", option); !gophercloud.ResponseCodeIs(err, 404) {
				t.Fatal(err)
			}
		}
		if err := api.DeleteWithOptions(context.Background(), "gone", option, addressgroups.WithDeleteIgnoreMissing(true)); err != nil {
			t.Fatal(err)
		}
		if err := api.DeleteWithOptions(context.Background(), "gone", addressgroups.WithDeleteIgnoreMissing(false), addressgroups.WithDeleteOptions(addressgroups.DeleteOpts{})); err != nil {
			t.Fatal(err)
		}
		if len(calls) != 4 {
			t.Fatal(calls)
		}
	})
}

func TestPythonAddressGroupDeletePreservesTerminalFailuresAndPreflight(t *testing.T) {
	t.Run("physical 404 read failure is not missing success", func(t *testing.T) {
		wireCause := errors.New("response read failed")
		var calls []nativeGroupCall
		api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
			response := nativeGroupWire(404, "")
			response.Body = io.NopCloser(iotest.ErrReader(wireCause))
			return response
		})
		err := api.DeleteWithOptions(context.Background(), "missing")
		nativeGroupOperation(t, err, "DeleteWithOptions")
		if !errors.Is(err, wireCause) || !gophercloud.ResponseCodeIs(err, 404) || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
	for _, lane := range []string{"before request", "response boundary"} {
		t.Run(lane, func(t *testing.T) {
			cause := errors.New("caller stopped deletion")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var calls []nativeGroupCall
			api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
				if lane == "response boundary" {
					cancel(cause)
				}
				return nativeGroupWire(200, "opaque")
			})
			if lane == "before request" {
				cancel(cause)
			}
			err := api.DeleteWithOptions(ctx, "missing")
			nativeGroupOperation(t, err, "DeleteWithOptions")
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
		var calls []nativeGroupCall
		api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(204, "") })
		err := api.DeleteWithOptions(context.Background(), "literal", nil)
		nativeGroupOperation(t, err, "DeleteWithOptions")
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	})
}

func TestPythonAddressGroupUpdateEmptyAttributesReturnsIDWithoutHTTP(t *testing.T) {
	var calls []nativeGroupCall
	api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeGroupWire(200, `{"address_group":`+nativeGroupRow+`}`)
	})
	value, err := api.UpdateWithAttributes(context.Background(), "literal.id")
	if err != nil || value == nil || value.ID != "literal.id" || value.Name != "" || len(calls) != 0 {
		t.Fatal(value, err, calls)
	}
	if _, err := api.UpdateWithAttributes(context.Background(), "literal.id", addressgroups.WithAttribute("id", "other")); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonAddressGroupUpdatePreservesChangedBodyAndResponseModel(t *testing.T) {
	var calls []nativeGroupCall
	api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeGroupWire(200, `{"address_group":`+nativeGroupRow+`}`)
	})
	value, err := api.UpdateWithAttributes(context.Background(), "literal.id", addressgroups.WithAttribute("name", ""), addressgroups.WithAttribute("description", nil), addressgroups.WithAttribute("addresses", []string{}), addressgroups.WithAttribute("x_extension", nil))
	if err != nil || value == nil || value.ID != "ag-1" || value.Name == "" {
		t.Fatal(value, err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != "/neutron/v2.0/address-groups/literal.id" || calls[0].query != "" {
		t.Fatal(calls)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(calls[0].body), &body); err != nil {
		t.Fatal(err)
	}
	if want := map[string]any{"address_group": map[string]any{"name": "", "description": nil, "addresses": []any{}, "x_extension": nil}}; !reflect.DeepEqual(body, want) {
		t.Fatal(body, want)
	}
}

func TestPythonAddressGroupUpdateStatusCancellationAndDecodePolicy(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 300, 399, 404, 409} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativeGroupCall
			body := `{"address_group":` + nativeGroupRow + `}`
			api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(code, body) })
			value, err := api.UpdateWithAttributes(context.Background(), "literal.id", addressgroups.WithAttribute("name", ""))
			if code == http.StatusNoContent {
				if err != nil || value != nil {
					t.Fatal("native 204 must keep an absent decoded model", value, err)
				}
			} else if code < 400 {
				if err != nil || value == nil || value.ID != "ag-1" {
					t.Fatal(value, err)
				}
			} else {
				nativeGroupOperation(t, err, "UpdateWithAttributes")
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
			var calls []nativeGroupCall
			api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
				if lane == "response boundary" {
					cancel(cause)
				}
				if lane == "decode" {
					return nativeGroupWire(200, `{"address_group":[]}`)
				}
				return nativeGroupWire(200, `{"address_group":`+nativeGroupRow+`}`)
			})
			var options []addressgroups.AttributeOption
			switch lane {
			case "empty before request":
				cancel(cause)
			case "empty option callback":
				options = []addressgroups.AttributeOption{func(*request.Config[struct{}]) error { cancel(cause); return nil }}
			default:
				options = []addressgroups.AttributeOption{addressgroups.WithAttribute("name", "")}
			}
			_, err := api.UpdateWithAttributes(ctx, "literal.id", options...)
			nativeGroupOperation(t, err, "UpdateWithAttributes")
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
