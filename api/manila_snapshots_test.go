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
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/snapshots"
	"github.com/gophercloud/gophercloud/v2"
)

const manilaSnapshotVersion = "2.40"

const manilaSnapshotJSON = `{"id":"snap-1","name":"nightly","description":"","share_id":"share-1","share_proto":"NFS","share_size":2,"size":1,"status":"available","project_id":"proj-1","created_at":"2026-01-02T03:04:05.000000","links":[{"rel":"self","href":"http://example.invalid/snap-1"}],"vendor_extra":1}`

func newManilaSnapshots(t *testing.T, cloud *testcloud.Cloud) *snapshots.API {
	t.Helper()
	client := cloud.Client("shared-file-system", "/v2")
	client.Microversion = manilaSnapshotVersion
	return snapshots.New(client)
}

func manilaSnapshotRequest(t *testing.T, r *http.Request, wantBody string) {
	t.Helper()
	if got := r.Header.Get("X-Openstack-Manila-Api-Version"); got != manilaSnapshotVersion {
		t.Errorf("%s %s microversion header=%q", r.Method, r.URL.Path, got)
	}
	if wantBody == "" {
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("request body: %v", err)
	}
	var got, want any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("body %q: %v", data, err)
		return
	}
	_ = json.Unmarshal([]byte(wantBody), &want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("body=%s want=%s", data, wantBody)
	}
}

func manilaSnapshotStr(value string) *string { return &value }

func TestManilaSnapshotCreateBodyExtensionsAndSuccessCodes(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {201, true}, {202, true}, {204, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /v2/snapshots", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			manilaSnapshotRequest(t, r, `{"snapshot":{"share_id":"share-1","name":"nightly","display_name":"legacy","vendor":{"enabled":false}}}`)
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, `{"snapshot":`+manilaSnapshotJSON+`}`)
		})
		value, err := newManilaSnapshots(t, cloud).Create(context.Background(),
			snapshots.CreateOpts{ShareID: "share-1", Name: "nightly", DisplayName: "legacy"},
			snapshots.WithCreateField("vendor", map[string]bool{"enabled": false}))
		if calls.Load() != 1 || tc.success != (err == nil) {
			t.Fatalf("status %d calls=%d value=%+v error=%v", tc.status, calls.Load(), value, err)
		}
		if tc.success {
			if value == nil || value.ID != "snap-1" || value.ShareProto != "NFS" || value.ShareSize != 2 || value.Size != 1 || value.CreatedAt.IsZero() || len(value.Links) != 1 {
				t.Fatalf("value=%+v", value)
			}
			continue
		}
		var unexpected gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &unexpected) || unexpected.Actual != 204 {
			t.Fatalf("error=%v", err)
		}
	}
}

func TestManilaSnapshotCreateOmitsEmptyOptionalFieldsAndRequiresShareID(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v2/snapshots", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		manilaSnapshotRequest(t, r, `{"snapshot":{"share_id":"share-1"}}`)
		testcloud.JSON(w, 202, `{"snapshot":`+manilaSnapshotJSON+`}`)
	})
	api := newManilaSnapshots(t, cloud)
	if _, err := api.Create(context.Background(), snapshots.CreateOpts{ShareID: "share-1", Name: "", Description: ""}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Create(context.Background(), snapshots.CreateOpts{Name: "x"}); err == nil {
		t.Fatal("missing share_id was accepted")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestManilaSnapshotRejectsCoreFieldCollisionsAndNilOptionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newManilaSnapshots(t, cloud)
	ctx := context.Background()
	checks := map[string]error{}
	_, checks["create core"] = api.Create(ctx, snapshots.CreateOpts{ShareID: "s"}, snapshots.WithCreateField("share_id", "other"))
	_, checks["create omitted core"] = api.Create(ctx, snapshots.CreateOpts{ShareID: "s"}, snapshots.WithCreateField("description", "d"))
	_, checks["create nil"] = api.Create(ctx, snapshots.CreateOpts{ShareID: "s"}, nil)
	_, checks["update core"] = api.Update(ctx, "snap-1", snapshots.UpdateOpts{}, snapshots.WithUpdateField("display_name", "x"))
	_, checks["update nil"] = api.Update(ctx, "snap-1", snapshots.UpdateOpts{}, nil)
	checks["reset core"] = api.ResetStatus(ctx, "snap-1", snapshots.ResetStatusOpts{Status: "error"}, snapshots.WithResetStatusField("status", "x"))
	checks["reset nil"] = api.ResetStatus(ctx, "snap-1", snapshots.ResetStatusOpts{Status: "error"}, nil)
	for name, err := range checks {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, err := range api.ListDetail(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("nil list option: %v", err)
		}
	}
}

func TestManilaSnapshotUpdateSendsEmptyStringsAndAcceptsOnly200(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PUT /v2/snapshots/snap-1", func(w http.ResponseWriter, r *http.Request) {
			manilaSnapshotRequest(t, r, `{"snapshot":{"display_name":"","display_description":"","vendor":null}}`)
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, `{"snapshot":`+manilaSnapshotJSON+`}`)
		})
		value, err := newManilaSnapshots(t, cloud).Update(context.Background(), "snap-1",
			snapshots.UpdateOpts{DisplayName: manilaSnapshotStr(""), DisplayDescription: manilaSnapshotStr("")},
			snapshots.WithUpdateField("vendor", nil))
		if tc.success != (err == nil) || (tc.success && (value == nil || value.ID != "snap-1")) {
			t.Fatalf("status %d value=%+v error=%v", tc.status, value, err)
		}
		if !tc.success {
			var unexpected gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &unexpected) || unexpected.Actual != tc.status {
				t.Fatalf("status %d error=%v", tc.status, err)
			}
		}
	}
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("PUT /v2/snapshots/snap-1", func(w http.ResponseWriter, r *http.Request) {
		manilaSnapshotRequest(t, r, `{"snapshot":{}}`)
		testcloud.JSON(w, 200, `{"snapshot":`+manilaSnapshotJSON+`}`)
	})
	if _, err := newManilaSnapshots(t, cloud).Update(context.Background(), "snap-1", snapshots.UpdateOpts{}); err != nil {
		t.Fatalf("empty update omits both fields: %v", err)
	}
}

