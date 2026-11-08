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

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestObjectWaitOptionsSnapshots(t *testing.T) {
	headers := map[string]string{"x-call": "factory"}
	callbacks, optionCalls := 0, 0
	wait := []resource.WaitOption{resource.WithPollInterval(time.Nanosecond), resource.WithTimeout(time.Second), resource.WithFailureStates("FAIL"), resource.WithProgressCallback(func(progress int) error {
		callbacks++
		if progress != 0 {
			t.Error(progress)
		}
		return nil
	})}
	full := WithObjectWaitOpts(ObjectWaitOpts{Headers: headers, Wait: wait, StatusHeader: "X-State"})
	// Caller mutation of the Wait slice after the call has no effect.
	headers["x-call"], wait[1], wait[2] = "outside", resource.WithTimeout(time.Nanosecond), resource.WithFailureStates("BUILD")
	for n := 0; n < 2; n++ {
		c := objectMetadataClient()
		calls := 0
		var retained *ObjectWaitOpts
		c.HTTPClient.Transport = testcloud.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
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
				retained.Wait[2] = resource.WithFailureStates("BUILD")
				retained.Wait[1] = resource.WithTimeout(time.Nanosecond)
				retained.StatusHeader = "X-Foreign"
				retained.Wait = append(retained.Wait, resource.WithProgressCallback(func(int) error { return errors.New("retained callback") }))
			}}
			return objectMetadataWire(r, 200, http.Header{"X-State": {state}}, body), nil
		})
		result, err := New(c).WaitForStatus(context.Background(), "box", "key", "READY", full, request.Custom(func(cfg *ObjectWaitOpts) error { optionCalls++; retained = cfg; return nil }), request.Header("X-Call", "overlay"))
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
	for _, tc := range []struct {
		name          string
		options       []ObjectWaitOption
		header        string
		emptyFailures bool
		timeout       time.Duration
	}{
		{"header wins", []ObjectWaitOption{WithObjectWait(resource.WithStatusAttribute("name")), WithObjectWaitStatusHeader("x-state")}, "X-State", false, 0},
		{"attribute wins", []ObjectWaitOption{WithObjectWaitStatusHeader("X-State"), WithObjectWait(resource.WithStatusAttribute("etag"))}, "ETag", false, 0},
		{"empty factory disables", []ObjectWaitOption{WithObjectWait(resource.WithFailureStates()), WithObjectWaitStatusHeader("X-State")}, "X-State", true, 0},
		{"empty full disables", []ObjectWaitOption{WithObjectWaitOpts(ObjectWaitOpts{Wait: []resource.WaitOption{resource.WithFailureStates(), resource.WithUnlimitedWait()}, StatusHeader: "X-State"})}, "X-State", true, 0},
		{"nil full resets", []ObjectWaitOption{WithObjectWait(resource.WithFailureStates("FAIL")), WithObjectWaitOpts(ObjectWaitOpts{}), WithObjectWaitStatusHeader("X-State")}, "X-State", false, 0},
		{"unlimited clears bound", []ObjectWaitOption{WithObjectWait(resource.WithTimeout(time.Second)), WithObjectWait(resource.WithUnlimitedWait()), WithObjectWaitStatusHeader("X-State")}, "X-State", false, 0},
		{"bound restores unlimited", []ObjectWaitOption{WithObjectWait(resource.WithUnlimitedWait()), WithObjectWait(resource.WithTimeout(time.Second)), WithObjectWaitStatusHeader("X-State")}, "X-State", false, time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, err := api.prepareObjectWait(context.Background(), "box", "key", "READY", false, tc.options)
			if err != nil || w.header != tc.header || w.timeout != tc.timeout || w.failures == nil || (len(w.failures) == 0) != tc.emptyFailures {
				t.Fatal(w, err)
			}
			if !tc.emptyFailures && (len(w.failures) != 1 || w.failures[0] != "ERROR") {
				t.Fatal("nil did not restore ERROR", w.options)
			}
		})
	}
	values := []string{"FAIL"}
	factory := WithObjectWait(resource.WithFailureStates(values...))
	values[0] = "outside"
	w, err := api.prepareObjectWait(context.Background(), "box", "key", "READY", false, []ObjectWaitOption{factory, WithObjectWaitHeaders(map[string]string{"X-Call": "first"}), WithObjectWaitHeaders(map[string]string{"x-call": "last"}), WithObjectWaitStatusHeader("X-State")})
	if err != nil || w.failures[0] != "FAIL" || w.callback != nil || len(w.options.Headers) != 1 || w.options.Headers["X-Call"] != "last" {
		t.Fatal(w, err)
	}
	// Disabling failures permits an ERROR observation to remain nonterminal.
	calls, callbacks := 0, 0
	c.HTTPClient.Transport = testcloud.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		state := "ERROR"
		if calls == 2 {
			state = "READY"
		}
		return objectMetadataOK(r, 200, http.Header{"X-State": {state}}), nil
	})
	result, err := api.WaitForStatus(context.Background(), "box", "key", "READY", WithObjectWaitStatusHeader("X-State"), WithObjectWait(resource.WithFailureStates()), WithObjectWait(resource.WithPollInterval(time.Nanosecond)), WithObjectWait(resource.WithTimeout(time.Second)), WithObjectWait(resource.WithProgressCallback(func(int) error { callbacks++; return nil })))
	if err != nil || result == nil || !result.Complete || calls != 2 || callbacks != 1 {
		t.Fatal(result, err, calls, callbacks)
	}
}

