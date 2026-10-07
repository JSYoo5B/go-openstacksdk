package image_test

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/image"
	"github.com/JSYoo5B/gophercloudsdk/image/v2/imagedata"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// The pinned DownloadMixin always fetches metadata first. DownloadTo covers
// its writer branch with fixed routes, retained evidence and Go error chains.
const downloadPrefix = "/reverse/glance/v2/"

type downloadTransport func(*http.Request) (*http.Response, error)

func (f downloadTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func downloadWire(code int, phase string, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"X-Phase": {phase}}, Body: body}
}

func downloadMetadata(fields string) string {
	return `{"id":"fixed","name":"needle","status":"queued"` + fields + `}`
}

func downloadDigest(algorithm, data string) string {
	var h hash.Hash
	switch algorithm {
	case "md5":
		h = md5.New()
	case "sha1":
		h = sha1.New()
	case "sha224":
		h = sha256.New224()
	case "sha256":
		h = sha256.New()
	case "sha384":
		h = sha512.New384()
	case "sha512":
		h = sha512.New()
	case "sha512_224":
		h = sha512.New512_224()
	case "sha512_256":
		h = sha512.New512_256()
	default:
		panic("unknown test algorithm: " + algorithm)
	}
	_, _ = io.WriteString(h, data)
	return hex.EncodeToString(h.Sum(nil))
}

type downloadBody struct {
	io.Reader
	closes, fastPaths atomic.Int32
	maxRead           atomic.Int64
	closeErr          error
}

func (r *downloadBody) Read(p []byte) (int, error) {
	for size := r.maxRead.Load(); int64(len(p)) > size; size = r.maxRead.Load() {
		if r.maxRead.CompareAndSwap(size, int64(len(p))) {
			break
		}
	}
	return r.Reader.Read(p)
}
func (r *downloadBody) Close() error { r.closes.Add(1); return r.closeErr }
func (r *downloadBody) WriteTo(io.Writer) (int64, error) {
	r.fastPaths.Add(1)
	return 0, errors.New("response WriterTo bypassed bounded copy")
}

type downloadSink struct {
	bytes.Buffer
	writes, closes, seeks, truncates, fastPaths atomic.Int32
	maxWrite                                    atomic.Int64
	onWrite                                     func([]byte) (int, error)
}

func (w *downloadSink) Write(p []byte) (int, error) {
	w.writes.Add(1)
	for size := w.maxWrite.Load(); int64(len(p)) > size; size = w.maxWrite.Load() {
		if w.maxWrite.CompareAndSwap(size, int64(len(p))) {
			break
		}
	}
	if w.onWrite != nil {
		return w.onWrite(p)
	}
	return w.Buffer.Write(p)
}
func (w *downloadSink) Close() error { w.closes.Add(1); return nil }
func (w *downloadSink) Seek(int64, int) (int64, error) {
	w.seeks.Add(1)
	return 0, errors.New("caller writer must not be sought")
}
func (w *downloadSink) Truncate(int64) error { w.truncates.Add(1); return nil }
func (w *downloadSink) ReadFrom(io.Reader) (int64, error) {
	w.fastPaths.Add(1)
	return 0, errors.New("writer ReaderFrom bypassed Write accounting")
}

func TestDownloadToDefaultSequenceAndEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("image", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + downloadPrefix
	client.MoreHeaders = map[string]string{"X-Source": "captured"}
	payload := strings.Repeat("image", 300000)
	digest := downloadDigest("sha512", payload)
	metadata := downloadMetadata(fmt.Sprintf(`,"os_hash_algo":"sha512","os_hash_value":%q,"checksum":"metadata-decoy","ID":"typed-decoy","file":"https://foreign.invalid/not-used","self":"/wrong"`, digest))
	body := &downloadBody{Reader: strings.NewReader(payload)}
	sink := &downloadSink{}
	_, _ = sink.Buffer.WriteString("prefix-")
	var sequence []string
	cloud.Provider.HTTPClient.Transport = downloadTransport(func(r *http.Request) (*http.Response, error) {
		sequence = append(sequence, r.Method+" "+r.URL.Path)
		if r.Method != http.MethodGet || r.Header.Get("X-Source") != "captured" || r.URL.RawQuery != "" {
			t.Error(r.Method, r.URL, r.Header)
		}
		if r.URL.Path == downloadPrefix+"images/fixed" {
			cloud.Provider.SetToken("after-metadata")
			wire := downloadWire(200, "metadata", io.NopCloser(strings.NewReader(metadata)))
			wire.Header.Set("Location", "https://foreign.invalid/image")
			return wire, nil
		}
		if r.URL.Path != downloadPrefix+"images/fixed/file" || r.Header.Get("X-Auth-Token") != "after-metadata" {
			t.Error("metadata retargeted file request", r.URL, r.Header)
		}
		wire := downloadWire(200, "file", body)
		wire.Header.Set("Content-MD5", "header-decoy")
		return wire, nil
	})
	v, err := image.New(client).DownloadTo(context.Background(), resource.ID("fixed"), sink)
	if err != nil || v == nil || v.ImageID != "fixed" || v.Image == nil || v.Image.ID != "fixed" || v.Image.Status != "queued" || v.Metadata == nil || v.Metadata.StatusCode != 200 || string(v.Metadata.Body) != metadata || v.Metadata.Header.Get("X-Phase") != "metadata" || v.StatusCode != 200 || v.Header.Get("X-Phase") != "file" || v.Header.Get("Content-MD5") != "header-decoy" || v.BytesWritten != int64(len(payload)) || sink.String() != "prefix-"+payload || v.Checksum == nil || v.Checksum.Algorithm != "sha512" || v.Checksum.Expected != digest || v.Checksum.Actual != digest || !v.Checksum.Complete || !v.Checksum.Verified {
		t.Fatal(v, err, sink.Len())
	}
	if !reflect.DeepEqual(sequence, []string{"GET " + downloadPrefix + "images/fixed", "GET " + downloadPrefix + "images/fixed/file"}) || body.closes.Load() != 1 || body.fastPaths.Load() != 0 || body.maxRead.Load() != 1048576 || sink.maxWrite.Load() > 1048576 || sink.fastPaths.Load() != 0 || sink.closes.Load() != 0 || sink.seeks.Load() != 0 || sink.truncates.Load() != 0 {
		t.Fatal(sequence, body.closes.Load(), body.maxRead.Load(), sink.maxWrite.Load(), sink.fastPaths.Load())
	}
	v.Header.Set("X-Phase", "caller")
	v.Metadata.Body[0] = '!'
	v.Image.Status = "caller"
	if v.Metadata.Header.Get("X-Phase") != "metadata" || client.ResourceBase != cloud.Server.URL+downloadPrefix || client.Endpoint != cloud.Server.URL+"/catalog/unused/" || client.MoreHeaders["X-Source"] != "captured" {
		t.Fatal("result aliases source or other phase", v, client)
	}
}

func TestDownloadToCompletePreflight(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts []image.DownloadImageOption
	}{
		{"nil option", []image.DownloadImageOption{nil}},
		{"zero chunk", []image.DownloadImageOption{image.WithDownloadChunkSize(0)}},
		{"negative chunk", []image.DownloadImageOption{image.WithDownloadChunkSize(-1)}},
		{"oversized chunk", []image.DownloadImageOption{image.WithDownloadChunkSize(67108865)}},
		{"empty store", []image.DownloadImageOption{image.WithDownloadStorePreferences("")}},
		{"comma store", []image.DownloadImageOption{image.WithDownloadStorePreferences("fast,slow")}},
		{"control store", []image.DownloadImageOption{image.WithDownloadStorePreferences("line\nbreak")}},
		{"invalid UTF8 store", []image.DownloadImageOption{image.WithDownloadStorePreferences(string([]byte{255}))}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			sink := &downloadSink{}
			cloud.Provider.HTTPClient.Transport = downloadTransport(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return downloadWire(500, "unexpected", io.NopCloser(strings.NewReader(`{}`))), nil
			})
			v, err := image.New(cloud.Client("image", downloadPrefix)).DownloadTo(context.Background(), resource.ID("fixed"), sink, tc.opts...)
			if v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 || sink.writes.Load() != 0 || sink.closes.Load() != 0 {
				t.Fatal(v, err, calls.Load(), sink.writes.Load(), sink.closes.Load())
			}
		})
	}
	for _, mode := range []string{"nil service", "nil client", "nil provider", "wrong type", "foreign base", "query base", "Range", "If-Range", "bad ID", "nil writer", "typed nil writer", "canceled", "nil context"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", downloadPrefix)
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = downloadTransport(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return downloadWire(500, "unexpected", io.NopCloser(strings.NewReader(`{}`))), nil
			})
			sink := &downloadSink{}
			var output io.Writer = sink
			ctx := context.Background()
			ref := resource.ID("fixed")
			want := error(resource.ErrInvalidOption)
			switch mode {
			case "nil client":
				client = nil
			case "nil provider":
				client.ProviderClient = nil
			case "wrong type":
				client.Type = "compute"
				want = resource.ErrUnsupported
			case "foreign base":
				client.ResourceBase = "https://foreign.invalid/v2/"
			case "query base":
				client.ResourceBase = cloud.Server.URL + downloadPrefix + "?unsafe=base"
			case "Range", "If-Range":
				client.MoreHeaders = map[string]string{mode: "bytes=0-3"}
			case "bad ID":
				ref = resource.ID("../escape")
			case "nil writer":
				output = nil
			case "typed nil writer":
				output = (*downloadSink)(nil)
			case "nil context":
				ctx = nil
			case "canceled":
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
				want = context.Canceled
			}
			service := image.New(client)
			if mode == "nil service" {
				service = nil
			}
			v, err := service.DownloadTo(ctx, ref, output)
			if v != nil || !errors.Is(err, want) || calls.Load() != 0 || sink.writes.Load() != 0 {
				t.Fatal(v, err, calls.Load(), sink.writes.Load())
			}
		})
	}
}

