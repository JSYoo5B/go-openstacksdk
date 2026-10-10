package endpoints_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/endpoints"
)

type nativeEndpointTransport func(*http.Request) (*http.Response, error)

func (transport nativeEndpointTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeEndpointWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeEndpointCall struct{ method, path, query, body string }

func nativeEndpointAPI(t *testing.T, calls *[]nativeEndpointCall, reply func(*http.Request) *http.Response) (*endpoints.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeEndpointTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeEndpointCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return endpoints.New(client), cloud
}

func nativeEndpointOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "endpoints" {
		t.Fatal("generated endpoints context", err, wrapped)
	}
}

const nativeEndpointRow = `{"id":"ep-1","interface":"public","name":"nova","region":"RegionOne","region_id":"RegionOne","service_id":"svc-1","url":"https://nova","enabled":true,"description":"d","links":{"self":"x"}}`

var nativeEndpointCreate = endpoints.CreateOpts{Availability: gophercloud.AvailabilityPublic, URL: "https://nova", ServiceID: "svc-1"}

func TestNativeEndpointRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeEndpointCall
	var cloud *testcloud.Cloud
	api, cloud := nativeEndpointAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeEndpointWire(204, "")
		case req.Method == http.MethodPost:
			// Create has no explicit OkCodes, so the POST default also accepts 202.
			return nativeEndpointWire(202, `{"endpoint":`+nativeEndpointRow+`}`)
		case req.Method == http.MethodPatch:
			// Update uses the PATCH default; a bodyless 204 succeeds with a nil endpoint.
			return nativeEndpointWire(204, "")
		case req.URL.Path == "/keystone/v3/endpoints":
			return nativeEndpointWire(200, `{"endpoints":[`+nativeEndpointRow+`],"links":{"next":"`+cloud.Server.URL+`/other/endpoints?page=2"}}`)
		case req.URL.Path == "/other/endpoints":
			return nativeEndpointWire(200, `{"endpoints":[{"id":"ep-2","interface":"internal"}],"links":{"next":null}}`)
		}
		return nativeEndpointWire(200, `{"endpoint":`+nativeEndpointRow+`}`)
	})
	enabled := false
	create := nativeEndpointCreate
	create.Enabled, create.Region, create.Name, create.Description = &enabled, "RegionOne", "nova", "d"
	created, err := api.Create(ctx, create, endpoints.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "ep-1" && created.Availability == gophercloud.AvailabilityPublic && created.RegionID == "RegionOne" && created.Enabled) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "ep-1")
	if err != nil || got.URL != "https://nova" {
		t.Fatal(got, err)
	}
	updated, err := api.Update(ctx, "ep-1", endpoints.UpdateOpts{Availability: gophercloud.AvailabilityInternal, Enabled: &enabled}, endpoints.WithUpdateField("x_extension", 2))
	if err != nil || updated != nil {
		t.Fatal(updated, err)
	}
	var rows []*endpoints.Endpoint
	for value, err := range api.List(ctx, endpoints.WithListOptions(endpoints.ListOpts{Availability: gophercloud.AvailabilityPublic, ServiceID: "svc-1", RegionID: "RegionOne"}), endpoints.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "ep-1" && rows[1].Availability == gophercloud.AvailabilityInternal) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "ep-1"); err != nil {
		t.Fatal(err)
	}
	base := "/keystone/v3/endpoints"
	want := []nativeEndpointCall{
		{http.MethodPost, base, "", `{"endpoint":{"description":"d","enabled":false,"interface":"public","name":"nova","region":"RegionOne","service_id":"svc-1","url":"https://nova","x_extension":1}}`},
		{http.MethodGet, base + "/ep-1", "", ""},
		{http.MethodPatch, base + "/ep-1", "", `{"endpoint":{"enabled":false,"interface":"internal","x_extension":2}}`},
		// Native List runs BuildQueryString on the generated builder instead of calling
		// ToEndpointListParams, so typed filters and WithListQuery are silently dropped.
		{http.MethodGet, base, "", ""},
		{http.MethodGet, "/other/endpoints", "page=2", ""},
		{http.MethodDelete, base + "/ep-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeEndpointStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*endpoints.API) error
	}{
		{"Create", []int{201, 202}, func(api *endpoints.API) error { _, err := api.Create(ctx, nativeEndpointCreate); return err }},
		{"Get", []int{200}, func(api *endpoints.API) error { _, err := api.Get(ctx, "ep-1"); return err }},
		{"Update", []int{200, 202, 204}, func(api *endpoints.API) error {
			_, err := api.Update(ctx, "ep-1", endpoints.UpdateOpts{})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *endpoints.API) error { return api.Delete(ctx, "ep-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeEndpointCall
				api, _ := nativeEndpointAPI(t, &calls, func(*http.Request) *http.Response { return nativeEndpointWire(code, `{}`) })
				err := call.call(api)
				nativeEndpointOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			`{}`:                false,
			`{"endpoint":null}`: false,
			`{"endpoint":[]}`:   true,
		} {
			var calls []nativeEndpointCall
			api, _ := nativeEndpointAPI(t, &calls, func(*http.Request) *http.Response { return nativeEndpointWire(200, body) })
			got, err := api.Get(ctx, "ep-1")
			if wantErr {
				nativeEndpointOperation(t, err, "Get")
			} else if err != nil || got != nil {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"endpoints":[]}`}, {204, ""}} {
			var calls []nativeEndpointCall
			api, _ := nativeEndpointAPI(t, &calls, func(*http.Request) *http.Response { return nativeEndpointWire(tc.code, tc.body) })
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
		var calls []nativeEndpointCall
		api, _ := nativeEndpointAPI(t, &calls, func(*http.Request) *http.Response { return nativeEndpointWire(201, `{}`) })
		missing := func(edit func(*endpoints.CreateOpts)) error {
			opts := nativeEndpointCreate
			edit(&opts)
			_, err := api.Create(ctx, opts)
			return err
		}
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create interface": {"Create", missing(func(o *endpoints.CreateOpts) { o.Availability = "" })},
			"create url":       {"Create", missing(func(o *endpoints.CreateOpts) { o.URL = "" })},
			"create service":   {"Create", missing(func(o *endpoints.CreateOpts) { o.ServiceID = "" })},
			"create core extension": {"Create", func() error {
				_, err := api.Create(ctx, nativeEndpointCreate, endpoints.WithCreateField("region", "x"))
				return err
			}()},
			"update core extension": {"Update", func() error {
				_, err := api.Update(ctx, "ep-1", endpoints.UpdateOpts{}, endpoints.WithUpdateField("url", "x"))
				return err
			}()},
			"update nil option": {"Update", func() error {
				_, err := api.Update(ctx, "ep-1", endpoints.UpdateOpts{}, nil)
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeEndpointOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeEndpointOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}

// The upstream List serializes a plain ListOpts correctly; only the generated
// builder wrapper loses the filters, so the facade List is pinned as unresolved.
func TestNativeEndpointListFiltersDroppedByGeneratedBuilder(t *testing.T) {
	ctx := context.Background()
	var calls []nativeEndpointCall
	api, _ := nativeEndpointAPI(t, &calls, func(*http.Request) *http.Response { return nativeEndpointWire(200, `{"endpoints":[]}`) })
	filters := endpoints.ListOpts{Availability: gophercloud.AvailabilityAdmin, ServiceID: "svc-1"}
	if _, err := upstream.List(api.RawClient(), filters).AllPages(ctx); err != nil {
		t.Fatal(err)
	}
	for _, err := range api.List(ctx, endpoints.WithListOptions(filters)) {
		t.Fatal(err)
	}
	for _, err := range api.List(ctx, endpoints.WithListQuery("interface", "admin")) {
		t.Fatal(err)
	}
	want := []nativeEndpointCall{
		{http.MethodGet, "/keystone/v3/endpoints", "interface=admin&service_id=svc-1", ""},
		{http.MethodGet, "/keystone/v3/endpoints", "", ""},
		{http.MethodGet, "/keystone/v3/endpoints", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}
