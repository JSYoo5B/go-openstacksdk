package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func connectionRoleIDs(rows []*network.RoleNetwork) []string {
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	return ids
}
func connectionRoleFixture(t *testing.T, cloud *testcloud.Cloud) *atomic.Int32 {
	t.Helper()
	calls := new(atomic.Int32)
	cloud.Mux.HandleFunc("GET /network/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"networks":[{"id":"public","name":"public","router:external":true,"tags":[],"subnets":null,"availability_zone_hints":[]},{"id":"phys","name":"physical","provider:physical_network":"physical"},{"id":"private","name":"private"}]}`)
	})
	return calls
}

func TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset(t *testing.T) {
	cloud := testcloud.New(t)
	calls := connectionRoleFixture(t, cloud)
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network"), sdk.WithNetworkRoles(network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "private", NATDestination: true, DefaultInterface: true})))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	getters := []struct {
		get  func(context.Context) ([]*network.RoleNetwork, error)
		want []string
	}{
		{conn.GetExternalIPv4Networks, []string{"public", "phys"}}, {conn.GetInternalIPv4Networks, []string{"private"}},
		{conn.GetExternalIPv6Networks, []string{"public"}}, {conn.GetInternalIPv6Networks, []string{"phys", "private"}},
		{conn.GetExternalIPv4FloatingNetworks, []string{"public"}}, {conn.GetExternalNetworks, []string{"public", "phys", "public"}},
		{conn.GetInternalNetworks, []string{"private", "phys", "private"}},
	}
	for _, g := range getters {
		rows, err := g.get(ctx)
		if err != nil || !reflect.DeepEqual(connectionRoleIDs(rows), g.want) {
			t.Fatalf("rows=%v err=%v want=%v", connectionRoleIDs(rows), err, g.want)
		}
		if len(rows) > 0 {
			rows[0].Name = "changed"
		}
	}
	for _, g := range []struct {
		get func(context.Context) (*network.RoleNetwork, error)
		id  string
	}{{conn.GetNATSource, "public"}, {conn.GetNATDestination, "private"}, {conn.GetDefaultNetwork, "private"}} {
		row, err := g.get(ctx)
		if err != nil || row.ID != g.id || row.Name == "changed" {
			t.Fatal(row, err)
		}
		row.ID = "changed"
	}
	snapshot, err := conn.GetNetworkRoles(ctx)
	if err != nil || snapshot.NATSource.ID != "public" || snapshot.NATDestination.ID != "private" || snapshot.DefaultNetwork.ID != "private" || calls.Load() != 1 {
		t.Fatal(snapshot, err, calls.Load())
	}
	// Preserve explicit [] versus null from the native network model.
	if snapshot.ExternalIPv4[0].Tags == nil || snapshot.ExternalIPv4[0].Subnets != nil || snapshot.ExternalIPv4[0].AvailabilityZoneHints == nil {
		t.Fatalf("array presence lost: %+v", snapshot.ExternalIPv4[0])
	}
	conn.ResetNetworkRoles()
	if _, err := conn.GetNetworkRoles(ctx); err != nil || calls.Load() != 2 {
		t.Fatal(err, calls.Load())
	}
}

func TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC(t *testing.T) {
	for _, scenario := range []string{"YAML", "typed replacement", "empty replacement"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := connectionRoleFixture(t, cloud)
			opts := defaultNetworkCloudOptions(t, cloud, "    networks: [{name: private, routes_externally: true, routes_ipv6_externally: false, nat_destination: true, default_interface: true}]\n")
			wantDefault := "private"
			if scenario == "typed replacement" {
				opts = append(opts, sdk.WithNetworkRoles(network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "phys", RoutesIPv6Externally: true, NATSource: true, NATDestination: true, DefaultInterface: true})))
				wantDefault = "phys"
			}
			if scenario == "empty replacement" {
				opts = append(opts, sdk.WithNetworkRoles())
				wantDefault = ""
			}
			var subnets, creates atomic.Int32
			cloud.Mux.HandleFunc("GET /network/v2.0/subnets", func(w http.ResponseWriter, r *http.Request) { subnets.Add(1); testcloud.JSON(w, 200, `{"subnets":[]}`) })
			cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
				creates.Add(1)
				var body struct{ Server map[string]any }
				_ = json.NewDecoder(r.Body).Decode(&body)
				if wantDefault == "" {
					if _, present := body.Server["networks"]; present {
						t.Error(body.Server)
					}
				} else if !reflect.DeepEqual(body.Server["networks"], []any{map[string]any{"uuid": wantDefault}}) {
					t.Error(body.Server)
				}
				testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
			})
			conn, err := sdk.Connect(context.Background(), opts...)
			if err != nil {
				t.Fatal(err)
			}
			roles, err := conn.GetNetworkRoles(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "YAML":
				if roles.DefaultNetwork.ID != "private" || !reflect.DeepEqual(connectionRoleIDs(roles.ExternalIPv4), []string{"public", "phys", "private"}) || !reflect.DeepEqual(connectionRoleIDs(roles.ExternalIPv6), []string{"public"}) {
					t.Fatal(roles)
				}
			case "typed replacement":
				if roles.DefaultNetwork.ID != "phys" || roles.NATSource.ID != "phys" || !reflect.DeepEqual(connectionRoleIDs(roles.ExternalIPv4), []string{"public"}) || !reflect.DeepEqual(connectionRoleIDs(roles.ExternalIPv6), []string{"public", "phys"}) {
					t.Fatal(roles)
				}
			case "empty replacement":
				if roles.DefaultNetwork != nil || len(roles.InternalIPv4) != 1 || subnets.Load() != 1 {
					t.Fatal(roles, subnets.Load())
				}
			}
			computeService, err := conn.Compute(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := computeService.Servers.Create(context.Background(), nicConnectionRequest()); err != nil || creates.Load() != 1 {
				t.Fatal(err, creates.Load())
			}
			wantLists := int32(1)
			if calls.Load() != wantLists {
				t.Fatal(calls.Load(), wantLists)
			}
		})
	}
}

func TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight(t *testing.T) {
	for _, scenario := range []string{"both disabled", "one disabled", "missing endpoint", "missing endpoint value", "catalog error", "nil context", "canceled"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			requests := connectionRoleFixture(t, cloud)
			var catalog atomic.Int32
			sentinel := errors.New("catalog failed")
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
				catalog.Add(1)
				if scenario == "missing endpoint" {
					return "", &gophercloud.ErrEndpointNotFound{}
				}
				if scenario == "missing endpoint value" {
					return "", gophercloud.ErrEndpointNotFound{}
				}
				return "", sentinel
			}
			opts := []sdk.ConnectionOption{}
			if scenario == "both disabled" {
				opts = append(opts, sdk.WithNetworkRoles(network.WithExternalNetworkDiscovery(false), network.WithInternalNetworkDiscovery(false)))
			}
			if scenario == "one disabled" {
				opts = append(opts, sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network"), sdk.WithNetworkRoles(network.WithExternalNetworkDiscovery(false), network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "private", NATDestination: true})))
			}
			conn, err := sdk.FromProvider(cloud.Provider, opts...)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			if scenario == "nil context" {
				ctx = nil
			}
			if scenario == "canceled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			result, err := conn.GetNetworkRoles(ctx)
			switch scenario {
			case "both disabled", "missing endpoint", "missing endpoint value":
				if err != nil || len(result.ExternalIPv4) != 0 {
					t.Fatal(result, err)
				}
			case "one disabled":
				if err != nil || conn.UseExternalNetwork() || !conn.UseInternalNetwork() || !reflect.DeepEqual(connectionRoleIDs(result.ExternalIPv4), []string{"public", "phys"}) || requests.Load() != 1 {
					t.Fatal(result, err)
				}
			case "catalog error":
				if !errors.Is(err, sentinel) {
					t.Fatal(err)
				}
			case "nil context":
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			case "canceled":
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			}
			wantCatalog := int32(0)
			if scenario == "missing endpoint" || scenario == "missing endpoint value" || scenario == "catalog error" {
				wantCatalog = 1
			}
			if catalog.Load() != wantCatalog {
				t.Fatal(catalog.Load())
			}
		})
	}
	if _, err := sdk.FromProvider(&gophercloud.ProviderClient{}, sdk.WithNetworkRoles(nil)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestConnectNetworkRoleBooleanFlagsAndExplicitPolicyReplacement(t *testing.T) {
	for _, scenario := range []string{"boolean false", "string false", "typed restore", "numeric invalid"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := connectionRoleFixture(t, cloud)
			flag := "false"
			if scenario == "string false" {
				flag = "'FaLsE'"
			}
			if scenario == "numeric invalid" {
				flag = "1"
			}
			opts := defaultNetworkCloudOptions(t, cloud, "    use_external_network: "+flag+"\n    use_internal_network: false\n    networks: [{name: private, nat_destination: true}]\n")
			if scenario == "typed restore" {
				opts = append(opts, sdk.WithNetworkRoles(network.WithExternalNetworkDiscovery(true), network.WithInternalNetworkDiscovery(false), network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "private", NATDestination: true})))
			}
			conn, err := sdk.Connect(context.Background(), opts...)
			if scenario == "numeric invalid" {
				if !errors.Is(err, resource.ErrInvalidOption) || conn != nil || calls.Load() != 0 {
					t.Fatal(conn, err, calls.Load())
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			roles, err := conn.GetNetworkRoles(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "typed restore" {
				if !conn.UseExternalNetwork() || conn.UseInternalNetwork() || calls.Load() != 1 || len(roles.ExternalIPv4) != 2 {
					t.Fatal(roles, calls.Load())
				}
			} else if conn.UseExternalNetwork() || conn.UseInternalNetwork() || calls.Load() != 0 || len(roles.ExternalIPv4) != 0 {
				t.Fatal(roles, calls.Load())
			}
		})
	}
}
