package objects

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestImageImportObjectBorrowedDataUsesOnlyOneOrdinaryPUT(t *testing.T) {
	for _, content := range []string{"borrowed image", ""} {
		t.Run("content="+content, func(t *testing.T) {
			c := objectMetadataClient()
			borrowed := &createCoreBorrowed{reader: strings.NewReader(content)}
			calls := 0
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls != 1 || borrowed.reads != 0 || r.Method != "PUT" || r.URL.EscapedPath() != "/v1/AUTH_a/box/dir%2Fimage" || r.GetBody != nil || r.ContentLength != -1 {
					t.Fatalf("unexpected borrowed request: calls=%d reads=%d request=%+v", calls, borrowed.reads, r)
				}
				if r.Header.Get("Content-Type") != "application/octet-stream" || r.Header.Get("X-Delete-After") != "86400" || r.Header.Get("X-Object-Meta-X-Sdk-Autocreated") != "true" || r.Header.Get("X-Trace") != "captured" || r.Header.Get("X-Object-Meta-X-Sdk-Md5") != "" {
					t.Fatal("image controls or ordinary header changed", r.Header)
				}
				data, err := io.ReadAll(r.Body)
				if err != nil || string(data) != content {
					t.Fatal(string(data), err)
				}
				if err := r.Body.Close(); err != nil {
					t.Fatal(err)
				}
				return objectMetadataWire(r, 299, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("actual"))), nil
			})
			// Ordinary data ignores the filename-only digest inputs, even invalid ones.
			result, err := New(c).CreateImageImportObject(context.Background(), ImageImportObjectRequest{Container: "box", Name: "dir/image", Data: borrowed, DataPresent: true, MD5: "ignored\n", SHA256: "arbitrary", Headers: map[string]string{"X-Trace": "captured"}})
			if err != nil || calls != 1 || result == nil || result.Created != nil || result.Uploaded == nil || result.Uploaded.Acknowledgement == nil || result.Uploaded.Acknowledgement.StatusCode != 299 || string(result.Uploaded.Acknowledgement.Body) != "actual" || len(result.Uploaded.Attempts) != 1 || borrowed.closes != 0 || borrowed.seeks != 0 {
				t.Fatal("borrowed PUT evidence/ownership", result, err, calls, borrowed)
			}
		})
	}
}

func TestImageImportObjectPresentNilDataIsEmptyPUT(t *testing.T) {
	c := objectMetadataClient()
	calls := 0
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		data, err := io.ReadAll(r.Body)
		if err != nil || len(data) != 0 || r.GetBody != nil || r.ContentLength != 0 || r.Method != "PUT" {
			t.Fatal("nil present data", r, string(data), err)
		}
		return objectMetadataOK(r, 204, http.Header{}), nil
	})
	result, err := New(c).CreateImageImportObject(context.Background(), ImageImportObjectRequest{Container: "box", Name: "image", DataPresent: true})
	if err != nil || calls != 1 || result == nil || result.Uploaded.Acknowledgement.StatusCode != 204 {
		t.Fatal(result, err, calls)
	}
}

func TestImageImportObjectBorrowedDataNeverRunsNativeReplayPolicies(t *testing.T) {
	for _, status := range []int{401, 429, 503, 307} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			c := objectMetadataClient()
			calls, hooks := 0, 0
			borrowed := &createCoreBorrowed{reader: strings.NewReader("once")}
			c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks++
				return nil
			}
			c.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { hooks++; return nil }
			c.MaxBackoffRetries = 3
			c.ReauthFunc = func(context.Context) error { hooks++; return nil }
			c.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { hooks++; return nil }
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				data, err := io.ReadAll(r.Body)
				if err != nil || string(data) != "once" || r.GetBody != nil {
					t.Fatal(string(data), err, r.GetBody == nil)
				}
				return objectMetadataWire(r, status, http.Header{"Location": {r.URL.String()}, "X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("native response"))), nil
			})
			result, err := New(c).CreateImageImportObject(context.Background(), ImageImportObjectRequest{Container: "box", Name: "image", Data: borrowed, DataPresent: true})
			if calls != 1 || hooks != 0 || borrowed.closes != 0 || borrowed.seeks != 0 || result == nil || result.Uploaded == nil || len(result.Uploaded.Attempts) != 1 || result.Uploaded.Attempts[0].Response.StatusCode != status || string(result.Uploaded.Attempts[0].Response.Body) != "native response" {
				t.Fatal(result, err, calls, hooks, borrowed)
			}
			if status >= 400 {
				var native gophercloud.ErrUnexpectedResponseCode
				if err == nil || !errors.As(err, &native) || native.Actual != status || result.Uploaded.Acknowledgement != nil {
					t.Fatal("native failure lost", result, err, native)
				}
			} else if err != nil || result.Uploaded.Acknowledgement == nil || result.Uploaded.Acknowledgement.StatusCode != 307 {
				t.Fatal("Source 3xx acceptance", result, err)
			}
		})
	}
}

