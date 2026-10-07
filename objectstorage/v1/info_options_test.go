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

	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestInfoOptionsFactoryAndCallbackOwnership(t *testing.T) {
	c := tempURLKeyTestClient()
	c.MoreHeaders = map[string]string{"X-Source": "captured"}
	headers := map[string]string{"x-trace": "factory"}
	size := int64(0)
	full := WithObjectSegmentSizeOpts(ObjectSegmentSizeOpts{Headers: headers, Size: &size})
	getFull := WithGetInfoOpts(GetInfoOpts{Headers: headers})
	headers["x-trace"], size = "outside", 90
	callbacks, calls := 0, 0
	var retained *ObjectSegmentSizeOpts
	option := func(cfg *ObjectSegmentSizeOpts) error {
		callbacks++
		if cfg.Headers["x-trace"] != "factory" || cfg.Size == nil || *cfg.Size != 0 {
			t.Fatal("factory aliases", cfg)
		}
		retained = cfg
		c.MoreHeaders["X-Source"] = "valid change"
		return nil
	}
	c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("X-Trace") != "owned" || len(r.Header.Values("X-Trace")) != 1 || r.Header.Get("X-Source") != "captured" {
			t.Fatal("request alias or source snapshot lost", r.Header)
		}
		body := &tempURLKeyTestBody{Reader: strings.NewReader(`{"swift":{"max_file_size":100}}`), onClose: func() { retained.Headers["x-trace"] = "retained"; *retained.Size = 99 }}
		return tempURLKeyTestWire(r, 200, http.Header{}, body), nil
	})
	first, err := New(c).GetObjectSegmentSize(context.Background(), full, option, WithObjectSegmentSizeHeader("X-Trace", "owned"))
	if err != nil || first == nil || first.RequestedSize != 0 || first.Size != 0 {
		t.Fatal(first, err)
	}
	c.MoreHeaders["X-Source"] = "captured"
	second, err := New(c).GetObjectSegmentSize(context.Background(), full, option, WithObjectSegmentSizeHeader("X-Trace", "owned"))
	if err != nil || second == nil || second.RequestedSize != 0 || second.Size != 0 || callbacks != 2 || calls != 2 {
		t.Fatal(second, err, callbacks, calls)
	}
	first.Info.Swift["max_file_size"][0] = '!'
	if string(second.Info.Swift["max_file_size"]) != "100" {
		t.Fatal("reused options alias responses")
	}
	c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("X-Trace") != "factory" {
			t.Fatal("GetInfo factory aliases", r.Header)
		}
		return tempURLKeyTestWire(r, 200, http.Header{}, io.NopCloser(strings.NewReader(`{}`))), nil
	})
	if result, err := New(c).GetInfo(context.Background(), getFull); result == nil || err != nil {
		t.Fatal(result, err)
	}
	for _, options := range [][]ObjectSegmentSizeOption{
		{WithObjectSegmentSize(0), WithoutObjectSegmentSize()},
		{WithObjectSegmentSize(0), WithObjectSegmentSizeHeaders(map[string]string{"X-Trace": "discarded"}), WithObjectSegmentSizeOpts(ObjectSegmentSizeOpts{})},
	} {
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Trace") != "" {
				t.Fatal("full replacement retained headers")
			}
			return tempURLKeyTestWire(r, 200, http.Header{}, io.NopCloser(strings.NewReader(`{"swift":{"max_file_size":2000000000}}`))), nil
		})
		if result, err := New(c).GetObjectSegmentSize(context.Background(), options...); result == nil || err != nil || result.RequestedSize != 1073741824 {
			t.Fatal(result, err)
		}
	}
}

