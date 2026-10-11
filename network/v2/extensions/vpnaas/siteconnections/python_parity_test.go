package siteconnections_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonSiteConnectionCall records the wire request openstacksdk would also send.
type pythonSiteConnectionCall struct{ method, path, query, body string }

type pythonSiteConnectionTransport func(*http.Request) (*http.Response, error)

func (transport pythonSiteConnectionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonSiteConnectionWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonSiteConnectionAPI(t *testing.T, reply func(*http.Request) *http.Response) (*siteconnections.API, *[]pythonSiteConnectionCall, *testcloud.Cloud) {
	t.Helper()
	calls := &[]pythonSiteConnectionCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonSiteConnectionTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonSiteConnectionCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return siteconnections.New(client), calls, cloud
}

const (
	pythonSiteConnectionBase = "/neutron/v2.0/vpn/ipsec-site-connections"
	pythonSiteConnectionRow  = `{"id":"sc-1","name":"site","description":"d","ikepolicy_id":"ike-1","ipsecpolicy_id":"ips-1","vpnservice_id":"vs-1","peer_address":"198.51.100.7","peer_id":"198.51.100.7","local_id":"","psk":"secret","peer_cidrs":["10.1.0.0/24"],"initiator":"bi-directional","mtu":1500,"dpd":{"action":"hold","interval":30,"timeout":120},"admin_state_up":true,"route_mode":"static","auth_mode":"psk","status":"ACTIVE","project_id":"p","tenant_id":"p"}`
)

func TestPythonSiteConnectionProxyRequests(t *testing.T) {
	ctx := context.Background()
	deletes := []int{404, 404, 409, 204, 404}
	api, calls, _ := pythonSiteConnectionAPI(t, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodDelete:
			code := deletes[0]
			deletes = deletes[1:]
			return pythonSiteConnectionWire(code, `{"NeutronError":{"type":"x"}}`)
		case http.MethodPost:
			return pythonSiteConnectionWire(201, `{"ipsec_site_connection":`+pythonSiteConnectionRow+`}`)
		}
		return pythonSiteConnectionWire(200, `{"ipsec_site_connection":`+pythonSiteConnectionRow+`}`)
	})
	// create_vpn_ipsec_site_connection(name="site", ikepolicy_id="ike-1", ipsecpolicy_id="ips-1",
	// vpnservice_id="vs-1", peer_address="198.51.100.7", peer_id="198.51.100.7", psk="secret",
	// peer_cidrs=["10.1.0.0/24"], dpd={"action": "hold", "interval": 30, "timeout": 120})
	created, err := api.Create(ctx, siteconnections.CreateOpts{
		Name: "site", IKEPolicyID: "ike-1", IPSecPolicyID: "ips-1", VPNServiceID: "vs-1", PeerAddress: "198.51.100.7", PeerID: "198.51.100.7",
		PSK: "secret", PeerCIDRs: []string{"10.1.0.0/24"}, DPD: &siteconnections.DPDCreateOpts{Action: siteconnections.ActionHold, Interval: 30, Timeout: 120},
	})
	if err != nil || created.ID != "sc-1" || created.Status != "ACTIVE" || created.MTU != 1500 || created.DPD.Timeout != 120 || created.RouteMode != "static" || created.AuthMode != "psk" || !created.AdminStateUp {
		t.Fatal(created, err)
	}
	// get_vpn_ipsec_site_connection("sc-1")
	if got, err := api.Get(ctx, "sc-1"); err != nil || !reflect.DeepEqual(got.PeerCIDRs, []string{"10.1.0.0/24"}) || got.Initiator != "bi-directional" {
		t.Fatal(got, err)
	}
	// delete_vpn_ipsec_site_connection("sc-1") ignores 404; ignore_missing=False reports it.
	if err := api.Remove(ctx, resource.ID("sc-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("sc-1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("sc-1")); !gophercloud.ResponseCodeIs(err, http.StatusConflict) {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("sc-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "sc-1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	want := []pythonSiteConnectionCall{
		{http.MethodPost, pythonSiteConnectionBase, "", `{"ipsec_site_connection":{"dpd":{"action":"hold","interval":30,"timeout":120},"ikepolicy_id":"ike-1","ipsecpolicy_id":"ips-1","name":"site","peer_address":"198.51.100.7","peer_cidrs":["10.1.0.0/24"],"peer_id":"198.51.100.7","psk":"secret","vpnservice_id":"vs-1"}}`},
		{http.MethodGet, pythonSiteConnectionBase + "/sc-1", "", ""},
	}
	for range 5 {
		want = append(want, pythonSiteConnectionCall{http.MethodDelete, pythonSiteConnectionBase + "/sc-1", "", ""})
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonSiteConnectionUpdateCannotSendEmptyValues(t *testing.T) {
	ctx := context.Background()
	api, calls, _ := pythonSiteConnectionAPI(t, func(*http.Request) *http.Response {
		return pythonSiteConnectionWire(200, `{"ipsec_site_connection":`+pythonSiteConnectionRow+`}`)
	})
	// update_vpn_ipsec_site_connection("sc-1", name="", mtu=1400)
	empty := ""
	if got, err := api.Update(ctx, "sc-1", siteconnections.UpdateOpts{Name: &empty, MTU: 1400}); err != nil || got.ID != "sc-1" {
		t.Fatal(got, err)
	}
	// update_vpn_ipsec_site_connection("sc-1", local_id="", peer_cidrs=[]) sends both keys in
	// Python; Go omits the empty values and rejects core keys as extension fields.
	if _, err := api.Update(ctx, "sc-1", siteconnections.UpdateOpts{LocalID: "", PeerCIDRs: []string{}}); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"local_id", "peer_cidrs"} {
		if _, err := api.Update(ctx, "sc-1", siteconnections.UpdateOpts{}, siteconnections.WithUpdateField(key, "")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(key, err)
		}
	}
	want := []pythonSiteConnectionCall{
		{http.MethodPut, pythonSiteConnectionBase + "/sc-1", "", `{"ipsec_site_connection":{"mtu":1400,"name":""}}`},
		{http.MethodPut, pythonSiteConnectionBase + "/sc-1", "", `{"ipsec_site_connection":{}}`},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonSiteConnectionListQueryAndLinks(t *testing.T) {
	ctx := context.Background()
	var cloud *testcloud.Cloud
	api, calls, cloud := pythonSiteConnectionAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") == "" {
			return pythonSiteConnectionWire(200, `{"ipsec_site_connections":[`+pythonSiteConnectionRow+`],"ipsec_site_connections_links":[{"rel":"next","href":"`+cloud.Server.URL+pythonSiteConnectionBase+`?limit=1&marker=sc-1&status=ACTIVE&vpnservice_id=vs-1"}]}`)
		}
		return pythonSiteConnectionWire(200, `{"ipsec_site_connections":[]}`)
	})
	// vpn_ipsec_site_connections(vpnservice_id="vs-1", status="ACTIVE", limit=1)
	var ids []string
	for value, err := range api.List(ctx, siteconnections.WithListOptions(siteconnections.ListOpts{VPNServiceID: "vs-1"}), siteconnections.WithListQuery("status", "ACTIVE"), siteconnections.WithListQuery("limit", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID+"/"+value.Status)
	}
	if !reflect.DeepEqual(ids, []string{"sc-1/ACTIVE"}) || len(*calls) != 2 {
		t.Fatal(ids, *calls)
	}
	first, _ := url.ParseQuery((*calls)[0].query)
	next, _ := url.ParseQuery((*calls)[1].query)
	if !reflect.DeepEqual(first, url.Values{"vpnservice_id": {"vs-1"}, "status": {"ACTIVE"}, "limit": {"1"}}) ||
		!reflect.DeepEqual(next, url.Values{"vpnservice_id": {"vs-1"}, "status": {"ACTIVE"}, "limit": {"1"}, "marker": {"sc-1"}}) {
		t.Fatal(first, next)
	}
}

func TestPythonSiteConnectionFindChainAndGetWithoutFields(t *testing.T) {
	ctx := context.Background()
	api, calls, _ := pythonSiteConnectionAPI(t, func(req *http.Request) *http.Response {
		switch {
		case req.URL.Path == pythonSiteConnectionBase+"/sc-1":
			return pythonSiteConnectionWire(200, `{"ipsec_site_connection":`+pythonSiteConnectionRow+`}`)
		case req.URL.Path == pythonSiteConnectionBase && req.URL.Query().Get("name") == "site":
			return pythonSiteConnectionWire(200, `{"ipsec_site_connections":[`+pythonSiteConnectionRow+`]}`)
		case req.URL.Path == pythonSiteConnectionBase:
			return pythonSiteConnectionWire(200, `{"ipsec_site_connections":[]}`)
		}
		return pythonSiteConnectionWire(404, `{"NeutronError":{"type":"IPsecSiteConnectionNotFound"}}`)
	})
	// find_vpn_ipsec_site_connection("site"): GET by ID, then list filtered by name.
	if got, err := api.Find(ctx, resource.ID("site"), resource.WithIgnoreMissing()); err != nil || got != nil {
		t.Fatal(got, err)
	}
	if got, err := api.Find(ctx, resource.Name("site"), resource.WithIgnoreMissing()); err != nil || got.ID != "sc-1" {
		t.Fatal(got, err)
	}
	if got, err := api.Find(ctx, resource.ID("sc-1"), resource.WithIgnoreMissing()); err != nil || got.Name != "site" {
		t.Fatal(got, err)
	}
	if _, err := api.Find(ctx, resource.ID("none")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := api.Find(ctx, resource.Name("none")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	// get_vpn_ipsec_site_connection(..., fields=[...]) has no Go counterpart: Get never sends a query.
	if _, err := api.Get(ctx, "sc-1"); err != nil {
		t.Fatal(err)
	}
	want := []pythonSiteConnectionCall{
		{http.MethodGet, pythonSiteConnectionBase + "/site", "", ""},
		{http.MethodGet, pythonSiteConnectionBase, "name=site", ""},
		{http.MethodGet, pythonSiteConnectionBase + "/sc-1", "", ""},
		{http.MethodGet, pythonSiteConnectionBase + "/none", "", ""},
		{http.MethodGet, pythonSiteConnectionBase, "name=none", ""},
		{http.MethodGet, pythonSiteConnectionBase + "/sc-1", "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}
