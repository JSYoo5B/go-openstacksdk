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

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

type cacheCoreCall struct {
	name, method, path, body string
	status                   int
	call                     func(*Service) (any, error)
}

func cacheCoreCalls(ctx context.Context) []cacheCoreCall {
	return []cacheCoreCall{
		{"get", "GET", "cache", `{"cached_images":[],"queued_images":[]}`, 200, func(s *Service) (any, error) { v, e := s.GetImageCache(ctx); return v, e }},
		{"queue", "PUT", "cache/image-id", "opaque\xff", 202, func(s *Service) (any, error) { v, e := s.QueueImage(ctx, resource.ID("image-id")); return v, e }},
		{"delete", "DELETE", "cache/image-id", "opaque\xff", 204, func(s *Service) (any, error) { v, e := s.CacheDeleteImage(ctx, resource.ID("image-id")); return v, e }},
		{"clear", "DELETE", "cache", "opaque\xff", 204, func(s *Service) (any, error) { v, e := s.ClearCache(ctx); return v, e }},
		{"nodes", "GET", "cache/nodes/image-id", `["","https://other.test/node","literal","literal"]`, 200, func(s *Service) (any, error) { v, e := s.CachedImageNodes(ctx, resource.ID("image-id")); return v, e }},
		{"clean", "POST", "cache/clean", "opaque\xff", 200, func(s *Service) (any, error) { v, e := s.CleanCache(ctx); return v, e }},
		{"prune", "POST", "cache/prune", `{"total_files_pruned":0,"total_bytes_pruned":0}`, 200, func(s *Service) (any, error) { v, e := s.PruneCache(ctx); return v, e }},
	}
}
func cacheCoreNil(value any) bool { return value == nil || reflect.ValueOf(value).IsNil() }

func TestImageCacheCoreFixedRoutesAndActualEvidence(t *testing.T) {
	for _, test := range cacheCoreCalls(context.Background()) {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			body := &deleteCoreBody{reader: strings.NewReader(test.body)}
			headers := http.Header{"X-Proof": {"before"}}
			client := deleteCoreClient(deleteCoreTransport(func(request *http.Request) (*http.Response, error) {
				calls++
				if request.Method != test.method || request.URL.EscapedPath() != "/reverse/glance/v2/"+test.path || request.URL.RawQuery != "" || request.Body != nil || request.Header.Get(cacheTargetHeader) != "" {
					t.Fatal(request.Method, request.URL, request.Body, request.Header)
				}
				return deleteCoreHTTP(test.status, body, headers), nil
			}))
			value, err := test.call(New(client))
			if err != nil || cacheCoreNil(value) || calls != 1 || body.closes != 1 {
				t.Fatal(value, err, calls, body.closes)
			}
			headers.Set("X-Proof", "after")
			switch v := value.(type) {
			case *ImageCache:
				if v.StatusCode != 200 || v.Header.Get("X-Proof") != "before" || v.CachedImages == nil || v.QueuedImages == nil {
					t.Fatal(v)
				}
			case *CachePruneResult:
				if v.StatusCode != 200 || v.Header.Get("X-Proof") != "before" || v.TotalFilesPruned != 0 || v.TotalBytesPruned != 0 {
					t.Fatal(v)
				}
			case *CachedImageNodesResult:
				if v.ImageID != "image-id" || v.StatusCode != 200 || v.Header.Get("X-Proof") != "before" || !reflect.DeepEqual(v.Nodes, []string{"", "https://other.test/node", "literal", "literal"}) {
					t.Fatal(v)
				}
			case *ImageCacheAcknowledgement:
				if v.StatusCode != test.status || v.Header.Get("X-Proof") != "before" || string(v.Body) != test.body {
					t.Fatal(v)
				}
				if test.name == "clear" {
					if v.Target == nil || *v.Target != CacheBoth || v.ImageID != "" {
						t.Fatal(v)
					}
				} else if v.Target != nil {
					t.Fatal(v)
				}
			}
		})
	}
	for _, target := range []CacheTarget{CacheBoth, CacheOnly, QueueOnly} {
		t.Run(fmt.Sprint("target", target), func(t *testing.T) {
			client := deleteCoreClient(deleteCoreTransport(func(r *http.Request) (*http.Response, error) {
				expected := map[CacheTarget]string{CacheBoth: "", CacheOnly: "cache", QueueOnly: "queue"}[target]
				if r.Header.Get(cacheTargetHeader) != expected {
					t.Fatal(r.Header)
				}
				return deleteCoreHTTP(204, io.NopCloser(strings.NewReader("")), nil), nil
			}))
			value, err := New(client).ClearCache(context.Background(), WithClearCacheTarget(target))
			if err != nil || value.Target == nil || *value.Target != target {
				t.Fatal(value, err)
			}
		})
	}
}

