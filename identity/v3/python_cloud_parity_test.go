package v3_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	v3 "github.com/JSYoo5B/go-openstacksdk/identity/v3"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/endpoints"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/services"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonCloudIdentityCall records one request relative to the identity v3 root.
type pythonCloudIdentityCall struct{ method, path, query, body string }

// pythonCloudIdentityReplies maps "METHOD /path" to a status and body; any
// other request gets a Keystone 404.
type pythonCloudIdentityReplies map[string]struct {
	code int
	body string
}

const pythonCloudIdentityRoot = "/keystone/v3"

func pythonCloudIdentityService(t *testing.T, calls *[]pythonCloudIdentityCall, replies pythonCloudIdentityReplies) *v3.Service {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		path := strings.TrimPrefix(req.URL.Path, pythonCloudIdentityRoot)
		*calls = append(*calls, pythonCloudIdentityCall{req.Method, path, req.URL.RawQuery, raw})
		code, body := 404, `{"error":{"code":404,"message":"not found"}}`
		if reply, ok := replies[req.Method+" "+path]; ok {
			code, body = reply.code, reply.body
		}
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + pythonCloudIdentityRoot + "/"
	return v3.New(client)
}

type pythonCloudIdentityFinder[T any] interface {
	Find(context.Context, resource.Ref, ...resource.LookupOption) (*T, error)
}

// pythonCloudIdentityFind mirrors Proxy.find_service/find_endpoint with the
// ignore_missing=True default the cloud get_* helpers use.
func pythonCloudIdentityFind[T any](ctx context.Context, api pythonCloudIdentityFinder[T], nameOrID string) (*T, error) {
	found, err := api.Find(ctx, resource.ID(nameOrID), resource.WithIgnoreMissing())
	if err != nil || found != nil {
		return found, err
	}
	return api.Find(ctx, resource.Name(nameOrID), resource.WithIgnoreMissing())
}

func pythonCloudIdentityExpect(t *testing.T, calls []pythonCloudIdentityCall, want ...pythonCloudIdentityCall) {
	t.Helper()
	if len(want) == 0 {
		want = nil
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls\n got %+v\nwant %+v", calls, want)
	}
}

func pythonCloudIdentityQuery(t *testing.T, raw string, want url.Values) {
	t.Helper()
	got, err := url.ParseQuery(raw)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("query %q: %v, want %v", raw, err, want)
	}
}

const (
	pythonCloudIdentityServiceRow  = `{"id":"svc-1","name":"nova","type":"compute","description":"d","enabled":true}`
	pythonCloudIdentityEndpointRow = `{"id":"ep-1","interface":"public","region_id":"RegionOne","service_id":"svc-1","url":"https://nova","enabled":true}`
)

