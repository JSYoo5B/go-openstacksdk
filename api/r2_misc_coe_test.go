package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	containerinfra "github.com/JSYoo5B/go-openstacksdk/containerinfra/v1"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func r2MiscCOEService(t *testing.T) (*containerinfra.Service, *testcloud.Cloud, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var clusterLists, templateLists atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/clusters", func(w http.ResponseWriter, r *http.Request) {
		clusterLists.Add(1)
		if r.URL.RawQuery != "" {
			t.Errorf("unexpected cluster query %q", r.URL.RawQuery)
		}
		testcloud.JSON(w, 200, `{"clusters":[{"uuid":"cl-1","name":"prod-a","status":"CREATE_COMPLETE","node_count":3,"floating_ip_enabled":true,"labels":{"k":"v"}},{"uuid":"cl-2","name":"prod-b","status":"CREATE_FAILED","node_count":1},{"uuid":"cl-3","name":"dev","status":"CREATE_COMPLETE"}]}`)
	})
	cloud.Mux.HandleFunc("GET /v1/clustertemplates", func(w http.ResponseWriter, r *http.Request) {
		templateLists.Add(1)
		testcloud.JSON(w, 200, `{"clustertemplates":[{"uuid":"t-1","name":"k8s","coe":"kubernetes","public":true},{"uuid":"t-2","name":"k8s","coe":"swarm"},{"uuid":"t-3","name":"other","coe":"kubernetes"}]}`)
	})
	return containerinfra.New(cloud.Client("container-infra", "/v1")), cloud, &clusterLists, &templateLists
}

func r2MiscClusterIDs(found *containerinfra.ClusterSearch) []string {
	var ids []string
	for _, c := range found.Clusters {
		ids = append(ids, c.UUID)
	}
	return ids
}

func TestR2MiscCOESearchClustersGlobFiltersAndExpressions(t *testing.T) {
	service, _, lists, _ := r2MiscCOEService(t)
	ctx := context.Background()
	found, err := service.SearchCOEClusters(ctx, "prod-*")
	if err != nil || len(found.Clusters) != 2 || lists.Load() != 1 {
		t.Fatalf("glob %v %v lists=%d", err, r2MiscClusterIDs(found), lists.Load())
	}
	if found, err = service.SearchCOEClusters(ctx, "cl-3"); err != nil || len(found.Clusters) != 1 || found.Clusters[0].Name != "dev" {
		t.Fatalf("by id %v %v", err, found)
	}
	if found, err = service.SearchCOEClusters(ctx, ""); err != nil || len(found.Clusters) != 3 {
		t.Fatalf("all %v", err)
	}
	filter := func(s string) containerinfra.SearchOption {
		return containerinfra.WithSearchFilters(json.RawMessage(s))
	}
	if found, err = service.SearchCOEClusters(ctx, "prod-*", filter(`{"status":"CREATE_COMPLETE"}`)); err != nil || len(found.Clusters) != 1 || found.Clusters[0].UUID != "cl-1" {
		t.Fatalf("status filter %v %v", err, r2MiscClusterIDs(found))
	}
	if found, err = service.SearchCOEClusters(ctx, "", filter(`{"is_floating_ip_enabled":true,"labels":{"k":"v"}}`)); err != nil || len(found.Clusters) != 1 || found.Clusters[0].UUID != "cl-1" {
		t.Fatalf("python attribute names %v %v", err, r2MiscClusterIDs(found))
	}
	found, err = service.SearchCOEClusters(ctx, "", filter(`"[?node_count>=`+"`2`"+`].name"`))
	if err != nil || !found.Expression || string(found.Value) != `["prod-a"]` || len(found.Clusters) != 0 {
		t.Fatalf("expression %v %+v", err, found)
	}
	if _, err = service.SearchCOEClusters(ctx, "", filter(`{`)); err == nil {
		t.Fatal("malformed filters accepted")
	}
	if _, err = service.SearchCOEClusters(ctx, "", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option %v", err)
	}
}

