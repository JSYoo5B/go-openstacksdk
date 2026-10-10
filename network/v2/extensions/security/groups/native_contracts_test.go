package groups_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeSecGroupTransport func(*http.Request) (*http.Response, error)

func (transport nativeSecGroupTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeSecGroupWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func nativeSecGroupClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return client
}

func nativeSecGroupOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "groups" {
		t.Fatal("generated groups context", err, wrapped)
	}
}

type nativeSecGroupCall struct{ method, path, query, body string }

func nativeSecGroupRecorder(cloud *testcloud.Cloud, calls *[]nativeSecGroupCall, reply func(*http.Request) *http.Response) {
	cloud.Provider.HTTPClient.Transport = nativeSecGroupTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeSecGroupCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
}

const nativeSecGroupRow = `{"id":"id-1","name":"web","stateful":true,"security_group_rules":[{"id":"r1","direction":"egress"}],"tags":["t"],"created_at":"2026-10-10T01:02:03Z"}`

func TestNativeSecGroupRoutesBodiesPagingAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeSecGroupCall
	nativeSecGroupRecorder(cloud, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeSecGroupWire(204, "")
		case req.Method == http.MethodPost:
			return nativeSecGroupWire(201, `{"security_group":`+nativeSecGroupRow+`}`)
		case req.URL.Path == "/neutron/v2.0/security-groups" && req.URL.Query().Get("marker") == "":
			return nativeSecGroupWire(200, `{"security_groups":[`+nativeSecGroupRow+`],"security_groups_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/security-groups?marker=x"}]}`)
		case req.URL.Path == "/other/security-groups":
			return nativeSecGroupWire(200, `{"security_groups":[{"id":"id-2","created_at":"2026-10-10T01:02:03"}]}`)
		}
		return nativeSecGroupWire(200, `{"security_group":`+nativeSecGroupRow+`}`)
	})
	api := groups.New(nativeSecGroupClient(cloud))
	ctx := context.Background()
	created, err := api.Create(ctx, groups.CreateOpts{Name: "web", Description: "d", Stateful: gophercloud.Disabled, ProjectID: "p"}, groups.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "id-1" && created.Stateful && created.Rules[0].Direction == "egress") {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "id-1")
	if err != nil || !(got.Name == "web" && got.CreatedAt.Year() == 2026) {
		t.Fatal(got, err)
	}
	var rows []*groups.SecGroup
	for value, err := range api.List(ctx, groups.WithListOptions(groups.ListOpts{Name: "web", Stateful: gophercloud.Enabled, Limit: 1, Tags: "t"})) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "id-1" && rows[1].ID == "id-2" && rows[1].CreatedAt.Second() == 3) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "id-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 5 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[2].query)
	want := []nativeSecGroupCall{
		{http.MethodPost, "/neutron/v2.0/security-groups", "", `{"security_group":{"description":"d","name":"web","project_id":"p","stateful":false,"x_extension":1}}`},
		{http.MethodGet, "/neutron/v2.0/security-groups/id-1", "", ""},
		{http.MethodGet, "/neutron/v2.0/security-groups", calls[2].query, ""},
		// The next href is followed as received.
		{http.MethodGet, "/other/security-groups", "marker=x", ""},
		{http.MethodDelete, "/neutron/v2.0/security-groups/id-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"name": {"web"}, "stateful": {"true"}, "limit": {"1"}, "tags": {"t"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeSecGroupStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*groups.API) error
	}{
		{"Create", []int{201, 202}, func(api *groups.API) error { _, err := api.Create(ctx, groups.CreateOpts{Name: "web"}); return err }},
		{"Get", []int{200}, func(api *groups.API) error { _, err := api.Get(ctx, "id-1"); return err }},
		{"Delete", []int{202, 204}, func(api *groups.API) error { return api.Delete(ctx, "id-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls []nativeSecGroupCall
				nativeSecGroupRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeSecGroupWire(code, `{"security_group":{}}`) })
				err := call.call(groups.New(nativeSecGroupClient(cloud)))
				nativeSecGroupOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"security_groups":[]}`}, {204, ""}} {
			cloud := testcloud.New(t)
			var calls []nativeSecGroupCall
			nativeSecGroupRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeSecGroupWire(tc.code, tc.body) })
			var errs []error
			for _, err := range groups.New(nativeSecGroupClient(cloud)).List(ctx) {
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
		cloud := testcloud.New(t)
		var calls []nativeSecGroupCall
		nativeSecGroupRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeSecGroupWire(201, `{}`) })
		api := groups.New(nativeSecGroupClient(cloud))
		for name, check := range map[string]func() error{
			"missing name": func() error { _, err := api.Create(ctx, groups.CreateOpts{}); return err },
			"name extension": func() error {
				_, err := api.Create(ctx, groups.CreateOpts{Name: "w"}, groups.WithCreateField("name", "x"))
				return err
			},
			"nil option": func() error { _, err := api.Create(ctx, groups.CreateOpts{Name: "w"}, nil); return err },
		} {
			err := check()
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeSecGroupOperation(t, err, "Create")
		}
		for _, err := range api.List(ctx, nil) {
			nativeSecGroupOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
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
