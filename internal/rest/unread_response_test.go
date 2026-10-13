package rest_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type unreadReader func([]byte) (int, error)

func (f unreadReader) Read(data []byte) (int, error) { return f(data) }
func unreadPolicy() rest.RejectionPolicy {
	return rest.RejectionPolicy{Codes: []int{401, 404, 429, 498, 503}, PreserveCleanRetry: true, StopFaultRetryCallbacks: true}
}
func unreadHeaders() rest.RequestHeaderPolicy {
	return rest.RequestHeaderPolicy{Values: map[string]string{"Accept": "*/*", "X-Owned": "fixed"}, Absent: []string{"X-Absent"}}
}
func unreadProof(t *testing.T, err error, status int, body string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "req-42" {
		t.Fatal(err, proof)
	}
	return proof
}

func TestUnreadResponsePreflightHasNoHTTPOrBorrowedEvidence(t *testing.T) {
	calls := 0
	client := responseClient(func(*http.Request) (*http.Response, error) { calls++; return wireResponse(200, http.NoBody), nil })
	canceled, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("preflight cancel")
	cancel(cause)
	bad := errors.New("initial source")
	cases := []struct {
		ctx         context.Context
		method, url string
		guard       func(context.Context) error
		codes       []int
		want        error
	}{
		{nil, "GET", client.ServiceURL("object"), nil, []int{200}, resource.ErrInvalidOption},
		{canceled, "GET", client.ServiceURL("object"), nil, []int{200}, cause},
		{context.Background(), "GET", client.ServiceURL("object"), func(context.Context) error { return bad }, []int{200}, bad},
		{context.Background(), "POST", client.ServiceURL("object"), nil, []int{200}, resource.ErrInvalidOption},
		{context.Background(), "GET", "https://foreign.test/object", nil, []int{200}, resource.ErrInvalidOption},
		{context.Background(), "GET", client.ServiceURL("object"), nil, nil, resource.ErrInvalidOption},
	}
	for _, tc := range cases {
		value, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(tc.ctx, client, tc.guard, tc.method, tc.url, unreadHeaders(), unreadPolicy(), tc.codes...)
		var proof *resource.ResponseError
		if value != nil || !errors.Is(err, tc.want) || errors.As(err, &proof) || calls != 0 {
			t.Fatal(value, err, proof, calls)
		}
	}
}

func TestUnreadResponseOpaqueBodyIsNotReadBeforeCallerAndEOFClosesOnce(t *testing.T) {
	for _, method := range []string{"GET", "HEAD"} {
		t.Run(method, func(t *testing.T) {
			payload := string([]byte{0, 255, 'x'})
			reads, calls := 0, 0
			reader := strings.NewReader(payload)
			body := &limitsGuardedBody{Reader: unreadReader(func(data []byte) (int, error) { reads++; return reader.Read(data) })}
			client := responseClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != method || req.Body != nil || req.Header.Get("Accept") != "*/*" || req.Header.Get("X-Owned") != "fixed" || req.Header.Get("X-Absent") != "" {
					t.Fatal(req)
				}
				return wireResponse(203, body), nil
			})
			value, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(context.Background(), client, nil, method, client.ServiceURL("object"), unreadHeaders(), unreadPolicy(), 203)
			if err != nil || value == nil || reads != 0 || calls != 1 || body.closes.Load() != 0 {
				t.Fatal(value, err, reads, calls, body.closes.Load())
			}
			got, err := io.ReadAll(value.Body)
			if err != nil || string(got) != payload || body.closes.Load() != 1 || value.Body.Close() != nil || value.Body.Close() != nil {
				t.Fatal(got, err, body.closes.Load())
			}
			if n, err := value.Body.Read(make([]byte, 1)); n != 0 || err != io.EOF {
				t.Fatal(n, err)
			}
		})
	}
}

