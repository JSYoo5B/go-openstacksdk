package bgpvpns_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonBGPVPNCall records the wire request openstacksdk would also send.
type pythonBGPVPNCall struct{ method, path, query, body string }

func pythonBGPVPNWire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}})
}

func pythonBGPVPNAPI(t *testing.T, reply func(*http.Request) *http.Response) (*bgpvpns.API, *[]pythonBGPVPNCall) {
	t.Helper()
	calls := &[]pythonBGPVPNCall{}
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonBGPVPNCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return bgpvpns.New(client), calls
}

func pythonBGPVPNPtr[T any](value T) *T { return &value }

const (
	pythonBGPVPNBase       = "/neutron/v2.0/bgpvpn/bgpvpns"
	pythonBGPVPNRow        = `{"id":"vpn-1","name":"vpn","type":"l3","route_targets":["64512:1"],"import_targets":[],"export_targets":[],"route_distinguishers":[],"networks":["n-1"],"routers":[],"ports":[],"local_pref":100,"vni":0,"project_id":"p","tenant_id":"p"}`
	pythonBGPVPNNetworkRow = `{"id":"na-1","network_id":"n-1","project_id":"p","tenant_id":"p"}`
	pythonBGPVPNRouterRow  = `{"id":"ra-1","router_id":"r-1","advertise_extra_routes":false,"project_id":"p","tenant_id":"p"}`
	pythonBGPVPNPortRow    = `{"id":"pa-1","port_id":"pt-1","advertise_fixed_ips":true,"routes":[{"type":"prefix","prefix":"10.0.0.0/24","local_pref":100}],"project_id":"p","tenant_id":"p"}`
)

