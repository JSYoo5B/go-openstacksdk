package objects

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestObjectCreateOptionsFactorySnapshots(t *testing.T) {
	headers, metadata := map[string]string{"x-call": "factory"}, map[string]string{"TAG": "factory"}
	size, use, generate := int64(7), false, false
	full := WithCreateObjectOpts(CreateObjectOpts{Headers: headers, Metadata: metadata, SegmentSize: &size, UseSLO: &use, GenerateChecksums: &generate, MD5: strings.Repeat("A", 32)})
	headers["x-call"], metadata["TAG"], size, use, generate = "outside", "outside", 0, true, true
	c := objectMetadataClient()
	p, err := New(c).captureCreateObject(context.Background(), "box", "key")
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 2; n++ {
		var retained *CreateObjectOpts
		callbacks := 0
		cfg, err := p.applyCreateOptions(context.Background(), []CreateObjectOption{full, func(value *CreateObjectOpts) error { callbacks++; retained = value; return nil }, WithCreateObjectHeader("X-Call", "overlay"), WithCreateObjectMetadataValue("tag", "overlay")})
		if err != nil || cfg.SegmentSize == nil || *cfg.SegmentSize != 7 || cfg.UseSLO == nil || *cfg.UseSLO || cfg.GenerateChecksums == nil || *cfg.GenerateChecksums || cfg.Headers["X-Call"] != "overlay" || cfg.Metadata["tag"] != "overlay" || callbacks != 1 {
			t.Fatal(cfg, err, callbacks)
		}
		retained.Headers["x-call"] = "retained"
		retained.Metadata["TAG"] = "retained"
		*retained.SegmentSize = 99
		*retained.UseSLO = true
		*retained.GenerateChecksums = true
		if cfg.Headers["X-Call"] != "overlay" || cfg.Metadata["tag"] != "overlay" || *cfg.SegmentSize != 7 || *cfg.UseSLO || *cfg.GenerateChecksums {
			t.Fatal("retained callback aliases options", cfg)
		}
		cfg.Headers["X-Call"] = "changed"
		cfg.Metadata["tag"] = "changed"
		*cfg.SegmentSize = 88
	}
	data := []byte("before")
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != "before" {
			t.Error("Data snapshot occurred after callback", string(body), err)
		}
		return objectMetadataOK(r, 201, http.Header{}), nil
	})
	result, err := New(c).CreateObject(context.Background(), "box", "key", CreateObjectInput{Data: data}, func(*CreateObjectOpts) error { copy(data, "after!"); return nil })
	if err != nil || result == nil || result.Size != 6 {
		t.Fatal(result, err)
	}
}

func TestObjectCreateOptionsDefaultsAndReplacement(t *testing.T) {
	c := objectMetadataClient()
	p, err := New(c).captureCreateObject(context.Background(), "box", "key")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := p.applyCreateOptions(context.Background(), []CreateObjectOption{WithCreateObjectSegmentSize(0), WithCreateObjectUseSLO(false), WithCreateObjectGenerateChecksums(false), WithCreateObjectMD5(strings.Repeat("A", 32)), WithCreateObjectSHA256(strings.Repeat("B", 64)), WithCreateObjectHeaders(map[string]string{"X-Call": "first"}), WithCreateObjectHeaders(map[string]string{"x-call": "last"}), WithCreateObjectMetadata(map[string]string{"Tag": "first"}), WithCreateObjectMetadata(map[string]string{"TAG": "last"})})
	if err = p.finishCreate(context.Background(), &cfg, "file", err); err != nil || cfg.SegmentSize == nil || *cfg.SegmentSize != 0 || cfg.UseSLO == nil || *cfg.UseSLO || cfg.GenerateChecksums == nil || *cfg.GenerateChecksums || cfg.Headers["X-Call"] != "last" || cfg.Metadata["tag"] != "last" || len(cfg.Headers) != 1 || len(cfg.Metadata) != 1 {
		t.Fatal(cfg, err)
	}
	for _, options := range [][]CreateObjectOption{
		{WithCreateObjectSegmentSize(3), WithoutCreateObjectSegmentSize(), WithCreateObjectUseSLO(false), WithoutCreateObjectUseSLO(), WithCreateObjectGenerateChecksums(false), WithoutCreateObjectGenerateChecksums(), WithCreateObjectMD5(""), WithCreateObjectSHA256("")},
		{WithCreateObjectHeader("X-Call", "discard"), WithCreateObjectMetadataValue("tag", "discard"), WithCreateObjectSegmentSize(3), WithCreateObjectOpts(CreateObjectOpts{})},
	} {
		cfg, err = p.applyCreateOptions(context.Background(), options)
		if err != nil || cfg.SegmentSize != nil || cfg.UseSLO != nil || cfg.GenerateChecksums != nil || cfg.MD5 != "" || cfg.SHA256 != "" || len(cfg.Headers) != 0 || len(cfg.Metadata) != 0 {
			t.Fatal("default reset failed", cfg, err)
		}
	}
	for _, option := range []CreateObjectOption{WithCreateObjectHeaders(map[string]string{"X-Call": "one", "x-call": "two"}), WithCreateObjectMetadata(map[string]string{"Tag": "one", "tag": "two"})} {
		if _, err := p.applyCreateOptions(context.Background(), []CreateObjectOption{option}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("one-map aliases accepted", err)
		}
	}
}

