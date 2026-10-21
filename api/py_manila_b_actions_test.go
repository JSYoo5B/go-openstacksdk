package api_test

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/replicas"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shares"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/snapshots"
	"github.com/gophercloud/gophercloud/v2"
)

func pyManilaBPost(path, body string) pyManilaBCall {
	return pyManilaBCall{method: "POST", path: path, body: body, version: pyManilaBVersion}
}

// Python ShareReplica/ShareSnapshot/Share action helpers POST one named body
// to /<collection>/{id}/action without a prior GET. The reset and resync
// bodies match Go exactly; Go Promote sends an empty object where Python sends
// null.
func TestPyManilaBReplicaAndStatusActionsMatchPythonBodies(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cases := []struct {
		name, path, body string
		call             func(*gophercloud.ServiceClient) error
	}{
		{"reset_share_replica_status", "/v2/share-replicas/rep-1/action", `{"reset_status":{"status":"error"}}`, func(c *gophercloud.ServiceClient) error {
			return replicas.New(c).ResetStatus(ctx, "rep-1", replicas.ResetStatusOpts{Status: "error"})
		}},
		{"reset_share_replica_state", "/v2/share-replicas/rep-1/action", `{"reset_replica_state":{"replica_state":"out_of_sync"}}`, func(c *gophercloud.ServiceClient) error {
			return replicas.New(c).ResetState(ctx, "rep-1", replicas.ResetStateOpts{State: "out_of_sync"})
		}},
		{"resync_share_replica", "/v2/share-replicas/rep-1/action", `{"resync":null}`, func(c *gophercloud.ServiceClient) error {
			return replicas.New(c).Resync(ctx, "rep-1")
		}},
		{"promote_share_replica", "/v2/share-replicas/rep-1/action", `{"promote":{}}`, func(c *gophercloud.ServiceClient) error {
			return replicas.New(c).Promote(ctx, "rep-1", replicas.PromoteOpts{})
		}},
		{"reset_share_snapshot_status", "/v2/snapshots/snap-1/action", `{"reset_status":{"status":"error"}}`, func(c *gophercloud.ServiceClient) error {
			return snapshots.New(c).ResetStatus(ctx, "snap-1", snapshots.ResetStatusOpts{Status: "error"})
		}},
		{"reset_share_status", "/v2/shares/share-1/action", `{"reset_status":{"status":"error"}}`, func(c *gophercloud.ServiceClient) error {
			return shares.New(c).ResetStatus(ctx, "share-1", shares.ResetStatusOpts{Status: "error"})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{"POST " + tc.path: {status: 202}})
			if err := tc.call(cloud.client(pyManilaBVersion)); err != nil {
				t.Fatal(err)
			}
			cloud.expect(pyManilaBPost(tc.path, tc.body))
		})
	}
}

// Python resize_share GETs the share, then sends extend (with force only when
// requested) or shrink, or nothing when the size matches or the direction is
// disabled. shares.Resize reproduces that decision table.
func TestPyManilaBResizeMatchesPythonDecisionTable(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	get := pyManilaBCall{method: "GET", path: "/v2/shares/share-1", version: pyManilaBVersion}
	action := func(body string) pyManilaBCall { return pyManilaBPost("/v2/shares/share-1/action", body) }
	cases := []struct {
		name    string
		size    int
		opts    []shares.ResizeOption
		want    shares.ResizeAction
		actions []pyManilaBCall
	}{
		{"extend", 5, nil, shares.ResizeExtend, []pyManilaBCall{action(`{"extend":{"new_size":10}}`)}},
		{"extend with force", 5, []shares.ResizeOption{shares.WithResizeForce()}, shares.ResizeExtend, []pyManilaBCall{action(`{"extend":{"new_size":10,"force":true}}`)}},
		{"extend disabled", 5, []shares.ResizeOption{shares.WithResizeNoExtend()}, shares.ResizeNone, nil},
		{"shrink", 20, nil, shares.ResizeShrink, []pyManilaBCall{action(`{"shrink":{"new_size":10}}`)}},
		{"shrink ignores force", 20, []shares.ResizeOption{shares.WithResizeForce()}, shares.ResizeShrink, []pyManilaBCall{action(`{"shrink":{"new_size":10}}`)}},
		{"shrink disabled", 20, []shares.ResizeOption{shares.WithResizeNoShrink()}, shares.ResizeNone, nil},
		{"same size", 10, nil, shares.ResizeNone, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{
				"GET /v2/shares/share-1":         {status: 200, body: `{"share":{"id":"share-1","size":` + strconv.Itoa(tc.size) + `}}`},
				"POST /v2/shares/share-1/action": {status: 202},
			})
			got, err := shares.New(cloud.client(pyManilaBVersion)).Resize(ctx, "share-1", 10, tc.opts...)
			if err != nil || got != tc.want {
				t.Fatalf("action=%q error=%v", got, err)
			}
			cloud.expect(append([]pyManilaBCall{get}, tc.actions...)...)
		})
	}
}

