package blockstorage_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

type detachVolumeContractTransport func(*http.Request) (*http.Response, error)

func (f detachVolumeContractTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
func detachVolumeContractResponse(code int, body io.ReadCloser, proof string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"X-Proof": {proof}}, Body: body}
}

type detachVolumeContractBody struct {
	data                  *strings.Reader
	readError, closeError error
	onClose               func()
	once                  sync.Once
	closes                atomic.Int32
}

func (b *detachVolumeContractBody) Read(p []byte) (int, error) {
	if b.readError != nil {
		n, _ := b.data.Read(p)
		cause := b.readError
		b.readError = nil
		return n, cause
	}
	return b.data.Read(p)
}
func (b *detachVolumeContractBody) Close() error {
	b.closes.Add(1)
	b.once.Do(func() {
		if b.onClose != nil {
			b.onClose()
		}
	})
	return b.closeError
}

func TestDetachVolumeTransportAcceptedReadCloseContextAndSourceFailuresRetainOpaqueProof(t *testing.T) {
	for _, phase := range []string{"delete", "poll"} {
		for _, mode := range []string{"read", "close", "context", "source"} {
			t.Run(phase+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				nova, cinder := detachVolumeContractClients(cloud)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("accepted " + phase + " " + mode + " failure")
				body := []byte{0, 0xff, 'p'}
				failStep := int32(1)
				code := 202
				if phase == "poll" {
					body = []byte(detachVolumeContractObservation("available", `null`))
					failStep = 2
					code = 200
				}
				broken := &detachVolumeContractBody{data: strings.NewReader(string(body))}
				if mode == "read" {
					broken.readError = cause
				}
				if mode == "close" {
					broken.closeError = cause
				}
				if mode == "context" {
					broken.onClose = func() { cancel(cause) }
				}
				if mode == "source" {
					broken.onClose = func() { nova.Microversion = "2.99" }
				}
				var calls, retries atomic.Int32
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries.Add(1)
					return err
				}
				cloud.Provider.HTTPClient.Transport = detachVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
					step := calls.Add(1)
					if step == failStep {
						return detachVolumeContractResponse(code, broken, phase+"-failure"), nil
					}
					if step == 1 {
						return detachVolumeContractResponse(204, io.NopCloser(strings.NewReader("")), "deleted"), nil
					}
					t.Error("accepted error caused resend, poll or cleanup", r.Method, r.URL)
					return nil, errors.New("unexpected HTTP")
				})
				result, err := blockstorage.DetachVolume(ctx, nova, cinder, detachVolumeContractInput())
				var accepted *resource.ResponseError
				if result == nil || err == nil || !errors.As(err, &accepted) || accepted.StatusCode != code || !bytes.Equal(accepted.Body, body) || accepted.Header.Get("X-Proof") != phase+"-failure" || calls.Load() != failStep || retries.Load() != 0 || broken.closes.Load() != 1 || result.Ready != nil {
					t.Fatalf("result=%+v error=%v calls=%d retries=%d closes=%d", result, err, calls.Load(), retries.Load(), broken.closes.Load())
				}
				if mode == "source" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, cause) {
					t.Fatal("accepted error cause lost", err)
				}
				if mode == "context" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				var proofBody []byte
				var proofHeader http.Header
				if phase == "delete" {
					if result.Deleted == nil || result.Deleted.StatusCode != 202 || result.LastAccepted != nil {
						t.Fatal(result)
					}
					proofBody, proofHeader = result.Deleted.Body, result.Deleted.Header
				} else {
					detachVolumeContractDeleted(t, result, 204, nil)
					if result.LastAccepted == nil || result.LastAccepted.Volume != nil {
						t.Fatal(result)
					}
					proofBody, proofHeader = result.LastAccepted.Body, result.LastAccepted.Header
				}
				if !bytes.Equal(proofBody, body) || proofHeader.Get("X-Proof") != phase+"-failure" {
					t.Fatal("accepted phase proof lost", result)
				}
				accepted.Body[0] = '!'
				accepted.Header.Set("X-Proof", "caller")
				if !bytes.Equal(proofBody, body) || proofHeader.Get("X-Proof") != phase+"-failure" {
					t.Fatal("error and result proof alias")
				}
				detachVolumeContractOperation(t, err)
			})
		}
	}
}

