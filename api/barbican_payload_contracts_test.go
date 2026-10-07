package api_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const payloadContractBase = "/reverse/barbican/v1/"
const payloadContractPath = payloadContractBase + "secrets/secret-id/payload"

type payloadContractTransport func(*http.Request) (*http.Response, error)

func (f payloadContractTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type payloadContractBody struct {
	io.ReadCloser
	reads, closes     atomic.Int32
	readErr, closeErr error
}

func (b *payloadContractBody) Read(p []byte) (int, error) {
	b.reads.Add(1)
	n, err := b.ReadCloser.Read(p)
	if b.readErr != nil {
		return n, b.readErr
	}
	return n, err
}
func (b *payloadContractBody) Close() error {
	b.closes.Add(1)
	err := b.ReadCloser.Close()
	if b.closeErr != nil {
		return b.closeErr
	}
	return err
}

type payloadContractTracking struct {
	calls  atomic.Int32
	mu     sync.Mutex
	bodies []*payloadContractBody
}

func payloadContractTrack(cloud *testcloud.Cloud, readErr, closeErr error) *payloadContractTracking {
	track := &payloadContractTracking{}
	base := cloud.Provider.HTTPClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	cloud.Provider.HTTPClient.Transport = payloadContractTransport(func(r *http.Request) (*http.Response, error) {
		track.calls.Add(1)
		response, err := base.RoundTrip(r)
		if err == nil {
			body := &payloadContractBody{ReadCloser: response.Body, readErr: readErr, closeErr: closeErr}
			response.Body = body
			track.mu.Lock()
			track.bodies = append(track.bodies, body)
			track.mu.Unlock()
		}
		return response, err
	})
	return track
}
func (track *payloadContractTracking) last(t *testing.T) *payloadContractBody {
	t.Helper()
	track.mu.Lock()
	defer track.mu.Unlock()
	if len(track.bodies) == 0 {
		t.Fatal("no physical response body")
	}
	return track.bodies[len(track.bodies)-1]
}
func payloadContractClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("key-manager", "/unused")
	client.ResourceBase = cloud.Server.URL + payloadContractBase
	client.MoreHeaders = map[string]string{"X-Source": "source"}
	return client
}
func payloadContractWire(t *testing.T, r *http.Request, path, accept, token string) {
	t.Helper()
	if r.Method != http.MethodGet || r.URL.Path != path || r.URL.RawQuery != "" || r.Header.Get("Accept") != accept || r.Header.Get("X-Auth-Token") != token || r.Header.Get("X-Source") != "source" {
		t.Errorf("method=%s url=%s headers=%v", r.Method, r.URL, r.Header)
	}
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) != 0 {
		t.Errorf("GET body=%x err=%v", body, err)
	}
}
func payloadContractOperation(t *testing.T, err error) {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != "GetPayload" || operation.Resource != "secrets" {
		t.Fatalf("operation context=%v", err)
	}
}

func TestBarbicanPayloadContractsExactBufferedBytes(t *testing.T) {
	for _, f := range []struct {
		name  string
		value []byte
	}{
		{"binary", []byte{0, 0xff, 'A', '\n', 0x80, 0}},
		{"empty", []byte{}},
	} {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			track := payloadContractTrack(cloud, nil, nil)
			cloud.Mux.HandleFunc(payloadContractPath, func(w http.ResponseWriter, r *http.Request) {
				payloadContractWire(t, r, payloadContractPath, "text/plain", "test-token")
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = w.Write(f.value)
			})
			var got []byte // Static proof: the public native facade returns buffered bytes.
			var err error
			got, err = secrets.New(payloadContractClient(cloud)).GetPayload(context.Background(), "secret-id")
			body := track.last(t)
			if err != nil || !bytes.Equal(got, f.value) || track.calls.Load() != 1 || body.reads.Load() == 0 || body.closes.Load() != 1 {
				t.Fatalf("bytes=%x err=%v calls=%d reads=%d closes=%d", got, err, track.calls.Load(), body.reads.Load(), body.closes.Load())
			}
		})
	}
}

