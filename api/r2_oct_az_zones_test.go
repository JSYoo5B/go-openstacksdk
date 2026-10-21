package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/availabilityzones"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func r2OctAZZonesAPI(t *testing.T, cloud *testcloud.Cloud) *availabilityzones.API {
	t.Helper()
	return availabilityzones.New(cloud.Client("load-balancer", "/v2"))
}

func r2OctAZStr(v string) *string { return &v }
func r2OctAZBool(v bool) *bool    { return &v }

func r2OctAZBody(t *testing.T, r *http.Request) string {
	t.Helper()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestR2OctAZZoneCreateBodyAndResponse(t *testing.T) {
	cloud := testcloud.New(t)
	var body string
	cloud.Mux.HandleFunc("POST /v2/lbaas/availabilityzones", func(w http.ResponseWriter, r *http.Request) {
		body = r2OctAZBody(t, r)
		w.Header().Set("X-Test", "yes")
		testcloud.JSON(w, 201, `{"availability_zone":{"name":"az1","description":null,"enabled":false,"availability_zone_profile_id":"p1","extra":1}}`)
	})
	got, err := r2OctAZZonesAPI(t, cloud).Create(context.Background(), availabilityzones.CreateOpts{
		Name: "az1", AvailabilityZoneProfileID: "p1", Description: r2OctAZStr(""), Enabled: r2OctAZBool(false),
	}, availabilityzones.WithCreateField("vendor", map[string]int{"n": 0}))
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"availability_zone":{"availability_zone_profile_id":"p1","description":"","enabled":false,"name":"az1","vendor":{"n":0}}}`
	if body != want {
		t.Fatalf("body %s", body)
	}
	if got.Name != "az1" || got.Enabled || got.StatusCode != 201 || got.Header.Get("X-Test") != "yes" || string(got.Body["description"]) != "null" || string(got.Body["extra"]) != "1" {
		t.Fatalf("%+v", got)
	}
}

func TestR2OctAZZoneCreateOmitsNilAndValidatesInputs(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	var body string
	cloud.Mux.HandleFunc("POST /v2/lbaas/availabilityzones", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body = r2OctAZBody(t, r)
		testcloud.JSON(w, 201, `{"availability_zone":{"name":"a"}}`)
	})
	api := r2OctAZZonesAPI(t, cloud)
	if _, err := api.Create(context.Background(), availabilityzones.CreateOpts{Name: "a", AvailabilityZoneProfileID: "p"}); err != nil {
		t.Fatal(err)
	}
	if body != `{"availability_zone":{"availability_zone_profile_id":"p","name":"a"}}` {
		t.Fatalf("body %s", body)
	}
	calls.Store(0)
	for name, call := range map[string]func() error{
		"missing name": func() error {
			_, err := api.Create(context.Background(), availabilityzones.CreateOpts{AvailabilityZoneProfileID: "p"})
			return err
		},
		"missing profile": func() error {
			_, err := api.Create(context.Background(), availabilityzones.CreateOpts{Name: "a"})
			return err
		},
		"core extension": func() error {
			_, err := api.Create(context.Background(), availabilityzones.CreateOpts{Name: "a", AvailabilityZoneProfileID: "p"}, availabilityzones.WithCreateField("enabled", true))
			return err
		},
		"nil option": func() error {
			_, err := api.Create(context.Background(), availabilityzones.CreateOpts{Name: "a", AvailabilityZoneProfileID: "p"}, nil)
			return err
		},
	} {
		if err := call(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("invalid input reached the server")
	}
}

func TestR2OctAZZoneCreateRejectsUnexpectedStatusAndNeverReplays(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v2/lbaas/availabilityzones", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"availability_zone":{"name":"a"}}`)
	})
	_, err := r2OctAZZonesAPI(t, cloud).Create(context.Background(), availabilityzones.CreateOpts{Name: "a", AvailabilityZoneProfileID: "p"})
	if err == nil || calls.Load() != 1 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}

