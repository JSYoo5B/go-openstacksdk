package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/reservation/v1/hosts"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const blazarHostBody = `{"host":{"id":"h-1","hypervisor_hostname":"node1","vcpus":0,"reservable":false,"status":"free","created_at":"2026-01-02 03:04:05","updated_at":null,"gpu":false,"slots":0}}`

func newBlazarHosts(t *testing.T, cloud *testcloud.Cloud) *hosts.API {
	t.Helper()
	return hosts.New(cloud.Client("reservation", "/v1"))
}

func blazarUnexpected(err error) int {
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if errors.As(err, &unexpected) {
		return unexpected.Actual
	}
	return 0
}

func blazarFields(t *testing.T, r *http.Request) map[string]string {
	t.Helper()
	var body map[string]json.RawMessage
	data, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(data, &body); err != nil {
		t.Errorf("request body %q: %v", data, err)
	}
	got := map[string]string{}
	for key, value := range body {
		got[key] = string(value)
	}
	return got
}

func TestBlazarHostCreateSendsFlatBodyWithExplicitCapabilitiesAndRequires201(t *testing.T) {
	for _, status := range []int{201, 200, 202, 204} {
		cloud := testcloud.New(t)
		var bodies []map[string]string
		cloud.Mux.HandleFunc("POST /v1/os-hosts", func(w http.ResponseWriter, r *http.Request) {
			bodies = append(bodies, blazarFields(t, r))
			if status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, status, blazarHostBody)
		})
		api := newBlazarHosts(t, cloud)
		value, err := api.Create(context.Background(), hosts.CreateOpts{
			Name: "node1", ExtraCapabilities: map[string]any{"gpu": false, "slots": 0, "note": "", "unset": nil},
		}, hosts.WithCreateField("vendor", false))
		if (status == 201) != (err == nil) || (status == 201 && (value == nil || value.ID != "h-1")) {
			t.Fatalf("status %d value=%+v error=%v", status, value, err)
		}
		if status != 201 && blazarUnexpected(err) != status {
			t.Fatalf("status %d error=%v", status, err)
		}
		want := map[string]string{"name": `"node1"`, "gpu": `false`, "slots": `0`, "note": `""`, "unset": `null`, "vendor": `false`}
		if !reflect.DeepEqual(bodies[0], want) {
			t.Fatalf("body=%v", bodies[0])
		}
		if status == 201 {
			if value.Reservable || value.VCPUs != 0 || value.UpdatedAt != nil || value.CreatedAt.IsZero() || value.ExtraCapabilities["gpu"] != false || value.ExtraCapabilities["slots"] != float64(0) {
				t.Fatalf("decoded host=%+v", value)
			}
			if _, polluted := value.ExtraCapabilities["id"]; polluted {
				t.Fatalf("core fields must not appear as capabilities: %v", value.ExtraCapabilities)
			}
		}
	}
}

func TestBlazarHostRejectsInvalidInputsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newBlazarHosts(t, cloud)
	ctx := context.Background()
	if _, err := api.Create(ctx, hosts.CreateOpts{}); err == nil {
		t.Fatal("missing name must fail locally")
	}
	if _, err := api.Create(ctx, hosts.CreateOpts{Name: "n", ExtraCapabilities: map[string]any{"name": "shadow"}}); err == nil {
		t.Fatal("capability named like a core field must fail locally")
	}
	if _, err := api.Create(ctx, hosts.CreateOpts{Name: "n"}, hosts.WithCreateField("name", "shadow")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("core field extension: %v", err)
	}
	if _, err := api.Create(ctx, hosts.CreateOpts{Name: "n", ExtraCapabilities: map[string]any{"gpu": true}}, hosts.WithCreateField("gpu", false)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("extension colliding with a capability: %v", err)
	}
	if _, err := api.Create(ctx, hosts.CreateOpts{Name: "n"}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	var missing gophercloud.ErrMissingInput
	if _, err := api.Update(ctx, "h-1", hosts.UpdateOpts{}); !errors.As(err, &missing) {
		t.Fatalf("empty update must fail locally: %v", err)
	}
	if _, err := api.Update(ctx, "h-1", hosts.UpdateOpts{ExtraCapabilities: map[string]any{"gpu": true}}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil update option: %v", err)
	}
	for _, err := range api.List(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list option: %v", err)
		}
	}
}

