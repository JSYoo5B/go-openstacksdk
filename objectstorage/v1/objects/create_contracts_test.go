package objects_test

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func objectCreateFile(t *testing.T, data []byte) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "explicit input")
	if err := os.WriteFile(name, data, 0600); err != nil {
		t.Fatal(err)
	}
	return name
}
func objectCreateHashes(data []byte) (string, string) {
	return fmt.Sprintf("%x", md5.Sum(data)), fmt.Sprintf("%x", sha256.Sum256(data))
}
func objectCreatePayload(t *testing.T, r *http.Request) []byte {
	t.Helper()
	if r.Body == nil {
		return nil
	}
	data, err := io.ReadAll(r.Body)
	closeErr := r.Body.Close()
	if err != nil || closeErr != nil {
		t.Errorf("request body read=%v close=%v", err, closeErr)
	}
	if r.ContentLength != int64(len(data)) || len(r.TransferEncoding) != 0 {
		t.Errorf("framing length=%d transfer=%v bytes=%d", r.ContentLength, r.TransferEncoding, len(data))
	}
	return data
}
func objectCreateInfo() *http.Response {
	return objectMetadataResponse(200, `{"swift":{"max_file_size":8},"slo":{"min_segment_size":1}}`)
}

type objectCreateBorrowed struct {
	*bytes.Reader
	closes, seeks atomic.Int32
}

func (r *objectCreateBorrowed) Close() error { r.closes.Add(1); return nil }
func (r *objectCreateBorrowed) Seek(offset int64, whence int) (int64, error) {
	r.seeks.Add(1)
	return r.Reader.Seek(offset, whence)
}

func TestCreateObjectContractsBytesWireAndSnapshots(t *testing.T) {
	t.Run("real literal PUT including present empty bytes", func(t *testing.T) {
		var calls atomic.Int32
		payload := []byte{'a', 0, 0xff, 'z'}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			index := calls.Add(1)
			expected := "/reverse/a%20b/v1/AUTH_wire/" + url.PathEscape(objectMetadataContainer) + "/" + url.PathEscape(objectMetadataKey)
			body, err := io.ReadAll(r.Body)
			want := payload
			if index == 2 {
				want = []byte{}
			}
			if err != nil || r.Method != "PUT" || r.RequestURI != expected || r.Header.Get("X-Auth-Token") != "wire-token" || !bytes.Equal(body, want) || r.ContentLength != int64(len(want)) || len(r.TransferEncoding) != 0 {
				t.Errorf("wire=%s %s length=%d body=%x err=%v", r.Method, r.RequestURI, r.ContentLength, body, err)
			}
			if r.Header.Get("ETag") != "" || r.Header.Get("X-Object-Meta-X-Sdk-Md5") != "literal-metadata" {
				t.Errorf("headers=%v", r.Header)
			}
			w.Header().Set("X-Trans-Id", "real-create")
			w.WriteHeader(201)
			_, _ = w.Write([]byte("wire-ack"))
		}))
		defer server.Close()
		provider := &gophercloud.ProviderClient{HTTPClient: *server.Client()}
		provider.UseTokenLock()
		provider.SetToken("wire-token")
		client := &gophercloud.ServiceClient{ProviderClient: provider, Type: "object-store", Endpoint: server.URL + "/catalogue/v1/AUTH_a/", ResourceBase: server.URL + "/reverse/a%20b/v1/AUTH_wire/"}
		for _, data := range [][]byte{payload, {}} {
			result, err := objects.New(client).CreateObject(context.Background(), objectMetadataContainer, objectMetadataKey, objects.CreateObjectInput{Data: data}, objects.WithCreateObjectMD5(strings.Repeat("a", 32)), objects.WithCreateObjectSegmentSize(0), objects.WithCreateObjectUseSLO(false), objects.WithCreateObjectMetadataValue("X-Sdk-Md5", "literal-metadata"))
			if err != nil || result == nil || result.Source != "bytes" || result.Mode != "ordinary" || result.Size != int64(len(data)) || result.Capabilities != nil || result.Discovery != nil || len(result.Segments) != 0 || result.MD5 != "" || result.Ordinary == nil || result.Ordinary.Acknowledgement.StatusCode != 201 || string(result.Ordinary.Acknowledgement.Body) != "wire-ack" {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		}
		if calls.Load() != 2 {
			t.Fatalf("physical calls=%d", calls.Load())
		}
	})
	t.Run("data and option maps captured before callbacks", func(t *testing.T) {
		data := []byte("frozen")
		headers := map[string]string{"X-Upload": "captured"}
		metadata := map[string]string{"Book": "captured"}
		factory := objects.WithCreateObjectOpts(objects.CreateObjectOpts{Headers: headers, Metadata: metadata})
		headers["X-Upload"] = "mutated"
		metadata["Book"] = "mutated"
		calls, callbacks := 0, 0
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if string(objectCreatePayload(t, r)) != "frozen" || r.Header.Get("X-Upload") != "captured" || r.Header.Get("X-Source") != "original" || r.Header.Get("X-Object-Meta-Book") != "captured" {
				t.Errorf("snapshot headers=%v", r.Header)
			}
			return objectMetadataResponse(202, "raw-ack"), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "original"}
		result, err := objects.New(client).CreateObject(context.Background(), "c", "o", objects.CreateObjectInput{Data: data}, factory, func(o *objects.CreateObjectOpts) error {
			callbacks++
			data[0] = 'X'
			client.MoreHeaders["X-Source"] = "valid-drift"
			return nil
		})
		if err != nil || callbacks != 1 || calls != 1 || result == nil || result.Ordinary.Acknowledgement.StatusCode != 202 {
			t.Fatalf("result=%+v err=%v calls=%d callbacks=%d", result, err, calls, callbacks)
		}
		attempt := result.Ordinary.Attempts[0].Response
		result.Ordinary.Acknowledgement.Body[0] = 'X'
		result.Ordinary.Acknowledgement.Header.Set("X-Trans-Id", "changed")
		if string(attempt.Body) != "raw-ack" || attempt.Header.Get("X-Trans-Id") != "actual-object" {
			t.Fatal("acknowledgement aliases physical proof")
		}
	})
}

