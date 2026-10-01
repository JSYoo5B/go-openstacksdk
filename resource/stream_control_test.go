package resource_test

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/attachinterfaces"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func collectControlledServers(t *testing.T, stream iter.Seq2[*servers.Server, error]) ([]string, error) {
	t.Helper()
	ids := make([]string, 0)
	var failure error
	for value, err := range stream {
		if err != nil {
			if value != nil || failure != nil {
				t.Fatal("expected one terminal error with nil row", value, failure, err)
			}
			failure = err
			continue
		}
		if failure != nil || value == nil {
			t.Fatal("row after terminal error or empty successful row", value, failure)
		}
		ids = append(ids, value.ID)
	}
	return ids, failure
}

func TestStreamControlLinkedPagesCapRawRowsBeforeOuterFilters(t *testing.T) {
	for _, tc := range []struct {
		maximum int
		want    []string
		calls   int32
	}{
		{0, []string{"one", "two", "three", "four"}, 2},
		{1, []string{"one"}, 1},
		{2, []string{"one", "two"}, 1},
		{3, []string{"one", "two", "three"}, 2},
	} {
		t.Run(fmt.Sprint(tc.maximum), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /nova/servers/detail", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("limit") != "7" || r.URL.Query().Has("max_items") || r.URL.Query().Has("paginated") || r.Header.Get("X-Page-Proof") != "kept" {
					t.Error("controls changed explicit query or pager headers", r.URL, r.Header)
				}
				if r.URL.Query().Get("marker") == "two" {
					testcloud.JSON(w, 200, `{"servers":[{"id":"three","name":"wanted"},{"id":"four","name":"wanted"}]}`)
					return
				}
				next := cloud.Server.URL + "/nova/servers/detail?limit=7&marker=two"
				testcloud.JSON(w, 200, fmt.Sprintf(`{"servers":[{"id":"one","name":"other"},{"id":"two","name":"other"}],"servers_links":[{"rel":"next","href":%q}]}`, next))
			})
			pager := servers.List(cloud.Client("compute", "/nova"), servers.ListOpts{Limit: 7})
			pager.Headers = map[string]string{"X-Page-Proof": "kept"}
			rows, err := collectControlledServers(t, resource.StreamWithControl(context.Background(), pager, servers.ExtractServers, resource.ListControl{MaxItems: tc.maximum}))
			if err != nil || !reflect.DeepEqual(rows, tc.want) || calls.Load() != tc.calls {
				t.Fatal(rows, err, calls.Load())
			}
		})
	}
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /nova/servers/detail", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.RawQuery != "" {
			t.Error("local cap inferred a wire limit", r.URL)
		}
		testcloud.JSON(w, 200, `{"servers":[{"id":"other","name":"other"},{"id":"wanted","name":"wanted"}]}`)
	})
	matched := 0
	for value, err := range resource.StreamWithControl(context.Background(), servers.List(cloud.Client("compute", "/nova"), nil), servers.ExtractServers, resource.ListControl{MaxItems: 1}) {
		if err != nil {
			t.Fatal(err)
		}
		if value.Name == "wanted" {
			matched++
		}
	}
	if matched != 0 || calls.Load() != 1 {
		t.Fatal("outer filter refilled raw cap", matched, calls.Load())
	}
}

func TestStreamControlNativeMarkerAndSinglePagesPreserveWireContracts(t *testing.T) {
	for _, tc := range []struct {
		name    string
		control resource.ListControl
		count   int
		calls   int32
	}{
		{"default", resource.ListControl{}, 3, 3},
		{"exact-cap", resource.ListControl{MaxItems: 2}, 2, 1},
		{"cross-page-cap", resource.ListControl{MaxItems: 3}, 3, 2},
		{"first-page", resource.ListControl{SinglePage: true}, 2, 1},
	} {
		t.Run("marker/"+tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /swift/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("limit") != "8" || r.URL.Query().Get("prefix") != "keep" || r.URL.Query().Has("max_items") || r.URL.Query().Has("paginated") {
					t.Error("marker query changed", r.URL)
				}
				switch r.URL.Query().Get("marker") {
				case "":
					testcloud.JSON(w, 200, `[{"name":"keep-one","count":1},{"name":"keep-two","count":2}]`)
				case "keep-two":
					testcloud.JSON(w, 200, `[{"name":"keep-three","count":3}]`)
				case "keep-three":
					testcloud.JSON(w, 200, `[]`)
				default:
					t.Error("unexpected marker", r.URL)
				}
			})
			count := 0
			pager := containers.List(cloud.Client("object-store", "/swift"), containers.ListOpts{Limit: 8, Prefix: "keep"})
			for value, err := range resource.StreamWithControl(context.Background(), pager, containers.ExtractInfo, tc.control) {
				if err != nil || value.Count != int64(count+1) {
					t.Fatal(value, err, count)
				}
				count++
			}
			if count != tc.count || calls.Load() != tc.calls {
				t.Fatal(count, calls.Load())
			}
		})
	}
	for _, maximum := range []int{0, 1} {
		t.Run(fmt.Sprint("single/", maximum), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /nova/servers/fixed/os-interface", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error("single-page endpoint acquired query", r.URL)
				}
				testcloud.JSON(w, 200, `{"interfaceAttachments":[{"port_id":"one"},{"port_id":"two"}],"links":false}`)
			})
			count := 0
			for value, err := range resource.StreamWithControl(context.Background(), attachinterfaces.List(cloud.Client("compute", "/nova"), "fixed"), attachinterfaces.ExtractInterfaces, resource.ListControl{MaxItems: maximum, SinglePage: true}) {
				if err != nil || value == nil {
					t.Fatal(value, err)
				}
				count++
			}
			want := 2
			if maximum == 1 {
				want = 1
			}
			if count != want || calls.Load() != 1 {
				t.Fatal(count, calls.Load())
			}
		})
	}
}

