package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func heatResourceScope(t *testing.T, api *stackresources.API) *stackresources.StackScope {
	t.Helper()
	scope, err := api.ForStack(stacks.StackIdentity{Name: "app", ID: "fixed"})
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func heatResourceBody(name, ownerName, ownerID, status string) map[string]any {
	value := map[string]any{
		"resource_name": name, "logical_resource_id": "logical-" + name,
		"physical_resource_id": "physical-" + name, "resource_status": status,
		"resource_status_reason": "state changed", "resource_type": "OS::Nova::Server",
		"creation_time": "2026-10-01T01:02:03Z", "updated_time": "2026-10-01T01:03:03",
		"attributes": map[string]any{"port": 8080}, "description": "server description",
		"required_by": []string{"dependent"},
	}
	if ownerName != "" {
		value["links"] = []map[string]string{
			{"rel": "stack", "href": "https://identity-only.invalid/heat/stacks/" + ownerName + "/" + ownerID},
			{"rel": "self", "href": "/heat/stacks/" + ownerName + "/" + ownerID + "/resources/" + name},
		}
	}
	return value
}

func heatResourceResponse(w http.ResponseWriter, key string, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{key: value})
}

func TestHeatStackResourcesKnownPairAndParentResolution(t *testing.T) {
	cloud := testcloud.New(t)
	api := stackresources.New(cloud.Client("orchestration", "/heat"))
	var requests atomic.Int32
	cloud.Mux.HandleFunc("GET /heat/stacks", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		testcloud.JSON(w, 200, `{"stacks":[{"stack_name":"app-copy","id":"other"},{"stack_name":"app","id":"fixed"}]}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/fixed", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Redirect(w, r, "/heat/stacks/app/fixed", http.StatusFound)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		testcloud.JSON(w, 200, `{"stack":{"stack_name":"app","id":"fixed"}}`)
	})
	known := heatResourceScope(t, api)
	copy := known.Identity()
	copy.ID = "changed"
	bound, err := known.InResource(context.Background(), resource.ID("node"))
	if err != nil || requests.Load() != 0 || known.Identity().ID != "fixed" || bound.Identity().Name != "node" {
		t.Fatalf("known pair performed a lookup: bound=%v err=%v requests=%d", bound, err, requests.Load())
	}
	for _, ref := range []resource.Ref{resource.Name("app"), resource.ID("fixed")} {
		scope, err := api.InStack(context.Background(), ref)
		if err != nil || scope.Identity() != known.Identity() {
			t.Fatalf("resolved=%v err=%v", scope, err)
		}
	}
	if requests.Load() != 3 {
		t.Fatalf("parent lookup requests=%d", requests.Load())
	}
}

func TestHeatStackResourcesSinglePageModelsAndLocalFilters(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("orchestration", "/heat")
	client.MoreHeaders = map[string]string{"X-Policy": "keep"}
	scope := heatResourceScope(t, stackresources.New(client))
	var requests atomic.Int32
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Policy") != "keep" || r.URL.Query().Get("name") != "" {
			t.Errorf("headers or local exact name policy changed: %s headers=%v", r.URL, r.Header)
		}
		if requests.Load() == 1 && (r.URL.Query().Get("nested_depth") != "2" || r.URL.Query().Get("with_detail") != "true") {
			t.Errorf("typed resource query=%s", r.URL)
		}
		if requests.Load() == 2 && r.URL.Query().Get("status") != "CREATE_COMPLETE" {
			t.Errorf("case-insensitive status was not normalized for Heat: %s", r.URL)
		}
		root := heatResourceBody("node", "app", "fixed", "CREATE_COMPLETE")
		child := heatResourceBody("node", "nested", "child", "UPDATE_IN_PROGRESS")
		child["parent_resource"] = "group"
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"resources": []any{root, child, heatResourceBody("node-copy", "app", "fixed", "CREATE_COMPLETE")},
			"links":     []any{map[string]string{"rel": "next", "href": "/must-not-follow"}},
		})
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("single-page list followed an invented pagination endpoint: %s", r.URL)
		http.Error(w, "unexpected", 500)
	})
	all, err := scope.All(context.Background(), stackresources.WithNestedDepth(2), stackresources.WithDetails(true))
	if err != nil || len(all) != 3 || requests.Load() != 1 {
		t.Fatalf("all=%v err=%v requests=%d", all, err, requests.Load())
	}
	if !all[0].Detailed || !all[0].OwnerKnown || all[0].Owner != scope.Identity() || all[1].Owner.ID != "child" || all[1].ParentResource != "group" {
		t.Fatalf("owner/detail fields were lost: root=%+v child=%+v", all[0], all[1])
	}
	if all[0].LogicalID != "logical-node" || all[0].PhysicalID != "physical-node" || all[0].Attributes["port"] != float64(8080) || all[0].CreationTime.IsZero() || all[0].UpdatedTime.IsZero() || len(all[0].RequiredBy) != 1 {
		t.Fatalf("native fields were lost: %+v", all[0])
	}
	filtered, err := scope.All(context.Background(), resource.WithName("node"), resource.WithStatus("create_complete"))
	if err != nil || len(filtered) != 1 || filtered[0].Name != "node" || filtered[0].Detailed {
		t.Fatalf("local name/status filter=%v err=%v", filtered, err)
	}
	for _, err := range scope.List(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if requests.Load() != 3 {
		t.Fatalf("break changed single-page behavior: requests=%d", requests.Load())
	}
}

func TestHeatStackResourcesExactFindRoutingAndMissing(t *testing.T) {
	cloud := testcloud.New(t)
	scope := heatResourceScope(t, stackresources.New(cloud.Client("orchestration", "/heat")))
	var lists, gets atomic.Int32
	var duplicate, empty atomic.Bool
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if empty.Load() {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		values := []any{heatResourceBody("node-copy", "app", "fixed", "CREATE_COMPLETE"), heatResourceBody("node", "app", "fixed", "CREATE_COMPLETE")}
		if duplicate.Load() {
			values = append(values, heatResourceBody("node", "app", "fixed", "UPDATE_COMPLETE"))
		}
		heatResourceResponse(w, "resources", values)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources/node", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		heatResourceResponse(w, "resource", heatResourceBody("node", "app", "fixed", "CREATE_COMPLETE"))
	})
	id, err := scope.ResolveID(context.Background(), resource.ID("node"))
	if err != nil || id != "node" || lists.Load() != 0 || gets.Load() != 0 {
		t.Fatalf("explicit routing ID was looked up: id=%s err=%v", id, err)
	}
	id, err = scope.ResolveID(context.Background(), resource.Name("node"))
	value, getErr := scope.Find(context.Background(), resource.ID(id))
	if err != nil || getErr != nil || id != "node" || value.Name != "node" || value.PhysicalID == id || value.LogicalID == id || lists.Load() != 1 || gets.Load() != 1 {
		t.Fatalf("name→routing ID→GET failed: id=%s value=%v err=%v/%v", id, value, err, getErr)
	}
	duplicate.Store(true)
	if _, err := scope.Find(context.Background(), resource.Name("node")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatalf("duplicate exact name=%v", err)
	}
	empty.Store(true)
	if _, err := scope.Find(context.Background(), resource.Name("node")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	missing, err := scope.Find(context.Background(), resource.Name("node"), resource.WithIgnoreMissing())
	if err != nil || missing != nil {
		t.Fatalf("missing=%v err=%v", missing, err)
	}
	all, err := scope.All(context.Background())
	if err != nil || all == nil || len(all) != 0 {
		t.Fatalf("204 list=%v err=%v", all, err)
	}
}

func TestHeatStackResourcesNestedOwnerCannotMutateRoot(t *testing.T) {
	cloud := testcloud.New(t)
	api := stackresources.New(cloud.Client("orchestration", "/heat"))
	root := heatResourceScope(t, api)
	var rootGets, patches atomic.Int32
	child := heatResourceBody("node", "nested", "child", "CHECK_FAILED")
	child["parent_resource"] = "group"
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources", func(w http.ResponseWriter, r *http.Request) {
		heatResourceResponse(w, "resources", []any{child})
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources/node", func(w http.ResponseWriter, r *http.Request) {
		rootGets.Add(1)
		heatResourceResponse(w, "resource", child)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/nested/child/resources/node", func(w http.ResponseWriter, r *http.Request) {
		heatResourceResponse(w, "resource", child)
	})
	cloud.Mux.HandleFunc("PATCH /heat/stacks/nested/child/resources/node", func(w http.ResponseWriter, r *http.Request) {
		patches.Add(1)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !reflect.DeepEqual(body, map[string]any{"mark_unhealthy": false, "resource_status_reason": "recovered"}) {
			t.Errorf("health change JSON=%v", body)
		}
		w.WriteHeader(http.StatusOK)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("nested identity changed into root mutation: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	for _, ref := range []resource.Ref{resource.Name("node"), resource.ID("node")} {
		if err := root.MarkUnhealthy(context.Background(), ref, false); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nested mutation was not rejected: %v", err)
		}
	}
	if rootGets.Load() != 1 || patches.Load() != 0 {
		t.Fatalf("named nested result was reinterpreted: gets=%d patches=%d", rootGets.Load(), patches.Load())
	}
	listed, err := root.All(context.Background(), stackresources.WithNestedDepth(1))
	if err != nil || len(listed) != 1 {
		t.Fatal(listed, err)
	}
	identity, err := listed[0].Identity()
	if err != nil || identity.Stack.ID != "child" || identity.Name != "node" {
		t.Fatalf("nested identity=%v err=%v", identity, err)
	}
	fixed, err := api.ForResource(identity)
	if err != nil {
		t.Fatal(err)
	}
	copy := fixed.Identity()
	copy.Stack.ID = "changed"
	copy.Name = "changed"
	if err := fixed.MarkUnhealthy(context.Background(), false, stackresources.WithHealthReason("recovered")); err != nil || patches.Load() != 1 || fixed.Identity() != identity {
		t.Fatalf("actual child health change failed: err=%v patches=%d identity=%v", err, patches.Load(), fixed.Identity())
	}
}

func TestHeatStackResourcesUnknownOwnerAndPhysicalFallbackAreBlocked(t *testing.T) {
	for _, fixture := range []struct{ name, responseName, ownerName string }{
		{"unknown owner", "node", ""}, {"physical fallback", "other", "app"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			scope := heatResourceScope(t, stackresources.New(cloud.Client("orchestration", "/heat")))
			var gets, patches atomic.Int32
			cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources/node", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				value := heatResourceBody(fixture.responseName, fixture.ownerName, "fixed", "CHECK_FAILED")
				value["parent_resource"] = "group"
				heatResourceResponse(w, "resource", value)
			})
			cloud.Mux.HandleFunc("PATCH /heat/stacks/app/fixed/resources/node", func(w http.ResponseWriter, r *http.Request) {
				patches.Add(1)
				w.WriteHeader(http.StatusOK)
			})
			err := scope.MarkUnhealthy(context.Background(), resource.ID("node"), true)
			if err == nil || patches.Load() != 0 || gets.Load() != 1 {
				t.Fatalf("unsafe health change: err=%v gets=%d patches=%d", err, gets.Load(), patches.Load())
			}
			if fixture.ownerName == "" && !errors.Is(err, resource.ErrUnsupported) {
				t.Fatalf("unknown owner must explain missing capability: %v", err)
			}
		})
	}
}

func TestHeatStackResourcesMetadataPreservesJSONAndHTTPCauses(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("orchestration", "/heat")
	client.MoreHeaders = map[string]string{"X-Policy": "keep"}
	scope := heatResourceScope(t, stackresources.New(client))
	var code atomic.Int32
	code.Store(200)
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources/node/metadata", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Policy") != "keep" {
			t.Errorf("metadata headers=%v", r.Header)
		}
		testcloud.JSON(w, int(code.Load()), `{"metadata":{"object":{"enabled":false,"count":3},"array":["a",2],"null":null}}`)
	})
	metadata, err := scope.Metadata(context.Background(), resource.ID("node"))
	if err != nil || metadata["object"].(map[string]any)["enabled"] != false || metadata["object"].(map[string]any)["count"] != float64(3) || len(metadata["array"].([]any)) != 2 || metadata["null"] != nil {
		t.Fatalf("metadata=%v err=%v", metadata, err)
	}
	for _, status := range []int{403, 404} {
		code.Store(int32(status))
		_, err := scope.Metadata(context.Background(), resource.ID("node"))
		var cause gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &cause) || cause.Actual != status || errors.Is(err, resource.ErrNotFound) != (status == 404) {
			t.Fatalf("metadata HTTP %d error=%v", status, err)
		}
	}
}

func TestHeatStackResourcesWaitUsesFixedResourceNameAndHeatFailureStates(t *testing.T) {
	for _, target := range []string{"CREATE_COMPLETE", "UPDATE_FAILED", "ROLLBACK_FAILED", "CHECK_FAILED"} {
		t.Run(target, func(t *testing.T) {
			cloud := testcloud.New(t)
			scope := heatResourceScope(t, stackresources.New(cloud.Client("orchestration", "/heat")))
			var gets, lists atomic.Int32
			cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				heatResourceResponse(w, "resources", []any{heatResourceBody("node", "app", "fixed", "CREATE_COMPLETE")})
			})
			cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources/node", func(w http.ResponseWriter, r *http.Request) {
				status := "CREATE_IN_PROGRESS"
				if gets.Add(1) > 1 {
					status = target
				}
				heatResourceResponse(w, "resource", heatResourceBody("node", "app", "fixed", status))
			})
			wanted := "CREATE_COMPLETE"
			if target == "CHECK_FAILED" {
				wanted = target // An explicitly requested health failure state is a valid target.
			}
			value, err := scope.Wait(context.Background(), resource.Name("node"), wanted, resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second))
			if target == "CREATE_COMPLETE" || target == wanted {
				if err != nil || value == nil || value.Name != "node" || !value.Detailed {
					t.Fatalf("wait=%v err=%v", value, err)
				}
			} else {
				var failed *resource.FailedStateError
				if !errors.As(err, &failed) || failed.ID != "node" || failed.Status != target {
					t.Fatalf("failed state=%v", err)
				}
			}
			if lists.Load() != 1 || gets.Load() != 2 {
				t.Fatalf("wait used physical/logical ID or a summary state: lists=%d gets=%d", lists.Load(), gets.Load())
			}
		})
	}
}

func TestHeatStackResourcesModelAndHTTPErrors(t *testing.T) {
	for _, fixture := range []struct {
		name, body string
		status     int
	}{
		{"HTTP403", `{"error":"denied"}`, 403},
		{"HTTP404", `{"error":"gone"}`, 404},
		{"nil resource", `{"resource":null}`, 200},
		{"invalid timestamp", `{"resource":{"resource_name":"node","updated_time":"invalid"}}`, 200},
		{"missing routing identity", `{"resource":{"logical_resource_id":"node"}}`, 200},
		{"wrong resource name", `{"resource":{"resource_name":"other"}}`, 200},
		{"owner mismatch", `{"resource":{"resource_name":"node","links":[{"rel":"stack","href":"/heat/stacks/other/fixed"}]}}`, 200},
		{"conflicting links", `{"resource":{"resource_name":"node","links":[{"rel":"stack","href":"/heat/stacks/app/fixed"},{"rel":"self","href":"/heat/stacks/app/other/resources/node"}]}}`, 200},
		{"invalid owner path", `{"resource":{"resource_name":"node","links":[{"rel":"stack","href":"/heat/stacks/app/fixed/extra"}]}}`, 200},
		{"wrong self name", `{"resource":{"resource_name":"node","links":[{"rel":"self","href":"/heat/stacks/app/fixed/resources/other"}]}}`, 200},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			scope := heatResourceScope(t, stackresources.New(cloud.Client("orchestration", "/heat")))
			cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources/node", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, fixture.status, fixture.body)
			})
			value, err := scope.Get(context.Background(), "node")
			if err == nil || value != nil {
				t.Fatalf("invalid model/HTTP result=%v err=%v", value, err)
			}
			if fixture.status != 200 {
				var cause gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &cause) || cause.Actual != fixture.status || errors.Is(err, resource.ErrNotFound) != (fixture.status == 404) {
					t.Fatalf("HTTP cause lost: %v", err)
				}
			}
		})
	}
}

func TestHeatStackResourcesValidationBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	api := stackresources.New(cloud.Client("orchestration", "/heat"))
	scope := heatResourceScope(t, api)
	var requests atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "unexpected request", 500)
	})
	for _, name := range []string{"", "../node", "a%2Fb", "node\x00", "."} {
		if _, err := scope.Get(context.Background(), name); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid name %q=%v", name, err)
		}
	}
	for _, options := range [][]resource.ListOption{
		{resource.WithPageSize(1)}, {resource.WithQuery("marker", "node")},
		{resource.WithQuery("unknown", "value")}, {stackresources.WithNestedDepth(-1)},
		{resource.WithQuery("with_detail", "invalid")}, {nil},
	} {
		if _, err := scope.All(context.Background(), options...); err == nil {
			t.Fatalf("invalid/unsupported query accepted: %v", options)
		}
	}
	if err := scope.MarkUnhealthy(context.Background(), resource.Name("node"), true, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := scope.Wait(context.Background(), resource.Name("node"), ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := scope.Wait(context.Background(), resource.Name("node"), "CREATE_COMPLETE", resource.WithTimeout(-1)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.ForResource(stackresources.ResourceIdentity{Stack: scope.Identity(), Name: "a/b"}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, getErr := scope.Get(ctx, "node")
	_, metaErr := scope.Metadata(ctx, resource.ID("node"))
	healthErr := scope.MarkUnhealthy(ctx, resource.ID("node"), true)
	_, parentErr := api.InStack(ctx, resource.Name("app"))
	_, bindErr := scope.InResource(ctx, resource.ID("node"))
	_, listErr := scope.All(ctx)
	for _, err := range []error{getErr, metaErr, healthErr, parentErr, bindErr, listErr} {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled context=%v", err)
		}
	}
	if requests.Load() != 0 {
		t.Fatalf("validation/cancellation made %d HTTP requests", requests.Load())
	}
}

func TestHeatStackResourcesListAndWaitErrorCauses(t *testing.T) {
	cloud := testcloud.New(t)
	scope := heatResourceScope(t, stackresources.New(cloud.Client("orchestration", "/heat")))
	var badTimestamp atomic.Bool
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources", func(w http.ResponseWriter, r *http.Request) {
		if badTimestamp.Load() {
			testcloud.JSON(w, 200, `{"resources":[{"resource_name":"node","updated_time":"invalid"}]}`)
			return
		}
		testcloud.JSON(w, 403, `{"error":"denied"}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources/node", func(w http.ResponseWriter, r *http.Request) {
		heatResourceResponse(w, "resource", heatResourceBody("node", "app", "fixed", "CREATE_IN_PROGRESS"))
	})
	_, err := scope.All(context.Background())
	var httpCause gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &httpCause) || httpCause.Actual != 403 {
		t.Fatalf("list HTTP cause lost: %v", err)
	}
	badTimestamp.Store(true)
	_, err = scope.All(context.Background())
	var decode *time.ParseError
	if !errors.As(err, &decode) {
		t.Fatalf("list decode cause lost: %v", err)
	}
	_, err = scope.Wait(context.Background(), resource.ID("node"), "CREATE_COMPLETE", resource.WithTimeout(10*time.Millisecond), resource.WithPollInterval(time.Second))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("wait timeout cause lost: %v", err)
	}
}

func TestHeatStackResourcesCancellationReachesHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	scope := heatResourceScope(t, stackresources.New(cloud.Client("orchestration", "/heat")))
	started := make(chan struct{})
	stopped := make(chan struct{})
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources/node/metadata", func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(stopped)
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := scope.Metadata(ctx, resource.ID("node"))
		result <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("metadata request never started")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("metadata cancellation did not return")
	}
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("server request did not receive cancellation")
	}
}

func TestHeatStackResourcesHealthHTTPErrorsAndMetadataDecodeCause(t *testing.T) {
	cloud := testcloud.New(t)
	scope := heatResourceScope(t, stackresources.New(cloud.Client("orchestration", "/heat")))
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources/node", func(w http.ResponseWriter, r *http.Request) {
		heatResourceResponse(w, "resource", heatResourceBody("node", "app", "fixed", "CHECK_FAILED"))
	})
	cloud.Mux.HandleFunc("PATCH /heat/stacks/app/fixed/resources/node", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 403, `{"error":"denied"}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources/node/metadata", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"metadata":"not an object"}`)
	})
	err := scope.MarkUnhealthy(context.Background(), resource.ID("node"), true)
	var cause gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &cause) || cause.Actual != 403 {
		t.Fatalf("health HTTP cause lost: %v", err)
	}
	_, err = scope.Metadata(context.Background(), resource.ID("node"))
	var decode *json.UnmarshalTypeError
	if !errors.As(err, &decode) {
		t.Fatalf("metadata decode cause lost: %v", err)
	}
	var nilView *stackresources.ResourceView
	if _, err := nilView.Identity(); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := (&stackresources.ResourceView{Resource: stackresources.Resource{Name: "node"}}).Identity(); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(fmt.Sprintf("unknown owner identity=%v", err))
	}
}
