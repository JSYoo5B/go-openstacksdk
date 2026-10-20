package subnets_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeSubnetTransport func(*http.Request) (*http.Response, error)

func (transport nativeSubnetTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeSubnetWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func nativeSubnetClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return client
}

func nativeSubnetOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "subnets" {
		t.Fatal("generated subnets context", err, wrapped)
	}
}

type nativeSubnetCall struct{ method, path, query, body string }

func nativeSubnetRecorder(cloud *testcloud.Cloud, calls *[]nativeSubnetCall, reply func(*http.Request) *http.Response) {
	cloud.Provider.HTTPClient.Transport = nativeSubnetTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeSubnetCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
}

const nativeSubnetRow = `{"id":"id-1","network_id":"n1","cidr":"10.0.0.0/24","ip_version":4,"gateway_ip":null,"allocation_pools":[{"start":"10.0.0.10","end":"10.0.0.20"}],"host_routes":[{"destination":"0.0.0.0/0","nexthop":"10.0.0.1"}],"created_at":"2026-10-10T01:02:03Z"}`

func TestNativeSubnetRoutesBodiesPagingAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeSubnetCall
	nativeSubnetRecorder(cloud, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeSubnetWire(204, "")
		case req.Method == http.MethodPost:
			return nativeSubnetWire(201, `{"subnet":`+nativeSubnetRow+`}`)
		case req.URL.Path == "/neutron/v2.0/subnets" && req.URL.Query().Get("marker") == "":
			return nativeSubnetWire(200, `{"subnets":[`+nativeSubnetRow+`],"subnets_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/subnets?marker=x"}]}`)
		case req.URL.Path == "/other/subnets":
			return nativeSubnetWire(200, `{"subnets":[{"id":"id-2","created_at":"2026-10-10T01:02:03"}]}`)
		}
		return nativeSubnetWire(200, `{"subnet":`+nativeSubnetRow+`}`)
	})
	api := subnets.New(nativeSubnetClient(cloud))
	ctx := context.Background()
	created, err := api.Create(ctx, subnets.CreateOpts{NetworkID: "n1", CIDR: "10.0.0.0/24", IPVersion: gophercloud.IPv4, GatewayIP: new(string), AllocationPools: []subnets.AllocationPool{{Start: "10.0.0.10", End: "10.0.0.20"}}}, subnets.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "id-1" && created.GatewayIP == "" && created.AllocationPools[0].End == "10.0.0.20" && created.HostRoutes[0].NextHop == "10.0.0.1") {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "id-1")
	if err != nil || !(got.CIDR == "10.0.0.0/24" && got.IPVersion == 4) {
		t.Fatal(got, err)
	}
	var rows []*subnets.Subnet
	for value, err := range api.List(ctx, subnets.WithListOptions(subnets.ListOpts{NetworkID: "n1", IPVersion: 6, EnableDHCP: gophercloud.Disabled, Limit: 1}), subnets.WithListQuery("extra", "1")) {
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
	want := []nativeSubnetCall{
		{http.MethodPost, "/neutron/v2.0/subnets", "", `{"subnet":{"allocation_pools":[{"end":"10.0.0.20","start":"10.0.0.10"}],"cidr":"10.0.0.0/24","gateway_ip":null,"ip_version":4,"network_id":"n1","x_extension":1}}`},
		{http.MethodGet, "/neutron/v2.0/subnets/id-1", "", ""},
		{http.MethodGet, "/neutron/v2.0/subnets", calls[2].query, ""},
		// The next href is followed as received.
		{http.MethodGet, "/other/subnets", "marker=x", ""},
		{http.MethodDelete, "/neutron/v2.0/subnets/id-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"network_id": {"n1"}, "ip_version": {"6"}, "enable_dhcp": {"false"}, "limit": {"1"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeSubnetStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*subnets.API) error
	}{
		{"Create", []int{201, 202}, func(api *subnets.API) error {
			_, err := api.Create(ctx, subnets.CreateOpts{NetworkID: "n1"})
			return err
		}},
		{"Get", []int{200}, func(api *subnets.API) error { _, err := api.Get(ctx, "id-1"); return err }},
		{"Delete", []int{202, 204}, func(api *subnets.API) error { return api.Delete(ctx, "id-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls []nativeSubnetCall
				nativeSubnetRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeSubnetWire(code, `{"subnet":{}}`) })
				err := call.call(subnets.New(nativeSubnetClient(cloud)))
				nativeSubnetOperation(t, err, call.name)
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
		}{{404, `{}`}, {200, `{"subnets":[]}`}, {204, ""}} {
			cloud := testcloud.New(t)
			var calls []nativeSubnetCall
			nativeSubnetRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeSubnetWire(tc.code, tc.body) })
			var errs []error
			for _, err := range subnets.New(nativeSubnetClient(cloud)).List(ctx) {
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
		var calls []nativeSubnetCall
		nativeSubnetRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeSubnetWire(201, `{}`) })
		api := subnets.New(nativeSubnetClient(cloud))
		for name, check := range map[string]func() error{
			"missing network": func() error { _, err := api.Create(ctx, subnets.CreateOpts{CIDR: "10.0.0.0/24"}); return err },
			"network_id extension": func() error {
				_, err := api.Create(ctx, subnets.CreateOpts{NetworkID: "n"}, subnets.WithCreateField("network_id", "x"))
				return err
			},
			"nil option": func() error { _, err := api.Create(ctx, subnets.CreateOpts{NetworkID: "n"}, nil); return err },
		} {
			err := check()
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeSubnetOperation(t, err, "Create")
		}
		for _, err := range api.List(ctx, nil) {
			nativeSubnetOperation(t, err, "List")
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
