package objects_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestIsObjectStaleContractsLazyFilesAndLiteralHEAD(t *testing.T) {
	for _, status := range []int{200, 204, 404} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			md := strings.Repeat("A", 32)
			calls := 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "HEAD" || r.URL.String() != objectMetadataEndpoint || r.URL.RawQuery != "" || r.Body != nil || r.Header.Get("X-Auth-Token") != "original-token" || r.Header.Get("X-Check") != "explicit" {
					t.Errorf("HEAD=%s %s headers=%v", r.Method, r.URL, r.Header)
				}
				response := objectMetadataResponse(status, "head-proof")
				response.Header.Set("X-Object-Meta-X-Sdk-Md5", md)
				// Hash comparison does not depend on unrelated object metadata parsing.
				response.Header.Set("Last-Modified", "not a date")
				response.Header.Set("Content-Length", "not an integer")
				return response, nil
			})
			client.ResourceBase = objectMetadataBase
			client.Endpoint = "https://swift.invalid/catalogue/v1/AUTH_else/"
			filename := filepath.Join(t.TempDir(), "does not exist")
			result, err := objects.New(client).IsObjectStale(context.Background(), objectMetadataContainer, objectMetadataKey, filename, objects.WithIsObjectStaleMD5(md), objects.WithIsObjectStaleHeader("X-Check", "explicit"))
			if err != nil || result == nil || result.Discovery == nil || result.Discovery.StatusCode != status || string(result.Discovery.Body) != "head-proof" || result.Stale == nil || *result.Stale != (status == 404) || calls != 1 {
				t.Fatalf("result=%+v err=%v HTTP=%d", result, err, calls)
			}
			if status != 404 && (result.RemoteMD5 == nil || *result.RemoteMD5 != md || result.MD5 != md || result.SHA256 != "") {
				t.Fatalf("hash projection=%+v", result)
			}
		})
	}
	t.Run("existing object hashes explicit file only after HEAD", func(t *testing.T) {
		filename := objectCreateFile(t, []byte("before"))
		md, sha := objectCreateHashes([]byte("after"))
		calls := 0
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != "HEAD" {
				t.Errorf("unexpected=%s", r.Method)
			}
			if err := os.WriteFile(filename, []byte("after"), 0600); err != nil {
				t.Errorf("rewrite=%v", err)
			}
			response := objectMetadataResponse(200, "discovery")
			response.Header.Set("X-Object-Meta-X-Sdk-Md5", md)
			response.Header.Set("X-Object-Meta-X-Sdk-Sha256", sha)
			return response, nil
		})
		result, err := objects.New(client).IsObjectStale(context.Background(), "c", "o", filename)
		if err != nil || result == nil || result.Stale == nil || *result.Stale || result.MD5 != md || result.SHA256 != sha || calls != 1 {
			t.Fatalf("result=%+v err=%v HTTP=%d", result, err, calls)
		}
	})
}