func TestStreamControlSinglePageAndExactCapSkipBrokenContinuation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		control resource.ListControl
		failed  bool
	}{
		{"default", resource.ListControl{}, true},
		{"first-page", resource.ListControl{SinglePage: true}, false},
		{"exact-cap", resource.ListControl{MaxItems: 2}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /nova/servers/detail", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, `{"servers":[{"id":"one"},{"id":"two"}],"servers_links":false}`)
			})
			rows, err := collectControlledServers(t, resource.StreamWithControl(context.Background(), servers.List(cloud.Client("compute", "/nova"), nil), servers.ExtractServers, tc.control))
			if !reflect.DeepEqual(rows, []string{"one", "two"}) || (err != nil) != tc.failed || calls.Load() != 1 {
				t.Fatal(rows, err, calls.Load())
			}
		})
	}
}

func TestStreamControlWholePageDecodeFailuresRemainObservable(t *testing.T) {
	for _, stage := range []string{"native-empty-check", "extractor"} {
		t.Run(stage, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, extracts atomic.Int32
			cloud.Mux.HandleFunc("GET /first", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, `{"servers":[{"id":"one"},false],"servers_links":false}`)
			})
			pager := pagination.NewPager(cloud.Client("compute", "/"), cloud.Server.URL+"/first", func(r pagination.PageResult) pagination.Page {
				if stage == "native-empty-check" {
					return servers.ServerPage{LinkedPageBase: pagination.LinkedPageBase{PageResult: r}}
				}
				return valuePage{LinkedPageBase: pagination.LinkedPageBase{PageResult: r}}
			})
			extract := func(page pagination.Page) ([]servers.Server, error) {
				extracts.Add(1)
				var body struct {
					Servers []servers.Server `json:"servers"`
				}
				if stage == "native-empty-check" {
					return servers.ExtractServers(page)
				}
				err := page.(valuePage).ExtractInto(&body)
				return body.Servers, err
			}
			rows, err := collectControlledServers(t, resource.StreamWithControl(context.Background(), pager, extract, resource.ListControl{MaxItems: 1, SinglePage: true}))
			wantExtracts := int32(0)
			if stage == "extractor" {
				wantExtracts = 1
			}
			if len(rows) != 0 || err == nil || calls.Load() != 1 || extracts.Load() != wantExtracts {
				t.Fatal("cap hid whole-page malformed row", rows, err, calls.Load(), extracts.Load())
			}
		})
	}
}

func TestStreamControlLateHTTPFailureKeepsNativeResponseEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /nova/servers/detail", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		next := cloud.Server.URL + "/denied"
		testcloud.JSON(w, 200, fmt.Sprintf(`{"servers":[{"id":"one"}],"servers_links":[{"rel":"next","href":%q}]}`, next))
	})
	cloud.Mux.HandleFunc("GET /denied", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("X-Request-ID", "late-failure")
		testcloud.JSON(w, 403, `{"error":{"reason":"denied"}}`)
	})
	rows, err := collectControlledServers(t, resource.StreamWithControl(context.Background(), servers.List(cloud.Client("compute", "/nova"), nil), servers.ExtractServers, resource.ListControl{MaxItems: 2}))
	var response gophercloud.ErrUnexpectedResponseCode
	if !reflect.DeepEqual(rows, []string{"one"}) || !errors.As(err, &response) || response.Actual != 403 || string(response.Body) != `{"error":{"reason":"denied"}}` || response.ResponseHeader.Get("X-Request-ID") != "late-failure" || calls.Load() != 2 {
		t.Fatal(rows, err, response, calls.Load())
	}
}

