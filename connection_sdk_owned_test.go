package gophercloudsdk

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

func TestSDKOwnedV1CatalogPreservesVersionProjectAndSharedPolicy(t *testing.T) {
	for _, check := range []struct {
		service        Service
		endpoint, want string
	}{
		{Clustering, "https://cloud.test/senlin", "https://cloud.test/senlin/v1/"},
		{Clustering, "https://cloud.test/proxy/senlin/v1/", "https://cloud.test/proxy/senlin/v1/"},
		{InstanceHA, "https://cloud.test/instance-ha/v1/project", "https://cloud.test/instance-ha/v1/project/"},
		{InstanceHA, "https://cloud.test/proxy/ha/v1/project%2Fopaque", "https://cloud.test/proxy/ha/v1/project%2Fopaque/"},
	} {
		t.Run(string(check.service)+check.endpoint, func(t *testing.T) {
			lookups := 0
			provider := &gophercloud.ProviderClient{EndpointLocator: func(o gophercloud.EndpointOpts) (string, error) {
				lookups++
				if o.Type != string(check.service) || o.Region != "region" || o.Availability != gophercloud.AvailabilityInternal {
					t.Fatalf("catalog selection: %#v", o)
				}
				return check.endpoint, nil
			}}
			provider.UseTokenLock()
			conn, err := FromProvider(provider, WithRegion("region"), WithInterface(gophercloud.AvailabilityInternal), WithMicroversion(check.service, "1.5"))
			if err != nil {
				t.Fatal(err)
			}
			client, err := conn.serviceClient(context.Background(), check.service)
			if err != nil || client.Endpoint != check.want || client.ProviderClient != provider || client.Type != string(check.service) || client.Microversion != "1.5" {
				t.Fatalf("client=%#v err=%v", client, err)
			}
			again, err := conn.serviceClient(context.Background(), check.service)
			if err != nil || again != client || lookups != 1 {
				t.Fatalf("cache client=%p again=%p lookups=%d err=%v", client, again, lookups, err)
			}
		})
	}
}

func TestSDKOwnedV1ExactOverridesAndCanceledCalls(t *testing.T) {
	provider := &gophercloud.ProviderClient{EndpointLocator: func(gophercloud.EndpointOpts) (string, error) {
		t.Fatal("explicit endpoint consulted catalog")
		return "", nil
	}}
	provider.UseTokenLock()
	conn, err := FromProvider(provider, WithEndpointFor(InstanceHA, "v1", "https://cloud.test/ha/v1/project"), WithMicroversion(InstanceHA, "1.3"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := conn.serviceClient(ctx, InstanceHA); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled service: %v", err)
	}
	client, err := conn.serviceClient(context.Background(), InstanceHA)
	if err != nil || client.Endpoint != "https://cloud.test/ha/v1/project/" || client.ProviderClient != provider {
		t.Fatalf("override: %#v %v", client, err)
	}
	if _, err := FromProvider(provider, WithMicroversion(Clustering, "2.0")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("wrong major: %v", err)
	}
	for _, endpoint := range []string{"https://cloud.test/ha/v2/project", "https://cloud.test/ha/v2"} {
		conn, err := FromProvider(provider, WithEndpoint(InstanceHA, endpoint))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := conn.serviceClient(context.Background(), InstanceHA); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatalf("accepted wrong endpoint version %q: %v", endpoint, err)
		}
	}
}

func TestMasakariProjectDiscoveryAndCanonicalHeader(t *testing.T) {
	var server *httptest.Server
	discovery, requests := 0, 0
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/proxy/instance-ha/v1/":
			discovery++
			if r.Header.Get("OpenStack-API-Version") != "" {
				t.Errorf("discovery used selected version: %#v", r.Header)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"id":"v1.0","min_version":"1.0","version":"1.3"}}`))
		case "/proxy/instance-ha/v1/project/ping":
			requests++
			if r.Header.Get("OpenStack-API-Version") != "instance-ha 1.3" || r.Header.Get("X-Auth-Token") != "shared-token" {
				t.Errorf("resource headers: %#v", r.Header)
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected URL %s", r.URL)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	provider := &gophercloud.ProviderClient{HTTPClient: *server.Client()}
	provider.UseTokenLock()
	provider.SetToken("shared-token")
	conn, err := FromProvider(provider, WithEndpoint(InstanceHA, server.URL+"/proxy/instance-ha/v1/project"), WithLatestMicroversion(InstanceHA))
	if err != nil {
		t.Fatal(err)
	}
	client, err := conn.serviceClient(context.Background(), InstanceHA)
	if err != nil {
		t.Fatal(err)
	}
	var output map[string]any
	if _, err := client.Get(context.Background(), client.ServiceURL("ping"), &output, nil); err != nil {
		t.Fatal(err)
	}
	selection, err := conn.Microversion(context.Background(), InstanceHA)
	if err != nil || selection.Selected != "1.3" || selection.DiscoveryURL != server.URL+"/proxy/instance-ha/v1/" || !selection.Negotiated || discovery != 1 || requests != 1 {
		t.Fatalf("selection=%#v discovery=%d requests=%d err=%v", selection, discovery, requests, err)
	}
}
