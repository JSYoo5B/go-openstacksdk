package objects

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type readTestTransport func(*http.Request) (*http.Response, error)

func (f readTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func readTestAPI(f readTestTransport) (*API, *gophercloud.ServiceClient) {
	provider := &gophercloud.ProviderClient{TokenID: "live-token", HTTPClient: http.Client{Transport: f}}
	source := &gophercloud.ServiceClient{ProviderClient: provider, Endpoint: "https://storage.test/reverse%20proxy/v1/a/", Type: "object-store"}
	return New(source), source
}
func readTestResponse(code int, headers http.Header, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: headers, Body: body}
}
func readTestProof(t *testing.T, err error, status int) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != status {
		t.Fatalf("expected owned HTTP%d proof, got %v", status, err)
	}
	return proof
}

type readTestBody struct {
	data                  []byte
	readErr, closeErr     error
	afterRead, afterClose func()
	reads, closes         int
	buffers               []int
	zero, invalid         bool
}

func (b *readTestBody) Read(p []byte) (int, error) {
	b.reads++
	b.buffers = append(b.buffers, len(p))
	if b.zero {
		return 0, nil
	}
	if b.invalid {
		return len(p) + 1, b.readErr
	}
	n := copy(p, b.data)
	b.data = b.data[n:]
	if b.afterRead != nil {
		b.afterRead()
	}
	if len(b.data) == 0 {
		if b.readErr != nil {
			return n, b.readErr
		}
		return n, io.EOF
	}
	return n, nil
}
func (b *readTestBody) Close() error {
	b.closes++
	if b.afterClose != nil {
		b.afterClose()
	}
	return b.closeErr
}
func (b *readTestBody) WriteTo(io.Writer) (int64, error) {
	return 0, errors.New("unexpected WriterTo shortcut")
}

type readTestWriter struct {
	bytes.Buffer
	limit                                    int
	err                                      error
	calls, closes, flushes, seeks, shortcuts int
}

func (w *readTestWriter) Write(p []byte) (int, error) {
	w.calls++
	n := len(p)
	if w.limit >= 0 && w.limit < n {
		n = w.limit
	}
	_, _ = w.Buffer.Write(p[:n])
	return n, w.err
}
func (w *readTestWriter) Close() error                   { w.closes++; return nil }
func (w *readTestWriter) Flush() error                   { w.flushes++; return nil }
func (w *readTestWriter) Seek(int64, int) (int64, error) { w.seeks++; return 0, nil }
func (w *readTestWriter) ReadFrom(io.Reader) (int64, error) {
	w.shortcuts++
	return 0, errors.New("unexpected ReaderFrom shortcut")
}

type readTestBadWriter int

func (w readTestBadWriter) Write(p []byte) (int, error) { return int(w), nil }