func TestDownloadToCanonicalIdentityAndIntegrity(t *testing.T) {
	payload := "payload"
	for _, algorithm := range []string{"md5", "sha1", "sha224", "sha256", "sha384", "sha512", "sha512_224", "sha512_256"} {
		t.Run(algorithm+" primary", func(t *testing.T) {
			digest := downloadDigest(algorithm, payload)
			cloud := testcloud.New(t)
			cloud.Provider.HTTPClient.Transport = downloadTransport(func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.URL.Path, "/file") {
					wire := downloadWire(200, "file", io.NopCloser(strings.NewReader(payload)))
					wire.Header.Set("Content-MD5", "ignored-header")
					return wire, nil
				}
				return downloadWire(200, "metadata", io.NopCloser(strings.NewReader(downloadMetadata(fmt.Sprintf(`,"os_hash_algo":%q,"os_hash_value":%q,"checksum":"ignored-metadata"`, algorithm, digest))))), nil
			})
			v, err := image.New(cloud.Client("image", downloadPrefix)).DownloadTo(context.Background(), resource.ID("fixed"), &downloadSink{})
			if err != nil || v == nil || v.Checksum == nil || v.Checksum.Algorithm != algorithm || v.Checksum.Expected != digest || v.Checksum.Actual != digest || !v.Checksum.Verified || !v.Checksum.Complete {
				t.Fatal(v, err)
			}
		})
	}
	md5Hex := downloadDigest("md5", payload)
	for _, tc := range []struct {
		name, fields, header, expected string
		disabled, mismatch, reject     bool
		unsupported                    bool
	}{
		{"metadata legacy wins", fmt.Sprintf(`,"checksum":%q`, md5Hex), "ignored-header", md5Hex, false, false, false, false},
		{"header only", "", md5Hex, md5Hex, false, false, false, false},
		{"incomplete primary fallback", fmt.Sprintf(`,"os_hash_algo":"sha512","checksum":%q`, md5Hex), "ignored-header", md5Hex, false, false, false, false},
		{"incomplete value fallback", `,"os_hash_value":"unused"`, md5Hex, md5Hex, false, false, false, false},
		{"no hashes", "", "", "", false, false, false, false},
		{"disabled malformed hash properties", `,"os_hash_algo":false,"os_hash_value":12`, "wrong", "", true, false, false, false},
		{"typed canonical hash", `,"os_hash_algo":false,"os_hash_value":"digest"`, "", "", false, false, true, false},
		{"unsupported primary", `,"os_hash_algo":"future-hash","os_hash_value":"digest"`, md5Hex, "", false, false, true, true},
		{"uppercase hex retained", "", strings.ToUpper(md5Hex), strings.ToUpper(md5Hex), false, true, false, false},
		{"base64 MD5 retained", "", func() string { v, _ := hex.DecodeString(md5Hex); return base64.StdEncoding.EncodeToString(v) }(), func() string { v, _ := hex.DecodeString(md5Hex); return base64.StdEncoding.EncodeToString(v) }(), false, true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var files atomic.Int32
			sink := &downloadSink{}
			cloud.Provider.HTTPClient.Transport = downloadTransport(func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.URL.Path, "/file") {
					files.Add(1)
					wire := downloadWire(200, "file", io.NopCloser(strings.NewReader(payload)))
					wire.Header.Set("Content-MD5", tc.header)
					return wire, nil
				}
				return downloadWire(200, "metadata", io.NopCloser(strings.NewReader(downloadMetadata(tc.fields)))), nil
			})
			var opts []image.DownloadImageOption
			if tc.disabled {
				opts = append(opts, image.WithDownloadChecksumVerification(false))
			}
			v, err := image.New(cloud.Client("image", downloadPrefix)).DownloadTo(context.Background(), resource.ID("fixed"), sink, opts...)
			if v == nil || v.Metadata == nil {
				t.Fatal(v, err)
			}
			if tc.reject {
				if err == nil || files.Load() != 0 || sink.writes.Load() != 0 || v.StatusCode != 0 || tc.unsupported && !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal(v, err, files.Load(), sink.writes.Load())
				}
				return
			}
			if files.Load() != 1 || sink.String() != payload || v.BytesWritten != int64(len(payload)) {
				t.Fatal(v, err, files.Load(), sink.String())
			}
			if tc.expected == "" {
				if err != nil || v.Checksum != nil {
					t.Fatal(v, err)
				}
			} else if tc.mismatch {
				var mismatch *image.DownloadChecksumMismatchError
				if !errors.Is(err, image.ErrChecksumMismatch) || !errors.As(err, &mismatch) || mismatch.Algorithm != "md5" || mismatch.Expected != tc.expected || mismatch.Actual != md5Hex || v.Checksum == nil || !v.Checksum.Complete || v.Checksum.Verified || v.Checksum.Actual != md5Hex {
					t.Fatal(v, err, mismatch)
				}
			} else if err != nil || v.Checksum == nil || !v.Checksum.Complete || !v.Checksum.Verified || v.Checksum.Expected != tc.expected {
				t.Fatal(v, err)
			}
		})
	}
	for _, metadata := range []string{`{"ID":"fixed"}`, `{"id":null}`, `{"id":"other","file":"/images/fixed/file"}`, `{"id":"../escape"}`} {
		t.Run("canonical ID "+metadata, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			sink := &downloadSink{}
			cloud.Provider.HTTPClient.Transport = downloadTransport(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return downloadWire(200, "metadata", io.NopCloser(strings.NewReader(metadata))), nil
			})
			v, err := image.New(cloud.Client("image", downloadPrefix)).DownloadTo(context.Background(), resource.ID("fixed"), sink)
			var proof *resource.ResponseError
			if v == nil || v.Metadata == nil || string(v.Metadata.Body) != metadata || !errors.As(err, &proof) || proof.StatusCode != 200 || calls.Load() != 1 || sink.writes.Load() != 0 || v.StatusCode != 0 {
				t.Fatal(v, err, proof, calls.Load())
			}
		})
	}
	t.Run("Name resolves exact matches across pages then fresh metadata", func(t *testing.T) {
		cloud := testcloud.New(t)
		var sequence createImportSequence
		cloud.Mux.HandleFunc("GET "+downloadPrefix+"images", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("marker") == "page2" {
				sequence.add("list2")
				testcloud.JSON(w, 200, `{"images":[{"id":"wrong","name":"needle-suffix"}]}`)
				return
			}
			sequence.add("list1")
			if r.URL.Query().Get("name") != "needle" {
				t.Error(r.URL.String())
			}
			// Native image pagination combines the service prefix with Glance's
			// version-relative next path; this is the existing list wire contract.
			testcloud.JSON(w, 200, `{"images":[{"id":"fixed","name":"needle","os_hash_algo":"future-hash","os_hash_value":"stale"}],"next":"/v2/images?marker=page2"}`)
		})
		cloud.Mux.HandleFunc("GET "+downloadPrefix+"images/fixed", func(w http.ResponseWriter, r *http.Request) {
			sequence.add("metadata")
			testcloud.JSON(w, 200, downloadMetadata(""))
		})
		cloud.Mux.HandleFunc("GET "+downloadPrefix+"images/fixed/file", func(w http.ResponseWriter, r *http.Request) { sequence.add("file"); _, _ = io.WriteString(w, payload) })
		sink := &downloadSink{}
		v, err := image.New(cloud.Client("image", downloadPrefix)).DownloadTo(context.Background(), resource.Name("needle"), sink)
		if err != nil || v == nil || v.ImageID != "fixed" || sink.String() != payload || !reflect.DeepEqual(sequence.value(), []string{"list1", "list2", "metadata", "file"}) {
			t.Fatal(v, err, sequence.value())
		}
	})
}

