package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/fixedrequest"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

const limitsGuardedTarget = "https://guarded.test/reverse/v3/limits?project_id=actual"

type limitsGuardedTransport func(*http.Request) (*http.Response, error)

func (f limitsGuardedTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func limitsGuardedClient(f limitsGuardedTransport) *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: f}}
	p.UseTokenLock()
	p.SetToken("original-token")
	return &gophercloud.ServiceClient{ProviderClient: p, Type: "volumev3", Endpoint: "https://guarded.test/reverse/v3/", Microversion: "3.38"}
}
func limitsGuardedSnapshot(source *gophercloud.ServiceClient, cause error) func(context.Context) error {
	provider, endpoint, base, role, version := source.ProviderClient, source.Endpoint, source.ResourceBase, source.Type, source.Microversion
	var sticky error
	return func(context.Context) error {
		if sticky == nil && (source.ProviderClient != provider || source.Endpoint != endpoint || source.ResourceBase != base || source.Type != role || source.Microversion != version) {
			sticky = cause
		}
		return sticky
	}
}

type limitsGuardedBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *limitsGuardedBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}
func limitsGuardedWire(code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"X-Request-Id": {"physical-original"}, "X-Wire-Status": {fmt.Sprint(code)}}, Body: body}
}

type limitsGuardedReader struct {
	prefix []byte
	cause  error
	onRead func()
}

func (r *limitsGuardedReader) Read(p []byte) (int, error) {
	if r.onRead != nil {
		r.onRead()
		r.onRead = nil
	}
	n := copy(p, r.prefix)
	r.prefix = r.prefix[n:]
	return n, r.cause
}

type limitsGuardedMarshal struct{ calls atomic.Int32 }

func (m *limitsGuardedMarshal) MarshalJSON() ([]byte, error) {
	m.calls.Add(1)
	return []byte(`{"value":1}`), nil
}

func TestDoJSONGuardedRejectsInitialSourceBeforeMarshalOrHTTP(t *testing.T) {
	var requests atomic.Int32
	client := limitsGuardedClient(func(*http.Request) (*http.Response, error) {
		requests.Add(1)
		return limitsGuardedWire(200, io.NopCloser(strings.NewReader("unexpected"))), nil
	})
	sourceCause := errors.New("initial source invariant failed")
	body := &limitsGuardedMarshal{}
	got, err := rest.DoJSONGuarded(context.Background(), client, func(context.Context) error { return sourceCause }, http.MethodPost, limitsGuardedTarget, body, nil, 200)
	if got != nil || !errors.Is(err, sourceCause) || body.calls.Load() != 0 || requests.Load() != 0 {
		t.Fatal(got, err, body.calls.Load(), requests.Load())
	}
	var proof *resource.ResponseError
	if errors.As(err, &proof) {
		t.Fatal("local preflight borrowed response", proof)
	}
}

func TestDoJSONGuardedStopsSourceChangingNativeRetryBeforeResend(t *testing.T) {
	for _, returnCallbackError := range []bool{false, true} {
		t.Run(fmt.Sprint(returnCallbackError), func(t *testing.T) {
			var requests, callbacks atomic.Int32
			rejected := &limitsGuardedBody{Reader: strings.NewReader("original503")}
			client := limitsGuardedClient(func(r *http.Request) (*http.Response, error) {
				requests.Add(1)
				if r.URL.String() != limitsGuardedTarget || r.Method != http.MethodGet || r.Body != nil {
					t.Error(r.Method, r.URL, r.Body)
				}
				return limitsGuardedWire(503, rejected), nil
			})
			sourceCause, callbackCause := errors.New("source changed during retry"), errors.New("caller retry failure")
			guard := limitsGuardedSnapshot(client, sourceCause)
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, count uint) error {
				callbacks.Add(1)
				if count > 1 {
					return original
				}
				client.Microversion = "3.60"
				if returnCallbackError {
					return callbackCause
				}
				return nil
			}
			originalHook := reflect.ValueOf(client.RetryFunc).Pointer()
			got, err := rest.DoJSONGuarded(context.Background(), client, guard, http.MethodGet, limitsGuardedTarget, nil, nil, 200)
			var native gophercloud.ErrUnexpectedResponseCode
			var proof *resource.ResponseError
			if got != nil || !errors.Is(err, sourceCause) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != "original503" || native.ResponseHeader.Get("X-Request-Id") != "physical-original" || errors.As(err, &proof) || requests.Load() != 1 || callbacks.Load() != 1 || rejected.closes.Load() != 1 || reflect.ValueOf(client.RetryFunc).Pointer() != originalHook {
				t.Fatal(got, err, native, requests.Load(), callbacks.Load(), rejected.closes.Load())
			}
			if returnCallbackError && !errors.Is(err, callbackCause) {
				t.Fatal("callback cause lost", err)
			}
			client.Microversion = "3.38"
			if !errors.Is(guard(context.Background()), sourceCause) {
				t.Fatal("known invariant violation was cleared")
			}
		})
	}
}

