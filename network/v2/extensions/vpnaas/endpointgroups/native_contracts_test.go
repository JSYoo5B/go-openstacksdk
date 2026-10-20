package endpointgroups_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeVPNTransport func(*http.Request) (*http.Response, error)

func (transport nativeVPNTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeVPNWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeVPNCall struct{ method, path, query, body string }

func nativeVPNAPI(t *testing.T, calls *[]nativeVPNCall, reply func(*http.Request) *http.Response) (*endpointgroups.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeVPNTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeVPNCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return endpointgroups.New(client), cloud
}

func nativeVPNOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "endpointgroups" {
		t.Fatal("generated endpointgroups context", err, wrapped)
	}
}

func nativeVPNStatuses(t *testing.T, name string, accepted []int, envelope string, call func(*endpointgroups.API) error) {
	t.Helper()
	for _, code := range []int{200, 201, 202, 204, 404} {
		if slices.Contains(accepted, code) {
			continue
		}
		t.Run(fmt.Sprintf("%s/%d", name, code), func(t *testing.T) {
			var calls []nativeVPNCall
			api, _ := nativeVPNAPI(t, &calls, func(*http.Request) *http.Response { return nativeVPNWire(code, `{"`+envelope+`":{}}`) })
			err := call(api)
			nativeVPNOperation(t, err, name)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, accepted) || len(calls) != 1 {
				t.Fatal(err, native)
			}
		})
	}
}

func nativeVPNListStatuses(t *testing.T, collection string, list func(*endpointgroups.API) []error) {
	t.Helper()
	for _, tc := range []struct {
		code int
		body string
	}{{404, `{}`}, {200, `{"` + collection + `":[]}`}, {204, ""}} {
		var calls []nativeVPNCall
		api, _ := nativeVPNAPI(t, &calls, func(*http.Request) *http.Response { return nativeVPNWire(tc.code, tc.body) })
		errs := list(api)
		var native gophercloud.ErrUnexpectedResponseCode
		switch {
		case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
		case tc.code == 200 && len(errs) == 0:
		case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
		default:
			t.Fatal(tc.code, errs)
		}
	}
}

const nativeVPNRow = `{"id":"x-1","name":"local","type":"subnet","endpoints":["sn-1"]}`

func TestNativeVPNEndpointGroupRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeVPNCall
	var cloud *testcloud.Cloud
	api, cloud := nativeVPNAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeVPNWire(204, "")
		case req.Method == http.MethodPost:
			return nativeVPNWire(201, `{"endpoint_group":`+nativeVPNRow+`}`)
		case req.URL.Path == "/neutron/v2.0/vpn/endpoint-groups":
			return nativeVPNWire(200, `{"endpoint_groups":[`+nativeVPNRow+`],"endpoint_groups_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/page?marker=x"}]}`)
		case req.URL.Path == "/other/page":
			return nativeVPNWire(200, `{"endpoint_groups":[{"id":"x-2","name":null}]}`)
		}
		return nativeVPNWire(200, `{"endpoint_group":`+nativeVPNRow+`}`)
	})
	created, err := api.Create(ctx, endpointgroups.CreateOpts{Name: "local", Type: endpointgroups.TypeSubnet, Endpoints: []string{"sn-1"}}, endpointgroups.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "x-1" && created.Type == "subnet" && created.Endpoints[0] == "sn-1") {
		t.Fatal(created, err)
	}
	// Endpoints has no omitempty, so a nil list is sent as null.
	if _, err := api.Create(ctx, endpointgroups.CreateOpts{}); err != nil {
		t.Fatal(err)
	}
	if got, err := api.Get(ctx, "x-1"); err != nil || got.ID != "x-1" {
		t.Fatal(got, err)
	}
	empty := ""
	if _, err := api.Update(ctx, "x-1", endpointgroups.UpdateOpts{Name: &empty}, endpointgroups.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for value, err := range api.List(ctx, endpointgroups.WithListOptions(endpointgroups.ListOpts{Name: "local", Type: "subnet"}), endpointgroups.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID+"/"+value.Name)
	}
	if !reflect.DeepEqual(ids, []string{"x-1/" + created.Name, "x-2/"}) {
		t.Fatal(ids)
	}
	if err := api.Delete(ctx, "x-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 7 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[4].query)
	base := "/neutron/v2.0/vpn/endpoint-groups"
	want := []nativeVPNCall{
		{http.MethodPost, base, "", `{"endpoint_group":{"endpoints":["sn-1"],"name":"local","type":"subnet","x_extension":1}}`},
		{http.MethodPost, base, "", `{"endpoint_group":{"endpoints":null}}`},
		{http.MethodGet, base + "/x-1", "", ""},

		{http.MethodPut, base + "/x-1", "", `{"endpoint_group":{"name":"","x_extension":1}}`},
		{http.MethodGet, base, calls[4].query, ""},
		{http.MethodGet, "/other/page", "marker=x", ""},
		{http.MethodDelete, base + "/x-1", "", ""},
	}

	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"name": {"local"}, "type": {"subnet"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeVPNEndpointGroupStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	nativeVPNStatuses(t, "Create", []int{201, 202}, "endpoint_group", func(api *endpointgroups.API) error {
		_, err := api.Create(ctx, endpointgroups.CreateOpts{})
		return err
	})
	nativeVPNStatuses(t, "Get", []int{200}, "endpoint_group", func(api *endpointgroups.API) error { _, err := api.Get(ctx, "x-1"); return err })
	nativeVPNStatuses(t, "Update", []int{200}, "endpoint_group", func(api *endpointgroups.API) error {
		_, err := api.Update(ctx, "x-1", endpointgroups.UpdateOpts{})
		return err
	})
	nativeVPNStatuses(t, "Delete", []int{202, 204}, "endpoint_group", func(api *endpointgroups.API) error { return api.Delete(ctx, "x-1") })
	nativeVPNListStatuses(t, "endpoint_groups", func(api *endpointgroups.API) (errs []error) {
		for _, err := range api.List(ctx) {
			errs = append(errs, err)
		}
		return errs
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeVPNCall
		api, _ := nativeVPNAPI(t, &calls, func(*http.Request) *http.Response { return nativeVPNWire(201, `{}`) })
		for name, err := range map[string]error{
			"core extension": func() error {
				_, err := api.Create(ctx, endpointgroups.CreateOpts{}, endpointgroups.WithCreateField("endpoints", nil))
				return err
			}(),
			"nil option": func() error { _, err := api.Update(ctx, "x-1", endpointgroups.UpdateOpts{}, nil); return err }(),
		} {
			if err == nil {
				t.Fatal(name, "accepted")
			}
			var wrapped *resource.OperationError
			if !errors.As(err, &wrapped) {
				t.Fatal(name, err)
			}
		}
		for _, err := range api.List(ctx, nil) {
			nativeVPNOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
