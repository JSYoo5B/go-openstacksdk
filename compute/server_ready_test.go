package compute_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func serverReadyOptions() []compute.ServerReadyOption {
	return []compute.ServerReadyOption{
		compute.WithServerReadyAutomaticIPOptions(automaticOptions()...),
		compute.WithServerReadyWaitOptions(resource.WithPollInterval(time.Millisecond)),
	}
}

func TestGetActiveServerStatusBranchesNeverReadAddressesOrDiscoverServices(t *testing.T) {
	for _, status := range []string{"BUILD", "SHUTOFF", "PAUSED", "", "unknown", "ERROR"} {
		t.Run(status, func(t *testing.T) {
			calls := 0
			service := compute.New(nil, compute.Dependencies{
				AddressNetworks: func(context.Context) (*network.Service, error) { calls++; return nil, errors.New("eager network") },
				AddressCompute: func(context.Context) (*gophercloud.ServiceClient, error) {
					calls++
					return nil, errors.New("eager compute")
				},
			})
			server := automaticServer(t, autoFixed)
			server.Status, server.Fault.Message = status, "boot failed"
			server.Addresses = map[string]any{"invalid": true}
			result, err := service.GetActiveServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, compute.WithServerReadyWaitOptions(resource.WithFailureStates()))
			if calls != 0 {
				t.Fatal("eager dependency", calls)
			}
			if status == "ERROR" {
				if !errors.Is(err, resource.ErrFailedState) || result == nil || result.Server.Fault.Message != "boot failed" || result.Assignment != nil {
					t.Fatal(result, err)
				}
			} else if result != nil || err != nil {
				t.Fatal(result, err)
			}
		})
	}
}

func TestGetActiveServerEmptyAddressRowsFailWithoutRefreshOrDelete(t *testing.T) {
	for _, addresses := range []string{`null`, `{}`, `{"private":null}`, `{"private":[]}`, `{"a":[],"b":null}`} {
		t.Run(addresses, func(t *testing.T) {
			f := newAutomaticFixture(t)
			opts := append(serverReadyOptions(), compute.WithServerReadyAutomaticIPOptions(compute.WithAutomaticIPEnabled(false)))
			result, err := f.service.GetActiveServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, addresses)}, opts...)
			if !errors.Is(err, compute.ErrServerAddressesUnavailable) || result == nil || result.Server.ID != "server" || result.Assignment != nil || f.raw.Load() != 0 || f.ports.Load() != 0 || f.posts.Load() != 0 || f.roles.Load() != 0 {
				t.Fatal(result, err, f.raw.Load(), f.ports.Load(), f.posts.Load(), f.roles.Load())
			}
		})
	}
}

func TestGetActiveServerAsyncReturnsAcceptedDownIPWithoutCompute(t *testing.T) {
	f := newAutomaticFixture(t)
	computeCalls, ipGets := 0, 0
	f.service = compute.New(nil, compute.Dependencies{
		AddressNetworks: func(context.Context) (*network.Service, error) { return f.network, nil },
		AddressCompute: func(context.Context) (*gophercloud.ServiceClient, error) {
			computeCalls++
			return nil, errors.New("async must not observe")
		},
	})
	base := f.cloud.Provider.HTTPClient.Transport
	f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/v2.0/floatingips/ip" {
			ipGets++
		}
		return base.RoundTrip(r)
	})
	// Even an explicit IP wait supplied by the caller is overridden by async Get.
	opts := append(serverReadyOptions(), compute.WithServerReadyAutomaticIPOptions(compute.WithAutomaticEnsureOptions(network.WithEnsureActive())))
	result, err := f.service.GetActiveServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, opts...)
	if err != nil || result == nil || result.Assignment == nil || !result.Assignment.Allocated || result.Assignment.FloatingIP.Status != "DOWN" || result.Observed || result.Server.Status != "ACTIVE" || !result.Decision.Needed || f.posts.Load() != 1 || f.raw.Load() != 0 || ipGets != 0 || computeCalls != 0 {
		t.Fatal(result, err, f.posts.Load(), f.raw.Load(), ipGets, computeCalls)
	}
}

func TestGetActiveServerSyncRequiresComputeBeforeMutationAndObservesTarget(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(map[bool]string{false: "observed", true: "missing"}[missing], func(t *testing.T) {
			f := newAutomaticFixture(t)
			if missing {
				f.service = compute.New(nil, compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) { return f.network, nil }})
			}
			f.rawBody = func(int32) (int, string) {
				return 203, `{"server":{"id":"server","status":"ACTIVE","name":"raw observed","addresses":{"private":[{"version":4,"addr":"9.9.9.9","OS-EXT-IPS:type":"floating"},{"version":4,"addr":"8.8.8.8","OS-EXT-IPS:type":"floating"}]}}}`
			}
			opts := append(serverReadyOptions(), compute.WithActiveServerWait(true))
			result, err := f.service.GetActiveServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, opts...)
			if missing {
				if !errors.Is(err, resource.ErrUnsupported) || result == nil || result.Assignment != nil || f.posts.Load() != 0 || f.raw.Load() != 0 {
					t.Fatal(result, err, f.posts.Load(), f.raw.Load())
				}
			} else if err != nil || result == nil || !result.Observed || result.Assignment == nil || result.Assignment.FloatingIP.Status != "ACTIVE" || result.Server.Name != "raw observed" || f.posts.Load() != 1 || f.raw.Load() != 1 {
				t.Fatal(result, err, f.posts.Load(), f.raw.Load())
			}
		})
	}
}