func TestR2MiscCOEGetClusterAmbiguityAndErrors(t *testing.T) {
	service, cloud, _, _ := r2MiscCOEService(t)
	ctx := context.Background()
	one, err := service.GetCOECluster(ctx, "dev")
	if err != nil || one == nil || one.UUID != "cl-3" {
		t.Fatalf("one %v %v", one, err)
	}
	if none, err := service.GetCOECluster(ctx, "missing"); err != nil || none != nil {
		t.Fatalf("none %v %v", none, err)
	}
	var ambiguous *resource.AmbiguousError
	if _, err := service.GetCOECluster(ctx, "prod-*"); !errors.As(err, &ambiguous) || len(ambiguous.IDs) != 2 {
		t.Fatalf("ambiguous %v", err)
	}
	if got, err := service.GetCOECluster(ctx, "prod-*", containerinfra.WithSearchFilters(json.RawMessage(`{"status":"CREATE_FAILED"}`))); err != nil || got == nil || got.UUID != "cl-2" {
		t.Fatalf("filtered %v %v", got, err)
	}
	if _, err := service.GetCOECluster(ctx, "", containerinfra.WithSearchFilters(json.RawMessage(`"[].name"`))); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("expression %v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := service.GetCOECluster(canceled, "dev"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled %v", err)
	}
	cloud.Mux.HandleFunc("GET /v1/failing", func(http.ResponseWriter, *http.Request) {})
}

func TestR2MiscCOESearchAndGetClusterTemplates(t *testing.T) {
	service, cloud, _, lists := r2MiscCOEService(t)
	ctx := context.Background()
	found, err := service.SearchClusterTemplates(ctx, "k8s*")
	if err != nil || len(found.Templates) != 2 || lists.Load() != 1 {
		t.Fatalf("glob %v %+v", err, found)
	}
	found, err = service.SearchClusterTemplates(ctx, "", containerinfra.WithSearchFilters(json.RawMessage(`{"is_public":true}`)))
	if err != nil || len(found.Templates) != 1 || found.Templates[0].UUID != "t-1" {
		t.Fatalf("public %v %+v", err, found)
	}
	found, err = service.SearchClusterTemplates(ctx, "", containerinfra.WithSearchFilters(json.RawMessage(`"[?coe=='kubernetes'].uuid"`)))
	if err != nil || !found.Expression || string(found.Value) != `["t-1","t-3"]` {
		t.Fatalf("expression %v %+v", err, found)
	}
	var ambiguous *resource.AmbiguousError
	if _, err = service.GetClusterTemplate(ctx, "k8s", containerinfra.WithSearchFilters(json.RawMessage(`{}`))); !errors.As(err, &ambiguous) {
		t.Fatalf("empty filters still use the search path: %v", err)
	}
	if _, err = service.GetClusterTemplate(ctx, "k8s", containerinfra.WithSearchFilters(json.RawMessage(`{"coe":"kubernetes","name":"k8s"}`))); err != nil {
		t.Fatal(err)
	}
	if _, err = service.GetClusterTemplate(ctx, "k8s", containerinfra.WithSearchFilters(json.RawMessage(`{"name":"k8s"}`))); !errors.As(err, &ambiguous) {
		t.Fatalf("filtered ambiguity %v", err)
	}
	if got, err := service.GetClusterTemplate(ctx, "k8s", containerinfra.WithSearchFilters(json.RawMessage(`{"coe":"swarm"}`))); err != nil || got == nil || got.UUID != "t-2" {
		t.Fatalf("filtered %v %v", got, err)
	}
	if got, err := service.GetClusterTemplate(ctx, "k8s", containerinfra.WithSearchFilters(json.RawMessage(`{"coe":"none"}`))); err != nil || got != nil {
		t.Fatalf("no match %v %v", got, err)
	}
	// Without filters the proxy find runs: GET by ID first, then exact list match.
	var members atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/clustertemplates/{id}", func(w http.ResponseWriter, r *http.Request) {
		members.Add(1)
		if r.PathValue("id") == "t-3" {
			testcloud.JSON(w, 200, `{"uuid":"t-3","name":"other"}`)
			return
		}
		testcloud.JSON(w, 404, `{}`)
	})
	if got, err := service.GetClusterTemplate(ctx, "t-3"); err != nil || got == nil || got.UUID != "t-3" || members.Load() != 1 {
		t.Fatalf("by id %v %v", got, err)
	}
	if got, err := service.GetClusterTemplate(ctx, "other"); err != nil || got == nil || got.UUID != "t-3" {
		t.Fatalf("by name %v %v", got, err)
	}
	if _, err := service.GetClusterTemplate(ctx, "k8s"); !errors.As(err, &ambiguous) {
		t.Fatalf("duplicate names %v", err)
	}
	if got, err := service.GetClusterTemplate(ctx, "absent"); err != nil || got != nil {
		t.Fatalf("absent %v %v", got, err)
	}
	if _, err := service.GetClusterTemplate(ctx, ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("empty %v", err)
	}
}
