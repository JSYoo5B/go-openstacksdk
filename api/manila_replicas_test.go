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
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/replicas"
	"github.com/gophercloud/gophercloud/v2"
)

const manilaReplicaVersion = "2.56"

const manilaReplicaJSON = `{"id":"rep-1","status":"available","replica_state":"in_sync","share_id":"share-1","share_network_id":"net-1","share_server_id":"srv-1","host":"host@backend","availability_zone":"az1","cast_rules_to_readonly":false,"created_at":"2026-01-02T03:04:05.000000","updated_at":"2026-01-02T03:04:06.000000","vendor_extra":1}`

func newManilaReplicas(t *testing.T, cloud *testcloud.Cloud) *replicas.API {
	t.Helper()
	client := cloud.Client("shared-file-system", "/v2")
	client.Microversion = manilaReplicaVersion
	return replicas.New(client)
}

func manilaReplicaVersionOK(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("X-Openstack-Manila-Api-Version"); got != manilaReplicaVersion {
		t.Errorf("%s %s microversion header=%q", r.Method, r.URL.Path, got)
	}
}

func manilaReplicaBody(t *testing.T, r *http.Request) string {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("request body: %v", err)
	}
	return string(data)
}

func manilaReplicaJSONEqual(t *testing.T, got, want string) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal([]byte(got), &g); err != nil {
		t.Errorf("body %q: %v", got, err)
		return
	}
	_ = json.Unmarshal([]byte(want), &w)
	if !reflect.DeepEqual(g, w) {
		t.Errorf("body=%s want=%s", got, want)
	}
}

func TestManilaReplicaCreateSendsEnvelopeAndRequires202(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{202, true}, {200, false}, {201, false}, {204, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /v2/share-replicas", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			manilaReplicaVersionOK(t, r)
			manilaReplicaJSONEqual(t, manilaReplicaBody(t, r), `{"share_replica":{"share_id":"share-1","share_network_id":"net-1","availability_zone":"az1","scheduler_hints":{"only_host":"h1"},"vendor":{"enabled":false}}}`)
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, `{"share_replica":`+manilaReplicaJSON+`}`)
		})
		value, err := newManilaReplicas(t, cloud).Create(context.Background(), replicas.CreateOpts{
			ShareID: "share-1", ShareNetworkID: "net-1", AvailabilityZone: "az1", SchedulerHints: map[string]string{"only_host": "h1"},
		}, replicas.WithCreateField("vendor", map[string]bool{"enabled": false}))
		if calls.Load() != 1 || tc.success != (err == nil) {
			t.Fatalf("status %d calls=%d value=%+v error=%v", tc.status, calls.Load(), value, err)
		}
		if tc.success {
			if value == nil || value.ID != "rep-1" || value.State != "in_sync" || value.Status != "available" || value.CastRulesToReadonly || value.CreatedAt.IsZero() || value.UpdatedAt.IsZero() {
				t.Fatalf("value=%+v", value)
			}
			continue
		}
		var unexpected gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &unexpected) || unexpected.Actual != tc.status {
			t.Fatalf("status %d error=%v", tc.status, err)
		}
	}
}

