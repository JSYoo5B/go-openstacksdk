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

	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const magnumClusterBody = `{"uuid":"cl-1","name":"prod","status":"CREATE_IN_PROGRESS","status_reason":"","node_count":0,"master_count":1,"floating_ip_enabled":false,"cluster_template_id":"tpl-1","created_at":"2026-01-02T03:04:05+00:00","updated_at":null,"vendor_extra":1}`

func newMagnumClusters(t *testing.T, cloud *testcloud.Cloud, microversion string) *clusters.API {
	t.Helper()
	client := cloud.Client("container-infra", "/v1")
	client.Microversion = microversion
	return clusters.New(client)
}

func magnumRawBody(t *testing.T, r *http.Request) json.RawMessage {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("request body: %v", err)
	}
	return data
}

func magnumRawFields(t *testing.T, r *http.Request) map[string]string {
	t.Helper()
	var body map[string]json.RawMessage
	if err := json.Unmarshal(magnumRawBody(t, r), &body); err != nil {
		t.Errorf("request body: %v", err)
	}
	got := map[string]string{}
	for key, value := range body {
		got[key] = string(value)
	}
	return got
}

func magnumIntPtr(value int) *int { return &value }
func magnumFalse() *bool          { value := false; return &value }
func magnumUnexpected(err error) int {
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if errors.As(err, &unexpected) {
		return unexpected.Actual
	}
	return 0
}

func TestMagnumClusterCreateSendsNullDefaultsAndExplicitZeroFalse(t *testing.T) {
	cloud := testcloud.New(t)
	var bodies []map[string]string
	var versions []string
	cloud.Mux.HandleFunc("POST /v1/clusters", func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, magnumRawFields(t, r))
		versions = append(versions, r.Header.Get("OpenStack-API-Version"))
		testcloud.JSON(w, 202, `{"uuid":"cl-1"}`)
	})
	api := newMagnumClusters(t, cloud, "1.10")
	id, err := api.Create(context.Background(), clusters.CreateOpts{
		ClusterTemplateID: "tpl-1", Name: "prod", NodeCount: magnumIntPtr(0), CreateTimeout: magnumIntPtr(0), FloatingIPEnabled: magnumFalse(),
		MergeLabels: magnumFalse(), Labels: map[string]string{},
	}, clusters.WithCreateField("vendor", map[string]bool{"on": false}))
	if err != nil || id != "cl-1" {
		t.Fatalf("id=%q error=%v", id, err)
	}
	want := map[string]string{"cluster_template_id": `"tpl-1"`, "name": `"prod"`, "node_count": `0`, "create_timeout": `0`, "floating_ip_enabled": `false`, "merge_labels": `false`, "vendor": `{"on":false}`}
	if !reflect.DeepEqual(bodies[0], want) {
		t.Fatalf("explicit body=%v", bodies[0])
	}
	// Omitted values: the native builder still sends create_timeout=null and name="".
	if _, err := api.Create(context.Background(), clusters.CreateOpts{ClusterTemplateID: "tpl-1"}); err != nil {
		t.Fatal(err)
	}
	want = map[string]string{"cluster_template_id": `"tpl-1"`, "create_timeout": `null`, "name": `""`}
	if !reflect.DeepEqual(bodies[1], want) {
		t.Fatalf("default body=%v", bodies[1])
	}
	if !reflect.DeepEqual(versions, []string{"container-infra 1.10", "container-infra 1.10"}) {
		t.Fatalf("versions=%v", versions)
	}
}

func TestMagnumClusterOmitsVersionHeaderWithoutClientMicroversion(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v1/clusters/cl-1", func(w http.ResponseWriter, r *http.Request) {
		if _, present := r.Header["Openstack-Api-Version"]; present {
			t.Errorf("unexpected version header %q", r.Header.Get("OpenStack-API-Version"))
		}
		testcloud.JSON(w, 200, magnumClusterBody)
	})
	if _, err := newMagnumClusters(t, cloud, "").Get(context.Background(), "cl-1"); err != nil {
		t.Fatal(err)
	}
}

func TestMagnumClusterCreateAcceptsOnly202AndNeverReplays(t *testing.T) {
	for _, status := range []int{202, 200, 201, 204, 500} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /v1/clusters", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, status, `{"uuid":"cl-1"}`)
		})
		id, err := newMagnumClusters(t, cloud, "").Create(context.Background(), clusters.CreateOpts{ClusterTemplateID: "tpl-1"})
		if (status == 202) != (err == nil) || (status == 202 && id != "cl-1") {
			t.Fatalf("status %d id=%q error=%v", status, id, err)
		}
		if status != 202 && magnumUnexpected(err) != status {
			t.Fatalf("status %d error=%v", status, err)
		}
		if calls.Load() != 1 {
			t.Fatalf("status %d was replayed: %d calls", status, calls.Load())
		}
	}
}

func TestMagnumClusterRejectsInvalidInputsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newMagnumClusters(t, cloud, "")
	ctx := context.Background()
	ok := clusters.CreateOpts{ClusterTemplateID: "tpl-1"}
	if _, err := api.Create(ctx, clusters.CreateOpts{}); err == nil {
		t.Fatal("missing template ID must fail locally")
	}
	for _, key := range []string{"name", "node_count", "create_timeout", "cluster_template_id"} {
		if _, err := api.Create(ctx, ok, clusters.WithCreateField(key, 1)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("create extension %q: %v", key, err)
		}
	}
	if _, err := api.Create(ctx, ok, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	if _, err := api.Resize(ctx, "cl-1", clusters.ResizeOpts{}); err == nil {
		t.Fatal("missing node count must fail locally")
	}
	if _, err := api.Resize(ctx, "cl-1", clusters.ResizeOpts{NodeCount: magnumIntPtr(1)}, clusters.WithResizeField("node_count", 2)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("resize collision: %v", err)
	}
	if _, err := api.Resize(ctx, "cl-1", clusters.ResizeOpts{NodeCount: magnumIntPtr(1)}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil resize option: %v", err)
	}
	if _, err := api.Upgrade(ctx, "cl-1", clusters.UpgradeOpts{}); err == nil {
		t.Fatal("missing upgrade template must fail locally")
	}
	if _, err := api.Upgrade(ctx, "cl-1", clusters.UpgradeOpts{ClusterTemplate: "t"}, clusters.WithUpgradeField("max_batch_size", 3)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("upgrade collision: %v", err)
	}
	if _, err := api.Upgrade(ctx, "cl-1", clusters.UpgradeOpts{ClusterTemplate: "t"}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil upgrade option: %v", err)
	}
	if _, err := api.Update(ctx, "cl-1", []clusters.UpdateOpts{{Path: "/node_count"}}); err == nil {
		t.Fatal("missing patch op must fail locally")
	}
	for _, err := range api.List(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list option: %v", err)
		}
	}
	for _, err := range api.ListDetail(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list-detail option: %v", err)
		}
	}
}

func TestMagnumClusterGetDecodesAndKeepsNativeStatusErrors(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v1/clusters/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("id") {
		case "cl-1":
			testcloud.JSON(w, 200, magnumClusterBody)
		case "accepted":
			testcloud.JSON(w, 202, magnumClusterBody)
		default:
			testcloud.JSON(w, 404, `{"errors":[{"detail":"gone"}]}`)
		}
	})
	api := newMagnumClusters(t, cloud, "")
	ctx := context.Background()
	value, err := api.Get(ctx, "cl-1")
	if err != nil || value == nil || value.UUID != "cl-1" || value.Status != "CREATE_IN_PROGRESS" || value.NodeCount != 0 || value.FloatingIPEnabled || value.CreatedAt.IsZero() || !value.UpdatedAt.IsZero() {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if _, err := api.Get(ctx, "missing"); magnumUnexpected(err) != 404 || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	if _, err := api.Get(ctx, "accepted"); magnumUnexpected(err) != 202 {
		t.Fatalf("Get accepts only 200: %v", err)
	}
	if _, err := api.Resources.Get(ctx, "missing"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("Collection.Get maps 404 to ErrNotFound: %v", err)
	}
}

func TestMagnumClusterListAndListDetailPaginateWithQueries(t *testing.T) {
	cloud := testcloud.New(t)
	seen := map[string][]url.Values{}
	handler := func(path string) {
		cloud.Mux.HandleFunc("GET /v1/clusters"+path, func(w http.ResponseWriter, r *http.Request) {
			seen[path] = append(seen[path], r.URL.Query())
			if r.URL.Query().Get("marker") == "cl-1" {
				testcloud.JSON(w, 200, `{"clusters":[{"uuid":"cl-2","name":"two"}]}`)
				return
			}
			next := cloud.Server.URL + "/v1/clusters" + path + "?marker=cl-1&limit=1"
			testcloud.JSON(w, 200, `{"clusters":[{"uuid":"cl-1","name":"one"}],"next":"`+next+`"}`)
		})
	}
	handler("")
	handler("/detail")
	api := newMagnumClusters(t, cloud, "")
	ctx := context.Background()
	opts := clusters.ListOpts{Limit: 1, SortKey: "name", SortDir: "desc"}
	var ids []string
	for value, err := range api.List(ctx, clusters.WithListOptions(opts), clusters.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.UUID)
	}
	for value, err := range api.ListDetail(ctx, clusters.WithListDetailOptions(opts), clusters.WithListDetailQuery("vendor", "y")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.UUID)
	}
	if !reflect.DeepEqual(ids, []string{"cl-1", "cl-2", "cl-1", "cl-2"}) {
		t.Fatalf("ids=%v", ids)
	}
	for path, vendor := range map[string]string{"": "x", "/detail": "y"} {
		got := seen[path]
		if len(got) != 2 {
			t.Fatalf("%q requests=%v", path, got)
		}
		first := got[0]
		if first.Get("limit") != "1" || first.Get("sort_key") != "name" || first.Get("sort_dir") != "desc" || first.Get("vendor") != vendor || first.Has("marker") {
			t.Fatalf("%q first query=%v", path, first)
		}
		if got[1].Get("marker") != "cl-1" {
			t.Fatalf("%q second query=%v", path, got[1])
		}
	}
}

