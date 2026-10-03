package objects

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

func TestObjectStaleOptionsSnapshotsAndReplacement(t *testing.T) {
	md := strings.Repeat("A", 32)
	headers := map[string]string{"x-call": "factory"}
	full := WithIsObjectStaleOpts(IsObjectStaleOpts{Headers: headers, MD5: md})
	headers["x-call"] = "outside"
	c := objectMetadataClient()
	c.MoreHeaders = map[string]string{"X-Source": "captured"}
	callbacks, calls := 0, 0
	var retained *IsObjectStaleOpts
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "HEAD" || r.Header.Get("X-Call") != "overlay" || r.Header.Get("X-Source") != "captured" {
			t.Error("stale snapshot changed", r.Method, r.Header)
		}
		retained.Headers["x-call"] = "retained"
		retained.MD5 = "changed"
		return objectMetadataOK(r, 200, http.Header{"X-Object-Meta-X-Sdk-Md5": {md}}), nil
	})
	for n := 0; n < 2; n++ {
		c.MoreHeaders["X-Source"] = "captured"
		result, err := New(c).IsObjectStale(context.Background(), "box", "key", "unread explicit file", full, func(cfg *IsObjectStaleOpts) error {
			callbacks++
			retained = cfg
			c.MoreHeaders["X-Source"] = "valid later source"
			return nil
		}, WithIsObjectStaleHeader("X-Call", "overlay"))
		if err != nil || result == nil || result.Stale == nil || *result.Stale || result.MD5 != md {
			t.Fatal(result, err)
		}
	}
	if callbacks != 2 || calls != 2 {
		t.Fatal(callbacks, calls)
	}
	p := createUploadPrepared(t, c)
	for _, options := range [][]IsObjectStaleOption{
		{WithIsObjectStaleHeader("X-Call", "first"), WithIsObjectStaleHeaders(map[string]string{"x-call": "last"}), WithIsObjectStaleMD5(md), WithIsObjectStaleSHA256(strings.Repeat("B", 64)), WithIsObjectStaleMD5(""), WithIsObjectStaleSHA256("")},
		{WithIsObjectStaleHeader("X-Call", "discard"), WithIsObjectStaleMD5(md), WithIsObjectStaleOpts(IsObjectStaleOpts{})},
	} {
		cfg, err := p.applyStaleOptions(context.Background(), options)
		if err != nil || cfg.MD5 != "" || cfg.SHA256 != "" {
			t.Fatal(cfg, err)
		}
		if len(cfg.Headers) != 0 && (len(cfg.Headers) != 1 || cfg.Headers["X-Call"] != "last") {
			t.Fatal("header overlay/full replacement", cfg)
		}
	}
}

func TestObjectStaleOptionsPreflightAndCauses(t *testing.T) {
	c := objectMetadataClient()
	calls := 0
	c.HTTPClient.Transport = objectMetadataTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
	for _, filename := range []string{"", "bad\x00file", "bad\xfffile"} {
		callbacks := 0
		result, err := New(c).IsObjectStale(context.Background(), "box", "key", filename, func(*IsObjectStaleOpts) error { callbacks++; return nil })
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
			t.Fatal(result, err, callbacks)
		}
	}
	for _, option := range []IsObjectStaleOption{nil, WithIsObjectStaleMD5("bad"), WithIsObjectStaleSHA256(strings.Repeat("g", 64)), WithIsObjectStaleHeaders(map[string]string{"X-Call": "one", "x-call": "two"}), WithIsObjectStaleHeader("Cookie", "auth"), WithIsObjectStaleHeader("X-Service-Token", "auth"), WithIsObjectStaleHeader("ETag", "owned"), WithIsObjectStaleHeader("Content-Length", "9"), WithIsObjectStaleHeader("X-Object-Meta-Tag", "owned")} {
		if result, err := New(c).IsObjectStale(context.Background(), "box", "key", "explicit missing", option); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(result, err)
		}
	}
	callbacks := 0
	for _, ctx := range []context.Context{nil, func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }()} {
		if result, err := New(c).IsObjectStale(ctx, "box", "key", "explicit", func(*IsObjectStaleOpts) error { callbacks++; return nil }); result != nil || err == nil || callbacks != 0 {
			t.Fatal(result, err, callbacks)
		}
	}
	cause := errors.New("stale callback cause")
	ctx, cancel := context.WithCancelCause(context.Background())
	canceled := errors.New("stale canceled")
	result, err := New(c).IsObjectStale(ctx, "box", "key", "explicit", func(*IsObjectStaleOpts) error {
		c.Endpoint = "http://changed.invalid/v1/a/"
		cancel(canceled)
		return cause
	})
	if result != nil || !errors.Is(err, cause) || !errors.Is(err, canceled) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatal("stale callback causes lost", result, err, calls)
	}
}

