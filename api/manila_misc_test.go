package api_test

import (
	"context"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/apiversions"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/availabilityzones"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/messages"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/schedulerstats"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/services"
)

func TestManilaMiscServicesListUsesQueryExtensionBecauseTypedFiltersAreNotEncoded(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/services", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"services":[{"binary":"manila-share","host":"h1@lvm","id":1,"state":"up","status":"enabled","zone":"nova","updated_at":"2026-01-02T03:04:05.000000"},{"binary":"manila-scheduler","host":"h2","id":2,"state":"down","status":"disabled","zone":"nova","updated_at":null}],"services_links":[{"rel":"next","href":"ignored"}]}`)
	})
	api := services.New(newManilaSupportClient(cloud))
	var got []services.Service
	// Gophercloud's ListOpts fields carry json tags only, so the typed filters
	// are not serialized; WithListQuery is the working filter path.
	for value, err := range api.List(context.Background(),
		services.WithListOptions(services.ListOpts{Host: "typed-host", Binary: "typed-binary", State: "up"}),
		services.WithListQuery("host", "h1@lvm")) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, *value)
	}
	if len(got) != 2 || got[0].ID != 1 || got[0].Binary != "manila-share" || got[0].UpdatedAt.IsZero() || got[1].State != "down" || !got[1].UpdatedAt.IsZero() {
		t.Fatalf("services=%+v", got)
	}
	if len(queries) != 1 || !reflect.DeepEqual(queries[0], url.Values{"host": {"h1@lvm"}}) {
		t.Fatalf("typed filters must be absent and the extension query present: %v", queries)
	}
}

func TestManilaMiscServicesListRejectsNilOptionAndFailureStatuses(t *testing.T) {
	cloud := testcloud.New(t)
	status := 403
	cloud.Mux.HandleFunc("GET /v2/services", func(w http.ResponseWriter, r *http.Request) {
		if status == 204 {
			w.WriteHeader(204)
			return
		}
		testcloud.JSON(w, status, `{}`)
	})
	api := services.New(newManilaSupportClient(cloud))
	for _, err := range api.List(context.Background(), nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil option: %v", err)
		}
	}
	for _, err := range api.List(context.Background()) {
		manilaSupportUnexpected(t, err, 403)
	}
	status = 204
	for value, err := range api.List(context.Background()) {
		t.Fatalf("204 list yielded value=%+v error=%v", value, err)
	}
}

