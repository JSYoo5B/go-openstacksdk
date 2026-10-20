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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/services"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeServiceTransport func(*http.Request) (*http.Response, error)

func (transport nativeServiceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeServiceWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeServiceCall struct{ method, path, query, body string }

func nativeServiceAPI(t *testing.T, calls *[]nativeServiceCall, reply func(*http.Request) *http.Response) (*services.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeServiceTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeServiceCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return services.New(client), cloud
}

func nativeServiceOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "services" {
		t.Fatal("generated services context", err, wrapped)
	}
}

const nativeServiceRow = `{"id":"svc-1","name":"nova","description":"compute","type":"compute","enabled":true,"links":{"self":"x"},"owner":"ops"}`

func TestNativeServiceRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeServiceCall
	var cloud *testcloud.Cloud
	api, cloud := nativeServiceAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeServiceWire(204, "")
		case req.Method == http.MethodPost:
			return nativeServiceWire(201, `{"service":`+nativeServiceRow+`}`)
		case req.URL.Path == "/keystone/v3/services":
			return nativeServiceWire(200, `{"services":[`+nativeServiceRow+`],"links":{"next":"`+cloud.Server.URL+`/other/services?page=2"}}`)
		case req.URL.Path == "/other/services":
			return nativeServiceWire(200, `{"services":[{"id":"svc-2","type":"image","enabled":false,"extra":{"k":"v"}}],"links":{"next":null}}`)
		}
		return nativeServiceWire(200, `{"service":`+nativeServiceRow+`}`)
	})
	enabled := false
	created, err := api.Create(ctx, services.CreateOpts{Type: "compute", Name: "nova", Enabled: &enabled, Extra: map[string]any{"owner": "ops"}}, services.WithCreateField("x_extension", 1))
	// Without an explicit extra object, Extra holds unknown keys plus copies of name and description.
	if err != nil || !(created.ID == "svc-1" && created.Enabled && reflect.DeepEqual(created.Extra, map[string]any{"owner": "ops", "name": "nova", "description": "compute"})) {
		t.Fatal(created, err)
	}
	// Type has no omitempty or required tag, so an empty type is sent.
	if _, err := api.Create(ctx, services.CreateOpts{}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "svc-1")
	if err != nil || got.Type != "compute" {
		t.Fatal(got, err)
	}
	empty := ""
	if _, err := api.Update(ctx, "svc-1", services.UpdateOpts{Name: &empty, Description: &empty, Enabled: &enabled, Extra: map[string]any{"owner": "infra"}}, services.WithUpdateField("x_extension", 2)); err != nil {
		t.Fatal(err)
	}
	var rows []*services.Service
	for value, err := range api.List(ctx, services.WithListOptions(services.ListOpts{ServiceType: "compute", Name: "nova"}), services.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "svc-1" && rows[1].ID == "svc-2" && reflect.DeepEqual(rows[1].Extra, map[string]any{"k": "v"})) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "svc-1"); err != nil {
		t.Fatal(err)
	}
	base := "/keystone/v3/services"
	want := []nativeServiceCall{
		{http.MethodPost, base, "", `{"service":{"enabled":false,"name":"nova","owner":"ops","type":"compute","x_extension":1}}`},
		{http.MethodPost, base, "", `{"service":{"type":""}}`},
		{http.MethodGet, base + "/svc-1", "", ""},
		{http.MethodPatch, base + "/svc-1", "", `{"service":{"description":"","enabled":false,"name":"","owner":"infra","x_extension":2}}`},
		{http.MethodGet, base, "extra=1&name=nova&type=compute", ""},
		{http.MethodGet, "/other/services", "page=2", ""},
		{http.MethodDelete, base + "/svc-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeServiceStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*services.API) error
	}{
		{"Create", []int{201}, func(api *services.API) error {
			_, err := api.Create(ctx, services.CreateOpts{Type: "compute"})
			return err
		}},
		{"Get", []int{200}, func(api *services.API) error { _, err := api.Get(ctx, "svc-1"); return err }},
		{"Update", []int{200}, func(api *services.API) error {
			_, err := api.Update(ctx, "svc-1", services.UpdateOpts{})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *services.API) error { return api.Delete(ctx, "svc-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeServiceCall
				api, _ := nativeServiceAPI(t, &calls, func(*http.Request) *http.Response { return nativeServiceWire(code, `{}`) })
				err := call.call(api)
				nativeServiceOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			`{}`:               false,
			`{"service":null}`: false,
			`{"service":[]}`:   true,
		} {
			var calls []nativeServiceCall
			api, _ := nativeServiceAPI(t, &calls, func(*http.Request) *http.Response { return nativeServiceWire(200, body) })
			got, err := api.Get(ctx, "svc-1")
			if wantErr {
				nativeServiceOperation(t, err, "Get")
			} else if err != nil || got != nil {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"services":[]}`}, {204, ""}} {
			var calls []nativeServiceCall
			api, _ := nativeServiceAPI(t, &calls, func(*http.Request) *http.Response { return nativeServiceWire(tc.code, tc.body) })
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
		var calls []nativeServiceCall
		api, _ := nativeServiceAPI(t, &calls, func(*http.Request) *http.Response { return nativeServiceWire(201, `{}`) })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create core extension": {"Create", func() error {
				_, err := api.Create(ctx, services.CreateOpts{Type: "compute"}, services.WithCreateField("type", "x"))
				return err
			}()},
			"create extra collision": {"Create", func() error {
				_, err := api.Create(ctx, services.CreateOpts{Type: "compute", Extra: map[string]any{"owner": "a"}}, services.WithCreateField("owner", "b"))
				return err
			}()},
			"update core extension": {"Update", func() error {
				_, err := api.Update(ctx, "svc-1", services.UpdateOpts{}, services.WithUpdateField("enabled", true))
				return err
			}()},
			"create nil option": {"Create", func() error {
				_, err := api.Create(ctx, services.CreateOpts{Type: "compute"}, nil)
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeServiceOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeServiceOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