func TestImageCacheCoreCanonicalPrecisionAndIndependentMetadata(t *testing.T) {
	text := `{"cached_images":[{"image_id":"","hits":-1,"size":9223372036854775807,"last_accessed":1.234567890123456789e+40,"last_modified":-0.0,"links":false,"created_at":{},"updated_at":42,"extension":9999999999999999999999},{"image_id":null,"hits":null,"size":null,"last_accessed":null,"last_modified":null}],"queued_images":["z","a","z"],"links":"passive","created_at":false,"next":"https://other.test","extension":0.1234567890123456789}`
	headers := http.Header{"X-Proof": {"wire"}}
	client := deleteCoreClient(deleteCoreTransport(func(*http.Request) (*http.Response, error) {
		return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(text)), headers), nil
	}))
	value, err := New(client).GetImageCache(context.Background())
	if err != nil || len(value.CachedImages) != 2 || !reflect.DeepEqual(value.QueuedImages, []string{"z", "a", "z"}) {
		t.Fatal(value, err)
	}
	row := value.CachedImages[0]
	if row.ImageID == nil || *row.ImageID != "" || row.Hits == nil || *row.Hits != -1 || row.Size == nil || *row.Size != 9223372036854775807 || row.LastAccessed.String() != "1.234567890123456789e+40" || row.LastModified.String() != "-0.0" {
		t.Fatal(row)
	}
	if row.CreatedAt != nil || row.UpdatedAt != nil || row.Links != nil || value.CreatedAt != nil || value.Links != nil || row.StatusCode != 200 || row.Header.Get("X-Proof") != "wire" {
		t.Fatal(row, value)
	}
	if n := value.CachedImages[1]; n.ImageID != nil || n.Hits != nil || n.Size != nil || n.LastAccessed != nil || n.LastModified != nil {
		t.Fatal(n)
	}
	row.Body["image_id"][0] = 'x'
	if *row.ImageID != "" || string(value.Body["cached_images"]) != text[strings.Index(text, "["):strings.Index(text, "],\"queued_images")+1] {
		t.Fatal(row, value.Body)
	}
	row.Header.Set("X-Proof", "row")
	value.Header.Set("X-Proof", "root")
	if headers.Get("X-Proof") != "wire" || value.CachedImages[1].Header.Get("X-Proof") != "wire" {
		t.Fatal(headers, value)
	}
	var standalone CachedImage
	if err := json.Unmarshal([]byte(`{"Image_id":false,"Hits":[],"Size":{},"Last_accessed":"quoted","links":{},"created_at":false}`), &standalone); err != nil || standalone.ImageID != nil || standalone.Hits != nil || standalone.Size != nil || standalone.LastAccessed != nil || standalone.Links != nil {
		t.Fatal(standalone, err)
	}
	var prune CachePruneResult
	if err := json.Unmarshal([]byte(`{"total_files_pruned":-1,"total_bytes_pruned":9223372036854775807,"links":false,"created_at":{},"ext":9999999999999999999}`), &prune); err != nil || prune.TotalFilesPruned != -1 || prune.TotalBytesPruned != 9223372036854775807 || prune.Links != nil || prune.CreatedAt != nil || string(prune.Body["ext"]) != "9999999999999999999" {
		t.Fatal(prune, err)
	}
}