func TestUnreadResponseEarlyCloseIsDistinctFromExhaustionAndIdempotent(t *testing.T) {
	body := &limitsGuardedBody{Reader: strings.NewReader("more data")}
	client := responseClient(func(*http.Request) (*http.Response, error) { return wireResponse(200, body), nil })
	value, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(context.Background(), client, nil, "GET", client.ServiceURL("object"), nilHeaderPolicy(), unreadPolicy(), 200)
	if err != nil {
		t.Fatal(err)
	}
	dst := make([]byte, 1)
	if n, err := value.Body.Read(dst); n != 1 || err != nil {
		t.Fatal(n, err)
	}
	if err := value.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if err := value.Body.Close(); err != nil || body.closes.Load() != 1 {
		t.Fatal(err, body.closes.Load())
	}
	if n, err := value.Body.Read(dst); n != 0 || !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(n, err)
	}
}
func nilHeaderPolicy() rest.RequestHeaderPolicy { return rest.RequestHeaderPolicy{} }

func TestUnreadResponseFailedCurrentChunkOwnsPrivateProofAndAllPhysicalCauses(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	readCause, closeCause, cancelCause, sourceCause := errors.New("read"), errors.New("close"), errors.New("custom cancel"), errors.New("changed source")
	reads, calls, hooks := 0, 0, 0
	changed := false
	var physical *http.Response
	body := &limitsGuardedBody{closeErr: closeCause, onClose: func() { changed = false; cancel(cancelCause) }}
	body.Reader = unreadReader(func(data []byte) (int, error) {
		reads++
		if reads == 1 {
			return copy(data, strings.Repeat("a", 32)), nil
		}
		changed = true
		physical.StatusCode = 299
		physical.Header.Set("X-Request-Id", "forged physical")
		return copy(data, "bad"), readCause
	})
	client := responseClient(func(*http.Request) (*http.Response, error) {
		calls++
		physical = wireResponse(203, body)
		return physical, nil
	})
	client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		hooks++
		return nil
	}
	client.ReauthFunc = func(context.Context) error { hooks++; return nil }
	guard := func(context.Context) error {
		if changed {
			return sourceCause
		}
		return nil
	}
	value, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(ctx, client, guard, "GET", client.ServiceURL("object"), unreadHeaders(), unreadPolicy(), 203)
	if err != nil || reads != 0 {
		t.Fatal(err, reads)
	}
	value.StatusCode = 277
	value.Header.Set("X-Request-Id", "forged public")
	dst := make([]byte, 64)
	if n, err := value.Body.Read(dst); n != 32 || err != nil {
		t.Fatal(n, err)
	}
	n, err := value.Body.Read(dst)
	if n != 3 || string(dst[:n]) != "bad" || reads != 2 || calls != 1 || hooks != 0 || body.closes.Load() != 1 {
		t.Fatal(n, err, reads, calls, hooks, body.closes.Load())
	}
	proof := unreadProof(t, err, 203, "bad")
	for _, cause := range []error{readCause, closeCause, cancelCause, sourceCause, context.Canceled} {
		if !errors.Is(err, cause) {
			t.Fatal(cause, err)
		}
	}
	// Caller error mutation cannot change retained lifecycle proof either.
	proof.StatusCode = 222
	proof.Header.Set("X-Request-Id", "forged error")
	proof.Body[0] = 'x'
	unreadProof(t, value.Body.Close(), 203, "bad")
	if body.closes.Load() != 1 {
		t.Fatal(body.closes.Load())
	}
}

