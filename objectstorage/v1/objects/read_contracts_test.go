package objects_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type objectReadTransport func(*http.Request) (*http.Response, error)

func (f objectReadTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type objectReadReader func([]byte) (int, error)

func (f objectReadReader) Read(p []byte) (int, error) { return f(p) }

type objectReadBody struct {
	reader                            io.Reader
	reads, closes, maxRead, shortcuts int
	closeErr                          error
	beforeClose                       func()
}

func (b *objectReadBody) Read(p []byte) (int, error) {
	b.reads++
	if len(p) > b.maxRead {
		b.maxRead = len(p)
	}
	return b.reader.Read(p)
}
func (b *objectReadBody) Close() error {
	b.closes++
	if b.beforeClose != nil {
		b.beforeClose()
	}
	return b.closeErr
}
func (b *objectReadBody) WriteTo(io.Writer) (int64, error) {
	b.shortcuts++
	return 0, errors.New("unexpected WriterTo shortcut")
}

type objectReadWriter struct {
	bytes.Buffer
	write                                                          func([]byte) (int, error)
	writes, maxWrite, closes, flushes, seeks, truncates, shortcuts int
}

func (w *objectReadWriter) Write(p []byte) (int, error) {
	w.writes++
	if len(p) > w.maxWrite {
		w.maxWrite = len(p)
	}
	if w.write != nil {
		return w.write(p)
	}
	return w.Buffer.Write(p)
}
func (w *objectReadWriter) Close() error                   { w.closes++; return nil }
func (w *objectReadWriter) Flush() error                   { w.flushes++; return nil }
func (w *objectReadWriter) Seek(int64, int) (int64, error) { w.seeks++; return 0, nil }
func (w *objectReadWriter) Truncate(int64) error           { w.truncates++; return nil }
func (w *objectReadWriter) ReadFrom(io.Reader) (int64, error) {
	w.shortcuts++
	return 0, errors.New("unexpected ReaderFrom shortcut")
}
func objectReadClient(f objectReadTransport) *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: f}}
	p.UseTokenLock()
	p.SetToken("initial-token")
	return &gophercloud.ServiceClient{ProviderClient: p, Endpoint: "https://swift.invalid/v1/A/", Type: "object-store"}
}
func objectReadWire(r *http.Request, status int, h http.Header, body *objectReadBody) *http.Response {
	return &http.Response{StatusCode: status, Header: h, Body: body, Request: r}
}
func objectReadBytes(data []byte) *objectReadBody {
	return &objectReadBody{reader: bytes.NewReader(data)}
}
func objectReadProof(t *testing.T, err error, status int) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != status {
		t.Fatalf("error=%v; want actual response %d", err, status)
	}
	return proof
}
func objectReadBorrowed(t *testing.T, w *objectReadWriter) {
	t.Helper()
	if w.closes+w.flushes+w.seeks+w.truncates+w.shortcuts != 0 {
		t.Fatalf("borrowed writer lifecycle/shortcut calls: %+v", w)
	}
}

