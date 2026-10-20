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

	"github.com/JSYoo5B/go-openstacksdk/containerinfra/apiversions"
	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/certificates"
	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/clustertemplates"
	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/nodegroups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func newMagnumPieces(t *testing.T, cloud *testcloud.Cloud, microversion string) (*clustertemplates.API, *nodegroups.API, *certificates.API, *apiversions.API) {
	t.Helper()
	client := cloud.Client("container-infra", "/v1")
	client.Microversion = microversion
	return clustertemplates.New(client), nodegroups.New(client), certificates.New(client), apiversions.New(client)
}

// ---- cluster templates ----

const magnumTemplateBody = `{"uuid":"tpl-1","name":"k8s","coe":"kubernetes","image_id":"img-1","public":false,"floating_ip_enabled":false,"tls_disabled":false,"hidden":false,"docker_volume_size":0,"created_at":"2026-01-02T03:04:05+00:00","vendor_extra":1}`

func TestMagnumTemplateCreateKeepsExplicitFalseAndRequires201(t *testing.T) {
	for _, status := range []int{201, 200, 202, 204} {
		cloud := testcloud.New(t)
		var bodies []map[string]string
		var version string
		cloud.Mux.HandleFunc("POST /v1/clustertemplates", func(w http.ResponseWriter, r *http.Request) {
			bodies = append(bodies, magnumRawFields(t, r))
			version = r.Header.Get("OpenStack-API-Version")
			if status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, status, magnumTemplateBody)
		})
		api, _, _, _ := newMagnumPieces(t, cloud, "1.10")
		value, err := api.Create(context.Background(), clustertemplates.CreateOpts{
			COE: "kubernetes", ImageID: "img-1", Public: magnumFalse(), FloatingIPEnabled: magnumFalse(), TLSDisabled: magnumFalse(), Hidden: magnumFalse(),
			DockerVolumeSize: magnumIntPtr(0), Labels: map[string]string{"k": ""},
		}, clustertemplates.WithCreateField("vendor", nil))
		if (status == 201) != (err == nil) || (status == 201 && (value == nil || value.UUID != "tpl-1" || value.Public || value.FloatingIPEnabled || value.CreatedAt.IsZero())) {
			t.Fatalf("status %d value=%+v error=%v", status, value, err)
		}
		if status != 201 && magnumUnexpected(err) != status {
			t.Fatalf("status %d error=%v", status, err)
		}
		want := map[string]string{"coe": `"kubernetes"`, "image_id": `"img-1"`, "public": `false`, "floating_ip_enabled": `false`, "tls_disabled": `false`, "hidden": `false`, "docker_volume_size": `0`, "labels": `{"k":""}`, "vendor": `null`}
		if !reflect.DeepEqual(bodies[0], want) || version != "container-infra 1.10" {
			t.Fatalf("body=%v version=%q", bodies[0], version)
		}
	}
}

func TestMagnumTemplateRejectsInvalidInputsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api, _, _, _ := newMagnumPieces(t, cloud, "")
	ctx := context.Background()
	ok := clustertemplates.CreateOpts{COE: "kubernetes", ImageID: "img-1"}
	if _, err := api.Create(ctx, clustertemplates.CreateOpts{COE: "kubernetes"}); err == nil {
		t.Fatal("missing image must fail locally")
	}
	for _, key := range []string{"coe", "image_id", "public", "hidden"} {
		if _, err := api.Create(ctx, ok, clustertemplates.WithCreateField(key, "x")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("extension %q: %v", key, err)
		}
	}
	if _, err := api.Create(ctx, ok, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	if _, err := api.Update(ctx, "tpl-1", []clustertemplates.UpdateOpts{{Path: "/name"}}); err == nil {
		t.Fatal("missing patch op must fail locally")
	}
	for _, err := range api.List(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list option: %v", err)
		}
	}
}

func TestMagnumTemplateGetListAndCollectionPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v1/clustertemplates/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "tpl-1" {
			testcloud.JSON(w, 200, magnumTemplateBody)
			return
		}
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v1/clustertemplates", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		if r.URL.Query().Get("marker") == "tpl-1" {
			testcloud.JSON(w, 200, `{"clustertemplates":[{"uuid":"tpl-2","name":"two"}]}`)
			return
		}
		next := cloud.Server.URL + "/v1/clustertemplates?marker=tpl-1&limit=1"
		testcloud.JSON(w, 200, `{"clustertemplates":[{"uuid":"tpl-1","name":"k8s"}],"next":"`+next+`"}`)
	})
	api, _, _, _ := newMagnumPieces(t, cloud, "")
	ctx := context.Background()
	value, err := api.Get(ctx, "tpl-1")
	if err != nil || value.COE != "kubernetes" || value.Public || value.DockerVolumeSize != 0 {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if _, err := api.Get(ctx, "missing"); magnumUnexpected(err) != 404 || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	var ids []string
	for value, err := range api.List(ctx, clustertemplates.WithListOptions(clustertemplates.ListOpts{Limit: 1, SortKey: "name", SortDir: "asc"}), clustertemplates.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.UUID)
	}
	if !reflect.DeepEqual(ids, []string{"tpl-1", "tpl-2"}) || len(queries) != 2 {
		t.Fatalf("ids=%v queries=%v", ids, queries)
	}
	if first := queries[0]; first.Get("limit") != "1" || first.Get("sort_key") != "name" || first.Get("sort_dir") != "asc" || first.Get("vendor") != "x" {
		t.Fatalf("first query=%v", first)
	}
	if queries[1].Get("marker") != "tpl-1" {
		t.Fatalf("second query=%v", queries[1])
	}
	if value, err := api.Find(ctx, resource.Name("two")); err != nil || value == nil || value.UUID != "tpl-2" {
		t.Fatalf("find by name: %+v %v", value, err)
	}
	if _, err := api.Find(ctx, resource.Name("none")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing name: %v", err)
	}
	if _, err := api.Resources.Get(ctx, "missing"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("Collection.Get maps 404 to ErrNotFound: %v", err)
	}
}

func TestMagnumTemplateUpdateSendsPatchArrayAndAccepts200Or202(t *testing.T) {
	for _, status := range []int{200, 202, 201, 204} {
		cloud := testcloud.New(t)
		var raw string
		cloud.Mux.HandleFunc("PATCH /v1/clustertemplates/tpl-1", func(w http.ResponseWriter, r *http.Request) {
			raw = string(magnumRawBody(t, r))
			if status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, status, magnumTemplateBody)
		})
		api, _, _, _ := newMagnumPieces(t, cloud, "")
		value, err := api.Update(context.Background(), "tpl-1", []clustertemplates.UpdateOpts{
			{Op: clustertemplates.ReplaceOp, Path: "/public", Value: false},
			{Op: clustertemplates.AddOp, Path: "/labels/k", Value: ""},
			{Op: clustertemplates.RemoveOp, Path: "/labels/old"},
		})
		success := status == 200 || status == 202
		if success != (err == nil) || (success && (value == nil || value.UUID != "tpl-1")) {
			t.Fatalf("status %d value=%+v error=%v", status, value, err)
		}
		if !success && magnumUnexpected(err) != status {
			t.Fatalf("status %d error=%v", status, err)
		}
		want := `[{"op":"replace","path":"/public","value":false},{"op":"add","path":"/labels/k","value":""},{"op":"remove","path":"/labels/old"}]`
		if raw != want {
			t.Fatalf("body=%s", raw)
		}
	}
}

func TestMagnumTemplateDeleteStatusesAndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("DELETE /v1/clustertemplates/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("id") {
		case "missing":
			testcloud.JSON(w, 404, `{}`)
		case "ok200":
			testcloud.JSON(w, 200, `{}`)
		default:
			w.WriteHeader(204)
		}
	})
	api, _, _, _ := newMagnumPieces(t, cloud, "")
	ctx := context.Background()
	if err := api.Delete(ctx, "tpl-1"); err != nil {
		t.Fatal(err)
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
	if err := api.Remove(ctx, resource.ID("missing"), resource.WithIgnoreMissing()); err != nil {
		t.Fatalf("explicit ignore-missing: %v", err)
	}
}

func TestMagnumTemplateCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	api, _, _, _ := newMagnumPieces(t, cloud, "")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Create(canceled, clustertemplates.CreateOpts{COE: "k", ImageID: "i"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("create: %v", err)
	}
	if _, err := api.Get(canceled, "tpl-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("get: %v", err)
	}
	if _, err := api.Update(canceled, "tpl-1", []clustertemplates.UpdateOpts{{Op: clustertemplates.ReplaceOp, Path: "/name", Value: "x"}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("update: %v", err)
	}
	if err := api.Delete(canceled, "tpl-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("delete: %v", err)
	}
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Get(deadline, "tpl-1"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}

// ---- node groups ----

const magnumNodeGroupBody = `{"uuid":"ng-1","name":"workers","cluster_id":"cl-1","role":"worker","node_count":0,"min_node_count":0,"max_node_count":0,"docker_volume_size":null,"is_default":false,"status":"CREATE_IN_PROGRESS","created_at":"2026-01-02T03:04:05+00:00"}`

func TestMagnumNodeGroupCreateKeepsZeroFalseAndRequires202(t *testing.T) {
	for _, status := range []int{202, 200, 201} {
		cloud := testcloud.New(t)
		var bodies []map[string]string
		cloud.Mux.HandleFunc("POST /v1/clusters/cl-1/nodegroups", func(w http.ResponseWriter, r *http.Request) {
			bodies = append(bodies, magnumRawFields(t, r))
			testcloud.JSON(w, status, magnumNodeGroupBody)
		})
		_, api, _, _ := newMagnumPieces(t, cloud, "1.9")
		scope, err := api.InCluster(context.Background(), resource.ID("cl-1"))
		if err != nil {
			t.Fatal(err)
		}
		value, err := scope.Create(context.Background(), nodegroups.CreateOpts{
			Name: "workers", NodeCount: magnumIntPtr(0), MaxNodeCount: magnumIntPtr(0), DockerVolumeSize: magnumIntPtr(0), MergeLabels: magnumFalse(),
		}, nodegroups.WithCreateField("vendor", false))
		if (status == 202) != (err == nil) || (status == 202 && (value == nil || value.UUID != "ng-1" || value.MaxNodeCount == nil || *value.MaxNodeCount != 0 || value.DockerVolumeSize != nil)) {
			t.Fatalf("status %d value=%+v error=%v", status, value, err)
		}
		if status != 202 && magnumUnexpected(err) != status {
			t.Fatalf("status %d error=%v", status, err)
		}
		want := map[string]string{"name": `"workers"`, "node_count": `0`, "max_node_count": `0`, "docker_volume_size": `0`, "merge_labels": `false`, "vendor": `false`}
		if !reflect.DeepEqual(bodies[0], want) {
			t.Fatalf("body=%v", bodies[0])
		}
		if status == 202 {
			// A plain-int MinNodeCount of zero is omitted by the native builder.
			if _, err := scope.Create(context.Background(), nodegroups.CreateOpts{Name: "min", MinNodeCount: 0}); err != nil {
				t.Fatal(err)
			}
			if want := map[string]string{"name": `"min"`}; !reflect.DeepEqual(bodies[1], want) {
				t.Fatalf("minimal body=%v", bodies[1])
			}
		}
	}
}

func TestMagnumNodeGroupRejectsInvalidInputsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	_, api, _, _ := newMagnumPieces(t, cloud, "")
	ctx := context.Background()
	if _, err := api.Create(ctx, "cl-1", nodegroups.CreateOpts{}); err == nil {
		t.Fatal("missing name must fail locally")
	}
	for _, key := range []string{"name", "node_count", "max_node_count", "role"} {
		if _, err := api.Create(ctx, "cl-1", nodegroups.CreateOpts{Name: "n"}, nodegroups.WithCreateField(key, 1)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("extension %q: %v", key, err)
		}
	}
	if _, err := api.Create(ctx, "cl-1", nodegroups.CreateOpts{Name: "n"}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	if _, err := api.Update(ctx, "cl-1", "ng-1", []nodegroups.UpdateOpts{{Path: "/node_count"}}); err == nil {
		t.Fatal("missing patch op must fail locally")
	}
	for _, err := range api.List(ctx, "cl-1", nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list option: %v", err)
		}
	}
	scope, err := api.InCluster(ctx, resource.ID("cl-1"))
	if err != nil {
		t.Fatalf("explicit parent IDs need no lookup request: %v", err)
	}
	if _, err := scope.Create(ctx, nodegroups.CreateOpts{Name: "n"}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("scope nil option: %v", err)
	}
}

func TestMagnumNodeGroupGetListUpdateDeleteUseClusterRoutes(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	var patch string
	cloud.Mux.HandleFunc("GET /v1/clusters/cl-1/nodegroups/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "ng-1" {
			testcloud.JSON(w, 200, magnumNodeGroupBody)
			return
		}
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v1/clusters/cl-1/nodegroups", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		if r.URL.Query().Get("marker") == "7" {
			testcloud.JSON(w, 200, `{"nodegroups":[{"uuid":"ng-2","name":"two"}]}`)
			return
		}
		next := cloud.Server.URL + "/v1/clusters/cl-1/nodegroups?marker=7&limit=1"
		testcloud.JSON(w, 200, `{"nodegroups":[{"uuid":"ng-1","name":"workers"}],"next":"`+next+`"}`)
	})
	cloud.Mux.HandleFunc("PATCH /v1/clusters/cl-1/nodegroups/ng-1", func(w http.ResponseWriter, r *http.Request) {
		patch = string(magnumRawBody(t, r))
		testcloud.JSON(w, 202, magnumNodeGroupBody)
	})
	cloud.Mux.HandleFunc("PATCH /v1/clusters/cl-1/nodegroups/ng-200", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, magnumNodeGroupBody)
	})
	cloud.Mux.HandleFunc("DELETE /v1/clusters/cl-1/nodegroups/{id}", func(w http.ResponseWriter, r *http.Request) {
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
	_, api, _, _ := newMagnumPieces(t, cloud, "")
	ctx := context.Background()
	value, err := api.Get(ctx, "cl-1", "ng-1")
	if err != nil || value.UUID != "ng-1" || value.IsDefault || value.NodeCount != 0 {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if _, err := api.Get(ctx, "cl-1", "missing"); magnumUnexpected(err) != 404 || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	var ids []string
	for value, err := range api.List(ctx, "cl-1", nodegroups.WithListOptions(nodegroups.ListOpts{Limit: 1, Role: "worker", SortKey: "name", SortDir: "desc"}), nodegroups.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.UUID)
	}
	if !reflect.DeepEqual(ids, []string{"ng-1", "ng-2"}) || len(queries) != 2 {
		t.Fatalf("ids=%v queries=%v", ids, queries)
	}
	if first := queries[0]; first.Get("limit") != "1" || first.Get("role") != "worker" || first.Get("sort_key") != "name" || first.Get("sort_dir") != "desc" || first.Get("vendor") != "x" || first.Has("marker") {
		t.Fatalf("first query=%v", first)
	}
	if queries[1].Get("marker") != "7" {
		t.Fatalf("second query=%v", queries[1])
	}
	updated, err := api.Update(ctx, "cl-1", "ng-1", []nodegroups.UpdateOpts{{Op: nodegroups.ReplaceOp, Path: "/node_count", Value: 0}, {Op: nodegroups.RemoveOp, Path: "/labels/old"}})
	if err != nil || updated.UUID != "ng-1" || patch != `[{"op":"replace","path":"/node_count","value":0},{"op":"remove","path":"/labels/old"}]` {
		t.Fatalf("updated=%+v patch=%s error=%v", updated, patch, err)
	}
	if _, err := api.Update(ctx, "cl-1", "ng-200", []nodegroups.UpdateOpts{{Op: nodegroups.ReplaceOp, Path: "/node_count", Value: 1}}); magnumUnexpected(err) != 200 {
		t.Fatalf("update accepts only 202: %v", err)
	}
	for _, id := range []string{"a", "ok202"} {
		if err := api.Delete(ctx, "cl-1", id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	if err := api.Delete(ctx, "cl-1", "ok200"); magnumUnexpected(err) != 200 {
		t.Fatalf("200 is not an accepted delete code: %v", err)
	}
	if err := api.Delete(ctx, "cl-1", "missing"); magnumUnexpected(err) != 404 || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native delete keeps the 404 response error: %v", err)
	}
	scope, err := api.InCluster(ctx, resource.ID("cl-1"))
	if err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("scope delete ignores missing by default: %v", err)
	}
	if value, err := scope.Find(ctx, resource.Name("two")); err != nil || value == nil || value.UUID != "ng-2" {
		t.Fatalf("scope find: %+v %v", value, err)
	}
	if _, err := scope.Find(ctx, resource.Name("none")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("scope missing name: %v", err)
	}
}

func TestMagnumNodeGroupCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	_, api, _, _ := newMagnumPieces(t, cloud, "")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Create(canceled, "cl-1", nodegroups.CreateOpts{Name: "n"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("create: %v", err)
	}
	if _, err := api.Get(canceled, "cl-1", "ng-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("get: %v", err)
	}
	if _, err := api.Update(canceled, "cl-1", "ng-1", []nodegroups.UpdateOpts{{Op: nodegroups.ReplaceOp, Path: "/node_count", Value: 1}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("update: %v", err)
	}
	if err := api.Delete(canceled, "cl-1", "ng-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("delete: %v", err)
	}
	for _, err := range api.List(canceled, "cl-1") {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, "cl-1", nodegroups.CreateOpts{Name: "n"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("an uncertain create must not be replayed: %d calls", calls.Load())
	}
}

// ---- certificates ----

func TestMagnumCertificateCreateGetAndUpdateStatusContracts(t *testing.T) {
	cloud := testcloud.New(t)
	var bodies []map[string]string
	var updateBody []byte
	var updateCalls atomic.Int32
	var version string
	cloud.Mux.HandleFunc("POST /v1/certificates", func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, magnumRawFields(t, r))
		version = r.Header.Get("OpenStack-API-Version")
		switch r.URL.Query().Get("status") {
		case "200":
			testcloud.JSON(w, 200, `{}`)
		default:
			testcloud.JSON(w, 201, `{"cluster_uuid":"cl-1","pem":"PEM","csr":"CSR"}`)
		}
	})
	cloud.Mux.HandleFunc("GET /v1/certificates/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "missing" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		testcloud.JSON(w, 200, `{"cluster_uuid":"cl-1","pem":"PEM","vendor_extra":1}`)
	})
	cloud.Mux.HandleFunc("PATCH /v1/certificates/{id}", func(w http.ResponseWriter, r *http.Request) {
		updateCalls.Add(1)
		updateBody, _ = io.ReadAll(r.Body)
		switch r.PathValue("id") {
		case "ok":
			testcloud.JSON(w, 202, `{}`)
		case "empty202":
			w.WriteHeader(202)
		case "ok200":
			w.WriteHeader(200)
		case "ok204":
			w.WriteHeader(204)
		default:
			testcloud.JSON(w, 404, `{}`)
		}
	})
	_, _, api, _ := newMagnumPieces(t, cloud, "1.10")
	ctx := context.Background()
	cert, err := api.Create(ctx, certificates.CreateOpts{ClusterUUID: "cl-1", CSR: "CSR"}, certificates.WithCreateField("vendor", false))
	if err != nil || cert == nil || cert.PEM != "PEM" || cert.ClusterUUID != "cl-1" {
		t.Fatalf("cert=%+v error=%v", cert, err)
	}
	if want := map[string]string{"cluster_uuid": `"cl-1"`, "csr": `"CSR"`, "vendor": `false`}; !reflect.DeepEqual(bodies[0], want) || version != "container-infra 1.10" {
		t.Fatalf("body=%v version=%q", bodies[0], version)
	}
	if _, err := api.Create(ctx, certificates.CreateOpts{BayUUID: "bay-1", CSR: "CSR"}); err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"bay_uuid": `"bay-1"`, "csr": `"CSR"`}; !reflect.DeepEqual(bodies[1], want) {
		t.Fatalf("bay body=%v", bodies[1])
	}
	got, err := api.Get(ctx, "cl-1")
	if err != nil || got.PEM != "PEM" {
		t.Fatalf("get=%+v error=%v", got, err)
	}
	if _, err := api.Get(ctx, "missing"); magnumUnexpected(err) != 404 || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	if err := api.Update(ctx, "ok"); err != nil || len(updateBody) != 0 {
		t.Fatalf("update error=%v body=%q", err, updateBody)
	}
	// An empty 202 body cannot be decoded by the native call, so the acknowledged
	// rotation surfaces as an error; callers must not blindly replay it.
	if err := api.Update(ctx, "empty202"); !errors.Is(err, io.EOF) {
		t.Fatalf("empty 202 body: %v", err)
	}
	for _, id := range []string{"ok200", "ok204", "missing"} {
		want := map[string]int{"ok200": 200, "ok204": 204, "missing": 404}[id]
		if err := api.Update(ctx, id); magnumUnexpected(err) != want {
			t.Fatalf("update %s accepts only 202: %v", id, err)
		}
	}
	if updateCalls.Load() != 5 {
		t.Fatalf("update calls=%d", updateCalls.Load())
	}
}

