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

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/transfer/accept"
	transferrequest "github.com/JSYoo5B/go-openstacksdk/dns/v2/transfer/request"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const (
	designateTransferRequestBody = `{"id":"tr-1","zone_id":"zone-1","zone_name":"example.org.","project_id":"p1","target_project_id":"p2","key":"SECRETKEY","description":"","status":"ACTIVE","created_at":"2026-01-02T03:04:05.000000","links":{"self":"http://dns.example/v2/zones/tasks/transfer_requests/tr-1"},"vendor_extra":1}`
	designateTransferAcceptBody  = `{"id":"ta-1","status":"COMPLETE","project_id":"p2","zone_id":"zone-1","key":"SECRETKEY","zone_transfer_request_id":"tr-1","created_at":"2026-01-02T03:04:05.000000","links":{"self":"http://dns.example/v2/zones/tasks/transfer_accepts/ta-1"}}`
)

func newDesignateTransferRequests(t *testing.T, cloud *testcloud.Cloud) *transferrequest.API {
	t.Helper()
	return transferrequest.New(cloud.Client("dns", "/v2"))
}

func newDesignateTransferAccepts(t *testing.T, cloud *testcloud.Cloud) *accept.API {
	t.Helper()
	return accept.New(cloud.Client("dns", "/v2"))
}

func TestDesignateTransferRequestCreateSendsZoneRouteFlatBodyAndExtensions(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v2/zones/zone-1/tasks/transfer_requests", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := map[string]string{"target_project_id": `"p2"`, "description": `"handoff"`, "flag": `false`, "nothing": `null`}
		if got := designateStrings(designateFlatBody(t, r)); !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 201, designateTransferRequestBody)
	})
	cloud.Mux.HandleFunc("POST /v2/zones/zone-2/tasks/transfer_requests", func(w http.ResponseWriter, r *http.Request) {
		if got := designateStrings(designateFlatBody(t, r)); len(got) != 0 {
			t.Errorf("zero options must send an empty object: %v", got)
		}
		testcloud.JSON(w, 202, designateTransferRequestBody)
	})
	api := newDesignateTransferRequests(t, cloud)
	value, err := api.Create(context.Background(), "zone-1", transferrequest.CreateOpts{TargetProjectID: "p2", Description: "handoff"},
		transferrequest.WithCreateField("flag", false), transferrequest.WithCreateField("nothing", nil))
	if err != nil || value == nil || value.ID != "tr-1" || value.Key != "SECRETKEY" || value.Status != "ACTIVE" || value.ZoneName != "example.org." || value.CreatedAt.IsZero() || calls.Load() != 1 {
		t.Fatalf("value=%+v error=%v calls=%d", value, err, calls.Load())
	}
	if _, err := api.Create(context.Background(), "zone-2", transferrequest.CreateOpts{}); err != nil {
		t.Fatal(err)
	}
}

func TestDesignateTransferRequestUpdateOmitsEmptyValuesAndKeepsExtensions(t *testing.T) {
	cloud := testcloud.New(t)
	var bodies []map[string]string
	cloud.Mux.HandleFunc("PATCH /v2/zones/tasks/transfer_requests/tr-1", func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, designateStrings(designateFlatBody(t, r)))
		testcloud.JSON(w, 202, designateTransferRequestBody)
	})
	api := newDesignateTransferRequests(t, cloud)
	ctx := context.Background()
	if _, err := api.Update(ctx, "tr-1", transferrequest.UpdateOpts{Description: "new"}, transferrequest.WithUpdateField("flag", false)); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "tr-1", transferrequest.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	want := []map[string]string{{"description": `"new"`, "flag": `false`}, {}}
	if !reflect.DeepEqual(bodies, want) {
		t.Fatalf("bodies=%v", bodies)
	}
}