func TestImageImportObjectPreflightRejectsConflictAndOwnedHeaderInputs(t *testing.T) {
	var typedNil *createCoreBorrowed
	tests := []struct {
		name    string
		request ImageImportObjectRequest
	}{
		{"filename and empty data", ImageImportObjectRequest{Filename: "file", DataPresent: true}},
		{"typed nil", ImageImportObjectRequest{DataPresent: true, Data: typedNil}},
		{"unmarked reader", ImageImportObjectRequest{Data: strings.NewReader("data")}},
		{"content type", ImageImportObjectRequest{DataPresent: true, Headers: map[string]string{"content-type": "application/octet-stream"}}},
		{"TTL", ImageImportObjectRequest{DataPresent: true, Headers: map[string]string{"x-delete-after": "86400"}}},
		{"autocreated", ImageImportObjectRequest{DataPresent: true, Headers: map[string]string{"X-Object-Meta-X-Sdk-Autocreated": "true"}}},
		{"header aliases", ImageImportObjectRequest{DataPresent: true, Headers: map[string]string{"X-A": "one", "x-a": "two"}}},
		{"invalid hash header", ImageImportObjectRequest{Filename: "unopened", MD5: "bad\nvalue"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := objectMetadataClient()
			calls := 0
			c.HTTPClient.Transport = objectMetadataTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("HTTP forbidden") })
			tc.request.Container, tc.request.Name = "box", "image"
			result, err := New(c).CreateImageImportObject(context.Background(), tc.request)
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(result, err, calls)
			}
		})
	}
}

func TestImageImportObjectFilenameReusesStaleAndArbitraryHashPolicy(t *testing.T) {
	for _, tc := range []struct {
		name, md, sha string
		head          int
		skip          bool
	}{
		{"arbitrary supplied", "md5 value", "sha value", 404, false},
		{"empty supplied", "", "", 404, false},
		{"matching arbitrary", "md5 value", "sha value", 299, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := objectMetadataClient()
			methods := []string{}
			puts := 0
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				methods = append(methods, r.Method)
				switch r.Method {
				case "GET":
					return objectMetadataWire(r, 299, http.Header{}, io.NopCloser(strings.NewReader(`{"swift":{"max_file_size":100},"slo":{"min_segment_size":1}}`))), nil
				case "HEAD":
					return objectMetadataOK(r, tc.head, http.Header{"X-Object-Meta-X-Sdk-Md5": {tc.md}, "X-Object-Meta-X-Sdk-Sha256": {tc.sha}}), nil
				case "PUT":
					puts++
					data, err := io.ReadAll(r.Body)
					if err != nil || string(data) != "local file" || r.Header.Get("X-Object-Meta-X-Sdk-Md5") != tc.md || r.Header.Get("X-Object-Meta-X-Sdk-Sha256") != tc.sha || r.Header.Get("X-Object-Meta-X-Sdk-Autocreated") != "true" || r.Header.Get("Content-Type") != "application/octet-stream" || r.Header.Get("X-Delete-After") != "86400" {
						t.Fatal(string(data), r.Header, err)
					}
					if tc.md == "" {
						if _, exists := r.Header["X-Object-Meta-X-Sdk-Md5"]; exists {
							t.Fatal("empty hashes generated", r.Header)
						}
					}
					return objectMetadataOK(r, 299, http.Header{}), nil
				}
				return nil, errors.New("unexpected method")
			})
			result, err := New(c).CreateImageImportObject(context.Background(), ImageImportObjectRequest{Container: "box", Name: "image", Filename: createCoreFile(t, "local file"), MD5: tc.md, SHA256: tc.sha})
			wantPuts := 1
			if tc.skip {
				wantPuts = 0
			}
			if err != nil || result == nil || result.Created == nil || result.Uploaded != nil || result.Created.Skipped != tc.skip || puts != wantPuts || len(methods) != 2+wantPuts {
				t.Fatal(result, err, methods)
			}
			if !tc.skip && result.Created.Ordinary.Acknowledgement.StatusCode != 299 {
				t.Fatal(result.Created.Ordinary)
			}
		})
	}
	c := objectMetadataClient()
	calls := 0
	c.HTTPClient.Transport = objectMetadataTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("forbidden") })
	_, err := New(c).CreateObject(context.Background(), "box", "image", CreateObjectInput{Filename: "unopened"}, WithCreateObjectMD5("md5 value"))
	if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatal("standalone hash policy changed", err, calls)
	}
}

