package images_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

type nativeDeleteTransport func(*http.Request) (*http.Response, error)

func (transport nativeDeleteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeDeleteBody struct {
	data              []byte
	readErr, closeErr error
	reads, closes     atomic.Int32
}

func (body *nativeDeleteBody) Read(target []byte) (int, error) {
	body.reads.Add(1)
	if len(body.data) > 0 {
		n := copy(target, body.data)
		body.data = body.data[n:]
		if len(body.data) == 0 && body.readErr != nil {
			return n, body.readErr
		}
		return n, nil
	}
	if body.readErr != nil {
		return 0, body.readErr
	}
	return 0, io.EOF
}
func (body *nativeDeleteBody) Close() error {
	body.closes.Add(1)
	return body.closeErr
}
func nativeDeleteWire(code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Body: body,
		Header: http.Header{"X-Native-Delete-Proof": {"actual"}, "Content-Type": {"application/json"}}}
}
func nativeDeleteOperation(t *testing.T, err error) *resource.OperationError {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != "Delete" || operation.Resource != "images" {
		t.Fatal("generated deletion context", err, operation)
	}
	var receipt *resource.ResponseError
	if errors.As(err, &receipt) {
		t.Fatal("native deletion fabricated an owned receipt", receipt)
	}
	return operation
}

// This facade preserves the native client's literal ServiceURL construction;
// escaping, validation and current-location/receipt ownership belong elsewhere.
func TestNativeImageDeleteRoutesHeadersAndOpaqueAcceptedBodies(t *testing.T) {
	for _, test := range []struct {
		name, id, path, query string
		base, override        bool
	}{
		{"resource base", "fixed", "/reverse/glance/v2/images/fixed", "", true, false},
		{"endpoint fallback", "fixed", "/catalog/unused/images/fixed", "", false, false},
		{"empty native ID", "", "/reverse/glance/v2/images/", "", true, false},
		{"native slash and query", "part/child?raw=1", "/reverse/glance/v2/images/part/child", "raw=1", true, false},
		{"service header overrides microversion", "fixed", "/reverse/glance/v2/images/fixed", "", true, true},
	} {
		for _, code := range []int{202, 204} {
			t.Run(fmt.Sprintf("%s/%d", test.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := nativeUpdateClient(cloud)
				if !test.base {
					client.ResourceBase = ""
				}
				client.Microversion = "2.10"
				client.MoreHeaders = map[string]string{"X-Source": "direct"}
				version := "image 2.10"
				if test.override {
					version = "image configured"
					client.MoreHeaders["OpenStack-API-Version"] = version
				}
				body := &nativeDeleteBody{data: []byte{0xff, '{', 'b', 'a', 'd'}}
				var calls atomic.Int32
				cloud.Provider.HTTPClient.Transport = nativeDeleteTransport(func(req *http.Request) (*http.Response, error) {
					calls.Add(1)
					if req.Method != http.MethodDelete || req.URL.Path != test.path || req.URL.RawQuery != test.query || req.Body != nil || req.ContentLength != 0 || req.Header.Get("Content-Type") != "" || req.Header.Get("Accept") != "application/json" || req.Header.Get("X-Source") != "direct" || req.Header.Get("X-Auth-Token") != "test-token" || req.Header.Get("OpenStack-API-Version") != version {
						t.Error(req.Method, req.URL, req.Header, req.Body, req.ContentLength)
					}
					return nativeDeleteWire(code, body), nil
				})
				api := images.New(client)
				if api.RawClient() != client {
					t.Fatal("native client identity changed")
				}
				if err := api.Delete(context.Background(), test.id); err != nil || calls.Load() != 1 || body.reads.Load() == 0 || body.closes.Load() != 1 {
					t.Fatal(err, calls.Load(), body.reads.Load(), body.closes.Load())
				}
			})
		}
	}
}

