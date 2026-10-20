package services_test

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
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/services"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeComputeServiceTransport func(*http.Request) (*http.Response, error)

func (transport nativeComputeServiceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeComputeServiceWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeComputeServiceCall struct{ method, path, query, body string }

func nativeComputeServiceAPI(t *testing.T, calls *[]nativeComputeServiceCall, reply func(*http.Request) *http.Response) *services.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeComputeServiceTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeComputeServiceCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	return services.New(client)
}

func nativeComputeServiceOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "services" {
		t.Fatal("generated services context", err, wrapped)
	}
}

func TestNativeComputeServiceRoutesBodiesAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeComputeServiceCall
	api := nativeComputeServiceAPI(t, &calls, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodDelete:
			return nativeComputeServiceWire(204, "")
		case http.MethodPut:
			return nativeComputeServiceWire(200, `{"service":{"id":"svc-uuid","binary":"nova-compute","host":"cmp1","status":"disabled","disabled_reason":"maint","forced_down":true,"state":"down","zone":"nova","updated_at":"2026-10-01T01:02:03.000000"}}`)
		}
		// 2.53+ rows use UUID strings; older rows use integers. The links array is never followed.
		return nativeComputeServiceWire(200, `{"services":[{"id":"svc-uuid","binary":"nova-compute","host":"cmp1","updated_at":null},{"id":4,"binary":"nova-scheduler","host":"ctl"}],"services_links":[{"rel":"next","href":"http://ignored/"}]}`)
	})
	var ids []string
	for value, err := range api.List(ctx, services.WithListOptions(services.ListOpts{Binary: "nova-compute", Host: "cmp1"}), services.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"svc-uuid", "4"}) {
		t.Fatal(ids)
	}
	updated, err := api.Update(ctx, "svc-uuid", services.UpdateOpts{Status: services.ServiceDisabled, DisabledReason: "maint", ForcedDown: true}, services.WithUpdateField("x_extension", 1))
	if err != nil || updated.Status != "disabled" || !updated.ForcedDown || updated.Zone != "nova" || !updated.UpdatedAt.Equal(time.Date(2026, 10, 1, 1, 2, 3, 0, time.UTC)) {
		t.Fatal(updated, err)
	}
	// forced_down=false is omitted, so it cannot be cleared through UpdateOpts.
	if _, err := api.Update(ctx, "svc-uuid", services.UpdateOpts{Status: services.ServiceEnabled}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "svc-uuid", services.UpdateOpts{}, services.WithUpdateField("forced_down_extra", false)); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "svc-uuid"); err != nil {
		t.Fatal(err)
	}
	base := "/nova/v2.1/os-services"
	want := []nativeComputeServiceCall{
		{http.MethodGet, base, "binary=nova-compute&extra=1&host=cmp1", ""},
		// The update body has no envelope; extensions sit at the root.
		{http.MethodPut, base + "/svc-uuid", "", `{"disabled_reason":"maint","forced_down":true,"status":"disabled","x_extension":1}`},
		{http.MethodPut, base + "/svc-uuid", "", `{"status":"enabled"}`},
		{http.MethodPut, base + "/svc-uuid", "", `{"forced_down_extra":false}`},
		{http.MethodDelete, base + "/svc-uuid", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeComputeServiceStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*services.API) error
	}{
		{"Update", []int{200}, func(api *services.API) error {
			_, err := api.Update(ctx, "svc", services.UpdateOpts{Status: services.ServiceEnabled})
			return err
		}},
		// Delete accepts only 204.
		{"Delete", []int{204}, func(api *services.API) error { return api.Delete(ctx, "svc") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeComputeServiceCall
				api := nativeComputeServiceAPI(t, &calls, func(*http.Request) *http.Response { return nativeComputeServiceWire(code, `{}`) })
				err := call.call(api)
				nativeComputeServiceOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			// A missing envelope is a zero service; a present object (even null) needs an id.
			`{}`:                      false,
			`{"service":null}`:        true,
			`{"service":{}}`:          true,
			`{"service":{"id":true}}`: true,
			`{"service":{"id":"s","updated_at":"2026-10-01T01:02:03Z"}}`: true,
			`{"service":{"id":2.0}}`: false,
		} {
			var calls []nativeComputeServiceCall
			api := nativeComputeServiceAPI(t, &calls, func(*http.Request) *http.Response { return nativeComputeServiceWire(200, body) })
			_, err := api.Update(ctx, "svc", services.UpdateOpts{})
			if wantErr {
				nativeComputeServiceOperation(t, err, "Update")
			} else if err != nil {
				t.Fatal(body, err)
			}
		}
	})
	t.Run("list pager status, empty page, bad row and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"services":[]}`}, {200, `{"services":[{"host":"h"}]}`}, {204, ""}} {
			var calls []nativeComputeServiceCall
			api := nativeComputeServiceAPI(t, &calls, func(*http.Request) *http.Response { return nativeComputeServiceWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			var wrapped *resource.OperationError
			switch {
			case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}) && !errors.As(errs[0], &wrapped):
			case tc.code == 200 && strings.Contains(tc.body, "[]") && len(errs) == 0:
			case tc.code == 200 && len(errs) == 1 && errs[0] != nil && strings.Contains(errs[0].Error(), "ID has unexpected type"):
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, tc.body, errs)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeComputeServiceCall
		api := nativeComputeServiceAPI(t, &calls, func(*http.Request) *http.Response { return nativeComputeServiceWire(200, `{}`) })
		// Omitted core keys are still reserved.
		_, err := api.Update(ctx, "svc", services.UpdateOpts{}, services.WithUpdateField("forced_down", false))
		nativeComputeServiceOperation(t, err, "Update")
		_, err = api.Update(ctx, "svc", services.UpdateOpts{}, nil)
		nativeComputeServiceOperation(t, err, "Update")
		for _, err := range api.List(ctx, nil) {
			nativeComputeServiceOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
