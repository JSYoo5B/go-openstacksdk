package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/orchestration/v1/stackevents"
	"github.com/JSYoo5B/gophercloudsdk/orchestration/v1/stacks"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

const heatEventBase = "/heat/v1/project/stacks/app/fixed"
const heatEventWire = `{"id":"wire-event-id","resource_name":"wire-resource","logical_resource_id":"logical","physical_resource_id":null,"event_time":"2026-10-01T02:03:04.123456","resource_status":"CREATE_COMPLETE","resource_status_reason":"ready","resource_type":"OS::Heat::RandomString","resource_properties":{"length":8,"salt":null},"links":[{"rel":"self","href":"https://example.invalid/event"}],"vendor":{"count":9007199254740993},"optional":null}`

func heatEventScope(t *testing.T, cloud *testcloud.Cloud) *stackevents.StackEventScope {
	t.Helper()
	scope, err := stackevents.New(cloud.Client("orchestration", "/heat/v1/project")).ForStack(stacks.StackIdentity{Name: "app", ID: "fixed"})
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func heatResourceEventScope(t *testing.T, scope *stackevents.StackEventScope, name string) *stackevents.ResourceEventScope {
	t.Helper()
	bound, err := scope.ForResource(name)
	if err != nil {
		t.Fatal(err)
	}
	return bound
}

func checkHeatEventRequest(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Method != http.MethodGet || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("Accept") != "application/json" {
		t.Errorf("request method=%s headers=%v", r.Method, r.Header)
	}
}

func TestHeatStackEventsResolvesNameOnceAndSeparatesStackAndResourceLists(t *testing.T) {
	cloud := testcloud.New(t)
	var lookups, stackLists, resourceLists atomic.Int32
	cloud.Mux.HandleFunc("GET /heat/v1/project/stacks", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		checkHeatEventRequest(t, r)
		if r.URL.Query().Get("name") != "" {
			t.Errorf("stack lookup invented a name query: %s", r.URL)
		}
		if r.URL.Query().Get("marker") == "" {
			testcloud.JSON(w, 200, `{"stacks":[{"stack_name":"app-copy","id":"other"}],"links":[{"rel":"next","href":"?marker=next"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"stacks":[{"stack_name":"app","id":"fixed"}]}`)
	})
	cloud.Mux.HandleFunc("GET "+heatEventBase+"/events", func(w http.ResponseWriter, r *http.Request) {
		stackLists.Add(1)
		checkHeatEventRequest(t, r)
		query := r.URL.Query()
		if query.Get("limit") != "2" || !reflect.DeepEqual(query["resource_action"], []string{"CREATE", "UPDATE"}) || !reflect.DeepEqual(query["resource_status"], []string{"COMPLETE", "FAILED"}) || !reflect.DeepEqual(query["resource_name"], []string{"random", "server"}) || query.Get("resource_type") != "OS::Heat::RandomString" || query.Get("sort_keys") != "created_at" || query.Get("sort_dir") != "asc" || query.Get("nested_depth") != "2" || query.Get("vendor") != "a&b" || r.Header.Get("X-Events") != "stack" {
			t.Errorf("stack list query=%s headers=%v", r.URL.RawQuery, r.Header)
		}
		w.Header().Set("X-Request-Id", "stack-page")
		switch query.Get("marker") {
		case "start":
			testcloud.JSON(w, 200, `{"events":[`+heatEventWire+`]}`)
		case "wire-event-id":
			testcloud.JSON(w, 200, `{"events":[]}`)
		default:
			t.Errorf("wrong marker %q", query.Get("marker"))
		}
	})
	cloud.Mux.HandleFunc("GET "+heatEventBase+"/resources/random/events", func(w http.ResponseWriter, r *http.Request) {
		resourceLists.Add(1)
		checkHeatEventRequest(t, r)
		if r.URL.Query().Get("resource_name") != "random" || r.URL.Query().Get("nested_depth") != "1" || r.Header.Get("X-Events") != "resource" {
			t.Errorf("resource list query=%s headers=%v", r.URL.RawQuery, r.Header)
		}
		if r.URL.Query().Get("marker") != "" {
			w.WriteHeader(204)
			return
		}
		w.Header().Set("X-Request-Id", "resource-page")
		testcloud.JSON(w, 200, `{"events":[`+heatEventWire+`]}`)
	})
	ctx := context.Background()
	scope, err := stackevents.New(cloud.Client("orchestration", "/heat/v1/project")).InStack(ctx, resource.Name("app"))
	if err != nil || scope.Identity() != (stacks.StackIdentity{Name: "app", ID: "fixed"}) {
		t.Fatalf("scope=%v err=%v", scope, err)
	}
	values, err := scope.All(ctx, stackevents.WithListOptions(stackevents.ListOpts{
		Marker: "start", Limit: 2,
		ResourceActions: []stackevents.ResourceAction{"CREATE", "UPDATE"}, ResourceStatuses: []stackevents.ResourceStatus{"COMPLETE", "FAILED"}, ResourceNames: []string{"random", "server"}, ResourceTypes: []string{"OS::Heat::RandomString"}, SortKey: "created_at", SortDir: "asc",
	}), stackevents.WithListQuery("nested_depth", "2"), stackevents.WithListQuery("vendor", "a&b"), stackevents.WithListHeader("X-Events", "stack"))
	if err != nil || len(values) != 1 {
		t.Fatalf("stack list=%v err=%v", values, err)
	}
	value := values[0]
	if value.Detailed || value.RequestStack != scope.Identity() || value.RequestResourceName != "" || value.RequestEventID != "" || value.ResourceName != "wire-resource" || value.ResourceType == nil || *value.ResourceType != "OS::Heat::RandomString" || value.Time.Nanosecond() != 123456000 || value.PhysicalResourceID != "" || value.Header.Get("X-Request-Id") != "stack-page" || string(value.Body["optional"]) != "null" || !strings.Contains(string(value.Body["vendor"]), "9007199254740993") {
		t.Fatalf("summary did not preserve actual response and request identity: %+v", value)
	}
	bound := heatResourceEventScope(t, scope, "random")
	values, err = bound.All(ctx, stackevents.WithListResourceEventsOptions(stackevents.ListResourceEventsOpts{ResourceNames: []string{"random"}}), stackevents.WithListResourceEventsQuery("nested_depth", "1"), stackevents.WithListResourceEventsHeader("X-Events", "resource"))
	if err != nil || len(values) != 1 || values[0].RequestResourceName != "random" || values[0].ResourceName != "wire-resource" || values[0].Header.Get("X-Request-Id") != "resource-page" {
		t.Fatalf("resource list=%v err=%v", values, err)
	}
	identity := bound.Identity()
	identity.Name = "changed"
	if scope.Identity().Name != "app" || bound.Identity().Name != "app" || bound.ResourceName() != "random" || lookups.Load() != 2 || stackLists.Load() != 2 || resourceLists.Load() != 2 {
		t.Fatalf("fixed identity changed or repeated lookup: lookup=%d stack=%d resource=%d", lookups.Load(), stackLists.Load(), resourceLists.Load())
	}
}

func TestHeatStackEventsGetUsesCompleteResourcePathAndPreservesRequestIdentity(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET "+heatEventBase+"/resources/random/events/request-event", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		checkHeatEventRequest(t, r)
		if r.URL.RawQuery != "" || r.Header.Get("X-Event") != "detail" || r.Header.Get("X-Policy") != "client" {
			t.Errorf("detail query=%s headers=%v", r.URL.RawQuery, r.Header)
		}
		w.Header().Set("X-Request-Id", "detail-request")
		testcloud.JSON(w, 200, `{"event":`+heatEventWire+`}`)
	})
	client := cloud.Client("orchestration", "/heat/v1/project")
	client.MoreHeaders = map[string]string{"X-Policy": "client"}
	scope, err := stackevents.New(client).ForStack(stacks.StackIdentity{Name: "app", ID: "fixed"})
	if err != nil || calls.Load() != 0 {
		t.Fatalf("ForStack made requests: scope=%v err=%v requests=%d", scope, err, calls.Load())
	}
	bound := heatResourceEventScope(t, scope, "random")
	for _, get := range []func() (*stackevents.EventResource, error){
		func() (*stackevents.EventResource, error) {
			return scope.Get(context.Background(), "random", "request-event", stackevents.WithGetHeader("X-Event", "detail"))
		},
		func() (*stackevents.EventResource, error) {
			return bound.Get(context.Background(), "request-event", stackevents.WithGetHeader("X-Event", "detail"))
		},
	} {
		value, err := get()
		if err != nil || !value.Detailed || value.ID != "wire-event-id" || value.ResourceName != "wire-resource" || value.RequestStack != scope.Identity() || value.RequestResourceName != "random" || value.RequestEventID != "request-event" || value.ResourceProperties["length"] != float64(8) || value.ResourceProperties["salt"] != nil || value.LogicalResourceID != "logical" || value.ResourceStatus != "CREATE_COMPLETE" || value.ResourceStatusReason != "ready" || len(value.Links) != 1 || value.Header.Get("X-Request-Id") != "detail-request" {
			t.Fatalf("detail=%+v err=%v", value, err)
		}
		if string(value.Body["physical_resource_id"]) != "null" || string(value.Body["resource_properties"]) == "" || string(value.Body["vendor"]) != `{"count":9007199254740993}` {
			t.Fatalf("raw fields lost: %+v", value.Body)
		}
	}
	if calls.Load() != 2 || client.Endpoint != cloud.Server.URL+"/heat/v1/project/" || client.MoreHeaders["X-Policy"] != "client" {
		t.Fatal("request count or existing client policy changed")
	}
}

