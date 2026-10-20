package objects_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type cloudGetWriter struct {
	bytes.Buffer
	writes, flushes, closes int
	writeErr, flushErr      error
	short                   bool
	onWrite, onFlush        func()
}

func (w *cloudGetWriter) Write(data []byte) (int, error) {
	w.writes++
	if w.onWrite != nil {
		w.onWrite()
	}
	if w.writeErr != nil {
		return 0, w.writeErr
	}
	if w.short {
		return 0, nil
	}
	return w.Buffer.Write(data)
}
func (w *cloudGetWriter) Flush() error {
	w.flushes++
	if w.onFlush != nil {
		w.onFlush()
	}
	return w.flushErr
}
func (w *cloudGetWriter) Close() error { w.closes++; return nil }

func TestPythonCloudObjectGetDefaultTextLowerHeadersEagerAndIgnoredControls(t *testing.T) {
	calls := 0
	payload := []byte{'a', 0, 255, 0xe2, 0x82}
	body := &objectMetadataBody{Reader: bytes.NewReader(payload)}
	wire := objectMetadataWire(203, body)
	wire.Header.Set("Content-Type", "application/json")
	wire.Header["X-Unknown"] = []string{"one", "two"}
	wire.Header["Set-Cookie"] = []string{"a=1", "b=2"}
	client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "GET" || r.URL.String() != objectMetadataBase+"box/dir%2Ffile" || r.Body != nil && r.Body != http.NoBody || r.Header.Get("Accept") != "*/*" || r.Header.Get("Accept-Encoding") != "gzip, deflate, zstd" || r.Header.Get("X-Newest") != "" {
			t.Error(r)
		}
		return wire, nil
	})
	result, e := objects.New(client).GetCloudObjectText(context.Background(), "box", "dir/file", objects.WithCloudGetQueryString("%broken\n?#"), objects.WithCloudGetStream(true), objects.WithCloudGetChunkSize(0))
	if e != nil || result == nil || result.Body == nil || *result.Body != "a\x00\ufffd\ufffd" || result.Missing || result.BytesWritten != 0 || calls != 1 || body.closes.Load() != 1 || result.Headers["x-unknown"] != "one, two" || result.Headers["set-cookie"] != "a=1, b=2" || result.Headers["content-type"] != "application/json" || result.Receipt.StatusCode != 203 {
		t.Fatal(result, e, calls, body.closes.Load())
	}
	for key := range result.Headers {
		if key != strings.ToLower(key) {
			t.Fatal(key)
		}
	}
	result.Headers["x-trans-id"] = "forged"
	result.Receipt.Header.Set("X-Trans-Id", "another")
	if wire.Header.Get("X-Trans-Id") != "actual-object" {
		t.Fatal(wire.Header)
	}
}

