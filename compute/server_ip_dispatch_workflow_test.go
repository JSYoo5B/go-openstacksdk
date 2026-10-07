package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/compute"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func TestServerIPDispatchAllWorkflowConsumersUseTheSameOrderedPolicy(t *testing.T) {
	for _, mode := range []string{"Get async", "Get sync", "Wait", "Create"} {
		t.Run(mode, func(t *testing.T) {
			f := newDispatchFixture(t, "")
			opts := append(dispatchOptions(), compute.WithFloatingIPAddresses(dispatchAddresses["a"], dispatchAddresses["b"]), compute.WithAutomaticEnsureOptions(network.WithEnsureNoWait()), compute.WithAutomaticIPEnabled(false))
			input := dispatchInput(t)
			input.Server = automaticServer(t, autoFloating)
			ready := []compute.ServerReadyOption{compute.WithServerReadyAutomaticIPOptions(opts...), compute.WithServerReadyWaitOptions(resource.WithPollInterval(time.Millisecond))}
			var result *compute.AutomaticServerIPResult
			var err error
			switch mode {
			case "Get async":
				computeGets := 0
				f.service = compute.New(nil, compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) { return f.network, nil }, AddressCompute: func(context.Context) (*gophercloud.ServiceClient, error) {
					computeGets++
					return nil, errors.New("async compute must stay lazy")
				}})
				result, err = f.service.GetActiveServer(context.Background(), input, ready...)
				if computeGets != 0 {
					t.Fatal(computeGets)
				}
			case "Get sync":
				result, err = f.service.GetActiveServer(context.Background(), input, append(ready, compute.WithActiveServerWait(true))...)
			case "Wait":
				input.Server.Status = "ERROR"
				result, err = f.service.WaitForServer(context.Background(), input, ready...)
			case "Create":
				f.cloud.Mux.HandleFunc("POST /v2.1/servers", func(w http.ResponseWriter, r *http.Request) {
					testServerCreateBody(t, r)
					testcloud.JSON(w, 202, `{"server":{"id":"server","adminPass":"creation-only"}}`)
				})
				created, createErr := f.service.CreateWithAutomaticFloatingIP(context.Background(), automaticCreateRequest(), compute.AutomaticServerCreateOptions{Server: automaticCreateOptions().Server, AutomaticIP: opts})
				err = createErr
				if created == nil || created.Creation.AdminPass != "creation-only" || created.Server.AdminPass != "" {
					t.Fatal(created, err)
				}
				result = created.Automatic
			}
			if err != nil || result == nil || result.Mode != compute.ServerIPExplicit || len(result.Attempts) != 2 || result.Assignment.FloatingIP.ID != "ip-b" || f.posts.Load() != 0 {
				t.Fatal(result, err, f.trace())
			}
			wantRaw, status, observed := int32(2), "ACTIVE", true
			if mode == "Get async" {
				wantRaw, status, observed = 0, "DOWN", false
			}
			if mode == "Wait" || mode == "Create" {
				wantRaw = 3
			}
			if result.Observed != observed || f.raw.Load() != wantRaw || result.Assignment.FloatingIP.Status != status {
				t.Fatal(result, f.trace())
			}
			for _, attempt := range result.Attempts {
				if !attempt.Completed || attempt.Observed != observed || attempt.Error != nil {
					t.Fatal(attempt)
				}
			}
			if mode == "Get async" {
				trace := strings.Join(f.trace(), ",")
				if strings.Count(trace, "get:a") != 1 || strings.Count(trace, "get:b") != 1 {
					t.Fatal("async readiness polling", trace)
				}
			}
		})
	}
}

func TestServerIPDispatchCreateRetainsCreationFirstIPAndSecondFailure(t *testing.T) {
	f := newDispatchFixture(t, "missing")
	posts := 0
	f.cloud.Mux.HandleFunc("POST /v2.1/servers", func(w http.ResponseWriter, r *http.Request) {
		posts++
		testcloud.JSON(w, 202, `{"server":{"id":"server","adminPass":"created"}}`)
	})
	opts := append(dispatchOptions(), compute.WithFloatingIPAddresses(dispatchAddresses["a"], dispatchAddresses["b"], dispatchAddresses["c"]))
	created, err := f.service.CreateWithAutomaticFloatingIP(context.Background(), automaticCreateRequest(), compute.AutomaticServerCreateOptions{Server: automaticCreateOptions().Server, AutomaticIP: opts})
	if !errors.Is(err, resource.ErrNotFound) || created == nil || created.Creation.AdminPass != "created" || created.Server.Name != "raw-2" || created.Automatic == nil || created.Automatic.Observed || len(created.Automatic.Attempts) != 2 || !created.Automatic.Attempts[0].Completed || !created.Automatic.Attempts[0].Observed || created.Automatic.Assignment.FloatingIP.ID != "ip-a" || posts != 1 || f.posts.Load() != 0 || f.raw.Load() != 2 {
		t.Fatal(created, err, f.trace())
	}
}