func TestDoJSONGuardedStopsSourceChangingNativeReauthenticationBeforeResend(t *testing.T) {
	var requests, reauth atomic.Int32
	rejected := &limitsGuardedBody{Reader: strings.NewReader("original401")}
	client := limitsGuardedClient(func(*http.Request) (*http.Response, error) {
		requests.Add(1)
		return limitsGuardedWire(401, rejected), nil
	})
	sourceCause := errors.New("source changed during native reauth")
	guard := limitsGuardedSnapshot(client, sourceCause)
	client.ReauthFunc = func(context.Context) error {
		reauth.Add(1)
		client.SetToken("new-live-token")
		client.ResourceBase = "https://guarded.test/changed/v3/"
		return nil
	}
	originalHook := reflect.ValueOf(client.ReauthFunc).Pointer()
	got, err := rest.DoJSONGuarded(context.Background(), client, guard, http.MethodGet, limitsGuardedTarget, nil, nil, 200)
	var native *gophercloud.ErrUnableToReauthenticate
	var proof *resource.ResponseError
	if got != nil || !errors.Is(err, sourceCause) || !errors.As(err, &native) || !gophercloud.ResponseCodeIs(native.ErrOriginal, 401) || !errors.Is(native.ErrReauth, sourceCause) || errors.As(err, &proof) || requests.Load() != 1 || reauth.Load() != 1 || rejected.closes.Load() != 1 || reflect.ValueOf(client.ReauthFunc).Pointer() != originalHook {
		t.Fatal(got, err, native, requests.Load(), reauth.Load(), rejected.closes.Load())
	}
}

func TestDoJSONGuardedChecksSourceAfterWaitingForLiveAuthentication(t *testing.T) {
	// The guard state is atomic; this tests waiting-boundary ordering without
	// racing ordinary ServiceClient configuration, which the SDK does not support.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var physical atomic.Int32
	client := limitsGuardedClient(func(*http.Request) (*http.Response, error) {
		physical.Add(1)
		return limitsGuardedWire(200, io.NopCloser(strings.NewReader("unexpected"))), nil
	})
	started, release := make(chan struct{}), make(chan struct{})
	var released atomic.Bool
	releaseAuth := func() {
		if released.CompareAndSwap(false, true) {
			close(release)
		}
	}
	defer releaseAuth()
	client.ReauthFunc = func(context.Context) error {
		close(started)
		<-release
		client.SetToken("after-shared-auth")
		return nil
	}
	authDone := make(chan error, 1)
	go func() { authDone <- client.Reauthenticate(context.Background(), "") }()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("shared reauth did not start")
	}
	sourceCause := errors.New("source invariant changed while shared auth waited")
	var invalid atomic.Bool
	var checks atomic.Int32
	entered := make(chan struct{})
	guard := func(context.Context) error {
		n := checks.Add(1)
		wasInvalid := invalid.Load()
		if n == 2 {
			close(entered)
		}
		if wasInvalid {
			return sourceCause
		}
		return nil
	}
	type outcome struct {
		response *rest.Response
		err      error
	}
	done := make(chan outcome, 1)
	go func() {
		r, e := rest.DoJSONGuarded(ctx, client, guard, http.MethodGet, limitsGuardedTarget, nil, nil, 200)
		done <- outcome{r, e}
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("physical attempt never entered source guard")
	}
	invalid.Store(true)
	releaseAuth()
	select {
	case got := <-done:
		if got.response != nil || !errors.Is(got.err, sourceCause) || physical.Load() != 0 || checks.Load() < 3 {
			t.Fatal(got.response, got.err, physical.Load(), checks.Load())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("guarded request stayed blocked")
	}
	select {
	case err := <-authDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shared auth stayed blocked")
	}
}

