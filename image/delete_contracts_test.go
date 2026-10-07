package image_test

import (
	"bytes"
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

	"github.com/JSYoo5B/gophercloudsdk/image"
	sdkimages "github.com/JSYoo5B/gophercloudsdk/image/v2/images"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const deletePrefix = "/reverse/glance/v2/"

type deleteTransport func(*http.Request) (*http.Response, error)

func (f deleteTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	wire, err := f(r)
	if wire != nil && wire.Request == nil {
		wire.Request = r
	}
	return wire, err
}

type deleteBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *deleteBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

func deleteWire(code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{
		"X-Request-Id": {"actual-delete"}, "X-Wire-Status": {fmt.Sprint(code)},
		"Content-Type": {"application/json"},
	}, Body: body}
}

type deleteReadFailure struct {
	prefix []byte
	cause  error
	onRead func()
}

type deleteContextReader struct {
	ctx    context.Context
	prefix []byte
}

func (b *deleteContextReader) Read(p []byte) (int, error) {
	if len(b.prefix) != 0 {
		n := copy(p, b.prefix)
		b.prefix = b.prefix[n:]
		return n, nil
	}
	<-b.ctx.Done()
	return 0, b.ctx.Err()
}

func (b *deleteReadFailure) Read(p []byte) (int, error) {
	if b.onRead != nil {
		b.onRead()
		b.onRead = nil
	}
	n := copy(p, b.prefix)
	b.prefix = b.prefix[n:]
	return n, b.cause
}

func deleteNative404(target string) gophercloud.ErrUnexpectedResponseCode {
	return gophercloud.ErrUnexpectedResponseCode{
		Method: http.MethodDelete, URL: target, Expected: []int{204}, Actual: 404,
		Body: []byte("nested callback evidence"), ResponseHeader: http.Header{"X-Nested": {"true"}},
	}
}

func TestDeleteImageWholeAndStoreRoutesAndEvidence(t *testing.T) {
	for _, store := range []string{"", "fast"} {
		t.Run("store="+store, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", "/catalog/unused/")
			client.ResourceBase = cloud.Server.URL + deletePrefix
			client.MoreHeaders = map[string]string{"X-Source": "captured"}
			client.Microversion = "2.10"
			cloud.Provider.SetToken("current-token")
			payload := []byte{0, 255, '{', 'x', 0}
			body := &deleteBody{Reader: bytes.NewReader(payload)}
			wire := deleteWire(204, body)
			wire.Header.Set("Location", "https://foreign.invalid/not-a-delete-target")
			var calls atomic.Int32
			wantPath := deletePrefix + "images/fixed"
			var options []image.DeleteImageOption
			if store != "" {
				wantPath = deletePrefix + "stores/fast/fixed"
				options = []image.DeleteImageOption{image.WithDeleteImageStore(store)}
			}
			cloud.Provider.HTTPClient.Transport = deleteTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Method != http.MethodDelete || r.URL.Path != wantPath || r.URL.RawQuery != "" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != "current-token" || r.Header.Get("OpenStack-API-Version") != "image 2.10" || r.Body != nil {
					t.Error(r.Method, r.URL, r.Header, r.Body)
				}
				return wire, nil
			})
			v, err := image.New(client).DeleteImage(context.Background(), resource.ID("fixed"), options...)
			if err != nil || v == nil || v.ImageID != "fixed" || v.StoreID != store || v.StatusCode != 204 || !bytes.Equal(v.Body, payload) || v.Header.Get("X-Request-Id") != "actual-delete" || v.Header.Get("Location") == "" || calls.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(v, err, calls.Load(), body.closes.Load())
			}
			wire.Header.Set("X-Request-Id", "later-wire")
			v.Body[0] = '!'
			if v.Header.Get("X-Request-Id") != "actual-delete" || payload[0] != 0 || client.ResourceBase != cloud.Server.URL+deletePrefix || client.MoreHeaders["X-Source"] != "captured" {
				t.Fatal("result aliases response or source", v, client, payload)
			}
		})
	}
}