func TestDetachVolumeTransportSnapshotsHeadersLiveDistinctProvidersAndSourceGuards(t *testing.T) {
	t.Run("entry headers and live independent tokens", func(t *testing.T) {
		cloud := testcloud.New(t)
		nova, cinder := detachVolumeContractClients(cloud)
		provider := &gophercloud.ProviderClient{HTTPClient: *cloud.Server.Client()}
		provider.UseTokenLock()
		provider.SetToken("cinder-initial")
		cinder.ProviderClient = provider
		nova.MoreHeaders["openstack-api-version"] = "compute 2.79"
		cinder.MoreHeaders["openstack-api-version"] = "volume 3.60"
		nova.Endpoint = strings.TrimSuffix(nova.Endpoint, "/")
		var calls, options atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if calls.Add(1) == 1 {
				detachVolumeContractWire(t, r, http.MethodDelete, detachVolumeContractDeletePath, "nova-live")
				w.Header().Set("X-Proof", "deleted")
				w.WriteHeader(204)
				return
			}
			detachVolumeContractWire(t, r, http.MethodGet, detachVolumeContractVolumePath, "cinder-live")
			testcloud.JSON(w, 200, detachVolumeContractObservation("available", `null`))
		})
		var retained *blockstorage.DetachVolumeOpts
		result, err := blockstorage.DetachVolume(context.Background(), nova, cinder, detachVolumeContractInput(), func(value *blockstorage.DetachVolumeOpts) error {
			options.Add(1)
			retained = value
			nova.MoreHeaders["x-source"] = "changed-nova"
			cinder.MoreHeaders["x-source"] = "changed-cinder"
			cloud.Provider.SetToken("nova-live")
			provider.SetToken("cinder-live")
			return nil
		}, func(*blockstorage.DetachVolumeOpts) error {
			options.Add(1)
			disabled := false
			retained.Wait = &disabled
			return nil
		})
		detachVolumeContractDeleted(t, result, 204, nil)
		if err != nil || result.Ready == nil || calls.Load() != 2 || options.Load() != 2 || nova.MoreHeaders["x-source"] != "changed-nova" {
			t.Fatalf("result=%+v error=%v calls=%d options=%d", result, err, calls.Load(), options.Load())
		}
	})
	mutations := []struct {
		name   string
		change func(*gophercloud.ServiceClient)
	}{
		{"provider", func(c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} }},
		{"endpoint", func(c *gophercloud.ServiceClient) { c.Endpoint += "changed/" }},
		{"resource base", func(c *gophercloud.ServiceClient) { c.ResourceBase += "changed/" }},
		{"type", func(c *gophercloud.ServiceClient) { c.Type = "image" }},
		{"microversion", func(c *gophercloud.ServiceClient) { c.Microversion = "9.99" }},
	}
	for _, mutation := range mutations {
		t.Run("option/"+mutation.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, _ := detachVolumeContractClients(cloud)
			var calls, options atomic.Int32
			cloud.Provider.HTTPClient.Transport = detachVolumeContractTransport(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return nil, errors.New("unexpected request")
			})
			result, err := blockstorage.DetachVolume(context.Background(), nova, nil, detachVolumeContractInput(), blockstorage.WithDetachVolumeWait(false), func(*blockstorage.DetachVolumeOpts) error { options.Add(1); mutation.change(nova); return nil }, func(*blockstorage.DetachVolumeOpts) error { options.Add(1); return nil })
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 || options.Load() != 1 {
				t.Fatalf("result=%+v error=%v calls=%d options=%d", result, err, calls.Load(), options.Load())
			}
		})
		t.Run("accepted delete/"+mutation.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := detachVolumeContractClients(cloud)
			var calls atomic.Int32
			opaque := []byte{0, 0xff, 'x'}
			cloud.Provider.HTTPClient.Transport = detachVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				detachVolumeContractWire(t, r, http.MethodDelete, detachVolumeContractDeletePath, "test-token")
				mutation.change(cinder)
				return detachVolumeContractResponse(202, io.NopCloser(strings.NewReader(string(opaque))), "deleted"), nil
			})
			result, err := blockstorage.DetachVolume(context.Background(), nova, cinder, detachVolumeContractInput())
			detachVolumeContractDeleted(t, result, 202, opaque)
			var accepted *resource.ResponseError
			if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &accepted) || accepted.StatusCode != 202 || result.LastAccepted != nil || result.Ready != nil || calls.Load() != 1 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
		})
	}
	t.Run("progress changes source after accepted poll", func(t *testing.T) {
		cloud := testcloud.New(t)
		nova, cinder := detachVolumeContractClients(cloud)
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = detachVolumeContractTransport(func(*http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 {
				return detachVolumeContractResponse(204, io.NopCloser(strings.NewReader("")), "deleted"), nil
			}
			return detachVolumeContractResponse(200, io.NopCloser(strings.NewReader(detachVolumeContractObservation("detaching", `[]`))), "poll"), nil
		})
		result, err := blockstorage.DetachVolume(context.Background(), nova, cinder, detachVolumeContractInput(), blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{ProgressCallback: func(int) error { nova.Microversion = "2.99"; return nil }}))
		detachVolumeContractDeleted(t, result, 204, nil)
		if !errors.Is(err, resource.ErrInvalidOption) || result.LastAccepted == nil || result.Ready != nil || calls.Load() != 2 {
			t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
		}
	})
}

