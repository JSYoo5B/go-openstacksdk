package bgpvpns_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeBGPVPNTransport func(*http.Request) (*http.Response, error)

func (transport nativeBGPVPNTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeBGPVPNWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeBGPVPNCall struct{ method, path, query, body string }

func nativeBGPVPNAPI(t *testing.T, calls *[]nativeBGPVPNCall, reply func(*http.Request) *http.Response) (*bgpvpns.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeBGPVPNTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeBGPVPNCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return bgpvpns.New(client), cloud
}

func nativeBGPVPNOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "bgpvpns" {
		t.Fatal("generated bgpvpns context", err, wrapped)
	}
}

func nativeBGPVPNPtr[T any](value T) *T { return &value }

const (
	nativeBGPVPNRow        = `{"id":"vpn-1","name":"vpn","type":"l3","shared":false,"route_targets":["64512:1"],"networks":["n-1"],"routers":[],"ports":null,"local_pref":null,"vni":0,"project_id":"p"}`
	nativeBGPVPNNetworkRow = `{"id":"na-1","network_id":"n-1","project_id":"p"}`
	nativeBGPVPNRouterRow  = `{"id":"ra-1","router_id":"r-1","advertise_extra_routes":true,"project_id":"p"}`
	nativeBGPVPNPortRow    = `{"id":"pa-1","port_id":"pt-1","advertise_fixed_ips":false,"routes":[{"type":"prefix","prefix":"10.0.0.0/24","local_pref":100}],"project_id":"p"}`
)

func TestNativeBGPVPNRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeBGPVPNCall
	api, _ := nativeBGPVPNAPI(t, &calls, func(req *http.Request) *http.Response {
		path, marker := req.URL.Path, req.URL.Query().Get("marker")
		switch {
		case req.Method == http.MethodDelete:
			return nativeBGPVPNWire(204, "")
		case strings.Contains(path, "/network_associations"):
			if req.Method == http.MethodGet && strings.HasSuffix(path, "/network_associations") {
				if marker != "" {
					return nativeBGPVPNWire(200, `{"network_associations":[]}`)
				}
				return nativeBGPVPNWire(200, `{"network_associations":[`+nativeBGPVPNNetworkRow+`]}`)
			}
			if req.Method == http.MethodPost {
				return nativeBGPVPNWire(201, `{"network_association":`+nativeBGPVPNNetworkRow+`}`)
			}
			return nativeBGPVPNWire(200, `{"network_association":`+nativeBGPVPNNetworkRow+`}`)
		case strings.Contains(path, "/router_associations"):
			if req.Method == http.MethodGet && strings.HasSuffix(path, "/router_associations") {
				return nativeBGPVPNWire(200, `{"router_associations":[`+nativeBGPVPNRouterRow+`]}`)
			}
			if req.Method == http.MethodPost {
				return nativeBGPVPNWire(201, `{"router_association":`+nativeBGPVPNRouterRow+`}`)
			}
			return nativeBGPVPNWire(200, `{"router_association":`+nativeBGPVPNRouterRow+`}`)
		case strings.Contains(path, "/port_associations"):
			if req.Method == http.MethodGet && strings.HasSuffix(path, "/port_associations") {
				return nativeBGPVPNWire(200, `{"port_associations":[`+nativeBGPVPNPortRow+`]}`)
			}
			if req.Method == http.MethodPost {
				return nativeBGPVPNWire(201, `{"port_association":`+nativeBGPVPNPortRow+`}`)
			}
			return nativeBGPVPNWire(200, `{"port_association":`+nativeBGPVPNPortRow+`}`)
		case req.Method == http.MethodPost:
			return nativeBGPVPNWire(201, `{"bgpvpn":`+nativeBGPVPNRow+`}`)
		case req.Method == http.MethodPut:
			return nativeBGPVPNWire(200, `{"bgpvpn":{"id":"vpn-1","local_pref":0}}`)
		case path == "/neutron/v2.0/bgpvpn/bgpvpns":
			// Paging follows only the marker built from the last ID, and only when the URL has limit.
			switch marker {
			case "":
				return nativeBGPVPNWire(200, `{"bgpvpns":[`+nativeBGPVPNRow+`],"bgpvpns_links":[{"rel":"next","href":"/never"}],"links":{"next":"/never"}}`)
			case "vpn-1":
				return nativeBGPVPNWire(200, `{"bgpvpns":[{"id":"vpn-2","local_pref":50}]}`)
			}
			return nativeBGPVPNWire(200, `{"bgpvpns":[]}`)
		}
		return nativeBGPVPNWire(200, `{"bgpvpn":`+nativeBGPVPNRow+`}`)
	})
	created, err := api.Create(ctx, bgpvpns.CreateOpts{Name: "vpn", Type: "l3", RouteTargets: []string{"64512:1"}, LocalPref: 100}, bgpvpns.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "vpn-1" && created.LocalPref == nil && created.Ports == nil && reflect.DeepEqual(created.Routers, []string{})) {
		t.Fatal(created, err)
	}
	// Every create field is omitempty, so a zero LocalPref or VNI cannot be sent.
	if _, err := api.Create(ctx, bgpvpns.CreateOpts{}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "vpn-1")
	if err != nil || !reflect.DeepEqual(got.Networks, []string{"n-1"}) {
		t.Fatal(got, err)
	}
	updated, err := api.Update(ctx, "vpn-1", bgpvpns.UpdateOpts{Name: nativeBGPVPNPtr(""), RouteTargets: &[]string{}, LocalPref: nativeBGPVPNPtr(0)}, bgpvpns.WithUpdateField("x_extension", 1))
	if err != nil || updated.LocalPref == nil || *updated.LocalPref != 0 {
		t.Fatal(updated, err)
	}
	if _, err := api.Update(ctx, "vpn-1", bgpvpns.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for value, err := range api.List(ctx, bgpvpns.WithListOptions(bgpvpns.ListOpts{Networks: []string{"n-1", "n-2"}, Fields: []string{"id"}, Limit: 1}), bgpvpns.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	// Without limit the first page is the only page.
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"vpn-1", "vpn-2", "vpn-1"}) {
		t.Fatal(ids)
	}
	network, err := api.CreateNetworkAssociation(ctx, "vpn-1", bgpvpns.CreateNetworkAssociationOpts{NetworkID: "n-1"}, bgpvpns.WithCreateNetworkAssociationField("x_extension", 1))
	if err != nil || network.ID != "na-1" || network.NetworkID != "n-1" {
		t.Fatal(network, err)
	}
	if got, err := api.GetNetworkAssociation(ctx, "vpn-1", "na-1"); err != nil || got.ProjectID != "p" {
		t.Fatal(got, err)
	}
	var associations []string
	for value, err := range api.ListNetworkAssociations(ctx, "vpn-1", bgpvpns.WithListNetworkAssociationsOptions(bgpvpns.ListNetworkAssociationsOpts{Limit: 1}), bgpvpns.WithListNetworkAssociationsQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		associations = append(associations, value.ID)
	}
	if err := api.DeleteNetworkAssociation(ctx, "vpn-1", "na-1"); err != nil {
		t.Fatal(err)
	}
	router, err := api.CreateRouterAssociation(ctx, "vpn-1", bgpvpns.CreateRouterAssociationOpts{RouterID: "r-1", AdvertiseExtraRoutes: nativeBGPVPNPtr(false)})
	if err != nil || router.RouterID != "r-1" || !router.AdvertiseExtraRoutes {
		t.Fatal(router, err)
	}
	if _, err := api.GetRouterAssociation(ctx, "vpn-1", "ra-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := api.UpdateRouterAssociation(ctx, "vpn-1", "ra-1", bgpvpns.UpdateRouterAssociationOpts{AdvertiseExtraRoutes: nativeBGPVPNPtr(true)}, bgpvpns.WithUpdateRouterAssociationField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	if _, err := api.UpdateRouterAssociation(ctx, "vpn-1", "ra-1", bgpvpns.UpdateRouterAssociationOpts{}); err != nil {
		t.Fatal(err)
	}
	for value, err := range api.ListRouterAssociations(ctx, "vpn-1", bgpvpns.WithListRouterAssociationsOptions(bgpvpns.ListRouterAssociationsOpts{Fields: []string{"id"}})) {
		if err != nil {
			t.Fatal(err)
		}
		associations = append(associations, value.ID)
	}
	if err := api.DeleteRouterAssociation(ctx, "vpn-1", "ra-1"); err != nil {
		t.Fatal(err)
	}
	port, err := api.CreatePortAssociation(ctx, "vpn-1", bgpvpns.CreatePortAssociationOpts{PortID: "pt-1", AdvertiseFixedIPs: nativeBGPVPNPtr(false), Routes: []bgpvpns.PortRoutes{
		{Type: "prefix", Prefix: "10.0.0.0/24", LocalPref: nativeBGPVPNPtr(100)},
		{Type: "bgpvpn", BGPVPNID: "vpn-2"},
	}}, bgpvpns.WithCreatePortAssociationField("x_extension", 1))
	if err != nil || port.PortID != "pt-1" || len(port.Routes) != 1 || *port.Routes[0].LocalPref != 100 {
		t.Fatal(port, err)
	}
	if _, err := api.GetPortAssociation(ctx, "vpn-1", "pa-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := api.UpdatePortAssociation(ctx, "vpn-1", "pa-1", bgpvpns.UpdatePortAssociationOpts{Routes: &[]bgpvpns.PortRoutes{}}, bgpvpns.WithUpdatePortAssociationField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	for value, err := range api.ListPortAssociations(ctx, "vpn-1") {
		if err != nil {
			t.Fatal(err)
		}
		associations = append(associations, value.ID)
	}
	if err := api.DeletePortAssociation(ctx, "vpn-1", "pa-1"); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "vpn-1"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(associations, []string{"na-1", "ra-1", "pa-1"}) {
		t.Fatal(associations)
	}
	base := "/neutron/v2.0/bgpvpn/bgpvpns"
	want := []nativeBGPVPNCall{
		{http.MethodPost, base, "", `{"bgpvpn":{"local_pref":100,"name":"vpn","route_targets":["64512:1"],"type":"l3","x_extension":1}}`},
		{http.MethodPost, base, "", `{"bgpvpn":{}}`},
		{http.MethodGet, base + "/vpn-1", "", ""},
		// Pointer fields send an empty name, an empty route target list and a zero local_pref.
		{http.MethodPut, base + "/vpn-1", "", `{"bgpvpn":{"local_pref":0,"name":"","route_targets":[],"x_extension":1}}`},
		{http.MethodPut, base + "/vpn-1", "", `{"bgpvpn":{}}`},
		{http.MethodGet, base, "extra=1&fields=id&limit=1&networks=n-1&networks=n-2", ""},
		{http.MethodGet, base, "extra=1&fields=id&limit=1&marker=vpn-1&networks=n-1&networks=n-2", ""},
		{http.MethodGet, base, "extra=1&fields=id&limit=1&marker=vpn-2&networks=n-1&networks=n-2", ""},
		{http.MethodGet, base, "", ""},
		{http.MethodPost, base + "/vpn-1/network_associations", "", `{"network_association":{"network_id":"n-1","x_extension":1}}`},
		{http.MethodGet, base + "/vpn-1/network_associations/na-1", "", ""},
		{http.MethodGet, base + "/vpn-1/network_associations", "extra=1&limit=1", ""},
		{http.MethodGet, base + "/vpn-1/network_associations", "extra=1&limit=1&marker=na-1", ""},
		{http.MethodDelete, base + "/vpn-1/network_associations/na-1", "", ""},
		{http.MethodPost, base + "/vpn-1/router_associations", "", `{"router_association":{"advertise_extra_routes":false,"router_id":"r-1"}}`},
		{http.MethodGet, base + "/vpn-1/router_associations/ra-1", "", ""},
		{http.MethodPut, base + "/vpn-1/router_associations/ra-1", "", `{"router_association":{"advertise_extra_routes":true,"x_extension":1}}`},
		{http.MethodPut, base + "/vpn-1/router_associations/ra-1", "", `{"router_association":{}}`},
		{http.MethodGet, base + "/vpn-1/router_associations", "fields=id", ""},
		{http.MethodDelete, base + "/vpn-1/router_associations/ra-1", "", ""},
		{http.MethodPost, base + "/vpn-1/port_associations", "", `{"port_association":{"advertise_fixed_ips":false,"port_id":"pt-1","routes":[{"local_pref":100,"prefix":"10.0.0.0/24","type":"prefix"},{"bgpvpn_id":"vpn-2","type":"bgpvpn"}],"x_extension":1}}`},
		{http.MethodGet, base + "/vpn-1/port_associations/pa-1", "", ""},
		{http.MethodPut, base + "/vpn-1/port_associations/pa-1", "", `{"port_association":{"routes":[],"x_extension":1}}`},
		{http.MethodGet, base + "/vpn-1/port_associations", "", ""},
		{http.MethodDelete, base + "/vpn-1/port_associations/pa-1", "", ""},
		{http.MethodDelete, base + "/vpn-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeBGPVPNStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*bgpvpns.API) error
	}{
		{"Create", []int{201, 202}, func(api *bgpvpns.API) error { _, err := api.Create(ctx, bgpvpns.CreateOpts{}); return err }},
		{"Get", []int{200}, func(api *bgpvpns.API) error { _, err := api.Get(ctx, "vpn-1"); return err }},
		{"Update", []int{200}, func(api *bgpvpns.API) error { _, err := api.Update(ctx, "vpn-1", bgpvpns.UpdateOpts{}); return err }},
		{"Delete", []int{202, 204}, func(api *bgpvpns.API) error { return api.Delete(ctx, "vpn-1") }},
		{"CreateNetworkAssociation", []int{201, 202}, func(api *bgpvpns.API) error {
			_, err := api.CreateNetworkAssociation(ctx, "vpn-1", bgpvpns.CreateNetworkAssociationOpts{NetworkID: "n-1"})
			return err
		}},
		{"GetNetworkAssociation", []int{200}, func(api *bgpvpns.API) error {
			_, err := api.GetNetworkAssociation(ctx, "vpn-1", "na-1")
			return err
		}},
		{"DeleteNetworkAssociation", []int{202, 204}, func(api *bgpvpns.API) error {
			return api.DeleteNetworkAssociation(ctx, "vpn-1", "na-1")
		}},
		{"CreateRouterAssociation", []int{201, 202}, func(api *bgpvpns.API) error {
			_, err := api.CreateRouterAssociation(ctx, "vpn-1", bgpvpns.CreateRouterAssociationOpts{RouterID: "r-1"})
			return err
		}},
		{"GetRouterAssociation", []int{200}, func(api *bgpvpns.API) error {
			_, err := api.GetRouterAssociation(ctx, "vpn-1", "ra-1")
			return err
		}},
		{"UpdateRouterAssociation", []int{200}, func(api *bgpvpns.API) error {
			_, err := api.UpdateRouterAssociation(ctx, "vpn-1", "ra-1", bgpvpns.UpdateRouterAssociationOpts{})
			return err
		}},
		{"DeleteRouterAssociation", []int{202, 204}, func(api *bgpvpns.API) error {
			return api.DeleteRouterAssociation(ctx, "vpn-1", "ra-1")
		}},
		{"CreatePortAssociation", []int{201, 202}, func(api *bgpvpns.API) error {
			_, err := api.CreatePortAssociation(ctx, "vpn-1", bgpvpns.CreatePortAssociationOpts{PortID: "pt-1"})
			return err
		}},
		{"GetPortAssociation", []int{200}, func(api *bgpvpns.API) error {
			_, err := api.GetPortAssociation(ctx, "vpn-1", "pa-1")
			return err
		}},
		{"UpdatePortAssociation", []int{200}, func(api *bgpvpns.API) error {
			_, err := api.UpdatePortAssociation(ctx, "vpn-1", "pa-1", bgpvpns.UpdatePortAssociationOpts{})
			return err
		}},
		{"DeletePortAssociation", []int{202, 204}, func(api *bgpvpns.API) error {
			return api.DeletePortAssociation(ctx, "vpn-1", "pa-1")
		}},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeBGPVPNCall
				api, _ := nativeBGPVPNAPI(t, &calls, func(*http.Request) *http.Response { return nativeBGPVPNWire(code, `{}`) })
				err := call.call(api)
				nativeBGPVPNOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope decode", func(t *testing.T) {
		gets := map[string]func(*bgpvpns.API) (string, error){
			"Get": func(api *bgpvpns.API) (string, error) {
				v, err := api.Get(ctx, "vpn-1")
				return v.ID, err
			},
			"GetNetworkAssociation": func(api *bgpvpns.API) (string, error) {
				v, err := api.GetNetworkAssociation(ctx, "vpn-1", "na-1")
				return v.ID, err
			},
			"GetRouterAssociation": func(api *bgpvpns.API) (string, error) {
				v, err := api.GetRouterAssociation(ctx, "vpn-1", "ra-1")
				return v.ID, err
			},
			"GetPortAssociation": func(api *bgpvpns.API) (string, error) {
				v, err := api.GetPortAssociation(ctx, "vpn-1", "pa-1")
				return v.ID, err
			},
		}
		for name, get := range gets {
			for body, wantErr := range map[string]bool{
				`{}`:                   false,
				`{"other":{"id":"x"}}`: true,
				`[]`:                   true,
			} {
				var calls []nativeBGPVPNCall
				api, _ := nativeBGPVPNAPI(t, &calls, func(*http.Request) *http.Response { return nativeBGPVPNWire(200, body) })
				id, err := get(api)
				if wantErr {
					nativeBGPVPNOperation(t, err, name)
				} else if err != nil || id != "" {
					t.Fatal(name, body, id, err)
				}
			}
		}
	})
	lists := map[string]func(*bgpvpns.API) []error{
		"List": func(api *bgpvpns.API) (errs []error) {
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			return errs
		},
		"ListNetworkAssociations": func(api *bgpvpns.API) (errs []error) {
			for _, err := range api.ListNetworkAssociations(ctx, "vpn-1") {
				errs = append(errs, err)
			}
			return errs
		},
		"ListRouterAssociations": func(api *bgpvpns.API) (errs []error) {
			for _, err := range api.ListRouterAssociations(ctx, "vpn-1") {
				errs = append(errs, err)
			}
			return errs
		},
		"ListPortAssociations": func(api *bgpvpns.API) (errs []error) {
			for _, err := range api.ListPortAssociations(ctx, "vpn-1") {
				errs = append(errs, err)
			}
			return errs
		},
	}
	t.Run("list pagers status, empty page and bodyless 204", func(t *testing.T) {
		for name, list := range lists {
			for _, tc := range []struct {
				code int
				body string
			}{{404, `{}`}, {200, `{"bgpvpns":[],"network_associations":[],"router_associations":[],"port_associations":[]}`}, {204, ""}} {
				var calls []nativeBGPVPNCall
				api, _ := nativeBGPVPNAPI(t, &calls, func(*http.Request) *http.Response { return nativeBGPVPNWire(tc.code, tc.body) })
				errs := list(api)
				var native gophercloud.ErrUnexpectedResponseCode
				switch {
				case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
				case tc.code == 200 && len(errs) == 0:
				case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
				default:
					t.Fatal(name, tc.code, errs)
				}
				if len(calls) != 1 {
					t.Fatal(name, calls)
				}
			}
		}
	})
	t.Run("204 without content type", func(t *testing.T) {
		// Only the BGP VPN page checks 204 in IsEmpty; association pages try to decode the raw bytes.
		for name, list := range lists {
			var calls []nativeBGPVPNCall
			api, _ := nativeBGPVPNAPI(t, &calls, func(*http.Request) *http.Response {
				return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}
			})
			errs := list(api)
			if (name == "List") != (len(errs) == 0) || len(calls) != 1 {
				t.Fatal(name, errs, calls)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeBGPVPNCall
		api, _ := nativeBGPVPNAPI(t, &calls, func(*http.Request) *http.Response { return nativeBGPVPNWire(201, `{}`) })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create extension": {"Create", func() error {
				_, err := api.Create(ctx, bgpvpns.CreateOpts{}, bgpvpns.WithCreateField("name", "x"))
				return err
			}()},
			"create nil option": {"Create", func() error { _, err := api.Create(ctx, bgpvpns.CreateOpts{}, nil); return err }()},
			"update extension": {"Update", func() error {
				_, err := api.Update(ctx, "vpn-1", bgpvpns.UpdateOpts{}, bgpvpns.WithUpdateField("route_targets", nil))
				return err
			}()},
			"network required": {"CreateNetworkAssociation", func() error {
				_, err := api.CreateNetworkAssociation(ctx, "vpn-1", bgpvpns.CreateNetworkAssociationOpts{ProjectID: "p"})
				return err
			}()},
			"network extension": {"CreateNetworkAssociation", func() error {
				_, err := api.CreateNetworkAssociation(ctx, "vpn-1", bgpvpns.CreateNetworkAssociationOpts{NetworkID: "n-1"}, bgpvpns.WithCreateNetworkAssociationField("network_id", "x"))
				return err
			}()},
			"router required": {"CreateRouterAssociation", func() error {
				_, err := api.CreateRouterAssociation(ctx, "vpn-1", bgpvpns.CreateRouterAssociationOpts{AdvertiseExtraRoutes: nativeBGPVPNPtr(true)})
				return err
			}()},
			"router update extension": {"UpdateRouterAssociation", func() error {
				_, err := api.UpdateRouterAssociation(ctx, "vpn-1", "ra-1", bgpvpns.UpdateRouterAssociationOpts{}, bgpvpns.WithUpdateRouterAssociationField("advertise_extra_routes", true))
				return err
			}()},
			"port required": {"CreatePortAssociation", func() error {
				_, err := api.CreatePortAssociation(ctx, "vpn-1", bgpvpns.CreatePortAssociationOpts{})
				return err
			}()},
			"port route type": {"CreatePortAssociation", func() error {
				_, err := api.CreatePortAssociation(ctx, "vpn-1", bgpvpns.CreatePortAssociationOpts{PortID: "pt-1", Routes: []bgpvpns.PortRoutes{{Prefix: "10.0.0.0/24"}}})
				return err
			}()},
			"port update extension": {"UpdatePortAssociation", func() error {
				_, err := api.UpdatePortAssociation(ctx, "vpn-1", "pa-1", bgpvpns.UpdatePortAssociationOpts{}, bgpvpns.WithUpdatePortAssociationField("routes", nil))
				return err
			}()},
			"port update route type": {"UpdatePortAssociation", func() error {
				_, err := api.UpdatePortAssociation(ctx, "vpn-1", "pa-1", bgpvpns.UpdatePortAssociationOpts{Routes: &[]bgpvpns.PortRoutes{{Prefix: "10.0.0.0/24"}}})
				return err
			}()},
			"port update nil option": {"UpdatePortAssociation", func() error {
				_, err := api.UpdatePortAssociation(ctx, "vpn-1", "pa-1", bgpvpns.UpdatePortAssociationOpts{}, nil)
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeBGPVPNOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeBGPVPNOperation(t, err, "List")
		}
		for _, err := range api.ListNetworkAssociations(ctx, "vpn-1", nil) {
			nativeBGPVPNOperation(t, err, "ListNetworkAssociations")
		}
		for _, err := range api.ListRouterAssociations(ctx, "vpn-1", nil) {
			nativeBGPVPNOperation(t, err, "ListRouterAssociations")
		}
		for _, err := range api.ListPortAssociations(ctx, "vpn-1", nil) {
			nativeBGPVPNOperation(t, err, "ListPortAssociations")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
