package ipsecpolicies_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonIPsecPolicyCall records the wire request openstacksdk would also send.
type pythonIPsecPolicyCall struct{ method, path, query, body string }

type pythonIPsecPolicyTransport func(*http.Request) (*http.Response, error)

func (transport pythonIPsecPolicyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonIPsecPolicyWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonIPsecPolicyAPI(t *testing.T, reply func(*http.Request) *http.Response) (*ipsecpolicies.API, *[]pythonIPsecPolicyCall, *testcloud.Cloud) {
	t.Helper()
	calls := &[]pythonIPsecPolicyCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonIPsecPolicyTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonIPsecPolicyCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return ipsecpolicies.New(client), calls, cloud
}

const (
	pythonIPsecPolicyBase = "/neutron/v2.0/vpn/ipsecpolicies"
	pythonIPsecPolicyRow  = `{"id":"ips-1","name":"ipsec","description":"d","auth_algorithm":"sha256","encryption_algorithm":"aes-128","encapsulation_mode":"tunnel","transform_protocol":"esp","pfs":"group5","lifetime":{"units":"seconds","value":3600},"project_id":"p","tenant_id":"p"}`
)

func TestPythonIPsecPolicyProxyRequests(t *testing.T) {
	ctx := context.Background()
	deletes := []int{404, 404, 409, 204, 404}
	api, calls, _ := pythonIPsecPolicyAPI(t, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodDelete:
			code := deletes[0]
			deletes = deletes[1:]
			return pythonIPsecPolicyWire(code, `{"NeutronError":{"type":"x"}}`)
		case http.MethodPost:
			return pythonIPsecPolicyWire(201, `{"ipsecpolicy":`+pythonIPsecPolicyRow+`}`)
		}
		return pythonIPsecPolicyWire(200, `{"ipsecpolicy":`+pythonIPsecPolicyRow+`}`)
	})
	// create_vpn_ipsec_policy(name="ipsec", encapsulation_mode="tunnel", transform_protocol="esp",
	// lifetime={"units": "seconds", "value": 3600})
	created, err := api.Create(ctx, ipsecpolicies.CreateOpts{
		Name: "ipsec", EncapsulationMode: ipsecpolicies.EncapsulationModeTunnel, TransformProtocol: ipsecpolicies.TransformProtocolESP,
		Lifetime: &ipsecpolicies.LifetimeCreateOpts{Units: ipsecpolicies.UnitSeconds, Value: 3600},
	})
	if err != nil || created.ID != "ips-1" || created.TransformProtocol != "esp" || created.EncapsulationMode != "tunnel" || created.PFS != "group5" || created.Lifetime.Value != 3600 {
		t.Fatal(created, err)
	}
	// get_vpn_ipsec_policy("ips-1")
	if got, err := api.Get(ctx, "ips-1"); err != nil || got.AuthAlgorithm != "sha256" {
		t.Fatal(got, err)
	}
	// update_vpn_ipsec_policy("ips-1", description="", pfs="group14")
	empty := ""
	if _, err := api.Update(ctx, "ips-1", ipsecpolicies.UpdateOpts{Description: &empty, PFS: ipsecpolicies.PFSGroup14}); err != nil {
		t.Fatal(err)
	}
	// delete_vpn_ipsec_policy("ips-1") ignores 404; ignore_missing=False reports it.
	if err := api.Remove(ctx, resource.ID("ips-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("ips-1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("ips-1")); !gophercloud.ResponseCodeIs(err, http.StatusConflict) {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("ips-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "ips-1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	want := []pythonIPsecPolicyCall{
		{http.MethodPost, pythonIPsecPolicyBase, "", `{"ipsecpolicy":{"encapsulation_mode":"tunnel","lifetime":{"units":"seconds","value":3600},"name":"ipsec","transform_protocol":"esp"}}`},
		{http.MethodGet, pythonIPsecPolicyBase + "/ips-1", "", ""},
		{http.MethodPut, pythonIPsecPolicyBase + "/ips-1", "", `{"ipsecpolicy":{"description":"","pfs":"group14"}}`},
	}
	for range 5 {
		want = append(want, pythonIPsecPolicyCall{http.MethodDelete, pythonIPsecPolicyBase + "/ips-1", "", ""})
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonIPsecPolicyListQueryAndLinks(t *testing.T) {
	ctx := context.Background()
	var cloud *testcloud.Cloud
	api, calls, cloud := pythonIPsecPolicyAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") == "" {
			return pythonIPsecPolicyWire(200, `{"ipsecpolicies":[`+pythonIPsecPolicyRow+`],"ipsecpolicies_links":[{"rel":"next","href":"`+cloud.Server.URL+pythonIPsecPolicyBase+`?limit=1&marker=ips-1&transform_protocol=esp"}]}`)
		}
		return pythonIPsecPolicyWire(200, `{"ipsecpolicies":[]}`)
	})
	// vpn_ipsec_policies(transform_protocol="esp", limit=1)
	var ids []string
	for value, err := range api.List(ctx, ipsecpolicies.WithListOptions(ipsecpolicies.ListOpts{TransformProtocol: "esp"}), ipsecpolicies.WithListQuery("limit", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID+"/"+value.TransformProtocol)
	}
	if !reflect.DeepEqual(ids, []string{"ips-1/esp"}) || len(*calls) != 2 {
		t.Fatal(ids, *calls)
	}
	first, _ := url.ParseQuery((*calls)[0].query)
	next, _ := url.ParseQuery((*calls)[1].query)
	if !reflect.DeepEqual(first, url.Values{"transform_protocol": {"esp"}, "limit": {"1"}}) ||
		!reflect.DeepEqual(next, url.Values{"transform_protocol": {"esp"}, "limit": {"1"}, "marker": {"ips-1"}}) {
		t.Fatal(first, next)
	}
}

func TestPythonIPsecPolicyFindChainAndGetWithoutFields(t *testing.T) {
	ctx := context.Background()
	api, calls, _ := pythonIPsecPolicyAPI(t, func(req *http.Request) *http.Response {
		switch {
		case req.URL.Path == pythonIPsecPolicyBase+"/ips-1":
			return pythonIPsecPolicyWire(200, `{"ipsecpolicy":`+pythonIPsecPolicyRow+`}`)
		case req.URL.Path == pythonIPsecPolicyBase && req.URL.Query().Get("name") == "ipsec":
			return pythonIPsecPolicyWire(200, `{"ipsecpolicies":[`+pythonIPsecPolicyRow+`]}`)
		case req.URL.Path == pythonIPsecPolicyBase:
			return pythonIPsecPolicyWire(200, `{"ipsecpolicies":[]}`)
		}
		return pythonIPsecPolicyWire(404, `{"NeutronError":{"type":"IPsecPolicyNotFound"}}`)
	})
	// find_vpn_ipsec_policy("ipsec"): GET by ID, then list filtered by name.
	if got, err := api.Find(ctx, resource.ID("ipsec"), resource.WithIgnoreMissing()); err != nil || got != nil {
		t.Fatal(got, err)
	}
	if got, err := api.Find(ctx, resource.Name("ipsec"), resource.WithIgnoreMissing()); err != nil || got.ID != "ips-1" {
		t.Fatal(got, err)
	}
	if got, err := api.Find(ctx, resource.ID("ips-1"), resource.WithIgnoreMissing()); err != nil || got.Name != "ipsec" {
		t.Fatal(got, err)
	}
	if _, err := api.Find(ctx, resource.ID("none")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := api.Find(ctx, resource.Name("none")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	// get_vpn_ipsec_policy(..., fields=[...]) has no Go counterpart: Get never sends a query.
	if _, err := api.Get(ctx, "ips-1"); err != nil {
		t.Fatal(err)
	}
	want := []pythonIPsecPolicyCall{
		{http.MethodGet, pythonIPsecPolicyBase + "/ipsec", "", ""},
		{http.MethodGet, pythonIPsecPolicyBase, "name=ipsec", ""},
		{http.MethodGet, pythonIPsecPolicyBase + "/ips-1", "", ""},
		{http.MethodGet, pythonIPsecPolicyBase + "/none", "", ""},
		{http.MethodGet, pythonIPsecPolicyBase, "name=none", ""},
		{http.MethodGet, pythonIPsecPolicyBase + "/ips-1", "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}
