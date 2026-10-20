package floatingips_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeFloatingIPTransport func(*http.Request) (*http.Response, error)

func (transport nativeFloatingIPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeFloatingIPWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func nativeFloatingIPClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return client
}

func nativeFloatingIPOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "floatingips" {
		t.Fatal("generated floatingips context", err, wrapped)
	}
}

type nativeFloatingIPCall struct{ method, path, query, body string }

func nativeFloatingIPRecorder(cloud *testcloud.Cloud, calls *[]nativeFloatingIPCall, reply func(*http.Request) *http.Response) {
	cloud.Provider.HTTPClient.Transport = nativeFloatingIPTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeFloatingIPCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
}

const nativeFloatingIPRow = `{"id":"id-1","floating_network_id":"ext","floating_ip_address":"203.0.113.5","port_id":"p1","fixed_ip_address":"10.0.0.5","router_id":"r1","status":"ACTIVE","created_at":"2026-10-10T01:02:03Z"}`

func TestNativeFloatingIPRoutesBodiesPagingAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeFloatingIPCall
	nativeFloatingIPRecorder(cloud, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeFloatingIPWire(204, "")
		case req.Method == http.MethodPost:
			return nativeFloatingIPWire(201, `{"floatingip":`+nativeFloatingIPRow+`}`)
		case req.URL.Path == "/neutron/v2.0/floatingips" && req.URL.Query().Get("marker") == "":
			return nativeFloatingIPWire(200, `{"floatingips":[`+nativeFloatingIPRow+`],"floatingips_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/floatingips?marker=x"}]}`)
		case req.URL.Path == "/other/floatingips":
			return nativeFloatingIPWire(200, `{"floatingips":[{"id":"id-2","created_at":"2026-10-10T01:02:03"}]}`)
		}
		return nativeFloatingIPWire(200, `{"floatingip":`+nativeFloatingIPRow+`}`)
	})
	api := floatingips.New(nativeFloatingIPClient(cloud))
	ctx := context.Background()
	created, err := api.Create(ctx, floatingips.CreateOpts{FloatingNetworkID: "ext", PortID: "p1"}, floatingips.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "id-1" && created.FloatingIP == "203.0.113.5" && created.RouterID == "r1") {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "id-1")
	if err != nil || !(got.Status == "ACTIVE" && got.CreatedAt.Year() == 2026) {
		t.Fatal(got, err)
	}
	var rows []*floatingips.FloatingIP
	for value, err := range api.List(ctx, floatingips.WithListOptions(floatingips.ListOpts{FloatingNetworkID: "ext", PortID: "p1", Status: "ACTIVE", Limit: 1}), floatingips.WithListQuery("extra", "1")) {
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
	want := []nativeFloatingIPCall{
		{http.MethodPost, "/neutron/v2.0/floatingips", "", `{"floatingip":{"floating_network_id":"ext","port_id":"p1","x_extension":1}}`},
		{http.MethodGet, "/neutron/v2.0/floatingips/id-1", "", ""},
		{http.MethodGet, "/neutron/v2.0/floatingips", calls[2].query, ""},
		// The next href is followed as received.
		{http.MethodGet, "/other/floatingips", "marker=x", ""},
		{http.MethodDelete, "/neutron/v2.0/floatingips/id-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"floating_network_id": {"ext"}, "port_id": {"p1"}, "status": {"ACTIVE"}, "limit": {"1"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeFloatingIPStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*floatingips.API) error
	}{
		{"Create", []int{201, 202}, func(api *floatingips.API) error {
			_, err := api.Create(ctx, floatingips.CreateOpts{FloatingNetworkID: "ext"})
			return err
		}},
		{"Get", []int{200}, func(api *floatingips.API) error { _, err := api.Get(ctx, "id-1"); return err }},
		{"Delete", []int{202, 204}, func(api *floatingips.API) error { return api.Delete(ctx, "id-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls []nativeFloatingIPCall
				nativeFloatingIPRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeFloatingIPWire(code, `{"floatingip":{}}`) })
				err := call.call(floatingips.New(nativeFloatingIPClient(cloud)))
				nativeFloatingIPOperation(t, err, call.name)
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
		}{{404, `{}`}, {200, `{"floatingips":[]}`}, {204, ""}} {
			cloud := testcloud.New(t)
			var calls []nativeFloatingIPCall
			nativeFloatingIPRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeFloatingIPWire(tc.code, tc.body) })
			var errs []error
			for _, err := range floatingips.New(nativeFloatingIPClient(cloud)).List(ctx) {
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
		var calls []nativeFloatingIPCall
		nativeFloatingIPRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeFloatingIPWire(201, `{}`) })
		api := floatingips.New(nativeFloatingIPClient(cloud))
		for name, check := range map[string]func() error{
			"missing network": func() error { _, err := api.Create(ctx, floatingips.CreateOpts{PortID: "p"}); return err },
			"network extension": func() error {
				_, err := api.Create(ctx, floatingips.CreateOpts{FloatingNetworkID: "e"}, floatingips.WithCreateField("floating_network_id", "x"))
				return err
			},
			"nil option": func() error {
				_, err := api.Create(ctx, floatingips.CreateOpts{FloatingNetworkID: "e"}, nil)
				return err
			},
		} {
			err := check()
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeFloatingIPOperation(t, err, "Create")
		}
		for _, err := range api.List(ctx, nil) {
			nativeFloatingIPOperation(t, err, "List")
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

func TestNativeFloatingIPCreateFullOptionsReplacementAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeFloatingIPCall
	nativeFloatingIPRecorder(cloud, &calls, func(*http.Request) *http.Response {
		return nativeFloatingIPWire(202, `{"floatingip":{"id":"f1","floating_network_id":"ext","floating_ip_address":"203.0.113.5","port_id":null,"fixed_ip_address":null,"router_id":null,"status":"DOWN","created_at":"2026-10-10T01:02:03","unknown":{"x":1}}}`)
	})
	api := floatingips.New(nativeFloatingIPClient(cloud))
	// WithCreateOptions replaces the positional options entirely.
	created, err := api.Create(context.Background(), floatingips.CreateOpts{FloatingNetworkID: "ignored", Description: "ignored"},
		floatingips.WithCreateOptions(floatingips.CreateOpts{FloatingNetworkID: "ext", Description: "d", FloatingIP: "203.0.113.5", PortID: "p1", FixedIP: "10.0.0.5", SubnetID: "sn", TenantID: "t", ProjectID: "p"}),
		floatingips.WithCreateField("dns_name", "vm"))
	if err != nil || created.ID != "f1" || created.PortID != "" || created.FixedIP != "" || created.Status != "DOWN" || created.CreatedAt.Second() != 3 {
		t.Fatal(created, err)
	}
	want := []nativeFloatingIPCall{{http.MethodPost, "/neutron/v2.0/floatingips", "", `{"floatingip":{"description":"d","dns_name":"vm","fixed_ip_address":"10.0.0.5","floating_ip_address":"203.0.113.5","floating_network_id":"ext","port_id":"p1","project_id":"p","subnet_id":"sn","tenant_id":"t"}}`}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	t.Run("invalid JSON and envelope errors keep context", func(t *testing.T) {
		for _, body := range []string{`{`, `{"floatingip":[]}`} {
			cloud := testcloud.New(t)
			var calls []nativeFloatingIPCall
			nativeFloatingIPRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeFloatingIPWire(201, body) })
			_, err := floatingips.New(nativeFloatingIPClient(cloud)).Create(context.Background(), floatingips.CreateOpts{FloatingNetworkID: "ext"})
			nativeFloatingIPOperation(t, err, "Create")
		}
	})
}