func TestObjectReadContractsWireAndBytes(t *testing.T) {
	t.Run("R01-three-binary-GETs", func(t *testing.T) {
		data := []byte{'a', 0, 0xff, 0xe9, '\n'}
		for _, mode := range []string{"buffer", "writer", "stream"} {
			t.Run(mode, func(t *testing.T) {
				body := objectReadBytes(data)
				requests := 0
				client := objectReadClient(func(r *http.Request) (*http.Response, error) {
					requests++
					if r.Method != "GET" || r.Body != nil || r.URL.EscapedPath() != "/v1/A/container/object" || r.URL.RawQuery != "" {
						t.Errorf("request=%s %s body=%v", r.Method, r.URL, r.Body)
					}
					return objectReadWire(r, 200, http.Header{"X-Object-Meta-Label": {"literal"}}, body), nil
				})
				api := objects.New(client)
				switch mode {
				case "buffer":
					v, err := api.GetObject(context.Background(), "container", "object")
					if err != nil || v == nil || !bytes.Equal(v.Body, data) || !v.Complete || v.NotModified || v.StatusCode != 200 || v.Metadata.Values["label"] != "literal" {
						t.Fatalf("buffer=%+v err=%v", v, err)
					}
				case "writer":
					w := &objectReadWriter{}
					v, err := api.DownloadObject(context.Background(), "container", "object", w)
					if err != nil || v == nil || !bytes.Equal(w.Bytes(), data) || v.BytesWritten != int64(len(data)) || !v.Complete || v.NotModified || v.StatusCode != 200 {
						t.Fatalf("writer=%+v data=%x err=%v", v, w.Bytes(), err)
					}
					objectReadBorrowed(t, w)
				case "stream":
					v, err := api.StreamObject(context.Background(), "container", "object")
					if err != nil || v == nil || v.Body == nil || body.reads != 0 || v.Complete || v.BytesRead != 0 {
						t.Fatalf("open=%+v reads=%d err=%v", v, body.reads, err)
					}
					got, err := io.ReadAll(v.Body)
					if err != nil || !bytes.Equal(got, data) || !v.Complete || v.BytesRead != int64(len(data)) || v.StatusCode != 200 {
						t.Fatalf("stream=%+v bytes=%x err=%v", v, got, err)
					}
					if err := v.Body.Close(); err != nil {
						t.Fatal(err)
					}
				}
				if requests != 1 || body.closes != 1 {
					t.Fatalf("requests=%d closes=%d", requests, body.closes)
				}
			})
		}
	})
	t.Run("R02-opaque-single-and-multipart206", func(t *testing.T) {
		for _, data := range [][]byte{[]byte("range"), []byte("--boundary\r\nContent-Range: bytes 1-2/9\r\n\r\n\xff\x00\r\n--boundary--\r\n")} {
			body := objectReadBytes(data)
			api := objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) {
				if r.Header.Get("Range") != "opaque server range" {
					t.Errorf("Range=%q", r.Header.Get("Range"))
				}
				return objectReadWire(r, 206, http.Header{"Content-Range": {"bytes 1-2/9"}, "Content-Length": {"999"}, "Etag": {"not-a-payload-checksum"}, "Content-Type": {"multipart/byteranges; boundary=boundary"}}, body), nil
			}))
			v, err := api.GetObject(context.Background(), "c", "o", objects.WithObjectReadRange("opaque server range"))
			if err != nil || v == nil || v.StatusCode != 206 || !v.Complete || !bytes.Equal(v.Body, data) || v.Header.Get("Content-Range") != "bytes 1-2/9" || *v.Metadata.ETag != "not-a-payload-checksum" {
				t.Fatalf("206=%+v err=%v", v, err)
			}
			if body.closes != 1 {
				t.Fatalf("closes=%d", body.closes)
			}
		}
	})
	t.Run("R03-304-no-read-or-write", func(t *testing.T) {
		for _, mode := range []string{"buffer", "writer", "stream"} {
			body := &objectReadBody{reader: objectReadReader(func([]byte) (int, error) { t.Error("304 body was read"); return 0, errors.New("304 must not read") })}
			api := objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) {
				return objectReadWire(r, 304, http.Header{"Etag": {"observed304"}, "Content-Length": {"42"}}, body), nil
			}))
			switch mode {
			case "buffer":
				v, err := api.GetObject(context.Background(), "c", "o")
				if err != nil || v == nil || !v.NotModified || !v.Complete || v.StatusCode != 304 || len(v.Body) != 0 || *v.Metadata.ContentLength != 42 {
					t.Fatalf("304=%+v err=%v", v, err)
				}
			case "writer":
				w := &objectReadWriter{}
				v, err := api.DownloadObject(context.Background(), "c", "o", w)
				if err != nil || v == nil || !v.NotModified || !v.Complete || v.BytesWritten != 0 || w.writes != 0 {
					t.Fatalf("304=%+v writes=%d err=%v", v, w.writes, err)
				}
				objectReadBorrowed(t, w)
			case "stream":
				v, err := api.StreamObject(context.Background(), "c", "o")
				if err != nil || v == nil || v.Body == nil || !v.NotModified || !v.Complete || v.BytesRead != 0 {
					t.Fatalf("304=%+v err=%v", v, err)
				}
				if n, err := v.Body.Read(make([]byte, 3)); n != 0 || err != io.EOF {
					t.Fatalf("304 Read=%d,%v", n, err)
				}
				if err := v.Body.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if body.reads != 0 || body.closes != 1 {
				t.Fatalf("%s reads=%d closes=%d", mode, body.reads, body.closes)
			}
		}
		closeCause := errors.New("304 Close")
		body := &objectReadBody{reader: bytes.NewReader(nil), closeErr: closeCause}
		api := objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) { return objectReadWire(r, 304, nil, body), nil }))
		v, err := api.StreamObject(context.Background(), "c", "o")
		if v == nil || !v.NotModified || !v.Complete || !errors.Is(err, closeCause) || body.closes != 1 {
			t.Fatalf("304 Close=%+v err=%v closes=%d", v, err, body.closes)
		}
		objectReadProof(t, err, 304)
	})
	t.Run("R04-literal-path-and-query-once", func(t *testing.T) {
		container, object := "c %雪", "o%2F/雪 :?#"
		query := url.Values{"filename": {" a +雪/?#"}, "multipart-manifest": {"opaque"}, "symlink": {"get"}, "version-id": {"v %/雪"}}
		var count atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count.Add(1)
			want := "/reverse%20prefix/v1/A/" + url.PathEscape(container) + "/" + url.PathEscape(object)
			if r.Method != "GET" || r.URL.EscapedPath() != want || !reflect.DeepEqual(r.URL.Query(), query) {
				t.Errorf("request=%s %s; want %s %v", r.Method, r.URL, want, query)
			}
			w.Header().Set("X-Object-Meta-Unicode", "雪")
			_, _ = w.Write([]byte{0xff, 0, 'x'})
		}))
		defer server.Close()
		client := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{HTTPClient: http.Client{}}, Endpoint: server.URL + "/catalog/v1/A/", ResourceBase: server.URL + "/reverse%20prefix/v1/A/", Type: "object-store"}
		v, err := objects.New(client).GetObject(context.Background(), container, object, objects.WithObjectReadFilename(query.Get("filename")), objects.WithObjectReadMultipartManifest("opaque"), objects.WithObjectReadSymlink("get"), objects.WithObjectReadVersionID(query.Get("version-id")))
		if err != nil || v == nil || !bytes.Equal(v.Body, []byte{0xff, 0, 'x'}) || v.Metadata.Values["unicode"] != "雪" || count.Load() != 1 {
			t.Fatalf("literal=%+v err=%v requests=%d", v, err, count.Load())
		}
	})
}

