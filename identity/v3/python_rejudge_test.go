package v3_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	v3 "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const (
	pythonRejudgeDomainRow  = `{"id":"dom-1","name":"dev","enabled":true}`
	pythonRejudgeGroupRow   = `{"id":"g-1","name":"ops","domain_id":"dom-1"}`
	pythonRejudgeRoleRow    = `{"id":"r-1","name":"admin","domain_id":"dom-1"}`
	pythonRejudgeProjectRow = `{"id":"p-1","name":"demo","domain_id":"dom-1","enabled":true}`
	pythonRejudgeUserRow    = `{"id":"u-1","name":"alice","domain_id":"dom-1","enabled":true}`
)

// pythonRejudgeList is a one-page Keystone list reply for kind.
func pythonRejudgeList(kind string, rows ...string) struct {
	code int
	body string
} {
	body := `{"` + kind + `":[`
	for i, row := range rows {
		if i > 0 {
			body += ","
		}
		body += row
	}
	return struct {
		code int
		body string
	}{200, body + `],"links":{"next":null}}`}
}

// pythonRejudgeReplies merges reply maps; later maps win.
func pythonRejudgeReplies(maps ...pythonCloudIdentityReplies) pythonCloudIdentityReplies {
	merged := pythonCloudIdentityReplies{}
	for _, replies := range maps {
		for key, value := range replies {
			merged[key] = value
		}
	}
	return merged
}

// pythonRejudgeFindCalls are the two requests Resource.find sends for a name:
// the direct GET (404 here) and the name-filtered list.
func pythonRejudgeFindCalls(kind, name, query string) []pythonCloudIdentityCall {
	listQuery := "name=" + name
	if query != "" {
		listQuery = query + "&" + listQuery
	}
	return []pythonCloudIdentityCall{{"GET", "/" + kind + "/" + name, query, ""}, {"GET", "/" + kind, listQuery, ""}}
}

func TestPythonRejudgeFindDomain(t *testing.T) {
	ctx := context.Background()
	byID := pythonCloudIdentityReplies{"GET /domains/dom-1": {200, `{"domain":` + pythonRejudgeDomainRow + `}`}}
	byName := pythonCloudIdentityReplies{"GET /domains": pythonRejudgeList("domains", pythonRejudgeDomainRow)}
	empty := pythonCloudIdentityReplies{"GET /domains": pythonRejudgeList("domains")}
	for _, tc := range []struct {
		name, input string
		options     []resource.IdentityFindOption
		replies     pythonCloudIdentityReplies
		wantID      string
		wantMissing bool
		want        []pythonCloudIdentityCall
	}{
		{"id GET only", "dom-1", nil, byID, "dom-1", false, []pythonCloudIdentityCall{{"GET", "/domains/dom-1", "", ""}}},
		{"name falls back to name list", "dev", nil, byName, "dom-1", false, pythonRejudgeFindCalls("domains", "dev", "")},
		{"ignore_missing=True returns nil", "ghost", nil, empty, "", false, pythonRejudgeFindCalls("domains", "ghost", "")},
		{"ignore_missing=False raises", "ghost", []resource.IdentityFindOption{resource.WithIdentityFindIgnoreMissing(false)}, empty, "", true, pythonRejudgeFindCalls("domains", "ghost", "")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonCloudIdentityCall
			svc := pythonCloudIdentityService(t, &calls, tc.replies)
			got, err := svc.Domains.FindIdentity(ctx, tc.input, tc.options...)
			switch {
			case tc.wantMissing:
				if got != nil || !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(got, err)
				}
			case tc.wantID == "":
				if got != nil || err != nil {
					t.Fatal(got, err)
				}
			default:
				if err != nil || got == nil || got.ID != tc.wantID || got.Name != "dev" || !got.Enabled {
					t.Fatal(got, err)
				}
			}
			pythonCloudIdentityExpect(t, calls, tc.want...)
		})
	}
}

