package objects

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"gophercloudsdk/resource"
)

func TestObjectReadOptionsSnapshots(t *testing.T) {
	newest := true
	date := time.Date(2024, 2, 3, 4, 5, 6, 0, time.FixedZone("offset", 9*3600))
	supplied := ObjectReadOpts{Headers: map[string]string{"x-call": "owned"}, Newest: &newest, IfModifiedSince: &date, Range: "literal", BufferSize: 1}
	full := WithObjectReadOpts(supplied)
	extra := map[string]string{"X-Extra": "snapshot"}
	overlay := WithObjectReadHeaders(extra)
	supplied.Headers["x-call"] = "caller"
	newest = false
	date = time.Time{}
	extra["X-Extra"] = "caller"
	var mu sync.Mutex
	calls := 0
	api, _ := readTestAPI(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("X-Call") != "owned" || r.Header.Get("X-Extra") != "snapshot" || r.Header.Get("X-Newest") != "true" || r.Header.Get("If-Modified-Since") != "Fri, 02 Feb 2024 19:05:06 GMT" {
			return nil, fmt.Errorf("factory input changed: %v", r.Header)
		}
		mu.Lock()
		calls++
		mu.Unlock()
		return readTestResponse(200, http.Header{}, io.NopCloser(strings.NewReader("ok"))), nil
	})
	var wg sync.WaitGroup
	failures := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := api.GetObject(context.Background(), "c", "o", full, overlay)
			if err != nil || result == nil || string(result.Body) != "ok" {
				failures <- fmt.Errorf("reused options result=%+v err=%v", result, err)
			}
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if calls != 2 {
		t.Fatalf("expected two independent requests, got %d", calls)
	}
}

func TestObjectReadOptionsHeaderAndDateInputs(t *testing.T) {
	date := time.Date(2024, 2, 3, 4, 5, 6, 0, time.FixedZone("offset", 9*3600))
	calls := 0
	api, _ := readTestAPI(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("X-Call") != "new" || r.Header.Get("X-Newest") != "false" || r.Header.Get("If-Match") != "opaque\tmatch" || r.Header.Get("If-None-Match") != "none" || r.Header.Get("Range") != "literal" {
			t.Errorf("header materialization: %v", r.Header)
		}
		if r.Header.Get("If-Modified-Since") != "" || r.Header.Get("If-Unmodified-Since") != "Fri, 02 Feb 2024 19:05:06 GMT" {
			t.Errorf("date clearing/UTC: %v", r.Header)
		}
		for key, value := range map[string]string{"filename": "한 ?#", "multipart-manifest": "get", "symlink": "get", "version-id": "v% ?#"} {
			if r.URL.Query().Get(key) != value {
				t.Errorf("query %s: %s", key, r.URL)
			}
		}
		return readTestResponse(200, http.Header{}, io.NopCloser(strings.NewReader("ok"))), nil
	})
	result, err := api.GetObject(context.Background(), "c", "o",
		WithObjectReadOpts(ObjectReadOpts{Headers: map[string]string{"x-call": "old"}}),
		WithObjectReadHeader("X-Call", "new"), WithObjectReadNewest(true), WithoutObjectReadNewest(), WithObjectReadNewest(false),
		WithObjectReadIfMatch("opaque\tmatch"), WithObjectReadIfNoneMatch("none"),
		WithObjectReadIfModifiedSince(date), WithoutObjectReadIfModifiedSince(),
		WithObjectReadIfUnmodifiedSince(time.Now()), WithoutObjectReadIfUnmodifiedSince(), WithObjectReadIfUnmodifiedSince(date),
		WithObjectReadRange("literal"), WithObjectReadFilename("한 ?#"), WithObjectReadMultipartManifest("get"), WithObjectReadSymlink("get"), WithObjectReadVersionID("v% ?#"), WithObjectReadBufferSize(1))
	if err != nil || result == nil || !result.Complete || calls != 1 {
		t.Fatalf("inputs: %+v %v calls=%d", result, err, calls)
	}
}