func TestObjectReadContractsOptionsAndPreflight(t *testing.T) {
	t.Run("R05-options-snapshots-and-parallel-reuse", func(t *testing.T) {
		newest := false
		d := time.Date(2020, 2, 3, 4, 5, 6, 0, time.FixedZone("custom", 9*3600))
		headers := map[string]string{"x-call": "factory"}
		factory := objects.WithObjectReadOpts(objects.ObjectReadOpts{Headers: headers, Newest: &newest, IfModifiedSince: &d, BufferSize: 2})
		headers["x-call"], newest, d = "mutated", true, time.Time{}
		var requests atomic.Int32
		client := objectReadClient(func(r *http.Request) (*http.Response, error) {
			requests.Add(1)
			if r.Header.Get("X-Call") != "final" || !strings.EqualFold(r.Header.Get("X-Newest"), "false") || r.Header.Get("If-Modified-Since") != "Sun, 02 Feb 2020 19:05:06 GMT" {
				t.Errorf("snapshot headers=%v", r.Header)
			}
			return objectReadWire(r, 200, nil, objectReadBytes([]byte("ok"))), nil
		})
		api := objects.New(client)
		var wg sync.WaitGroup
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, err := api.GetObject(context.Background(), "c", "o", factory, objects.WithObjectReadHeader("X-Call", "final"))
				if err != nil || v == nil || string(v.Body) != "ok" {
					t.Errorf("parallel result=%+v err=%v", v, err)
				}
			}()
		}
		wg.Wait()
		if requests.Load() != 4 {
			t.Fatalf("requests=%d", requests.Load())
		}
		var held *objects.ObjectReadOpts
		callbacks := 0
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		client.HTTPClient.Transport = objectReadTransport(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Call") != "before-copy" || r.Header.Get("X-Source") != "captured" {
				t.Errorf("callback headers=%v", r.Header)
			}
			return objectReadWire(r, 200, nil, objectReadBytes(nil)), nil
		})
		v, err := api.GetObject(context.Background(), "c", "o", func(o *objects.ObjectReadOpts) error {
			callbacks++
			o.Headers["X-Call"] = "before-copy"
			held = o
			client.MoreHeaders["X-Source"] = "later-valid-source"
			return nil
		}, func(*objects.ObjectReadOpts) error {
			callbacks++
			held.Headers["X-Call"] = "stale-handle"
			return nil
		})
		if err != nil || v == nil || callbacks != 2 {
			t.Fatalf("callbacks=%d value=%+v err=%v", callbacks, v, err)
		}
	})
	t.Run("R06-typed-conditions-and-clear-helpers", func(t *testing.T) {
		modified := time.Date(2024, 1, 2, 3, 4, 5, 0, time.FixedZone("offset", -7*3600))
		unmodified := modified.Add(time.Hour)
		calls := 0
		api := objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				want := map[string]string{"If-Match": "\"one\", \"two\"", "If-None-Match": "literal\tvalue", "If-Modified-Since": modified.UTC().Format(http.TimeFormat), "If-Unmodified-Since": unmodified.UTC().Format(http.TimeFormat), "Range": "not a locally parsed range", "X-Call": "one", "X-Other": "two"}
				for k, v := range want {
					if r.Header.Get(k) != v {
						t.Errorf("%s=%q want %q", k, r.Header.Get(k), v)
					}
				}
				if !strings.EqualFold(r.Header.Get("X-Newest"), "true") {
					t.Errorf("newest=%q", r.Header.Get("X-Newest"))
				}
			} else if r.Header.Get("X-Newest") != "" || r.Header.Get("If-Modified-Since") != "" || r.Header.Get("If-Unmodified-Since") != "" || r.Header.Get("If-Match") != "" {
				t.Errorf("cleared headers=%v", r.Header)
			}
			return objectReadWire(r, 200, nil, objectReadBytes(nil)), nil
		}))
		_, err := api.GetObject(context.Background(), "c", "o", objects.WithObjectReadHeaders(map[string]string{"X-Call": "one", "X-Other": "two"}), objects.WithObjectReadNewest(true), objects.WithObjectReadIfMatch("\"one\", \"two\""), objects.WithObjectReadIfNoneMatch("literal\tvalue"), objects.WithObjectReadIfModifiedSince(modified), objects.WithObjectReadIfUnmodifiedSince(unmodified), objects.WithObjectReadRange("not a locally parsed range"), objects.WithObjectReadBufferSize(1))
		if err != nil {
			t.Fatal(err)
		}
		_, err = api.GetObject(context.Background(), "c", "o", objects.WithObjectReadNewest(true), objects.WithObjectReadIfModifiedSince(modified), objects.WithObjectReadIfUnmodifiedSince(unmodified), objects.WithoutObjectReadNewest(), objects.WithoutObjectReadIfModifiedSince(), objects.WithoutObjectReadIfUnmodifiedSince(), objects.WithObjectReadIfMatch("old"), objects.WithObjectReadOpts(objects.ObjectReadOpts{}))
		if err != nil || calls != 2 {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	})
	t.Run("R07-owned-header-boundaries", func(t *testing.T) {
		reserved := []string{"Range", "If-Range", "If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since", "X-Newest", "X-Auth-Token", "X-Service-Token", "Cookie", "Content-Length", "Transfer-Encoding", "X-Object-Meta-Key", "X-Remove-Object-Meta-Key"}
		for _, key := range reserved {
			for _, source := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/source=%t", key, source), func(t *testing.T) {
					requests, callbacks := 0, 0
					client := objectReadClient(func(r *http.Request) (*http.Response, error) {
						requests++
						return objectReadWire(r, 200, nil, objectReadBytes(nil)), nil
					})
					var options []objects.ObjectReadOption
					if source {
						client.MoreHeaders = map[string]string{key: "x"}
					} else {
						options = append(options, objects.WithObjectReadHeader(key, "x"))
					}
					options = append(options, func(*objects.ObjectReadOpts) error { callbacks++; return nil })
					v, err := objects.New(client).GetObject(context.Background(), "c", "o", options...)
					if v != nil || !errors.Is(err, resource.ErrInvalidOption) || requests != 0 || (source && callbacks != 0) {
						t.Fatalf("value=%+v err=%v requests=%d callbacks=%d", v, err, requests, callbacks)
					}
				})
			}
		}
		api := objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Literal") != " 雪\tvalue " || r.Header.Get("If-Match") != " quoted literal " {
				t.Errorf("literal headers=%v", r.Header)
			}
			return objectReadWire(r, 200, nil, objectReadBytes(nil)), nil
		}))
		if _, err := api.GetObject(context.Background(), "c", "o", objects.WithObjectReadHeader("X-Literal", " 雪\tvalue "), objects.WithObjectReadIfMatch(" quoted literal ")); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("R08-zero-HTTP-complete-preflight", func(t *testing.T) {
		cause := errors.New("preflight canceled")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		cancel(cause)
		requests := 0
		client := objectReadClient(func(r *http.Request) (*http.Response, error) {
			requests++
			return objectReadWire(r, 200, nil, objectReadBytes(nil)), nil
		})
		api := objects.New(client)
		badDate := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
		cases := []struct {
			name              string
			ctx               context.Context
			container, object string
			options           []objects.ObjectReadOption
		}{
			{"nil-context", nil, "c", "o", nil}, {"canceled", ctx, "c", "o", nil}, {"container-dot", context.Background(), ".", "o", nil}, {"container-slash", context.Background(), "a/b", "o", nil}, {"object-dot-segment", context.Background(), "c", "a/../b", nil}, {"object-backslash", context.Background(), "c", "a\\b", nil}, {"object-control", context.Background(), "c", "a\x7fb", nil}, {"object-invalidUTF8", context.Background(), "c", string([]byte{0xff}), nil},
			{"nil-option", context.Background(), "c", "o", []objects.ObjectReadOption{nil}}, {"negative-buffer", context.Background(), "c", "o", []objects.ObjectReadOption{objects.WithObjectReadBufferSize(-1)}}, {"large-buffer", context.Background(), "c", "o", []objects.ObjectReadOption{objects.WithObjectReadBufferSize(16*1024*1024 + 1)}}, {"date-year", context.Background(), "c", "o", []objects.ObjectReadOption{objects.WithObjectReadIfModifiedSince(badDate)}}, {"query-control", context.Background(), "c", "o", []objects.ObjectReadOption{objects.WithObjectReadVersionID("v\n")}}, {"query-invalidUTF8", context.Background(), "c", "o", []objects.ObjectReadOption{objects.WithObjectReadFilename(string([]byte{0xff}))}}, {"typed-field-control", context.Background(), "c", "o", []objects.ObjectReadOption{objects.WithObjectReadRange("x\r\n")}},
		}
		for _, c := range cases {
			v, err := api.GetObject(c.ctx, c.container, c.object, c.options...)
			if v != nil || err == nil || requests != 0 {
				t.Fatalf("%s value=%+v err=%v requests=%d", c.name, v, err, requests)
			}
			if c.name == "canceled" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatalf("custom cancellation=%v", err)
			}
		}
		var nilWriter *objectReadWriter
		if v, err := api.DownloadObject(context.Background(), "c", "o", nilWriter); v != nil || !errors.Is(err, resource.ErrInvalidOption) || requests != 0 {
			t.Fatalf("nil writer=%+v err=%v requests=%d", v, err, requests)
		}
		var nilAPI *objects.API
		if v, err := nilAPI.GetObject(context.Background(), "c", "o"); v != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil API=%+v err=%v", v, err)
		}
		for _, mutate := range []func(){func() { client.ResourceBase = "https://other.invalid/v1/A/" }, func() { client.ResourceBase = client.Endpoint; client.Type = "image" }, func() { client.Type = "object-store"; client.Endpoint = "https://swift.invalid/v1/A/?query=x" }} {
			mutate()
			if v, err := api.StreamObject(context.Background(), "c", "o"); v != nil || err == nil || requests != 0 {
				t.Fatalf("bad source=%+v err=%v requests=%d", v, err, requests)
			}
		}
	})
}

