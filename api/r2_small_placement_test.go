package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/traits"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestR2SmallPlacementCreateInventoryBodyAndStatus(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("placement", "/placement")
	var body map[string]json.RawMessage
	var header http.Header
	cloud.Mux.HandleFunc("/placement/resource_providers/rp-1/inventories", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method=%s", r.Method)
		}
		header = r.Header.Clone()
		_ = json.NewDecoder(r.Body).Decode(&body)
		testcloud.JSON(w, 201, `{"resource_provider_generation":3,"total":8,"reserved":0,"allocation_ratio":1.5}`)
	})
	zero := 0
	value, err := resourceproviders.New(client).CreateInventory(context.Background(), "rp-1", "VCPU",
		resourceproviders.CreateInventoryOpts{Total: 8, Reserved: &zero}, resourceproviders.WithCreateInventoryField("custom", true))
	if err != nil || value.StatusCode != 201 || *value.ResourceProviderGeneration != 3 || *value.AllocationRatio != 1.5 {
		t.Fatalf("value=%+v err=%v", value, err)
	}
	if string(body["resource_class"]) != `"VCPU"` || string(body["total"]) != "8" || string(body["reserved"]) != "0" || string(body["custom"]) != "true" || len(body) != 4 {
		t.Fatalf("body=%v", body)
	}
	if header.Get("OpenStack-API-Version") != "" {
		t.Fatalf("unexpected microversion header %v", header)
	}
}

func TestR2SmallPlacementCreateInventoryRejectsBeforeHTTPAndKeepsServerErrors(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("placement", "/placement")
	var calls atomic.Int32
	status := 409
	cloud.Mux.HandleFunc("/placement/resource_providers/rp%2F1/inventories", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, status, `{"errors":[{"status":409}]}`)
	})
	api := resourceproviders.New(client)
	ctx := context.Background()
	for name, call := range map[string]func() error{
		"empty provider": func() error {
			_, err := api.CreateInventory(ctx, " ", "VCPU", resourceproviders.CreateInventoryOpts{Total: 1})
			return err
		},
		"empty class": func() error {
			_, err := api.CreateInventory(ctx, "rp", "", resourceproviders.CreateInventoryOpts{Total: 1})
			return err
		},
		"owned class": func() error {
			_, err := api.CreateInventory(ctx, "rp", "VCPU", resourceproviders.CreateInventoryOpts{Total: 1}, resourceproviders.WithCreateInventoryField("resource_class", "X"))
			return err
		},
		"core collision": func() error {
			_, err := api.CreateInventory(ctx, "rp", "VCPU", resourceproviders.CreateInventoryOpts{Total: 1}, resourceproviders.WithCreateInventoryField("total", 2))
			return err
		},
		"reserved header": func() error {
			_, err := api.CreateInventory(ctx, "rp", "VCPU", resourceproviders.CreateInventoryOpts{Total: 1}, resourceproviders.WithCreateInventoryHeader("X-Auth-Token", "x"))
			return err
		},
	} {
		if err := call(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("rejected input reached HTTP")
	}
	// A 200 is not the documented 201 and a 409 is never replayed; the provider ID is escaped.
	if _, err := api.CreateInventory(ctx, "rp/1", "VCPU", resourceproviders.CreateInventoryOpts{Total: 1}); err == nil || calls.Load() != 1 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := api.CreateInventory(canceled, "rp", "VCPU", resourceproviders.CreateInventoryOpts{Total: 1}); !errors.Is(err, context.Canceled) || calls.Load() != 1 {
		t.Fatalf("err=%v", err)
	}
}

func TestR2SmallPlacementInventoryAndTraitDeletionWaits(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	client := cloud.Client("placement", "/placement")
	var inv, tr atomic.Int32
	cloud.Mux.HandleFunc("/placement/resource_providers/rp-1/inventories/VCPU", func(w http.ResponseWriter, r *http.Request) {
		if inv.Add(1) < 3 {
			testcloud.JSON(w, 200, `{"resource_provider_generation":1,"total":1}`)
			return
		}
		testcloud.JSON(w, 404, `{"errors":[{"status":404}]}`)
	})
	cloud.Mux.HandleFunc("/placement/traits/CUSTOM_X", func(w http.ResponseWriter, r *http.Request) {
		if tr.Add(1) < 2 {
			w.WriteHeader(204)
			return
		}
		testcloud.JSON(w, 404, `{"errors":[{"status":404}]}`)
	})
	ctx := context.Background()
	fast := resource.WithPollInterval(time.Millisecond)
	if err := resourceproviders.New(client).InventoryResources("rp-1").WaitDeleted(ctx, resource.ID("VCPU"), fast, resource.WithTimeout(5*time.Second)); err != nil || inv.Load() != 3 {
		t.Fatalf("inventory err=%v polls=%d", err, inv.Load())
	}
	if err := traits.New(client).Resources().WaitDeleted(ctx, resource.ID("CUSTOM_X"), fast, resource.WithTimeout(5*time.Second)); err != nil || tr.Load() != 2 {
		t.Fatalf("trait err=%v polls=%d", err, tr.Load())
	}
}
