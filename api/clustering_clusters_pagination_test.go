package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestClusteringClustersContinuationGuardsBreakAndCancellation(t *testing.T) {
	for _, mode := range []string{"follow", "break", "cancel", "cycle", "foreign", "filter-change", "missing-marker"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /v1/clusters", func(w http.ResponseWriter, r *http.Request) {
				count := calls.Add(1)
				w.Header().Set("X-Request-Id", "continuation-evidence")
				if mode == "missing-marker" {
					testcloud.JSON(w, 200, `{"clusters":[{"name":"not-a-marker"}]}`)
					return
				}
				if mode == "cycle" {
					testcloud.JSON(w, 200, `{"clusters":[{"id":"same"}]}`)
					return
				}
				if count == 1 {
					link := "?marker=next"
					if mode == "foreign" {
						link = "https://foreign.invalid/v1/clusters?marker=next"
					}
					if mode == "filter-change" {
						link = "?marker=next&vendor=changed"
					}
					w.Header().Set("Link", "<"+link+">; rel=\"next\"")
					testcloud.JSON(w, 200, `{"clusters":[{"id":"first"}]}`)
					return
				}
				if r.URL.Query().Get("vendor") != "retained" || r.URL.Query().Get("marker") != "next" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"clusters":[]}`)
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var resultErr error
			for _, err := range clusters.New(cloud.Client("clustering", "/v1")).List(ctx, clusters.WithListOptions(clusters.ListOpts{Limit: 5}), clusters.WithListQuery("vendor", "retained")) {
				if err != nil {
					resultErr = err
					break
				}
				if mode == "break" {
					break
				}
				if mode == "cancel" {
					cancel()
				}
			}
			switch mode {
			case "follow":
				if resultErr != nil || calls.Load() != 2 {
					t.Fatal(resultErr, calls.Load())
				}
			case "break":
				if resultErr != nil || calls.Load() != 1 {
					t.Fatal(resultErr, calls.Load())
				}
			case "cancel":
				if !errors.Is(resultErr, context.Canceled) || calls.Load() != 1 {
					t.Fatal(resultErr, calls.Load())
				}
			case "cycle":
				var cycle *resource.PaginationCycleError
				if !errors.As(resultErr, &cycle) || calls.Load() != 2 {
					t.Fatal(resultErr, calls.Load())
				}
			default:
				var responseErr *resource.ResponseError
				if !errors.Is(resultErr, resource.ErrInvalidOption) || !errors.As(resultErr, &responseErr) || responseErr.Header.Get("X-Request-Id") != "continuation-evidence" || calls.Load() != 1 {
					t.Fatal(resultErr, calls.Load())
				}
			}
		})
	}
}

func TestClusteringClustersLocalFilterJSONTypesAndNull(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v1/clusters", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"clusters":[{"id":"match","metadata":{"flag":false,"array":[{"x":1}],"nested":{"key":null,"extra":true}}},{"id":"array-extra","metadata":{"flag":false,"array":[{"x":1,"extra":2}],"nested":{"key":null}}},{"id":"wrong-type","metadata":{"flag":0,"array":[{"x":1}],"nested":{"key":null}}},{"id":"empty","metadata":{}},{"id":"null","metadata":null},{"id":"missing"}]}`)
	})
	api := clusters.New(cloud.Client("clustering", "/v1"))
	values, err := api.All(context.Background(), clusters.WithListFilter("metadata", map[string]any{"flag": false, "array": []map[string]any{{"x": 1.0}}, "nested": map[string]any{"key": nil}}))
	if err != nil || len(values) != 1 || values[0].ID != "match" {
		t.Fatal(values, err)
	}
	values, err = api.All(context.Background(), clusters.WithListFilter("metadata", nil))
	if err != nil || len(values) != 2 || values[0].ID != "null" || values[1].ID != "missing" {
		t.Fatal(values, err)
	}
}

func TestClusteringClustersSharedStatusAndSortUseActualWireQuery(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/clusters", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("status") != "ACTIVE" || r.URL.Query().Get("sort") != "vendor_field:desc" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"clusters":[{"id":"active","status":"active"},{"id":"other","status":"ERROR"}]}`)
	})
	api := clusters.New(cloud.Client("clustering", "/v1"))
	values, err := api.Resources.All(context.Background(), resource.WithStatus("ACTIVE"), resource.WithQuery("sort", "vendor_field:desc"))
	if err != nil || len(values) != 1 || values[0].ID != "active" || calls.Load() != 1 {
		t.Fatal(values, err, calls.Load())
	}
	if _, err := api.Resources.All(context.Background(), resource.WithQuery("sort", "name:wrong")); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
		t.Fatal(err, calls.Load())
	}
}