func TestUnreadResponseEOFObservesSourceBeforeCloseCanRestoreAndJoinCloseFailure(t *testing.T) {
	changed := false
	sourceCause, closeCause := errors.New("read boundary source"), errors.New("EOF close")
	body := &limitsGuardedBody{Reader: unreadReader(func(data []byte) (int, error) { changed = true; return copy(data, "tail"), io.EOF }), closeErr: closeCause, onClose: func() { changed = false }}
	client := responseClient(func(*http.Request) (*http.Response, error) { return wireResponse(200, body), nil })
	guard := func(context.Context) error {
		if changed {
			return sourceCause
		}
		return nil
	}
	value, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(context.Background(), client, guard, "GET", client.ServiceURL("object"), unreadHeaders(), unreadPolicy(), 200)
	if err != nil {
		t.Fatal(err)
	}
	dst := make([]byte, 10)
	n, err := value.Body.Read(dst)
	if n != 4 || !errors.Is(err, sourceCause) || !errors.Is(err, closeCause) || body.closes.Load() != 1 {
		t.Fatal(n, err, body.closes.Load())
	}
	unreadProof(t, err, 200, "tail")
	unreadProof(t, value.Body.Close(), 200, "tail")
}

func TestUnreadResponseCloseFaultAndCancellationBeforeFirstReadRetainActualEvidence(t *testing.T) {
	for _, mode := range []string{"close", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New(mode)
			reads := 0
			body := &limitsGuardedBody{Reader: unreadReader(func(data []byte) (int, error) { reads++; return copy(data, "unread"), io.EOF })}
			if mode == "close" {
				body.closeErr = cause
			}
			client := responseClient(func(*http.Request) (*http.Response, error) { return wireResponse(206, body), nil })
			value, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(ctx, client, nil, "GET", client.ServiceURL("object"), unreadHeaders(), unreadPolicy(), 206)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "cancel" {
				cancel(cause)
				_, err = value.Body.Read(make([]byte, 10))
			} else {
				err = value.Body.Close()
			}
			if !errors.Is(err, cause) || reads != 0 || body.closes.Load() != 1 {
				t.Fatal(err, reads, body.closes.Load())
			}
			unreadProof(t, err, 206, "")
			unreadProof(t, value.Body.Close(), 206, "")
		})
	}
}

func TestUnreadResponseInvalidReaderCountsAndNoProgressHaveActualProof(t *testing.T) {
	for _, mode := range []string{"negative", "excess", "negative EOF", "excess EOF", "no progress"} {
		t.Run(mode, func(t *testing.T) {
			reads := 0
			body := &limitsGuardedBody{Reader: unreadReader(func(data []byte) (int, error) {
				reads++
				switch mode {
				case "negative":
					return -1, nil
				case "excess":
					return len(data) + 1, nil
				case "negative EOF":
					return -1, io.EOF
				case "excess EOF":
					return len(data) + 1, io.EOF
				default:
					return 0, nil
				}
			})}
			client := responseClient(func(*http.Request) (*http.Response, error) { return wireResponse(200, body), nil })
			value, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(context.Background(), client, nil, "GET", client.ServiceURL("object"), unreadHeaders(), unreadPolicy(), 200)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 100; i++ {
				_, err = value.Body.Read(make([]byte, 2))
				if err != nil {
					break
				}
			}
			if err == nil || body.closes.Load() != 1 {
				t.Fatal(err, reads, body.closes.Load())
			}
			if mode == "no progress" {
				if !errors.Is(err, io.ErrNoProgress) || reads != 100 {
					t.Fatal(err, reads)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) || errors.Is(err, io.EOF) || reads != 1 {
				t.Fatal(err, reads)
			}
			unreadProof(t, err, 200, "")
		})
	}
}

func TestUnreadResponseCleanRetryReauthAndBackoffKeepOwnedRequestAndLiveToken(t *testing.T) {
	for _, code := range []int{401, 503, 429, 498} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls, retries, reauth, backoff, reads := 0, 0, 0, 0, 0
			rejected := &limitsGuardedBody{Reader: strings.NewReader("native rejection")}
			accepted := &limitsGuardedBody{Reader: unreadReader(func(data []byte) (int, error) { reads++; return copy(data, "opaque"), io.EOF })}
			client := responseClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.Path != "/v1/project/object" || req.Body != nil || req.Header.Get("Accept") != "*/*" || req.Header.Get("X-Owned") != "fixed" || req.Header.Get("X-Absent") != "" {
					t.Fatal(req)
				}
				if calls == 1 {
					return wireResponse(code, rejected), nil
				}
				if req.Header.Get("X-Auth-Token") != "fresh" {
					t.Fatal(req.Header)
				}
				return wireResponse(200, accepted), nil
			})
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				client.SetToken("fresh")
				return nil
			}
			client.ReauthFunc = func(context.Context) error { reauth++; client.SetToken("fresh"); return nil }
			client.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoff++
				client.SetToken("fresh")
				return nil
			}
			original := reflect.ValueOf(client.RetryFunc).Pointer()
			value, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(context.Background(), client, nil, "GET", client.ServiceURL("object"), unreadHeaders(), unreadPolicy(), 200)
			if err != nil || value == nil || calls != 2 || rejected.closes.Load() != 1 || accepted.closes.Load() != 0 || reads != 0 || reflect.ValueOf(client.RetryFunc).Pointer() != original {
				t.Fatal(value, err, calls, reads, rejected.closes.Load(), accepted.closes.Load())
			}
			if code == 401 && reauth != 1 || code == 503 && retries != 1 || (code == 429 || code == 498) && backoff != 1 {
				t.Fatal(retries, reauth, backoff)
			}
			data, err := io.ReadAll(value.Body)
			if err != nil || string(data) != "opaque" || accepted.closes.Load() != 1 {
				t.Fatal(err, data, accepted.closes.Load())
			}
		})
	}
}

