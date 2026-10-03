package objects

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

func TestObjectWaitCoreFreshPollingAndCallbacks(t *testing.T) {
	for _, deletion := range []bool{true, false} {
		c := objectMetadataClient()
		c.ResourceBase = "http://swift.invalid/reverse%2Fprefix/v1/a/"
		c.MoreHeaders = map[string]string{"X-Source": "captured"}
		calls, callbacks := 0, 0
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			want := "/reverse%2Fprefix/v1/a/" + url.PathEscape(objectMetadataContainer) + "/" + url.PathEscape(objectMetadataName)
			if r.Method != "HEAD" || r.Body != nil || r.URL.EscapedPath() != want || r.URL.RawQuery != "" || r.Header.Get("X-Source") != "captured" {
				t.Error("poll scope changed", r.Method, r.URL, r.Header)
			}
			if calls > 1 && r.Header.Get("X-Auth-Token") != "fresh" {
				t.Error("poll used stale auth", r.Header)
			}
			code := 200
			state := "BUILD"
			if calls == 2 {
				code = 204
			}
			if deletion && calls == 3 {
				code = 404
			}
			if !deletion && calls == 2 {
				state = "rEaDy"
			}
			h := http.Header{"X-Proof": {"kept"}, "X-Object-Meta-State": {state}, "Last-Modified": {"unrelated invalid date"}, "Content-Length": {"unrelated invalid size"}}
			return objectMetadataWire(r, code, h, io.NopCloser(strings.NewReader("poll proof"))), nil
		})
		options := []ObjectWaitOption{WithObjectWaitPollInterval(time.Nanosecond), WithObjectWaitTimeout(time.Second), WithObjectWaitStatusHeader("X-Object-Meta-State"), WithObjectWaitProgressCallback(func(progress int) error {
			callbacks++
			if progress != 0 {
				t.Error("Swift fabricated progress", progress)
			}
			c.SetToken("fresh")
			c.MoreHeaders["X-Source"] = "valid later drift"
			return nil
		})}
		var result *ObjectWaitResult
		var err error
		if deletion {
			result, err = New(c).WaitForDelete(context.Background(), objectMetadataContainer, objectMetadataName, options...)
		} else {
			result, err = New(c).WaitForStatus(context.Background(), objectMetadataContainer, objectMetadataName, "READY", options...)
		}
		wantPolls := 2
		if deletion {
			wantPolls = 3
		}
		if err != nil || result == nil || !result.Complete || result.Deleted != deletion || result.Polls != wantPolls || calls != wantPolls || callbacks != wantPolls-1 || result.Last == nil || len(result.Last.Attempts) != 1 || result.Last.Attempts[0].LogicalAttempt != wantPolls {
			t.Fatal(result, err, calls, callbacks)
		}
		if deletion && result.Status != nil || !deletion && (result.Status == nil || *result.Status != "rEaDy") {
			t.Fatal("selected state commit", result)
		}
		result.Last.Acknowledgement.Body[0] = '!'
		result.Last.Acknowledgement.Header.Set("X-Proof", "changed")
		if string(result.Last.Attempts[0].Response.Body) != "poll proof" || result.Last.Attempts[0].Response.Header.Get("X-Proof") != "kept" {
			t.Fatal("last acknowledgement aliases attempt")
		}
	}
}

