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

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

const ownershipTarget = "https://service.test/reverse/v2/objects/fixed?selector=original"

type ownershipTransport func(*http.Request) (*http.Response, error)

func (f ownershipTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func ownershipClient(transport ownershipTransport) *gophercloud.ServiceClient {
	provider := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: transport}}
	provider.UseTokenLock()
	provider.SetToken("initial")
	return &gophercloud.ServiceClient{ProviderClient: provider, Type: "image", Endpoint: "https://service.test/reverse/v2/"}
}

type ownershipBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *ownershipBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type ownershipReadFailure struct {
	prefix []byte
	cause  error
	onRead func()
}

func (b *ownershipReadFailure) Read(p []byte) (int, error) {
	if b.onRead != nil {
		b.onRead()
		b.onRead = nil
	}
	n := copy(p, b.prefix)
	b.prefix = b.prefix[n:]
	return n, b.cause
}

func ownershipWire(code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{
		"X-Request-Id": {"actual-response"}, "X-Wire-Status": {fmt.Sprint(code)},
	}, Body: body}
}

type ownershipMarshal struct {
	calls atomic.Int32
	cause error
}

func (m *ownershipMarshal) MarshalJSON() ([]byte, error) {
	m.calls.Add(1)
	return []byte(`{"value":1}`), m.cause
}

func TestRESTOwnershipAcceptedEvidenceAndCauses(t *testing.T) {
	for _, code := range []int{200, 201, 204} {
		for _, mode := range []string{"success", "read", "close", "read close", "read close cancel", "close cancel"} {
			t.Run(fmt.Sprintf("status%d %s", code, mode), func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("accepted read failed"), errors.New("accepted close failed"), errors.New("custom accepted cancellation")
				payload := []byte{0, 255, '{', 'x', 0}
				body := &ownershipBody{Reader: bytes.NewReader(payload)}
				wire := ownershipWire(code, body)
				if strings.Contains(mode, "read") {
					body.Reader = &ownershipReadFailure{prefix: append([]byte(nil), payload...), cause: readCause, onRead: func() {
						wire.StatusCode = 500
						wire.Header.Set("X-Request-Id", "changed during Read")
					}}
				}
				if strings.Contains(mode, "close") {
					body.closeErr = closeCause
				}
				body.onClose = func() {
					wire.StatusCode = 503
					wire.Header.Set("X-Request-Id", "changed during Close")
					if strings.Contains(mode, "cancel") {
						cancel(cancelCause)
					}
				}
				var requests, retries atomic.Int32
				method := map[int]string{200: http.MethodGet, 201: http.MethodPost, 204: http.MethodDelete}[code]
				client := ownershipClient(func(r *http.Request) (*http.Response, error) {
					requests.Add(1)
					if r.Method != method || r.URL.String() != ownershipTarget || r.Body != nil {
						t.Error(r.Method, r.URL, r.Body)
					}
					return wire, nil
				})
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries.Add(1)
					return errors.New("accepted body failure must not replay")
				}
				v, err := rest.DoJSON(ctx, client, method, ownershipTarget, nil, nil, code)
				if v == nil || v.StatusCode != code || v.Header.Get("X-Request-Id") != "actual-response" || !bytes.Equal(v.Body, payload) || requests.Load() != 1 || retries.Load() != 0 || body.closes.Load() != 1 {
					t.Fatal(v, err, requests.Load(), retries.Load(), body.closes.Load())
				}
				if mode == "success" {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != code || proof.Header.Get("X-Request-Id") != "actual-response" || !bytes.Equal(proof.Body, payload) || strings.Contains(mode, "read") && !errors.Is(err, readCause) || strings.Contains(mode, "close") && !errors.Is(err, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal(v, err, proof)
				}
				v.Body[0] = '!'
				v.Header.Set("X-Request-Id", "caller")
				if !bytes.Equal(proof.Body, payload) || proof.Header.Get("X-Request-Id") != "actual-response" {
					t.Fatal("result/error evidence aliases", v, proof)
				}
			})
		}
	}
}

