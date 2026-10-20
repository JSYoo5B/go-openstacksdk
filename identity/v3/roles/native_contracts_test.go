package roles_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeRoleTransport func(*http.Request) (*http.Response, error)

func (transport nativeRoleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeRoleCall struct{ method, path, query, body string }

func nativeRoleAPI(t *testing.T, calls *[]nativeRoleCall, reply func(*http.Request) (int, string)) (*roles.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeRoleTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeRoleCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return roles.New(client), cloud
}

func nativeRoleOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "roles" {
		t.Fatal("generated roles context", err, wrapped)
	}
}

const nativeRoleRow = `{"id":"r-1","name":"reader","domain_id":null,"description":"desc","options":{"immutable":true},"links":{"self":"x"},"color":"blue"}`

func TestNativeRoleRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeRoleCall
	var cloud *testcloud.Cloud
	api, cloud := nativeRoleAPI(t, &calls, func(req *http.Request) (int, string) {
		switch {
		case req.Method == http.MethodDelete:
			return 204, ""
		case req.Method == http.MethodPost:
			return 201, `{"role":` + nativeRoleRow + `}`
		case req.URL.Path == "/keystone/v3/roles":
			return 200, `{"roles":[` + nativeRoleRow + `],"links":{"next":"` + cloud.Server.URL + `/other/roles?page=2"}}`
		case req.URL.Path == "/other/roles":
			return 200, `{"roles":[{"id":"r-2","extra":{"k":"v"}}],"links":{"next":null}}`
		}
		return 200, `{"role":` + nativeRoleRow + `}`
	})
	blank := ""
	created, err := api.Create(ctx, roles.CreateOpts{Name: "reader", DomainID: "default", Description: "desc", Options: map[roles.Option]any{roles.Immutable: true}, Extra: map[string]any{"color": "blue"}}, roles.WithCreateField("x_extension", 1))
	// Remaining keys feed Extra, and description is copied there although it also has a field.
	if err != nil || !(created.ID == "r-1" && created.DomainID == "" && created.Description == "desc" && created.Options[roles.Immutable] == true && reflect.DeepEqual(created.Extra, map[string]any{"color": "blue", "description": "desc"})) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "r-1")
	if err != nil || got.Name != "reader" {
		t.Fatal(got, err)
	}
	if _, err := api.Update(ctx, "r-1", roles.UpdateOpts{Name: "viewer", Description: &blank, Options: map[roles.Option]any{roles.Immutable: false}, Extra: map[string]any{"color": nil}}, roles.WithUpdateField("x_extension", 2)); err != nil {
		t.Fatal(err)
	}
	var rows []*roles.Role
	for value, err := range api.List(ctx, roles.WithListOptions(roles.ListOpts{DomainID: "default", Name: "reader", Filters: map[string]string{"name__contains": "ea"}}), roles.WithListQuery("x", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "r-1" && reflect.DeepEqual(rows[1].Extra, map[string]any{"k": "v"})) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "r-1"); err != nil {
		t.Fatal(err)
	}
	base := "/keystone/v3/roles"
	want := []nativeRoleCall{
		{http.MethodPost, base, "", `{"role":{"color":"blue","description":"desc","domain_id":"default","name":"reader","options":{"immutable":true},"x_extension":1}}`},
		{http.MethodGet, base + "/r-1", "", ""},
		{http.MethodPatch, base + "/r-1", "", `{"role":{"color":null,"description":"","name":"viewer","options":{"immutable":false},"x_extension":2}}`},
		// The q:"-" Filters map also leaks as a stray "-" parameter before the real filter.
		{http.MethodGet, base, "-=%7B%27name__contains%27%3A%27ea%27%7D&domain_id=default&name=reader&name__contains=ea&x=1", ""},
		{http.MethodGet, "/other/roles", "page=2", ""},
		{http.MethodDelete, base + "/r-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeRoleAssignmentRoutesQueryAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeRoleCall
	var cloud *testcloud.Cloud
	api, cloud := nativeRoleAPI(t, &calls, func(req *http.Request) (int, string) {
		switch {
		case req.Method != http.MethodGet:
			return 204, ""
		case req.URL.Path == "/keystone/v3/role_assignments":
			return 200, `{"role_assignments":[{"role":{"id":"r-1","name":"reader"},"scope":{"project":{"id":"p-1","name":"proj","domain":{"id":"default","name":"Default"}}},"user":{"id":"u-1","name":"alice","domain":{"id":"default"}},"links":{"assignment":"x"}}],"links":{"next":"` + cloud.Server.URL + `/other/assignments?page=2"}}`
		case req.URL.Path == "/other/assignments":
			return 200, `{"role_assignments":[{"role":{"id":"r-2"},"scope":{"system":{"all":true}},"group":{"id":"g-1"}}],"links":{"next":null}}`
		}
		return 200, `{"roles":[{"id":"r-1","name":"reader"}],"links":{"next":null}}`
	})
	if err := api.Assign(ctx, "r-1", roles.AssignOpts{UserID: "u-1", ProjectID: "p-1"}); err != nil {
		t.Fatal(err)
	}
	if err := api.Assign(ctx, "r-1", roles.AssignOpts{GroupID: "g-1", DomainID: "d-1"}); err != nil {
		t.Fatal(err)
	}
	// WithAssignOptions replaces the positional options.
	if err := api.Assign(ctx, "r-1", roles.AssignOpts{}, roles.WithAssignOptions(roles.AssignOpts{UserID: "u-1", System: true})); err != nil {
		t.Fatal(err)
	}
	if err := api.Validate(ctx, "r-1", roles.ValidateOpts{GroupID: "g-1", System: true}); err != nil {
		t.Fatal(err)
	}
	if err := api.Unassign(ctx, "r-1", roles.UnassignOpts{UserID: "u-1", DomainID: "d-1"}); err != nil {
		t.Fatal(err)
	}
	var names []string
	for value, err := range api.ListAssignmentsOnResource(ctx, roles.WithListAssignmentsOnResourceOptions(roles.ListAssignmentsOnResourceOpts{UserID: "u-1", ProjectID: "p-1"})) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, value.Name)
	}
	for value, err := range api.ListAssignmentsOnResource(ctx, roles.WithListAssignmentsOnResourceOptions(roles.ListAssignmentsOnResourceOpts{GroupID: "g-1", System: true})) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, value.ID)
	}
	effective, yes := false, true
	var assignments []*roles.RoleAssignment
	for value, err := range api.ListAssignments(ctx, roles.WithListAssignmentsOptions(roles.ListAssignmentsOpts{GroupID: "g-1", RoleID: "r-1", ScopeDomainID: "d-1", ScopeProjectID: "p-1", ScopeSystem: "all", UserID: "u-1", Effective: &effective, IncludeNames: &yes, IncludeSubtree: &yes}), roles.WithListAssignmentsQuery("x", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		assignments = append(assignments, value)
	}
	if !reflect.DeepEqual(names, []string{"reader", "r-1"}) || len(assignments) != 2 {
		t.Fatal(names, assignments)
	}
	first, second := assignments[0], assignments[1]
	if !(first.Role.Name == "reader" && first.Scope.Project.Domain.Name == "Default" && first.Scope.System == nil && first.User.ID == "u-1" && first.Group.ID == "") ||
		!(second.Scope.System != nil && second.Scope.System.All && second.Group.ID == "g-1" && second.Scope.Project.ID == "") {
		t.Fatal(first, second)
	}
	want := []nativeRoleCall{
		{http.MethodPut, "/keystone/v3/projects/p-1/users/u-1/roles/r-1", "", ""},
		{http.MethodPut, "/keystone/v3/domains/d-1/groups/g-1/roles/r-1", "", ""},
		{http.MethodPut, "/keystone/v3/system/users/u-1/roles/r-1", "", ""},
		{http.MethodHead, "/keystone/v3/system/groups/g-1/roles/r-1", "", ""},
		{http.MethodDelete, "/keystone/v3/domains/d-1/users/u-1/roles/r-1", "", ""},
		{http.MethodGet, "/keystone/v3/projects/p-1/users/u-1/roles", "", ""},
		{http.MethodGet, "/keystone/v3/system/groups/g-1/roles", "", ""},
		// Booleans are sent as true/false text; Keystone treats any value but 0 as set.
		{http.MethodGet, "/keystone/v3/role_assignments", "effective=false&group.id=g-1&include_names=true&include_subtree=true&role.id=r-1&scope.domain.id=d-1&scope.project.id=p-1&scope.system=all&user.id=u-1&x=1", ""},
		{http.MethodGet, "/other/assignments", "page=2", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeRoleInferenceRoutesAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeRoleCall
	rule := `{"role_inference":{"prior_role":{"id":"r-1","name":"admin","links":{"self":"a"}},"implies":{"id":"r-2","name":"member","links":{"self":"b"}}},"links":{"self":"c"}}`
	api, _ := nativeRoleAPI(t, &calls, func(req *http.Request) (int, string) {
		switch {
		case req.Method == http.MethodDelete:
			return 204, ""
		case req.Method == http.MethodPut:
			return 201, rule
		case req.URL.Path == "/keystone/v3/role_inferences":
			// The rule list is one response; links.next is not followed.
			return 200, `{"role_inferences":[{"prior_role":{"id":"r-1","name":"admin","description":"d"},"implies":[{"id":"r-2","name":"member"},{"id":"r-3"}]}],"links":{"self":"x","next":"http://never/next"}}`
		}
		return 200, rule
	})
	created, err := api.CreateRoleInferenceRule(ctx, "r-1", "r-2")
	if err != nil || !(created.RoleInference.PriorRole.ID == "r-1" && created.RoleInference.ImpliedRole.Name == "member" && created.Links["self"] == "c") {
		t.Fatal(created, err)
	}
	got, err := api.GetRoleInferenceRule(ctx, "r-1", "r-2")
	if err != nil || got.RoleInference.ImpliedRole.ID != "r-2" {
		t.Fatal(got, err)
	}
	list, err := api.ListRoleInferenceRules(ctx)
	if err != nil || !(len(list.RoleInferenceRuleList) == 1 && list.RoleInferenceRuleList[0].PriorRole.Description == "d" && len(list.RoleInferenceRuleList[0].ImpliedRoles) == 2 && list.Links["next"] == "http://never/next") {
		t.Fatal(list, err)
	}
	if err := api.DeleteRoleInferenceRule(ctx, "r-1", "r-2"); err != nil {
		t.Fatal(err)
	}
	want := []nativeRoleCall{
		{http.MethodPut, "/keystone/v3/roles/r-1/implies/r-2", "", ""},
		{http.MethodGet, "/keystone/v3/roles/r-1/implies/r-2", "", ""},
		{http.MethodGet, "/keystone/v3/role_inferences", "", ""},
		{http.MethodDelete, "/keystone/v3/roles/r-1/implies/r-2", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeRoleStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*roles.API) error
	}{
		{"Create", []int{201}, func(api *roles.API) error { _, err := api.Create(ctx, roles.CreateOpts{Name: "r"}); return err }},
		{"Get", []int{200}, func(api *roles.API) error { _, err := api.Get(ctx, "r-1"); return err }},
		{"Update", []int{200}, func(api *roles.API) error { _, err := api.Update(ctx, "r-1", roles.UpdateOpts{Name: "n"}); return err }},
		{"Delete", []int{202, 204}, func(api *roles.API) error { return api.Delete(ctx, "r-1") }},
		{"Assign", []int{204}, func(api *roles.API) error {
			return api.Assign(ctx, "r-1", roles.AssignOpts{UserID: "u-1", ProjectID: "p-1"})
		}},
		{"Validate", []int{204}, func(api *roles.API) error {
			return api.Validate(ctx, "r-1", roles.ValidateOpts{UserID: "u-1", ProjectID: "p-1"})
		}},
		{"Unassign", []int{204}, func(api *roles.API) error {
			return api.Unassign(ctx, "r-1", roles.UnassignOpts{UserID: "u-1", ProjectID: "p-1"})
		}},
		{"CreateRoleInferenceRule", []int{201}, func(api *roles.API) error { _, err := api.CreateRoleInferenceRule(ctx, "r-1", "r-2"); return err }},
		{"GetRoleInferenceRule", []int{200}, func(api *roles.API) error { _, err := api.GetRoleInferenceRule(ctx, "r-1", "r-2"); return err }},
		{"DeleteRoleInferenceRule", []int{204}, func(api *roles.API) error { return api.DeleteRoleInferenceRule(ctx, "r-1", "r-2") }},
		{"ListRoleInferenceRules", []int{200}, func(api *roles.API) error { _, err := api.ListRoleInferenceRules(ctx); return err }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeRoleCall
				api, _ := nativeRoleAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
				err := call.call(api)
				nativeRoleOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("inference results on error", func(t *testing.T) {
		var calls []nativeRoleCall
		api, _ := nativeRoleAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{}` })
		// The native inference extractors return a non-nil zero value together with the error.
		rule, err := api.GetRoleInferenceRule(ctx, "r-1", "r-2")
		if err == nil || rule == nil || rule.RoleInference.PriorRole.ID != "" {
			t.Fatal(rule, err)
		}
		list, err := api.ListRoleInferenceRules(ctx)
		if err == nil || list == nil || list.RoleInferenceRuleList != nil {
			t.Fatal(list, err)
		}
	})
	for name, list := range map[string]func(*roles.API) []error{
		"List": func(api *roles.API) (errs []error) {
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			return errs
		},
		"ListAssignments": func(api *roles.API) (errs []error) {
			for _, err := range api.ListAssignments(ctx) {
				errs = append(errs, err)
			}
			return errs
		},
		"ListAssignmentsOnResource": func(api *roles.API) (errs []error) {
			opts := roles.ListAssignmentsOnResourceOpts{UserID: "u-1", DomainID: "d-1"}
			for _, err := range api.ListAssignmentsOnResource(ctx, roles.WithListAssignmentsOnResourceOptions(opts)) {
				errs = append(errs, err)
			}
			return errs
		},
	} {
		t.Run(name+" pager status, empty page and bodyless 204", func(t *testing.T) {
			for _, tc := range []struct {
				code int
				body string
			}{{404, `{}`}, {200, `{"roles":[],"role_assignments":[],"links":{"next":null}}`}, {204, ""}} {
				var calls []nativeRoleCall
				api, _ := nativeRoleAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, tc.body })
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
		var calls []nativeRoleCall
		api, _ := nativeRoleAPI(t, &calls, func(*http.Request) (int, string) { return 204, `{}` })
		checks := map[string]struct {
			operation string
			err       error
		}{
			"create name": {"Create", func() error { _, err := api.Create(ctx, roles.CreateOpts{DomainID: "d"}); return err }()},
			"create core extension": {"Create", func() error {
				_, err := api.Create(ctx, roles.CreateOpts{Name: "r"}, roles.WithCreateField("options", map[string]any{}))
				return err
			}()},
			"update nil option": {"Update", func() error {
				_, err := api.Update(ctx, "r-1", roles.UpdateOpts{}, nil)
				return err
			}()},
			"assign nil option": {"Assign", api.Assign(ctx, "r-1", roles.AssignOpts{UserID: "u", ProjectID: "p"}, nil)},
		}
		for name, opts := range map[string]roles.AssignOpts{
			"no actor":   {ProjectID: "p"},
			"two actors": {UserID: "u", GroupID: "g", ProjectID: "p"},
			"no scope":   {UserID: "u"},
			"two scopes": {UserID: "u", ProjectID: "p", System: true},
		} {
			checks["assign "+name] = struct {
				operation string
				err       error
			}{"Assign", api.Assign(ctx, "r-1", opts)}
			checks["validate "+name] = struct {
				operation string
				err       error
			}{"Validate", api.Validate(ctx, "r-1", roles.ValidateOpts(opts))}
			checks["unassign "+name] = struct {
				operation string
				err       error
			}{"Unassign", api.Unassign(ctx, "r-1", roles.UnassignOpts(opts))}
		}
		for name, check := range checks {
			var missing gophercloud.ErrMissingInput
			if check.err == nil || (strings.Contains(name, " no ") || strings.Contains(name, " two ")) && !errors.As(check.err, &missing) {
				t.Fatal(name, check.err)
			}
			nativeRoleOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeRoleOperation(t, err, "List")
		}
		for _, err := range api.ListAssignments(ctx, nil) {
			nativeRoleOperation(t, err, "ListAssignments")
		}
		for _, err := range api.ListAssignmentsOnResource(ctx, nil) {
			nativeRoleOperation(t, err, "ListAssignmentsOnResource")
		}
		// Without options the native actor/scope check fails in the pager and is not wrapped.
		var native []error
		for _, err := range api.ListAssignmentsOnResource(ctx) {
			native = append(native, err)
		}
		for _, err := range api.List(ctx, roles.WithListOptions(roles.ListOpts{Filters: map[string]string{"x": "1"}})) {
			native = append(native, err)
		}
		var missing gophercloud.ErrMissingInput
		var invalid roles.InvalidListFilter
		var wrapped *resource.OperationError
		if len(native) != 2 || !errors.As(native[0], &missing) || errors.As(native[0], &wrapped) || !errors.As(native[1], &invalid) {
			t.Fatal(native)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
