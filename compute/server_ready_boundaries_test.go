package compute_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestWaitForServerRawFailuresKeepLastMatchingModelAndCauses(t *testing.T) {
	for _, scenario := range []string{"empty", "nil timeout", "wrong ID", "malformed", "ERROR", "first HTTP", "later HTTP", "progress source", "progress cancel"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAutomaticFixture(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("readiness canceled")
			opts := append(serverReadyOptions(), compute.WithServerReadyAutomaticIPOptions(compute.WithAutomaticIPEnabled(false)), compute.WithServerReadyWaitOptions(resource.WithTimeout(20*time.Millisecond), resource.WithFailureStates(), resource.WithProgressCallback(func(int) {
				if scenario == "progress source" {
					f.service.API = nil
				}
				if scenario == "progress cancel" {
					cancel(cause)
				}
			})))
			f.rawBody = func(n int32) (int, string) {
				switch scenario {
				case "empty":
					return 200, `{"server":{"id":"server","status":"ACTIVE","name":"empty raw","addresses":{"private":[]}}}`
				case "nil timeout":
					return 200, `{"server":{"id":"server","status":"ACTIVE","name":"nil raw","addresses":null}}`
				case "wrong ID":
					return 200, `{"server":{"id":"other","status":"ACTIVE","addresses":` + autoFixed + `}}`
				case "malformed":
					return 200, `{"server":`
				case "ERROR":
					return 203, `{"server":{"id":"server","status":"ERROR","fault":{"message":"raw boot failure"}}}`
				case "first HTTP":
					return 404, `{"error":"missing"}`
				case "later HTTP":
					if n > 1 {
						return 403, `{"error":"denied"}`
					}
				}
				return 200, `{"server":{"id":"server","status":"BUILD","name":"last raw","progress":25}}`
			}
			server := automaticServer(t, autoFloating)
			server.Name = "supplied"
			result, err := f.service.WaitForServer(ctx, compute.AutomaticFloatingIPRequest{Server: server}, opts...)
			if result == nil || err == nil || result.Server.ID != "server" || result.Assignment != nil || f.posts.Load() != 0 || f.roles.Load() != 0 {
				t.Fatal(result, err, f.posts.Load(), f.roles.Load())
			}
			switch scenario {
			case "empty":
				if !errors.Is(err, compute.ErrServerAddressesUnavailable) || result.Server.Name != "empty raw" || f.raw.Load() != 1 {
					t.Fatal(result, err, f.raw.Load())
				}
			case "nil timeout":
				if !errors.Is(err, context.DeadlineExceeded) || result.Server.Name != "nil raw" {
					t.Fatal(result, err)
				}
			case "ERROR":
				if !errors.Is(err, resource.ErrFailedState) || result.Server.Fault.Message != "raw boot failure" || f.raw.Load() != 1 {
					t.Fatal(result, err, f.raw.Load())
				}
			case "progress cancel":
				if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || result.Server.Name != "last raw" || f.raw.Load() != 1 {
					t.Fatal(result, err, f.raw.Load())
				}
			case "progress source":
				if !errors.Is(err, resource.ErrInvalidOption) || result.Server.Name != "last raw" || f.raw.Load() != 1 {
					t.Fatal(result, err, f.raw.Load())
				}
			case "first HTTP", "later HTTP":
				var proof gophercloud.ErrUnexpectedResponseCode
				wantCode, wantName, wantReads := 404, "supplied", int32(1)
				if scenario == "later HTTP" {
					wantCode, wantName, wantReads = 403, "last raw", 2
				}
				if !errors.As(err, &proof) || proof.Actual != wantCode || result.Server.Name != wantName || f.raw.Load() != wantReads {
					t.Fatal(result, err, proof, f.raw.Load())
				}
			default:
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || result.Server.Name != "supplied" || f.raw.Load() != 1 {
					t.Fatal(result, err, proof, f.raw.Load())
				}
			}
		})
	}
}

