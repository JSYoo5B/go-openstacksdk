package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/baremetal/v1/conductors"
	"gophercloudsdk/baremetal/v1/drivers"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func checkBareMetalReadOnlyIdentity[T any](t *testing.T, cloud *testcloud.Cloud, collection *resource.Collection[T], base, envelope, linksKey, identityKey string, model map[string]any, identity func(*T) string, checkDetails func(*T)) {
	t.Helper()
	var getCalls, firstPages, nextPages atomic.Int32
	var duplicates, forbiddenList, checkExtraQuery atomic.Bool
	cloud.Mux.HandleFunc(base, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method=%s", r.Method)
		}
		if version := r.Header.Get("X-OpenStack-Ironic-API-Version"); version != "1.49" {
			t.Errorf("microversion=%q", version)
		}
		if forbiddenList.Load() {
			testcloud.JSON(w, 403, "{}")
			return
		}
		if r.URL.Query().Get("name") != "" {
			t.Errorf("name filter must be local for this native list: %s", r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("marker") == "next" {
			nextPages.Add(1)
			values := []any{model}
			if duplicates.Load() {
				values = append(values, maps.Clone(model))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{envelope: values})
			return
		}
		firstPages.Add(1)
		if checkExtraQuery.Load() && (r.URL.Query().Get("vendor") != "a&b" || r.URL.Query().Get("limit") != "2") {
			t.Errorf("query=%s", r.URL.RawQuery)
		}
		other := maps.Clone(model)
		other[identityKey] = "target-suffix"
		_ = json.NewEncoder(w).Encode(map[string]any{
			envelope: []any{other},
			linksKey: []any{map[string]any{"rel": "next", "href": cloud.Server.URL + base + "?marker=next"}},
		})
	})
	cloud.Mux.HandleFunc(base+"/target", func(w http.ResponseWriter, r *http.Request) {
		getCalls.Add(1)
		if r.Method != http.MethodGet {
			t.Errorf("read-only model used method %s", r.Method)
		}
		if version := r.Header.Get("X-OpenStack-Ironic-API-Version"); version != "1.49" {
			t.Errorf("microversion=%q", version)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(model)
	})
	cloud.Mux.HandleFunc(base+"/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, "{}") })
	cloud.Mux.HandleFunc(base+"/forbidden", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 403, "{}") })
	ctx := context.Background()
	if collection == nil {
		t.Fatal("missing read-only Collection")
	}
	if id, err := collection.ResolveID(ctx, resource.ID("target")); err != nil || id != "target" || getCalls.Load() != 0 || firstPages.Load() != 0 {
		t.Fatalf("id=%q get=%d list=%d err=%v", id, getCalls.Load(), firstPages.Load(), err)
	}
	value, err := collection.Find(ctx, resource.ID("target"))
	if err != nil || value == nil || identity(value) != "target" || getCalls.Load() != 1 || firstPages.Load() != 0 {
		t.Fatalf("value=%v get=%d list=%d err=%v", value, getCalls.Load(), firstPages.Load(), err)
	}
	checkDetails(value)
	value, err = collection.Find(ctx, resource.Name("target"))
	if err != nil || value == nil || identity(value) != "target" || firstPages.Load() != 1 || nextPages.Load() != 1 || getCalls.Load() != 1 {
		t.Fatalf("value=%v pages=%d/%d get=%d err=%v", value, firstPages.Load(), nextPages.Load(), getCalls.Load(), err)
	}
	checkDetails(value)
	if id, err := collection.ResolveID(ctx, resource.Name("target")); err != nil || id != "target" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	checkExtraQuery.Store(true)
	values, err := collection.All(ctx, resource.WithName("target"), resource.WithPageSize(2), resource.WithQuery("vendor", "a&b"))
	checkExtraQuery.Store(false)
	if err != nil || len(values) != 1 || identity(values[0]) != "target" {
		t.Fatalf("values=%v err=%v", values, err)
	}
	beforeFirst, beforeNext := firstPages.Load(), nextPages.Load()
	for item, err := range collection.List(ctx) {
		if err != nil || identity(item) != "target-suffix" {
			t.Fatalf("item=%v err=%v", item, err)
		}
		break
	}
	if firstPages.Load() != beforeFirst+1 || nextPages.Load() != beforeNext {
		t.Fatalf("iterator fetched after break: first=%d next=%d", firstPages.Load(), nextPages.Load())
	}
	duplicates.Store(true)
	if _, err := collection.Find(ctx, resource.Name("target")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatalf("duplicate error=%v", err)
	}
	duplicates.Store(false)
	for _, ref := range []resource.Ref{resource.ID("missing"), resource.Name("missing")} {
		if _, err := collection.Find(ctx, ref); !errors.Is(err, resource.ErrNotFound) {
			t.Fatalf("missing ref=%v error=%v", ref, err)
		}
		if value, err := collection.Find(ctx, ref, resource.WithIgnoreMissing()); err != nil || value != nil {
			t.Fatalf("missing value=%v error=%v", value, err)
		}
	}
	_, err = collection.Get(ctx, "forbidden")
	var response gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &response) || response.Actual != 403 || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("get error=%v", err)
	}
	forbiddenList.Store(true)
	if _, err := collection.Find(ctx, resource.Name("target")); !errors.As(err, &response) || response.Actual != 403 || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("find error=%v", err)
	}
	forbiddenList.Store(false)
	beforeFirst, beforeNext = firstPages.Load(), nextPages.Load()
	beforeGet := getCalls.Load()
	if _, err := collection.Get(ctx, ".."); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("invalid ID error=%v", err)
	}
	if err := collection.Delete(ctx, resource.Name("target")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("delete error=%v", err)
	}
	if _, err := collection.Wait(ctx, resource.Name("target"), "ACTIVE"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("wait error=%v", err)
	}
	if _, err := collection.All(ctx, resource.WithStatus("ACTIVE")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("status filter error=%v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := collection.Get(canceled, "target"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled get error=%v", err)
	}
	if _, err := collection.All(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled list error=%v", err)
	}
	if firstPages.Load() != beforeFirst || nextPages.Load() != beforeNext || getCalls.Load() != beforeGet {
		t.Fatal("validation, unsupported policy or cancellation caused an HTTP request")
	}
}

func TestBareMetalNamedReadOnlyCollectionsUseNativeIdentity(t *testing.T) {
	t.Run("conductor hostname", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := cloud.Client("baremetal", "/v1")
		client.Microversion = "1.49"
		api := conductors.New(client)
		checkBareMetalReadOnlyIdentity(t, cloud, api.Resources, "/v1/conductors", "conductors", "conductor_links", "hostname", map[string]any{"hostname": "target", "alive": true, "drivers": []string{"redfish"}}, func(value *conductors.Conductor) string { return value.Hostname }, func(value *conductors.Conductor) {
			if !value.Alive || len(value.Drivers) != 1 || value.Drivers[0] != "redfish" {
				t.Fatalf("conductor details=%+v", value)
			}
		})
	})
	t.Run("driver name", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := cloud.Client("baremetal", "/v1")
		client.Microversion = "1.49"
		api := drivers.New(client)
		checkBareMetalReadOnlyIdentity(t, cloud, api.Resources, "/v1/drivers", "drivers", "drivers_links", "name", map[string]any{"name": "target", "hosts": []string{"conductor-01"}}, func(value *drivers.Driver) string { return value.Name }, func(value *drivers.Driver) {
			if len(value.Hosts) != 1 || value.Hosts[0] != "conductor-01" {
				t.Fatalf("driver details=%+v", value)
			}
		})
	})
}
