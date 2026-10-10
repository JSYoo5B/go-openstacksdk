package subnetpools_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativePoolTransport func(*http.Request) (*http.Response, error)

func (transport nativePoolTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativePoolWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativePoolCall struct{ method, path, query, body string }

func nativePoolAPI(t *testing.T, calls *[]nativePoolCall, reply func(*http.Request) *http.Response) (*subnetpools.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativePoolTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativePoolCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return subnetpools.New(client), cloud
}

func nativePoolOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "subnetpools" {
		t.Fatal("generated subnetpools context", err, wrapped)
	}
}

const nativePoolRow = `{"id":"sp-1","name":"pool","prefixes":["10.0.0.0/16"],"default_prefixlen":"24","min_prefixlen":8,"max_prefixlen":32,"address_scope_id":null,"ip_version":4,"shared":false,"is_default":true,"revision_number":3,"created_at":"2026-10-11T01:02:03Z"}`

func TestNativeSubnetPoolRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativePoolCall
	var cloud *testcloud.Cloud
	api, cloud := nativePoolAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativePoolWire(204, "")
		case req.Method == http.MethodPost:
			return nativePoolWire(201, `{"subnetpool":`+nativePoolRow+`}`)
		case req.URL.Path == "/neutron/v2.0/subnetpools":
			return nativePoolWire(200, `{"subnetpools":[`+nativePoolRow+`],"subnetpools_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/subnetpools?marker=x"}]}`)
		case req.URL.Path == "/other/subnetpools":
			return nativePoolWire(200, `{"subnetpools":[{"id":"sp-2","default_prefixlen":64,"min_prefixlen":"48","max_prefixlen":128,"created_at":"2026-10-11T01:02:03"}]}`)
		}
		return nativePoolWire(200, `{"subnetpool":`+nativePoolRow+`}`)
	})
	created, err := api.Create(ctx, subnetpools.CreateOpts{Name: "pool", Prefixes: []string{"10.0.0.0/16"}, DefaultPrefixLen: 24, IsDefault: true}, subnetpools.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "sp-1" && created.DefaultPrefixLen == 24 && created.MinPrefixLen == 8 && created.AddressScopeID == "" && created.IsDefault && created.RevisionNumber == 3) {
		t.Fatal(created, err)
	}
	// Prefixes has no omitempty, so a nil list is sent as null.
	if _, err := api.Create(ctx, subnetpools.CreateOpts{}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "sp-1")
	if err != nil || !(got.Prefixes[0] == "10.0.0.0/16" && got.CreatedAt.Year() == 2026) {
		t.Fatal(got, err)
	}
	var rows []*subnetpools.SubnetPool
	for value, err := range api.List(ctx, subnetpools.WithListOptions(subnetpools.ListOpts{Name: "pool", IPVersion: 4, Shared: gophercloud.Disabled, Limit: 1}), subnetpools.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "sp-1" && rows[1].ID == "sp-2" && rows[1].MinPrefixLen == 48 && rows[1].CreatedAt.Second() == 3) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "sp-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 6 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[3].query)
	want := []nativePoolCall{
		{http.MethodPost, "/neutron/v2.0/subnetpools", "", `{"subnetpool":{"default_prefixlen":24,"is_default":true,"name":"pool","prefixes":["10.0.0.0/16"],"x_extension":1}}`},
		{http.MethodPost, "/neutron/v2.0/subnetpools", "", `{"subnetpool":{"name":"","prefixes":null}}`},
		{http.MethodGet, "/neutron/v2.0/subnetpools/sp-1", "", ""},
		{http.MethodGet, "/neutron/v2.0/subnetpools", calls[3].query, ""},
		{http.MethodGet, "/other/subnetpools", "marker=x", ""},
		{http.MethodDelete, "/neutron/v2.0/subnetpools/sp-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"name": {"pool"}, "ip_version": {"4"}, "shared": {"false"}, "limit": {"1"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeSubnetPoolPrefixLengthDecodeFailures(t *testing.T) {
	ctx := context.Background()
	for name, row := range map[string]string{
		// The native decoder accepts only numbers and numeric strings.
		"missing":     `{"id":"sp-1","min_prefixlen":8,"max_prefixlen":32}`,
		"null":        `{"id":"sp-1","default_prefixlen":null,"min_prefixlen":8,"max_prefixlen":32}`,
		"non-numeric": `{"id":"sp-1","default_prefixlen":24,"min_prefixlen":"x","max_prefixlen":32}`,
		"boolean":     `{"id":"sp-1","default_prefixlen":24,"min_prefixlen":8,"max_prefixlen":true}`,
	} {
		t.Run(name, func(t *testing.T) {
			var calls []nativePoolCall
			api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(200, `{"subnetpool":`+row+`}`) })
			got, err := api.Get(ctx, "sp-1")
			nativePoolOperation(t, err, "Get")
			// Extract keeps the partially decoded envelope value beside the error.
			if got == nil || got.ID != "sp-1" || len(calls) != 1 {
				t.Fatal(got, calls)
			}
		})
	}
}