func TestManilaMiscSchedulerStatsListAndDetailRoutesParseCapacity(t *testing.T) {
	cloud := testcloud.New(t)
	var paths []string
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/scheduler-stats/pools", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		paths, queries = append(paths, r.URL.Path), append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"pools":[{"name":"h1@lvm#pool","host":"h1","backend":"lvm","pool":"pool"}]}`)
	})
	cloud.Mux.HandleFunc("GET /v2/scheduler-stats/pools/detail", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		paths, queries = append(paths, r.URL.Path), append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"pools":[{"name":"h1@lvm#pool","host":"h1","backend":"lvm","pool":"pool","capabilities":{"qos":false,"snapshot_support":false,"driver_handles_share_servers":false,"total_capacity_gb":"infinite","free_capacity_gb":12.5,"allocated_capacity_gb":null,"reserved_percentage":5,"vendor_name":"Open Source"}}]}`)
	})
	api := schedulerstats.New(newManilaSupportClient(cloud))
	ctx := context.Background()
	for value, err := range api.List(ctx, schedulerstats.WithListOptions(schedulerstats.ListOpts{PoolName: "typed"}), schedulerstats.WithListQuery("pool_name", "pool")) {
		if err != nil || value.Name != "h1@lvm#pool" || value.Backend != "lvm" || value.Capabilities.VendorName != "" {
			t.Fatalf("value=%+v error=%v", value, err)
		}
	}
	for value, err := range api.ListDetail(ctx, schedulerstats.WithListDetailQuery("backend_name", "lvm")) {
		c := value.Capabilities
		if err != nil || c.Qos || c.SnapshotSupport || !math.IsInf(c.TotalCapacityGB, 1) || c.FreeCapacityGB != 12.5 || c.AllocatedCapacityGB != 0 || c.ReservedPercentage != 5 || c.VendorName != "Open Source" {
			t.Fatalf("value=%+v error=%v", value, err)
		}
	}
	if !reflect.DeepEqual(paths, []string{"/v2/scheduler-stats/pools", "/v2/scheduler-stats/pools/detail"}) {
		t.Fatalf("paths=%v", paths)
	}
	// Typed ListOpts filters are json-tagged only upstream and are not encoded.
	if !reflect.DeepEqual(queries, []url.Values{{"pool_name": {"pool"}}, {"backend_name": {"lvm"}}}) {
		t.Fatalf("queries=%v", queries)
	}
}

func TestManilaMiscSchedulerStatsRejectsNilOptionsAndNoContentIsEmpty(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	api := schedulerstats.New(newManilaSupportClient(cloud))
	for _, err := range api.List(context.Background(), nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list option: %v", err)
		}
	}
	for _, err := range api.ListDetail(context.Background(), nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil detail option: %v", err)
		}
	}
	for value, err := range api.List(context.Background()) {
		t.Fatalf("204 yielded value=%+v error=%v", value, err)
	}
	for value, err := range api.ListDetail(context.Background()) {
		t.Fatalf("204 detail yielded value=%+v error=%v", value, err)
	}
}

func TestManilaMiscAvailabilityZonesListParsesTimestampsAndStaysSinglePage(t *testing.T) {
	cloud := testcloud.New(t)
	var requests atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/os-availability-zone", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		requests.Add(1)
		testcloud.JSON(w, 200, `{"availability_zones":[{"id":"az-1","name":"nova","created_at":"2026-01-02T03:04:05.000000","updated_at":null},{"id":"az-2","name":"zone-b"}],"availability_zones_links":[{"rel":"next","href":"ignored"}]}`)
	})
	var names []string
	var first availabilityzones.AvailabilityZone
	for value, err := range availabilityzones.New(newManilaSupportClient(cloud)).List(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		if len(names) == 0 {
			first = *value
		}
		names = append(names, value.Name)
	}
	if !reflect.DeepEqual(names, []string{"nova", "zone-b"}) || requests.Load() != 1 || first.ID != "az-1" || first.CreatedAt.IsZero() || !first.UpdatedAt.IsZero() {
		t.Fatalf("names=%v requests=%d first=%+v", names, requests.Load(), first)
	}
}

func TestManilaMiscAvailabilityZonesListReportsFailureAndHonorsCancellation(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/os-availability-zone", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 500, `{}`)
	})
	api := availabilityzones.New(newManilaSupportClient(cloud))
	for _, err := range api.List(context.Background()) {
		manilaSupportUnexpected(t, err, 500)
	}
	calls.Store(0)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled list reached HTTP: %d", calls.Load())
	}
}

const manilaMessageBody = `{"id":"m-1","project_id":"p1","action_id":"002","detail_id":"003","message_level":"ERROR","request_id":"req-1","resource_id":"r-1","resource_type":"SHARE","user_message":"No valid host","created_at":"2026-01-02T03:04:05.000000","expires_at":"2026-01-03T03:04:05.000000"}`

func TestManilaMiscMessagesGetListDeleteAndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/messages/{id}", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		if r.PathValue("id") == "missing" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		testcloud.JSON(w, 200, `{"message":`+manilaMessageBody+`}`)
	})
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/messages", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"messages":[`+manilaMessageBody+`,{"id":"m-2"}],"messages_links":[{"rel":"next","href":"ignored"}]}`)
	})
	deleted := map[string]int{"m-1": 202, "m-2": 204, "m-3": 200, "missing": 404}
	cloud.Mux.HandleFunc("DELETE /v2/messages/{id}", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		w.WriteHeader(deleted[r.PathValue("id")])
	})
	api := messages.New(newManilaSupportClient(cloud))
	ctx := context.Background()
	value, err := api.Get(ctx, "m-1")
	if err != nil || value.MessageLevel != "ERROR" || value.ResourceType != "SHARE" || value.UserMessage != "No valid host" || value.CreatedAt.IsZero() || value.ExpiresAt.IsZero() {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	_, err = api.Get(ctx, "missing")
	manilaSupportUnexpected(t, err, 404)
	var ids []string
	for value, err := range api.List(ctx, messages.WithListOptions(messages.ListOpts{ResourceType: "SHARE", SortKey: "created_at", SortDir: "desc", Limit: 5}), messages.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	want := url.Values{"resource_type": {"SHARE"}, "sort_key": {"created_at"}, "sort_dir": {"desc"}, "limit": {"5"}, "vendor": {"x"}}
	if !reflect.DeepEqual(ids, []string{"m-1", "m-2"}) || len(queries) != 1 || !reflect.DeepEqual(queries[0], want) {
		t.Fatalf("ids=%v queries=%v", ids, queries)
	}
	for _, id := range []string{"m-1", "m-2"} {
		if err := api.Delete(ctx, id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	manilaSupportUnexpected(t, api.Delete(ctx, "m-3"), 200)
	manilaSupportUnexpected(t, api.Delete(ctx, "missing"), 404)
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores a missing message by default: %v", err)
	}
}

func TestManilaMiscMessagesRejectNilOptionAndHonorCancellation(t *testing.T) {
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
	api := messages.New(newManilaSupportClient(cloud))
	for _, err := range api.List(context.Background(), nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil option: %v", err)
		}
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(canceled, "m-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if err := api.Delete(canceled, "m-1"); !errors.Is(err, context.Canceled) {
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
	if err := api.Delete(deadline, "m-1"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}

func TestManilaMiscAPIVersionsListAndGetUseUnversionedEndpoint(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		testcloud.JSON(w, 300, `{"versions":[{"id":"v2.0","status":"CURRENT","version":"2.80","min_version":"2.0","updated":"2015-08-27T11:33:21Z"},{"id":"v1.0","status":"DEPRECATED","version":"","min_version":"","updated":"2014-01-01T00:00:00Z"}]}`)
	})
	cloud.Mux.HandleFunc("GET /v2.0/", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		testcloud.JSON(w, 200, `{"versions":[{"id":"v2.0","status":"CURRENT","version":"2.80","min_version":"2.0","updated":"2015-08-27T11:33:21Z"}]}`)
	})
	// The service endpoint carries a version path that upstream strips.
	api := apiversions.New(cloud.Client("shared-file-system", "/v2/"))
	api.RawClient().Microversion = manilaSupportVersion
	ctx := context.Background()
	var ids []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatalf("300 Multiple Choices is an accepted discovery status: %v", err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"v2.0", "v1.0"}) {
		t.Fatalf("ids=%v", ids)
	}
	value, err := api.Get(ctx, "v2.0")
	if err != nil || value.Version != "2.80" || value.MinVersion != "2.0" || value.Status != "CURRENT" || value.Updated.IsZero() {
		t.Fatalf("value=%+v error=%v", value, err)
	}
}

