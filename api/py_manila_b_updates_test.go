package api_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharenetworks"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shares"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharetypes"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/snapshots"
	"github.com/gophercloud/gophercloud/v2"
)

func pyManilaBPut(path, body string) pyManilaBCall {
	return pyManilaBCall{method: "PUT", path: path, body: body, version: pyManilaBVersion}
}

// Python Proxy._update builds a new Resource from the supplied attributes and
// commits them with PUT <base_path>/{id}, wrapping the dirty attributes in the
// resource key. Empty strings and false are real dirty values. The Python
// attribute names name/description (not typed in Go) travel as extension fields.
func TestPyManilaBUpdateOperationsPutResourceEnvelopeWithEmptyAndFalseValues(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	empty := ""
	name := "renamed"
	no := false
	cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{
		"PUT /v2/shares/share-1":      {status: 200, body: `{"share":{"id":"share-1","name":"renamed","is_public":false}}`},
		"PUT /v2/snapshots/snap-1":    {status: 200, body: `{"snapshot":{"id":"snap-1","name":"renamed"}}`},
		"PUT /v2/share-networks/sn-1": {status: 200, body: `{"share_network":{"id":"sn-1","name":"renamed"}}`},
		"PUT /v2/types/ty-1":          {status: 200, body: `{"share_type":{"id":"ty-1","name":"renamed"}}`},
	})
	client := cloud.client(pyManilaBVersion)
	share, err := shares.New(client).Update(ctx, "share-1", shares.UpdateOpts{DisplayName: &name, DisplayDescription: &empty, IsPublic: &no}, shares.WithUpdateField("description", ""))
	if err != nil || share.ID != "share-1" || share.IsPublic {
		t.Fatalf("share=%+v error=%v", share, err)
	}
	snapshot, err := snapshots.New(client).Update(ctx, "snap-1", snapshots.UpdateOpts{DisplayName: &name, DisplayDescription: &empty}, snapshots.WithUpdateField("description", ""))
	if err != nil || snapshot.ID != "snap-1" {
		t.Fatalf("snapshot=%+v error=%v", snapshot, err)
	}
	network, err := sharenetworks.New(client).Update(ctx, "sn-1", sharenetworks.UpdateOpts{Name: &name, Description: &empty, NeutronNetID: "net-2"})
	if err != nil || network.ID != "sn-1" {
		t.Fatalf("network=%+v error=%v", network, err)
	}
	shareType, err := sharetypes.New(client).Update(ctx, "ty-1", sharetypes.UpdateOpts{Name: &name, Description: &empty, IsPublic: &no})
	if err != nil || shareType.ID != "ty-1" {
		t.Fatalf("type=%+v error=%v", shareType, err)
	}
	cloud.expect(
		pyManilaBPut("/v2/shares/share-1", `{"share":{"display_name":"renamed","display_description":"","is_public":false,"description":""}}`),
		pyManilaBPut("/v2/snapshots/snap-1", `{"snapshot":{"display_name":"renamed","display_description":"","description":""}}`),
		pyManilaBPut("/v2/share-networks/sn-1", `{"share_network":{"name":"renamed","description":"","neutron_net_id":"net-2"}}`),
		pyManilaBPut("/v2/types/ty-1", `{"share_type":{"name":"renamed","description":"","share_type_access:is_public":false}}`),
	)
}

// Python Proxy._update with no attributes would send an empty envelope. Go
// omits nil pointers, and only the success code 200 passes (Python accepts any
// 2xx).
func TestPyManilaBUpdateOperationsOmitNilFieldsAndRequireStatus200(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{"PUT /v2/shares/share-1": {status: 200, body: `{"share":{"id":"share-1"}}`}})
	if _, err := shares.New(cloud.client(pyManilaBVersion)).Update(ctx, "share-1", shares.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	cloud.expect(pyManilaBPut("/v2/shares/share-1", `{"share":{}}`))
	for _, status := range []int{202, 204} {
		odd := newPyManilaBCloud(t, map[string]pyManilaBStub{"PUT /v2/snapshots/snap-1": {status: status}})
		if _, err := snapshots.New(odd.client(pyManilaBVersion)).Update(ctx, "snap-1", snapshots.UpdateOpts{}); err == nil || !gophercloud.ResponseCodeIs(err, status) {
			t.Fatalf("status %d error=%v", status, err)
		}
	}
}

// Python update_share_type_extra_specs POSTs {"extra_specs": specs} to
// /types/{id}/extra_specs and reads the returned extra_specs; set_extra_specs
// does not require a prior GET.
func TestPyManilaBShareTypeExtraSpecsPostSpecsWithoutFetch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{"POST /v2/types/ty-1/extra_specs": {status: 200, body: `{"extra_specs":{"snapshot_support":"True","empty":""}}`}})
	got, err := sharetypes.New(cloud.client(pyManilaBVersion)).SetExtraSpecs(ctx, "ty-1", sharetypes.SetExtraSpecsOpts{ExtraSpecs: map[string]any{"snapshot_support": "True", "empty": ""}})
	if err != nil || !reflect.DeepEqual(map[string]any(got), map[string]any{"snapshot_support": "True", "empty": ""}) {
		t.Fatalf("specs=%v error=%v", got, err)
	}
	cloud.expect(pyManilaBPost("/v2/types/ty-1/extra_specs", `{"extra_specs":{"snapshot_support":"True","empty":""}}`))
}