func TestBlazarHostUpdateSendsFlatCapabilitiesAndRequires200(t *testing.T) {
	for _, status := range []int{200, 201, 202, 204} {
		cloud := testcloud.New(t)
		var bodies []map[string]string
		cloud.Mux.HandleFunc("PUT /v1/os-hosts/h-1", func(w http.ResponseWriter, r *http.Request) {
			bodies = append(bodies, blazarFields(t, r))
			if status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, status, blazarHostBody)
		})
		api := newBlazarHosts(t, cloud)
		value, err := api.Update(context.Background(), "h-1", hosts.UpdateOpts{ExtraCapabilities: map[string]any{"gpu": false, "slots": 0, "note": ""}}, hosts.WithUpdateField("vendor", nil))
		if (status == 200) != (err == nil) || (status == 200 && (value == nil || value.ID != "h-1")) {
			t.Fatalf("status %d value=%+v error=%v", status, value, err)
		}
		if status != 200 && blazarUnexpected(err) != status {
			t.Fatalf("status %d error=%v", status, err)
		}
		want := map[string]string{"gpu": `false`, "slots": `0`, "note": `""`, "vendor": `null`}
		if !reflect.DeepEqual(bodies[0], want) {
			t.Fatalf("body=%v", bodies[0])
		}
	}
}

func TestBlazarHostGetListAndCollectionPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v1/os-hosts/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "h-1" {
			testcloud.JSON(w, 200, blazarHostBody)
			return
		}
		testcloud.JSON(w, 404, `{"error":"gone"}`)
	})
	cloud.Mux.HandleFunc("GET /v1/os-hosts", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"hosts":[{"id":"h-1","status":"free","gpu":true},{"id":"h-2","status":"reserved"}],"next":"ignored"}`)
	})
	api := newBlazarHosts(t, cloud)
	ctx := context.Background()
	value, err := api.Get(ctx, "h-1")
	if err != nil || value.HypervisorHostname != "node1" || value.Reservable {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if _, err := api.Get(ctx, "missing"); blazarUnexpected(err) != 404 || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	if _, err := api.Resources.Get(ctx, "missing"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("Collection.Get maps 404 to ErrNotFound: %v", err)
	}
	var ids []string
	for value, err := range api.List(ctx, hosts.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"h-1", "h-2"}) || len(queries) != 1 || queries[0].Get("vendor") != "x" {
		t.Fatalf("ids=%v queries=%v", ids, queries)
	}
	queries = nil
	all, err := api.All(ctx, resource.WithStatus("reserved"))
	if err != nil || len(all) != 1 || all[0].ID != "h-2" {
		t.Fatalf("all=%v error=%v", all, err)
	}
	if len(queries) != 1 || queries[0].Has("status") {
		t.Fatalf("status must be filtered locally: %v", queries)
	}
}

func TestBlazarHostDeleteRequires204AndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("DELETE /v1/os-hosts/{id}", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.PathValue("id") {
		case "missing":
			testcloud.JSON(w, 404, `{}`)
		case "ok200":
			testcloud.JSON(w, 200, `{}`)
		case "ok202":
			testcloud.JSON(w, 202, `{}`)
		case "conflict":
			testcloud.JSON(w, 409, `{}`)
		default:
			w.WriteHeader(204)
		}
	})
	api := newBlazarHosts(t, cloud)
	ctx := context.Background()
	if err := api.Delete(ctx, "h-1"); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]int{"ok200": 200, "ok202": 202, "conflict": 409, "missing": 404} {
		if err := api.Delete(ctx, id); blazarUnexpected(err) != want || errors.Is(err, resource.ErrNotFound) {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("conflict")); blazarUnexpected(err) != 409 {
		t.Fatalf("Remove keeps non-404 errors: %v", err)
	}
	if calls.Load() != 7 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestBlazarHostCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	api := newBlazarHosts(t, cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Create(canceled, hosts.CreateOpts{Name: "n"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("create: %v", err)
	}
	if _, err := api.Get(canceled, "h-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("get: %v", err)
	}
	if _, err := api.Update(canceled, "h-1", hosts.UpdateOpts{ExtraCapabilities: map[string]any{"a": 1}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("update: %v", err)
	}
	if err := api.Delete(canceled, "h-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("delete: %v", err)
	}
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, hosts.CreateOpts{Name: "n"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("an uncertain create must not be replayed: %d calls", calls.Load())
	}
}