func TestPyManilaBResizeFailuresStopBeforeAction(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{"GET /v2/shares/share-1": {status: 404, body: `{}`}})
	action, err := shares.New(cloud.client(pyManilaBVersion)).Resize(ctx, "share-1", 10)
	if err == nil || !gophercloud.ResponseCodeIs(err, 404) || action != shares.ResizeNone {
		t.Fatalf("action=%q error=%v", action, err)
	}
	cloud.expect(pyManilaBCall{method: "GET", path: "/v2/shares/share-1", version: pyManilaBVersion})

	rejected := newPyManilaBCloud(t, nil)
	api := shares.New(rejected.client(pyManilaBVersion))
	if _, err := api.Resize(ctx, "", 10); err == nil {
		t.Fatal("empty ID accepted")
	}
	if _, err := api.Resize(ctx, "share-1", 10, nil); err == nil {
		t.Fatal("nil option accepted")
	}
	rejected.expect()

	// A rejected extend (the service answers 400) is returned and not replayed.
	denied := newPyManilaBCloud(t, map[string]pyManilaBStub{
		"GET /v2/shares/share-1":         {status: 200, body: `{"share":{"id":"share-1","size":1}}`},
		"POST /v2/shares/share-1/action": {status: 400, body: `{"badRequest":{"message":"no"}}`},
	})
	if _, err := shares.New(denied.client(pyManilaBVersion)).Resize(ctx, "share-1", 10); err == nil || !gophercloud.ResponseCodeIs(err, 400) {
		t.Fatalf("error=%v", err)
	}
	if got := len(denied.snapshot()); got != 2 {
		t.Fatalf("requests=%d", got)
	}
}

// Python revert_share_to_snapshot, unmanage_share, soft_delete_share and
// restore_share first GET the share (a 404 therefore precedes the action) and
// then POST one action. Go sends only the action: the pre-fetch carries no
// request data.
func TestPyManilaBShareActionsSendOnlyTheActionBody(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := "/v2/shares/share-1/action"
	cases := []struct {
		name, body string
		call       func(*shares.API) error
	}{
		{"revert_share_to_snapshot", `{"revert":{"snapshot_id":"snap-1"}}`, func(a *shares.API) error {
			return a.Revert(ctx, "share-1", shares.RevertOpts{SnapshotID: "snap-1"})
		}},
		{"unmanage_share", `{"unmanage":null}`, func(a *shares.API) error { return a.Unmanage(ctx, "share-1") }},
		{"soft_delete_share", `{"soft_delete":null}`, func(a *shares.API) error { return a.SoftDelete(ctx, "share-1") }},
		{"restore_share", `{"restore":null}`, func(a *shares.API) error { return a.Restore(ctx, "share-1") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{"POST " + path: {status: 202}})
			if err := tc.call(shares.New(cloud.client(pyManilaBVersion))); err != nil {
				t.Fatal(err)
			}
			cloud.expect(pyManilaBPost(path, tc.body))

			for _, status := range []int{200, 204} {
				odd := newPyManilaBCloud(t, map[string]pyManilaBStub{"POST " + path: {status: status}})
				if err := tc.call(shares.New(odd.client(pyManilaBVersion))); err == nil || !gophercloud.ResponseCodeIs(err, status) {
					t.Fatalf("status %d accepted: %v", status, err)
				}
			}
		})
	}
	rejected := newPyManilaBCloud(t, nil)
	api := shares.New(rejected.client(pyManilaBVersion))
	if api.SoftDelete(ctx, "") == nil || api.Restore(ctx, "") == nil {
		t.Fatal("empty ID accepted")
	}
	rejected.expect()
}

