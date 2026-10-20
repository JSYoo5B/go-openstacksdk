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

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shares"
	"github.com/gophercloud/gophercloud/v2"
)

const manilaShareBody = `{"share":{"id":"share-1","name":"data","size":1,"status":"creating","share_proto":"NFS","is_public":false,"snapshot_support":false,"has_replicas":false,"metadata":{"empty":""},"created_at":"2026-01-02T03:04:05.000000","vendor_extra":1}}`

// newManilaShares binds the shares API to an isolated fixture. An empty
// version leaves the client without a selected microversion.
func newManilaShares(cloud *testcloud.Cloud, version string) *shares.API {
	client := cloud.Client("sharev2", "/v2")
	client.Microversion = version
	return shares.New(client)
}

func manilaShareFalse() *bool { value := false; return &value }

func manilaShareRead(t *testing.T, r *http.Request) string {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("request body: %v", err)
	}
	return string(data)
}

func manilaShareVersion(t *testing.T, r *http.Request, want string) {
	t.Helper()
	got := r.Header.Values("X-Openstack-Manila-Api-Version")
	if want == "" {
		if len(got) != 0 {
			t.Errorf("unexpected microversion header %v", got)
		}
		return
	}
	if len(got) != 1 || got[0] != want {
		t.Errorf("microversion header=%v want %q", got, want)
	}
}

func manilaShareCode(t *testing.T, err error, want int) {
	t.Helper()
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &unexpected) || unexpected.Actual != want {
		t.Fatalf("want unexpected response code %d, got %v", want, err)
	}
}

func TestManilaShareCreateSendsExactBodyAndAcceptsOnly200Or201(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {201, true}, {202, false}, {204, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /v2/shares", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			manilaShareVersion(t, r, "2.82")
			want := `{"share":{"is_public":false,"metadata":{"empty":""},"scheduler_hints":{"same_host":"share-0"},"share_proto":"NFS","size":1,"vendor":{"enabled":false}}}`
			if got := manilaShareRead(t, r); got != want {
				t.Errorf("body=%s", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, manilaShareBody)
		})
		value, err := newManilaShares(cloud, "2.82").Create(context.Background(), shares.CreateOpts{
			ShareProto: "NFS", Size: 1, IsPublic: manilaShareFalse(), Metadata: map[string]string{"empty": ""},
			SchedulerHints: &shares.SchedulerHints{SameHost: "share-0"},
		}, shares.WithCreateField("vendor", map[string]bool{"enabled": false}))
		if tc.success != (err == nil) || calls.Load() != 1 {
			t.Fatalf("status %d value=%+v error=%v calls=%d", tc.status, value, err, calls.Load())
		}
		if tc.success && (value == nil || value.ID != "share-1" || value.IsPublic || value.Status != "creating" || value.CreatedAt.IsZero() || value.Metadata["empty"] != "") {
			t.Fatalf("value=%+v", value)
		}
		if !tc.success {
			manilaShareCode(t, err, tc.status)
		}
	}
}

func TestManilaShareCreateOmitsUnsetOptionalFieldsAndMicroversion(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /v2/shares", func(w http.ResponseWriter, r *http.Request) {
		manilaShareVersion(t, r, "")
		if got := manilaShareRead(t, r); got != `{"share":{"share_proto":"CEPHFS","size":0}}` {
			t.Errorf("body=%s", got)
		}
		testcloud.JSON(w, 200, manilaShareBody)
	})
	if _, err := newManilaShares(cloud, "").Create(context.Background(), shares.CreateOpts{ShareProto: "CEPHFS"}); err != nil {
		t.Fatal(err)
	}
}

func TestManilaShareRejectsProtectedAndNilOptionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newManilaShares(cloud, "2.82")
	ctx := context.Background()
	check := func(name string, err error) {
		t.Helper()
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	_, err := api.Create(ctx, shares.CreateOpts{ShareProto: "NFS", Size: 1}, shares.WithCreateField("share_proto", "CIFS"))
	check("create protected", err)
	_, err = api.Create(ctx, shares.CreateOpts{}, nil)
	check("create nil", err)
	_, err = api.Update(ctx, "share-1", shares.UpdateOpts{}, shares.WithUpdateField("is_public", true))
	check("update protected", err)
	_, err = api.Update(ctx, "share-1", shares.UpdateOpts{}, nil)
	check("update nil", err)
	check("extend protected", api.Extend(ctx, "share-1", shares.ExtendOpts{NewSize: 2}, shares.WithExtendField("new_size", 3)))
	check("extend nil", api.Extend(ctx, "share-1", shares.ExtendOpts{}, nil))
	check("shrink protected", api.Shrink(ctx, "share-1", shares.ShrinkOpts{NewSize: 1}, shares.WithShrinkField("new_size", 3)))
	check("shrink nil", api.Shrink(ctx, "share-1", shares.ShrinkOpts{}, nil))
	check("revert protected", api.Revert(ctx, "share-1", shares.RevertOpts{SnapshotID: "s"}, shares.WithRevertField("snapshot_id", "x")))
	check("revert nil", api.Revert(ctx, "share-1", shares.RevertOpts{}, nil))
	check("reset protected", api.ResetStatus(ctx, "share-1", shares.ResetStatusOpts{Status: "error"}, shares.WithResetStatusField("status", "active")))
	check("reset nil", api.ResetStatus(ctx, "share-1", shares.ResetStatusOpts{}, nil))
	_, err = api.GrantAccess(ctx, "share-1", shares.GrantAccessOpts{}, shares.WithGrantAccessField("access_to", "x"))
	check("grant protected", err)
	_, err = api.GrantAccess(ctx, "share-1", shares.GrantAccessOpts{}, nil)
	check("grant nil", err)
	check("revoke protected", api.RevokeAccess(ctx, "share-1", shares.RevokeAccessOpts{}, shares.WithRevokeAccessField("access_id", "x")))
	check("revoke nil", api.RevokeAccess(ctx, "share-1", shares.RevokeAccessOpts{}, nil))
	_, err = api.SetMetadata(ctx, "share-1", shares.SetMetadataOpts{}, shares.WithSetMetadataField("metadata", 1))
	check("set metadata protected", err)
	_, err = api.SetMetadata(ctx, "share-1", shares.SetMetadataOpts{}, nil)
	check("set metadata nil", err)
	_, err = api.UpdateMetadata(ctx, "share-1", shares.UpdateMetadataOpts{}, shares.WithUpdateMetadataField("metadata", 1))
	check("update metadata protected", err)
	_, err = api.UpdateMetadata(ctx, "share-1", shares.UpdateMetadataOpts{}, nil)
	check("update metadata nil", err)
	for _, err := range api.ListDetail(ctx, nil) {
		check("list nil", err)
	}
}

func TestManilaShareUpdateSendsEmptyValuesAndAcceptsOnly200(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PUT /v2/shares/share-1", func(w http.ResponseWriter, r *http.Request) {
			manilaShareVersion(t, r, "2.82")
			if got := manilaShareRead(t, r); got != `{"share":{"display_description":"","display_name":"","is_public":false,"vendor":null}}` {
				t.Errorf("body=%s", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, manilaShareBody)
		})
		empty := ""
		value, err := newManilaShares(cloud, "2.82").Update(context.Background(), "share-1", shares.UpdateOpts{
			DisplayName: &empty, DisplayDescription: &empty, IsPublic: manilaShareFalse(),
		}, shares.WithUpdateField("vendor", nil))
		if tc.success != (err == nil) || (tc.success && (value == nil || value.ID != "share-1")) {
			t.Fatalf("status %d value=%+v error=%v", tc.status, value, err)
		}
		if !tc.success {
			manilaShareCode(t, err, tc.status)
		}
	}
}