// delete_domain(name_or_id=...) finds the domain, disables it and deletes it
// with ignore_missing=False.
func TestPythonRejudgeCloudDeleteDomainByName(t *testing.T) {
	ctx := context.Background()
	disabled := false
	deleteDomain := func(svc *v3.Service, nameOrID string) (bool, error) {
		found, err := svc.Domains.FindIdentity(ctx, nameOrID)
		if err != nil || found == nil {
			return false, err
		}
		if _, err := svc.Domains.Update(ctx, found.ID, domains.UpdateOpts{Enabled: &disabled}); err != nil {
			return false, err
		}
		return true, svc.Domains.Delete(ctx, found.ID)
	}
	disable := pythonCloudIdentityCall{"PATCH", "/domains/dom-1", "", `{"domain":{"enabled":false}}`}
	remove := pythonCloudIdentityCall{"DELETE", "/domains/dom-1", "", ""}
	for _, tc := range []struct {
		name, input string
		deleteCode  int
		wantDeleted bool
		check       func(error) bool
		want        []pythonCloudIdentityCall
	}{
		{"deleted", "dev", 204, true, func(err error) bool { return err == nil }, append(pythonRejudgeFindCalls("domains", "dev", ""), disable, remove)},
		{"delete 404 raises", "dev", 404, true, func(err error) bool { return gophercloud.ResponseCodeIs(err, http.StatusNotFound) }, append(pythonRejudgeFindCalls("domains", "dev", ""), disable, remove)},
		{"missing returns False without PATCH", "ghost", 204, false, func(err error) bool { return err == nil }, pythonRejudgeFindCalls("domains", "ghost", "")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonCloudIdentityCall
			rows := []string{pythonRejudgeDomainRow}
			if tc.input == "ghost" {
				rows = nil
			}
			svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{
				"GET /domains":          pythonRejudgeList("domains", rows...),
				"PATCH /domains/dom-1":  {200, `{"domain":{"id":"dom-1","name":"dev","enabled":false}}`},
				"DELETE /domains/dom-1": {tc.deleteCode, ""},
			})
			deleted, err := deleteDomain(svc, tc.input)
			if deleted != tc.wantDeleted || !tc.check(err) {
				t.Fatal(deleted, err)
			}
			pythonCloudIdentityExpect(t, calls, tc.want...)
		})
	}
}

