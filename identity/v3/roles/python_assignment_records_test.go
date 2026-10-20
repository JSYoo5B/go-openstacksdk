package roles_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pyAssignmentRecordTransport func(*http.Request) (*http.Response, error)

func (transport pyAssignmentRecordTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pyAssignmentRecordCall struct{ method, path, query, accept string }

func pyAssignmentRecordAPI(t *testing.T, calls *[]pyAssignmentRecordCall, reply func(*http.Request) (int, string)) *roles.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pyAssignmentRecordTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, pyAssignmentRecordCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, req.Header.Get("Accept")})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return roles.New(client)
}

const pyAssignmentRecordPath = "/keystone/v3/role_assignments"
const pyAssignmentRecordInherited = `{"role":{"id":"r1","name":"member"},"scope":{"project":{"id":"p1"},"OS-INHERIT:inherited_to":"projects"},"user":{"id":"u1"},"links":{"assignment":"http://k/a1"}}`
const pyAssignmentRecordDirect = `{"role":{"id":"r1"},"scope":{"project":{"id":"p1"}},"group":{"id":"g1"},"links":{"assignment":"http://k/a2","membership":"http://k/m"}}`

func TestPythonRoleAssignmentRecordsKeepScopeLinksAndQuery(t *testing.T) {
	var calls []pyAssignmentRecordCall
	api := pyAssignmentRecordAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"role_assignments":[` + pyAssignmentRecordInherited + `,` + pyAssignmentRecordDirect + `],"links":{"self":"x","previous":null,"next":null}}`
	})
	values, err := api.AllAssignmentRecords(context.Background())
	if err != nil || len(values) != 2 || len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != pyAssignmentRecordPath || calls[0].query != "" || calls[0].accept != "application/json" {
		t.Fatal(err, values, calls)
	}
	v := values[0]
	if string(v.Scope) != `{"project":{"id":"p1"},"OS-INHERIT:inherited_to":"projects"}` || string(v.Role) != `{"id":"r1","name":"member"}` || string(v.User) != `{"id":"u1"}` || v.Group != nil || string(v.Links) != `{"assignment":"http://k/a1"}` {
		t.Fatalf("decoded assignment: %#v", v)
	}
	if len(v.Body) != 7 || string(v.Body["id"]) != "null" || string(v.Body["group"]) != "null" || v.StatusCode != 200 {
		t.Fatalf("raw views: %#v", v)
	}

	calls = nil
	_, err = api.AllAssignmentRecords(context.Background(),
		roles.WithAssignmentRecordListQuery("role_id", "r1"),
		roles.WithAssignmentRecordListQuery("user.id", "ignored-server-name"),
		roles.WithAssignmentRecordListQuery("user_id", "u1"),
		roles.WithAssignmentRecordListQuery("scope_project_id", "p1"),
		roles.WithAssignmentRecordListQuery("inherited_to", "projects"),
		roles.WithAssignmentRecordListQuery("effective", true),
		roles.WithAssignmentRecordListQuery("include_names", false),
		roles.WithAssignmentRecordListQuery("unknown", "x"))
	want := "effective=True&include_names=False&role.id=r1&scope.OS-INHERIT%3Ainherited_to=projects&scope.project.id=p1&user.id=u1"
	if err != nil || calls[0].query != want {
		t.Fatal(err, calls)
	}
}

func TestPythonRoleAssignmentRecordsNestedLocalFiltersAndPaging(t *testing.T) {
	var calls []pyAssignmentRecordCall
	api := pyAssignmentRecordAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"role_assignments":[` + pyAssignmentRecordInherited + `,` + pyAssignmentRecordDirect + `]}`
	})
	values, err := api.AllAssignmentRecords(context.Background(), roles.WithAssignmentRecordListQuery("scope", map[string]any{"OS-INHERIT:inherited_to": "projects"}))
	if err != nil || len(values) != 1 || string(values[0].User) != `{"id":"u1"}` || calls[0].query != "" {
		t.Fatal(err, values, calls)
	}
	values, err = api.AllAssignmentRecords(context.Background(), roles.WithAssignmentRecordListQuery("group", map[string]any{"id": "g1"}), roles.WithAssignmentRecordListQuery("role", map[string]any{"id": "r1"}))
	if err != nil || len(values) != 1 || string(values[0].Group) != `{"id":"g1"}` {
		t.Fatal(err, values)
	}
	values, err = api.AllAssignmentRecords(context.Background(), roles.WithAssignmentRecordListQuery("user", map[string]any{"id": "u2"}))
	if err != nil || len(values) != 0 {
		t.Fatal(err, values)
	}

	// Keystone assignments carry no id. Python sets marker None for the
	// limit follow-up and aborts as an endless loop; Go fails before it.
	calls = nil
	values = nil
	failed := false
	for value, err := range api.ListAssignmentRecords(context.Background(), roles.WithAssignmentRecordListLimit(5)) {
		if err != nil {
			var wrapped *resource.OperationError
			if !errors.As(err, &wrapped) || wrapped.Operation != "List" || wrapped.Resource != "role_assignments" || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			failed = true
			break
		}
		values = append(values, value)
	}
	if !failed || len(values) != 2 || len(calls) != 1 || calls[0].query != "limit=5" {
		t.Fatal(failed, values, calls)
	}

	for _, code := range []int{400, 403, 500} {
		var calls []pyAssignmentRecordCall
		api := pyAssignmentRecordAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
		_, err := api.AllAssignmentRecords(context.Background())
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != code {
			t.Fatal(code, err)
		}
	}
}
