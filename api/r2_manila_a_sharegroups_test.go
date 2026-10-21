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
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharegroups"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharegroupsnapshots"
)

const r2ManilaAGroupJSON = `{"share_group":{"id":"g1","name":"n","status":"available","share_types":["t1"],"source_share_group_snapshot_id":null,"extra":1}}`

func TestR2ManilaAShareGroupCreateBodyAndAcknowledgement(t *testing.T) {
	cases := []struct {
		name string
		opts sharegroups.CreateOpts
		body string
	}{
		{"empty", sharegroups.CreateOpts{}, `{"share_group":{}}`},
		{"types-and-empty-description", sharegroups.CreateOpts{Name: request.Present("n"), Description: request.Present(""), ShareTypes: []string{"t1"}, ShareGroupTypeID: request.Present("gt")},
			`{"share_group":{"description":"","name":"n","share_group_type_id":"gt","share_types":["t1"]}}`},
		{"empty-types-list", sharegroups.CreateOpts{ShareTypes: []string{}}, `{"share_group":{"share_types":[]}}`},
		{"source-snapshot", sharegroups.CreateOpts{SourceShareGroupSnapshotID: request.Present("snap"), AvailabilityZone: request.Null[string]()}, `{"share_group":{"availability_zone":null,"source_share_group_snapshot_id":"snap"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("POST /v2/share-groups", func(w http.ResponseWriter, r *http.Request) {
				r2ManilaAVersion(t, r, "2.55")
				if got := r2ManilaARead(t, r); got != tc.body {
					t.Errorf("body = %s", got)
				}
				testcloud.JSON(w, 202, r2ManilaAGroupJSON)
			})
			value, err := sharegroups.New(r2ManilaAClient(cloud, "2.55")).Create(context.Background(), tc.opts)
			if err != nil || value.ID != "g1" || value.SourceShareGroupSnapshotID != "" || string(value.Evidence.Body["source_share_group_snapshot_id"]) != "null" || value.Evidence.StatusCode != 202 || len(value.ShareTypes) != 1 {
				t.Fatalf("value=%+v err=%v", value, err)
			}
		})
	}
}

func TestR2ManilaAShareGroupFloorProtectedFieldsAndNoReplay(t *testing.T) {
	cloud := testcloud.New(t)
	calls := r2ManilaAGuard(t, cloud)
	ctx := context.Background()
	for _, version := range []string{"", "2.54", "2.31"} {
		if _, err := sharegroups.New(r2ManilaAClient(cloud, version)).Get(ctx, "g1"); !errors.Is(err, resource.ErrUnsupported) {
			t.Errorf("%q: %v", version, err)
		}
	}
	api := sharegroups.New(r2ManilaAClient(cloud, "2.55"))
	for _, key := range []string{"name", "share_types", "id", "status", "project_id", "share_group"} {
		if _, err := api.Create(ctx, sharegroups.CreateOpts{}, sharegroups.WithCreateField(key, "x")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", key, err)
		}
	}
	if _, err := api.Update(ctx, "g1", sharegroups.UpdateOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Errorf("empty update: %v", err)
	}
	if _, err := api.Create(r2ManilaACanceled(), sharegroups.CreateOpts{}); !errors.Is(err, context.Canceled) {
		t.Error(err)
	}
	if calls.Load() != 0 {
		t.Fatal("local failure sent a request")
	}
	cloud2 := testcloud.New(t)
	var posts atomic.Int32
	cloud2.Mux.HandleFunc("POST /v2/share-groups", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		testcloud.JSON(w, 503, `{}`)
	})
	_, err := sharegroups.New(r2ManilaAClient(cloud2, "latest")).Create(ctx, sharegroups.CreateOpts{})
	r2ManilaACode(t, err, 503)
	if posts.Load() != 1 {
		t.Fatalf("replayed: %d", posts.Load())
	}
}

func TestR2ManilaAShareGroupReadsUpdateDeleteAndRoutes(t *testing.T) {
	cloud := testcloud.New(t)
	var listPath, listQuery string
	list := func(w http.ResponseWriter, r *http.Request) {
		listPath, listQuery = r.URL.Path, r.URL.RawQuery
		testcloud.JSON(w, 200, `{"share_groups":[{"id":"g1","name":"n"},{"id":"g2","name":"m"}]}`)
	}
	cloud.Mux.HandleFunc("GET /v2/share-groups", list)
	cloud.Mux.HandleFunc("GET /v2/share-groups/detail", list)
	cloud.Mux.HandleFunc("GET /v2/share-groups/g1", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, r2ManilaAGroupJSON) })
	cloud.Mux.HandleFunc("PUT /v2/share-groups/g1", func(w http.ResponseWriter, r *http.Request) {
		if got := r2ManilaARead(t, r); got != `{"share_group":{"description":null,"name":"new"}}` {
			t.Errorf("body = %s", got)
		}
		testcloud.JSON(w, 200, r2ManilaAGroupJSON)
	})
	var del atomic.Int32
	del.Store(202)
	cloud.Mux.HandleFunc("DELETE /v2/share-groups/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, int(del.Load()), `{}`) })
	api := sharegroups.New(r2ManilaAClient(cloud, "2.55"))
	ctx := context.Background()
	n := 0
	for _, err := range api.List(ctx, sharegroups.WithListOptions(sharegroups.ListOpts{Name: "n"})) {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 2 || listPath != "/v2/share-groups" || listQuery != "name=n" {
		t.Fatalf("n=%d path=%s query=%s", n, listPath, listQuery)
	}
	for _, err := range api.ListDetail(ctx) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if listPath != "/v2/share-groups/detail" {
		t.Fatal(listPath)
	}
	if v, err := api.Get(ctx, "g1"); err != nil || v.Status != "available" {
		t.Fatal(v, err)
	}
	if v, err := api.Update(ctx, "g1", sharegroups.UpdateOpts{Name: request.Present("new"), Description: request.Null[string]()}); err != nil || v.ID != "g1" {
		t.Fatal(v, err)
	}
	if err := api.Delete(ctx, "g1"); err != nil {
		t.Fatal(err)
	}
	del.Store(404)
	r2ManilaACode(t, api.Delete(ctx, "g1"), 404)
	if err := api.Remove(ctx, resource.ID("g1")); err != nil {
		t.Errorf("default ignore_missing: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("g1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Errorf("strict: %v", err)
	}
}

// find_share_group: GET by ID first, then an ID-or-name match across the list;
// Python searches /share-groups (summary rows), selected with Details(false).
func TestR2ManilaAShareGroupFindIdentityMatchesPythonFind(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, summary, detail atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/share-groups/by-name", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		testcloud.JSON(w, 400, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v2/share-groups/dup", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); testcloud.JSON(w, 403, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/share-groups/boom", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); testcloud.JSON(w, 500, `{}`) })
	rows := `{"share_groups":[{"id":"g1","name":"by-name"},{"id":"g2","name":"dup"},{"id":"g3","name":"dup"}]}`
	cloud.Mux.HandleFunc("GET /v2/share-groups", func(w http.ResponseWriter, r *http.Request) {
		summary.Add(1)
		if r.URL.RawQuery != "" {
			t.Errorf("Python sends no name filter: %s", r.URL.RawQuery)
		}
		testcloud.JSON(w, 200, rows)
	})
	cloud.Mux.HandleFunc("GET /v2/share-groups/detail", func(w http.ResponseWriter, r *http.Request) { detail.Add(1); testcloud.JSON(w, 200, rows) })
	api := sharegroups.New(r2ManilaAClient(cloud, "2.55"))
	ctx := context.Background()
	v, err := api.FindIdentity(ctx, "by-name", resource.WithIdentityFindDetails(false))
	if err != nil || v == nil || v.ID != "g1" || gets.Load() != 1 || summary.Load() != 1 || detail.Load() != 0 {
		t.Fatalf("v=%v err=%v gets=%d", v, err, gets.Load())
	}
	if _, err := api.FindIdentity(ctx, "dup", resource.WithIdentityFindDetails(false)); err == nil || !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatalf("duplicate names: %v", err)
	}
	if _, err := api.FindIdentity(ctx, "boom"); err == nil {
		t.Fatal("500 must not fall back")
	}
	if v, err := api.FindIdentity(ctx, "dup"); err == nil || v != nil || detail.Load() != 1 {
		t.Fatalf("default searches detail rows: %v %v %d", v, err, detail.Load())
	}
}

const r2ManilaAGroupSnapshotJSON = `{"share_group_snapshot":{"id":"s1","share_group_id":"g1","status":"creating","size":null,"members":[{"id":"m"}]}}`

func TestR2ManilaAShareGroupSnapshotLifecycleRoutes(t *testing.T) {
	cloud := testcloud.New(t)
	var listPath string
	list := func(w http.ResponseWriter, r *http.Request) {
		listPath = r.URL.Path
		testcloud.JSON(w, 200, `{"share_group_snapshots":[{"id":"s1"}]}`)
	}
	cloud.Mux.HandleFunc("GET /v2/share-group-snapshots", list)
	cloud.Mux.HandleFunc("GET /v2/share-group-snapshots/detail", list)
	cloud.Mux.HandleFunc("POST /v2/share-group-snapshots", func(w http.ResponseWriter, r *http.Request) {
		r2ManilaAVersion(t, r, "2.60")
		if got := r2ManilaARead(t, r); got != `{"share_group_snapshot":{"description":"","name":"n","share_group_id":"g1"}}` {
			t.Errorf("body = %s", got)
		}
		testcloud.JSON(w, 202, r2ManilaAGroupSnapshotJSON)
	})
	cloud.Mux.HandleFunc("GET /v2/share-group-snapshots/s1", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, r2ManilaAGroupSnapshotJSON) })
	cloud.Mux.HandleFunc("PUT /v2/share-group-snapshots/s1", func(w http.ResponseWriter, r *http.Request) {
		if got := r2ManilaARead(t, r); got != `{"share_group_snapshot":{"name":"x"}}` {
			t.Errorf("body = %s", got)
		}
		testcloud.JSON(w, 200, r2ManilaAGroupSnapshotJSON)
	})
	cloud.Mux.HandleFunc("DELETE /v2/share-group-snapshots/s1", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(202) })
	cloud.Mux.HandleFunc("POST /v2/share-group-snapshots/s1/action", func(w http.ResponseWriter, r *http.Request) {
		if got := r2ManilaARead(t, r); got != `{"reset_status":{"status":"error"}}` {
			t.Errorf("body = %s", got)
		}
		w.WriteHeader(202)
	})
	cloud.Mux.HandleFunc("GET /v2/share-group-snapshots/s1/members", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"share_group_snapshot_members":[{"id":"m1","share_id":"sh","size":null}],"other":true}`)
	})
	api := sharegroupsnapshots.New(r2ManilaAClient(cloud, "2.60"))
	ctx := context.Background()
	for _, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if listPath != "/v2/share-group-snapshots/detail" {
		t.Fatalf("details default: %s", listPath)
	}
	for _, err := range api.List(ctx, sharegroupsnapshots.WithListDetails(false), sharegroupsnapshots.WithListOptions(sharegroupsnapshots.ListOpts{ShareGroupID: "g1"})) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if listPath != "/v2/share-group-snapshots" {
		t.Fatalf("summary: %s", listPath)
	}
	v, err := api.Create(ctx, sharegroupsnapshots.CreateOpts{ShareGroupID: "g1", Name: request.Present("n"), Description: request.Present("")})
	if err != nil || v.ID != "s1" || v.Size != nil || string(v.Members) != `[{"id":"m"}]` {
		t.Fatal(v, err)
	}
	if _, err := api.Get(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "s1", sharegroupsnapshots.UpdateOpts{Name: request.Present("x")}); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if err := api.ResetStatus(ctx, "s1", "error"); err != nil {
		t.Fatal(err)
	}
	m, err := api.ListMembers(ctx, "s1")
	if err != nil || len(m.Items) != 1 || m.Items[0].ID != "m1" || m.Items[0].Size != nil || string(m.Evidence.Body["other"]) != "true" {
		t.Fatalf("%+v %v", m, err)
	}
}