// Python manage_share and manage_share_snapshot POST the required members plus
// arbitrary **params to /manage and read the resource from the response.
func TestPyManilaBManageSharesAndSnapshotsSendBodiesAndKeepFalseAndEmpty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	str := func(s string) *string { return &s }
	no := false
	cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{
		"POST /v2/shares/manage":    {status: 202, body: `{"share":{"id":"share-9","status":"manage_starting","share_proto":"NFS"}}`},
		"POST /v2/snapshots/manage": {status: 202, body: `{"snapshot":{"id":"snap-9","status":"manage_starting","share_id":"share-1"}}`},
	})
	client := cloud.client(pyManilaBVersion)
	share, err := shares.New(client).Manage(ctx, shares.ManageOpts{
		Protocol: "NFS", ExportPath: "10.0.0.1:/vol", ServiceHost: "host@backend#pool",
		Name: str(""), IsPublic: &no, DriverOptions: map[string]string{},
	}, shares.WithManageField("vendor", map[string]any{"tier": 1}))
	if err != nil || share == nil || share.ID != "share-9" || share.Status != "manage_starting" {
		t.Fatalf("share=%+v error=%v", share, err)
	}
	snapshot, err := snapshots.New(client).Manage(ctx, snapshots.ManageOpts{
		ShareID: "share-1", ProviderLocation: "loc", Description: str(""), DriverOptions: map[string]string{},
	})
	if err != nil || snapshot == nil || snapshot.ID != "snap-9" {
		t.Fatalf("snapshot=%+v error=%v", snapshot, err)
	}
	cloud.expect(
		pyManilaBPost("/v2/shares/manage", `{"share":{"protocol":"NFS","export_path":"10.0.0.1:/vol","service_host":"host@backend#pool","name":"","is_public":false,"driver_options":{},"vendor":{"tier":1}}}`),
		pyManilaBPost("/v2/snapshots/manage", `{"snapshot":{"share_id":"share-1","provider_location":"loc","description":"","driver_options":{}}}`),
	)

	// Omitted optionals stay omitted; required and core inputs fail before HTTP.
	bare := newPyManilaBCloud(t, map[string]pyManilaBStub{"POST /v2/shares/manage": {status: 200, body: `{"share":{"id":"s"}}`}})
	if _, err := shares.New(bare.client("")).Manage(ctx, shares.ManageOpts{Protocol: "NFS", ExportPath: "p", ServiceHost: "h"}); err != nil {
		t.Fatal(err)
	}
	bare.expect(pyManilaBCall{method: "POST", path: "/v2/shares/manage", body: `{"share":{"protocol":"NFS","export_path":"p","service_host":"h"}}`})

	rejected := newPyManilaBCloud(t, nil)
	api := shares.New(rejected.client(pyManilaBVersion))
	if _, err := api.Manage(ctx, shares.ManageOpts{Protocol: "NFS", ExportPath: "p"}); err == nil {
		t.Fatal("missing service host accepted")
	}
	if _, err := api.Manage(ctx, shares.ManageOpts{Protocol: "NFS", ExportPath: "p", ServiceHost: "h"}, shares.WithManageField("name", "x")); err == nil {
		t.Fatal("core extension accepted")
	}
	if _, err := snapshots.New(rejected.client(pyManilaBVersion)).Manage(ctx, snapshots.ManageOpts{ShareID: "s"}); err == nil {
		t.Fatal("missing provider location accepted")
	}
	if _, err := snapshots.New(rejected.client(pyManilaBVersion)).Manage(ctx, snapshots.ManageOpts{ShareID: "s", ProviderLocation: "l"}, snapshots.WithManageField("share_id", "x")); err == nil {
		t.Fatal("core snapshot extension accepted")
	}
	rejected.expect()

	// Anything but 200/202 is an error; the manage is never replayed.
	odd := newPyManilaBCloud(t, map[string]pyManilaBStub{"POST /v2/shares/manage": {status: 204}})
	if _, err := shares.New(odd.client("")).Manage(ctx, shares.ManageOpts{Protocol: "NFS", ExportPath: "p", ServiceHost: "h"}); err == nil || !gophercloud.ResponseCodeIs(err, 204) {
		t.Fatalf("error=%v", err)
	}
	if len(odd.snapshot()) != 1 {
		t.Fatalf("replayed: %v", odd.snapshot())
	}
}