func TestObjectReadContractsMetadataAndEvidence(t *testing.T) {
	t.Run("R09-header-presence-precision-and-independence", func(t *testing.T) {
		for _, absent := range []bool{true, false} {
			header := http.Header{}
			if !absent {
				header = http.Header{"Content-Length": {"9223372036854775807"}, "Etag": {"literal-composite-etag"}, "Last-Modified": {"not locally parsed"}, "X-Timestamp": {"0000.10000"}, "Content-Disposition": {""}, "X-Object-Meta-Case": {"雪"}, "Content-Range": {"passive"}}
			}
			body := objectReadBytes([]byte{0xff, 0})
			api := objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) { return objectReadWire(r, 200, header, body), nil }))
			v, err := api.GetObject(context.Background(), "c", "o")
			if err != nil || v == nil || v.Metadata == nil || !v.Complete {
				t.Fatalf("projection=%+v err=%v", v, err)
			}
			if absent {
				if v.Metadata.ContentLength != nil || v.Metadata.ETag != nil || v.Metadata.LastModified != nil || len(v.Metadata.Values) != 0 {
					t.Fatalf("absent metadata=%+v", v.Metadata)
				}
			} else {
				if *v.Metadata.ContentLength != int64(9223372036854775807) || *v.Metadata.LastModified != "not locally parsed" || *v.Metadata.Timestamp != "0000.10000" || *v.Metadata.ContentDisposition != "" || v.Metadata.Values["case"] != "雪" || v.Header.Get("Content-Range") != "passive" {
					t.Fatalf("literal metadata=%+v", v.Metadata)
				}
				header.Set("ETag", "changed-wire-header")
				if v.Header.Get("ETag") != "literal-composite-etag" {
					t.Fatal("exported header aliases transport header")
				}
				v.Header.Set("ETag", "changed-result-header")
				if *v.Metadata.ETag != "literal-composite-etag" {
					t.Fatal("metadata aliases exported header")
				}
			}
			if !bytes.Equal(v.Body, []byte{0xff, 0}) || body.closes != 1 {
				t.Fatalf("body=%x closes=%d", v.Body, body.closes)
			}
		}
	})
	t.Run("R10-atomic-projection-before-content", func(t *testing.T) {
		for name, header := range map[string]http.Header{
			"fractional-count": {"Content-Length": {"1.0"}}, "overflow-count": {"Content-Length": {"9223372036854775808"}}, "multiple-count": {"Content-Length": {"1", "2"}}, "duplicate-custom-case": {"X-Object-Meta-Key": {"a"}, "x-object-meta-key": {"b"}}, "nonASCII-suffix": {"X-Object-Meta-K": {"x"}}, "empty-suffix": {"X-Object-Meta-": {"x"}},
		} {
			for _, mode := range []string{"buffer", "writer", "stream"} {
				t.Run(name+"/"+mode, func(t *testing.T) {
					closeCause := errors.New("projection Close")
					body := &objectReadBody{reader: objectReadReader(func([]byte) (int, error) { t.Error("projection failure drained content"); return 0, io.EOF }), closeErr: closeCause}
					api := objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) { return objectReadWire(r, 206, header, body), nil }))
					var err error
					switch mode {
					case "buffer":
						v, e := api.GetObject(context.Background(), "c", "o")
						err = e
						if v == nil || v.StatusCode != 206 || v.Metadata != nil || v.Body != nil || v.Complete {
							t.Fatalf("buffer projection=%+v", v)
						}
					case "writer":
						w := &objectReadWriter{}
						v, e := api.DownloadObject(context.Background(), "c", "o", w)
						err = e
						if v == nil || v.StatusCode != 206 || v.Metadata != nil || v.BytesWritten != 0 || w.writes != 0 || v.Complete {
							t.Fatalf("writer projection=%+v writes=%d", v, w.writes)
						}
						objectReadBorrowed(t, w)
					case "stream":
						v, e := api.StreamObject(context.Background(), "c", "o")
						err = e
						if v == nil || v.StatusCode != 206 || v.Metadata != nil || v.Body != nil || v.Complete {
							t.Fatalf("stream projection=%+v", v)
						}
					}
					proof := objectReadProof(t, err, 206)
					if !errors.Is(err, closeCause) || !reflect.DeepEqual(proof.Header, header) || len(proof.Body) != 0 || body.reads != 0 || body.closes != 1 {
						t.Fatalf("proof=%+v err=%v reads=%d closes=%d", proof, err, body.reads, body.closes)
					}
				})
			}
		}
	})
	t.Run("R11-partial-buffer-read-close-and-context-causes", func(t *testing.T) {
		readCause, closeCause, cancelCause := errors.New("partial Read"), errors.New("partial Close"), errors.New("custom cancellation")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		body := &objectReadBody{reader: objectReadReader(func(p []byte) (int, error) { return copy(p, []byte{0, 0xff, 'p'}), readCause }), closeErr: closeCause}
		requests, retries := 0, 0
		client := objectReadClient(func(r *http.Request) (*http.Response, error) {
			requests++
			return objectReadWire(r, 200, http.Header{"Etag": {"initial"}}, body), nil
		})
		client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			retries++
			return errors.New("must not retry accepted body")
		}
		// Valid bytes returned with a Read error remain evidence even when that read cancels the context.
		body.reader = objectReadReader(func(p []byte) (int, error) {
			n := copy(p, []byte{0, 0xff, 'p'})
			cancel(cancelCause)
			return n, readCause
		})
		v, err := objects.New(client).GetObject(ctx, "c", "o")
		if v == nil || v.Metadata == nil || v.Complete || !bytes.Equal(v.Body, []byte{0, 0xff, 'p'}) || requests != 1 || retries != 0 || body.closes != 1 {
			t.Fatalf("partial=%+v err=%v req/retries=%d/%d closes=%d", v, err, requests, retries, body.closes)
		}
		for _, cause := range []error{readCause, closeCause, context.Canceled, cancelCause} {
			if !errors.Is(err, cause) {
				t.Fatalf("lost cause %v in %v", cause, err)
			}
		}
		proof := objectReadProof(t, err, 200)
		if !bytes.Equal(proof.Body, v.Body) || proof.Header.Get("ETag") != "initial" {
			t.Fatalf("partial proof=%+v", proof)
		}
		v.Body[0] = 9
		v.Header.Set("ETag", "mutated")
		if proof.Body[0] != 0 || proof.Header.Get("ETag") != "initial" {
			t.Fatal("error evidence aliases exported result")
		}
	})
	t.Run("R12-opening-source-and-native-failure-evidence", func(t *testing.T) {
		for _, mode := range []string{"buffer", "writer", "stream"} {
			body := objectReadBytes([]byte("must not read"))
			var client *gophercloud.ServiceClient
			client = objectReadClient(func(r *http.Request) (*http.Response, error) {
				client.ResourceBase = client.Endpoint + "retarget/"
				return objectReadWire(r, 200, http.Header{"X-Proof": {"opening"}}, body), nil
			})
			api := objects.New(client)
			var err error
			switch mode {
			case "buffer":
				v, e := api.GetObject(context.Background(), "c", "o")
				err = e
				if v == nil || v.Metadata != nil || v.Body != nil {
					t.Fatalf("opening=%+v", v)
				}
			case "writer":
				w := &objectReadWriter{}
				v, e := api.DownloadObject(context.Background(), "c", "o", w)
				err = e
				if v == nil || v.Metadata != nil || w.writes != 0 {
					t.Fatalf("opening=%+v writes=%d", v, w.writes)
				}
			case "stream":
				v, e := api.StreamObject(context.Background(), "c", "o")
				err = e
				if v == nil || v.Metadata != nil || v.Body != nil {
					t.Fatalf("opening=%+v", v)
				}
			}
			proof := objectReadProof(t, err, 200)
			if !errors.Is(err, resource.ErrInvalidOption) || proof.Header.Get("X-Proof") != "opening" || body.reads != 0 || body.closes != 1 {
				t.Fatalf("opening proof=%+v err=%v reads/closes=%d/%d", proof, err, body.reads, body.closes)
			}
		}
		for _, status := range []int{404, 412, 416} {
			closeCause := errors.New("native discarded Close")
			body := objectReadBytes([]byte("native-rejected-body"))
			body.closeErr = closeCause
			api := objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) {
				return objectReadWire(r, status, http.Header{"X-Native": {"proof"}}, body), nil
			}))
			v, err := api.GetObject(context.Background(), "c", "o")
			var native gophercloud.ErrUnexpectedResponseCode
			if v != nil || !errors.As(err, &native) || native.Actual != status || string(native.Body) != "native-rejected-body" || native.ResponseHeader.Get("X-Native") != "proof" || errors.Is(err, closeCause) || body.closes != 1 {
				t.Fatalf("native status=%d result=%+v error=%v bodyclose=%d", status, v, err, body.closes)
			}
		}
	})
}

