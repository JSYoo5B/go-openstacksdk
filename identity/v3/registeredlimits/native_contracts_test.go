package registeredlimits_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/registeredlimits"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeRegisteredLimitTransport func(*http.Request) (*http.Response, error)

func (transport nativeRegisteredLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeRegisteredLimitWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeRegisteredLimitCall struct{ method, path, query, body string }

func nativeRegisteredLimitAPI(t *testing.T, calls *[]nativeRegisteredLimitCall, reply func(*http.Request) *http.Response) (*registeredlimits.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeRegisteredLimitTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeRegisteredLimitCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return registeredlimits.New(client), cloud
}

func nativeRegisteredLimitOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "registeredlimits" {
		t.Fatal("generated registeredlimits context", err, wrapped)
	}
}

const nativeRegisteredLimitRow = `{"id":"rl-1","region_id":"RegionOne","service_id":"svc-1","description":"d","resource_name":"cores","default_limit":10,"links":{"self":"x"}}`

var nativeRegisteredLimitCreate = registeredlimits.CreateOpts{ServiceID: "svc-1", ResourceName: "cores"}

func TestNativeRegisteredLimitRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeRegisteredLimitCall
	var cloud *testcloud.Cloud
	api, cloud := nativeRegisteredLimitAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeRegisteredLimitWire(204, "")
		case req.Method == http.MethodPost:
			return nativeRegisteredLimitWire(201, `{"registered_limits":[`+nativeRegisteredLimitRow+`,{"id":"rl-2"}]}`)
		case req.URL.Path == "/keystone/v3/registered_limits":
			return nativeRegisteredLimitWire(200, `{"registered_limits":[`+nativeRegisteredLimitRow+`],"links":{"next":"`+cloud.Server.URL+`/other/registered?page=2"}}`)
		case req.URL.Path == "/other/registered":
			return nativeRegisteredLimitWire(200, `{"registered_limits":[{"id":"rl-2"}],"links":{"next":null}}`)
		}
		return nativeRegisteredLimitWire(200, `{"registered_limit":`+nativeRegisteredLimitRow+`}`)
	})
	second := registeredlimits.CreateOpts{ServiceID: "svc-1", ResourceName: "ram", RegionID: "RegionOne", Description: "x", DefaultLimit: 5}
	created, err := api.BatchCreate(ctx, registeredlimits.BatchCreateOpts{nativeRegisteredLimitCreate, second}, registeredlimits.WithBatchCreateField("x_extension", 1))
	if err != nil || !(len(created) == 2 && created[0].ID == "rl-1" && created[0].DefaultLimit == 10 && created[1].ID == "rl-2") {
		t.Fatal(created, err)
	}
	if _, err := api.BatchCreate(ctx, registeredlimits.BatchCreateOpts{}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "rl-1")
	if err != nil || got.ResourceName != "cores" {
		t.Fatal(got, err)
	}
	zero, empty := 0, ""
	if _, err := api.Update(ctx, "rl-1", registeredlimits.UpdateOpts{Description: &empty, DefaultLimit: &zero, RegionID: "RegionTwo", ServiceID: "svc-2", ResourceName: "ram"}, registeredlimits.WithUpdateField("x_extension", 2)); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for value, err := range api.List(ctx, registeredlimits.WithListOptions(registeredlimits.ListOpts{RegionID: "RegionOne", ServiceID: "svc-1", ResourceName: "cores"}), registeredlimits.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"rl-1", "rl-2"}) {
		t.Fatal(ids)
	}
	if err := api.Delete(ctx, "rl-1"); err != nil {
		t.Fatal(err)
	}
	base := "/keystone/v3/registered_limits"
	want := []nativeRegisteredLimitCall{
		// default_limit is required but numeric, so 0 passes and is always sent; extensions land at the root.
		{http.MethodPost, base, "", `{"registered_limits":[{"default_limit":0,"resource_name":"cores","service_id":"svc-1"},{"default_limit":5,"description":"x","region_id":"RegionOne","resource_name":"ram","service_id":"svc-1"}],"x_extension":1}`},
		{http.MethodPost, base, "", `{"registered_limits":[]}`},
		{http.MethodGet, base + "/rl-1", "", ""},
		{http.MethodPatch, base + "/rl-1", "", `{"registered_limit":{"default_limit":0,"description":"","region_id":"RegionTwo","resource_name":"ram","service_id":"svc-2","x_extension":2}}`},
		{http.MethodGet, base, "extra=1&region_id=RegionOne&resource_name=cores&service_id=svc-1", ""},
		{http.MethodGet, "/other/registered", "page=2", ""},
		{http.MethodDelete, base + "/rl-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeRegisteredLimitStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*registeredlimits.API) error
	}{
		{"BatchCreate", []int{201}, func(api *registeredlimits.API) error {
			_, err := api.BatchCreate(ctx, registeredlimits.BatchCreateOpts{nativeRegisteredLimitCreate})
			return err
		}},
		{"Get", []int{200}, func(api *registeredlimits.API) error { _, err := api.Get(ctx, "rl-1"); return err }},
		{"Update", []int{200}, func(api *registeredlimits.API) error {
			_, err := api.Update(ctx, "rl-1", registeredlimits.UpdateOpts{})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *registeredlimits.API) error { return api.Delete(ctx, "rl-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeRegisteredLimitCall
				api, _ := nativeRegisteredLimitAPI(t, &calls, func(*http.Request) *http.Response { return nativeRegisteredLimitWire(code, `{}`) })
				err := call.call(api)
				nativeRegisteredLimitOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			`{}`:                        false,
			`{"registered_limit":null}`: false,
			`{"registered_limit":[]}`:   true,
		} {
			var calls []nativeRegisteredLimitCall
			api, _ := nativeRegisteredLimitAPI(t, &calls, func(*http.Request) *http.Response { return nativeRegisteredLimitWire(200, body) })
			got, err := api.Get(ctx, "rl-1")
			if wantErr {
				nativeRegisteredLimitOperation(t, err, "Get")
			} else if err != nil || got != nil {
				t.Fatal(body, got, err)
			}
		}
		var calls []nativeRegisteredLimitCall
		api, _ := nativeRegisteredLimitAPI(t, &calls, func(*http.Request) *http.Response { return nativeRegisteredLimitWire(201, `{}`) })
		if created, err := api.BatchCreate(ctx, registeredlimits.BatchCreateOpts{nativeRegisteredLimitCreate}); err != nil || created != nil {
			t.Fatal(created, err)
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"registered_limits":[]}`}, {204, ""}} {
			var calls []nativeRegisteredLimitCall
			api, _ := nativeRegisteredLimitAPI(t, &calls, func(*http.Request) *http.Response { return nativeRegisteredLimitWire(tc.code, tc.body) })
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
		var calls []nativeRegisteredLimitCall
		api, _ := nativeRegisteredLimitAPI(t, &calls, func(*http.Request) *http.Response { return nativeRegisteredLimitWire(201, `{}`) })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"service": {"BatchCreate", func() error {
				_, err := api.BatchCreate(ctx, registeredlimits.BatchCreateOpts{{ResourceName: "ram"}})
				return err
			}()},
			"resource name": {"BatchCreate", func() error {
				_, err := api.BatchCreate(ctx, registeredlimits.BatchCreateOpts{nativeRegisteredLimitCreate, {ServiceID: "svc-1"}})
				return err
			}()},
			"registered_limits extension": {"BatchCreate", func() error {
				_, err := api.BatchCreate(ctx, registeredlimits.BatchCreateOpts{nativeRegisteredLimitCreate}, registeredlimits.WithBatchCreateField("registered_limits", nil))
				return err
			}()},
			"update core extension": {"Update", func() error {
				_, err := api.Update(ctx, "rl-1", registeredlimits.UpdateOpts{}, registeredlimits.WithUpdateField("default_limit", 1))
				return err
			}()},
			"update nil option": {"Update", func() error {
				_, err := api.Update(ctx, "rl-1", registeredlimits.UpdateOpts{}, nil)
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeRegisteredLimitOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeRegisteredLimitOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
