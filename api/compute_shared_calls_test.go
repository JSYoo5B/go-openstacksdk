package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/availabilityzones"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/instanceactions"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/remoteconsoles"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/serverexternalevents"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/services"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/shareattachments"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/tags"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/volumeattach"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Shared calls that a category admits but a compute operation never
// supported fail before HTTP.
func TestComputeSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	client := cloud.Client("compute", "/compute/v2.1/")
	client.Microversion = "2.95"
	legacy := cloud.Client("compute", "/compute/v2.1/")
	legacy.Microversion = "2.35"
	first := func(seq func(func(*availabilityzones.AvailabilityZoneRecord, error) bool)) (err error) {
		for _, e := range seq {
			return e
		}
		return nil
	}

	_, err := aggregates.New(client).UpdateRecord(ctx, 1, request.Microversion("2.1"))
	requireUnsupportedCall(t, err, "Microversion")
	requireUnsupportedCall(t, aggregates.New(client).PrecacheImages(ctx, 1, nil, request.Field("x", 1)), "Field")
	requireUnsupportedCall(t, first(availabilityzones.New(client).ListRecords(ctx, request.Headers(nil))), "Headers")
	_, err = extensions.New(client).FindRecord(ctx, "e1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	_, err = flavors.New(client).FetchExtraSpecs(ctx, flavors.FlavorExtraSpecsRequest{ID: "f1"}, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	for _, err := range flavors.New(client).ListRecords(ctx, request.Headers(nil)) {
		requireUnsupportedCall(t, err, "Headers")
	}
	_, err = flavors.New(client).FindFlavor(ctx, "f1", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = flavors.New(client).UpdateRecord(ctx, "f1", request.Microversion("2.1"))
	requireUnsupportedCall(t, err, "Microversion")

	_, err = hypervisors.New(client).FindRecord(ctx, "h1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = services.New(client).FindRecord(ctx, "s1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = services.New(client).UpdateRecord(ctx, "s1", request.Microversion("2.53"))
	requireUnsupportedCall(t, err, "Microversion")

	_, err = images.New(legacy).GetRecord(ctx, "i1", request.Microversion("2.1"))
	requireUnsupportedCall(t, err, "Microversion")
	requireUnsupportedCall(t, images.New(legacy).DeleteRecord(ctx, "i1", request.Header("X-Test", "1")), "Header")
	_, err = images.New(legacy).FindRecord(ctx, "i1", request.Query("name", "x"))
	requireUnsupportedCall(t, err, "Query")
	for _, err := range images.New(legacy).ListRecords(ctx, request.Microversion("2.1")) {
		requireUnsupportedCall(t, err, "Microversion")
	}
	for _, err := range instanceactions.New(client).ListRecords(ctx, "s1", request.Header("X-Test", "1")) {
		requireUnsupportedCall(t, err, "Header")
	}

	_, err = keypairs.New(client).CreateKeypair(ctx, request.Microversion("2.1"))
	requireUnsupportedCall(t, err, "Microversion")
	requireUnsupportedCall(t, keypairs.New(client).DeleteKeypair(ctx, "k1", request.Headers(nil)), "Headers")
	_, err = keypairs.New(client).FindKeypair(ctx, "k1", request.Query("user_id", "u"))
	requireUnsupportedCall(t, err, "Query")
	for _, err := range keypairs.New(client).ListRecords(ctx, request.Headers(nil)) {
		requireUnsupportedCall(t, err, "Headers")
	}

	_, err = quotasets.New(client).GetQuotaClassRecord(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	quota, err := quotasets.New(client).InProject(ctx, resource.ID("p1"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = quota.GetRecord(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = quota.Reset(ctx, request.Microversion("2.1"))
	requireUnsupportedCall(t, err, "Microversion")

	_, err = remoteconsoles.New(client).CreateConsole(ctx, "s1", request.Microversion("2.8"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = serverexternalevents.New(client).CreateRecords(ctx, []serverexternalevents.Event{{"name": "network-changed", "server_uuid": "s1"}}, request.Field("x", 1))
	requireUnsupportedCall(t, err, "Field")
	_, err = servergroups.New(client).FindRecord(ctx, "g1", request.Query("name", "x"))
	requireUnsupportedCall(t, err, "Query")

	_, err = servers.New(client).ConsoleOutput(ctx, "s1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	requireUnsupportedCall(t, servers.New(client).LockWithOptions(ctx, "s1", request.Microversion("2.73")), "Microversion")
	_, err = servers.New(client).FindRecord(ctx, "s1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = servers.New(client).GetServerMigrationRecord(ctx, "s1", "m1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	for _, err := range shareattachments.New(client).ListRecords(ctx, "s1", request.Headers(nil)) {
		requireUnsupportedCall(t, err, "Headers")
	}
	tagScope, err := tags.New(client).InServer(ctx, resource.ID("s1"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = tagScope.Check(ctx, "t1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	for _, err := range volumeattach.New(client).ListRecords(ctx, "s1", request.Marker("m")) {
		requireUnsupportedCall(t, err, "Marker")
	}
	_, err = volumeattach.New(client).UpdateRecord(ctx, "s1", "v1", request.Field("x", 1))
	requireUnsupportedCall(t, err, "Field")

	service := compute.New(client, compute.Dependencies{})
	_, err = service.ListAvailabilityZoneNames(ctx, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = service.AllFlavors(ctx, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = service.ListKeypairs(ctx, request.Limit(1))
	requireUnsupportedCall(t, err, "Limit")
	_, err = service.SearchServerGroups(ctx, "g1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = service.CreateKeypair(ctx, "k1", request.Microversion("2.1"))
	requireUnsupportedCall(t, err, "Microversion")
	if n := calls.Load(); n != 0 {
		t.Fatalf("calls = %d", n)
	}
}

// request.Paginated keeps caller order on a translated list: a later
// Paginated(false) stops after the first page even after Paginated(true).
func TestComputeImagesPaginatedCallerOrder(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /compute/v2.1/images/detail", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"images":[{"id":"i1","name":"one"}],"images_links":[{"rel":"next","href":"`+cloud.Server.URL+`/compute/v2.1/images/detail?marker=i1"}]}`)
	})
	var got int
	for _, err := range images.New(cloud.Client("compute", "/compute/v2.1/")).ListRecords(context.Background(), request.Paginated(true), request.Paginated(false)) {
		if err != nil {
			t.Fatal(err)
		}
		got++
	}
	if got != 1 || calls.Load() != 1 {
		t.Fatal(got, calls.Load())
	}
}
