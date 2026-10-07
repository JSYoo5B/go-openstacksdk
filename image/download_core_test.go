package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type downloadCoreTransport func(*http.Request) (*http.Response, error)

func (value downloadCoreTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return value(request)
}

type downloadCoreBody struct {
	reader   io.Reader
	closes   int
	closeErr error
}

func (value *downloadCoreBody) Read(buffer []byte) (int, error) { return value.reader.Read(buffer) }
func (value *downloadCoreBody) Close() error                    { value.closes++; return value.closeErr }

func downloadCoreClient(transport downloadCoreTransport) *gophercloud.ServiceClient {
	provider := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: transport}}
	provider.UseTokenLock()
	provider.SetToken("before")
	return &gophercloud.ServiceClient{ProviderClient: provider, Type: "image", Endpoint: "https://example.test/catalog/", ResourceBase: "https://example.test/reverse/glance/v2/"}
}

func downloadCoreHTTP(status int, body io.ReadCloser, headers http.Header) *http.Response {
	return &http.Response{StatusCode: status, Body: body, Header: headers}
}

type downloadCoreWriter struct {
	bytes.Buffer
	writes            []int
	closed, seeks     int
	readerFromInvoked bool
}

func (value *downloadCoreWriter) Write(buffer []byte) (int, error) {
	value.writes = append(value.writes, len(buffer))
	return value.Buffer.Write(buffer)
}
func (value *downloadCoreWriter) Close() error { value.closed++; return nil }
func (value *downloadCoreWriter) Seek(int64, int) (int64, error) {
	value.seeks++
	return 0, nil
}
func (value *downloadCoreWriter) ReadFrom(io.Reader) (int64, error) {
	value.readerFromInvoked = true
	return 0, errors.New("copy fast path bypassed owned buffer")
}

func TestDownloadCoreFixedRouteBoundedCopyAndOwnedEvidence(t *testing.T) {
	const metadata = `{"id":"fixed","ID":"typed-decoy","file":"https://foreign.test/evil","os_hash_algo":"sha256","os_hash_value":"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad","checksum":"disagreeing-md5","status":"saving"}`
	metaBody := &downloadCoreBody{reader: strings.NewReader(metadata)}
	fileBody := &downloadCoreBody{reader: strings.NewReader("abc")}
	var client *gophercloud.ServiceClient
	var retained *DownloadImageOpts
	var sequence []string
	var callbacks int
	client = downloadCoreClient(func(r *http.Request) (*http.Response, error) {
		sequence = append(sequence, r.URL.Path)
		if r.Header.Get("X-Snapshot") != "before" || callbacks != 1 {
			t.Errorf("headers=%v callbacks=%d", r.Header, callbacks)
		}
		if strings.HasSuffix(r.URL.Path, "/file") {
			if r.URL.Query().Get("prefer") != "a &/?=,a &/?=,slow" || len(r.URL.Query()) != 1 || r.Header.Get("X-Auth-Token") != "after" {
				t.Errorf("file query=%v headers=%v", r.URL.Query(), r.Header)
			}
			return downloadCoreHTTP(200, fileBody, http.Header{"X-File": {"actual"}, "Content-Md5": {"disagreeing-header"}}), nil
		}
		*retained.ChunkSize, retained.StorePreferences[0] = 999, "changed"
		client.MoreHeaders["X-Snapshot"] = "changed"
		client.ResourceBase = "https://example.test/later-valid/"
		client.ProviderClient.SetToken("after")
		return downloadCoreHTTP(200, metaBody, http.Header{"X-Metadata": {"actual"}, "Openstack-Image-Import-Methods": {"glance-direct,web-download"}, "Location": {"https://foreign.test/evil"}}), nil
	})
	client.MoreHeaders = map[string]string{"X-Snapshot": "before"}
	writer := &downloadCoreWriter{}
	_, _ = writer.Buffer.WriteString("existing-")
	result, err := New(client).DownloadTo(context.Background(), resource.ID("fixed"), writer, func(value *DownloadImageOpts) error {
		callbacks++
		retained = value
		size := 2
		value.ChunkSize, value.StorePreferences = &size, []string{"a &/?=", "a &/?=", "slow"}
		return nil
	})
	if err != nil || result == nil || result.ImageID != "fixed" || result.Image == nil || result.Metadata == nil || string(result.Metadata.Body) != metadata || result.Metadata.StatusCode != 200 || result.Metadata.Header.Get("X-Metadata") != "actual" || result.Header.Get("X-File") != "actual" || result.StatusCode != 200 || result.BytesWritten != 3 || writer.String() != "existing-abc" || !reflect.DeepEqual(writer.writes, []int{2, 1}) || writer.closed != 0 || writer.seeks != 0 || writer.readerFromInvoked || metaBody.closes != 1 || fileBody.closes != 1 || result.Checksum == nil || !result.Checksum.Complete || !result.Checksum.Verified || result.Checksum.Algorithm != "sha256" {
		t.Fatalf("result=%+v err=%v writer=%+v closes=%d/%d", result, err, writer, metaBody.closes, fileBody.closes)
	}
	if !reflect.DeepEqual(sequence, []string{"/reverse/glance/v2/images/fixed", "/reverse/glance/v2/images/fixed/file"}) || !reflect.DeepEqual(result.Image.OpenStackImageImportMethods, []string{"glance-direct", "web-download"}) {
		t.Fatalf("sequence=%v image=%+v", sequence, result.Image)
	}
	result.Metadata.Header.Set("X-File", "changed")
	result.Metadata.Body[0] = '!'
	if result.Header.Get("X-File") != "actual" {
		t.Fatal("metadata aliases binary evidence")
	}
}

