package resource_test

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"gophercloudsdk/clustering/v1/nodes"
	"gophercloudsdk/compute"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestCollectionListControlsCountBeforeNameAndStatusFilters(t *testing.T) {
	for _, owned := range []bool{false, true} {
		t.Run(fmt.Sprint("owned=", owned), func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			path, envelope := "/nova/v2.1/project/servers/detail", "servers"
			if owned {
				path, envelope = "/senlin/v1/nodes", "nodes"
			}
			cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
				ordinal := requests.Add(1)
				if r.URL.Query().Has("max_items") || r.URL.Query().Has("paginated") {
					t.Error("local controls leaked to HTTP", r.URL)
				}
				if owned && ordinal < 3 && r.URL.Query().Get("limit") != fmt.Sprint(ordinal+1) {
					t.Error("owned limit hint", r.URL)
				}
				if !owned && r.URL.Query().Has("limit") {
					t.Error("native binding inferred a server page option", r.URL)
				}
				testcloud.JSON(w, 200, fmt.Sprintf(`{"%s":[{"id":"wrong-name","name":"other","status":"ACTIVE"},{"id":"wrong-status","name":"wanted","status":"ERROR"},{"id":"match","name":"wanted","status":"ACTIVE"}]}`, envelope))
			})
			checkListControlFilters(t, owned, cloud, &requests)
		})
	}
}

func checkListControlFilters(t *testing.T, owned bool, cloud *testcloud.Cloud, calls *atomic.Int32) {
	t.Helper()
	if owned {
		collection := nodes.New(cloud.Client("clustering", "/senlin/v1")).Resources
		checkListControlRows(t, collection, func(v *nodes.Node) string { return v.ID }, calls)
	} else {
		collection := compute.New(cloud.Client("compute", "/nova/v2.1/project"), compute.Dependencies{}).Servers.Collection
		checkListControlRows(t, collection, func(v *compute.Server) string { return v.ID }, calls)
	}
}

func checkListControlRows[T any](t *testing.T, collection *resource.Collection[T], id func(*T) string, calls *atomic.Int32) {
	t.Helper()
	for _, maximum := range []int{2, 3, 0} {
		rows, err := collection.All(context.Background(), resource.WithName("wanted"), resource.WithStatus("ACTIVE"), resource.WithMaxItems(maximum))
		want := 1
		if maximum == 2 {
			want = 0
		}
		if err != nil || len(rows) != want || (want == 1 && id(rows[0]) != "match") {
			t.Fatal(maximum, rows, err)
		}
	}
	if calls.Load() != 3 {
		t.Fatal("raw cap refilled after filtering", calls.Load())
	}
}

func TestCollectionSinglePageAndCapDoNotEvaluateCyclicContinuation(t *testing.T) {
	for _, owned := range []bool{false, true} {
		for _, capOnly := range []bool{false, true} {
			t.Run(fmt.Sprint("owned=", owned, "/cap=", capOnly), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				path, key := "/nova/v2.1/project/servers/detail", "servers"
				if owned {
					path, key = "/senlin/v1/nodes", "nodes"
				}
				cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					self := cloud.Server.URL + r.URL.RequestURI()
					testcloud.JSON(w, 200, fmt.Sprintf(`{"%s":[{"id":"one","name":"wanted","status":"ACTIVE"}],"%s_links":[{"rel":"next","href":%q}]}`, key, key, self))
				})
				option := resource.WithPaginated(false)
				if capOnly {
					option = resource.WithMaxItems(1)
				}
				if owned {
					collection := nodes.New(cloud.Client("clustering", "/senlin/v1")).Resources
					rows, err := collection.All(context.Background(), option)
					if err != nil || len(rows) != 1 {
						t.Fatal(rows, err)
					}
					if rows, err := collection.All(context.Background()); rows != nil || !errors.Is(err, resource.ErrPaginationCycle) {
						t.Fatal(rows, err)
					}
				} else {
					collection := compute.New(cloud.Client("compute", "/nova/v2.1/project"), compute.Dependencies{}).Servers.Collection
					rows, err := collection.All(context.Background(), option)
					if err != nil || len(rows) != 1 {
						t.Fatal(rows, err)
					}
					if rows, err := collection.All(context.Background()); rows != nil || !errors.Is(err, resource.ErrPaginationCycle) {
						t.Fatal(rows, err)
					}
				}
				if calls.Load() != 2 {
					t.Fatal("evaluated or fetched continuation", calls.Load())
				}
			})
		}
	}
}

