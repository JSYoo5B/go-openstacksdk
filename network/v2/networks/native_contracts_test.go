package networks_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeNetworkTransport func(*http.Request) (*http.Response, error)

func (transport nativeNetworkTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeNetworkWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func nativeNetworkClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return client
}

func nativeNetworkOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "networks" {
		t.Fatal("generated networks context", err, wrapped)
	}
}

type nativeNetworkCall struct{ method, path, query, body string }

func nativeNetworkRecorder(cloud *testcloud.Cloud, calls *[]nativeNetworkCall, reply func(*http.Request) *http.Response) {
	cloud.Provider.HTTPClient.Transport = nativeNetworkTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeNetworkCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
}

const nativeNetworkRow = `{"id":"id-1","name":"net","status":"ACTIVE","subnets":["sn"],"admin_state_up":true,"shared":true,"tags":["t"],"revision_number":3,"created_at":"2026-10-10T01:02:03Z"}`

func TestNativeNetworkRoutesBodiesPagingAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeNetworkCall
	nativeNetworkRecorder(cloud, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeNetworkWire(204, "")
		case req.Method == http.MethodPost:
			return nativeNetworkWire(201, `{"network":`+nativeNetworkRow+`}`)
		case req.URL.Path == "/neutron/v2.0/networks" && req.URL.Query().Get("marker") == "":
			return nativeNetworkWire(200, `{"networks":[`+nativeNetworkRow+`],"networks_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/networks?marker=x"}]}`)
		case req.URL.Path == "/other/networks":
			return nativeNetworkWire(200, `{"networks":[{"id":"id-2","created_at":"2026-10-10T01:02:03"}]}`)
		}
		return nativeNetworkWire(200, `{"network":`+nativeNetworkRow+`}`)
	})
	api := networks.New(nativeNetworkClient(cloud))
	ctx := context.Background()
	created, err := api.Create(ctx, networks.CreateOpts{Name: "net", AdminStateUp: gophercloud.Enabled, Shared: gophercloud.Enabled, AvailabilityZoneHints: []string{"az1"}}, networks.WithCreateField("mtu", 1450))
	if err != nil || !(created.ID == "id-1" && created.RevisionNumber == 3 && created.CreatedAt.Year() == 2026 && created.Shared) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "id-1")
	if err != nil || !(got.Status == "ACTIVE" && reflect.DeepEqual(got.Subnets, []string{"sn"})) {
		t.Fatal(got, err)
	}
	var rows []*networks.Network
	for value, err := range api.List(ctx, networks.WithListOptions(networks.ListOpts{Name: "net", Shared: gophercloud.Enabled, Limit: 1, Tags: "t"}), networks.WithListQuery("extra", "1")) {
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
	want := []nativeNetworkCall{
		{http.MethodPost, "/neutron/v2.0/networks", "", `{"network":{"admin_state_up":true,"availability_zone_hints":["az1"],"mtu":1450,"name":"net","shared":true}}`},
		{http.MethodGet, "/neutron/v2.0/networks/id-1", "", ""},
		{http.MethodGet, "/neutron/v2.0/networks", calls[2].query, ""},
		// The next href is followed as received.
		{http.MethodGet, "/other/networks", "marker=x", ""},
		{http.MethodDelete, "/neutron/v2.0/networks/id-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"name": {"net"}, "shared": {"true"}, "limit": {"1"}, "tags": {"t"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeNetworkStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*networks.API) error
	}{
		{"Create", []int{201, 202}, func(api *networks.API) error { _, err := api.Create(ctx, networks.CreateOpts{Name: "n"}); return err }},
		{"Get", []int{200}, func(api *networks.API) error { _, err := api.Get(ctx, "id-1"); return err }},
		{"Delete", []int{202, 204}, func(api *networks.API) error { return api.Delete(ctx, "id-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls []nativeNetworkCall
				nativeNetworkRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeNetworkWire(code, `{"network":{}}`) })
				err := call.call(networks.New(nativeNetworkClient(cloud)))
				nativeNetworkOperation(t, err, call.name)
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
		}{{404, `{}`}, {200, `{"networks":[]}`}, {204, ""}} {
			cloud := testcloud.New(t)
			var calls []nativeNetworkCall
			nativeNetworkRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeNetworkWire(tc.code, tc.body) })
			var errs []error
			for _, err := range networks.New(nativeNetworkClient(cloud)).List(ctx) {
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
		var calls []nativeNetworkCall
		nativeNetworkRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeNetworkWire(201, `{}`) })
		api := networks.New(nativeNetworkClient(cloud))
		for name, check := range map[string]func() error{
			"name extension": func() error {
				_, err := api.Create(ctx, networks.CreateOpts{}, networks.WithCreateField("name", "x"))
				return err
			},
			"nil option": func() error { _, err := api.Create(ctx, networks.CreateOpts{}, nil); return err },
		} {
			err := check()
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeNetworkOperation(t, err, "Create")
		}
		for _, err := range api.List(ctx, nil) {
			nativeNetworkOperation(t, err, "List")
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