func TestPythonCloudObjectGetRequestsHeaderCharsetDefaultsReplacementAndAliases(t *testing.T) {
	for _, tc := range []struct {
		name, contentType string
		payload           []byte
		want              string
	}{
		{"text default latin1", "text/plain", []byte{0x80, 0xe9, 0xff}, "\u0080éÿ"},
		{"case sensitive media text substring", "application/x-mytext", []byte{0xe9}, "é"},
		{"json defaultutf8", "application/json", []byte{0xff, 0xe2, 0x82}, "\ufffd\ufffd"},
		{"explicitlatin1", "application/octet-stream; charset=latin_1", []byte{0x80, 0xe9}, "\u0080é"},
		{"explicitwindows1252", "application/octet-stream; charset=cp1252", []byte{0x80, 0xe9}, "€é"},
		{"explicitascii", "text/plain; charset='ascii'", []byte{0x80, 'x'}, "\ufffdx"},
		{"duplicate charset lastwins", "text/plain; charset=ascii; charset=iso-8859-1", []byte{0xe9}, "é"},
		{"empty charset fallback", "text/plain; charset=", []byte{0xe2, 0x82}, "\ufffd"},
		{"unknown charset fallback", "text/plain; charset=definitely-unknown", []byte{0xff}, "\ufffd"},
		{"explicitutf8 keepsbom", "application/json; charset=utf-8", []byte{0xef, 0xbb, 0xbf, 'a'}, "\ufeffa"},
		{"utf8sig stripsbom", "text/plain; charset=utf-8-sig", []byte{0xef, 0xbb, 0xbf, 'a'}, "a"},
		{"iso8859 alias", "x; charset=iso8859", []byte{0xe9}, "é"},
		{"ANSI1986 alias", "x; charset=ANSI_X3.4_1986", []byte{0xc3, 0xa9}, "\ufffd\ufffd"},
		{"ShiftJIS raw IANA alias", "x; charset=shift_jis", []byte{0x93, 0xfa, 0x96, 0x7b}, "日本"},
		{"explicit UTF32 short replacement", "x; charset=utf-32", []byte{0x41}, "\ufffd"},
		{"explicit UTF32 shortBOM replacement", "x; charset=utf-32", []byte{0xff, 0xfe, 0}, "\ufffd"},
		{"emptybody", "", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &objectMetadataBody{Reader: bytes.NewReader(tc.payload)}
			wire := objectMetadataWire(200, body)
			wire.Header.Set("Content-Type", tc.contentType)
			result, e := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })).GetCloudObjectText(context.Background(), "box", "file")
			if e != nil || result.Body == nil || *result.Body != tc.want || body.closes.Load() != 1 {
				t.Fatal(result, e, body.closes.Load())
			}
		})
	}
}

func TestPythonCloudObjectGetUnmarkedBOMAndExplicitUTF16UTF32(t *testing.T) {
	for _, tc := range []struct {
		name, contentType string
		payload           []byte
		want              string
	}{
		{"UTF8BOM", "", []byte{0xef, 0xbb, 0xbf, 'a'}, "a"},
		{"UTF16LEBOM", "", []byte{0xff, 0xfe, 'a', 0}, "a"},
		{"UTF16BEBOM", "", []byte{0xfe, 0xff, 0, 'a'}, "a"},
		{"UTF32LEBOM", "", []byte{0xff, 0xfe, 0, 0, 'a', 0, 0, 0}, "a"},
		{"UTF32BEBOM", "", []byte{0, 0, 0xfe, 0xff, 0, 0, 0, 'a'}, "a"},
		{"UTF16LE noBOM", "x; charset=utf-16le", []byte{'a', 0, 0x00, 0xd8}, "a\ufffd"},
		{"UTF16LE surrogate oddtail", "x; charset=utf-16-le", []byte{0, 0xd8, 0x41}, "\ufffd"},
		{"UTF16BE surrogate oddtail", "x; charset=utf-16-be", []byte{0xd8, 0, 0x41}, "\ufffd"},
		{"UTF16 BOM surrogate oddtail", "x; charset=utf-16", []byte{0xfe, 0xff, 0xd8, 0, 0x41}, "\ufffd"},
		{"UTF16BE explicitkeepsBOM", "x; charset=utf-16-be", []byte{0xfe, 0xff, 0, 'a'}, "\ufeffa"},
		{"UTF32 noBOM", "x; charset=utf-32le", []byte{'a', 0, 0, 0}, "a"},
		{"validUTF8noheader", "", []byte("한글"), "한글"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &objectMetadataBody{Reader: bytes.NewReader(tc.payload)}
			wire := objectMetadataWire(200, body)
			wire.Header.Set("Content-Type", tc.contentType)
			result, e := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })).GetCloudObjectText(context.Background(), "box", "file")
			if e != nil || result.Body == nil || *result.Body != tc.want || body.closes.Load() != 1 {
				t.Fatal(result, e, body.closes.Load())
			}
		})
	}
}

