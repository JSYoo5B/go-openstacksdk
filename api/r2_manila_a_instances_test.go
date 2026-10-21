package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/limits"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareinstances"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/snapshotinstances"
)

func TestR2ManilaAShareInstanceReadsActionsAndPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	var query string
	cloud.Mux.HandleFunc("GET /v2/share_instances", func(w http.ResponseWriter, r *http.Request) {
		r2ManilaAVersion(t, r, "2.3")
		query = r.URL.RawQuery
		testcloud.JSON(w, 200, `{"share_instances":[{"id":"i1","status":"available","cast_rules_to_readonly":false,"progress":null}]}`)
	})
	cloud.Mux.HandleFunc("GET /v2/share_instances/i1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"share_instance":{"id":"i1","status":"available"}}`)
	})
	var bodies []string
	var status atomic.Int32
	status.Store(202)
	cloud.Mux.HandleFunc("POST /v2/share_instances/i1/action", func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, r2ManilaARead(t, r))
		testcloud.JSON(w, int(status.Load()), `{}`)
	})
	api := shareinstances.New(r2ManilaAClient(cloud, "2.3"))
	ctx := context.Background()
	for v, err := range api.List(ctx, shareinstances.WithListOptions(shareinstances.ListOpts{ExportLocationID: "e"})) {
		if err != nil || v.CastRulesToReadOnly == nil || *v.CastRulesToReadOnly || string(v.Evidence.Body["progress"]) != "null" {
			t.Fatal(v, err)
		}
	}
	if query != "export_location_id=e" {
		t.Fatal(query)
	}
	if v, err := api.Get(ctx, "i1"); err != nil || v.Status != "available" {
		t.Fatal(v, err)
	}
	if err := api.ResetStatus(ctx, "i1", "error"); err != nil {
		t.Fatal(err)
	}
	if err := api.ForceDelete(ctx, "i1"); err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 2 || bodies[0] != `{"reset_status":{"status":"error"}}` || bodies[1] != `{"force_delete":null}` {
		t.Fatalf("bodies=%v", bodies)
	}
	status.Store(404)
	r2ManilaACode(t, api.ForceDelete(ctx, "i1"), 404)
	if err := api.Remove(ctx, resource.ID("i1")); err != nil {
		t.Errorf("default ignore_missing: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("i1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Errorf("strict: %v", err)
	}
	status.Store(500)
	bodies = nil
	r2ManilaACode(t, api.ResetStatus(ctx, "i1", "x"), 500)
	if len(bodies) != 1 {
		t.Fatal("replayed")
	}
}

func TestR2ManilaAShareInstanceLocalFailuresSendNothing(t *testing.T) {
	cloud := testcloud.New(t)
	calls := r2ManilaAGuard(t, cloud)
	ctx := context.Background()
	if _, err := shareinstances.New(r2ManilaAClient(cloud, "2.2")).Get(ctx, "i1"); !errors.Is(err, resource.ErrUnsupported) {
		t.Error(err)
	}
	api := shareinstances.New(r2ManilaAClient(cloud, "2.3"))
	if err := api.ResetStatus(ctx, "i1", ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Error(err)
	}
	if err := api.ForceDelete(ctx, "a/b"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Error(err)
	}
	if err := api.ForceDelete(r2ManilaACanceled(), "i1"); !errors.Is(err, context.Canceled) {
		t.Error(err)
	}
	if calls.Load() != 0 {
		t.Fatal("request sent")
	}
}

func TestR2ManilaASnapshotInstanceRoutesAndDetails(t *testing.T) {
	cloud := testcloud.New(t)
	var path, query string
	list := func(w http.ResponseWriter, r *http.Request) {
		r2ManilaAVersion(t, r, "2.19")
		path, query = r.URL.Path, r.URL.RawQuery
		testcloud.JSON(w, 200, `{"snapshot_instances":[{"id":"x","status":"available"}]}`)
	}
	cloud.Mux.HandleFunc("GET /v2/snapshot-instances", list)
	cloud.Mux.HandleFunc("GET /v2/snapshot-instances/detail", list)
	cloud.Mux.HandleFunc("GET /v2/snapshot-instances/x", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"snapshot_instance":{"id":"x","progress":"100%"}}`)
	})
	cloud.Mux.HandleFunc("GET /v2/snapshot-instances/gone", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := snapshotinstances.New(r2ManilaAClient(cloud, "2.19"))
	ctx := context.Background()
	for _, err := range api.List(ctx, snapshotinstances.WithListOptions(snapshotinstances.ListOpts{SnapshotID: "s"})) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if path != "/v2/snapshot-instances/detail" || query != "snapshot_id=s" {
		t.Fatal(path, query)
	}
	for _, err := range api.List(ctx, snapshotinstances.WithListDetails(false)) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if path != "/v2/snapshot-instances" {
		t.Fatal(path)
	}
	if v, err := api.Get(ctx, "x"); err != nil || v.Progress != "100%" {
		t.Fatal(v, err)
	}
	_, err := api.Get(ctx, "gone")
	r2ManilaACode(t, err, 404)
	if _, err := snapshotinstances.New(r2ManilaAClient(cloud, "2.18")).Get(ctx, "x"); !errors.Is(err, resource.ErrUnsupported) {
		t.Error(err)
	}
	if _, err := api.Find(ctx, resource.ID("gone")); !errors.Is(err, resource.ErrNotFound) {
		t.Errorf("Find reports missing by default: %v", err)
	}
	if v, err := api.Find(ctx, resource.ID("gone"), resource.WithIgnoreMissing()); err != nil || v != nil {
		t.Errorf("ignore missing: %v %v", v, err)
	}
}

func TestR2ManilaALimitsKeepsNestedAbsoluteNullAndEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/limits", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "x=1" {
			t.Errorf("query %s", r.URL.RawQuery)
		}
		w.Header().Set("X-L", "k")
		testcloud.JSON(w, 200, `{"limits":{"rate":[{"verb":"GET"}],"absolute":{"maxTotalShares":50,"totalSharesUsed":0,"maxTotalShareGigabytes":null,"vendor":9007199254740993}}}`)
	})
	api := limits.New(r2ManilaAClient(cloud, ""))
	v, err := api.Get(context.Background(), limits.WithQuery("x", "1"))
	if err != nil || *v.Absolute.MaxTotalShares != 50 || *v.Absolute.TotalSharesUsed != 0 || v.Absolute.MaxTotalShareGigabytes != nil || v.Absolute.MaxTotalSnapshotGigabytes != nil ||
		len(v.Rate) != 1 || v.Evidence.Header.Get("X-L") != "k" || v.Evidence.StatusCode != 200 {
		t.Fatalf("%+v %v", v, err)
	}
	if _, err := api.Get(r2ManilaACanceled()); !errors.Is(err, context.Canceled) {
		t.Error(err)
	}
	cloud2 := testcloud.New(t)
	cloud2.Mux.HandleFunc("GET /v2/limits", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"nolimits":{}}`) })
	if _, err := limits.New(r2ManilaAClient(cloud2, "")).Get(context.Background()); err == nil {
		t.Error("missing limits member accepted")
	}
	cloud3 := testcloud.New(t)
	cloud3.Mux.HandleFunc("GET /v2/limits", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 202, `{}`) })
	_, err = limits.New(r2ManilaAClient(cloud3, "")).Get(context.Background())
	r2ManilaACode(t, err, 202)
}
