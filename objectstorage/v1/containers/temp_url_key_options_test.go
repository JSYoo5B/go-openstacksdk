package containers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"gophercloudsdk/resource"
)

func TestContainerTempURLKeyOptionsSnapshotsAndReplacement(t *testing.T) {
	c := containerMetadataClient("http://cloud.invalid/v1/AUTH_x/")
	headers := map[string]string{"x-trace": "factory"}
	full := WithSetTempURLKeyOpts(SetTempURLKeyOpts{Headers: headers, Secondary: true})
	headers["x-trace"] = "outside"
	var retained *SetTempURLKeyOpts
	calls := 0
	c.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("X-Trace") != "owned" || len(r.Header.Values("X-Trace")) != 1 || len(r.Header.Values("X-Container-Meta-Temp-URL-Key-2")) != 1 {
			t.Fatal("snapshot/overwrite", r.Header)
		}
		return containerMetadataWire(r, 204, http.Header{}, io.NopCloser(strings.NewReader(""))), nil
	})
	option := func(cfg *SetTempURLKeyOpts) error {
		if cfg.Headers["x-trace"] != "factory" {
			t.Fatal("factory aliases", cfg.Headers)
		}
		cfg.Headers["X-Added"] = "yes"
		retained = cfg
		return nil
	}
	for i := 0; i < 2; i++ {
		if _, err := New(c).SetTempURLKey(context.Background(), "백업 %2F?#", "key", full, option, WithSetTempURLKeyHeader("X-Trace", "owned")); err != nil {
			t.Fatal(err)
		}
		retained.Headers["x-trace"] = "retained"
		retained.Secondary = false
	}
	if calls != 2 {
		t.Fatal(calls)
	}
	c.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
		if len(r.Header.Values("X-Container-Meta-Temp-URL-Key")) != 1 || len(r.Header.Values("X-Container-Meta-Temp-URL-Key-2")) != 0 || r.Header.Get("X-Trace") != "" {
			t.Fatal("FullOpts did not replace", r.Header)
		}
		return containerMetadataWire(r, 204, http.Header{}, io.NopCloser(strings.NewReader(""))), nil
	})
	if _, err := New(c).SetTempURLKey(context.Background(), "백업 %2F?#", "key", full, WithSetTempURLKeyOpts(SetTempURLKeyOpts{})); err != nil {
		t.Fatal(err)
	}
}

func TestContainerTempURLKeyOptionsParallelAndInvalidHeaders(t *testing.T) {
	c := containerMetadataClient("http://cloud.invalid/v1/AUTH_x/")
	var calls atomic.Int32
	c.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.Header.Get("X-Trace") != "stable" || len(r.Header.Values("X-Container-Meta-Temp-URL-Key-2")) != 1 {
			t.Error("shared options mutated", r.Header)
		}
		return containerMetadataWire(r, 204, http.Header{}, io.NopCloser(strings.NewReader(""))), nil
	})
	h := map[string]string{"X-Trace": "stable"}
	option := WithSetTempURLKeyHeaders(h)
	h["X-Trace"] = "outside"
	var workers sync.WaitGroup
	for i := 0; i < 6; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if _, err := New(c).SetTempURLKey(context.Background(), "백업 %2F?#", "key", option, WithSetTempURLKeySecondary(true)); err != nil {
				t.Error(err)
			}
		}()
	}
	workers.Wait()
	if calls.Load() != 6 {
		t.Fatal(calls.Load())
	}
	for _, headers := range []map[string]string{{"X-Trace": "a", "x-trace": "b"}, {"X-Account-Meta-K": "value"}, {"X-Container-Meta-Temp-URL-Key": "override"}, {"X-Trace": "bad\nvalue"}} {
		if result, err := New(c).SetTempURLKey(context.Background(), "백업 %2F?#", "key", WithSetTempURLKeyHeaders(headers)); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid header accepted %#v %v", headers, err)
		}
	}
	if result, err := New(c).SetTempURLKey(context.Background(), "백업 %2F?#", "key", nil); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err)
	}
	if calls.Load() != 6 {
		t.Fatal("invalid options reached wire", calls.Load())
	}
}