func TestPythonCloudIdentityServiceHelpers(t *testing.T) {
	ctx := context.Background()
	enabled := true
	byName := pythonCloudIdentityReplies{
		"GET /services": {200, `{"services":[` + pythonCloudIdentityServiceRow + `],"links":{"next":null}}`},
	}
	byID := pythonCloudIdentityReplies{"GET /services/svc-1": {200, `{"service":` + pythonCloudIdentityServiceRow + `}`}}

	t.Run("create_service", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{"POST /services": {201, `{"service":` + pythonCloudIdentityServiceRow + `}`}})
		created, err := svc.Services.Create(ctx, services.CreateOpts{Name: "nova", Type: "compute", Description: "d", Enabled: &enabled})
		if err != nil || created.ID != "svc-1" || created.Name != "nova" || !created.Enabled {
			t.Fatal(created, err)
		}
		// Python create_service without type or service_type sends type null.
		if _, err := svc.Services.Create(ctx, services.CreateOpts{Name: "nova", Enabled: &enabled, Extra: map[string]any{"type": nil}}); err != nil {
			t.Fatal(err)
		}
		pythonCloudIdentityExpect(t, calls,
			pythonCloudIdentityCall{"POST", "/services", "", `{"service":{"description":"d","enabled":true,"name":"nova","type":"compute"}}`},
			pythonCloudIdentityCall{"POST", "/services", "", `{"service":{"enabled":true,"name":"nova","type":null}}`},
		)
	})

	t.Run("list_services", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		svc := pythonCloudIdentityService(t, &calls, byName)
		rows, err := svc.Services.All(ctx)
		if err != nil || len(rows) != 1 || rows[0].ID != "svc-1" || rows[0].Type != "compute" {
			t.Fatal(rows, err)
		}
		pythonCloudIdentityExpect(t, calls, pythonCloudIdentityCall{"GET", "/services", "", ""})
	})

	t.Run("get_service", func(t *testing.T) {
		for _, tc := range []struct {
			name, input, wantID string
			replies             pythonCloudIdentityReplies
			want                []pythonCloudIdentityCall
		}{
			{"id", "svc-1", "svc-1", byID, []pythonCloudIdentityCall{{"GET", "/services/svc-1", "", ""}}},
			{"name", "nova", "svc-1", byName, []pythonCloudIdentityCall{{"GET", "/services/nova", "", ""}, {"GET", "/services", "name=nova", ""}}},
			{"missing returns nil", "ghost", "", pythonCloudIdentityReplies{"GET /services": {200, `{"services":[],"links":{"next":null}}`}}, []pythonCloudIdentityCall{{"GET", "/services/ghost", "", ""}, {"GET", "/services", "name=ghost", ""}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var calls []pythonCloudIdentityCall
				svc := pythonCloudIdentityService(t, &calls, tc.replies)
				got, err := pythonCloudIdentityFind[services.Service](ctx, svc.Services, tc.input)
				if err != nil || (got == nil) != (tc.wantID == "") || got != nil && got.ID != tc.wantID {
					t.Fatal(got, err)
				}
				pythonCloudIdentityExpect(t, calls, tc.want...)
			})
		}
	})

	t.Run("update_service", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		replies := pythonCloudIdentityReplies{"PATCH /services/svc-1": {200, `{"service":{"id":"svc-1","name":"nova","type":"volume","description":"new","enabled":true}}`}}
		for k, v := range byName {
			replies[k] = v
		}
		svc := pythonCloudIdentityService(t, &calls, replies)
		found, err := pythonCloudIdentityFind[services.Service](ctx, svc.Services, "nova")
		if err != nil || found == nil {
			t.Fatal(found, err)
		}
		description := "new"
		updated, err := svc.Services.Update(ctx, found.ID, services.UpdateOpts{Description: &description, Type: "volume"})
		if err != nil || updated.Type != "volume" || updated.Description != "new" {
			t.Fatal(updated, err)
		}
		pythonCloudIdentityExpect(t, calls,
			pythonCloudIdentityCall{"GET", "/services/nova", "", ""},
			pythonCloudIdentityCall{"GET", "/services", "name=nova", ""},
			pythonCloudIdentityCall{"PATCH", "/services/svc-1", "", `{"service":{"description":"new","type":"volume"}}`},
		)
	})

	t.Run("delete_service", func(t *testing.T) {
		for _, tc := range []struct {
			name      string
			status    int
			input     string
			wantFound bool
			check     func(error) bool
		}{
			{"deleted", 204, "svc-1", true, func(err error) bool { return err == nil }},
			{"delete 404 ignored", 404, "svc-1", true, func(err error) bool { return err == nil }},
			{"delete conflict returned", 409, "svc-1", true, func(err error) bool { return gophercloud.ResponseCodeIs(err, http.StatusConflict) }},
			{"not found skips delete", 0, "ghost", false, func(err error) bool { return err == nil }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var calls []pythonCloudIdentityCall
				replies := pythonCloudIdentityReplies{"GET /services": {200, `{"services":[],"links":{"next":null}}`}, "DELETE /services/svc-1": {tc.status, ""}}
				for k, v := range byID {
					replies[k] = v
				}
				svc := pythonCloudIdentityService(t, &calls, replies)
				found, err := pythonCloudIdentityFind[services.Service](ctx, svc.Services, tc.input)
				if err != nil || (found != nil) != tc.wantFound {
					t.Fatal(found, err)
				}
				if found == nil {
					pythonCloudIdentityExpect(t, calls, pythonCloudIdentityCall{"GET", "/services/ghost", "", ""}, pythonCloudIdentityCall{"GET", "/services", "name=ghost", ""})
					return
				}
				if err := svc.Services.Remove(ctx, resource.ID(found.ID)); !tc.check(err) {
					t.Fatal(err)
				}
				pythonCloudIdentityExpect(t, calls, pythonCloudIdentityCall{"GET", "/services/svc-1", "", ""}, pythonCloudIdentityCall{"DELETE", "/services/svc-1", "", ""})
			})
		}
	})
}

