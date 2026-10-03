package objects

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

func TestObjectDeleteOptionsSnapshots(t *testing.T) {
	c := objectMetadataClient()
	c.Endpoint = "http://swift.invalid/catalog/v1/a/"
	c.ResourceBase = "http://swift.invalid/reverse%2Fprefix/v1/a/"
	c.MoreHeaders = map[string]string{"X-Source": "captured"}
	headers := map[string]string{"x-call": "factory"}
	newest, ignore := false, true
	version := "version %2F ?# 한글"
	full := WithDeleteObjectOpts(DeleteObjectOpts{Headers: headers, Newest: &newest, IgnoreMissing: &ignore, VersionID: version})
	headers["x-call"], newest, ignore = "outside", true, false
	var retained *DeleteObjectOpts
	callbacks, calls := 0, 0
	option := func(cfg *DeleteObjectOpts) error {
		callbacks++
		retained = cfg
		c.MoreHeaders["X-Source"] = "valid later source"
		if cfg.Newest == nil || *cfg.Newest || cfg.IgnoreMissing == nil || !*cfg.IgnoreMissing || cfg.Headers["x-call"] != "factory" {
			t.Fatal("factory aliases", cfg)
		}
		return nil
	}
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.EscapedPath() != "/reverse%2Fprefix/v1/a/"+url.PathEscape(objectMetadataContainer)+"/"+url.PathEscape(objectMetadataName) || r.URL.Query().Get("version-id") != version || r.URL.Query().Get("multipart-manifest") != "" || r.Header.Get("X-Newest") != "false" || r.Header.Get("X-Call") != "final" || r.Header.Get("X-Source") != "captured" {
			t.Fatal("owned route/header lost", r.URL, r.Header)
		}
		if r.Method == "HEAD" {
			body := &objectMetadataBody{Reader: strings.NewReader(""), onClose: func() {
				retained.Headers["X-Call"] = "retained"
				*retained.Newest = true
				*retained.IgnoreMissing = false
				retained.StaticLargeObject = deleteCoreBool(true)
				retained.VersionID = "changed"
			}}
			return objectMetadataWire(r, 200, http.Header{}, body), nil
		}
		return objectMetadataWire(r, 404, http.Header{}, io.NopCloser(strings.NewReader("actual missing"))), nil
	})
	for n := 0; n < 2; n++ {
		c.MoreHeaders["X-Source"] = "captured"
		result, err := New(c).DeleteObject(context.Background(), objectMetadataContainer, objectMetadataName, full, option, WithDeleteObjectHeader("X-Call", "final"))
		if result == nil || err != nil || result.StaticLargeObject == nil || *result.StaticLargeObject || !result.IgnoredMissing || result.Discovery == nil || result.Deletion == nil || result.Deletion.StatusCode != 404 {
			t.Fatal(result, err)
		}
	}
	if callbacks != 2 || calls != 4 {
		t.Fatal("callbacks replayed or options changed", callbacks, calls)
	}
	for _, options := range [][]DeleteObjectOption{
		{WithDeleteObjectStaticLargeObject(true), WithoutDeleteObjectStaticLargeObject(), WithDeleteObjectNewest(true), WithoutDeleteObjectNewest(), WithDeleteObjectIgnoreMissing(false), WithoutDeleteObjectIgnoreMissing()},
		{WithDeleteObjectStaticLargeObject(true), WithDeleteObjectVersionID("discard"), WithDeleteObjectHeader("X-Call", "discard"), WithDeleteObjectOpts(DeleteObjectOpts{})},
	} {
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			if r.Method != "HEAD" || r.URL.RawQuery != "" || r.Header.Get("X-Newest") != "" || r.Header.Get("X-Call") != "" {
				t.Fatal("clear/full replacement failed", r.Method, r.URL, r.Header)
			}
			return objectMetadataWire(r, 404, http.Header{}, io.NopCloser(strings.NewReader("missing"))), nil
		})
		if result, err := New(c).DeleteObject(context.Background(), "box", "key", options...); result == nil || err != nil || !result.IgnoredMissing || result.StaticLargeObject != nil || result.Deletion != nil {
			t.Fatal(result, err)
		}
	}
}