func TestDetachVolumeTransportWaitBudgetStartsAfterDeleteAndZeroRemainsUnlimited(t *testing.T) {
	t.Run("positive budget excludes opaque body handling", func(t *testing.T) {
		cloud := testcloud.New(t)
		nova, cinder := detachVolumeContractClients(cloud)
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = detachVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 {
				if _, bounded := r.Context().Deadline(); bounded {
					t.Error("wait-only deadline leaked into DELETE")
				}
				body := &detachVolumeContractBody{data: strings.NewReader("opaque"), onClose: func() { time.Sleep(100 * time.Millisecond) }}
				return detachVolumeContractResponse(202, body, "deleted"), nil
			}
			return detachVolumeContractResponse(200, io.NopCloser(strings.NewReader(detachVolumeContractObservation("available", `null`))), "ready"), nil
		})
		timeout := 50 * time.Millisecond
		result, err := blockstorage.DetachVolume(context.Background(), nova, cinder, detachVolumeContractInput(), blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{Timeout: &timeout}))
		detachVolumeContractDeleted(t, result, 202, []byte("opaque"))
		if err != nil || result.Ready == nil || calls.Load() != 2 {
			t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
		}
	})
	t.Run("zero budget permits nonterminal then next poll", func(t *testing.T) {
		cloud := testcloud.New(t)
		nova, cinder := detachVolumeContractClients(cloud)
		var calls, callbacks atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			switch calls.Add(1) {
			case 1:
				w.Header().Set("X-Proof", "deleted")
				w.WriteHeader(204)
			case 2:
				testcloud.JSON(w, 200, detachVolumeContractObservation("detaching", `null`))
			case 3:
				testcloud.JSON(w, 200, detachVolumeContractObservation("available", `null`))
			default:
				t.Error("unexpected zero-budget phase", r.Method, r.URL)
				w.WriteHeader(500)
			}
		})
		zero, interval := time.Duration(0), time.Millisecond
		result, err := blockstorage.DetachVolume(context.Background(), nova, cinder, detachVolumeContractInput(), blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{Timeout: &zero, PollInterval: &interval, ProgressCallback: func(int) error { callbacks.Add(1); return nil }}))
		detachVolumeContractDeleted(t, result, 204, nil)
		if err != nil || result.Ready == nil || calls.Load() != 3 || callbacks.Load() != 1 {
			t.Fatalf("result=%+v error=%v calls=%d callbacks=%d", result, err, calls.Load(), callbacks.Load())
		}
	})
}