func TestBarbicanPayloadContractsMediaAndExtensions(t *testing.T) {
	for _, f := range []struct {
		name, accept string
		source       bool
		options      []secrets.GetPayloadOption
	}{
		{"default", "text/plain", false, nil},
		{"empty typed", "text/plain", false, []secrets.GetPayloadOption{secrets.WithGetPayloadOptions(secrets.GetPayloadOpts{})}},
		{"explicit binary", "application/octet-stream", false, []secrets.GetPayloadOption{secrets.WithGetPayloadOptions(secrets.GetPayloadOpts{PayloadContentType: "application/octet-stream"})}},
		{"replacement", "application/json", false, []secrets.GetPayloadOption{secrets.WithGetPayloadOptions(secrets.GetPayloadOpts{PayloadContentType: "discarded"}), secrets.WithGetPayloadOptions(secrets.GetPayloadOpts{PayloadContentType: "application/json"})}},
		{"raw source override", "source/media", true, []secrets.GetPayloadOption{secrets.WithGetPayloadOptions(secrets.GetPayloadOpts{PayloadContentType: "typed/media"})}},
	} {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := payloadContractClient(cloud)
			if f.source {
				client.MoreHeaders["Accept"] = f.accept
			}
			var calls atomic.Int32
			want := []byte{0xff, 0, 'x'}
			cloud.Mux.HandleFunc(payloadContractPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				payloadContractWire(t, r, payloadContractPath, f.accept, "test-token")
				if r.Header.Get("X-Vendor-Setting") != "false" {
					t.Errorf("extension=%v", r.Header)
				}
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				_, _ = w.Write(want)
			})
			options := append(append([]secrets.GetPayloadOption(nil), f.options...), secrets.WithGetPayloadHeader("X-Vendor-Setting", "false"))
			got, err := secrets.New(client).GetPayload(context.Background(), "secret-id", options...)
			if err != nil || !bytes.Equal(got, want) || calls.Load() != 1 || client.MoreHeaders["X-Source"] != "source" {
				t.Fatalf("bytes=%x err=%v calls=%d source=%v", got, err, calls.Load(), client.MoreHeaders)
			}
		})
	}
}

func TestBarbicanPayloadContractsPreflight(t *testing.T) {
	sentinel := errors.New("option stopped")
	for _, f := range []struct {
		name   string
		option secrets.GetPayloadOption
		cause  error
	}{
		{"nil", nil, resource.ErrInvalidOption},
		{"field", request.WithField[secrets.GetPayloadOpts]("vendor", false), resource.ErrInvalidOption},
		{"query", request.WithQuery[secrets.GetPayloadOpts]("vendor", "a&b"), resource.ErrInvalidOption},
		{"argument", request.WithArgument[secrets.GetPayloadOpts]("vendor", "value"), resource.ErrInvalidOption},
		{"reserved accept", secrets.WithGetPayloadHeader("aCcEpT", "extension/media"), resource.ErrInvalidOption},
		{"bad name", secrets.WithGetPayloadHeader("Bad Header", "value"), resource.ErrInvalidOption},
		{"bad value", secrets.WithGetPayloadHeader("X-Vendor", "line\nvalue"), resource.ErrInvalidOption},
		{"callback error", func(*request.Config[secrets.GetPayloadOpts]) error { return sentinel }, sentinel},
	} {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			track := payloadContractTrack(cloud, nil, nil)
			got, err := secrets.New(payloadContractClient(cloud)).GetPayload(context.Background(), "secret-id", f.option)
			if got != nil || !errors.Is(err, f.cause) || track.calls.Load() != 0 {
				t.Fatalf("bytes=%x err=%v calls=%d", got, err, track.calls.Load())
			}
			payloadContractOperation(t, err)
		})
	}
	t.Run("typed accept is reserved too", func(t *testing.T) {
		cloud := testcloud.New(t)
		track := payloadContractTrack(cloud, nil, nil)
		got, err := secrets.New(payloadContractClient(cloud)).GetPayload(context.Background(), "secret-id", secrets.WithGetPayloadOptions(secrets.GetPayloadOpts{PayloadContentType: "typed/media"}), secrets.WithGetPayloadHeader("Accept", "extension/media"))
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || track.calls.Load() != 0 {
			t.Fatalf("bytes=%x err=%v calls=%d", got, err, track.calls.Load())
		}
		payloadContractOperation(t, err)
	})
}