func TestNativeImageDeleteStrictStatusEvidenceAndErrResult(t *testing.T) {
	for _, code := range []int{200, 201, 203, 301, 404, 409, 503} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := nativeUpdateClient(cloud)
			body := &nativeDeleteBody{data: []byte("native rejection bytes")}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = nativeDeleteTransport(func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				return nativeDeleteWire(code, body), nil
			})
			err := images.New(client).Delete(context.Background(), "fixed")
			operation := nativeDeleteOperation(t, err)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || native.Method != http.MethodDelete || native.URL != cloud.Server.URL+nativeUpdatePath || !reflect.DeepEqual(native.Expected, []int{202, 204}) || native.ResponseHeader.Get("X-Native-Delete-Proof") != "actual" || calls.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(err, native, calls.Load(), body.closes.Load())
			}
			if _, direct := operation.Cause.(gophercloud.ErrUnexpectedResponseCode); !direct {
				t.Fatal("native cause replaced", operation.Cause)
			}
			th.AssertEquals(t, "native rejection bytes", string(native.Body))
		})
	}
	t.Run("inherited ErrResult preserves exact error", func(t *testing.T) {
		marker := errors.New("native result error")
		var result images.DeleteResult
		if result.ExtractErr() != nil {
			t.Fatal(result.ExtractErr())
		}
		result.Err = marker
		if result.ExtractErr() != marker {
			t.Fatal("native extraction changed cause", result.ExtractErr())
		}
	})
}

func TestNativeImageDeleteContextRetryAndBackoff(t *testing.T) {
	t.Run("canceled context never reaches HTTP", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc(nativeUpdatePath, func(w http.ResponseWriter, req *http.Request) { calls.Add(1); w.WriteHeader(204) })
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := images.New(nativeUpdateClient(cloud)).Delete(ctx, "fixed")
		if !errors.Is(err, context.Canceled) || calls.Load() != 0 {
			t.Fatal(err, calls.Load())
		}
		nativeDeleteOperation(t, err)
	})
	t.Run("native rejection retries same request with live token", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := nativeUpdateClient(cloud)
		client.MoreHeaders = map[string]string{"X-Source": "retry"}
		ctx := context.WithValue(context.Background(), "native-delete-proof", "caller")
		var calls atomic.Int32
		cloud.Mux.HandleFunc(nativeUpdatePath, func(w http.ResponseWriter, req *http.Request) {
			call := calls.Add(1)
			wantToken := "test-token"
			if call == 2 {
				wantToken = "rotated"
			}
			if req.Method != http.MethodDelete || req.URL.RawQuery != "" || req.Header.Get("X-Auth-Token") != wantToken || req.Header.Get("X-Source") != "retry" || req.ContentLength != 0 {
				t.Error(req.Method, req.URL, req.Header, req.ContentLength)
			}
			if call == 1 {
				testcloud.JSON(w, 503, `{"message":"native retry"}`)
				return
			}
			testcloud.JSON(w, 202, `not JSON`)
		})
		hooks := 0
		client.RetryFunc = func(hookCtx context.Context, method, target string, opts *gophercloud.RequestOpts, err error, retries uint) error {
			hooks++
			if hookCtx.Value("native-delete-proof") != "caller" || method != http.MethodDelete || target != cloud.Server.URL+nativeUpdatePath || !gophercloud.ResponseCodeIs(err, 503) || retries != 1 || opts.OkCodes != nil || opts.JSONBody != nil || opts.RawBody != nil || opts.JSONResponse != nil || opts.KeepResponseBody {
				t.Error(method, target, opts, err, retries)
			}
			client.SetToken("rotated")
			return nil
		}
		if err := images.New(client).Delete(ctx, "fixed"); err != nil || calls.Load() != 2 || hooks != 1 {
			t.Fatal(err, calls.Load(), hooks)
		}
	})
	t.Run("native rate limit backoff", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := nativeUpdateClient(cloud)
		ctx := context.WithValue(context.Background(), "native-delete-proof", "backoff")
		var calls atomic.Int32
		cloud.Mux.HandleFunc(nativeUpdatePath, func(w http.ResponseWriter, req *http.Request) {
			if calls.Add(1) == 1 {
				testcloud.JSON(w, 429, `{"message":"backoff"}`)
				return
			}
			w.WriteHeader(204)
		})
		hooks := 0
		client.RetryBackoffFunc = func(hookCtx context.Context, native *gophercloud.ErrUnexpectedResponseCode, cause error, retries uint) error {
			hooks++
			if hookCtx.Value("native-delete-proof") != "backoff" || native.Actual != 429 || native.Method != http.MethodDelete || native.URL != cloud.Server.URL+nativeUpdatePath || !reflect.DeepEqual(native.Expected, []int{202, 204}) || cause != nil || retries != 1 {
				t.Error(native, cause, retries)
			}
			return nil
		}
		if err := images.New(client).Delete(ctx, "fixed"); err != nil || calls.Load() != 2 || hooks != 1 {
			t.Fatal(err, calls.Load(), hooks)
		}
	})
	t.Run("terminal native retry error replaces rejection", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := nativeUpdateClient(cloud)
		marker := errors.New("terminal native retry")
		var calls atomic.Int32
		cloud.Mux.HandleFunc(nativeUpdatePath, func(w http.ResponseWriter, req *http.Request) {
			calls.Add(1)
			testcloud.JSON(w, 503, `native rejection`)
		})
		hooks := 0
		client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			hooks++
			if !gophercloud.ResponseCodeIs(err, 503) {
				t.Error(err)
			}
			return marker
		}
		err := images.New(client).Delete(context.Background(), "fixed")
		operation := nativeDeleteOperation(t, err)
		if operation.Cause != marker || !errors.Is(err, marker) || calls.Load() != 1 || hooks != 1 {
			t.Fatal(err, calls.Load(), hooks)
		}
	})
}