// delete_group, delete_role, delete_project and delete_user find with
// ignore_missing=True and then call the Proxy delete with its default
// ignore_missing=True, which Remove reproduces.
func TestPythonRejudgeCloudFindThenRemove(t *testing.T) {
	ctx := context.Background()
	type remover func(svc *v3.Service, nameOrID string, query []resource.IdentityFindOption) (bool, error)
	findRemove := func(find func(*v3.Service, string, []resource.IdentityFindOption) (string, error), remove func(*v3.Service, string) error) remover {
		return func(svc *v3.Service, nameOrID string, query []resource.IdentityFindOption) (bool, error) {
			id, err := find(svc, nameOrID, query)
			if err != nil || id == "" {
				return false, err
			}
			return true, remove(svc, id)
		}
	}
	kinds := []struct {
		helper, kind, row, name, id, domainQuery string
		run                                      remover
	}{
		{"delete_group", "groups", pythonRejudgeGroupRow, "ops", "g-1", "", findRemove(
			func(svc *v3.Service, x string, o []resource.IdentityFindOption) (string, error) {
				got, err := svc.Groups.FindIdentity(ctx, x, o...)
				if got == nil {
					return "", err
				}
				return got.ID, err
			},
			func(svc *v3.Service, id string) error { return svc.Groups.Remove(ctx, resource.ID(id)) })},
		{"delete_role", "roles", pythonRejudgeRoleRow, "admin", "r-1", "domain_id=dom-1", findRemove(
			func(svc *v3.Service, x string, o []resource.IdentityFindOption) (string, error) {
				got, err := svc.Roles.FindIdentity(ctx, x, o...)
				if got == nil {
					return "", err
				}
				return got.ID, err
			},
			func(svc *v3.Service, id string) error { return svc.Roles.Remove(ctx, resource.ID(id)) })},
		{"delete_project", "projects", pythonRejudgeProjectRow, "demo", "p-1", "domain_id=dom-1", findRemove(
			func(svc *v3.Service, x string, o []resource.IdentityFindOption) (string, error) {
				got, err := svc.Projects.FindIdentity(ctx, x, o...)
				if got == nil {
					return "", err
				}
				return got.ID, err
			},
			func(svc *v3.Service, id string) error { return svc.Projects.Remove(ctx, resource.ID(id)) })},
		{"delete_user", "users", pythonRejudgeUserRow, "alice", "u-1", "domain_id=dom-1", findRemove(
			func(svc *v3.Service, x string, o []resource.IdentityFindOption) (string, error) {
				got, err := svc.Users.FindIdentity(ctx, x, o...)
				if got == nil {
					return "", err
				}
				return got.ID, err
			},
			func(svc *v3.Service, id string) error { return svc.Users.Remove(ctx, resource.ID(id)) })},
	}
	for _, kind := range kinds {
		t.Run(kind.helper, func(t *testing.T) {
			queries := []string{""}
			if kind.domainQuery != "" {
				queries = append(queries, kind.domainQuery)
			}
			for _, query := range queries {
				var options []resource.IdentityFindOption
				if query != "" {
					options = append(options, resource.WithIdentityFindQuery("domain_id", "dom-1"))
				}
				remove := pythonCloudIdentityCall{"DELETE", "/" + kind.kind + "/" + kind.id, "", ""}
				for _, tc := range []struct {
					name, input string
					deleteCode  int
					wantDeleted bool
					check       func(error) bool
					want        []pythonCloudIdentityCall
				}{
					{"deleted " + query, kind.name, 204, true, func(err error) bool { return err == nil }, append(pythonRejudgeFindCalls(kind.kind, kind.name, query), remove)},
					{"delete 404 ignored " + query, kind.name, 404, true, func(err error) bool { return err == nil }, append(pythonRejudgeFindCalls(kind.kind, kind.name, query), remove)},
					{"delete 409 returned " + query, kind.name, 409, true, func(err error) bool { return gophercloud.ResponseCodeIs(err, http.StatusConflict) }, append(pythonRejudgeFindCalls(kind.kind, kind.name, query), remove)},
					{"missing returns False " + query, "ghost", 204, false, func(err error) bool { return err == nil }, pythonRejudgeFindCalls(kind.kind, "ghost", query)},
				} {
					t.Run(tc.name, func(t *testing.T) {
						var calls []pythonCloudIdentityCall
						rows := []string{kind.row}
						if tc.input == "ghost" {
							rows = nil
						}
						svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{
							"GET /" + kind.kind:                    pythonRejudgeList(kind.kind, rows...),
							"DELETE /" + kind.kind + "/" + kind.id: {tc.deleteCode, ""},
						})
						deleted, err := kind.run(svc, tc.input, options)
						if deleted != tc.wantDeleted || !tc.check(err) {
							t.Fatal(deleted, err)
						}
						pythonCloudIdentityExpect(t, calls, tc.want...)
					})
				}
			}
		})
	}
}