func TestObjectCreateOptionsPreflightAndCallbackCauses(t *testing.T) {
	c := objectMetadataClient()
	calls := 0
	c.HTTPClient.Transport = objectMetadataTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
	var nilReader *bytes.Reader
	for _, input := range []CreateObjectInput{{}, {Data: []byte{}, Reader: strings.NewReader("other")}, {Reader: nilReader}, {Filename: "bad\x00file"}} {
		callbacks := 0
		result, err := New(c).CreateObject(context.Background(), "box", "key", input, func(*CreateObjectOpts) error { callbacks++; return nil })
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
			t.Fatal("invalid input reached callback", result, err, callbacks)
		}
	}
	for _, option := range []CreateObjectOption{nil, WithCreateObjectSegmentSize(-1), WithCreateObjectMD5("short"), WithCreateObjectSHA256(strings.Repeat("g", 64)), WithCreateObjectHeader("ETag", "owned"), WithCreateObjectHeader("Cookie", "auth"), WithCreateObjectHeader("X-Service-Token", "auth"), WithCreateObjectHeader("X-Copy-From", "/other/key"), WithCreateObjectMetadataValue("x-object-meta-tag", "bad"), WithCreateObjectMetadataValue("tag", "bad\nvalue")} {
		reader := &createCoreBorrowed{reader: strings.NewReader("unread")}
		result, err := New(c).CreateObject(context.Background(), "box", "key", CreateObjectInput{Reader: reader}, option)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || reader.reads != 0 {
			t.Fatal("cheap preflight consumed Reader", result, err, reader)
		}
	}
	if result, err := New(c).CreateObject(context.Background(), "box", "key", CreateObjectInput{Data: []byte{}}, WithCreateObjectGenerateChecksums(true)); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err)
	}
	for _, ctx := range []context.Context{nil, func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }()} {
		callbacks := 0
		result, err := New(c).CreateObject(ctx, "box", "key", CreateObjectInput{Data: []byte{}}, func(*CreateObjectOpts) error { callbacks++; return nil })
		if result != nil || err == nil || callbacks != 0 {
			t.Fatal(result, err, callbacks)
		}
	}
	cause := errors.New("option cause")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancelCause := errors.New("callback cancellation")
	result, err := New(c).CreateObject(ctx, "box", "key", CreateObjectInput{Reader: strings.NewReader("unread")}, func(*CreateObjectOpts) error {
		c.Endpoint = "http://changed.invalid/v1/a/"
		cancel(cancelCause)
		return cause
	})
	if result != nil || !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatal("callback causes lost", result, err, calls)
	}
}

func TestObjectCreateOptionsParallelReuse(t *testing.T) {
	c := objectMetadataClient()
	var calls atomic.Int32
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		data, err := io.ReadAll(r.Body)
		if err != nil || string(data) != "stable" || r.Header.Get("X-Call") != "factory" || r.Header.Get("X-Object-Meta-Tag") != "factory" {
			t.Error("parallel factory drift", string(data), r.Header, err)
		}
		return objectMetadataWire(r, 201, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("ack"))), nil
	})
	headers, metadata := map[string]string{"X-Call": "factory"}, map[string]string{"tag": "factory"}
	option := WithCreateObjectOpts(CreateObjectOpts{Headers: headers, Metadata: metadata})
	headers["X-Call"], metadata["tag"] = "outside", "outside"
	api := New(c)
	var workers sync.WaitGroup
	for n := 0; n < 6; n++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			result, err := api.CreateObject(context.Background(), "box", "key", CreateObjectInput{Data: []byte("stable")}, option)
			if err != nil || result == nil || result.Ordinary.Acknowledgement.StatusCode != 201 {
				t.Errorf("parallel Create %+v %v", result, err)
				return
			}
			result.Ordinary.Acknowledgement.Body[0] = '!'
			result.Ordinary.Acknowledgement.Header.Set("X-Proof", "changed")
			if string(result.Ordinary.Attempts[0].Response.Body) != "ack" || result.Ordinary.Attempts[0].Response.Header.Get("X-Proof") != "kept" {
				t.Error("acknowledgement aliases physical proof")
			}
		}()
	}
	workers.Wait()
	if calls.Load() != 6 {
		t.Fatal(calls.Load())
	}
}
