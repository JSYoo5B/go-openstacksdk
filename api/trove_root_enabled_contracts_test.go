package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/db/v1/instances"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/internal/troveroot"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const troveRootContractBase = "/reverse/trove/project/"
const troveRootContractPath = troveRootContractBase + "instances/selected/root"

type troveRootContractTransport func(*http.Request) (*http.Response, error)

func (f troveRootContractTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func troveRootContractClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("database", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + troveRootContractBase
	client.MoreHeaders = map[string]string{"X-Source": "native"}
	return client
}

func troveRootContractWire(t *testing.T, r *http.Request, path, token string) {
	t.Helper()
	if r.Method != http.MethodGet || r.URL.Path != path || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != token || r.Header.Get("X-Source") != "native" {
		t.Errorf("method=%s url=%s headers=%v", r.Method, r.URL, r.Header)
	}
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) != 0 {
			t.Errorf("GET body=%q error=%v", body, err)
		}
	}
}

func troveRootContractOperation(t *testing.T, err error) *resource.OperationError {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != "IsRootEnabled" || operation.Resource != "instances" || operation.Cause == nil {
		t.Fatalf("operation context=%v", err)
	}
	return operation
}

func TestTroveRootEnabledContractsObjectFieldSemantics(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       bool
	}{
		{"true", `{"rootEnabled":true}`, true},
		{"false", `{"rootEnabled":false}`, false},
		{"absent", `{}`, false},
		{"wrong case", `{"RootEnabled":true}`, false},
		{"null field", `{"rootEnabled":null}`, false},
		{"string field", `{"rootEnabled":"true"}`, false},
		{"number field", `{"rootEnabled":9007199254740993}`, false},
		{"array field", `{"rootEnabled":[true]}`, false},
		{"object field", `{"rootEnabled":{"enabled":true}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				troveRootContractWire(t, r, troveRootContractPath, "test-token")
				testcloud.JSON(w, 200, tc.body)
			})
			var enabled bool // The public facade keeps its bool, error signature.
			var err error
			enabled, err = instances.New(troveRootContractClient(cloud)).IsRootEnabled(context.Background(), "selected")
			if enabled != tc.want || err != nil || calls.Load() != 1 {
				t.Fatalf("enabled=%v want=%v error=%v requests=%d", enabled, tc.want, err, calls.Load())
			}
		})
	}
}

func TestTroveRootEnabledContractsSuccessfulNonObjectsDoNotRetryOrFallback(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `"root"`, `23`, `true`, `false`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				troveRootContractWire(t, r, troveRootContractPath, "test-token")
				testcloud.JSON(w, 200, body)
			})
			enabled, err := instances.New(troveRootContractClient(cloud)).IsRootEnabled(context.Background(), "selected")
			var envelope *troveroot.EnvelopeError
			var native gophercloud.ErrUnexpectedResponseCode
			if enabled || !errors.As(err, &envelope) || envelope.Actual == "" || errors.As(err, &native) || calls.Load() != 1 || retries.Load() != 0 {
				t.Fatalf("enabled=%v error=%v requests=%d retries=%d", enabled, err, calls.Load(), retries.Load())
			}
			troveRootContractOperation(t, err)
		})
	}
}

func TestTroveRootEnabledContractsNativeStatusErrors(t *testing.T) {
	for _, status := range []int{403, 404, 500, 204} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			wantBody := []byte(`{"rootEnabled":true,"error":"native failure"}`)
			if status == 204 {
				wantBody = nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				troveRootContractWire(t, r, troveRootContractPath, "test-token")
				w.Header().Set("X-Proof", "physical-root-response")
				w.WriteHeader(status)
				_, _ = w.Write(wantBody)
			})
			enabled, err := instances.New(troveRootContractClient(cloud)).IsRootEnabled(context.Background(), "selected")
			var native gophercloud.ErrUnexpectedResponseCode
			var envelope *troveroot.EnvelopeError
			if enabled || !errors.As(err, &native) || native.Actual != status || native.Method != http.MethodGet || native.URL != cloud.Server.URL+troveRootContractPath || !reflect.DeepEqual(native.Expected, []int{200}) || !bytes.Equal(native.Body, wantBody) || native.ResponseHeader.Get("X-Proof") != "physical-root-response" || !gophercloud.ResponseCodeIs(err, status) || errors.As(err, &envelope) || calls.Load() != 1 {
				t.Fatalf("enabled=%v native=%+v error=%v requests=%d", enabled, native, err, calls.Load())
			}
			troveRootContractOperation(t, err)
		})
	}
}

func TestTroveRootEnabledContractsNativeJSONErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		cause      error
	}{
		{"empty", "", io.EOF},
		{"incomplete", `{`, io.ErrUnexpectedEOF},
		{"invalid", `{"rootEnabled":}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				troveRootContractWire(t, r, troveRootContractPath, "test-token")
				testcloud.JSON(w, 200, tc.body)
			})
			enabled, err := instances.New(troveRootContractClient(cloud)).IsRootEnabled(context.Background(), "selected")
			var envelope *troveroot.EnvelopeError
			var syntax *json.SyntaxError
			if enabled || err == nil || errors.As(err, &envelope) || calls.Load() != 1 {
				t.Fatalf("enabled=%v error=%v requests=%d", enabled, err, calls.Load())
			}
			if tc.cause != nil && !errors.Is(err, tc.cause) || tc.cause == nil && !errors.As(err, &syntax) {
				t.Fatalf("native JSON cause changed: %v", err)
			}
			troveRootContractOperation(t, err)
		})
	}
}