func TestRESTOwnershipCompletePreflight(t *testing.T) {
	for _, mode := range []string{"nil context", "canceled", "nil client", "nil provider", "foreign target", "no accepted codes", "initial marshal error"} {
		t.Run(mode, func(t *testing.T) {
			var requests atomic.Int32
			client := ownershipClient(func(*http.Request) (*http.Response, error) {
				requests.Add(1)
				return ownershipWire(500, io.NopCloser(strings.NewReader("unexpected"))), nil
			})
			ctx, target, codes := context.Background(), ownershipTarget, []int{201}
			cause := errors.New("preflight cause")
			body := &ownershipMarshal{}
			want := error(resource.ErrInvalidOption)
			switch mode {
			case "nil context":
				ctx = nil
			case "canceled":
				parent, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx, want = parent, context.Canceled
			case "nil client":
				client = nil
			case "nil provider":
				client.ProviderClient = nil
			case "foreign target":
				target = "https://foreign.test/objects"
			case "no accepted codes":
				codes = nil
			case "initial marshal error":
				body.cause, want = cause, cause
			}
			v, err := rest.DoJSON(ctx, client, http.MethodPost, target, body, nil, codes...)
			wantMarshals := int32(0)
			if mode == "initial marshal error" {
				wantMarshals = 1
			}
			if v != nil || !errors.Is(err, want) || requests.Load() != 0 || body.calls.Load() != wantMarshals || mode == "canceled" && !errors.Is(err, cause) {
				t.Fatal(v, err, requests.Load(), body.calls.Load())
			}
		})
	}
}

