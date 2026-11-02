package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Shared calls that a category admits but an image operation never supported
// fail before HTTP.
func TestImageSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	client := cloud.Client("image", "/image/")
	s := image.New(client)

	// Header-merging options (GetCall): request.Headers would replace.
	_, err := s.GetImage(ctx, resource.ID("i1"), request.Headers(map[string]string{"X-Test": "1"}))
	requireUnsupportedCall(t, err, "Headers")
	_, err = s.GetImage(ctx, resource.ID("i1"), request.Microversion("2.1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = s.GetImageLocations(ctx, resource.ID("i1"), request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")

	// Lists (ListCall): no query, Paginated or replacing headers.
	for _, call := range []image.ListImagesOption{request.Query("name", "x"), request.Paginated(false), request.Headers(nil), request.Microversion("2.1")} {
		entry, err := request.Entry(call)
		if err != nil {
			t.Fatal(err)
		}
		_, err = s.AllImages(ctx, call)
		requireUnsupportedCall(t, err, entry.Kind.String())
	}
	_, err = s.AllTasks(ctx, request.Query("type", "import"))
	requireUnsupportedCall(t, err, "Query")
	_, err = s.AllImageMemberRecords(ctx, resource.ID("i1"), request.Query("status", "accepted"))
	requireUnsupportedCall(t, err, "Query")

	// Deletes (DeleteCall): DeleteImage takes no headers; ClearCache no
	// IgnoreMissing.
	_, err = s.DeleteImage(ctx, resource.ID("i1"), request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")
	_, err = s.ClearCache(ctx, request.IgnoreMissing(true))
	requireUnsupportedCall(t, err, "IgnoreMissing")

	// Writes (WriteCall): typed field options only.
	_, err = s.UpdateImage(ctx, resource.ID("i1"), request.Field("name", "x"))
	requireUnsupportedCall(t, err, "Field")

	// Finds (FindCall): list filters go through WithFindImageRecordListOptions.
	_, err = s.FindImageRecord(ctx, "i1", request.Query("name", "x"))
	requireUnsupportedCall(t, err, "Query")

	// Image v2 packages.
	tags, err := metadeftags.New(client).InNamespace(ctx, "ns")
	if err != nil {
		t.Fatal(err)
	}
	_, err = tags.All(ctx, request.Microversion("2.1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = tags.All(ctx, request.Paginated(false))
	requireUnsupportedCall(t, err, "Paginated")
	_, err = tags.Get(ctx, "t", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = tags.Delete(ctx, "t", request.IgnoreMissing(true))
	requireUnsupportedCall(t, err, "IgnoreMissing")
	_, err = tags.Create(ctx, "t", request.Field("x", 1))
	requireUnsupportedCall(t, err, "Field")
	_, err = serviceinfo.New(client).AllStores(ctx, request.Limit(1))
	requireUnsupportedCall(t, err, "Limit")
	_, err = images.New(client).AllTaskRecords(ctx, "i1", request.Header("X-Test", "1"))
	requireUnsupportedCall(t, err, "Header")

	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}
}

// request.Header merges into the headers collected by WithGetImageHeaders in
// caller order, like a one-entry WithGetImageHeaders map; list paging calls
// apply in caller order too.
func TestImageSharedCallsMergeHeadersInCallerOrder(t *testing.T) {
	cloud := testcloud.New(t)
	var got http.Header
	var query string
	cloud.Mux.HandleFunc("/image/images/i1", func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"i1","name":"one"}`))
	})
	cloud.Mux.HandleFunc("/image/images", func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"images":[]}`))
	})
	s := image.New(cloud.Client("image", "/image/"))

	_, err := s.GetImage(context.Background(), resource.ID("i1"),
		request.Header("x-first", "1"),
		image.WithGetImageHeaders(map[string]string{"X-Second": "2", "X-First": "replaced"}),
		request.Header("x-second", "last"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Get("X-First") != "replaced" || got.Get("X-Second") != "last" || len(got.Values("X-Second")) != 1 {
		t.Fatal(got)
	}
	if _, err := s.GetImage(context.Background(), resource.ID("i1"), request.Header("Authorization", "x")); err == nil {
		t.Fatal("SDK-owned header accepted")
	}

	_, err = s.AllImages(context.Background(), request.Limit(1), request.Marker("a"), request.Limit(2), request.Marker("b"), request.MaxItems(1))
	if err != nil || query != "limit=2&marker=b" {
		t.Fatal(query, err)
	}
}
