package openstack_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestConnectionRouterInterfacesClassifyCompletePortsWithConfiguredLocation(t *testing.T) {
	cloud := testcloud.New(t)
	configured := "configured"
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/v2.0/"), sdk.WithCloudLocation(resource.CloudLocation{Cloud: &configured}))
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("device_id") != "r" {
			t.Fatal(r.URL)
		}
		testcloud.JSON(w, 200, `{"ports":[{"id":"g","device_owner":"network:router_gateway","location":{"cloud":"forged"}},{"id":"i","device_owner":"network:router_interface","location":{"cloud":"forged"}},{"id":"ha","device_owner":"network:router_ha_interface"}]}`)
	})
	result, err := conn.ListRouterInterfaces(context.Background(), network.RouterInterfaceRouterID("r"))
	if err != nil || result == nil {
		t.Fatal(result, err)
	}
	if calls != 1 || len(result.Inventory) != 3 || len(result.Ports) != 2 || string(result.Ports[0].Resource.Body["id"]) != `"i"` || string(result.Ports[1].Resource.Body["id"]) != `"g"` {
		t.Fatal(result, calls)
	}
	var location map[string]json.RawMessage
	if err := json.Unmarshal(result.Ports[0].Resource.Body["location"], &location); err != nil || string(location["cloud"]) != `"configured"` {
		t.Fatal(location, err)
	}
	if string(result.Ports[0].Wire.Body["location"]) != `{"cloud":"forged"}` || result.Ports[0].StatusCode != 200 {
		t.Fatal(result.Ports[0])
	}
	explicit := "explicit"
	option := network.WithRouterInterfaceReadCurrentLocation(resource.CloudLocation{Cloud: &explicit})
	explicit = "changed"
	result, err = conn.ListRouterInterfaces(context.Background(), network.RouterInterfaceRouterID("r"), option)
	if err != nil || result == nil || len(result.Ports) != 2 {
		t.Fatal(result, err)
	}
	if err := json.Unmarshal(result.Ports[0].Resource.Body["location"], &location); err != nil || string(location["cloud"]) != `"explicit"` || calls != 2 {
		t.Fatal(location, err, calls)
	}
}
