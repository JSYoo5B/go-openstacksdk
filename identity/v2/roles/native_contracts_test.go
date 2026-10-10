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

	"github.com/JSYoo5B/go-openstacksdk/identity/v2/roles"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeV2RoleTransport func(*http.Request) (*http.Response, error)

func (transport nativeV2RoleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeV2RoleCall struct{ method, path, query, body string }

func nativeV2RoleAPI(t *testing.T, calls *[]nativeV2RoleCall, reply func(*http.Request) (int, string)) *roles.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeV2RoleTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeV2RoleCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v2.0/"
	return roles.New(client)
}

func nativeV2RoleOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "roles" {
		t.Fatal("generated roles context", err, wrapped)
	}
}

func TestNativeV2RoleRoutesAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeV2RoleCall
	api := nativeV2RoleAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.Method {
		case http.MethodPut:
			return 201, ""
		case http.MethodDelete:
			return 204, ""
		}
		// Role has no JSON tags, so keys match field names without case; roles_links is never followed.
		return 200, `{"roles":[{"id":"r1","name":"admin","description":"d","serviceId":"svc"},{"ID":"r2","service_id":"ignored"}],"roles_links":[{"rel":"next","href":"/never"}]}`
	})
	var rows []roles.Role
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, *value)
	}
	if err := api.AddUser(ctx, "t-1", "u-1", "r-1"); err != nil {
		t.Fatal(err)
	}
	if err := api.DeleteUser(ctx, "t-1", "u-1", "r-1"); err != nil {
		t.Fatal(err)
	}
	wantRows := []roles.Role{{ID: "r1", Name: "admin", Description: "d", ServiceID: "svc"}, {ID: "r2"}}
	member := "/keystone/v2.0/tenants/t-1/users/u-1/roles/OS-KSADM/r-1"
	want := []nativeV2RoleCall{
		{http.MethodGet, "/keystone/v2.0/OS-KSADM/roles", "", ""},
		// AddUser sends PUT without a request body.
		{http.MethodPut, member, "", ""},
		{http.MethodDelete, member, "", ""},
	}
	if !reflect.DeepEqual(rows, wantRows) || !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v %+v", rows, calls)
	}
	if roles.ExtPath != "OS-KSADM" || roles.RolePath != "roles" || roles.UserPath != "users" {
		t.Fatal(roles.ExtPath, roles.RolePath, roles.UserPath)
	}
}

func TestNativeV2RoleStatusesAndListPager(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*roles.API) error
	}{
		{"AddUser", []int{200, 201}, func(api *roles.API) error { return api.AddUser(ctx, "t", "u", "r") }},
		{"DeleteUser", []int{202, 204}, func(api *roles.API) error { return api.DeleteUser(ctx, "t", "u", "r") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeV2RoleCall
				api := nativeV2RoleAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
				err := call.call(api)
				if len(calls) != 1 {
					t.Fatal(calls)
				}
				if slices.Contains(call.accepted, code) {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				nativeV2RoleOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"roles":[]}`}, {200, `{}`}, {204, ""}} {
			var calls []nativeV2RoleCall
			api := nativeV2RoleAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, tc.body })
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
}
