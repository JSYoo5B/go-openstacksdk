package rbacpolicies_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeRBACTransport func(*http.Request) (*http.Response, error)

func (transport nativeRBACTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeRBACWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeRBACCall struct{ method, path, query, body string }

func nativeRBACAPI(t *testing.T, calls *[]nativeRBACCall, reply func(*http.Request) *http.Response) (*rbacpolicies.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeRBACTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeRBACCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return rbacpolicies.New(client), cloud
}

func nativeRBACOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "rbacpolicies" {
		t.Fatal("generated rbacpolicies context", err, wrapped)
	}
}

const nativeRBACRow = `{"id":"rb-1","action":"access_as_shared","object_type":"network","object_id":"net-1","target_tenant":"*","project_id":"p","tags":null}`

var nativeRBACCreate = rbacpolicies.CreateOpts{Action: rbacpolicies.ActionAccessShared, ObjectType: "network", ObjectID: "net-1", TargetTenant: "*"}

func TestNativeRBACPolicyRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeRBACCall
	var cloud *testcloud.Cloud
	api, cloud := nativeRBACAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeRBACWire(204, "")
		case req.Method == http.MethodPost:
			return nativeRBACWire(201, `{"rbac_policy":`+nativeRBACRow+`}`)
		case req.Method == http.MethodPut:
			// Update also accepts 201.
			return nativeRBACWire(201, `{"rbac_policy":`+nativeRBACRow+`}`)
		case req.URL.Path == "/neutron/v2.0/rbac-policies":
			// rbac_policies_links is ignored; only a links.next string is followed.
			return nativeRBACWire(200, `{"rbac_policies":[`+nativeRBACRow+`],"rbac_policies_links":[{"rel":"next","href":"`+cloud.Server.URL+`/never"}],"links":{"next":"`+cloud.Server.URL+`/other/rbac"}}`)
		case req.URL.Path == "/other/rbac":
			return nativeRBACWire(200, `{"rbac_policies":[{"id":"rb-2","action":"access_as_external"}]}`)
		}
		return nativeRBACWire(200, `{"rbac_policy":`+nativeRBACRow+`}`)
	})
	created, err := api.Create(ctx, nativeRBACCreate, rbacpolicies.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "rb-1" && created.Action == rbacpolicies.ActionAccessShared && created.TargetTenant == "*" && created.Tags == nil) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "rb-1")
	if err != nil || got.ObjectID != "net-1" {
		t.Fatal(got, err)
	}
	if _, err := api.Update(ctx, "rb-1", rbacpolicies.UpdateOpts{TargetTenant: "p2"}, rbacpolicies.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	var rows []*rbacpolicies.RBACPolicy
	for value, err := range api.List(ctx, rbacpolicies.WithListOptions(rbacpolicies.ListOpts{ObjectType: "network", Action: rbacpolicies.ActionAccessShared, Limit: 1}), rbacpolicies.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "rb-1" && rows[1].Action == rbacpolicies.ActionAccessExternal) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "rb-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 6 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[3].query)
	want := []nativeRBACCall{
		{http.MethodPost, "/neutron/v2.0/rbac-policies", "", `{"rbac_policy":{"action":"access_as_shared","object_id":"net-1","object_type":"network","target_tenant":"*","x_extension":1}}`},
		{http.MethodGet, "/neutron/v2.0/rbac-policies/rb-1", "", ""},
		{http.MethodPut, "/neutron/v2.0/rbac-policies/rb-1", "", `{"rbac_policy":{"target_tenant":"p2","x_extension":1}}`},
		{http.MethodGet, "/neutron/v2.0/rbac-policies", calls[3].query, ""},
		{http.MethodGet, "/other/rbac", "", ""},
		{http.MethodDelete, "/neutron/v2.0/rbac-policies/rb-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"object_type": {"network"}, "action": {"access_as_shared"}, "limit": {"1"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeRBACPolicyStrictStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*rbacpolicies.API) error
	}{
		{"Create", []int{201, 202}, func(api *rbacpolicies.API) error { _, err := api.Create(ctx, nativeRBACCreate); return err }},
		{"Get", []int{200}, func(api *rbacpolicies.API) error { _, err := api.Get(ctx, "rb-1"); return err }},
		{"Update", []int{200, 201}, func(api *rbacpolicies.API) error {
			_, err := api.Update(ctx, "rb-1", rbacpolicies.UpdateOpts{TargetTenant: "p2"})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *rbacpolicies.API) error { return api.Delete(ctx, "rb-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeRBACCall
				api, _ := nativeRBACAPI(t, &calls, func(*http.Request) *http.Response { return nativeRBACWire(code, `{"rbac_policy":{}}`) })
				err := call.call(api)
				nativeRBACOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			// An empty object or null envelope yields a zero value; other keys without the envelope fail.
			`{}`:                      false,
			`{"rbac_policy":null}`:    false,
			`{"other":{"id":"rb-1"}}`: true,
			`{"rbac_policy":[]}`:      true,
		} {
			var calls []nativeRBACCall
			api, _ := nativeRBACAPI(t, &calls, func(*http.Request) *http.Response { return nativeRBACWire(200, body) })
			got, err := api.Get(ctx, "rb-1")
			if wantErr {
				nativeRBACOperation(t, err, "Get")
			} else if err != nil || got == nil || got.ID != "" {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"rbac_policies":[]}`}, {204, ""}} {
			var calls []nativeRBACCall
			api, _ := nativeRBACAPI(t, &calls, func(*http.Request) *http.Response { return nativeRBACWire(tc.code, tc.body) })
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
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeRBACCall
		api, _ := nativeRBACAPI(t, &calls, func(*http.Request) *http.Response { return nativeRBACWire(201, `{}`) })
		missing := nativeRBACCreate
		missing.TargetTenant = ""
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create target": {"Create", func() error { _, err := api.Create(ctx, missing); return err }()},
			"create action": {"Create", func() error {
				_, err := api.Create(ctx, rbacpolicies.CreateOpts{ObjectType: "network", ObjectID: "n", TargetTenant: "*"})
				return err
			}()},
			"update target": {"Update", func() error { _, err := api.Update(ctx, "rb-1", rbacpolicies.UpdateOpts{}); return err }()},
			"create extension": {"Create", func() error {
				_, err := api.Create(ctx, nativeRBACCreate, rbacpolicies.WithCreateField("object_id", "x"))
				return err
			}()},
			"update nil option": {"Update", func() error {
				_, err := api.Update(ctx, "rb-1", rbacpolicies.UpdateOpts{TargetTenant: "p2"}, nil)
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeRBACOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeRBACOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