func TestImageCacheCoreMalformedModelsRetainAcceptedProof(t *testing.T) {
	tables := []struct {
		name   string
		bodies []string
		call   func(*Service) (any, error)
	}{
		{"get", []string{"", `null`, `[]`, `{}`, `{"cached_images":null,"queued_images":[]}`, `{"cached_images":[],"queued_images":null}`, `{"cached_images":[null],"queued_images":[]}`, `{"cached_images":[[]],"queued_images":[]}`, `{"cached_images":[{"hits":1.0}],"queued_images":[]}`, `{"cached_images":[{"size":9223372036854775808}],"queued_images":[]}`, `{"cached_images":[{"last_accessed":"1"}],"queued_images":[]}`, `{"cached_images":[{"last_modified":true}],"queued_images":[]}`, `{"cached_images":[],"queued_images":[null]}`, `{"cached_images":[],"queued_images":[2]}`, "{\"cached_images\":[],\"queued_images\":[],\"unknown\":\"\xff\"}"}, func(s *Service) (any, error) { v, e := s.GetImageCache(context.Background()); return v, e }},
		{"prune", []string{`{}`, `{"total_files_pruned":null,"total_bytes_pruned":0}`, `{"total_files_pruned":"1","total_bytes_pruned":0}`, `{"total_files_pruned":1e0,"total_bytes_pruned":0}`, `{"total_files_pruned":0,"total_bytes_pruned":9223372036854775808}`}, func(s *Service) (any, error) { v, e := s.PruneCache(context.Background()); return v, e }},
		{"nodes", []string{`null`, `{}`, `[null]`, `[1]`, `[false]`, "[\"\xff\"]"}, func(s *Service) (any, error) {
			v, e := s.CachedImageNodes(context.Background(), resource.ID("id"))
			return v, e
		}},
	}
	for _, group := range tables {
		for i, text := range group.bodies {
			t.Run(fmt.Sprintf("%s/%d", group.name, i), func(t *testing.T) {
				body := &deleteCoreBody{reader: strings.NewReader(text)}
				client := deleteCoreClient(deleteCoreTransport(func(*http.Request) (*http.Response, error) {
					return deleteCoreHTTP(200, body, http.Header{"X-Proof": {"raw"}}), nil
				}))
				value, err := group.call(New(client))
				var proof *resource.ResponseError
				if !cacheCoreNil(value) || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != text || proof.Header.Get("X-Proof") != "raw" || body.closes != 1 {
					t.Fatal(value, err, proof, body.closes)
				}
			})
		}
	}
	prior := int64(7)
	row := CachedImage{Hits: &prior}
	if err := json.Unmarshal([]byte(`{"hits":1,"last_accessed":"1"}`), &row); err == nil || row.Hits != &prior {
		t.Fatal(row, err)
	}
	cache := ImageCache{QueuedImages: []string{"prior"}}
	if err := json.Unmarshal([]byte(`{"cached_images":[],"queued_images":[null]}`), &cache); err == nil || !reflect.DeepEqual(cache.QueuedImages, []string{"prior"}) {
		t.Fatal(cache, err)
	}
	prune := CachePruneResult{TotalFilesPruned: 7}
	if err := json.Unmarshal([]byte(`{"total_files_pruned":1,"total_bytes_pruned":null}`), &prune); err == nil || prune.TotalFilesPruned != 7 {
		t.Fatal(prune, err)
	}
}