func TestObjectReadOptionsPreflight(t *testing.T) {
	for _, test := range []struct {
		name   string
		option ObjectReadOption
	}{
		{"nil", nil},
		{"negative-buffer", WithObjectReadBufferSize(-1)},
		{"large-buffer", WithObjectReadBufferSize(16*1024*1024 + 1)},
		{"query-control", WithObjectReadFilename("line\nname")},
		{"field-control", WithObjectReadIfMatch("bad\rvalue")},
		{"date-year-zero", WithObjectReadIfModifiedSince(time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC))},
		{"aliases", WithObjectReadHeaders(map[string]string{"x-call": "a", "X-Call": "b"})},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			api, _ := readTestAPI(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
			result, err := api.GetObject(context.Background(), "c", "o", test.option)
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatalf("preflight: %+v %v calls=%d", result, err, calls)
			}
		})
	}
	for _, header := range []string{"Range", "If-Range", "If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since", "X-Newest", "X-Service-Token", "Cookie", "X-Object-Meta-K", "X-Object-Sysmeta-K"} {
		t.Run("source/"+header, func(t *testing.T) {
			calls, callbacks := 0, 0
			api, source := readTestAPI(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
			source.MoreHeaders = map[string]string{header: "value"}
			option := func(*ObjectReadOpts) error { callbacks++; return nil }
			result, err := api.GetObject(context.Background(), "c", "o", option)
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
				t.Fatalf("source preflight: %+v %v HTTP=%d callbacks=%d", result, err, calls, callbacks)
			}
		})
	}
	t.Run("literal-name-and-typed-nil-writer", func(t *testing.T) {
		calls, callbacks := 0, 0
		api, _ := readTestAPI(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
		option := func(*ObjectReadOpts) error { callbacks++; return nil }
		for _, object := range []string{"", "../name", "name/..", "name\\bad", "bad\x7f", "bad\xff"} {
			result, err := api.GetObject(context.Background(), "c", object, option)
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("name %q: %+v %v", object, result, err)
			}
		}
		var writer *readTestWriter
		result, err := api.DownloadObject(context.Background(), "c", "o", writer, option)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
			t.Fatalf("writer preflight: %+v %v HTTP=%d callbacks=%d", result, err, calls, callbacks)
		}
	})
}

func TestObjectReadOptionsCallbacksAndDefaults(t *testing.T) {
	t.Run("callback-snapshots-full-replacement-default", func(t *testing.T) {
		calls, callbacks := 0, 0
		var retained *ObjectReadOpts
		body := &readTestBody{data: []byte("ok")}
		api, source := readTestAPI(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("X-Source") != "initial" || r.Header.Get("X-Call") != "owned" || r.Header.Get("X-Newest") != "" || r.Header.Get("If-Match") != "" {
				t.Errorf("snapshot/replacement: %v", r.Header)
			}
			return readTestResponse(200, http.Header{}, body), nil
		})
		source.MoreHeaders = map[string]string{"X-Source": "initial"}
		first := func(cfg *ObjectReadOpts) error {
			callbacks++
			if cfg.Headers == nil {
				return errors.New("Headers not initialized")
			}
			cfg.Headers["X-Call"] = "old"
			cfg.IfMatch = "discarded"
			return nil
		}
		last := func(cfg *ObjectReadOpts) error {
			callbacks++
			retained = cfg
			cfg.Headers["X-Call"] = "owned"
			source.MoreHeaders["X-Source"] = "later"
			return nil
		}
		tail := func(cfg *ObjectReadOpts) error {
			callbacks++
			retained.Headers["X-Call"] = "retained caller"
			return nil
		}
		result, err := api.GetObject(context.Background(), "c", "o", first, WithObjectReadOpts(ObjectReadOpts{}), last, tail)
		if err != nil || result == nil || string(result.Body) != "ok" || calls != 1 || callbacks != 3 || len(body.buffers) != 1 || body.buffers[0] != 32*1024 {
			t.Fatalf("callbacks: %+v %v", result, err)
		}
		retained.Headers["X-Call"] = "caller"
		if result.Metadata == nil || result.Header == nil {
			t.Fatal("result missing raw/projected observation")
		}
	})
	t.Run("cause-and-source-guard-joined-once", func(t *testing.T) {
		cause := errors.New("callback-fault")
		calls, callbacks := 0, 0
		api, source := readTestAPI(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
		option := func(cfg *ObjectReadOpts) error {
			callbacks++
			source.MoreHeaders = map[string]string{"Range": "injected"}
			return cause
		}
		result, err := api.GetObject(context.Background(), "c", "o", option)
		if result != nil || !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 1 || calls != 0 {
			t.Fatalf("callback/guard: %+v %v callbacks=%d HTTP=%d", result, err, callbacks, calls)
		}
	})
}
