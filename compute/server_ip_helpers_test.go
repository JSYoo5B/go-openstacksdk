package compute_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func standaloneIPOptions(options ...compute.AutomaticFloatingIPOption) []compute.ServerIPOption {
	return []compute.ServerIPOption{compute.WithServerIPAutomaticOptions(options...)}
}

func TestStandaloneServerIPAsyncPreservesOrderedAttachmentsWithoutReadiness(t *testing.T) {
	for _, list := range []bool{false, true} {
		t.Run(fmt.Sprint(list), func(t *testing.T) {
			f := newDispatchFixture(t, "")
			rawClients := 0
			f.service = compute.New(nil, compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) { return f.network, nil }, AddressCompute: func(context.Context) (*gophercloud.ServiceClient, error) {
				rawClients++
				return nil, errors.New("async must not discover Compute")
			}})
			server := automaticServer(t, "null")
			server.Status = "BUILD"
			addresses := []string{dispatchAddresses["a"], dispatchAddresses["b"]}
			options := standaloneIPOptions(append(dispatchOptions(), compute.WithFloatingIPAddresses(addresses...), compute.WithAutomaticIPEnabled(false))...)
			var result *compute.AutomaticServerIPResult
			var err error
			if list {
				result, err = f.service.AddIPList(context.Background(), server, addresses, options...)
			} else {
				result, err = f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server, Network: resource.ID("ignored/network")}, options...)
			}
			trace := strings.Join(f.trace(), ",")
			if err != nil || result == nil || result.Mode != compute.ServerIPExplicit || result.Server.Status != "BUILD" || len(result.Attempts) != 2 || result.Observed || result.Assignment.FloatingIP.Status != "DOWN" || f.raw.Load() != 0 || rawClients != 0 || f.posts.Load() != 0 || strings.Count(trace, "get:a") != 1 || strings.Count(trace, "get:b") != 1 || !strings.Contains(trace, "put:a,list:b") {
				t.Fatal(result, err, trace, rawClients)
			}
			for _, attempt := range result.Attempts {
				if !attempt.Completed || attempt.Observed || attempt.Error != nil {
					t.Fatal(attempt)
				}
			}
		})
	}
}

func TestStandaloneServerIPWaitObservesExactAddressWithoutServerOrIPActive(t *testing.T) {
	for _, status := range []string{"BUILD", "ERROR"} {
		t.Run(status, func(t *testing.T) {
			f := newDispatchFixture(t, "")
			base := f.cloud.Provider.HTTPClient.Transport
			f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
				response, err := base.RoundTrip(r)
				if err == nil && r.URL.Path == "/v2.1/servers/server" {
					data, readErr := io.ReadAll(response.Body)
					closeErr := response.Body.Close()
					if readErr != nil || closeErr != nil {
						t.Fatal(readErr, closeErr)
					}
					response.Body = io.NopCloser(strings.NewReader(strings.Replace(string(data), `"status":"ACTIVE"`, `"status":"`+status+`"`, 1)))
				}
				return response, err
			})
			server := automaticServer(t, "null")
			server.Status = "SHUTOFF"
			options := append(standaloneIPOptions(dispatchOptions()...), compute.WithServerIPWait(true))
			result, err := f.service.AddIPList(context.Background(), server, []string{dispatchAddresses["a"], dispatchAddresses["b"]}, options...)
			trace := strings.Join(f.trace(), ",")
			if err != nil || result == nil || !result.Observed || result.Server.Status != status || f.raw.Load() != 2 || result.Assignment.FloatingIP.Status != "DOWN" || strings.Count(trace, "get:a") != 1 || strings.Count(trace, "get:b") != 1 {
				t.Fatal(result, err, trace)
			}
			for _, attempt := range result.Attempts {
				if !attempt.Completed || !attempt.Observed {
					t.Fatal(attempt)
				}
			}
		})
	}
}