func TestManilaShareUpdateOmitsUnsetPointerFields(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("PUT /v2/shares/share-1", func(w http.ResponseWriter, r *http.Request) {
		if got := manilaShareRead(t, r); got != `{"share":{}}` {
			t.Errorf("body=%s", got)
		}
		testcloud.JSON(w, 200, manilaShareBody)
	})
	if _, err := newManilaShares(cloud, "2.82").Update(context.Background(), "share-1", shares.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
}

func TestManilaShareGetDecodesModelAndKeepsNative404(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/shares/share-1", func(w http.ResponseWriter, r *http.Request) {
		manilaShareVersion(t, r, "2.82")
		testcloud.JSON(w, 200, manilaShareBody)
	})
	cloud.Mux.HandleFunc("GET /v2/shares/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{"itemNotFound":{"code":404}}`)
	})
	cloud.Mux.HandleFunc("GET /v2/shares/no-content", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	api := newManilaShares(cloud, "2.82")
	value, err := api.Get(context.Background(), "share-1")
	if err != nil || value.Name != "data" || value.ShareProto != "NFS" || value.IsPublic || value.SnapshotSupport || value.HasReplicas || value.CreatedAt.Year() != 2026 {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	_, err = api.Get(context.Background(), "missing")
	manilaShareCode(t, err, 404)
	if errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native Get must not map 404 to ErrNotFound: %v", err)
	}
	_, err = api.Get(context.Background(), "no-content")
	manilaShareCode(t, err, 204)
}

func TestManilaShareDeleteAcceptsOnly202Or204AndRemoveIgnoresMissing(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{202, true}, {204, true}, {200, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("DELETE /v2/shares/share-1", func(w http.ResponseWriter, r *http.Request) {
			manilaShareVersion(t, r, "2.82")
			if body := manilaShareRead(t, r); body != "" {
				t.Errorf("delete body=%q", body)
			}
			w.WriteHeader(tc.status)
		})
		err := newManilaShares(cloud, "2.82").Delete(context.Background(), "share-1")
		if tc.success != (err == nil) {
			t.Fatalf("status %d error=%v", tc.status, err)
		}
		if !tc.success {
			manilaShareCode(t, err, tc.status)
		}
	}
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("DELETE /v2/shares/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	api := newManilaShares(cloud, "2.82")
	manilaShareCode(t, api.Delete(context.Background(), "missing"), 404)
	if err := api.Remove(context.Background(), resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores a missing share by default: %v", err)
	}
	if err := api.Remove(context.Background(), resource.ID("missing"), resource.WithMissingError()); err == nil {
		t.Fatal("Remove with WithMissingError must report the missing share")
	}
}

func TestManilaShareListDetailSendsFiltersAndFollowsOffsetPages(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v2/shares/detail", func(w http.ResponseWriter, r *http.Request) {
		manilaShareVersion(t, r, "2.82")
		requests = append(requests, r.URL.Query())
		switch r.URL.Query().Get("offset") {
		case "":
			testcloud.JSON(w, 200, `{"shares":[{"id":"share-1","name":"one"}]}`)
		case "1":
			testcloud.JSON(w, 200, `{"shares":[{"id":"share-2","name":"two"}]}`)
		default:
			testcloud.JSON(w, 200, `{"shares":[]}`)
		}
	})
	api := newManilaShares(cloud, "2.82")
	var ids []string
	for value, err := range api.ListDetail(context.Background(),
		shares.WithListDetailOptions(shares.ListOpts{Name: "n", Status: "available", IsPublic: manilaShareFalse(), Limit: 1, AllTenants: true, ProjectID: "p1"}),
		shares.WithListDetailQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"share-1", "share-2"}) || len(requests) != 3 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	first := requests[0]
	for key, want := range map[string]string{"name": "n", "status": "available", "is_public": "false", "limit": "1", "all_tenants": "true", "project_id": "p1", "vendor": "x"} {
		if first.Get(key) != want {
			t.Fatalf("first query %s=%q in %v", key, first.Get(key), first)
		}
	}
	if requests[1].Get("offset") != "1" || requests[2].Get("offset") != "2" || requests[1].Get("limit") != "1" {
		t.Fatalf("offset paging queries=%v", requests)
	}
}