func TestObjectReadCoreWireAndStatus(t *testing.T) {
	for _, method := range []string{"get", "download", "stream"} {
		for _, status := range []int{200, 206, 304} {
			t.Run(fmt.Sprintf("%s/%d", method, status), func(t *testing.T) {
				payload := []byte{'a', 0, 0xff, 'z'}
				probe := &readTestBody{data: append([]byte(nil), payload...)}
				headers := http.Header{"X-Object-Meta-Tag": {"literal"}, "Content-Length": {"999"}, "Etag": {"unverified"}, "Content-Type": {"multipart/byteranges; boundary=x"}}
				calls := 0
				api, _ := readTestAPI(func(r *http.Request) (*http.Response, error) {
					calls++
					expected := "/reverse%20proxy/v1/a/" + url.PathEscape("c% ?#") + "/" + url.PathEscape("folder/obj% ?#")
					if r.Method != "GET" || r.URL.EscapedPath() != expected || r.Body != nil {
						t.Errorf("wrong fixed GET: %s %s", r.Method, r.URL)
					}
					if r.URL.Query().Get("filename") != "a ?#" || r.Header.Get("Range") != "opaque-range" || r.Header.Get("X-Auth-Token") != "live-token" {
						t.Errorf("missing owned query/header: %s %v", r.URL, r.Header)
					}
					return readTestResponse(status, headers, probe), nil
				})
				options := []ObjectReadOption{WithObjectReadRange("opaque-range"), WithObjectReadFilename("a ?#"), WithObjectReadBufferSize(2)}
				var content []byte
				var metadata *MetadataInfo
				var observed http.Header
				var complete, notModified bool
				var err error
				switch method {
				case "get":
					result, e := api.GetObject(context.Background(), "c% ?#", "folder/obj% ?#", options...)
					if result == nil {
						t.Fatalf("nil result: %v", e)
					}
					content, metadata, observed, complete, notModified, err = result.Body, result.Metadata, result.Header, result.Complete, result.NotModified, e
				case "download":
					writer := &readTestWriter{limit: -1}
					result, e := api.DownloadObject(context.Background(), "c% ?#", "folder/obj% ?#", writer, options...)
					if result == nil {
						t.Fatalf("nil result: %v", e)
					}
					content, metadata, observed, complete, notModified, err = writer.Bytes(), result.Metadata, result.Header, result.Complete, result.NotModified, e
					if result.BytesWritten != int64(len(content)) || writer.closes+writer.flushes+writer.seeks+writer.shortcuts != 0 {
						t.Fatal("writer ownership or count changed")
					}
				case "stream":
					result, e := api.StreamObject(context.Background(), "c% ?#", "folder/obj% ?#", options...)
					if result == nil || result.Body == nil || e != nil {
						t.Fatalf("stream open: %+v %v", result, e)
					}
					content, err = io.ReadAll(result.Body)
					if closeErr := result.Body.Close(); closeErr != nil {
						t.Fatal(closeErr)
					}
					metadata, observed, complete, notModified = result.Metadata, result.Header, result.Complete, result.NotModified
					if result.BytesRead != int64(len(content)) {
						t.Fatal("wrong stream count")
					}
				}
				if err != nil || !complete || notModified != (status == 304) || calls != 1 || probe.closes != 1 {
					t.Fatalf("observation err=%v complete=%t 304=%t calls=%d close=%d", err, complete, notModified, calls, probe.closes)
				}
				if status == 304 {
					if len(content) != 0 || probe.reads != 0 {
						t.Fatal("304 payload exposed")
					}
				} else if !bytes.Equal(content, payload) {
					t.Fatalf("binary content altered: %v", content)
				}
				if metadata == nil || metadata.Values["tag"] != "literal" || metadata.ContentLength == nil || *metadata.ContentLength != 999 {
					t.Fatal("metadata or unverified length lost")
				}
				observed.Set("X-Object-Meta-Tag", "caller")
				if headers.Get("X-Object-Meta-Tag") != "literal" || metadata.Values["tag"] != "literal" {
					t.Fatal("raw/projected metadata alias")
				}
			})
		}
	}
}

func TestObjectReadCoreBorrowedWriter(t *testing.T) {
	fault := errors.New("write-fault")
	for _, test := range []struct {
		name     string
		output   io.Writer
		count    int64
		complete bool
		cause    error
	}{
		{"short", &readTestWriter{limit: 2}, 2, false, io.ErrShortWrite},
		{"full-count-error", &readTestWriter{limit: -1, err: fault}, 4, true, fault},
		{"negative", readTestBadWriter(-1), 0, false, resource.ErrInvalidOption},
		{"oversize", readTestBadWriter(99), 0, false, resource.ErrInvalidOption},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := &readTestBody{data: []byte("data")}
			api, _ := readTestAPI(func(*http.Request) (*http.Response, error) { return readTestResponse(200, http.Header{}, body), nil })
			result, err := api.DownloadObject(context.Background(), "c", "o", test.output)
			if result == nil || result.BytesWritten != test.count || result.Complete != test.complete || !errors.Is(err, test.cause) || body.closes != 1 {
				t.Fatalf("wrong copy: %+v %v close=%d", result, err, body.closes)
			}
			proof := readTestProof(t, err, 200)
			if proof.Body != nil {
				t.Fatal("writer error accumulated a payload")
			}
			if writer, ok := test.output.(*readTestWriter); ok && writer.closes+writer.flushes+writer.seeks+writer.shortcuts != 0 {
				t.Fatal("borrowed writer cleanup/shortcut")
			}
		})
	}
	t.Run("cancel-between-read-and-write", func(t *testing.T) {
		cause := errors.New("cancel-cause")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		body := &readTestBody{data: []byte("data"), afterRead: func() { cancel(cause) }}
		writer := &readTestWriter{limit: -1}
		api, _ := readTestAPI(func(*http.Request) (*http.Response, error) { return readTestResponse(200, http.Header{}, body), nil })
		result, err := api.DownloadObject(ctx, "c", "o", writer)
		if result == nil || result.BytesWritten != 0 || result.Complete || writer.calls != 0 || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || body.closes != 1 {
			t.Fatalf("cancel delivery: %+v %v", result, err)
		}
	})
}

