package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestPyDNSTSIGKeyRemoveAndFindMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	var deletes atomic.Int32
	cloud.Mux.HandleFunc("DELETE /v2/tsigkeys/missing", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v2/tsigkeys/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := newDesignateTSIGKeys(t, cloud)
	ctx := context.Background()
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("delete_tsigkey ignore_missing default: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("missing"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("ignore_missing=False: %v", err)
	}
	if deletes.Load() != 2 {
		t.Fatalf("deletes=%d", deletes.Load())
	}
	if v, err := api.Find(ctx, resource.ID("missing"), resource.WithIgnoreMissing()); err != nil || v != nil {
		t.Fatalf("find ignore missing: %v %v", v, err)
	}
	if _, err := api.Find(ctx, resource.ID("missing")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("find strict default: %v", err)
	}
}

func TestPyDNSZoneShareUnshareMissingIsExplicitlyIgnorable(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("DELETE /v2/zones/zone-1/shares/gone", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 404, `{}`)
	})
	api := newDesignateZones(t, cloud)
	ctx := context.Background()
	if err := api.Unshare(ctx, "zone-1", "gone"); err == nil {
		t.Fatal("native Unshare keeps the 404")
	}
	if err := resource.IgnoreMissing(api.Unshare(ctx, "zone-1", "gone")); err != nil {
		t.Fatalf("ignore_missing=True equivalent: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestPyDNSRecordSetDeleteIsSingleRequestWithoutPreFetch(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, deletes atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/zones/zone-1/recordsets/rr-1", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		testcloud.JSON(w, 200, designateRecordSetBody)
	})
	cloud.Mux.HandleFunc("DELETE /v2/zones/zone-1/recordsets/rr-1", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		w.WriteHeader(202)
	})
	if err := newDesignateRecordSets(t, cloud).Delete(context.Background(), "zone-1", "rr-1"); err != nil {
		t.Fatal(err)
	}
	if gets.Load() != 0 || deletes.Load() != 1 {
		t.Fatalf("gets=%d deletes=%d", gets.Load(), deletes.Load())
	}
}

func TestPyDNSTransferRequestRemoveDefaultsToIgnoreMissing(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("DELETE /v2/zones/tasks/transfer_requests/gone", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	api := newDesignateTransferRequests(t, cloud)
	ctx := context.Background()
	if err := api.Remove(ctx, resource.ID("gone")); err != nil {
		t.Fatalf("default: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("gone"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("strict: %v", err)
	}
}
