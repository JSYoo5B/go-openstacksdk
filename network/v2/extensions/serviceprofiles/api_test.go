package serviceprofiles_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type profileCall struct{ method, path, query, body, accept string }

const profileBase = "/neutron/v2.0/service_profiles"

func profileAPI(t *testing.T, calls *[]profileCall, reply func(*http.Request) (int, string)) *serviceprofiles.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, profileCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Get("Accept")})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return serviceprofiles.New(client)
}

func profileFailure(t *testing.T, err error, operation string, code int) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "serviceprofiles" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
	if code == 0 {
		return
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

// The row carries only tenant_id; project_id falls back to it.
const profileRow = `{"id":"sp1","description":"lbaas","driver":"neutron_lbaas.Driver","enabled":false,"metainfo":"{\"k\": 1}","tenant_id":"p1"}`

func profileCheck(t *testing.T, v *serviceprofiles.ServiceProfile) {
	t.Helper()
	if v == nil || v.ID != "sp1" || v.Name != nil || *v.Description != "lbaas" || *v.Driver != "neutron_lbaas.Driver" || *v.IsEnabled || *v.MetaInfo != `{"k": 1}` || *v.ProjectID != "p1" || *v.TenantID != "p1" {
		t.Fatalf("decoded service profile: %#v", v)
	}
	if string(v.Resource.Body["name"]) != "null" || string(v.Resource.Body["is_enabled"]) != "false" || v.Wire.Body["project_id"] != nil {
		t.Fatalf("raw views: %#v %#v", v.Resource, v.Wire)
	}
}

func TestPythonServiceProfileCreatePostsAttrsWithWireNames(t *testing.T) {
	var calls []profileCall
	api := profileAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{"service_profile":` + profileRow + `}` })
	v, err := api.Create(context.Background(),
		serviceprofiles.WithCreateField("driver", "neutron_lbaas.Driver"),
		serviceprofiles.WithCreateField("meta_info", `{"k": 1}`),
		serviceprofiles.WithCreateField("is_enabled", false),
		serviceprofiles.WithCreateField("project_id", "p1"))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"service_profile":{"driver":"neutron_lbaas.Driver","enabled":false,"metainfo":"{\"k\": 1}","project_id":"p1"}}`
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != profileBase || calls[0].body != want {
		t.Fatal(calls)
	}
	profileCheck(t, v)
	for _, code := range []int{400, 403, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []profileCall
			api := profileAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
			_, err := api.Create(context.Background(), serviceprofiles.WithCreateField("driver", "d"))
			profileFailure(t, err, "Create", code)
		})
	}
}

func TestPythonServiceProfileGetMapsFieldsAndReportsMissing(t *testing.T) {
	var calls []profileCall
	api := profileAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"service_profile":` + profileRow + `}` })
	v, err := api.Get(context.Background(), "sp1", serviceprofiles.WithGetFields("meta_info", "is_enabled", "project_id"))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != profileBase+"/sp1" || calls[0].query != "fields=metainfo&fields=enabled&fields=project_id" {
		t.Fatal(calls)
	}
	profileCheck(t, v)
	api = profileAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{}` })
	_, err = api.Get(context.Background(), "gone")
	profileFailure(t, err, "Get", 404)
}

func TestPythonServiceProfileUpdatePutsOnlyGivenAttrs(t *testing.T) {
	var calls []profileCall
	api := profileAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"service_profile":` + profileRow + `}` })
	v, err := api.Update(context.Background(), "sp1", serviceprofiles.WithUpdateField("is_enabled", false), serviceprofiles.WithUpdateField("description", "lbaas"))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != profileBase+"/sp1" || calls[0].body != `{"service_profile":{"description":"lbaas","enabled":false}}` {
		t.Fatal(calls)
	}
	profileCheck(t, v)
	calls = nil
	if v, err = api.Update(context.Background(), "sp1"); err != nil || len(calls) != 0 || v.ID != "sp1" {
		t.Fatal(v, err, calls)
	}
	if _, err = api.Update(context.Background(), "sp1", serviceprofiles.WithUpdateField("service_profile", "x")); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonServiceProfileDeleteIgnoresMissingByDefault(t *testing.T) {
	for _, tc := range []struct {
		code    int
		strict  bool
		ignored bool
		fails   bool
	}{{204, false, false, false}, {404, false, true, false}, {404, true, false, true}, {409, false, false, true}} {
		t.Run(fmt.Sprint(tc.code, tc.strict), func(t *testing.T) {
			var calls []profileCall
			api := profileAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, "" })
			var options []serviceprofiles.DeleteOption
			if tc.strict {
				options = append(options, serviceprofiles.WithDeleteIgnoreMissing(false))
			}
			result, err := api.Delete(context.Background(), "sp1", options...)
			if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != profileBase+"/sp1" {
				t.Fatal(calls)
			}
			if tc.fails {
				profileFailure(t, err, "Delete", tc.code)
				return
			}
			if err != nil || result.StatusCode != tc.code || result.IgnoredMissing != tc.ignored {
				t.Fatal(result, err)
			}
		})
	}
}

func TestPythonServiceProfileListMapsQueryAndFiltersBodyLocally(t *testing.T) {
	var calls []profileCall
	api := profileAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("marker") == "b" {
			return 200, `{"service_profiles":[{"id":"c","name":"lb","tenant_id":"p1"}]}`
		}
		return 200, `{"service_profiles":[{"id":"a","name":"lb","tenant_id":"p2"},{"id":"b","name":"other","tenant_id":"p1"}],"service_profiles_links":[{"rel":"next","href":"?marker=b"}]}`
	})
	values, err := api.All(context.Background(),
		serviceprofiles.WithListQuery("project_id", "p1"),
		serviceprofiles.WithListQuery("is_enabled", false),
		serviceprofiles.WithListQuery("driver", "d"),
		serviceprofiles.WithListQuery("name", "lb"),
		serviceprofiles.WithListQuery("tenant_id", "p1"),
		serviceprofiles.WithListQuery("sort_key", "id"))
	if err != nil {
		t.Fatal(err)
	}
	// sort_key is not declared for ServiceProfile, so Python drops it. The
	// next link's query is merged over the original query, as Python does.
	if len(calls) != 2 || calls[0].path != profileBase || calls[0].query != "driver=d&enabled=False&project_id=p1" || calls[0].accept != "application/json" || calls[1].query != "driver=d&enabled=False&marker=b&project_id=p1" {
		t.Fatal(calls)
	}
	if len(values) != 1 || values[0].ID != "c" {
		t.Fatalf("%#v", values)
	}
}

func TestPythonServiceProfileFindListsWithoutNameHint(t *testing.T) {
	var calls []profileCall
	api := profileAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path != profileBase {
			return 404, ""
		}
		return 200, `{"service_profiles":[{"id":"x","name":"other"},{"id":"sp1","name":"lb"}]}`
	})
	v, err := api.Find(context.Background(), "lb")
	if err != nil || v == nil || v.ID != "sp1" {
		t.Fatal(v, err)
	}
	if len(calls) != 2 || calls[0].path != profileBase+"/lb" || calls[1].path != profileBase || calls[1].query != "" {
		t.Fatal(calls)
	}
	for _, strict := range []bool{false, true} {
		calls = nil
		api = profileAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Path != profileBase {
				return 403, ""
			}
			return 200, `{"service_profiles":[]}`
		})
		v, err = api.Find(context.Background(), "none", serviceprofiles.WithFindIgnoreMissing(!strict))
		if v != nil || len(calls) != 2 || strict != errors.Is(err, resource.ErrNotFound) {
			t.Fatal(strict, v, err, calls)
		}
	}
}
