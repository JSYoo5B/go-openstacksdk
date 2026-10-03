package objects

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	swift "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1"
	"gophercloudsdk/resource"
)

func TestObjectMetadataOptionsFactorySnapshots(t *testing.T) {
	c := objectMetadataClient()
	flag := false
	headers := map[string]string{"x-call": "old"}
	option := WithGetMetadataOpts(GetMetadataOpts{Headers: headers, Newest: &flag})
	headers["x-call"] = "changed"
	flag = true
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("X-Call") != "new" || r.Header.Get("X-Newest") != "false" {
			t.Errorf("factory/overwrite %#v", r.Header)
		}
		return objectMetadataOK(r, 200, http.Header{}), nil
	})
	if _, err := New(c).GetMetadata(context.Background(), "container", "object", option, WithGetMetadataHeader("X-CaLl", "new")); err != nil {
		t.Fatal(err)
	}
	input := map[string]string{"Key": "  책%20\t "}
	writeHeaders := map[string]string{"x-call": "factory"}
	moption := WithMetadataOpts(MetadataOpts{Headers: writeHeaders})
	writeHeaders["x-call"] = "changed"
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method == "HEAD" {
			return objectMetadataOK(r, 200, http.Header{"X-Object-Meta-Keep": {"before"}}), nil
		}
		if r.Header.Get("X-Object-Meta-Key") != "  책%20\t " || r.Header.Get("X-Object-Meta-Keep") != "before" || r.Header.Get("X-Call") != "new" {
			t.Errorf("owned input %#v", r.Header)
		}
		return objectMetadataOK(r, 202, http.Header{}), nil
	})
	if _, err := New(c).SetMetadata(context.Background(), "container", "object", input, moption, WithMetadataHeader("X-Call", "new"), func(p *MetadataOpts) error { input["Key"] = "changed"; return nil }); err != nil {
		t.Fatal(err)
	}
	keys := []string{"Key"}
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method == "HEAD" {
			return objectMetadataOK(r, 200, http.Header{"X-Object-Meta-Key": {"before"}, "X-Object-Meta-Keep": {"safe"}}), nil
		}
		if _, ok := r.Header["X-Object-Meta-Key"]; ok || r.Header.Get("X-Object-Meta-Keep") != "safe" {
			t.Errorf("delete slice alias %#v", r.Header)
		}
		return objectMetadataOK(r, 202, http.Header{}), nil
	})
	if _, err := New(c).DeleteMetadata(context.Background(), "container", "object", keys, func(*MetadataOpts) error { keys[0] = "Keep"; return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestObjectMetadataOptionsReplacementAndRetainedHandles(t *testing.T) {
	c := objectMetadataClient()
	var first, last *GetMetadataOpts
	invoked := 0
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("X-Call") != "owned" || r.Header.Get("X-Newest") != "false" {
			t.Errorf("retained handle %#v", r.Header)
		}
		return objectMetadataOK(r, 200, http.Header{}), nil
	})
	_, err := New(c).GetMetadata(context.Background(), "container", "object", func(p *GetMetadataOpts) error {
		invoked++
		if p.Headers == nil {
			t.Fatal("nil initialized map")
		}
		p.Headers["X-Call"] = "owned"
		v := false
		p.Newest = &v
		first = p
		return nil
	}, func(p *GetMetadataOpts) error {
		invoked++
		first.Headers["X-Call"] = "retained"
		*first.Newest = true
		last = p
		return nil
	})
	if err != nil || invoked != 2 {
		t.Fatalf("callbacks%d %v", invoked, err)
	}
	last.Headers["X-Call"] = "after"
	*last.Newest = true
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("X-Old") != "" || len(r.Header.Values("X-Newest")) != 0 || r.Header.Get("X-New") != "final" {
			t.Errorf("replacement %#v", r.Header)
		}
		return objectMetadataOK(r, 200, http.Header{}), nil
	})
	if _, err := New(c).GetMetadata(context.Background(), "container", "object", WithGetMetadataHeader("X-Old", "gone"), WithGetMetadataNewest(true), WithGetMetadataOpts(GetMetadataOpts{}), WithGetMetadataHeaders(map[string]string{"X-New": "plural"}), WithGetMetadataHeader("x-new", "final"), WithGetMetadataNewest(false), WithoutGetMetadataNewest()); err != nil {
		t.Fatal(err)
	}
	var retained *MetadataOpts
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		code := 200
		if r.Method == "POST" {
			code = 202
			if r.Header.Get("X-Old") != "" || r.Header.Get("X-New") != "final" {
				t.Errorf("mutation replacement %#v", r.Header)
			}
		}
		return objectMetadataOK(r, code, http.Header{}), nil
	})
	if _, err := New(c).SetMetadata(context.Background(), "container", "object", nil, WithMetadataHeader("X-Old", "gone"), WithMetadataOpts(MetadataOpts{}), WithMetadataHeaders(map[string]string{"X-New": "plural"}), func(p *MetadataOpts) error { retained = p; return nil }, WithMetadataHeader("x-new", "final"), func(*MetadataOpts) error { retained.Headers["X-New"] = "retained"; return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestObjectMetadataOptionsCompletePreflightAndCauses(t *testing.T) {
	c := objectMetadataClient()
	var requests atomic.Int32
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		return objectMetadataOK(r, 200, http.Header{}), nil
	})
	a := New(c)
	for _, name := range []string{"", "/", ".", "..", "bad\\name", "line\n", string([]byte{255})} {
		invoked := 0
		got, err := a.GetMetadata(context.Background(), name, "object", func(*GetMetadataOpts) error { invoked++; return nil })
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || invoked != 0 {
			t.Fatalf("container%q %#v %v callbacks%d", name, got, err, invoked)
		}
	}
	var native swift.ErrEmptyContainerName
	_, err := a.GetMetadata(context.Background(), "", "object")
	if !errors.As(err, &native) {
		t.Fatalf("lost native container cause %v", err)
	}
	for _, name := range []string{"", ".", "..", "a/../b", "a/./b", "\\", "\x7f", string([]byte{255})} {
		got, err := a.GetMetadata(context.Background(), "container", name)
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("object%q %#v %v", name, got, err)
		}
	}
	for _, m := range []map[string]string{{"Key": "a", "key": "b"}, {"X-Object-Meta-Key": "bad"}, {"": "bad"}, {"K": "bad"}, {"Key": "line\n"}, {"Key": string([]byte{255})}} {
		got, err := a.SetMetadata(context.Background(), "container", "object", m)
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("input%#v %#v %v", m, got, err)
		}
	}
	for _, keys := range [][]string{{"Key", "KEY"}, {""}, {"X-Object-Meta-Key"}, {"K"}} {
		got, err := a.DeleteMetadata(context.Background(), "container", "object", keys)
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("keys%#v %#v %v", keys, got, err)
		}
	}
	for _, key := range []string{"Authorization", "X-Auth-Token", "Content-Length", "X-Newest", "X-Object-Meta-Key", "X-Remove-Object-Meta-Key", "X-Symlink-Target", "X-Object-Sysmeta-Key", "X-Object-Transient-Sysmeta-Key"} {
		got, err := a.SetMetadata(context.Background(), "container", "object", nil, WithMetadataHeader(key, "bad"))
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("reserved%s %#v %v", key, got, err)
		}
	}
	for _, h := range []map[string]string{{"X-Call": "a", "x-call": "b"}, {"X-K": "bad"}, {"X-Call": "\n"}} {
		if _, err := a.GetMetadata(context.Background(), "container", "object", WithGetMetadataHeaders(h)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("bad plural %v", err)
		}
	}
	if _, err := a.GetMetadata(nil, "container", "object"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := a.GetMetadata(context.Background(), "container", "object", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("custom cancel")
	optionErr := errors.New("option failed")
	_, err = a.SetMetadata(ctx, "container", "object", nil, func(*MetadataOpts) error {
		cancel(cause)
		c.ResourceBase = "http://swift.invalid/other/"
		return optionErr
	})
	for _, e := range []error{context.Canceled, cause, optionErr, resource.ErrInvalidOption} {
		if !errors.Is(err, e) {
			t.Fatalf("lost cause%v: %v", e, err)
		}
	}
	if requests.Load() != 0 {
		t.Fatalf("preflight issued%d HTTP", requests.Load())
	}
}

func TestObjectMetadataOptionsParallelReuseAndHeaderBoundaries(t *testing.T) {
	c := objectMetadataClient()
	c.MoreHeaders = map[string]string{"X-Call": "source", "Content-Type": "application/source"}
	var calls atomic.Int32
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		code := 200
		if r.Method == "HEAD" {
			if r.URL.RawQuery == "" {
				if r.Header.Get("X-Call") != "get" || r.Header.Get("X-Newest") != "false" {
					t.Errorf("getter option %#v", r.Header)
				}
			} else if r.Header.Get("X-Call") != "source" || r.Header.Get("X-Newest") != "" {
				t.Errorf("write headers leaked read %#v", r.Header)
			}
			return objectMetadataOK(r, code, http.Header{"Content-Type": {"application/observed"}}), nil
		}
		if r.Header.Get("X-Call") != "write" || r.Header.Get("Content-Type") != "application/write" || r.Header.Get("X-Newest") != "" {
			t.Errorf("source/write precedence %#v", r.Header)
		}
		return objectMetadataOK(r, 202, http.Header{}), nil
	})
	get := WithGetMetadataOpts(GetMetadataOpts{Headers: map[string]string{"X-Call": "get"}, Newest: new(bool)})
	write := WithMetadataHeaders(map[string]string{"X-Call": "write", "Content-Type": "application/write"})
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := New(c).GetMetadata(context.Background(), "container", "object", get); err != nil {
				t.Error(err)
			}
			if _, err := New(c).SetMetadata(context.Background(), "container", "object", nil, write); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 36 || c.MoreHeaders["X-Call"] != "source" || c.MoreHeaders["Content-Type"] != "application/source" {
		t.Fatalf("reuse calls%d source%#v", calls.Load(), c.MoreHeaders)
	}
	for _, configure := range []func(*gophercloud.ServiceClient){func(c *gophercloud.ServiceClient) { c.Type = "image" }, func(c *gophercloud.ServiceClient) { c.ResourceBase = "http://foreign.invalid/base/" }, func(c *gophercloud.ServiceClient) { c.Endpoint += "?query=x" }, func(c *gophercloud.ServiceClient) { c.MoreHeaders = map[string]string{"X-Symlink-Target": "reserved"} }} {
		c := objectMetadataClient()
		configure(c)
		invoked := false
		_, err := New(c).GetMetadata(context.Background(), "container", "object", func(*GetMetadataOpts) error { invoked = true; return nil })
		if err == nil || invoked {
			t.Fatalf("source preflight %v callback%v", err, invoked)
		}
	}
}