func TestPythonCloudObjectGetStatisticalCharsetGapIsExplicitWithOwnedActualEvidence(t *testing.T) {
	for _, contentType := range []string{"", "application/octet-stream", "Text/Plain"} {
		t.Run(contentType, func(t *testing.T) {
			payload := []byte{'c', 'a', 'f', 0xe9}
			body := &objectMetadataBody{Reader: bytes.NewReader(payload)}
			wire := objectMetadataWire(200, body)
			wire.Header.Set("Content-Type", contentType)
			result, e := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })).GetCloudObjectText(context.Background(), "box", "file")
			if !errors.Is(e, resource.ErrUnsupported) || result == nil || result.Body != nil || result.Receipt.StatusCode != 200 || body.closes.Load() != 1 {
				t.Fatal(result, e, body.closes.Load())
			}
			proof := objectMetadataProof(t, e, 200, string(payload))
			result.Receipt.Header.Set("X-Trans-Id", "forged")
			result.Headers["x-trans-id"] = "forged2"
			if proof.Header.Get("X-Trans-Id") != "actual-object" || !bytes.Equal(proof.Body, payload) {
				t.Fatal(proof)
			}
		})
	}
}

func TestPythonCloudObjectGetPathOutputBinaryOwnedCloseAndMissingNoTruncation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output.bin")
	if e := os.WriteFile(path, []byte("old"), 0o600); e != nil {
		t.Fatal(e)
	}
	payload := []byte{0, 255, 'a', 'b', 'c'}
	calls := 0
	client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
		calls++
		if strings.HasSuffix(r.URL.Path, "missing") {
			return objectMetadataResponse(404, "not found"), nil
		}
		wire := objectMetadataResponse(200, string(payload))
		wire.Header.Set("Content-Type", "application/octet-stream")
		return wire, nil
	})
	result, e := objects.New(client).GetCloudObjectText(context.Background(), "box", "missing", objects.WithCloudGetOutfile(path))
	stored, _ := os.ReadFile(path)
	if e != nil || !result.Missing || result.Body != nil || result.Headers != nil || result.Receipt.StatusCode != 404 || string(stored) != "old" {
		t.Fatal(result, e, stored)
	}
	result, e = objects.New(client).GetCloudObjectText(context.Background(), "box", "file", objects.WithCloudGetOutfile(path), objects.WithCloudGetChunkSize(2))
	stored, _ = os.ReadFile(path)
	if e != nil || result.Missing || result.Body != nil || result.BytesWritten != int64(len(payload)) || !bytes.Equal(stored, payload) || calls != 2 {
		t.Fatal(result, e, stored, calls)
	}
	// A complete path result has relinquished its owned handle; it can be moved.
	if e := os.Rename(path, path+".done"); e != nil {
		t.Fatal(e)
	}
}

func TestPythonCloudObjectGetBorrowedWriterFlushOnlyAndNoStatisticalTextRead(t *testing.T) {
	writer := &cloudGetWriter{}
	body := &objectMetadataBody{Reader: strings.NewReader("caf\xe9")}
	wire := objectMetadataWire(200, body)
	wire.Header.Set("Content-Type", "application/octet-stream")
	result, e := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })).GetCloudObjectText(context.Background(), "box", "file", objects.WithCloudGetWriter(writer), objects.WithCloudGetChunkSize(2))
	if e != nil || result.Body != nil || result.BytesWritten != 4 || writer.String() != "caf\xe9" || writer.writes != 2 || writer.flushes != 1 || writer.closes != 0 || body.closes.Load() != 1 {
		t.Fatal(result, e, writer.writes, writer.flushes, writer.closes, body.closes.Load())
	}
	raw := &bytes.Buffer{}
	result, e = objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return objectMetadataResponse(200, "raw"), nil })).GetCloudObjectText(context.Background(), "box", "file", objects.WithCloudGetWriter(raw))
	if e != nil || raw.String() != "raw" || result.BytesWritten != 3 {
		t.Fatal(result, e, raw)
	}
}

func TestPythonCloudObjectGetFalseyNonNoneOutfileSkipsReadWriteTextAndChunkValidation(t *testing.T) {
	for _, option := range []objects.CloudGetOption{objects.WithCloudGetOutfile(""), objects.WithoutCloudGetContent()} {
		reads := 0
		body := &objectMetadataBody{Reader: objectMetadataReader(func(data []byte) (int, error) { reads++; return copy(data, "unexpected"), io.EOF })}
		result, e := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return objectMetadataWire(200, body), nil })).GetCloudObjectText(context.Background(), "box", "file", option, objects.WithCloudGetChunkSize(-1))
		if e != nil || result.Body != nil || result.Missing || result.Receipt.StatusCode != 200 || reads != 0 || body.closes.Load() != 1 {
			t.Fatal(result, e, reads, body.closes.Load())
		}
	}
}

