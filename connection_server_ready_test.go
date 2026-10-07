package gophercloudsdk_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/compute"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func TestConnectionReadyStaticBranchesAndInvalidPolicyKeepEndpointsLazy(t *testing.T) {
	for _, scenario := range []string{"BUILD", "ERROR", "empty", "disabled", "invalid wait"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			locates := 0
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
				locates++
				return "", errors.New("endpoint must remain lazy")
			}
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			server := connectionAddressServer()
			server.Status = "ACTIVE"
			opts := []compute.ServerReadyOption{compute.WithServerReadyAutomaticIPOptions(compute.WithAutomaticIPEnabled(false))}
			call := conn.GetActiveServer
			switch scenario {
			case "BUILD", "ERROR":
				server.Status, server.Addresses = scenario, map[string]any{"malformed": true}
			case "empty":
				server.Addresses = nil
			case "invalid wait":
				call = conn.WaitForServer
				opts = append(opts, compute.WithServerReadyWaitOptions(resource.WithStatusAttribute("Status")))
			}
			result, err := call(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, opts...)
			if locates != 0 {
				t.Fatal(locates, err)
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
			case "invalid wait":
				if result != nil || !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal(result, err)
				}
			case "disabled":
				if result == nil || err != nil || result.Decision.Reason != compute.AutomaticIPDisabled {
					t.Fatal(result, err)
				}
			}
		})
	}
}

func connectionReadyFixture(t *testing.T, scenario string) (*testcloud.Cloud, *sdk.Connection, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network"), sdk.WithNetworkRoles(network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "private", NATDestination: true})), sdk.WithServerAddressPolicy(compute.WithAddressReachability(false)))
	if err != nil {
		t.Fatal(err)
	}
	posts, raw := &atomic.Int32{}, &atomic.Int32{}
	cloud.Mux.HandleFunc("GET /network/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"networks":[{"id":"private","name":"private"},{"id":"external","name":"external","router:external":true}]}`)
	})
	port := `{"id":"port","device_id":"server","network_id":"private","fixed_ips":[{"ip_address":"10.0.0.10"}]}`
	cloud.Mux.HandleFunc("GET /network/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"ports":[`+port+`]}`)
	})
	cloud.Mux.HandleFunc("GET /network/v2.0/ports/port", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"port":`+port+`}`) })
	cloud.Mux.HandleFunc("GET /network/v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"floatingips":[]}`) })
	cloud.Mux.HandleFunc("POST /network/v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		testcloud.JSON(w, 201, `{"floatingip":{"id":"ip","floating_network_id":"external","port_id":"port","fixed_ip_address":"10.0.0.10","floating_ip_address":"8.8.8.8","status":"DOWN"}}`)
	})
	cloud.Mux.HandleFunc("GET /network/v2.0/floatingips/ip", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"floatingip":{"id":"ip","floating_network_id":"external","port_id":"port","fixed_ip_address":"10.0.0.10","floating_ip_address":"8.8.8.8","status":"ACTIVE"}}`)
	})
	cloud.Mux.HandleFunc("GET /compute/servers/server", func(w http.ResponseWriter, r *http.Request) {
		raw.Add(1)
		if scenario == "progress" {
			testcloud.JSON(w, 200, `{"server":{"id":"server","status":"BUILD","name":"matching raw","progress":25}}`)
			return
		}
		if scenario == "retry" {
			testcloud.JSON(w, 503, `{"error":"unavailable"}`)
			return
		}
		testcloud.JSON(w, 200, `{"server":{"id":"server","status":"ACTIVE","name":"matching raw","addresses":{"private":[{"version":4,"addr":"10.0.0.10","OS-EXT-IPS:type":"fixed"}]}}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	return cloud, conn, posts, raw
}

func connectionReadyOptions() []compute.ServerReadyOption {
	return []compute.ServerReadyOption{
		compute.WithServerReadyAutomaticIPOptions(compute.WithAutomaticIPTimeout(time.Second), compute.WithAutomaticIPPollInterval(time.Millisecond), compute.WithAutomaticEnsureOptions(network.WithEnsureReuse(false), network.WithEnsureWait(resource.WithPollInterval(time.Millisecond)))),
		compute.WithServerReadyWaitOptions(resource.WithPollInterval(time.Millisecond)),
	}
}

