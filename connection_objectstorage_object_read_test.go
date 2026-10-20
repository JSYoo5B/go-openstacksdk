package openstack_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type connectionObjectReadBody struct {
	io.Reader
	closes, reads int
	closeErr      error
}

func (b *connectionObjectReadBody) Read(p []byte) (int, error) { b.reads++; return b.Reader.Read(p) }
func (b *connectionObjectReadBody) Close() error               { b.closes++; return b.closeErr }

type connectionObjectReadWriter struct {
	bytes.Buffer
	closes, flushes int
}

func (w *connectionObjectReadWriter) Close() error { w.closes++; return nil }
func (w *connectionObjectReadWriter) Flush() error { w.flushes++; return nil }

func TestConnectionObjectReadSharesClientAndOwnsTransfer(t *testing.T) {
	ctx := context.Background()
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("swift-0")
	const endpoint = "https://cloud.test/unused/v1/AUTH_account%25/"
	const target = "https://cloud.test/reverse%25/v1/AUTH_account%25/%EB%B0%B1%EC%97%85%20%252F%3F%23/folder%2F%EB%82%B4%EC%9A%A9%20%252F%3F%23"
	const container = "백업 %2F?#"
	const object = "folder/내용 %2F?#"
	binary := []byte{'a', 0, 255, 'z'}
	closeErr := errors.New("stream cleanup")
	bodies := []*connectionObjectReadBody{
		{Reader: bytes.NewReader(binary)},
		{Reader: strings.NewReader("must not read304")},
		{Reader: strings.NewReader("opaque multipart"), closeErr: closeErr},
	}
	var source *gophercloud.ServiceClient
	calls := 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		n := calls
		calls++
		if n >= len(bodies) {
			t.Fatal("unexpected request", n)
		}
		if r.Method != http.MethodGet || r.URL.String() != target || r.Header.Get("X-Source") != fmt.Sprint(n) || r.Header.Get("X-Auth-Token") != fmt.Sprintf("swift-%d", n+1) || r.Header.Get("X-Newest") != "false" || r.Header.Get("X-Call") != "owned" {
			t.Fatal(n, r.Method, r.URL, r.Header)
		}
		h := http.Header{"X-Proof": {fmt.Sprint(n)}, "X-Object-Meta-Owner": {""}, "X-Unknown": {"one", "two"}, "ETag": {"not-a-checksum"}}
		code := []int{200, 304, 206}[n]
		if n == 2 {
			h.Set("Content-Type", "multipart/byteranges; boundary=test")
			h.Set("Content-Length", "9007199254740995")
		}
		source.MoreHeaders["X-Source"] = fmt.Sprint(n + 1)
		return &http.Response{Request: r, StatusCode: code, Header: h, Body: bodies[n]}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.ObjectStorage, "v1", endpoint))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.ObjectStorage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	versioned, err := conn.ObjectStorageV1(ctx)
	if err != nil || service != versioned || service.Objects.RawClient() != service.RawClient() || service.RawClient().ProviderClient != provider {
		t.Fatal("shared facade", err)
	}
	source = service.RawClient()
	source.ResourceBase = "https://cloud.test/reverse%25/v1/AUTH_account%25/"
	source.MoreHeaders = map[string]string{"X-Source": "0"}
	headers := map[string]string{"X-Call": "owned"}
	opts := []objects.ObjectReadOption{objects.WithObjectReadHeaders(headers), objects.WithObjectReadNewest(false), func(o *objects.ObjectReadOpts) error { provider.SetToken(fmt.Sprintf("swift-%d", calls+1)); return nil }}
	headers["X-Call"] = "late"
	get, err := service.Objects.GetObject(ctx, container, object, opts...)
	if err != nil || get == nil || !get.Complete || get.NotModified || get.StatusCode != 200 || !bytes.Equal(get.Body, binary) || get.Metadata == nil || get.Metadata.Values["owner"] != "" || bodies[0].closes != 1 || calls != 1 {
		t.Fatal(get, err, calls)
	}
	if _, present := get.Metadata.Values["owner"]; !present {
		t.Fatal("missing explicitly empty metadata")
	}
	output := &connectionObjectReadWriter{}
	download, err := service.Objects.DownloadObject(ctx, container, object, output, opts...)
	if err != nil || download == nil || !download.NotModified || !download.Complete || download.BytesWritten != 0 || output.Len() != 0 || output.closes != 0 || output.flushes != 0 || bodies[1].closes != 1 || bodies[1].reads != 0 || calls != 2 {
		t.Fatal(download, err, calls)
	}
	stream, err := service.Objects.StreamObject(ctx, container, object, opts...)
	if err != nil || stream == nil || stream.Body == nil || stream.Complete || stream.BytesRead != 0 || bodies[2].closes != 0 || bodies[2].reads != 0 || calls != 3 {
		t.Fatal(stream, err, calls)
	}
	stream.Header.Set("X-Proof", "changed")
	data, err := io.ReadAll(stream.Body)
	var proof *resource.ResponseError
	if !errors.Is(err, closeErr) || !errors.As(err, &proof) || proof.StatusCode != 206 || proof.Header.Get("X-Proof") != "2" || len(proof.Body) != 0 || string(data) != "opaque multipart" || stream.BytesRead != int64(len(data)) || !stream.Complete || stream.Metadata.ContentLength == nil || *stream.Metadata.ContentLength != 9007199254740995 || bodies[2].closes != 1 {
		t.Fatal(string(data), stream, err, proof)
	}
	if err := stream.Body.Close(); !errors.Is(err, closeErr) || bodies[2].closes != 1 {
		t.Fatal("idempotent close", err, bodies[2].closes)
	}
	if get.Header.Get("X-Proof") != "0" || download.Header.Get("X-Proof") != "1" || get.Body[2] != 255 || calls != 3 || source.ProviderClient != provider {
		t.Fatal("transfer proof aliases or source changed")
	}
}