func TestServerIPDispatchPureInvalidAndKnownUnsupportedCreateStopBeforePOST(t *testing.T) {
	for _, scenario := range []string{"IP", "pool", "Nova", "None"} {
		t.Run(scenario, func(t *testing.T) {
			f := newDispatchFixture(t, "")
			option := compute.WithFloatingIPAddresses("malformed")
			want := resource.ErrInvalidOption
			opts := dispatchOptions()
			if scenario == "pool" {
				option = compute.WithFloatingIPPool(resource.ID("bad/pool"))
			}
			if scenario == "Nova" || scenario == "None" {
				option = compute.WithFloatingIPAddresses(dispatchAddresses["a"])
				source := compute.FloatingIPNova
				if scenario == "None" {
					source = compute.FloatingIPNone
				}
				opts = append(opts, compute.WithAutomaticAddressOptions(compute.WithFloatingIPSource(source)))
				want = resource.ErrUnsupported
			}
			created, err := f.service.CreateWithAutomaticFloatingIP(context.Background(), automaticCreateRequest(), compute.AutomaticServerCreateOptions{Server: automaticCreateOptions().Server, AutomaticIP: append(opts, option)})
			if created != nil || !errors.Is(err, want) || len(f.trace()) != 0 {
				t.Fatal(created, err, f.trace())
			}
		})
	}
}

func TestServerIPDispatchExplicitSelectorsPreserveGetReadinessBoundaries(t *testing.T) {
	for _, scenario := range []string{"BUILD", "ERROR", "empty"} {
		f := newDispatchFixture(t, "")
		input := dispatchInput(t)
		if scenario == "empty" {
			input.Server.Addresses = nil
		} else {
			input.Server.Status = scenario
		}
		result, err := f.service.GetActiveServer(context.Background(), input, compute.WithServerReadyAutomaticIPOptions(compute.WithFloatingIPAddresses(dispatchAddresses["a"])))
		if len(f.trace()) != 0 {
			t.Fatal(f.trace())
		}
		switch scenario {
		case "BUILD":
			if result != nil || err != nil {
				t.Fatal(result, err)
			}
		case "ERROR":
			if result == nil || !errors.Is(err, resource.ErrFailedState) {
				t.Fatal(result, err)
			}
		case "empty":
			if result == nil || !errors.Is(err, compute.ErrServerAddressesUnavailable) {
				t.Fatal(result, err)
			}
		}
	}
}

func TestServerIPDispatchOneDeadlineBoundsAllItemsAndPreflightsObservation(t *testing.T) {
	for _, scenario := range []string{"deadline", "capability"} {
		t.Run(scenario, func(t *testing.T) {
			f := newDispatchFixture(t, "raw cancel")
			opts := append(dispatchOptions(), compute.WithFloatingIPAddresses(dispatchAddresses["a"], dispatchAddresses["b"], dispatchAddresses["c"]))
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			if scenario == "deadline" {
				parentDeadline, _ := ctx.Deadline()
				var deadline time.Time
				base := f.cloud.Provider.HTTPClient.Transport
				f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
					got, exists := r.Context().Deadline()
					if deadline.IsZero() {
						deadline = got
					}
					if !exists || !got.Equal(deadline) || !got.Before(parentDeadline) {
						t.Errorf("stage enlarged or restarted deadline: %v / %v", got, deadline)
					}
					return base.RoundTrip(r)
				})
				opts = append(opts, compute.WithAutomaticIPTimeout(500*time.Millisecond))
				// Expire the same parent after the first item completed and the
				// second reached observation; no third lookup may start.
				ctx, cancel = context.WithCancel(ctx)
				defer cancel()
				opts = append(opts, compute.WithAutomaticIPProgress(func(*compute.Server) error { cancel(); return nil }))
			} else {
				f.service = compute.New(nil, compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) { return f.network, nil }})
			}
			result, err := f.service.EnsureServerFloatingIP(ctx, dispatchInput(t), opts...)
			if scenario == "deadline" {
				if !errors.Is(err, context.Canceled) || result == nil || result.Observed || len(result.Attempts) != 2 || !result.Attempts[0].Observed || result.Assignment.FloatingIP.ID != "ip-b" || f.raw.Load() != 2 {
					t.Fatal(result, err, f.trace())
				}
			} else if !errors.Is(err, resource.ErrUnsupported) || result == nil || len(result.Attempts) != 0 || len(f.trace()) != 0 {
				t.Fatal(result, err, f.trace())
			}
		})
	}
}