func TestDownloadToPreparedSnapshotsAndPreferences(t *testing.T) {
	cloud := testcloud.New(t)
	chunk, verify := 3, false
	preferences := []string{"fast", "slow & east?", "fast"}
	bulk := image.WithDownloadImageOpts(image.DownloadImageOpts{ChunkSize: &chunk, VerifyChecksum: &verify, StorePreferences: preferences})
	chunk, verify, preferences[0] = -1, true, "changed"
	var retained *image.DownloadImageOpts
	var callbacks, calls atomic.Int32
	body := &downloadBody{Reader: strings.NewReader("payload")}
	cloud.Provider.HTTPClient.Transport = downloadTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if callbacks.Load() != 1 {
			t.Error("option callback deferred or repeated", callbacks.Load())
		}
		*retained.ChunkSize = 0
		*retained.VerifyChecksum = true
		retained.StorePreferences[0] = "late"
		if strings.HasSuffix(r.URL.Path, "/file") {
			want := url.Values{"prefer": {"fast,slow & east?,fast"}}.Encode()
			if r.URL.RawQuery != want || len(r.URL.Query()) != 1 {
				t.Error(r.URL.String(), want)
			}
			return downloadWire(200, "file", body), nil
		}
		if r.URL.RawQuery != "" {
			t.Error("prefer leaked to metadata", r.URL.String())
		}
		return downloadWire(200, "metadata", io.NopCloser(strings.NewReader(downloadMetadata(`,"os_hash_algo":"future-hash","os_hash_value":"wrong"`)))), nil
	})
	sink := &downloadSink{}
	v, err := image.New(cloud.Client("image", downloadPrefix)).DownloadTo(context.Background(), resource.ID("fixed"), sink, image.WithDownloadChunkSize(-1), bulk, func(opts *image.DownloadImageOpts) error { callbacks.Add(1); retained = opts; return nil })
	if err != nil || v == nil || v.Checksum != nil || sink.String() != "payload" || calls.Load() != 2 || callbacks.Load() != 1 || body.maxRead.Load() != 3 || sink.maxWrite.Load() > 3 {
		t.Fatal(v, err, calls.Load(), callbacks.Load(), body.maxRead.Load(), sink.maxWrite.Load())
	}
	for _, opts := range [][]image.DownloadImageOption{
		{image.WithDownloadStorePreferences("discarded"), image.WithDownloadImageOpts(image.DownloadImageOpts{})},
		{image.WithDownloadStorePreferences("discarded"), image.WithDownloadStorePreferences()},
	} {
		t.Run("replacement omits prefer", func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Provider.HTTPClient.Transport = downloadTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.RawQuery != "" {
					t.Error(r.URL.String())
				}
				if strings.HasSuffix(r.URL.Path, "/file") {
					return downloadWire(200, "file", io.NopCloser(strings.NewReader("payload"))), nil
				}
				return downloadWire(200, "metadata", io.NopCloser(strings.NewReader(downloadMetadata("")))), nil
			})
			v, err := image.New(cloud.Client("image", downloadPrefix)).DownloadTo(context.Background(), resource.ID("fixed"), &downloadSink{}, opts...)
			if err != nil || v == nil || v.BytesWritten != 7 {
				t.Fatal(v, err)
			}
		})
	}
}