func TestObjectReadContractsWriterAndStreamOwnership(t *testing.T) {
	t.Run("R13-bounded-copy-and-actual-write-counts", func(t *testing.T) {
		body := objectReadBytes([]byte("abcdefghijk"))
		w := &objectReadWriter{}
		api := objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) { return objectReadWire(r, 200, nil, body), nil }))
		v, err := api.DownloadObject(context.Background(), "c", "o", w, objects.WithObjectReadBufferSize(4))
		if err != nil || v == nil || !v.Complete || v.BytesWritten != 11 || w.String() != "abcdefghijk" || body.maxRead != 4 || w.maxWrite > 4 || body.shortcuts != 0 || body.closes != 1 {
			t.Fatalf("bounded=%+v writer=%+v body=%+v err=%v", v, w, body, err)
		}
		objectReadBorrowed(t, w)
		readCause, writeCause := errors.New("simultaneous Read"), errors.New("simultaneous Write")
		cases := []struct {
			name   string
			reader objectReadReader
			write  func([]byte) (int, error)
			want   error
			count  int64
		}{
			{"short-write", func(p []byte) (int, error) { return copy(p, "abc"), io.EOF }, func([]byte) (int, error) { return 1, nil }, io.ErrShortWrite, 1},
			{"read-and-write", func(p []byte) (int, error) { return copy(p, "abc"), readCause }, func([]byte) (int, error) { return 1, writeCause }, writeCause, 1},
			{"invalid-negative-reader", func([]byte) (int, error) { return -1, nil }, nil, nil, 0},
			{"invalid-large-reader", func(p []byte) (int, error) { return len(p) + 1, nil }, nil, nil, 0},
			{"invalid-negative-writer", func(p []byte) (int, error) { return copy(p, "abc"), io.EOF }, func([]byte) (int, error) { return -1, nil }, nil, 0},
			{"invalid-large-writer", func(p []byte) (int, error) { return copy(p, "abc"), io.EOF }, func(p []byte) (int, error) { return len(p) + 1, nil }, nil, 0},
			{"no-progress", func([]byte) (int, error) { return 0, nil }, nil, io.ErrNoProgress, 0},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				b := &objectReadBody{reader: c.reader}
				w := &objectReadWriter{write: c.write}
				api := objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) {
					return objectReadWire(r, 206, http.Header{"X-Proof": {"count"}}, b), nil
				}))
				v, err := api.DownloadObject(context.Background(), "c", "o", w, objects.WithObjectReadBufferSize(4))
				if v == nil || v.Complete || v.BytesWritten != c.count || err == nil || (c.want != nil && !errors.Is(err, c.want)) || b.closes != 1 || b.shortcuts != 0 {
					t.Fatalf("case=%s result=%+v err=%v body=%+v", c.name, v, err, b)
				}
				proof := objectReadProof(t, err, 206)
				if len(proof.Body) != 0 {
					t.Fatalf("writer error retained content: %x", proof.Body)
				}
				if c.name == "read-and-write" && !errors.Is(err, readCause) {
					t.Fatalf("lost Read cause: %v", err)
				}
				if c.name == "no-progress" && b.reads != 100 {
					t.Fatalf("empty reads=%d", b.reads)
				}
				objectReadBorrowed(t, w)
			})
		}
		closeCause := errors.New("copy Close only")
		body = objectReadBytes([]byte("ok"))
		body.closeErr = closeCause
		api = objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) { return objectReadWire(r, 200, nil, body), nil }))
		w = &objectReadWriter{}
		v, err = api.DownloadObject(context.Background(), "c", "o", w)
		if v == nil || !v.Complete || v.BytesWritten != 2 || !errors.Is(err, closeCause) || body.closes != 1 {
			t.Fatalf("Close-only=%+v err=%v", v, err)
		}
		objectReadProof(t, err, 200)
		fullWriteCause := errors.New("full-count Write at EOF")
		body = &objectReadBody{reader: objectReadReader(func(p []byte) (int, error) { return copy(p, "abc"), io.EOF })}
		api = objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) { return objectReadWire(r, 200, nil, body), nil }))
		w = &objectReadWriter{write: func(p []byte) (int, error) { return len(p), fullWriteCause }}
		v, err = api.DownloadObject(context.Background(), "c", "o", w)
		if v == nil || v.BytesWritten != 3 || !v.Complete || !errors.Is(err, fullWriteCause) || body.closes != 1 {
			t.Fatalf("full-count Write=%+v err=%v", v, err)
		}
		objectReadBorrowed(t, w)
	})
	t.Run("R14-eager-stream-caller-buffer-and-EOF", func(t *testing.T) {
		requests := 0
		body := &objectReadBody{reader: objectReadReader(func(p []byte) (int, error) { return copy(p, "abc"), io.EOF })}
		api := objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) {
			requests++
			return objectReadWire(r, 200, nil, body), nil
		}))
		v, err := api.StreamObject(context.Background(), "c", "o", objects.WithObjectReadBufferSize(1))
		if err != nil || v == nil || v.Body == nil || requests != 1 || body.reads != 0 || body.closes != 0 {
			t.Fatalf("open=%+v err=%v req=%d body=%+v", v, err, requests, body)
		}
		if n, err := v.Body.Read(nil); n != 0 || err != nil || body.reads != 0 {
			t.Fatalf("zero Read=%d,%v bodyReads=%d", n, err, body.reads)
		}
		buf := make([]byte, 12)
		n, err := v.Body.Read(buf)
		if n != 3 || err != io.EOF || string(buf[:n]) != "abc" || !v.Complete || v.BytesRead != 3 || body.maxRead != 12 || body.closes != 1 {
			t.Fatalf("Read=%d,%v stream=%+v body=%+v", n, err, v, body)
		}
		if n, err := v.Body.Read(buf); n != 0 || err != io.EOF || body.reads != 1 {
			t.Fatalf("terminal Read=%d,%v reads=%d", n, err, body.reads)
		}
		if err := v.Body.Close(); err != nil || body.closes != 1 {
			t.Fatalf("terminal Close=%v closes=%d", err, body.closes)
		}
		closeCause := errors.New("EOF Close")
		body = &objectReadBody{reader: objectReadReader(func(p []byte) (int, error) { return copy(p, "z"), io.EOF }), closeErr: closeCause}
		api = objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) {
			return objectReadWire(r, 206, http.Header{"X-Proof": {"EOF"}}, body), nil
		}))
		v, err = api.StreamObject(context.Background(), "c", "o")
		if err != nil {
			t.Fatal(err)
		}
		n, err = v.Body.Read(buf)
		if n != 1 || !errors.Is(err, closeCause) || !v.Complete || v.BytesRead != 1 || body.closes != 1 {
			t.Fatalf("EOF Close=%d,%v stream=%+v", n, err, v)
		}
		proof := objectReadProof(t, err, 206)
		if len(proof.Body) != 0 || proof.Header.Get("X-Proof") != "EOF" {
			t.Fatalf("EOF proof=%+v", proof)
		}
	})
	t.Run("R15-early-idempotent-Close-immutable-proof", func(t *testing.T) {
		closeCause := errors.New("early Close")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		body := objectReadBytes([]byte("unread"))
		body.closeErr = closeCause
		api := objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) {
			return objectReadWire(r, 206, http.Header{"X-Proof": {"original"}}, body), nil
		}))
		v, err := api.StreamObject(ctx, "c", "o")
		if err != nil || v == nil {
			t.Fatalf("open=%+v err=%v", v, err)
		}
		v.Header.Set("X-Proof", "caller changed")
		v.StatusCode = 999
		err = v.Body.Close()
		proof := objectReadProof(t, err, 206)
		if !errors.Is(err, closeCause) || proof.Header.Get("X-Proof") != "original" || len(proof.Body) != 0 || v.Complete || v.BytesRead != 0 || body.reads != 0 || body.closes != 1 {
			t.Fatalf("early Close=%v proof=%+v result=%+v body=%+v", err, proof, v, body)
		}
		cancel(errors.New("late cancellation"))
		again := v.Body.Close()
		if !errors.Is(again, closeCause) || errors.Is(again, context.Canceled) || body.closes != 1 {
			t.Fatalf("repeat Close=%v closes=%d", again, body.closes)
		}
		if n, readErr := v.Body.Read(make([]byte, 4)); n != 0 || readErr == nil || body.reads != 0 || body.closes != 1 {
			t.Fatalf("closed Read=%d,%v body=%+v", n, readErr, body)
		}
	})
	t.Run("R16-partial-context-and-original-source-guards", func(t *testing.T) {
		readCause, closeCause, cancelCause := errors.New("stream Read"), errors.New("stream Close"), errors.New("stream caller cancellation")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		body := &objectReadBody{reader: objectReadReader(func(p []byte) (int, error) { n := copy(p, "abc"); cancel(cancelCause); return n, readCause }), closeErr: closeCause}
		requests := 0
		api := objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) {
			requests++
			return objectReadWire(r, 200, http.Header{"X-Proof": {"initial"}}, body), nil
		}))
		v, err := api.StreamObject(ctx, "c", "o")
		if err != nil {
			t.Fatal(err)
		}
		v.Header.Set("X-Proof", "caller changed")
		buf := make([]byte, 4)
		n, err := v.Body.Read(buf)
		if n != 3 || string(buf[:n]) != "abc" || v.BytesRead != 3 || v.Complete || body.closes != 1 || requests != 1 {
			t.Fatalf("partial Read=%d,%v result=%+v", n, err, v)
		}
		for _, cause := range []error{readCause, closeCause, context.Canceled, cancelCause} {
			if !errors.Is(err, cause) {
				t.Fatalf("lost %v in %v", cause, err)
			}
		}
		proof := objectReadProof(t, err, 200)
		if proof.Header.Get("X-Proof") != "initial" || len(proof.Body) != 0 {
			t.Fatalf("stream proof=%+v", proof)
		}
		for _, mutate := range []func(*gophercloud.ServiceClient){func(c *gophercloud.ServiceClient) { c.Endpoint += "changed/" }, func(c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} }, func(c *gophercloud.ServiceClient) { c.MoreHeaders = map[string]string{"Range": "late reserved"} }} {
			body = objectReadBytes([]byte("unread"))
			client := objectReadClient(func(r *http.Request) (*http.Response, error) { return objectReadWire(r, 200, nil, body), nil })
			v, err = objects.New(client).StreamObject(context.Background(), "c", "o")
			if err != nil {
				t.Fatal(err)
			}
			mutate(client)
			n, err = v.Body.Read(buf)
			if n != 0 || !errors.Is(err, resource.ErrInvalidOption) || body.reads != 0 || body.closes != 1 || v.BytesRead != 0 {
				t.Fatalf("source Read=%d,%v body=%+v", n, err, body)
			}
			objectReadProof(t, err, 200)
		}
		ctx2, cancel2 := context.WithCancelCause(context.Background())
		defer cancel2(nil)
		body = &objectReadBody{reader: objectReadReader(func(p []byte) (int, error) { n := copy(p, "read but not written"); cancel2(cancelCause); return n, nil })}
		w := &objectReadWriter{}
		api = objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) { return objectReadWire(r, 200, nil, body), nil }))
		download, err := api.DownloadObject(ctx2, "c", "o", w)
		if download == nil || download.BytesWritten != 0 || download.Complete || w.writes != 0 || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || body.closes != 1 {
			t.Fatalf("cancel between Read/Write=%+v err=%v writes=%d", download, err, w.writes)
		}
		objectReadBorrowed(t, w)
		for _, mode := range []string{"buffer", "stream"} {
			t.Run("EOF-postRead-cancel/"+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				body := &objectReadBody{reader: objectReadReader(func(p []byte) (int, error) { n := copy(p, "EOF"); cancel(cancelCause); return n, io.EOF })}
				api := objects.New(objectReadClient(func(r *http.Request) (*http.Response, error) { return objectReadWire(r, 200, nil, body), nil }))
				if mode == "buffer" {
					v, err := api.GetObject(ctx, "c", "o")
					if v == nil || string(v.Body) != "EOF" || !v.Complete || !errors.Is(err, cancelCause) {
						t.Fatalf("EOF buffer=%+v err=%v", v, err)
					}
				} else {
					v, err := api.StreamObject(ctx, "c", "o")
					if err != nil {
						t.Fatal(err)
					}
					n, err := v.Body.Read(make([]byte, 4))
					if n != 3 || !v.Complete || v.BytesRead != 3 || !errors.Is(err, cancelCause) {
						t.Fatalf("EOF stream=%+v n=%d err=%v", v, n, err)
					}
				}
				if body.closes != 1 {
					t.Fatalf("EOF %s closes=%d", mode, body.closes)
				}
			})
		}
	})
}

