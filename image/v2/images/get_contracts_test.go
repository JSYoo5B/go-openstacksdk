package images_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func nativeGetOperation(t *testing.T, err error) *resource.OperationError {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != "Get" || operation.Resource != "images" {
		t.Fatal("generated get context", err, operation)
	}
	var receipt *resource.ResponseError
	if errors.As(err, &receipt) {
		t.Fatal("native get fabricated an owned receipt", receipt)
	}
	return operation
}

// The facade keeps the native literal ServiceURL, default headers and the
// pinned Extract header merge and Image UnmarshalJSON projection.
func TestNativeImageGetRoutesHeadersAndNativeProjection(t *testing.T) {
	for _, test := range []struct {
		name, id, path, query string
		base                  bool
	}{
		{"resource base", "fixed", "/reverse/glance/v2/images/fixed", "", true},
		{"endpoint fallback", "fixed", "/catalog/unused/images/fixed", "", false},
		{"empty native ID", "", "/reverse/glance/v2/images/", "", true},
		{"native slash and query", "part/child?raw=1", "/reverse/glance/v2/images/part/child", "raw=1", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := nativeUpdateClient(cloud)
			if !test.base {
				client.ResourceBase = ""
			}
			client.Microversion = "2.10"
			client.MoreHeaders = map[string]string{"X-Source": "direct"}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = nativeDeleteTransport(func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				if req.Method != http.MethodGet || req.URL.Path != test.path || req.URL.RawQuery != test.query || req.Body != nil || req.Header.Get("Accept") != "application/json" || req.Header.Get("X-Source") != "direct" || req.Header.Get("X-Auth-Token") != "test-token" || req.Header.Get("OpenStack-API-Version") != "image 2.10" {
					t.Error(req.Method, req.URL, req.Header)
				}
				wire := nativeDeleteWire(200, io.NopCloser(strings.NewReader(`{"id":"fixed","name":"n","status":"active","tags":["a"],"min_disk":2,"protected":true,"os_hidden":true,"size":42,"created_at":"2026-10-10T00:00:00Z","self":"/v2/images/fixed","properties":"literal","hw_vendor":"x","openstack-image-store-ids":"body ignored"}`)))
				wire.Header.Set("OpenStack-image-import-methods", "glance-direct,web-download")
				// Pinned UnmarshalJSON trims the whole value, then drops empty comma fields.
				wire.Header.Set("OpenStack-image-store-ids", " a,,b ")
				return wire, nil
			})
			api := images.New(client)
			if api.RawClient() != client {
				t.Fatal("native client identity changed")
			}
			got, err := api.Get(context.Background(), test.id)
			if err != nil || calls.Load() != 1 || got == nil {
				t.Fatal(got, err, calls.Load())
			}
			want := map[string]any{"hw_vendor": "x", "properties": "literal"}
			if got.ID != "fixed" || got.Name != "n" || got.Status != "active" || !reflect.DeepEqual(got.Tags, []string{"a"}) || got.MinDiskGigabytes != 2 || !got.Protected || !got.Hidden || got.SizeBytes != 42 || !got.CreatedAt.Equal(time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)) || !reflect.DeepEqual(got.Properties, want) {
				t.Fatal(got)
			}
			if !reflect.DeepEqual(got.OpenStackImageImportMethods, []string{"glance-direct", "web-download"}) || !reflect.DeepEqual(got.OpenStackImageStoreIDs, []string{"a", "b"}) {
				t.Fatal("pinned header merge and comma split changed", got.OpenStackImageImportMethods, got.OpenStackImageStoreIDs)
			}
		})
	}
	t.Run("null size is zero", func(t *testing.T) {
		cloud := testcloud.New(t)
		cloud.Provider.HTTPClient.Transport = nativeDeleteTransport(func(req *http.Request) (*http.Response, error) {
			return nativeDeleteWire(200, io.NopCloser(strings.NewReader(`{"id":"fixed","size":null}`))), nil
		})
		got, err := images.New(nativeUpdateClient(cloud)).Get(context.Background(), "fixed")
		if err != nil || got.SizeBytes != 0 || len(got.Properties) != 0 || got.OpenStackImageImportMethods != nil {
			t.Fatal(got, err)
		}
	})
}

func TestNativeImageGetStrictStatusAndDecodeErrors(t *testing.T) {
	for _, code := range []int{201, 203, 204, 301, 404, 503} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = nativeDeleteTransport(func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				return nativeDeleteWire(code, io.NopCloser(strings.NewReader(`{"id":"fixed"}`))), nil
			})
			got, err := images.New(nativeUpdateClient(cloud)).Get(context.Background(), "fixed")
			operation := nativeGetOperation(t, err)
			var native gophercloud.ErrUnexpectedResponseCode
			if got != nil || !errors.As(err, &native) || native.Actual != code || native.Method != http.MethodGet || !reflect.DeepEqual(native.Expected, []int{200}) || native.ResponseHeader.Get("X-Native-Delete-Proof") != "actual" || calls.Load() != 1 {
				t.Fatal(got, err, native)
			}
			if _, direct := operation.Cause.(gophercloud.ErrUnexpectedResponseCode); !direct {
				t.Fatal("native cause replaced", operation.Cause)
			}
		})
	}
	for _, test := range []struct {
		name, body string
		partial    bool
	}{
		// Native ExtractInto keeps the partially decoded Image with its error.
		{"string size", `{"id":"fixed","size":"42"}`, true},
		{"invalid JSON", `{"id":`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Provider.HTTPClient.Transport = nativeDeleteTransport(func(req *http.Request) (*http.Response, error) {
				return nativeDeleteWire(200, io.NopCloser(strings.NewReader(test.body))), nil
			})
			got, err := images.New(nativeUpdateClient(cloud)).Get(context.Background(), "fixed")
			nativeGetOperation(t, err)
			if err == nil || test.partial != (got != nil && got.ID == "fixed") {
				t.Fatal(got, err)
			}
		})
	}
	t.Run("canceled context", func(t *testing.T) {
		cloud := testcloud.New(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := images.New(nativeUpdateClient(cloud)).Get(ctx, "fixed")
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		nativeGetOperation(t, err)
	})
}