func TestPythonCloudIdentityEndpointHelpers(t *testing.T) {
	ctx := context.Background()
	enabled, disabled := true, false

	t.Run("create_endpoint url and interface", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{
			"GET /services":   {200, `{"services":[` + pythonCloudIdentityServiceRow + `],"links":{"next":null}}`},
			"POST /endpoints": {201, `{"endpoint":` + pythonCloudIdentityEndpointRow + `}`},
		})
		service, err := pythonCloudIdentityFind[services.Service](ctx, svc.Services, "nova")
		if err != nil || service == nil {
			t.Fatal(service, err)
		}
		created, err := svc.Endpoints.Create(ctx, endpoints.CreateOpts{Availability: gophercloud.AvailabilityPublic, URL: "https://nova", ServiceID: service.ID, Enabled: &enabled}, endpoints.WithCreateField("region_id", nil))
		if err != nil || created.ID != "ep-1" || created.ServiceID != "svc-1" {
			t.Fatal(created, err)
		}
		pythonCloudIdentityExpect(t, calls,
			pythonCloudIdentityCall{"GET", "/services/nova", "", ""},
			pythonCloudIdentityCall{"GET", "/services", "name=nova", ""},
			pythonCloudIdentityCall{"POST", "/endpoints", "", `{"endpoint":{"enabled":true,"interface":"public","region_id":null,"service_id":"svc-1","url":"https://nova"}}`},
		)
	})

	t.Run("create_endpoint public internal admin urls", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{
			"GET /services/svc-1": {200, `{"service":` + pythonCloudIdentityServiceRow + `}`},
			"POST /endpoints":     {201, `{"endpoint":` + pythonCloudIdentityEndpointRow + `}`},
		})
		service, err := pythonCloudIdentityFind[services.Service](ctx, svc.Services, "svc-1")
		if err != nil || service == nil {
			t.Fatal(service, err)
		}
		want := []pythonCloudIdentityCall{{"GET", "/services/svc-1", "", ""}}
		for _, iface := range []gophercloud.Availability{gophercloud.AvailabilityPublic, gophercloud.AvailabilityInternal, gophercloud.AvailabilityAdmin} {
			endpointURL := "https://" + string(iface)
			if _, err := svc.Endpoints.Create(ctx, endpoints.CreateOpts{Availability: iface, URL: endpointURL, ServiceID: service.ID, Enabled: &enabled}, endpoints.WithCreateField("region_id", "RegionOne")); err != nil {
				t.Fatal(err)
			}
			want = append(want, pythonCloudIdentityCall{"POST", "/endpoints", "", `{"endpoint":{"enabled":true,"interface":"` + string(iface) + `","region_id":"RegionOne","service_id":"svc-1","url":"` + endpointURL + `"}}`})
		}
		pythonCloudIdentityExpect(t, calls, want...)
	})

	t.Run("update_endpoint", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{"PATCH /endpoints/ep-1": {200, `{"endpoint":` + pythonCloudIdentityEndpointRow + `}`}})
		if _, err := svc.Endpoints.Update(ctx, "ep-1", endpoints.UpdateOpts{ServiceID: "svc-2", Enabled: &disabled, URL: "https://new"}, endpoints.WithUpdateField("region_id", "R2")); err != nil {
			t.Fatal(err)
		}
		pythonCloudIdentityExpect(t, calls, pythonCloudIdentityCall{"PATCH", "/endpoints/ep-1", "", `{"endpoint":{"enabled":false,"region_id":"R2","service_id":"svc-2","url":"https://new"}}`})
	})

	t.Run("list_endpoints", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{"GET /endpoints": {200, `{"endpoints":[` + pythonCloudIdentityEndpointRow + `],"links":{"next":null}}`}})
		rows, err := svc.Endpoints.All(ctx)
		if err != nil || len(rows) != 1 || rows[0].ID != "ep-1" || rows[0].RegionID != "RegionOne" {
			t.Fatal(rows, err)
		}
		pythonCloudIdentityExpect(t, calls, pythonCloudIdentityCall{"GET", "/endpoints", "", ""})
	})

	t.Run("get_endpoint and delete_endpoint", func(t *testing.T) {
		for _, tc := range []struct {
			name, input string
			found       bool
			want        []pythonCloudIdentityCall
		}{
			{"id hit then delete", "ep-1", true, []pythonCloudIdentityCall{{"GET", "/endpoints/ep-1", "", ""}, {"DELETE", "/endpoints/ep-1", "", ""}}},
			// Endpoint declares no name query, so the fallback lists everything.
			{"missing skips delete", "ep-9", false, []pythonCloudIdentityCall{{"GET", "/endpoints/ep-9", "", ""}, {"GET", "/endpoints", "", ""}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var calls []pythonCloudIdentityCall
				svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{
					"GET /endpoints/ep-1":    {200, `{"endpoint":` + pythonCloudIdentityEndpointRow + `}`},
					"GET /endpoints":         {200, `{"endpoints":[` + pythonCloudIdentityEndpointRow + `],"links":{"next":null}}`},
					"DELETE /endpoints/ep-1": {204, ""},
				})
				found, err := pythonCloudIdentityFind[endpoints.Endpoint](ctx, svc.Endpoints, tc.input)
				if err != nil || (found != nil) != tc.found {
					t.Fatal(found, err)
				}
				if found != nil {
					if err := svc.Endpoints.Remove(ctx, resource.ID(tc.input)); err != nil {
						t.Fatal(err)
					}
				}
				pythonCloudIdentityExpect(t, calls, tc.want...)
			})
		}
	})
}

