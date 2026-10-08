package openstack_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestMicroversionNovaCatalogScopeAndSharedConcurrentClient(t *testing.T) {
	cloud := testcloud.New(t)
	var lookups, discoveries atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		lookups.Add(1)
		if opts.Type != "compute" || opts.Region != "region" {
			t.Error(opts)
		}
		return cloud.Server.URL + "/proxy/compute/v2.1/project/", nil
	}
	cloud.Mux.HandleFunc("GET /proxy/compute/v2.1/{$}", func(w http.ResponseWriter, r *http.Request) {
		discoveries.Add(1)
		if r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != "" || r.Header.Get("X-OpenStack-Nova-API-Version") != "" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"version":{"id":"v2.1","status":"CURRENT","min_version":"2.1","version":"2.110"}}`)
	})
	cloud.Mux.HandleFunc("GET /proxy/compute/v2.1/project/servers/server", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("OpenStack-API-Version") != "compute 2.100" || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.100" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"server":{"id":"server"}}`)
	})
	c, err := sdk.FromProvider(cloud.Provider, sdk.WithRegion("region"), sdk.WithMicroversionRange(sdk.Compute, "2.9", "2.100"))
	if err != nil {
		t.Fatal(err)
	}
	clients := make([]*gophercloud.ServiceClient, 24)
	var wg sync.WaitGroup
	for i := range clients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client, err := raw(c.ComputeV2(context.Background()))
			if err != nil {
				t.Error(err)
				return
			}
			clients[i] = client
		}()
	}
	wg.Wait()
	for _, client := range clients {
		if client != clients[0] || client.Microversion != "2.100" {
			t.Fatal(client)
		}
	}
	manual, err := c.Compute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if manual.RawClient() != clients[0] {
		t.Fatal("generated and high-level clients differ")
	}
	if _, err := manual.Servers.Get(context.Background(), "server"); err != nil {
		t.Fatal(err)
	}
	selection, err := c.Microversion(context.Background(), sdk.Compute)
	if err != nil {
		t.Fatal(err)
	}
	if !selection.Negotiated || selection.Selected != "2.100" || selection.RequestedExact != "" || selection.RequestedMinimum != "2.9" || selection.RequestedMaximum != "2.100" || selection.SupportedMinimum != "2.1" || selection.SupportedMaximum != "2.110" || selection.DiscoveryURL != cloud.Server.URL+"/proxy/compute/v2.1/" {
		t.Fatalf("selection=%+v", selection)
	}
	selection.Selected = "2.9"
	again, err := c.Microversion(context.Background(), sdk.Compute)
	if err != nil || again.Selected != "2.100" {
		t.Fatalf("selection was mutated: %+v %v", again, err)
	}
	if lookups.Load() != 1 || discoveries.Load() != 1 {
		t.Fatalf("lookups=%d discoveries=%d", lookups.Load(), discoveries.Load())
	}
}

func TestMicroversionCinderScopeAndDefaultVersionRestriction(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /volume/v3/{$}", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"versions":[{"id":"v3.0","status":"CURRENT","min_version":"3.0","version":"3.71"}]}`)
	})
	cloud.Mux.HandleFunc("GET /volume/v3/project/volumes/volume", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("OpenStack-API-Version") != "volume 3.71" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.71" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"volume":{"id":"volume"}}`)
	})
	c, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/volume/v3/project"), sdk.WithLatestMicroversion(sdk.BlockStorage))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.BlockStorageV2(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	service, err := c.BlockStorageV3(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Volumes.Get(context.Background(), "volume"); err != nil {
		t.Fatal(err)
	}
	selection, err := c.Microversion(context.Background(), sdk.BlockStorage)
	if err != nil || selection.RequestedMaximum != "latest" || selection.Selected != "3.71" {
		t.Fatalf("selection=%+v err=%v", selection, err)
	}
}

