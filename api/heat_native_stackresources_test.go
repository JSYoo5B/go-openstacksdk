package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources"
)

func heatNativeResourcesAPI(cloud *testcloud.Cloud, headers map[string]string) *stackresources.API {
	client := cloud.Client("orchestration", "/heat")
	client.MoreHeaders = headers
	return stackresources.New(client)
}

func TestHeatNativeStackResourcesFindFollowsNameRedirectAndKeepsErrors(t *testing.T) {
	cloud := testcloud.New(t)
	api := heatNativeResourcesAPI(cloud, map[string]string{"X-Policy": "keep"})
	var hits atomic.Int32
	cloud.Mux.HandleFunc("GET /heat/stacks/app/resources", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.RawQuery != "" {
			t.Errorf("native Find must not add a query: %s", r.URL)
		}
		http.Redirect(w, r, "/heat/stacks/app/fixed/resources", http.StatusFound)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Policy") != "keep" || r.Header.Get("Accept") != "application/json" {
			t.Errorf("redirect lost headers: %v", r.Header)
		}
		testcloud.JSON(w, 200, `{"resources":[{"resource_name":"web","logical_resource_id":"web","physical_resource_id":"p1","resource_type":"OS::Nova::Server","resource_status":"CREATE_COMPLETE","resource_status_reason":"","required_by":["db"],"creation_time":"2026-10-01T01:02:03Z","updated_time":"2026-10-01T01:02:04","links":[{"rel":"stack","href":"https://example.invalid/stack"}]},{"resource_name":"db","physical_resource_id":null}]}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/empty/resources", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"resources":[]}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/missing/resources", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{"error":{"message":"missing"}}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/denied/resources", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 403, `{}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/bad/resources", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"resources":[{"resource_name":"x","creation_time":"yesterday"}]}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/malformed/resources", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"resources":`)
	})
	values, err := api.Find(context.Background(), "app")
	if err != nil || len(values) != 2 || hits.Load() != 1 {
		t.Fatalf("values=%v err=%v hits=%d", values, err, hits.Load())
	}
	web := values[0]
	if web.Name != "web" || web.PhysicalID != "p1" || web.Type != "OS::Nova::Server" || web.Status != "CREATE_COMPLETE" || web.CreationTime.IsZero() || web.UpdatedTime.IsZero() ||
		!reflect.DeepEqual(web.RequiredBy, []any{"db"}) || len(web.Links) != 1 || values[1].Name != "db" || values[1].PhysicalID != "" {
		t.Fatalf("decoded resources=%#v", values)
	}
	if values, err := api.Find(context.Background(), "empty"); err != nil || len(values) != 0 {
		t.Fatalf("empty=%v err=%v", values, err)
	}
	_, err = api.Find(context.Background(), "missing")
	heatNativeRequireStatus(t, err, 404)
	_, err = api.Find(context.Background(), "denied")
	heatNativeRequireStatus(t, err, 403)
	if _, err := api.Find(context.Background(), "bad"); err == nil {
		t.Fatal("invalid timestamp was accepted")
	}
	if _, err := api.Find(context.Background(), "malformed"); err == nil {
		t.Fatal("malformed JSON was accepted")
	}
	if _, err := api.Find(heatNativeCanceled(), "app"); !errors.Is(err, context.Canceled) || hits.Load() != 1 {
		t.Fatalf("canceled Find err=%v hits=%d", err, hits.Load())
	}
}

func TestHeatNativeResourceTypesListSchemaAndTemplate(t *testing.T) {
	cloud := testcloud.New(t)
	api := heatNativeResourcesAPI(cloud, map[string]string{"X-Policy": "keep"})
	var calls atomic.Int32
	check := func(r *http.Request) {
		calls.Add(1)
		if r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Policy") != "keep" {
			t.Errorf("headers=%v", r.Header)
		}
	}
	cloud.Mux.HandleFunc("GET /heat/resource_types", func(w http.ResponseWriter, r *http.Request) {
		check(r)
		switch r.URL.Query().Get("mode") {
		case "":
			testcloud.JSON(w, 200, `{"resource_types":["OS::Nova::Server","OS::Cinder::Volume","OS::Heat::None"]}`)
		}
	})
	cloud.Mux.HandleFunc("GET /heat/resource_types/OS::Nova::Server", func(w http.ResponseWriter, r *http.Request) {
		check(r)
		testcloud.JSON(w, 200, `{"resource_type":"OS::Nova::Server","attributes":{"name":{"type":"string"}},"properties":{"flavor":{"required":true,"default":null}},"support_status":{"status":"SUPPORTED","version":null}}`)
	})
	cloud.Mux.HandleFunc("GET /heat/resource_types/OS::Missing", func(w http.ResponseWriter, r *http.Request) {
		check(r)
		testcloud.JSON(w, 404, `{"error":{"message":"missing type"}}`)
	})
	cloud.Mux.HandleFunc("GET /heat/resource_types/OS::Forbidden", func(w http.ResponseWriter, r *http.Request) {
		check(r)
		testcloud.JSON(w, 403, `{}`)
	})
	cloud.Mux.HandleFunc("GET /heat/resource_types/OS::Bad", func(w http.ResponseWriter, r *http.Request) {
		check(r)
		testcloud.JSON(w, 200, `{"properties":`)
	})
	cloud.Mux.HandleFunc("GET /heat/resource_types/OS::Nova::Server/template", func(w http.ResponseWriter, r *http.Request) {
		check(r)
		testcloud.JSON(w, 200, `{"heat_template_version":"2016-10-14","resources":{"server":{"type":"OS::Nova::Server","properties":{"flavor":null}}}}`)
	})
	cloud.Mux.HandleFunc("GET /heat/resource_types/OS::Missing/template", func(w http.ResponseWriter, r *http.Request) {
		check(r)
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /heat/resource_types/OS::Bad/template", func(w http.ResponseWriter, r *http.Request) {
		check(r)
		testcloud.JSON(w, 200, `not json`)
	})

	var types []string
	for value, err := range api.ListTypes(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		types = append(types, *value)
	}
	if !reflect.DeepEqual(types, []string{"OS::Nova::Server", "OS::Cinder::Volume", "OS::Heat::None"}) {
		t.Fatalf("server order was not preserved: %v", types)
	}
	if calls.Load() != 1 {
		t.Fatalf("ListTypes must be one non-paged request, calls=%d", calls.Load())
	}

	schema, err := api.Schema(context.Background(), "OS::Nova::Server")
	if err != nil || schema == nil || schema.ResourceType != "OS::Nova::Server" || schema.Attributes["name"] == nil ||
		!reflect.DeepEqual(schema.Properties["flavor"], map[string]any{"required": true, "default": nil}) ||
		!reflect.DeepEqual(schema.SupportStatus, map[string]any{"status": "SUPPORTED", "version": nil}) {
		t.Fatalf("schema=%#v err=%v", schema, err)
	}
	_, err = api.Schema(context.Background(), "OS::Missing")
	heatNativeRequireStatus(t, err, 404)
	_, err = api.Schema(context.Background(), "OS::Forbidden")
	heatNativeRequireStatus(t, err, 403)
	if _, err := api.Schema(context.Background(), "OS::Bad"); err == nil {
		t.Fatal("malformed schema was accepted")
	}

	template, err := api.Template(context.Background(), "OS::Nova::Server")
	want := "{\n  \"heat_template_version\": \"2016-10-14\",\n  \"resources\": {\n    \"server\": {\n      \"properties\": {\n        \"flavor\": null\n      },\n      \"type\": \"OS::Nova::Server\"\n    }\n  }\n}"
	if err != nil || string(template) != want {
		t.Fatalf("template=%q err=%v", template, err)
	}
	_, err = api.Template(context.Background(), "OS::Missing")
	heatNativeRequireStatus(t, err, 404)
	if _, err := api.Template(context.Background(), "OS::Bad"); err == nil {
		t.Fatal("non-JSON template was accepted")
	}

	// Error and cancellation behavior of the paged list.
	cloud.Mux.HandleFunc("GET /heat/other/resource_types", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 500, `{}`) })
	failing := stackresources.New(cloud.Client("orchestration", "/heat/other"))
	for _, err := range failing.ListTypes(context.Background()) {
		heatNativeRequireStatus(t, err, 500)
	}
	cloud.Mux.HandleFunc("GET /heat/empty/resource_types", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	for value, err := range stackresources.New(cloud.Client("orchestration", "/heat/empty")).ListTypes(context.Background()) {
		t.Fatalf("204 produced value=%v err=%v", value, err)
	}
	before := calls.Load()
	for _, err := range api.ListTypes(heatNativeCanceled()) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled ListTypes err=%v", err)
		}
	}
	for _, call := range []func() error{
		func() error { _, err := api.Schema(heatNativeCanceled(), "OS::Nova::Server"); return err },
		func() error { _, err := api.Template(heatNativeCanceled(), "OS::Nova::Server"); return err },
	} {
		if err := call(); !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled call err=%v", err)
		}
	}
	if calls.Load() != before {
		t.Fatal("canceled calls reached the server")
	}
}

func TestHeatNativeStackResourcesMarkUnhealthyPhysicalIDAndExtensions(t *testing.T) {
	cloud := testcloud.New(t)
	api := heatNativeResourcesAPI(cloud, nil)
	bodies := make(chan map[string]any, 16)
	var status atomic.Int32
	status.Store(204)
	var paths []string
	cloud.Mux.HandleFunc("PATCH /heat/stacks/app/fixed/resources/", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.EscapedPath())
		bodies <- heatNativeBody(t, r)
		if r.Header.Get("Content-Type") != "application/json" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Errorf("headers=%v", r.Header)
		}
		code := int(status.Load())
		if code == 200 || code == 202 || code == 204 {
			w.WriteHeader(code)
			return
		}
		testcloud.JSON(w, code, `{}`)
	})
	physical := "6f3b8f1c-0000-4000-8000-000000000001"
	// The native route sends the segment unchanged, so a physical ID is targeted without any GET preflight.
	if err := api.MarkUnhealthy(context.Background(), "app", "fixed", physical, stackresources.MarkUnhealthyOpts{MarkUnhealthy: true, ResourceStatusReason: "bad disk"}); err != nil {
		t.Fatal(err)
	}
	if body := <-bodies; !reflect.DeepEqual(body, map[string]any{"mark_unhealthy": true, "resource_status_reason": "bad disk"}) || paths[0] != "/heat/stacks/app/fixed/resources/"+physical {
		t.Fatalf("physical-ID request path=%v body=%v", paths, body)
	}
	// False is preserved and an empty reason is omitted.
	if err := api.MarkUnhealthy(context.Background(), "app", "fixed", "web", stackresources.MarkUnhealthyOpts{}); err != nil {
		t.Fatal(err)
	}
	if body := <-bodies; !reflect.DeepEqual(body, map[string]any{"mark_unhealthy": false}) {
		t.Fatalf("zero opts body=%v", body)
	}
	// Option helpers and extensions.
	if err := api.MarkUnhealthy(context.Background(), "app", "fixed", "web", stackresources.MarkUnhealthyOpts{},
		stackresources.WithMarkUnhealthyOptions(stackresources.MarkUnhealthyOpts{MarkUnhealthy: true, ResourceStatusReason: "why"}),
		stackresources.WithMarkUnhealthyField("vendor", map[string]any{"keep": nil})); err != nil {
		t.Fatal(err)
	}
	if body := <-bodies; !reflect.DeepEqual(body, map[string]any{"mark_unhealthy": true, "resource_status_reason": "why", "vendor": map[string]any{"keep": nil}}) {
		t.Fatalf("extended body=%v", body)
	}
	for _, code := range []int32{200, 202} {
		status.Store(code)
		if err := api.MarkUnhealthy(context.Background(), "app", "fixed", "web", stackresources.MarkUnhealthyOpts{MarkUnhealthy: true}); err != nil {
			t.Fatalf("status %d: %v", code, err)
		}
		<-bodies
	}
	for _, code := range []int32{201, 400, 403, 404, 409, 500} {
		status.Store(code)
		heatNativeRequireStatus(t, api.MarkUnhealthy(context.Background(), "app", "fixed", "web", stackresources.MarkUnhealthyOpts{MarkUnhealthy: true}), int(code))
		<-bodies
	}
	// Extensions cannot replace typed fields; nothing is sent.
	if err := api.MarkUnhealthy(context.Background(), "app", "fixed", "web", stackresources.MarkUnhealthyOpts{MarkUnhealthy: true},
		stackresources.WithMarkUnhealthyField("mark_unhealthy", false)); err == nil {
		t.Fatal("typed mark_unhealthy field was overridden by an extension")
	}
	if err := api.MarkUnhealthy(context.Background(), "app", "fixed", "web", stackresources.MarkUnhealthyOpts{}, nil); err == nil {
		t.Fatal("nil option was accepted")
	}
	if err := api.MarkUnhealthy(heatNativeCanceled(), "app", "fixed", "web", stackresources.MarkUnhealthyOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled err=%v", err)
	}
	if len(bodies) != 0 {
		t.Fatalf("pre-HTTP failures sent %d requests", len(bodies))
	}
}
