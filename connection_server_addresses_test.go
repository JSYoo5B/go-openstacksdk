package gophercloudsdk_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func connectionAddressServer() *compute.Server {
	return &compute.Server{ID: "server", Status: "ACTIVE", Addresses: map[string]any{"private": []any{map[string]any{"version": 4, "addr": "10.0.0.1", "OS-EXT-IPS:type": "fixed", "OS-EXT-IPS-MAC:mac_addr": "mac"}}}}
}

func TestConnectionServerAddressFacadesShareRoleCacheAndFreshSupplementation(t *testing.T) {
	cloud := testcloud.New(t)
	roles := connectionRoleFixture(t, cloud)
	ports, ips := 0, 0
	cloud.Mux.HandleFunc("GET /network/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		ports++
		if r.URL.Query().Get("device_id") != "server" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"ports":[{"id":"port","device_id":"server","mac_address":"mac"}]}`)
	})
	cloud.Mux.HandleFunc("GET /network/v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		ips++
		testcloud.JSON(w, 200, `{"floatingips":[{"port_id":"port","fixed_ip_address":"10.0.0.1","floating_ip_address":"8.8.8.8"}]}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute"), sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network"), sdk.WithNetworkRoles(network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "private", NATDestination: true})), sdk.WithServerAddressPolicy(compute.WithPrivateCloud(true), compute.WithAddressReachability(false), compute.WithLocalIPv6(false)))
	if err != nil {
		t.Fatal(err)
	}
	server := connectionAddressServer()
	ctx := context.Background()
	view, err := conn.ExpandServerInterfaces(ctx, server)
	if err != nil || view.InterfaceIP != "10.0.0.1" || view.AccessIPv4 != "10.0.0.1" || view.PublicIPv4 != "8.8.8.8" {
		t.Fatal(view, err)
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.ExpandServerInterfaces(ctx, server)
	if err != nil || second.InterfaceIP != view.InterfaceIP || roles.Load() != 1 || ports != 2 || ips != 2 {
		t.Fatalf("view=%+v err=%v role=%d ports=%d ips=%d", second, err, roles.Load(), ports, ips)
	}
	private, err := conn.GetServerPrivateIP(ctx, server)
	if err != nil || private != "10.0.0.1" || ports != 2 || ips != 2 {
		t.Fatalf("standalone private=%q err=%v ports=%d ips=%d", private, err, ports, ips)
	}
	conn.ResetNetworkRoles()
	if _, err := conn.GetServerPrivateIP(ctx, server); err != nil || roles.Load() != 2 {
		t.Fatal(err, roles.Load())
	}
}

func TestConnectionServerAddressShortcutsAndInvalidInputsSelectNoEndpoint(t *testing.T) {
	cloud := testcloud.New(t)
	locates := 0
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates++
		return "", errors.New("unexpected endpoint")
	}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithServerAddressPolicy(compute.WithAddressReachability(false)))
	if err != nil {
		t.Fatal(err)
	}
	server := &compute.Server{AccessIPv4: "8.8.8.8", Addresses: map[string]any{"bad": func() {}}}
	if got, err := conn.GetServerPublicIP(context.Background(), server); err != nil || got != "8.8.8.8" || locates != 0 {
		t.Fatal(got, err, locates)
	}
	if _, err := conn.GetServerPrivateIP(context.Background(), server, nil); !errors.Is(err, resource.ErrInvalidOption) || locates != 0 {
		t.Fatal(err, locates)
	}
	if _, err := conn.ExpandServerInterfaces(nil, server); !errors.Is(err, resource.ErrInvalidOption) || locates != 0 {
		t.Fatal(err, locates)
	}
	if _, err := conn.ExpandServerInterfaces(context.Background(), server); !errors.Is(err, resource.ErrInvalidOption) || locates != 0 {
		t.Fatal(err, locates)
	}
}

func TestConnectionServerAddressMissingNeutronRetainsNovaAndIPv6(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { return "", &gophercloud.ErrEndpointNotFound{} }
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithServerAddressPolicy(compute.WithLocalIPv6(true), compute.WithAddressReachability(false)))
	if err != nil {
		t.Fatal(err)
	}
	server := connectionAddressServer()
	server.AccessIPv6 = "2001:db8::1"
	view, err := conn.ExpandServerInterfaces(context.Background(), server)
	if err != nil || view.SupplementalError != nil || view.PrivateIPv4 != "10.0.0.1" || view.InterfaceIP != "2001:db8::1" {
		t.Fatal(view, err)
	}
}

func TestConnectServerAddressYAMLSourceAndTypedReplacement(t *testing.T) {
	for _, source := range []string{"none", "nova", "typed neutron"} {
		t.Run(source, func(t *testing.T) {
			cloud := testcloud.New(t)
			configuredSource := source
			if source == "typed neutron" {
				configuredSource = "none"
			}
			opts := defaultNetworkCloudOptions(t, cloud, "    private: true\n    force_ipv4: true\n    floating_ip_source: "+configuredSource+"\n    networks: [{name: private, nat_destination: true}]\n")
			if source == "typed neutron" {
				opts = append(opts, sdk.WithServerAddressPolicy(compute.WithFloatingIPSource(compute.FloatingIPNeutron), compute.WithAddressReachability(false), compute.WithLocalIPv6(false)))
			}
			connectionRoleFixture(t, cloud)
			ports, neutron, nova := 0, 0, 0
			cloud.Mux.HandleFunc("GET /network/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
				ports++
				testcloud.JSON(w, 200, `{"ports":[{"id":"port","device_id":"server","mac_address":"mac"}]}`)
			})
			cloud.Mux.HandleFunc("GET /network/v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				neutron++
				testcloud.JSON(w, 200, `{"floatingips":[{"port_id":"port","fixed_ip_address":"10.0.0.1","floating_ip_address":"8.8.8.8"}]}`)
			})
			cloud.Mux.HandleFunc("GET /compute/os-floating-ips", func(w http.ResponseWriter, r *http.Request) {
				nova++
				testcloud.JSON(w, 200, `{"floating_ips":[{"port_id":"port","fixed_ip":"10.0.0.1","ip":"8.8.8.8"}]}`)
			})
			conn, err := sdk.Connect(context.Background(), opts...)
			if err != nil {
				t.Fatal(err)
			}
			server := connectionAddressServer()
			server.AccessIPv6 = "2001:db8::1"
			view, err := conn.ExpandServerInterfaces(context.Background(), server, compute.WithAddressReachability(false))
			wantPorts, wantNeutron, wantNova, wantPublic, wantInterface, wantV6 := 0, 0, 0, "", "10.0.0.1", ""
			if source == "nova" {
				wantPorts, wantNova, wantPublic = 1, 1, "8.8.8.8"
			}
			if source == "typed neutron" {
				wantPorts, wantNeutron, wantPublic, wantInterface, wantV6 = 1, 1, "8.8.8.8", "8.8.8.8", "2001:db8::1"
			}
			if err != nil || view.PublicIPv4 != wantPublic || view.InterfaceIP != wantInterface || view.PublicIPv6 != wantV6 || ports != wantPorts || neutron != wantNeutron || nova != wantNova {
				t.Fatalf("view=%+v err=%v ports=%d neutron=%d nova=%d", view, err, ports, neutron, nova)
			}
		})
	}
}
