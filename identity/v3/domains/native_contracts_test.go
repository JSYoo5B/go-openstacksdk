package domains_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeDomainTransport func(*http.Request) (*http.Response, error)

func (transport nativeDomainTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeDomainCall struct{ method, path, query, body string }

func nativeDomainAPI(t *testing.T, calls *[]nativeDomainCall, reply func(*http.Request) (int, string)) (*domains.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeDomainTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeDomainCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return domains.New(client), cloud
}

func nativeDomainOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "domains" {
		t.Fatal("generated domains context", err, wrapped)
	}
}

const nativeDomainRow = `{"id":"d-1","name":"dom","description":"desc","enabled":true,"links":{"self":"x"},"options":{}}`

func TestNativeDomainRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeDomainCall
	var cloud *testcloud.Cloud
	api, cloud := nativeDomainAPI(t, &calls, func(req *http.Request) (int, string) {
		switch {
		case req.Method == http.MethodDelete:
			return 204, ""
		case req.Method == http.MethodPost:
			return 201, `{"domain":` + nativeDomainRow + `}`
		case req.URL.Path == "/keystone/v3/domains":
			return 200, `{"domains":[` + nativeDomainRow + `],"links":{"next":"` + cloud.Server.URL + `/other/domains?page=2","previous":null}}`
		case req.URL.Path == "/other/domains":
			return 200, `{"domains":[{"id":"d-2","enabled":false}],"links":{"next":null}}`
		case req.URL.Path == "/keystone/v3/auth/domains":
			return 200, `{"domains":[{"id":"d-3","name":"mine"}],"links":{"self":"x"}}`
		}
		return 200, `{"domain":` + nativeDomainRow + `}`
	})
	disabled, enabled, blank := false, true, ""
	created, err := api.Create(ctx, domains.CreateOpts{Name: "dom", Description: "desc", Enabled: &disabled}, domains.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "d-1" && created.Name == "dom" && created.Enabled && created.Links["self"] == "x") {
		t.Fatal(created, err)
	}
	// Empty optional create fields are omitted.
	if _, err := api.Create(ctx, domains.CreateOpts{Name: "dom"}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "d-1")
	if err != nil || got.Description != "desc" {
		t.Fatal(got, err)
	}
	// A pointer to an empty description clears it; an empty update still sends the envelope.
	if _, err := api.Update(ctx, "d-1", domains.UpdateOpts{Name: "new", Description: &blank, Enabled: &enabled}, domains.WithUpdateField("x_extension", "v")); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "d-1", domains.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for value, err := range api.List(ctx, domains.WithListOptions(domains.ListOpts{Enabled: &disabled, Name: "dom", Limit: 2}), domains.WithListQuery("x", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, fmt.Sprint(value.ID, "=", value.Enabled))
	}
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	for value, err := range api.ListAvailable(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID+"="+value.Name)
	}
	if err := api.Delete(ctx, "d-1"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []string{"d-1=true", "d-2=false", "d-1", "d-2", "d-3=mine"}) {
		t.Fatal(ids)
	}
	base := "/keystone/v3/domains"
	want := []nativeDomainCall{
		{http.MethodPost, base, "", `{"domain":{"description":"desc","enabled":false,"name":"dom","x_extension":1}}`},
		{http.MethodPost, base, "", `{"domain":{"name":"dom"}}`},
		{http.MethodGet, base + "/d-1", "", ""},
		{http.MethodPatch, base + "/d-1", "", `{"domain":{"description":"","enabled":true,"name":"new","x_extension":"v"}}`},
		{http.MethodPatch, base + "/d-1", "", `{"domain":{}}`},
		{http.MethodGet, base, "enabled=false&limit=2&name=dom&x=1", ""},
		{http.MethodGet, "/other/domains", "page=2", ""},
		{http.MethodGet, base, "", ""},
		{http.MethodGet, "/other/domains", "page=2", ""},
		{http.MethodGet, "/keystone/v3/auth/domains", "", ""},
		{http.MethodDelete, base + "/d-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeDomainStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*domains.API) error
	}{
		{"Create", []int{201}, func(api *domains.API) error { _, err := api.Create(ctx, domains.CreateOpts{Name: "dom"}); return err }},
		{"Get", []int{200}, func(api *domains.API) error { _, err := api.Get(ctx, "d-1"); return err }},
		{"Update", []int{200}, func(api *domains.API) error {
			_, err := api.Update(ctx, "d-1", domains.UpdateOpts{Name: "n"})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *domains.API) error { return api.Delete(ctx, "d-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeDomainCall
				api, _ := nativeDomainAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
				err := call.call(api)
				nativeDomainOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			// A missing or null envelope is a nil result without an error.
			`{}`:              false,
			`{"domain":null}`: false,
			`{"domain":[]}`:   true,
		} {
			var calls []nativeDomainCall
			api, _ := nativeDomainAPI(t, &calls, func(*http.Request) (int, string) { return 200, body })
			got, err := api.Get(ctx, "d-1")
			if wantErr {
				nativeDomainOperation(t, err, "Get")
			} else if err != nil || got != nil {
				t.Fatal(body, got, err)
			}
		}
	})
	for name, list := range map[string]func(*domains.API) []error{
		"List": func(api *domains.API) (errs []error) {
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			return errs
		},
		"ListAvailable": func(api *domains.API) (errs []error) {
			for _, err := range api.ListAvailable(ctx) {
				errs = append(errs, err)
			}
			return errs
		},
	} {
		t.Run(name+" pager status, empty page and bodyless 204", func(t *testing.T) {
			for _, tc := range []struct {
				code int
				body string
			}{{404, `{}`}, {200, `{"domains":[],"links":{"next":null}}`}, {204, ""}} {
				var calls []nativeDomainCall
				api, _ := nativeDomainAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, tc.body })
				errs := list(api)
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
	}
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeDomainCall
		api, _ := nativeDomainAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{}` })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create name": {"Create", func() error { _, err := api.Create(ctx, domains.CreateOpts{Description: "d"}); return err }()},
			"create core extension": {"Create", func() error {
				_, err := api.Create(ctx, domains.CreateOpts{Name: "dom"}, domains.WithCreateField("enabled", true))
				return err
			}()},
			"update core extension": {"Update", func() error {
				_, err := api.Update(ctx, "d-1", domains.UpdateOpts{}, domains.WithUpdateField("description", "x"))
				return err
			}()},
			"update nil option": {"Update", func() error {
				_, err := api.Update(ctx, "d-1", domains.UpdateOpts{}, nil)
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeDomainOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeDomainOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
