package objects

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

func TestDirectoryMarkerOptionsFactoryAndCallbackSnapshots(t *testing.T) {
	headers, metadata := map[string]string{"x-call": "factory", "Content-Type": "caller/type"}, map[string]string{"Tag": "factory"}
	full := WithDirectoryMarkerOpts(DirectoryMarkerOpts{Headers: headers, Metadata: metadata})
	pluralHeaders := WithDirectoryMarkerHeaders(headers)
	pluralMetadata := WithDirectoryMarkerMetadata(metadata)
	headers["x-call"], headers["Content-Type"], metadata["Tag"] = "outside", "bad\nmedia", "outside"
	callbacks := 0
	for n := 0; n < 2; n++ {
		c := objectMetadataClient()
		var retained *DirectoryMarkerOpts
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Call") != "overlay" || r.Header.Get("X-Object-Meta-Tag") != "overlay" || r.Header.Get("Content-Type") != "application/directory" {
				t.Error("factory or callback maps aliased", r.Header)
			}
			body := &objectMetadataBody{Reader: strings.NewReader("raw"), onClose: func() {
				retained.Headers["x-call"] = "retained"
				retained.Headers["X-Detect-Content-Type"] = "true"
				retained.Metadata["Tag"] = "retained"
			}}
			return objectMetadataWire(r, 201, http.Header{}, body), nil
		})
		result, err := New(c).CreateDirectoryMarkerObject(context.Background(), "box", "key", full, pluralHeaders, pluralMetadata, func(cfg *DirectoryMarkerOpts) error { callbacks++; retained = cfg; return nil }, WithDirectoryMarkerHeader("X-Call", "overlay"), WithDirectoryMarkerMetadataValue("tag", "overlay"))
		if err != nil || result == nil || result.Ordinary.Acknowledgement == nil {
			t.Fatal("retained maps changed completed operation", result, err)
		}
	}
	if callbacks != 2 || metadata["Tag"] != "outside" || headers["Content-Type"] != "bad\nmedia" {
		t.Fatal("factory callback replay or caller map mutation", callbacks, headers, metadata)
	}
}

func TestDirectoryMarkerOptionsReplacementAndOverlay(t *testing.T) {
	for _, reset := range []bool{false, true} {
		c := objectMetadataClient()
		calls := 0
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("X-Drop") != "" || r.Header.Get("X-Object-Meta-Drop") != "" || r.Header.Get("Content-Type") != "application/directory" {
				t.Error("full opts did not replace", r.Header)
			}
			if !reset && (r.Header.Get("X-Keep") != "last" || r.Header.Get("X-Object-Meta-Tag") != "last") {
				t.Error("canonical overlay failed", r.Header)
			}
			if reset && (r.Header.Get("X-Keep") != "" || r.Header.Get("X-Object-Meta-Tag") != "") {
				t.Error("empty full opts did not reset", r.Header)
			}
			for key := range r.Header {
				if key == "x-keep" || key == "x-object-meta-tag" {
					t.Error("overlay retained header alias", r.Header)
				}
			}
			return objectMetadataOK(r, 201, http.Header{}), nil
		})
		options := []DirectoryMarkerOption{
			WithDirectoryMarkerHeader("X-Drop", "earlier"), WithDirectoryMarkerMetadataValue("Drop", "earlier"),
			WithDirectoryMarkerOpts(DirectoryMarkerOpts{Headers: map[string]string{"x-keep": "first"}, Metadata: map[string]string{"TAG": "first"}}),
			WithDirectoryMarkerHeaders(map[string]string{"X-Keep": "last"}), WithDirectoryMarkerMetadata(map[string]string{"tag": "last"}),
		}
		if reset {
			options = append(options, WithDirectoryMarkerOpts(DirectoryMarkerOpts{}))
		}
		result, err := New(c).CreateDirectoryMarkerObject(context.Background(), "box", "key", options...)
		if err != nil || result == nil || calls != 1 {
			t.Fatal(result, err, calls)
		}
	}
}

