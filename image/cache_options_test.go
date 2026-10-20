package image

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestImageCacheOptionsSnapshotReplacementAndLastWins(t *testing.T) {
	input := map[string]string{"X-First": "snapshot"}
	full := WithCacheOpts(CacheOpts{Headers: input})
	merge := WithCacheHeaders(input)
	input["X-First"] = "mutated"
	for _, option := range []CacheOption{full, merge} {
		value, err := parseCacheOptions([]CacheOption{WithCacheHeader("X-Discard", "old"), option, WithCacheHeader("x-first", "last")})
		if err != nil || value.Headers["X-First"] != "last" {
			t.Fatal(value, err)
		}
		again, err := parseCacheOptions([]CacheOption{option})
		if err != nil || again.Headers["X-First"] != "snapshot" {
			t.Fatal(again, err)
		}
	}
	value, err := parseCacheOptions([]CacheOption{WithCacheHeader("X-Discard", "old"), WithCacheOpts(CacheOpts{})})
	if err != nil || value.Headers == nil || len(value.Headers) != 0 {
		t.Fatal(value, err)
	}
	falseValue := false
	deleteFull := WithCacheDeleteOpts(CacheDeleteOpts{Headers: map[string]string{"X-A": "snapshot"}, IgnoreMissing: &falseValue})
	falseValue = true
	deleted, err := parseCacheDeleteOptions([]CacheDeleteOption{deleteFull, WithCacheDeleteHeader("x-a", "last"), WithCacheDeleteHeaders(map[string]string{"X-B": "merged"})})
	if err != nil || *deleted.IgnoreMissing || deleted.Headers["X-A"] != "last" || deleted.Headers["X-B"] != "merged" {
		t.Fatal(deleted, err)
	}
	deleted, err = parseCacheDeleteOptions([]CacheDeleteOption{WithCacheDeleteIgnoreMissing(false), WithCacheDeleteOpts(CacheDeleteOpts{})})
	if err != nil || deleted.IgnoreMissing == nil || !*deleted.IgnoreMissing || deleted.Headers == nil {
		t.Fatal(deleted, err)
	}
	clearInput := map[string]string{"X-A": "snapshot"}
	clearFull := WithClearCacheOpts(ClearCacheOpts{Headers: clearInput, Target: QueueOnly})
	clearInput["X-A"] = "changed"
	cleared, err := parseClearCacheOptions([]ClearCacheOption{clearFull, WithClearCacheHeader("x-a", "last"), WithClearCacheHeaders(map[string]string{"X-B": "merged"}), WithClearCacheTarget(CacheOnly)})
	if err != nil || cleared.Target != CacheOnly || cleared.Headers["X-A"] != "last" || cleared.Headers["X-B"] != "merged" {
		t.Fatal(cleared, err)
	}
	cleared, err = parseClearCacheOptions([]ClearCacheOption{WithClearCacheTarget(QueueOnly), WithClearCacheOpts(ClearCacheOpts{})})
	if err != nil || cleared.Target != CacheBoth || cleared.Headers == nil {
		t.Fatal(cleared, err)
	}
}

func TestImageCacheOptionsOwnPointersCallbacksAndSlices(t *testing.T) {
	var retained *CacheDeleteOpts
	secondCalls := 0
	options := make([]CacheDeleteOption, 2)
	options[0] = func(config *CacheDeleteOpts) error {
		if config.Headers == nil || config.IgnoreMissing == nil || !*config.IgnoreMissing {
			t.Fatal(config)
		}
		retained = config
		config.Headers["X-A"] = "owned"
		*config.IgnoreMissing = false
		options[1] = func(*CacheDeleteOpts) error { t.Fatal("changed caller slice executed"); return nil }
		return nil
	}
	options[1] = func(config *CacheDeleteOpts) error {
		secondCalls++
		retained.Headers["X-A"] = "late"
		*retained.IgnoreMissing = true
		if config.Headers["X-A"] != "owned" || *config.IgnoreMissing {
			t.Fatal(config)
		}
		retained = config
		return nil
	}
	client := deleteCoreClient(deleteCoreTransport(func(r *http.Request) (*http.Response, error) {
		retained.Headers["X-A"] = "transport"
		*retained.IgnoreMissing = true
		if r.Header.Get("X-A") != "owned" {
			t.Fatal(r.Header)
		}
		return deleteCoreHTTP(204, io.NopCloser(strings.NewReader("")), nil), nil
	}))
	value, err := New(client).CacheDeleteImage(context.Background(), resource.ID("id"), options...)
	if value == nil || err != nil || secondCalls != 1 {
		t.Fatal(value, err, secondCalls)
	}
	var old *ClearCacheOpts
	cleared, err := parseClearCacheOptions([]ClearCacheOption{func(config *ClearCacheOpts) error {
		old = config
		config.Target = QueueOnly
		config.Headers["X-A"] = "owned"
		return nil
	}, func(config *ClearCacheOpts) error {
		old.Target = CacheOnly
		old.Headers["X-A"] = "late"
		if config.Target != QueueOnly || config.Headers["X-A"] != "owned" {
			t.Fatal(config)
		}
		return nil
	}})
	if err != nil || cleared.Target != QueueOnly || cleared.Headers["X-A"] != "owned" {
		t.Fatal(cleared, err)
	}
	cause := errors.New("option cause")
	client = deleteCoreClient(deleteCoreTransport(func(*http.Request) (*http.Response, error) { t.Fatal("HTTP after option error"); return nil, nil }))
	if value, err := New(client).CleanCache(context.Background(), func(*CacheOpts) error { return cause }); value != nil || !errors.Is(err, cause) {
		t.Fatal(value, err)
	}
}