func TestStandaloneServerIPDefaultsUseOneSixtySecondHTTPBudget(t *testing.T) {
	for _, mode := range []string{"default", "override", "parent", "unlimited"} {
		t.Run(mode, func(t *testing.T) {
			f := newDispatchFixture(t, "")
			base := f.cloud.Provider.HTTPClient.Transport
			parentLength := 3 * time.Minute
			if mode == "parent" {
				parentLength = 30 * time.Second
			}
			ctx, cancel := context.WithTimeout(context.Background(), parentLength)
			defer cancel()
			parent, _ := ctx.Deadline()
			start := time.Now()
			var first time.Time
			requests := 0
			f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
				deadline, ok := r.Context().Deadline()
				if !ok {
					t.Fatal("missing HTTP deadline")
				}
				requests++
				if first.IsZero() {
					first = deadline
				} else if !first.Equal(deadline) {
					t.Fatal("renewed item budget", first, deadline)
				}
				return base.RoundTrip(r)
			})
			common := []compute.AutomaticFloatingIPOption{}
			if mode == "override" {
				common = append(common, compute.WithAutomaticIPTimeout(17*time.Second))
			}
			if mode == "unlimited" {
				common = append(common, compute.WithUnlimitedAutomaticIPTimeout())
			}
			result, err := f.service.AddIPList(ctx, automaticServer(t, "null"), []string{dispatchAddresses["a"], dispatchAddresses["b"]}, standaloneIPOptions(common...)...)
			if err != nil || result == nil || len(result.Attempts) != 2 || requests < 8 {
				t.Fatal(result, err, requests)
			}
			if mode == "parent" || mode == "unlimited" {
				if !first.Equal(parent) {
					t.Fatal(first, parent)
				}
				if mode == "unlimited" && !first.After(start.Add(time.Minute)) {
					t.Fatal("unlimited retained the default 60-second cap", first, start)
				}
			} else {
				want := 60 * time.Second
				if mode == "override" {
					want = 17 * time.Second
				}
				elapsed := first.Sub(start)
				if elapsed < want-time.Second || elapsed > want+time.Second || !first.Before(parent) {
					t.Fatal(mode, elapsed, first, parent)
				}
			}
		})
	}
}

func TestStandaloneServerIPPositionalEmptyListNeverFallsThroughToAutomatic(t *testing.T) {
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNeutron, compute.FloatingIPNone, compute.FloatingIPNova} {
		f := newDispatchFixture(t, "")
		options := append(standaloneIPOptions(compute.WithFloatingIPPool(resource.ID("ignored/pool")), compute.WithFloatingIPAddresses("ignored invalid IPv4"), compute.WithAutomaticAddressOptions(compute.WithFloatingIPSource(source))), compute.WithServerIPWait(true))
		result, err := f.service.AddIPList(context.Background(), automaticServer(t, autoFixed), nil, options...)
		if err != nil || result == nil || result.Mode != compute.ServerIPExplicit || result.Decision.Needed || result.Decision.Reason != compute.AutomaticIPEmptyAddressList || result.Observed || len(result.Attempts) != 0 || result.Assignment != nil || len(f.trace()) != 0 {
			t.Fatal(result, err, f.trace())
		}
	}
	f := newDispatchFixture(t, "")
	result, err := f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, standaloneIPOptions(compute.WithFloatingIPAddresses(), compute.WithAutomaticIPEnabled(false))...)
	if err != nil || result.Mode != compute.ServerIPAutomatic || result.Decision.Reason != compute.AutomaticIPDisabled || len(f.trace()) != 0 {
		t.Fatal(result, err, f.trace())
	}
}

func TestStandaloneServerIPPositionalSelectionAndOptionsArePreparedOnce(t *testing.T) {
	f := newDispatchFixture(t, "")
	server := automaticServer(t, autoFixed)
	addresses := []string{dispatchAddresses["a"], dispatchAddresses["b"]}
	outer, inner, destination, wait := 0, 0, 0, 0
	common := []compute.AutomaticFloatingIPOption{
		beforeDispatchOption(func() { inner++ }, compute.WithFloatingIPPool(resource.ID("ignored/pool"))),
		compute.WithFloatingIPAddresses("ignored malformed IP"),
		compute.WithAutomaticEnsureOptions(beforeDispatchOption(func() { destination++ }, network.WithEnsureNoWait()), network.WithEnsureWait(beforeDispatchOption(func() { wait++ }, resource.WithPollInterval(time.Millisecond)))),
	}
	option := beforeDispatchOption(func() {
		outer++
		server.ID = "caller-changed"
		server.Status = "BUILD"
		addresses[0] = dispatchAddresses["c"]
	}, compute.WithServerIPAutomaticOptions(common...))
	common[0] = nil
	result, err := f.service.AddIPList(context.Background(), server, addresses, option)
	if err != nil || result == nil || result.Server.ID != "server" || result.Server.Status != "ACTIVE" || len(result.Attempts) != 2 || result.Attempts[0].RequestedAddress != dispatchAddresses["a"] || outer != 1 || inner != 1 || destination != 1 || wait != 1 || strings.Contains(strings.Join(f.trace(), ","), "list:c") {
		t.Fatal(result, err, outer, inner, destination, wait, f.trace())
	}
}