func TestObjectWaitCoreSelectorsAndUnsupported(t *testing.T) {
	for _, tc := range []struct{ attribute, header string }{
		{"name", ""}, {"container", ""}, {"content_type", "Content-Type"}, {"etag", "ETag"}, {"content_encoding", "Content-Encoding"}, {"content_disposition", "Content-Disposition"},
		{"manifest", "X-Object-Manifest"}, {"object_manifest", "X-Object-Manifest"}, {"timestamp", "X-Timestamp"}, {"last_modified_at", "Last-Modified"}, {"updated_at", "Last-Modified"},
		{"delete_at", "X-Delete-At"}, {"accept_ranges", "Accept-Ranges"}, {"access_control_allow_origin", "Access-Control-Allow-Origin"}, {"expires_at", "Expires"}, {"signature", "Signature"},
	} {
		t.Run(tc.attribute, func(t *testing.T) {
			c := objectMetadataClient()
			calls := 0
			target := "literal value"
			if tc.attribute == "name" {
				target = "literal/key"
			}
			if tc.attribute == "container" {
				target = "literal-box"
			}
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				h := http.Header{}
				if tc.header != "" {
					h.Set(tc.header, target)
				}
				return objectMetadataOK(r, 200, h), nil
			})
			result, err := New(c).WaitForStatus(context.Background(), "literal-box", "literal/key", strings.ToUpper(target), WithObjectWaitStatusAttribute(tc.attribute), WithObjectWaitTimeout(100*time.Millisecond))
			if err != nil || result == nil || !result.Complete || result.Deleted || result.Status == nil || *result.Status != target || result.Polls != 1 || calls != 1 {
				t.Fatal(result, err, calls)
			}
		})
	}
	c := objectMetadataClient()
	calls := 0
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return objectMetadataWire(r, 200, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("selected missing"))), nil
	})
	for _, attribute := range []string{"", "status", "content_length", "is_newest", "is_static_large_object", "delete_after", "range", "unknown"} {
		result, err := New(c).WaitForStatus(context.Background(), "box", "key", "READY", WithObjectWaitStatusAttribute(attribute))
		if result != nil || !errors.Is(err, resource.ErrUnsupported) || calls != 0 {
			t.Fatal("unsupported selector reached HTTP", attribute, result, err, calls)
		}
	}
	result, err := New(c).WaitForStatus(context.Background(), "box", "key", "READY", WithObjectWaitStatusHeader("X-Object-Meta-State"))
	objectMetadataProof(t, err, 200, "selected missing")
	if result == nil || result.Status != nil || result.Complete || result.Polls != 1 || !errors.Is(err, resource.ErrUnsupported) || calls != 1 {
		t.Fatal(result, err, calls)
	}
}

func TestObjectWaitCoreProjectionAndTerminalStates(t *testing.T) {
	for _, tc := range []struct {
		name, target                         string
		headers                              http.Header
		code                                 int
		failure, notFound, invalid, complete bool
	}{
		{"failure", "READY", http.Header{"X-State": {"eRrOr"}}, 200, true, false, false, false},
		{"target wins failure", "ERROR", http.Header{"X-State": {"error"}}, 200, false, false, false, true},
		{"present empty target", "", http.Header{"X-State": {""}}, 204, false, false, false, true},
		{"gone", "READY", http.Header{}, 404, false, true, false, false},
		{"aliases", "READY", http.Header{"X-State": {"READY"}, "x-state": {"READY"}}, 200, false, false, true, false},
		{"multiple", "READY", http.Header{"X-State": {"READY", "READY"}}, 200, false, false, true, false},
		{"zero values", "READY", http.Header{"X-State": nil}, 200, false, false, true, false},
		{"invalidUTF8", "READY", http.Header{"X-State": {"\xff"}}, 200, false, false, true, false},
		{"Unicode control", "READY", http.Header{"X-State": {"READY\u0085"}}, 200, false, false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := objectMetadataClient()
			calls, callbacks := 0, 0
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				h := tc.headers.Clone()
				h.Set("X-Proof", "kept")
				return objectMetadataWire(r, tc.code, h, io.NopCloser(strings.NewReader("terminal proof"))), nil
			})
			result, err := New(c).WaitForStatus(context.Background(), "box", "key", tc.target, WithObjectWaitStatusHeader("X-State"), WithObjectWaitTimeout(time.Second), WithObjectWaitProgressCallback(func(int) error { callbacks++; return nil }))
			if result == nil || result.Complete != tc.complete || result.Deleted || calls != 1 || callbacks != 0 || result.Polls != 1 {
				t.Fatal(result, err, calls, callbacks)
			}
			if tc.complete {
				if err != nil || result.Status == nil {
					t.Fatal(result, err)
				}
				return
			}
			proof := objectMetadataProof(t, err, tc.code, "terminal proof")
			if tc.failure {
				var failure *resource.FailedStateError
				if !errors.Is(err, resource.ErrFailedState) || !errors.As(err, &failure) || failure.Status != "eRrOr" || result.Status == nil || *result.Status != "eRrOr" {
					t.Fatal(result, err)
				}
			} else if result.Status != nil {
				t.Fatal("dirty/missing status committed", result)
			}
			if tc.notFound && !errors.Is(err, resource.ErrNotFound) || tc.invalid && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			result.Last.Acknowledgement.Body[0] = '!'
			result.Last.Acknowledgement.Header.Set("X-Proof", "changed")
			if string(proof.Body) != "terminal proof" || proof.Header.Get("X-Proof") != "kept" {
				t.Fatal("terminal error aliases result")
			}
		})
	}
}