func TestTroveRootEnabledContractsTransportAndContext(t *testing.T) {
	t.Run("transport", func(t *testing.T) {
		cloud := testcloud.New(t)
		cause := errors.New("root transport failure")
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = troveRootContractTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			troveRootContractWire(t, r, troveRootContractPath, "test-token")
			return nil, cause
		})
		enabled, err := instances.New(troveRootContractClient(cloud)).IsRootEnabled(context.Background(), "selected")
		var envelope *troveroot.EnvelopeError
		if enabled || !errors.Is(err, cause) || errors.As(err, &envelope) || calls.Load() != 1 {
			t.Fatalf("enabled=%v error=%v attempts=%d", enabled, err, calls.Load())
		}
		troveRootContractOperation(t, err)
	})
	t.Run("canceled context", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			t.Error("canceled request reached HTTP server")
			testcloud.JSON(w, 200, `{"rootEnabled":true}`)
		})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		enabled, err := instances.New(troveRootContractClient(cloud)).IsRootEnabled(ctx, "selected")
		var envelope *troveroot.EnvelopeError
		if enabled || !errors.Is(err, context.Canceled) || errors.As(err, &envelope) || calls.Load() != 0 {
			t.Fatalf("enabled=%v error=%v requests=%d", enabled, err, calls.Load())
		}
		troveRootContractOperation(t, err)
	})
}

func TestTroveRootEnabledContractsNativeReauthenticationBackoffAndRetry(t *testing.T) {
	cloud := testcloud.New(t)
	client := troveRootContractClient(cloud)
	var attempts, reauths, backoffs, retries atomic.Int32
	cloud.Provider.ReauthFunc = func(context.Context) error {
		reauths.Add(1)
		cloud.Provider.SetToken("refreshed-token")
		return nil
	}
	cloud.Provider.RetryBackoffFunc = func(_ context.Context, code *gophercloud.ErrUnexpectedResponseCode, _ error, count uint) error {
		backoffs.Add(1)
		if code.Actual != 429 || count != 1 {
			t.Errorf("backoff status=%d count=%d", code.Actual, count)
		}
		return nil
	}
	cloud.Provider.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, err error, count uint) error {
		retries.Add(1)
		if !gophercloud.ResponseCodeIs(err, 503) {
			return err
		}
		if method != http.MethodGet || target != cloud.Server.URL+troveRootContractPath || count != 2 || options.JSONResponse == nil || options.JSONBody != nil || options.RawBody != nil || options.KeepResponseBody {
			t.Errorf("retry method=%s target=%s count=%d options=%+v", method, target, count, options)
		}
		options.MoreHeaders["X-Native-Retry"] = "preserved"
		return nil
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		token := "refreshed-token"
		if n == 1 {
			token = "test-token"
		}
		troveRootContractWire(t, r, troveRootContractPath, token)
		switch n {
		case 1:
			testcloud.JSON(w, 401, `{"error":"expired"}`)
		case 2:
			testcloud.JSON(w, 429, `{"error":"backoff"}`)
		case 3:
			testcloud.JSON(w, 503, `{"error":"retry"}`)
		default:
			if r.Header.Get("X-Native-Retry") != "preserved" {
				t.Error("native retry option mutation was lost")
			}
			testcloud.JSON(w, 200, `{"rootEnabled":true}`)
		}
	})
	api := instances.New(client)
	enabled, err := api.IsRootEnabled(context.Background(), "selected")
	if !enabled || err != nil || attempts.Load() != 4 || reauths.Load() != 1 || backoffs.Load() != 1 || retries.Load() != 1 || api.RawClient() != client || client.ProviderClient != cloud.Provider || !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Source": "native"}) || cloud.Provider.Token() != "refreshed-token" {
		t.Fatalf("enabled=%v error=%v attempts=%d callbacks=%d/%d/%d", enabled, err, attempts.Load(), reauths.Load(), backoffs.Load(), retries.Load())
	}
}