func TestDeleteImageCompletePreflight(t *testing.T) {
	for _, value := range []string{"", ".", "..", "../escape", "white space", "colon:store", "query?x", "percent%2f", "slash/store", "back\\slash", "line\nbreak", string([]byte{255})} {
		t.Run(fmt.Sprintf("invalid store %q", value), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = deleteTransport(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return deleteWire(500, io.NopCloser(strings.NewReader("unexpected"))), nil
			})
			v, err := image.New(cloud.Client("image", deletePrefix)).DeleteImage(context.Background(), resource.Name("needle"), image.WithDeleteImageStore(value))
			if v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(v, err, calls.Load())
			}
		})
	}
	for _, mode := range []string{"nil service", "nil client", "nil provider", "wrong type", "foreign base", "query base", "base without slash", "bad ID", "invalid UTF8 ID", "invalid UTF8 name", "zero ref", "nil option", "nil context", "canceled", "owned auth header", "invalid header key", "invalid header value", "conflicting aliases", "conflicting microversion", "invalid microversion", "callback provider change", "callback type change", "callback nested404"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", deletePrefix)
			var calls, callbacks atomic.Int32
			cloud.Provider.HTTPClient.Transport = deleteTransport(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return deleteWire(500, io.NopCloser(strings.NewReader("unexpected"))), nil
			})
			ctx, ref := context.Background(), resource.ID("fixed")
			want := error(resource.ErrInvalidOption)
			var options []image.DeleteImageOption
			cause := errors.New("preflight cause")
			switch mode {
			case "nil client":
				client = nil
			case "nil provider":
				client.ProviderClient = nil
			case "wrong type":
				client.Type, want = "compute", resource.ErrUnsupported
			case "foreign base":
				client.ResourceBase = "https://foreign.invalid/v2/"
			case "query base":
				client.ResourceBase = cloud.Server.URL + deletePrefix + "?query=bad"
			case "base without slash":
				client.ResourceBase = cloud.Server.URL + "/v2"
			case "bad ID":
				ref = resource.ID("../escape")
			case "invalid UTF8 ID":
				ref = resource.ID(string([]byte{255}))
			case "invalid UTF8 name":
				ref = resource.Name(string([]byte{255}))
			case "zero ref":
				ref = resource.Ref{}
			case "nil option":
				options = []image.DeleteImageOption{nil}
			case "nil context":
				ctx = nil
			case "canceled":
				parent, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx, want = parent, context.Canceled
			case "owned auth header":
				client.MoreHeaders = map[string]string{"x-auth-token": "override"}
			case "invalid header key":
				client.MoreHeaders = map[string]string{"X Bad": "value"}
			case "invalid header value":
				client.MoreHeaders = map[string]string{"X-Test": "bad\r\nvalue"}
			case "conflicting aliases":
				client.MoreHeaders = map[string]string{"X-Test": "first", "x-test": "second"}
			case "conflicting microversion":
				client.Microversion = "2.10"
				client.MoreHeaders = map[string]string{"OpenStack-API-Version": "compute 2.10"}
			case "invalid microversion":
				client.Microversion = "2.10\nunsafe"
			case "callback provider change", "callback type change", "callback nested404":
				options = []image.DeleteImageOption{func(*image.DeleteImageOpts) error {
					callbacks.Add(1)
					if mode == "callback provider change" {
						client.ProviderClient = &gophercloud.ProviderClient{HTTPClient: cloud.Provider.HTTPClient}
					} else if mode == "callback type change" {
						client.Type = "compute"
					} else {
						return errors.Join(cause, deleteNative404("callback://not-wire"))
					}
					return nil
				}}
				if mode == "callback type change" {
					want = resource.ErrUnsupported
				} else if mode == "callback nested404" {
					want = cause
				}
			}
			service := image.New(client)
			if mode == "nil service" {
				service = nil
			}
			v, err := service.DeleteImage(ctx, ref, options...)
			if v != nil || !errors.Is(err, want) || calls.Load() != 0 || strings.HasPrefix(mode, "callback") && callbacks.Load() != 1 {
				t.Fatal(v, err, calls.Load(), callbacks.Load())
			}
			if mode == "canceled" && !errors.Is(err, cause) {
				t.Fatal("custom cancellation cause lost", err)
			}
		})
	}
}

