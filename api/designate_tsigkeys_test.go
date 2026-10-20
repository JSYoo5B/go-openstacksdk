package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/tsigkeys"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const designateTSIGKeyBody = `{"id":"key-1","name":"transfer","algorithm":"hmac-sha256","secret":"c2VjcmV0","scope":"ZONE","resource_id":"zone-1","created_at":"2026-01-02T03:04:05.000000","links":{"self":"http://dns.example/v2/tsigkeys/key-1"},"vendor_extra":1}`

func newDesignateTSIGKeys(t *testing.T, cloud *testcloud.Cloud) *tsigkeys.API {
	t.Helper()
	return tsigkeys.New(cloud.Client("dns", "/v2"))
}

func TestDesignateTSIGKeyCreateSendsFlatBodyAndExtensionValues(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v2/tsigkeys", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := map[string]string{"name": `"transfer"`, "algorithm": `"hmac-sha256"`, "secret": `"c2VjcmV0"`, "scope": `"ZONE"`, "resource_id": `"zone-1"`, "flag": `false`, "nothing": `null`}
		if got := designateStrings(designateFlatBody(t, r)); !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 201, designateTSIGKeyBody)
	})
	value, err := newDesignateTSIGKeys(t, cloud).Create(context.Background(), tsigkeys.CreateOpts{
		Name: "transfer", Algorithm: "hmac-sha256", Secret: "c2VjcmV0", Scope: "ZONE", ResourceID: "zone-1",
	}, tsigkeys.WithCreateField("flag", false), tsigkeys.WithCreateField("nothing", nil))
	if err != nil || value == nil || value.ID != "key-1" || value.Secret != "c2VjcmV0" || value.ResourceID != "zone-1" || value.CreatedAt.IsZero() {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestDesignateTSIGKeyCreateRequiresAllCoreFieldsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	if _, err := newDesignateTSIGKeys(t, cloud).Create(context.Background(), tsigkeys.CreateOpts{Name: "transfer"}); err == nil {
		t.Fatal("missing required fields must fail before HTTP")
	}
}

func TestDesignateTSIGKeyUpdateSendsOnlySetFieldsAndAcceptsOnly200(t *testing.T) {
	for _, status := range []int{200, 201, 202, 204} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PATCH /v2/tsigkeys/key-1", func(w http.ResponseWriter, r *http.Request) {
			want := map[string]string{"name": `"renamed"`, "flag": `false`}
			if got := designateStrings(designateFlatBody(t, r)); !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			if status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, status, designateTSIGKeyBody)
		})
		value, err := newDesignateTSIGKeys(t, cloud).Update(context.Background(), "key-1", tsigkeys.UpdateOpts{Name: "renamed"}, tsigkeys.WithUpdateField("flag", false))
		if (status == 200) != (err == nil) || (err == nil && value.ID != "key-1") {
			t.Fatalf("status %d value=%+v error=%v", status, value, err)
		}
		if err != nil {
			var unexpected gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &unexpected) || unexpected.Actual != status {
				t.Fatalf("status %d error=%v", status, err)
			}
		}
	}
}

func TestDesignateTSIGKeyCreateAndDeleteStatusCodes(t *testing.T) {
	for _, status := range []int{200, 201, 202, 204} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("POST /v2/tsigkeys", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, status, designateTSIGKeyBody) })
		cloud.Mux.HandleFunc("DELETE /v2/tsigkeys/key-1", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) })
		api := newDesignateTSIGKeys(t, cloud)
		_, createErr := api.Create(context.Background(), tsigkeys.CreateOpts{Name: "n", Algorithm: "a", Secret: "s", Scope: "POOL", ResourceID: "r"})
		deleteErr := api.Delete(context.Background(), "key-1")
		if (status == 201) != (createErr == nil) || (status == 204) != (deleteErr == nil) {
			t.Fatalf("status %d create=%v delete=%v", status, createErr, deleteErr)
		}
	}
}

func TestDesignateTSIGKeyDeleteMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("DELETE /v2/tsigkeys/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{"code":404}`) })
	api := newDesignateTSIGKeys(t, cloud)
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if err := api.Delete(context.Background(), "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native delete keeps the 404 response error: %v", err)
	}
	if err := api.Remove(context.Background(), resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
}