func TestHeatStackEventsInStackIDReusesCanonicalLookup(t *testing.T) {
	cloud := testcloud.New(t)
	var lookup, list atomic.Int32
	cloud.Mux.HandleFunc("GET /heat/v1/project/stacks/fixed", func(w http.ResponseWriter, r *http.Request) {
		lookup.Add(1)
		checkHeatEventRequest(t, r)
		http.Redirect(w, r, heatEventBase, http.StatusFound)
	})
	cloud.Mux.HandleFunc("GET "+heatEventBase, func(w http.ResponseWriter, r *http.Request) {
		lookup.Add(1)
		checkHeatEventRequest(t, r)
		testcloud.JSON(w, 200, `{"stack":{"stack_name":"app","id":"fixed"}}`)
	})
	cloud.Mux.HandleFunc("GET "+heatEventBase+"/events", func(w http.ResponseWriter, r *http.Request) {
		list.Add(1)
		testcloud.JSON(w, 200, `{"events":[]}`)
	})
	scope, err := stackevents.New(cloud.Client("orchestration", "/heat/v1/project")).InStack(context.Background(), resource.ID("fixed"))
	if err != nil || scope.Identity() != (stacks.StackIdentity{Name: "app", ID: "fixed"}) {
		t.Fatalf("scope=%v err=%v", scope, err)
	}
	for range 2 {
		if values, err := scope.All(context.Background()); err != nil || values == nil || len(values) != 0 {
			t.Fatalf("empty events=%v err=%v", values, err)
		}
	}
	if lookup.Load() != 2 || list.Load() != 2 {
		t.Fatalf("canonical lookup repeated: lookups=%d lists=%d", lookup.Load(), list.Load())
	}
}