type downloadCoreReadFailure struct {
	data  string
	cause error
}

func (value *downloadCoreReadFailure) Read(buffer []byte) (int, error) {
	n := copy(buffer, value.data)
	value.data = value.data[n:]
	return n, value.cause
}

func TestDownloadCoreRetainsAcceptedMetadataAndStopsInvalidIdentity(t *testing.T) {
	readCause, closeCause := errors.New("metadata read failed"), errors.New("metadata close failed")
	for _, test := range []struct {
		name, body string
		reader     io.Reader
		closeErr   error
		cause      error
	}{
		{"malformed", "{bad", nil, nil, nil},
		{"missing lowercase ID", `{"ID":"fixed"}`, nil, nil, nil},
		{"wrong canonical ID", `{"id":"decoy"}`, nil, nil, nil},
		{"native type", `{"id":"fixed","min_ram":"bad"}`, nil, nil, nil},
		{"native timestamp", `{"id":"fixed","created_at":"bad"}`, nil, nil, nil},
		{"invalid UTF8", string([]byte{'{', '"', 'i', 'd', '"', ':', '"', 255, '"', '}'}), nil, nil, nil},
		{"accepted read", "prefix", &downloadCoreReadFailure{data: "prefix", cause: readCause}, nil, readCause},
		{"accepted close", `{"id":"fixed"}`, nil, closeCause, closeCause},
	} {
		t.Run(test.name, func(t *testing.T) {
			reader := test.reader
			if reader == nil {
				reader = strings.NewReader(test.body)
			}
			body := &downloadCoreBody{reader: reader, closeErr: test.closeErr}
			var requests int
			client := downloadCoreClient(func(r *http.Request) (*http.Response, error) {
				requests++
				return downloadCoreHTTP(200, body, http.Header{"X-Metadata": {"actual"}}), nil
			})
			writer := &downloadCoreWriter{}
			result, err := New(client).DownloadTo(context.Background(), resource.ID("fixed"), writer)
			var responseErr *resource.ResponseError
			if err == nil || !errors.As(err, &responseErr) || result == nil || result.ImageID != "fixed" || result.Metadata == nil || string(result.Metadata.Body) != test.body || result.StatusCode != 0 || result.Header != nil || requests != 1 || len(writer.writes) != 0 || body.closes != 1 {
				t.Fatalf("result=%+v err=%v requests=%d writes=%v closes=%d", result, err, requests, writer.writes, body.closes)
			}
			if test.cause != nil && !errors.Is(err, test.cause) {
				t.Fatal("metadata cause lost", err)
			}
		})
	}
}

type downloadCoreWriteFailure struct {
	count int
	cause error
}

func (value downloadCoreWriteFailure) Write([]byte) (int, error) { return value.count, value.cause }