func TestDoJSONGuardedAcceptedEvidenceJoinsReadCloseSourceAndContextWithoutReplay(t *testing.T) {
	for _, code := range []int{200, 204} {
		for _, mode := range []string{"read source", "close source", "read close source cancel"} {
			t.Run(fmt.Sprintf("%d %s", code, mode), func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				sourceCause, readCause, closeCause, cancelCause := errors.New("accepted source changed"), errors.New("accepted read failed"), errors.New("accepted close failed"), errors.New("custom cancel cause")
				payload := []byte{0, 255, '{', 'x', 0}
				body := &limitsGuardedBody{Reader: bytes.NewReader(payload)}
				wire := limitsGuardedWire(code, body)
				var requests, retries atomic.Int32
				client := limitsGuardedClient(func(*http.Request) (*http.Response, error) { requests.Add(1); return wire, nil })
				guard := limitsGuardedSnapshot(client, sourceCause)
				if strings.Contains(mode, "read") {
					body.Reader = &limitsGuardedReader{prefix: bytes.Clone(payload), cause: readCause, onRead: func() {
						client.Microversion = "3.60"
						wire.StatusCode = 500
						wire.Header.Set("X-Request-Id", "changed during Read")
					}}
				}
				if strings.Contains(mode, "close") {
					body.closeErr = closeCause
				}
				body.onClose = func() {
					client.Type = "other"
					wire.StatusCode = 503
					wire.Header.Set("X-Request-Id", "changed during Close")
					if strings.Contains(mode, "cancel") {
						cancel(cancelCause)
					}
				}
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries.Add(1)
					return errors.New("accepted failure must not replay")
				}
				got, err := rest.DoJSONGuarded(ctx, client, guard, http.MethodGet, limitsGuardedTarget, nil, nil, code)
				var proof *resource.ResponseError
				if got == nil || !errors.Is(err, sourceCause) || !errors.As(err, &proof) || got.StatusCode != code || proof.StatusCode != code || !bytes.Equal(got.Body, payload) || !bytes.Equal(proof.Body, payload) || got.Header.Get("X-Request-Id") != "physical-original" || proof.Header.Get("X-Request-Id") != "physical-original" || requests.Load() != 1 || retries.Load() != 0 || body.closes.Load() != 1 {
					t.Fatal(got, err, proof, requests.Load(), retries.Load(), body.closes.Load())
				}
				if strings.Contains(mode, "read") && !errors.Is(err, readCause) || strings.Contains(mode, "close") && !errors.Is(err, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal("joined cause lost", err)
				}
				got.Body[0] = '!'
				got.Header.Set("X-Request-Id", "caller-mutated")
				if !bytes.Equal(proof.Body, payload) || proof.Header.Get("X-Request-Id") != "physical-original" {
					t.Fatal("proof aliases response", proof)
				}
			})
		}
	}
}

func TestDoJSONGuardedExpandedNativeStatusKeepsActualRejectionAndSourceCause(t *testing.T) {
	var requests, callbacks atomic.Int32
	first := &limitsGuardedBody{Reader: strings.NewReader("original503")}
	second := &limitsGuardedBody{Reader: strings.NewReader("actual201")}
	client := limitsGuardedClient(func(*http.Request) (*http.Response, error) {
		if requests.Add(1) == 1 {
			return limitsGuardedWire(503, first), nil
		}
		return limitsGuardedWire(201, second), nil
	})
	sourceCause := errors.New("source changed after native-expanded response")
	guard := limitsGuardedSnapshot(client, sourceCause)
	second.onClose = func() { client.Microversion = "latest" }
	client.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, original error, count uint) error {
		callbacks.Add(1)
		if count > 1 {
			return original
		}
		opts.OkCodes = append(opts.OkCodes, 201)
		return nil
	}
	got, err := rest.DoJSONGuarded(context.Background(), client, guard, http.MethodGet, limitsGuardedTarget, nil, nil, 200)
	var native gophercloud.ErrUnexpectedResponseCode
	var proof *resource.ResponseError
	if got != nil || !errors.Is(err, sourceCause) || !errors.As(err, &native) || native.Actual != 201 || !reflect.DeepEqual(native.Expected, []int{200}) || native.URL != limitsGuardedTarget || native.Method != http.MethodGet || string(native.Body) != "actual201" || native.ResponseHeader.Get("X-Request-Id") != "physical-original" || errors.As(err, &proof) || requests.Load() != 2 || callbacks.Load() != 1 || first.closes.Load() != 1 || second.closes.Load() != 1 {
		t.Fatal(got, err, native, requests.Load(), callbacks.Load(), first.closes.Load(), second.closes.Load())
	}
}