func TestCreateObjectContractsFileReaderAndHashes(t *testing.T) {
	t.Run("borrow current cursor once without Close or Seek", func(t *testing.T) {
		data := []byte("owned")
		reader := &objectCreateBorrowed{Reader: bytes.NewReader([]byte("prefixowned"))}
		_, _ = reader.Reader.Seek(6, io.SeekStart)
		md, sha := objectCreateHashes(data)
		var methods []string
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			methods = append(methods, r.Method)
			if r.Method != "PUT" && (r.Header.Get("X-Upload") != "" || r.Header.Get("X-Object-Meta-Book") != "") {
				t.Errorf("upload headers leaked=%v", r.Header)
			}
			switch r.Method {
			case "GET":
				if r.URL.String() != "https://swift.invalid/reverse/a%20b/info" || r.Body != nil {
					t.Errorf("capability target=%s body=%v", r.URL, r.Body)
				}
				return objectCreateInfo(), nil
			case "HEAD":
				return objectMetadataResponse(404, "missing"), nil
			case "PUT":
				if !bytes.Equal(objectCreatePayload(t, r), data) || r.Header.Get("X-Object-Meta-X-Sdk-Md5") != md || r.Header.Get("X-Object-Meta-X-Sdk-Sha256") != sha || r.Header.Get("ETag") != "" {
					t.Errorf("upload headers=%v", r.Header)
				}
				return objectMetadataResponse(201, "created"), nil
			default:
				t.Errorf("unexpected method=%s", r.Method)
				return objectMetadataResponse(500, "unexpected"), nil
			}
		})
		result, err := objects.New(client).CreateObject(context.Background(), "c", "o", objects.CreateObjectInput{Reader: reader}, objects.WithCreateObjectHeader("X-Upload", "only-write"), objects.WithCreateObjectMetadataValue("Book", "literal"), objects.WithCreateObjectMD5(strings.Repeat("0", 32)))
		if err != nil || result == nil || result.Source != "reader" || result.Mode != "ordinary" || result.MD5 != md || result.SHA256 != sha || !reflect.DeepEqual(methods, []string{"GET", "HEAD", "PUT"}) || reader.closes.Load() != 0 || reader.seeks.Load() != 0 {
			t.Fatalf("result=%+v err=%v methods=%v Close=%d Seek=%d", result, err, methods, reader.closes.Load(), reader.seeks.Load())
		}
	})
	t.Run("file snapshot and fallback hashes do not become upload metadata", func(t *testing.T) {
		data := []byte("stable")
		filename := objectCreateFile(t, data)
		md, sha := objectCreateHashes(data)
		var methods []string
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			methods = append(methods, r.Method)
			switch r.Method {
			case "GET":
				if err := os.WriteFile(filename, []byte("changed after preparation"), 0600); err != nil {
					t.Errorf("rewrite=%v", err)
				}
				return objectCreateInfo(), nil
			case "HEAD":
				return objectMetadataResponse(204, ""), nil
			case "PUT":
				if !bytes.Equal(objectCreatePayload(t, r), data) || r.Header.Get("X-Object-Meta-X-Sdk-Md5") != "caller-md5" || r.Header.Get("X-Object-Meta-X-Sdk-Sha256") != "" {
					t.Errorf("stable bytes/hash metadata=%v", r.Header)
				}
				return objectMetadataResponse(201, ""), nil
			default:
				t.Errorf("unexpected=%s", r.Method)
				return objectMetadataResponse(500, ""), nil
			}
		})
		result, err := objects.New(client).CreateObject(context.Background(), "c", "o", objects.CreateObjectInput{Filename: filename}, objects.WithCreateObjectGenerateChecksums(false), objects.WithCreateObjectMetadataValue("x-sdk-md5", "caller-md5"))
		if err != nil || result == nil || result.Source != "file" || result.MD5 != md || result.SHA256 != sha || !reflect.DeepEqual(methods, []string{"GET", "HEAD", "PUT"}) {
			t.Fatalf("result=%+v err=%v methods=%v", result, err, methods)
		}
	})
	t.Run("matching hash skips writes despite metadata changes", func(t *testing.T) {
		data := []byte("same")
		md, sha := objectCreateHashes(data)
		calls := 0
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method == "GET" {
				return objectCreateInfo(), nil
			}
			if r.Method != "HEAD" {
				t.Errorf("write after match=%s", r.Method)
			}
			response := objectMetadataResponse(200, "head-proof")
			response.Header.Set("X-Object-Meta-X-Sdk-Md5", md)
			response.Header.Set("X-Object-Meta-X-Sdk-Sha256", sha)
			return response, nil
		})
		result, err := objects.New(client).CreateObject(context.Background(), "c", "o", objects.CreateObjectInput{Filename: objectCreateFile(t, data)}, objects.WithCreateObjectMetadataValue("changed", "yes"))
		if err != nil || result == nil || !result.Skipped || result.Ordinary != nil || result.Manifest != nil || result.Mode != "" || calls != 2 || result.Discovery == nil || string(result.Discovery.Body) != "head-proof" {
			t.Fatalf("result=%+v err=%v HTTP=%d", result, err, calls)
		}
	})
	t.Run("capability fallback and malformed accepted bounds", func(t *testing.T) {
		for _, status := range []int{404, 412, 200} {
			calls := 0
			body := "unavailable"
			if status == 200 {
				body = `{"swift":{"max_file_size":1.5}}`
			}
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method == "GET" {
					return objectMetadataResponse(status, body), nil
				}
				if r.Method == "HEAD" {
					return objectMetadataResponse(404, "missing"), nil
				}
				if r.Method != "PUT" || string(objectCreatePayload(t, r)) != "x" {
					t.Errorf("fallback upload=%s", r.Method)
				}
				return objectMetadataResponse(201, "created"), nil
			})
			result, err := objects.New(client).CreateObject(context.Background(), "c", "o", objects.CreateObjectInput{Reader: strings.NewReader("x")})
			if result == nil || result.Capabilities == nil || result.Capabilities.Response == nil || result.Capabilities.Response.StatusCode != status || string(result.Capabilities.Response.Body) != body {
				t.Fatalf("capability=%+v err=%v", result, err)
			}
			if status == 200 {
				if err == nil || result.Capabilities.Size != 0 || result.Capabilities.MaxFileSize != 0 || calls != 1 || result.Discovery != nil || result.Ordinary != nil {
					t.Fatalf("malformed result=%+v err=%v calls=%d", result, err, calls)
				}
				objectMetadataProof(t, err, 200, body)
			} else if err != nil || !result.Capabilities.UsedFallback || result.Capabilities.MaxFileSize != 2684354561 || result.Capabilities.Size != 1073741824 || calls != 3 {
				t.Fatalf("fallback result=%+v err=%v calls=%d", result, err, calls)
			}
		}
	})
}

