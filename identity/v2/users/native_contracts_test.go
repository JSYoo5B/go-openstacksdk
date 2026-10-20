package users_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v2/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeV2UserTransport func(*http.Request) (*http.Response, error)

func (transport nativeV2UserTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeV2UserCall struct{ method, path, query, body string }

func nativeV2UserAPI(t *testing.T, calls *[]nativeV2UserCall, reply func(*http.Request) (int, string)) (*users.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeV2UserTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeV2UserCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v2.0/"
	return users.New(client), cloud
}

func nativeV2UserOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "users" {
		t.Fatal("generated users context", err, wrapped)
	}
}

// Keystone v2 answers with tenantId, but the native model decodes only tenant_id.
const nativeV2UserRow = `{"id":"u-1","name":"alice","username":"alice","enabled":true,"email":"a@example.invalid","tenantId":"ignored","tenant_id":"t-1"}`

func TestNativeV2UserRoutesBodiesAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeV2UserCall
	api, cloud := nativeV2UserAPI(t, &calls, func(req *http.Request) (int, string) {
		switch {
		case req.Method == http.MethodDelete:
			// Delete never decodes a body, even when one is returned.
			return 204, `{"user":` + nativeV2UserRow + `}`
		case req.Method == http.MethodPost:
			return 201, `{"user":` + nativeV2UserRow + `}`
		case req.URL.Path == "/keystone/v2.0/users":
			// users_links is never followed by the single-page pager.
			return 200, `{"users":[` + nativeV2UserRow + `,{"ID":"u-2","tenantId":"t-9"}],"users_links":[{"rel":"next","href":"/never"}]}`
		case req.URL.Path == "/keystone/v2.0/tenants/t-1/users/u-1/roles":
			return 200, `{"roles":[{"id":"r1","name":"member"}],"roles_links":[{"rel":"next","href":"/never"}]}`
		}
		return 200, `{"user":` + nativeV2UserRow + `}`
	})
	enabled, disabled := true, false
	created, err := api.Create(ctx, users.CreateOpts{Name: "alice", TenantID: "t-1", Enabled: &enabled, Email: "a@example.invalid"}, users.WithCreateField("x_extension", 1))
	want := users.User{ID: "u-1", Name: "alice", Username: "alice", Enabled: true, Email: "a@example.invalid", TenantID: "t-1"}
	if err != nil || !reflect.DeepEqual(*created, want) {
		t.Fatal(created, err)
	}
	if _, err := api.Create(ctx, users.CreateOpts{Username: "bob"}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "alice")
	if err != nil || !reflect.DeepEqual(*got, want) {
		t.Fatal(got, err)
	}
	if _, err := api.Update(ctx, "u-1", users.UpdateOpts{Email: "b@example.invalid", Enabled: &disabled}, users.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "u-1", users.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	var rows []users.User
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, *value)
	}
	var roles []users.Role
	for value, err := range api.ListRoles(ctx, "t-1", "u-1") {
		if err != nil {
			t.Fatal(err)
		}
		roles = append(roles, *value)
	}
	deleted, err := api.Delete(ctx, "u-1")
	if err != nil || deleted != nil {
		t.Fatal(deleted, err)
	}
	if !reflect.DeepEqual(rows, []users.User{want, {ID: "u-2"}}) || !reflect.DeepEqual(roles, []users.Role{{ID: "r1", Name: "member"}}) {
		t.Fatalf("%+v %+v", rows, roles)
	}
	if got := api.ResourceURL(ctx, "u-1"); got != cloud.Server.URL+"/keystone/v2.0/users/u-1" {
		t.Fatal(got)
	}
	base := "/keystone/v2.0/users"
	wantCalls := []nativeV2UserCall{
		{http.MethodPost, base, "", `{"user":{"email":"a@example.invalid","enabled":true,"name":"alice","tenantId":"t-1","x_extension":1}}`},
		{http.MethodPost, base, "", `{"user":{"username":"bob"}}`},
		{http.MethodGet, base + "/alice", "", ""},
		{http.MethodPut, base + "/u-1", "", `{"user":{"email":"b@example.invalid","enabled":false,"x_extension":1}}`},
		{http.MethodPut, base + "/u-1", "", `{"user":{}}`},
		{http.MethodGet, base, "", ""},
		{http.MethodGet, "/keystone/v2.0/tenants/t-1/users/u-1/roles", "", ""},
		{http.MethodDelete, base + "/u-1", "", ""},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeV2UserStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	valid := users.CreateOpts{Name: "alice"}
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*users.API) error
	}{
		{"Create", []int{200, 201}, func(api *users.API) error { _, err := api.Create(ctx, valid); return err }},
		{"Get", []int{200}, func(api *users.API) error { _, err := api.Get(ctx, "u-1"); return err }},
		{"Update", []int{200}, func(api *users.API) error { _, err := api.Update(ctx, "u-1", users.UpdateOpts{Name: "x"}); return err }},
		{"Delete", []int{202, 204}, func(api *users.API) error { _, err := api.Delete(ctx, "u-1"); return err }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeV2UserCall
				api, _ := nativeV2UserAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
				err := call.call(api)
				nativeV2UserOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{`{}`: false, `{"user":null}`: false, `{"user":[]}`: true} {
			var calls []nativeV2UserCall
			api, _ := nativeV2UserAPI(t, &calls, func(*http.Request) (int, string) { return 200, body })
			got, err := api.Get(ctx, "u-1")
			if wantErr {
				nativeV2UserOperation(t, err, "Get")
			} else if err != nil || got != nil {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list pagers status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"users":[],"roles":[]}`}, {200, `{}`}, {204, ""}} {
			var calls []nativeV2UserCall
			api, _ := nativeV2UserAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, tc.body })
			var errs []error
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			for _, err := range api.ListRoles(ctx, "t-1", "u-1") {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			switch {
			case tc.code == 404 && len(errs) == 2 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}) && errors.As(errs[1], &native):
			case tc.code == 200 && len(errs) == 0:
			case tc.code == 204 && len(errs) == 2 && errors.Is(errs[0], io.EOF) && errors.Is(errs[1], io.EOF):
			default:
				t.Fatal(tc.code, errs)
			}
			if len(calls) != 2 {
				t.Fatal(calls)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeV2UserCall
		api, _ := nativeV2UserAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{}` })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create name or username": {"Create", func() error {
				_, err := api.Create(ctx, users.CreateOpts{Email: "a@example.invalid"})
				var missing gophercloud.ErrMissingInput
				if !errors.As(err, &missing) {
					t.Fatal(err)
				}
				return err
			}()},
			"create extension": {"Create", func() error { _, err := api.Create(ctx, valid, users.WithCreateField("tenantId", "t")); return err }()},
			"create nil":       {"Create", func() error { _, err := api.Create(ctx, valid, nil); return err }()},
			"update extension": {"Update", func() error {
				_, err := api.Update(ctx, "u-1", users.UpdateOpts{}, users.WithUpdateField("email", "x"))
				return err
			}()},
			"update nil": {"Update", func() error { _, err := api.Update(ctx, "u-1", users.UpdateOpts{}, nil); return err }()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeV2UserOperation(t, check.err, check.operation)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