type downloadReadFailure struct {
	prefix []byte
	cause  error
}

func (r *downloadReadFailure) Read(p []byte) (int, error) {
	n := copy(p, r.prefix)
	r.prefix = r.prefix[n:]
	return n, r.cause
}

func TestDownloadToPartialTransferFailures(t *testing.T) {
	readCause := errors.New("actual image response read failed")
	for _, tc := range []struct {
		name, metadata string
		code           int
		read           bool
	}{
		{"rejected metadata", `{"message":"missing"}`, 404, false},
		{"malformed metadata", `{"id":"fixed"`, 200, false},
		{"trailing metadata", `{"id":"fixed"}{}`, 200, false},
		{"array metadata", `[]`, 200, false},
		{"native field metadata", downloadMetadata(`,"min_ram":"bad"`), 200, false},
		{"native time metadata", downloadMetadata(`,"created_at":"bad"`), 200, false},
		{"UTF8 metadata", downloadMetadata(`,"vendor":"` + string([]byte{255}) + `"`), 200, false},
		{"read metadata", downloadMetadata(""), 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			sink := &downloadSink{}
			body := &downloadBody{Reader: strings.NewReader(tc.metadata)}
			if tc.read {
				body.Reader = &downloadReadFailure{prefix: []byte(tc.metadata), cause: readCause}
			}
			cloud.Provider.HTTPClient.Transport = downloadTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return downloadWire(tc.code, "metadata", body), nil
			})
			v, err := image.New(cloud.Client("image", downloadPrefix)).DownloadTo(context.Background(), resource.ID("fixed"), sink)
			if err == nil || calls.Load() != 1 || sink.writes.Load() != 0 || body.closes.Load() != 1 {
				t.Fatal(v, err, calls.Load(), sink.writes.Load(), body.closes.Load())
			}
			if tc.code != 200 {
				if v != nil || !gophercloud.ResponseCodeIs(err, tc.code) {
					t.Fatal(v, err)
				}
				return
			}
			var proof *resource.ResponseError
			if v == nil || v.Metadata == nil || v.Metadata.StatusCode != 200 || string(v.Metadata.Body) != tc.metadata || v.Metadata.Header.Get("X-Phase") != "metadata" || v.StatusCode != 0 || v.BytesWritten != 0 || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != tc.metadata {
				t.Fatal(v, err, proof)
			}
			if tc.read && !errors.Is(err, readCause) {
				t.Fatal("metadata read cause lost", err)
			}
			v.Metadata.Body[0] = '!'
			v.Metadata.Header.Set("X-Phase", "caller")
			if string(proof.Body) != tc.metadata || proof.Header.Get("X-Phase") != "metadata" {
				t.Fatal("metadata proof aliases caller result", proof)
			}
		})
	}
	for _, code := range []int{403, 404, 206} {
		t.Run("rejected file "+fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			body := &downloadBody{Reader: strings.NewReader("actual rejected file body")}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = downloadTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if strings.HasSuffix(r.URL.Path, "/file") {
					return downloadWire(code, "file", body), nil
				}
				return downloadWire(200, "metadata", io.NopCloser(strings.NewReader(downloadMetadata("")))), nil
			})
			sink := &downloadSink{}
			v, err := image.New(cloud.Client("image", downloadPrefix)).DownloadTo(context.Background(), resource.ID("fixed"), sink)
			var native gophercloud.ErrUnexpectedResponseCode
			if v == nil || v.Metadata == nil || v.StatusCode != 0 || v.BytesWritten != 0 || len(v.Header) != 0 || !errors.As(err, &native) || native.Actual != code || string(native.Body) != "actual rejected file body" || native.ResponseHeader.Get("X-Phase") != "file" || body.closes.Load() != 1 || sink.writes.Load() != 0 || calls.Load() != 2 {
				t.Fatal(v, err, native, body.closes.Load(), calls.Load())
			}
		})
	}
	for _, mode := range []string{"read", "write", "short write", "read and write", "read and close", "read write close canceled", "close only"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			writeCause, closeCause, cancelCause := errors.New("writer rejected suffix"), errors.New("response close failed"), errors.New("caller canceled partial transfer")
			body := &downloadBody{Reader: strings.NewReader("payload")}
			if strings.Contains(mode, "read") {
				body.Reader = &downloadReadFailure{prefix: []byte("abc"), cause: readCause}
			}
			if strings.Contains(mode, "close") {
				body.closeErr = closeCause
			}
			sink := &downloadSink{}
			if strings.Contains(mode, "write") {
				sink.onWrite = func(p []byte) (int, error) {
					_, _ = sink.Buffer.Write(p[:2])
					if mode == "read write close canceled" {
						cancel(cancelCause)
					}
					if mode == "short write" {
						return 2, nil
					}
					return 2, writeCause
				}
			}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = downloadTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if strings.HasSuffix(r.URL.Path, "/file") {
					return downloadWire(200, "file", body), nil
				}
				return downloadWire(200, "metadata", io.NopCloser(strings.NewReader(downloadMetadata(fmt.Sprintf(`,"checksum":%q`, downloadDigest("md5", "payload")))))), nil
			})
			v, err := image.New(cloud.Client("image", downloadPrefix)).DownloadTo(ctx, resource.ID("fixed"), sink, image.WithDownloadChunkSize(3))
			wantBytes := int64(3)
			if strings.Contains(mode, "write") {
				wantBytes = 2
			}
			if mode == "close only" {
				wantBytes = 7
			}
			if err == nil || v == nil || v.Metadata == nil || v.StatusCode != 200 || v.Header.Get("X-Phase") != "file" || v.BytesWritten != wantBytes || int64(sink.Len()) != wantBytes || body.closes.Load() != 1 || calls.Load() != 2 || sink.closes.Load() != 0 {
				t.Fatal(v, err, wantBytes, sink.Len(), body.closes.Load(), calls.Load())
			}
			if strings.Contains(mode, "read") && !errors.Is(err, readCause) || strings.Contains(mode, "close") && !errors.Is(err, closeCause) || strings.Contains(mode, "write") && mode != "short write" && !errors.Is(err, writeCause) || mode == "short write" && !errors.Is(err, io.ErrShortWrite) {
				t.Fatal("transfer cause lost", err)
			}
			if mode == "read write close canceled" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
				t.Fatal("cancellation cause lost", err)
			}
			if v.Checksum == nil {
				t.Fatal("selected checksum evidence absent", v, err)
			}
			if mode == "close only" {
				if !v.Checksum.Complete || !v.Checksum.Verified || v.Checksum.Actual != downloadDigest("md5", "payload") {
					t.Fatal("close failure erased completed integrity evidence", v.Checksum, err)
				}
			} else if v.Checksum.Complete || v.Checksum.Verified || v.Checksum.Actual != "" {
				t.Fatal("partial bytes claimed complete digest", v.Checksum, err)
			}
		})
	}
}

