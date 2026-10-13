package objectstorage_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/klauspost/compress/zstd"
)

func TestSwiftCloudObjectStreamLazyAliasIgnoredQueryAndCleanMissing(t *testing.T) {
	fixture := testcloud.New(t)
	requests, options := 0, 0
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requests++
		raw, e := io.ReadAll(r.Body)
		if e != nil || len(raw) != 0 || r.URL.RawQuery != "" || r.Method != "GET" || r.Header.Get("X-Newest") != "" || r.Header.Get("Accept") != "*/*" {
			t.Error(r, e)
		}
		w.Header().Set("X-Trans-Id", "cloud-actual")
		if r.URL.Path == "/account/box/missing" {
			w.WriteHeader(404)
			_, _ = w.Write([]byte("not found"))
			return
		}
		_, _ = w.Write([]byte{0, 255, 'a', 'b', 'c'})
	})
	cloud := objectstorage.New(fixture.Client("object-store", "/account/"))
	stream := cloud.StreamObjectChunks(context.Background(), "box", "file", objectstorage.WithObjectStreamOpts(objectstorage.ObjectStreamOpts{QueryString: "?bad\n%"}), objectstorage.WithObjectStreamChunkSize(2), func(*objectstorage.ObjectStreamOpts) error { options++; return nil })
	if requests != 0 || options != 0 {
		t.Fatal(requests, options)
	}
	var chunks [][]byte
	for chunk, e := range stream.Chunks() {
		if e != nil {
			t.Fatal(e)
		}
		chunks = append(chunks, chunk)
	}
	if !reflect.DeepEqual(chunks, [][]byte{{0, 255}, {'a', 'b'}, {'c'}}) || stream.Err() != nil || requests != 1 || options != 1 || stream.Receipt().Header.Get("X-Trans-Id") != "cloud-actual" {
		t.Fatal(chunks, stream.Err(), requests, options, stream.Receipt())
	}
	missing := cloud.StreamObjectChunks(context.Background(), "box", "missing", objectstorage.WithObjectStreamQueryString("bad?query"))
	for chunk, e := range missing.Chunks() {
		t.Fatal(chunk, e)
	}
	if missing.Err() != nil || missing.Receipt().StatusCode != 404 || requests != 2 {
		t.Fatal(missing.Err(), missing.Receipt(), requests)
	}
}

func TestSwiftCloudObjectStreamDefaultTransportGzipAndZstandard(t *testing.T) {
	input := bytes.Repeat([]byte{0, 255, 'x'}, 1000)
	fixture := testcloud.New(t)
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept-Encoding") != "gzip, deflate, zstd" {
			t.Error(r.Header)
		}
		if r.URL.Path == "/account/box/gzip" {
			w.Header().Set("Content-Encoding", "gzip")
			writer := gzip.NewWriter(w)
			_, _ = writer.Write(input)
			if e := writer.Close(); e != nil {
				t.Error(e)
			}
		} else {
			w.Header().Set("Content-Encoding", "zstd")
			writer, e := zstd.NewWriter(w, zstd.WithEncoderConcurrency(1))
			if e != nil {
				t.Error(e)
				return
			}
			_, _ = writer.Write(input)
			if e := writer.Close(); e != nil {
				t.Error(e)
			}
		}
	})
	cloud := objectstorage.New(fixture.Client("object-store", "/account/"))
	for _, object := range []string{"gzip", "zstd"} {
		stream := cloud.StreamObjectChunks(context.Background(), "box", object)
		var chunks [][]byte
		for chunk, e := range stream.Chunks() {
			if e != nil {
				t.Fatal(object, e)
			}
			chunks = append(chunks, chunk)
		}
		if stream.Receipt().Header.Get("Content-Length") == "" {
			t.Fatal(object, stream.Receipt())
		}
		if stream.Receipt().Header.Get("Content-Encoding") != object {
			t.Fatal(object, stream.Receipt())
		}
		if stream.Err() != nil || !bytes.Equal(bytes.Join(chunks, nil), input) || len(chunks) != 3 || len(chunks[0]) != 1024 {
			t.Fatal(object, stream.Err(), len(chunks))
		}
	}
}

func TestSwiftCloudObjectStreamNilCancelledAndClosedHandleStayLazy(t *testing.T) {
	fixture := testcloud.New(t)
	calls := 0
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(200) })
	cloud := objectstorage.New(fixture.Client("object-store", "/account/"))
	var nilCloud *objectstorage.Cloud
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, tc := range []struct {
		stream objectstorage.ObjectChunkStream
		cause  error
	}{{nilCloud.StreamObjectChunks(context.Background(), "box", "file"), resource.ErrInvalidOption}, {cloud.StreamObjectChunks(ctx, "box", "file"), context.Canceled}, {cloud.StreamObjectChunks(nil, "box", "file"), resource.ErrInvalidOption}} {
		for _, e := range tc.stream.Chunks() {
			if !errors.Is(e, tc.cause) {
				t.Fatal(e, tc.cause)
			}
		}
		if !errors.Is(tc.stream.Err(), tc.cause) {
			t.Fatal(tc.stream.Err(), tc.cause)
		}
	}
	unopened := cloud.StreamObjectChunks(context.Background(), "box", "file")
	if unopened.Close() != nil {
		t.Fatal(unopened.Err())
	}
	for chunk, e := range unopened.Chunks() {
		t.Fatal(chunk, e)
	}
	if calls != 0 {
		t.Fatal(calls)
	}
}
