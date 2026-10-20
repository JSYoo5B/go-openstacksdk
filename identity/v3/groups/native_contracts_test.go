package groups_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeGroupTransport func(*http.Request) (*http.Response, error)

func (transport nativeGroupTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeGroupCall struct{ method, path, query, body string }

func nativeGroupAPI(t *testing.T, calls *[]nativeGroupCall, reply func(*http.Request) (int, string)) (*groups.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeGroupTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeGroupCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return groups.New(client), cloud
}

func nativeGroupOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "groups" {
		t.Fatal("generated groups context", err, wrapped)
	}
}

const nativeGroupRow = `{"id":"g-1","name":"ops","description":"desc","domain_id":"default","links":{"self":"x"},"email":"ops@example.com"}`

func TestNativeGroupRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeGroupCall
	var cloud *testcloud.Cloud
	api, cloud := nativeGroupAPI(t, &calls, func(req *http.Request) (int, string) {
		switch {
		case req.Method == http.MethodDelete:
			return 204, ""
		case req.Method == http.MethodPost:
			return 201, `{"group":` + nativeGroupRow + `}`
		case req.URL.Path == "/keystone/v3/groups":
			return 200, `{"groups":[` + nativeGroupRow + `],"links":{"next":"` + cloud.Server.URL + `/other/groups?page=2"}}`
		case req.URL.Path == "/other/groups":
			// A nested extra object replaces the remaining-key collection.
			return 200, `{"groups":[{"id":"g-2","extra":{"k":"v"},"email":"ignored"}],"links":{"next":null}}`
		}
		return 200, `{"group":` + nativeGroupRow + `}`
	})
	blank := ""
	created, err := api.Create(ctx, groups.CreateOpts{Name: "ops", Description: "desc", DomainID: "default", Extra: map[string]any{"email": "ops@example.com"}}, groups.WithCreateField("x_extension", 1))
	// Unknown response keys are collected into Extra.
	if err != nil || !(created.ID == "g-1" && created.DomainID == "default" && reflect.DeepEqual(created.Extra, map[string]any{"email": "ops@example.com"})) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "g-1")
	if err != nil || got.Name != "ops" {
		t.Fatal(got, err)
	}
	if _, err := api.Update(ctx, "g-1", groups.UpdateOpts{Description: &blank, Extra: map[string]any{"email": nil}}, groups.WithUpdateField("x_extension", 2)); err != nil {
		t.Fatal(err)
	}
	var rows []*groups.Group
	for value, err := range api.List(ctx, groups.WithListOptions(groups.ListOpts{DomainID: "default", Name: "ops", Filters: map[string]string{"name__contains": "o p"}}), groups.WithListQuery("x", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "g-1" && reflect.DeepEqual(rows[1].Extra, map[string]any{"k": "v"})) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "g-1"); err != nil {
		t.Fatal(err)
	}
	base := "/keystone/v3/groups"
	// BuildQueryString does not skip the q:"-" Filters map, so a stray "-" parameter
	// carrying the map's Python-style text precedes the real NAME__COMPARATOR filter.
	want := []nativeGroupCall{
		{http.MethodPost, base, "", `{"group":{"description":"desc","domain_id":"default","email":"ops@example.com","name":"ops","x_extension":1}}`},
		{http.MethodGet, base + "/g-1", "", ""},
		{http.MethodPatch, base + "/g-1", "", `{"group":{"description":"","email":null,"x_extension":2}}`},
		{http.MethodGet, base, "-=%7B%27name__contains%27%3A%27o+p%27%7D&domain_id=default&name=ops&name__contains=o+p&x=1", ""},
		{http.MethodGet, "/other/groups", "page=2", ""},
		{http.MethodDelete, base + "/g-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeGroupStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*groups.API) error
	}{
		{"Create", []int{201}, func(api *groups.API) error { _, err := api.Create(ctx, groups.CreateOpts{Name: "ops"}); return err }},
		{"Get", []int{200}, func(api *groups.API) error { _, err := api.Get(ctx, "g-1"); return err }},
		{"Update", []int{200}, func(api *groups.API) error {
			_, err := api.Update(ctx, "g-1", groups.UpdateOpts{Name: "n"})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *groups.API) error { return api.Delete(ctx, "g-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeGroupCall
				api, _ := nativeGroupAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
				err := call.call(api)
				nativeGroupOperation(t, err, call.name)
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
		}{{404, `{}`}, {200, `{"groups":[],"links":{"next":null}}`}, {204, ""}} {
			var calls []nativeGroupCall
			api, _ := nativeGroupAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, tc.body })
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
		var calls []nativeGroupCall
		api, _ := nativeGroupAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{}` })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create name": {"Create", func() error { _, err := api.Create(ctx, groups.CreateOpts{DomainID: "d"}); return err }()},
			"create core extension": {"Create", func() error {
				_, err := api.Create(ctx, groups.CreateOpts{Name: "ops"}, groups.WithCreateField("domain_id", "x"))
				return err
			}()},
			// Extra keys are already in the envelope, so an extension with the same key collides.
			"create extra collision": {"Create", func() error {
				_, err := api.Create(ctx, groups.CreateOpts{Name: "ops", Extra: map[string]any{"email": "a"}}, groups.WithCreateField("email", "b"))
				return err
			}()},
			"update nil option": {"Update", func() error {
				_, err := api.Update(ctx, "g-1", groups.UpdateOpts{}, nil)
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeGroupOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeGroupOperation(t, err, "List")
		}
		// An invalid filter name fails while building the native pager and is not wrapped.
		var filterErrs []error
		for _, err := range api.List(ctx, groups.WithListOptions(groups.ListOpts{Filters: map[string]string{"name": "x"}})) {
			filterErrs = append(filterErrs, err)
		}
		var invalid groups.InvalidListFilter
		var wrapped *resource.OperationError
		if len(filterErrs) != 1 || !errors.As(filterErrs[0], &invalid) || invalid.FilterName != "name" || errors.As(filterErrs[0], &wrapped) {
			t.Fatal(filterErrs)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
