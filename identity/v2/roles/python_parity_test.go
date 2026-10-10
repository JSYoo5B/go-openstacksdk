package roles_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v2/roles"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

type pythonRoleTransport func(*http.Request) (*http.Response, error)

func (transport pythonRoleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonRoleCall struct{ method, path, query string }

// Python roles() with no query sends the same GET /OS-KSADM/roles, but it would
// also follow roles_links. The native pager stops after one page and has no
// query, get, create, update or delete call for OS-KSADM/roles.
func TestPythonRoleListDefaultRequest(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []pythonRoleCall
	cloud.Provider.HTTPClient.Transport = pythonRoleTransport(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, pythonRoleCall{req.Method, req.URL.Path, req.URL.RawQuery})
		body := `{"roles":[{"id":"r-1","name":"admin","description":"d"}],"roles_links":[{"rel":"next","href":"` + cloud.Server.URL + `/keystone/v2.0/OS-KSADM/roles?marker=r-1"}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v2.0/"
	var rows []roles.Role
	for value, err := range roles.New(client).List(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, *value)
	}
	if !reflect.DeepEqual(rows, []roles.Role{{ID: "r-1", Name: "admin", Description: "d"}}) || !reflect.DeepEqual(calls, []pythonRoleCall{{http.MethodGet, "/keystone/v2.0/OS-KSADM/roles", ""}}) {
		t.Fatalf("%+v %+v", rows, calls)
	}
}
