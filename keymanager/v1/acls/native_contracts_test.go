package acls_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/acls"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeACLTransport func(*http.Request) (*http.Response, error)

func (transport nativeACLTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeACLWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}, "X-ACL-Proof": {"actual"}}}
}

func nativeACLClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("key-manager", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/barbican/v1/"
	client.MoreHeaders = map[string]string{"X-Source": "direct"}
	return client
}

const nativeACLRow = `{"read":{"created":"2026-10-10T01:02:03","project-access":false,"updated":"2026-10-10T01:02:04","users":["u1","u2"]}}`

func TestNativeACLRoutesBodiesAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	client := nativeACLClient(cloud)
	type exchange struct{ method, path, body string }
	var seen []exchange
	cloud.Provider.HTTPClient.Transport = nativeACLTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		if req.Header.Get("X-Source") != "direct" || req.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(req.Header)
		}
		seen = append(seen, exchange{req.Method, req.URL.Path, raw})
		switch req.Method {
		case http.MethodGet:
			return nativeACLWire(200, nativeACLRow), nil
		case http.MethodDelete:
			return nativeACLWire(200, ""), nil
		}
		return nativeACLWire(200, `{"acl_ref":"https://kms/v1/secrets/s1/acl"}`), nil
	})
	api := acls.New(client)
	if api.RawClient() != client {
		t.Fatal("native client identity changed")
	}
	ctx := context.Background()
	want := acls.ACL{"read": acls.ACLDetails{Created: time.Date(2026, 10, 10, 1, 2, 3, 0, time.UTC), Updated: time.Date(2026, 10, 10, 1, 2, 4, 0, time.UTC), Users: []string{"u1", "u2"}}}
	for _, get := range []func(context.Context, string) (*acls.ACL, error){api.GetSecretACL, api.GetContainerACL} {
		got, err := get(ctx, "s1")
		if err != nil || !reflect.DeepEqual(*got, want) {
			t.Fatal(got, err)
		}
	}
	users := []string{"u1"}
	access := false
	read := acls.SetOpts{{Type: "read", Users: &users, ProjectAccess: &access}}
	ref, err := api.SetSecretACL(ctx, "s1", read)
	if err != nil || *ref != "https://kms/v1/secrets/s1/acl" {
		t.Fatal(ref, err)
	}
	if _, err := api.SetContainerACL(ctx, "c1", acls.SetOpts{{Type: "read"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.UpdateSecretACL(ctx, "s1", read, acls.WithUpdateSecretACLField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := api.UpdateContainerACL(ctx, "c1", acls.SetOpts{{Type: "read"}, {Type: "write", ProjectAccess: &access}}, acls.WithUpdateContainerACLField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	if err := api.DeleteSecretACL(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if err := api.DeleteContainerACL(ctx, "a/b"); err != nil {
		t.Fatal(err)
	}
	wantSeen := []exchange{
		{http.MethodGet, "/barbican/v1/secrets/s1/acl", ""},
		{http.MethodGet, "/barbican/v1/containers/s1/acl", ""},
		{http.MethodPut, "/barbican/v1/secrets/s1/acl", `{"read":{"project-access":false,"users":["u1"]}}`},
		{http.MethodPut, "/barbican/v1/containers/c1/acl", `{"read":{}}`},
		// One ACL type makes a single object envelope, so the shared merge
		// places the extension inside it; two types keep it at the top level.
		{http.MethodPatch, "/barbican/v1/secrets/s1/acl", `{"read":{"project-access":false,"users":["u1"],"x_extension":1}}`},
		{http.MethodPatch, "/barbican/v1/containers/c1/acl", `{"read":{},"write":{"project-access":false},"x_extension":1}`},
		{http.MethodDelete, "/barbican/v1/secrets/s1/acl", ""},
		{http.MethodDelete, "/barbican/v1/containers/a/b/acl", ""},
	}
	if !reflect.DeepEqual(seen, wantSeen) {
		t.Fatalf("%+v", seen)
	}
}

func TestNativeACLStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	read := acls.SetOpts{{Type: "read"}}
	for _, call := range []struct {
		name string
		call func(*acls.API) error
	}{
		{"GetSecretACL", func(api *acls.API) error { _, err := api.GetSecretACL(ctx, "s1"); return err }},
		{"GetContainerACL", func(api *acls.API) error { _, err := api.GetContainerACL(ctx, "c1"); return err }},
		{"SetSecretACL", func(api *acls.API) error { _, err := api.SetSecretACL(ctx, "s1", read); return err }},
		{"SetContainerACL", func(api *acls.API) error { _, err := api.SetContainerACL(ctx, "c1", read); return err }},
		{"UpdateSecretACL", func(api *acls.API) error { _, err := api.UpdateSecretACL(ctx, "s1", read); return err }},
		{"UpdateContainerACL", func(api *acls.API) error { _, err := api.UpdateContainerACL(ctx, "c1", read); return err }},
		{"DeleteSecretACL", func(api *acls.API) error { return api.DeleteSecretACL(ctx, "s1") }},
		{"DeleteContainerACL", func(api *acls.API) error { return api.DeleteContainerACL(ctx, "c1") }},
	} {
		for _, code := range []int{201, 202, 204, 404} {
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Provider.HTTPClient.Transport = nativeACLTransport(func(req *http.Request) (*http.Response, error) {
					requests.Add(1)
					return nativeACLWire(code, nativeACLRow), nil
				})
				err := call.call(acls.New(nativeACLClient(cloud)))
				var wrapped *resource.OperationError
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &wrapped) || wrapped.Operation != call.name || wrapped.Resource != "acls" || !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || requests.Load() != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("required type, collision and nil option", func(t *testing.T) {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeACLTransport(func(req *http.Request) (*http.Response, error) {
			requests.Add(1)
			return nativeACLWire(200, `{}`), nil
		})
		api := acls.New(nativeACLClient(cloud))
		checks := map[string]func() error{
			"missing type": func() error { _, err := api.SetSecretACL(ctx, "s1", acls.SetOpts{{}}); return err },
			"nested collision": func() error {
				users := []string{"u"}
				_, err := api.SetSecretACL(ctx, "s1", acls.SetOpts{{Type: "read", Users: &users}}, acls.WithSetSecretACLField("users", []string{}))
				return err
			},
			"nil option": func() error { _, err := api.UpdateContainerACL(ctx, "c1", read, nil); return err },
		}
		for name, check := range checks {
			if err := check(); err == nil {
				t.Fatal(name, "accepted")
			}
		}
		if requests.Load() != 0 {
			t.Fatal(requests.Load())
		}
	})
}
