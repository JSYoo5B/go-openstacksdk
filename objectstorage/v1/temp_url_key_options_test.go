package v1

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

func TestTempURLKeyOptionsSnapshotsAndReplacement(t *testing.T) {
	t.Run("owned before and after callbacks across both phases", func(t *testing.T) {
		c := tempURLKeyTestClient()
		headers := map[string]string{"x-trace": "factory"}
		newest := false
		full := WithGetTempURLKeyOpts(GetTempURLKeyOpts{Container: "box", Headers: headers, Newest: &newest})
		headers["x-trace"] = "outside"
		newest = true
		var retained *GetTempURLKeyOpts
		callbacks, calls := 0, 0
		option := func(cfg *GetTempURLKeyOpts) error {
			callbacks++
			if cfg.Headers["x-trace"] != "factory" || cfg.Newest == nil || *cfg.Newest {
				t.Fatal("factory snapshot aliases", cfg)
			}
			retained = cfg
			return nil
		}
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("X-Trace") != "owned" || len(r.Header.Values("X-Trace")) != 1 || r.Header.Get("X-Newest") != "false" {
				t.Fatal("callback alias/last overwrite", r.Header)
			}
			h := http.Header{}
			b := &tempURLKeyTestBody{Reader: strings.NewReader("raw")}
			if calls%2 == 1 {
				b.onClose = func() {
					retained.Headers["x-trace"] = "retained"
					*retained.Newest = true
					retained.Container = "changed"
				}
			} else {
				h["X-Account-Meta-Temp-URL-Key-2"] = []string{"selected"}
			}
			return tempURLKeyTestWire(r, 204, h, b), nil
		})
		first, err := New(c).GetTempURLKey(context.Background(), full, option, WithGetTempURLKeyHeader("X-Trace", "owned"))
		if err != nil || first == nil || string(first.Key) != "selected" {
			t.Fatal(first, err)
		}
		second, err := New(c).GetTempURLKey(context.Background(), full, option, WithGetTempURLKeyHeader("X-Trace", "owned"))
		if err != nil || second == nil || string(second.Key) != "selected" || callbacks != 2 || calls != 4 {
			t.Fatal(second, err, callbacks, calls)
		}
		first.Key[0] = '!'
		first.Account.Body[0] = '!'
		first.Account.Metadata.Values["temp-url-key-2"] = "changed"
		first.Account.Header.Set("X-Test", "changed")
		if string(second.Key) != "selected" || string(second.Account.Body) != "raw" || second.Account.Metadata.Values["temp-url-key-2"] != "selected" || second.Account.Header.Get("X-Test") != "" {
			t.Fatal("two calls alias proofs")
		}
	})
	for _, tc := range []struct {
		name    string
		options []GetTempURLKeyOption
		newest  string
		present bool
	}{
		{"explicit false after clear", []GetTempURLKeyOption{WithGetTempURLKeyContainer("box"), WithGetTempURLKeyContainer(""), WithGetTempURLKeyNewest(true), WithoutGetTempURLKeyNewest(), WithGetTempURLKeyNewest(false)}, "false", true},
		{"clear only", []GetTempURLKeyOption{WithGetTempURLKeyNewest(true), WithoutGetTempURLKeyNewest()}, "", false},
		{"full replacement", []GetTempURLKeyOption{WithGetTempURLKeyContainer("box"), WithGetTempURLKeyHeaders(map[string]string{"X-Trace": "remove"}), WithGetTempURLKeyNewest(true), WithGetTempURLKeyOpts(GetTempURLKeyOpts{})}, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := tempURLKeyTestClient()
			calls := 0
			c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.String() != c.Endpoint || r.Header.Get("X-Trace") != "" || r.Header.Get("X-Newest") != tc.newest || (len(r.Header.Values("X-Newest")) == 1) != tc.present {
					t.Fatal("replacement/clear lost", r.URL, r.Header)
				}
				return tempURLKeyTestWire(r, 204, http.Header{}, io.NopCloser(strings.NewReader(""))), nil
			})
			if result, err := New(c).GetTempURLKey(context.Background(), tc.options...); result == nil || err != nil || calls != 1 || result.Container != nil {
				t.Fatal(result, err, calls)
			}
		})
	}
}

func TestTempURLKeyOptionsParallelReuseAndPreflight(t *testing.T) {
	c := tempURLKeyTestClient()
	var calls atomic.Int32
	c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.Method != "HEAD" || r.Header.Get("X-Trace") != "stable" || r.Header.Get("X-Newest") != "false" {
			t.Error("shared option mutated", r.Header)
		}
		h := http.Header{}
		if strings.HasSuffix(r.URL.EscapedPath(), "AUTH_x/") {
			h["X-Account-Meta-Temp-URL-Key-2"] = []string{"key"}
		}
		return tempURLKeyTestWire(r, 204, h, io.NopCloser(strings.NewReader("raw"))), nil
	})
	headers := map[string]string{"X-Trace": "stable"}
	option := WithGetTempURLKeyHeaders(headers)
	headers["X-Trace"] = "outside"
	var workers sync.WaitGroup
	for i := 0; i < 6; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			result, err := New(c).GetTempURLKey(context.Background(), option, WithGetTempURLKeyContainer("box"), WithGetTempURLKeyNewest(false))
			if err != nil || result == nil || string(result.Key) != "key" || !result.Secondary || result.FromContainer {
				t.Errorf("parallel discovery failed %#v %v", result, err)
				return
			}
			result.Key[0] = '!'
			if result.Account.Metadata.Values["temp-url-key-2"] != "key" {
				t.Error("selected bytes alias projection")
			}
		}()
	}
	workers.Wait()
	if calls.Load() != 12 {
		t.Fatal("parallel phase count", calls.Load())
	}
	for _, h := range []map[string]string{
		{"X-Trace": "a", "x-trace": "b"}, {"X-K": "value"}, {"X-Newest": "true"}, {"Content-Length": "0"}, {"Authorization": "secret"},
		{"X-Account-Meta-Temp-URL-Key": "secret"}, {"X-Container-Meta-Temp-URL-Key-2": "secret"}, {"X-Remove-Account-Meta-Book": ""}, {"X-Remove-Container-Meta-Book": ""}, {"X-Trace": "bad\nvalue"},
	} {
		if result, err := New(c).GetTempURLKey(context.Background(), WithGetTempURLKeyHeaders(h)); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid option accepted %#v %#v %v", h, result, err)
		}
	}
	if result, err := New(c).GetTempURLKey(context.Background(), nil); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err)
	}
	if calls.Load() != 12 {
		t.Fatal("invalid options sent HTTP", calls.Load())
	}
	c.MoreHeaders = map[string]string{"X-Newest": "true"}
	callbacks := 0
	if result, err := New(c).GetTempURLKey(context.Background(), func(*GetTempURLKeyOpts) error { callbacks++; return nil }); result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
		t.Fatal("source preflight after callback", result, err, callbacks)
	}
	if calls.Load() != 12 {
		t.Fatal("source preflight sent HTTP", calls.Load())
	}
}