func TestCreateObjectContractsSLOAndDLOWire(t *testing.T) {
	for _, useSLO := range []bool{true, false} {
		t.Run(fmt.Sprintf("SLO=%v", useSLO), func(t *testing.T) {
			data := []byte("abcde")
			container, key := objectMetadataContainer+" &+", objectMetadataKey+" &+"
			var mu sync.Mutex
			segments := map[string][]byte{}
			var manifest []byte
			var finalHeader http.Header
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				if r.Method == "GET" {
					return objectCreateInfo(), nil
				}
				if r.Method == "HEAD" {
					return objectMetadataResponse(404, "missing"), nil
				}
				body := objectCreatePayload(t, r)
				if r.Method != "PUT" || r.Header.Get("ETag") != "" || r.Header.Get("Content-Type") != "logical/type" {
					t.Errorf("PUT=%s headers=%v", r.URL, r.Header)
				}
				if r.URL.Path == "/reverse/a b/v1/AUTH_account/"+container+"/"+key {
					mu.Lock()
					manifest = append([]byte{}, body...)
					finalHeader = r.Header.Clone()
					mu.Unlock()
					if useSLO && r.URL.RawQuery != "multipart-manifest=put" || !useSLO && r.URL.RawQuery != "" {
						t.Errorf("manifest query=%s", r.URL.RawQuery)
					}
					return objectMetadataResponse(202, "manifest-ack"), nil
				}
				name := strings.TrimPrefix(r.URL.Path, "/reverse/a b/v1/AUTH_account/"+container+"/")
				if r.URL.RawQuery != "" || r.Header.Get("If-None-Match") != "*" {
					t.Errorf("segment query/condition=%s %v", r.URL, r.Header)
				}
				mu.Lock()
				segments[name] = append([]byte{}, body...)
				mu.Unlock()
				response := objectMetadataResponse(201, "segment-ack")
				response.Header.Set("ETag", `"opaque-etag"`)
				return response, nil
			})
			result, err := objects.New(client).CreateObject(context.Background(), container, key, objects.CreateObjectInput{Reader: bytes.NewReader(data)}, objects.WithCreateObjectSegmentSize(2), objects.WithCreateObjectUseSLO(useSLO), objects.WithCreateObjectHeader("Content-Type", "logical/type"))
			mode := "slo"
			if !useSLO {
				mode = "dlo"
			}
			if err != nil || result == nil || result.Mode != mode || len(result.Segments) != 3 || result.Manifest == nil || result.Manifest.Acknowledgement.StatusCode != 202 || !result.ManifestAmbiguous || len(result.Cleanup) != 0 {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if !regexp.MustCompile(`^` + regexp.QuoteMeta(key) + `/\.gophercloudsdk-upload-[0-9a-f]{32}/$`).MatchString(result.SegmentPrefix) {
				t.Fatalf("prefix=%q", result.SegmentPrefix)
			}
			for i, segment := range result.Segments {
				wantName := result.SegmentPrefix + fmt.Sprintf("%06d", i)
				start, end := i*2, i*2+2
				if end > len(data) {
					end = len(data)
				}
				if segment.Name != wantName || segment.Index != int64(i) || segment.Offset != int64(start) || segment.Size != int64(end-start) || !segment.Created || segment.Ambiguous || segment.ETag == nil || *segment.ETag != "opaque-etag" || !bytes.Equal(segments[wantName], data[start:end]) {
					t.Fatalf("segment=%+v bytes=%q", segment, segments[wantName])
				}
			}
			if useSLO {
				var entries []struct {
					Path string `json:"path"`
					Size int64  `json:"size_bytes"`
					ETag string `json:"etag"`
				}
				if err := json.Unmarshal(manifest, &entries); err != nil || len(entries) != 3 || finalHeader.Get("Accept") != "application/json" || finalHeader.Get("X-Object-Manifest") != "" {
					t.Fatalf("manifest=%s headers=%v err=%v", manifest, finalHeader, err)
				}
				for i, entry := range entries {
					if entry.Path != "/"+container+"/"+result.Segments[i].Name || entry.Size != result.Segments[i].Size || entry.ETag != "opaque-etag" {
						t.Fatalf("entry=%+v", entry)
					}
				}
			} else {
				decoded, err := url.PathUnescape(finalHeader.Get("X-Object-Manifest"))
				if err != nil || decoded != container+"/"+result.SegmentPrefix || len(manifest) != 0 || strings.ContainsAny(finalHeader.Get("X-Object-Manifest"), "?&+") {
					t.Fatalf("DLO header=%q decoded=%q bytes=%s err=%v", finalHeader.Get("X-Object-Manifest"), decoded, manifest, err)
				}
			}
		})
	}
	t.Run("SLO redirect cannot add ETag or replace forced JSON Accept aliases", func(t *testing.T) {
		manifestCalls, redirects := 0, 0
		var manifestBytes []byte
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			if r.Method == "GET" {
				return objectCreateInfo(), nil
			}
			if r.Method == "HEAD" {
				return objectMetadataResponse(404, "missing"), nil
			}
			payload := objectCreatePayload(t, r)
			if r.URL.RawQuery == "multipart-manifest=put" {
				manifestCalls++
				if manifestCalls == 1 {
					manifestBytes = append([]byte{}, payload...)
				} else if !bytes.Equal(payload, manifestBytes) {
					t.Errorf("manifest replay changed=%s first=%s", payload, manifestBytes)
				}
				accepts := 0
				for key, values := range r.Header {
					if strings.EqualFold(key, "ETag") {
						t.Errorf("request ETag survived=%v", r.Header)
					}
					if strings.EqualFold(key, "Accept") {
						accepts++
						if len(values) != 1 || values[0] != "application/json" {
							t.Errorf("Accept=%v", r.Header)
						}
					}
				}
				if accepts != 1 || r.Header.Get("Content-Type") != "logical/type" {
					t.Errorf("manifest media=%v", r.Header)
				}
				if manifestCalls == 1 {
					response := objectMetadataResponse(307, "redirect proof")
					response.Header.Set("Location", r.URL.String())
					return response, nil
				}
				return objectMetadataResponse(201, "manifest proof"), nil
			}
			return objectMetadataResponse(201, "segment proof"), nil
		})
		client.HTTPClient.CheckRedirect = func(next *http.Request, _ []*http.Request) error {
			redirects++
			next.Header["eTAG"] = []string{"foreign"}
			next.Header["aCcEpT"] = []string{"foreign"}
			return nil
		}
		result, err := objects.New(client).CreateObject(context.Background(), "c", "o", objects.CreateObjectInput{Reader: strings.NewReader("abcd")}, objects.WithCreateObjectSegmentSize(2), objects.WithCreateObjectHeader("Content-Type", "logical/type"))
		if err != nil || result == nil || result.Manifest == nil || manifestCalls != 2 || redirects != 1 || len(result.Manifest.Attempts) != 2 || result.Manifest.Acknowledgement.StatusCode != 201 {
			t.Fatalf("result=%+v err=%v manifest=%d redirect=%d", result, err, manifestCalls, redirects)
		}
		if result.Manifest.Attempts[0].Response.StatusCode != 307 || string(result.Manifest.Attempts[0].Response.Body) != "redirect proof" || result.Manifest.Attempts[1].LogicalAttempt != 1 || result.Manifest.Attempts[1].PhysicalAttempt != 2 {
			t.Fatalf("manifest physical proof=%+v", result.Manifest)
		}
	})
}