func TestMicroversionRootDiscoveryShapesAndHeaders(t *testing.T) {
	for _, check := range []struct {
		name                                                                   string
		service                                                                sdk.Service
		endpoint, discovery, requestPath, header, legacyHeader, body, selected string
	}{
		{"placement", sdk.Placement, "/placement", "/placement/", "/placement/ping", "placement 1.39", "", `{"versions":[{"id":"v1.0","status":"CURRENT","min_version":"1.0","max_version":"1.39"}]}`, "1.39"},
		{"baremetal", sdk.BareMetal, "/ironic", "/ironic/", "/ironic/v1/ping", "baremetal 1.87", "X-OpenStack-Ironic-API-Version", `{"id":"v1","status":"CURRENT","min_version":"1.1","version":"1.87"}`, "1.87"},
		{"magnum", sdk.ContainerInfra, "/magnum/v1", "/magnum/", "/magnum/v1/ping", "container-infra 1.11", "", `{"versions":{"values":[{"id":"v1","status":"CURRENT","min_version":"1.1","max_version":"1.11"}]}}`, "1.11"},
		{"manila", sdk.SharedFileSystem, "/share/v2/project", "/share/v2/", "/share/v2/project/ping", "shared-file-system 2.89", "X-OpenStack-Manila-API-Version", `{"version":{"id":"v2.0","min_version":"2.0","version":"2.89"}}`, "2.89"},
	} {
		t.Run(check.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var discoveries atomic.Int32
			cloud.Mux.HandleFunc("GET "+check.discovery+"{$}", func(w http.ResponseWriter, r *http.Request) {
				discoveries.Add(1)
				if r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 200, check.body)
			})
			if check.service == sdk.ContainerInfra {
				cloud.Mux.HandleFunc("GET /magnum/v1/{$}", func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/magnum/v1/" {
						t.Error("wrong discovery URL", r.URL)
					}
					testcloud.JSON(w, 200, `{"id":"v1"}`)
				})
			}
			cloud.Mux.HandleFunc("GET "+check.requestPath, func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("OpenStack-API-Version") != check.header || (check.legacyHeader != "" && r.Header.Get(check.legacyHeader) != check.selected) {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 200, `{}`)
			})
			c, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(check.service, cloud.Server.URL+check.endpoint), sdk.WithLatestMicroversion(check.service))
			if err != nil {
				t.Fatal(err)
			}
			selection, err := c.Microversion(context.Background(), check.service)
			if err != nil {
				t.Fatal(err)
			}
			if selection.Selected != check.selected || selection.DiscoveryURL != cloud.Server.URL+check.discovery || discoveries.Load() != 1 {
				t.Fatalf("selection=%+v calls=%d", selection, discoveries.Load())
			}
			var client *gophercloud.ServiceClient
			switch check.service {
			case sdk.Placement:
				client, err = raw(c.Placement(context.Background()))
			case sdk.BareMetal:
				client, err = raw(c.BareMetal(context.Background()))
			case sdk.ContainerInfra:
				client, err = raw(c.ContainerInfra(context.Background()))
			case sdk.SharedFileSystem:
				client, err = raw(c.SharedFileSystem(context.Background()))
			}
			if err != nil {
				t.Fatal(err)
			}
			var body map[string]any
			if _, err := client.Get(context.Background(), client.ServiceURL("ping"), &body, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMicroversionExactPrecedenceAndDefaultNoDiscovery(t *testing.T) {
	cloud := testcloud.New(t)
	var requests atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		t.Error("unexpected discovery", r.URL)
		w.WriteHeader(500)
	})
	for _, check := range []struct {
		name                string
		options             []sdk.ConnectionOption
		selected, requested string
	}{
		{"default", nil, "", ""},
		{"exact-first", []sdk.ConnectionOption{sdk.WithMicroversion(sdk.Compute, "2.52"), sdk.WithMicroversionRange(sdk.Compute, "2.70", "2.90")}, "2.52", "2.90"},
		{"exact-last", []sdk.ConnectionOption{sdk.WithLatestMicroversion(sdk.Compute), sdk.WithMicroversion(sdk.Compute, "2.52")}, "2.52", "latest"},
	} {
		t.Run(check.name, func(t *testing.T) {
			options := append([]sdk.ConnectionOption{sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/nova/v2.1/project")}, check.options...)
			c, err := sdk.FromProvider(cloud.Provider, options...)
			if err != nil {
				t.Fatal(err)
			}
			selection, err := c.Microversion(context.Background(), sdk.Compute)
			if err != nil {
				t.Fatal(err)
			}
			if selection.Negotiated || selection.Selected != check.selected || selection.RequestedExact != check.selected || selection.RequestedMaximum != check.requested || selection.SupportedMinimum != "" || selection.DiscoveryURL != "" {
				t.Fatalf("selection=%+v", selection)
			}
		})
	}
	if requests.Load() != 0 {
		t.Fatal(requests.Load())
	}
}

func TestMicroversionInvalidRanges(t *testing.T) {
	for _, option := range []sdk.ConnectionOption{
		sdk.WithMicroversionRange(sdk.Compute, "2.90", "2.9"), sdk.WithMicroversionRange(sdk.Compute, "latest", "latest"),
		sdk.WithMicroversionRange(sdk.Compute, "3.1", "latest"), sdk.WithMicroversionRange(sdk.Compute, "", "2.1.1"),
		sdk.WithMicroversionRange(sdk.Compute, "2.-1", ""), sdk.WithMicroversionRange(sdk.Compute, "2.1", "3.1"),
		sdk.WithMicroversionRange(sdk.Compute, "2.999999999999999999999999", "latest"),
		sdk.WithMicroversion(sdk.Compute, "2.999999999999999999999999"),
	} {
		if _, err := sdk.FromProvider(&gophercloud.ProviderClient{}, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("accepted invalid range: %v", err)
		}
	}
	for _, service := range []sdk.Service{sdk.Network, sdk.ObjectStorage, sdk.Image, sdk.Service("unknown")} {
		if _, err := sdk.FromProvider(&gophercloud.ProviderClient{}, sdk.WithLatestMicroversion(service)); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(service, err)
		}
	}
}