func TestBarbicanPayloadContractsNativeStatusErrors(t *testing.T) {
	for _, status := range []int{201, 204, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			track := payloadContractTrack(cloud, nil, nil)
			want := []byte{0xff, 0, 'e'}
			if status == 204 {
				want = nil
			}
			cloud.Mux.HandleFunc(payloadContractPath, func(w http.ResponseWriter, r *http.Request) {
				payloadContractWire(t, r, payloadContractPath, "text/plain", "test-token")
				w.Header().Set("X-Proof", "physical-payload")
				w.WriteHeader(status)
				_, _ = w.Write(want)
			})
			got, err := secrets.New(payloadContractClient(cloud)).GetPayload(context.Background(), "secret-id")
			var native gophercloud.ErrUnexpectedResponseCode
			if got != nil || !errors.As(err, &native) || native.Actual != status || !gophercloud.ResponseCodeIs(err, status) || !bytes.Equal(native.Body, want) || native.ResponseHeader.Get("X-Proof") != "physical-payload" || track.calls.Load() != 1 {
				t.Fatalf("bytes=%x native=%+v err=%v calls=%d", got, native, err, track.calls.Load())
			}
			payloadContractOperation(t, err)
			if track.last(t).closes.Load() != 1 {
				t.Fatalf("rejected body closes=%d", track.last(t).closes.Load())
			}
		})
	}
}

func TestBarbicanPayloadContractsReadAndClose(t *testing.T) {
	readFault, closeFault := errors.New("payload read fault"), errors.New("payload close fault")
	for _, f := range []struct {
		name              string
		readErr, closeErr error
	}{
		{"partial read", readFault, nil},
		{"context read", context.Canceled, nil},
		{"close ignored", nil, closeFault},
	} {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			track := payloadContractTrack(cloud, f.readErr, f.closeErr)
			var retries atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			want := []byte("accepted-payload")
			cloud.Mux.HandleFunc(payloadContractPath, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(want) })
			got, err := secrets.New(payloadContractClient(cloud)).GetPayload(context.Background(), "secret-id")
			if f.readErr != nil {
				if got != nil || !errors.Is(err, f.readErr) {
					t.Fatalf("partial bytes=%x err=%v", got, err)
				}
				payloadContractOperation(t, err)
			} else if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("native Close behavior bytes=%x err=%v", got, err)
			}
			body := track.last(t)
			if body.reads.Load() == 0 || body.closes.Load() != 1 || track.calls.Load() != 1 || retries.Load() != 0 {
				t.Fatalf("accepted read retried or closed twice: reads=%d closes=%d calls=%d retries=%d", body.reads.Load(), body.closes.Load(), track.calls.Load(), retries.Load())
			}
		})
	}
}