func TestPythonCloudIdentityDomainGroupRoleHelpers(t *testing.T) {
	ctx := context.Background()
	enabled := true

	t.Run("create_domain", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{"POST /domains": {201, `{"domain":{"id":"dom-1","name":"d","description":"x","enabled":true}}`}})
		created, err := svc.Domains.Create(ctx, domains.CreateOpts{Name: "d", Description: "x", Enabled: &enabled})
		if err != nil || created.ID != "dom-1" || !created.Enabled {
			t.Fatal(created, err)
		}
		if _, err := svc.Domains.Create(ctx, domains.CreateOpts{Name: "d", Enabled: &enabled}); err != nil {
			t.Fatal(err)
		}
		pythonCloudIdentityExpect(t, calls,
			pythonCloudIdentityCall{"POST", "/domains", "", `{"domain":{"description":"x","enabled":true,"name":"d"}}`},
			pythonCloudIdentityCall{"POST", "/domains", "", `{"domain":{"enabled":true,"name":"d"}}`},
		)
	})

	t.Run("list_groups", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{"GET /groups": {200, `{"groups":[{"id":"g-1","name":"ops","domain_id":"dom-1"}],"links":{"next":null}}`}})
		var ids []string
		for _, options := range [][]groups.ListOption{nil, {groups.WithListOptions(groups.ListOpts{DomainID: "dom-1"})}} {
			for group, err := range svc.Groups.List(ctx, options...) {
				if err != nil {
					t.Fatal(err)
				}
				ids = append(ids, group.ID+"/"+group.DomainID)
			}
		}
		if !reflect.DeepEqual(ids, []string{"g-1/dom-1", "g-1/dom-1"}) {
			t.Fatal(ids)
		}
		pythonCloudIdentityExpect(t, calls, pythonCloudIdentityCall{"GET", "/groups", "", ""}, pythonCloudIdentityCall{"GET", "/groups", "domain_id=dom-1", ""})
	})

	t.Run("create_group", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{
			"GET /domains/dom-1": {200, `{"domain":{"id":"dom-1","name":"d","enabled":true}}`},
			"POST /groups":       {201, `{"group":{"id":"g-1","name":"g","description":"desc","domain_id":"dom-1"}}`},
		})
		domain, err := svc.Domains.Get(ctx, "dom-1")
		if err != nil {
			t.Fatal(err)
		}
		created, err := svc.Groups.Create(ctx, groups.CreateOpts{Name: "g", Description: "desc", DomainID: domain.ID})
		if err != nil || created.ID != "g-1" || created.DomainID != "dom-1" {
			t.Fatal(created, err)
		}
		if _, err := svc.Groups.Create(ctx, groups.CreateOpts{Name: "g"}); err != nil {
			t.Fatal(err)
		}
		// identity.get_domain raises on 404, so no group POST follows.
		if _, err := svc.Domains.Get(ctx, "dom-9"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			t.Fatal(err)
		}
		pythonCloudIdentityExpect(t, calls,
			pythonCloudIdentityCall{"GET", "/domains/dom-1", "", ""},
			pythonCloudIdentityCall{"POST", "/groups", "", `{"group":{"description":"desc","domain_id":"dom-1","name":"g"}}`},
			pythonCloudIdentityCall{"POST", "/groups", "", `{"group":{"name":"g"}}`},
			pythonCloudIdentityCall{"GET", "/domains/dom-9", "", ""},
		)
	})

	t.Run("create_role", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{"POST /roles": {201, `{"role":{"id":"r-1","name":"r","domain_id":"dom-1"}}`}})
		created, err := svc.Roles.Create(ctx, roles.CreateOpts{Name: "r", DomainID: "dom-1"})
		if err != nil || created.ID != "r-1" || created.DomainID != "dom-1" {
			t.Fatal(created, err)
		}
		if _, err := svc.Roles.Create(ctx, roles.CreateOpts{Name: "r"}); err != nil {
			t.Fatal(err)
		}
		pythonCloudIdentityExpect(t, calls,
			pythonCloudIdentityCall{"POST", "/roles", "", `{"role":{"domain_id":"dom-1","name":"r"}}`},
			pythonCloudIdentityCall{"POST", "/roles", "", `{"role":{"name":"r"}}`},
		)
	})

	t.Run("list_role_assignments", func(t *testing.T) {
		row := `{"role":{"id":"r-1"},"scope":{"project":{"id":"p-1"}},"user":{"id":"u-1"},"links":{"assignment":"x"}}`
		for _, tc := range []struct {
			name    string
			options []roles.ListAssignmentsOption
			query   url.Values
		}{
			{"no filters", nil, url.Values{}},
			{"user project role effective inherited_to", []roles.ListAssignmentsOption{
				roles.WithListAssignmentsOptions(roles.ListAssignmentsOpts{UserID: "u-1", ScopeProjectID: "p-1", RoleID: "r-1"}),
				roles.WithListAssignmentsQuery("effective", "True"),
				roles.WithListAssignmentsQuery("scope.OS-INHERIT:inherited_to", "projects"),
			}, url.Values{"user.id": {"u-1"}, "scope.project.id": {"p-1"}, "role.id": {"r-1"}, "effective": {"True"}, "scope.OS-INHERIT:inherited_to": {"projects"}}},
			{"group domain", []roles.ListAssignmentsOption{roles.WithListAssignmentsOptions(roles.ListAssignmentsOpts{GroupID: "g-1", ScopeDomainID: "dom-1"})}, url.Values{"group.id": {"g-1"}, "scope.domain.id": {"dom-1"}}},
			{"system", []roles.ListAssignmentsOption{roles.WithListAssignmentsOptions(roles.ListAssignmentsOpts{UserID: "u-1", ScopeSystem: "all"})}, url.Values{"user.id": {"u-1"}, "scope.system": {"all"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var calls []pythonCloudIdentityCall
				svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{"GET /role_assignments": {200, `{"role_assignments":[` + row + `],"links":{"next":null}}`}})
				var got []roles.RoleAssignment
				for assignment, err := range svc.Roles.ListAssignments(ctx, tc.options...) {
					if err != nil {
						t.Fatal(err)
					}
					got = append(got, *assignment)
				}
				if len(got) != 1 || got[0].Role.ID != "r-1" || got[0].Scope.Project.ID != "p-1" || got[0].User.ID != "u-1" {
					t.Fatalf("%+v", got)
				}
				if len(calls) != 1 || calls[0].method != "GET" || calls[0].path != "/role_assignments" {
					t.Fatalf("%+v", calls)
				}
				pythonCloudIdentityQuery(t, calls[0].query, tc.query)
			})
		}
	})
}