func TestDownloadCoreJoinsReadWriteCloseAndPreservesFullDigestProof(t *testing.T) {
	readCause, writeCause, closeCause := errors.New("payload read failed"), errors.New("output write failed"), errors.New("response close failed")
	for _, test := range []struct {
		name      string
		reader    io.Reader
		writer    io.Writer
		wantBytes int64
		causes    []error
		complete  bool
	}{
		{"simultaneous read write close", &downloadCoreReadFailure{data: "abc", cause: readCause}, downloadCoreWriteFailure{count: 2, cause: writeCause}, 2, []error{readCause, writeCause, closeCause}, false},
		{"short write and read", &downloadCoreReadFailure{data: "abc", cause: readCause}, downloadCoreWriteFailure{count: 2}, 2, []error{readCause, io.ErrShortWrite, closeCause}, false},
		{"complete bytes close failed", strings.NewReader("abc"), &bytes.Buffer{}, 3, []error{closeCause}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fileBody := &downloadCoreBody{reader: test.reader, closeErr: closeCause}
			var calls int
			client := downloadCoreClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if strings.HasSuffix(r.URL.Path, "/file") {
					return downloadCoreHTTP(200, fileBody, http.Header{"X-File": {"actual"}}), nil
				}
				return downloadCoreHTTP(200, io.NopCloser(strings.NewReader(`{"id":"fixed","checksum":"900150983cd24fb0d6963f7d28e17f72"}`)), http.Header{}), nil
			})
			result, err := New(client).DownloadTo(context.Background(), resource.ID("fixed"), test.writer)
			if result == nil || err == nil || result.StatusCode != 200 || result.Header.Get("X-File") != "actual" || result.BytesWritten != test.wantBytes || result.Checksum == nil || result.Checksum.Complete != test.complete || result.Checksum.Verified != test.complete || fileBody.closes != 1 || calls != 2 {
				t.Fatalf("result=%+v err=%v closes=%d calls=%d", result, err, fileBody.closes, calls)
			}
			for _, cause := range test.causes {
				if !errors.Is(err, cause) {
					t.Errorf("cause %v lost from %v", cause, err)
				}
			}
			if test.complete && result.Checksum.Actual != "900150983cd24fb0d6963f7d28e17f72" || !test.complete && result.Checksum.Actual != "" {
				t.Fatal("incorrect final digest", result.Checksum)
			}
		})
	}
}

func TestDownloadCoreNoDataRejectedPartialAndLiteralHashMismatch(t *testing.T) {
	for _, test := range []struct {
		name     string
		status   int
		headers  http.Header
		expected string
		wantErr  error
	}{
		{"no data", 204, http.Header{}, "900150983cd24fb0d6963f7d28e17f72", nil},
		{"unsolicited partial status", 206, http.Header{}, "900150983cd24fb0d6963f7d28e17f72", nil},
		{"unsolicited Content-Range", 200, http.Header{"Content-Range": {"bytes 0-2/100"}}, "900150983cd24fb0d6963f7d28e17f72", resource.ErrInvalidOption},
		{"literal uppercase mismatch", 200, http.Header{}, "900150983CD24FB0D6963F7D28E17F72", ErrChecksumMismatch},
	} {
		t.Run(test.name, func(t *testing.T) {
			fileBody := &downloadCoreBody{reader: strings.NewReader("abc")}
			client := downloadCoreClient(func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.URL.Path, "/file") {
					return downloadCoreHTTP(test.status, fileBody, test.headers), nil
				}
				return downloadCoreHTTP(200, io.NopCloser(strings.NewReader(fmt.Sprintf(`{"id":"fixed","checksum":%q}`, test.expected))), http.Header{}), nil
			})
			writer := &downloadCoreWriter{}
			result, err := New(client).DownloadTo(context.Background(), resource.ID("fixed"), writer)
			if result == nil || result.Metadata == nil || fileBody.closes != 1 {
				t.Fatalf("result=%+v err=%v closes=%d", result, err, fileBody.closes)
			}
			if test.status == 206 {
				if !gophercloud.ResponseCodeIs(err, 206) || result.StatusCode != 0 || result.Header != nil || result.Checksum == nil || result.Checksum.Complete || result.BytesWritten != 0 {
					t.Fatal(result, err)
				}
				return
			}
			if !errors.Is(err, test.wantErr) || result.StatusCode != test.status {
				t.Fatal(result, err)
			}
			if test.wantErr == ErrChecksumMismatch {
				var mismatch *DownloadChecksumMismatchError
				if !errors.As(err, &mismatch) || mismatch.Expected != test.expected || mismatch.Actual != "900150983cd24fb0d6963f7d28e17f72" || result.BytesWritten != 3 || !result.Checksum.Complete || result.Checksum.Verified {
					t.Fatal(result, err)
				}
			} else if result.BytesWritten != 0 || len(writer.writes) != 0 {
				t.Fatal(result, writer)
			}
			if test.status == 204 && result.Checksum != nil {
				t.Fatal("no-data checksum claims verification", result)
			}
		})
	}
}