func TestObjectReadCoreResponseAndSource(t *testing.T) {
	t.Run("partial-read-close-independent-proof", func(t *testing.T) {
		readFault, closeFault := errors.New("read-fault"), errors.New("close-fault")
		body := &readTestBody{data: []byte("partial"), readErr: readFault, closeErr: closeFault}
		api, _ := readTestAPI(func(*http.Request) (*http.Response, error) {
			return readTestResponse(206, http.Header{"X-Object-Meta-K": {"v"}}, body), nil
		})
		result, err := api.GetObject(context.Background(), "c", "o")
		if result == nil || result.Metadata == nil || string(result.Body) != "partial" || result.Complete || !errors.Is(err, readFault) || !errors.Is(err, closeFault) || body.closes != 1 {
			t.Fatalf("partial response: %+v %v", result, err)
		}
		proof := readTestProof(t, err, 206)
		result.Body[0] = 'X'
		result.Header.Set("X-Object-Meta-K", "changed")
		if string(proof.Body) != "partial" || proof.Header.Get("X-Object-Meta-K") != "v" {
			t.Fatal("mutable result corrupted immutable proof")
		}
	})
	t.Run("atomic-header-projection", func(t *testing.T) {
		closeFault := errors.New("close-fault")
		body := &readTestBody{data: []byte("must-not-read"), closeErr: closeFault}
		api, _ := readTestAPI(func(*http.Request) (*http.Response, error) {
			return readTestResponse(200, http.Header{"Content-Length": {"not-number"}}, body), nil
		})
		result, err := api.GetObject(context.Background(), "c", "o")
		if result == nil || result.Metadata != nil || result.Body != nil || body.reads != 0 || body.closes != 1 || !errors.Is(err, closeFault) || !strings.Contains(err.Error(), "content length") {
			t.Fatalf("atomic projection: %+v %v", result, err)
		}
		if readTestProof(t, err, 200).Body != nil {
			t.Fatal("invented/drained payload")
		}
	})
	t.Run("EOF-with-source-header-fault", func(t *testing.T) {
		body := &readTestBody{data: []byte("complete")}
		api, source := readTestAPI(func(*http.Request) (*http.Response, error) { return readTestResponse(200, http.Header{}, body), nil })
		body.afterRead = func() { source.MoreHeaders = map[string]string{"Range": "changed"} }
		result, err := api.GetObject(context.Background(), "c", "o")
		if result == nil || result.Metadata == nil || !result.Complete || string(result.Body) != "complete" || !errors.Is(err, resource.ErrInvalidOption) || body.closes != 1 {
			t.Fatalf("observed EOF lost: %+v %v", result, err)
		}
		readTestProof(t, err, 200)
	})
	t.Run("joined-EOF-is-a-read-fault", func(t *testing.T) {
		cause := errors.New("joined-read-fault")
		body := &readTestBody{data: []byte("data"), readErr: errors.Join(io.EOF, cause)}
		api, _ := readTestAPI(func(*http.Request) (*http.Response, error) { return readTestResponse(200, http.Header{}, body), nil })
		result, err := api.GetObject(context.Background(), "c", "o")
		if result == nil || result.Complete || string(result.Body) != "data" || !errors.Is(err, cause) || !errors.Is(err, io.EOF) {
			t.Fatalf("joined EOF swallowed: %+v %v", result, err)
		}
	})
}