func TestMicroversionFailuresRemainUncachedAndPreserveErrors(t *testing.T) {
	for _, check := range []struct {
		name, body  string
		code        int
		unsupported bool
	}{
		{"no-overlap", `{"version":{"id":"v2.1","min_version":"2.1","version":"2.9"}}`, 200, true},
		{"malformed-bounds", `{"version":{"id":"v2.1","min_version":"2.10","version":"2.9"}}`, 200, false},
		{"missing-bounds", `{"version":{"id":"v2.1"}}`, 200, true},
		{"conflicting-bounds", `{"versions":[{"id":"v2.1","min_version":"2.1","version":"2.10"},{"id":"v2.1","min_version":"2.1","version":"2.11"}]}`, 200, false},
		{"wrong-major-bounds", `{"version":{"id":"v2.1","min_version":"2.1","version":"3.90"}}`, 200, false},
		{"forbidden", `{}`, 403, false},
		{"bad-document", `{"version":`, 200, false},
	} {
		t.Run(check.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /compute/v2.1/{$}", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					testcloud.JSON(w, check.code, check.body)
					return
				}
				testcloud.JSON(w, 200, `{"version":{"id":"v2.1","min_version":"2.1","version":"2.100"}}`)
			})
			cloud.Mux.HandleFunc("GET /compute/{$}", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"versions":[]}`) })
			c, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute/v2.1/project"), sdk.WithMicroversionRange(sdk.Compute, "2.10", "2.90"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err = c.ComputeV2(context.Background()); err == nil || (check.unsupported && !errors.Is(err, resource.ErrUnsupported)) {
				t.Fatalf("first err=%v", err)
			}
			if check.code == 403 {
				var responseErr gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &responseErr) || responseErr.Actual != 403 {
					t.Fatalf("lost HTTP error: %v", err)
				}
			}
			service, err := c.ComputeV2(context.Background())
			if err != nil || service.RawClient().Microversion != "2.90" || calls.Load() != 2 {
				t.Fatalf("retry err=%v calls=%d", err, calls.Load())
			}
		})
	}
}

func TestMicroversionCancelledDiscoveryCanBeRetried(t *testing.T) {
	cloud := testcloud.New(t)
	started := make(chan struct{})
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /compute/v2.1/{$}", func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			close(started)
			<-r.Context().Done()
			return
		}
		testcloud.JSON(w, 200, `{"version":{"id":"v2.1","min_version":"2.1","version":"2.100"}}`)
	})
	c, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute/v2.1/project"), sdk.WithLatestMicroversion(sdk.Compute))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := c.ComputeV2(ctx); done <- err }()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	selection, err := c.Microversion(context.Background(), sdk.Compute)
	if err != nil || selection.Selected != "2.100" || calls.Load() != 2 {
		t.Fatalf("retry selection=%+v err=%v calls=%d", selection, err, calls.Load())
	}
	if _, err := c.Microversion(ctx, sdk.Compute); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

// Observing the first cancellation check makes this regression deterministic:
// cancellation occurs after its nil snapshot and before the cache lookup.
type observedMicroversionContext struct {
	context.Context
	checked chan struct{}
	resume  chan struct{}
	once    sync.Once
}

func (ctx *observedMicroversionContext) Err() error {
	err := ctx.Context.Err()
	ctx.once.Do(func() { close(ctx.checked); <-ctx.resume })
	return err
}

func TestMicroversionCancellationWhileWaitingForSharedDiscovery(t *testing.T) {
	cloud := testcloud.New(t)
	discovering, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /compute/v2.1/{$}", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		close(discovering)
		<-release
		testcloud.JSON(w, 200, `{"version":{"id":"v2.1","min_version":"2.1","version":"2.100"}}`)
	})
	c, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute/v2.1/project"), sdk.WithLatestMicroversion(sdk.Compute))
	if err != nil {
		t.Fatal(err)
	}
	first := make(chan error, 1)
	go func() { _, err := c.ComputeV2(context.Background()); first <- err }()
	<-discovering
	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &observedMicroversionContext{Context: base, checked: make(chan struct{}), resume: make(chan struct{})}
	second := make(chan error, 1)
	go func() {
		service, err := c.ComputeV2(ctx)
		if service != nil {
			second <- fmt.Errorf("cancelled cache lookup returned a service")
			return
		}
		second <- err
	}()
	<-ctx.checked
	cancel()
	close(ctx.resume)
	close(release)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	if err := <-second; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("discovery calls=%d", calls.Load())
	}
}

func TestMicroversionVersionRootFallbackAndLegacyNova(t *testing.T) {
	for _, api := range []string{"v2.1", "v2"} {
		t.Run(api, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("GET /compute/"+api+"/{$}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) })
			cloud.Mux.HandleFunc("GET /compute/{$}", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 300, `{"versions":[{"id":"v2.0","status":"SUPPORTED"},{"id":"v2.1","status":"CURRENT","min_version":"2.1","version":"2.100","links":[{"rel":"self","href":"https://untrusted.example/v2.1/"}]}]}`)
			})
			c, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, fmt.Sprintf("%s/compute/%s/project", cloud.Server.URL, api)), sdk.WithLatestMicroversion(sdk.Compute))
			if err != nil {
				t.Fatal(err)
			}
			selection, err := c.Microversion(context.Background(), sdk.Compute)
			if api == "v2" {
				if !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal(err)
				}
			} else if err != nil || selection.Selected != "2.100" || selection.DiscoveryURL != cloud.Server.URL+"/compute/" {
				t.Fatalf("selection=%+v err=%v", selection, err)
			}
		})
	}
}