func TestDownloadToNoDataAndNativeCompatibility(t *testing.T) {
	t.Run("204 preserves no-data response without writing or hashing", func(t *testing.T) {
		cloud := testcloud.New(t)
		body := &downloadBody{Reader: strings.NewReader("ignored no-data response")}
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = downloadTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if strings.HasSuffix(r.URL.Path, "/file") {
				return downloadWire(204, "file", body), nil
			}
			return downloadWire(200, "metadata", io.NopCloser(strings.NewReader(downloadMetadata(`,"checksum":"unused"`)))), nil
		})
		sink := &downloadSink{}
		v, err := image.New(cloud.Client("image", downloadPrefix)).DownloadTo(context.Background(), resource.ID("fixed"), sink)
		if err != nil || v == nil || v.Metadata == nil || v.StatusCode != 204 || v.Header.Get("X-Phase") != "file" || v.BytesWritten != 0 || v.Checksum != nil || sink.writes.Load() != 0 || body.closes.Load() != 1 || calls.Load() != 2 {
			t.Fatal(v, err, sink.writes.Load(), body.closes.Load(), calls.Load())
		}
	})
	for _, code := range []int{200, 204} {
		t.Run("native raw stream "+fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			body := &downloadBody{Reader: strings.NewReader("raw-native")}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = downloadTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Method != http.MethodGet || r.URL.Path != downloadPrefix+"images/fixed/file" || r.URL.RawQuery != "" {
					t.Error("raw Download acquired workflow behavior", r.Method, r.URL)
				}
				return downloadWire(code, "raw-native", body), nil
			})
			v, err := imagedata.New(cloud.Client("image", downloadPrefix)).Download(context.Background(), "fixed")
			if code == 204 {
				if v != nil || !gophercloud.ResponseCodeIs(err, 204) || body.closes.Load() < 1 || calls.Load() != 1 {
					t.Fatal(v, err, body.closes.Load(), calls.Load())
				}
				return
			}
			if err != nil || v == nil || v.Body != body || v.Header != body || body.closes.Load() != 0 {
				t.Fatal(v, err, body.closes.Load())
			}
			data, err := io.ReadAll(v)
			_ = v.Close()
			if err != nil || string(data) != "raw-native" || body.closes.Load() != 1 || calls.Load() != 1 {
				t.Fatal(string(data), err, body.closes.Load(), calls.Load())
			}
		})
	}
}