func TestImageCacheOptionsProtectedHeadersAndCompleteDefaults(t *testing.T) {
	for _, headers := range []map[string]string{{cacheTargetHeader: ""}, {"x-image-cache-clear-target": "cache"}, {"X-A": "one", "x-a": "two"}, {"Authorization": "secret"}, {"Content-Length": "1"}, {"Accept": "application/json"}, {"OpenStack-API-Version": "image 2.18"}, {"X-A": "bad\nvalue"}, {"Bad Key": "x"}} {
		t.Run(fmt.Sprint(headers), func(t *testing.T) {
			for _, apply := range []CacheOption{WithCacheOpts(CacheOpts{Headers: headers}), WithCacheHeaders(headers)} {
				if _, err := parseCacheOptions([]CacheOption{apply}); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			}
			if _, err := parseCacheDeleteOptions([]CacheDeleteOption{WithCacheDeleteHeaders(headers)}); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if _, err := parseClearCacheOptions([]ClearCacheOption{WithClearCacheHeaders(headers)}); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
	if _, err := parseCacheOptions([]CacheOption{nil}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := parseCacheDeleteOptions([]CacheDeleteOption{nil}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := parseClearCacheOptions([]ClearCacheOption{nil}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, target := range []CacheTarget{3, 255} {
		client := deleteCoreClient(deleteCoreTransport(func(*http.Request) (*http.Response, error) { t.Fatal("HTTP invalid target"); return nil, nil }))
		if value, err := New(client).ClearCache(context.Background(), WithClearCacheTarget(target)); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(value, err)
		}
	}
	value, err := parseCacheOptions([]CacheOption{WithCacheHeaders(map[string]string{"x-a": "same", "X-A": "same"}), WithCacheHeader("x-a", "last")})
	if err != nil || len(value.Headers) != 1 || value.Headers["X-A"] != "last" {
		t.Fatal(value, err)
	}
}

func TestImageCacheOptionsParallelReusableHelpers(t *testing.T) {
	input := map[string]string{"X-A": "snapshot"}
	missing := false
	common := []CacheOption{WithCacheOpts(CacheOpts{Headers: input}), WithCacheHeaders(input), WithCacheHeader("X-B", "last")}
	deleted := []CacheDeleteOption{WithCacheDeleteOpts(CacheDeleteOpts{Headers: input, IgnoreMissing: &missing}), WithCacheDeleteHeaders(input), WithCacheDeleteHeader("X-B", "last"), WithCacheDeleteIgnoreMissing(false)}
	cleared := []ClearCacheOption{WithClearCacheOpts(ClearCacheOpts{Headers: input, Target: CacheOnly}), WithClearCacheHeaders(input), WithClearCacheHeader("X-B", "last"), WithClearCacheTarget(QueueOnly)}
	input["X-A"] = "late"
	missing = true
	for i := 0; i < 12; i++ {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			a, e := parseCacheOptions(common)
			if e != nil || a.Headers["X-A"] != "snapshot" || a.Headers["X-B"] != "last" {
				t.Fatal(a, e)
			}
			b, e := parseCacheDeleteOptions(deleted)
			if e != nil || *b.IgnoreMissing || b.Headers["X-A"] != "snapshot" {
				t.Fatal(b, e)
			}
			c, e := parseClearCacheOptions(cleared)
			if e != nil || c.Target != QueueOnly || c.Headers["X-A"] != "snapshot" {
				t.Fatal(c, e)
			}
			a.Headers["X-A"] = "owned"
			b.Headers["X-A"] = "owned"
			*b.IgnoreMissing = true
			c.Headers["X-A"] = "owned"
		})
	}
}
