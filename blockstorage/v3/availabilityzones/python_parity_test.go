package availabilityzones_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/availabilityzones"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

type pythonZoneTransport func(*http.Request) (*http.Response, error)

func (transport pythonZoneTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func TestPythonZoneListMatchesProxyRequest(t *testing.T) {
	var calls []string
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonZoneTransport(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, req.Method+" "+req.URL.Path+"?"+req.URL.RawQuery)
		body := `{"availabilityZoneInfo":[{"zoneName":"nova","zoneState":{"available":true}},{"zoneName":"az2","zoneState":{"available":false}}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
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