func TestPythonCloudIdentityProjectUserHelpers(t *testing.T) {
	ctx := context.Background()
	enabled := true

	t.Run("create_project", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{"POST /projects": {201, `{"project":{"id":"p-1","name":"p","domain_id":"dom-1","enabled":true,"description":null}}`}})
		// Python passes description=None explicitly, so the default body has null.
		created, err := svc.Projects.Create(ctx, projects.CreateOpts{Name: "p", DomainID: "dom-1", Enabled: &enabled, Extra: map[string]any{"description": nil}})
		if err != nil || created.ID != "p-1" || created.DomainID != "dom-1" || !created.Enabled {
			t.Fatal(created, err)
		}
		if _, err := svc.Projects.Create(ctx, projects.CreateOpts{Name: "p", DomainID: "dom-1", Description: "d", Enabled: &enabled, ParentID: "par"}); err != nil {
			t.Fatal(err)
		}
		pythonCloudIdentityExpect(t, calls,
			pythonCloudIdentityCall{"POST", "/projects", "", `{"project":{"description":null,"domain_id":"dom-1","enabled":true,"name":"p"}}`},
			pythonCloudIdentityCall{"POST", "/projects", "", `{"project":{"description":"d","domain_id":"dom-1","enabled":true,"name":"p","parent_id":"par"}}`},
		)
	})

	t.Run("list_users", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{"GET /users": {200, `{"users":[{"id":"u-1","name":"alice","domain_id":"dom-1","enabled":true}],"links":{"next":null}}`}})
		for _, options := range [][]users.ListOption{nil, {users.WithListOptions(users.ListOpts{DomainID: "dom-1", Name: "alice"})}} {
			for user, err := range svc.Users.List(ctx, options...) {
				if err != nil || user.ID != "u-1" || user.Name != "alice" {
					t.Fatal(user, err)
				}
			}
		}
		if len(calls) != 2 || calls[0] != (pythonCloudIdentityCall{"GET", "/users", "", ""}) || calls[1].path != "/users" {
			t.Fatalf("%+v", calls)
		}
		pythonCloudIdentityQuery(t, calls[1].query, url.Values{"domain_id": {"dom-1"}, "name": {"alice"}})
	})

	t.Run("get_user_by_id", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{"GET /users/u-1": {200, `{"user":{"id":"u-1","name":"alice","domain_id":"dom-1","enabled":true,"email":"a@example.org"}}`}})
		user, err := svc.Users.Get(ctx, "u-1")
		if err != nil || user.ID != "u-1" || user.Extra["email"] != "a@example.org" {
			t.Fatal(user, err)
		}
		if _, err := svc.Users.Get(ctx, "u-9"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			t.Fatal(err)
		}
		pythonCloudIdentityExpect(t, calls, pythonCloudIdentityCall{"GET", "/users/u-1", "", ""}, pythonCloudIdentityCall{"GET", "/users/u-9", "", ""})
	})
}

