package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/availabilityzones"
)

func TestR2OctAZManilaListZonesUsesV2RouteAndPinnedMicroversion(t *testing.T) {
	cloud := testcloud.New(t)
	var legacy atomic.Int32
	var version string
	cloud.Mux.HandleFunc("GET /v2/availability-zones", func(w http.ResponseWriter, r *http.Request) {
		version = r.Header.Get("X-OpenStack-Manila-API-Version")
		testcloud.JSON(w, 200, `{"availability_zones":[{"id":"1","name":"nova","created_at":"2020-01-01T00:00:00.000000","updated_at":null,"x":true},{"id":"2","name":"b"}]}`)
	})
	cloud.Mux.HandleFunc("GET /v2/os-availability-zone", func(w http.ResponseWriter, r *http.Request) { legacy.Add(1) })
	client := cloud.Client("shared-file-system", "/v2")
	client.Microversion = "2.60"
	api := availabilityzones.New(client)
	zones, err := api.AllZones(context.Background())
	if err != nil || len(zones) != 2 || legacy.Load() != 0 {
		t.Fatalf("%v %v legacy=%d", zones, err, legacy.Load())
	}
	if version != "2.7" || client.Microversion != "2.60" {
		t.Fatalf("version %q client %q", version, client.Microversion)
	}
	z := zones[0]
	if z.ID != "1" || z.Name != "nova" || z.CreatedAt != "2020-01-01T00:00:00.000000" || z.UpdatedAt != "" || string(z.Body["updated_at"]) != "null" || string(z.Body["x"]) != "true" || z.StatusCode != 200 {
		t.Fatalf("%+v", z)
	}
	if _, present := zones[1].Body["created_at"]; present {
		t.Fatal("absent timestamp must stay absent in Body")
	}
}

func TestR2OctAZManilaListZonesErrorsAndCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/availability-zones", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := availabilityzones.New(cloud.Client("shared-file-system", "/v2"))
	if _, err := api.AllZones(context.Background()); err == nil {
		t.Fatal("404 must surface")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.AllZones(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
	cloud.Mux.HandleFunc("GET /v2/os-availability-zone", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"availability_zones":[{"id":"1","name":"legacy","created_at":"2020-01-01T00:00:00.000000"}]}`)
	})
	var legacy []*availabilityzones.AvailabilityZone
	for z, err := range api.List(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		legacy = append(legacy, z)
	}
	if len(legacy) != 1 || legacy[0].Name != "legacy" {
		t.Fatalf("legacy List must stay on os-availability-zone: %v", legacy)
	}
}