func TestObjectWaitCoreFailureEvidenceAndCallbackCauses(t *testing.T) {
	for _, mode := range []string{"accepted read", "accepted Close", "source restored", "rejected IO"} {
		t.Run(mode, func(t *testing.T) {
			c := objectMetadataClient()
			cause := errors.New("body boundary cause")
			calls, retries, callbacks := 0, 0, 0
			c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				code := 404
				var body io.ReadCloser
				switch mode {
				case "accepted read":
					body = &readTestBody{data: []byte("raw"), readErr: cause}
				case "accepted Close":
					body = &objectMetadataBody{Reader: strings.NewReader("raw"), closeErr: cause}
				case "source restored":
					c.MoreHeaders = map[string]string{"Range": "bytes=0-0"}
					body = &objectMetadataBody{Reader: strings.NewReader("raw"), onClose: func() { c.MoreHeaders = nil }}
				case "rejected IO":
					code = 501
					body = &readTestBody{data: []byte("raw"), readErr: cause, closeErr: cause}
				}
				return objectMetadataWire(r, code, http.Header{"X-Proof": {"kept"}}, body), nil
			})
			result, err := New(c).WaitForDelete(context.Background(), "box", "key", WithObjectWaitProgressCallback(func(int) error { callbacks++; return nil }))
			if result == nil || result.Last == nil || len(result.Last.Attempts) != 1 || result.Complete || result.Deleted || result.Status != nil || result.Polls != 1 || calls != 1 || retries != 0 || callbacks != 0 || result.Last.Attempts[0].Error == nil {
				t.Fatal("dirty response completed waiter", result, err, calls, retries, callbacks)
			}
			if mode == "source restored" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if mode != "rejected IO" {
				objectMetadataProof(t, err, 404, "raw")
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 501 || string(native.Body) != "raw" || result.Last.Acknowledgement != nil {
					t.Fatal(result, err, native)
				}
			}
		})
	}
	c := objectMetadataClient()
	ctx, cancel := context.WithCancelCause(context.Background())
	callbackCause, cancelCause := errors.New("callback cause"), errors.New("callback canceled")
	calls, callbacks := 0, 0
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return objectMetadataWire(r, 200, http.Header{"X-State": {"BUILD"}, "X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("callback proof"))), nil
	})
	result, err := New(c).WaitForStatus(ctx, "box", "key", "READY", WithObjectWaitStatusHeader("X-State"), WithObjectWaitProgressCallback(func(progress int) error {
		callbacks++
		c.MoreHeaders = map[string]string{"If-Match": "foreign"}
		cancel(cancelCause)
		return callbackCause
	}))
	objectMetadataProof(t, err, 200, "callback proof")
	if result == nil || result.Status != nil || result.Complete || result.Deleted || calls != 1 || callbacks != 1 || !errors.Is(err, callbackCause) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err, calls, callbacks)
	}
}

type waitCoreCancelJar struct {
	calls  int
	cancel context.CancelCauseFunc
	cause  error
}

func (j *waitCoreCancelJar) Cookies(*url.URL) []*http.Cookie {
	j.calls++
	if j.calls == 2 {
		j.cancel(j.cause)
	}
	return nil
}
func (*waitCoreCancelJar) SetCookies(*url.URL, []*http.Cookie) {}