func TestDetachVolumeTransportNativeAuthBackoffAndRetryPreserveBodylessFixedDelete(t *testing.T) {
	cloud := testcloud.New(t)
	nova, cinder := detachVolumeContractClients(cloud)
	var calls, deletes, reauths, backoffs, retries atomic.Int32
	cloud.Provider.ReauthFunc = func(context.Context) error { reauths.Add(1); cloud.Provider.SetToken("refreshed-token"); return nil }
	cloud.Provider.RetryBackoffFunc = func(_ context.Context, code *gophercloud.ErrUnexpectedResponseCode, _ error, count uint) error {
		backoffs.Add(1)
		if code.Actual != 429 || count != 1 {
			t.Errorf("backoff=%+v count=%d", code, count)
		}
		return nil
	}
	cloud.Provider.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, err error, count uint) error {
		retries.Add(1)
		if !gophercloud.ResponseCodeIs(err, 503) {
			return err
		}
		if method != http.MethodDelete || target != cloud.Server.URL+detachVolumeContractDeletePath || count != 2 || options.JSONBody != nil || options.RawBody != nil || options.JSONResponse != nil || !options.KeepResponseBody || !reflect.DeepEqual(options.OkCodes, []int{202, 204}) {
			t.Errorf("retry method=%s target=%s count=%d options=%+v", method, target, count, options)
		}
		if options.MoreHeaders == nil {
			options.MoreHeaders = map[string]string{}
		}
		options.MoreHeaders["X-Retry"] = "native"
		return nil
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method == http.MethodGet {
			detachVolumeContractWire(t, r, http.MethodGet, detachVolumeContractVolumePath, "refreshed-token")
			testcloud.JSON(w, 200, detachVolumeContractObservation("available", `null`))
			return
		}
		count := deletes.Add(1)
		token := "refreshed-token"
		if count == 1 {
			token = "test-token"
		}
		detachVolumeContractWire(t, r, http.MethodDelete, detachVolumeContractDeletePath, token)
		switch count {
		case 1:
			testcloud.JSON(w, 401, `{"error":"expired"}`)
		case 2:
			testcloud.JSON(w, 429, `{"error":"throttled"}`)
		case 3:
			testcloud.JSON(w, 503, `{"error":"retry"}`)
		default:
			if r.Header.Get("X-Retry") != "native" {
				t.Error("native retry header lost")
			}
			w.Header().Set("X-Proof", "deleted")
			w.WriteHeader(204)
		}
	})
	result, err := blockstorage.DetachVolume(context.Background(), nova, cinder, detachVolumeContractInput())
	detachVolumeContractDeleted(t, result, 204, nil)
	if err != nil || result.Ready == nil || calls.Load() != 5 || deletes.Load() != 4 || reauths.Load() != 1 || backoffs.Load() != 1 || retries.Load() != 1 {
		t.Fatalf("result=%+v error=%v calls=%d deletes=%d reauths=%d backoffs=%d retries=%d", result, err, calls.Load(), deletes.Load(), reauths.Load(), backoffs.Load(), retries.Load())
	}
}