func TestObjectStaleOptionsNativeSnapshotAndProof(t *testing.T) {
	c := objectMetadataClient()
	c.MoreHeaders = map[string]string{"X-Source": "captured"}
	md := strings.Repeat("a", 32)
	calls, retries, callbacks := 0, 0, 0
	c.RetryFunc = func(ctx context.Context, method, target string, opts *gophercloud.RequestOpts, original error, count uint) error {
		retries++
		if method != "HEAD" || count != 1 || !gophercloud.ResponseCodeIs(original, 503) {
			t.Error("native stale retry", method, original, count)
		}
		if native, ok := original.(gophercloud.ErrUnexpectedResponseCode); ok {
			native.Body[0] = '!'
			native.ResponseHeader.Set("X-Proof", "callback mutated")
		}
		c.MoreHeaders["X-Source"] = "valid later source"
		c.SetToken("fresh")
		opts.MoreHeaders = map[string]string{"X-Native": "override"}
		return nil
	}
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "HEAD" || r.Body != nil {
			t.Error("stale retry body", r.Method, r.Body)
		}
		if calls == 1 {
			if r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Call") != "owned" {
				t.Error(r.Header)
			}
			return objectMetadataWire(r, 503, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("rejected"))), nil
		}
		if r.Header.Get("X-Native") != "override" || r.Header.Get("X-Auth-Token") != "fresh" || r.Header.Get("X-Source") != "" {
			t.Error("native header/auth policy lost", r.Header)
		}
		return objectMetadataOK(r, 200, http.Header{"X-Object-Meta-X-Sdk-Md5": {md}}), nil
	})
	result, err := New(c).IsObjectStale(context.Background(), "box", "key", "explicit unread", WithIsObjectStaleMD5(md), WithIsObjectStaleHeader("X-Call", "owned"), func(*IsObjectStaleOpts) error { callbacks++; return nil })
	if err != nil || result == nil || result.Stale == nil || *result.Stale || calls != 2 || retries != 1 || callbacks != 1 {
		t.Fatal(result, err, calls, retries, callbacks)
	}
	// The private physical history exposes the callback-clone boundary directly.
	c = objectMetadataClient()
	calls = 0
	c.RetryFunc = func(ctx context.Context, method, target string, opts *gophercloud.RequestOpts, original error, count uint) error {
		native := original.(gophercloud.ErrUnexpectedResponseCode)
		native.Body[0] = '!'
		native.ResponseHeader.Set("X-Proof", "changed")
		return nil
	}
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		code := 503
		if calls == 2 {
			code = 200
		}
		return objectMetadataWire(r, code, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("rejected"))), nil
	})
	p := createUploadPrepared(t, c)
	out := p.exchange(context.Background(), "HEAD", p.metadata.target, "discovery", nil, nil, 1, 200, 204, 404)
	if out.err != nil || len(out.phase.Attempts) != 2 {
		t.Fatal(out)
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(out.phase.Attempts[0].Error, &native) || string(native.Body) != "rejected" || native.ResponseHeader.Get("X-Proof") != "kept" || string(out.phase.Attempts[0].Response.Body) != "rejected" {
		t.Fatal("callback mutated retained native proof", out.phase.Attempts[0], native)
	}
}