func TestMagnumCertificateRejectsInvalidInputsAndOtherStatusesBeforeReplay(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v1/certificates", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 202, `{}`)
	})
	_, _, api, _ := newMagnumPieces(t, cloud, "")
	ctx := context.Background()
	for name, opts := range map[string]certificates.CreateOpts{
		"missing csr":    {ClusterUUID: "cl-1"},
		"both ids":       {ClusterUUID: "cl-1", BayUUID: "bay-1", CSR: "c"},
		"neither id":     {CSR: "c"},
		"empty everyone": {},
	} {
		if _, err := api.Create(ctx, opts); err == nil {
			t.Fatalf("%s must fail locally", name)
		}
	}
	for _, key := range []string{"csr", "cluster_uuid", "bay_uuid"} {
		if _, err := api.Create(ctx, certificates.CreateOpts{ClusterUUID: "cl-1", CSR: "c"}, certificates.WithCreateField(key, "x")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("extension %q: %v", key, err)
		}
	}
	if _, err := api.Create(ctx, certificates.CreateOpts{ClusterUUID: "cl-1", CSR: "c"}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid input reached HTTP: %d", calls.Load())
	}
	if _, err := api.Create(ctx, certificates.CreateOpts{ClusterUUID: "cl-1", CSR: "c"}); magnumUnexpected(err) != 202 {
		t.Fatalf("create requires 201: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestMagnumCertificateCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	_, _, api, _ := newMagnumPieces(t, cloud, "")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Create(canceled, certificates.CreateOpts{ClusterUUID: "cl-1", CSR: "c"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("create: %v", err)
	}
	if _, err := api.Get(canceled, "cl-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("get: %v", err)
	}
	if err := api.Update(canceled, "cl-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("update: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if err := api.Update(deadline, "cl-1"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("an uncertain rotation must not be replayed: %d calls", calls.Load())
	}
}

// ---- API versions ----

func TestMagnumAPIVersionsListGetUseServiceRoot(t *testing.T) {
	cloud := testcloud.New(t)
	var paths []string
	var versions []string
	cloud.Mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		versions = append(versions, r.Header.Get("OpenStack-API-Version"))
		testcloud.JSON(w, 200, `{"versions":[{"id":"v1","status":"CURRENT","max_version":"1.10","min_version":"1.1"},{"id":"v2","status":"EXPERIMENTAL","max_version":"2.0","min_version":"2.0"}]}`)
	})
	cloud.Mux.HandleFunc("GET /v1/{$}", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		testcloud.JSON(w, 200, `{"versions":[{"id":"v1","status":"CURRENT","max_version":"1.10","min_version":"1.1"}]}`)
	})
	cloud.Mux.HandleFunc("GET /none/{$}", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"versions":[]}`)
	})
	cloud.Mux.HandleFunc("GET /dup/{$}", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"versions":[{"id":"a"},{"id":"b"}]}`)
	})
	cloud.Mux.HandleFunc("GET /missing/{$}", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	_, _, _, api := newMagnumPieces(t, cloud, "1.10")
	ctx := context.Background()
	var ids []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID+":"+value.Status+":"+value.Version+":"+value.MinVersion)
	}
	if !reflect.DeepEqual(ids, []string{"v1:CURRENT:1.10:1.1", "v2:EXPERIMENTAL:2.0:2.0"}) {
		t.Fatalf("ids=%v", ids)
	}
	value, err := api.Get(ctx, "v1")
	if err != nil || value.ID != "v1" || value.Version != "1.10" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if !reflect.DeepEqual(paths, []string{"/", "/v1/"}) || versions[0] != "container-infra 1.10" {
		t.Fatalf("paths=%v versions=%v", paths, versions)
	}
	var notFound apiversions.ErrVersionNotFound
	if _, err := api.Get(ctx, "none"); !errors.As(err, &notFound) {
		t.Fatalf("empty versions list: %v", err)
	}
	var multiple apiversions.ErrMultipleVersionsFound
	if _, err := api.Get(ctx, "dup"); !errors.As(err, &multiple) || multiple.Count != 2 {
		t.Fatalf("multiple versions: %v", err)
	}
	if _, err := api.Get(ctx, "missing"); magnumUnexpected(err) != 404 || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	if found, err := api.Find(ctx, resource.ID("v1")); err != nil || found == nil || found.ID != "v1" {
		t.Fatalf("collection find by ID: %+v %v", found, err)
	}
	if _, err := api.Find(ctx, resource.ID("missing")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("collection maps 404 to ErrNotFound: %v", err)
	}
}

func TestMagnumAPIVersionsCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	_, _, _, api := newMagnumPieces(t, cloud, "")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(canceled, "v1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("get: %v", err)
	}
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Get(deadline, "v1"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
