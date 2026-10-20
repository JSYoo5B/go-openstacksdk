package projectendpoints_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeProjectEndpointTransport func(*http.Request) (*http.Response, error)

func (transport nativeProjectEndpointTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeProjectEndpointWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeProjectEndpointCall struct{ method, path, query, body string }

func nativeProjectEndpointAPI(t *testing.T, calls *[]nativeProjectEndpointCall, reply func(*http.Request) *http.Response) (*projectendpoints.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeProjectEndpointTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeProjectEndpointCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return projectendpoints.New(client), cloud
}

func nativeProjectEndpointOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "projectendpoints" {
		t.Fatal("generated projectendpoints context", err, wrapped)
	}
}

func TestNativeProjectEndpointRoutesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeProjectEndpointCall
	var cloud *testcloud.Cloud
	api, cloud := nativeProjectEndpointAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodPut || req.Method == http.MethodDelete:
			return nativeProjectEndpointWire(204, "")
		case req.URL.Path == "/other/project-endpoints":
			return nativeProjectEndpointWire(200, `{"endpoints":[{"id":"ep-2","interface":"internal"}],"links":{"next":null}}`)
		}
		return nativeProjectEndpointWire(200, `{"endpoints":[{"id":"ep-1","interface":"public","region":"RegionOne","service_id":"svc-1","url":"https://nova","enabled":true}],"links":{"next":"`+cloud.Server.URL+`/other/project-endpoints?page=2"}}`)
	})
	if err := api.Create(ctx, "p-1", "ep-1"); err != nil {
		t.Fatal(err)
	}
	var rows []*projectendpoints.Endpoint
	for value, err := range api.List(ctx, "p-1") {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "ep-1" && rows[0].Availability == gophercloud.AvailabilityPublic && rows[0].URL == "https://nova" && rows[1].Availability == gophercloud.AvailabilityInternal) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "p-1", "ep-1"); err != nil {
		t.Fatal(err)
	}
	base := "/keystone/v3/OS-EP-FILTER/projects/p-1/endpoints"
	want := []nativeProjectEndpointCall{
		// The association PUT has no request body.
		{http.MethodPut, base + "/ep-1", "", ""},
		{http.MethodGet, base, "", ""},
		{http.MethodGet, "/other/project-endpoints", "page=2", ""},
		{http.MethodDelete, base + "/ep-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeProjectEndpointStatuses(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*projectendpoints.API) error
	}{
		// Both association calls pin OkCodes to 204 only.
		{"Create", []int{204}, func(api *projectendpoints.API) error { return api.Create(ctx, "p-1", "ep-1") }},
		{"Delete", []int{204}, func(api *projectendpoints.API) error { return api.Delete(ctx, "p-1", "ep-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeProjectEndpointCall
				api, _ := nativeProjectEndpointAPI(t, &calls, func(*http.Request) *http.Response { return nativeProjectEndpointWire(code, `{}`) })
				err := call.call(api)
				nativeProjectEndpointOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"endpoints":[]}`}, {204, ""}} {
			var calls []nativeProjectEndpointCall
			api, _ := nativeProjectEndpointAPI(t, &calls, func(*http.Request) *http.Response { return nativeProjectEndpointWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.List(ctx, "p-1") {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			var wrapped *resource.OperationError
			switch {
			// List streams carry the native error without request.Wrap context.
			case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}) && !errors.As(errs[0], &wrapped):
			case tc.code == 200 && len(errs) == 0:
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, errs)
			}
		}
	})
}