func TestIsObjectStaleContractsHashProjectionAndAtomicErrors(t *testing.T) {
	md, sha := strings.Repeat("a", 32), strings.Repeat("b", 64)
	mismatch := strings.Repeat("c", 64)
	for _, tc := range []struct {
		name                 string
		header               http.Header
		localMD5, localSHA   string
		stale                bool
		remoteMD5, remoteSHA *string
	}{
		{"legacy digest matches", http.Header{"X-Object-Meta-X-Shade-Md5": {md}}, md, "", false, &md, nil},
		{"one available local digest matches", http.Header{"X-Object-Meta-X-Sdk-Sha256": {sha}}, "", sha, false, nil, &sha},
		{"current empty shadows matching legacy", http.Header{"X-Object-Meta-X-Sdk-Md5": {""}, "X-Object-Meta-X-Shade-Md5": {md}}, md, "", true, new(string), nil},
		{"available mismatch defeats other match", http.Header{"X-Object-Meta-X-Sdk-Md5": {md}, "X-Object-Meta-X-Sdk-Sha256": {mismatch}}, md, sha, true, &md, &mismatch},
		{"missing remote digest defeats other match", http.Header{"X-Object-Meta-X-Sdk-Md5": {md}}, md, sha, true, &md, nil},
		{"no available remote digest", http.Header{"ETag": {md}}, md, sha, true, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
				response := objectMetadataResponse(200, "observed")
				for key, values := range tc.header {
					response.Header[key] = append([]string{}, values...)
				}
				return response, nil
			})
			result, err := objects.New(client).IsObjectStale(context.Background(), "c", "o", filepath.Join(t.TempDir(), "unread"), objects.WithIsObjectStaleMD5(tc.localMD5), objects.WithIsObjectStaleSHA256(tc.localSHA))
			if err != nil || result == nil || result.Stale == nil || *result.Stale != tc.stale {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if tc.remoteMD5 != nil && (result.RemoteMD5 == nil || *result.RemoteMD5 != *tc.remoteMD5) || tc.remoteMD5 == nil && result.RemoteMD5 != nil {
				t.Fatalf("remoteMD5=%v expected=%v", result.RemoteMD5, tc.remoteMD5)
			}
			if tc.remoteSHA != nil && (result.RemoteSHA256 == nil || *result.RemoteSHA256 != *tc.remoteSHA) || tc.remoteSHA == nil && result.RemoteSHA256 != nil {
				t.Fatalf("remoteSHA=%v expected=%v", result.RemoteSHA256, tc.remoteSHA)
			}
		})
	}
	t.Run("alias or control fault leaves decision and projection uncommitted", func(t *testing.T) {
		for _, header := range []http.Header{
			{"X-Object-Meta-X-Sdk-Md5": {md}, "x-object-meta-x-sdk-md5": {md}},
			{"X-Object-Meta-X-Sdk-Md5": {md, md}},
			{"X-Object-Meta-X-Sdk-Md5": {md}, "X-Object-Meta-X-Sdk-Sha256": {"bad\x7fvalue"}},
		} {
			client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
				response := objectMetadataResponse(200, "bad projection")
				for key, values := range header {
					response.Header[key] = values
				}
				return response, nil
			})
			result, err := objects.New(client).IsObjectStale(context.Background(), "c", "o", filepath.Join(t.TempDir(), "unread"), objects.WithIsObjectStaleMD5(md))
			if result == nil || result.Discovery == nil || result.Stale != nil || result.RemoteMD5 != nil || result.RemoteSHA256 != nil || err == nil {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			objectMetadataProof(t, err, 200, "bad projection")
		}
	})
	t.Run("local open error keeps accepted discovery", func(t *testing.T) {
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
			return objectMetadataResponse(200, "before local IO"), nil
		})
		result, err := objects.New(client).IsObjectStale(context.Background(), "c", "o", filepath.Join(t.TempDir(), "missing file"))
		if result == nil || result.Discovery == nil || string(result.Discovery.Body) != "before local IO" || result.Stale != nil || !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	})
}