func TestR2OctAZZoneGetAndMissing(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/lbaas/availabilityzones/az1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"availability_zone":{"name":"az1","enabled":true}}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/availabilityzones/gone", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{"faultstring":"nope"}`)
	})
	api := r2OctAZZonesAPI(t, cloud)
	got, err := api.Get(context.Background(), "az1")
	if err != nil || got.Name != "az1" || !got.Enabled {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err := api.Get(context.Background(), "gone"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("%v", err)
	}
	for _, name := range []string{"", "a/b", "..", "a?b"} {
		if _, err := api.Get(context.Background(), name); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%q: %v", name, err)
		}
	}
	if _, err := api.Get(context.Background(), "az1"); err != nil {
		t.Fatal(err)
	}
}

func TestR2OctAZZoneUpdateBodyAndEmpty(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	var body string
	cloud.Mux.HandleFunc("PUT /v2/lbaas/availabilityzones/az1", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body = r2OctAZBody(t, r)
		testcloud.JSON(w, 200, `{"availability_zone":{"name":"az1","enabled":false}}`)
	})
	api := r2OctAZZonesAPI(t, cloud)
	got, err := api.Update(context.Background(), "az1", availabilityzones.UpdateOpts{Enabled: r2OctAZBool(false), Description: r2OctAZStr("")})
	if err != nil || got.Enabled {
		t.Fatalf("%+v %v", got, err)
	}
	if body != `{"availability_zone":{"description":"","enabled":false}}` {
		t.Fatalf("body %s", body)
	}
	calls.Store(0)
	if _, err := api.Update(context.Background(), "az1", availabilityzones.UpdateOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("empty update: %v", err)
	}
	if _, err := api.Update(context.Background(), "az1", availabilityzones.UpdateOpts{}, availabilityzones.WithUpdateField("enabled", true)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("core collision: %v", err)
	}
	if _, err := api.Update(context.Background(), "a/b", availabilityzones.UpdateOpts{Enabled: r2OctAZBool(true)}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("bad name: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatal("invalid update reached the server")
	}
	if _, err := api.Update(context.Background(), "az1", availabilityzones.UpdateOpts{}, availabilityzones.WithUpdateField("vendor", nil)); err != nil || body != `{"availability_zone":{"vendor":null}}` {
		t.Fatalf("extension-only update: %v %s", err, body)
	}
}

func TestR2OctAZZoneDeleteRemoveAndConflict(t *testing.T) {
	cloud := testcloud.New(t)
	var deletes atomic.Int32
	cloud.Mux.HandleFunc("DELETE /v2/lbaas/availabilityzones/az1", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		w.WriteHeader(204)
	})
	cloud.Mux.HandleFunc("DELETE /v2/lbaas/availabilityzones/gone", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("DELETE /v2/lbaas/availabilityzones/busy", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(100)
		testcloud.JSON(w, 409, `{}`)
	})
	api := r2OctAZZonesAPI(t, cloud)
	if err := api.Delete(context.Background(), "az1"); err != nil || deletes.Load() != 1 {
		t.Fatalf("%v %d", err, deletes.Load())
	}
	if err := api.Delete(context.Background(), "gone"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("strict delete: %v", err)
	}
	if err := api.Remove(context.Background(), resource.ID("gone")); err != nil {
		t.Fatalf("ignore-missing remove: %v", err)
	}
	if err := api.Remove(context.Background(), resource.ID("gone"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("strict remove: %v", err)
	}
	deletes.Store(0)
	if err := api.Delete(context.Background(), "busy"); err == nil || deletes.Load() != 100 {
		t.Fatalf("conflict must surface once: %v %d", err, deletes.Load())
	}
}

