package api_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents"
)

func TestHeatNativeStackEventsFindFollowsNameRedirectAndKeepsErrors(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("orchestration", "/heat")
	client.MoreHeaders = map[string]string{"X-Policy": "keep"}
	api := stackevents.New(client)
	var hits atomic.Int32
	cloud.Mux.HandleFunc("GET /heat/stacks/app/events", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.RawQuery != "" {
			t.Errorf("native Find must not add a query: %s", r.URL)
		}
		http.Redirect(w, r, "/heat/stacks/app/fixed/events", http.StatusFound)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Policy") != "keep" {
			t.Errorf("redirect lost headers: %v", r.Header)
		}
		testcloud.JSON(w, 200, `{"events":[{"id":"e1","resource_name":"web","logical_resource_id":"web","physical_resource_id":null,"event_time":"2026-10-01T01:02:03Z","resource_status":"CREATE_IN_PROGRESS","resource_status_reason":"started","resource_properties":{"n":null},"links":[{"rel":"self","href":"https://example.invalid/e1"}]},{"id":"e2","event_time":"2026-10-01T01:02:04.5"}],"links":[{"rel":"next","href":"?marker=e2"}]}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/empty/events", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"events":[]}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/missing/events", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/denied/events", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 403, `{}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/badtime/events", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"events":[{"id":"x","event_time":"yesterday"}]}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/malformed/events", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"events":`)
	})
	values, err := api.Find(context.Background(), "app")
	if err != nil || len(values) != 2 || hits.Load() != 1 {
		t.Fatalf("values=%v err=%v hits=%d", values, err, hits.Load())
	}
	first := values[0]
	if first.ID != "e1" || first.ResourceName != "web" || first.PhysicalResourceID != "" || first.Time.IsZero() || first.ResourceStatus != "CREATE_IN_PROGRESS" ||
		first.ResourceStatusReason != "started" || !reflect.DeepEqual(first.ResourceProperties, map[string]any{"n": nil}) || len(first.Links) != 1 || values[1].ID != "e2" || values[1].Time.IsZero() {
		t.Fatalf("decoded events=%#v", values)
	}
	if hits.Load() != 1 {
		t.Fatalf("non-paged Find followed the next link: hits=%d", hits.Load())
	}
	if values, err := api.Find(context.Background(), "empty"); err != nil || len(values) != 0 {
		t.Fatalf("empty=%v err=%v", values, err)
	}
	_, err = api.Find(context.Background(), "missing")
	heatNativeRequireStatus(t, err, 404)
	_, err = api.Find(context.Background(), "denied")
	heatNativeRequireStatus(t, err, 403)
	if _, err := api.Find(context.Background(), "badtime"); err == nil {
		t.Fatal("invalid event_time was accepted")
	}
	if _, err := api.Find(context.Background(), "malformed"); err == nil {
		t.Fatal("malformed JSON was accepted")
	}
	if _, err := api.Find(heatNativeCanceled(), "app"); !errors.Is(err, context.Canceled) || hits.Load() != 1 {
		t.Fatalf("canceled Find err=%v hits=%d", err, hits.Load())
	}
}

func TestHeatNativeStackEventsResourceListSerializesEveryTypedOption(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	var headers []http.Header
	cloud.Mux.HandleFunc("GET "+heatEventBase+"/resources/web/events", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		headers = append(headers, r.Header.Clone())
		switch r.URL.Query().Get("marker") {
		case "start":
			testcloud.JSON(w, 200, `{"events":[{"id":"e1","resource_name":"wire-name"}]}`)
		case "e1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected marker in %s", r.URL)
		}
	})
	scope := heatResourceEventScope(t, heatEventScope(t, cloud), "web")
	values, err := scope.All(context.Background(),
		stackevents.WithListResourceEventsOptions(stackevents.ListResourceEventsOpts{
			Marker: "start", Limit: 3,
			ResourceActions:  []string{"CREATE", "ROLLBACK", "CREATE"},
			ResourceStatuses: []string{"FAILED", "COMPLETE"},
			ResourceNames:    []string{"web", "db"},
			ResourceTypes:    []string{"OS::Nova::Server", "OS::Cinder::Volume"},
			SortKey:          stackevents.SortKey("created_at"),
			SortDir:          stackevents.SortDir("desc"),
		}),
		stackevents.WithListResourceEventsQuery("nested_depth", "2"),
		stackevents.WithListResourceEventsHeader("X-Events", "full"))
	if err != nil || len(values) != 1 || values[0].ID != "e1" || len(queries) != 2 {
		t.Fatalf("values=%v err=%v queries=%v", values, err, queries)
	}
	first := queries[0]
	want := url.Values{
		"marker": {"start"}, "limit": {"3"},
		"resource_action": {"CREATE", "ROLLBACK", "CREATE"}, "resource_status": {"FAILED", "COMPLETE"},
		"resource_name": {"web", "db"}, "resource_type": {"OS::Nova::Server", "OS::Cinder::Volume"},
		"sort_keys": {"created_at"}, "sort_dir": {"desc"}, "nested_depth": {"2"},
	}
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("first query=%v\nwant %v", first, want)
	}
	// The next-page marker replaces only the marker; the other typed options are repeated unchanged.
	second := url.Values{}
	for key, value := range queries[1] {
		second[key] = value
	}
	if second.Get("marker") != "e1" {
		t.Fatalf("second marker=%v", second["marker"])
	}
	second.Set("marker", "start")
	if !reflect.DeepEqual(second, want) {
		t.Fatalf("second query=%v\nwant %v", second, want)
	}
	for _, h := range headers {
		if h.Get("X-Events") != "full" || h.Get("X-Auth-Token") != "test-token" {
			t.Errorf("headers=%v", h)
		}
	}
	// Zero options send no query at all.
	queries = nil
	if _, err := scope.All(context.Background(), stackevents.WithListResourceEventsOptions(stackevents.ListResourceEventsOpts{Marker: "start"})); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(queries[0], url.Values{"marker": {"start"}}) {
		t.Fatalf("minimal query=%v", queries[0])
	}
}
