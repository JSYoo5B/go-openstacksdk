package traits_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/traits"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeTraitTransport func(*http.Request) (*http.Response, error)

func (transport nativeTraitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeTraitWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeTraitCall struct{ method, path, query, body, version string }

func nativeTraitAPI(t *testing.T, calls *[]nativeTraitCall, reply func(*http.Request) *http.Response) *traits.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeTraitTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeTraitCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("OpenStack-API-Version")})
		return reply(req), nil
	})
	client := cloud.Client("placement", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/placement/"
	client.Microversion = "1.6"
	return traits.New(client)
}

func nativeTraitOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "traits" {
		t.Fatal("generated traits context", err, wrapped)
	}
}

func TestNativeTraitsRoutesQueryAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeTraitCall
	api := nativeTraitAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodPut:
			return nativeTraitWire(201, "")
		case req.Method == http.MethodGet && req.URL.Path == "/placement/traits":
			return nativeTraitWire(200, `{"traits":["CUSTOM_A","HW_CPU_X86_AVX"]}`)
		}
		return nativeTraitWire(204, "")
	})
	if err := api.Create(ctx, "CUSTOM_A"); err != nil {
		t.Fatal(err)
	}
	// Get only checks existence and answers 204 without a body.
	if err := api.Get(ctx, "CUSTOM_A"); err != nil {
		t.Fatal(err)
	}
	associated := false
	var names []string
	for value, err := range api.List(ctx, traits.WithListOptions(traits.ListOpts{Name: "in:CUSTOM_A,HW_CPU_X86_AVX", Associated: &associated}), traits.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, *value)
	}
	if err := api.Delete(ctx, "CUSTOM_A"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(names, []string{"CUSTOM_A", "HW_CPU_X86_AVX"}) {
		t.Fatal(names)
	}
	base := "/placement/traits"
	want := []nativeTraitCall{
		{http.MethodPut, base + "/CUSTOM_A", "", "", "placement 1.6"},
		{http.MethodGet, base + "/CUSTOM_A", "", "", "placement 1.6"},
		// A false Associated pointer is still sent.
		{http.MethodGet, base, "associated=false&extra=1&name=in%3ACUSTOM_A%2CHW_CPU_X86_AVX", "", "placement 1.6"},
		{http.MethodDelete, base + "/CUSTOM_A", "", "", "placement 1.6"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeTraitsStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*traits.API) error
	}{
		{"Create", []int{201, 204}, func(api *traits.API) error { return api.Create(ctx, "CUSTOM_A") }},
		{"Get", []int{204}, func(api *traits.API) error { return api.Get(ctx, "CUSTOM_A") }},
		{"Delete", []int{204}, func(api *traits.API) error { return api.Delete(ctx, "CUSTOM_A") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404, 409} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeTraitCall
				api := nativeTraitAPI(t, &calls, func(*http.Request) *http.Response { return nativeTraitWire(code, `{}`) })
				err := call.call(api)
				nativeTraitOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("list status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"traits":[]}`}, {200, `{"traits":[1]}`}, {204, ""}} {
			var calls []nativeTraitCall
			api := nativeTraitAPI(t, &calls, func(*http.Request) *http.Response { return nativeTraitWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			switch {
			case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
			case tc.code == 200 && tc.body == `{"traits":[]}` && len(errs) == 0:
			case tc.code == 200 && len(errs) == 1 && errs[0] != nil:
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, errs)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeTraitCall
		api := nativeTraitAPI(t, &calls, func(*http.Request) *http.Response { return nativeTraitWire(200, `{}`) })
		for name, options := range map[string][]traits.ListOption{
			"empty query key": {traits.WithListQuery("", "x")},
			"nil option":      {nil},
		} {
			var errs []error
			for _, err := range api.List(ctx, options...) {
				errs = append(errs, err)
			}
			if len(errs) != 1 {
				t.Fatal(name, errs)
			}
			nativeTraitOperation(t, errs[0], "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
