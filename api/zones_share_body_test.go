package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/zones"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const zoneShareBodyResponse = `{"id":"share-1","zone_id":"zone-1","project_id":"p1","target_project_id":"p"}`

func TestZoneShareBodySendsTargetProjectAndExtensionFields(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	var want map[string]string
	cloud.Mux.HandleFunc("POST /v2/zones/zone-1/shares", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if got := designateStrings(designateFlatBody(t, r)); !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v want=%v", got, want)
		}
		testcloud.JSON(w, 201, zoneShareBodyResponse)
	})
	api := newDesignateZones(t, cloud)
	want = map[string]string{"target_project_id": `"p"`}
	share, err := api.Share(context.Background(), "zone-1", zones.ShareZoneOpts{TargetProjectID: "p"})
	if err != nil || share == nil || share.ID != "share-1" {
		t.Fatalf("share=%+v error=%v", share, err)
	}
	want = map[string]string{"target_project_id": `"p"`, "flag": `false`, "nothing": `null`}
	if _, err = api.Share(context.Background(), "zone-1", zones.ShareZoneOpts{TargetProjectID: "p"}, zones.WithShareField("flag", false), zones.WithShareField("nothing", nil)); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestZoneShareBodyRejectsProtectedFieldBeforeRequest(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v2/zones/zone-1/shares", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 201, zoneShareBodyResponse)
	})
	_, err := newDesignateZones(t, cloud).Share(context.Background(), "zone-1", zones.ShareZoneOpts{TargetProjectID: "p"}, zones.WithShareField("target_project_id", "other"))
	if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
		t.Fatalf("error=%v calls=%d", err, calls.Load())
	}
}

func TestZoneShareBodyAcceptsOnlyCreated(t *testing.T) {
	for _, status := range []int{200, 202, 204} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /v2/zones/zone-1/shares", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			testcloud.JSON(w, status, zoneShareBodyResponse)
		})
		share, err := newDesignateZones(t, cloud).Share(context.Background(), "zone-1", zones.ShareZoneOpts{TargetProjectID: "p"})
		if err == nil || share != nil || calls.Load() != 1 {
			t.Fatalf("status=%d share=%+v error=%v calls=%d", status, share, err, calls.Load())
		}
	}
}