func TestPythonCloudObjectGetWriterFailureShortWriteFlushFailurePreserveStageAndCleanup(t *testing.T) {
	for _, mode := range []string{"write", "short", "flush"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("writer " + mode)
			writer := &cloudGetWriter{}
			switch mode {
			case "write":
				writer.writeErr = cause
			case "short":
				writer.short = true
			case "flush":
				writer.flushErr = cause
			}
			body := &objectMetadataBody{Reader: strings.NewReader("abcd")}
			result, e := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return objectMetadataWire(206, body), nil })).GetCloudObjectText(context.Background(), "box", "file", objects.WithCloudGetWriter(writer), objects.WithCloudGetChunkSize(2))
			if mode == "short" {
				cause = io.ErrShortWrite
			}
			if !errors.Is(e, cause) || result.Body != nil || writer.closes != 0 || body.closes.Load() != 1 || mode != "flush" && writer.flushes != 0 || mode == "flush" && (writer.flushes != 1 || result.BytesWritten != 4) {
				t.Fatal(result, e, writer.writes, writer.flushes, writer.closes, body.closes.Load())
			}
			var proof *resource.ResponseError
			if !errors.As(e, &proof) || proof.StatusCode != 206 || proof.Header.Get("X-Trans-Id") != "actual-object" {
				t.Fatal(e, proof)
			}
		})
	}
}

func TestPythonCloudObjectGetTranslatedIntegerHeaderKeepsRawValueAndReadFailuresRemainFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output")
	if e := os.WriteFile(path, []byte("old"), 0o600); e != nil {
		t.Fatal(e)
	}
	reads := 0
	body := &objectMetadataBody{Reader: objectMetadataReader(func(data []byte) (int, error) { reads++; return copy(data, "unexpected"), io.EOF })}
	wire := objectMetadataWire(200, body)
	wire.Header.Set("Content-Length", "bad")
	result, e := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })).GetCloudObjectText(context.Background(), "box", "file", objects.WithCloudGetOutfile(path))
	stored, _ := os.ReadFile(path)
	if e != nil || result.Receipt.StatusCode != 200 || result.Headers["content-length"] != "bad" || reads != 1 || string(stored) != "unexpected" || body.closes.Load() != 1 {
		t.Fatal(result, e, reads, stored, body.closes.Load())
	}
	cause := errors.New("buffered physical read before Header")
	body = &objectMetadataBody{Reader: objectMetadataReader(func(data []byte) (int, error) { return copy(data, "partial"), cause })}
	wire = objectMetadataWire(200, body)
	wire.Header.Set("Content-Length", "bad")
	_, e = objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })).GetCloudObjectText(context.Background(), "box", "file")
	if !errors.Is(e, cause) || errors.Is(e, resource.ErrInvalidOption) {
		t.Fatal(e)
	}
	objectMetadataProof(t, e, 200, "partial")
}

func TestPythonCloudObjectGetPathOpenFailurePrecedesBodyCodecAndAlwaysClosesResponse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missingdir", "output")
	reads := 0
	body := &objectMetadataBody{Reader: objectMetadataReader(func(data []byte) (int, error) { reads++; return copy(data, "broken"), io.EOF })}
	wire := objectMetadataWire(200, body)
	wire.Header.Set("Content-Encoding", "gzip")
	result, e := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })).GetCloudObjectText(context.Background(), "box", "file", objects.WithCloudGetOutfile(path))
	var pathErr *os.PathError
	if !errors.As(e, &pathErr) || result.Receipt.StatusCode != 200 || reads != 0 || body.closes.Load() != 1 {
		t.Fatal(result, e, reads, body.closes.Load())
	}
}