// The helpers below stay unresolved; these subtests pin only the parts the Go
// API reproduces once IDs are known.
func TestPythonCloudIdentityUnresolvedPartialSequences(t *testing.T) {
	ctx := context.Background()
	enabled, disabled := true, false

	t.Run("FindIdentity sequences behind get_*", func(t *testing.T) {
		for _, tc := range []struct {
			kind, query string
			find        func(*v3.Service) (bool, error)
		}{
			{"projects", "domain_id=dom-1", func(svc *v3.Service) (bool, error) {
				got, err := svc.Projects.FindIdentity(ctx, "demo", resource.WithIdentityFindQuery("domain_id", "dom-1"))
				return got != nil, err
			}},
			{"users", "domain_id=dom-1", func(svc *v3.Service) (bool, error) {
				got, err := svc.Users.FindIdentity(ctx, "demo", resource.WithIdentityFindQuery("domain_id", "dom-1"))
				return got != nil, err
			}},
			{"groups", "domain_id=dom-1", func(svc *v3.Service) (bool, error) {
				got, err := svc.Groups.FindIdentity(ctx, "demo", resource.WithIdentityFindQuery("domain_id", "dom-1"))
				return got != nil, err
			}},
			{"roles", "domain_id=dom-1", func(svc *v3.Service) (bool, error) {
				got, err := svc.Roles.FindIdentity(ctx, "demo", resource.WithIdentityFindQuery("domain_id", "dom-1"))
				return got != nil, err
			}},
			{"domains", "", func(svc *v3.Service) (bool, error) {
				got, err := svc.Domains.FindIdentity(ctx, "demo")
				return got != nil, err
			}},
		} {
			t.Run(tc.kind, func(t *testing.T) {
				var calls []pythonCloudIdentityCall
				svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{"GET /" + tc.kind: {200, `{"` + tc.kind + `":[{"id":"x-1","name":"demo"}],"links":{"next":null}}`}})
				found, err := tc.find(svc)
				if err != nil || !found {
					t.Fatal(found, err)
				}
				if len(calls) != 2 || calls[0] != (pythonCloudIdentityCall{"GET", "/" + tc.kind + "/demo", tc.query, ""}) || calls[1].path != "/"+tc.kind {
					t.Fatalf("%+v", calls)
				}
				want, _ := url.ParseQuery(tc.query)
				want.Set("name", "demo")
				pythonCloudIdentityQuery(t, calls[1].query, want)
			})
		}
	})

	t.Run("update_domain and delete_domain by domain_id", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		row := `{"domain":{"id":"dom-1","name":"n","enabled":false}}`
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{"PATCH /domains/dom-1": {200, row}, "DELETE /domains/dom-1": {204, ""}})
		// update_domain drops falsy name/description and keeps enabled=False.
		if _, err := svc.Domains.Update(ctx, "dom-1", domains.UpdateOpts{Name: "n", Enabled: &disabled}); err != nil {
			t.Fatal(err)
		}
		// delete_domain disables first, then deletes with ignore_missing=False.
		if _, err := svc.Domains.Update(ctx, "dom-1", domains.UpdateOpts{Enabled: &disabled}); err != nil {
			t.Fatal(err)
		}
		if err := svc.Domains.Delete(ctx, "dom-1"); err != nil {
			t.Fatal(err)
		}
		pythonCloudIdentityExpect(t, calls,
			pythonCloudIdentityCall{"PATCH", "/domains/dom-1", "", `{"domain":{"enabled":false,"name":"n"}}`},
			pythonCloudIdentityCall{"PATCH", "/domains/dom-1", "", `{"domain":{"enabled":false}}`},
			pythonCloudIdentityCall{"DELETE", "/domains/dom-1", "", ""},
		)
	})

	t.Run("create_user without default_project", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{"POST /users": {201, `{"user":{"id":"u-1","name":"alice","domain_id":"dom-1","enabled":true}}`}})
		created, err := svc.Users.Create(ctx, users.CreateOpts{Name: "alice", DomainID: "dom-1", Enabled: &enabled, Password: "fixture-pw", Extra: map[string]any{"email": nil}})
		if err != nil || created.ID != "u-1" {
			t.Fatal(created, err)
		}
		pythonCloudIdentityExpect(t, calls, pythonCloudIdentityCall{"POST", "/users", "", `{"user":{"domain_id":"dom-1","email":null,"enabled":true,"name":"alice","password":"fixture-pw"}}`})
	})

	t.Run("group membership by IDs", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{
			"PUT /groups/g-1/users/u-1":    {204, ""},
			"HEAD /groups/g-1/users/u-1":   {204, ""},
			"DELETE /groups/g-1/users/u-1": {204, ""},
		})
		if err := svc.Users.AddToGroup(ctx, "g-1", "u-1"); err != nil {
			t.Fatal(err)
		}
		if member, err := svc.Users.IsMemberOfGroup(ctx, "g-1", "u-1"); err != nil || !member {
			t.Fatal(member, err)
		}
		if member, err := svc.Users.IsMemberOfGroup(ctx, "g-1", "u-9"); err != nil || member {
			t.Fatal(member, err)
		}
		if err := svc.Users.RemoveFromGroup(ctx, "g-1", "u-1"); err != nil {
			t.Fatal(err)
		}
		pythonCloudIdentityExpect(t, calls,
			pythonCloudIdentityCall{"PUT", "/groups/g-1/users/u-1", "", ""},
			pythonCloudIdentityCall{"HEAD", "/groups/g-1/users/u-1", "", ""},
			pythonCloudIdentityCall{"HEAD", "/groups/g-1/users/u-9", "", ""},
			pythonCloudIdentityCall{"DELETE", "/groups/g-1/users/u-1", "", ""},
		)
	})

	t.Run("grant_role and revoke_role on a project by IDs", func(t *testing.T) {
		var calls []pythonCloudIdentityCall
		path := "/projects/p-1/users/u-1/roles/r-1"
		svc := pythonCloudIdentityService(t, &calls, pythonCloudIdentityReplies{"PUT " + path: {204, ""}, "DELETE " + path: {204, ""}})
		target := roles.ValidateOpts{UserID: "u-1", ProjectID: "p-1"}
		// Python treats the 404 HEAD as "no assignment"; Go returns the 404.
		if err := svc.Roles.Validate(ctx, "r-1", target); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			t.Fatal(err)
		}
		if err := svc.Roles.Assign(ctx, "r-1", roles.AssignOpts{UserID: "u-1", ProjectID: "p-1"}); err != nil {
			t.Fatal(err)
		}
		if err := svc.Roles.Unassign(ctx, "r-1", roles.UnassignOpts{UserID: "u-1", ProjectID: "p-1"}); err != nil {
			t.Fatal(err)
		}
		pythonCloudIdentityExpect(t, calls,
			pythonCloudIdentityCall{"HEAD", path, "", ""},
			pythonCloudIdentityCall{"PUT", path, "", ""},
			pythonCloudIdentityCall{"DELETE", path, "", ""},
		)
	})
}
