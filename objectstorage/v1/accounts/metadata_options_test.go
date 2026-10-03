package accounts

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

func TestAccountMetadataOptionsFactorySnapshots(t *testing.T) {
	prepared, err := metadataInput(map[string]string{"Literal": "  책%20\t "})()
	if err != nil || prepared["X-Account-Meta-Literal"] != "  책%20\t " {
		t.Fatalf("SDK trimmed input %#v %v", prepared, err)
	}
	flag := false
	headers := map[string]string{"X-Trace": "factory"}
	option := WithGetMetadataOpts(GetMetadataOpts{Headers: headers, Newest: &flag})
	headers["X-Trace"] = "changed"
	flag = true
	c := accountMetadataClient("http://account.invalid/v1/AUTH_x/")
	c.HTTPClient.Transport = accountMetadataTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("X-Trace") != "factory" || r.Header.Get("X-Newest") != "false" {
			t.Errorf("factory alias %#v", r.Header)
		}
		return accountMetadataWire(r, 204, http.Header{}, io.NopCloser(strings.NewReader(""))), nil
	})
	if _, err := New(c).GetMetadata(context.Background(), option); err != nil {
		t.Fatal(err)
	}
	metadata := map[string]string{"Key": "before"}
	keys := []string{"Key"}
	c.HTTPClient.Transport = accountMetadataTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("X-Account-Meta-Key") != "before" {
			t.Errorf("input alias %#v", r.Header)
		}
		return accountMetadataWire(r, 204, http.Header{}, io.NopCloser(strings.NewReader(""))), nil
	})
	if _, err := New(c).SetMetadata(context.Background(), metadata, func(*MetadataOpts) error { metadata["Key"] = "changed"; return nil }); err != nil {
		t.Fatal(err)
	}
	c.HTTPClient.Transport = accountMetadataTransport(func(r *http.Request) (*http.Response, error) {
		if _, ok := r.Header["X-Account-Meta-Key"]; !ok {
			t.Errorf("keys alias %#v", r.Header)
		}
		return accountMetadataWire(r, 204, http.Header{}, io.NopCloser(strings.NewReader(""))), nil
	})
	if _, err := New(c).DeleteMetadata(context.Background(), keys, func(*MetadataOpts) error { keys[0] = "changed"; return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestAccountMetadataOptionsReplacementAndNewest(t *testing.T) {
	for _, mixed := range []string{"X-Call", "x-CaLl"} {
		get, err := applyGetMetadataOptions([]GetMetadataOption{WithGetMetadataOpts(GetMetadataOpts{Headers: map[string]string{"x-call": "old"}}), WithGetMetadataHeader(mixed, "new")})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := validateMetadataHeaders(get.Headers); err != nil || len(get.Headers) != 1 || get.Headers["X-Call"] != "new" {
			t.Fatalf("Get singular overwrite %#v %v", get.Headers, err)
		}
		mutation, err := applyMetadataOptions([]MetadataOption{WithMetadataOpts(MetadataOpts{Headers: map[string]string{"x-call": "old"}}), WithMetadataHeader(mixed, "new")})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := validateMetadataHeaders(mutation.Headers); err != nil || len(mutation.Headers) != 1 || mutation.Headers["X-Call"] != "new" {
			t.Fatalf("mutation singular overwrite %#v %v", mutation.Headers, err)
		}
	}
	cfg, err := applyGetMetadataOptions([]GetMetadataOption{WithGetMetadataNewest(true), WithGetMetadataHeader("X-Trace", "one"), WithGetMetadataOpts(GetMetadataOpts{}), WithGetMetadataHeaders(map[string]string{"x-trace": "two"}), WithGetMetadataHeader("X-Trace", "three")})
	if err != nil || cfg.Newest != nil || cfg.Headers["X-Trace"] != "three" {
		t.Fatalf("replace/last wins %+v %v", cfg, err)
	}
	cfg, err = applyGetMetadataOptions([]GetMetadataOption{WithGetMetadataNewest(false), WithoutGetMetadataNewest()})
	if err != nil || cfg.Newest != nil {
		t.Fatalf("clear %+v %v", cfg, err)
	}
	m, err := applyMetadataOptions([]MetadataOption{WithMetadataHeader("X-Old", "one"), WithMetadataOpts(MetadataOpts{}), WithMetadataHeaders(map[string]string{"X-Trace": "new"})})
	if err != nil || len(m.Headers) != 1 || m.Headers["X-Trace"] != "new" {
		t.Fatalf("mutation replace %+v %v", m, err)
	}
	for _, values := range []map[string]string{{"X-Trace": "same", "x-trace": "same"}, {"X-Newest": "true"}, {"X-Trace": "bad\nvalue"}} {
		if _, err := applyGetMetadataOptions([]GetMetadataOption{WithGetMetadataHeaders(values)}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("plural aliases/invalid headers %v", err)
		}
	}
}

func TestAccountMetadataOptionsRetainedCallbacksAndCauses(t *testing.T) {
	var first *GetMetadataOpts
	var last *GetMetadataOpts
	calls := 0
	cfg, err := applyGetMetadataOptions([]GetMetadataOption{func(p *GetMetadataOpts) error {
		calls++
		if p.Headers == nil {
			t.Fatal("uninitialized headers")
		}
		p.Headers["X-Trace"] = "owned"
		value := false
		p.Newest = &value
		first = p
		return nil
	}, func(p *GetMetadataOpts) error {
		calls++
		first.Headers["X-Trace"] = "retained"
		*first.Newest = true
		first.Headers = nil
		last = p
		return nil
	}})
	last.Headers["X-Trace"] = "late"
	*last.Newest = true
	if err != nil || calls != 2 || cfg.Headers["X-Trace"] != "owned" || cfg.Newest == nil || *cfg.Newest {
		t.Fatalf("retained config mutated %+v %v", cfg, err)
	}
	c := accountMetadataClient("http://account.invalid/v1/AUTH_x/")
	a := New(c)
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("cancel")
	optionErr := errors.New("option")
	_, err = a.GetMetadata(ctx, func(*GetMetadataOpts) error { cancel(cause); return optionErr })
	if !errors.Is(err, optionErr) || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatalf("joined causes %v", err)
	}
}

func TestAccountMetadataOptionsParallelReuseAndSourceHeaders(t *testing.T) {
	c := accountMetadataClient("http://account.invalid/v1/AUTH_x/")
	c.MoreHeaders = map[string]string{"X-Trace": "source", "Content-Type": "text/plain"}
	var calls atomic.Int32
	c.HTTPClient.Transport = accountMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.Header.Get("X-Trace") != "option" || r.Header.Get("Content-Type") != "text/plain" {
			t.Errorf("ordinary headers %#v", r.Header)
		}
		return accountMetadataWire(r, 204, http.Header{}, io.NopCloser(strings.NewReader(""))), nil
	})
	get := WithGetMetadataHeaders(map[string]string{"X-Trace": "option"})
	mutation := WithMetadataHeaders(map[string]string{"X-Trace": "option"})
	a := New(c)
	var wg sync.WaitGroup
	fail := make(chan error, 12)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := a.GetMetadata(context.Background(), get); err != nil {
				fail <- err
			}
			if _, err := a.SetMetadata(context.Background(), nil, mutation); err != nil {
				fail <- err
			}
		}()
	}
	wg.Wait()
	close(fail)
	for err := range fail {
		t.Error(err)
	}
	if calls.Load() != 12 || c.MoreHeaders["X-Trace"] != "source" {
		t.Fatalf("reuse changed source %d %#v", calls.Load(), c.MoreHeaders)
	}
}