func TestNativeSubnetPoolPrefixOperations(t *testing.T) {
	ctx := context.Background()
	var calls []nativePoolCall
	api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response {
		return nativePoolWire(200, `{"prefixes":["10.0.0.0/16","10.2.0.0/16"]}`)
	})
	added, err := api.AddPrefixes(ctx, "sp-1", subnetpools.PrefixesOpsOpts{Prefixes: []string{"10.2.0.0/16"}}, subnetpools.WithAddPrefixesField("x_extension", 1))
	if err != nil || !reflect.DeepEqual(added, []string{"10.0.0.0/16", "10.2.0.0/16"}) {
		t.Fatal(added, err)
	}
	if _, err := api.RemovePrefixes(ctx, "sp-1", subnetpools.PrefixesOpsOpts{Prefixes: []string{"10.2.0.0/16"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.AddPrefixes(ctx, "sp-1", subnetpools.PrefixesOpsOpts{}); err != nil {
		t.Fatal(err)
	}
	want := []nativePoolCall{
		// The prefix body has no envelope, so an extension sits beside prefixes.
		{http.MethodPut, "/neutron/v2.0/subnetpools/sp-1/add_prefixes", "", `{"prefixes":["10.2.0.0/16"],"x_extension":1}`},
		{http.MethodPut, "/neutron/v2.0/subnetpools/sp-1/remove_prefixes", "", `{"prefixes":["10.2.0.0/16"]}`},
		{http.MethodPut, "/neutron/v2.0/subnetpools/sp-1/add_prefixes", "", `{}`},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	calls = nil
	for operation, err := range map[string]error{
		"AddPrefixes": func() error {
			_, err := api.AddPrefixes(ctx, "sp-1", subnetpools.PrefixesOpsOpts{}, subnetpools.WithAddPrefixesField("prefixes", nil))
			return err
		}(),
		"RemovePrefixes": func() error {
			_, err := api.RemovePrefixes(ctx, "sp-1", subnetpools.PrefixesOpsOpts{}, nil)
			return err
		}(),
	} {
		if err == nil {
			t.Fatal(operation, "accepted")
		}
		nativePoolOperation(t, err, operation)
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}

func TestNativeSubnetPoolStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	prefixes := subnetpools.PrefixesOpsOpts{Prefixes: []string{"10.2.0.0/16"}}
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*subnetpools.API) error
	}{
		// Create narrows the default POST codes to 201.
		{"Create", []int{201}, func(api *subnetpools.API) error { _, err := api.Create(ctx, subnetpools.CreateOpts{}); return err }},
		{"Get", []int{200}, func(api *subnetpools.API) error { _, err := api.Get(ctx, "sp-1"); return err }},
		{"Delete", []int{202, 204}, func(api *subnetpools.API) error { return api.Delete(ctx, "sp-1") }},
		{"AddPrefixes", []int{200}, func(api *subnetpools.API) error { _, err := api.AddPrefixes(ctx, "sp-1", prefixes); return err }},
		{"RemovePrefixes", []int{200}, func(api *subnetpools.API) error { _, err := api.RemovePrefixes(ctx, "sp-1", prefixes); return err }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativePoolCall
				api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(code, `{"subnetpool":{}}`) })
				err := call.call(api)
				nativePoolOperation(t, err, call.name)
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
		}{{404, `{}`}, {200, `{"subnetpools":[]}`}, {204, ""}} {
			var calls []nativePoolCall
			api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(tc.code, tc.body) })
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
		var calls []nativePoolCall
		api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(201, `{}`) })
		for name, check := range map[string]func() error{
			"prefixes extension": func() error {
				_, err := api.Create(ctx, subnetpools.CreateOpts{}, subnetpools.WithCreateField("prefixes", []string{}))
				return err
			},
			"nil option": func() error { _, err := api.Create(ctx, subnetpools.CreateOpts{}, nil); return err },
		} {
			err := check()
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativePoolOperation(t, err, "Create")
		}
		for _, err := range api.List(ctx, nil) {
			nativePoolOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
