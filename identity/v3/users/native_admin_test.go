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
	"time"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeUserTransport func(*http.Request) (*http.Response, error)

func (transport nativeUserTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeUserCall struct{ method, path, query, body string }

func nativeUserAPI(t *testing.T, calls *[]nativeUserCall, reply func(*http.Request) (int, string)) (*users.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeUserTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeUserCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return users.New(client), cloud
}

func nativeUserOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "users" {
		t.Fatal("generated users context", err, wrapped)
	}
}

// Keystone sends password_expires_at without a zone; enabled may arrive as a string.
const nativeUserRow = `{"id":"u-1","name":"alice","domain_id":"default","default_project_id":"p-1","description":"desc","enabled":"true","password_expires_at":"2026-10-11T01:02:03.000000","options":{"ignore_password_expiry":true},"links":{"self":"x"},"email":"a@example.com"}`

func TestNativeUserAdminRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeUserCall
	var cloud *testcloud.Cloud
	api, cloud := nativeUserAPI(t, &calls, func(req *http.Request) (int, string) {
		switch {
		case req.Method == http.MethodDelete, req.Method == http.MethodHead, req.Method == http.MethodPut:
			return 204, ""
		case req.Method == http.MethodPost:
			return 201, `{"user":` + nativeUserRow + `}`
		case req.URL.Path == "/keystone/v3/users":
			return 200, `{"users":[` + nativeUserRow + `],"links":{"next":"` + cloud.Server.URL + `/other/users?page=2"}}`
		case req.URL.Path == "/other/users":
			return 200, `{"users":[{"id":"u-2","enabled":false,"password_expires_at":null}],"links":{"next":null}}`
		case req.URL.Path == "/keystone/v3/groups/g-1/users":
			return 200, `{"users":[{"id":"u-3","enabled":null}],"links":{"next":null}}`
		}
		return 200, `{"user":` + nativeUserRow + `}`
	})
	disabled, blank := false, ""
	created, err := api.Create(ctx, users.CreateOpts{Name: "alice", DefaultProjectID: "p-1", Description: "desc", DomainID: "default", Enabled: &disabled, Password: "secret", Options: map[users.Option]any{users.IgnorePasswordExpiry: true}, Extra: map[string]any{"email": "a@example.com"}}, users.WithCreateField("x_extension", 1))
	expires := time.Date(2026, 10, 11, 1, 2, 3, 0, time.UTC)
	// password_expires_at is removed from Extra, while links has its own field.
	if err != nil || !(created.ID == "u-1" && created.Enabled && created.PasswordExpiresAt.Equal(expires) && created.Options["ignore_password_expiry"] == true && reflect.DeepEqual(created.Extra, map[string]any{"email": "a@example.com"})) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "u-1")
	if err != nil || got.DefaultProjectID != "p-1" {
		t.Fatal(got, err)
	}
	if _, err := api.Update(ctx, "u-1", users.UpdateOpts{Description: &blank, Enabled: &disabled, Password: "new", Extra: map[string]any{"email": nil}}, users.WithUpdateField("x_extension", 2)); err != nil {
		t.Fatal(err)
	}
	var rows []string
	for value, err := range api.List(ctx, users.WithListOptions(users.ListOpts{DomainID: "default", Enabled: &disabled, IdPID: "idp", Name: "alice", PasswordExpiresAt: "lt:2026-10-11T00:00:00Z", ProtocolID: "saml2", UniqueID: "uid", Filters: map[string]string{"name__icontains": "al"}}), users.WithListQuery("x", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, fmt.Sprintf("%s/%t/%t", value.ID, value.Enabled, value.PasswordExpiresAt.IsZero()))
	}
	for value, err := range api.ListInGroup(ctx, "g-1", users.WithListInGroupOptions(users.ListOpts{Name: "bob"}), users.WithListInGroupQuery("x", "2")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, fmt.Sprintf("%s/%t", value.ID, value.Enabled))
	}
	if !reflect.DeepEqual(rows, []string{"u-1/true/false", "u-2/false/true", "u-3/false"}) {
		t.Fatal(rows)
	}
	if err := api.AddToGroup(ctx, "g-1", "u-1"); err != nil {
		t.Fatal(err)
	}
	member, err := api.IsMemberOfGroup(ctx, "g-1", "u-1")
	if err != nil || !member {
		t.Fatal(member, err)
	}
	if err := api.RemoveFromGroup(ctx, "g-1", "u-1"); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "u-1"); err != nil {
		t.Fatal(err)
	}
	base := "/keystone/v3/users"
	want := []nativeUserCall{
		{http.MethodPost, base, "", `{"user":{"default_project_id":"p-1","description":"desc","domain_id":"default","email":"a@example.com","enabled":false,"name":"alice","options":{"ignore_password_expiry":true},"password":"secret","x_extension":1}}`},
		{http.MethodGet, base + "/u-1", "", ""},
		{http.MethodPatch, base + "/u-1", "", `{"user":{"description":"","email":null,"enabled":false,"password":"new","x_extension":2}}`},
		// The q:"-" Filters map also leaks as a stray "-" parameter before the real filter.
		{http.MethodGet, base, "-=%7B%27name__icontains%27%3A%27al%27%7D&domain_id=default&enabled=false&idp_id=idp&name=alice&name__icontains=al&password_expires_at=lt%3A2026-10-11T00%3A00%3A00Z&protocol_id=saml2&unique_id=uid&x=1", ""},
		{http.MethodGet, "/other/users", "page=2", ""},
		{http.MethodGet, "/keystone/v3/groups/g-1/users", "name=bob&x=2", ""},
		{http.MethodPut, "/keystone/v3/groups/g-1/users/u-1", "", ""},
		{http.MethodHead, "/keystone/v3/groups/g-1/users/u-1", "", ""},
		{http.MethodDelete, "/keystone/v3/groups/g-1/users/u-1", "", ""},
		{http.MethodDelete, base + "/u-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeUserAdminStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*users.API) error
	}{
		{"Create", []int{201}, func(api *users.API) error { _, err := api.Create(ctx, users.CreateOpts{Name: "alice"}); return err }},
		{"Get", []int{200}, func(api *users.API) error { _, err := api.Get(ctx, "u-1"); return err }},
		{"Update", []int{200}, func(api *users.API) error { _, err := api.Update(ctx, "u-1", users.UpdateOpts{Name: "n"}); return err }},
		{"Delete", []int{202, 204}, func(api *users.API) error { return api.Delete(ctx, "u-1") }},
		{"AddToGroup", []int{204}, func(api *users.API) error { return api.AddToGroup(ctx, "g-1", "u-1") }},
		// 404 is an accepted "not a member" answer.
		{"IsMemberOfGroup", []int{204, 404}, func(api *users.API) error { _, err := api.IsMemberOfGroup(ctx, "g-1", "u-1"); return err }},
		{"RemoveFromGroup", []int{204}, func(api *users.API) error { return api.RemoveFromGroup(ctx, "g-1", "u-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeUserCall
				api, _ := nativeUserAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
				err := call.call(api)
				nativeUserOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("membership 404 is false", func(t *testing.T) {
		var calls []nativeUserCall
		api, _ := nativeUserAPI(t, &calls, func(*http.Request) (int, string) { return 404, "" })
		member, err := api.IsMemberOfGroup(ctx, "g-1", "u-1")
		if err != nil || member || len(calls) != 1 {
			t.Fatal(member, err, calls)
		}
	})
	t.Run("user decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			`{}`: false,
			`{"user":{"id":"u","enabled":null,"password_expires_at":null}}`: false,
			// The zone-less layout rejects a trailing Z.
			`{"user":{"id":"u","password_expires_at":"2026-10-11T01:02:03Z"}}`: true,
			`{"user":{"id":"u","enabled":"maybe"}}`:                            true,
			`{"user":{"id":"u","enabled":1}}`:                                  true,
		} {
			var calls []nativeUserCall
			api, _ := nativeUserAPI(t, &calls, func(*http.Request) (int, string) { return 200, body })
			got, err := api.Get(ctx, "u-1")
			if wantErr {
				nativeUserOperation(t, err, "Get")
			} else if err != nil || (got != nil && (got.Enabled || !got.PasswordExpiresAt.IsZero())) {
				t.Fatal(body, got, err)
			}
		}
	})
	for name, list := range map[string]func(*users.API) []error{
		"List": func(api *users.API) (errs []error) {
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			return errs
		},
		"ListInGroup": func(api *users.API) (errs []error) {
			for _, err := range api.ListInGroup(ctx, "g-1") {
				errs = append(errs, err)
			}
			return errs
		},
	} {
		t.Run(name+" pager status, empty page and bodyless 204", func(t *testing.T) {
			for _, tc := range []struct {
				code int
				body string
			}{{404, `{}`}, {200, `{"users":[],"links":{"next":null}}`}, {204, ""}} {
				var calls []nativeUserCall
				api, _ := nativeUserAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, tc.body })
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
		var calls []nativeUserCall
		api, _ := nativeUserAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{}` })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create name": {"Create", func() error { _, err := api.Create(ctx, users.CreateOpts{Password: "x"}); return err }()},
			"create core extension": {"Create", func() error {
				_, err := api.Create(ctx, users.CreateOpts{Name: "alice"}, users.WithCreateField("password", "x"))
				return err
			}()},
			"update extra collision": {"Update", func() error {
				_, err := api.Update(ctx, "u-1", users.UpdateOpts{Extra: map[string]any{"email": "a"}}, users.WithUpdateField("email", "b"))
				return err
			}()},
			"update nil option": {"Update", func() error {
				_, err := api.Update(ctx, "u-1", users.UpdateOpts{}, nil)
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeUserOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeUserOperation(t, err, "List")
		}
		for _, err := range api.ListInGroup(ctx, "g-1", nil) {
			nativeUserOperation(t, err, "ListInGroup")
		}
		var filterErrs []error
		for _, err := range api.ListInGroup(ctx, "g-1", users.WithListInGroupOptions(users.ListOpts{Filters: map[string]string{"name__": "x"}})) {
			filterErrs = append(filterErrs, err)
		}
		var invalid users.InvalidListFilter
		if len(filterErrs) != 1 || !errors.As(filterErrs[0], &invalid) || invalid.FilterName != "name__" {
			t.Fatal(filterErrs)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