func TestRESTOwnershipRetryRequestSnapshots(t *testing.T) {
	for _, kind := range []string{"HTTP503", "transport"} {
		for _, change := range []string{"KeepResponseBody", "JSONResponse", "RawBody", "changed JSONBody", "in-place RawMessage", "nil to null", "null to nil", "unsupported JSONBody", "marshal and callback causes"} {
			t.Run(kind+" "+change, func(t *testing.T) {
				var requests, callbacks atomic.Int32
				transportCause, marshalCause, callbackCause := errors.New("prebody transport failed"), errors.New("retry encoding failed"), errors.New("retry callback failed")
				body := &ownershipBody{Reader: strings.NewReader("original prebody failure")}
				badMarshal := &ownershipMarshal{cause: marshalCause}
				input := any(map[string]int{"value": 1})
				if change == "nil to null" {
					input = nil
				} else if change == "null to nil" {
					input = json.RawMessage("null")
				}
				client := ownershipClient(func(r *http.Request) (*http.Response, error) {
					if r.Method != http.MethodPost || r.URL.String() != ownershipTarget {
						t.Error(r.Method, r.URL)
					}
					if requests.Add(1) > 1 {
						return ownershipWire(201, io.NopCloser(strings.NewReader("unexpected replay"))), nil
					}
					if kind == "transport" {
						return nil, transportCause
					}
					return ownershipWire(503, body), nil
				})
				client.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, original error, _ uint) error {
					callbacks.Add(1)
					if method != http.MethodPost || target != ownershipTarget || !options.KeepResponseBody || options.JSONResponse != nil || options.RawBody != nil {
						t.Error(method, target, options, original)
					}
					switch change {
					case "KeepResponseBody":
						options.KeepResponseBody = false
					case "JSONResponse":
						options.JSONResponse = new(any)
					case "RawBody":
						options.RawBody = strings.NewReader(`{"value":1}`)
					case "changed JSONBody":
						options.JSONBody = map[string]int{"value": 2}
					case "in-place RawMessage":
						owned, ok := options.JSONBody.(json.RawMessage)
						if !ok || string(owned) != `{"value":1}` {
							t.Fatal("serialized callback body is unavailable", options.JSONBody)
						}
						owned[len(owned)-2] = '2'
					case "nil to null":
						options.JSONBody = json.RawMessage("null")
					case "null to nil":
						options.JSONBody = nil
					case "unsupported JSONBody":
						options.JSONBody = make(chan int)
					case "marshal and callback causes":
						options.JSONBody = badMarshal
						return callbackCause
					}
					return nil
				}
				originalRetry := reflect.ValueOf(client.RetryFunc).Pointer()
				v, err := rest.DoJSON(context.Background(), client, http.MethodPost, ownershipTarget, input, nil, 201)
				if v != nil || !errors.Is(err, resource.ErrInvalidOption) || requests.Load() != 1 || callbacks.Load() != 1 || reflect.ValueOf(client.RetryFunc).Pointer() != originalRetry {
					t.Fatal(v, err, requests.Load(), callbacks.Load())
				}
				if kind == "HTTP503" {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != "original prebody failure" || native.ResponseHeader.Get("X-Request-Id") != "actual-response" || body.closes.Load() != 1 {
						t.Fatal("original native error or ownership lost", err, native, body.closes.Load())
					}
				} else if !errors.Is(err, transportCause) || body.closes.Load() != 0 {
					t.Fatal("original transport cause lost", err, body.closes.Load())
				}
				if change == "unsupported JSONBody" {
					var unsupported *json.UnsupportedTypeError
					if !errors.As(err, &unsupported) {
						t.Fatal("second encoding cause lost", err)
					}
				}
				if change == "marshal and callback causes" && (!errors.Is(err, marshalCause) || !errors.Is(err, callbackCause) || badMarshal.calls.Load() != 1) {
					t.Fatal("encoding/callback causes lost", err, badMarshal.calls.Load())
				}
			})
		}
	}
	for _, mode := range []string{"unchanged serialized snapshot", "identical replacement", "identical null replacement", "unchanged nil"} {
		t.Run(mode, func(t *testing.T) {
			var requests, callbacks atomic.Int32
			input := map[string]int{"value": 1}
			headers, codes := map[string]string{"X-Extension": "original"}, []int{201}
			bodyInput, expected := any(input), `{"value":1}`
			if mode == "identical null replacement" {
				bodyInput, expected = json.RawMessage("null"), "null"
			} else if mode == "unchanged nil" {
				bodyInput, expected = nil, ""
			}
			first := &ownershipBody{Reader: strings.NewReader("retry first")}
			accepted := &ownershipBody{Reader: strings.NewReader("accepted raw proof")}
			client := ownershipClient(func(r *http.Request) (*http.Response, error) {
				var raw []byte
				if r.Body != nil {
					var err error
					raw, err = io.ReadAll(r.Body)
					if err != nil {
						t.Fatal(err)
					}
				}
				if r.Method != http.MethodPost || r.URL.String() != ownershipTarget || string(raw) != expected || r.Header.Get("X-Extension") != "original" || (expected == "") != (r.Body == nil) {
					t.Error(r.Method, r.URL, string(raw), r.Header, r.Body)
				}
				if requests.Add(1) == 1 {
					return ownershipWire(503, first), nil
				}
				return ownershipWire(201, accepted), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, original error, _ uint) error {
				callbacks.Add(1)
				if !gophercloud.ResponseCodeIs(original, 503) {
					return original
				}
				input["value"], headers["X-Extension"], codes[0] = 2, "mutated", 200
				if mode == "identical replacement" {
					options.JSONBody = map[string]int{"value": 1}
				} else if mode == "identical null replacement" {
					options.JSONBody = (*string)(nil)
				}
				return nil
			}
			v, err := rest.DoJSON(context.Background(), client, http.MethodPost, ownershipTarget, bodyInput, headers, codes...)
			if err != nil || v == nil || v.StatusCode != 201 || string(v.Body) != "accepted raw proof" || requests.Load() != 2 || callbacks.Load() != 1 || first.closes.Load() != 1 || accepted.closes.Load() != 1 || input["value"] != 2 || headers["X-Extension"] != "mutated" || codes[0] != 200 {
				t.Fatal(v, err, requests.Load(), callbacks.Load(), first.closes.Load(), accepted.closes.Load())
			}
		})
	}
}