func TestManilaMiscAPIVersionsGetReportsMissingAndAmbiguousDocuments(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v9/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"versions":[]}`) })
	cloud.Mux.HandleFunc("GET /v8/", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"versions":[{"id":"a"},{"id":"b"}]}`)
	})
	cloud.Mux.HandleFunc("GET /v7/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := apiversions.New(cloud.Client("shared-file-system", "/v2/"))
	ctx := context.Background()
	_, err := api.Get(ctx, "v9")
	var notFound apiversions.ErrVersionNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("empty document: %v", err)
	}
	_, err = api.Get(ctx, "v8")
	var multiple apiversions.ErrMultipleVersionsFound
	if !errors.As(err, &multiple) || multiple.Count != 2 {
		t.Fatalf("ambiguous document: %v", err)
	}
	_, err = api.Get(ctx, "v7")
	manilaSupportUnexpected(t, err, 404)
	if errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native Get must not add ErrNotFound: %v", err)
	}
}

func TestManilaMiscAPIVersionsCommonLookupGetsByIDAndListsAll(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v2.0/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"versions":[{"id":"v2.0","status":"CURRENT"}]}`)
	})
	cloud.Mux.HandleFunc("GET /v7/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"versions":[{"id":"v2.0","status":"CURRENT"},{"id":"v1.0","status":"DEPRECATED"}]}`)
	})
	api := apiversions.New(cloud.Client("shared-file-system", "/v2/"))
	ctx := context.Background()
	value, err := api.Find(ctx, resource.ID("v2.0"))
	if err != nil || value.Status != "CURRENT" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if _, err := api.Find(ctx, resource.ID("v7")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("Find maps a missing document to ErrNotFound: %v", err)
	}
	all, err := api.All(ctx)
	if err != nil || len(all) != 2 {
		t.Fatalf("all=%v error=%v", all, err)
	}
	calls.Store(0)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := api.Get(canceled, "v2.0"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
}