func TestMagnumClusterCollectionUsesDetailListForNamesAndFiltersStatusLocally(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v1/clusters/detail", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"clusters":[{"uuid":"cl-1","name":"prod","status":"CREATE_COMPLETE"},{"uuid":"cl-2","name":"prod-2","status":"CREATE_FAILED"}]}`)
	})
	cloud.Mux.HandleFunc("GET /v1/clusters", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("collection lookups must use the detail route: %s", r.URL)
		testcloud.JSON(w, 200, `{"clusters":[]}`)
	})
	api := newMagnumClusters(t, cloud, "")
	ctx := context.Background()
	value, err := api.Find(ctx, resource.Name("prod"))
	if err != nil || value == nil || value.UUID != "cl-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if _, err := api.Find(ctx, resource.Name("nope")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing name: %v", err)
	}
	if value, err := api.Find(ctx, resource.Name("nope"), resource.WithIgnoreMissing()); err != nil || value != nil {
		t.Fatalf("ignore missing: %+v %v", value, err)
	}
	queries = nil
	all, err := api.All(ctx, resource.WithStatus("CREATE_COMPLETE"))
	if err != nil || len(all) != 1 || all[0].UUID != "cl-1" {
		t.Fatalf("all=%v error=%v", all, err)
	}
	if len(queries) != 1 || queries[0].Has("status") {
		t.Fatalf("status must not be sent as a wire filter: %v", queries)
	}
}

func TestMagnumClusterUpdateSendsPatchArrayAndAccepts200Or202(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, true}, {201, false}, {204, false}} {
		cloud := testcloud.New(t)
		var raw string
		var contentType string
		cloud.Mux.HandleFunc("PATCH /v1/clusters/cl-1", func(w http.ResponseWriter, r *http.Request) {
			raw = string(magnumRawBody(t, r))
			contentType = r.Header.Get("Content-Type")
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, `{"uuid":"cl-1"}`)
		})
		id, err := newMagnumClusters(t, cloud, "").Update(context.Background(), "cl-1", []clusters.UpdateOpts{
			{Op: clusters.ReplaceOp, Path: "/node_count", Value: 0},
			{Op: clusters.ReplaceOp, Path: "/labels/flag", Value: false},
			{Op: clusters.RemoveOp, Path: "/labels/old"},
		})
		if tc.success != (err == nil) || (tc.success && id != "cl-1") {
			t.Fatalf("status %d id=%q error=%v", tc.status, id, err)
		}
		if !tc.success && magnumUnexpected(err) != tc.status {
			t.Fatalf("status %d error=%v", tc.status, err)
		}
		want := `[{"op":"replace","path":"/node_count","value":0},{"op":"replace","path":"/labels/flag","value":false},{"op":"remove","path":"/labels/old"}]`
		if raw != want || contentType != "application/json" {
			t.Fatalf("body=%s content-type=%q", raw, contentType)
		}
	}
}