func TestPyManilaBUnmanageSnapshotPostsActionWithoutFetch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{"POST /v2/snapshots/snap-1/action": {status: 202}})
	if err := snapshots.New(cloud.client(pyManilaBVersion)).Unmanage(ctx, "snap-1"); err != nil {
		t.Fatal(err)
	}
	cloud.expect(pyManilaBPost("/v2/snapshots/snap-1/action", `{"unmanage":null}`))
	if err := snapshots.New(cloud.client("")).Unmanage(ctx, ""); err == nil {
		t.Fatal("empty ID accepted")
	}
	if len(cloud.snapshot()) != 1 {
		t.Fatal("empty ID reached the server")
	}
	odd := newPyManilaBCloud(t, map[string]pyManilaBStub{"POST /v2/snapshots/snap-1/action": {status: 200}})
	if err := snapshots.New(odd.client("")).Unmanage(ctx, "snap-1"); err == nil || !gophercloud.ResponseCodeIs(err, 200) {
		t.Fatalf("error=%v", err)
	}
}

// Python set_share_metadata POSTs {"metadata": {...}} to merge, or PUTs it with
// replace=True (and update_share_metadata forwards to it). Go SetMetadata is
// the POST form and ReplaceMetadata the PUT form; both keep empty maps and
// empty values.
func TestPyManilaBShareMetadataMergeAndReplaceUsePostAndPut(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := "/v2/shares/share-1/metadata"
	cloud := newPyManilaBCloud(t, map[string]pyManilaBStub{
		"POST " + path: {status: 200, body: `{"metadata":{"a":"1","empty":""}}`},
		"PUT " + path:  {status: 200, body: `{"metadata":{"a":"1"}}`},
	})
	api := shares.New(cloud.client(pyManilaBVersion))
	merged, err := api.SetMetadata(ctx, "share-1", shares.SetMetadataOpts{Metadata: map[string]string{"a": "1", "empty": ""}})
	if err != nil || !reflect.DeepEqual(merged, map[string]string{"a": "1", "empty": ""}) {
		t.Fatalf("merged=%v error=%v", merged, err)
	}
	replaced, err := api.ReplaceMetadata(ctx, "share-1", map[string]string{"a": "1"})
	if err != nil || !reflect.DeepEqual(replaced, map[string]string{"a": "1"}) {
		t.Fatalf("replaced=%v error=%v", replaced, err)
	}
	if _, err := api.SetMetadata(ctx, "share-1", shares.SetMetadataOpts{Metadata: map[string]string{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.ReplaceMetadata(ctx, "share-1", nil); err != nil {
		t.Fatal(err)
	}
	cloud.expect(
		pyManilaBPost(path, `{"metadata":{"a":"1","empty":""}}`),
		pyManilaBCall{method: "PUT", path: path, body: `{"metadata":{"a":"1"}}`, version: pyManilaBVersion},
		pyManilaBPost(path, `{"metadata":{}}`),
		pyManilaBCall{method: "PUT", path: path, body: `{"metadata":{}}`, version: pyManilaBVersion},
	)
}
