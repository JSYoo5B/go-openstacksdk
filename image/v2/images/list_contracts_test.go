package images_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

func nativeListIDs(t *testing.T, seq func(func(*images.Image, error) bool)) ([]string, error) {
	t.Helper()
	var ids []string
	for value, err := range seq {
		if err != nil {
			return ids, err
		}
		ids = append(ids, value.ID)
	}
	return ids, nil
}

// The facade forwards the pinned ListOpts query builder and appends extension
// query values; zero-valued fields are omitted by native BuildQueryString,
// which also drops the int64 SizeMin/SizeMax fields in v2.15.0.
func TestNativeImageListQueryAndExtensions(t *testing.T) {
	cloud := testcloud.New(t)
	client := nativeUpdateClient(cloud)
	client.MoreHeaders = map[string]string{"X-Source": "direct"}
	var query url.Values
	cloud.Provider.HTTPClient.Transport = nativeDeleteTransport(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/reverse/glance/v2/images" || req.Header.Get("Accept") != "application/json" || req.Header.Get("X-Source") != "direct" || req.Body != nil {
			t.Error(req.Method, req.URL, req.Header)
		}
		query = req.URL.Query()
		return nativeDeleteWire(200, io.NopCloser(strings.NewReader(`{"images":[]}`))), nil
	})
	created := time.Date(2026, 10, 10, 1, 2, 3, 0, time.UTC)
	opts := images.ListOpts{Limit: 2, Marker: "m", Name: "n", Visibility: images.ImageVisibilityShared, Hidden: true, MemberStatus: images.ImageMemberStatusAll,
		Owner: "o", Status: images.ImageStatusActive, SizeMin: 1, SizeMax: 9, Sort: "name:asc", SortKey: "id", SortDir: "desc", Tags: []string{"a", "b"},
		CreatedAtQuery: &images.ImageDateQuery{Date: created, Filter: images.FilterGTE}, UpdatedAtQuery: &images.ImageDateQuery{Date: created},
		ContainerFormat: "bare", DiskFormat: "raw"}
	ids, err := nativeListIDs(t, images.New(client).List(context.Background(), images.WithListOptions(opts), images.WithListQuery("os_distro", "ubuntu")))
	want := url.Values{"limit": {"2"}, "marker": {"m"}, "name": {"n"}, "visibility": {"shared"}, "os_hidden": {"true"}, "member_status": {"all"},
		"owner": {"o"}, "status": {"active"}, "sort": {"name:asc"}, "sort_key": {"id"}, "sort_dir": {"desc"},
		"tag": {"a", "b"}, "created_at": {"gte:2026-10-10T01:02:03Z"}, "updated_at": {"2026-10-10T01:02:03Z"}, "container_format": {"bare"}, "disk_format": {"raw"}, "os_distro": {"ubuntu"}}
	if err != nil || len(ids) != 0 || !reflect.DeepEqual(query, want) {
		t.Fatal(ids, err, query)
	}
	ids, err = nativeListIDs(t, images.New(client).List(context.Background()))
	if err != nil || len(ids) != 0 || len(query) != 0 {
		t.Fatal("zero ListOpts sends no query", ids, err, query)
	}
}

// Native next links keep only path and query; the host and base come from the
// client ServiceURL. Empty pages and 204 stop without following next.
func TestNativeImageListPagingStopsAndErrors(t *testing.T) {
	t.Run("next path rebased onto service URL", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		var urls []string
		cloud.Provider.HTTPClient.Transport = nativeDeleteTransport(func(req *http.Request) (*http.Response, error) {
			urls = append(urls, req.URL.Path+"?"+req.URL.RawQuery)
			switch calls.Add(1) {
			case 1:
				return nativeDeleteWire(200, io.NopCloser(strings.NewReader(`{"images":[{"id":"a"},{"id":"b"}],"next":"https://foreign.test/v2/images?marker=b"}`))), nil
			default:
				return nativeDeleteWire(200, io.NopCloser(strings.NewReader(`{"images":[{"id":"c"}]}`))), nil
			}
		})
		ids, err := nativeListIDs(t, images.New(nativeUpdateClient(cloud)).List(context.Background()))
		if err != nil || !reflect.DeepEqual(ids, []string{"a", "b", "c"}) || len(urls) != 2 || urls[1] != "/reverse/glance/v2/images?marker=b" {
			t.Fatal(ids, err, urls)
		}
	})
	t.Run("empty page ignores next", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeDeleteTransport(func(req *http.Request) (*http.Response, error) {
			calls.Add(1)
			return nativeDeleteWire(200, io.NopCloser(strings.NewReader(`{"images":[],"next":"/v2/images?marker=x"}`))), nil
		})
		ids, err := nativeListIDs(t, images.New(nativeUpdateClient(cloud)).List(context.Background()))
		if err != nil || len(ids) != 0 || calls.Load() != 1 {
			t.Fatal(ids, err, calls.Load())
		}
	})
	t.Run("bodyless 204 fails native JSON page decode", func(t *testing.T) {
		cloud := testcloud.New(t)
		cloud.Provider.HTTPClient.Transport = nativeDeleteTransport(func(req *http.Request) (*http.Response, error) {
			return nativeDeleteWire(204, io.NopCloser(strings.NewReader(""))), nil
		})
		ids, err := nativeListIDs(t, images.New(nativeUpdateClient(cloud)).List(context.Background()))
		if !errors.Is(err, io.EOF) || len(ids) != 0 {
			t.Fatal(ids, err)
		}
	})
	t.Run("late page failure keeps earlier rows", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeDeleteTransport(func(req *http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 {
				return nativeDeleteWire(200, io.NopCloser(strings.NewReader(`{"images":[{"id":"a"}],"next":"/v2/images?marker=a"}`))), nil
			}
			return nativeDeleteWire(503, io.NopCloser(strings.NewReader(`{"message":"late"}`))), nil
		})
		ids, err := nativeListIDs(t, images.New(nativeUpdateClient(cloud)).List(context.Background()))
		var native gophercloud.ErrUnexpectedResponseCode
		if !reflect.DeepEqual(ids, []string{"a"}) || !errors.As(err, &native) || native.Actual != 503 || calls.Load() != 2 {
			t.Fatal(ids, err)
		}
	})
	t.Run("early break sends no further page", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeDeleteTransport(func(req *http.Request) (*http.Response, error) {
			calls.Add(1)
			return nativeDeleteWire(200, io.NopCloser(strings.NewReader(`{"images":[{"id":"a"},{"id":"b"}],"next":"/v2/images?marker=b"}`))), nil
		})
		for value, err := range images.New(nativeUpdateClient(cloud)).List(context.Background()) {
			if err != nil || value.ID != "a" {
				t.Fatal(value, err)
			}
			break
		}
		if calls.Load() != 1 {
			t.Fatal(calls.Load())
		}
	})
	t.Run("decode failure and invalid query option", func(t *testing.T) {
		cloud := testcloud.New(t)
		cloud.Provider.HTTPClient.Transport = nativeDeleteTransport(func(req *http.Request) (*http.Response, error) {
			return nativeDeleteWire(200, io.NopCloser(strings.NewReader(`{"images":[{"id":"a","size":"x"}]}`))), nil
		})
		if _, err := nativeListIDs(t, images.New(nativeUpdateClient(cloud)).List(context.Background())); err == nil {
			t.Fatal("native row decode error was hidden")
		}
		if _, err := nativeListIDs(t, images.New(nativeUpdateClient(cloud)).List(context.Background(), nil)); err == nil {
			t.Fatal("nil option accepted")
		}
	})
}