// Native body disposal intentionally differs from owned acknowledgement error
// handling: accepted read errors survive, while rejected read and Close errors
// are ignored by the pinned upstream provider. No accepted JSON decode occurs.
func TestNativeImageDeleteTransportAndBodyErrorsRemainNative(t *testing.T) {
	for _, test := range []struct {
		name            string
		code            int
		read, close     bool
		transport       bool
		wantReadCause   bool
		wantNativeProof bool
	}{
		{name: "accepted read error", code: 202, read: true, wantReadCause: true},
		{name: "accepted Close error ignored", code: 204, close: true},
		{name: "rejected read error ignored", code: 404, read: true, wantNativeProof: true},
		{name: "rejected Close error ignored", code: 404, close: true, wantNativeProof: true},
		{name: "transport cause retained", transport: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := nativeUpdateClient(cloud)
			readMarker, closeMarker, transportMarker := errors.New("native body read"), errors.New("native body Close"), errors.New("native transport")
			body := &nativeDeleteBody{data: []byte("partial proof")}
			if test.read {
				body.readErr = readMarker
			}
			if test.close {
				body.closeErr = closeMarker
			}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = nativeDeleteTransport(func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				if test.transport {
					return nil, transportMarker
				}
				return nativeDeleteWire(test.code, body), nil
			})
			hooks := 0
			if test.wantReadCause {
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, _ error, _ uint) error {
					hooks++
					return errors.New("unexpected accepted replay")
				}
			}
			err := images.New(client).Delete(context.Background(), "fixed")
			if test.wantNativeProof {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "partial proof" || native.ResponseHeader.Get("X-Native-Delete-Proof") != "actual" {
					t.Fatal(err, native)
				}
			} else if test.transport {
				if !errors.Is(err, transportMarker) {
					t.Fatal(err)
				}
			} else if test.wantReadCause {
				if !errors.Is(err, readMarker) {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if errors.Is(err, closeMarker) || (!test.wantReadCause && errors.Is(err, readMarker)) || calls.Load() != 1 || hooks != 0 {
				t.Fatal(err, calls.Load(), hooks)
			}
			if !test.transport && body.closes.Load() != 1 {
				t.Fatal("native disposal changed", body.closes.Load())
			}
			if err != nil {
				nativeDeleteOperation(t, err)
			}
		})
	}
}
