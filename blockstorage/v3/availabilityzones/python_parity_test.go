package availabilityzones_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/availabilityzones"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

func TestPythonZoneListMatchesProxyRequest(t *testing.T) {
	var calls []string
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, req.Method+" "+req.URL.Path+"?"+req.URL.RawQuery)
		body := `{"availabilityZoneInfo":[{"zoneName":"nova","zoneState":{"available":true}},{"zoneName":"az2","zoneState":{"available":false}}]}`
		return testcloud.Response(200, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	// availability_zones() lists availabilityZoneInfo with zoneName and zoneState.
	var zones []string
	for zone, err := range availabilityzones.New(client).List(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		state := "down"
		if zone.ZoneState.Available {
			state = "up"
		}
		zones = append(zones, zone.ZoneName+"/"+state)
	}
	if !reflect.DeepEqual(zones, []string{"nova/up", "az2/down"}) || !reflect.DeepEqual(calls, []string{"GET /cinder/v3/project/os-availability-zone?"}) {
		t.Fatal(zones, calls)
	}
}
