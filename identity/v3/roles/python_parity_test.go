package roles_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonRoleTransport func(*http.Request) (*http.Response, error)

func (transport pythonRoleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonRoleCall struct{ method, path, query, body string }

func pythonRoleAPI(t *testing.T, calls *[]pythonRoleCall, reply func(*http.Request) (int, string)) *roles.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonRoleTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonRoleCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return roles.New(client)
}

const (
	pythonRoleBase = "/keystone/v3/roles"
	pythonRoleRow  = `{"id":"r-1","name":"reader","description":"desc","domain_id":"dom-1","options":{"immutable":true},"links":{"self":"https://keystone.invalid/v3/roles/r-1"}}`
)

// Python create_role/get_role/update_role use the "role" resource_key and
// PATCH; update carries the ID in the dirty body.
func TestPythonRoleCreateGetUpdateRequests(t *testing.T) {
	ctx := context.Background()
	var calls []pythonRoleCall
	api := pythonRoleAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.Method {
		case http.MethodPost:
			return 201, `{"role":` + pythonRoleRow + `}`
		case http.MethodPatch:
			return 200, `{"role":{"id":"r-1","name":"viewer","description":null}}`
		}
		return 200, `{"role":` + pythonRoleRow + `}`
	})
	check := func(value *roles.Role, err error) {
		t.Helper()
		if err != nil || value.ID != "r-1" || value.Name != "reader" || value.Description != "desc" || value.DomainID != "dom-1" ||
			!reflect.DeepEqual(value.Options, map[roles.Option]any{roles.Immutable: true}) || value.Links["self"] != "https://keystone.invalid/v3/roles/r-1" {
			t.Fatalf("%#v %v", value, err)
		}
	}
	check(api.Create(ctx, roles.CreateOpts{Name: "reader", Description: "desc", DomainID: "dom-1", Options: map[roles.Option]any{roles.Immutable: true}}))
	check(api.Get(ctx, "r-1"))
	// Python update_role(r, name="viewer", description=None).
	updated, err := api.Update(ctx, "r-1", roles.UpdateOpts{Name: "viewer", Extra: map[string]any{"description": nil}}, roles.WithUpdateField("id", "r-1"))
	if err != nil || updated.Name != "viewer" {
		t.Fatalf("%#v %v", updated, err)
	}
	wantCalls := []pythonRoleCall{
		{http.MethodPost, pythonRoleBase, "", `{"role":{"description":"desc","domain_id":"dom-1","name":"reader","options":{"immutable":true}}}`},
		{http.MethodGet, pythonRoleBase + "/r-1", "", ""},
		{http.MethodPatch, pythonRoleBase + "/r-1", "", `{"role":{"description":null,"id":"r-1","name":"viewer"}}`},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("%+v", calls)
	}
	calls = nil
	if _, err := api.Create(ctx, roles.CreateOpts{Description: "desc"}); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonRoleDeleteIgnoreMissing(t *testing.T) {
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
			var calls []pythonRoleCall
			api := pythonRoleAPI(t, &calls, func(*http.Request) (int, string) { return tc.status, "" })
			err := api.Remove(ctx, resource.ID("r-1"), tc.options...)
			if tc.missing != errors.Is(err, resource.ErrNotFound) || tc.conflict != gophercloud.ResponseCodeIs(err, http.StatusConflict) || !tc.missing && !tc.conflict && err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []pythonRoleCall{{http.MethodDelete, pythonRoleBase + "/r-1", "", ""}}) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

func pythonRoleIDs(t *testing.T, values func(func(*roles.Role, error) bool)) []string {
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

func TestPythonRoleListRequests(t *testing.T) {
	ctx := context.Background()
	t.Run("default single request", func(t *testing.T) {
		var calls []pythonRoleCall
		api := pythonRoleAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"roles":[` + pythonRoleRow + `],"links":{"self":"x","next":null,"previous":null}}`
		})
		if ids := pythonRoleIDs(t, api.List(ctx)); !reflect.DeepEqual(ids, []string{"r-1"}) || !reflect.DeepEqual(calls, []pythonRoleCall{{http.MethodGet, pythonRoleBase, "", ""}}) {
			t.Fatalf("%v %+v", ids, calls)
		}
	})
	t.Run("query mapping", func(t *testing.T) {
		var calls []pythonRoleCall
		api := pythonRoleAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"roles":[]}` })
		pythonRoleIDs(t, api.List(ctx, roles.WithListOptions(roles.ListOpts{DomainID: "dom-1", Name: "reader"})))
		if !reflect.DeepEqual(calls, []pythonRoleCall{{http.MethodGet, pythonRoleBase, "domain_id=dom-1&name=reader", ""}}) {
			t.Fatalf("%+v", calls)
		}
	})
	t.Run("limit sends no marker request", func(t *testing.T) {
		var calls []pythonRoleCall
		api := pythonRoleAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"roles":[` + pythonRoleRow + `],"links":{"next":null}}`
		})
		// Python follows with GET ?limit=1&marker=r-1 until an empty page.
		pythonRoleIDs(t, api.List(ctx, roles.WithListQuery("limit", "1")))
		if !reflect.DeepEqual(calls, []pythonRoleCall{{http.MethodGet, pythonRoleBase, "limit=1", ""}}) {
			t.Fatalf("%+v", calls)
		}
	})
	t.Run("no local Body filter", func(t *testing.T) {
		var calls []pythonRoleCall
		api := pythonRoleAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"roles":[]}` })
		if _, err := api.All(ctx, resource.WithBodyFilter("description", "desc")); !errors.Is(err, resource.ErrUnsupported) || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	})
}

