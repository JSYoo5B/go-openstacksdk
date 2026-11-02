package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/capabilities"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/defaulttypes"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/limits"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablevolumes"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/resourcefilters"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/services"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Shared calls that the list category admits but the block storage snapshot
// and backup lists never supported fail before HTTP.
func TestBlockStorageSnapshotBackupSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	client := cloud.Client("volumev3", "/volume/v3/")

	for _, call := range []blockstorage.VolumeSnapshotListOption{request.Header("X-Test", "1"), request.Query("status", "available"), request.Limit(1), request.Marker("m")} {
		entry, err := request.Entry(call)
		if err != nil {
			t.Fatal(err)
		}
		_, err = blockstorage.ListVolumeSnapshots(ctx, client, call)
		requireUnsupportedCall(t, err, entry.Kind.String())
		_, err = blockstorage.PrepareVolumeSnapshotListOptions(ctx, call)
		requireUnsupportedCall(t, err, entry.Kind.String())
		_, err = blockstorage.ListVolumeBackups(ctx, client, call)
		requireUnsupportedCall(t, err, entry.Kind.String())
		_, err = blockstorage.PrepareVolumeBackupListOptions(ctx, call)
		requireUnsupportedCall(t, err, entry.Kind.String())
	}
	_, err := blockstorage.ListVolumeBackups(ctx, client, request.Custom(func(*int) error { return nil }))
	requireUnsupportedCall(t, err, "Custom")
	if calls.Load() != 0 {
		t.Fatalf("HTTP calls = %d", calls.Load())
	}
}

// Shared list controls apply in caller order: the later call wins.
func TestBlockStorageSnapshotListSharedCallsCallerOrder(t *testing.T) {
	policy, err := blockstorage.PrepareVolumeSnapshotListOptions(context.Background(),
		request.Paginated(true), request.MaxItems(2), request.Microversion("3.1"), request.Headers(map[string]string{"A": "1"}),
		request.Paginated(false), request.MaxItems(5), request.Microversion("3.60"), request.Headers(map[string]string{"B": "2"}))
	if err != nil || policy.Paginated == nil || *policy.Paginated || policy.MaxItems == nil || *policy.MaxItems != 5 || policy.Microversion == nil || *policy.Microversion != "3.60" || len(policy.Headers) != 1 || policy.Headers["B"] != "2" {
		t.Fatal(policy, err)
	}
	if _, err := blockstorage.PrepareVolumeBackupListOptions(context.Background(), request.MaxItems(1), nil); err == nil {
		t.Fatal("nil option accepted")
	}
}