func TestDesignateTransferStatusCodes(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name, method, path string
		ok                 []int
		call               func(cloud *testcloud.Cloud) error
	}{
		{"request create", "POST", "/v2/zones/zone-1/tasks/transfer_requests", []int{201, 202}, func(c *testcloud.Cloud) error {
			_, err := newDesignateTransferRequests(t, c).Create(ctx, "zone-1", transferrequest.CreateOpts{})
			return err
		}},
		{"request update", "PATCH", "/v2/zones/tasks/transfer_requests/tr-1", []int{200, 202}, func(c *testcloud.Cloud) error {
			_, err := newDesignateTransferRequests(t, c).Update(ctx, "tr-1", transferrequest.UpdateOpts{})
			return err
		}},
		{"request delete", "DELETE", "/v2/zones/tasks/transfer_requests/tr-1", []int{204}, func(c *testcloud.Cloud) error {
			return newDesignateTransferRequests(t, c).Delete(ctx, "tr-1")
		}},
		{"accept create", "POST", "/v2/zones/tasks/transfer_accepts", []int{201, 202}, func(c *testcloud.Cloud) error {
			_, err := newDesignateTransferAccepts(t, c).Create(ctx, accept.CreateOpts{Key: "k", ZoneTransferRequestID: "tr-1"})
			return err
		}},
	} {
		for _, status := range []int{200, 201, 202, 204} {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc(tc.method+" "+tc.path, func(w http.ResponseWriter, r *http.Request) {
				if status == 204 {
					w.WriteHeader(204)
					return
				}
				testcloud.JSON(w, status, designateTransferRequestBody)
			})
			accepted := false
			for _, ok := range tc.ok {
				accepted = accepted || ok == status
			}
			err := tc.call(cloud)
			if accepted != (err == nil) {
				t.Fatalf("%s status %d error=%v", tc.name, status, err)
			}
			if !accepted {
				var unexpected gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &unexpected) || unexpected.Actual != status {
					t.Fatalf("%s status %d error=%v", tc.name, status, err)
				}
			}
		}
	}
}

func TestDesignateTransferAcceptCreateSendsFlatBodyAndRequiresCoreFields(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v2/zones/tasks/transfer_accepts", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := map[string]string{"key": `"SECRETKEY"`, "zone_transfer_request_id": `"tr-1"`, "flag": `false`}
		if got := designateStrings(designateFlatBody(t, r)); !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 201, designateTransferAcceptBody)
	})
	api := newDesignateTransferAccepts(t, cloud)
	value, err := api.Create(context.Background(), accept.CreateOpts{Key: "SECRETKEY", ZoneTransferRequestID: "tr-1"}, accept.WithCreateField("flag", false))
	if err != nil || value == nil || value.ID != "ta-1" || value.Status != "COMPLETE" || value.ZoneID != "zone-1" || value.ZoneTransferRequestID != "tr-1" || value.CreatedAt.IsZero() {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if _, err := api.Create(context.Background(), accept.CreateOpts{Key: "SECRETKEY"}); err == nil {
		t.Fatal("missing zone_transfer_request_id must fail before HTTP")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestDesignateTransferRejectsProtectedOrUnsupportedExtensionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	requests := newDesignateTransferRequests(t, cloud)
	accepts := newDesignateTransferAccepts(t, cloud)
	ctx := context.Background()
	invalid := func(name string, err error) {
		t.Helper()
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	_, err := requests.Create(ctx, "zone-1", transferrequest.CreateOpts{}, transferrequest.WithCreateField("description", "x"))
	invalid("protected request create field", err)
	_, err = requests.Create(ctx, "zone-1", transferrequest.CreateOpts{}, nil)
	invalid("nil request create option", err)
	_, err = requests.Update(ctx, "tr-1", transferrequest.UpdateOpts{}, transferrequest.WithUpdateField("target_project_id", "x"))
	invalid("protected request update field", err)
	_, err = requests.Update(ctx, "tr-1", transferrequest.UpdateOpts{}, nil)
	invalid("nil request update option", err)
	_, err = accepts.Create(ctx, accept.CreateOpts{Key: "k", ZoneTransferRequestID: "tr-1"}, accept.WithCreateField("key", "x"))
	invalid("protected accept create field", err)
	_, err = accepts.Create(ctx, accept.CreateOpts{Key: "k", ZoneTransferRequestID: "tr-1"}, nil)
	invalid("nil accept create option", err)
	for _, err := range requests.List(ctx, nil) {
		invalid("nil request list option", err)
	}
	for _, err := range accepts.List(ctx, nil) {
		invalid("nil accept list option", err)
	}
}

func TestDesignateTransferRequestReadsListsAndDeletes(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/zones/tasks/transfer_requests/tr-1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, designateTransferRequestBody)
	})
	cloud.Mux.HandleFunc("GET /v2/zones/tasks/transfer_requests/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("DELETE /v2/zones/tasks/transfer_requests/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "missing" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		w.WriteHeader(204)
	})
	cloud.Mux.HandleFunc("GET /v2/zones/tasks/transfer_requests", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		if r.URL.Query().Get("marker") == "tr-1" {
			testcloud.JSON(w, 200, `{"transfer_requests":[{"id":"tr-2","status":"ACTIVE"}],"links":{}}`)
			return
		}
		next := cloud.Server.URL + "/v2/zones/tasks/transfer_requests?status=ACTIVE&marker=tr-1"
		testcloud.JSON(w, 200, `{"transfer_requests":[{"id":"tr-1","status":"ACTIVE"}],"links":{"next":"`+next+`"}}`)
	})
	api := newDesignateTransferRequests(t, cloud)
	ctx := context.Background()
	value, err := api.Get(ctx, "tr-1")
	if err != nil || value.TargetProjectID != "p2" || value.ZoneID != "zone-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.Get(ctx, "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	if err := api.Delete(ctx, "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native delete keeps the 404 response error: %v", err)
	}
	if err := api.Delete(ctx, "tr-1"); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
	var ids []string
	for item, err := range api.List(ctx, transferrequest.WithListOptions(transferrequest.ListOpts{Status: "ACTIVE"}), transferrequest.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, item.ID)
	}
	if !reflect.DeepEqual(ids, []string{"tr-1", "tr-2"}) || len(queries) != 2 || queries[0].Get("status") != "ACTIVE" || queries[0].Get("vendor") != "x" || queries[1].Get("marker") != "tr-1" {
		t.Fatalf("ids=%v queries=%v", ids, queries)
	}
}