func TestDeleteImagePreparedSnapshotsAndSource(t *testing.T) {
	t.Run("bulk and retained callback pointers own strict store policy", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := cloud.Client("image", deletePrefix)
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		store, ignore := "fast", false
		bulk := image.WithDeleteImageOpts(image.DeleteImageOpts{StoreID: &store, IgnoreMissing: &ignore})
		store, ignore = "../changed", true
		var retained *image.DeleteImageOpts
		var calls, callbacks atomic.Int32
		body := &deleteBody{Reader: strings.NewReader("missing")}
		cloud.Provider.HTTPClient.Transport = deleteTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			*retained.StoreID, *retained.IgnoreMissing = "late", true
			if r.Method != http.MethodDelete || r.URL.Path != deletePrefix+"stores/fast/fixed" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != "live-after-options" {
				t.Error(r.Method, r.URL, r.Header)
			}
			return deleteWire(404, body), nil
		})
		options := []image.DeleteImageOption{bulk, func(o *image.DeleteImageOpts) error {
			callbacks.Add(1)
			retained = o
			client.ResourceBase = cloud.Server.URL + "/later/v2/"
			client.MoreHeaders = map[string]string{"X-Source": "later"}
			cloud.Provider.SetToken("live-after-options")
			return nil
		}}
		v, err := image.New(client).DeleteImage(context.Background(), resource.ID("fixed"), options...)
		if v != nil || !errors.Is(err, resource.ErrNotFound) || !gophercloud.ResponseCodeIs(err, 404) || callbacks.Load() != 1 || calls.Load() != 1 || body.closes.Load() != 1 || client.MoreHeaders["X-Source"] != "later" {
			t.Fatal(v, err, callbacks.Load(), calls.Load(), body.closes.Load())
		}
	})
	for _, mode := range []string{"bulk replaces invalid store", "empty bulk resets store and ignore", "last explicit false wins"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var options []image.DeleteImageOption
			path, code := deletePrefix+"images/fixed", 204
			switch mode {
			case "bulk replaces invalid store":
				store := "fast"
				options = []image.DeleteImageOption{image.WithDeleteImageStore(""), image.WithDeleteImageOpts(image.DeleteImageOpts{StoreID: &store})}
				path = deletePrefix + "stores/fast/fixed"
			case "empty bulk resets store and ignore":
				options = []image.DeleteImageOption{image.WithDeleteImageStore("fast"), image.WithDeleteImageIgnoreMissing(false), image.WithDeleteImageOpts(image.DeleteImageOpts{})}
				code = 404
			case "last explicit false wins":
				options = []image.DeleteImageOption{image.WithDeleteImageIgnoreMissing(true), image.WithDeleteImageIgnoreMissing(false)}
				code = 404
			}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = deleteTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Method != http.MethodDelete || r.URL.Path != path {
					t.Error(r.Method, r.URL)
				}
				return deleteWire(code, io.NopCloser(strings.NewReader("raw"))), nil
			})
			v, err := image.New(cloud.Client("image", deletePrefix)).DeleteImage(context.Background(), resource.ID("fixed"), options...)
			if calls.Load() != 1 || mode == "last explicit false wins" && (!errors.Is(err, resource.ErrNotFound) || v != nil) || mode == "empty bulk resets store and ignore" && (err != nil || v != nil) || mode == "bulk replaces invalid store" && (err != nil || v == nil || v.StoreID != "fast") {
				t.Fatal(v, err, calls.Load())
			}
		})
	}
	for _, change := range []string{"valid prefix and headers", "provider", "type"} {
		t.Run("source changed during name resolution "+change, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", deletePrefix)
			client.MoreHeaders = map[string]string{"X-Source": "captured"}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = deleteTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if !strings.HasPrefix(r.URL.Path, deletePrefix) || r.Header.Get("X-Source") != "captured" {
					t.Error("source changes retargeted request", r.URL, r.Header)
				}
				if r.Method == http.MethodDelete {
					if r.URL.Path != deletePrefix+"images/fixed" || r.Header.Get("X-Auth-Token") != "after-list" {
						t.Error(r.URL, r.Header)
					}
					return deleteWire(204, io.NopCloser(strings.NewReader(""))), nil
				}
				switch change {
				case "valid prefix and headers":
					client.ResourceBase = cloud.Server.URL + "/later/v2/"
					client.MoreHeaders = map[string]string{"X-Source": "later"}
				case "provider":
					client.ProviderClient = &gophercloud.ProviderClient{HTTPClient: cloud.Provider.HTTPClient}
				case "type":
					client.Type = "compute"
				}
				cloud.Provider.SetToken("after-list")
				return deleteWire(200, io.NopCloser(strings.NewReader(`{"images":[{"id":"fixed","name":"needle"}]}`))), nil
			})
			v, err := image.New(client).DeleteImage(context.Background(), resource.Name("needle"))
			if change == "valid prefix and headers" {
				if err != nil || v == nil || calls.Load() != 2 {
					t.Fatal(v, err, calls.Load())
				}
			} else {
				want := error(resource.ErrInvalidOption)
				if change == "type" {
					want = resource.ErrUnsupported
				}
				if !errors.Is(err, want) || v != nil || calls.Load() != 1 {
					t.Fatal(v, err, calls.Load())
				}
			}
		})
	}
}

