package servergroups_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeGroupTransport func(*http.Request) (*http.Response, error)

func (transport nativeGroupTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeGroupWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func nativeGroupClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	return client
}

func nativeGroupOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "servergroups" {
		t.Fatal("generated server group context", err, wrapped)
	}
}

type nativeGroupCall struct{ method, path, query, body string }

const nativeGroupRow = `{"id":"g1","name":"web","policies":["anti-affinity"],"members":["s1"],"user_id":"u","project_id":"p","policy":"anti-affinity","rules":{"max_server_per_host":2}}`

func TestNativeServerGroupsRoutesBodiesAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeGroupCall
	cloud.Provider.HTTPClient.Transport = nativeGroupTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		calls = append(calls, nativeGroupCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		switch {
		case req.Method == http.MethodDelete:
			return nativeGroupWire(204, ""), nil
		case req.URL.Path == "/nova/v2.1/os-server-groups" && req.Method == http.MethodGet:
			// A links entry is ignored: the native page is a single page.
			return nativeGroupWire(200, `{"server_groups":[`+nativeGroupRow+`,{"id":"g2","policy":null}],"server_groups_links":[{"rel":"next","href":"http://other/next"}]}`), nil
		}
		return nativeGroupWire(200, `{"server_group":`+nativeGroupRow+`}`), nil
	})
	api := servergroups.New(nativeGroupClient(cloud))
	ctx := context.Background()
	two := 2
	created, err := api.Create(ctx, servergroups.CreateOpts{Name: "web", Policies: []string{"anti-affinity"}, Policy: "anti-affinity", Rules: &servergroups.Rules{MaxServerPerHost: two}}, servergroups.WithCreateField("x_extension", 1))
	if err != nil || created.ID != "g1" || *created.Policy != "anti-affinity" || created.Rules.MaxServerPerHost != 2 || !reflect.DeepEqual(created.Members, []string{"s1"}) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "g1")
	if err != nil || got.ProjectID != "p" || got.UserID != "u" {
		t.Fatal(got, err)
	}
	var ids []string
	for value, err := range api.List(ctx, servergroups.WithListOptions(servergroups.ListOpts{AllProjects: true, Limit: 5, Offset: 10}), servergroups.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		if value.ID == "g2" && value.Policy != nil {
			t.Fatal("null policy decodes to nil", value.Policy)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"g1", "g2"}) {
		t.Fatal(ids)
	}
	if err := api.Delete(ctx, "g1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 4 {
		t.Fatalf("%+v", calls)
	}
	listQuery, _ := url.ParseQuery(calls[2].query)
	want := []nativeGroupCall{
		{http.MethodPost, "/nova/v2.1/os-server-groups", "", `{"server_group":{"name":"web","policies":["anti-affinity"],"policy":"anti-affinity","rules":{"max_server_per_host":2},"x_extension":1}}`},
		{http.MethodGet, "/nova/v2.1/os-server-groups/g1", "", ""},
		{http.MethodGet, "/nova/v2.1/os-server-groups", calls[2].query, ""},
		{http.MethodDelete, "/nova/v2.1/os-server-groups/g1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(listQuery, url.Values{"all_projects": {"true"}, "limit": {"5"}, "offset": {"10"}, "extra": {"1"}}) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeServerGroupsStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*servergroups.API) error
	}{
		{"Create", []int{200}, func(api *servergroups.API) error {
			_, err := api.Create(ctx, servergroups.CreateOpts{Name: "g", Policy: "affinity"})
			return err
		}},
		{"Get", []int{200}, func(api *servergroups.API) error { _, err := api.Get(ctx, "g1"); return err }},
		{"Delete", []int{202, 204}, func(api *servergroups.API) error { return api.Delete(ctx, "g1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Provider.HTTPClient.Transport = nativeGroupTransport(func(req *http.Request) (*http.Response, error) {
					requests.Add(1)
					return nativeGroupWire(code, `{"server_group":{"id":"g1"}}`), nil
				})
				err := call.call(servergroups.New(nativeGroupClient(cloud)))
				nativeGroupOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || requests.Load() != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("List native pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
			want func([]error) bool
		}{
			{404, `{}`, func(errs []error) bool {
				var native gophercloud.ErrUnexpectedResponseCode
				return len(errs) == 1 && errors.As(errs[0], &native) && native.Actual == 404 && reflect.DeepEqual(native.Expected, []int{200, 204, 300})
			}},
			{200, `{"server_groups":[]}`, func(errs []error) bool { return len(errs) == 0 }},
			{204, "", func(errs []error) bool { return len(errs) == 1 && errors.Is(errs[0], io.EOF) }},
		} {
			cloud := testcloud.New(t)
			cloud.Provider.HTTPClient.Transport = nativeGroupTransport(func(req *http.Request) (*http.Response, error) {
				return nativeGroupWire(tc.code, tc.body), nil
			})
			var errs []error
			for value, err := range servergroups.New(nativeGroupClient(cloud)).List(ctx) {
				if value != nil {
					t.Fatal(value)
				}
				errs = append(errs, err)
			}
			if !tc.want(errs) {
				t.Fatal(tc.code, errs)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeGroupTransport(func(req *http.Request) (*http.Response, error) {
			requests.Add(1)
			return nativeGroupWire(200, `{}`), nil
		})
		api := servergroups.New(nativeGroupClient(cloud))
		for name, check := range map[string]func() error{
			"missing name": func() error { _, err := api.Create(ctx, servergroups.CreateOpts{Policy: "affinity"}); return err },
			"name extension": func() error {
				_, err := api.Create(ctx, servergroups.CreateOpts{Name: "g"}, servergroups.WithCreateField("name", "x"))
				return err
			},
			"nil option": func() error { _, err := api.Create(ctx, servergroups.CreateOpts{Name: "g"}, nil); return err },
		} {
			err := check()
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeGroupOperation(t, err, "Create")
		}
		for _, err := range api.List(ctx, nil) {
			nativeGroupOperation(t, err, "List")
		}
		if requests.Load() != 0 {
			t.Fatal(requests.Load())
		}
	})
}

func contains(values []int, value int) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