func TestTroveRootEnabledContractsNativeDecodeAndTransportRetry(t *testing.T) {
	for _, mode := range []string{"decode", "transport"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var attempts, requests, retries atomic.Int32
			cause := errors.New("temporary root transport failure")
			base := cloud.Provider.HTTPClient.Transport
			if base == nil {
				base = http.DefaultTransport
			}
			cloud.Provider.HTTPClient.Transport = troveRootContractTransport(func(r *http.Request) (*http.Response, error) {
				if attempts.Add(1) == 1 && mode == "transport" {
					return nil, cause
				}
				return base.RoundTrip(r)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, err error, count uint) error {
				retries.Add(1)
				want := cause
				if mode == "decode" {
					want = io.ErrUnexpectedEOF
				}
				if !errors.Is(err, want) {
					return err
				}
				if method != http.MethodGet || target != cloud.Server.URL+troveRootContractPath || count != 1 || options.JSONResponse == nil {
					t.Errorf("retry method=%s target=%s count=%d options=%+v", method, target, count, options)
				}
				cloud.Provider.SetToken("retry-token")
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := requests.Add(1)
				if mode == "decode" && n == 1 {
					troveRootContractWire(t, r, troveRootContractPath, "test-token")
					testcloud.JSON(w, 200, `{`)
					return
				}
				troveRootContractWire(t, r, troveRootContractPath, "retry-token")
				testcloud.JSON(w, 200, `{"rootEnabled":true}`)
			})
			enabled, err := instances.New(troveRootContractClient(cloud)).IsRootEnabled(context.Background(), "selected")
			wantRequests := int32(2)
			if mode == "transport" {
				wantRequests = 1
			}
			if !enabled || err != nil || attempts.Load() != 2 || requests.Load() != wantRequests || retries.Load() != 1 {
				t.Fatalf("enabled=%v error=%v attempts=%d requests=%d retries=%d", enabled, err, attempts.Load(), requests.Load(), retries.Load())
			}
		})
	}
}

func TestTroveRootEnabledContractsNativeCallbackFailure(t *testing.T) {
	cloud := testcloud.New(t)
	stop := errors.New("native root retry refused")
	var requests, retries atomic.Int32
	cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, count uint) error {
		retries.Add(1)
		if !errors.Is(err, io.ErrUnexpectedEOF) || count != 1 {
			t.Errorf("callback cause=%v count=%d", err, count)
		}
		return stop
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		troveRootContractWire(t, r, troveRootContractPath, "test-token")
		testcloud.JSON(w, 200, `{`)
	})
	enabled, err := instances.New(troveRootContractClient(cloud)).IsRootEnabled(context.Background(), "selected")
	operation := troveRootContractOperation(t, err)
	if enabled || !errors.Is(err, stop) || operation.Cause != stop || requests.Load() != 1 || retries.Load() != 1 {
		t.Fatalf("enabled=%v error=%v requests=%d retries=%d", enabled, err, requests.Load(), retries.Load())
	}
}

func TestTroveRootEnabledContractsNativeRedirectPolicy(t *testing.T) {
	for _, follow := range []bool{false, true} {
		t.Run(fmt.Sprint(follow), func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests, redirects atomic.Int32
			redirectPath := troveRootContractBase + "root-redirect"
			cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, _ []*http.Request) error {
				redirects.Add(1)
				if !follow {
					return http.ErrUseLastResponse
				}
				next.Header.Set("X-Native-Redirect", "preserved")
				return nil
			}
			cloud.Mux.HandleFunc(troveRootContractPath, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				troveRootContractWire(t, r, troveRootContractPath, "test-token")
				w.Header().Set("Location", redirectPath)
				w.WriteHeader(http.StatusTemporaryRedirect)
			})
			cloud.Mux.HandleFunc(redirectPath, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				troveRootContractWire(t, r, redirectPath, "test-token")
				if r.Header.Get("X-Native-Redirect") != "preserved" {
					t.Error("native redirect option mutation was lost")
				}
				testcloud.JSON(w, 200, `{"rootEnabled":true}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				t.Errorf("root query used an SDK fallback: %s %s", r.Method, r.URL)
				w.WriteHeader(http.StatusInternalServerError)
			})
			enabled, err := instances.New(troveRootContractClient(cloud)).IsRootEnabled(context.Background(), "selected")
			if follow {
				if !enabled || err != nil || requests.Load() != 2 {
					t.Fatalf("enabled=%v error=%v requests=%d", enabled, err, requests.Load())
				}
			} else {
				if enabled || !gophercloud.ResponseCodeIs(err, http.StatusTemporaryRedirect) || requests.Load() != 1 {
					t.Fatalf("enabled=%v error=%v requests=%d", enabled, err, requests.Load())
				}
				troveRootContractOperation(t, err)
			}
			if redirects.Load() != 1 {
				t.Fatalf("redirect callbacks=%d", redirects.Load())
			}
		})
	}
}
