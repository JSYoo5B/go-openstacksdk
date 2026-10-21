package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/services"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharenetworksubnets"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/snapshots"
)

const r2ManilaASubnetJSON = `{"share_network_subnet":{"id":"sub1","share_network_id":"net1","mtu":1500,"segmentation_id":null,"ip_version":4,"metadata":{"k":"v"}}}`

func r2ManilaAScope(t *testing.T, cloud *testcloud.Cloud, version string) *sharenetworksubnets.Scope {
	t.Helper()
	scope, err := sharenetworksubnets.New(r2ManilaAClient(cloud, version)).InNetwork(context.Background(), "net1")
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestR2ManilaASubnetCreateUsesHyphenatedEnvelopeAndRoutes(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /v2/share-networks/net1/subnets", func(w http.ResponseWriter, r *http.Request) {
		r2ManilaAVersion(t, r, "2.78")
		if got := r2ManilaARead(t, r); got != `{"share-network-subnet":{"availability_zone":"","metadata":{"a":""},"neutron_net_id":"n"}}` {
			t.Errorf("body = %s", got)
		}
		testcloud.JSON(w, 202, r2ManilaASubnetJSON)
	})
	cloud.Mux.HandleFunc("GET /v2/share-networks/net1/subnets", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"share_network_subnets":[{"id":"sub1"},{"id":"sub2"}]}`)
	})
	cloud.Mux.HandleFunc("GET /v2/share-networks/net1/subnets/sub1", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, r2ManilaASubnetJSON) })
	var del atomic.Int32
	del.Store(202)
	cloud.Mux.HandleFunc("DELETE /v2/share-networks/net1/subnets/sub1", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, int(del.Load()), `{}`) })
	scope := r2ManilaAScope(t, cloud, "2.78")
	ctx := context.Background()
	v, err := scope.Create(ctx, sharenetworksubnets.CreateOpts{NeutronNetID: request.Present("n"), AvailabilityZone: request.Present(""), Metadata: map[string]string{"a": ""}})
	if err != nil || v.ID != "sub1" || string(v.MTU) != "1500" || v.SegmentationID != nil || string(v.Evidence.Body["segmentation_id"]) != "null" || v.Metadata["k"] != "v" {
		t.Fatalf("%+v %v", v, err)
	}
	n := 0
	for _, err := range scope.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 2 {
		t.Fatal(n)
	}
	if _, err := scope.Get(ctx, "sub1"); err != nil {
		t.Fatal(err)
	}
	if err := scope.Remove(ctx, resource.ID("sub1")); err != nil {
		t.Fatal(err)
	}
	del.Store(404)
	if err := scope.Remove(ctx, resource.ID("sub1")); err != nil {
		t.Errorf("default ignore_missing: %v", err)
	}
	if err := scope.Remove(ctx, resource.ID("sub1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Errorf("strict: %v", err)
	}
	r2ManilaACode(t, scope.Delete(ctx, "sub1"), 404)
}

func TestR2ManilaASubnetLocalChecksSendNothing(t *testing.T) {
	cloud := testcloud.New(t)
	calls := r2ManilaAGuard(t, cloud)
	ctx := context.Background()
	if _, err := sharenetworksubnets.New(r2ManilaAClient(cloud, "2.78")).InNetwork(ctx, "a/b"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Error(err)
	}
	if _, err := r2ManilaAScope(t, cloud, "2.50").Get(ctx, "s"); !errors.Is(err, resource.ErrUnsupported) {
		t.Error(err)
	}
	if _, err := r2ManilaAScope(t, cloud, "2.77").GetMetadata(ctx, "s"); !errors.Is(err, resource.ErrUnsupported) {
		t.Error("metadata needs 2.78:", err)
	}
	scope := r2ManilaAScope(t, cloud, "2.78")
	for _, key := range []string{"neutron_net_id", "metadata", "id", "share_network_id", "share-network-subnet"} {
		if _, err := scope.Create(ctx, sharenetworksubnets.CreateOpts{}, sharenetworksubnets.WithCreateField(key, 1)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", key, err)
		}
	}
	if _, err := scope.GetMetadatum(ctx, "s", ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Error(err)
	}
	if err := scope.DeleteMetadata(ctx, "s", []string{"a", ""}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Error(err)
	}
	if _, err := scope.Create(r2ManilaACanceled(), sharenetworksubnets.CreateOpts{}); !errors.Is(err, context.Canceled) {
		t.Error(err)
	}
	if calls.Load() != 0 {
		t.Fatal("request sent")
	}
}

func TestR2ManilaASubnetMetadataReadsWritesAndDeleteOrder(t *testing.T) {
	cloud := testcloud.New(t)
	base := "/v2/share-networks/net1/subnets/sub1/metadata"
	cloud.Mux.HandleFunc("GET "+base, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"metadata":{"e":"","k":"v"}}`) })
	cloud.Mux.HandleFunc("GET "+base+"/a%2Fb", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"meta":{"a/b":"x"}}`) })
	cloud.Mux.HandleFunc("GET "+base+"/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	var bodies []string
	write := func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, r.Method+" "+r2ManilaARead(t, r))
		testcloud.JSON(w, 200, `{"metadata":{"k":"v"}}`)
	}
	cloud.Mux.HandleFunc("POST "+base, write)
	cloud.Mux.HandleFunc("PUT "+base, write)
	var deleted []string
	cloud.Mux.HandleFunc("DELETE "+base+"/{key}", func(w http.ResponseWriter, r *http.Request) {
		deleted = append(deleted, r.PathValue("key"))
		if r.PathValue("key") == "bad" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		testcloud.JSON(w, 200, `{}`)
	})
	scope := r2ManilaAScope(t, cloud, "2.78")
	ctx := context.Background()
	if m, err := scope.GetMetadata(ctx, "sub1"); err != nil || !reflect.DeepEqual(m, map[string]string{"e": "", "k": "v"}) {
		t.Fatal(m, err)
	}
	if m, err := scope.GetMetadatum(ctx, "sub1", "a/b"); err != nil || m["a/b"] != "x" {
		t.Fatal(m, err)
	}
	_, err := scope.GetMetadatum(ctx, "sub1", "missing")
	r2ManilaACode(t, err, 404)
	if _, err := scope.SetMetadata(ctx, "sub1", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := scope.ReplaceMetadata(ctx, "sub1", map[string]string{"k": ""}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(bodies, []string{`POST {"metadata":{}}`, `PUT {"metadata":{"k":""}}`}) {
		t.Fatal(bodies)
	}
	if err := scope.DeleteMetadata(ctx, "sub1", []string{"b", "a", "b", "bad", "never"}); err == nil {
		t.Fatal("404 must stop the loop")
	}
	if !reflect.DeepEqual(deleted, []string{"b", "a", "bad"}) {
		t.Fatalf("deleted=%v", deleted)
	}
	deleted = nil
	if err := scope.DeleteMetadata(ctx, "sub1", nil); err != nil || deleted != nil {
		t.Fatal("nil keys must send nothing", err)
	}
}

func TestR2ManilaASnapshotMetadataSetAndReplace(t *testing.T) {
	cloud := testcloud.New(t)
	var seen []string
	h := func(w http.ResponseWriter, r *http.Request) {
		r2ManilaAVersion(t, r, "2.82")
		seen = append(seen, r.Method+" "+r2ManilaARead(t, r))
		testcloud.JSON(w, 200, `{"metadata":{"k":"v"}}`)
	}
	cloud.Mux.HandleFunc("POST /v2/snapshots/snap-1/metadata", h)
	cloud.Mux.HandleFunc("PUT /v2/snapshots/snap-1/metadata", h)
	cloud.Mux.HandleFunc("POST /v2/snapshots/bad/metadata", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 202, `{}`) })
	api := snapshots.New(r2ManilaAClient(cloud, "2.82"))
	ctx := context.Background()
	if m, err := api.SetMetadata(ctx, "snap-1", map[string]string{"k": ""}); err != nil || m["k"] != "v" {
		t.Fatal(m, err)
	}
	if _, err := api.SetMetadata(ctx, "snap-1", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := api.ReplaceMetadata(ctx, "snap-1", nil); err != nil {
		t.Fatal(err)
	}
	want := []string{`POST {"metadata":{"k":""}}`, `POST {"metadata":{}}`, `PUT {"metadata":{}}`}
	if !reflect.DeepEqual(seen, want) {
		t.Fatal(seen)
	}
	_, err := api.SetMetadata(ctx, "bad", nil)
	r2ManilaACode(t, err, 202)
	if _, err := api.SetMetadata(ctx, "a/b", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Error(err)
	}
	if _, err := api.ReplaceMetadata(r2ManilaACanceled(), "snap-1", nil); !errors.Is(err, context.Canceled) {
		t.Error(err)
	}
}

func TestR2ManilaAServiceEnableDisableEnsureShares(t *testing.T) {
	cloud := testcloud.New(t)
	var bodies []string
	cloud.Mux.HandleFunc("PUT /v2/services/enable", func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, "enable "+r2ManilaARead(t, r))
		testcloud.JSON(w, 200, `{"host":"h","binary":"manila-share","disabled":false}`)
	})
	cloud.Mux.HandleFunc("PUT /v2/services/disable", func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, "disable "+r2ManilaARead(t, r))
		testcloud.JSON(w, 200, `{"service":{"host":"h","binary":"manila-share","disabled":true,"disabled_reason":"r"}}`)
	})
	cloud.Mux.HandleFunc("POST /v2/services/ensure-shares", func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, "ensure "+r2ManilaARead(t, r))
		w.WriteHeader(202)
	})
	api := services.New(r2ManilaAClient(cloud, "2.86"))
	ctx := context.Background()
	v, err := api.Enable(ctx, "h", "manila-share")
	if err != nil || v.Disabled == nil || *v.Disabled || v.Host != "h" {
		t.Fatal(v, err)
	}
	v, err = api.Disable(ctx, "h", "manila-share", services.DisableOpts{DisabledReason: request.Present("r")})
	if err != nil || !*v.Disabled || v.DisabledReason != "r" {
		t.Fatal(v, err)
	}
	if _, err := api.Disable(ctx, "h", "manila-share", services.DisableOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Disable(ctx, "h", "manila-share", services.DisableOpts{DisabledReason: request.Present("")}); err != nil {
		t.Fatal(err)
	}
	if err := api.EnsureShares(ctx, "h"); err != nil {
		t.Fatal(err)
	}
	want := []string{`enable {"binary":"manila-share","host":"h"}`, `disable {"binary":"manila-share","disabled_reason":"r","host":"h"}`,
		`disable {"binary":"manila-share","host":"h"}`, `disable {"binary":"manila-share","disabled_reason":"","host":"h"}`, `ensure {"host":"h"}`}
	if !reflect.DeepEqual(bodies, want) {
		t.Fatalf("bodies=%q", bodies)
	}
}

func TestR2ManilaAServiceActionsFloorsErrorsAndNoReplay(t *testing.T) {
	cloud := testcloud.New(t)
	calls := r2ManilaAGuard(t, cloud)
	ctx := context.Background()
	if _, err := services.New(r2ManilaAClient(cloud, "2.6")).Enable(ctx, "h", "b"); !errors.Is(err, resource.ErrUnsupported) {
		t.Error(err)
	}
	if _, err := services.New(r2ManilaAClient(cloud, "2.82")).Disable(ctx, "h", "b", services.DisableOpts{DisabledReason: request.Present("r")}); !errors.Is(err, resource.ErrUnsupported) {
		t.Error("reason needs 2.83:", err)
	}
	if err := services.New(r2ManilaAClient(cloud, "2.85")).EnsureShares(ctx, "h"); !errors.Is(err, resource.ErrUnsupported) {
		t.Error(err)
	}
	api := services.New(r2ManilaAClient(cloud, "2.86"))
	if _, err := api.Enable(ctx, "", "b"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Error(err)
	}
	if err := api.EnsureShares(ctx, ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Error(err)
	}
	for _, key := range []string{"host", "binary", "disabled_reason"} {
		if _, err := api.Disable(ctx, "h", "b", services.DisableOpts{DisabledReason: request.Present("r")}, services.WithDisableField(key, 1)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", key, err)
		}
	}
	if err := api.EnsureShares(r2ManilaACanceled(), "h"); !errors.Is(err, context.Canceled) {
		t.Error(err)
	}
	if calls.Load() != 0 {
		t.Fatal("request sent")
	}
	cloud2 := testcloud.New(t)
	var posts atomic.Int32
	cloud2.Mux.HandleFunc("POST /v2/services/ensure-shares", func(w http.ResponseWriter, r *http.Request) { posts.Add(1); testcloud.JSON(w, 403, `{}`) })
	cloud2.Mux.HandleFunc("PUT /v2/services/enable", func(w http.ResponseWriter, r *http.Request) { posts.Add(1); testcloud.JSON(w, 204, ``) })
	api2 := services.New(r2ManilaAClient(cloud2, "2.86"))
	r2ManilaACode(t, api2.EnsureShares(ctx, "h"), 403)
	_, err := api2.Enable(ctx, "h", "b")
	r2ManilaACode(t, err, 204)
	if posts.Load() != 2 {
		t.Fatal("replayed")
	}
}