func TestDeleteImageExactNameResolutionAndFailures(t *testing.T) {
	for _, mode := range []string{"one exact match across all pages", "ambiguous across pages", "default missing", "strict missing", "unsafe resolved ID", "HTTP404", "malformed JSON", "native model decode", "read failure", "transport nested404", "custom NotFound", "missing with provider changed", "missing with canceled context"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", deletePrefix)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("name resolution cause")
			var sequence []string
			var bodies []*deleteBody
			var options []image.DeleteImageOption
			if mode == "strict missing" {
				options = []image.DeleteImageOption{image.WithDeleteImageIgnoreMissing(false)}
			} else if mode == "one exact match across all pages" {
				options = []image.DeleteImageOption{image.WithDeleteImageStore("fast")}
			}
			cloud.Provider.HTTPClient.Transport = deleteTransport(func(r *http.Request) (*http.Response, error) {
				sequence = append(sequence, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
				if r.Method == http.MethodDelete {
					if r.URL.Path != deletePrefix+"stores/fast/fixed" || r.URL.RawQuery != "" {
						t.Error(r.Method, r.URL)
					}
					return deleteWire(204, io.NopCloser(strings.NewReader(""))), nil
				}
				if r.Method != http.MethodGet || r.URL.Path != deletePrefix+"images" {
					t.Error("name lookup performed another request", r.Method, r.URL)
				}
				if mode == "transport nested404" {
					return nil, errors.Join(cause, deleteNative404(r.URL.String()))
				}
				if mode == "custom NotFound" {
					return nil, &resource.NotFoundError{Resource: "image", Reference: "needle", Cause: cause}
				}
				code, payload := 200, `{"images":[]}`
				switch mode {
				case "one exact match across all pages", "ambiguous across pages":
					if r.URL.Query().Get("marker") == "second" {
						payload = `{"images":[{"id":"near","name":"needle-suffix"}]}`
						if mode == "ambiguous across pages" {
							payload = `{"images":[{"id":"other","name":"needle"}]}`
						}
					} else {
						if r.URL.Query().Get("name") != "needle" {
							t.Error("literal name query lost", r.URL)
						}
						// Native Glance continuation links are version-relative.
						payload = `{"images":[{"id":"fixed","name":"needle","status":"killed","protected":true}],"next":"/v2/images?marker=second"}`
					}
				case "unsafe resolved ID":
					payload = `{"images":[{"id":"../escape","name":"needle"}]}`
				case "HTTP404":
					code, payload = 404, "wire lookup is not logical absence"
				case "malformed JSON":
					payload = `{"images":`
				case "native model decode":
					payload = `{"images":[{"id":"fixed","name":12}]}`
				case "missing with provider changed":
					client.ProviderClient = &gophercloud.ProviderClient{HTTPClient: cloud.Provider.HTTPClient}
				}
				body := &deleteBody{Reader: strings.NewReader(payload)}
				if mode == "read failure" {
					body.Reader = &deleteReadFailure{prefix: []byte(`{"images":`), cause: cause}
				}
				if mode == "missing with canceled context" {
					body.onClose = func() { cancel(cause) }
				}
				bodies = append(bodies, body)
				return deleteWire(code, body), nil
			})
			v, err := image.New(client).DeleteImage(ctx, resource.Name("needle"), options...)
			switch mode {
			case "one exact match across all pages":
				if err != nil || v == nil || v.ImageID != "fixed" || v.StoreID != "fast" || len(sequence) != 3 || !strings.HasPrefix(sequence[2], "DELETE ") {
					t.Fatal(v, err, sequence)
				}
			case "default missing":
				if v != nil || err != nil || len(sequence) != 1 {
					t.Fatal(v, err, sequence)
				}
			default:
				if v != nil || err == nil || strings.Contains(strings.Join(sequence, "|"), "DELETE ") {
					t.Fatal(v, err, sequence)
				}
			}
			if mode == "ambiguous across pages" && (!errors.Is(err, resource.ErrAmbiguous) || len(sequence) != 2) || mode == "strict missing" && !errors.Is(err, resource.ErrNotFound) || mode == "unsafe resolved ID" && !errors.Is(err, resource.ErrInvalidOption) || mode == "HTTP404" && !gophercloud.ResponseCodeIs(err, 404) || mode == "missing with provider changed" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(v, err, sequence)
			}
			if (mode == "read failure" || mode == "transport nested404" || mode == "custom NotFound" || mode == "missing with canceled context") && !errors.Is(err, cause) {
				t.Fatal("lookup cause lost", err)
			}
			if mode == "missing with canceled context" && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation became ignored absence", err)
			}
			for _, body := range bodies {
				if body.closes.Load() != 1 {
					t.Fatal("native list body ownership changed", body.closes.Load())
				}
			}
		})
	}
}