func TestR2ManilaAShareGroupSnapshotValidationErrorsAndPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	calls := r2ManilaAGuard(t, cloud)
	ctx := context.Background()
	if _, err := sharegroupsnapshots.New(r2ManilaAClient(cloud, "2.54")).Get(ctx, "s1"); !errors.Is(err, resource.ErrUnsupported) {
		t.Error(err)
	}
	api := sharegroupsnapshots.New(r2ManilaAClient(cloud, "2.55"))
	if _, err := api.Create(ctx, sharegroupsnapshots.CreateOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Error("share_group_id is required:", err)
	}
	if _, err := api.Create(ctx, sharegroupsnapshots.CreateOpts{ShareGroupID: "g"}, sharegroupsnapshots.WithCreateField("share_group_id", "h")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Error(err)
	}
	if err := api.ResetStatus(ctx, "s1", " "); !errors.Is(err, resource.ErrInvalidOption) {
		t.Error(err)
	}
	if err := api.ResetStatus(r2ManilaACanceled(), "s1", "error"); !errors.Is(err, context.Canceled) {
		t.Error(err)
	}
	if _, err := api.ListMembers(ctx, "a/b"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Error(err)
	}
	if calls.Load() != 0 {
		t.Fatal("local failure sent a request")
	}
	cloud2 := testcloud.New(t)
	var posts atomic.Int32
	cloud2.Mux.HandleFunc("POST /v2/share-group-snapshots/s1/action", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		testcloud.JSON(w, 409, `{}`)
	})
	cloud2.Mux.HandleFunc("GET /v2/share-group-snapshots/s1/members", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api2 := sharegroupsnapshots.New(r2ManilaAClient(cloud2, "2.55"))
	r2ManilaACode(t, api2.ResetStatus(ctx, "s1", "error"), 409)
	if posts.Load() != 1 {
		t.Fatal("reset replayed")
	}
	_, err := api2.ListMembers(ctx, "s1")
	r2ManilaACode(t, err, 404)
}
