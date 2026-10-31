package projectendpoints_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pyProjectEndpointRecordCall struct{ method, path, query, accept string }

func pyProjectEndpointRecordAPI(t *testing.T, calls *[]pyProjectEndpointRecordCall, reply func(*http.Request) (int, string)) *projectendpoints.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, pyProjectEndpointRecordCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, req.Header.Get("Accept")})
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return projectendpoints.New(client)
}

const pyProjectEndpointRecordPath = "/keystone/v3/OS-EP-FILTER/projects/p1/endpoints"
const pyProjectEndpointRecordRows = `{"endpoints":[` +
	`{"id":"e1","interface":"public","enabled":true,"region_id":"RegionOne","region":"RegionOne","service_id":"s1","url":"https://a","links":{"self":"x"}},` +
	`{"id":"e2","interface":"public","enabled":false,"region_id":"RegionOne","service_id":"s1","url":"https://b"}],` +
	`"links":{"self":"x","next":null,"previous":null}}`

func TestPythonProjectEndpointRecordListQueryFiltersAndModel(t *testing.T) {
	var calls []pyProjectEndpointRecordCall
	api := pyProjectEndpointRecordAPI(t, &calls, func(*http.Request) (int, string) { return 200, pyProjectEndpointRecordRows })
	values, err := api.AllRecords(context.Background(), "p1")
	if err != nil || len(values) != 2 || len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != pyProjectEndpointRecordPath || calls[0].query != "" || calls[0].accept != "application/json" {
		t.Fatal(err, values, calls)
	}
	v := values[0]
	if v.ID != "e1" || v.ProjectID != "p1" || *v.Interface != "public" || !*v.Enabled || *v.RegionID != "RegionOne" || *v.ServiceID != "s1" || *v.URL != "https://a" || v.Links["self"] != "x" || v.Name != nil {
		t.Fatalf("decoded project endpoint: %#v", v)
	}
	if len(v.Body) != 8 || string(v.Body["name"]) != "null" || string(v.Wire["region"]) != `"RegionOne"` || v.StatusCode != 200 {
		t.Fatalf("raw views: %#v", v)
	}

	calls = nil
	values, err = api.AllRecords(context.Background(), "p1",
		projectendpoints.WithRecordListQuery("interface", "public"),
		projectendpoints.WithRecordListQuery("region_id", "RegionOne"),
		projectendpoints.WithRecordListQuery("service_id", "s1"),
		projectendpoints.WithRecordListQuery("is_enabled", false),
		projectendpoints.WithRecordListQuery("project_id", "other"))
	if err != nil || len(values) != 1 || values[0].ID != "e2" || calls[0].query != "interface=public&region_id=RegionOne&service_id=s1" {
		t.Fatal(err, values, calls)
	}

	calls = nil
	values, err = api.AllRecords(context.Background(), "a/b", projectendpoints.WithRecordListQuery("url", "https://a"))
	if err != nil || len(values) != 1 || values[0].ID != "e1" || values[0].ProjectID != "a/b" || calls[0].path != "/keystone/v3/OS-EP-FILTER/projects/a%2Fb/endpoints" {
		t.Fatal(err, values, calls)
	}

	calls = nil
	if _, err = api.AllRecords(context.Background(), ""); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonProjectEndpointRecordListPagingAndStatuses(t *testing.T) {
	var calls []pyProjectEndpointRecordCall
	pages := []string{`{"endpoints":[{"id":"e1"},{"id":"e2"}]}`, `{"endpoints":[]}`}
	api := pyProjectEndpointRecordAPI(t, &calls, func(*http.Request) (int, string) {
		page := pages[0]
		pages = pages[1:]
		return 200, page
	})
	values, err := api.AllRecords(context.Background(), "p1", projectendpoints.WithRecordListLimit(2))
	if err != nil || len(values) != 2 || len(calls) != 2 || calls[0].query != "limit=2" || calls[1].query != "limit=2&marker=e2" {
		t.Fatal(err, values, calls)
	}

	calls = nil
	pages = []string{`{"endpoints":[{"id":"e1"},{"id":"e2"}]}`}
	values, err = api.AllRecords(context.Background(), "p1", projectendpoints.WithRecordListMaxItems(1))
	if err != nil || len(values) != 1 || len(calls) != 1 || calls[0].query != "limit=1" {
		t.Fatal(err, values, calls)
	}

	for _, code := range []int{400, 403, 404, 500} {
		var calls []pyProjectEndpointRecordCall
		api := pyProjectEndpointRecordAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
		_, err := api.AllRecords(context.Background(), "p1")
		var wrapped *resource.OperationError
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &wrapped) || wrapped.Operation != "List" || wrapped.Resource != "projectendpoints" || !errors.As(err, &native) || native.Actual != code {
			t.Fatal(code, err)
		}
	}
}