func TestImageImportObjectAbsentDataFallsBackToNameFilename(t *testing.T) {
	filename := createCoreFile(t, "inferred source")
	c := objectMetadataClient()
	puts := 0
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		switch r.Method {
		case "GET":
			return objectMetadataWire(r, 200, http.Header{}, io.NopCloser(strings.NewReader(`{"swift":{"max_file_size":100}}`))), nil
		case "HEAD":
			return objectMetadataOK(r, 404, http.Header{}), nil
		case "PUT":
			puts++
			data, err := io.ReadAll(r.Body)
			if err != nil || string(data) != "inferred source" {
				t.Fatal(string(data), err)
			}
			return objectMetadataOK(r, 201, http.Header{}), nil
		}
		return nil, errors.New("unexpected")
	})
	result, err := New(c).CreateImageImportObject(context.Background(), ImageImportObjectRequest{Container: "box", Name: filename})
	if err != nil || result == nil || result.Created == nil || puts != 1 {
		t.Fatal(result, err, puts)
	}
}

func TestImageImportObjectFilenameReusesSLOWithBroadSourceAcknowledgements(t *testing.T) {
	c := objectMetadataClient()
	var mu sync.Mutex
	segments, manifests := 0, 0
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		switch r.Method {
		case "GET":
			return objectMetadataWire(r, 200, http.Header{}, io.NopCloser(strings.NewReader(`{"swift":{"max_file_size":2},"slo":{"min_segment_size":1}}`))), nil
		case "HEAD":
			return objectMetadataOK(r, 404, http.Header{}), nil
		case "PUT":
			data, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			if r.Header.Get("X-Delete-After") != "86400" || r.Header.Get("Content-Type") != "application/octet-stream" || r.Header.Get("X-Object-Meta-X-Sdk-Autocreated") != "true" || r.Header.Get("X-Object-Meta-X-Sdk-Md5") != "supplied" {
				t.Error(r.Header)
			}
			mu.Lock()
			defer mu.Unlock()
			if r.URL.Query().Get("multipart-manifest") == "put" {
				manifests++
				if !strings.Contains(string(data), `"size_bytes":2`) || !strings.Contains(string(data), `"size_bytes":1`) {
					t.Error("missing reused manifest", string(data))
				}
				return objectMetadataOK(r, 299, http.Header{}), nil
			}
			segments++
			if len(data) != 2 && len(data) != 1 {
				t.Error(string(data))
			}
			return objectMetadataOK(r, 202, http.Header{"Etag": {"segment"}}), nil
		}
		return nil, errors.New("unexpected")
	})
	result, err := New(c).CreateImageImportObject(context.Background(), ImageImportObjectRequest{Container: "box", Name: "image", Filename: createCoreFile(t, "abc"), MD5: "supplied"})
	if err != nil || result == nil || result.Created == nil || result.Created.Mode != "slo" || segments != 2 || manifests != 1 || len(result.Created.Segments) != 2 || result.Created.Manifest.Acknowledgement.StatusCode != 299 {
		t.Fatal(result, err, segments, manifests)
	}
	for _, segment := range result.Created.Segments {
		if !segment.Created || segment.Upload.Acknowledgement.StatusCode != 202 {
			t.Fatal(segment)
		}
	}
}