func TestUnreadResponseFaultyNativeRejectionsKeepPinnedStatusAndAllCausesWithoutHooks(t *testing.T) {
	for _, code := range []int{401, 404, 429, 498, 503} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls, hooks := 0, 0
			readCause, closeCause := errors.New("rejected read"), errors.New("rejected close")
			var physical *http.Response
			body := &limitsGuardedBody{Reader: unreadReader(func(data []byte) (int, error) {
				physical.StatusCode = 299
				physical.Header.Set("X-Request-Id", "body forged")
				return copy(data, "rejected actual"), readCause
			}), closeErr: closeCause}
			client := responseClient(func(*http.Request) (*http.Response, error) {
				calls++
				physical = wireResponse(code, body)
				return physical, nil
			})
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks++
				return nil
			}
			client.ReauthFunc = func(context.Context) error { hooks++; return nil }
			client.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { hooks++; return nil }
			value, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(context.Background(), client, nil, "GET", client.ServiceURL("object"), unreadHeaders(), unreadPolicy(), 200)
			if value != nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || calls != 1 || hooks != 0 || body.closes.Load() != 1 {
				t.Fatal(value, err, calls, hooks, body.closes.Load())
			}
			original := err
			if code == 401 {
				var auth *gophercloud.ErrUnableToReauthenticate
				if !errors.As(err, &auth) || !errors.Is(auth.ErrReauth, readCause) || !errors.Is(auth.ErrReauth, closeCause) {
					t.Fatal(err, auth)
				}
				original = auth.ErrOriginal
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(original, &native) || native.Actual != code || native.Method != "GET" || native.URL != client.ServiceURL("object") || string(native.Body) != "rejected actual" || native.ResponseHeader.Get("X-Request-Id") != "req-42" {
				t.Fatal(err, native)
			}
		})
	}
}

func TestUnreadResponseDefaultFalseStillCallsOrdinaryRetryOnObservedFault(t *testing.T) {
	readCause := errors.New("baseline read fault")
	calls, callbacks := 0, 0
	body := &limitsGuardedBody{Reader: &limitsGuardedReader{prefix: []byte("native503"), cause: readCause}}
	client := responseClient(func(*http.Request) (*http.Response, error) { calls++; return wireResponse(503, body), nil })
	client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		callbacks++
		return nil
	}
	policy := unreadPolicy()
	policy.StopFaultRetryCallbacks = false
	value, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(context.Background(), client, nil, "GET", client.ServiceURL("object"), unreadHeaders(), policy, 200)
	if value != nil || !errors.Is(err, readCause) || calls != 1 || callbacks != 1 || body.closes.Load() != 1 {
		t.Fatal(value, err, calls, callbacks, body.closes.Load())
	}
}