func TestR2OctAZZoneListQueryPagingAndRaw(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []string
	cloud.Mux.HandleFunc("GET /v2/lbaas/availabilityzones", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		if r.URL.Query().Get("marker") == "" {
			testcloud.JSON(w, 200, `{"availability_zones":[{"name":"a","enabled":false}],"availability_zones_links":[{"rel":"next","href":"`+cloud.Server.URL+`/v2/lbaas/availabilityzones?marker=a&limit=1"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"availability_zones":[{"name":"b","description":null}]}`)
	})
	api := r2OctAZZonesAPI(t, cloud)
	var names []string
	for z, err := range api.List(context.Background(), availabilityzones.WithListOptions(availabilityzones.ListOpts{
		Enabled: r2OctAZBool(false), Description: r2OctAZStr(""), Limit: 1, Fields: []string{"name", "enabled"}, PageReverse: r2OctAZBool(false),
	}), availabilityzones.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, z.Name)
		if z.Name == "b" && string(z.Body["description"]) != "null" {
			t.Fatalf("raw body lost null: %v", z.Body)
		}
	}
	if len(names) != 2 || names[0] != "a" || names[1] != "b" || len(queries) != 2 {
		t.Fatalf("names=%v queries=%v", names, queries)
	}
	want := "description=&enabled=false&fields=name&fields=enabled&limit=1&page_reverse=False&vendor=x"
	if queries[0] != want {
		t.Fatalf("query %q", queries[0])
	}
	if queries[1] != "description=&enabled=false&fields=name&fields=enabled&limit=1&marker=a&page_reverse=False&vendor=x" {
		t.Fatalf("continuation lost filters: %q", queries[1])
	}
	if _, err := availabilityzones.New(cloud.Client("load-balancer", "/v2")).All(context.Background(), resource.WithMaxItems(1)); err != nil {
		t.Fatal(err)
	}
	for z, err := range api.List(context.Background(), availabilityzones.WithListQuery("enabled", "true")) {
		if err == nil || z != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("core query collision: %v", err)
		}
	}
}

func TestR2OctAZZoneListCycleAndCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/lbaas/availabilityzones", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"availability_zones":[],"availability_zones_links":[{"rel":"next","href":"`+cloud.Server.URL+`/v2/lbaas/availabilityzones"}]}`)
	})
	api := r2OctAZZonesAPI(t, cloud)
	var gotErr error
	for _, err := range api.List(context.Background()) {
		gotErr = err
	}
	if !errors.Is(gotErr, resource.ErrPaginationCycle) {
		t.Fatalf("%v", gotErr)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, err := range api.List(ctx) {
		gotErr = err
	}
	if !errors.Is(gotErr, context.Canceled) {
		t.Fatalf("%v", gotErr)
	}
	if _, err := api.Create(ctx, availabilityzones.CreateOpts{Name: "a", AvailabilityZoneProfileID: "p"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
}

func TestR2OctAZZoneFindIdentityGetFirstThenList(t *testing.T) {
	cloud := testcloud.New(t)
	var order []string
	cloud.Mux.HandleFunc("GET /v2/lbaas/availabilityzones/direct", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "get-direct")
		testcloud.JSON(w, 200, `{"availability_zone":{"name":"direct"}}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/availabilityzones/listed", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "get-listed")
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/availabilityzones", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "list?"+r.URL.RawQuery)
		testcloud.JSON(w, 200, `{"availability_zones":[{"name":"listed"}]}`)
	})
	api := r2OctAZZonesAPI(t, cloud)
	if z, err := api.FindIdentity(context.Background(), "direct"); err != nil || z.Name != "direct" {
		t.Fatalf("%+v %v", z, err)
	}
	if z, err := api.FindIdentity(context.Background(), "listed"); err != nil || z.Name != "listed" {
		t.Fatalf("%+v %v", z, err)
	}
	want := []string{"get-direct", "get-listed", "list?name=listed"}
	if len(order) != 3 || order[0] != want[0] || order[1] != want[1] || order[2] != want[2] {
		t.Fatalf("%v", order)
	}
	if z, err := api.Find(context.Background(), resource.Name("listed")); err != nil || z.Name != "listed" {
		t.Fatalf("name find %+v %v", z, err)
	}
}
