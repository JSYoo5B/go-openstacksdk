package labels_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type labelCall struct{ method, path, query, body, accept string }

const labelBase = "/neutron/v2.0/metering/metering-labels"

func labelAPI(t *testing.T, calls *[]labelCall, reply func(*http.Request) (int, string)) *labels.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, labelCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Get("Accept")})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return labels.New(client)
}

func labelFailure(t *testing.T, err error, operation string, code int) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "meteringlabels" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if code != 0 && (!errors.As(err, &native) || native.Actual != code) {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

const labelRow = `{"id":"ml1","name":"web","description":"http","project_id":"p1","tenant_id":"p1","shared":true}`

func labelCheck(t *testing.T, v *labels.MeteringLabel) {
	t.Helper()
	if v == nil || v.ID != "ml1" || *v.Name != "web" || *v.Description != "http" || *v.ProjectID != "p1" || *v.TenantID != "p1" || !*v.IsShared {
		t.Fatalf("decoded metering label: %#v", v)
	}
	if string(v.Resource.Body["is_shared"]) != "true" || string(v.Wire.Body["shared"]) != "true" {
		t.Fatalf("raw views: %#v %#v", v.Resource, v.Wire)
	}
}

func TestPythonMeteringLabelCreatePostsAttrsWithWireNames(t *testing.T) {
	var calls []labelCall
	api := labelAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{"metering_label":` + labelRow + `}` })
	v, err := api.Create(context.Background(),
		labels.WithCreateField("name", "web"),
		labels.WithCreateField("is_shared", true),
		labels.WithCreateField("tenant_id", "p1"))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"metering_label":{"name":"web","shared":true,"tenant_id":"p1"}}`
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != labelBase || calls[0].body != want {
		t.Fatal(calls)
	}
	labelCheck(t, v)
	for _, code := range []int{400, 409} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []labelCall
			api := labelAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
			_, err := api.Create(context.Background())
			labelFailure(t, err, "Create", code)
		})
	}
}

func TestPythonMeteringLabelGetMapsFieldsAndReportsMissing(t *testing.T) {
	var calls []labelCall
	api := labelAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"metering_label":` + labelRow + `}` })
	v, err := api.Get(context.Background(), "ml1", labels.WithGetFields("is_shared", "name"))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != labelBase+"/ml1" || calls[0].query != "fields=shared&fields=name" {
		t.Fatal(calls)
	}
	labelCheck(t, v)
	api = labelAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{}` })
	_, err = api.Get(context.Background(), "gone")
	labelFailure(t, err, "Get", 404)
}

func TestPythonMeteringLabelUpdatePutsOnlyGivenAttrs(t *testing.T) {
	var calls []labelCall
	api := labelAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"metering_label":` + labelRow + `}` })
	v, err := api.Update(context.Background(), "ml1", labels.WithUpdateField("description", "http"))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != labelBase+"/ml1" || calls[0].body != `{"metering_label":{"description":"http"}}` {
		t.Fatal(calls)
	}
	labelCheck(t, v)
	calls = nil
	if v, err = api.Update(context.Background(), "ml1"); err != nil || len(calls) != 0 || v.ID != "ml1" {
		t.Fatal(v, err, calls)
	}
	if _, err = api.Update(context.Background(), "ml1", labels.WithUpdateField("metering_label", "x")); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonMeteringLabelDeleteIgnoresMissingByDefault(t *testing.T) {
	for _, tc := range []struct {
		code    int
		strict  bool
		ignored bool
		fails   bool
	}{{204, false, false, false}, {404, false, true, false}, {404, true, false, true}, {409, false, false, true}} {
		t.Run(fmt.Sprint(tc.code, tc.strict), func(t *testing.T) {
			var calls []labelCall
			api := labelAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, "" })
			var options []labels.DeleteOption
			if tc.strict {
				options = append(options, labels.WithDeleteIgnoreMissing(false))
			}
			result, err := api.Delete(context.Background(), "ml1", options...)
			if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != labelBase+"/ml1" {
				t.Fatal(calls)
			}
			if tc.fails {
				labelFailure(t, err, "Delete", tc.code)
				return
			}
			if err != nil || result.StatusCode != tc.code || result.IgnoredMissing != tc.ignored {
				t.Fatal(result, err)
			}
		})
	}
}

func TestPythonMeteringLabelListMapsQueryAndFiltersBodyLocally(t *testing.T) {
	var calls []labelCall
	api := labelAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Query().Get("marker") == "b" {
			return 200, `{"metering_labels":[{"id":"c","tenant_id":"p1"}]}`
		}
		return 200, `{"metering_labels":[{"id":"a","tenant_id":"p2"},{"id":"b","project_id":"p1","tenant_id":"p1"}],"metering_labels_links":[{"rel":"next","href":"?marker=b"}]}`
	})
	values, err := api.All(context.Background(),
		labels.WithListQuery("is_shared", true),
		labels.WithListQuery("project_id", "p1"),
		labels.WithListQuery("sort_key", "name"),
		labels.WithListQuery("tenant_id", "p1"),
		labels.WithListQuery("driver", "ignored"))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0].path != labelBase || calls[0].query != "project_id=p1&shared=True&sort_key=name" || calls[0].accept != "application/json" || calls[1].query != "marker=b&project_id=p1&shared=True&sort_key=name" {
		t.Fatal(calls)
	}
	if len(values) != 2 || values[0].ID != "b" || values[1].ID != "c" || *values[1].ProjectID != "p1" {
		t.Fatalf("%#v", values)
	}
}

func TestPythonMeteringLabelFindFallsBackToNameList(t *testing.T) {
	var calls []labelCall
	api := labelAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path != labelBase {
			return 404, ""
		}
		return 200, `{"metering_labels":[` + labelRow + `]}`
	})
	v, err := api.Find(context.Background(), "web")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0].path != labelBase+"/web" || calls[1].path != labelBase || calls[1].query != "name=web" {
		t.Fatal(calls)
	}
	labelCheck(t, v)
	for _, strict := range []bool{false, true} {
		calls = nil
		api = labelAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Path != labelBase {
				return 404, ""
			}
			return 200, `{"metering_labels":[]}`
		})
		v, err = api.Find(context.Background(), "none", labels.WithFindIgnoreMissing(!strict))
		if v != nil || len(calls) != 2 || strict != errors.Is(err, resource.ErrNotFound) {
			t.Fatal(strict, v, err, calls)
		}
	}
}