func TestMagnumClusterResizeKeepsZeroAndAccepts200Or202(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, true}, {201, false}, {204, false}} {
		cloud := testcloud.New(t)
		var bodies []map[string]string
		cloud.Mux.HandleFunc("POST /v1/clusters/cl-1/actions/resize", func(w http.ResponseWriter, r *http.Request) {
			bodies = append(bodies, magnumRawFields(t, r))
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, `{"uuid":"cl-1"}`)
		})
		api := newMagnumClusters(t, cloud, "")
		id, err := api.Resize(context.Background(), "cl-1", clusters.ResizeOpts{NodeCount: magnumIntPtr(0), NodesToRemove: []string{"n1"}, NodeGroup: "workers"},
			clusters.WithResizeField("vendor", false))
		if tc.success != (err == nil) || (tc.success && id != "cl-1") {
			t.Fatalf("status %d id=%q error=%v", tc.status, id, err)
		}
		if !tc.success && magnumUnexpected(err) != tc.status {
			t.Fatalf("status %d error=%v", tc.status, err)
		}
		want := map[string]string{"node_count": `0`, "nodes_to_remove": `["n1"]`, "nodegroup": `"workers"`, "vendor": `false`}
		if !reflect.DeepEqual(bodies[0], want) {
			t.Fatalf("body=%v", bodies[0])
		}
		if tc.status == 200 {
			// Empty NodesToRemove and NodeGroup are omitted by the native builder.
			if _, err := api.Resize(context.Background(), "cl-1", clusters.ResizeOpts{NodeCount: magnumIntPtr(3), NodesToRemove: []string{}}); err != nil {
				t.Fatal(err)
			}
			if want := map[string]string{"node_count": `3`}; !reflect.DeepEqual(bodies[1], want) {
				t.Fatalf("minimal body=%v", bodies[1])
			}
		}
	}
}

func TestMagnumClusterUpgradeDefaultsBatchSizeAndRequires202(t *testing.T) {
	for _, status := range []int{202, 200, 201} {
		cloud := testcloud.New(t)
		var bodies []map[string]string
		cloud.Mux.HandleFunc("POST /v1/clusters/cl-1/actions/upgrade", func(w http.ResponseWriter, r *http.Request) {
			bodies = append(bodies, magnumRawFields(t, r))
			testcloud.JSON(w, status, `{"uuid":"cl-1"}`)
		})
		api := newMagnumClusters(t, cloud, "")
		id, err := api.Upgrade(context.Background(), "cl-1", clusters.UpgradeOpts{ClusterTemplate: "tpl-2"})
		if (status == 202) != (err == nil) || (status == 202 && id != "cl-1") {
			t.Fatalf("status %d id=%q error=%v", status, id, err)
		}
		if !reflect.DeepEqual(bodies[0], map[string]string{"cluster_template": `"tpl-2"`, "max_batch_size": `1`}) {
			t.Fatalf("default body=%v", bodies[0])
		}
		if status == 202 {
			if _, err := api.Upgrade(context.Background(), "cl-1", clusters.UpgradeOpts{ClusterTemplate: "tpl-2", MaxBatchSize: magnumIntPtr(0), NodeGroup: "workers"}); err != nil {
				t.Fatal(err)
			}
			want := map[string]string{"cluster_template": `"tpl-2"`, "max_batch_size": `0`, "nodegroup": `"workers"`}
			if !reflect.DeepEqual(bodies[1], want) {
				t.Fatalf("explicit body=%v", bodies[1])
			}
		}
	}
}

func TestMagnumClusterDeleteStatusesAndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("DELETE /v1/clusters/{id}", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.PathValue("id") {
		case "missing":
			testcloud.JSON(w, 404, `{}`)
		case "ok200":
			testcloud.JSON(w, 200, `{}`)
		case "ok202":
			w.WriteHeader(202)
		default:
			w.WriteHeader(204)
		}
	})
	api := newMagnumClusters(t, cloud, "")
	ctx := context.Background()
	for _, id := range []string{"a", "ok202"} {
		if err := api.Delete(ctx, id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	if err := api.Delete(ctx, "ok200"); magnumUnexpected(err) != 200 {
		t.Fatalf("200 is not an accepted delete code: %v", err)
	}
	if err := api.Delete(ctx, "missing"); magnumUnexpected(err) != 404 || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native delete keeps the 404 response error: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
	if calls.Load() != 5 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestMagnumClusterCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	api := newMagnumClusters(t, cloud, "")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Create(canceled, clusters.CreateOpts{ClusterTemplateID: "t"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("create: %v", err)
	}
	if _, err := api.Get(canceled, "cl-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("get: %v", err)
	}
	if _, err := api.Update(canceled, "cl-1", []clusters.UpdateOpts{{Op: clusters.ReplaceOp, Path: "/node_count", Value: 1}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("update: %v", err)
	}
	if _, err := api.Resize(canceled, "cl-1", clusters.ResizeOpts{NodeCount: magnumIntPtr(1)}); !errors.Is(err, context.Canceled) {
		t.Fatalf("resize: %v", err)
	}
	if _, err := api.Upgrade(canceled, "cl-1", clusters.UpgradeOpts{ClusterTemplate: "t"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("upgrade: %v", err)
	}
	if err := api.Delete(canceled, "cl-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("delete: %v", err)
	}
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("list: %v", err)
		}
	}
	for _, err := range api.ListDetail(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("list detail: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, clusters.CreateOpts{ClusterTemplateID: "t"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("an uncertain create must not be replayed: %d calls", calls.Load())
	}
}
