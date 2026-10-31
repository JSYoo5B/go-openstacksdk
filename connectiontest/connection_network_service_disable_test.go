package connectiontest_test

import (
	"context"
	"strings"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

// TestNetworkRolesHonourHasNetworkConfiguration covers the has_service
// ('network') check of _find_interesting_networks: a has_network: false flag,
// from WithServiceEnabled or clouds.yaml, makes every role getter empty without
// a network list, even with configured roles that would otherwise fail
// validation. Re-enabling the type restores discovery, and use_* flags still
// win first.
func TestNetworkRolesHonourHasNetworkConfiguration(t *testing.T) {
	for _, scenario := range []string{"typed disable", "yaml disable", "yaml disable typed enable", "yaml enable"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := connectionRoleFixture(t, cloud)
			flag := "false"
			if scenario == "yaml enable" {
				flag = "true"
			}
			settings := "    networks: [{name: private, nat_destination: true, default_interface: true}, {name: public, nat_source: true}]\n"
			if strings.HasPrefix(scenario, "yaml") {
				settings += "    has_network: " + flag + "\n"
			}
			opts := defaultNetworkCloudOptions(t, cloud, settings)
			switch scenario {
			case "typed disable":
				opts = append(opts, sdk.WithServiceEnabled("Network", false))
			case "yaml disable typed enable":
				opts = append(opts, sdk.WithServiceEnabled("network", true))
			}
			conn, err := sdk.Connect(context.Background(), opts...)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			destination, err := conn.GetNATDestination(ctx)
			if err != nil {
				t.Fatal(err)
			}
			source, err := conn.GetNATSource(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defaultNetwork, err := conn.GetDefaultNetwork(ctx)
			if err != nil {
				t.Fatal(err)
			}
			enabled := scenario == "yaml enable" || scenario == "yaml disable typed enable"
			if conn.ServiceEnabled("network") != enabled || !conn.UseExternalNetwork() || !conn.UseInternalNetwork() {
				t.Fatal(conn.ServiceEnabled("network"))
			}
			if !enabled {
				if destination != nil || source != nil || defaultNetwork != nil || calls.Load() != 0 {
					t.Fatal(destination, source, defaultNetwork, calls.Load())
				}
				return
			}
			if destination == nil || destination.ID != "private" || source == nil || source.ID != "public" || defaultNetwork == nil || defaultNetwork.ID != "private" || calls.Load() != 1 {
				t.Fatal(destination, source, defaultNetwork, calls.Load())
			}
		})
	}
}

// TestFloatingIPHelpersUseNovaWhenNetworkServiceDisabled covers the
// has_service('network') half of _use_neutron_floating: with the Neutron
// source but has_network: false, AddAutoIP and AddIPList use the Nova lane and
// never contact the network endpoint.
func TestFloatingIPHelpersUseNovaWhenNetworkServiceDisabled(t *testing.T) {
	for _, helper := range []string{"AddAutoIP", "AddIPList", "AddIPsToServer"} {
		t.Run(helper, func(t *testing.T) {
			attached := false
			body := func() string {
				return `{"server":{"id":"srv-1","status":"ACTIVE","addresses":{"private":[{"version":4,"addr":"10.0.0.8","OS-EXT-IPS:type":"fixed"},{"version":4,"addr":"198.51.100.21","OS-EXT-IPS:type":"floating"}]}}}`
			}
			conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, r2CloudNetNovaAutoHandler(&attached, body), sdk.WithServiceEnabled("network", false))
			server := &compute.Server{ID: "srv-1", Status: "ACTIVE"}
			var err error
			switch helper {
			case "AddAutoIP":
				var result *compute.AddAutoIPResult
				result, err = conn.AddAutoIP(context.Background(), server, compute.WithAddAutoIPWait(true), compute.WithAddAutoIPPollInterval(time.Millisecond))
				if err == nil && (result.Backend != compute.FloatingIPNova || result.NeutronFloating || !result.Reused || !result.Observed) {
					t.Fatal(result)
				}
			case "AddIPList":
				var result *compute.AutomaticServerIPResult
				result, err = conn.AddIPList(context.Background(), server, []string{"198.51.100.21"})
				if err == nil && (result.Decision == nil || result.Decision.Backend != compute.FloatingIPNova || result.NovaAssignment == nil) {
					t.Fatal(result)
				}
			case "AddIPsToServer":
				var result *compute.AutomaticServerIPResult
				result, err = conn.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, compute.WithServerIPAutomaticOptions(compute.WithFloatingIPAddresses("198.51.100.21")))
				if err == nil && (result.Decision == nil || result.Decision.Backend != compute.FloatingIPNova || result.NovaAssignment == nil) {
					t.Fatal(result)
				}
			}
			if err != nil {
				t.Fatal(err, f.keys())
			}
			for _, key := range f.keys() {
				if strings.Contains(key, "/network/") {
					t.Fatal("network endpoint contacted", f.keys())
				}
			}
			if f.count("POST /compute/servers/srv-1/action") != 1 {
				t.Fatal(f.keys())
			}
		})
	}
}