func TestObjectDeleteOptionsPreflight(t *testing.T) {
	c := objectMetadataClient()
	calls := 0
	c.HTTPClient.Transport = objectMetadataTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
	for _, headers := range []map[string]string{{"X-Call": "one", "x-call": "two"}, {"X-K": "value"}, {"X-Call": "bad\nvalue"}, {"Cookie": "auth"}, {"X-Service-Token": "auth"}, {"X-Static-Large-Object": "true"}, {"X-Newest": "true"}, {"Authorization": "auth"}, {"X-Object-Meta-Field": "owned"}} {
		if result, err := New(c).DeleteObject(context.Background(), "box", "key", WithDeleteObjectHeaders(headers)); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("invalid headers accepted", headers, result, err)
		}
		c.MoreHeaders = headers
		callbacks := 0
		if result, err := New(c).DeleteObject(context.Background(), "box", "key", func(*DeleteObjectOpts) error { callbacks++; return nil }); result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
			t.Fatal("source preflight after callback", result, err, callbacks)
		}
		c.MoreHeaders = nil
	}
	for _, option := range []DeleteObjectOption{nil, WithDeleteObjectVersionID("\xff"), WithDeleteObjectVersionID("line\nversion")} {
		if result, err := New(c).DeleteObject(context.Background(), "box", "key", option); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(result, err)
		}
	}
	callbacks := 0
	if result, err := New(c).DeleteObject(nil, "box", "key", func(*DeleteObjectOpts) error { callbacks++; return nil }); result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
		t.Fatal(result, err, callbacks)
	}
	for _, names := range [][2]string{{"bad/box", "key"}, {"box", "../key"}, {"box", "bad\\key"}} {
		if result, err := New(c).DeleteObject(context.Background(), names[0], names[1], func(*DeleteObjectOpts) error { callbacks++; return nil }); result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
			t.Fatal(result, err, callbacks)
		}
	}
	if calls != 0 {
		t.Fatal("invalid inputs reached HTTP", calls)
	}
}

func TestObjectDeleteOptionsParallelReuse(t *testing.T) {
	c := objectMetadataClient()
	var calls atomic.Int32
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.Method != "DELETE" || r.Header.Get("X-Call") != "stable" || r.Header.Get("X-Newest") != "false" || r.URL.Query().Get("version-id") != "v% ?#" {
			t.Error("parallel factory changed", r.Method, r.URL, r.Header)
		}
		return objectMetadataWire(r, 204, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("ack"))), nil
	})
	known, newest := false, false
	headers := map[string]string{"X-Call": "stable"}
	option := WithDeleteObjectOpts(DeleteObjectOpts{Headers: headers, StaticLargeObject: &known, Newest: &newest, VersionID: "v% ?#"})
	headers["X-Call"], known, newest = "outside", true, true
	api := New(c)
	var workers sync.WaitGroup
	for n := 0; n < 6; n++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			result, err := api.DeleteObject(context.Background(), "box", "key", option)
			if result == nil || err != nil || result.Discovery != nil || result.StaticLargeObject == nil || *result.StaticLargeObject || string(result.Deletion.Body) != "ack" {
				t.Errorf("parallel delete failed %+v %v", result, err)
				return
			}
			result.Deletion.Body[0] = '!'
			result.Deletion.Header.Set("X-Proof", "changed")
		}()
	}
	workers.Wait()
	if calls.Load() != 6 {
		t.Fatal("known flag performed discovery", calls.Load())
	}
}

func TestObjectDeleteOptionsNativeRetrySnapshots(t *testing.T) {
	c := objectMetadataClient()
	c.MoreHeaders = map[string]string{"X-Source": "captured"}
	calls, retries, callbacks := 0, 0, 0
	c.RetryFunc = func(ctx context.Context, method, target string, options *gophercloud.RequestOpts, original error, count uint) error {
		retries++
		if method != "DELETE" || count != 1 || !gophercloud.ResponseCodeIs(original, 503) {
			t.Fatal("retry policy changed", method, target, original, count)
		}
		c.MoreHeaders["X-Source"] = "valid later source"
		c.SetToken("fresh")
		options.MoreHeaders = map[string]string{"Accept": "text/xml", "accept": "text/plain", "X-Native": "override"}
		return nil
	}
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "DELETE" || r.URL.Query().Get("multipart-manifest") != "delete" || r.URL.Query().Get("version-id") != "v% ?#" || r.Body != nil {
			t.Fatal("retry target/body changed", r.Method, r.URL)
		}
		accepts := 0
		for key, values := range r.Header {
			if strings.EqualFold(key, "Accept") {
				accepts++
				if len(values) != 1 || values[0] != "application/json" {
					t.Fatal("SLO Accept lost", r.Header)
				}
			}
		}
		if accepts != 1 {
			t.Fatal("Accept aliases survived", r.Header)
		}
		if calls == 1 {
			if r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Call") != "owned" || r.Header.Get("X-Auth-Token") != "token" {
				t.Fatal(r.Header)
			}
			return objectMetadataWire(r, 503, http.Header{}, io.NopCloser(strings.NewReader("retry"))), nil
		}
		if r.Header.Get("X-Source") != "" || r.Header.Get("X-Native") != "override" || r.Header.Get("X-Auth-Token") != "fresh" {
			t.Fatal("native headers/auth lost", r.Header)
		}
		return objectMetadataWire(r, 200, http.Header{}, io.NopCloser(strings.NewReader(deleteCoreBulkOK))), nil
	})
	result, err := New(c).DeleteObject(context.Background(), "box", "key", WithDeleteObjectStaticLargeObject(true), WithDeleteObjectVersionID("v% ?#"), WithDeleteObjectHeader("X-Call", "owned"), func(*DeleteObjectOpts) error { callbacks++; return nil })
	if result == nil || result.Bulk == nil || err != nil || calls != 2 || retries != 1 || callbacks != 1 {
		t.Fatal(result, err, calls, retries, callbacks)
	}
}
