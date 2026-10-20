package networkipavailabilities_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeIPAvailabilityTransport func(*http.Request) (*http.Response, error)

func (transport nativeIPAvailabilityTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeIPAvailabilityWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeIPAvailabilityCall struct{ method, path, query, body string }

func nativeIPAvailabilityAPI(t *testing.T, calls *[]nativeIPAvailabilityCall, reply func(*http.Request) *http.Response) (*networkipavailabilities.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeIPAvailabilityTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeIPAvailabilityCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return networkipavailabilities.New(client), cloud
}

func nativeIPAvailabilityOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "networkipavailabilities" {
		t.Fatal("generated networkipavailabilities context", err, wrapped)
	}
}

// IPv6 totals exceed uint64; the exponent form is also accepted and converted to an integer string.
const nativeIPAvailabilityRow = `{"network_id":"net-1","network_name":"public","project_id":"p","total_ips":340282366920938463463374607431768211456,"used_ips":3,"subnet_ip_availability":[{"subnet_id":"sub-1","cidr":"2001:db8::/64","ip_version":6,"total_ips":18446744073709551615,"used_ips":1e3}]}`

func TestNativeIPAvailabilityRoutesSinglePageAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeIPAvailabilityCall
	var cloud *testcloud.Cloud
	api, cloud := nativeIPAvailabilityAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path == "/neutron/v2.0/network-ip-availabilities" {
			// The list is a single page; neither links form is followed.
			return nativeIPAvailabilityWire(200, `{"network_ip_availabilities":[`+nativeIPAvailabilityRow+`,{"network_id":"net-2","total_ips":0,"used_ips":0}],"network_ip_availabilities_links":[{"rel":"next","href":"`+cloud.Server.URL+`/never"}],"links":{"next":"`+cloud.Server.URL+`/never"}}`)
		}
		return nativeIPAvailabilityWire(200, `{"network_ip_availability":`+nativeIPAvailabilityRow+`}`)
	})
	got, err := api.Get(ctx, "net-1")
	if err != nil || !(got.NetworkName == "public" && got.TotalIPs == "340282366920938463463374607431768211456" && got.UsedIPs == "3" &&
		len(got.SubnetIPAvailabilities) == 1 && got.SubnetIPAvailabilities[0].TotalIPs == "18446744073709551615" &&
		got.SubnetIPAvailabilities[0].UsedIPs == "1000" && got.SubnetIPAvailabilities[0].IPVersion == 6) {
		t.Fatal(got, err)
	}
	var rows []*networkipavailabilities.NetworkIPAvailability
	for value, err := range api.List(ctx, networkipavailabilities.WithListOptions(networkipavailabilities.ListOpts{NetworkName: "public", IPVersion: "6", ProjectID: "p"}), networkipavailabilities.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].TotalIPs == "340282366920938463463374607431768211456" && rows[1].NetworkID == "net-2" && rows[1].TotalIPs == "0") {
		t.Fatal(rows)
	}
	query, _ := url.ParseQuery(calls[1].query)
	want := []nativeIPAvailabilityCall{
		{http.MethodGet, "/neutron/v2.0/network-ip-availabilities/net-1", "", ""},
		{http.MethodGet, "/neutron/v2.0/network-ip-availabilities", calls[1].query, ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"network_name": {"public"}, "ip_version": {"6"}, "project_id": {"p"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeIPAvailabilityStrictStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, code := range []int{201, 202, 204, 404} {
		t.Run(fmt.Sprintf("Get/%d", code), func(t *testing.T) {
			var calls []nativeIPAvailabilityCall
			api, _ := nativeIPAvailabilityAPI(t, &calls, func(*http.Request) *http.Response { return nativeIPAvailabilityWire(code, `{}`) })
			_, err := api.Get(ctx, "net-1")
			nativeIPAvailabilityOperation(t, err, "Get")
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || len(calls) != 1 {
				t.Fatal(err, native)
			}
		})
	}
	t.Run("envelope and count decode", func(t *testing.T) {
		for body, want := range map[string]string{
			// The plain envelope pointer stays nil, so these return no value and no error.
			`{}`:                               "nil",
			`{"network_ip_availability":null}`: "nil",
			`{"network_ip_availability":{"network_id":"n","total_ips":1,"used_ips":0}}`: "n",
			// Both counts are required by the native decoder, including inside every subnet.
			`{"network_ip_availability":{"network_id":"n","used_ips":0}}`:                                         "error",
			`{"network_ip_availability":{"network_id":"n","total_ips":null,"used_ips":0}}`:                        "error",
			`{"network_ip_availability":{"total_ips":1,"used_ips":0,"subnet_ip_availability":[{"total_ips":1}]}}`: "error",
			`{"network_ip_availability":[]}`:                                                                      "error",
		} {
			var calls []nativeIPAvailabilityCall
			api, _ := nativeIPAvailabilityAPI(t, &calls, func(*http.Request) *http.Response { return nativeIPAvailabilityWire(200, body) })
			got, err := api.Get(ctx, "net-1")
			switch want {
			case "error":
				nativeIPAvailabilityOperation(t, err, "Get")
			case "nil":
				if err != nil || got != nil {
					t.Fatal(body, got, err)
				}
			default:
				if err != nil || got == nil || got.NetworkID != want {
					t.Fatal(body, got, err)
				}
			}
		}
	})
	t.Run("list pager status, empty page, bodyless 204 and count decode", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"network_ip_availabilities":[]}`}, {204, ""}, {200, `{"network_ip_availabilities":[{"network_id":"n"}]}`}} {
			var calls []nativeIPAvailabilityCall
			api, _ := nativeIPAvailabilityAPI(t, &calls, func(*http.Request) *http.Response { return nativeIPAvailabilityWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			var wrapped *resource.OperationError
			switch {
			case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && !errors.As(errs[0], &wrapped) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
			case tc.code == 200 && strings.Contains(tc.body, `"n"`) && len(errs) == 1 && errs[0] != nil && !errors.As(errs[0], &wrapped):
			case tc.code == 200 && !strings.Contains(tc.body, `"n"`) && len(errs) == 0:
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, tc.body, errs)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeIPAvailabilityCall
		api, _ := nativeIPAvailabilityAPI(t, &calls, func(*http.Request) *http.Response { return nativeIPAvailabilityWire(200, `{}`) })
		var errs []error
		for _, err := range api.List(ctx, nil) {
			nativeIPAvailabilityOperation(t, err, "List")
			errs = append(errs, err)
		}
		if len(errs) != 1 || len(calls) != 0 {
			t.Fatal(errs, calls)
		}
	})
}