// Shared calls that a category admits but a block storage volume, snapshot or
// backup operation never supported fail before HTTP.
func TestBlockStorageV3VolumeSnapshotBackupSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	client := cloud.Client("volumev3", "/volume/v3/")

	// Metadata scopes (GetCall): request.Header only.
	scope, err := volumes.New(client).MetadataIn(ctx, resource.ID("v1"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = scope.Get(ctx, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = scope.DeleteKeys(ctx, []string{}, request.Microversion("3.1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = backups.New(client).PythonMetadata().FetchMetadata(ctx, "b1", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")

	// Record metadata (GetCall): request.Header only.
	_, err = volumes.New(client).FetchMetadataRecord(ctx, "v1", request.Microversion("3.1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = snapshots.New(client).FetchMetadataRecord(ctx, "s1", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")

	// Finds (FindCall): no single Header or Microversion.
	_, err = volumes.New(client).FindRecord(ctx, "v1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = snapshots.New(client).FindRecord(ctx, "s1", request.Microversion("3.1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = backups.New(client).FindRecord(ctx, "b1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	// Summary (GetCall): no single Header.
	_, err = volumes.New(client).SummaryRecord(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	// Writes (WriteCall).
	_, err = volumes.New(client).InitializeWithConnector(ctx, "v1", map[string]any{}, request.Microversion("3.1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = volumes.New(client).TerminateWithConnector(ctx, "v1", map[string]any{}, request.Field("x", 1))
	requireUnsupportedCall(t, err, "Field")
	_, err = snapshots.New(client).ManageRecord(ctx, "v1", nil, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = snapshots.New(client).UnmanageRecord(ctx, "s1", request.Field("x", 1))
	requireUnsupportedCall(t, err, "Field")

	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}
}

// request.Header merges into the headers collected by WithMetadataHeaders in
// caller order, canonicalized like a one-entry WithMetadataHeaders map.
func TestBlockStorageMetadataSharedHeaderMergesInCallerOrder(t *testing.T) {
	cloud := testcloud.New(t)
	var got http.Header
	cloud.Mux.HandleFunc("/volume/v3/volumes/v1/metadata", func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"metadata":{}}`))
	})
	scope, err := volumes.New(cloud.Client("volumev3", "/volume/v3/")).MetadataIn(context.Background(), resource.ID("v1"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = scope.Get(context.Background(),
		volumes.WithMetadataHeaders(map[string]string{"X-A": "1", "X-B": "1"}),
		request.Header("x-b", "2"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Get("X-A") != "1" || got.Get("X-B") != "2" {
		t.Fatalf("headers = %v", got)
	}
}

// Shared calls that a category admits but a Cinder group, group snapshot,
// consistency group or consistency group snapshot operation never supported
// fail before HTTP.
func TestBlockStorageGroupSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	client := cloud.Client("volumev3", "/volume/v3/")
	client.Microversion = "3.38"

	g := groups.New(client)
	_, err := g.GetRecord(ctx, "g1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = g.CreateRecord(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = g.CreateFromSourceRecord(ctx, request.Field("name", "x"))
	requireUnsupportedCall(t, err, "Field")
	_, err = g.UpdateRecord(ctx, "g1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	requireUnsupportedCall(t, g.DeleteRecord(ctx, "g1", request.Header("X-Test", "1")), "Header")
	for _, err := range g.ListRecords(ctx, request.Header("X-Test", "1")) {
		requireUnsupportedCall(t, err, "Header")
	}
	_, err = g.FindRecord(ctx, "g1", request.Query("name", "x"))
	requireUnsupportedCall(t, err, "Query")
	_, err = g.EnableReplicationRecord(ctx, "g1", request.Field("x", 1))
	requireUnsupportedCall(t, err, "Field")
	_, err = g.FailoverReplicationRecord(ctx, "g1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = g.ResetStatusRecord(ctx, "g1", "available", request.Field("x", 1))
	requireUnsupportedCall(t, err, "Field")

	gs := groupsnapshots.New(client)
	_, err = gs.GetRecord(ctx, "s1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = gs.CreateRecord(ctx, map[string]any{"group_id": "g1"}, request.Field("name", "x"))
	requireUnsupportedCall(t, err, "Field")
	_, err = gs.DeleteRecord(ctx, "s1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	for _, err := range gs.ListRecords(ctx, request.Header("X-Test", "1")) {
		requireUnsupportedCall(t, err, "Header")
	}
	_, err = gs.FindRecord(ctx, "s1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = gs.ResetStatusRecord(ctx, "s1", "available", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	cg := consistencygroups.New(client)
	_, err = cg.GetRecord(ctx, "c1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = cg.CreateFromSourceRecord(ctx, request.Field("name", "x"))
	requireUnsupportedCall(t, err, "Field")
	requireUnsupportedCall(t, cg.DeleteRecord(ctx, "c1", request.IgnoreMissing(true)), "IgnoreMissing")
	for _, err := range cg.ListRecords(ctx, request.Header("X-Test", "1")) {
		requireUnsupportedCall(t, err, "Header")
	}
	_, err = cg.FindRecord(ctx, "c1", request.Query("name", "x"))
	requireUnsupportedCall(t, err, "Query")

	cgs := consistencygroupsnapshots.New(client)
	_, err = cgs.GetRecord(ctx, "s1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = cgs.CreateRecord(ctx, map[string]any{"consistencygroup_id": "c1"}, request.Field("name", "x"))
	requireUnsupportedCall(t, err, "Field")
	_, err = cgs.DeleteRecord(ctx, "s1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	for _, err := range cgs.ListRecords(ctx, request.Header("X-Test", "1")) {
		requireUnsupportedCall(t, err, "Header")
	}
	_, err = cgs.FindRecord(ctx, "s1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	if n := calls.Load(); n != 0 {
		t.Fatalf("unsupported calls reached HTTP %d times", n)
	}
}

// Shared calls that a category admits but a block storage administration
// operation never supported fail before HTTP.
func TestBlockStorageAdminSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	client := cloud.Client("volume", "/volume/v3/")
	client.Microversion = "3.70"
	name := "c1"
	first := func(seq func(func(*manageablevolumes.ManageableRecord, error) bool)) (err error) {
		for _, e := range seq {
			return e
		}
		return nil
	}

	// GetCall and GetQueryCall.
	_, err := grouptypes.New(client).Get(ctx, "g1", request.Microversion("3.11"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = defaulttypes.New(client).Get(ctx, "p1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = capabilities.New(client).GetRecord(ctx, "h1", request.Microversion("3.0"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = messages.New(client).GetRecord(ctx, "m1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = transfers.New(client).GetRecord(ctx, "t1", request.Microversion("3.55"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = resourcefilters.New(client).AllRecords(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = services.New(client).GetLogLevels(ctx, request.Microversion("3.32"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = limits.New(client).Fetch(ctx, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	scope, err := quotasets.New(client).InProject(ctx, resource.ID("p1"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = scope.GetRecord(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = scope.RevertRecord(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = quotasets.New(client).GetQuotaClassRecord(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	// WriteCall.
	_, err = grouptypes.New(client).Create(ctx, request.Microversion("3.11"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = clusters.New(client).Enable(ctx, clusters.Target{Name: &name}, request.Field("x", 1))
	requireUnsupportedCall(t, err, "Field")
	_, err = services.New(client).Enable(ctx, services.ServiceTarget{}, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = transfers.New(client).CreateRecord(ctx, request.Microversion("3.55"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = transfers.New(client).AcceptRecord(ctx, "t1", request.Field("auth_key", "k"))
	requireUnsupportedCall(t, err, "Field")
	_, err = attachments.New(client).CreateRecord(ctx, "v1", request.Microversion("3.54"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = volumetypes.New(client).CreateEncryptionWithAttributes(ctx, "vt1", request.Field("cipher", "aes"))
	requireUnsupportedCall(t, err, "Field")
	_, err = quotasets.New(client).UpdateQuotaClassRecord(ctx, "default", request.Field("volumes", 1))
	requireUnsupportedCall(t, err, "Field")

	// DeleteCall.
	requireUnsupportedCall(t, grouptypes.New(client).Delete(ctx, "g1", request.Microversion("3.11")), "Microversion")
	requireUnsupportedCall(t, transfers.New(client).DeleteRecord(ctx, "t1", request.Microversion("3.55")), "Microversion")
	_, err = scope.Reset(ctx, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")

	// FindCall.
	_, err = grouptypes.New(client).Find(ctx, "g1", request.Query("name", "x"))
	requireUnsupportedCall(t, err, "Query")
	_, err = clusters.New(client).Find(ctx, "c1", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = services.New(client).FindRecord(ctx, "s1", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = transfers.New(client).FindRecord(ctx, "t1", request.Query("name", "x"))
	requireUnsupportedCall(t, err, "Query")
	_, err = qos.New(client).FindRecord(ctx, "q1", request.Microversion("3.0"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = volumetypes.New(client).FindRecord(ctx, "vt1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	// ListCall.
	for _, err := range defaulttypes.New(client).List(ctx, request.Limit(1)) {
		requireUnsupportedCall(t, err, "Limit")
	}
	for _, err := range grouptypes.New(client).List(ctx, request.Microversion("3.11")) {
		requireUnsupportedCall(t, err, "Microversion")
	}
	for _, err := range clusters.New(client).List(ctx, request.Header("X-Test", "1")) {
		requireUnsupportedCall(t, err, "Header")
	}
	for _, err := range transfers.New(client).ListRecords(ctx, request.Microversion("3.55")) {
		requireUnsupportedCall(t, err, "Microversion")
	}
	for _, err := range messages.New(client).ListRecords(ctx, request.Header("X-Test", "1")) {
		requireUnsupportedCall(t, err, "Header")
	}
	for _, err := range attachments.New(client).ListRecords(ctx, request.Header("X-Test", "1")) {
		requireUnsupportedCall(t, err, "Header")
	}
	requireUnsupportedCall(t, first(manageablevolumes.New(client).ListRecords(ctx, request.Marker("m"))), "Marker")

	if n := calls.Load(); n != 0 {
		t.Fatalf("calls = %d", n)
	}
}

// request.Paginated keeps caller order on a translated list: a later
// Paginated(false) stops after the first page even after Paginated(true).
func TestBlockStorageGroupTypesPaginatedCallerOrder(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /volume/v3/group_types", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"group_types":[{"id":"g1","name":"one"}],"group_types_links":[{"rel":"next","href":"`+cloud.Server.URL+`/volume/v3/group_types?marker=g1"}]}`)
	})
	client := cloud.Client("volume", "/volume/v3/")
	client.Microversion = "3.11"
	var got int
	for _, err := range grouptypes.New(client).List(context.Background(), request.Paginated(true), request.Paginated(false)) {
		if err != nil {
			t.Fatal(err)
		}
		got++
	}
	if got != 1 || calls.Load() != 1 {
		t.Fatal(got, calls.Load())
	}
}