func TestImageCacheCoreCompletePreflightAndOwnedTargetHeader(t *testing.T) {
	for _, test := range cacheCoreCalls(context.Background()) {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := deleteCoreClient(deleteCoreTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") }))
			client.MoreHeaders = map[string]string{"x-image-cache-clear-target": ""}
			value, err := test.call(New(client))
			if !cacheCoreNil(value) || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(value, err, calls)
			}
		})
	}
	for _, bad := range []string{"callback", "name-success", "name-failure"} {
		t.Run(bad, func(t *testing.T) {
			calls := 0
			cause := errors.New("lookup failure")
			var client *gophercloud.ServiceClient
			client = deleteCoreClient(deleteCoreTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				client.MoreHeaders["X-IMAGE-CACHE-CLEAR-TARGET"] = "queue"
				if bad == "name-failure" {
					return nil, cause
				}
				return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(`{"images":[{"id":"id","name":"wanted"}]}`)), nil), nil
			}))
			client.MoreHeaders = map[string]string{}
			ref := resource.Name("wanted")
			var options []CacheOption
			if bad == "callback" {
				ref = resource.ID("id")
				options = []CacheOption{func(*CacheOpts) error { client.MoreHeaders[cacheTargetHeader] = "cache"; return nil }}
			}
			value, err := New(client).QueueImage(context.Background(), ref, options...)
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != map[string]int{"callback": 0, "name-success": 1, "name-failure": 1}[bad] {
				t.Fatal(value, err, calls)
			}
			if bad == "name-failure" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
	client := deleteCoreClient(deleteCoreTransport(func(*http.Request) (*http.Response, error) { t.Fatal("HTTP"); return nil, nil }))
	callbacks := 0
	client.Type = "compute"
	if value, err := New(client).CleanCache(context.Background(), func(*CacheOpts) error { callbacks++; return nil }); value != nil || !errors.Is(err, resource.ErrUnsupported) || callbacks != 0 {
		t.Fatal(value, err, callbacks)
	}
	var nilService *Service
	if value, err := nilService.GetImageCache(context.Background()); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(value, err)
	}
	client.Type = "image"
	if value, err := New(client).GetImageCache(nil); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(value, err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("canceled preflight")
	cancel(cause)
	if value, err := New(client).CleanCache(ctx); value != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(value, err)
	}
}

func TestImageCacheCoreAcknowledgementFailuresAndMissingPolicy(t *testing.T) {
	for _, name := range []string{"queue", "delete", "clear", "clean"} {
		t.Run(name, func(t *testing.T) {
			readErr, closeErr, ctxCause := errors.New("read"), errors.New("close"), errors.New("context")
			ctx, cancel := context.WithCancelCause(context.Background())
			var test cacheCoreCall
			for _, candidate := range cacheCoreCalls(ctx) {
				if candidate.name == name {
					test = candidate
				}
			}
			header := http.Header{"X-Proof": {"before"}}
			body := &deleteCoreBody{reader: deleteCoreReader(func(p []byte) (int, error) {
				copy(p, "partial\xff")
				header.Set("X-Proof", "after")
				cancel(ctxCause)
				return len("partial\xff"), readErr
			}), closeErr: closeErr}
			client := deleteCoreClient(deleteCoreTransport(func(*http.Request) (*http.Response, error) { return deleteCoreHTTP(test.status, body, header), nil }))
			client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				t.Fatal("accepted replay")
				return nil
			}
			value, err := test.call(New(client))
			ack, ok := value.(*ImageCacheAcknowledgement)
			var proof *resource.ResponseError
			if !ok || ack == nil || !errors.As(err, &proof) || ack.StatusCode != test.status || proof.StatusCode != test.status || ack.Header.Get("X-Proof") != "before" || string(ack.Body) != "partial\xff" || body.closes != 1 {
				t.Fatal(value, err, proof, body.closes)
			}
			for _, cause := range []error{readErr, closeErr, ctxCause, context.Canceled} {
				if !errors.Is(err, cause) {
					t.Fatal(err, cause)
				}
			}
			ack.Body[0] = 'X'
			ack.Header.Set("X-Proof", "ack")
			if string(proof.Body) != "partial\xff" || proof.Header.Get("X-Proof") != "before" {
				t.Fatal(proof)
			}
		})
	}
	for _, mode := range []string{"default", "strict", "read", "close", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New(mode)
			hooks := 0
			body := &deleteCoreBody{reader: strings.NewReader("missing")}
			if mode == "read" {
				body.reader = deleteCoreReader(func(p []byte) (int, error) { copy(p, "part"); return 4, cause })
			}
			if mode == "close" {
				body.closeErr = cause
			}
			if mode == "cancel" {
				body.reader = deleteCoreReader(func(p []byte) (int, error) { cancel(cause); return 0, io.EOF })
			}
			client := deleteCoreClient(deleteCoreTransport(func(*http.Request) (*http.Response, error) {
				return deleteCoreHTTP(404, body, http.Header{"X-Proof": {"404"}}), nil
			}))
			client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks++
				return cause
			}
			var options []CacheDeleteOption
			if mode == "strict" {
				options = []CacheDeleteOption{WithCacheDeleteIgnoreMissing(false)}
			}
			value, err := New(client).CacheDeleteImage(ctx, resource.ID("id"), options...)
			if value != nil || body.closes != 1 {
				t.Fatal(value, err, body.closes)
			}
			if mode == "default" {
				if err != nil || hooks != 0 {
					t.Fatal(err, hooks)
				}
			} else if mode == "strict" {
				if !gophercloud.ResponseCodeIs(err, 404) || !errors.Is(err, cause) || hooks != 1 {
					t.Fatal(err, hooks)
				}
			} else {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 404 || !errors.Is(err, cause) || hooks != 0 {
					t.Fatal(err, proof, hooks)
				}
			}
		})
	}
	calls := 0
	client := deleteCoreClient(deleteCoreTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(`{"images":[]}`)), nil), nil
	}))
	if value, err := New(client).CacheDeleteImage(context.Background(), resource.Name("absent")); value != nil || !errors.Is(err, resource.ErrNotFound) || calls != 1 {
		t.Fatal(value, err, calls)
	}
}