func TestStandaloneServerIPPoolUsesSamePriorityAndAsyncAllocationPolicy(t *testing.T) {
	for _, reuse := range []bool{false, true} {
		f := newDispatchFixture(t, "")
		server := automaticServer(t, "null")
		server.Status = "BUILD"
		options := standaloneIPOptions(compute.WithFloatingIPPool(resource.ID("pool-network")), compute.WithFloatingIPAddresses("ignored invalid"), compute.WithAutomaticIPEnabled(false), compute.WithAutomaticEnsureOptions(network.WithEnsureProject("owner"), network.WithEnsureReuse(reuse)))
		result, err := f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server, Network: resource.ID("ignored/network")}, options...)
		if err != nil || result == nil || result.Mode != compute.ServerIPPool || result.Observed || len(result.Attempts) != 1 || !result.Attempts[0].Completed || result.Assignment.FloatingIP.Status != "DOWN" || result.Assignment.Reused != reuse || result.Assignment.Allocated == reuse || f.raw.Load() != 0 {
			t.Fatal(result, err, f.trace())
		}
		wantPosts := int32(1)
		if reuse {
			wantPosts = 0
		}
		if f.posts.Load() != wantPosts || strings.Contains(strings.Join(f.trace(), ","), "list:a") {
			t.Fatal(f.trace(), f.posts.Load())
		}
	}
}

func TestStandaloneServerIPSecondFailurePreservesAsyncAndObservedHistory(t *testing.T) {
	for _, wait := range []bool{false, true} {
		for _, failure := range []string{"missing", "close"} {
			t.Run(fmt.Sprint(wait, failure), func(t *testing.T) {
				f := newDispatchFixture(t, failure)
				cause := errors.New("second accepted Close")
				if failure == "close" {
					base := f.cloud.Provider.HTTPClient.Transport
					f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
						response, err := base.RoundTrip(r)
						if err == nil && r.Method == "PUT" && r.URL.Path == "/v2.0/floatingips/ip-b" {
							response.Body = automaticCloseBody{ReadCloser: response.Body, close: func() error { return cause }}
						}
						return response, err
					})
				}
				options := append(standaloneIPOptions(dispatchOptions()...), compute.WithServerIPWait(wait))
				result, err := f.service.AddIPList(context.Background(), automaticServer(t, "null"), []string{dispatchAddresses["a"], dispatchAddresses["b"], dispatchAddresses["c"]}, options...)
				if result == nil || err == nil || result.Observed || len(result.Attempts) != 2 || !result.Attempts[0].Completed || result.Attempts[0].Observed != wait || result.Attempts[1].Completed || result.Attempts[1].Error == nil || strings.Contains(strings.Join(f.trace(), ","), "list:c") {
					t.Fatal(result, err, f.trace())
				}
				wantRaw := int32(0)
				if wait {
					wantRaw = 1
				}
				if f.raw.Load() != wantRaw {
					t.Fatal(f.raw.Load())
				}
				if failure == "missing" {
					if !errors.Is(err, resource.ErrNotFound) || result.Assignment.FloatingIP.ID != "ip-a" {
						t.Fatal(result, err)
					}
				} else {
					var proof *resource.ResponseError
					if !errors.Is(err, cause) || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Dispatch-Proof") != "accepted-b" || result.Assignment.FloatingIP.ID != "ip-b" || result.Assignment.FloatingIP.PortID != "port" {
						t.Fatal(result, err, proof)
					}
				}
			})
		}
	}
}

func TestStandaloneServerIPAutomaticAsyncAndAcceptedRefreshModels(t *testing.T) {
	for _, refreshFailure := range []bool{false, true} {
		t.Run(fmt.Sprint(refreshFailure), func(t *testing.T) {
			f := newAutomaticFixture(t)
			server := automaticServer(t, autoFixed)
			server.Status = "BUILD"
			cause := errors.New("standalone raw Close")
			if refreshFailure {
				server.Addresses = nil
				f.rawBody = func(int32) (int, string) {
					return 203, `{"server":{"id":"server","name":"accepted-refresh","status":"BUILD","addresses":` + autoFixed + `}}`
				}
				base := f.cloud.Provider.HTTPClient.Transport
				f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if err == nil && r.URL.Path == "/v2.1/servers/server" {
						response.Body = automaticCloseBody{ReadCloser: response.Body, close: func() error { return cause }}
					}
					return response, err
				})
			}
			result, err := f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, standaloneIPOptions(automaticOptions()...)...)
			if refreshFailure {
				var proof *resource.ResponseError
				if !errors.Is(err, cause) || !errors.As(err, &proof) || proof.StatusCode != 203 || result == nil || result.Server.Name != "accepted-refresh" || result.Server.Status != "BUILD" || result.Assignment != nil || f.posts.Load() != 0 || f.raw.Load() != 1 {
					t.Fatal(result, err, proof)
				}
			} else if err != nil || result == nil || result.Mode != compute.ServerIPAutomatic || result.Assignment == nil || !result.Assignment.Allocated || result.Assignment.FloatingIP.Status != "DOWN" || result.Observed || f.posts.Load() != 1 || f.raw.Load() != 0 {
				t.Fatal(result, err, f.posts.Load(), f.raw.Load())
			}
		})
	}
}