// Python wait_for_status/wait_for_delete poll Resource.fetch(skip_cache=True)
// until the attribute matches case-insensitively, a failure state appears
// (default ['ERROR']) or, for deletion, the resource is 404 or reports
// "deleted". Go Collection waits poll the fixed ID with GET and accept the
// Python defaults through explicit options.
func TestPyManilaBWaitForStatusPollsFetchUntilTargetOrFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fast := []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithTimeout(5 * time.Second)}
	get := pyManilaBCall{method: "GET", path: "/v2/shares/share-1", version: pyManilaBVersion}

	// Case-insensitive match after intermediate states; each poll is one GET.
	cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{"GET /v2/shares/share-1": {status: 200, bodies: []string{
		`{"share":{"id":"share-1","status":"creating"}}`, `{"share":{"id":"share-1","status":"creating"}}`, `{"share":{"id":"share-1","status":"AVAILABLE"}}`}}})
	got, err := shares.New(cloud.client(pyManilaBVersion)).WaitFor(ctx, resource.ID("share-1"), "available", fast...)
	if err != nil || got == nil || got.Status != "AVAILABLE" {
		t.Fatalf("share=%+v error=%v", got, err)
	}
	cloud.expect(get, get, get)

	// Python's default failure list is ['ERROR'] only; an explicit ERROR list
	// fails on error and keeps polling error_deleting until the caller stops.
	failing := newPyManilaBCloud(t, map[string]pyManilaBStub{"GET /v2/shares/share-1": {status: 200, bodies: []string{
		`{"share":{"id":"share-1","status":"creating"}}`, `{"share":{"id":"share-1","status":"error"}}`}}})
	_, err = shares.New(failing.client(pyManilaBVersion)).WaitFor(ctx, resource.ID("share-1"), "available", append(fast, resource.WithFailureStates("ERROR"))...)
	var failed *resource.FailedStateError
	if err == nil || !errors.As(err, &failed) || failed.Status != "error" {
		t.Fatalf("error=%v", err)
	}
	failing.expect(get, get)

	pending := newPyManilaBCloud(t, map[string]pyManilaBStub{"GET /v2/shares/share-1": {status: 200, body: `{"share":{"id":"share-1","status":"error_deleting"}}`}})
	_, err = shares.New(pending.client(pyManilaBVersion)).WaitFor(ctx, resource.ID("share-1"), "available",
		resource.WithPollInterval(time.Millisecond), resource.WithTimeout(60*time.Millisecond), resource.WithFailureStates("ERROR"))
	if err == nil || errors.As(err, &failed) {
		t.Fatalf("error_deleting must not be a failure under an ERROR-only list: %v", err)
	}
	if len(pending.snapshot()) < 2 {
		t.Fatalf("expected repeated polling, got %d requests", len(pending.snapshot()))
	}
}

func TestPyManilaBWaitForDeleteEndsOnNotFoundOrDeletedStatus(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	fast := []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithTimeout(5 * time.Second)}
	get := pyManilaBCall{method: "GET", path: "/v2/snapshots/snap-1", version: pyManilaBVersion}

	notFound := newPyManilaBCloud(t, map[string]pyManilaBStub{"GET /v2/snapshots/snap-1": {status: 404, body: `{}`}})
	if err := snapshots.New(notFound.client(pyManilaBVersion)).WaitForDeletion(ctx, resource.ID("snap-1"), fast...); err != nil {
		t.Fatal(err)
	}
	notFound.expect(get)

	deleted := newPyManilaBCloud(t, map[string]pyManilaBStub{"GET /v2/snapshots/snap-1": {status: 200, bodies: []string{
		`{"snapshot":{"id":"snap-1","status":"deleting"}}`, `{"snapshot":{"id":"snap-1","status":"Deleted"}}`}}})
	if err := snapshots.New(deleted.client(pyManilaBVersion)).WaitForDeletion(ctx, resource.ID("snap-1"), fast...); err != nil {
		t.Fatal(err)
	}
	deleted.expect(get, get)

	// Authorization failures end the wait instead of counting as deletion.
	denied := newPyManilaBCloud(t, map[string]pyManilaBStub{"GET /v2/snapshots/snap-1": {status: 403, body: `{}`}})
	if err := snapshots.New(denied.client(pyManilaBVersion)).WaitForDeletion(ctx, resource.ID("snap-1"), fast...); err == nil || !gophercloud.ResponseCodeIs(err, 403) {
		t.Fatalf("error=%v", err)
	}
	denied.expect(get)
}
