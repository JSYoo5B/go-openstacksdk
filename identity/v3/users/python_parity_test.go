package users_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonUserCall struct{ method, path, query, body string }

func pythonUserAPI(t *testing.T, calls *[]pythonUserCall, reply func(*http.Request) (int, string)) *users.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonUserCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return users.New(client)
}

const (
	pythonUserBase = "/keystone/v3/users"
	pythonUserRow  = `{"id":"u-1","name":"alice","description":"desc","domain_id":"dom-1","default_project_id":"p-1","email":"a@example.invalid","enabled":true,"password_expires_at":"2026-11-06T15:32:17.000000","federated":[],"options":{},"links":{"self":"https://keystone.invalid/v3/users/u-1"}}`
)

// Python create_user/get_user/update_user use the "user" resource_key and
// PATCH; update carries the ID in the dirty body.
func TestPythonUserCreateGetUpdateRequests(t *testing.T) {
	ctx := context.Background()
	var calls []pythonUserCall
	api := pythonUserAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.Method {
		case http.MethodPost:
			return 201, `{"user":` + pythonUserRow + `}`
		case http.MethodPatch:
			return 200, `{"user":{"id":"u-1","name":"alice","enabled":false,"email":null}}`
		}
		return 200, `{"user":` + pythonUserRow + `}`
	})
	enabled, disabled := true, false
	want := users.User{
		ID: "u-1", Name: "alice", Description: "desc", DomainID: "dom-1", DefaultProjectID: "p-1", Enabled: true,
		PasswordExpiresAt: time.Date(2026, 11, 6, 15, 32, 17, 0, time.UTC),
		Options:           map[string]any{},
		Links:             map[string]any{"self": "https://keystone.invalid/v3/users/u-1"},
		Extra:             map[string]any{"email": "a@example.invalid", "federated": []any{}},
	}
	created, err := api.Create(ctx, users.CreateOpts{
		Name: "alice", Description: "desc", DomainID: "dom-1", DefaultProjectID: "p-1", Enabled: &enabled, Password: "fixture-secret",
		Extra: map[string]any{"email": "a@example.invalid"},
	})
	if err != nil || !reflect.DeepEqual(*created, want) {
		t.Fatalf("%#v %v", created, err)
	}
	got, err := api.Get(ctx, "u-1")
	if err != nil || !reflect.DeepEqual(*got, want) {
		t.Fatalf("%#v %v", got, err)
	}
	// Python update_user(u, is_enabled=False, email=None).
	updated, err := api.Update(ctx, "u-1", users.UpdateOpts{Enabled: &disabled, Extra: map[string]any{"email": nil}}, users.WithUpdateField("id", "u-1"))
	if err != nil || updated.Enabled || updated.Extra["email"] != nil {
		t.Fatalf("%#v %v", updated, err)
	}
	wantCalls := []pythonUserCall{
		{http.MethodPost, pythonUserBase, "", `{"user":{"default_project_id":"p-1","description":"desc","domain_id":"dom-1","email":"a@example.invalid","enabled":true,"name":"alice","password":"fixture-secret"}}`},
		{http.MethodGet, pythonUserBase + "/u-1", "", ""},
		{http.MethodPatch, pythonUserBase + "/u-1", "", `{"user":{"email":null,"enabled":false,"id":"u-1"}}`},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("%+v", calls)
	}
	calls = nil
	if _, err := api.Create(ctx, users.CreateOpts{Description: "desc"}); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonUserDeleteIgnoreMissing(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name     string
		status   int
		options  []resource.LookupOption
		missing  bool
		conflict bool
	}{
		{name: "deleted", status: 204},
		{name: "missing ignored by default", status: 404},
		{name: "ignore_missing=False", status: 404, options: []resource.LookupOption{resource.WithMissingError()}, missing: true},
		{name: "conflict is not missing", status: 409, conflict: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonUserCall
			api := pythonUserAPI(t, &calls, func(*http.Request) (int, string) { return tc.status, "" })
			err := api.Remove(ctx, resource.ID("u-1"), tc.options...)
			if tc.missing != errors.Is(err, resource.ErrNotFound) || tc.conflict != gophercloud.ResponseCodeIs(err, http.StatusConflict) || !tc.missing && !tc.conflict && err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []pythonUserCall{{http.MethodDelete, pythonUserBase + "/u-1", "", ""}}) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

func pythonUserIDs(t *testing.T, values func(func(*users.User, error) bool)) []string {
	t.Helper()
	var ids []string
	for value, err := range values {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	return ids
}

func TestPythonUserListRequests(t *testing.T) {
	ctx := context.Background()
	t.Run("default single request", func(t *testing.T) {
		var calls []pythonUserCall
		api := pythonUserAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"users":[` + pythonUserRow + `],"links":{"self":"x","next":null,"previous":null}}`
		})
		if ids := pythonUserIDs(t, api.List(ctx)); !reflect.DeepEqual(ids, []string{"u-1"}) || !reflect.DeepEqual(calls, []pythonUserCall{{http.MethodGet, pythonUserBase, "", ""}}) {
			t.Fatalf("%v %+v", ids, calls)
		}
	})
	t.Run("query mapping and Python bool text", func(t *testing.T) {
		var calls []pythonUserCall
		api := pythonUserAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"users":[]}` })
		// Python users(domain_id=..., name=..., unique_id=..., idp_id=...,
		// protocol_id=..., is_enabled=True).
		pythonUserIDs(t, api.List(ctx,
			users.WithListOptions(users.ListOpts{DomainID: "dom-1", Name: "alice", UniqueID: "uq", IdPID: "idp", ProtocolID: "saml2"}),
			users.WithListQuery("enabled", "True")))
		want := "domain_id=dom-1&enabled=True&idp_id=idp&name=alice&protocol_id=saml2&unique_id=uq"
		if len(calls) != 1 || calls[0].query != want {
			t.Fatalf("%+v", calls)
		}
	})
	t.Run("limit sends no marker request", func(t *testing.T) {
		var calls []pythonUserCall
		api := pythonUserAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"users":[` + pythonUserRow + `],"links":{"next":null}}`
		})
		// Python follows with GET ?limit=1&marker=u-1 until an empty page.
		pythonUserIDs(t, api.List(ctx, users.WithListQuery("limit", "1")))
		if !reflect.DeepEqual(calls, []pythonUserCall{{http.MethodGet, pythonUserBase, "limit=1", ""}}) {
			t.Fatalf("%+v", calls)
		}
	})
	t.Run("no local Body filter", func(t *testing.T) {
		var calls []pythonUserCall
		api := pythonUserAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"users":[]}` })
		if _, err := api.All(ctx, resource.WithBodyFilter("email", "a@example.invalid")); !errors.Is(err, resource.ErrUnsupported) || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	})
}

