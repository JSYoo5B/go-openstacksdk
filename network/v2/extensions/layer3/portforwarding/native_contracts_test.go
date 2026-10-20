package portforwarding_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeForwardTransport func(*http.Request) (*http.Response, error)

func (transport nativeForwardTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeForwardWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeForwardCall struct{ method, path, query, body string }

func nativeForwardAPI(t *testing.T, calls *[]nativeForwardCall, reply func(*http.Request) *http.Response) (*portforwarding.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeForwardTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeForwardCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return portforwarding.New(client), cloud
}

func nativeForwardOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "portforwarding" {
		t.Fatal("generated portforwarding context", err, wrapped)
	}
}

const nativeForwardRow = `{"id":"pf-1","internal_port_id":"port","internal_ip_address":"10.0.0.5","internal_port":22,"external_port":2222,"protocol":"tcp","description":null}`

func TestNativePortForwardingRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeForwardCall
	var cloud *testcloud.Cloud
	api, cloud := nativeForwardAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeForwardWire(204, "")
		case req.Method == http.MethodPost:
			return nativeForwardWire(201, `{"port_forwarding":`+nativeForwardRow+`}`)
		case req.URL.Path == "/neutron/v2.0/floatingips/fip-1/port_forwardings":
			return nativeForwardWire(200, `{"port_forwardings":[`+nativeForwardRow+`],"port_forwarding_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/pf?marker=x"}]}`)
		case req.URL.Path == "/other/pf":
			// The plural port_forwardings_links key is not read by the native page.
			return nativeForwardWire(200, `{"port_forwardings":[{"id":"pf-2","external_port_range":"2000:2010"}],"port_forwardings_links":[{"rel":"next","href":"`+cloud.Server.URL+`/never"}]}`)
		}
		return nativeForwardWire(200, `{"port_forwarding":`+nativeForwardRow+`}`)
	})
	created, err := api.Create(ctx, "fip-1", portforwarding.CreateOpts{InternalPortID: "port", InternalIPAddress: "10.0.0.5", InternalPort: 22, ExternalPort: 2222, Protocol: "tcp"}, portforwarding.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "pf-1" && created.ExternalPort == 2222 && created.Description == "") {
		t.Fatal(created, err)
	}
	// internal_port_id, internal_ip_address and protocol are always sent.
	if _, err := api.Create(ctx, "fip-1", portforwarding.CreateOpts{InternalPortRange: "22:30", ExternalPortRange: "2022:2030"}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "fip-1", "pf-1")
	if err != nil || got.InternalIPAddress != "10.0.0.5" {
		t.Fatal(got, err)
	}
	description := ""
	if _, err := api.Update(ctx, "fip-1", "pf-1", portforwarding.UpdateOpts{Description: &description, ExternalPort: 2223}, portforwarding.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "fip-1", "pf-1", portforwarding.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	var rows []*portforwarding.PortForwarding
	for value, err := range api.List(ctx, "fip-1", portforwarding.WithListOptions(portforwarding.ListOpts{Protocol: "tcp", ExternalPort: "2222", Limit: 1}), portforwarding.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "pf-1" && rows[1].ExternalPortRange == "2000:2010") {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "fip-1", "pf-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 8 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[5].query)
	base := "/neutron/v2.0/floatingips/fip-1/port_forwardings"
	want := []nativeForwardCall{
		{http.MethodPost, base, "", `{"port_forwarding":{"external_port":2222,"internal_ip_address":"10.0.0.5","internal_port":22,"internal_port_id":"port","protocol":"tcp","x_extension":1}}`},
		{http.MethodPost, base, "", `{"port_forwarding":{"external_port_range":"2022:2030","internal_ip_address":"","internal_port_id":"","internal_port_range":"22:30","protocol":""}}`},
		{http.MethodGet, base + "/pf-1", "", ""},
		{http.MethodPut, base + "/pf-1", "", `{"port_forwarding":{"description":"","external_port":2223,"x_extension":1}}`},
		{http.MethodPut, base + "/pf-1", "", `{"port_forwarding":{}}`},
		{http.MethodGet, base, calls[5].query, ""},
		{http.MethodGet, "/other/pf", "marker=x", ""},
		{http.MethodDelete, base + "/pf-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"protocol": {"tcp"}, "external_port": {"2222"}, "limit": {"1"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativePortForwardingStrictStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*portforwarding.API) error
	}{
		{"Create", []int{201, 202}, func(api *portforwarding.API) error {
			_, err := api.Create(ctx, "fip-1", portforwarding.CreateOpts{})
			return err
		}},
		{"Get", []int{200}, func(api *portforwarding.API) error { _, err := api.Get(ctx, "fip-1", "pf-1"); return err }},
		{"Update", []int{200}, func(api *portforwarding.API) error {
			_, err := api.Update(ctx, "fip-1", "pf-1", portforwarding.UpdateOpts{})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *portforwarding.API) error { return api.Delete(ctx, "fip-1", "pf-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeForwardCall
				api, _ := nativeForwardAPI(t, &calls, func(*http.Request) *http.Response {
					return nativeForwardWire(code, `{"port_forwarding":{}}`)
				})
				err := call.call(api)
				nativeForwardOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			// An empty object yields a zero value; other keys without the envelope fail.
			`{}`:                       false,
			`{"other":{"id":"pf-1"}}`:  true,
			`{"port_forwarding":[]}`:   true,
			`{"port_forwarding":null}`: false,
			`{"port_forwarding":{"id"`: true,
		} {
			var calls []nativeForwardCall
			api, _ := nativeForwardAPI(t, &calls, func(*http.Request) *http.Response { return nativeForwardWire(200, body) })
			got, err := api.Get(ctx, "fip-1", "pf-1")
			if wantErr {
				nativeForwardOperation(t, err, "Get")
			} else if err != nil || got == nil || got.ID != "" {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"port_forwardings":[]}`}, {204, ""}} {
			var calls []nativeForwardCall
			api, _ := nativeForwardAPI(t, &calls, func(*http.Request) *http.Response { return nativeForwardWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.List(ctx, "fip-1") {
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
		var calls []nativeForwardCall
		api, _ := nativeForwardAPI(t, &calls, func(*http.Request) *http.Response { return nativeForwardWire(201, `{}`) })
		for operation, err := range map[string]error{
			"Create": func() error {
				_, err := api.Create(ctx, "fip-1", portforwarding.CreateOpts{}, portforwarding.WithCreateField("protocol", "udp"))
				return err
			}(),
			"Update": func() error {
				_, err := api.Update(ctx, "fip-1", "pf-1", portforwarding.UpdateOpts{}, nil)
				return err
			}(),
		} {
			if err == nil {
				t.Fatal(operation, "accepted")
			}
			nativeForwardOperation(t, err, operation)
		}
		for _, err := range api.List(ctx, "fip-1", nil) {
			nativeForwardOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
