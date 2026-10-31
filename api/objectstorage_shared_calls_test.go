package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	swift "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// Shared calls that a category admits but an object storage operation never
// supported fail before HTTP. request.Headers is among them: the operations'
// own With<Op>Headers options merge, while request.Headers replaces.
func TestObjectStorageSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	client := cloud.Client("object-store", "/swift/v1/AUTH_p")

	_, err := objects.New(client).GetMetadata(ctx, "box", "key", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = objects.New(client).GetMetadata(ctx, "box", "key", request.Microversion("1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = objects.New(client).SetMetadata(ctx, "box", "key", map[string]string{"a": "b"}, request.Field("x", 1))
	requireUnsupportedCall(t, err, "Field")
	_, err = objects.New(client).DeleteObject(ctx, "box", "key", request.Microversion("1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = objects.New(client).GetObject(ctx, "box", "key", request.Headers(map[string]string{"X-A": "1"}))
	requireUnsupportedCall(t, err, "Headers")
	for _, err := range objects.New(client).ListRecords(ctx, "box", request.Header("X-A", "1")) {
		requireUnsupportedCall(t, err, "Header")
	}
	for _, err := range containers.New(client).ListRecords(ctx, request.Headers(nil)) {
		requireUnsupportedCall(t, err, "Headers")
	}
	_, err = containers.New(client).CreateContainer(ctx, "box", request.Microversion("1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = containers.New(client).DeleteContainer(ctx, "box", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = containers.New(client).DeleteRecord(ctx, "box", request.Microversion("1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = containers.New(client).SetTempURLKey(ctx, "box", "key", request.Field("x", 1))
	requireUnsupportedCall(t, err, "Field")
	_, err = accounts.New(client).GetMetadata(ctx, request.Microversion("1"))
	requireUnsupportedCall(t, err, "Microversion")
	_, err = accounts.New(client).SetTempURLKey(ctx, "key", request.Headers(nil))
	requireUnsupportedCall(t, err, "Headers")
	_, err = swift.New(client).GetTempURLKey(ctx, request.Microversion("1"))
	requireUnsupportedCall(t, err, "Microversion")
	if n := calls.Load(); n != 0 {
		t.Fatalf("calls = %d", n)
	}
}

// request.Header is validated and merged like a one-entry With<Op>Headers
// map, in caller order: a later Header replaces a canonical-equal earlier
// name, and both forms reach the wire.
func TestObjectStorageSharedHeaderMergesInCallerOrder(t *testing.T) {
	cloud := testcloud.New(t)
	var got http.Header
	cloud.Mux.HandleFunc("/swift/v1/AUTH_p/box/key", func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.WriteHeader(http.StatusNoContent)
	})
	api := objects.New(cloud.Client("object-store", "/swift/v1/AUTH_p"))
	_, err := api.DeleteObject(context.Background(), "box", "key",
		objects.WithDeleteObjectStaticLargeObject(false),
		objects.WithDeleteObjectHeaders(map[string]string{"x-trace": "first", "X-Keep": "map"}),
		request.Header("X-Trace", "last"),
		request.IgnoreMissing(false),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Get("X-Trace") != "last" || got.Get("X-Keep") != "map" || len(got.Values("X-Trace")) != 1 {
		t.Fatal(got)
	}
}

// request.Paginated(false) replaces the removed SinglePage option and keeps
// caller order with an earlier Paginated(true).
func TestObjectStorageRecordListPaginatedCallerOrder(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/swift/v1/AUTH_p/box", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `[{"name":"a","hash":"h","bytes":1,"last_modified":"2020-01-01T00:00:00.000000","content_type":"text/plain"}]`)
	})
	api := objects.New(cloud.Client("object-store", "/swift/v1/AUTH_p"))
	n := 0
	for _, err := range api.ListRecords(context.Background(), "box", request.Limit(1), request.Paginated(true), request.Paginated(false)) {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 1 || calls.Load() != 1 {
		t.Fatal(n, calls.Load())
	}
}