func TestManilaReplicaCreateOmitsEmptyOptionalFieldsAndRequiresShareID(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v2/share-replicas", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		manilaReplicaJSONEqual(t, manilaReplicaBody(t, r), `{"share_replica":{"share_id":"share-1"}}`)
		testcloud.JSON(w, 202, `{"share_replica":`+manilaReplicaJSON+`}`)
	})
	api := newManilaReplicas(t, cloud)
	if _, err := api.Create(context.Background(), replicas.CreateOpts{ShareID: "share-1", SchedulerHints: map[string]string{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Create(context.Background(), replicas.CreateOpts{}); err == nil {
		t.Fatal("missing share_id was accepted")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestManilaReplicaRejectsCoreFieldCollisionsAndNilOptionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newManilaReplicas(t, cloud)
	ctx := context.Background()
	checks := map[string]error{}
	_, checks["create core"] = api.Create(ctx, replicas.CreateOpts{ShareID: "s"}, replicas.WithCreateField("share_id", "other"))
	_, checks["create omitted core"] = api.Create(ctx, replicas.CreateOpts{ShareID: "s"}, replicas.WithCreateField("availability_zone", "az"))
	_, checks["create nil"] = api.Create(ctx, replicas.CreateOpts{ShareID: "s"}, nil)
	checks["promote core"] = api.Promote(ctx, "rep-1", replicas.PromoteOpts{}, replicas.WithPromoteField("quiesce_wait_time", 5))
	checks["promote nil"] = api.Promote(ctx, "rep-1", replicas.PromoteOpts{}, nil)
	checks["reset state core"] = api.ResetState(ctx, "rep-1", replicas.ResetStateOpts{State: "in_sync"}, replicas.WithResetStateField("replica_state", "x"))
	checks["reset state nil"] = api.ResetState(ctx, "rep-1", replicas.ResetStateOpts{State: "in_sync"}, nil)
	checks["reset status core"] = api.ResetStatus(ctx, "rep-1", replicas.ResetStatusOpts{Status: "error"}, replicas.WithResetStatusField("status", "x"))
	checks["reset status nil"] = api.ResetStatus(ctx, "rep-1", replicas.ResetStatusOpts{Status: "error"}, nil)
	for name, err := range checks {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for range api.List(ctx, nil) {
	}
	for range api.ListDetail(ctx, nil) {
	}
	for _, seq := range []func() error{
		func() error {
			for _, err := range api.List(ctx, nil) {
				return err
			}
			return nil
		},
		func() error {
			for _, err := range api.ListDetail(ctx, nil) {
				return err
			}
			return nil
		},
	} {
		if err := seq(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("nil list option: %v", err)
		}
	}
}

func TestManilaReplicaActionsSendNamedBodiesAndRequire202(t *testing.T) {
	type action struct {
		name string
		want string
		call func(*replicas.API) error
	}
	actions := []action{
		{"promote default", `{"promote":{}}`, func(a *replicas.API) error {
			return a.Promote(context.Background(), "rep-1", replicas.PromoteOpts{})
		}},
		{"promote quiesce and extension", `{"promote":{"quiesce_wait_time":30,"vendor":false}}`, func(a *replicas.API) error {
			return a.Promote(context.Background(), "rep-1", replicas.PromoteOpts{QuiesceWaitTime: 30}, replicas.WithPromoteField("vendor", false))
		}},
		{"resync", `{"resync":null}`, func(a *replicas.API) error { return a.Resync(context.Background(), "rep-1") }},
		{"force delete", `{"force_delete":null}`, func(a *replicas.API) error { return a.ForceDelete(context.Background(), "rep-1") }},
		{"reset status", `{"reset_status":{"status":"error","vendor":{}}}`, func(a *replicas.API) error {
			return a.ResetStatus(context.Background(), "rep-1", replicas.ResetStatusOpts{Status: "error"}, replicas.WithResetStatusField("vendor", struct{}{}))
		}},
		{"reset state", `{"reset_replica_state":{"replica_state":"out_of_sync"}}`, func(a *replicas.API) error {
			return a.ResetState(context.Background(), "rep-1", replicas.ResetStateOpts{State: "out_of_sync"})
		}},
	}
	for _, act := range actions {
		for _, status := range []int{202, 200, 204} {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("POST /v2/share-replicas/rep-1/action", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				manilaReplicaVersionOK(t, r)
				manilaReplicaJSONEqual(t, manilaReplicaBody(t, r), act.want)
				w.WriteHeader(status)
			})
			err := act.call(newManilaReplicas(t, cloud))
			if calls.Load() != 1 || (status == 202) != (err == nil) {
				t.Fatalf("%s status %d calls=%d error=%v", act.name, status, calls.Load(), err)
			}
			if status != 202 {
				var unexpected gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &unexpected) || unexpected.Actual != status {
					t.Fatalf("%s status %d error=%v", act.name, status, err)
				}
			}
		}
	}
}

func TestManilaReplicaResetActionsRequireNonEmptyTargetBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
	})
	api := newManilaReplicas(t, cloud)
	if err := api.ResetStatus(context.Background(), "rep-1", replicas.ResetStatusOpts{}); err == nil {
		t.Fatal("empty status accepted")
	}
	if err := api.ResetState(context.Background(), "rep-1", replicas.ResetStateOpts{}); err == nil {
		t.Fatal("empty replica_state accepted")
	}
}