func TestObjectReadContractsNativePolicy(t *testing.T) {
	t.Run("R17-prebody-retry-live-token-and-reauth-fields", func(t *testing.T) {
		for _, advanced := range []bool{false, true} {
			first, accepted := objectReadBytes([]byte("503-body")), objectReadBytes([]byte("accepted"))
			requests, hooks := 0, 0
			var client *gophercloud.ServiceClient
			client = objectReadClient(func(r *http.Request) (*http.Response, error) {
				requests++
				if r.URL.String() != "https://swift.invalid/v1/A/c/o" || r.Method != "GET" || r.Body != nil {
					t.Errorf("fixed request=%s %s body=%v", r.Method, r.URL, r.Body)
				}
				if requests == 1 {
					if r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != "initial-token" {
						t.Errorf("first headers=%v", r.Header)
					}
					return objectReadWire(r, 503, http.Header{"X-Native": {"503"}}, first), nil
				}
				if r.Header.Get("X-Auth-Token") != "latest-token" {
					t.Errorf("latest token=%q", r.Header.Get("X-Auth-Token"))
				}
				if advanced {
					if r.Header.Get("X-Source") != "" || r.Header.Get("X-Object-Meta-Native") != "advanced" {
						t.Errorf("advanced wholesale headers=%v", r.Header)
					}
				} else if r.Header.Get("X-Source") != "captured" {
					t.Errorf("normal retry headers=%v", r.Header)
				}
				return objectReadWire(r, 200, nil, accepted), nil
			})
			client.MoreHeaders = map[string]string{"X-Source": "captured"}
			client.RetryFunc = func(ctx context.Context, method, target string, o *gophercloud.RequestOpts, original error, count uint) error {
				hooks++
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(original, &native) || native.Actual != 503 || count != 1 {
					t.Errorf("retry original=%v count=%d", original, count)
				}
				client.SetToken("latest-token")
				if advanced {
					o.MoreHeaders = map[string]string{"X-Object-Meta-Native": "advanced"}
				}
				return nil
			}
			v, err := objects.New(client).GetObject(context.Background(), "c", "o")
			if err != nil || v == nil || string(v.Body) != "accepted" || requests != 2 || hooks != 1 || first.closes != 1 || accepted.closes != 1 || client.MoreHeaders["X-Source"] != "captured" {
				t.Fatalf("retry advanced=%t result=%+v err=%v requests/hooks=%d/%d closes=%d/%d source=%v", advanced, v, err, requests, hooks, first.closes, accepted.closes, client.MoreHeaders)
			}
		}
		for _, fail := range []bool{false, true} {
			requests, reauth := 0, 0
			cause := errors.New("reauth failed")
			var client *gophercloud.ServiceClient
			client = objectReadClient(func(r *http.Request) (*http.Response, error) {
				requests++
				if requests == 1 {
					return objectReadWire(r, 401, http.Header{"X-Proof": {"401"}}, objectReadBytes([]byte("unauthorized"))), nil
				}
				if r.Header.Get("X-Auth-Token") != "reauth-token" {
					t.Errorf("reauth headers=%v", r.Header)
				}
				return objectReadWire(r, 200, nil, objectReadBytes([]byte("ok"))), nil
			})
			client.ReauthFunc = func(context.Context) error {
				reauth++
				if fail {
					return cause
				}
				client.SetToken("reauth-token")
				return nil
			}
			v, err := objects.New(client).GetObject(context.Background(), "c", "o")
			if fail {
				var native *gophercloud.ErrUnableToReauthenticate
				var original gophercloud.ErrUnexpectedResponseCode
				if v != nil || !errors.As(err, &native) || native.ErrReauth != cause || !errors.As(native.ErrOriginal, &original) || original.Actual != 401 || string(original.Body) != "unauthorized" || requests != 1 || reauth != 1 {
					t.Fatalf("reauth failure=%+v err=%v native=%+v", v, err, native)
				}
			} else if err != nil || v == nil || string(v.Body) != "ok" || requests != 2 || reauth != 1 {
				t.Fatalf("reauth=%+v err=%v requests=%d", v, err, requests)
			}
		}
	})
	t.Run("R18-owned-body-guard-and-expanded-code", func(t *testing.T) {
		callbackCause := errors.New("callback failure")
		for _, kind := range []string{"keep", "JSONBody", "RawBody", "JSONResponse", "source"} {
			t.Run(kind, func(t *testing.T) {
				requests, hooks := 0, 0
				body := objectReadBytes([]byte("original503"))
				forbidden := objectReadBytes([]byte("must not send"))
				client := objectReadClient(func(r *http.Request) (*http.Response, error) {
					requests++
					if r.Body != nil {
						t.Error("GET carried a body")
					}
					return objectReadWire(r, 503, http.Header{"X-Proof": {"native503"}}, body), nil
				})
				client.RetryFunc = func(ctx context.Context, method, target string, o *gophercloud.RequestOpts, original error, count uint) error {
					hooks++
					switch kind {
					case "keep":
						o.KeepResponseBody = false
					case "JSONBody":
						o.JSONBody = map[string]string{"x": "y"}
					case "RawBody":
						o.RawBody = forbidden
					case "JSONResponse":
						o.JSONResponse = &struct{}{}
					case "source":
						client.Endpoint += "changed/"
					}
					return callbackCause
				}
				v, err := objects.New(client).GetObject(context.Background(), "c", "o")
				var original gophercloud.ErrUnexpectedResponseCode
				if v != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, callbackCause) || !errors.As(err, &original) || original.Actual != 503 || string(original.Body) != "original503" || requests != 1 || hooks != 1 || body.closes != 1 || forbidden.reads != 0 || forbidden.closes != 0 {
					t.Fatalf("guard=%s result=%+v err=%v requests/hooks=%d/%d body=%+v forbidden=%+v", kind, v, err, requests, hooks, body, forbidden)
				}
			})
		}
		requests, hooks := 0, 0
		first := objectReadBytes([]byte("503"))
		closeCause := errors.New("expanded-code Close")
		unwanted := &objectReadBody{reader: objectReadReader(func([]byte) (int, error) { t.Error("expanded status drained body"); return 0, io.EOF }), closeErr: closeCause}
		client := objectReadClient(func(r *http.Request) (*http.Response, error) {
			requests++
			if requests == 1 {
				return objectReadWire(r, 503, nil, first), nil
			}
			return objectReadWire(r, 202, http.Header{"X-Proof": {"202"}}, unwanted), nil
		})
		client.RetryFunc = func(ctx context.Context, method, target string, o *gophercloud.RequestOpts, original error, count uint) error {
			hooks++
			o.OkCodes = append(o.OkCodes, 202)
			return nil
		}
		v, err := objects.New(client).StreamObject(context.Background(), "c", "o")
		var native gophercloud.ErrUnexpectedResponseCode
		if v != nil || !errors.As(err, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{200, 206, 304}) || len(native.Body) != 0 || native.ResponseHeader.Get("X-Proof") != "202" || !errors.Is(err, closeCause) || requests != 2 || hooks != 1 || unwanted.reads != 0 || unwanted.closes != 1 {
			t.Fatalf("expanded=%+v err=%v native=%+v unwanted=%+v", v, err, native, unwanted)
		}
	})
	t.Run("R19-redirect-and-HTTP-timeout-policy", func(t *testing.T) {
		for _, same := range []bool{true, false} {
			var requests, redirects atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if requests.Add(1) == 1 {
					target := r.URL.String()
					if !same {
						target = "/different"
					}
					http.Redirect(w, r, target, http.StatusTemporaryRedirect)
					return
				}
				_, _ = w.Write([]byte("ok"))
			}))
			client := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{HTTPClient: http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { redirects.Add(1); return nil }}}, Endpoint: server.URL + "/v1/A/", Type: "object-store"}
			v, err := objects.New(client).GetObject(context.Background(), "c", "o")
			server.Close()
			if same {
				if err != nil || v == nil || string(v.Body) != "ok" || requests.Load() != 2 || redirects.Load() != 1 {
					t.Fatalf("same redirect=%+v err=%v requests=%d", v, err, requests.Load())
				}
			} else if v != nil || !errors.Is(err, resource.ErrInvalidOption) || requests.Load() != 1 || redirects.Load() != 1 {
				t.Fatalf("different redirect=%+v err=%v requests=%d", v, err, requests.Load())
			}
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Proof", "timeout")
			w.WriteHeader(200)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		}))
		defer server.Close()
		client := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{HTTPClient: http.Client{Timeout: 250 * time.Millisecond}}, Endpoint: server.URL + "/v1/A/", Type: "object-store"}
		v, err := objects.New(client).GetObject(context.Background(), "c", "o")
		if v == nil || v.StatusCode != 200 || v.Complete || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("HTTP timeout=%+v err=%v", v, err)
		}
		objectReadProof(t, err, 200)
	})
	t.Run("R20-existing-native-Download-ABI", func(t *testing.T) {
		for _, status := range []int{200, 206, 304} {
			body := objectReadBytes([]byte("native-body"))
			date := time.Date(2024, 1, 1, 3, 4, 5, 0, time.FixedZone("SGT", 8*3600))
			client := objectReadClient(func(r *http.Request) (*http.Response, error) {
				want := url.Values{"expires": {"123"}, "signature": {"sig"}, "multipart-manifest": {"get"}, "version-id": {"version"}}
				if !reflect.DeepEqual(r.URL.Query(), want) || r.Header.Get("If-Match") != "native" || r.Header.Get("If-Modified-Since") != date.Format(time.RFC1123) {
					t.Errorf("native request=%s headers=%v", r.URL, r.Header)
				}
				return objectReadWire(r, status, http.Header{"Etag": {"native-etag"}, "Content-Length": {"11"}}, body), nil
			})
			v, err := objects.New(client).Download(context.Background(), "c", "o", objects.WithDownloadOptions(objects.DownloadOpts{IfMatch: "native", IfModifiedSince: date, Expires: "123", Signature: "sig", MultipartManifest: "get", ObjectVersionID: "version"}))
			if err != nil || v == nil || v.Header.ETag != "native-etag" || body.closes != 0 {
				t.Fatalf("native status=%d result=%+v err=%v", status, v, err)
			}
			got, err := io.ReadAll(v)
			if err != nil || string(got) != "native-body" {
				t.Fatalf("native bytes=%q err=%v", got, err)
			}
			if err := v.Close(); err != nil || body.closes != 1 {
				t.Fatalf("native Close=%v closes=%d", err, body.closes)
			}
		}
	})
}