func TestIsObjectStaleContractsDirtyMissingAndLiveGuards(t *testing.T) {
	t.Run("physical404 read fault retains consumed bytes and cause", func(t *testing.T) {
		readErr := io.ErrUnexpectedEOF
		body := &objectMetadataBody{Reader: objectMetadataReader(func(p []byte) (int, error) { return copy(p, "partial missing"), readErr })}
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) { return objectMetadataWire(404, body), nil })
		result, err := objects.New(client).IsObjectStale(context.Background(), "c", "o", filepath.Join(t.TempDir(), "unread"))
		if !errors.Is(err, readErr) || result != nil && result.Stale != nil || body.closes.Load() != 1 {
			t.Fatalf("result=%+v err=%v closes=%d", result, err, body.closes.Load())
		}
		objectMetadataProof(t, err, 404, "partial missing")
	})
	t.Run("physical404 Close error cannot become stale success", func(t *testing.T) {
		closeErr := errors.New("missing response Close")
		calls := 0
		body := &objectMetadataBody{Reader: strings.NewReader("physical missing"), closeErr: closeErr}
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return objectMetadataWire(404, body), nil })
		result, err := objects.New(client).IsObjectStale(context.Background(), "c", "o", filepath.Join(t.TempDir(), "unread"))
		if err == nil || !errors.Is(err, closeErr) || result != nil && result.Stale != nil || calls != 1 || body.closes.Load() != 1 {
			t.Fatalf("result=%+v err=%v calls=%d closes=%d", result, err, calls, body.closes.Load())
		}
		objectMetadataProof(t, err, 404, "physical missing")
	})
	t.Run("accepted source drift stays observed after Close restores it", func(t *testing.T) {
		var client *gophercloud.ServiceClient
		body := &objectMetadataBody{Reader: strings.NewReader("accepted proof"), onClose: func() { client.Endpoint = objectMetadataBase }}
		client = objectMetadataClient(func(*http.Request) (*http.Response, error) {
			client.Endpoint = "https://changed.invalid/v1/AUTH_changed/"
			response := objectMetadataWire(200, body)
			response.Header.Set("X-Object-Meta-X-Sdk-Md5", strings.Repeat("a", 32))
			return response, nil
		})
		result, err := objects.New(client).IsObjectStale(context.Background(), "c", "o", filepath.Join(t.TempDir(), "unread"), objects.WithIsObjectStaleMD5(strings.Repeat("a", 32)))
		if result == nil || result.Discovery == nil || result.Stale != nil || result.RemoteMD5 != nil || !errors.Is(err, resource.ErrInvalidOption) || body.closes.Load() != 1 {
			t.Fatalf("result=%+v err=%v closes=%d", result, err, body.closes.Load())
		}
		objectMetadataProof(t, err, 200, "accepted proof")
	})
	t.Run("option snapshots and fresh native retry retain caller policy", func(t *testing.T) {
		md := strings.Repeat("A", 32)
		headers := map[string]string{"X-Check": "captured"}
		factory := objects.WithIsObjectStaleHeaders(headers)
		headers["X-Check"] = "changed"
		calls, callbacks, hooks := 0, 0, 0
		var client *gophercloud.ServiceClient
		client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != "HEAD" || r.Body != nil || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Check") != "captured" {
				t.Errorf("HEAD snapshot=%v", r.Header)
			}
			if calls == 1 {
				return objectMetadataResponse(503, "retry proof"), nil
			}
			response := objectMetadataResponse(204, "accepted proof")
			response.Header.Set("X-Object-Meta-X-Sdk-Md5", md)
			return response, nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			hooks++
			return nil
		}
		result, err := objects.New(client).IsObjectStale(context.Background(), "c", "o", filepath.Join(t.TempDir(), "unread"), factory, objects.WithIsObjectStaleMD5(md), func(*objects.IsObjectStaleOpts) error {
			callbacks++
			client.MoreHeaders["X-Source"] = "valid drift"
			return nil
		})
		if err != nil || result == nil || result.Stale == nil || *result.Stale || calls != 2 || hooks != 1 || callbacks != 1 || result.Discovery.StatusCode != 204 {
			t.Fatalf("result=%+v err=%v calls=%d hooks=%d callbacks=%d", result, err, calls, hooks, callbacks)
		}
	})
	t.Run("nil option and malformed original header reject before callback or HTTP", func(t *testing.T) {
		calls, callbacks := 0, 0
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return objectMetadataResponse(200, ""), nil })
		api := objects.New(client)
		if result, err := api.IsObjectStale(context.Background(), "c", "o", "explicit", nil); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("result=%+v err=%v", result, err)
		}
		client.MoreHeaders = map[string]string{"X-Source": "bad\nvalue"}
		result, err := api.IsObjectStale(context.Background(), "c", "o", "explicit", func(*objects.IsObjectStaleOpts) error { callbacks++; return nil })
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
			t.Fatalf("result=%+v err=%v calls=%d callbacks=%d", result, err, calls, callbacks)
		}
	})
}