func TestBarbicanPayloadContractsNativeReplay(t *testing.T) {
	t.Run("reauth backoff retry", func(t *testing.T) {
		cloud := testcloud.New(t)
		track := payloadContractTrack(cloud, nil, nil)
		var attempts, reauths, backoffs, retries atomic.Int32
		cloud.Provider.ReauthFunc = func(context.Context) error { reauths.Add(1); cloud.Provider.SetToken("refreshed-token"); return nil }
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
			if method != http.MethodGet || target != cloud.Server.URL+payloadContractPath || count != 2 || options.MoreHeaders["Accept"] != "application/octet-stream" || !options.KeepResponseBody || options.JSONResponse != nil {
				t.Errorf("retry method=%s target=%s count=%d options=%+v", method, target, count, options)
			}
			options.MoreHeaders["X-Native-Retry"] = "preserved"
			return nil
		}
		want := []byte{0xff, 0, 1}
		cloud.Mux.HandleFunc(payloadContractPath, func(w http.ResponseWriter, r *http.Request) {
			n := attempts.Add(1)
			token := "refreshed-token"
			if n == 1 {
				token = "test-token"
			}
			payloadContractWire(t, r, payloadContractPath, "application/octet-stream", token)
			if r.Header.Get("X-Vendor") != "false" {
				t.Errorf("extension=%v", r.Header)
			}
			switch n {
			case 1:
				testcloud.JSON(w, 401, `{"error":"expired"}`)
			case 2:
				testcloud.JSON(w, 429, `{"error":"rate-limited"}`)
			case 3:
				testcloud.JSON(w, 503, `{"error":"retry"}`)
			default:
				if r.Header.Get("X-Native-Retry") != "preserved" {
					t.Error("native option mutation lost")
				}
				_, _ = w.Write(want)
			}
		})
		got, err := secrets.New(payloadContractClient(cloud)).GetPayload(context.Background(), "secret-id", secrets.WithGetPayloadOptions(secrets.GetPayloadOpts{PayloadContentType: "application/octet-stream"}), secrets.WithGetPayloadHeader("X-Vendor", "false"))
		if err != nil || !bytes.Equal(got, want) || attempts.Load() != 4 || track.calls.Load() != 4 || reauths.Load() != 1 || backoffs.Load() != 1 || retries.Load() != 1 || track.last(t).closes.Load() != 1 {
			t.Fatalf("bytes=%x err=%v attempts=%d callbacks=%d/%d/%d", got, err, attempts.Load(), reauths.Load(), backoffs.Load(), retries.Load())
		}
	})
	for _, omit := range []bool{false, true} {
		t.Run("native header mutation "+fmt.Sprint(omit), func(t *testing.T) {
			cloud := testcloud.New(t)
			var attempts atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, err error, _ uint) error {
				if !gophercloud.ResponseCodeIs(err, 503) {
					return err
				}
				if omit {
					options.OmitHeaders = []string{"Accept"}
				} else {
					options.MoreHeaders["Accept"] = "native/media"
				}
				return nil
			}
			cloud.Mux.HandleFunc(payloadContractPath, func(w http.ResponseWriter, r *http.Request) {
				n := attempts.Add(1)
				accept := "text/plain"
				if n > 1 {
					accept = "native/media"
					if omit {
						accept = ""
					}
				}
				payloadContractWire(t, r, payloadContractPath, accept, "test-token")
				if n == 1 {
					w.WriteHeader(503)
					return
				}
				_, _ = w.Write([]byte("native-policy"))
			})
			got, err := secrets.New(payloadContractClient(cloud)).GetPayload(context.Background(), "secret-id")
			if err != nil || !bytes.Equal(got, []byte("native-policy")) || attempts.Load() != 2 {
				t.Fatalf("bytes=%x err=%v attempts=%d", got, err, attempts.Load())
			}
		})
	}
	t.Run("transport retry before extraction", func(t *testing.T) {
		cloud := testcloud.New(t)
		failure := errors.New("temporary transport fault")
		var attempts, retries atomic.Int32
		base := cloud.Provider.HTTPClient.Transport
		if base == nil {
			base = http.DefaultTransport
		}
		cloud.Provider.HTTPClient.Transport = payloadContractTransport(func(r *http.Request) (*http.Response, error) {
			if attempts.Add(1) == 1 {
				return nil, failure
			}
			return base.RoundTrip(r)
		})
		cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, count uint) error {
			retries.Add(1)
			if !errors.Is(err, failure) || count != 1 {
				return err
			}
			cloud.Provider.SetToken("retry-token")
			return nil
		}
		cloud.Mux.HandleFunc(payloadContractPath, func(w http.ResponseWriter, r *http.Request) {
			payloadContractWire(t, r, payloadContractPath, "text/plain", "retry-token")
			_, _ = w.Write([]byte("transport-recovered"))
		})
		got, err := secrets.New(payloadContractClient(cloud)).GetPayload(context.Background(), "secret-id")
		if err != nil || !bytes.Equal(got, []byte("transport-recovered")) || attempts.Load() != 2 || retries.Load() != 1 {
			t.Fatalf("bytes=%x err=%v attempts=%d retries=%d", got, err, attempts.Load(), retries.Load())
		}
	})
}

func TestBarbicanPayloadContractsNativeRedirect(t *testing.T) {
	for _, follow := range []bool{false, true} {
		t.Run(fmt.Sprint(follow), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, redirects atomic.Int32
			cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, _ []*http.Request) error {
				redirects.Add(1)
				if !follow {
					return http.ErrUseLastResponse
				}
				next.Header.Set("Accept", "redirect/media")
				return nil
			}
			cloud.Mux.HandleFunc(payloadContractPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				payloadContractWire(t, r, payloadContractPath, "text/plain", "test-token")
				w.Header().Set("Location", payloadContractBase+"payload-redirect")
				w.WriteHeader(307)
			})
			cloud.Mux.HandleFunc(payloadContractBase+"payload-redirect", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				payloadContractWire(t, r, payloadContractBase+"payload-redirect", "redirect/media", "test-token")
				_, _ = w.Write([]byte("redirect-payload"))
			})
			got, err := secrets.New(payloadContractClient(cloud)).GetPayload(context.Background(), "secret-id")
			if follow {
				if err != nil || !bytes.Equal(got, []byte("redirect-payload")) || calls.Load() != 2 {
					t.Fatalf("bytes=%x err=%v calls=%d", got, err, calls.Load())
				}
			} else {
				if got != nil || !gophercloud.ResponseCodeIs(err, 307) || calls.Load() != 1 {
					t.Fatalf("bytes=%x err=%v calls=%d", got, err, calls.Load())
				}
				payloadContractOperation(t, err)
			}
			if redirects.Load() != 1 {
				t.Fatalf("redirect callbacks=%d", redirects.Load())
			}
		})
	}
}
