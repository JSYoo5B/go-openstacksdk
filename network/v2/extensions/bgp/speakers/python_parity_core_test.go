package speakers_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers"
)

type pythonParityCoreSpeakerCall struct{ method, path, body string }

func pythonParityCoreSpeakerAPI(t *testing.T, calls *[]pythonParityCoreSpeakerCall, reply func(*http.Request) string) *speakers.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonParityCoreSpeakerCall{req.Method, req.URL.Path, raw})
		return testcloud.Response(200, reply(req), http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return speakers.New(client)
}

// Python add_gateway_network_to_speaker and remove_gateway_network_from_speaker
// PUT {"network_id": id} to the speaker actions; get_advertised_routes_of_speaker
// GETs the routes. Go sends the same requests and decodes the route rows.
func TestPythonParityCoreSpeakerGatewayNetworksAndRoutes(t *testing.T) {
	ctx := context.Background()
	var calls []pythonParityCoreSpeakerCall
	api := pythonParityCoreSpeakerAPI(t, &calls, func(req *http.Request) string {
		switch {
		case strings.HasSuffix(req.URL.Path, "/add_gateway_network"):
			return `{"network_id":"net-1"}`
		case strings.HasSuffix(req.URL.Path, "/get_advertised_routes"):
			return `{"advertised_routes":[{"destination":"10.0.0.0/24","next_hop":"203.0.113.1"}]}`
		}
		return ``
	})
	added, err := api.AddGatewayNetwork(ctx, "sp-1", speakers.AddGatewayNetworkOpts{NetworkID: "net-1"})
	if err != nil || added.NetworkID != "net-1" {
		t.Fatal(added, err)
	}
	if err := api.RemoveGatewayNetwork(ctx, "sp-1", speakers.RemoveGatewayNetworkOpts{NetworkID: "net-1"}); err != nil {
		t.Fatal(err)
	}
	var routes []speakers.AdvertisedRoute
	for route, err := range api.GetAdvertisedRoutes(ctx, "sp-1") {
		if err != nil {
			t.Fatal(err)
		}
		routes = append(routes, *route)
	}
	if !reflect.DeepEqual(routes, []speakers.AdvertisedRoute{{Destination: "10.0.0.0/24", NextHop: "203.0.113.1"}}) {
		t.Fatalf("%+v", routes)
	}
	base := "/neutron/v2.0/bgp-speakers/sp-1/"
	want := []pythonParityCoreSpeakerCall{
		{http.MethodPut, base + "add_gateway_network", `{"network_id":"net-1"}`},
		{http.MethodPut, base + "remove_gateway_network", `{"network_id":"net-1"}`},
		{http.MethodGet, base + "get_advertised_routes", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}
