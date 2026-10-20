package objectstorage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestSwiftCloudObjectGetEagerLowerHeadersTextAndMissingReceipt(t *testing.T) {
	fixture := testcloud.New(t)
	calls := 0
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		data, e := io.ReadAll(r.Body)
		if e != nil || len(data) != 0 || r.Method != "GET" || r.URL.RawQuery != "" || r.Header.Get("Accept-Encoding") != "gzip, deflate, zstd" {
			t.Error(r, e)
		}
		w.Header().Set("X-Trans-Id", "cloud-get-actual")
		if r.URL.Path == "/account/box/missing" {
			w.WriteHeader(404)
			_, _ = w.Write([]byte("missing"))
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte{0x80, 0xe9})
	})
	cloud := objectstorage.New(fixture.Client("object-store", "/account/"))
	result, e := cloud.GetObjectText(context.Background(), "box", "file", objectstorage.WithObjectGetQueryString("bad\n%"), objectstorage.WithObjectGetStream(true), objectstorage.WithObjectGetChunkSize(0))
	if e != nil || result.Body == nil || *result.Body != "\u0080é" || result.Headers["x-trans-id"] != "cloud-get-actual" || calls != 1 {
		t.Fatal(result, e, calls)
	}
	result, e = cloud.GetObjectText(context.Background(), "box", "missing")
	if e != nil || !result.Missing || result.Headers != nil || result.Body != nil || result.Receipt.StatusCode != 404 || calls != 2 {
		t.Fatal(result, e, calls)
	}
}

func TestSwiftCloudObjectGetFileWriterAndFalseyOutputOptions(t *testing.T) {
	fixture := testcloud.New(t)
	payload := []byte{0, 255, 'x'}
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(payload)
	})
	cloud := objectstorage.New(fixture.Client("object-store", "/account/"))
	output := filepath.Join(t.TempDir(), "body.bin")
	result, e := cloud.GetObjectText(context.Background(), "box", "file", objectstorage.WithObjectGetOutfile(output), objectstorage.WithObjectGetChunkSize(1))
	stored, _ := os.ReadFile(output)
	if e != nil || result.Body != nil || result.BytesWritten != 3 || !bytes.Equal(stored, payload) {
		t.Fatal(result, e, stored)
	}
	writer := &bytes.Buffer{}
	result, e = cloud.GetObjectText(context.Background(), "box", "file", objectstorage.WithObjectGetWriter(writer))
	if e != nil || !bytes.Equal(writer.Bytes(), payload) || result.Body != nil || result.BytesWritten != 3 {
		t.Fatal(result, e, writer.Bytes())
	}
	result, e = cloud.GetObjectText(context.Background(), "box", "file", objectstorage.WithoutObjectGetContent())
	if e != nil || result.Body != nil || result.Missing {
		t.Fatal(result, e)
	}
}

func TestSwiftCloudObjectGetNilCancelledAndStatisticalGapStayExplicit(t *testing.T) {
	var nilCloud *objectstorage.Cloud
	if result, e := nilCloud.GetObjectText(context.Background(), "box", "file"); result != nil || !errors.Is(e, resource.ErrInvalidOption) {
		t.Fatal(result, e)
	}
	fixture := testcloud.New(t)
	calls := 0
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte("caf\xe9"))
	})
	cloud := objectstorage.New(fixture.Client("object-store", "/account/"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if result, e := cloud.GetObjectText(ctx, "box", "file"); result != nil || !errors.Is(e, context.Canceled) || calls != 0 {
		t.Fatal(result, e, calls)
	}
	result, e := cloud.GetObjectText(context.Background(), "box", "file")
	if !errors.Is(e, resource.ErrUnsupported) || result.Body != nil || result.Receipt.StatusCode != 200 || calls != 1 {
		t.Fatal(result, e, calls)
	}
}