func TestHeatStackEventsPreflightValidatesIdentityPathsAndOptions(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 500, "{}")
	})
	api := stackevents.New(cloud.Client("orchestration", "/heat/v1/project"))
	for _, identity := range []stacks.StackIdentity{{}, {Name: "app"}, {Name: "bad/name", ID: "fixed"}, {Name: "app", ID: "bad%id"}, {Name: "app", ID: "nul\x00"}} {
		if _, err := api.ForStack(identity); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("invalid identity=%v err=%v", identity, err)
		}
	}
	if _, err := stackevents.New(nil).ForStack(stacks.StackIdentity{Name: "app", ID: "fixed"}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	scope := heatEventScope(t, cloud)
	bound := heatResourceEventScope(t, scope, "random")
	ctx := context.Background()
	for _, value := range []string{"", ".", "..", "bad/name", "bad%id", "bad?query", "bad#fragment", "two names", "nul\x00"} {
		if _, err := scope.ForResource(value); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("resource=%q err=%v", value, err)
		}
		if _, err := bound.Get(ctx, value); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("eventID=%q err=%v", value, err)
		}
	}
	for _, ref := range []resource.Ref{{}, resource.ID("bad/id"), resource.Name(" ")} {
		if _, err := api.InStack(ctx, ref); !errors.Is(err, resource.ErrInvalidOption) {
			t.Error(err)
		}
	}
	for _, option := range []stackevents.GetOption{nil, request.WithQuery[stackevents.GetOpts]("query", "unsupported"), request.WithField[stackevents.GetOpts]("field", false), request.WithArgument[stackevents.GetOpts]("arg", false), stackevents.WithGetHeader("x-auth-token", "override"), stackevents.WithGetHeader("accept", "override"), stackevents.WithGetHeader("openstack-api-version", "override")} {
		if _, err := bound.Get(ctx, "event-id", option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("get option=%v", err)
		}
	}
	for _, option := range []stackevents.ListOption{nil, request.WithField[stackevents.ListOpts]("field", false), request.WithArgument[stackevents.ListOpts]("arg", false), stackevents.WithListHeader("Accept", "override")} {
		if _, err := scope.All(ctx, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("list option=%v", err)
		}
	}
	for _, option := range []stackevents.ListResourceEventsOption{nil, request.WithField[stackevents.ListResourceEventsOpts]("field", false), request.WithArgument[stackevents.ListResourceEventsOpts]("arg", false), stackevents.WithListResourceEventsHeader("X-Auth-Token", "override")} {
		if _, err := bound.All(ctx, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("resource list option=%v", err)
		}
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	for _, run := range []func() error{
		func() error { _, err := api.InStack(canceled, resource.Name("app")); return err },
		func() error { _, err := scope.Get(canceled, "random", "event-id"); return err },
		func() error { _, err := bound.Get(canceled, "event-id"); return err },
		func() error { _, err := scope.All(canceled); return err },
		func() error { _, err := bound.All(canceled); return err },
	} {
		if err := run(); !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("preflight errors made %d HTTP requests", calls.Load())
	}
}