func TestPythonCloudObjectGetBufferedPhysicalFaultsAnd401OriginalRemainActual(t *testing.T) {
	for _, status := range []int{200, 206, 401, 404, 429, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			readCause, closeCause, unexpected := errors.New("get physical read"), errors.New("get physical close"), errors.New("bounded unexpected reauth")
			calls, reads, reauths, retries := 0, 0, 0, 0
			body := &objectMetadataBody{Reader: objectMetadataReader(func(dst []byte) (int, error) {
				reads++
				if reads > 1 {
					return 0, io.EOF
				}
				return copy(dst, "partial actual"), readCause
			}), closeErr: closeCause}
			client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
				calls++
				if calls > 1 {
					return objectMetadataResponse(500, "bounded unexpected replay"), nil
				}
				return objectMetadataWire(status, body), nil
			})
			client.ProviderClient.ReauthFunc = func(context.Context) error { reauths++; return unexpected }
			client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return unexpected
			}
			result, e := objects.New(client).GetCloudObjectText(context.Background(), "box", "file")
			if result == nil || result.Body != nil || result.Missing || result.Receipt.StatusCode != status || calls != 1 || reauths != 0 || retries != 0 || body.closes.Load() != 1 {
				t.Fatal(result, e, calls, reauths, retries, body.closes.Load())
			}
			if status == 401 {
				var auth *gophercloud.ErrUnableToReauthenticate
				if !errors.As(e, &auth) || !errors.Is(auth.ErrReauth, readCause) || !errors.Is(auth.ErrReauth, closeCause) {
					t.Fatal(e, auth)
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(auth.ErrOriginal, &native) || native.Actual != 401 || string(native.Body) != "partial actual" {
					t.Fatal(e, native)
				}
			} else {
				if !errors.Is(e, readCause) || !errors.Is(e, closeCause) {
					t.Fatal(e)
				}
				if status < 400 {
					objectMetadataProof(t, e, status, "partial actual")
				}
			}
		})
	}
}

func TestPythonCloudObjectGetWriterSourceCancellationIsStickyAcrossRestoringClose(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("writer custom cancellation")
	var client *gophercloud.ServiceClient
	body := &objectMetadataBody{Reader: strings.NewReader("abcdef"), onClose: func() { client.Endpoint = objectMetadataBase }}
	client = objectMetadataClient(func(*http.Request) (*http.Response, error) { return objectMetadataWire(203, body), nil })
	writer := &cloudGetWriter{onWrite: func() { client.Endpoint = objectMetadataBase + "changed/"; cancel(cause) }}
	result, e := objects.New(client).GetCloudObjectText(ctx, "box", "file", objects.WithCloudGetWriter(writer), objects.WithCloudGetChunkSize(2))
	if !errors.Is(e, resource.ErrInvalidOption) || !errors.Is(e, context.Canceled) || !errors.Is(e, cause) || result.BytesWritten != 2 || writer.String() != "ab" || writer.flushes != 0 || writer.closes != 0 || body.closes.Load() != 1 {
		t.Fatal(result, e, writer.String(), writer.flushes, writer.closes, body.closes.Load())
	}
	objectMetadataProof(t, e, 203, "ab")
}