func TestPythonBGPVPNProxyRequests(t *testing.T) {
	ctx := context.Background()
	deletes := []int{404, 404, 409, 204, 404}
	api, calls := pythonBGPVPNAPI(t, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodDelete:
			code := deletes[0]
			deletes = deletes[1:]
			return pythonBGPVPNWire(code, `{"NeutronError":{"type":"x"}}`)
		case http.MethodPost:
			return pythonBGPVPNWire(201, `{"bgpvpn":`+pythonBGPVPNRow+`}`)
		}
		return pythonBGPVPNWire(200, `{"bgpvpn":`+pythonBGPVPNRow+`}`)
	})
	// create_bgpvpn(name="vpn", type="l3", route_targets=["64512:1"], local_pref=100)
	created, err := api.Create(ctx, bgpvpns.CreateOpts{Name: "vpn", Type: "l3", RouteTargets: []string{"64512:1"}, LocalPref: 100})
	if err != nil || created.ID != "vpn-1" || created.Type != "l3" || created.LocalPref == nil || *created.LocalPref != 100 || !reflect.DeepEqual(created.RouteTargets, []string{"64512:1"}) {
		t.Fatal(created, err)
	}
	// get_bgpvpn("vpn-1")
	if got, err := api.Get(ctx, "vpn-1"); err != nil || got.ProjectID != "p" || !reflect.DeepEqual(got.Networks, []string{"n-1"}) {
		t.Fatal(got, err)
	}
	// update_bgpvpn("vpn-1", name="", import_targets=[], local_pref=0)
	if _, err := api.Update(ctx, "vpn-1", bgpvpns.UpdateOpts{Name: pythonBGPVPNPtr(""), ImportTargets: &[]string{}, LocalPref: pythonBGPVPNPtr(0)}); err != nil {
		t.Fatal(err)
	}
	// delete_bgpvpn("vpn-1") ignores 404; ignore_missing=False reports it.
	if err := api.Remove(ctx, resource.ID("vpn-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("vpn-1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("vpn-1")); !gophercloud.ResponseCodeIs(err, http.StatusConflict) {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("vpn-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "vpn-1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	want := []pythonBGPVPNCall{
		{http.MethodPost, pythonBGPVPNBase, "", `{"bgpvpn":{"local_pref":100,"name":"vpn","route_targets":["64512:1"],"type":"l3"}}`},
		{http.MethodGet, pythonBGPVPNBase + "/vpn-1", "", ""},
		{http.MethodPut, pythonBGPVPNBase + "/vpn-1", "", `{"bgpvpn":{"import_targets":[],"local_pref":0,"name":""}}`},
	}
	for range 5 {
		want = append(want, pythonBGPVPNCall{http.MethodDelete, pythonBGPVPNBase + "/vpn-1", "", ""})
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonBGPVPNCreateCannotSendZeroLocalPref(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonBGPVPNAPI(t, func(*http.Request) *http.Response {
		return pythonBGPVPNWire(201, `{"bgpvpn":`+pythonBGPVPNRow+`}`)
	})
	// create_bgpvpn(name="vpn", local_pref=0) sends "local_pref": 0; Go omits it.
	if _, err := api.Create(ctx, bgpvpns.CreateOpts{Name: "vpn", LocalPref: 0}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Create(ctx, bgpvpns.CreateOpts{Name: "vpn"}, bgpvpns.WithCreateField("local_pref", 0)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	want := []pythonBGPVPNCall{{http.MethodPost, pythonBGPVPNBase, "", `{"bgpvpn":{"name":"vpn"}}`}}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonBGPVPNListQueryAndMarkerPaging(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonBGPVPNAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") == "" {
			return pythonBGPVPNWire(200, `{"bgpvpns":[`+pythonBGPVPNRow+`]}`)
		}
		return pythonBGPVPNWire(200, `{"bgpvpns":[]}`)
	})
	// bgpvpns(name="vpn", type="l3", networks=["n-1"], limit=1): Python repeats the request
	// with the last ID as marker while limit is set, as the Go marker pager does.
	var ids []string
	for value, err := range api.List(ctx, bgpvpns.WithListOptions(bgpvpns.ListOpts{Networks: []string{"n-1"}, Limit: 1}), bgpvpns.WithListQuery("name", "vpn"), bgpvpns.WithListQuery("type", "l3")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"vpn-1"}) || len(*calls) != 2 {
		t.Fatal(ids, *calls)
	}
	first, _ := url.ParseQuery((*calls)[0].query)
	next, _ := url.ParseQuery((*calls)[1].query)
	if !reflect.DeepEqual(first, url.Values{"networks": {"n-1"}, "limit": {"1"}, "name": {"vpn"}, "type": {"l3"}}) ||
		!reflect.DeepEqual(next, url.Values{"networks": {"n-1"}, "limit": {"1"}, "name": {"vpn"}, "type": {"l3"}, "marker": {"vpn-1"}}) {
		t.Fatal(first, next)
	}
}

func TestPythonBGPVPNFindListsWithoutNameAndGetHasNoFields(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonBGPVPNAPI(t, func(req *http.Request) *http.Response {
		switch req.URL.Path {
		case pythonBGPVPNBase + "/vpn-1":
			return pythonBGPVPNWire(200, `{"bgpvpn":`+pythonBGPVPNRow+`}`)
		case pythonBGPVPNBase:
			return pythonBGPVPNWire(200, `{"bgpvpns":[`+pythonBGPVPNRow+`]}`)
		}
		return pythonBGPVPNWire(404, `{"NeutronError":{"type":"BGPVPNNotFound"}}`)
	})
	// Explicit Ref lookup: the ID GET stops on 404 and the name list sends name=.
	if got, err := api.Find(ctx, resource.ID("vpn"), resource.WithIgnoreMissing()); err != nil || got != nil {
		t.Fatal(got, err)
	}
	if got, err := api.Find(ctx, resource.Name("vpn")); err != nil || got.ID != "vpn-1" {
		t.Fatal(got, err)
	}
	// get_bgpvpn(..., fields=[...]) has no Go counterpart: Get never sends a query.
	if _, err := api.Get(ctx, "vpn-1"); err != nil {
		t.Fatal(err)
	}
	want := []pythonBGPVPNCall{
		{http.MethodGet, pythonBGPVPNBase + "/vpn", "", ""},
		{http.MethodGet, pythonBGPVPNBase, "name=vpn", ""},
		{http.MethodGet, pythonBGPVPNBase + "/vpn-1", "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonBGPVPNAssociationRequests(t *testing.T) {
	ctx := context.Background()
	deletes := []int{204, 404, 409}
	api, calls := pythonBGPVPNAPI(t, func(req *http.Request) *http.Response {
		path := req.URL.Path
		if req.Method == http.MethodDelete {
			code := deletes[0]
			deletes = append(deletes[1:], code)
			return pythonBGPVPNWire(code, `{"NeutronError":{"type":"x"}}`)
		}
		code := http.StatusOK
		if req.Method == http.MethodPost {
			code = http.StatusCreated
		}
		for _, kind := range []struct{ name, row string }{{"network", pythonBGPVPNNetworkRow}, {"router", pythonBGPVPNRouterRow}, {"port", pythonBGPVPNPortRow}} {
			if req.Method == http.MethodGet && strings.HasSuffix(path, "/"+kind.name+"_associations") {
				if req.URL.Query().Get("marker") != "" {
					return pythonBGPVPNWire(code, `{"`+kind.name+`_associations":[]}`)
				}
				return pythonBGPVPNWire(code, `{"`+kind.name+`_associations":[`+kind.row+`]}`)
			}
			if strings.Contains(path, "/"+kind.name+"_associations") {
				return pythonBGPVPNWire(code, `{"`+kind.name+`_association":`+kind.row+`}`)
			}
		}
		return pythonBGPVPNWire(code, `{"network_association":`+pythonBGPVPNNetworkRow+`}`)
	})
	// create_bgpvpn_network_association("vpn-1", network_id="n-1") and the router/port variants.
	network, err := api.CreateNetworkAssociation(ctx, "vpn-1", bgpvpns.CreateNetworkAssociationOpts{NetworkID: "n-1"})
	if err != nil || network.ID != "na-1" || network.NetworkID != "n-1" {
		t.Fatal(network, err)
	}
	router, err := api.CreateRouterAssociation(ctx, "vpn-1", bgpvpns.CreateRouterAssociationOpts{RouterID: "r-1", AdvertiseExtraRoutes: pythonBGPVPNPtr(false)})
	if err != nil || router.ID != "ra-1" || router.AdvertiseExtraRoutes {
		t.Fatal(router, err)
	}
	route := bgpvpns.PortRoutes{Type: "prefix", Prefix: "10.0.0.0/24", LocalPref: pythonBGPVPNPtr(100)}
	port, err := api.CreatePortAssociation(ctx, "vpn-1", bgpvpns.CreatePortAssociationOpts{PortID: "pt-1", Routes: []bgpvpns.PortRoutes{route}})
	if err != nil || port.ID != "pa-1" || !port.AdvertiseFixedIPs || len(port.Routes) != 1 || *port.Routes[0].LocalPref != 100 {
		t.Fatal(port, err)
	}
	// get_bgpvpn_*_association("vpn-1", id) without fields.
	if got, err := api.GetNetworkAssociation(ctx, "vpn-1", "na-1"); err != nil || got.NetworkID != "n-1" {
		t.Fatal(got, err)
	}
	if got, err := api.GetRouterAssociation(ctx, "vpn-1", "ra-1"); err != nil || got.RouterID != "r-1" {
		t.Fatal(got, err)
	}
	if got, err := api.GetPortAssociation(ctx, "vpn-1", "pa-1"); err != nil || got.PortID != "pt-1" {
		t.Fatal(got, err)
	}
	// update_bgpvpn_router_association("vpn-1", "ra-1", advertise_extra_routes=True) and
	// update_bgpvpn_port_association("vpn-1", "pa-1", routes=[], advertise_fixed_ips=False).
	if _, err := api.UpdateRouterAssociation(ctx, "vpn-1", "ra-1", bgpvpns.UpdateRouterAssociationOpts{AdvertiseExtraRoutes: pythonBGPVPNPtr(true)}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.UpdatePortAssociation(ctx, "vpn-1", "pa-1", bgpvpns.UpdatePortAssociationOpts{Routes: &[]bgpvpns.PortRoutes{}, AdvertiseFixedIPs: pythonBGPVPNPtr(false)}); err != nil {
		t.Fatal(err)
	}
	// bgpvpn_network_associations("vpn-1"), bgpvpn_router_associations("vpn-1", limit=1)
	// and bgpvpn_port_associations("vpn-1", fields=["id"]).
	var ids []string
	for value, err := range api.ListNetworkAssociations(ctx, "vpn-1") {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	for value, err := range api.ListRouterAssociations(ctx, "vpn-1", bgpvpns.WithListRouterAssociationsOptions(bgpvpns.ListRouterAssociationsOpts{Limit: 1})) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	for value, err := range api.ListPortAssociations(ctx, "vpn-1", bgpvpns.WithListPortAssociationsOptions(bgpvpns.ListPortAssociationsOpts{Fields: []string{"id"}})) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"na-1", "ra-1", "pa-1"}) {
		t.Fatal(ids)
	}
	// delete_bgpvpn_*_association ignores 404 through resource.IgnoreMissing; 409 stays an error.
	for _, remove := range []func() error{
		func() error { return api.DeleteNetworkAssociation(ctx, "vpn-1", "na-1") },
		func() error { return api.DeleteRouterAssociation(ctx, "vpn-1", "ra-1") },
		func() error { return api.DeletePortAssociation(ctx, "vpn-1", "pa-1") },
	} {
		if err := resource.IgnoreMissing(remove()); err != nil {
			t.Fatal(err)
		}
		if err := resource.IgnoreMissing(remove()); err != nil {
			t.Fatal(err)
		}
		if err := resource.IgnoreMissing(remove()); !gophercloud.ResponseCodeIs(err, http.StatusConflict) {
			t.Fatal(err)
		}
	}
	base := pythonBGPVPNBase + "/vpn-1/"
	want := []pythonBGPVPNCall{
		{http.MethodPost, base + "network_associations", "", `{"network_association":{"network_id":"n-1"}}`},
		{http.MethodPost, base + "router_associations", "", `{"router_association":{"advertise_extra_routes":false,"router_id":"r-1"}}`},
		{http.MethodPost, base + "port_associations", "", `{"port_association":{"port_id":"pt-1","routes":[{"local_pref":100,"prefix":"10.0.0.0/24","type":"prefix"}]}}`},
		{http.MethodGet, base + "network_associations/na-1", "", ""},
		{http.MethodGet, base + "router_associations/ra-1", "", ""},
		{http.MethodGet, base + "port_associations/pa-1", "", ""},
		{http.MethodPut, base + "router_associations/ra-1", "", `{"router_association":{"advertise_extra_routes":true}}`},
		{http.MethodPut, base + "port_associations/pa-1", "", `{"port_association":{"advertise_fixed_ips":false,"routes":[]}}`},
		{http.MethodGet, base + "network_associations", "", ""},
		{http.MethodGet, base + "router_associations", "limit=1", ""},
		{http.MethodGet, base + "router_associations", "limit=1&marker=ra-1", ""},
		{http.MethodGet, base + "port_associations", "fields=id", ""},
	}
	for _, kind := range []string{"network_associations/na-1", "router_associations/ra-1", "port_associations/pa-1"} {
		for range 3 {
			want = append(want, pythonBGPVPNCall{http.MethodDelete, base + kind, "", ""})
		}
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonBGPVPNPortAssociationFindHasNoHelper(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonBGPVPNAPI(t, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/port_associations") {
			return pythonBGPVPNWire(200, `{"port_associations":[`+pythonBGPVPNPortRow+`]}`)
		}
		return pythonBGPVPNWire(404, `{"NeutronError":{"type":"BGPVPNPortAssocNotFound"}}`)
	})
	// find_bgpvpn_port_association("pa-9", "vpn-1"): Python GETs the ID, then lists the
	// association collection and matches ID or name. Go only offers the two raw calls.
	if _, err := api.GetPortAssociation(ctx, "vpn-1", "pa-9"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	var matched []string
	for value, err := range api.ListPortAssociations(ctx, "vpn-1") {
		if err != nil {
			t.Fatal(err)
		}
		if value.ID == "pa-9" {
			matched = append(matched, value.ID)
		}
	}
	if len(matched) != 0 {
		t.Fatal(matched)
	}
	base := pythonBGPVPNBase + "/vpn-1/port_associations"
	want := []pythonBGPVPNCall{
		{http.MethodGet, base + "/pa-9", "", ""},
		{http.MethodGet, base, "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}
