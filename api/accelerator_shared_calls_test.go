package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/attributes"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/deployables"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/deviceprofiles"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/devices"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func acceleratorSharedCallsClient(t *testing.T, reply func(*http.Request) (int, string)) (*gophercloud.ServiceClient, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		code, body := reply(req)
		return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("accelerator", "/v2")
	client.Microversion = "2.10"
	return client, &calls
}

const acceleratorSharedCallsUUID = "11111111-1111-4111-8111-111111111111"

// Shared calls that a category admits but a Cyborg operation never supported
// fail before HTTP.
func TestAcceleratorSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	client, calls := acceleratorSharedCallsClient(t, func(*http.Request) (int, string) { return 500, `{}` })
	id := resource.ID(acceleratorSharedCallsUUID)

	// Creates take Field and Header only.
	_, err := attributes.New(client).Create(ctx, attributes.CreateOpts{DeployableID: 1, Key: "k", Value: "v"}, request.Microversion("2.1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = deviceprofiles.New(client).Create(ctx, deviceprofiles.CreateOpts{Name: "p", Groups: []deviceprofiles.Group{{"k": "v"}}}, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = acceleratorrequests.New(client).Create(ctx, acceleratorrequests.CreateOpts{DeviceProfileName: "p"}, request.Microversion("2.1"))
	requireUnsupportedCall(t, err, "Microversion")

	// Actions, program and binding patches take Header only.
	_, err = devices.New(client).Enable(ctx, id, request.Field("vendor", true))
	requireUnsupportedCall(t, err, "Field")
	_, err = deployables.New(client).Program(ctx, id, acceleratorSharedCallsUUID, request.Field("vendor", true))
	requireUnsupportedCall(t, err, "Field")
	_, err = acceleratorrequests.New(client).Unbind(ctx, id, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")

	// Batch deletes are strict and take Header only.
	_, err = acceleratorrequests.New(client).DeleteMany(ctx, []string{acceleratorSharedCallsUUID}, request.IgnoreMissing(true))
	requireUnsupportedCall(t, err, "IgnoreMissing")
	_, err = acceleratorrequests.New(client).DeleteByInstance(ctx, acceleratorSharedCallsUUID, request.Microversion("2.1"))
	requireUnsupportedCall(t, err, "Microversion")

	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}
}

// Shared calls apply in caller order: the later call wins.
func TestAcceleratorSharedCallsApplyInCallerOrder(t *testing.T) {
	var header http.Header
	client, calls := acceleratorSharedCallsClient(t, func(r *http.Request) (int, string) {
		header = r.Header.Clone()
		return 204, ``
	})
	if _, err := acceleratorrequests.New(client).DeleteMany(context.Background(), []string{acceleratorSharedCallsUUID},
		request.Header("X-Vendor", "first"), request.Header("X-Vendor", "second")); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 || header.Get("X-Vendor") != "second" {
		t.Fatal(calls.Load(), header)
	}
}