func TestDownloadToProviderSourceAndRetryBoundaries(t *testing.T) {
	t.Run("metadata reauth file retry occur before output and use live token", func(t *testing.T) {
		cloud := testcloud.New(t)
		cloud.Provider.SetToken("initial")
		var metadataCalls, fileCalls, reauth, retries atomic.Int32
		sink := &downloadSink{}
		body := &downloadBody{Reader: strings.NewReader("payload")}
		cloud.Provider.ReauthFunc = func(context.Context) error { reauth.Add(1); cloud.Provider.SetToken("reauthenticated"); return nil }
		cloud.Provider.RetryFunc = func(_ context.Context, method, target string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			retries.Add(1)
			if method != http.MethodGet || !strings.HasSuffix(target, "/file") || !gophercloud.ResponseCodeIs(err, 503) {
				return err
			}
			cloud.Provider.SetToken("retry-file")
			return nil
		}
		cloud.Provider.HTTPClient.Transport = downloadTransport(func(r *http.Request) (*http.Response, error) {
			if sink.writes.Load() != 0 {
				t.Error("request replayed after writer bytes")
			}
			if !strings.HasSuffix(r.URL.Path, "/file") {
				if metadataCalls.Add(1) == 1 {
					if r.Header.Get("X-Auth-Token") != "initial" {
						t.Error(r.Header)
					}
					return downloadWire(401, "metadata", io.NopCloser(strings.NewReader(`{}`))), nil
				}
				if r.Header.Get("X-Auth-Token") != "reauthenticated" {
					t.Error(r.Header)
				}
				cloud.Provider.SetToken("after-metadata")
				return downloadWire(200, "metadata", io.NopCloser(strings.NewReader(downloadMetadata("")))), nil
			}
			if fileCalls.Add(1) == 1 {
				if r.Header.Get("X-Auth-Token") != "after-metadata" {
					t.Error(r.Header)
				}
				return downloadWire(503, "file", io.NopCloser(strings.NewReader(`{}`))), nil
			}
			if r.Header.Get("X-Auth-Token") != "retry-file" {
				t.Error(r.Header)
			}
			return downloadWire(200, "file", body), nil
		})
		v, err := image.New(cloud.Client("image", downloadPrefix)).DownloadTo(context.Background(), resource.ID("fixed"), sink)
		if err != nil || v == nil || sink.String() != "payload" || metadataCalls.Load() != 2 || fileCalls.Load() != 2 || reauth.Load() != 1 || retries.Load() != 1 || body.closes.Load() != 1 || cloud.Provider.ReauthFunc == nil || cloud.Provider.RetryFunc == nil {
			t.Fatal(v, err, metadataCalls.Load(), fileCalls.Load(), reauth.Load(), retries.Load(), body.closes.Load())
		}
	})
	for _, change := range []string{"valid prefix and headers", "provider", "type"} {
		t.Run("original source changes after metadata "+change, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", downloadPrefix)
			client.MoreHeaders = map[string]string{"X-Source": "captured"}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = downloadTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if !strings.HasPrefix(r.URL.Path, downloadPrefix) || r.Header.Get("X-Source") != "captured" {
					t.Error("source change retargeted request", r.URL, r.Header)
				}
				if strings.HasSuffix(r.URL.Path, "/file") {
					return downloadWire(200, "file", io.NopCloser(strings.NewReader("payload"))), nil
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
				return downloadWire(200, "metadata", io.NopCloser(strings.NewReader(downloadMetadata("")))), nil
			})
			sink := &downloadSink{}
			v, err := image.New(client).DownloadTo(context.Background(), resource.ID("fixed"), sink)
			if change == "valid prefix and headers" {
				if err != nil || v == nil || sink.String() != "payload" || calls.Load() != 2 || client.MoreHeaders["X-Source"] != "later" {
					t.Fatal(v, err, calls.Load(), client)
				}
			} else {
				want := error(resource.ErrInvalidOption)
				if change == "type" {
					want = resource.ErrUnsupported
				}
				if !errors.Is(err, want) || v == nil || v.Metadata == nil || v.StatusCode != 0 || sink.writes.Load() != 0 || calls.Load() != 1 {
					t.Fatal(v, err, calls.Load(), sink.writes.Load())
				}
			}
		})
	}
	t.Run("binary redirects do not follow or alter caller redirect policy", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls, redirects atomic.Int32
		cloud.Provider.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects.Add(1); return nil }
		body := &downloadBody{Reader: strings.NewReader("redirect proof")}
		cloud.Provider.HTTPClient.Transport = downloadTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if strings.HasSuffix(r.URL.Path, "/file") {
				wire := downloadWire(303, "file", body)
				wire.Header.Set("Location", cloud.Server.URL+"/redirected")
				return wire, nil
			}
			return downloadWire(200, "metadata", io.NopCloser(strings.NewReader(downloadMetadata("")))), nil
		})
		v, err := image.New(cloud.Client("image", downloadPrefix)).DownloadTo(context.Background(), resource.ID("fixed"), &downloadSink{})
		if v == nil || v.Metadata == nil || v.StatusCode != 0 || !gophercloud.ResponseCodeIs(err, 303) || calls.Load() != 2 || redirects.Load() != 0 || body.closes.Load() != 1 {
			t.Fatal(v, err, calls.Load(), redirects.Load(), body.closes.Load())
		}
		if err := cloud.Provider.HTTPClient.CheckRedirect(nil, nil); err != nil || redirects.Load() != 1 {
			t.Fatal("caller redirect policy changed", err, redirects.Load())
		}
	})
}