func TestStandaloneServerIPPreflightStopsBeforeAnyRequest(t *testing.T) {
	for _, scenario := range []string{"nil context", "canceled", "nil server", "unsafe ID", "nil option", "bad address", "common project", "missing raw"} {
		t.Run(scenario, func(t *testing.T) {
			f := newDispatchFixture(t, "")
			ctx := context.Background()
			server := automaticServer(t, autoFixed)
			addresses := []string{dispatchAddresses["a"]}
			options := []compute.ServerIPOption{}
			cause := errors.New("caller canceled")
			want := resource.ErrInvalidOption
			switch scenario {
			case "nil context":
				ctx = nil
			case "canceled":
				var cancel context.CancelCauseFunc
				ctx, cancel = context.WithCancelCause(ctx)
				cancel(cause)
				want = cause
			case "nil server":
				server = nil
			case "unsafe ID":
				server.ID = "unsafe/server"
			case "nil option":
				options = append(options, nil)
			case "bad address":
				addresses = []string{"::1"}
			case "common project":
				options = standaloneIPOptions(compute.WithAutomaticEnsureOptions(network.WithEnsureProject("invalid/project")))
			case "missing raw":
				f.service = compute.New(nil, compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) { return f.network, nil }})
				options = append(options, compute.WithServerIPWait(true))
				want = resource.ErrUnsupported
			}
			result, err := f.service.AddIPList(ctx, server, addresses, options...)
			if !errors.Is(err, want) || len(f.trace()) != 0 || result != nil && len(result.Attempts) != 0 {
				t.Fatal(result, err, f.trace())
			}
		})
	}
}

func TestStandaloneServerIPExpiredObservationKeepsAssignmentAndStopsNextItem(t *testing.T) {
	f := newDispatchFixture(t, "")
	base := f.cloud.Provider.HTTPClient.Transport
	rawCalls := 0
	f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/v2.1/servers/server" {
			rawCalls++
			<-r.Context().Done()
			return nil, r.Context().Err()
		}
		return base.RoundTrip(r)
	})
	options := []compute.ServerIPOption{compute.WithServerIPWait(true), compute.WithServerIPAutomaticOptions(compute.WithAutomaticIPTimeout(time.Second))}
	result, err := f.service.AddIPList(context.Background(), automaticServer(t, "null"), []string{dispatchAddresses["a"], dispatchAddresses["b"]}, options...)
	if !errors.Is(err, context.DeadlineExceeded) || result == nil || result.Assignment == nil || result.Assignment.FloatingIP.ID != "ip-a" || result.Observed || len(result.Attempts) != 1 || result.Attempts[0].Completed || result.Attempts[0].Error == nil || rawCalls != 1 || strings.Contains(strings.Join(f.trace(), ","), "list:b") {
		t.Fatal(result, err, rawCalls, f.trace())
	}
}

func TestStandaloneServerIPAlreadyAttachedAddressesAndPoolKeepComputeLazy(t *testing.T) {
	for _, pool := range []bool{false, true} {
		t.Run(fmt.Sprint(pool), func(t *testing.T) {
			f := newDispatchFixture(t, "")
			key := "a"
			if pool {
				key = "pool"
			}
			f.bound[key] = true
			rawClients := 0
			f.service = compute.New(nil, compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) { return f.network, nil }, AddressCompute: func(context.Context) (*gophercloud.ServiceClient, error) {
				rawClients++
				return nil, errors.New("already attached async must stay lazy")
			}})
			server := automaticServer(t, "null")
			server.Status = "BUILD"
			var result *compute.AutomaticServerIPResult
			var err error
			if pool {
				result, err = f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, standaloneIPOptions(compute.WithFloatingIPPool(resource.ID("pool-network")), compute.WithAutomaticEnsureOptions(network.WithEnsureProject("owner")))...)
			} else {
				result, err = f.service.AddIPList(context.Background(), server, []string{dispatchAddresses["a"]})
			}
			trace := strings.Join(f.trace(), ",")
			wantGets := 1
			if pool {
				wantGets = 0
			}
			if err != nil || result == nil || result.Assignment == nil || result.Assignment.FloatingIP.ID != "ip-"+key || !result.Assignment.Reused || result.Assignment.Allocated || result.Observed || len(result.Attempts) != 1 || !result.Attempts[0].Completed || result.Server.Addresses != nil || rawClients != 0 || f.raw.Load() != 0 || f.posts.Load() != 0 || strings.Contains(trace, "put:") || strings.Contains(trace, "allocate:") || strings.Count(trace, "get:"+key) != wantGets {
				t.Fatal(result, err, trace, rawClients)
			}
		})
	}
}