func TestStreamControlOptionsAreLazyCopiedAndReusable(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /nova/servers/detail", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"servers":[{"id":"one"},{"id":"two"},{"id":"three"}],"servers_links":false}`)
	})
	pager := servers.List(cloud.Client("compute", "/nova"), nil)
	control := resource.ListControl{MaxItems: 2, SinglePage: true}
	stream := resource.StreamWithControl(context.Background(), pager, servers.ExtractServers, control)
	control.MaxItems, control.SinglePage = 0, false
	negative := resource.StreamWithControl(context.Background(), pager, servers.ExtractServers, resource.ListControl{MaxItems: -1})
	if calls.Load() != 0 {
		t.Fatal("constructed iterator fetched HTTP", calls.Load())
	}
	for range 2 {
		rows, err := collectControlledServers(t, stream)
		if err != nil || !reflect.DeepEqual(rows, []string{"one", "two"}) {
			t.Fatal("control/counter was shared with caller or earlier iteration", rows, err)
		}
	}
	if rows, err := collectControlledServers(t, negative); len(rows) != 0 || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 2 {
		t.Fatal("negative cap reached HTTP", rows, err, calls.Load())
	}
	rows, err := collectControlledServers(t, resource.StreamWithControl(context.Background(), pager, servers.ExtractServers, resource.ListControl{}))
	if !reflect.DeepEqual(rows, []string{"one", "two", "three"}) || err == nil || calls.Load() != 3 {
		t.Fatal("zero controls suppressed existing continuation error", rows, err, calls.Load())
	}
}

func TestStreamControlCancellationBeforeRowsAndAtTerminalBoundaries(t *testing.T) {
	for _, mode := range []string{"before-http", "during-extract", "between-rows", "cap", "single-page", "empty-extraction"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /nova/servers/detail", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, `{"servers":[{"id":"one"},{"id":"two"}],"servers_links":false}`)
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			control := resource.ListControl{}
			if mode == "before-http" {
				cancel()
			}
			if mode == "cap" {
				control.MaxItems = 1
			}
			if mode == "single-page" || mode == "empty-extraction" {
				control.SinglePage = true
			}
			extract := func(page pagination.Page) ([]servers.Server, error) {
				rows, err := servers.ExtractServers(page)
				if mode == "during-extract" || mode == "empty-extraction" {
					cancel()
				}
				if mode == "empty-extraction" {
					rows = nil
				}
				return rows, err
			}
			seen, failures := 0, 0
			for value, err := range resource.StreamWithControl(ctx, servers.List(cloud.Client("compute", "/nova"), nil), extract, control) {
				if err != nil {
					failures++
					if value != nil || !errors.Is(err, context.Canceled) {
						t.Fatal(value, err)
					}
					continue
				}
				seen++
				if mode != "single-page" || seen == 2 {
					cancel()
				}
			}
			wantSeen, wantCalls := 1, int32(1)
			if mode == "before-http" || mode == "during-extract" || mode == "empty-extraction" {
				wantSeen = 0
			}
			if mode == "before-http" {
				wantCalls = 0
			}
			if mode == "single-page" {
				wantSeen = 2
			}
			if seen != wantSeen || failures != 1 || calls.Load() != wantCalls {
				t.Fatal(seen, failures, calls.Load())
			}
		})
	}
}

func TestStreamControlConsumerBreakWinsOverCancellationAndNextLinks(t *testing.T) {
	for _, control := range []resource.ListControl{{}, {MaxItems: 1}, {SinglePage: true}} {
		t.Run(fmt.Sprint(control), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, nextCalls atomic.Int32
			cloud.Mux.HandleFunc("GET /first", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, `{"items":[{"id":"one"},{"id":"two"}]}`)
			})
			pager := pagination.NewPager(cloud.Client("test", "/"), cloud.Server.URL+"/first", func(result pagination.PageResult) pagination.Page {
				return failingNextPage{PageResult: result, calls: &nextCalls}
			})
			extract := func(page pagination.Page) ([]cycleItem, error) {
				var body struct {
					Items []cycleItem `json:"items"`
				}
				err := page.(failingNextPage).ExtractInto(&body)
				return body.Items, err
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			stream := resource.StreamWithControl(ctx, pager, extract, control)
			seen := 0
			stream(func(value *cycleItem, err error) bool {
				seen++
				if value == nil || err != nil || value.ID != "one" {
					t.Fatal(value, err)
				}
				cancel()
				return false
			})
			if seen != 1 || calls.Load() != 1 || nextCalls.Load() != 0 {
				t.Fatal("break evaluated more rows, cancellation error or continuation", seen, calls.Load(), nextCalls.Load())
			}
		})
	}
}