type downloadContextReader struct {
	ctx    context.Context
	prefix []byte
}

func (r *downloadContextReader) Read(p []byte) (int, error) {
	if len(r.prefix) > 0 {
		n := copy(p, r.prefix)
		r.prefix = r.prefix[n:]
		return n, nil
	}
	<-r.ctx.Done()
	return 0, r.ctx.Err()
}

func TestDownloadToBorrowedWriterCancellationAndClose(t *testing.T) {
	t.Run("caller releases blocked Write after custom cancellation", func(t *testing.T) {
		cloud := testcloud.New(t)
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		cause, writeCause := errors.New("caller canceled blocked output"), errors.New("blocked writer stopped")
		started, release := make(chan struct{}), make(chan struct{})
		var signal, releasing sync.Once
		t.Cleanup(func() { releasing.Do(func() { close(release) }) })
		body := &downloadBody{Reader: strings.NewReader("payload")}
		cloud.Provider.HTTPClient.Transport = downloadTransport(func(r *http.Request) (*http.Response, error) {
			if strings.HasSuffix(r.URL.Path, "/file") {
				return downloadWire(200, "file", body), nil
			}
			return downloadWire(200, "metadata", io.NopCloser(strings.NewReader(downloadMetadata(fmt.Sprintf(`,"checksum":%q`, downloadDigest("md5", "payload")))))), nil
		})
		sink := &downloadSink{}
		sink.onWrite = func(p []byte) (int, error) {
			signal.Do(func() { close(started) })
			<-release
			_, _ = sink.Buffer.Write(p[:1])
			return 1, writeCause
		}
		type outcome struct {
			value *image.DownloadImageResult
			err   error
		}
		finished := make(chan outcome, 1)
		go func() {
			v, err := image.New(cloud.Client("image", downloadPrefix)).DownloadTo(ctx, resource.ID("fixed"), sink, image.WithDownloadChunkSize(3))
			finished <- outcome{v, err}
		}()
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("download did not start borrowed writer")
		}
		cancel(cause)
		if sink.closes.Load() != 0 {
			t.Error("SDK closed writer during cancellation")
		}
		releasing.Do(func() { close(release) })
		select {
		case got := <-finished:
			if got.value == nil || got.value.Metadata == nil || got.value.StatusCode != 200 || got.value.BytesWritten != 1 || !errors.Is(got.err, context.Canceled) || !errors.Is(got.err, cause) || !errors.Is(got.err, writeCause) || body.closes.Load() != 1 || sink.closes.Load() != 0 || got.value.Checksum == nil || got.value.Checksum.Complete || got.value.Checksum.Verified {
				t.Fatal(got, body.closes.Load(), sink.closes.Load())
			}
		case <-time.After(time.Second):
			t.Fatal("caller release did not complete transfer")
		}
	})
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
			var body *downloadBody
			cloud.Provider.HTTPClient.Transport = downloadTransport(func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.URL.Path, "/file") {
					body = &downloadBody{Reader: &downloadContextReader{ctx: r.Context(), prefix: []byte("abc")}}
					return downloadWire(200, "file", body), nil
				}
				return downloadWire(200, "metadata", io.NopCloser(strings.NewReader(downloadMetadata("")))), nil
			})
			sink := &downloadSink{}
			v, err := image.New(cloud.Client("image", downloadPrefix)).DownloadTo(ctx, resource.ID("fixed"), sink, image.WithDownloadChunkSize(3))
			if !errors.Is(err, context.DeadlineExceeded) || v == nil || v.Metadata == nil || v.StatusCode != 200 || v.BytesWritten != 3 || sink.String() != "abc" || body == nil || body.closes.Load() != 1 || sink.closes.Load() != 0 {
				t.Fatal(v, err, sink.String(), body)
			}
			if mode == "HTTP client timeout" && cloud.Provider.HTTPClient.Timeout != 100*time.Millisecond {
				t.Fatal("caller HTTP timeout changed")
			}
		})
	}
}
