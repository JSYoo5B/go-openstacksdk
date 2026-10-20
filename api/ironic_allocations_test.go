package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/allocations"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const ironicAPIVersionHeader = "X-OpenStack-Ironic-API-Version"

// ironicClient returns a baremetal-typed client. A non-empty version makes
// Gophercloud send the Ironic microversion header.
func ironicClient(cloud *testcloud.Cloud, service, version string) *gophercloud.ServiceClient {
	client := cloud.Client(service, "/v1")
	client.Microversion = version
	return client
}

// ironicForbidHTTP fails the test if any request reaches the fixture.
func ironicForbidHTTP(t *testing.T, cloud *testcloud.Cloud) {
	t.Helper()
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
}

// ironicRequireStatus asserts that err keeps the native unexpected-status error.
func ironicRequireStatus(t *testing.T, err error, status int) {
	t.Helper()
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &unexpected) || unexpected.Actual != status {
		t.Fatalf("want native unexpected status %d, got %v", status, err)
	}
}

// ironicBlockingCloud counts every request and holds it until the client gives up.
func ironicBlockingCloud(t *testing.T) (*testcloud.Cloud, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	return cloud, &calls
}

const ironicAllocationBody = `{"uuid":"alloc-1","name":"web","state":"allocating","resource_class":"baremetal","node_uuid":null,"last_error":null,"candidate_nodes":["n1"],"traits":["CUSTOM_A"],"extra":{"k":"v"},"created_at":"2026-01-02T03:04:05+00:00","updated_at":null,"links":[]}`

func newIronicAllocations(t *testing.T, cloud *testcloud.Cloud, version string) *allocations.API {
	t.Helper()
	return allocations.New(ironicClient(cloud, "baremetal", version))
}

func TestIronicAllocationCreateSendsBodyAndReturnsAcknowledgementState(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v1/allocations", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if got := r.Header.Get(ironicAPIVersionHeader); got != "1.52" {
			t.Errorf("microversion header=%q", got)
		}
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("body: %v", err)
		}
		got := map[string]string{}
		for key, value := range body {
			got[key] = string(value)
		}
		want := map[string]string{
			"resource_class": `"baremetal"`, "name": `"web"`, "uuid": `"alloc-1"`,
			"candidate_nodes": `["n1"]`, "traits": `["CUSTOM_A"]`, "extra": `{"k":"v"}`,
			"vendor": `{"enabled":false}`,
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 202, ironicAllocationBody)
	})
	value, err := newIronicAllocations(t, cloud, "1.52").Create(context.Background(), allocations.CreateOpts{
		ResourceClass: "baremetal", Name: "web", UUID: "alloc-1", CandidateNodes: []string{"n1"},
		Traits: []string{"CUSTOM_A"}, Extra: map[string]string{"k": "v"},
	}, allocations.WithCreateField("vendor", map[string]bool{"enabled": false}))
	if err != nil || value == nil || value.UUID != "alloc-1" || calls.Load() != 1 {
		t.Fatalf("value=%+v error=%v calls=%d", value, err, calls.Load())
	}
	// Acknowledgement is not completion: the node is not assigned yet.
	if value.State != "allocating" || value.NodeUUID != "" || value.LastError != "" || !value.UpdatedAt.IsZero() || value.CreatedAt.IsZero() {
		t.Fatalf("acknowledged allocation=%+v", value)
	}
}

func TestIronicAllocationCreateOmitsEmptyOptionalFieldsAndAcceptsOnly201Or202(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{201, true}, {202, true}, {200, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("POST /v1/allocations", func(w http.ResponseWriter, r *http.Request) {
			var body map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != 1 || string(body["resource_class"]) != `"baremetal"` {
				t.Errorf("empty optional values must be omitted: %v err=%v", body, err)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, ironicAllocationBody)
		})
		api := newIronicAllocations(t, cloud, "")
		value, err := api.Create(context.Background(), allocations.CreateOpts{ResourceClass: "baremetal", Name: "", CandidateNodes: []string{}, Extra: map[string]string{}})
		if tc.success != (err == nil) || (tc.success && value == nil) {
			t.Fatalf("status %d value=%+v error=%v", tc.status, value, err)
		}
		if !tc.success {
			ironicRequireStatus(t, err, tc.status)
		}
	}
}

func TestIronicAllocationRejectsInvalidInputBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	ironicForbidHTTP(t, cloud)
	api := newIronicAllocations(t, cloud, "")
	ctx := context.Background()
	if _, err := api.Create(ctx, allocations.CreateOpts{ResourceClass: "baremetal"}, allocations.WithCreateField("name", "shadow")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("core field collision: %v", err)
	}
	if _, err := api.Create(ctx, allocations.CreateOpts{ResourceClass: "baremetal"}, allocations.WithCreateField("resource_class", "other")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("required field collision: %v", err)
	}
	if _, err := api.Create(ctx, allocations.CreateOpts{ResourceClass: "baremetal"}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	if _, err := api.Create(ctx, allocations.CreateOpts{}); err == nil {
		t.Fatal("missing required resource_class must fail before HTTP")
	}
	for _, err := range api.List(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list option: %v", err)
		}
	}
	for _, err := range api.List(ctx, allocations.WithListQuery(" ", "x")) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("empty query key: %v", err)
		}
	}
}