func TestUnreadResponseNativeRetryCannotChangeBodyDecodeOrOwnedHeaders(t *testing.T) {
	mutations := map[string]func(*gophercloud.RequestOpts){
		"body":     func(o *gophercloud.RequestOpts) { o.JSONBody = map[string]int{"body": 1} },
		"raw":      func(o *gophercloud.RequestOpts) { o.RawBody = strings.NewReader("body") },
		"decoded":  func(o *gophercloud.RequestOpts) { o.JSONResponse = new(any) },
		"consumed": func(o *gophercloud.RequestOpts) { o.KeepResponseBody = false },
		"owned":    func(o *gophercloud.RequestOpts) { o.MoreHeaders["X-Owned"] = "changed" },
		"absent":   func(o *gophercloud.RequestOpts) { o.MoreHeaders["X-Absent"] = "changed" },
		"omitted":  func(o *gophercloud.RequestOpts) { o.OmitHeaders = []string{"X-Owned"} },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			calls := 0
			client := responseClient(func(*http.Request) (*http.Response, error) {
				calls++
				return wireResponse(503, io.NopCloser(strings.NewReader("native503"))), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
				mutate(options)
				return nil
			}
			value, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(context.Background(), client, nil, "GET", client.ServiceURL("object"), unreadHeaders(), unreadPolicy(), 200)
			var native gophercloud.ErrUnexpectedResponseCode
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != "native503" || calls != 1 {
				t.Fatal(value, err, native, calls)
			}
		})
	}
}

func TestUnreadResponseBackoffPinsActualReceiptBeforeCallbackCanMutateIt(t *testing.T) {
	calls := 0
	callbackCause := errors.New("backoff failed")
	client := responseClient(func(*http.Request) (*http.Response, error) {
		calls++
		return wireResponse(429, io.NopCloser(strings.NewReader("actual429"))), nil
	})
	client.RetryBackoffFunc = func(_ context.Context, native *gophercloud.ErrUnexpectedResponseCode, _ error, _ uint) error {
		native.Actual = 299
		native.Body[0] = 'x'
		native.ResponseHeader.Set("X-Request-Id", "forged callback")
		return callbackCause
	}
	value, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(context.Background(), client, nil, "GET", client.ServiceURL("object"), unreadHeaders(), unreadPolicy(), 200)
	var native gophercloud.ErrUnexpectedResponseCode
	if value != nil || !errors.Is(err, callbackCause) || !errors.As(err, &native) || native.Actual != 429 || string(native.Body) != "actual429" || native.ResponseHeader.Get("X-Request-Id") != "req-42" || calls != 1 {
		t.Fatal(value, err, native, calls)
	}
}

func TestUnreadResponseExpandedNativeSuccessIsStillRejectedBySDKPolicy(t *testing.T) {
	calls := 0
	client := responseClient(func(*http.Request) (*http.Response, error) {
		calls++
		return wireResponse(503, io.NopCloser(strings.NewReader("actual503"))), nil
	})
	client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
		o.OkCodes = []int{503}
		return nil
	}
	value, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(context.Background(), client, nil, "GET", client.ServiceURL("object"), unreadHeaders(), unreadPolicy(), 200)
	var native gophercloud.ErrUnexpectedResponseCode
	var terminal interface{ TerminalSDKFailure() bool }
	if value != nil || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != "actual503" || !errors.As(err, &terminal) || !terminal.TerminalSDKFailure() || calls != 2 {
		t.Fatal(value, err, native, calls)
	}
}