func TestDesignateTransferAcceptReadsAndLists(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/zones/tasks/transfer_accepts/ta-1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, designateTransferAcceptBody)
	})
	cloud.Mux.HandleFunc("GET /v2/zones/tasks/transfer_accepts/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/zones/tasks/transfer_accepts", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		if r.URL.Query().Get("marker") == "ta-1" {
			testcloud.JSON(w, 200, `{"transfer_accepts":[{"id":"ta-2"}],"links":{}}`)
			return
		}
		next := cloud.Server.URL + "/v2/zones/tasks/transfer_accepts?status=COMPLETE&marker=ta-1"
		testcloud.JSON(w, 200, `{"transfer_accepts":[{"id":"ta-1","status":"COMPLETE"}],"links":{"next":"`+next+`"}}`)
	})
	api := newDesignateTransferAccepts(t, cloud)
	ctx := context.Background()
	value, err := api.Get(ctx, "ta-1")
	if err != nil || value.ProjectID != "p2" || value.Key != "SECRETKEY" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.Get(ctx, "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	var ids []string
	for item, err := range api.List(ctx, accept.WithListOptions(accept.ListOpts{Status: "COMPLETE"}), accept.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, item.ID)
	}
	if !reflect.DeepEqual(ids, []string{"ta-1", "ta-2"}) || len(queries) != 2 || queries[0].Get("status") != "COMPLETE" || queries[0].Get("vendor") != "x" || queries[1].Get("marker") != "ta-1" {
		t.Fatalf("ids=%v queries=%v", ids, queries)
	}
}

func TestDesignateTransferCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	requests := newDesignateTransferRequests(t, cloud)
	accepts := newDesignateTransferAccepts(t, cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	acceptOpts := accept.CreateOpts{Key: "k", ZoneTransferRequestID: "tr-1"}
	if _, err := requests.Create(canceled, "zone-1", transferrequest.CreateOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled request create: %v", err)
	}
	if _, err := requests.Get(canceled, "tr-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled request get: %v", err)
	}
	if err := requests.Delete(canceled, "tr-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled request delete: %v", err)
	}
	if _, err := accepts.Create(canceled, acceptOpts); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled accept create: %v", err)
	}
	if _, err := accepts.Get(canceled, "ta-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled accept get: %v", err)
	}
	for _, err := range accepts.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled accept list: %v", err)
		}
	}
	for _, err := range requests.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled request list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := accepts.Create(deadline, acceptOpts); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight accept deadline: %v", err)
	}
	if _, err := requests.Update(deadline, "tr-1", transferrequest.UpdateOpts{}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight request deadline: %v", err)
	}
}