func TestObjectWaitOptionsPreflightAndCauses(t *testing.T) {
	c := objectMetadataClient()
	calls := 0
	c.HTTPClient.Transport = testcloud.RoundTripperFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
	zero, negative := time.Duration(0), -time.Second
	for _, option := range []ObjectWaitOption{
		nil, WithObjectWait(resource.WithPollInterval(0)), WithObjectWait(resource.WithPollInterval(-1)), WithObjectWait(resource.WithTimeout(0)), WithObjectWait(resource.WithTimeout(-1)), WithObjectWaitOpts(ObjectWaitOpts{Wait: []resource.WaitOption{resource.WithPollInterval(zero)}}), WithObjectWaitOpts(ObjectWaitOpts{Wait: []resource.WaitOption{resource.WithTimeout(negative)}}),
		WithObjectWaitHeaders(map[string]string{"X-Call": "one", "x-call": "two"}), request.Header("Range", "bytes=0-0"), request.Header("if-match", "value"), request.Header("If-Unmodified-Since", "date"), request.Header("X-Newest", "true"), request.Header("Cookie", "auth"), request.Header("Content-Length", "9"),
		WithObjectWait(resource.WithStatusAttribute("nested.status")), WithObjectWaitStatusHeader("X-State bad"), WithObjectWait(resource.WithFailureStates("bad\u0085")), WithObjectWaitOpts(ObjectWaitOpts{Wait: []resource.WaitOption{resource.WithStatusAttribute("name")}, StatusHeader: "X-State"}), WithObjectWait(resource.WithStatusAttribute("name"), resource.WithProgressCallback(nil)),
	} {
		result, err := New(c).WaitForDelete(context.Background(), "box", "key", option)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("cheap invalid option reached waiter", result, err)
		}
	}
	for _, headers := range []map[string]string{{"Range": "bytes=0-0"}, {"If-None-Match": "*"}, {"X-Newest": "true"}} {
		c.MoreHeaders = headers
		callbacks := 0
		result, err := New(c).WaitForDelete(context.Background(), "box", "key", request.Custom(func(*ObjectWaitOpts) error { callbacks++; return nil }))
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
			t.Fatal("source preflight ran callback", result, err, callbacks)
		}
		c.MoreHeaders = nil
	}
	for _, target := range []string{"bad\xff", "bad\u0085"} {
		callbacks := 0
		result, err := New(c).WaitForStatus(context.Background(), "box", "key", target, request.Custom(func(*ObjectWaitOpts) error { callbacks++; return nil }))
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
			t.Fatal(result, err, callbacks)
		}
	}
	for _, ctx := range []context.Context{nil, func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }()} {
		callbacks := 0
		result, err := New(c).WaitForDelete(ctx, "box", "key", request.Custom(func(*ObjectWaitOpts) error { callbacks++; return nil }))
		if result != nil || err == nil || callbacks != 0 {
			t.Fatal(result, err, callbacks)
		}
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause, canceled := errors.New("option cause"), errors.New("option canceled")
	result, err := New(c).WaitForDelete(ctx, "box", "key", request.Custom(func(*ObjectWaitOpts) error {
		c.MoreHeaders = map[string]string{"If-Match": "foreign"}
		cancel(canceled)
		return cause
	}))
	if result != nil || !errors.Is(err, cause) || !errors.Is(err, canceled) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatal("preflight causes lost", result, err, calls)
	}
	c.MoreHeaders = nil
	c.HTTPClient.Transport = testcloud.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return objectMetadataOK(r, 404, http.Header{}), nil
	})
	result, err = New(c).WaitForDelete(context.Background(), "box", "key", WithObjectWait(resource.WithStatusAttribute("unknown but valid")))
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("malformed ignored selector accepted", result, err)
	}
	result, err = New(c).WaitForDelete(context.Background(), "box", "key", WithObjectWait(resource.WithStatusAttribute("unknown_but_valid")), WithObjectWait(resource.WithFailureStates("ERROR")))
	if err != nil || result == nil || !result.Deleted || result.Status != nil || calls != 1 {
		t.Fatal("Delete used status selectors", result, err, calls)
	}
}