// Python role_assignments(**query) maps role_id/user_id/group_id/scope_* to
// dotted keys and sends bool values as True/False text.
func TestPythonRoleAssignmentsRequests(t *testing.T) {
	ctx := context.Background()
	var calls []pythonRoleCall
	api := pythonRoleAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"role_assignments":[{"role":{"id":"r-1","name":"reader"},"user":{"id":"u-1","name":"alice","domain":{"id":"dom-1"}},` +
			`"scope":{"project":{"id":"p-1","name":"proj","domain":{"id":"dom-1"}},"OS-INHERIT:inherited_to":"projects"},` +
			`"links":{"assignment":"https://keystone.invalid/v3/projects/p-1/users/u-1/roles/r-1"}}],"links":{"self":"x","next":null}}`
	})
	collect := func(options ...roles.ListAssignmentsOption) []*roles.RoleAssignment {
		var values []*roles.RoleAssignment
		for value, err := range api.ListAssignments(ctx, options...) {
			if err != nil {
				t.Fatal(err)
			}
			values = append(values, value)
		}
		return values
	}
	yes := true
	values := collect(roles.WithListAssignmentsOptions(roles.ListAssignmentsOpts{UserID: "u-1", RoleID: "r-1", ScopeProjectID: "p-1", IncludeNames: &yes}))
	// Python role_assignments(user_id="u-1", role_id="r-1",
	// scope_project_id="p-1", include_names=True, inherited_to="projects").
	collect(roles.WithListAssignmentsOptions(roles.ListAssignmentsOpts{UserID: "u-1", RoleID: "r-1", ScopeProjectID: "p-1"}),
		roles.WithListAssignmentsQuery("include_names", "True"), roles.WithListAssignmentsQuery("scope.OS-INHERIT:inherited_to", "projects"))
	// Python role_assignments(limit=1) would add GET ?limit=1&marker=<id>.
	collect(roles.WithListAssignmentsQuery("limit", "1"))
	wantCalls := []pythonRoleCall{
		{http.MethodGet, "/keystone/v3/role_assignments", "include_names=true&role.id=r-1&scope.project.id=p-1&user.id=u-1", ""},
		{http.MethodGet, "/keystone/v3/role_assignments", "include_names=True&role.id=r-1&scope.OS-INHERIT%3Ainherited_to=projects&scope.project.id=p-1&user.id=u-1", ""},
		{http.MethodGet, "/keystone/v3/role_assignments", "limit=1", ""},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("%+v", calls)
	}
	// The native model keeps no links and no OS-INHERIT:inherited_to scope key.
	want := roles.RoleAssignment{
		Role:  roles.AssignedRole{ID: "r-1", Name: "reader"},
		User:  roles.User{ID: "u-1", Name: "alice", Domain: roles.Domain{ID: "dom-1"}},
		Scope: roles.Scope{Project: roles.Project{ID: "p-1", Name: "proj", Domain: roles.Domain{ID: "dom-1"}}},
	}
	if len(values) != 1 || !reflect.DeepEqual(*values[0], want) {
		t.Fatalf("%#v", values)
	}
}

// Python role_assignments_filter lists the roles of one actor on one target
// with a single GET and no query.
func TestPythonRoleAssignmentsFilterRoutes(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		opts roles.ListAssignmentsOnResourceOpts
		path string
	}{
		{"domain group", roles.ListAssignmentsOnResourceOpts{DomainID: "dom-1", GroupID: "g-1"}, "/keystone/v3/domains/dom-1/groups/g-1/roles"},
		{"domain user", roles.ListAssignmentsOnResourceOpts{DomainID: "dom-1", UserID: "u-1"}, "/keystone/v3/domains/dom-1/users/u-1/roles"},
		{"project group", roles.ListAssignmentsOnResourceOpts{ProjectID: "p-1", GroupID: "g-1"}, "/keystone/v3/projects/p-1/groups/g-1/roles"},
		{"project user", roles.ListAssignmentsOnResourceOpts{ProjectID: "p-1", UserID: "u-1"}, "/keystone/v3/projects/p-1/users/u-1/roles"},
		{"system group", roles.ListAssignmentsOnResourceOpts{System: true, GroupID: "g-1"}, "/keystone/v3/system/groups/g-1/roles"},
		{"system user", roles.ListAssignmentsOnResourceOpts{System: true, UserID: "u-1"}, "/keystone/v3/system/users/u-1/roles"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonRoleCall
			api := pythonRoleAPI(t, &calls, func(*http.Request) (int, string) {
				return 200, `{"roles":[{"id":"r-1","name":"reader","links":{"self":"https://keystone.invalid/v3/roles/r-1"}}],"links":{"self":"x","next":null}}`
			})
			var names []string
			for value, err := range api.ListAssignmentsOnResource(ctx, roles.WithListAssignmentsOnResourceOptions(tc.opts)) {
				if err != nil {
					t.Fatal(err)
				}
				names = append(names, value.ID+":"+value.Name)
			}
			if !reflect.DeepEqual(names, []string{"r-1:reader"}) || !reflect.DeepEqual(calls, []pythonRoleCall{{http.MethodGet, tc.path, "", ""}}) {
				t.Fatalf("%v %+v", names, calls)
			}
		})
	}
	t.Run("two targets rejected before HTTP", func(t *testing.T) {
		// Python picks the domain when domain and project are both given.
		var calls []pythonRoleCall
		api := pythonRoleAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"roles":[]}` })
		for _, err := range api.ListAssignmentsOnResource(ctx, roles.WithListAssignmentsOnResourceOptions(roles.ListAssignmentsOnResourceOpts{DomainID: "dom-1", ProjectID: "p-1", UserID: "u-1"})) {
			if err == nil {
				t.Fatal("expected error")
			}
		}
		if len(calls) != 0 {
			t.Fatalf("%+v", calls)
		}
	})
}

