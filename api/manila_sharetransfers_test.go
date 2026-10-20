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
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharetransfers"
	"github.com/gophercloud/gophercloud/v2"
)

const manilaTransferVersion = "2.77"

const manilaTransferJSON = `{"id":"tr-1","name":"handoff","accepted":false,"auth_key":"synthetic-key","source_project_id":"proj-a","destination_project_id":null,"resource_id":"share-1","resource_type":"share","created_at":"2026-01-02T03:04:05.000000","expires_at":"2026-01-02T04:04:05.000000","links":[{"rel":"self","href":"http://example.invalid/tr-1"}]}`

func newManilaTransfers(t *testing.T, cloud *testcloud.Cloud) *sharetransfers.API {
	t.Helper()
	client := cloud.Client("shared-file-system", "/v2")
	client.Microversion = manilaTransferVersion
	return sharetransfers.New(client)
}

func manilaTransferRequest(t *testing.T, r *http.Request, wantBody string) {
	t.Helper()
	if got := r.Header.Get("X-Openstack-Manila-Api-Version"); got != manilaTransferVersion {
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

func TestManilaTransferCreateBodyExtensionsAndRequires202(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{202, true}, {200, false}, {201, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /v2/share-transfers", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			manilaTransferRequest(t, r, `{"transfer":{"share_id":"share-1","name":"handoff","vendor":false}}`)
			testcloud.JSON(w, tc.status, `{"transfer":`+manilaTransferJSON+`}`)
		})
		value, err := newManilaTransfers(t, cloud).Create(context.Background(),
			sharetransfers.CreateOpts{ShareID: "share-1", Name: "handoff"}, sharetransfers.WithCreateField("vendor", false))
		if calls.Load() != 1 || tc.success != (err == nil) {
			t.Fatalf("status %d calls=%d value=%+v error=%v", tc.status, calls.Load(), value, err)
		}
		if tc.success {
			if value.ID != "tr-1" || value.Accepted || value.AuthKey != "synthetic-key" || value.SourceProjectID != "proj-a" || value.DestinationProjectID != "" || value.ResourceType != "share" || value.CreatedAt.IsZero() || value.ExpiresAt.IsZero() {
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

func TestManilaTransferCreateOmitsEmptyNameAndRequiresShareID(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v2/share-transfers", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		manilaTransferRequest(t, r, `{"transfer":{"share_id":"share-1"}}`)
		testcloud.JSON(w, 202, `{"transfer":`+manilaTransferJSON+`}`)
	})
	api := newManilaTransfers(t, cloud)
	if _, err := api.Create(context.Background(), sharetransfers.CreateOpts{ShareID: "share-1", Name: ""}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Create(context.Background(), sharetransfers.CreateOpts{Name: "x"}); err == nil {
		t.Fatal("missing share_id was accepted")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestManilaTransferRejectsCoreFieldCollisionsAndNilOptionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newManilaTransfers(t, cloud)
	ctx := context.Background()
	checks := map[string]error{}
	_, checks["create core"] = api.Create(ctx, sharetransfers.CreateOpts{ShareID: "s"}, sharetransfers.WithCreateField("share_id", "other"))
	_, checks["create omitted core"] = api.Create(ctx, sharetransfers.CreateOpts{ShareID: "s"}, sharetransfers.WithCreateField("name", "n"))
	_, checks["create nil"] = api.Create(ctx, sharetransfers.CreateOpts{ShareID: "s"}, nil)
	checks["accept core"] = api.Accept(ctx, "tr-1", sharetransfers.AcceptOpts{AuthKey: "k"}, sharetransfers.WithAcceptField("auth_key", "other"))
	checks["accept omitted false core"] = api.Accept(ctx, "tr-1", sharetransfers.AcceptOpts{AuthKey: "k"}, sharetransfers.WithAcceptField("clear_access_rules", false))
	checks["accept nil"] = api.Accept(ctx, "tr-1", sharetransfers.AcceptOpts{AuthKey: "k"}, nil)
	for name, err := range checks {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, err := range api.List(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("nil list option: %v", err)
		}
	}
	for _, err := range api.ListDetail(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("nil list detail option: %v", err)
		}
	}
	// Status filtering is unsupported by this binding and must fail locally.
	if _, err := api.All(ctx, resource.WithStatus("available")); err == nil {
		t.Error("status filter was accepted")
	}
}

func TestManilaTransferAcceptSendsNamedBodyOmitsFalseAndRequires202(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts sharetransfers.AcceptOpts
		want string
	}{
		{"clear rules", sharetransfers.AcceptOpts{AuthKey: "k", ClearAccessRules: true}, `{"accept":{"auth_key":"k","clear_access_rules":true,"vendor":null}}`},
		{"false is omitted", sharetransfers.AcceptOpts{AuthKey: "k", ClearAccessRules: false}, `{"accept":{"auth_key":"k","vendor":null}}`},
	} {
		for _, status := range []int{202, 200, 204} {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("POST /v2/share-transfers/tr-1/accept", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				manilaTransferRequest(t, r, tc.want)
				w.WriteHeader(status)
			})
			err := newManilaTransfers(t, cloud).Accept(context.Background(), "tr-1", tc.opts, sharetransfers.WithAcceptField("vendor", nil))
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
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
	})
	if err := newManilaTransfers(t, cloud).Accept(context.Background(), "tr-1", sharetransfers.AcceptOpts{}); err == nil {
		t.Fatal("empty auth_key accepted")
	}
}

func TestManilaTransferDeleteAcceptsSuccessCodesAndKeepsNative404(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, true}, {204, true}, {201, false}, {404, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("DELETE /v2/share-transfers/tr-1", func(w http.ResponseWriter, r *http.Request) {
			manilaTransferRequest(t, r, "")
			w.WriteHeader(tc.status)
		})
		err := newManilaTransfers(t, cloud).Delete(context.Background(), "tr-1")
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
	cloud.Mux.HandleFunc("DELETE /v2/share-transfers/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := newManilaTransfers(t, cloud)
	if err := api.Remove(context.Background(), resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
	if err := api.Remove(context.Background(), resource.ID("missing"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("strict Remove: %v", err)
	}
}

func TestManilaTransferGetDecodesModelAndKeepsNative404(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/share-transfers/tr-1", func(w http.ResponseWriter, r *http.Request) {
		manilaTransferRequest(t, r, "")
		testcloud.JSON(w, 200, `{"transfer":`+manilaTransferJSON+`}`)
	})
	cloud.Mux.HandleFunc("GET /v2/share-transfers/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := newManilaTransfers(t, cloud)
	value, err := api.Get(context.Background(), "tr-1")
	if err != nil || value.Name != "handoff" || value.ResourceID != "share-1" || len(value.Links) != 1 {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.Get(context.Background(), "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	if got, err := api.Find(context.Background(), resource.ID("missing"), resource.WithIgnoreMissing()); err != nil || got != nil {
		t.Fatalf("Find ignore-missing got=%v err=%v", got, err)
	}
}

func TestManilaTransferListAndListDetailFiltersFollowOffsetPaging(t *testing.T) {
	opts := sharetransfers.ListOpts{AllTenants: true, Name: "a", NamePattern: "n", SortKey: "created_at", SortDir: "asc", Limit: 1, Marker: "m-1"}
	for _, tc := range []struct {
		name string
		path string
		call func(*sharetransfers.API) func(func(*sharetransfers.Transfer, error) bool)
	}{
		{"list", "/v2/share-transfers", func(a *sharetransfers.API) func(func(*sharetransfers.Transfer, error) bool) {
			return a.List(context.Background(), sharetransfers.WithListOptions(opts), sharetransfers.WithListQuery("vendor", "x"))
		}},
		{"detail", "/v2/share-transfers/detail", func(a *sharetransfers.API) func(func(*sharetransfers.Transfer, error) bool) {
			return a.ListDetail(context.Background(), sharetransfers.WithListDetailOptions(opts), sharetransfers.WithListDetailQuery("vendor", "x"))
		}},
	} {
		cloud := testcloud.New(t)
		var requests []url.Values
		cloud.Mux.HandleFunc("GET "+tc.path, func(w http.ResponseWriter, r *http.Request) {
			manilaTransferRequest(t, r, "")
			requests = append(requests, r.URL.Query())
			switch r.URL.Query().Get("offset") {
			case "":
				testcloud.JSON(w, 200, `{"transfers":[{"id":"tr-1","name":"a"}]}`)
			case "1":
				testcloud.JSON(w, 200, `{"transfers":[{"id":"tr-2","name":"b"}]}`)
			default:
				testcloud.JSON(w, 200, `{"transfers":[]}`)
			}
		})
		var ids []string
		for value, err := range tc.call(newManilaTransfers(t, cloud)) {
			if err != nil {
				t.Fatal(tc.name, err)
			}
			ids = append(ids, value.ID)
		}
		if !reflect.DeepEqual(ids, []string{"tr-1", "tr-2"}) || len(requests) != 3 {
			t.Fatalf("%s ids=%v requests=%v", tc.name, ids, requests)
		}
		want := url.Values{"all_tenants": {"true"}, "name": {"a"}, "name~": {"n"}, "sort_key": {"created_at"}, "sort_dir": {"asc"}, "limit": {"1"}, "marker": {"m-1"}, "vendor": {"x"}}
		if !reflect.DeepEqual(requests[0], want) {
			t.Fatalf("%s first query=%v", tc.name, requests[0])
		}
		if requests[1].Get("offset") != "1" || requests[2].Get("offset") != "2" {
			t.Fatalf("%s paging=%v", tc.name, requests)
		}
	}
	cloud := testcloud.New(t)
	var query url.Values
	cloud.Mux.HandleFunc("GET /v2/share-transfers", func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		testcloud.JSON(w, 200, `{"transfers":[]}`)
	})
	for range newManilaTransfers(t, cloud).List(context.Background(), sharetransfers.WithListOptions(sharetransfers.ListOpts{AllTenants: false})) {
	}
	if len(query) != 0 {
		t.Fatalf("AllTenants=false is a zero value and must be omitted: %v", query)
	}
}

func TestManilaTransferCommonLookupFindsExactNameThroughDetailList(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/share-transfers/detail", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"transfers":[{"id":"tr-1","name":"web"},{"id":"tr-2","name":"web-2"},{"id":"tr-3","name":"dup"},{"id":"tr-4","name":"dup"}]}`)
	})
	api := newManilaTransfers(t, cloud)
	value, err := api.Find(context.Background(), resource.Name("web"))
	if err != nil || value == nil || value.ID != "tr-1" || len(queries) != 1 || queries[0].Get("name") != "web" {
		t.Fatalf("value=%+v error=%v queries=%v", value, err, queries)
	}
	if _, err := api.Find(context.Background(), resource.Name("absent")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing name: %v", err)
	}
	var ambiguous *resource.AmbiguousError
	if _, err := api.Find(context.Background(), resource.Name("dup")); !errors.As(err, &ambiguous) {
		t.Fatalf("duplicate name: %v", err)
	}
}

func TestManilaTransferCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	api := newManilaTransfers(t, cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(canceled, "tr-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("get: %v", err)
	}
	if _, err := api.Create(canceled, sharetransfers.CreateOpts{ShareID: "s"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("create: %v", err)
	}
	if err := api.Accept(canceled, "tr-1", sharetransfers.AcceptOpts{AuthKey: "k"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("accept: %v", err)
	}
	if err := api.Delete(canceled, "tr-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("delete: %v", err)
	}
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("list: %v", err)
		}
		break
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if err := api.Accept(deadline, "tr-1", sharetransfers.AcceptOpts{AuthKey: "k"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