func TestDoJSONGuardedKeepsLiveAuthenticationAndOwnedRequestOnAllowedRetry(t *testing.T) {
	var requests, callbacks atomic.Int32
	var bodies []*limitsGuardedBody
	client := limitsGuardedClient(nil)
	client.MoreHeaders = map[string]string{"X-Snapshot": "captured"}
	sourceCause := errors.New("unexpected source change")
	guard := limitsGuardedSnapshot(client, sourceCause)
	client.HTTPClient.Transport = limitsGuardedTransport(func(r *http.Request) (*http.Response, error) {
		n := requests.Add(1)
		raw, err := io.ReadAll(r.Body)
		token := "original-token"
		code := 503
		if n == 2 {
			token = "retry-live-token"
			code = 200
		}
		if n > 2 {
			return nil, errors.New("extra retry")
		}
		if err != nil || string(raw) != `{"value":1}` || r.URL.String() != limitsGuardedTarget || r.Method != http.MethodPost || r.Header.Get("X-Snapshot") != "captured" || r.Header.Get("X-Auth-Token") != token {
			t.Error(r.Method, r.URL, string(raw), r.Header, err)
		}
		body := &limitsGuardedBody{Reader: strings.NewReader(fmt.Sprint(code))}
		bodies = append(bodies, body)
		return limitsGuardedWire(code, body), nil
	})
	client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, count uint) error {
		callbacks.Add(1)
		if count > 1 {
			return original
		}
		client.SetToken("retry-live-token")
		client.MoreHeaders["X-Snapshot"] = "ordinary-later"
		return nil
	}
	originalHook := reflect.ValueOf(client.RetryFunc).Pointer()
	got, err := rest.DoJSONGuarded(context.Background(), client, guard, http.MethodPost, limitsGuardedTarget, json.RawMessage(`{"value":1}`), nil, 200)
	if err != nil || got == nil || got.StatusCode != 200 || string(got.Body) != "200" || requests.Load() != 2 || callbacks.Load() != 1 || reflect.ValueOf(client.RetryFunc).Pointer() != originalHook || client.MoreHeaders["X-Snapshot"] != "ordinary-later" {
		t.Fatal(got, err, requests.Load(), callbacks.Load())
	}
	for _, body := range bodies {
		if body.closes.Load() != 1 {
			t.Fatal("wrong close count", body.closes.Load())
		}
	}
}

func TestNewGuardedStopsSourceChangingNativeBackoffBeforePhysicalResend(t *testing.T) {
	var requests, backoffs atomic.Int32
	first := &limitsGuardedBody{Reader: strings.NewReader("original429")}
	source := limitsGuardedClient(func(*http.Request) (*http.Response, error) {
		requests.Add(1)
		return limitsGuardedWire(429, first), nil
	})
	source.MaxBackoffRetries = 1
	cause := errors.New("source changed during backoff")
	guard := limitsGuardedSnapshot(source, cause)
	source.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
		backoffs.Add(1)
		source.Microversion = "3.60"
		return nil
	}
	originalHook := reflect.ValueOf(source.RetryBackoffFunc).Pointer()
	owned, err := fixedrequest.NewGuarded(source, http.MethodGet, limitsGuardedTarget, guard)
	if err != nil {
		t.Fatal(err)
	}
	got, err := owned.Request(context.Background(), http.MethodGet, limitsGuardedTarget, &gophercloud.RequestOpts{OkCodes: []int{200}, KeepResponseBody: true})
	if got != nil || !errors.Is(err, cause) || requests.Load() != 1 || backoffs.Load() != 1 || first.closes.Load() != 1 || reflect.ValueOf(source.RetryBackoffFunc).Pointer() != originalHook {
		t.Fatal(got, err, requests.Load(), backoffs.Load(), first.closes.Load())
	}
}