func TestObjectWaitCoreTimersAndLastActualPhase(t *testing.T) {
	c := objectMetadataClient()
	api := New(c)
	deleteDefaults, err := api.prepareObjectWait(context.Background(), "box", "key", "", true, nil)
	if err != nil || deleteDefaults.interval != 2*time.Second || deleteDefaults.timeout != 120*time.Second {
		t.Fatal(deleteDefaults, err)
	}
	statusDefaults, err := api.prepareObjectWait(context.Background(), "box", "key", "READY", false, []ObjectWaitOption{WithObjectWaitStatusHeader("X-State")})
	if err != nil || statusDefaults.interval != 2*time.Second || statusDefaults.timeout != 0 {
		t.Fatal(statusDefaults, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	calls := 0
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return objectMetadataWire(r, 200, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("last clean"))), nil
	})
	start := time.Now()
	result, err := api.WaitForDelete(ctx, "box", "key")
	objectMetadataProof(t, err, 200, "last clean")
	if !errors.Is(err, context.DeadlineExceeded) || result == nil || result.Polls != 1 || calls != 1 || result.Complete || time.Since(start) > time.Second {
		t.Fatal("pause ignored caller deadline", result, err, calls, time.Since(start))
	}
	c = objectMetadataClient()
	ctx, cancelCause := context.WithCancelCause(context.Background())
	defer cancelCause(nil)
	cause := errors.New("canceled before second physical request")
	jar := &waitCoreCancelJar{cancel: cancelCause, cause: cause}
	c.HTTPClient.Jar = jar
	calls, callbacks := 0, 0
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return objectMetadataWire(r, 200, http.Header{"X-State": {"BUILD"}, "X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("actual first phase"))), nil
	})
	result, err = New(c).WaitForStatus(ctx, "box", "key", "READY", WithObjectWaitStatusHeader("X-State"), WithObjectWaitUnlimitedWait(), WithObjectWaitPollInterval(time.Nanosecond), WithObjectWaitProgressCallback(func(int) error { callbacks++; return nil }))
	objectMetadataProof(t, err, 200, "actual first phase")
	if result == nil || result.Polls != 2 || calls != 1 || callbacks != 1 || jar.calls != 2 || result.Last == nil || len(result.Last.Attempts) != 1 || result.Last.Attempts[0].LogicalAttempt != 1 || result.Status == nil || *result.Status != "BUILD" || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatal("empty exchange discarded actual last phase", result, err, calls, callbacks, jar.calls)
	}
}

func TestObjectWaitCoreHeaderPolicyIsolation(t *testing.T) {
	for _, waiting := range []bool{true, false} {
		c := objectMetadataClient()
		calls, retries := 0, 0
		c.RetryFunc = func(ctx context.Context, method, target string, opts *gophercloud.RequestOpts, original error, count uint) error {
			retries++
			opts.MoreHeaders["Range"] = "bytes=0-0"
			return nil
		}
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			code := 503
			if calls == 2 {
				code = 201
				if waiting {
					code = 404
				}
			}
			return objectMetadataWire(r, code, http.Header{}, io.NopCloser(strings.NewReader("physical"))), nil
		})
		if waiting {
			result, err := New(c).WaitForDelete(context.Background(), "box", "key")
			if result == nil || result.Complete || !errors.Is(err, resource.ErrInvalidOption) || result.Last == nil || len(result.Last.Attempts) != 1 || calls != 1 || retries != 1 {
				t.Fatal("native conditional waiter header escaped", result, err, calls, retries)
			}
		} else {
			result, err := New(c).CreateObject(context.Background(), "box", "key", CreateObjectInput{Data: []byte("unchanged existing policy")})
			if err != nil || result == nil || result.Ordinary.Acknowledgement.StatusCode != 201 || len(result.Ordinary.Attempts) != 2 || calls != 2 || retries != 1 {
				t.Fatal("wait policy changed ordinary Create", result, err, calls, retries)
			}
		}
	}
}