func TestPythonCloudObjectGetOptionsSnapshotsPreflightAndDestinationReset(t *testing.T) {
	calls := 0
	client := objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return objectMetadataResponse(200, "ok"), nil })
	api := objects.New(client)
	path := filepath.Join(t.TempDir(), "output")
	size := 2
	option := objects.WithCloudGetOpts(objects.CloudGetOpts{ChunkSize: &size, Outfile: &path})
	path = ""
	size = 0
	result, e := api.GetCloudObjectText(context.Background(), "box", "file", option)
	if e != nil || result.BytesWritten != 2 {
		t.Fatal(result, e)
	}
	result, e = api.GetCloudObjectText(context.Background(), "box", "file", objects.WithCloudGetOutfile(""), objects.WithoutCloudGetOutfile())
	if e != nil || result.Body == nil || *result.Body != "ok" {
		t.Fatal(result, e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var nilWriter *cloudGetWriter
	for _, tc := range []struct {
		api               *objects.API
		ctx               context.Context
		container, object string
		options           []objects.CloudGetOption
	}{
		{nil, context.Background(), "box", "file", nil}, {api, nil, "box", "file", nil}, {api, ctx, "box", "file", nil}, {api, context.Background(), "", "file", nil}, {api, context.Background(), "box", "", nil}, {api, context.Background(), "box", "file", []objects.CloudGetOption{nil}}, {api, context.Background(), "box", "file", []objects.CloudGetOption{objects.WithCloudGetWriter(nilWriter)}}, {api, context.Background(), "box", "file", []objects.CloudGetOption{objects.WithCloudGetOpts(objects.CloudGetOpts{Writer: &cloudGetWriter{}, OmitContent: true})}}, {api, context.Background(), "box", "file", []objects.CloudGetOption{objects.WithCloudGetWriter(&cloudGetWriter{}), objects.WithCloudGetChunkSize(0)}},
	} {
		result, e := tc.api.GetCloudObjectText(tc.ctx, tc.container, tc.object, tc.options...)
		if result != nil || e == nil {
			t.Fatal(result, e)
		}
	}
	if calls != 2 {
		t.Fatal(calls)
	}
}

func TestPythonCloudObjectGetSharedCodecsAndCompressedMissingFaults(t *testing.T) {
	input := []byte("decoded text")
	for _, format := range []string{"gzip", "x-gzip", "deflate", "raw-deflate", "zstd"} {
		t.Run(format, func(t *testing.T) {
			encoding := format
			if format == "raw-deflate" {
				encoding = "deflate"
			}
			payload := cloudStreamEncode(t, format, input)
			body := &objectMetadataBody{Reader: bytes.NewReader(payload)}
			wire := objectMetadataWire(200, body)
			wire.Header.Set("Content-Encoding", encoding)
			result, e := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })).GetCloudObjectText(context.Background(), "box", "file")
			if e != nil || result.Body == nil || *result.Body != string(input) || result.Headers["content-encoding"] != encoding || body.closes.Load() != 1 {
				t.Fatal(result, e, body.closes.Load())
			}
		})
	}
	for _, valid := range []bool{false, true} {
		t.Run(fmt.Sprintf("gzip404/%v", valid), func(t *testing.T) {
			payload := []byte("broken")
			if valid {
				payload = cloudStreamEncode(t, "gzip", []byte("not found"))
			}
			body := &objectMetadataBody{Reader: bytes.NewReader(payload)}
			wire := objectMetadataWire(404, body)
			wire.Header.Set("Content-Encoding", "gzip")
			result, e := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })).GetCloudObjectText(context.Background(), "box", "file")
			if body.closes.Load() != 1 || result.Receipt.StatusCode != 404 {
				t.Fatal(result, e, body.closes.Load())
			}
			if valid {
				if e != nil || !result.Missing {
					t.Fatal(result, e)
				}
			} else {
				if e == nil || result.Missing {
					t.Fatal(result, e)
				}
				objectMetadataProof(t, e, 404, string(payload))
			}
		})
	}
}

func TestPythonCloudObjectGetAcceptedReadFaultNeverCallsDestination(t *testing.T) {
	cause := errors.New("bad accepted source read")
	body := &objectMetadataBody{Reader: objectMetadataReader(func(dst []byte) (int, error) { return copy(dst, "partial"), cause })}
	writer := &cloudGetWriter{}
	result, e := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return objectMetadataWire(200, body), nil })).GetCloudObjectText(context.Background(), "box", "file", objects.WithCloudGetWriter(writer))
	if !errors.Is(e, cause) || result.BytesWritten != 0 || writer.writes != 0 || writer.flushes != 0 || writer.closes != 0 || body.closes.Load() != 1 {
		t.Fatal(result, e, writer.writes, writer.flushes, writer.closes, body.closes.Load())
	}
	objectMetadataProof(t, e, 200, "partial")
}