func TestWaitForServerAlwaysRefreshesAndWaitsForRawAddressRows(t *testing.T) {
	for _, status := range []string{"ACTIVE", "ERROR", "BUILD", ""} {
		t.Run(status, func(t *testing.T) {
			f := newAutomaticFixture(t)
			f.rawBody = func(n int32) (int, string) {
				if n == 1 {
					return 200, `{"server":{"id":"server","status":"BUILD","progress":25}}`
				}
				if n == 2 {
					return 203, `{"server":{"id":"server","status":"ACTIVE","addresses":null}}`
				}
				return 200, `{"server":{"id":"server","status":"ACTIVE","name":"latest","addresses":` + autoFloating + `}}`
			}
			server := automaticServer(t, autoFixed)
			server.Status, server.Addresses = status, map[string]any{"malformed": true}
			var progress []int
			opts := append(serverReadyOptions(), compute.WithServerReadyWaitOptions(resource.WithProgressCallback(func(n int) { progress = append(progress, n); server.ID = "caller changed" })))
			result, err := f.service.WaitForServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, opts...)
			if err != nil || result == nil || result.Server.ID != "server" || result.Server.Name != "latest" || result.Decision.Reason != compute.AutomaticIPExistingFloating || result.Assignment != nil || f.raw.Load() != 3 || f.posts.Load() != 0 || !reflect.DeepEqual(progress, []int{25, 0}) {
				t.Fatal(result, err, f.raw.Load(), f.posts.Load(), progress)
			}
		})
	}
}

func TestWaitForServerForcesIPActiveAndRawConvergence(t *testing.T) {
	f := newAutomaticFixture(t)
	ipGets := 0
	base := f.cloud.Provider.HTTPClient.Transport
	f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/v2.0/floatingips/ip" {
			ipGets++
		}
		return base.RoundTrip(r)
	})
	f.rawBody = func(n int32) (int, string) {
		if n == 1 {
			return 200, `{"server":{"id":"server","status":"ACTIVE","addresses":` + autoFixed + `}}`
		}
		return 200, `{"server":{"id":"server","status":"ACTIVE","addresses":` + autoFloating + `}}`
	}
	opts := append(serverReadyOptions(), compute.WithActiveServerWait(false), compute.WithServerReadyAutomaticIPOptions(compute.WithAutomaticEnsureOptions(network.WithEnsureNoWait())))
	result, err := f.service.WaitForServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, `null`)}, opts...)
	if err != nil || result == nil || !result.Observed || result.Assignment == nil || result.Assignment.FloatingIP.Status != "ACTIVE" || f.posts.Load() != 1 || f.raw.Load() != 2 || ipGets != 1 {
		t.Fatal(result, err, f.posts.Load(), f.raw.Load(), ipGets)
	}
}

func TestServerReadyInvalidPolicyFailsBeforeIOForBothEntrypoints(t *testing.T) {
	for _, wait := range []bool{false, true} {
		for _, scenario := range []string{"nil", "nil automatic", "nil wait", "status", "timeout", "unsafe ID", "unsafe network", "nil server"} {
			t.Run(map[bool]string{false: "Get/", true: "Wait/"}[wait]+scenario, func(t *testing.T) {
				f := newAutomaticFixture(t)
				input := compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}
				opts := serverReadyOptions()
				switch scenario {
				case "nil":
					opts = append(opts, nil)
				case "nil automatic":
					opts = append(opts, compute.WithServerReadyAutomaticIPOptions(nil))
				case "nil wait":
					opts = append(opts, compute.WithServerReadyWaitOptions(nil))
				case "status":
					opts = append(opts, compute.WithServerReadyWaitOptions(resource.WithStatusAttribute("Status")))
				case "timeout":
					opts = append(opts, compute.WithServerReadyAutomaticIPOptions(compute.WithAutomaticIPTimeout(0)))
				case "unsafe ID":
					input.Server.ID = "../unsafe"
				case "unsafe network":
					input.Network = resource.ID("../unsafe")
				case "nil server":
					input.Server = nil
				}
				call := f.service.GetActiveServer
				if wait {
					call = f.service.WaitForServer
				}
				result, err := call(context.Background(), input, opts...)
				want := resource.ErrInvalidOption
				if scenario == "status" {
					want = resource.ErrUnsupported
				}
				if result != nil || !errors.Is(err, want) || f.raw.Load() != 0 || f.posts.Load() != 0 || f.roles.Load() != 0 || f.ports.Load() != 0 {
					t.Fatal(result, err, f.raw.Load(), f.posts.Load(), f.roles.Load(), f.ports.Load())
				}
			})
		}
	}
}
