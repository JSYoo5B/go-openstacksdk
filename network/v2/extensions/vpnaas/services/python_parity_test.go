package services_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonVPNServiceCall records the wire request openstacksdk would also send.
type pythonVPNServiceCall struct{ method, path, query, body string }

type pythonVPNServiceTransport func(*http.Request) (*http.Response, error)

func (transport pythonVPNServiceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonVPNServiceWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonVPNServiceAPI(t *testing.T, reply func(*http.Request) *http.Response) (*services.API, *[]pythonVPNServiceCall, *testcloud.Cloud) {
	t.Helper()
	calls := &[]pythonVPNServiceCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonVPNServiceTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonVPNServiceCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return services.New(client), calls, cloud
}

const (
	pythonVPNServiceBase = "/neutron/v2.0/vpn/vpnservices"
	pythonVPNServiceRow  = `{"id":"vs-1","name":"vpn","description":"d","router_id":"r1","subnet_id":"s1","admin_state_up":true,"status":"ACTIVE","external_v4_ip":"203.0.113.5","external_v6_ip":"2001:db8::5","project_id":"p","tenant_id":"p"}`
)

func TestPythonVPNServiceProxyRequests(t *testing.T) {
	ctx := context.Background()
	deletes := []int{404, 404, 409, 204, 404}
	api, calls, _ := pythonVPNServiceAPI(t, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodDelete:
			code := deletes[0]
			deletes = deletes[1:]
			return pythonVPNServiceWire(code, `{"NeutronError":{"type":"x"}}`)
		case http.MethodPost:
			return pythonVPNServiceWire(201, `{"vpnservice":`+pythonVPNServiceRow+`}`)
		}
		return pythonVPNServiceWire(200, `{"vpnservice":`+pythonVPNServiceRow+`}`)
	})
	// create_vpn_service(name="vpn", router_id="r1", is_admin_state_up=True)
	created, err := api.Create(ctx, services.CreateOpts{Name: "vpn", RouterID: "r1", AdminStateUp: gophercloud.Enabled})
	if err != nil || created.ID != "vs-1" || created.Status != "ACTIVE" || !created.AdminStateUp || created.ExternalV4IP != "203.0.113.5" || created.ExternalV6IP != "2001:db8::5" || created.SubnetID != "s1" || created.ProjectID != "p" {
		t.Fatal(created, err)
	}
	// get_vpn_service("vs-1")
	if got, err := api.Get(ctx, "vs-1"); err != nil || got.RouterID != "r1" || got.Description != "d" {
		t.Fatal(got, err)
	}
	// update_vpn_service("vs-1", name="", is_admin_state_up=False)
	empty := ""
	if got, err := api.Update(ctx, "vs-1", services.UpdateOpts{Name: &empty, AdminStateUp: gophercloud.Disabled}); err != nil || got.ID != "vs-1" {
		t.Fatal(got, err)
	}
	// delete_vpn_service("vs-1") ignores 404; ignore_missing=False reports it.
	if err := api.Remove(ctx, resource.ID("vs-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("vs-1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("vs-1")); !gophercloud.ResponseCodeIs(err, http.StatusConflict) {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("vs-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "vs-1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	want := []pythonVPNServiceCall{
		{http.MethodPost, pythonVPNServiceBase, "", `{"vpnservice":{"admin_state_up":true,"name":"vpn","router_id":"r1"}}`},
		{http.MethodGet, pythonVPNServiceBase + "/vs-1", "", ""},
		{http.MethodPut, pythonVPNServiceBase + "/vs-1", "", `{"vpnservice":{"admin_state_up":false,"name":""}}`},
	}
	for range 5 {
		want = append(want, pythonVPNServiceCall{http.MethodDelete, pythonVPNServiceBase + "/vs-1", "", ""})
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonVPNServiceListQueryAndLinks(t *testing.T) {
	ctx := context.Background()
	var cloud *testcloud.Cloud
	api, calls, cloud := pythonVPNServiceAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") == "" {
			return pythonVPNServiceWire(200, `{"vpnservices":[`+pythonVPNServiceRow+`],"vpnservices_links":[{"rel":"next","href":"`+cloud.Server.URL+pythonVPNServiceBase+`?admin_state_up=true&limit=1&marker=vs-1&router_id=r1"}]}`)
		}
		return pythonVPNServiceWire(200, `{"vpnservices":[]}`)
	})
	// vpn_services(router_id="r1", is_admin_state_up=True, limit=1)
	var ids []string
	for value, err := range api.List(ctx, services.WithListOptions(services.ListOpts{RouterID: "r1", AdminStateUp: gophercloud.Enabled}), services.WithListQuery("limit", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID+"/"+value.Status)
	}
	if !reflect.DeepEqual(ids, []string{"vs-1/ACTIVE"}) || len(*calls) != 2 {
		t.Fatal(ids, *calls)
	}
	first, _ := url.ParseQuery((*calls)[0].query)
	next, _ := url.ParseQuery((*calls)[1].query)
	if !reflect.DeepEqual(first, url.Values{"router_id": {"r1"}, "admin_state_up": {"true"}, "limit": {"1"}}) ||
		!reflect.DeepEqual(next, url.Values{"router_id": {"r1"}, "admin_state_up": {"true"}, "limit": {"1"}, "marker": {"vs-1"}}) {
		t.Fatal(first, next)
	}
}

func TestPythonVPNServiceFindChainAndGetWithoutFields(t *testing.T) {
	ctx := context.Background()
	api, calls, _ := pythonVPNServiceAPI(t, func(req *http.Request) *http.Response {
		switch {
		case req.URL.Path == pythonVPNServiceBase+"/vs-1":
			return pythonVPNServiceWire(200, `{"vpnservice":`+pythonVPNServiceRow+`}`)
		case req.URL.Path == pythonVPNServiceBase && req.URL.Query().Get("name") == "vpn":
			return pythonVPNServiceWire(200, `{"vpnservices":[`+pythonVPNServiceRow+`]}`)
		case req.URL.Path == pythonVPNServiceBase:
			return pythonVPNServiceWire(200, `{"vpnservices":[]}`)
		}
		return pythonVPNServiceWire(404, `{"NeutronError":{"type":"VPNServiceNotFound"}}`)
	})
	// find_vpn_service("vpn"): GET by ID, then list filtered by name.
	if got, err := api.Find(ctx, resource.ID("vpn"), resource.WithIgnoreMissing()); err != nil || got != nil {
		t.Fatal(got, err)
	}
	if got, err := api.Find(ctx, resource.Name("vpn")); err != nil || got.ID != "vs-1" {
		t.Fatal(got, err)
	}
	// find_vpn_service("vs-1") stops after the ID GET.
	if got, err := api.Find(ctx, resource.ID("vs-1"), resource.WithIgnoreMissing()); err != nil || got.Name != "vpn" {
		t.Fatal(got, err)
	}
	// ignore_missing=False raises on both steps.
	if _, err := api.Find(ctx, resource.ID("none")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := api.Find(ctx, resource.Name("none")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if got, err := api.Find(ctx, resource.Name("none"), resource.WithIgnoreMissing()); err != nil || got != nil {
		t.Fatal(got, err)
	}
	// get_vpn_service(..., fields=[...]) has no Go counterpart: Get never sends a query.
	if _, err := api.Get(ctx, "vs-1"); err != nil {
		t.Fatal(err)
	}
	want := []pythonVPNServiceCall{
		{http.MethodGet, pythonVPNServiceBase + "/vpn", "", ""},
		{http.MethodGet, pythonVPNServiceBase, "name=vpn", ""},
		{http.MethodGet, pythonVPNServiceBase + "/vs-1", "", ""},
		{http.MethodGet, pythonVPNServiceBase + "/none", "", ""},
		{http.MethodGet, pythonVPNServiceBase, "name=none", ""},
		{http.MethodGet, pythonVPNServiceBase, "name=none", ""},
		{http.MethodGet, pythonVPNServiceBase + "/vs-1", "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}