func TestImageCacheCoreCapturedSourceAndPrebodyGuards(t *testing.T) {
	calls := 0
	var client *gophercloud.ServiceClient
	client = deleteCoreClient(deleteCoreTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.EscapedPath() != "/reverse/glance/v2/cache/id" || r.Header.Get("X-Extra") != "captured" || r.Header.Get("X-Auth-Token") != "live" {
			t.Fatal(r.URL, r.Header)
		}
		return deleteCoreHTTP(202, io.NopCloser(strings.NewReader("")), nil), nil
	}))
	client.MoreHeaders = map[string]string{"X-Extra": "captured"}
	value, err := New(client).QueueImage(context.Background(), resource.ID("id"), func(*CacheOpts) error {
		client.ResourceBase = "https://example.test/changed/v2/"
		client.MoreHeaders["X-Extra"] = "later"
		client.ProviderClient.SetToken("live")
		return nil
	})
	if value == nil || err != nil || calls != 1 {
		t.Fatal(value, err, calls)
	}
	for _, mutation := range []string{"keep", "json-response", "json-null", "raw-body", "codes"} {
		t.Run(mutation, func(t *testing.T) {
			calls := 0
			closeErr := errors.New("close unexpected")
			body := &deleteCoreBody{reader: strings.NewReader("unexpected"), closeErr: closeErr}
			client := deleteCoreClient(deleteCoreTransport(func(*http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return deleteCoreHTTP(503, io.NopCloser(strings.NewReader("retry")), nil), nil
				}
				return deleteCoreHTTP(202, body, nil), nil
			}))
			client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				switch mutation {
				case "keep":
					opts.KeepResponseBody = false
				case "json-response":
					opts.JSONResponse = &struct{}{}
				case "json-null":
					opts.JSONBody = json.RawMessage(`null`)
				case "raw-body":
					opts.RawBody = strings.NewReader("new")
				case "codes":
					opts.OkCodes = []int{202}
				}
				return nil
			}
			value, err := New(client).CleanCache(context.Background())
			if value != nil {
				t.Fatal(value, err)
			}
			if mutation == "codes" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{200}) || !bytes.Equal(native.Body, []byte("unexpected")) || !errors.Is(err, closeErr) || calls != 2 || body.closes != 1 {
					t.Fatal(err, native, calls, body.closes)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || calls != 1 {
				t.Fatal(err, calls)
			}
		})
	}
}
