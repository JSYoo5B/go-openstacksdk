package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/deployables"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/deviceprofiles"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/devices"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type cyborgCloudList struct {
	name string
	path string
	body string
	all  func(*testcloud.Cloud, context.Context, ...resource.ListOption) (int, error)
}

func cyborgCloudLists() []cyborgCloudList {
	return []cyborgCloudList{
		{"deployables", "/v2/deployables", `{"deployables":[{"uuid":"d1","name":"dep","parent_id":null,"num_accelerators":0}]}`,
			func(c *testcloud.Cloud, ctx context.Context, o ...resource.ListOption) (int, error) {
				v, err := deployables.New(c.Client("accelerator", "/v2")).All(ctx, o...)
				return len(v), err
			}},
		{"devices", "/v2/devices", `{"devices":[{"uuid":"v1","id":1,"hostname":"h"}]}`,
			func(c *testcloud.Cloud, ctx context.Context, o ...resource.ListOption) (int, error) {
				v, err := devices.New(c.Client("accelerator", "/v2")).All(ctx, o...)
				return len(v), err
			}},
		{"device_profiles", "/v2/device_profiles", `{"device_profiles":[{"uuid":"` + cyborgCloudProfileUUID + `","name":"p","description":"","groups":[]}]}`,
			func(c *testcloud.Cloud, ctx context.Context, o ...resource.ListOption) (int, error) {
				v, err := deviceprofiles.New(c.Client("accelerator", "/v2")).All(ctx, o...)
				return len(v), err
			}},
		{"accelerator_requests", "/v2/accelerator_requests", `{"arqs":[{"uuid":"a1","state":"Initial","hostname":null}]}`,
			func(c *testcloud.Cloud, ctx context.Context, o ...resource.ListOption) (int, error) {
				v, err := acceleratorrequests.New(c.Client("accelerator", "/v2")).All(ctx, o...)
				return len(v), err
			}},
	}
}

func TestCyborgCloudListsSendNoQueryWithoutFiltersAndForwardExplicitOnes(t *testing.T) {
	for _, tc := range cyborgCloudLists() {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var queries []string
			cloud.Mux.HandleFunc("GET "+tc.path, func(w http.ResponseWriter, r *http.Request) {
				queries = append(queries, r.URL.RawQuery)
				testcloud.JSON(w, 200, tc.body)
			})
			if n, err := tc.all(cloud, context.Background()); err != nil || n != 1 {
				t.Fatalf("no filters: %d %v", n, err)
			}
			if n, err := tc.all(cloud, context.Background(), resource.WithQuery("flag", "false")); err != nil || n != 1 {
				t.Fatalf("filter: %d %v", n, err)
			}
			if len(queries) != 2 || queries[0] != "" || queries[1] != "flag=false" {
				t.Fatalf("queries=%q", queries)
			}
		})
	}
}

func TestCyborgCloudListsReturnEmptyNonNilAndRejectCallerPaging(t *testing.T) {
	for _, tc := range cyborgCloudLists() {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+tc.path, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, `{"`+map[string]string{"deployables": "deployables", "devices": "devices", "device_profiles": "device_profiles", "accelerator_requests": "arqs"}[tc.name]+`":[]}`)
			})
			if n, err := tc.all(cloud, context.Background()); err != nil || n != 0 || calls.Load() != 1 {
				t.Fatalf("empty list: %d %v calls=%d", n, err, calls.Load())
			}
			for _, key := range []string{"limit", "marker"} {
				if _, err := tc.all(cloud, context.Background(), resource.WithQuery(key, "1")); !errors.Is(err, resource.ErrUnsupported) {
					t.Fatalf("%s: %v", key, err)
				}
			}
			if calls.Load() != 1 {
				t.Fatalf("caller paging reached HTTP: %d", calls.Load())
			}
		})
	}
}

func TestCyborgCloudListsKeepNativeHTTPErrors(t *testing.T) {
	for _, tc := range cyborgCloudLists() {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+tc.path, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 404, `{"error":"missing"}`)
			})
			_, err := tc.all(cloud, context.Background())
			if !cyborgCloudIsStatus(err, 404) || calls.Load() != 1 {
				t.Fatalf("err=%v calls=%d", err, calls.Load())
			}
		})
	}
}

func TestCyborgCloudListsCancellationAndDeadlineCause(t *testing.T) {
	t.Parallel()
	for _, tc := range cyborgCloudLists() {
		t.Run(tc.name, func(t *testing.T) {
			cloud, calls := cyborgCloudNoHTTP(t)
			if _, err := tc.all(cloud, cyborgCloudCanceled()); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
				t.Fatalf("err=%v calls=%d", err, calls.Load())
			}
			hang := testcloud.New(t)
			var hung atomic.Int32
			cyborgCloudHang(hang, "GET "+tc.path, &hung)
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			if _, err := tc.all(hang, ctx); !errors.Is(err, context.DeadlineExceeded) || hung.Load() != 1 {
				t.Fatalf("deadline err=%v calls=%d", err, hung.Load())
			}
		})
	}
}

func TestCyborgCloudListsFollowServerContinuationOnly(t *testing.T) {
	cloud := testcloud.New(t)
	var pages atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/devices", func(w http.ResponseWriter, r *http.Request) {
		pages.Add(1)
		if r.URL.Query().Get("marker") == "second" {
			testcloud.JSON(w, 200, `{"devices":[{"uuid":"v2","id":2}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"devices":[{"uuid":"v1","id":1}],"next":"?marker=second"}`)
	})
	values, err := devices.New(cloud.Client("accelerator", "/v2")).All(context.Background())
	if err != nil || len(values) != 2 || values[1].UUID != "v2" || pages.Load() != 2 {
		t.Fatalf("values=%v err=%v pages=%d", values, err, pages.Load())
	}
}