func TestDesignateTSIGKeyRejectsProtectedOrUnsupportedExtensionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newDesignateTSIGKeys(t, cloud)
	ctx := context.Background()
	create := tsigkeys.CreateOpts{Name: "n", Algorithm: "a", Secret: "s", Scope: "POOL", ResourceID: "r"}
	if _, err := api.Create(ctx, create, tsigkeys.WithCreateField("secret", "shadow")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected create field: %v", err)
	}
	if _, err := api.Update(ctx, "key-1", tsigkeys.UpdateOpts{}, tsigkeys.WithUpdateField("resource_id", "x")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected update field: %v", err)
	}
	if _, err := api.Create(ctx, create, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	if _, err := api.Update(ctx, "key-1", tsigkeys.UpdateOpts{}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil update option: %v", err)
	}
	seen := 0
	for _, err := range api.List(ctx, nil) {
		seen++
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list option: %v", err)
		}
	}
	if seen != 1 {
		t.Fatalf("invalid list yielded %d items", seen)
	}
}

func TestDesignateTSIGKeyListFiltersFollowLinksNextAndRejectStatusFilter(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v2/tsigkeys", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "key-1" {
			testcloud.JSON(w, 200, `{"tsigkeys":[{"id":"key-2","name":"two"}],"links":{}}`)
			return
		}
		next := cloud.Server.URL + "/v2/tsigkeys?marker=key-1&limit=1"
		testcloud.JSON(w, 200, `{"tsigkeys":[{"id":"key-1","name":"one"}],"links":{"next":"`+next+`"}}`)
	})
	api := newDesignateTSIGKeys(t, cloud)
	var ids []string
	for value, err := range api.List(context.Background(),
		tsigkeys.WithListOptions(tsigkeys.ListOpts{Name: "one", Algorithm: "hmac-sha256", Scope: "ZONE", Limit: 1}), tsigkeys.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"key-1", "key-2"}) || len(requests) != 2 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	want := url.Values{"name": {"one"}, "algorithm": {"hmac-sha256"}, "scope": {"ZONE"}, "limit": {"1"}, "vendor": {"x"}}
	if !reflect.DeepEqual(requests[0], want) || requests[1].Get("marker") != "key-1" {
		t.Fatalf("requests=%v", requests)
	}
	requests = nil
	if _, err := api.All(context.Background(), resource.WithStatus("ACTIVE")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("TSIG keys have no status; the common status filter must be rejected: %v", err)
	}
	if len(requests) != 0 {
		t.Fatalf("status rejection must not reach HTTP: %v", requests)
	}
}

func TestDesignateTSIGKeyGetAndFindExactName(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/tsigkeys/key-1", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, designateTSIGKeyBody) })
	cloud.Mux.HandleFunc("GET /v2/tsigkeys/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/tsigkeys", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"tsigkeys":[{"id":"key-1","name":"transfer"},{"id":"key-2","name":"transfer-2"}]}`)
	})
	api := newDesignateTSIGKeys(t, cloud)
	ctx := context.Background()
	value, err := api.Get(ctx, "key-1")
	if err != nil || value.Name != "transfer" || value.Algorithm != "hmac-sha256" || value.Scope != "ZONE" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.Get(ctx, "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	found, err := api.Find(ctx, resource.Name("transfer"))
	if err != nil || found == nil || found.ID != "key-1" || len(queries) == 0 || queries[0].Get("name") != "transfer" {
		t.Fatalf("found=%+v error=%v queries=%v", found, err, queries)
	}
}

func TestDesignateTSIGKeyCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	api := newDesignateTSIGKeys(t, cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	create := tsigkeys.CreateOpts{Name: "n", Algorithm: "a", Secret: "s", Scope: "POOL", ResourceID: "r"}
	if _, err := api.Get(canceled, "key-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if _, err := api.Create(canceled, create); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled create: %v", err)
	}
	if err := api.Delete(canceled, "key-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled delete: %v", err)
	}
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, create); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