func TestCreateObjectContractsNativeReplayAndPhysicalEvidence(t *testing.T) {
	for _, mode := range []string{"retry", "backoff", "reauth", "redirect", "transport"} {
		t.Run(mode, func(t *testing.T) {
			data := []byte{'a', 0, 'b', 0xff}
			calls, hooks := 0, 0
			transportCause := errors.New("first native transport fault")
			var oldBody io.ReadCloser
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 2 && oldBody != nil {
					_ = oldBody.Close()
				}
				if r.Method != "PUT" || r.URL.String() != objectMetadataEndpoint || !bytes.Equal(objectCreatePayload(t, r), data) {
					t.Errorf("replay request=%s %s", r.Method, r.URL)
				}
				oldBody = r.Body
				if mode == "reauth" && calls == 2 && r.Header.Get("X-Auth-Token") != "refreshed" {
					t.Errorf("auth=%v", r.Header)
				}
				if calls == 1 {
					if mode == "transport" {
						return nil, transportCause
					}
					status := map[string]int{"retry": 503, "backoff": 429, "reauth": 401, "redirect": 307}[mode]
					response := objectMetadataResponse(status, "first-proof")
					if mode == "redirect" {
						response.Header.Set("Location", objectMetadataEndpoint)
					}
					return response, nil
				}
				return objectMetadataResponse(201, "accepted-proof"), nil
			})
			switch mode {
			case "retry", "transport":
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks++
					return nil
				}
			case "backoff":
				client.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { hooks++; return nil }
			case "reauth":
				client.ReauthFunc = func(context.Context) error { hooks++; client.ProviderClient.SetToken("refreshed"); return nil }
			case "redirect":
				client.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { hooks++; return nil }
			}
			result, err := objects.New(client).CreateObject(context.Background(), objectMetadataContainer, objectMetadataKey, objects.CreateObjectInput{Data: data})
			if err != nil || result == nil || result.Ordinary == nil || calls != 2 || hooks != 1 || len(result.Ordinary.Attempts) != 2 || result.Ordinary.Acknowledgement.StatusCode != 201 {
				t.Fatalf("result=%+v err=%v HTTP=%d hooks=%d", result, err, calls, hooks)
			}
			for i, attempt := range result.Ordinary.Attempts {
				if attempt.LogicalAttempt != 1 || attempt.PhysicalAttempt != i+1 || attempt.Response == nil && !(mode == "transport" && i == 0) {
					t.Fatalf("attempt=%+v", attempt)
				}
			}
			first := result.Ordinary.Attempts[0]
			if mode == "transport" {
				if first.Response != nil || !errors.Is(first.Error, transportCause) {
					t.Fatalf("transport attempt=%+v", first)
				}
			} else if string(first.Response.Body) != "first-proof" {
				t.Fatalf("first attempt=%+v", first)
			}
			if string(result.Ordinary.Attempts[1].Response.Body) != "accepted-proof" {
				t.Fatalf("attempts=%+v", result.Ordinary.Attempts)
			}
		})
	}
	t.Run("native body replacement is terminal and retains rejection", func(t *testing.T) {
		calls := 0
		callbackCause := errors.New("native policy refused")
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			objectCreatePayload(t, r)
			return objectMetadataResponse(503, "rejected"), nil
		})
		client.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
			opts.RawBody = strings.NewReader("foreign")
			return callbackCause
		}
		result, err := objects.New(client).CreateObject(context.Background(), "c", "o", objects.CreateObjectInput{Data: []byte("owned")})
		if result == nil || result.Ordinary == nil || calls != 1 || !errors.Is(err, callbackCause) || !errors.Is(err, resource.ErrInvalidOption) || len(result.Ordinary.Attempts) != 1 || string(result.Ordinary.Attempts[0].Response.Body) != "rejected" {
			t.Fatalf("result=%+v err=%v HTTP=%d", result, err, calls)
		}
	})
	t.Run("redirect framing mutation is terminal before another physical PUT", func(t *testing.T) {
		calls, redirects := 0, 0
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			objectCreatePayload(t, r)
			response := objectMetadataResponse(307, "redirect framing proof")
			response.Header.Set("Location", r.URL.String())
			return response, nil
		})
		client.HTTPClient.CheckRedirect = func(next *http.Request, _ []*http.Request) error { redirects++; next.ContentLength = 999; return nil }
		result, err := objects.New(client).CreateObject(context.Background(), "c", "o", objects.CreateObjectInput{Data: []byte("owned")})
		if result == nil || result.Ordinary == nil || result.Ordinary.Acknowledgement != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || redirects != 1 || len(result.Ordinary.Attempts) != 1 || result.Ordinary.Attempts[0].Response.StatusCode != 307 || string(result.Ordinary.Attempts[0].Response.Body) != "redirect framing proof" {
			t.Fatalf("result=%+v err=%v calls=%d redirects=%d", result, err, calls, redirects)
		}
	})
}