func TestObjectWaitOptionsParallelReuse(t *testing.T) {
	c := objectMetadataClient()
	var calls, callbacks atomic.Int32
	c.HTTPClient.Transport = testcloud.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.Method != "HEAD" || r.Header.Get("X-Call") != "factory" {
			t.Error(r.Method, r.Header)
		}
		return objectMetadataWire(r, 200, http.Header{"X-State": {"READY"}, "X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("ack"))), nil
	})
	headers := map[string]string{"X-Call": "factory"}
	wait := []resource.WaitOption{resource.WithFailureStates("FAIL"), resource.WithPollInterval(time.Nanosecond), resource.WithTimeout(time.Second), resource.WithProgressCallback(func(int) error { callbacks.Add(1); return nil })}
	option := WithObjectWaitOpts(ObjectWaitOpts{Headers: headers, Wait: wait, StatusHeader: "X-State"})
	headers["X-Call"], wait[0], wait[2] = "outside", resource.WithFailureStates("READY"), resource.WithTimeout(time.Nanosecond)
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

func TestObjectWaitRepeatedCallsLaterWinsAndStructWaitIsSnapshot(t *testing.T) {
	api := New(objectMetadataClient())
	w, err := api.prepareObjectWait(context.Background(), "box", "key", "READY", false, []ObjectWaitOption{
		WithObjectWait(resource.WithTimeout(time.Second), resource.WithFailureStates("A")),
		WithObjectWaitStatusHeader("X-State"),
		WithObjectWait(resource.WithUnlimitedWait(), resource.WithPollInterval(time.Hour)),
		WithObjectWait(resource.WithFailureStates("B"), resource.WithPollInterval(time.Millisecond)),
	})
	if err != nil || w.timeout != 0 || len(w.failures) != 1 || w.failures[0] != "B" || w.header != "X-State" || w.pacer.Next() != time.Millisecond {
		t.Fatal(w, err)
	}
	wait := []resource.WaitOption{resource.WithTimeout(time.Second)}
	option := WithObjectWaitOpts(ObjectWaitOpts{Wait: wait, StatusHeader: "X-State"})
	wait[0] = resource.WithTimeout(time.Hour)
	w, err = api.prepareObjectWait(context.Background(), "box", "key", "READY", false, []ObjectWaitOption{option})
	if err != nil || w.timeout != time.Second {
		t.Fatal(w, err)
	}
	deletion, err := api.prepareObjectWait(context.Background(), "box", "key", "", true, []ObjectWaitOption{WithObjectWait(resource.WithTimeout(time.Minute)), WithObjectWait(resource.WithTimeout(3 * time.Second))})
	if err != nil || deletion.timeout != 3*time.Second || deletion.pacer.Next() != 2*time.Second {
		t.Fatal(deletion, err)
	}
}
