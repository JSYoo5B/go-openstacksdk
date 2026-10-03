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
	"time"

	"gophercloudsdk/resource"
)

func TestObjectWaitOptionsSnapshots(t *testing.T) {
	headers, states := map[string]string{"x-call": "factory"}, []string{"FAIL"}
	interval, timeout := time.Nanosecond, time.Second
	callbacks, optionCalls := 0, 0
	full := WithObjectWaitOpts(ObjectWaitOpts{Headers: headers, Interval: &interval, Timeout: &timeout, StatusHeader: "X-State", FailureStates: states, ProgressCallback: func(progress int) error {
		callbacks++
		if progress != 0 {
			t.Error(progress)
		}
		return nil
	}})
	headers["x-call"], states[0], interval, timeout = "outside", "BUILD", 0, -1
	for n := 0; n < 2; n++ {
		c := objectMetadataClient()
		calls := 0
		var retained *ObjectWaitOpts
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("X-Call") != "overlay" {
				t.Error("option headers drifted", r.Header)
			}
			state := "BUILD"
			if calls == 2 {
				state = "READY"
			}
			body := &objectMetadataBody{Reader: strings.NewReader("raw"), onClose: func() {
				retained.Headers["x-call"] = "retained"
				retained.FailureStates[0] = "BUILD"
				*retained.Interval = 0
				*retained.Timeout = -1
				retained.StatusHeader = "X-Foreign"
				retained.ProgressCallback = func(int) error { return errors.New("retained callback") }
			}}
			return objectMetadataWire(r, 200, http.Header{"X-State": {state}}, body), nil
		})
		result, err := New(c).WaitForStatus(context.Background(), "box", "key", "READY", full, func(cfg *ObjectWaitOpts) error { optionCalls++; retained = cfg; return nil }, WithObjectWaitHeader("X-Call", "overlay"))
		if err != nil || result == nil || !result.Complete || result.Polls != 2 || calls != 2 || result.Status == nil || *result.Status != "READY" {
			t.Fatal("retained callback/factory aliased waiter", result, err, calls)
		}
	}
	if callbacks != 2 || optionCalls != 2 {
		t.Fatal("option or progress callback replayed", callbacks, optionCalls)
	}
}