func TestManilaShareListDetailWithoutLimitReadsOnePage(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/shares/detail", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.RawQuery != "" {
			t.Errorf("unexpected query %q", r.URL.RawQuery)
		}
		testcloud.JSON(w, 200, `{"shares":[{"id":"share-1"}]}`)
	})
	count := 0
	for _, err := range newManilaShares(cloud, "").ListDetail(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	if count != 1 || calls.Load() != 1 {
		t.Fatalf("count=%d calls=%d", count, calls.Load())
	}
}

func TestManilaShareListDetailExtensionQueryReplacesTypedFilter(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/shares/detail", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query()["name"]; !reflect.DeepEqual(got, []string{"override"}) {
			t.Errorf("name=%v", got)
		}
		testcloud.JSON(w, 200, `{"shares":[]}`)
	})
	for _, err := range newManilaShares(cloud, "").ListDetail(context.Background(),
		shares.WithListDetailOptions(shares.ListOpts{Name: "typed"}), shares.WithListDetailQuery("name", "override")) {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestManilaShareListDetailStopsOnHTTPFailure(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/shares/detail", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 500, `{}`)
	})
	var failed error
	for _, err := range newManilaShares(cloud, "").ListDetail(context.Background()) {
		failed = err
	}
	manilaShareCode(t, failed, 500)
	if calls.Load() != 1 {
		t.Fatalf("a failed list read was retried: %d", calls.Load())
	}
}

func TestManilaShareCommonLookupFindsExactNameMapsStatusAndLimitsItems(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/shares/data", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/shares/detail", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"shares":[{"id":"share-1","name":"data"},{"id":"share-2","name":"data-2"}]}`)
	})
	api := newManilaShares(cloud, "2.82")
	value, err := api.Find(context.Background(), resource.Name("data"))
	if err != nil || value == nil || value.ID != "share-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if len(queries) == 0 || queries[0].Get("name") != "data" {
		t.Fatalf("queries=%v", queries)
	}
	queries = nil
	if _, err := api.All(context.Background(), resource.WithStatus("available")); err != nil {
		t.Fatal(err)
	}
	if len(queries) != 1 || queries[0].Get("status") != "available" {
		t.Fatalf("status filter queries=%v", queries)
	}
	all, err := api.All(context.Background(), resource.WithMaxItems(1))
	if err != nil || len(all) != 1 || all[0].ID != "share-1" {
		t.Fatalf("all=%v error=%v", all, err)
	}
}

func TestManilaShareCancellationPreservesCauseAndNeverReplays(t *testing.T) {
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
	api := newManilaShares(cloud, "2.82")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(canceled, "share-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if _, err := api.Create(canceled, shares.CreateOpts{ShareProto: "NFS", Size: 1}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled create: %v", err)
	}
	if err := api.Extend(canceled, "share-1", shares.ExtendOpts{NewSize: 2}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled extend: %v", err)
	}
	if err := api.Delete(canceled, "share-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled delete: %v", err)
	}
	var listErr error
	for _, err := range api.ListDetail(canceled) {
		listErr = err
	}
	if !errors.Is(listErr, context.Canceled) {
		t.Fatalf("pre-canceled list: %v", listErr)
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if err := api.Extend(deadline, "share-1", shares.ExtendOpts{NewSize: 2}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("an uncertain mutation was replayed: %d calls", calls.Load())
	}
}

func TestManilaShareServerErrorOnMutationIsNotReplayed(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v2/shares", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 503, `{}`)
	})
	_, err := newManilaShares(cloud, "2.82").Create(context.Background(), shares.CreateOpts{ShareProto: "NFS", Size: 1})
	manilaShareCode(t, err, 503)
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}