func TestWaitForServerAcceptedGETCloseKeepsRawModelAndResponseProof(t *testing.T) {
	for _, scenario := range []string{"close", "cancel", "source", "ancestor"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAutomaticFixture(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("accepted GET stopped")
			changed := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if changed {
					return cause
				}
				return nil
			})
			f.rawBody = func(int32) (int, string) {
				return 203, `{"server":{"id":"server","status":"ACTIVE","name":"accepted raw","addresses":` + autoFloating + `}}`
			}
			base := f.cloud.Provider.HTTPClient.Transport
			f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
				response, err := base.RoundTrip(r)
				if err == nil && r.URL.Path == "/v2.1/servers/server" {
					response.Header.Set("X-Readiness-Proof", "accepted")
					response.Body = automaticCloseBody{ReadCloser: response.Body, close: func() error {
						switch scenario {
						case "cancel":
							cancel(cause)
						case "source":
							f.service.API = nil
						case "ancestor":
							changed = true
						default:
							return cause
						}
						return nil
					}}
				}
				return response, err
			})
			retries := 0
			f.cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return cause
			}
			result, err := f.service.WaitForServer(ctx, compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, serverReadyOptions()...)
			var proof *resource.ResponseError
			if result == nil || result.Server.Name != "accepted raw" || result.Assignment != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || proof.Header.Get("X-Readiness-Proof") != "accepted" || f.raw.Load() != 1 || f.posts.Load() != 0 || retries != 0 {
				t.Fatal(result, err, proof, f.raw.Load(), retries)
			}
			if scenario != "source" && !errors.Is(err, cause) || scenario == "source" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestWaitForServerSharesWholeDeadlineAndRetainsAssignmentOnCancel(t *testing.T) {
	f := newAutomaticFixture(t)
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("stop raw convergence")
	f.rawBody = func(int32) (int, string) {
		return 200, `{"server":{"id":"server","status":"ACTIVE","name":"matching raw","addresses":` + autoFixed + `}}`
	}
	var deadlines []time.Time
	base := f.cloud.Provider.HTTPClient.Transport
	f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
		deadline, ok := r.Context().Deadline()
		if !ok {
			t.Error("missing deadline", r.URL)
		}
		deadlines = append(deadlines, deadline)
		return base.RoundTrip(r)
	})
	opts := append(serverReadyOptions(), compute.WithServerReadyAutomaticIPOptions(compute.WithAutomaticIPTimeout(250*time.Millisecond), compute.WithAutomaticIPProgress(func(value *compute.Server) error {
		value.ID, value.Name = "callback changed", "callback changed"
		value.Addresses["private"] = []any{}
		cancel(cause)
		return nil
	})))
	result, err := f.service.WaitForServer(ctx, compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFloating)}, opts...)
	if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || result == nil || result.Server.ID != "server" || result.Server.Name != "matching raw" || result.Assignment == nil || !result.Assignment.Allocated || result.Assignment.FloatingIP.Status != "ACTIVE" || result.Observed || f.posts.Load() != 1 || f.raw.Load() != 2 {
		t.Fatal(result, err, f.posts.Load(), f.raw.Load())
	}
	if len(deadlines) < 5 {
		t.Fatal(deadlines)
	}
	for _, deadline := range deadlines {
		if !deadline.Equal(deadlines[0]) {
			t.Fatal("overall budget restarted", deadlines)
		}
	}
}

func TestWaitForServerDefaultWholeBudgetIs180Seconds(t *testing.T) {
	f := newAutomaticFixture(t)
	start := time.Now()
	var remaining time.Duration
	base := f.cloud.Provider.HTTPClient.Transport
	f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
		deadline, ok := r.Context().Deadline()
		if !ok {
			t.Error("missing default whole deadline")
		}
		remaining = deadline.Sub(start)
		return base.RoundTrip(r)
	})
	result, err := f.service.WaitForServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, compute.WithServerReadyAutomaticIPOptions(compute.WithAutomaticIPEnabled(false)))
	if err != nil || result == nil || remaining < 179*time.Second || remaining > 181*time.Second || f.raw.Load() != 1 {
		t.Fatal(result, err, remaining, f.raw.Load())
	}
}
