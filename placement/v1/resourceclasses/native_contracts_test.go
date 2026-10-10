package resourceclasses_test

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
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeRCTransport func(*http.Request) (*http.Response, error)

func (transport nativeRCTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeRCWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeRCCall struct{ method, path, query, body, version string }

func nativeRCAPI(t *testing.T, calls *[]nativeRCCall, reply func(*http.Request) *http.Response) *resourceclasses.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeRCTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeRCCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("OpenStack-API-Version")})
		return reply(req), nil
	})
	client := cloud.Client("placement", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/placement/"
	client.Microversion = "1.7"
	return resourceclasses.New(client)
}

func nativeRCOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "resourceclasses" {
		t.Fatal("generated resourceclasses context", err, wrapped)
	}
}

func TestNativeResourceClassesRoutesBodiesAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeRCCall
	api := nativeRCAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodPost:
			return nativeRCWire(201, "")
		case req.Method == http.MethodPut:
			return nativeRCWire(204, "")
		case req.Method == http.MethodDelete:
			return nativeRCWire(204, "")
		case req.URL.Path == "/placement/resource_classes":
			return nativeRCWire(200, `{"resource_classes":[{"name":"VCPU","links":[{"href":"/resource_classes/VCPU","rel":"self"}]},{"name":"CUSTOM_GPU"}]}`)
		}
		return nativeRCWire(200, `{"name":"CUSTOM_GPU","links":[{"href":"/resource_classes/CUSTOM_GPU","rel":"self"}]}`)
	})
	if err := api.Create(ctx, resourceclasses.CreateOpts{Name: "CUSTOM_GPU"}, resourceclasses.WithCreateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "CUSTOM_GPU")
	if err != nil || got.Name != "CUSTOM_GPU" || got.Links[0].Rel != "self" {
		t.Fatal(got, err)
	}
	// Update is an idempotent PUT with no body.
	if err := api.Update(ctx, "CUSTOM_GPU"); err != nil {
		t.Fatal(err)
	}
	var names []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, value.Name)
	}
	if err := api.Delete(ctx, "CUSTOM_GPU"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(names, []string{"VCPU", "CUSTOM_GPU"}) {
		t.Fatal(names)
	}
	base := "/placement/resource_classes"
	want := []nativeRCCall{
		{http.MethodPost, base, "", `{"name":"CUSTOM_GPU","x_extension":1}`, "placement 1.7"},
		{http.MethodGet, base + "/CUSTOM_GPU", "", "", "placement 1.7"},
		{http.MethodPut, base + "/CUSTOM_GPU", "", "", "placement 1.7"},
		{http.MethodGet, base, "", "", "placement 1.7"},
		{http.MethodDelete, base + "/CUSTOM_GPU", "", "", "placement 1.7"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeResourceClassesStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*resourceclasses.API) error
	}{
		{"Create", []int{201}, func(api *resourceclasses.API) error {
			return api.Create(ctx, resourceclasses.CreateOpts{Name: "CUSTOM_GPU"})
		}},
		{"Get", []int{200}, func(api *resourceclasses.API) error { _, err := api.Get(ctx, "CUSTOM_GPU"); return err }},
		// Update accepts 201 for a new class and 204 for an existing one.
		{"Update", []int{201, 204}, func(api *resourceclasses.API) error { return api.Update(ctx, "CUSTOM_GPU") }},
		{"Delete", []int{204}, func(api *resourceclasses.API) error { return api.Delete(ctx, "CUSTOM_GPU") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeRCCall
				api := nativeRCAPI(t, &calls, func(*http.Request) *http.Response { return nativeRCWire(code, `{}`) })
				err := call.call(api)
				nativeRCOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("plain object decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			`{}`:          false,
			`null`:        false,
			`{"name":1}`:  true,
			`["VCPU"]`:    true,
			`{"links":1}`: true,
		} {
			var calls []nativeRCCall
			api := nativeRCAPI(t, &calls, func(*http.Request) *http.Response { return nativeRCWire(200, body) })
			got, err := api.Get(ctx, "VCPU")
			if wantErr {
				nativeRCOperation(t, err, "Get")
			} else if err != nil || got == nil || got.Name != "" {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"resource_classes":[]}`}, {204, ""}} {
			var calls []nativeRCCall
			api := nativeRCAPI(t, &calls, func(*http.Request) *http.Response { return nativeRCWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			switch {
			case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
			case tc.code == 200 && len(errs) == 0:
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, errs)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeRCCall
		api := nativeRCAPI(t, &calls, func(*http.Request) *http.Response { return nativeRCWire(201, "") })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create name":      {"Create", api.Create(ctx, resourceclasses.CreateOpts{})},
			"create extension": {"Create", api.Create(ctx, resourceclasses.CreateOpts{Name: "CUSTOM_GPU"}, resourceclasses.WithCreateField("name", "x"))},
			"create nil":       {"Create", api.Create(ctx, resourceclasses.CreateOpts{Name: "CUSTOM_GPU"}, nil)},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeRCOperation(t, check.err, check.operation)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
