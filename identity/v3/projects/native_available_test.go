package projects_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeSelfTransport func(*http.Request) (*http.Response, error)

func (transport nativeSelfTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeSelfCall struct{ method, path, query, body string }

func nativeSelfClient(t *testing.T, calls *[]nativeSelfCall, reply func(*http.Request) (int, string)) (*gophercloud.ServiceClient, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeSelfTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeSelfCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return client, cloud
}

func TestNativeProjectListAvailable(t *testing.T) {
	ctx := context.Background()
	var calls []nativeSelfCall
	var cloud *testcloud.Cloud
	client, cloud := nativeSelfClient(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == "/other/projects" {
			return 200, `{"projects":[{"id":"p2","name":"b","enabled":false}],"links":{"next":null}}`
		}
		return 200, `{"projects":[{"id":"p1","name":"a","domain_id":"d","enabled":true}],"links":{"next":"` + cloud.Server.URL + `/other/projects?page=2"}}`
	})
	var names []string
	for value, err := range projects.New(client).ListAvailable(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, value.ID+"="+value.Name)
	}
	want := []nativeSelfCall{{http.MethodGet, "/keystone/v3/auth/projects", "", ""}, {http.MethodGet, "/other/projects", "page=2", ""}}
	if !reflect.DeepEqual(names, []string{"p1=a", "p2=b"}) || !reflect.DeepEqual(calls, want) {
		t.Fatal(names, calls)
	}
	for _, tc := range []struct {
		code int
		body string
	}{{404, `{}`}, {200, `{"projects":[],"links":{"next":null}}`}} {
		var calls []nativeSelfCall
		client, _ := nativeSelfClient(t, &calls, func(*http.Request) (int, string) { return tc.code, tc.body })
		var errs []error
		for _, err := range projects.New(client).ListAvailable(ctx) {
			errs = append(errs, err)
		}
		if (tc.code == 404) != (len(errs) == 1) {
			t.Fatal(tc.code, errs)
		}
	}
}

func TestNativeUserChangePassword(t *testing.T) {
	ctx := context.Background()
	var calls []nativeSelfCall
	client, _ := nativeSelfClient(t, &calls, func(*http.Request) (int, string) { return 204, "" })
	api := users.New(client)
	if err := api.ChangePassword(ctx, "u1", users.ChangePasswordOpts{OriginalPassword: "old", Password: "new"}, users.WithChangePasswordField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	// Neither password is required or omitted when empty.
	if err := api.ChangePassword(ctx, "u1", users.ChangePasswordOpts{}); err != nil {
		t.Fatal(err)
	}
	want := []nativeSelfCall{
		{http.MethodPost, "/keystone/v3/users/u1/password", "", `{"user":{"original_password":"old","password":"new","x_extension":1}}`},
		{http.MethodPost, "/keystone/v3/users/u1/password", "", `{"user":{"original_password":"","password":""}}`},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	for _, code := range []int{200, 201, 202, 401} {
		var calls []nativeSelfCall
		client, _ := nativeSelfClient(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
		err := users.New(client).ChangePassword(ctx, "u1", users.ChangePasswordOpts{})
		var wrapped *resource.OperationError
		var native gophercloud.ErrUnexpectedResponseCode
		// Only 204 is accepted.
		if !errors.As(err, &wrapped) || wrapped.Operation != "ChangePassword" || !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{204}) {
			t.Fatal(code, err)
		}
	}
	var none []nativeSelfCall
	client, _ = nativeSelfClient(t, &none, func(*http.Request) (int, string) { return 204, "" })
	for name, err := range map[string]error{
		"core extension": users.New(client).ChangePassword(ctx, "u1", users.ChangePasswordOpts{}, users.WithChangePasswordField("password", "x")),
		"nil option":     users.New(client).ChangePassword(ctx, "u1", users.ChangePasswordOpts{}, nil),
	} {
		var wrapped *resource.OperationError
		if !errors.As(err, &wrapped) || wrapped.Operation != "ChangePassword" {
			t.Fatal(name, err)
		}
	}
	if len(none) != 0 {
		t.Fatal(none)
	}
}