// get_user(name_or_id, domain_id=None) is Proxy.find_user with
// ignore_missing=True and the domain_id query.
func TestPythonRejudgeCloudGetUser(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name, input, query string
		options            []resource.IdentityFindOption
		replies            pythonCloudIdentityReplies
		wantID             string
		want               []pythonCloudIdentityCall
	}{
		{"by id", "u-1", "", nil, pythonCloudIdentityReplies{"GET /users/u-1": {200, `{"user":` + pythonRejudgeUserRow + `}`}}, "u-1", []pythonCloudIdentityCall{{"GET", "/users/u-1", "", ""}}},
		{"by name", "alice", "", nil, pythonCloudIdentityReplies{"GET /users": pythonRejudgeList("users", pythonRejudgeUserRow)}, "u-1", pythonRejudgeFindCalls("users", "alice", "")},
		{"by name in domain", "alice", "domain_id=dom-1", []resource.IdentityFindOption{resource.WithIdentityFindQuery("domain_id", "dom-1")}, pythonCloudIdentityReplies{"GET /users": pythonRejudgeList("users", pythonRejudgeUserRow)}, "u-1", pythonRejudgeFindCalls("users", "alice", "domain_id=dom-1")},
		{"missing returns None", "ghost", "", nil, pythonCloudIdentityReplies{"GET /users": pythonRejudgeList("users")}, "", pythonRejudgeFindCalls("users", "ghost", "")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonCloudIdentityCall
			svc := pythonCloudIdentityService(t, &calls, tc.replies)
			got, err := svc.Users.FindIdentity(ctx, tc.input, tc.options...)
			if err != nil || (got == nil) != (tc.wantID == "") || got != nil && (got.ID != tc.wantID || got.Name != "alice" || got.DomainID != "dom-1" || !got.Enabled) {
				t.Fatal(got, err)
			}
			pythonCloudIdentityExpect(t, calls, tc.want...)
		})
	}
}

// update_role, update_project and update_user find the target and then PATCH
// only the attributes the caller passes.
func TestPythonRejudgeCloudFindThenUpdate(t *testing.T) {
	ctx := context.Background()
	disabled := false
	description := "d"

	t.Run("update_role", func(t *testing.T) {
		for _, query := range []string{"", "domain_id=dom-1"} {
			var options []resource.IdentityFindOption
			if query != "" {
				options = append(options, resource.WithIdentityFindQuery("domain_id", "dom-1"))
			}
			var calls []pythonCloudIdentityCall
			svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{
				"GET /roles":       pythonRejudgeList("roles", pythonRejudgeRoleRow),
				"PATCH /roles/r-1": {200, `{"role":{"id":"r-1","name":"operator","domain_id":"dom-1"}}`},
			})
			role, err := svc.Roles.FindIdentity(ctx, "admin", options...)
			if err != nil || role == nil {
				t.Fatal(role, err)
			}
			updated, err := svc.Roles.Update(ctx, role.ID, roles.UpdateOpts{Name: "operator"})
			if err != nil || updated.ID != "r-1" || updated.Name != "operator" {
				t.Fatal(updated, err)
			}
			// update_role returns False for a missing role and sends no PATCH.
			missing, err := svc.Roles.FindIdentity(ctx, "ghost", options...)
			if err != nil || missing != nil {
				t.Fatal(missing, err)
			}
			want := append(pythonRejudgeFindCalls("roles", "admin", query), pythonCloudIdentityCall{"PATCH", "/roles/r-1", "", `{"role":{"name":"operator"}}`})
			want = append(want, pythonRejudgeFindCalls("roles", "ghost", query)...)
			pythonCloudIdentityExpect(t, calls, want...)
		}
	})

	t.Run("update_project", func(t *testing.T) {
		for _, query := range []string{"", "domain_id=dom-1"} {
			// update_project finds with ignore_missing=False.
			options := []resource.IdentityFindOption{resource.WithIdentityFindIgnoreMissing(false)}
			if query != "" {
				options = append(options, resource.WithIdentityFindQuery("domain_id", "dom-1"))
			}
			var calls []pythonCloudIdentityCall
			svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{
				"GET /projects":       pythonRejudgeList("projects", pythonRejudgeProjectRow),
				"PATCH /projects/p-1": {200, `{"project":{"id":"p-1","name":"demo","domain_id":"dom-1","enabled":false,"description":"d"}}`},
			})
			project, err := svc.Projects.FindIdentity(ctx, "demo", options...)
			if err != nil || project == nil {
				t.Fatal(project, err)
			}
			updated, err := svc.Projects.Update(ctx, project.ID, projects.UpdateOpts{Enabled: &disabled, Description: &description})
			if err != nil || updated.Enabled || updated.Description != "d" {
				t.Fatal(updated, err)
			}
			if missing, err := svc.Projects.FindIdentity(ctx, "ghost", options...); missing != nil || !errors.Is(err, resource.ErrNotFound) {
				t.Fatal(missing, err)
			}
			want := append(pythonRejudgeFindCalls("projects", "demo", query), pythonCloudIdentityCall{"PATCH", "/projects/p-1", "", `{"project":{"description":"d","enabled":false}}`})
			want = append(want, pythonRejudgeFindCalls("projects", "ghost", query)...)
			pythonCloudIdentityExpect(t, calls, want...)
		}
	})

	t.Run("update_user", func(t *testing.T) {
		for _, query := range []string{"", "domain_id=dom-1"} {
			var options []resource.IdentityFindOption
			if query != "" {
				options = append(options, resource.WithIdentityFindQuery("domain_id", "dom-1"))
			}
			var calls []pythonCloudIdentityCall
			svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{
				"GET /users":       pythonRejudgeList("users", pythonRejudgeUserRow),
				"PATCH /users/u-1": {200, `{"user":{"id":"u-1","name":"alice2","domain_id":"dom-1","enabled":false,"email":"a@example.org"}}`},
			})
			user, err := svc.Users.FindIdentity(ctx, "alice", options...)
			if err != nil || user == nil {
				t.Fatal(user, err)
			}
			updated, err := svc.Users.Update(ctx, user.ID, users.UpdateOpts{Name: "alice2", Enabled: &disabled, Extra: map[string]any{"email": "a@example.org"}})
			if err != nil || updated.Name != "alice2" || updated.Enabled || updated.Extra["email"] != "a@example.org" {
				t.Fatal(updated, err)
			}
			// update_user raises SDKException for a missing user; no PATCH follows.
			if missing, err := svc.Users.FindIdentity(ctx, "ghost", options...); err != nil || missing != nil {
				t.Fatal(missing, err)
			}
			want := append(pythonRejudgeFindCalls("users", "alice", query), pythonCloudIdentityCall{"PATCH", "/users/u-1", "", `{"user":{"email":"a@example.org","enabled":false,"name":"alice2"}}`})
			want = append(want, pythonRejudgeFindCalls("users", "ghost", query)...)
			pythonCloudIdentityExpect(t, calls, want...)
		}
	})
}

