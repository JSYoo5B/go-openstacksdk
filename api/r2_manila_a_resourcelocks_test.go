package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/resourcelocks"
)

const r2ManilaALockJSON = `{"resource_lock":{"id":"lock-1","resource_id":"share-1","resource_type":"share","resource_action":"delete","lock_reason":null,"lock_context":"user","user_id":"u","project_id":"p","created_at":"c","updated_at":null,"vendor":{"n":9223372036854775807}}}`

func TestR2ManilaAResourceLockCreateBodyPreservesEmptyOmissionAndEvidence(t *testing.T) {
	cases := []struct {
		name string
		opts resourcelocks.CreateOpts
		body string
	}{
		{"minimal", resourcelocks.CreateOpts{ResourceID: "share-1", ResourceType: "share"}, `{"resource_lock":{"resource_id":"share-1","resource_type":"share"}}`},
		{"empty-reason-and-action", resourcelocks.CreateOpts{ResourceID: "share-1", ResourceType: "share", ResourceAction: request.Present("delete"), LockReason: request.Present("")}, `{"resource_lock":{"lock_reason":"","resource_action":"delete","resource_id":"share-1","resource_type":"share"}}`},
		{"null-reason", resourcelocks.CreateOpts{ResourceID: "a", ResourceType: "access_rule", LockReason: request.Null[string]()}, `{"resource_lock":{"lock_reason":null,"resource_id":"a","resource_type":"access_rule"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("POST /v2/resource-locks", func(w http.ResponseWriter, r *http.Request) {
				r2ManilaAVersion(t, r, "2.81")
				if got := r2ManilaARead(t, r); got != tc.body {
					t.Errorf("body = %s", got)
				}
				w.Header().Set("X-Lock", "kept")
				testcloud.JSON(w, 200, r2ManilaALockJSON)
			})
			value, err := resourcelocks.New(r2ManilaAClient(cloud, "2.81")).Create(context.Background(), tc.opts)
			if err != nil || value.ID != "lock-1" || value.LockReason != "" || string(value.Evidence.Body["lock_reason"]) != "null" ||
				value.Evidence.Body["vendor"] == nil || value.Evidence.Header.Get("X-Lock") != "kept" || value.Evidence.StatusCode != 200 {
				t.Fatalf("value=%+v err=%v", value, err)
			}
		})
	}
}

func TestR2ManilaAResourceLockCreateExtensionsAndProtectedFields(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /v2/resource-locks", func(w http.ResponseWriter, r *http.Request) {
		if got := r2ManilaARead(t, r); got != `{"resource_lock":{"resource_id":"s","resource_type":"share","vendor":false}}` {
			t.Errorf("body = %s", got)
		}
		testcloud.JSON(w, 200, r2ManilaALockJSON)
	})
	api := resourcelocks.New(r2ManilaAClient(cloud, "2.81"))
	ctx := context.Background()
	base := resourcelocks.CreateOpts{ResourceID: "s", ResourceType: "share"}
	if _, err := api.Create(ctx, base, resourcelocks.WithCreateField("vendor", false)); err != nil {
		t.Fatal(err)
	}
	calls := r2ManilaAGuard(t, cloud)
	for _, key := range []string{"resource_id", "resource_action", "id", "lock_context", "user_id", "project_id", "created_at", "resource_lock"} {
		if _, err := api.Create(ctx, base, resourcelocks.WithCreateField(key, "x")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: err=%v", key, err)
		}
	}
	for _, bad := range []resourcelocks.CreateOpts{{ResourceType: "share"}, {ResourceID: "s"}, {ResourceID: " ", ResourceType: "share"}} {
		if _, err := api.Create(ctx, bad); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("missing required: %v", err)
		}
	}
	if _, err := api.Create(ctx, base, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Errorf("nil option: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatal("invalid input reached HTTP")
	}
}

func TestR2ManilaAResourceLockVersionFloorCancellationAndNoReplay(t *testing.T) {
	cloud := testcloud.New(t)
	calls := r2ManilaAGuard(t, cloud)
	ctx := context.Background()
	for _, version := range []string{"", "2.80", "2.0"} {
		api := resourcelocks.New(r2ManilaAClient(cloud, version))
		if _, err := api.Get(ctx, "lock-1"); !errors.Is(err, resource.ErrUnsupported) {
			t.Errorf("version %q: %v", version, err)
		}
	}
	if _, err := resourcelocks.New(r2ManilaAClient(cloud, "3.1")).Get(ctx, "lock-1"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Error("bad major version accepted")
	}
	api := resourcelocks.New(r2ManilaAClient(cloud, "2.81"))
	if _, err := api.Create(r2ManilaACanceled(), resourcelocks.CreateOpts{ResourceID: "s", ResourceType: "share"}); !errors.Is(err, context.Canceled) {
		t.Error(err)
	}
	if _, err := api.Get(ctx, "a/b"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Error(err)
	}
	if calls.Load() != 0 {
		t.Fatal("local failures sent requests")
	}

	cloud2 := testcloud.New(t)
	var posts atomic.Int32
	status := 500
	cloud2.Mux.HandleFunc("POST /v2/resource-locks", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		testcloud.JSON(w, status, `{}`)
	})
	api2 := resourcelocks.New(r2ManilaAClient(cloud2, "latest"))
	for _, code := range []int{500, 202, 409} {
		status = code
		posts.Store(0)
		_, err := api2.Create(ctx, resourcelocks.CreateOpts{ResourceID: "s", ResourceType: "share"})
		r2ManilaACode(t, err, code)
		if posts.Load() != 1 {
			t.Fatalf("status %d replayed %d times", code, posts.Load())
		}
	}
}

func TestR2ManilaAResourceLockListQueriesPagingAndMalformedRows(t *testing.T) {
	cloud := testcloud.New(t)
	var raw string
	cloud.Mux.HandleFunc("GET /v2/resource-locks", func(w http.ResponseWriter, r *http.Request) {
		r2ManilaAVersion(t, r, "2.81")
		raw = r.URL.RawQuery
		if r.URL.Query().Get("lock_reason~") == "boom" {
			testcloud.JSON(w, 200, `{"resource_locks":[{"id":"a"},3]}`)
			return
		}
		testcloud.JSON(w, 200, `{"resource_locks":[{"id":"a","lock_reason":null},{"id":"b"},{"id":"c"}],"count":3}`)
	})
	api := resourcelocks.New(r2ManilaAClient(cloud, "2.81"))
	zero, no := 0, false
	var ids []string
	for value, err := range api.List(context.Background(),
		resourcelocks.WithListOptions(resourcelocks.ListOpts{ResourceType: "share", Limit: &zero, WithCount: &no, LockReasonLike: "a b", SortDir: "desc"}),
		resourcelocks.WithListQuery("vendor", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if len(ids) != 3 || raw != "limit=0&lock_reason~=a+b&resource_type=share&sort_dir=desc&vendor=1&with_count=false" {
		t.Fatalf("ids=%v query=%s", ids, raw)
	}
	capped := 0
	for _, err := range api.List(context.Background(), resourcelocks.WithListMaxItems(2)) {
		if err != nil {
			t.Fatal(err)
		}
		capped++
	}
	if capped != 2 {
		t.Fatalf("capped=%d", capped)
	}
	for _, option := range []resourcelocks.ListOption{resourcelocks.WithListQuery("resource_type", "x"), resourcelocks.WithListMaxItems(-1)} {
		for _, err := range api.List(context.Background(), option) {
			if !errors.Is(err, resource.ErrInvalidOption) {
				t.Errorf("err=%v", err)
			}
		}
	}
	for _, err := range api.List(context.Background(), resourcelocks.WithListQuery("lock_reason~", "boom")) {
		if err == nil {
			continue
		}
		return
	}
	t.Fatal("malformed row was not reported")
}

func TestR2ManilaAResourceLockUpdateGetDeleteAndRemovePolicies(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("PUT /v2/resource-locks/lock-1", func(w http.ResponseWriter, r *http.Request) {
		if got := r2ManilaARead(t, r); got != `{"resource_lock":{"lock_reason":null,"resource_action":"show"}}` {
			t.Errorf("body = %s", got)
		}
		testcloud.JSON(w, 200, r2ManilaALockJSON)
	})
	cloud.Mux.HandleFunc("GET /v2/resource-locks/lock-1", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, r2ManilaALockJSON) })
	var deleteStatus atomic.Int32
	deleteStatus.Store(204)
	cloud.Mux.HandleFunc("DELETE /v2/resource-locks/", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, int(deleteStatus.Load()), `{}`)
	})
	api := resourcelocks.New(r2ManilaAClient(cloud, "2.82"))
	ctx := context.Background()
	if v, err := api.Update(ctx, "lock-1", resourcelocks.UpdateOpts{ResourceAction: request.Present("show"), LockReason: request.Null[string]()}); err != nil || v.ID != "lock-1" {
		t.Fatalf("%v %v", v, err)
	}
	if _, err := api.Update(ctx, "lock-1", resourcelocks.UpdateOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Errorf("empty update: %v", err)
	}
	if _, err := api.Update(ctx, "lock-1", resourcelocks.UpdateOpts{LockReason: request.Present("x")}, resourcelocks.WithUpdateField("resource_id", "z")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Errorf("protected: %v", err)
	}
	if v, err := api.Get(ctx, "lock-1"); err != nil || v.ProjectID != "p" {
		t.Fatalf("%v %v", v, err)
	}
	for _, code := range []int{200, 202, 204} {
		deleteStatus.Store(int32(code))
		if err := api.Delete(ctx, "lock-1"); err != nil {
			t.Errorf("delete %d: %v", code, err)
		}
	}
	deleteStatus.Store(404)
	r2ManilaACode(t, api.Delete(ctx, "gone"), 404)
	if err := api.Remove(ctx, resource.ID("gone")); err != nil {
		t.Errorf("default Remove should ignore missing: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("gone"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Errorf("ignore_missing=False: %v", err)
	}
	deleteStatus.Store(403)
	if err := api.Remove(ctx, resource.ID("gone")); err == nil {
		t.Error("403 must not be swallowed")
	}
}

func TestR2ManilaAResourceLockFindIdentityFallsBackToList(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/resource-locks/other", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/resource-locks", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"resource_locks":[{"id":"x"},{"id":"other"}]}`)
	})
	api := resourcelocks.New(r2ManilaAClient(cloud, "2.81"))
	if v, err := api.FindIdentity(context.Background(), "other"); err != nil || v == nil || v.ID != "other" {
		t.Fatalf("%v %v", v, err)
	}
	if v, err := api.FindIdentity(context.Background(), "nope"); err != nil || v != nil {
		t.Fatalf("ignore_missing default: %v %v", v, err)
	}
	if _, err := api.FindIdentity(context.Background(), "nope", resource.WithIdentityFindIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("strict: %v", err)
	}
}