func TestObjectWaitOptionsDefaultsSelectorsAndResets(t *testing.T) {
	c := objectMetadataClient()
	api := New(c)
	zero := time.Duration(0)
	for _, tc := range []struct {
		name          string
		options       []ObjectWaitOption
		header        string
		emptyFailures bool
		timeout       time.Duration
	}{
		{"header wins", []ObjectWaitOption{WithObjectWaitStatusAttribute("name"), WithObjectWaitStatusHeader("x-state")}, "X-State", false, 0},
		{"attribute wins", []ObjectWaitOption{WithObjectWaitStatusHeader("X-State"), WithObjectWaitStatusAttribute("etag")}, "ETag", false, 0},
		{"empty factory disables", []ObjectWaitOption{WithObjectWaitFailureStates(), WithObjectWaitStatusHeader("X-State")}, "X-State", true, 0},
		{"empty full disables", []ObjectWaitOption{WithObjectWaitOpts(ObjectWaitOpts{FailureStates: []string{}, StatusHeader: "X-State", Timeout: &zero})}, "X-State", true, 0},
		{"nil full resets", []ObjectWaitOption{WithObjectWaitFailureStates("FAIL"), WithObjectWaitOpts(ObjectWaitOpts{}), WithObjectWaitStatusHeader("X-State")}, "X-State", false, 0},
		{"unlimited clears bound", []ObjectWaitOption{WithObjectWaitTimeout(time.Second), WithObjectWaitUnlimitedWait(), WithObjectWaitStatusHeader("X-State")}, "X-State", false, 0},
		{"bound restores unlimited", []ObjectWaitOption{WithObjectWaitUnlimitedWait(), WithObjectWaitTimeout(time.Second), WithObjectWaitStatusHeader("X-State")}, "X-State", false, time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, err := api.prepareObjectWait(context.Background(), "box", "key", "READY", false, tc.options)
			if err != nil || w.header != tc.header || w.timeout != tc.timeout || w.options.FailureStates == nil || (len(w.options.FailureStates) == 0) != tc.emptyFailures {
				t.Fatal(w, err)
			}
			if !tc.emptyFailures && (len(w.options.FailureStates) != 1 || w.options.FailureStates[0] != "ERROR") {
				t.Fatal("nil did not restore ERROR", w.options)
			}
		})
	}
	values := []string{"FAIL"}
	factory := WithObjectWaitFailureStates(values...)
	values[0] = "outside"
	w, err := api.prepareObjectWait(context.Background(), "box", "key", "READY", false, []ObjectWaitOption{factory, WithObjectWaitHeaders(map[string]string{"X-Call": "first"}), WithObjectWaitHeaders(map[string]string{"x-call": "last"}), WithObjectWaitStatusHeader("X-State"), WithObjectWaitProgressCallback(func(int) error { return errors.New("removed") }), WithoutObjectWaitProgressCallback()})
	if err != nil || w.options.FailureStates[0] != "FAIL" || w.options.ProgressCallback != nil || len(w.options.Headers) != 1 || w.options.Headers["X-Call"] != "last" {
		t.Fatal(w, err)
	}
	// Disabling failures permits an ERROR observation to remain nonterminal.
	calls, callbacks := 0, 0
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		state := "ERROR"
		if calls == 2 {
			state = "READY"
		}
		return objectMetadataOK(r, 200, http.Header{"X-State": {state}}), nil
	})
	result, err := api.WaitForStatus(context.Background(), "box", "key", "READY", WithObjectWaitStatusHeader("X-State"), WithObjectWaitFailureStates(), WithObjectWaitPollInterval(time.Nanosecond), WithObjectWaitTimeout(time.Second), WithObjectWaitProgressCallback(func(int) error { callbacks++; return nil }))
	if err != nil || result == nil || !result.Complete || calls != 2 || callbacks != 1 {
		t.Fatal(result, err, calls, callbacks)
	}
}