type pythonRoleTarget struct {
	name                                 string
	userID, groupID, projectID, domainID string
	system                               bool
	path                                 string
}

var pythonRoleTargets = []pythonRoleTarget{
	{name: "domain user", domainID: "dom-1", userID: "u-1", path: "/keystone/v3/domains/dom-1/users/u-1/roles/r-1"},
	{name: "domain group", domainID: "dom-1", groupID: "g-1", path: "/keystone/v3/domains/dom-1/groups/g-1/roles/r-1"},
	{name: "project user", projectID: "p-1", userID: "u-1", path: "/keystone/v3/projects/p-1/users/u-1/roles/r-1"},
	{name: "project group", projectID: "p-1", groupID: "g-1", path: "/keystone/v3/projects/p-1/groups/g-1/roles/r-1"},
	{name: "system user", system: true, userID: "u-1", path: "/keystone/v3/system/users/u-1/roles/r-1"},
	{name: "system group", system: true, groupID: "g-1", path: "/keystone/v3/system/groups/g-1/roles/r-1"},
}

// Python assign_*/unassign_* (inherited=False) send PUT/DELETE and discard
// the status; validate_* sends HEAD and returns True only for 204.
func TestPythonRoleAssignUnassignValidate(t *testing.T) {
	ctx := context.Background()
	for _, target := range pythonRoleTargets {
		ops := []struct {
			method string
			call   func(*roles.API) error
		}{
			{http.MethodPut, func(api *roles.API) error {
				return api.Assign(ctx, "r-1", roles.AssignOpts{UserID: target.userID, GroupID: target.groupID, ProjectID: target.projectID, DomainID: target.domainID, System: target.system})
			}},
			{http.MethodDelete, func(api *roles.API) error {
				return api.Unassign(ctx, "r-1", roles.UnassignOpts{UserID: target.userID, GroupID: target.groupID, ProjectID: target.projectID, DomainID: target.domainID, System: target.system})
			}},
			{http.MethodHead, func(api *roles.API) error {
				return api.Validate(ctx, "r-1", roles.ValidateOpts{UserID: target.userID, GroupID: target.groupID, ProjectID: target.projectID, DomainID: target.domainID, System: target.system})
			}},
		}
		for _, op := range ops {
			t.Run(target.name+" "+op.method, func(t *testing.T) {
				for _, status := range []int{204, 404, 200} {
					var calls []pythonRoleCall
					api := pythonRoleAPI(t, &calls, func(*http.Request) (int, string) { return status, "" })
					err := op.call(api)
					// Go errors for any status but 204; Python returns None for
					// assign/unassign and False for validate.
					if (status == 204) != (err == nil) || status != 204 && !gophercloud.ResponseCodeIs(err, status) {
						t.Fatal(status, err)
					}
					if !reflect.DeepEqual(calls, []pythonRoleCall{{op.method, target.path, "", ""}}) {
						t.Fatalf("%+v", calls)
					}
				}
			})
		}
	}
}
