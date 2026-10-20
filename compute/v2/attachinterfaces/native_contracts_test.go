package attachinterfaces_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/attachinterfaces"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeInterfaceTransport func(*http.Request) (*http.Response, error)

func (transport nativeInterfaceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeInterfaceWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func nativeInterfaceClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	return client
}

func nativeInterfaceOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "attachinterfaces" {
		t.Fatal("generated interface attachment context", err, wrapped)
	}
}

type nativeInterfaceCall struct{ method, path, query, body string }

const nativeInterfaceRow = `{"port_state":"ACTIVE","fixed_ips":[{"subnet_id":"sn","ip_address":"10.0.0.5"}],"port_id":"p1","net_id":"n1","mac_addr":"fa:16:3e:00:00:01"}`

func TestNativeAttachInterfacesRoutesBodiesAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeInterfaceCall
	cloud.Provider.HTTPClient.Transport = nativeInterfaceTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		calls = append(calls, nativeInterfaceCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		switch {
		case req.Method == http.MethodDelete:
			return nativeInterfaceWire(202, ""), nil
		case req.Method == http.MethodGet && strings.HasSuffix(req.URL.Path, "/os-interface"):
			return nativeInterfaceWire(200, `{"interfaceAttachments":[`+nativeInterfaceRow+`,{"port_id":"p2"}],"interfaceAttachments_links":[{"rel":"next","href":"http://other/next"}]}`), nil
		}
		return nativeInterfaceWire(200, `{"interfaceAttachment":`+nativeInterfaceRow+`}`), nil
	})
	api := attachinterfaces.New(nativeInterfaceClient(cloud))
	ctx := context.Background()
	created, err := api.Create(ctx, "s1", attachinterfaces.CreateOpts{NetworkID: "n1", FixedIPs: []attachinterfaces.FixedIP{{SubnetID: "sn", IPAddress: "10.0.0.5"}, {IPAddress: "10.0.0.6"}}}, attachinterfaces.WithCreateField("tag", "eth1"))
	if err != nil || created.PortID != "p1" || created.MACAddr != "fa:16:3e:00:00:01" || created.FixedIPs[0].IPAddress != "10.0.0.5" {
		t.Fatal(created, err)
	}
	if _, err := api.Create(ctx, "s1", attachinterfaces.CreateOpts{}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "s1", "p1")
	if err != nil || got.PortState != "ACTIVE" || got.NetID != "n1" {
		t.Fatal(got, err)
	}
	var ports []string
	for value, err := range api.List(ctx, "s1") {
		if err != nil {
			t.Fatal(err)
		}
		ports = append(ports, value.PortID)
	}
	if !reflect.DeepEqual(ports, []string{"p1", "p2"}) {
		t.Fatal(ports)
	}
	if err := api.Delete(ctx, "s1", "p1"); err != nil {
		t.Fatal(err)
	}
	want := []nativeInterfaceCall{
		// fixed_ips entries omit an empty subnet; the tag extension stays inside the envelope.
		{http.MethodPost, "/nova/v2.1/servers/s1/os-interface", "", `{"interfaceAttachment":{"fixed_ips":[{"ip_address":"10.0.0.5","subnet_id":"sn"},{"ip_address":"10.0.0.6"}],"net_id":"n1","tag":"eth1"}}`},
		// No field is required: an empty attachment asks Nova to choose the network.
		{http.MethodPost, "/nova/v2.1/servers/s1/os-interface", "", `{"interfaceAttachment":{}}`},
		{http.MethodGet, "/nova/v2.1/servers/s1/os-interface/p1", "", ""},
		{http.MethodGet, "/nova/v2.1/servers/s1/os-interface", "", ""},
		{http.MethodDelete, "/nova/v2.1/servers/s1/os-interface/p1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeAttachInterfacesStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*attachinterfaces.API) error
	}{
		{"Create", []int{200}, func(api *attachinterfaces.API) error {
			_, err := api.Create(ctx, "s1", attachinterfaces.CreateOpts{PortID: "p1"})
			return err
		}},
		{"Get", []int{200}, func(api *attachinterfaces.API) error { _, err := api.Get(ctx, "s1", "p1"); return err }},
		{"Delete", []int{202, 204}, func(api *attachinterfaces.API) error { return api.Delete(ctx, "s1", "p1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Provider.HTTPClient.Transport = nativeInterfaceTransport(func(req *http.Request) (*http.Response, error) {
					requests.Add(1)
					return nativeInterfaceWire(code, `{"interfaceAttachment":{"port_id":"p1"}}`), nil
				})
				err := call.call(attachinterfaces.New(nativeInterfaceClient(cloud)))
				nativeInterfaceOperation(t, err, call.name)
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
			{200, `{"interfaceAttachments":[]}`, func(errs []error) bool { return len(errs) == 0 }},
			{204, "", func(errs []error) bool { return len(errs) == 1 && errors.Is(errs[0], io.EOF) }},
		} {
			cloud := testcloud.New(t)
			cloud.Provider.HTTPClient.Transport = nativeInterfaceTransport(func(req *http.Request) (*http.Response, error) {
				return nativeInterfaceWire(tc.code, tc.body), nil
			})
			var errs []error
			for value, err := range attachinterfaces.New(nativeInterfaceClient(cloud)).List(ctx, "s1") {
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
		cloud.Provider.HTTPClient.Transport = nativeInterfaceTransport(func(req *http.Request) (*http.Response, error) {
			requests.Add(1)
			return nativeInterfaceWire(200, `{}`), nil
		})
		api := attachinterfaces.New(nativeInterfaceClient(cloud))
		for name, check := range map[string]func() error{
			"port_id extension": func() error {
				_, err := api.Create(ctx, "s1", attachinterfaces.CreateOpts{}, attachinterfaces.WithCreateField("port_id", "x"))
				return err
			},
			"nil option": func() error { _, err := api.Create(ctx, "s1", attachinterfaces.CreateOpts{}, nil); return err },
		} {
			err := check()
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeInterfaceOperation(t, err, "Create")
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