func TestUnreadResponseRejectedSourceRestorationAndCancellationStopEveryFaultHook(t *testing.T) {
	for _, code := range []int{401, 429, 498, 503} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			sourceCause, cancelCause, closeCause := errors.New("source changed during rejected Read"), errors.New("rejected custom cancel"), errors.New("rejected Close")
			calls, hooks := 0, 0
			changed := false
			body := &limitsGuardedBody{Reader: unreadReader(func(data []byte) (int, error) { changed = true; return copy(data, "actual bytes"), io.EOF }), closeErr: closeCause, onClose: func() { changed = false; cancel(cancelCause) }}
			client := responseClient(func(*http.Request) (*http.Response, error) { calls++; return wireResponse(code, body), nil })
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks++
				return nil
			}
			client.ReauthFunc = func(context.Context) error { hooks++; return nil }
			client.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { hooks++; return nil }
			guard := func(context.Context) error {
				if changed {
					return sourceCause
				}
				return nil
			}
			value, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(ctx, client, guard, "GET", client.ServiceURL("object"), unreadHeaders(), unreadPolicy(), 200)
			if value != nil || calls != 1 || hooks != 0 || body.closes.Load() != 1 {
				t.Fatal(value, err, calls, hooks, body.closes.Load())
			}
			for _, cause := range []error{sourceCause, cancelCause, closeCause, context.Canceled} {
				if !errors.Is(err, cause) {
					t.Fatal(cause, err)
				}
			}
			original := err
			if code == 401 {
				var auth *gophercloud.ErrUnableToReauthenticate
				if !errors.As(err, &auth) || !errors.Is(auth.ErrReauth, sourceCause) || !errors.Is(auth.ErrReauth, context.Canceled) {
					t.Fatal(err, auth)
				}
				original = auth.ErrOriginal
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(original, &native) || native.Actual != code || string(native.Body) != "actual bytes" || native.Method != "GET" || native.URL != client.ServiceURL("object") || native.ResponseHeader.Get("X-Request-Id") != "req-42" {
				t.Fatal(err, native)
			}
		})
	}
}

func TestUnreadResponseIdenticalCleanNativeRejectionRemainsDirect(t *testing.T) {
	calls, callbacks := 0, 0
	client := responseClient(func(*http.Request) (*http.Response, error) {
		calls++
		return wireResponse(404, io.NopCloser(strings.NewReader("native404"))), nil
	})
	client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
		callbacks++
		return original
	}
	value, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(context.Background(), client, nil, "GET", client.ServiceURL("object"), unreadHeaders(), unreadPolicy(), 200)
	native, ok := err.(gophercloud.ErrUnexpectedResponseCode)
	var proof *resource.ResponseError
	if value != nil || !ok || native.Actual != 404 || string(native.Body) != "native404" || native.ResponseHeader.Get("X-Request-Id") != "req-42" || errors.As(err, &proof) || calls != 1 || callbacks != 1 {
		t.Fatal(value, err, native, calls, callbacks)
	}
}

func TestUnreadResponseZeroLengthCallerReadsPreserveUnreadPhysicalBody(t *testing.T) {
	calls, reads := 0, 0
	reader := strings.NewReader("opaque unread payload")
	body := &limitsGuardedBody{Reader: unreadReader(func(data []byte) (int, error) { reads++; return reader.Read(data) })}
	client := responseClient(func(*http.Request) (*http.Response, error) { calls++; return wireResponse(200, body), nil })
	value, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(context.Background(), client, nil, "GET", client.ServiceURL("object"), unreadHeaders(), unreadPolicy(), 200)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 150; i++ {
		if n, err := value.Body.Read(nil); n != 0 || err != nil {
			t.Fatal(i, n, err)
		}
	}
	if reads != 0 || calls != 1 || body.closes.Load() != 0 {
		t.Fatal(reads, calls, body.closes.Load())
	}
	got, err := io.ReadAll(value.Body)
	if err != nil || string(got) != "opaque unread payload" || calls != 1 || body.closes.Load() != 1 || value.Body.Close() != nil {
		t.Fatal(err, string(got), calls, body.closes.Load())
	}
}