// Python group_users(group, **attrs) lists GET groups/{id}/users with the User
// query mapping.
func TestPythonUserGroupUsersRequests(t *testing.T) {
	ctx := context.Background()
	var calls []pythonUserCall
	api := pythonUserAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"users":[` + pythonUserRow + `],"links":{"next":null}}`
	})
	ids := pythonUserIDs(t, api.ListInGroup(ctx, "g-1"))
	pythonUserIDs(t, api.ListInGroup(ctx, "g-1", users.WithListInGroupOptions(users.ListOpts{PasswordExpiresAt: "lt:2026-12-01T00:00:00Z"})))
	// Python group_users(g, limit=1) would add GET ?limit=1&marker=u-1.
	pythonUserIDs(t, api.ListInGroup(ctx, "g-1", users.WithListInGroupQuery("limit", "1")))
	want := []pythonUserCall{
		{http.MethodGet, "/keystone/v3/groups/g-1/users", "", ""},
		{http.MethodGet, "/keystone/v3/groups/g-1/users", "password_expires_at=lt%3A2026-12-01T00%3A00%3A00Z", ""},
		{http.MethodGet, "/keystone/v3/groups/g-1/users", "limit=1", ""},
	}
	if !reflect.DeepEqual(ids, []string{"u-1"}) || !reflect.DeepEqual(calls, want) {
		t.Fatalf("%v %+v", ids, calls)
	}
}

// Python update_password posts {"user": {"password", "original_password"}} and
// raises only for status 400 and above.
func TestPythonUserUpdatePassword(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		status int
		ok     bool
	}{{204, true}, {401, false}, {200, false}} {
		var calls []pythonUserCall
		api := pythonUserAPI(t, &calls, func(*http.Request) (int, string) { return tc.status, "" })
		err := api.ChangePassword(ctx, "u-1", users.ChangePasswordOpts{OriginalPassword: "fixture-old", Password: "fixture-new"})
		if tc.ok != (err == nil) || !tc.ok && !gophercloud.ResponseCodeIs(err, tc.status) {
			t.Fatal(tc.status, err)
		}
		want := []pythonUserCall{{http.MethodPost, pythonUserBase + "/u-1/password", "", `{"user":{"original_password":"fixture-old","password":"fixture-new"}}`}}
		if !reflect.DeepEqual(calls, want) {
			t.Fatalf("%+v", calls)
		}
	}
}

// Python add_user_to_group/remove_user_from_group raise for status 400 and
// above; check_user_in_group maps 404 to False and only 204 to True.
func TestPythonUserGroupMembership(t *testing.T) {
	ctx := context.Background()
	member := pythonUserCall{http.MethodPut, "/keystone/v3/groups/g-1/users/u-1", "", ""}
	t.Run("add and remove", func(t *testing.T) {
		var calls []pythonUserCall
		api := pythonUserAPI(t, &calls, func(*http.Request) (int, string) { return 204, "" })
		if err := api.AddToGroup(ctx, "g-1", "u-1"); err != nil {
			t.Fatal(err)
		}
		if err := api.RemoveFromGroup(ctx, "g-1", "u-1"); err != nil {
			t.Fatal(err)
		}
		removed := member
		removed.method = http.MethodDelete
		if !reflect.DeepEqual(calls, []pythonUserCall{member, removed}) {
			t.Fatalf("%+v", calls)
		}
	})
	t.Run("add error", func(t *testing.T) {
		var calls []pythonUserCall
		api := pythonUserAPI(t, &calls, func(*http.Request) (int, string) { return 404, "" })
		if err := api.AddToGroup(ctx, "g-1", "u-1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			t.Fatal(err)
		}
		if err := api.RemoveFromGroup(ctx, "g-1", "u-1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			t.Fatal(err)
		}
	})
	for _, tc := range []struct {
		name   string
		status int
		member bool
		err    bool
	}{
		{"member", 204, true, false},
		{"not member", 404, false, false},
		{"forbidden raises", 403, false, true},
		{"200 is Python False and a Go error", 200, false, true},
	} {
		t.Run("check "+tc.name, func(t *testing.T) {
			var calls []pythonUserCall
			api := pythonUserAPI(t, &calls, func(*http.Request) (int, string) { return tc.status, "" })
			ok, err := api.IsMemberOfGroup(ctx, "g-1", "u-1")
			if ok != tc.member || tc.err != (err != nil) {
				t.Fatal(ok, err)
			}
			head := member
			head.method = http.MethodHead
			if !reflect.DeepEqual(calls, []pythonUserCall{head}) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}