func TestManilaSnapshotActionsSendNamedBodiesAndRequire202(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
		call func(*snapshots.API) error
	}{
		{"reset status", `{"reset_status":{"status":"error","vendor":1}}`, func(a *snapshots.API) error {
			return a.ResetStatus(context.Background(), "snap-1", snapshots.ResetStatusOpts{Status: "error"}, snapshots.WithResetStatusField("vendor", 1))
		}},
		{"reset status empty", `{"reset_status":{"status":""}}`, func(a *snapshots.API) error {
			return a.ResetStatus(context.Background(), "snap-1", snapshots.ResetStatusOpts{})
		}},
		{"force delete", `{"force_delete":null}`, func(a *snapshots.API) error { return a.ForceDelete(context.Background(), "snap-1") }},
	} {
		for _, status := range []int{202, 200, 204} {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("POST /v2/snapshots/snap-1/action", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				manilaSnapshotRequest(t, r, tc.want)
				w.WriteHeader(status)
			})
			err := tc.call(newManilaSnapshots(t, cloud))
			if calls.Load() != 1 || (status == 202) != (err == nil) {
				t.Fatalf("%s status %d calls=%d error=%v", tc.name, status, calls.Load(), err)
			}
			if status != 202 {
				var unexpected gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &unexpected) || unexpected.Actual != status {
					t.Fatalf("%s status %d error=%v", tc.name, status, err)
				}
			}
		}
	}
}