func TestDownloadCoreCanonicalHashSelectionAndAlgorithms(t *testing.T) {
	for _, test := range []struct{ algorithm, digest string }{
		{"md5", "900150983cd24fb0d6963f7d28e17f72"},
		{"sha1", "a9993e364706816aba3e25717850c26c9cd0d89d"},
		{"sha224", "23097d223405d8228642a477bda255b32aadbce4bda0b3f7e36c9da7"},
		{"sha256", "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		{"sha384", "cb00753f45a35e8bb5a03d699ac65007272c32ab0eded1631a8b605a43ff5bed8086072ba1e7cc2358baeca134c825a7"},
		{"sha512", "ddaf35a193617abacc417349ae20413112e6fa4e89a97ea20a9eeee64b55d39a2192992a274fc1a836ba3c23a3feebbd454d4423643ce80e2a9ac94fa54ca49f"},
		{"sha512_224", "4634270f707b6a54daae7530460842e20e37ed265ceee9a43e8924aa"},
		{"sha512_256", "53048e2681941ef99b2e29b76b4c7dabe4c2d0c634fc6d46e0e2f13107e7af23"},
	} {
		t.Run(test.algorithm, func(t *testing.T) {
			body := fmt.Sprintf(`{"os_hash_algo":%q,"os_hash_value":%q,"checksum":"wrong fallback"}`, test.algorithm, test.digest)
			var fields map[string]json.RawMessage
			if err := json.Unmarshal([]byte(body), &fields); err != nil {
				t.Fatal(err)
			}
			digest, selected, err := prepareDownloadChecksum(&rest.Response{Body: []byte(body), StatusCode: 200}, fields)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = digest.Write([]byte("abc"))
			actual := fmt.Sprintf("%x", digest.Sum(nil))
			if selected.Algorithm != test.algorithm || selected.Expected != test.digest || actual != test.digest {
				t.Fatalf("selected=%+v actual=%s", selected, actual)
			}
		})
	}
	for _, test := range []struct {
		body      string
		want      error
		expected  string
		selection bool
	}{
		{`{"os_hash_algo":"sha256","checksum":"legacy"}`, nil, "legacy", true},
		{`{"os_hash_value":"unpaired","checksum":null}`, nil, "", false},
		{`{"OS_HASH_ALGO":"sha256","OS_HASH_VALUE":"ignored"}`, nil, "", false},
		{`{"os_hash_algo":"unsupported","os_hash_value":"digest","checksum":"fallback"}`, resource.ErrUnsupported, "", false},
		{`{"os_hash_algo":1,"os_hash_value":"digest"}`, nil, "", false},
	} {
		var fields map[string]json.RawMessage
		_ = json.Unmarshal([]byte(test.body), &fields)
		_, selected, err := prepareDownloadChecksum(&rest.Response{Body: []byte(test.body), StatusCode: 200}, fields)
		if strings.Contains(test.body, `"os_hash_algo":1`) {
			var typeErr *json.UnmarshalTypeError
			if !errors.As(err, &typeErr) {
				t.Fatal(err)
			}
		} else if !errors.Is(err, test.want) || (selected != nil) != test.selection || selected != nil && selected.Expected != test.expected {
			t.Fatalf("body=%s selected=%+v err=%v", test.body, selected, err)
		}
	}
}