func TestDetachVolumeTransportNativeRetryCannotAddBodyOrExpandAcceptedProof(t *testing.T) {
	for _, tc := range []struct {
		name       string
		mutate     func(*gophercloud.RequestOpts)
		secondCode int
	}{
		{"JSON body", func(o *gophercloud.RequestOpts) { o.JSONBody = map[string]any{"force": true} }, 204},
		{"raw body", func(o *gophercloud.RequestOpts) { o.RawBody = strings.NewReader("changed") }, 204},
		{"JSON response target", func(o *gophercloud.RequestOpts) { o.JSONResponse = new(any) }, 204},
		{"response ownership", func(o *gophercloud.RequestOpts) { o.KeepResponseBody = false }, 204},
		{"success policy expansion", func(o *gophercloud.RequestOpts) { o.OkCodes = append(o.OkCodes, 200) }, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, _ := detachVolumeContractClients(cloud)
			var calls atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, err error, _ uint) error {
				if !gophercloud.ResponseCodeIs(err, 503) {
					return err
				}
				tc.mutate(options)
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				detachVolumeContractWire(t, r, http.MethodDelete, detachVolumeContractDeletePath, "test-token")
				if calls.Add(1) == 1 {
					testcloud.JSON(w, 503, `{"error":"retry"}`)
				} else {
					testcloud.JSON(w, tc.secondCode, `{"accepted":true}`)
				}
			})
			result, err := blockstorage.DetachVolume(context.Background(), nova, nil, detachVolumeContractInput(), blockstorage.WithDetachVolumeWait(false))
			if err == nil || result == nil || result.Deleted != nil || result.LastAccepted != nil || result.Ready != nil {
				t.Fatalf("result=%+v error=%v", result, err)
			}
			if tc.secondCode == 200 {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 200 || !reflect.DeepEqual(native.Expected, []int{202, 204}) || calls.Load() != 2 {
					t.Fatalf("expanded policy error=%v calls=%d", err, calls.Load())
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
				t.Fatalf("body ownership error=%v calls=%d", err, calls.Load())
			}
		})
	}
	t.Run("redirect cannot replace fixed delete target", func(t *testing.T) {
		cloud := testcloud.New(t)
		nova, _ := detachVolumeContractClients(cloud)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.Method != http.MethodDelete || r.URL.Path != detachVolumeContractDeletePath {
				t.Error("redirect sent outside fixed scope", r.Method, r.URL)
			}
			w.Header().Set("Location", cloud.Server.URL+"/other/target")
			w.WriteHeader(302)
		})
		result, err := blockstorage.DetachVolume(context.Background(), nova, nil, detachVolumeContractInput(), blockstorage.WithDetachVolumeWait(false))
		if result == nil || result.Deleted != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
			t.Fatalf("redirect result=%+v error=%v calls=%d", result, err, calls.Load())
		}
	})
}

func TestDetachVolumeTransportRejectedIOAndContextDoNotInventAcceptedAcknowledgement(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelled), func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, _ := detachVolumeContractClients(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("detach transport failed")
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = detachVolumeContractTransport(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				if cancelled {
					cancel(cause)
				}
				return nil, cause
			})
			result, err := blockstorage.DetachVolume(ctx, nova, nil, detachVolumeContractInput(), blockstorage.WithDetachVolumeWait(false))
			var accepted *resource.ResponseError
			if result == nil || result.Deleted != nil || result.LastAccepted != nil || result.Ready != nil || !errors.Is(err, cause) || cancelled && !errors.Is(err, context.Canceled) || errors.As(err, &accepted) || calls.Load() != 1 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
			detachVolumeContractOperation(t, err)
		})
	}
}