func TestImageImportObjectOuterGuardRetainsRequestReadAndAcceptedResponseEvidence(t *testing.T) {
	for _, during := range []string{"request Read", "response Read", "response Close"} {
		t.Run(during, func(t *testing.T) {
			c := objectMetadataClient()
			outer := errors.New("image source changed")
			changed := false
			calls := 0
			ctx := rest.WithOperationSources(rest.WithOperationGuard(context.Background(), func(context.Context) error {
				if changed {
					return outer
				}
				return nil
			}))
			borrowed := &createCoreBorrowed{reader: strings.NewReader("image")}
			if during == "request Read" {
				borrowed.reader = objectMetadataReader(func(data []byte) (int, error) { changed = true; return copy(data, "partial"), io.EOF })
			}
			response := &objectMetadataBody{Reader: strings.NewReader("actual"), onClose: func() { changed = false }}
			if during == "response Read" {
				read := false
				response.Reader = objectMetadataReader(func(data []byte) (int, error) {
					if read {
						return 0, io.EOF
					}
					read = true
					changed = true
					return copy(data, "actual"), io.EOF
				})
			}
			if during == "response Close" {
				response.onClose = func() { changed = true }
			}
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				_, readErr := io.ReadAll(r.Body)
				if during == "request Read" && !errors.Is(readErr, outer) {
					t.Fatal("request guard lost", readErr)
				}
				return objectMetadataWire(r, 201, http.Header{"X-Proof": {"kept"}}, response), nil
			})
			result, err := New(c).CreateImageImportObject(ctx, ImageImportObjectRequest{Container: "box", Name: "image", DataPresent: true, Data: borrowed})
			if !errors.Is(err, outer) || result == nil || result.Uploaded == nil || result.Uploaded.Acknowledgement == nil || result.Uploaded.Acknowledgement.StatusCode != 201 || string(result.Uploaded.Acknowledgement.Body) != "actual" || calls != 1 || borrowed.closes != 0 || borrowed.seeks != 0 || response.closes.Load() != 1 {
				t.Fatal(result, err, calls, borrowed, response.closes.Load())
			}
			proof := objectMetadataProof(t, err, 201, "actual")
			result.Uploaded.Acknowledgement.Body[0] = '!'
			if string(proof.Body) != "actual" {
				t.Fatal("receipt/error aliases", proof)
			}
		})
	}
}

func TestImageImportObjectRegistersOnlySourceGuardAndKeepsBindingAfterReturn(t *testing.T) {
	for _, field := range []string{"endpoint", "base", "type", "provider", "API", "TTL"} {
		t.Run(field, func(t *testing.T) {
			c := objectMetadataClient()
			a := New(c)
			outerCalls := 0
			ctx := rest.WithOperationSources(rest.WithOperationGuard(context.Background(), func(context.Context) error {
				outerCalls++
				if outerCalls > 100 {
					t.Fatal("recursive outer guard")
				}
				return nil
			}))
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) { return objectMetadataOK(r, 201, http.Header{}), nil })
			result, err := a.CreateImageImportObject(ctx, ImageImportObjectRequest{Container: "box", Name: "image", DataPresent: true})
			if err != nil || result == nil || outerCalls == 0 {
				t.Fatal(result, err, outerCalls)
			}
			switch field {
			case "endpoint":
				c.Endpoint = "http://changed.invalid/v1/AUTH_a/"
			case "base":
				c.ResourceBase = "http://swift.invalid/else/"
			case "type":
				c.Type = "image"
			case "provider":
				c.ProviderClient = &gophercloud.ProviderClient{}
			case "API":
				a.client = objectMetadataClient()
			case "TTL":
				c.MoreHeaders = map[string]string{"X-Delete-After": "1"}
			}
			if err := rest.CheckOperationGuard(ctx); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("registered Swift source lost", err)
			}
		})
	}
}

func TestImageImportObjectFilenameNativeRetryCannotReplaceImageControls(t *testing.T) {
	for _, header := range []string{"Content-Type", "X-Delete-After", "X-Object-Meta-X-Sdk-Autocreated"} {
		t.Run(header, func(t *testing.T) {
			c := objectMetadataClient()
			puts, hooks := 0, 0
			c.RetryFunc = func(_ context.Context, method, target string, opts *gophercloud.RequestOpts, original error, count uint) error {
				hooks++
				if opts.MoreHeaders == nil {
					opts.MoreHeaders = map[string]string{}
				}
				opts.MoreHeaders[header] = "changed"
				return nil
			}
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				switch r.Method {
				case "GET":
					return objectMetadataWire(r, 200, http.Header{}, io.NopCloser(strings.NewReader(`{"swift":{"max_file_size":100}}`))), nil
				case "HEAD":
					return objectMetadataOK(r, 404, http.Header{}), nil
				case "PUT":
					puts++
					_, _ = io.ReadAll(r.Body)
					return objectMetadataWire(r, 503, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("failed PUT"))), nil
				}
				return nil, errors.New("unexpected")
			})
			result, err := New(c).CreateImageImportObject(context.Background(), ImageImportObjectRequest{Container: "box", Name: "image", Filename: createCoreFile(t, "file"), MD5: "supplied"})
			if !errors.Is(err, resource.ErrInvalidOption) || result == nil || result.Created == nil || result.Created.Ordinary == nil || len(result.Created.Ordinary.Attempts) != 1 || puts != 1 || hooks != 1 || result.Created.Ordinary.Attempts[0].Response.StatusCode != 503 {
				t.Fatal(result, err, puts, hooks)
			}
		})
	}
}
