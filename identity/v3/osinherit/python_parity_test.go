package osinherit_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonInheritTransport func(*http.Request) (*http.Response, error)

func (transport pythonInheritTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonInheritCall struct{ method, path, query, body string }

func pythonInheritAPI(t *testing.T, calls *[]pythonInheritCall, status int) *osinherit.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonInheritTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonInheritCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return osinherit.New(client)
}

// Python assign_*/unassign_*/validate_* on a domain or project with
// inherited=True join OS-INHERIT and inherited_to_projects around the direct
// route; assign/unassign discard the status and validate is True only for 204.
func TestPythonInheritedRoleAssignUnassignValidate(t *testing.T) {
	ctx := context.Background()
	for _, target := range []struct {
		name                                 string
		userID, groupID, projectID, domainID string
		path                                 string
	}{
		{name: "domain user", domainID: "dom-1", userID: "u-1", path: "/keystone/v3/OS-INHERIT/domains/dom-1/users/u-1/roles/r-1/inherited_to_projects"},
		{name: "domain group", domainID: "dom-1", groupID: "g-1", path: "/keystone/v3/OS-INHERIT/domains/dom-1/groups/g-1/roles/r-1/inherited_to_projects"},
		{name: "project user", projectID: "p-1", userID: "u-1", path: "/keystone/v3/OS-INHERIT/projects/p-1/users/u-1/roles/r-1/inherited_to_projects"},
		{name: "project group", projectID: "p-1", groupID: "g-1", path: "/keystone/v3/OS-INHERIT/projects/p-1/groups/g-1/roles/r-1/inherited_to_projects"},
	} {
		ops := []struct {
			method string
			call   func(*osinherit.API) error
		}{
			{http.MethodPut, func(api *osinherit.API) error {
				return api.Assign(ctx, "r-1", osinherit.AssignOpts{UserID: target.userID, GroupID: target.groupID, ProjectID: target.projectID, DomainID: target.domainID})
			}},
			{http.MethodDelete, func(api *osinherit.API) error {
				return api.Unassign(ctx, "r-1", osinherit.UnassignOpts{UserID: target.userID, GroupID: target.groupID, ProjectID: target.projectID, DomainID: target.domainID})
			}},
			{http.MethodHead, func(api *osinherit.API) error {
				return api.Validate(ctx, "r-1", osinherit.ValidateOpts{UserID: target.userID, GroupID: target.groupID, ProjectID: target.projectID, DomainID: target.domainID})
			}},
		}
		for _, op := range ops {
			t.Run(target.name+" "+op.method, func(t *testing.T) {
				for _, status := range []int{204, 404, 200} {
					var calls []pythonInheritCall
					err := op.call(pythonInheritAPI(t, &calls, status))
					if (status == 204) != (err == nil) || status != 204 && !gophercloud.ResponseCodeIs(err, status) {
						t.Fatal(status, err)
					}
					if !reflect.DeepEqual(calls, []pythonInheritCall{{op.method, target.path, "", ""}}) {
						t.Fatalf("%+v", calls)
					}
				}
			})
		}
	}
}