func TestCreateObjectContractsAcceptedFaultsAndUnconfirmedSegments(t *testing.T) {
	t.Run("physical response plus transport error is evidence without acknowledgement", func(t *testing.T) {
		transportCause := errors.New("transport returned response and error")
		body := &objectMetadataBody{Reader: strings.NewReader("unconsumed response")}
		calls := 0
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			objectCreatePayload(t, r)
			return objectMetadataWire(201, body), transportCause
		})
		result, err := objects.New(client).CreateObject(context.Background(), "c", "o", objects.CreateObjectInput{Data: []byte("owned")})
		if result == nil || result.Ordinary == nil || result.Ordinary.Acknowledgement != nil || calls != 1 || !errors.Is(err, transportCause) || len(result.Ordinary.Attempts) != 1 || body.closes.Load() != 1 {
			t.Fatalf("result=%+v err=%v calls=%d closes=%d", result, err, calls, body.closes.Load())
		}
		attempt := result.Ordinary.Attempts[0]
		if attempt.Response == nil || attempt.Response.StatusCode != 201 || attempt.Response.Header.Get("X-Trans-Id") != "actual-object" || len(attempt.Response.Body) != 0 || !errors.Is(attempt.Error, transportCause) {
			t.Fatalf("physical observation=%+v", attempt)
		}
	})
	t.Run("request Close observes source drift even when transport restores it", func(t *testing.T) {
		calls := 0
		var client *gophercloud.ServiceClient
		client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			data, err := io.ReadAll(r.Body)
			if err != nil || string(data) != "owned" {
				t.Errorf("payload=%q err=%v", data, err)
			}
			client.Endpoint = "https://changed.invalid/v1/AUTH_changed/"
			_ = r.Body.Close()
			client.Endpoint = objectMetadataBase
			return objectMetadataResponse(201, "transport acknowledgement"), nil
		})
		result, err := objects.New(client).CreateObject(context.Background(), "c", "o", objects.CreateObjectInput{Data: []byte("owned")})
		if result == nil || result.Ordinary == nil || result.Ordinary.Acknowledgement == nil || calls != 1 || !errors.Is(err, resource.ErrInvalidOption) || result.Ordinary.Acknowledgement.StatusCode != 201 {
			t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
		}
		objectMetadataProof(t, err, 201, "transport acknowledgement")
	})
	t.Run("accepted Close and transient source drift preserve proof without replay", func(t *testing.T) {
		closeErr := errors.New("accepted Close failed")
		calls, hooks := 0, 0
		var client *gophercloud.ServiceClient
		original := objectMetadataBase
		body := &objectMetadataBody{Reader: strings.NewReader("accepted bytes"), closeErr: closeErr, onClose: func() { client.Endpoint = original }}
		client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			objectCreatePayload(t, r)
			client.Endpoint = "https://changed.invalid/v1/AUTH_changed/"
			return objectMetadataWire(201, body), nil
		})
		client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			hooks++
			return nil
		}
		result, err := objects.New(client).CreateObject(context.Background(), "c", "o", objects.CreateObjectInput{Data: []byte("owned")})
		if result == nil || result.Ordinary == nil || calls != 1 || hooks != 0 || body.closes.Load() != 1 || !errors.Is(err, closeErr) || !errors.Is(err, resource.ErrInvalidOption) || len(result.Ordinary.Attempts) != 1 || result.Ordinary.Acknowledgement == nil || string(result.Ordinary.Acknowledgement.Body) != "accepted bytes" {
			t.Fatalf("result=%+v err=%v HTTP=%d hooks=%d closes=%d", result, err, calls, hooks, body.closes.Load())
		}
		proof := objectMetadataProof(t, err, 201, "accepted bytes")
		result.Ordinary.Attempts[0].Response.Body[0] = 'X'
		if string(proof.Body) != "accepted bytes" || string(result.Ordinary.Acknowledgement.Body) != "accepted bytes" {
			t.Fatal("response proofs share bytes")
		}
	})
	t.Run("segment202 is an acknowledgement without owned content or manifest", func(t *testing.T) {
		var manifests, deletes atomic.Int32
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			switch r.Method {
			case "GET":
				return objectCreateInfo(), nil
			case "HEAD":
				return objectMetadataResponse(404, "missing"), nil
			case "DELETE":
				deletes.Add(1)
				return objectMetadataResponse(204, ""), nil
			case "PUT":
				objectCreatePayload(t, r)
				if r.URL.RawQuery != "" {
					manifests.Add(1)
				}
				return objectMetadataResponse(202, "unconfirmed"), nil
			default:
				t.Errorf("unexpected=%s", r.Method)
				return objectMetadataResponse(500, ""), nil
			}
		})
		result, err := objects.New(client).CreateObject(context.Background(), "c", "o", objects.CreateObjectInput{Reader: strings.NewReader("abcd")}, objects.WithCreateObjectSegmentSize(2))
		var semantic *objects.ObjectCreateUnconfirmedSegmentError
		if result == nil || !errors.As(err, &semantic) || semantic.Name == "" || result.Manifest != nil || len(result.Cleanup) != 0 || manifests.Load() != 0 || deletes.Load() != 0 {
			t.Fatalf("result=%+v err=%v manifest=%d delete=%d", result, err, manifests.Load(), deletes.Load())
		}
		objectMetadataProof(t, err, 202, "unconfirmed")
		started := 0
		for _, segment := range result.Segments {
			if segment.Upload != nil && len(segment.Upload.Attempts) != 0 {
				started++
				if segment.Created || !segment.Ambiguous || segment.Upload.Acknowledgement == nil || segment.Upload.Acknowledgement.StatusCode != 202 || len(segment.Upload.Attempts) != 1 {
					t.Fatalf("segment=%+v", segment)
				}
			}
		}
		if started == 0 {
			t.Fatal("no physical segment evidence")
		}
	})
}