func TestDetachVolumeTransportBlockedPollCancellationKeepsPriorAcceptedObservation(t *testing.T) {
	for _, callerCancelled := range []bool{false, true} {
		t.Run(fmt.Sprint(callerCancelled), func(t *testing.T) {
			cloud := testcloud.New(t)
			nova, cinder := detachVolumeContractClients(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("caller canceled blocked detach poll")
			started := make(chan struct{})
			finished := make(chan struct{})
			if callerCancelled {
				go func() {
					defer close(finished)
					select {
					case <-started:
						cancel(cause)
					case <-ctx.Done():
					}
				}()
			} else {
				close(finished)
			}
			var calls atomic.Int32
			firstPoll := detachVolumeContractObservation("detaching", `null`)
			cloud.Provider.HTTPClient.Transport = detachVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
				switch calls.Add(1) {
				case 1:
					detachVolumeContractWire(t, r, http.MethodDelete, detachVolumeContractDeletePath, "test-token")
					return detachVolumeContractResponse(204, io.NopCloser(strings.NewReader("")), "deleted"), nil
				case 2:
					detachVolumeContractWire(t, r, http.MethodGet, detachVolumeContractVolumePath, "test-token")
					return detachVolumeContractResponse(200, io.NopCloser(strings.NewReader(firstPoll)), "last-accepted"), nil
				case 3:
					detachVolumeContractWire(t, r, http.MethodGet, detachVolumeContractVolumePath, "test-token")
					close(started)
					<-r.Context().Done()
					return nil, r.Context().Err()
				default:
					t.Error("blocked poll caused extra request", r.Method, r.URL)
					return nil, errors.New("unexpected HTTP")
				}
			})
			timeout, interval := 200*time.Millisecond, time.Millisecond
			policy := blockstorage.DetachVolumeWaitOpts{Timeout: &timeout, PollInterval: &interval}
			if callerCancelled {
				policy.Timeout = nil
			}
			result, err := blockstorage.DetachVolume(ctx, nova, cinder, detachVolumeContractInput(), blockstorage.WithDetachVolumeWaitPolicy(policy))
			cancel(nil)
			<-finished
			detachVolumeContractDeleted(t, result, 204, nil)
			want := error(context.DeadlineExceeded)
			if callerCancelled {
				want = context.Canceled
			}
			var accepted *resource.ResponseError
			if !errors.Is(err, want) || callerCancelled && !errors.Is(err, cause) || errors.As(err, &accepted) || result.LastAccepted == nil || result.LastAccepted.Volume == nil || string(result.LastAccepted.Body) != firstPoll || result.LastAccepted.Header.Get("X-Proof") != "last-accepted" || result.Ready != nil || calls.Load() != 3 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
			}
			detachVolumeContractOperation(t, err)
		})
	}
}

func TestDetachVolumeTransportLaterFailureKeepsPriorAcceptedProof(t *testing.T) {
	cloud := testcloud.New(t)
	nova, cinder := detachVolumeContractClients(cloud)
	cause := errors.New("later Cinder transport failed")
	var calls atomic.Int32
	body := detachVolumeContractObservation("detaching", `null`)
	cloud.Provider.HTTPClient.Transport = detachVolumeContractTransport(func(r *http.Request) (*http.Response, error) {
		switch calls.Add(1) {
		case 1:
			detachVolumeContractWire(t, r, http.MethodDelete, detachVolumeContractDeletePath, "test-token")
			return detachVolumeContractResponse(204, io.NopCloser(strings.NewReader("")), "deleted"), nil
		case 2:
			detachVolumeContractWire(t, r, http.MethodGet, detachVolumeContractVolumePath, "test-token")
			return detachVolumeContractResponse(200, io.NopCloser(strings.NewReader(body)), "last-accepted"), nil
		case 3:
			detachVolumeContractWire(t, r, http.MethodGet, detachVolumeContractVolumePath, "test-token")
			return nil, cause
		default:
			t.Error("transport failure caused extra request", r.Method, r.URL)
			return nil, errors.New("unexpected request")
		}
	})
	interval := time.Millisecond
	result, err := blockstorage.DetachVolume(context.Background(), nova, cinder, detachVolumeContractInput(), blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{PollInterval: &interval}))
	detachVolumeContractDeleted(t, result, 204, nil)
	var accepted *resource.ResponseError
	if !errors.Is(err, cause) || errors.As(err, &accepted) || result.LastAccepted == nil || result.LastAccepted.Volume == nil || string(result.LastAccepted.Body) != body || result.LastAccepted.Header.Get("X-Proof") != "last-accepted" || result.Ready != nil || calls.Load() != 3 {
		t.Fatalf("result=%+v error=%v calls=%d", result, err, calls.Load())
	}
	detachVolumeContractOperation(t, err)
}