func TestConnectionGetActiveAsyncNeutronDoesNotLocateCompute(t *testing.T) {
	cloud, conn, posts, raw := connectionReadyFixture(t, "")
	locates := 0
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates++
		return "", errors.New("Compute endpoint must remain lazy")
	}
	server := connectionAddressServer()
	server.Status = "ACTIVE"
	result, err := conn.GetActiveServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, connectionReadyOptions()...)
	if err != nil || result == nil || result.Assignment == nil || !result.Assignment.Allocated || result.Assignment.FloatingIP.Status != "DOWN" || result.Observed || locates != 0 || posts.Load() != 1 || raw.Load() != 0 {
		t.Fatal(result, err, locates, posts.Load(), raw.Load())
	}
}

type readyConnectionCloseBody struct {
	io.ReadCloser
	close func() error
}

func (b readyConnectionCloseBody) Close() error { return errors.Join(b.ReadCloser.Close(), b.close()) }

func TestConnectionReadyLazyComputeGuardSurvivesReadinessAndIPStages(t *testing.T) {
	for _, scenario := range []string{"progress", "accepted close", "retry", "IP progress"} {
		t.Run(scenario, func(t *testing.T) {
			cloud, conn, posts, raw := connectionReadyFixture(t, scenario)
			locates := 0
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates++
				if opts.Type != "compute" {
					t.Error(opts.Type)
				}
				return cloud.Server.URL + "/compute/", nil
			}
			change := func() {
				service, err := conn.Compute(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				service.API = nil
			}
			opts := connectionReadyOptions()
			if scenario == "progress" {
				opts = append(opts, compute.WithServerReadyWaitOptions(resource.WithProgressCallback(func(int) { change() })))
			}
			if scenario == "IP progress" {
				opts = append(opts, compute.WithServerReadyAutomaticIPOptions(compute.WithAutomaticIPProgress(func(*compute.Server) error { change(); return nil })))
			}
			if scenario == "accepted close" {
				base := cloud.Provider.HTTPClient.Transport
				cloud.Provider.HTTPClient.Transport = serverWorkflowTransport(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if err == nil && r.URL.Path == "/compute/servers/server" {
						response.Header.Set("X-Lazy-Proof", "accepted")
						response.Body = readyConnectionCloseBody{ReadCloser: response.Body, close: func() error { change(); return nil }}
					}
					return response, err
				})
			}
			if scenario == "retry" {
				cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					change()
					return nil
				}
			}
			server := connectionAddressServer()
			server.Status, server.Name = "ACTIVE", "supplied"
			result, err := conn.WaitForServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, opts...)
			if result == nil || !errors.Is(err, resource.ErrInvalidOption) || locates != 1 || result.Observed {
				t.Fatal(result, err, locates)
			}
			if scenario == "IP progress" {
				if result.Assignment == nil || !result.Assignment.Allocated || posts.Load() != 1 || raw.Load() != 2 || result.Server.Name != "matching raw" {
					t.Fatal(result, err, posts.Load(), raw.Load())
				}
			} else if result.Assignment != nil || posts.Load() != 0 || raw.Load() != 1 {
				t.Fatal(result, err, posts.Load(), raw.Load())
			}
			if scenario == "accepted close" {
				var proof *resource.ResponseError
				if result.Server.Name != "matching raw" || !errors.As(err, &proof) || proof.Header.Get("X-Lazy-Proof") != "accepted" {
					t.Fatal(result, err, proof)
				}
			}
			if scenario == "retry" {
				var proof gophercloud.ErrUnexpectedResponseCode
				if result.Server.Name != "supplied" || !errors.As(err, &proof) || proof.Actual != 503 {
					t.Fatal(result, err, proof)
				}
			}
		})
	}
}
