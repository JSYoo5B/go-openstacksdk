package projects_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeProjectTransport func(*http.Request) (*http.Response, error)

func (transport nativeProjectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeProjectCall struct{ method, path, query, body string }

func nativeProjectAPI(t *testing.T, calls *[]nativeProjectCall, reply func(*http.Request) (int, string)) (*projects.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeProjectTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeProjectCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return projects.New(client), cloud
}

func nativeProjectOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "projects" {
		t.Fatal("generated projects context", err, wrapped)
	}
}

const nativeProjectRow = `{"id":"p-1","name":"proj","description":"desc","domain_id":"default","enabled":true,"is_domain":false,"parent_id":"default","tags":["a"],"options":{"immutable":true},"links":{"self":"x"},"owner":"ops"}`

func TestNativeProjectAdminRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeProjectCall
	var cloud *testcloud.Cloud
	api, cloud := nativeProjectAPI(t, &calls, func(req *http.Request) (int, string) {
		switch {
		case req.Method == http.MethodDelete:
			return 204, ""
		case req.Method == http.MethodPost:
			return 201, `{"project":` + nativeProjectRow + `}`
		case req.Method == http.MethodPut:
			// Only top-level projects and links are decoded; a top-level tags list is ignored.
			return 200, `{"projects":[` + nativeProjectRow + `],"links":{"self":"y"},"tags":["ignored"]}`
		case req.URL.Path == "/keystone/v3/projects/p-1/tags":
			return 200, `{"tags":["a","b"]}`
		case req.URL.Path == "/keystone/v3/projects":
			return 200, `{"projects":[` + nativeProjectRow + `],"links":{"next":"` + cloud.Server.URL + `/other/projects?page=2"}}`
		case req.URL.Path == "/other/projects":
			return 200, `{"projects":[{"id":"p-2","extra":{"k":"v"}}],"links":{"next":null}}`
		}
		return 200, `{"project":` + nativeProjectRow + `}`
	})
	disabled, yes, blank := false, true, ""
	noTags := []string{}
	created, err := api.Create(ctx, projects.CreateOpts{Name: "proj", DomainID: "default", Enabled: &disabled, IsDomain: &disabled, ParentID: "default", Description: "desc", Tags: []string{"a"}, Options: map[projects.Option]any{projects.Immutable: true}, Extra: map[string]any{"owner": "ops"}}, projects.WithCreateField("x_extension", 1))
	// Project has no Links field, so links joins the remaining keys in Extra.
	if err != nil || !(created.ID == "p-1" && created.Enabled && created.ParentID == "default" && reflect.DeepEqual(created.Tags, []string{"a"}) && created.Options[projects.Immutable] == true && reflect.DeepEqual(created.Extra, map[string]any{"owner": "ops", "links": map[string]any{"self": "x"}})) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "p-1")
	if err != nil || got.Name != "proj" {
		t.Fatal(got, err)
	}
	// A pointer to an empty tag list clears tags, unlike ModifyTags below.
	if _, err := api.Update(ctx, "p-1", projects.UpdateOpts{Name: "new", Description: &blank, Enabled: &yes, Tags: &noTags, Options: map[projects.Option]any{projects.Immutable: false}}, projects.WithUpdateField("x_extension", 2)); err != nil {
		t.Fatal(err)
	}
	var rows []*projects.Project
	for value, err := range api.List(ctx, projects.WithListOptions(projects.ListOpts{DomainID: "default", Enabled: &disabled, IsDomain: &yes, Name: "proj", ParentID: "root", Tags: "a,b", TagsAny: "c", NotTags: "d", NotTagsAny: "e", Limit: 5, Filters: map[string]string{"name__startswith": "pr"}}), projects.WithListQuery("x", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "p-1" && reflect.DeepEqual(rows[1].Extra, map[string]any{"k": "v"})) {
		t.Fatal(rows)
	}
	tags, err := api.ListTags(ctx, "p-1")
	if err != nil || !reflect.DeepEqual(tags.Tags, []string{"a", "b"}) {
		t.Fatal(tags, err)
	}
	modified, err := api.ModifyTags(ctx, "p-1", projects.ModifyTagsOpts{Tags: []string{"a", "b"}}, projects.WithModifyTagsField("x_extension", 3))
	if err != nil || !(len(modified.Projects) == 1 && modified.Projects[0].ID == "p-1" && modified.Links["self"] == "y") {
		t.Fatal(modified, err)
	}
	// An empty tag list is omitted, so ModifyTags cannot clear tags; DeleteTags does.
	if _, err := api.ModifyTags(ctx, "p-1", projects.ModifyTagsOpts{}); err != nil {
		t.Fatal(err)
	}
	if err := api.DeleteTags(ctx, "p-1"); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "p-1"); err != nil {
		t.Fatal(err)
	}
	base := "/keystone/v3/projects"
	want := []nativeProjectCall{
		{http.MethodPost, base, "", `{"project":{"description":"desc","domain_id":"default","enabled":false,"is_domain":false,"name":"proj","options":{"immutable":true},"owner":"ops","parent_id":"default","tags":["a"],"x_extension":1}}`},
		{http.MethodGet, base + "/p-1", "", ""},
		{http.MethodPatch, base + "/p-1", "", `{"project":{"description":"","enabled":true,"name":"new","options":{"immutable":false},"tags":[],"x_extension":2}}`},
		// The q:"-" Filters map also leaks as a stray "-" parameter before the real filter.
		{http.MethodGet, base, "-=%7B%27name__startswith%27%3A%27pr%27%7D&domain_id=default&enabled=false&is_domain=true&limit=5&name=proj&name__startswith=pr&not-tags=d&not-tags-any=e&parent_id=root&tags=a%2Cb&tags-any=c&x=1", ""},
		{http.MethodGet, "/other/projects", "page=2", ""},
		{http.MethodGet, base + "/p-1/tags", "", ""},
		{http.MethodPut, base + "/p-1/tags", "", `{"tags":["a","b"],"x_extension":3}`},
		{http.MethodPut, base + "/p-1/tags", "", `{}`},
		{http.MethodDelete, base + "/p-1/tags", "", ""},
		{http.MethodDelete, base + "/p-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeProjectAdminStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*projects.API) error
	}{
		// Create keeps the native POST default.
		{"Create", []int{201, 202}, func(api *projects.API) error { _, err := api.Create(ctx, projects.CreateOpts{Name: "p"}); return err }},
		{"Get", []int{200}, func(api *projects.API) error { _, err := api.Get(ctx, "p-1"); return err }},
		{"Update", []int{200}, func(api *projects.API) error {
			_, err := api.Update(ctx, "p-1", projects.UpdateOpts{Name: "n"})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *projects.API) error { return api.Delete(ctx, "p-1") }},
		{"ListTags", []int{200}, func(api *projects.API) error { _, err := api.ListTags(ctx, "p-1"); return err }},
		{"ModifyTags", []int{200}, func(api *projects.API) error {
			_, err := api.ModifyTags(ctx, "p-1", projects.ModifyTagsOpts{Tags: []string{"a"}})
			return err
		}},
		{"DeleteTags", []int{204}, func(api *projects.API) error { return api.DeleteTags(ctx, "p-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeProjectCall
				api, _ := nativeProjectAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
				err := call.call(api)
				nativeProjectOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("tag results on error", func(t *testing.T) {
		var calls []nativeProjectCall
		api, _ := nativeProjectAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{}` })
		// The native tag extractors return a non-nil zero value together with the error.
		tags, err := api.ListTags(ctx, "p-1")
		if err == nil || tags == nil || tags.Tags != nil {
			t.Fatal(tags, err)
		}
		modified, err := api.ModifyTags(ctx, "p-1", projects.ModifyTagsOpts{Tags: []string{"a"}})
		if err == nil || modified == nil || modified.Projects != nil {
			t.Fatal(modified, err)
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"projects":[],"links":{"next":null}}`}, {204, ""}} {
			var calls []nativeProjectCall
			api, _ := nativeProjectAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, tc.body })
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
		var calls []nativeProjectCall
		api, _ := nativeProjectAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{}` })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create name": {"Create", func() error { _, err := api.Create(ctx, projects.CreateOpts{DomainID: "d"}); return err }()},
			"create core extension": {"Create", func() error {
				_, err := api.Create(ctx, projects.CreateOpts{Name: "p"}, projects.WithCreateField("tags", []string{"x"}))
				return err
			}()},
			"create extra collision": {"Create", func() error {
				_, err := api.Create(ctx, projects.CreateOpts{Name: "p", Extra: map[string]any{"owner": "a"}}, projects.WithCreateField("owner", "b"))
				return err
			}()},
			"update nil option": {"Update", func() error {
				_, err := api.Update(ctx, "p-1", projects.UpdateOpts{}, nil)
				return err
			}()},
			"modify tags core extension": {"ModifyTags", func() error {
				_, err := api.ModifyTags(ctx, "p-1", projects.ModifyTagsOpts{}, projects.WithModifyTagsField("tags", []string{"x"}))
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeProjectOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeProjectOperation(t, err, "List")
		}
		var filterErrs []error
		for _, err := range api.List(ctx, projects.WithListOptions(projects.ListOpts{Filters: map[string]string{"__x": "1"}})) {
			filterErrs = append(filterErrs, err)
		}
		var invalid projects.InvalidListFilter
		if len(filterErrs) != 1 || !errors.As(filterErrs[0], &invalid) || invalid.FilterName != "__x" {
			t.Fatal(filterErrs)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
