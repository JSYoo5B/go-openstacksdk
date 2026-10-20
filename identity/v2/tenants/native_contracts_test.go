package tenants_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeV2TenantTransport func(*http.Request) (*http.Response, error)

func (transport nativeV2TenantTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeV2TenantCall struct{ method, path, query, body string }

func nativeV2TenantAPI(t *testing.T, calls *[]nativeV2TenantCall, reply func(*http.Request) (int, string)) (*tenants.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeV2TenantTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeV2TenantCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v2.0/"
	return tenants.New(client), cloud
}

func nativeV2TenantOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "tenants" {
		t.Fatal("generated tenants context", err, wrapped)
	}
}

const nativeV2TenantRow = `{"id":"t-1","name":"demo","description":"d","enabled":true}`

func TestNativeV2TenantRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeV2TenantCall
	var cloud *testcloud.Cloud
	api, cloud := nativeV2TenantAPI(t, &calls, func(req *http.Request) (int, string) {
		switch {
		case req.Method == http.MethodDelete:
			return 204, ""
		case req.Method == http.MethodPost:
			// Create also accepts 200.
			return 200, `{"tenant":` + nativeV2TenantRow + `}`
		case req.URL.Path == "/keystone/v2.0/tenants":
			// Only a tenants_links rel=next entry is followed; links.next is ignored.
			return 200, `{"tenants":[` + nativeV2TenantRow + `],"links":{"next":"` + cloud.Server.URL + `/never"},"tenants_links":[{"rel":"next","href":"` + cloud.Server.URL + `/other/tenants?marker=t-1"}]}`
		case req.URL.Path == "/other/tenants":
			return 200, `{"tenants":[{"id":"t-2","enabled":false}],"tenants_links":[]}`
		}
		return 200, `{"tenant":` + nativeV2TenantRow + `}`
	})
	enabled, disabled := true, false
	created, err := api.Create(ctx, tenants.CreateOpts{Name: "demo", Description: "d", Enabled: &enabled}, tenants.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "t-1" && created.Enabled && created.Description == "d") {
		t.Fatal(created, err)
	}
	if _, err := api.Create(ctx, tenants.CreateOpts{Name: "bare"}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "t-1")
	if err != nil || got.Name != "demo" {
		t.Fatal(got, err)
	}
	empty := ""
	if _, err := api.Update(ctx, "t-1", tenants.UpdateOpts{Name: "renamed", Description: &empty, Enabled: &disabled}, tenants.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "t-1", tenants.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for value, err := range api.List(ctx, tenants.WithListOptions(tenants.ListOpts{Marker: "t-0", Limit: 1}), tenants.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if err := api.Delete(ctx, "t-1"); err != nil {
		t.Fatal(err)
	}
	base := "/keystone/v2.0/tenants"
	want := []nativeV2TenantCall{
		{http.MethodPost, base, "", `{"tenant":{"description":"d","enabled":true,"name":"demo","x_extension":1}}`},
		{http.MethodPost, base, "", `{"tenant":{"name":"bare"}}`},
		{http.MethodGet, base + "/t-1", "", ""},
		// Description is a pointer on update, so an empty string is still sent.
		{http.MethodPut, base + "/t-1", "", `{"tenant":{"description":"","enabled":false,"name":"renamed","x_extension":1}}`},
		{http.MethodPut, base + "/t-1", "", `{"tenant":{}}`},
		{http.MethodGet, base, "extra=1&limit=1&marker=t-0", ""},
		{http.MethodGet, "/other/tenants", "marker=t-1", ""},
		{http.MethodDelete, base + "/t-1", "", ""},
	}
	if !reflect.DeepEqual(ids, []string{"t-1", "t-2"}) || !reflect.DeepEqual(calls, want) {
		t.Fatalf("%v %+v", ids, calls)
	}
}

func TestNativeV2TenantStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	valid := tenants.CreateOpts{Name: "demo"}
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*tenants.API) error
	}{
		{"Create", []int{200, 201}, func(api *tenants.API) error { _, err := api.Create(ctx, valid); return err }},
		{"Get", []int{200}, func(api *tenants.API) error { _, err := api.Get(ctx, "t-1"); return err }},
		{"Update", []int{200}, func(api *tenants.API) error {
			_, err := api.Update(ctx, "t-1", tenants.UpdateOpts{Name: "x"})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *tenants.API) error { return api.Delete(ctx, "t-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeV2TenantCall
				api, _ := nativeV2TenantAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
				err := call.call(api)
				nativeV2TenantOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{`{}`: false, `{"tenant":null}`: false, `{"tenant":[]}`: true} {
			var calls []nativeV2TenantCall
			api, _ := nativeV2TenantAPI(t, &calls, func(*http.Request) (int, string) { return 200, body })
			got, err := api.Get(ctx, "t-1")
			if wantErr {
				nativeV2TenantOperation(t, err, "Get")
			} else if err != nil || got != nil {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"tenants":[]}`}, {204, ""}} {
			var calls []nativeV2TenantCall
			api, _ := nativeV2TenantAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, tc.body })
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
		var calls []nativeV2TenantCall
		api, _ := nativeV2TenantAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create name":      {"Create", func() error { _, err := api.Create(ctx, tenants.CreateOpts{Description: "d"}); return err }()},
			"create extension": {"Create", func() error { _, err := api.Create(ctx, valid, tenants.WithCreateField("enabled", true)); return err }()},
			"create nil":       {"Create", func() error { _, err := api.Create(ctx, valid, nil); return err }()},
			"update extension": {"Update", func() error {
				_, err := api.Update(ctx, "t-1", tenants.UpdateOpts{}, tenants.WithUpdateField("description", "x"))
				return err
			}()},
			"update nil": {"Update", func() error { _, err := api.Update(ctx, "t-1", tenants.UpdateOpts{}, nil); return err }()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeV2TenantOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeV2TenantOperation(t, err, "List")
		}
		for _, err := range api.List(ctx, tenants.WithListQuery(" ", "x")) {
			nativeV2TenantOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
