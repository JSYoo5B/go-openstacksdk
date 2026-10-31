package regions_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/regions"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonRegionCall struct{ method, path, query, body string }

func pythonRegionAPI(t *testing.T, calls *[]pythonRegionCall, reply func(*http.Request) (int, string)) (*regions.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonRegionCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return regions.New(client), cloud
}

const (
	pythonRegionBase = "/keystone/v3/regions"
	pythonRegionRow  = `{"id":"RegionOne","description":"main","parent_region_id":"Root","links":{"self":"x"}}`
)

// Python create_region sends the dirty id/description/parent_region_id in the
// "region" envelope; update_region sends only the changed attributes.
func TestPythonRegionCreateGetUpdateRequests(t *testing.T) {
	ctx := context.Background()
	var calls []pythonRegionCall
	api, _ := pythonRegionAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.Method {
		case http.MethodPost:
			return 201, `{"region":` + pythonRegionRow + `}`
		case http.MethodPatch:
			return 200, `{"region":{"id":"RegionOne","description":"","parent_region_id":"Root"}}`
		}
		return 200, `{"region":` + pythonRegionRow + `}`
	})
	check := func(got *regions.Region, description string) bool {
		return got != nil && got.ID == "RegionOne" && got.Description == description && got.ParentRegionID == "Root" && reflect.DeepEqual(got.Links, map[string]any{"self": "x"})
	}
	created, err := api.Create(ctx, regions.CreateOpts{ID: "RegionOne", Description: "main", ParentRegionID: "Root"})
	if err != nil || !check(created, "main") {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "RegionOne")
	if err != nil || !check(got, "main") {
		t.Fatal(got, err)
	}
	empty := ""
	updated, err := api.Update(ctx, "RegionOne", regions.UpdateOpts{Description: &empty})
	if err != nil || updated.ID != "RegionOne" || updated.Description != "" {
		t.Fatal(updated, err)
	}
	wantCalls := []pythonRegionCall{
		{http.MethodPost, pythonRegionBase, "", `{"region":{"description":"main","id":"RegionOne","parent_region_id":"Root"}}`},
		{http.MethodGet, pythonRegionBase + "/RegionOne", "", ""},
		{http.MethodPatch, pythonRegionBase + "/RegionOne", "", `{"region":{"description":""}}`},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("%+v", calls)
	}
	// Python update_region(parent_region_id=None) sends null; the typed field
	// omits it and the extension key is rejected before HTTP.
	calls = nil
	if _, err := api.Update(ctx, "RegionOne", regions.UpdateOpts{}, regions.WithUpdateField("parent_region_id", nil)); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonRegionDeleteIgnoreMissing(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		status  int
		options []resource.LookupOption
		check   func(error) bool
	}{
		{"deleted", 204, nil, func(err error) bool { return err == nil }},
		{"missing ignored by default", 404, nil, func(err error) bool { return err == nil }},
		{"ignore_missing=False", 404, []resource.LookupOption{resource.WithMissingError()}, func(err error) bool { return errors.Is(err, resource.ErrNotFound) }},
		{"conflict is returned", 409, nil, func(err error) bool { return gophercloud.ResponseCodeIs(err, http.StatusConflict) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonRegionCall
			api, _ := pythonRegionAPI(t, &calls, func(*http.Request) (int, string) { return tc.status, "" })
			if err := api.Remove(ctx, resource.ID("RegionOne"), tc.options...); !tc.check(err) {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []pythonRegionCall{{http.MethodDelete, pythonRegionBase + "/RegionOne", "", ""}}) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

// Python find_region falls back to GET regions after a missing ID; the Go
// collection has no region name selector, so only the ID GET is available.
func TestPythonRegionFindByIDOnly(t *testing.T) {
	ctx := context.Background()
	var calls []pythonRegionCall
	api, _ := pythonRegionAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == pythonRegionBase+"/RegionOne" {
			return 200, `{"region":` + pythonRegionRow + `}`
		}
		return 404, `{"error":{"code":404}}`
	})
	if got, err := api.Find(ctx, resource.ID("RegionOne"), resource.WithIgnoreMissing()); err != nil || got == nil || got.ID != "RegionOne" {
		t.Fatal(got, err)
	}
	if got, err := api.Find(ctx, resource.ID("ghost"), resource.WithIgnoreMissing()); err != nil || got != nil {
		t.Fatal(got, err)
	}
	want := []pythonRegionCall{{http.MethodGet, pythonRegionBase + "/RegionOne", "", ""}, {http.MethodGet, pythonRegionBase + "/ghost", "", ""}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	calls = nil
	if _, err := api.Find(ctx, resource.Name("ghost"), resource.WithIgnoreMissing()); !errors.Is(err, resource.ErrUnsupported) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

// Python regions() sends parent_region_id/limit/marker and ignores the
// Keystone links.next dict; Go forwards the query and follows links.next.
func TestPythonRegionListQueryAndPaging(t *testing.T) {
	ctx := context.Background()
	var calls []pythonRegionCall
	var cloud *testcloud.Cloud
	api, cloud := pythonRegionAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("page") == "2" {
			return 200, `{"regions":[{"id":"RegionTwo"}],"links":{"next":null}}`
		}
		return 200, `{"regions":[` + pythonRegionRow + `],"links":{"next":"` + cloud.Server.URL + pythonRegionBase + `?page=2"}}`
	})
	rows, err := api.All(ctx, resource.WithQuery("parent_region_id", "Root"), resource.WithQuery("limit", "1"), resource.WithQuery("marker", "Root"))
	if err != nil || len(rows) != 2 || rows[0].ID != "RegionOne" || rows[1].ID != "RegionTwo" {
		t.Fatal(rows, err)
	}
	want := []pythonRegionCall{
		{http.MethodGet, pythonRegionBase, "limit=1&marker=Root&parent_region_id=Root", ""},
		{http.MethodGet, pythonRegionBase, "page=2", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestPythonRegionWaitHelpers(t *testing.T) {
	ctx := context.Background()
	t.Run("status attribute is required", func(t *testing.T) {
		var calls []pythonRegionCall
		api, _ := pythonRegionAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"region":` + pythonRegionRow + `}` })
		// Region has no status attribute; Python raises AttributeError.
		if _, err := api.WaitFor(ctx, resource.ID("RegionOne"), "active"); !errors.Is(err, resource.ErrUnsupported) || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	})
	t.Run("selected attribute", func(t *testing.T) {
		var calls []pythonRegionCall
		api, _ := pythonRegionAPI(t, &calls, func(*http.Request) (int, string) {
			if len(calls) == 1 {
				return 200, `{"region":{"id":"RegionOne","description":"pending"}}`
			}
			return 200, `{"region":` + pythonRegionRow + `}`
		})
		got, err := api.WaitFor(ctx, resource.ID("RegionOne"), "MAIN", resource.WithStatusAttribute("description"), resource.WithPollInterval(time.Millisecond))
		get := pythonRegionCall{http.MethodGet, pythonRegionBase + "/RegionOne", "", ""}
		if err != nil || got.Description != "main" || !reflect.DeepEqual(calls, []pythonRegionCall{get, get}) {
			t.Fatal(got, err, calls)
		}
	})
	t.Run("deletion polls until 404", func(t *testing.T) {
		var calls []pythonRegionCall
		api, _ := pythonRegionAPI(t, &calls, func(*http.Request) (int, string) {
			if len(calls) == 1 {
				return 200, `{"region":` + pythonRegionRow + `}`
			}
			return 404, ""
		})
		var progress []int
		err := api.WaitForDeletion(ctx, resource.ID("RegionOne"), resource.WithTimeout(120*time.Second), resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(p int) { progress = append(progress, p) }))
		get := pythonRegionCall{http.MethodGet, pythonRegionBase + "/RegionOne", "", ""}
		if err != nil || !reflect.DeepEqual(calls, []pythonRegionCall{get, get}) || !reflect.DeepEqual(progress, []int{0}) {
			t.Fatal(err, calls, progress)
		}
	})
}