func TestRESTOwnershipUnexpectedStatusAndNativeBoundary(t *testing.T) {
	for _, mode := range []string{"successfully read", "read close cancel"} {
		t.Run("expanded OkCodes "+mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("unexpected read failed"), errors.New("unexpected close failed"), errors.New("unexpected cancellation")
			first := &ownershipBody{Reader: strings.NewReader("first503")}
			body := &ownershipBody{Reader: strings.NewReader("unexpected raw body")}
			if mode == "read close cancel" {
				body.Reader = &ownershipReadFailure{prefix: []byte("unexpected raw body"), cause: readCause}
				body.closeErr = closeCause
				body.onClose = func() { cancel(cancelCause) }
			}
			var requests, callbacks atomic.Int32
			client := ownershipClient(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodPost || r.URL.String() != ownershipTarget {
					t.Error(r.Method, r.URL)
				}
				if requests.Add(1) == 1 {
					return ownershipWire(503, first), nil
				}
				return ownershipWire(202, body), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
				callbacks.Add(1)
				options.OkCodes = []int{201, 202}
				return nil
			}
			v, err := rest.DoJSON(ctx, client, http.MethodPost, ownershipTarget, nil, nil, 201)
			var native gophercloud.ErrUnexpectedResponseCode
			var proof *resource.ResponseError
			if v != nil || !errors.As(err, &native) || native.Actual != 202 || native.Method != http.MethodPost || native.URL != ownershipTarget || !reflect.DeepEqual(native.Expected, []int{201}) || string(native.Body) != "unexpected raw body" || native.ResponseHeader.Get("X-Request-Id") != "actual-response" || errors.As(err, &proof) || requests.Load() != 2 || callbacks.Load() != 1 || first.closes.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(v, err, native, requests.Load(), callbacks.Load(), first.closes.Load(), body.closes.Load())
			}
			if mode == "read close cancel" && (!errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
				t.Fatal("unexpected status pipeline causes lost", err)
			}
		})
	}
	t.Run("ordinary rejection stays native-owned", func(t *testing.T) {
		body := &ownershipBody{Reader: strings.NewReader("native forbidden")}
		var requests atomic.Int32
		client := ownershipClient(func(*http.Request) (*http.Response, error) { requests.Add(1); return ownershipWire(403, body), nil })
		v, err := rest.DoJSON(context.Background(), client, http.MethodGet, ownershipTarget, nil, nil, 200)
		var native gophercloud.ErrUnexpectedResponseCode
		var proof *resource.ResponseError
		if v != nil || !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != "native forbidden" || errors.As(err, &proof) || requests.Load() != 1 || body.closes.Load() != 1 {
			t.Fatal(v, err, native, requests.Load(), body.closes.Load())
		}
	})
	t.Run("native reauth failure keeps original fields", func(t *testing.T) {
		cause := errors.New("native reauth failed")
		body := &ownershipBody{Reader: strings.NewReader("original401")}
		var requests, reauth atomic.Int32
		client := ownershipClient(func(*http.Request) (*http.Response, error) { requests.Add(1); return ownershipWire(401, body), nil })
		client.ReauthFunc = func(context.Context) error { reauth.Add(1); return cause }
		v, err := rest.DoJSON(context.Background(), client, http.MethodGet, ownershipTarget, nil, nil, 200)
		var native *gophercloud.ErrUnableToReauthenticate
		if v != nil || !errors.As(err, &native) || !errors.Is(native.ErrReauth, cause) || !gophercloud.ResponseCodeIs(native.ErrOriginal, 401) || requests.Load() != 1 || reauth.Load() != 1 || body.closes.Load() != 1 {
			t.Fatal(v, err, native, requests.Load(), reauth.Load(), body.closes.Load())
		}
	})
}