func TestObjectWaitOptionsPreflightAndCauses(t *testing.T) {
	c := objectMetadataClient()
	calls := 0
	c.HTTPClient.Transport = objectMetadataTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
	zero, negative := time.Duration(0), -time.Second
	for _, option := range []ObjectWaitOption{
		nil, WithObjectWaitPollInterval(0), WithObjectWaitPollInterval(-1), WithObjectWaitTimeout(0), WithObjectWaitTimeout(-1), WithObjectWaitOpts(ObjectWaitOpts{Interval: &zero}), WithObjectWaitOpts(ObjectWaitOpts{Timeout: &negative}),
		WithObjectWaitHeaders(map[string]string{"X-Call": "one", "x-call": "two"}), WithObjectWaitHeader("Range", "bytes=0-0"), WithObjectWaitHeader("if-match", "value"), WithObjectWaitHeader("If-Unmodified-Since", "date"), WithObjectWaitHeader("X-Newest", "true"), WithObjectWaitHeader("Cookie", "auth"), WithObjectWaitHeader("Content-Length", "9"),
		WithObjectWaitStatusAttribute("nested.status"), WithObjectWaitStatusHeader("X-State bad"), WithObjectWaitFailureStates("bad\u0085"), WithObjectWaitOpts(ObjectWaitOpts{StatusAttribute: "name", StatusHeader: "X-State"}),
	} {
		result, err := New(c).WaitForDelete(context.Background(), "box", "key", option)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("cheap invalid option reached waiter", result, err)
		}
	}
	for _, headers := range []map[string]string{{"Range": "bytes=0-0"}, {"If-None-Match": "*"}, {"X-Newest": "true"}} {
		c.MoreHeaders = headers
		callbacks := 0
		result, err := New(c).WaitForDelete(context.Background(), "box", "key", func(*ObjectWaitOpts) error { callbacks++; return nil })
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
			t.Fatal("source preflight ran callback", result, err, callbacks)
		}
		c.MoreHeaders = nil
	}
	for _, target := range []string{"bad\xff", "bad\u0085"} {
		callbacks := 0
		result, err := New(c).WaitForStatus(context.Background(), "box", "key", target, func(*ObjectWaitOpts) error { callbacks++; return nil })
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
			t.Fatal(result, err, callbacks)
		}
	}
	for _, ctx := range []context.Context{nil, func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }()} {
		callbacks := 0
		result, err := New(c).WaitForDelete(ctx, "box", "key", func(*ObjectWaitOpts) error { callbacks++; return nil })
		if result != nil || err == nil || callbacks != 0 {
			t.Fatal(result, err, callbacks)
		}
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause, canceled := errors.New("option cause"), errors.New("option canceled")
	result, err := New(c).WaitForDelete(ctx, "box", "key", func(*ObjectWaitOpts) error {
		c.MoreHeaders = map[string]string{"If-Match": "foreign"}
		cancel(canceled)
		return cause
	})
	if result != nil || !errors.Is(err, cause) || !errors.Is(err, canceled) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatal("preflight causes lost", result, err, calls)
	}
	c.MoreHeaders = nil
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return objectMetadataOK(r, 404, http.Header{}), nil
	})
	result, err = New(c).WaitForDelete(context.Background(), "box", "key", WithObjectWaitStatusAttribute("unknown but valid"))
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("malformed ignored selector accepted", result, err)
	}
	result, err = New(c).WaitForDelete(context.Background(), "box", "key", WithObjectWaitStatusAttribute("unknown_but_valid"), WithObjectWaitFailureStates("ERROR"))
	if err != nil || result == nil || !result.Deleted || result.Status != nil || calls != 1 {
		t.Fatal("Delete used status selectors", result, err, calls)
	}
}

func TestObjectWaitOptionsParallelReuse(t *testing.T) {
	c := objectMetadataClient()
	var calls, callbacks atomic.Int32
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.Method != "HEAD" || r.Header.Get("X-Call") != "factory" {
			t.Error(r.Method, r.Header)
		}
		return objectMetadataWire(r, 200, http.Header{"X-State": {"READY"}, "X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("ack"))), nil
	})
	headers, failures := map[string]string{"X-Call": "factory"}, []string{"FAIL"}
	interval, timeout := time.Nanosecond, time.Second
	option := WithObjectWaitOpts(ObjectWaitOpts{Headers: headers, FailureStates: failures, StatusHeader: "X-State", Interval: &interval, Timeout: &timeout, ProgressCallback: func(int) error { callbacks.Add(1); return nil }})
	headers["X-Call"], failures[0], interval, timeout = "outside", "READY", 0, -1
	api := New(c)
	var workers sync.WaitGroup
	for n := 0; n < 6; n++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			result, err := api.WaitForStatus(context.Background(), "box", "key", "READY", option)
			if err != nil || result == nil || !result.Complete || result.Polls != 1 || result.Status == nil || *result.Status != "READY" {
				t.Errorf("parallel waiter %+v %v", result, err)
				return
			}
			result.Last.Acknowledgement.Body[0] = '!'
			result.Last.Acknowledgement.Header.Set("X-Proof", "changed")
			if string(result.Last.Attempts[0].Response.Body) != "ack" || result.Last.Attempts[0].Response.Header.Get("X-Proof") != "kept" {
				t.Error("parallel acknowledgement aliases attempt")
			}
		}()
	}
	workers.Wait()
	if calls.Load() != 6 || callbacks.Load() != 0 {
		t.Fatal(calls.Load(), callbacks.Load())
	}
}
