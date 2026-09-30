package gophercloudsdk_test

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func raw[T interface {
	RawClient() *gophercloud.ServiceClient
}](service T, err error) (*gophercloud.ServiceClient, error) {
	if err != nil {
		return nil, err
	}
	return service.RawClient(), nil
}

func TestEveryServiceUsesSharedCatalogPolicyAndProvider(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	var lookups atomic.Int32
	provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		lookups.Add(1)
		if opts.Region != "region" || opts.Availability != gophercloud.AvailabilityInternal {
			t.Errorf("opts=%+v", opts)
		}
		return "https://cloud.example/" + opts.Type + "/", nil
	}
	c, err := sdk.FromProvider(provider, sdk.WithRegion("region"), sdk.WithInterface(gophercloud.AvailabilityInternal))
	if err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		service sdk.Service
		get     func(context.Context) (*gophercloud.ServiceClient, error)
	}{
		{sdk.Compute, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.ComputeV2(ctx)) }},
		{sdk.Network, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.NetworkV2(ctx)) }},
		{sdk.Image, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.ImageV2(ctx)) }},
		{sdk.BlockStorage, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.BlockStorageV3(ctx)) }},
		{sdk.BareMetal, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.BareMetal(ctx)) }},
		{sdk.BareMetalIntrospection, func(ctx context.Context) (*gophercloud.ServiceClient, error) {
			return raw(c.BareMetalIntrospection(ctx))
		}},
		{sdk.Container, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.Container(ctx)) }},
		{sdk.ContainerInfra, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.ContainerInfra(ctx)) }},
		{sdk.Database, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.Database(ctx)) }},
		{sdk.DNS, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.DNS(ctx)) }},
		{sdk.Identity, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.Identity(ctx)) }},
		{sdk.KeyManager, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.KeyManager(ctx)) }},
		{sdk.LoadBalancer, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.LoadBalancer(ctx)) }},
		{sdk.Messaging, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.Messaging(ctx)) }},
		{sdk.Metric, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.Metric(ctx)) }},
		{sdk.ObjectStorage, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.ObjectStorage(ctx)) }},
		{sdk.Orchestration, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.Orchestration(ctx)) }},
		{sdk.Placement, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.Placement(ctx)) }},
		{sdk.Reservation, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.Reservation(ctx)) }},
		{sdk.SharedFileSystem, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.SharedFileSystem(ctx)) }},
		{sdk.Workflow, func(ctx context.Context) (*gophercloud.ServiceClient, error) { return raw(c.Workflow(ctx)) }},
	}
	for _, check := range checks {
		t.Run(string(check.service), func(t *testing.T) {
			client, err := check.get(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if client.ProviderClient != provider || client.Type != string(check.service) {
				t.Fatalf("client=%+v", client)
			}
			again, err := check.get(context.Background())
			if err != nil || again != client {
				t.Fatalf("cache err=%v", err)
			}
		})
	}
	if lookups.Load() != int32(len(checks)) {
		t.Fatalf("lookups=%d", lookups.Load())
	}
}

func TestConcurrentVersionProxiesAndIndependentEndpoints(t *testing.T) {
	cloud := testcloud.New(t)
	c, err := sdk.FromProvider(cloud.Provider,
		sdk.WithEndpointFor(sdk.BlockStorage, "v2", cloud.Server.URL+"/volume/v2/project"),
		sdk.WithEndpointFor(sdk.BlockStorage, "v3", cloud.Server.URL+"/volume/v3/project"),
		sdk.WithEndpoint(sdk.LoadBalancer, cloud.Server.URL+"/octavia/v2.0"),
	)
	if err != nil {
		t.Fatal(err)
	}
	clients := make([]*gophercloud.ServiceClient, 20)
	var wg sync.WaitGroup
	for i := range clients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client, lookupErr := raw(c.LoadBalancer(context.Background()))
			clients[i] = client
			if lookupErr != nil {
				t.Error(lookupErr)
			}
		}()
	}
	wg.Wait()
	for _, client := range clients {
		if client != clients[0] || client.ServiceURL("loadbalancers") != cloud.Server.URL+"/octavia/v2.0/loadbalancers" {
			t.Fatal(client)
		}
	}
	v2, err := c.BlockStorageV2(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	v3, err := c.BlockStorageV3(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if v2.RawClient() == v3.RawClient() || v2.RawClient().Endpoint != cloud.Server.URL+"/volume/v2/project/" || v3.RawClient().Endpoint != cloud.Server.URL+"/volume/v3/project/" {
		t.Fatalf("v2=%+v v3=%+v", v2.RawClient(), v3.RawClient())
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.LoadBalancer(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestAdditionalMicroversionsAndZaqarClientHeaders(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/baremetal/v1/ping", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-OpenStack-Ironic-API-Version") != "1.70" || r.Header.Get("OpenStack-API-Version") != "baremetal 1.70" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{}`)
	})
	c, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BareMetal, cloud.Server.URL+"/baremetal/v1"), sdk.WithMicroversion(sdk.BareMetal, "1.70"), sdk.WithEndpoint(sdk.Messaging, cloud.Server.URL+"/message/v2"))
	if err != nil {
		t.Fatal(err)
	}
	bare, err := c.BareMetal(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if _, err := bare.RawClient().Get(context.Background(), bare.RawClient().ServiceURL("ping"), &body, nil); err != nil {
		t.Fatal(err)
	}
	message, err := c.Messaging(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	id := message.RawClient().MoreHeaders["Client-ID"]
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(id) {
		t.Fatal(id)
	}
	if message.Queues.RawClient() != message.RawClient() || message.Messages.RawClient() != message.RawClient() {
		t.Fatal("resource clients differ")
	}
	for _, option := range []sdk.ConnectionOption{sdk.WithMicroversion(sdk.BareMetal, "2.1"), sdk.WithEndpointFor(sdk.Identity, "v4", "https://example.com"), sdk.WithMessagingClientID("not-a-uuid")} {
		if _, err := sdk.FromProvider(cloud.Provider, option); err == nil {
			t.Fatal("invalid option accepted")
		}
	}
	c, err = sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/volume/v3/project"), sdk.WithMicroversion(sdk.BlockStorage, "3.70"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.BlockStorageV2(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