func TestRESTOwnershipFixedScopeAndLiveProvider(t *testing.T) {
	t.Run("prebody hooks keep source body route and live auth", func(t *testing.T) {
		var requests, reauth, backoff, retries atomic.Int32
		var bodies []*ownershipBody
		client := ownershipClient(nil)
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		client.ReauthFunc = func(context.Context) error { reauth.Add(1); client.SetToken("reauth"); return nil }
		client.RetryBackoffFunc = func(_ context.Context, native *gophercloud.ErrUnexpectedResponseCode, _ error, _ uint) error {
			backoff.Add(1)
			if native.Actual != 429 || native.URL != ownershipTarget || native.Method != http.MethodPost {
				t.Error(native)
			}
			client.SetToken("backoff")
			return nil
		}
		client.RetryFunc = func(_ context.Context, method, target string, _ *gophercloud.RequestOpts, original error, _ uint) error {
			retries.Add(1)
			if method != http.MethodPost || target != ownershipTarget || !gophercloud.ResponseCodeIs(original, 503) {
				return original
			}
			client.SetToken("retry")
			client.ResourceBase = "https://service.test/later/v2/"
			client.MoreHeaders = map[string]string{"X-Source": "later"}
			return nil
		}
		originalRetry := reflect.ValueOf(client.RetryFunc).Pointer()
		client.HTTPClient.Transport = ownershipTransport(func(r *http.Request) (*http.Response, error) {
			n := int(requests.Add(1)) - 1
			codes, tokens := []int{401, 429, 503, 201}, []string{"initial", "reauth", "backoff", "retry"}
			if n >= len(codes) {
				return nil, errors.New("extra mutation request")
			}
			raw, err := io.ReadAll(r.Body)
			if err != nil || r.Method != http.MethodPost || r.URL.String() != ownershipTarget || string(raw) != `{"value":1}` || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != tokens[n] {
				t.Error(r.Method, r.URL, string(raw), r.Header, err)
			}
			body := &ownershipBody{Reader: strings.NewReader(fmt.Sprint(codes[n]))}
			bodies = append(bodies, body)
			return ownershipWire(codes[n], body), nil
		})
		v, err := rest.DoJSON(context.Background(), client, http.MethodPost, ownershipTarget, map[string]int{"value": 1}, nil, 201)
		if err != nil || v == nil || v.StatusCode != 201 || string(v.Body) != "201" || requests.Load() != 4 || reauth.Load() != 1 || backoff.Load() != 1 || retries.Load() != 1 || reflect.ValueOf(client.RetryFunc).Pointer() != originalRetry || client.MoreHeaders["X-Source"] != "later" {
			t.Fatal(v, err, requests.Load(), reauth.Load(), backoff.Load(), retries.Load())
		}
		for _, body := range bodies {
			if body.closes.Load() != 1 {
				t.Fatal("response closed more than once", body.closes.Load())
			}
		}
	})
	t.Run("redirect cannot leave fixed scope or replace provider policy", func(t *testing.T) {
		body := &ownershipBody{Reader: strings.NewReader("redirect proof")}
		var requests, redirects atomic.Int32
		client := ownershipClient(func(r *http.Request) (*http.Response, error) {
			requests.Add(1)
			if r.URL.String() != ownershipTarget {
				t.Error("foreign redirect reached transport", r.URL)
			}
			wire := ownershipWire(307, body)
			wire.Header.Set("Location", "https://foreign.test/object")
			return wire, nil
		})
		client.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects.Add(1); return nil }
		originalRedirect := reflect.ValueOf(client.HTTPClient.CheckRedirect).Pointer()
		v, err := rest.DoJSON(context.Background(), client, http.MethodGet, ownershipTarget, nil, nil, 200)
		if v != nil || !errors.Is(err, resource.ErrInvalidOption) || requests.Load() != 1 || redirects.Load() != 1 || body.closes.Load() != 1 || reflect.ValueOf(client.HTTPClient.CheckRedirect).Pointer() != originalRedirect {
			t.Fatal(v, err, requests.Load(), redirects.Load(), body.closes.Load())
		}
	})
}
