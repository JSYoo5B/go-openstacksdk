package connectiontest_test

import (
	"context"
	"errors"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const regionValuesSettings = "    networks: [{name: public, nat_destination: true, default_interface: true}]\n" +
	"    regions:\n" +
	"      - name: r1\n" +
	"        values:\n" +
	"          networks: [{name: private, nat_destination: true, default_interface: true}]\n" +
	"      - r2\n" +
	"      - name: r3\n" +
	"        values: {use_external_network: false, use_internal_network: false}\n" +
	"      - name: r4\n" +
	"        values: {has_network: false, networks: null}\n"

// TestCloudRegionValuesOverlayNetworkRoleConfiguration covers the region step
// of OpenStackConfig.get_one for network role settings: the selected regions
// entry's non-null values replace the cloud settings, including the whole
// networks list, use_* flags and has_<service> flags. Without WithRegion the
// first entry is selected (or the entry named by region_name); an explicit
// region must name an entry, and malformed entries are configuration errors.
func TestCloudRegionValuesOverlayNetworkRoleConfiguration(t *testing.T) {
	for _, scenario := range []struct {
		name, region, extra string
		destination         string
		calls               int32
		invalid             bool
	}{
		{name: "first entry by default", destination: "private", calls: 1},
		{name: "explicit value-less entry", region: "r2", destination: "public", calls: 1},
		{name: "explicit flags entry", region: "r3", calls: 0},
		{name: "has_network disable and null kept", region: "r4", calls: 0},
		{name: "region_name selects entry", extra: "    region_name: r3\n", calls: 0},
		{name: "explicit unknown region", region: "r9", invalid: true},
		{name: "invalid entry", extra: "    regions: [{name: r1, extra: true}]\n", invalid: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := connectionRoleFixture(t, cloud)
			settings := regionValuesSettings
			if scenario.name == "invalid entry" {
				settings = scenario.extra
			} else {
				settings += scenario.extra
			}
			opts := defaultNetworkCloudOptions(t, cloud, settings)
			if scenario.region != "" {
				opts = append(opts, sdk.WithRegion(scenario.region))
			}
			conn, err := sdk.Connect(context.Background(), opts...)
			if scenario.invalid {
				if !errors.Is(err, resource.ErrInvalidOption) || conn != nil {
					t.Fatal(conn, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			destination, err := conn.GetNATDestination(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defaultNetwork, err := conn.GetDefaultNetwork(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if conn.ServiceEnabled("network") != (scenario.name != "has_network disable and null kept") || calls.Load() != scenario.calls {
				t.Fatal(conn.ServiceEnabled("network"), calls.Load())
			}
			if scenario.destination == "" {
				if destination != nil || defaultNetwork != nil {
					t.Fatal(destination, defaultNetwork)
				}
				return
			}
			if destination == nil || destination.ID != scenario.destination || defaultNetwork == nil || defaultNetwork.ID != scenario.destination {
				t.Fatal(destination, defaultNetwork)
			}
		})
	}
}