func TestIronicAllocationGetAndDeleteStatusPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	var deletes []string
	cloud.Mux.HandleFunc("GET /v1/allocations/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(ironicAPIVersionHeader) != "" {
			t.Errorf("no microversion header expected: %v", r.Header)
		}
		switch r.PathValue("id") {
		case "alloc-1":
			testcloud.JSON(w, 200, ironicAllocationBody)
		case "accepted":
			testcloud.JSON(w, 202, ironicAllocationBody)
		default:
			testcloud.JSON(w, 404, `{"error_message":"gone"}`)
		}
	})
	cloud.Mux.HandleFunc("DELETE /v1/allocations/{id}", func(w http.ResponseWriter, r *http.Request) {
		deletes = append(deletes, r.PathValue("id"))
		switch r.PathValue("id") {
		case "a202":
			w.WriteHeader(202)
		case "a204":
			w.WriteHeader(204)
		case "a200":
			w.WriteHeader(200)
		default:
			testcloud.JSON(w, 404, `{"error_message":"gone"}`)
		}
	})
	api := newIronicAllocations(t, cloud, "")
	ctx := context.Background()
	value, err := api.Get(ctx, "alloc-1")
	if err != nil || value.Name != "web" || value.ResourceClass != "baremetal" || !reflect.DeepEqual(value.Extra, map[string]string{"k": "v"}) {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	_, err = api.Get(ctx, "accepted")
	ironicRequireStatus(t, err, 202)
	_, err = api.Get(ctx, "missing")
	ironicRequireStatus(t, err, 404)
	for _, id := range []string{"a202", "a204"} {
		if err := api.Delete(ctx, id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	ironicRequireStatus(t, api.Delete(ctx, "a200"), 200)
	ironicRequireStatus(t, api.Delete(ctx, "missing"), 404)
	if !reflect.DeepEqual(deletes, []string{"a202", "a204", "a200", "missing"}) {
		t.Fatalf("deletes=%v", deletes)
	}
}

func TestIronicAllocationRemoveIgnoresMissingByDefaultAndCanRequireIt(t *testing.T) {
	cloud := testcloud.New(t)
	var deleteCalls atomic.Int32
	cloud.Mux.HandleFunc("DELETE /v1/allocations/missing", func(w http.ResponseWriter, r *http.Request) {
		deleteCalls.Add(1)
		testcloud.JSON(w, 404, `{}`)
	})
	api := newIronicAllocations(t, cloud, "")
	if err := api.Remove(context.Background(), resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores a missing allocation by default: %v", err)
	}
	if err := api.Remove(context.Background(), resource.ID("missing"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("WithMissingError: %v", err)
	}
	if deleteCalls.Load() != 2 {
		t.Fatalf("an ID reference deletes directly without a prior lookup: %d", deleteCalls.Load())
	}
}

func TestIronicAllocationListFiltersFollowLinksAndExtensionQuery(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v1/allocations", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "alloc-1" {
			testcloud.JSON(w, 200, `{"allocations":[{"uuid":"alloc-2","name":"two"}]}`)
			return
		}
		next := cloud.Server.URL + "/v1/allocations?marker=alloc-1&limit=1"
		testcloud.JSON(w, 200, `{"allocations":[{"uuid":"alloc-1","name":"one"}],"allocations_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	api := newIronicAllocations(t, cloud, "")
	var ids []string
	for value, err := range api.List(context.Background(),
		allocations.WithListOptions(allocations.ListOpts{Node: "n1", ResourceClass: "baremetal", State: allocations.AllocationState("allocating"), Fields: []string{"uuid", "name"}, Limit: 1, SortKey: "name", SortDir: "desc"}),
		allocations.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.UUID)
	}
	if !reflect.DeepEqual(ids, []string{"alloc-1", "alloc-2"}) || len(requests) != 2 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	want := url.Values{"node": {"n1"}, "resource_class": {"baremetal"}, "state": {"allocating"}, "fields": {"uuid,name"}, "limit": {"1"}, "sort_key": {"name"}, "sort_dir": {"desc"}, "vendor": {"x"}}
	if !reflect.DeepEqual(requests[0], want) {
		t.Fatalf("first query=%v want %v", requests[0], want)
	}
	if requests[1].Get("marker") != "alloc-1" {
		t.Fatalf("second query=%v", requests[1])
	}
}

func TestIronicAllocationCollectionAllHonorsSinglePageAndMaxItems(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/allocations", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		next := cloud.Server.URL + "/v1/allocations?marker=alloc-2"
		testcloud.JSON(w, 200, `{"allocations":[{"uuid":"alloc-1"},{"uuid":"alloc-2"}],"allocations_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	api := newIronicAllocations(t, cloud, "")
	values, err := api.All(context.Background(), resource.WithPaginated(false), resource.WithMaxItems(1))
	if err != nil || len(values) != 1 || values[0].UUID != "alloc-1" || calls.Load() != 1 {
		t.Fatalf("values=%d error=%v calls=%d", len(values), err, calls.Load())
	}
}

func TestIronicAllocationCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud, calls := ironicBlockingCloud(t)
	api := newIronicAllocations(t, cloud, "")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(canceled, "alloc-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if _, err := api.Create(canceled, allocations.CreateOpts{ResourceClass: "baremetal"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled create: %v", err)
	}
	if err := api.Delete(canceled, "alloc-1"); !errors.Is(err, context.Canceled) {
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
	if _, err := api.Create(deadline, allocations.CreateOpts{ResourceClass: "baremetal"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