// create_user(name, default_project=..., domain_id=...) resolves the project
// through get_project before the POST.
func TestPythonRejudgeCloudCreateUserDefaultProject(t *testing.T) {
	ctx := context.Background()
	enabled := true
	for _, tc := range []struct {
		name, project string
		rows          []string
		body          string
	}{
		{"project found", "demo", []string{pythonRejudgeProjectRow}, `{"user":{"default_project_id":"p-1","domain_id":"dom-1","email":null,"enabled":true,"name":"alice"}}`},
		{"project missing drops default_project_id", "ghost", nil, `{"user":{"domain_id":"dom-1","email":null,"enabled":true,"name":"alice"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonCloudIdentityCall
			svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{
				"GET /projects": pythonRejudgeList("projects", tc.rows...),
				"POST /users":   {201, `{"user":` + pythonRejudgeUserRow + `}`},
			})
			project, err := svc.Projects.FindIdentity(ctx, tc.project)
			if err != nil {
				t.Fatal(err)
			}
			opts := users.CreateOpts{Name: "alice", DomainID: "dom-1", Enabled: &enabled, Extra: map[string]any{"email": nil}}
			if project != nil {
				opts.DefaultProjectID = project.ID
			}
			created, err := svc.Users.Create(ctx, opts)
			if err != nil || created.ID != "u-1" {
				t.Fatal(created, err)
			}
			want := append(pythonRejudgeFindCalls("projects", tc.project, ""), pythonCloudIdentityCall{"POST", "/users", "", tc.body})
			pythonCloudIdentityExpect(t, calls, want...)
		})
	}
}

// add_user_to_group, is_user_in_group and remove_user_from_group resolve the
// user and then the group with get_user/get_group before the membership call.
func TestPythonRejudgeCloudUserGroupMembership(t *testing.T) {
	ctx := context.Background()
	resolve := func(svc *v3.Service, user, group string) (string, string, bool, error) {
		foundUser, err := svc.Users.FindIdentity(ctx, user)
		if err != nil || foundUser == nil {
			return "", "", false, err
		}
		foundGroup, err := svc.Groups.FindIdentity(ctx, group)
		if err != nil || foundGroup == nil {
			return "", "", false, err
		}
		return foundUser.ID, foundGroup.ID, true, nil
	}
	lookups := append(pythonRejudgeFindCalls("users", "alice", ""), pythonRejudgeFindCalls("groups", "ops", "")...)
	for _, tc := range []struct {
		name string
		call func(svc *v3.Service, groupID, userID string) error
		want pythonCloudIdentityCall
	}{
		{"add_user_to_group", func(svc *v3.Service, g, u string) error { return svc.Users.AddToGroup(ctx, g, u) }, pythonCloudIdentityCall{"PUT", "/groups/g-1/users/u-1", "", ""}},
		{"is_user_in_group", func(svc *v3.Service, g, u string) error {
			member, err := svc.Users.IsMemberOfGroup(ctx, g, u)
			if err == nil && !member {
				return errors.New("want member")
			}
			return err
		}, pythonCloudIdentityCall{"HEAD", "/groups/g-1/users/u-1", "", ""}},
		{"remove_user_from_group", func(svc *v3.Service, g, u string) error { return svc.Users.RemoveFromGroup(ctx, g, u) }, pythonCloudIdentityCall{"DELETE", "/groups/g-1/users/u-1", "", ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonCloudIdentityCall
			svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{
				"GET /users":                        pythonRejudgeList("users", pythonRejudgeUserRow),
				"GET /groups":                       pythonRejudgeList("groups", pythonRejudgeGroupRow),
				tc.want.method + " " + tc.want.path: {204, ""},
			})
			userID, groupID, ok, err := resolve(svc, "alice", "ops")
			if err != nil || !ok {
				t.Fatal(ok, err)
			}
			if err := tc.call(svc, groupID, userID); err != nil {
				t.Fatal(err)
			}
			pythonCloudIdentityExpect(t, calls, append(append([]pythonCloudIdentityCall(nil), lookups...), tc.want)...)
		})
	}

	t.Run("missing group raises before the membership call", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{
			"GET /users":  pythonRejudgeList("users", pythonRejudgeUserRow),
			"GET /groups": pythonRejudgeList("groups"),
		})
		if _, _, ok, err := resolve(svc, "alice", "ghost"); ok || err != nil {
			t.Fatal(ok, err)
		}
		pythonCloudIdentityExpect(t, calls, append(pythonRejudgeFindCalls("users", "alice", ""), pythonRejudgeFindCalls("groups", "ghost", "")...)...)
	})

	t.Run("is_user_in_group false for 404", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{
			"GET /users":  pythonRejudgeList("users", pythonRejudgeUserRow),
			"GET /groups": pythonRejudgeList("groups", pythonRejudgeGroupRow),
		})
		userID, groupID, ok, err := resolve(svc, "alice", "ops")
		if err != nil || !ok {
			t.Fatal(ok, err)
		}
		if member, err := svc.Users.IsMemberOfGroup(ctx, groupID, userID); err != nil || member {
			t.Fatal(member, err)
		}
		pythonCloudIdentityExpect(t, calls, append(append([]pythonCloudIdentityCall(nil), lookups...), pythonCloudIdentityCall{"HEAD", "/groups/g-1/users/u-1", "", ""})...)
	})
}