func TestCollectionListControlOptionsRemainLazyAndOwned(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"nodes":[{"id":"one"},{"id":"two"},{"id":"three"}]}`)
	})
	collection := nodes.New(cloud.Client("clustering", "/senlin/v1")).Resources
	options := []resource.ListOption{resource.WithMaxItems(1), resource.WithMaxItems(2), resource.WithPaginated(true), resource.WithPaginated(false)}
	stream := collection.List(context.Background(), options...)
	options[1], options[3] = resource.WithMaxItems(0), nil
	if calls.Load() != 0 {
		t.Fatal("eager list", calls.Load())
	}
	for range 2 {
		count := 0
		for value, err := range stream {
			if err != nil || value == nil {
				t.Fatal(value, err)
			}
			count++
		}
		if count != 2 {
			t.Fatal("caller changed saved controls or count survived iteration", count)
		}
	}
	if rows, err := collection.All(context.Background(), resource.WithMaxItems(-1)); rows != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 2 {
		t.Fatal("negative control reached HTTP", rows, err, calls.Load())
	}
	for _, key := range []string{"max_items", "paginated"} {
		if rows, err := collection.All(context.Background(), resource.WithQuery(key, "1")); rows != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 2 {
			t.Fatal("local control bypassed typed options and reached HTTP", key, rows, err, calls.Load())
		}
	}
}

func TestOpaqueIteratorCapsRawRowsAndRejectsUnknownPageBoundaries(t *testing.T) {
	var starts, yielded atomic.Int32
	collection := resource.NewCollection(resource.Adapter[cycleItem]{Kind: "opaque", Name: func(v *cycleItem) string { return v.Name },
		Iterate: func(ctx context.Context, _ url.Values) iter.Seq2[*cycleItem, error] {
			return func(yield func(*cycleItem, error) bool) {
				starts.Add(1)
				for _, name := range []string{"other", "other", "wanted"} {
					yielded.Add(1)
					if !yield(&cycleItem{Name: name}, nil) {
						return
					}
				}
			}
		}})
	rows, err := collection.All(context.Background(), resource.WithName("wanted"), resource.WithMaxItems(2))
	if err != nil || len(rows) != 0 || yielded.Load() != 2 || starts.Load() != 1 {
		t.Fatal(rows, err, yielded.Load(), starts.Load())
	}
	rows, err = collection.All(context.Background(), resource.WithPaginated(false))
	if rows != nil || !errors.Is(err, resource.ErrUnsupported) || starts.Load() != 1 {
		t.Fatal("opaque page boundary was guessed", rows, err, starts.Load())
	}
}

func TestNativeListControlsKeepPageDecodeErrorsAndCancellation(t *testing.T) {
	for _, mode := range []string{"whole-page-decode", "cancel-at-cap", "break-at-cap"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /nova/v2.1/project/servers/detail", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				rows := `{"id":"first"}`
				if mode == "whole-page-decode" {
					rows += `,false`
				}
				testcloud.JSON(w, 200, `{"servers":[`+rows+`],"servers_links":[{"rel":"next","href":"https://foreign.example/unused"}]}`)
			})
			collection := compute.New(cloud.Client("compute", "/nova/v2.1/project"), compute.Dependencies{}).Servers.Collection
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			seen := 0
			var finalErr error
			for value, err := range collection.List(ctx, resource.WithMaxItems(1)) {
				if err != nil {
					finalErr = err
					continue
				}
				if value.ID != "first" {
					t.Fatal(value)
				}
				seen++
				cancel()
				if mode == "break-at-cap" {
					break
				}
			}
			if calls.Load() != 1 {
				t.Fatal("unexpected continuation", calls.Load())
			}
			switch mode {
			case "whole-page-decode":
				if finalErr == nil || seen != 0 {
					t.Fatal("native extraction must retain whole-page decoding", seen, finalErr)
				}
			case "cancel-at-cap":
				if seen != 1 || !errors.Is(finalErr, context.Canceled) {
					t.Fatal(seen, finalErr)
				}
			case "break-at-cap":
				if seen != 1 || finalErr != nil {
					t.Fatal(seen, finalErr)
				}
			}
		})
	}
}