func TestPythonCloudObjectGetOptionMutationGuardsSourceBeforeDestinationOrHTTP(t *testing.T) {
	calls, callbacks := 0, 0
	client := objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return objectMetadataResponse(200, "ok"), nil })
	writer := &cloudGetWriter{}
	result, e := objects.New(client).GetCloudObjectText(context.Background(), "box", "file", objects.WithCloudGetWriter(writer), func(*objects.CloudGetOpts) error {
		callbacks++
		client.Endpoint = objectMetadataBase + "changed/"
		return nil
	}, func(*objects.CloudGetOpts) error { callbacks++; client.Endpoint = objectMetadataBase; return nil })
	if result != nil || !errors.Is(e, resource.ErrInvalidOption) || callbacks != 1 || calls != 0 || writer.writes != 0 || writer.flushes != 0 || writer.closes != 0 {
		t.Fatal(result, e, callbacks, calls, writer.writes)
	}
}

func TestPythonCloudObjectGetDecoderConstructorsFollowPathOpenAndSkipFalseyOutfile(t *testing.T) {
	encoding := "gzip,gzip,gzip,gzip,gzip,gzip"
	calls, reads := 0, 0
	makeAPI := func() (*objects.API, *objectMetadataBody) {
		body := &objectMetadataBody{Reader: objectMetadataReader(func(dst []byte) (int, error) { reads++; return copy(dst, "unread"), io.EOF })}
		wire := objectMetadataWire(200, body)
		wire.Header.Set("Content-Encoding", encoding)
		return objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return wire, nil })), body
	}
	api, body := makeAPI()
	result, e := api.GetCloudObjectText(context.Background(), "box", "file", objects.WithCloudGetOutfile(""))
	if e != nil || result.Body != nil || result.Missing || reads != 0 || body.closes.Load() != 1 {
		t.Fatal(result, e, reads, body.closes.Load())
	}
	api, body = makeAPI()
	path := filepath.Join(t.TempDir(), "missingdir", "output")
	result, e = api.GetCloudObjectText(context.Background(), "box", "file", objects.WithCloudGetOutfile(path))
	var pathError *os.PathError
	if !errors.As(e, &pathError) || errors.Is(e, resource.ErrInvalidOption) || reads != 0 || body.closes.Load() != 1 || result.Receipt.StatusCode != 200 {
		t.Fatal(result, e, reads, body.closes.Load())
	}
	api, body = makeAPI()
	path = filepath.Join(t.TempDir(), "existing-output")
	if e := os.WriteFile(path, []byte("old"), 0o600); e != nil {
		t.Fatal(e)
	}
	result, e = api.GetCloudObjectText(context.Background(), "box", "file", objects.WithCloudGetOutfile(path))
	stored, _ := os.ReadFile(path)
	if !errors.Is(e, resource.ErrInvalidOption) || len(stored) != 0 || reads != 0 || body.closes.Load() != 1 || result.BytesWritten != 0 || calls != 3 {
		t.Fatal(result, e, stored, reads, body.closes.Load(), calls)
	}
}

func TestPythonCloudObjectGetDetectorGapIncludesUTF8NULMalformedBOMAndKnownPythonCodecs(t *testing.T) {
	for _, tc := range []struct {
		name, contentType string
		payload           []byte
	}{
		{"UTF8-valid NUL bigendian", "", []byte{0, 'A', 0, 'B'}},
		{"UTF8-valid NUL littleendian", "", []byte{'A', 0, 'B', 0}},
		{"malformed UTF16BOM", "", []byte{0xff, 0xfe, 'A'}},
		{"validPython unicode_escape", "x; charset=unicode_escape", []byte(`\u0041`)},
		{"validPython palmos", "x; charset=palmos", []byte{0xff}},
		{"validPython cp1006", "x; charset=cp1006", []byte{0xff}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &objectMetadataBody{Reader: bytes.NewReader(tc.payload)}
			wire := objectMetadataWire(200, body)
			wire.Header.Set("Content-Type", tc.contentType)
			result, e := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })).GetCloudObjectText(context.Background(), "box", "file")
			if !errors.Is(e, resource.ErrUnsupported) || result.Body != nil || result.Receipt.StatusCode != 200 || body.closes.Load() != 1 {
				t.Fatal(result, e, body.closes.Load())
			}
			objectMetadataProof(t, e, 200, string(tc.payload))
		})
	}
}