func TestManilaReplicaDeleteAccepts202And204AndKeepsNative404(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{202, true}, {204, true}, {200, false}, {404, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("DELETE /v2/share-replicas/rep-1", func(w http.ResponseWriter, r *http.Request) {
			manilaReplicaVersionOK(t, r)
			w.WriteHeader(tc.status)
		})
		err := newManilaReplicas(t, cloud).Delete(context.Background(), "rep-1")
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
	cloud.Mux.HandleFunc("DELETE /v2/share-replicas/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	if err := newManilaReplicas(t, cloud).Remove(context.Background(), resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
	if err := newManilaReplicas(t, cloud).Remove(context.Background(), resource.ID("missing"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("Remove strict missing: %v", err)
	}
}

func TestManilaReplicaGetAndExportLocations(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/share-replicas/rep-1", func(w http.ResponseWriter, r *http.Request) {
		manilaReplicaVersionOK(t, r)
		testcloud.JSON(w, 200, `{"share_replica":`+manilaReplicaJSON+`}`)
	})
	loc := `{"id":"loc-1","path":"10.0.0.1:/share","share_instance_id":"inst-1","is_admin_only":false,"preferred":true,"availability_zone":"az1","replica_state":"in_sync","created_at":"2026-01-02T03:04:05.000000","updated_at":"2026-01-02T03:04:06.000000"}`
	cloud.Mux.HandleFunc("GET /v2/share-replicas/rep-1/export-locations", func(w http.ResponseWriter, r *http.Request) {
		manilaReplicaVersionOK(t, r)
		testcloud.JSON(w, 200, `{"export_locations":[`+loc+`]}`)
	})
	cloud.Mux.HandleFunc("GET /v2/share-replicas/rep-1/export-locations/loc-1", func(w http.ResponseWriter, r *http.Request) {
		manilaReplicaVersionOK(t, r)
		testcloud.JSON(w, 200, `{"export_location":`+loc+`}`)
	})
	cloud.Mux.HandleFunc("GET /v2/share-replicas/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/share-replicas/rep-1/export-locations/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := newManilaReplicas(t, cloud)
	ctx := context.Background()
	value, err := api.Get(ctx, "rep-1")
	if err != nil || value.ID != "rep-1" || value.Host != "host@backend" || value.ShareServerID != "srv-1" || value.CreatedAt.IsZero() {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	list, err := api.ListExportLocations(ctx, "rep-1")
	if err != nil || len(list) != 1 || list[0].ID != "loc-1" || list[0].IsAdminOnly || !list[0].Preferred || list[0].State != "in_sync" || list[0].UpdatedAt.IsZero() {
		t.Fatalf("list=%+v error=%v", list, err)
	}
	one, err := api.GetExportLocation(ctx, "rep-1", "loc-1")
	if err != nil || one.Path != "10.0.0.1:/share" || one.ShareInstanceID != "inst-1" {
		t.Fatalf("one=%+v error=%v", one, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.Get(ctx, "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	if _, err := api.GetExportLocation(ctx, "rep-1", "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native export location 404: %v", err)
	}
	if _, err := api.Find(ctx, resource.ID("missing"), resource.WithIgnoreMissing()); err != nil {
		t.Fatalf("Find ignore-missing: %v", err)
	}
}

func TestManilaReplicaListAndListDetailFiltersFollowOffsetPaging(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
		call func(*replicas.API) func(func(*replicas.Replica, error) bool)
	}{
		{"list", "/v2/share-replicas", func(a *replicas.API) func(func(*replicas.Replica, error) bool) {
			return a.List(context.Background(), replicas.WithListOptions(replicas.ListOpts{ShareID: "share-1", Limit: 1}), replicas.WithListQuery("vendor", "x"))
		}},
		{"detail", "/v2/share-replicas/detail", func(a *replicas.API) func(func(*replicas.Replica, error) bool) {
			return a.ListDetail(context.Background(), replicas.WithListDetailOptions(replicas.ListOpts{ShareID: "share-1", Limit: 1}), replicas.WithListDetailQuery("vendor", "x"))
		}},
	} {
		cloud := testcloud.New(t)
		var requests []url.Values
		cloud.Mux.HandleFunc("GET "+tc.path, func(w http.ResponseWriter, r *http.Request) {
			manilaReplicaVersionOK(t, r)
			requests = append(requests, r.URL.Query())
			switch r.URL.Query().Get("offset") {
			case "":
				testcloud.JSON(w, 200, `{"share_replicas":[{"id":"rep-1","status":"available"}]}`)
			case "1":
				testcloud.JSON(w, 200, `{"share_replicas":[{"id":"rep-2","status":"error"}]}`)
			default:
				testcloud.JSON(w, 200, `{"share_replicas":[]}`)
			}
		})
		var ids []string
		for value, err := range tc.call(newManilaReplicas(t, cloud)) {
			if err != nil {
				t.Fatal(tc.name, err)
			}
			ids = append(ids, value.ID)
		}
		if !reflect.DeepEqual(ids, []string{"rep-1", "rep-2"}) || len(requests) != 3 {
			t.Fatalf("%s ids=%v requests=%v", tc.name, ids, requests)
		}
		first := requests[0]
		if first.Get("share_id") != "share-1" || first.Get("limit") != "1" || first.Get("vendor") != "x" || first.Has("offset") {
			t.Fatalf("%s first query=%v", tc.name, first)
		}
		if requests[1].Get("offset") != "1" || requests[2].Get("offset") != "2" || requests[1].Get("share_id") != "share-1" {
			t.Fatalf("%s paging queries=%v", tc.name, requests)
		}
	}
}

func TestManilaReplicaListQueryExtensionReplacesTypedFilter(t *testing.T) {
	cloud := testcloud.New(t)
	var query url.Values
	cloud.Mux.HandleFunc("GET /v2/share-replicas", func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		testcloud.JSON(w, 200, `{"share_replicas":[]}`)
	})
	for range newManilaReplicas(t, cloud).List(context.Background(), replicas.WithListOptions(replicas.ListOpts{ShareID: "typed"}), replicas.WithListQuery("share_id", "extension")) {
	}
	if !reflect.DeepEqual(query["share_id"], []string{"extension"}) {
		t.Fatalf("query=%v", query)
	}
}

func TestManilaReplicaCommonLookupUsesDetailListAndLocalStatus(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/share-replicas/detail", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"share_replicas":[{"id":"rep-1","status":"available"},{"id":"rep-2","status":"error"}]}`)
	})
	api := newManilaReplicas(t, cloud)
	values, err := api.All(context.Background(), resource.WithStatus("error"), resource.WithQuery("share_id", "share-1"))
	if err != nil || len(values) != 1 || values[0].ID != "rep-2" {
		t.Fatalf("values=%v error=%v", values, err)
	}
	if len(queries) != 1 || queries[0].Has("status") || queries[0].Get("share_id") != "share-1" {
		t.Fatalf("status must be filtered locally, queries=%v", queries)
	}
}

func TestManilaReplicaCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	api := newManilaReplicas(t, cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(canceled, "rep-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("get: %v", err)
	}
	if _, err := api.Create(canceled, replicas.CreateOpts{ShareID: "s"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("create: %v", err)
	}
	if err := api.Promote(canceled, "rep-1", replicas.PromoteOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("promote: %v", err)
	}
	if err := api.Resync(canceled, "rep-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("resync: %v", err)
	}
	if err := api.ForceDelete(canceled, "rep-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("force delete: %v", err)
	}
	if err := api.Delete(canceled, "rep-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("delete: %v", err)
	}
	if _, err := api.ListExportLocations(canceled, "rep-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("export locations: %v", err)
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
	if err := api.ResetState(deadline, "rep-1", replicas.ResetStateOpts{State: "in_sync"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