func TestDirectoryMarkerOptionsPreflightAndJoinedCauses(t *testing.T) {
	c := objectMetadataClient()
	calls := 0
	c.HTTPClient.Transport = objectMetadataTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected wire") })
	for _, option := range []DirectoryMarkerOption{
		nil,
		WithDirectoryMarkerOpts(DirectoryMarkerOpts{Headers: map[string]string{"Content-Type": "one", "content-type": "two"}}),
		WithDirectoryMarkerHeader("Content-Type", "bad\nmedia"),
		WithDirectoryMarkerHeader("X-Detect-Content-Type", "false"),
		WithDirectoryMarkerHeader("Content-Length", "0"),
		WithDirectoryMarkerHeader("Cookie", "auth"),
		WithDirectoryMarkerHeader("X-Object-Meta-Tag", "raw metadata"),
		WithDirectoryMarkerHeader("X-Object-Manifest", "other/prefix"),
		WithDirectoryMarkerMetadataValue("X-Object-Meta-Tag", "prefixed"),
		WithDirectoryMarkerOpts(DirectoryMarkerOpts{Metadata: map[string]string{"Tag": "one", "tag": "two"}}),
	} {
		result, err := New(c).CreateDirectoryMarkerObject(context.Background(), "box", "key", option)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("invalid marker option reached wire", result, err)
		}
	}
	for _, headers := range []map[string]string{{"Content-Type": "one", "content-type": "two"}, {"Content-Type": "bad\nmedia"}, {"x-detect-content-type": "false"}} {
		c.MoreHeaders = headers
		callbacks := 0
		result, err := New(c).CreateDirectoryMarkerObject(context.Background(), "box", "key", func(*DirectoryMarkerOpts) error { callbacks++; return nil })
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
			t.Fatal("invalid original source ran callback", result, err, callbacks)
		}
	}
	c.MoreHeaders = nil
	for _, names := range [][2]string{{"", "key"}, {"box", ""}, {"box", "folder/../key"}} {
		callbacks := 0
		result, err := New(c).CreateDirectoryMarkerObject(context.Background(), names[0], names[1], func(*DirectoryMarkerOpts) error { callbacks++; return nil })
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
			t.Fatal("invalid target ran callback", result, err, callbacks)
		}
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause, canceled := errors.New("marker option cause"), errors.New("marker option canceled")
	result, err := New(c).CreateDirectoryMarkerObject(ctx, "box", "key", func(*DirectoryMarkerOpts) error {
		c.MoreHeaders = map[string]string{"X-Detect-Content-Type": "true"}
		cancel(canceled)
		return cause
	})
	if result != nil || !errors.Is(err, cause) || !errors.Is(err, canceled) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatal("callback/context/source causes lost", result, err, calls)
	}
	c.MoreHeaders = nil
	callbacks := 0
	result, err = New(c).CreateDirectoryMarkerObject(ctx, "box", "key", func(*DirectoryMarkerOpts) error { callbacks++; return nil })
	if result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, canceled) || callbacks != 0 || calls != 0 {
		t.Fatal("canceled source ran callback", result, err, callbacks, calls)
	}
}

func TestDirectoryMarkerOptionsParallelReuse(t *testing.T) {
	c := objectMetadataClient()
	var calls, callbacks atomic.Int32
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.Method != "PUT" || r.Header.Get("Content-Type") != "application/directory" || r.Header.Get("X-Call") != "factory" || r.Header.Get("X-Object-Meta-Tag") != "factory" {
			t.Error("parallel marker factory changed", r.Method, r.Header)
		}
		return objectMetadataWire(r, 201, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("raw"))), nil
	})
	headers, metadata := map[string]string{"X-Call": "factory"}, map[string]string{"Tag": "factory"}
	option := WithDirectoryMarkerOpts(DirectoryMarkerOpts{Headers: headers, Metadata: metadata})
	headers["X-Call"], metadata["Tag"] = "outside", "outside"
	api := New(c)
	var workers sync.WaitGroup
	for n := 0; n < 6; n++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			result, err := api.CreateDirectoryMarkerObject(context.Background(), "box", "key", option, func(*DirectoryMarkerOpts) error { callbacks.Add(1); return nil })
			if err != nil || result == nil || result.Ordinary.Acknowledgement == nil || len(result.Ordinary.Attempts) != 1 {
				t.Errorf("parallel marker %+v %v", result, err)
				return
			}
			result.Ordinary.Acknowledgement.Body[0] = '!'
			result.Ordinary.Acknowledgement.Header.Set("X-Proof", "changed")
			if string(result.Ordinary.Attempts[0].Response.Body) != "raw" || result.Ordinary.Attempts[0].Response.Header.Get("X-Proof") != "kept" {
				t.Error("parallel result proof aliases acknowledgement")
			}
		}()
	}
	workers.Wait()
	if calls.Load() != 6 || callbacks.Load() != 6 {
		t.Fatal("parallel factory/callback replay", calls.Load(), callbacks.Load())
	}
}