func TestDeleteImageNotFoundPolicyAndClassification(t *testing.T) {
	for _, store := range []string{"", "fast"} {
		for _, ignore := range []bool{true, false} {
			t.Run(fmt.Sprintf("plain404 store=%s ignore=%v", store, ignore), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("image", deletePrefix)
				var calls, retries atomic.Int32
				body := &deleteBody{Reader: strings.NewReader("raw missing bytes")}
				wire := deleteWire(404, body)
				cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries.Add(1)
					return errors.New("handled404 must bypass native retry")
				}
				path := deletePrefix + "images/fixed"
				options := []image.DeleteImageOption{image.WithDeleteImageIgnoreMissing(ignore)}
				if store != "" {
					path = deletePrefix + "stores/fast/fixed"
					options = append(options, image.WithDeleteImageStore(store))
				}
				cloud.Provider.HTTPClient.Transport = deleteTransport(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					if r.Method != http.MethodDelete || r.URL.Path != path {
						t.Error(r.Method, r.URL)
					}
					return wire, nil
				})
				v, err := image.New(client).DeleteImage(context.Background(), resource.ID("fixed"), options...)
				if v != nil || calls.Load() != 1 || retries.Load() != 0 || body.closes.Load() != 1 {
					t.Fatal(v, err, calls.Load(), retries.Load(), body.closes.Load())
				}
				if ignore {
					if err != nil {
						t.Fatal(err)
					}
				} else {
					var missing *resource.NotFoundError
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.Is(err, resource.ErrNotFound) || !errors.As(err, &missing) || missing.Reference != "fixed" || !errors.As(err, &native) || native.Actual != 404 || native.Method != http.MethodDelete || native.URL != cloud.Server.URL+path || !reflect.DeepEqual(native.Expected, []int{204}) || string(native.Body) != "raw missing bytes" || native.ResponseHeader.Get("X-Request-Id") != "actual-delete" {
						t.Fatal(err, missing, native)
					}
					wire.Header.Set("X-Request-Id", "later-wire")
					if native.ResponseHeader.Get("X-Request-Id") != "actual-delete" {
						t.Fatal("native404 evidence aliases wire", native)
					}
				}
			})
		}
	}
	for _, mode := range []string{"read", "close", "read and close", "cancel during read", "cancel during close", "read close and cancel"} {
		t.Run("404 pipeline "+mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("404 read cause"), errors.New("404 close cause"), errors.New("404 custom cancellation")
			body := &deleteBody{Reader: strings.NewReader("raw missing bytes")}
			if strings.Contains(mode, "read") {
				body.Reader = &deleteReadFailure{prefix: []byte("partial missing"), cause: readCause}
			}
			if strings.Contains(mode, "close") {
				body.closeErr = closeCause
			}
			if strings.Contains(mode, "cancel") {
				body.onClose = func() { cancel(cancelCause) }
				if mode == "cancel during read" {
					body.Reader = &deleteReadFailure{prefix: []byte("partial missing"), cause: readCause, onRead: func() { cancel(cancelCause) }}
				}
			}
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries.Add(1)
				return errors.New("replay")
			}
			cloud.Provider.HTTPClient.Transport = deleteTransport(func(*http.Request) (*http.Response, error) { calls.Add(1); return deleteWire(404, body), nil })
			v, err := image.New(cloud.Client("image", deletePrefix)).DeleteImage(ctx, resource.ID("fixed"))
			var native gophercloud.ErrUnexpectedResponseCode
			var missing *resource.NotFoundError
			if v != nil || err == nil || !errors.As(err, &native) || native.Actual != 404 || !reflect.DeepEqual(native.Expected, []int{204}) || native.ResponseHeader.Get("X-Request-Id") != "actual-delete" || errors.As(err, &missing) || calls.Load() != 1 || retries.Load() != 0 || body.closes.Load() != 1 {
				t.Fatal(v, err, native, calls.Load(), retries.Load(), body.closes.Load())
			}
			wantBody := "raw missing bytes"
			if strings.Contains(mode, "read") {
				wantBody = "partial missing"
				if !errors.Is(err, readCause) {
					t.Fatal("read cause lost", err)
				}
			}
			if string(native.Body) != wantBody || strings.Contains(mode, "close") && !errors.Is(err, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
				t.Fatal(native, err)
			}
		})
	}
}

