package api_test

import (
	"context"
	"iter"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/limits"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/quotaclasssets"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/resourcelocks"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/services"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharegroups"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharegroupsnapshots"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareinstances"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharenetworks"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharenetworksubnets"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shares"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharetransfers"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/snapshotinstances"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/snapshots"
)

func manilaSharedCallsCloud(t *testing.T) (*testcloud.Cloud, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 500, `{}`)
	})
	return cloud, &calls
}

func requireUnsupportedSeq[T any](t *testing.T, seq iter.Seq2[T, error], kind string) {
	t.Helper()
	var err error
	for _, err = range seq {
		break
	}
	requireUnsupportedCall(t, err, kind)
}

// Shared calls that a category admits but a hand-written Manila operation
// never supported fail before HTTP.
func TestSharedFileSystemsSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	cloud, calls := manilaSharedCallsCloud(t)
	client := r2ManilaAClient(cloud, "2.82")

	// Single-response lists take Query and MaxItems only.
	for _, call := range []request.ListCall{request.Limit(1), request.Marker("m"), request.Paginated(false), request.Header("X-Test", "1"), request.Headers(nil), request.Microversion("2.82")} {
		entry, err := request.Entry(call)
		if err != nil {
			t.Fatal(err)
		}
		requireUnsupportedSeq(t, resourcelocks.New(client).List(ctx, call), entry.Kind.String())
		requireUnsupportedSeq(t, sharegroups.New(client).ListDetail(ctx, call), entry.Kind.String())
		requireUnsupportedSeq(t, sharegroupsnapshots.New(client).List(ctx, call), entry.Kind.String())
		requireUnsupportedSeq(t, shareinstances.New(client).List(ctx, call), entry.Kind.String())
		requireUnsupportedSeq(t, snapshotinstances.New(client).List(ctx, call), entry.Kind.String())
	}
	subnets, err := sharenetworksubnets.New(client).InNetwork(ctx, "net")
	if err != nil {
		t.Fatal(err)
	}
	requireUnsupportedSeq(t, subnets.List(ctx, request.Limit(1)), "Limit")

	// Summary lists take Query only.
	requireUnsupportedSeq(t, shares.New(client).List(ctx, request.MaxItems(1)), "MaxItems")
	requireUnsupportedSeq(t, snapshots.New(client).List(ctx, request.Header("X-Test", "1")), "Header")
	requireUnsupportedSeq(t, sharenetworks.New(client).List(ctx, request.Limit(1)), "Limit")

	// FindByNameOrID takes Query and IgnoreMissing.
	_, err = shares.New(client).FindByNameOrID(ctx, "x", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = sharetransfers.New(client).FindByNameOrID(ctx, "x", request.Microversion("2.82"))
	requireUnsupportedCall(t, err, "Microversion")

	// Limits Get takes Query only.
	_, err = limits.New(client).Get(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	// Writes take Field only.
	_, err = sharegroups.New(client).Create(ctx, sharegroups.CreateOpts{}, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = sharegroupsnapshots.New(client).Update(ctx, "id", sharegroupsnapshots.UpdateOpts{}, request.Microversion("2.82"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = resourcelocks.New(client).Create(ctx, resourcelocks.CreateOpts{ResourceID: "s", ResourceType: "share"}, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = subnets.Create(ctx, sharenetworksubnets.CreateOpts{}, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = shares.New(client).Manage(ctx, shares.ManageOpts{}, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = snapshots.New(client).Manage(ctx, snapshots.ManageOpts{}, request.Microversion("2.82"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = services.New(client).Disable(ctx, "host", "manila-share", services.DisableOpts{}, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = sharenetworks.New(client).UpdateMembers(ctx, "net", sharenetworks.UpdateMembersOpts{}, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = shareaccessrules.New(client).AllowAccessAction(ctx, "share", shareaccessrules.AllowActionOpts{}, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	// Deny actions take IgnoreMissing only.
	_, err = shareaccessrules.New(client).DenyAccessAction(ctx, "share", "rule", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	if n := calls.Load(); n != 0 {
		t.Fatalf("unsupported calls sent %d requests", n)
	}
}

// The access rule scope and the quota scopes keep their own categories.
func TestSharedFileSystemsScopedSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	cloud, calls := manilaSharedCallsCloud(t)
	client := shareAccessClient(cloud, "2.82")
	scope, err := shareaccessrules.New(client).InShare(ctx, resource.ID("share-id"))
	if err != nil {
		t.Fatal(err)
	}
	calls.Store(0)

	// Get takes Header; Allow Header and Field; Deny Header and
	// IgnoreMissing (WithDenyField carries fields); List Header, Query,
	// MaxItems and Paginated.
	_, err = scope.Get(ctx, "rule", request.Microversion("2.82"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = scope.Get(ctx, "rule", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = scope.Allow(ctx, shareaccessrules.AllowOpts{AccessType: "ip", AccessTo: "192.0.2.1"}, request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	err = scope.Deny(ctx, "rule", request.Microversion("2.82"))
	requireUnsupportedCall(t, err, "Microversion")
	for _, call := range []request.ListCall{request.Limit(1), request.Marker("m"), request.Headers(nil), request.Microversion("2.82")} {
		entry, err := request.Entry(call)
		if err != nil {
			t.Fatal(err)
		}
		_, err = scope.All(ctx, call)
		requireUnsupportedCall(t, err, entry.Kind.String())
	}
	if n := calls.Load(); n != 0 {
		t.Fatalf("unsupported access rule calls sent %d requests", n)
	}

	// Quota updates take Field; quota reset takes IgnoreMissing.
	quotaClient := r2ManilaAClient(cloud, "2.62")
	project, err := quotasets.New(quotaClient).InProject(ctx, resource.ID("project"))
	if err != nil {
		t.Fatal(err)
	}
	calls.Store(0)
	_, err = project.Update(ctx, quotasets.UpdateOpts{}, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = project.Reset(ctx, request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = project.Reset(ctx, nil)
	if err == nil || calls.Load() != 0 {
		t.Fatalf("nil reset option err=%v calls=%d", err, calls.Load())
	}
	class, err := quotaclasssets.New(quotaClient).InClass(ctx, "default")
	if err != nil {
		t.Fatal(err)
	}
	_, err = class.Update(ctx, quotaclasssets.UpdateOpts{}, request.Microversion("2.62"))
	requireUnsupportedCall(t, err, "Microversion")
	if n := calls.Load(); n != 0 {
		t.Fatalf("unsupported quota calls sent %d requests", n)
	}
}

// Shared calls keep caller order: the later Query, MaxItems, Field and
// IgnoreMissing win.
func TestSharedFileSystemsSharedCallsCallerOrder(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/resource-locks", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.RawQuery; got != "vendor=b" {
			t.Errorf("query = %q", got)
		}
		testcloud.JSON(w, 200, `{"resource_locks":[{"id":"a"},{"id":"b"},{"id":"c"}]}`)
	})
	cloud.Mux.HandleFunc("POST /v2/share-groups", func(w http.ResponseWriter, r *http.Request) {
		if got := r2ManilaARead(t, r); got != `{"share_group":{"vendor":"later"}}` {
			t.Errorf("body = %s", got)
		}
		testcloud.JSON(w, 202, `{"share_group":{"id":"g"}}`)
	})
	cloud.Mux.HandleFunc("POST /v2/shares/share/action", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	client := r2ManilaAClient(cloud, "2.81")
	var ids []string
	for value, err := range resourcelocks.New(client).List(ctx, request.Query("vendor", "a"), request.MaxItems(1), request.Query("vendor", "b"), request.MaxItems(2)) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if len(ids) != 2 {
		t.Fatalf("ids = %v", ids)
	}
	groups := sharegroups.New(r2ManilaAClient(cloud, "2.55"))
	if _, err := groups.Create(ctx, sharegroups.CreateOpts{}, request.Field("vendor", "earlier"), request.Field("vendor", "later")); err != nil {
		t.Fatal(err)
	}
	rules := shareaccessrules.New(client)
	if result, err := rules.DenyAccessAction(ctx, "share", "rule", request.IgnoreMissing(false), request.IgnoreMissing(true)); err != nil || result != nil {
		t.Fatalf("ignored missing result=%v err=%v", result, err)
	}
	if _, err := rules.DenyAccessAction(ctx, "share", "rule", request.IgnoreMissing(true), request.IgnoreMissing(false)); err == nil {
		t.Fatal("strict deny must report 404")
	}
}