func TestCreateObjectContractsCleanupAndStickyAmbiguity(t *testing.T) {
	for _, scenario := range []string{"clean rejection", "transport error", "response and transport error"} {
		t.Run(scenario, func(t *testing.T) {
			ambiguous := scenario != "clean rejection"
			var mu sync.Mutex
			created := map[string]bool{}
			var deleted []string
			manifestCalls := 0
			transportCause := errors.New("manifest transport uncertain")
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				switch r.Method {
				case "GET":
					return objectCreateInfo(), nil
				case "HEAD":
					return objectMetadataResponse(404, "missing"), nil
				case "PUT":
					objectCreatePayload(t, r)
					if r.URL.RawQuery == "multipart-manifest=put" {
						manifestCalls++
						if ambiguous && manifestCalls == 1 {
							if scenario == "response and transport error" {
								return objectMetadataResponse(201, "unconsumed manifest"), transportCause
							}
							return nil, transportCause
						}
						return objectMetadataResponse(400, "manifest rejected"), nil
					}
					name := strings.TrimPrefix(r.URL.Path, "/reverse/a b/v1/AUTH_account/c/")
					mu.Lock()
					created[name] = true
					mu.Unlock()
					return objectMetadataResponse(201, "segment created"), nil
				case "DELETE":
					name := strings.TrimPrefix(r.URL.Path, "/reverse/a b/v1/AUTH_account/c/")
					if r.URL.RawQuery != "" || r.Body != nil {
						t.Errorf("cleanup route=%s body=%v", r.URL, r.Body)
					}
					mu.Lock()
					deleted = append(deleted, name)
					mu.Unlock()
					return objectMetadataResponse(204, "deleted"), nil
				default:
					t.Errorf("unexpected=%s", r.Method)
					return objectMetadataResponse(500, ""), nil
				}
			})
			result, err := objects.New(client).CreateObject(context.Background(), "c", "o", objects.CreateObjectInput{Reader: strings.NewReader("abcde")}, objects.WithCreateObjectSegmentSize(2))
			if result == nil || err == nil || result.Manifest == nil || result.Manifest.Acknowledgement != nil || manifestCalls != 3 || len(result.Manifest.Attempts) != 3 || result.ManifestAmbiguous != ambiguous || len(created) != 3 {
				t.Fatalf("result=%+v err=%v manifests=%d created=%v", result, err, manifestCalls, created)
			}
			if ambiguous {
				if len(deleted) != 0 || len(result.Cleanup) != 0 || !errors.Is(result.Manifest.Attempts[0].Error, transportCause) {
					t.Fatalf("ambiguous cleanup=%v result=%+v", deleted, result)
				}
				first := result.Manifest.Attempts[0].Response
				if scenario == "response and transport error" {
					if first == nil || first.StatusCode != 201 || len(first.Body) != 0 {
						t.Fatalf("uncertain manifest response=%+v", first)
					}
				} else if first != nil {
					t.Fatalf("transport-only response=%+v", first)
				}
			} else {
				if len(deleted) != 3 || len(result.Cleanup) != 3 {
					t.Fatalf("cleanup=%v result=%+v", deleted, result)
				}
				for i, cleanup := range result.Cleanup {
					if !created[cleanup.Name] || cleanup.Name != result.Segments[i].Name || cleanup.Deletion == nil || cleanup.Deletion.Acknowledgement.StatusCode != 204 || len(cleanup.Deletion.Attempts) != 1 {
						t.Fatalf("cleanup=%+v", cleanup)
					}
				}
			}
		})
	}
}