func TestDeleteImageAcceptedProofAndCancellation(t *testing.T) {
	for _, mode := range []string{"read", "close", "read and close", "read close and cancel", "close cancellation", "nested404 read"} {
		t.Run("204 pipeline "+mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("204 read cause"), errors.New("204 close cause"), errors.New("204 custom cancellation")
			body := &deleteBody{Reader: strings.NewReader("raw accepted bytes")}
			wantBody := "raw accepted bytes"
			if strings.Contains(mode, "read") {
				wantBody = "partial accepted"
				cause := error(readCause)
				if mode == "nested404 read" {
					cause = errors.Join(readCause, deleteNative404("body://nested-only"))
				}
				body.Reader = &deleteReadFailure{prefix: []byte(wantBody), cause: cause}
			}
			if strings.Contains(mode, "close") {
				body.closeErr = closeCause
			}
			if strings.Contains(mode, "cancel") {
				body.onClose = func() { cancel(cancelCause) }
			}
			wire := deleteWire(204, body)
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries.Add(1)
				return errors.New("must not replay accepted DELETE")
			}
			cloud.Provider.HTTPClient.Transport = deleteTransport(func(*http.Request) (*http.Response, error) { calls.Add(1); return wire, nil })
			v, err := image.New(cloud.Client("image", deletePrefix)).DeleteImage(ctx, resource.ID("fixed"), image.WithDeleteImageStore("fast"))
			var proof *resource.ResponseError
			if v == nil || v.ImageID != "fixed" || v.StoreID != "fast" || v.StatusCode != 204 || string(v.Body) != wantBody || v.Header.Get("X-Request-Id") != "actual-delete" || !errors.As(err, &proof) || proof.StatusCode != 204 || string(proof.Body) != wantBody || proof.Header.Get("X-Request-Id") != "actual-delete" || calls.Load() != 1 || retries.Load() != 0 || body.closes.Load() != 1 {
				t.Fatal(v, err, proof, calls.Load(), retries.Load(), body.closes.Load())
			}
			if strings.Contains(mode, "read") && !errors.Is(err, readCause) || strings.Contains(mode, "close") && !errors.Is(err, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
				t.Fatal("pipeline cause lost", err)
			}
			v.Body[0] = '!'
			v.Header.Set("X-Request-Id", "caller")
			wire.Header.Set("X-Request-Id", "wire-later")
			if string(proof.Body) != wantBody || proof.Header.Get("X-Request-Id") != "actual-delete" {
				t.Fatal("result/error evidence aliases", v, proof)
			}
		})
	}
	for _, mode := range []string{"parent deadline", "HTTP client timeout"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx := context.Background()
			if mode == "parent deadline" {
				parent, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
				ctx = parent
			} else {
				cloud.Provider.HTTPClient.Timeout = 100 * time.Millisecond
			}
			var body *deleteBody
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = deleteTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				body = &deleteBody{Reader: &deleteContextReader{ctx: r.Context(), prefix: []byte("accepted prefix")}}
				return deleteWire(204, body), nil
			})
			v, err := image.New(cloud.Client("image", deletePrefix)).DeleteImage(ctx, resource.ID("fixed"))
			var proof *resource.ResponseError
			if !errors.Is(err, context.DeadlineExceeded) || v == nil || v.StatusCode != 204 || string(v.Body) != "accepted prefix" || !errors.As(err, &proof) || proof.StatusCode != 204 || string(proof.Body) != "accepted prefix" || calls.Load() != 1 || body == nil || body.closes.Load() != 1 {
				t.Fatal(v, err, proof, calls.Load(), body)
			}
		})
	}
}