func TestHeatStackEventsScopesEscapeUnicodeSegmentsAndRemainConcurrent(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		checkHeatEventRequest(t, r)
		if !strings.Contains(r.RequestURI, "%ED%95%9C%EA%B8%80") || r.URL.Path != "/heat/stacks/한글/app-id/resources/리소스/events/이벤트" {
			t.Errorf("escaped URI=%q path=%q", r.RequestURI, r.URL.Path)
		}
		w.Header().Set("X-Request-Id", "request")
		testcloud.JSON(w, 200, `{"event":`+heatEventWire+`}`)
	})
	api := stackevents.New(cloud.Client("orchestration", "/heat"))
	scope, err := api.ForStack(stacks.StackIdentity{Name: "한글", ID: "app-id"})
	if err != nil {
		t.Fatal(err)
	}
	bound := heatResourceEventScope(t, scope, "리소스")
	var work sync.WaitGroup
	for range 8 {
		work.Add(1)
		go func() {
			defer work.Done()
			value, err := bound.Get(context.Background(), "이벤트")
			if err != nil {
				t.Error(err)
				return
			}
			if value.RequestStack != scope.Identity() || value.RequestResourceName != "리소스" || value.RequestEventID != "이벤트" {
				t.Errorf("request identity=%+v", value)
			}
			value.Header.Set("X-Request-Id", "caller mutation")
			value.Body["vendor"] = json.RawMessage(`false`)
		}()
	}
	work.Wait()
	if calls.Load() != 8 {
		t.Fatalf("calls=%d", calls.Load())
	}
}