func TestInfoOptionsHeaderValidationAndPreflight(t *testing.T) {
	c := tempURLKeyTestClient()
	calls := 0
	c.HTTPClient.Transport = tempURLKeyTestTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
	for _, headers := range []map[string]string{
		{"X-Trace": "one", "x-trace": "two"}, {"X-K": "value"}, {"X-Trace": "bad\nvalue"}, {"Authorization": "auth"}, {"X-Auth-Token": "auth"},
		{"Cookie": "auth"}, {"X-Service-Token": "auth"}, {"Host": "other"}, {"Content-Length": "0"}, {"X-Newest": "true"}, {"X-Account-Meta-Key": "value"},
	} {
		if result, err := New(c).GetInfo(context.Background(), WithGetInfoHeaders(headers)); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("invalid GetInfo headers accepted", headers, result, err)
		}
		if result, err := New(c).GetObjectSegmentSize(context.Background(), WithObjectSegmentSizeOpts(ObjectSegmentSizeOpts{Headers: headers})); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("invalid segment headers accepted", headers, result, err)
		}
		c.MoreHeaders = headers
		callbacks := 0
		if result, err := New(c).GetInfo(context.Background(), func(*GetInfoOpts) error { callbacks++; return nil }); result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
			t.Fatal("source validation ran after callbacks", result, err, callbacks)
		}
		c.MoreHeaders = nil
	}
	if result, err := New(c).GetInfo(context.Background(), nil); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err)
	}
	if result, err := New(c).GetObjectSegmentSize(context.Background(), nil); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err)
	}
	if result, err := New(c).GetObjectSegmentSize(context.Background(), WithObjectSegmentSize(-1)); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err)
	}
	callbacks := 0
	if result, err := New(c).GetInfo(nil, func(*GetInfoOpts) error { callbacks++; return nil }); result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
		t.Fatal(result, err, callbacks)
	}
	if calls != 0 {
		t.Fatal("invalid options sent HTTP", calls)
	}
}

func TestInfoOptionsParallelReuse(t *testing.T) {
	c := tempURLKeyTestClient()
	var calls atomic.Int32
	c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.Header.Get("X-Trace") != "stable" {
			t.Error("shared header option changed", r.Header)
		}
		return tempURLKeyTestWire(r, 200, http.Header{}, io.NopCloser(strings.NewReader(`{"swift":{"max_file_size":100},"slo":{}}`))), nil
	})
	headers := map[string]string{"x-trace": "stable"}
	get := WithGetInfoHeaders(headers)
	segment := WithObjectSegmentSizeHeaders(headers)
	headers["x-trace"] = "outside"
	var workers sync.WaitGroup
	for n := 0; n < 6; n++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			info, err := New(c).GetInfo(context.Background(), get)
			if err != nil || info == nil || string(info.Swift["max_file_size"]) != "100" {
				t.Errorf("GetInfo reuse failed %#v %v", info, err)
				return
			}
			result, err := New(c).GetObjectSegmentSize(context.Background(), segment, WithObjectSegmentSize(0))
			if err != nil || result == nil || result.Size != 0 {
				t.Errorf("segment reuse failed %#v %v", result, err)
				return
			}
			info.Swift["max_file_size"][0] = '!'
			if string(result.Info.Swift["max_file_size"]) != "100" {
				t.Error("concurrent calls alias results")
			}
		}()
	}
	workers.Wait()
	if calls.Load() != 12 {
		t.Fatal("parallel request count", calls.Load())
	}
}

func TestInfoOptionsNativeRetrySnapshots(t *testing.T) {
	c := tempURLKeyTestClient()
	calls, retries, callbacks := 0, 0, 0
	c.RetryFunc = func(ctx context.Context, method, target string, options *gophercloud.RequestOpts, original error, count uint) error {
		retries++
		if method != "GET" || !strings.HasSuffix(target, "/info") || count != 1 {
			t.Fatal("retry scope changed", method, target, count)
		}
		options.MoreHeaders["X-Trace"] = "native override"
		c.SetToken("fresh")
		return nil
	}
	c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			if r.Header.Get("X-Trace") != "owned" || r.Header.Get("X-Auth-Token") != "token" {
				t.Fatal("first attempt snapshot lost", r.Header)
			}
			return tempURLKeyTestWire(r, 503, http.Header{}, io.NopCloser(strings.NewReader("retry"))), nil
		}
		if r.Header.Get("X-Trace") != "native override" || r.Header.Get("X-Auth-Token") != "fresh" || r.Method != "GET" || r.Body != nil {
			t.Fatal("native headers/live auth lost", r.Header, r.Method, r.Body)
		}
		return tempURLKeyTestWire(r, 200, http.Header{}, io.NopCloser(strings.NewReader(`{}`))), nil
	})
	result, err := New(c).GetInfo(context.Background(), WithGetInfoHeader("X-Trace", "owned"), func(*GetInfoOpts) error { callbacks++; return nil })
	if result == nil || err != nil || calls != 2 || retries != 1 || callbacks != 1 {
		t.Fatal(result, err, calls, retries, callbacks)
	}
}