func TestDeleteImageProviderHooksAndRedirects(t *testing.T) {
	for _, backoffCode := range []int{429, 498} {
		t.Run(fmt.Sprintf("prebody reauth backoff%d retry fixed DELETE", backoffCode), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", deletePrefix)
			client.MoreHeaders = map[string]string{"X-Source": "captured"}
			cloud.Provider.SetToken("initial")
			var calls, reauth, backoff, retries atomic.Int32
			var bodies []*deleteBody
			cloud.Provider.ReauthFunc = func(context.Context) error { reauth.Add(1); cloud.Provider.SetToken("reauth"); return nil }
			cloud.Provider.RetryBackoffFunc = func(_ context.Context, native *gophercloud.ErrUnexpectedResponseCode, _ error, _ uint) error {
				backoff.Add(1)
				if native.Actual != backoffCode || native.Method != http.MethodDelete || native.URL != cloud.Server.URL+deletePrefix+"stores/fast/fixed" {
					t.Error(native)
				}
				cloud.Provider.SetToken("backoff")
				return nil
			}
			cloud.Provider.RetryFunc = func(_ context.Context, method, target string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				if method != http.MethodDelete || target != cloud.Server.URL+deletePrefix+"stores/fast/fixed" || !gophercloud.ResponseCodeIs(err, 503) {
					return err
				}
				cloud.Provider.SetToken("retry")
				client.ResourceBase = cloud.Server.URL + "/later/v2/"
				client.MoreHeaders = map[string]string{"X-Source": "later"}
				return nil
			}
			cloud.Provider.HTTPClient.Transport = deleteTransport(func(r *http.Request) (*http.Response, error) {
				n := int(calls.Add(1)) - 1
				codes := []int{401, backoffCode, 503, 204}
				tokens := []string{"initial", "reauth", "backoff", "retry"}
				if n >= len(codes) {
					return nil, errors.New("unexpected replay")
				}
				if r.Method != http.MethodDelete || r.URL.Path != deletePrefix+"stores/fast/fixed" || r.URL.RawQuery != "" || r.Body != nil || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != tokens[n] {
					t.Error(r.Method, r.URL, r.Header)
				}
				body := &deleteBody{Reader: strings.NewReader(fmt.Sprint(codes[n]))}
				bodies = append(bodies, body)
				return deleteWire(codes[n], body), nil
			})
			v, err := image.New(client).DeleteImage(context.Background(), resource.ID("fixed"), image.WithDeleteImageStore("fast"))
			if err != nil || v == nil || v.StatusCode != 204 || string(v.Body) != "204" || calls.Load() != 4 || reauth.Load() != 1 || backoff.Load() != 1 || retries.Load() != 1 || cloud.Provider.ReauthFunc == nil || cloud.Provider.RetryBackoffFunc == nil || cloud.Provider.RetryFunc == nil {
				t.Fatal(v, err, calls.Load(), reauth.Load(), backoff.Load(), retries.Load())
			}
			for _, body := range bodies {
				if body.closes.Load() != 1 {
					t.Fatal("retry response ownership changed", body.closes.Load())
				}
			}
		})
	}
	for _, field := range []string{"KeepResponseBody", "JSONResponse", "JSONBody", "RawBody"} {
		t.Run("retry cannot change owned "+field, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, callbacks atomic.Int32
			body := &deleteBody{Reader: strings.NewReader("prebody failure")}
			cloud.Provider.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, err error, _ uint) error {
				callbacks.Add(1)
				if method != http.MethodDelete || target != cloud.Server.URL+deletePrefix+"images/fixed" || !gophercloud.ResponseCodeIs(err, 503) || !options.KeepResponseBody || options.JSONResponse != nil || options.JSONBody != nil || options.RawBody != nil {
					t.Error(method, target, options, err)
				}
				switch field {
				case "KeepResponseBody":
					options.KeepResponseBody = false
				case "JSONResponse":
					options.JSONResponse = new(any)
				case "JSONBody":
					options.JSONBody = map[string]string{"unexpected": "body"}
				case "RawBody":
					options.RawBody = strings.NewReader("unexpected body")
				}
				return nil
			}
			cloud.Provider.HTTPClient.Transport = deleteTransport(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodDelete || r.URL.Path != deletePrefix+"images/fixed" || r.URL.RawQuery != "" || r.Body != nil {
					t.Error("retry changed deletion request", r.Method, r.URL, r.Body)
				}
				if calls.Add(1) == 1 {
					return deleteWire(503, body), nil
				}
				return deleteWire(204, io.NopCloser(strings.NewReader("unexpected replay"))), nil
			})
			v, err := image.New(cloud.Client("image", deletePrefix)).DeleteImage(context.Background(), resource.ID("fixed"))
			var native gophercloud.ErrUnexpectedResponseCode
			if v != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != "prebody failure" || native.ResponseHeader.Get("X-Request-Id") != "actual-delete" || calls.Load() != 1 || callbacks.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(v, err, native, calls.Load(), callbacks.Load(), body.closes.Load())
			}
		})
	}
	for _, mode := range []string{"transport nested404", "RetryFunc nested404", "reauth nested404", "retry changes accepted codes"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", deletePrefix)
			cause := errors.New("configured policy cause")
			var calls, callbacks atomic.Int32
			var bodies []*deleteBody
			if mode == "reauth nested404" {
				cloud.Provider.ReauthFunc = func(context.Context) error {
					callbacks.Add(1)
					return errors.Join(cause, deleteNative404("reauth://not-wire"))
				}
			} else if mode != "transport nested404" {
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, err error, _ uint) error {
					callbacks.Add(1)
					if mode == "retry changes accepted codes" {
						options.OkCodes = []int{202}
						return nil
					}
					return errors.Join(cause, deleteNative404("retry://not-wire"))
				}
			}
			cloud.Provider.HTTPClient.Transport = deleteTransport(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if mode == "transport nested404" {
					return nil, errors.Join(cause, deleteNative404(r.URL.String()))
				}
				code := 503
				if mode == "reauth nested404" {
					code = 401
				} else if mode == "retry changes accepted codes" && n == 2 {
					code = 202
				}
				body := &deleteBody{Reader: strings.NewReader(fmt.Sprint(code))}
				bodies = append(bodies, body)
				return deleteWire(code, body), nil
			})
			v, err := image.New(client).DeleteImage(context.Background(), resource.ID("fixed"))
			if v != nil || err == nil {
				t.Fatal(v, err, calls.Load(), callbacks.Load())
			}
			if mode == "retry changes accepted codes" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{204}) || string(native.Body) != "202" || calls.Load() != 2 {
					t.Fatal("callback expanded public acknowledgement policy", err, native, calls.Load())
				}
			} else if mode == "reauth nested404" {
				var native *gophercloud.ErrUnableToReauthenticate
				if !errors.As(err, &native) || !errors.Is(native.ErrReauth, cause) || !gophercloud.ResponseCodeIs(native.ErrReauth, 404) || !gophercloud.ResponseCodeIs(native.ErrOriginal, 401) || calls.Load() != 1 || callbacks.Load() != 1 {
					t.Fatal("native reauth fields lost", err, native, calls.Load(), callbacks.Load())
				}
			} else if !errors.Is(err, cause) || !gophercloud.ResponseCodeIs(err, 404) || calls.Load() != 1 {
				t.Fatal("nested404 policy error became absence", err, calls.Load(), callbacks.Load())
			}
			for _, body := range bodies {
				if body.closes.Load() != 1 {
					t.Fatal("native rejected body closed twice", body.closes.Load())
				}
			}
		})
	}
	for _, code := range []int{301, 302, 303, 307, 308} {
		t.Run(fmt.Sprintf("redirect%d", code), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, redirects atomic.Int32
			cloud.Provider.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects.Add(1); return nil }
			body := &deleteBody{Reader: strings.NewReader("redirect proof")}
			cloud.Provider.HTTPClient.Transport = deleteTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Method != http.MethodDelete || r.URL.Path != deletePrefix+"images/fixed" {
					t.Error(r.Method, r.URL)
				}
				wire := deleteWire(code, body)
				wire.Header.Set("Location", "https://foreign.invalid/redirected")
				return wire, nil
			})
			v, err := image.New(cloud.Client("image", deletePrefix)).DeleteImage(context.Background(), resource.ID("fixed"))
			if v != nil || !gophercloud.ResponseCodeIs(err, code) || calls.Load() != 1 || redirects.Load() != 0 || body.closes.Load() != 1 {
				t.Fatal(v, err, calls.Load(), redirects.Load(), body.closes.Load())
			}
			if err := cloud.Provider.HTTPClient.CheckRedirect(nil, nil); err != nil || redirects.Load() != 1 {
				t.Fatal("caller redirect policy changed", err, redirects.Load())
			}
		})
	}
}

