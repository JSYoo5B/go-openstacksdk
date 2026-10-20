package endpointgroups_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonEndpointGroupCall records the wire request openstacksdk would also send.
type pythonEndpointGroupCall struct{ method, path, query, body string }

type pythonEndpointGroupTransport func(*http.Request) (*http.Response, error)

func (transport pythonEndpointGroupTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonEndpointGroupWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonEndpointGroupAPI(t *testing.T, reply func(*http.Request) *http.Response) (*endpointgroups.API, *[]pythonEndpointGroupCall, *testcloud.Cloud) {
	t.Helper()
	calls := &[]pythonEndpointGroupCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonEndpointGroupTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonEndpointGroupCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return endpointgroups.New(client), calls, cloud
}

const (
	pythonEndpointGroupBase = "/neutron/v2.0/vpn/endpoint-groups"
	pythonEndpointGroupRow  = `{"id":"eg-1","name":"peers","description":"d","type":"cidr","endpoints":["10.0.0.0/24"],"project_id":"p","tenant_id":"p"}`
)

func TestPythonEndpointGroupProxyRequests(t *testing.T) {
	ctx := context.Background()
	deletes := []int{404, 404, 409, 204, 404}
	api, calls, _ := pythonEndpointGroupAPI(t, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodDelete:
			code := deletes[0]
			deletes = deletes[1:]
			return pythonEndpointGroupWire(code, `{"NeutronError":{"type":"x"}}`)
		case http.MethodPost:
			return pythonEndpointGroupWire(201, `{"endpoint_group":`+pythonEndpointGroupRow+`}`)
		}
		return pythonEndpointGroupWire(200, `{"endpoint_group":`+pythonEndpointGroupRow+`}`)
	})
	// create_vpn_endpoint_group(name="peers", type="cidr", endpoints=["10.0.0.0/24"])
	created, err := api.Create(ctx, endpointgroups.CreateOpts{Name: "peers", Type: endpointgroups.TypeCIDR, Endpoints: []string{"10.0.0.0/24"}})
	if err != nil || created.ID != "eg-1" || created.Type != "cidr" || !reflect.DeepEqual(created.Endpoints, []string{"10.0.0.0/24"}) || created.ProjectID != "p" {
		t.Fatal(created, err)
	}
	// get_vpn_endpoint_group("eg-1")
	if got, err := api.Get(ctx, "eg-1"); err != nil || got.Description != "d" {
		t.Fatal(got, err)
	}
	// update_vpn_endpoint_group("eg-1", description="")
	empty := ""
	if got, err := api.Update(ctx, "eg-1", endpointgroups.UpdateOpts{Description: &empty}); err != nil || got.ID != "eg-1" {
		t.Fatal(got, err)
	}
	// delete_vpn_endpoint_group("eg-1") ignores 404; ignore_missing=False reports it.
	if err := api.Remove(ctx, resource.ID("eg-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("eg-1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("eg-1")); !gophercloud.ResponseCodeIs(err, http.StatusConflict) {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("eg-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "eg-1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	want := []pythonEndpointGroupCall{
		{http.MethodPost, pythonEndpointGroupBase, "", `{"endpoint_group":{"endpoints":["10.0.0.0/24"],"name":"peers","type":"cidr"}}`},
		{http.MethodGet, pythonEndpointGroupBase + "/eg-1", "", ""},
		{http.MethodPut, pythonEndpointGroupBase + "/eg-1", "", `{"endpoint_group":{"description":""}}`},
	}
	for range 5 {
		want = append(want, pythonEndpointGroupCall{http.MethodDelete, pythonEndpointGroupBase + "/eg-1", "", ""})
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonEndpointGroupListQueryAndLinks(t *testing.T) {
	ctx := context.Background()
	var cloud *testcloud.Cloud
	api, calls, cloud := pythonEndpointGroupAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") == "" {
			return pythonEndpointGroupWire(200, `{"endpoint_groups":[`+pythonEndpointGroupRow+`],"endpoint_groups_links":[{"rel":"next","href":"`+cloud.Server.URL+pythonEndpointGroupBase+`?endpoint_type=cidr&limit=1&marker=eg-1&name=peers"}]}`)
		}
		return pythonEndpointGroupWire(200, `{"endpoint_groups":[]}`)
	})
	// vpn_endpoint_groups(name="peers", type="cidr", limit=1); Python maps type to endpoint_type.
	var ids []string
	for value, err := range api.List(ctx, endpointgroups.WithListOptions(endpointgroups.ListOpts{Name: "peers"}), endpointgroups.WithListQuery("endpoint_type", "cidr"), endpointgroups.WithListQuery("limit", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID+"/"+value.Type)
	}
	if !reflect.DeepEqual(ids, []string{"eg-1/cidr"}) || len(*calls) != 2 {
		t.Fatal(ids, *calls)
	}
	first, _ := url.ParseQuery((*calls)[0].query)
	next, _ := url.ParseQuery((*calls)[1].query)
	if !reflect.DeepEqual(first, url.Values{"name": {"peers"}, "endpoint_type": {"cidr"}, "limit": {"1"}}) ||
		!reflect.DeepEqual(next, url.Values{"name": {"peers"}, "endpoint_type": {"cidr"}, "limit": {"1"}, "marker": {"eg-1"}}) {
		t.Fatal(first, next)
	}
}

func TestPythonEndpointGroupFindChainAndGetWithoutFields(t *testing.T) {
	ctx := context.Background()
	api, calls, _ := pythonEndpointGroupAPI(t, func(req *http.Request) *http.Response {
		switch {
		case req.URL.Path == pythonEndpointGroupBase+"/eg-1":
			return pythonEndpointGroupWire(200, `{"endpoint_group":`+pythonEndpointGroupRow+`}`)
		case req.URL.Path == pythonEndpointGroupBase && req.URL.Query().Get("name") == "peers":
			return pythonEndpointGroupWire(200, `{"endpoint_groups":[`+pythonEndpointGroupRow+`]}`)
		case req.URL.Path == pythonEndpointGroupBase:
			return pythonEndpointGroupWire(200, `{"endpoint_groups":[]}`)
		}
		return pythonEndpointGroupWire(404, `{"NeutronError":{"type":"EndpointGroupNotFound"}}`)
	})
	// find_vpn_endpoint_group("peers"): GET by ID, then list filtered by name.
	if got, err := api.Find(ctx, resource.ID("peers"), resource.WithIgnoreMissing()); err != nil || got != nil {
		t.Fatal(got, err)
	}
	if got, err := api.Find(ctx, resource.Name("peers")); err != nil || got.ID != "eg-1" {
		t.Fatal(got, err)
	}
	if got, err := api.Find(ctx, resource.ID("eg-1"), resource.WithIgnoreMissing()); err != nil || got.Name != "peers" {
		t.Fatal(got, err)
	}
	if _, err := api.Find(ctx, resource.ID("none")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := api.Find(ctx, resource.Name("none")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	// get_vpn_endpoint_group(..., fields=[...]) has no Go counterpart: Get never sends a query.
	if _, err := api.Get(ctx, "eg-1"); err != nil {
		t.Fatal(err)
	}
	want := []pythonEndpointGroupCall{
		{http.MethodGet, pythonEndpointGroupBase + "/peers", "", ""},
		{http.MethodGet, pythonEndpointGroupBase, "name=peers", ""},
		{http.MethodGet, pythonEndpointGroupBase + "/eg-1", "", ""},
		{http.MethodGet, pythonEndpointGroupBase + "/none", "", ""},
		{http.MethodGet, pythonEndpointGroupBase, "name=none", ""},
		{http.MethodGet, pythonEndpointGroupBase + "/eg-1", "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}