func TestManilaSnapshotDeleteAndGetKeepNative404UntilCommonLookup(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{202, true}, {204, true}, {200, false}, {404, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("DELETE /v2/snapshots/snap-1", func(w http.ResponseWriter, r *http.Request) {
			manilaSnapshotRequest(t, r, "")
			w.WriteHeader(tc.status)
		})
		err := newManilaSnapshots(t, cloud).Delete(context.Background(), "snap-1")
		if tc.success != (err == nil) {
			t.Fatalf("status %d error=%v", tc.status, err)
		}
		if !tc.success {
			var unexpected gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &unexpected) || unexpected.Actual != tc.status || errors.Is(err, resource.ErrNotFound) {
				t.Fatalf("status %d error=%v", tc.status, err)
			}
		}
	}
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/snapshots/snap-1", func(w http.ResponseWriter, r *http.Request) {
		manilaSnapshotRequest(t, r, "")
		testcloud.JSON(w, 200, `{"snapshot":`+manilaSnapshotJSON+`}`)
	})
	cloud.Mux.HandleFunc("/v2/snapshots/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := newManilaSnapshots(t, cloud)
	value, err := api.Get(context.Background(), "snap-1")
	if err != nil || value.Name != "nightly" || value.Description != "" || value.ProjectID != "proj-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.Get(context.Background(), "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	if err := api.Delete(context.Background(), "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native delete keeps the 404 response error: %v", err)
	}
	if err := api.Remove(context.Background(), resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
	if err := api.Remove(context.Background(), resource.ID("missing"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("strict Remove: %v", err)
	}
	if got, err := api.Find(context.Background(), resource.ID("missing"), resource.WithIgnoreMissing()); err != nil || got != nil {
		t.Fatalf("Find ignore-missing got=%v err=%v", got, err)
	}
}

func TestManilaSnapshotListDetailFiltersOmitFalseAndFollowOffsetPaging(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v2/snapshots/detail", func(w http.ResponseWriter, r *http.Request) {
		manilaSnapshotRequest(t, r, "")
		requests = append(requests, r.URL.Query())
		switch r.URL.Query().Get("offset") {
		case "":
			testcloud.JSON(w, 200, `{"snapshots":[{"id":"snap-1","name":"a"}]}`)
		case "1":
			testcloud.JSON(w, 200, `{"snapshots":[{"id":"snap-2","name":"b"}]}`)
		default:
			testcloud.JSON(w, 200, `{"snapshots":[]}`)
		}
	})
	api := newManilaSnapshots(t, cloud)
	var ids []string
	for value, err := range api.ListDetail(context.Background(),
		snapshots.WithListDetailOptions(snapshots.ListOpts{AllTenants: true, Name: "a", NamePattern: "n", ShareID: "share-1", Size: 1, Status: "available", Limit: 1, SortKey: "created_at", SortDir: "desc", ProjectID: "proj-1", Description: "d", DescriptionPattern: "dp"}),
		snapshots.WithListDetailQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"snap-1", "snap-2"}) || len(requests) != 3 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	want := url.Values{"all_tenants": {"true"}, "name": {"a"}, "name~": {"n"}, "share_id": {"share-1"}, "size": {"1"}, "status": {"available"}, "limit": {"1"}, "sort_key": {"created_at"}, "sort_dir": {"desc"}, "project_id": {"proj-1"}, "description": {"d"}, "description~": {"dp"}, "vendor": {"x"}}
	if !reflect.DeepEqual(requests[0], want) {
		t.Fatalf("first query=%v", requests[0])
	}
	if requests[1].Get("offset") != "1" || requests[2].Get("offset") != "2" {
		t.Fatalf("paging=%v", requests)
	}
	requests = nil
	for range api.ListDetail(context.Background(), snapshots.WithListDetailOptions(snapshots.ListOpts{AllTenants: false})) {
	}
	if len(requests) != 1 || len(requests[0]) != 0 {
		t.Fatalf("AllTenants=false is a zero value and must be omitted: %v", requests)
	}
}

func TestManilaSnapshotCommonLookupFindsExactNameThroughDetailList(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/snapshots/detail", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"snapshots":[{"id":"snap-1","name":"web","status":"available"},{"id":"snap-2","name":"web-2","status":"error"}]}`)
	})
	api := newManilaSnapshots(t, cloud)
	value, err := api.Find(context.Background(), resource.Name("web"))
	if err != nil || value == nil || value.ID != "snap-1" || len(queries) != 1 || queries[0].Get("name") != "web" {
		t.Fatalf("value=%+v error=%v queries=%v", value, err, queries)
	}
	if _, err := api.Find(context.Background(), resource.Name("absent")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing name: %v", err)
	}
	queries = nil
	values, err := api.All(context.Background(), resource.WithStatus("error"))
	if err != nil || len(values) != 1 || values[0].ID != "snap-2" || len(queries) != 1 || queries[0].Get("status") != "error" {
		t.Fatalf("values=%v error=%v queries=%v", values, err, queries)
	}
}

func TestManilaSnapshotCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	api := newManilaSnapshots(t, cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(canceled, "snap-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("get: %v", err)
	}
	if _, err := api.Create(canceled, snapshots.CreateOpts{ShareID: "s"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("create: %v", err)
	}
	if _, err := api.Update(canceled, "snap-1", snapshots.UpdateOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("update: %v", err)
	}
	if err := api.Delete(canceled, "snap-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("delete: %v", err)
	}
	if err := api.ForceDelete(canceled, "snap-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("force delete: %v", err)
	}
	if err := api.ResetStatus(canceled, "snap-1", snapshots.ResetStatusOpts{Status: "error"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("reset status: %v", err)
	}
	for _, err := range api.ListDetail(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("list detail: %v", err)
		}
		break
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, snapshots.CreateOpts{ShareID: "s"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
