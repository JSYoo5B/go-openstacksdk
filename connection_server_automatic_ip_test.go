package openstack_test

import (
	"context"
	"errors"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionAutomaticIPKnownSkipsUseFrozenPolicyWithoutEndpoints(t *testing.T) {
	for _, scenario := range []string{"private", "disabled source", "floating6", "call overlay"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			locates := 0
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
				locates++
				return "", errors.New("endpoint must stay lazy")
			}
			addresses := []compute.ServerAddressOption{compute.WithAddressReachability(false)}
			server := connectionAddressServer()
			var options []compute.AutomaticFloatingIPOption
			switch scenario {
			case "private":
				addresses = append(addresses, compute.WithPrivateCloud(true))
				server.Addresses = nil
			case "disabled source":
				addresses = append(addresses, compute.WithFloatingIPSource(compute.FloatingIPNone))
				server.Addresses = nil
			case "floating6":
				server.Addresses = map[string]any{"private": []any{map[string]any{"version": 6, "addr": "2001:db8::1", "OS-EXT-IPS:type": "floating"}}}
			case "call overlay":
				options = append(options, compute.WithAutomaticAddressOptions(compute.WithPrivateCloud(true)))
				server.Addresses = nil
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithServerAddressPolicy(addresses...))
			if err != nil {
				t.Fatal(err)
			}
			result, err := conn.EnsureServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, options...)
			if err != nil || result == nil || result.Decision.Needed || result.Decision.Reason == compute.AutomaticIPUndetermined || result.Assignment != nil || locates != 0 {
				t.Fatal(result, err, locates)
			}
		})
	}
}

func TestConnectionAutomaticIPMissingNetworkRequiresComputeForNovaExecution(t *testing.T) {
	cloud := testcloud.New(t)
	locates := 0
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		locates++
		if (locates == 1 && opts.Type != "network") || (locates == 2 && opts.Type != "compute") {
			t.Error("unexpected endpoint order", opts.Type, locates)
		}
		return "", &gophercloud.ErrEndpointNotFound{}
	}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithServerAddressPolicy(compute.WithAddressReachability(false)))
	if err != nil {
		t.Fatal(err)
	}
	result, err := conn.EnsureServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: connectionAddressServer()})
	if result == nil || !result.Decision.Needed || result.Decision.Backend != compute.FloatingIPNova || !errors.Is(err, resource.ErrUnsupported) || result.Assignment != nil || result.NovaAssignment != nil || locates != 2 {
		t.Fatal(result, err, locates)
	}
}
