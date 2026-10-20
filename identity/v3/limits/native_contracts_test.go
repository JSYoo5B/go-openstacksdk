package limits_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/limits"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeLimitTransport func(*http.Request) (*http.Response, error)

func (transport nativeLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeLimitWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeLimitCall struct{ method, path, query, body string }

func nativeLimitAPI(t *testing.T, calls *[]nativeLimitCall, reply func(*http.Request) *http.Response) (*limits.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeLimitTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeLimitCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return limits.New(client), cloud
}

func nativeLimitOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "limits" {
		t.Fatal("generated limits context", err, wrapped)
	}
}

const nativeLimitRow = `{"id":"lim-1","region_id":"RegionOne","project_id":"p","domain_id":"","service_id":"svc-1","description":"d","resource_name":"cores","resource_limit":10,"links":{"self":"x"}}`

var nativeLimitCreate = limits.CreateOpts{ServiceID: "svc-1", ResourceName: "cores", ProjectID: "p"}

func TestNativeLimitRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeLimitCall
	var cloud *testcloud.Cloud
	api, cloud := nativeLimitAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeLimitWire(204, "")
		case req.Method == http.MethodPost:
			return nativeLimitWire(201, `{"limits":[`+nativeLimitRow+`,{"id":"lim-2","resource_limit":0}]}`)
		case req.URL.Path == "/keystone/v3/limits/model":
			return nativeLimitWire(200, `{"model":{"name":"flat","description":"Limit enforcement and validation does not take project hierarchy into consideration."}}`)
		case req.URL.Path == "/keystone/v3/limits":
			return nativeLimitWire(200, `{"limits":[`+nativeLimitRow+`],"links":{"next":"`+cloud.Server.URL+`/other/limits?page=2"}}`)
		case req.URL.Path == "/other/limits":
			return nativeLimitWire(200, `{"limits":[{"id":"lim-2"}],"links":{"next":null}}`)
		}
		return nativeLimitWire(200, `{"limit":`+nativeLimitRow+`}`)
	})
	second := limits.CreateOpts{ServiceID: "svc-1", ResourceName: "ram", DomainID: "d", RegionID: "RegionOne", Description: "x", ResourceLimit: 5}
	created, err := api.BatchCreate(ctx, limits.BatchCreateOpts{nativeLimitCreate, second}, limits.WithBatchCreateField("x_extension", 1))
	if err != nil || !(len(created) == 2 && created[0].ID == "lim-1" && created[0].ResourceLimit == 10 && created[1].ID == "lim-2") {
		t.Fatal(created, err)
	}
	// An empty batch is not rejected and sends an empty array.
	if _, err := api.BatchCreate(ctx, nil); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "lim-1")
	if err != nil || got.ResourceName != "cores" {
		t.Fatal(got, err)
	}
	model, err := api.GetEnforcementModel(ctx)
	if err != nil || model.Name != "flat" {
		t.Fatal(model, err)
	}
	zero, empty := 0, ""
	if _, err := api.Update(ctx, "lim-1", limits.UpdateOpts{Description: &empty, ResourceLimit: &zero}, limits.WithUpdateField("x_extension", 2)); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for value, err := range api.List(ctx, limits.WithListOptions(limits.ListOpts{RegionID: "RegionOne", ProjectID: "p", DomainID: "d", ServiceID: "svc-1", ResourceName: "cores"}), limits.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"lim-1", "lim-2"}) {
		t.Fatal(ids)
	}
	if err := api.Delete(ctx, "lim-1"); err != nil {
		t.Fatal(err)
	}
	base := "/keystone/v3/limits"
	want := []nativeLimitCall{
		// Rows have no per-limit envelope, resource_limit is always sent, and extensions land at the root beside limits.
		{http.MethodPost, base, "", `{"limits":[{"project_id":"p","resource_limit":0,"resource_name":"cores","service_id":"svc-1"},{"description":"x","domain_id":"d","region_id":"RegionOne","resource_limit":5,"resource_name":"ram","service_id":"svc-1"}],"x_extension":1}`},
		{http.MethodPost, base, "", `{"limits":[]}`},
		{http.MethodGet, base + "/lim-1", "", ""},
		{http.MethodGet, base + "/model", "", ""},
		{http.MethodPatch, base + "/lim-1", "", `{"limit":{"description":"","resource_limit":0,"x_extension":2}}`},
		{http.MethodGet, base, "domain_id=d&extra=1&project_id=p&region_id=RegionOne&resource_name=cores&service_id=svc-1", ""},
		{http.MethodGet, "/other/limits", "page=2", ""},
		{http.MethodDelete, base + "/lim-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeLimitStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*limits.API) error
	}{
		{"BatchCreate", []int{201}, func(api *limits.API) error {
			_, err := api.BatchCreate(ctx, limits.BatchCreateOpts{nativeLimitCreate})
			return err
		}},
		{"Get", []int{200}, func(api *limits.API) error { _, err := api.Get(ctx, "lim-1"); return err }},
		{"GetEnforcementModel", []int{200}, func(api *limits.API) error { _, err := api.GetEnforcementModel(ctx); return err }},
		{"Update", []int{200}, func(api *limits.API) error { _, err := api.Update(ctx, "lim-1", limits.UpdateOpts{}); return err }},
		{"Delete", []int{202, 204}, func(api *limits.API) error { return api.Delete(ctx, "lim-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeLimitCall
				api, _ := nativeLimitAPI(t, &calls, func(*http.Request) *http.Response { return nativeLimitWire(code, `{}`) })
				err := call.call(api)
				nativeLimitOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			`{}`:             false,
			`{"limit":null}`: false,
			`{"limit":[]}`:   true,
		} {
			var calls []nativeLimitCall
			api, _ := nativeLimitAPI(t, &calls, func(*http.Request) *http.Response { return nativeLimitWire(200, body) })
			got, err := api.Get(ctx, "lim-1")
			if wantErr {
				nativeLimitOperation(t, err, "Get")
			} else if err != nil || got != nil {
				t.Fatal(body, got, err)
			}
		}
		var calls []nativeLimitCall
		api, _ := nativeLimitAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.Method == http.MethodPost {
				return nativeLimitWire(201, `{}`)
			}
			return nativeLimitWire(200, `{"model":null}`)
		})
		// A batch reply without limits and a null model both decode to nil without an error.
		created, err := api.BatchCreate(ctx, limits.BatchCreateOpts{nativeLimitCreate})
		model, modelErr := api.GetEnforcementModel(ctx)
		if err != nil || created != nil || modelErr != nil || model != nil {
			t.Fatal(created, err, model, modelErr)
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"limits":[]}`}, {204, ""}} {
			var calls []nativeLimitCall
			api, _ := nativeLimitAPI(t, &calls, func(*http.Request) *http.Response { return nativeLimitWire(tc.code, tc.body) })
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
		var calls []nativeLimitCall
		api, _ := nativeLimitAPI(t, &calls, func(*http.Request) *http.Response { return nativeLimitWire(201, `{}`) })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"service": {"BatchCreate", func() error {
				_, err := api.BatchCreate(ctx, limits.BatchCreateOpts{nativeLimitCreate, {ResourceName: "ram"}})
				return err
			}()},
			"resource name": {"BatchCreate", func() error {
				_, err := api.BatchCreate(ctx, limits.BatchCreateOpts{{ServiceID: "svc-1"}})
				return err
			}()},
			// The batch type is a slice, so only the existing root limits key is protected.
			"limits extension": {"BatchCreate", func() error {
				_, err := api.BatchCreate(ctx, limits.BatchCreateOpts{nativeLimitCreate}, limits.WithBatchCreateField("limits", nil))
				return err
			}()},
			"update core extension": {"Update", func() error {
				_, err := api.Update(ctx, "lim-1", limits.UpdateOpts{}, limits.WithUpdateField("resource_limit", 1))
				return err
			}()},
			"batch nil option": {"BatchCreate", func() error {
				_, err := api.BatchCreate(ctx, limits.BatchCreateOpts{nativeLimitCreate}, nil)
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeLimitOperation(t, check.err, check.operation)
		}
		var missing gophercloud.ErrMissingInput
		if _, err := api.BatchCreate(ctx, limits.BatchCreateOpts{{ResourceName: "ram"}}); !errors.As(err, &missing) || missing.Argument != "ServiceID" {
			t.Fatal(err)
		}
		for _, err := range api.List(ctx, nil) {
			nativeLimitOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
