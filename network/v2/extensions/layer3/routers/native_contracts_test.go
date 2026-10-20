package routers_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeRouterTransport func(*http.Request) (*http.Response, error)

func (transport nativeRouterTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeRouterWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func nativeRouterClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return client
}

func nativeRouterOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "routers" {
		t.Fatal("generated routers context", err, wrapped)
	}
}

type nativeRouterCall struct{ method, path, query, body string }

func nativeRouterRecorder(cloud *testcloud.Cloud, calls *[]nativeRouterCall, reply func(*http.Request) *http.Response) {
	cloud.Provider.HTTPClient.Transport = nativeRouterTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeRouterCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
}

const nativeRouterRow = `{"id":"id-1","name":"edge","status":"ACTIVE","admin_state_up":true,"distributed":false,"external_gateway_info":{"network_id":"ext"},"routes":[{"destination":"10.1.0.0/24","nexthop":"10.0.0.9"}],"created_at":"2026-10-10T01:02:03Z"}`

func TestNativeRouterRoutesBodiesPagingAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeRouterCall
	nativeRouterRecorder(cloud, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeRouterWire(204, "")
		case req.Method == http.MethodPost:
			return nativeRouterWire(201, `{"router":`+nativeRouterRow+`}`)
		case req.URL.Path == "/neutron/v2.0/routers" && req.URL.Query().Get("marker") == "":
			return nativeRouterWire(200, `{"routers":[`+nativeRouterRow+`],"routers_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/routers?marker=x"}]}`)
		case req.URL.Path == "/other/routers":
			return nativeRouterWire(200, `{"routers":[{"id":"id-2","external_gateway_info":null,"created_at":"2026-10-10T01:02:03"}]}`)
		}
		return nativeRouterWire(200, `{"router":`+nativeRouterRow+`}`)
	})
	api := routers.New(nativeRouterClient(cloud))
	ctx := context.Background()
	created, err := api.Create(ctx, routers.CreateOpts{Name: "edge", AdminStateUp: gophercloud.Enabled, Distributed: gophercloud.Disabled, GatewayInfo: &routers.GatewayInfo{NetworkID: "ext", EnableSNAT: gophercloud.Enabled}, AvailabilityZoneHints: []string{"az1"}}, routers.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "id-1" && created.GatewayInfo.NetworkID == "ext" && created.Routes[0].NextHop == "10.0.0.9") {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "id-1")
	if err != nil || !(got.Status == "ACTIVE" && got.CreatedAt.Year() == 2026) {
		t.Fatal(got, err)
	}
	var rows []*routers.Router
	for value, err := range api.List(ctx, routers.WithListOptions(routers.ListOpts{Name: "edge", Distributed: gophercloud.Disabled, Limit: 1}), routers.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "id-1" && rows[1].ID == "id-2" && rows[1].GatewayInfo.NetworkID == "" && rows[1].CreatedAt.Second() == 3) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "id-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 5 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[2].query)
	want := []nativeRouterCall{
		{http.MethodPost, "/neutron/v2.0/routers", "", `{"router":{"admin_state_up":true,"availability_zone_hints":["az1"],"distributed":false,"external_gateway_info":{"enable_snat":true,"network_id":"ext"},"name":"edge","x_extension":1}}`},
		{http.MethodGet, "/neutron/v2.0/routers/id-1", "", ""},
		{http.MethodGet, "/neutron/v2.0/routers", calls[2].query, ""},
		// The next href is followed as received.
		{http.MethodGet, "/other/routers", "marker=x", ""},
		{http.MethodDelete, "/neutron/v2.0/routers/id-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"name": {"edge"}, "distributed": {"false"}, "limit": {"1"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeRouterStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*routers.API) error
	}{
		{"Create", []int{201, 202}, func(api *routers.API) error { _, err := api.Create(ctx, routers.CreateOpts{}); return err }},
		{"Get", []int{200}, func(api *routers.API) error { _, err := api.Get(ctx, "id-1"); return err }},
		{"Delete", []int{202, 204}, func(api *routers.API) error { return api.Delete(ctx, "id-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls []nativeRouterCall
				nativeRouterRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeRouterWire(code, `{"router":{}}`) })
				err := call.call(routers.New(nativeRouterClient(cloud)))
				nativeRouterOperation(t, err, call.name)
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
		}{{404, `{}`}, {200, `{"routers":[]}`}, {204, ""}} {
			cloud := testcloud.New(t)
			var calls []nativeRouterCall
			nativeRouterRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeRouterWire(tc.code, tc.body) })
			var errs []error
			for _, err := range routers.New(nativeRouterClient(cloud)).List(ctx) {
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
		var calls []nativeRouterCall
		nativeRouterRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeRouterWire(201, `{}`) })
		api := routers.New(nativeRouterClient(cloud))
		for name, check := range map[string]func() error{
			"name extension": func() error {
				_, err := api.Create(ctx, routers.CreateOpts{}, routers.WithCreateField("name", "x"))
				return err
			},
			"nil option": func() error { _, err := api.Create(ctx, routers.CreateOpts{}, nil); return err },
		} {
			err := check()
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeRouterOperation(t, err, "Create")
		}
		for _, err := range api.List(ctx, nil) {
			nativeRouterOperation(t, err, "List")
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

func TestNativeRouterInterfacesAndExternalGateways(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls []nativeRouterCall
	nativeRouterRecorder(cloud, &calls, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "_interface") {
			return nativeRouterWire(200, `{"id":"r1","subnet_id":"sn","port_id":"p1","tenant_id":"t"}`)
		}
		return nativeRouterWire(200, `{"router":{"id":"r1","external_gateway_info":{"network_id":"ext","enable_snat":true,"external_fixed_ips":[{"ip_address":"203.0.113.2","subnet_id":"esn"}]}}}`)
	})
	api := routers.New(nativeRouterClient(cloud))
	info, err := api.AddInterface(ctx, "r1", routers.AddInterfaceOpts{SubnetID: "sn"}, routers.WithAddInterfaceField("x_extension", 1))
	if err != nil || info.PortID != "p1" || info.SubnetID != "sn" {
		t.Fatal(info, err)
	}
	if _, err := api.RemoveInterface(ctx, "r1", routers.RemoveInterfaceOpts{SubnetID: "sn", PortID: "p1"}); err != nil {
		t.Fatal(err)
	}
	gateways := []routers.GatewayInfo{{NetworkID: "ext", EnableSNAT: gophercloud.Disabled, ExternalFixedIPs: []routers.ExternalFixedIP{{SubnetID: "esn"}}}}
	router, err := api.AddExternalGateways(ctx, "r1", routers.AddExternalGatewaysOpts{ExternalGateways: gateways}, routers.WithAddExternalGatewaysField("x_extension", 1))
	if err != nil || router.GatewayInfo.NetworkID != "ext" || router.GatewayInfo.ExternalFixedIPs[0].IPAddress != "203.0.113.2" {
		t.Fatal(router, err)
	}
	if _, err := api.UpdateExternalGateways(ctx, "r1", routers.UpdateExternalGatewaysOpts{ExternalGateways: gateways}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.RemoveExternalGateways(ctx, "r1", routers.RemoveExternalGatewaysOpts{ExternalGateways: []routers.GatewayInfo{}}); err != nil {
		t.Fatal(err)
	}
	gatewayBody := `{"router":{"external_gateways":[{"enable_snat":false,"external_fixed_ips":[{"subnet_id":"esn"}],"network_id":"ext"}]`
	want := []nativeRouterCall{
		// The interface body has no envelope, so an extension sits beside subnet_id.
		{http.MethodPut, "/neutron/v2.0/routers/r1/add_router_interface", "", `{"subnet_id":"sn","x_extension":1}`},
		{http.MethodPut, "/neutron/v2.0/routers/r1/remove_router_interface", "", `{"port_id":"p1","subnet_id":"sn"}`},
		{http.MethodPut, "/neutron/v2.0/routers/r1/add_external_gateways", "", gatewayBody[:len(gatewayBody)] + `,"x_extension":1}}`},
		{http.MethodPut, "/neutron/v2.0/routers/r1/update_external_gateways", "", gatewayBody + `}}`},
		// An empty, non-nil gateway list is sent as [].
		{http.MethodPut, "/neutron/v2.0/routers/r1/remove_external_gateways", "", `{"router":{"external_gateways":[]}}`},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	for _, call := range []struct {
		name string
		call func(*routers.API) error
	}{
		{"AddInterface", func(api *routers.API) error {
			_, err := api.AddInterface(ctx, "r1", routers.AddInterfaceOpts{PortID: "p"})
			return err
		}},
		{"RemoveInterface", func(api *routers.API) error {
			_, err := api.RemoveInterface(ctx, "r1", routers.RemoveInterfaceOpts{PortID: "p"})
			return err
		}},
		{"AddExternalGateways", func(api *routers.API) error {
			_, err := api.AddExternalGateways(ctx, "r1", routers.AddExternalGatewaysOpts{ExternalGateways: gateways})
			return err
		}},
		{"UpdateExternalGateways", func(api *routers.API) error {
			_, err := api.UpdateExternalGateways(ctx, "r1", routers.UpdateExternalGatewaysOpts{ExternalGateways: gateways})
			return err
		}},
		{"RemoveExternalGateways", func(api *routers.API) error {
			_, err := api.RemoveExternalGateways(ctx, "r1", routers.RemoveExternalGatewaysOpts{ExternalGateways: gateways})
			return err
		}},
	} {
		for _, code := range []int{201, 202, 204, 404} {
			cloud := testcloud.New(t)
			var calls []nativeRouterCall
			nativeRouterRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeRouterWire(code, `{"router":{}}`) })
			err := call.call(routers.New(nativeRouterClient(cloud)))
			nativeRouterOperation(t, err, call.name)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || len(calls) != 1 {
				t.Fatal(call.name, err)
			}
		}
	}
	t.Run("preflight", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls []nativeRouterCall
		nativeRouterRecorder(cloud, &calls, func(*http.Request) *http.Response { return nativeRouterWire(200, `{}`) })
		api := routers.New(nativeRouterClient(cloud))
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"add interface neither": {"AddInterface", func() error { _, err := api.AddInterface(ctx, "r1", routers.AddInterfaceOpts{}); return err }()},
			"add interface both": {"AddInterface", func() error {
				_, err := api.AddInterface(ctx, "r1", routers.AddInterfaceOpts{SubnetID: "s", PortID: "p"})
				return err
			}()},
			"remove interface neither": {"RemoveInterface", func() error { _, err := api.RemoveInterface(ctx, "r1", routers.RemoveInterfaceOpts{}); return err }()},
			"nil gateways": {"AddExternalGateways", func() error {
				_, err := api.AddExternalGateways(ctx, "r1", routers.AddExternalGatewaysOpts{})
				return err
			}()},
			"gateways extension": {"UpdateExternalGateways", func() error {
				_, err := api.UpdateExternalGateways(ctx, "r1", routers.UpdateExternalGatewaysOpts{ExternalGateways: gateways}, routers.WithUpdateExternalGatewaysField("external_gateways", nil))
				return err
			}()},
			"subnet extension": {"AddInterface", func() error {
				_, err := api.AddInterface(ctx, "r1", routers.AddInterfaceOpts{PortID: "p"}, routers.WithAddInterfaceField("subnet_id", "s"))
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeRouterOperation(t, check.err, check.operation)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