func TestDeleteImageNativeDeleteCompatibility(t *testing.T) {
	for _, code := range []int{200, 202, 403, 409} {
		t.Run(fmt.Sprintf("workflow rejects%d", code), func(t *testing.T) {
			cloud := testcloud.New(t)
			body := &deleteBody{Reader: strings.NewReader("server-owned error or unexpected acknowledgement")}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = deleteTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Method != http.MethodDelete || r.URL.Path != deletePrefix+"stores/fast/fixed" {
					t.Error(r.Method, r.URL)
				}
				return deleteWire(code, body), nil
			})
			v, err := image.New(cloud.Client("image", deletePrefix)).DeleteImage(context.Background(), resource.ID("fixed"), image.WithDeleteImageStore("fast"))
			var native gophercloud.ErrUnexpectedResponseCode
			if v != nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != "server-owned error or unexpected acknowledgement" || native.ResponseHeader.Get("X-Request-Id") != "actual-delete" || calls.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(v, err, native, calls.Load(), body.closes.Load())
			}
		})
	}
	for _, code := range []int{202, 204, 404} {
		for _, mode := range []string{"native Delete", "native Remove", "manual Collection.Delete"} {
			t.Run(fmt.Sprintf("%s%d", mode, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("image", deletePrefix)
				body := &deleteBody{Reader: strings.NewReader("native raw bytes")}
				var calls atomic.Int32
				cloud.Provider.HTTPClient.Transport = deleteTransport(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					if r.Method != http.MethodDelete || r.URL.Path != deletePrefix+"images/fixed" {
						t.Error(r.Method, r.URL)
					}
					return deleteWire(code, body), nil
				})
				var err error
				switch mode {
				case "native Delete":
					err = sdkimages.New(client).Delete(context.Background(), "fixed")
				case "native Remove":
					err = sdkimages.New(client).Remove(context.Background(), resource.ID("fixed"))
				case "manual Collection.Delete":
					err = image.New(client).Images.Delete(context.Background(), resource.ID("fixed"))
				}
				if code == 404 && mode == "native Delete" {
					if !gophercloud.ResponseCodeIs(err, 404) {
						t.Fatal(err)
					}
				} else if err != nil {
					t.Fatal("native ABI changed", err)
				}
				if calls.Load() != 1 || body.closes.Load() != 1 {
					t.Fatal(calls.Load(), body.closes.Load())
				}
			})
		}
	}
}