func testServerCreateBody(t *testing.T, r *http.Request) {
	t.Helper()
	var body struct{ Server map[string]any }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Error(err)
	}
	if body.Server["imageRef"] != "image" || body.Server["flavorRef"] != "flavor" {
		t.Error(body.Server)
	}
}

func beforeDispatchOption[T any](before func(), option func(*T) error) func(*T) error {
	return func(value *T) error { before(); return option(value) }
}

func TestServerIPDispatchPinsSuppliedServerBeforeOptionsAndPreparesPolicyOnce(t *testing.T) {
	for _, planOnly := range []bool{false, true} {
		f := newDispatchFixture(t, "")
		input := dispatchInput(t)
		outer, ensure, wait, address := 0, 0, 0, 0
		wrappedWait := beforeDispatchOption(func() { wait++ }, resource.WithPollInterval(time.Millisecond))
		wrappedEnsure := beforeDispatchOption(func() { ensure++ }, network.WithEnsureWait(wrappedWait))
		wrappedAddresses := beforeDispatchOption(func() { address++ }, compute.WithAddressReachability(false))
		wrappedList := beforeDispatchOption(func() { outer++; input.Server.ID, input.Server.Status = "caller-replaced", "SHUTOFF" }, compute.WithFloatingIPAddresses(dispatchAddresses["a"], dispatchAddresses["b"]))
		opts := []compute.AutomaticFloatingIPOption{wrappedList, compute.WithAutomaticEnsureOptions(wrappedEnsure), compute.WithAutomaticAddressOptions(wrappedAddresses), compute.WithAutomaticIPTimeout(time.Second)}
		if planOnly {
			decision, err := f.service.PlanServerFloatingIP(context.Background(), input, opts...)
			if err != nil || decision.Server.ID != "server" || len(decision.AttachmentSelections) != 2 {
				t.Fatal(decision, err)
			}
		} else {
			result, err := f.service.EnsureServerFloatingIP(context.Background(), input, opts...)
			if err != nil || !result.Observed || result.Server.ID != "server" || len(result.Attempts) != 2 {
				t.Fatal(result, err)
			}
		}
		if outer != 1 || ensure != 1 || wait != 1 || address != 1 {
			t.Fatal(outer, ensure, wait, address)
		}
	}
}

func TestServerIPDispatchAcceptedRawCloseKeepsCurrentServerAndAllKnownIPs(t *testing.T) {
	f := newDispatchFixture(t, "")
	cause := errors.New("second raw close")
	base := f.cloud.Provider.HTTPClient.Transport
	f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
		response, err := base.RoundTrip(r)
		if err == nil && r.URL.Path == "/v2.1/servers/server" && f.raw.Load() == 2 {
			response.Header.Set("X-Raw-Dispatch", "second")
			response.Body = automaticCloseBody{ReadCloser: response.Body, close: func() error { return cause }}
		}
		return response, err
	})
	opts := append(dispatchOptions(), compute.WithFloatingIPAddresses(dispatchAddresses["a"], dispatchAddresses["b"], dispatchAddresses["c"]))
	result, err := f.service.EnsureServerFloatingIP(context.Background(), dispatchInput(t), opts...)
	var proof *resource.ResponseError
	if !errors.Is(err, cause) || !errors.As(err, &proof) || proof.StatusCode != 203 || proof.Header.Get("X-Raw-Dispatch") != "second" || result == nil || result.Server.Name != "raw-2" || result.Observed || len(result.Attempts) != 2 || !result.Attempts[0].Observed || result.Attempts[1].Observed || result.Assignment.FloatingIP.ID != "ip-b" || f.raw.Load() != 2 {
		t.Fatal(result, err, proof, f.trace())
	}
}
