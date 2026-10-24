package compute_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// waitLaneOptions builds WaitForServer options over the lane fixture, whose
// raw server is BUILD on the first GET and ACTIVE afterwards.
func waitLaneOptions(f *laneFixture, extra ...compute.AutomaticFloatingIPOption) []compute.ServerReadyOption {
	f.ipStatus = "ACTIVE"
	f.rawStatus = func(n int32) string {
		if n == 1 {
			return "BUILD"
		}
		return "ACTIVE"
	}
	return []compute.ServerReadyOption{
		compute.WithServerReadyAutomaticIPOptions(laneOptions(extra...)...),
		compute.WithServerReadyWaitOptions(resource.WithPollInterval(time.Millisecond)),
	}
}

// Python's wait_for_server polls cloud get_server, which applies
// meta.add_server_interfaces to the found server, and passes that expanded
// server to get_active_server. Go expands the ACTIVE poll result once: the
// BUILD poll performs no Neutron request, the port's bound IP becomes a
// supplemental floating row, Interfaces reports it, and the automatic lane
// skips on it instead of assigning another IP.
func TestWaitForServerExpandsActivePollBeforeIPDecision(t *testing.T) {
	f := newLaneFixture(t)
	options := waitLaneOptions(f, compute.WithAutomaticEnsureOptions(network.WithEnsureProject("owner"), network.WithEnsurePort(resource.ID("port-p"))))
	f.bound = [2]string{"port-p", "10.0.0.10"}
	f.shownAfter = 1 << 30
	result, err := f.service.WaitForServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, options...)
	trace := f.trace()
	if err != nil || result == nil || result.Mode != compute.ServerIPAutomatic || result.Decision.Needed || result.Decision.Reason != compute.AutomaticIPExistingFloating || result.Assignment != nil || result.Observed || result.Server.Name != "raw-2" || f.rawCalls.Load() != 2 {
		t.Fatal(result, err, trace)
	}
	if !strings.HasPrefix(trace, "raw,raw,ports,") || strings.Contains(trace, "put") || strings.Contains(trace, "allocate") || len(f.bodies) != 0 {
		t.Fatal("expansion must follow the ACTIVE poll only", trace)
	}
	view := result.Interfaces
	if view == nil || view.PublicIPv4 != dispatchAddresses["pool"] || view.AccessIPv4 != dispatchAddresses["pool"] || view.PrivateIPv4 != "10.0.0.10" || view.SupplementalError != nil {
		t.Fatal(view)
	}
	supplemental := 0
	for _, row := range view.Addresses["private"] {
		if row.Supplemental && row.Type == "floating" && row.Address == dispatchAddresses["pool"] {
			supplemental++
		}
	}
	if supplemental != 1 || result.Decision.Addresses != view {
		t.Fatal(view.Addresses, result.Decision.Addresses)
	}
}

// The address lane decides _attach_ip_to_server's short circuit on the
// expanded server too: the requested IP, already bound in Neutron but not yet
// shown by raw Nova, is the only floating address of the expanded model, so
// the item completes without a refresh, port lookup, PUT or observation.
func TestWaitForServerAddressLaneShortCircuitsOnExpandedServer(t *testing.T) {
	f := newLaneFixture(t)
	options := waitLaneOptions(f, compute.WithFloatingIPAddresses(dispatchAddresses["pool"]))
	f.bound = [2]string{"port-p", "10.0.0.10"}
	f.shownAfter = 1 << 30
	result, err := f.service.WaitForServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, options...)
	trace := f.trace()
	if err != nil || result == nil || result.Mode != compute.ServerIPExplicit || len(result.Attempts) != 1 || !result.Attempts[0].ShortCircuited || !result.Attempts[0].Completed || result.Attempts[0].Observed || result.Observed || f.rawCalls.Load() != 2 || result.Interfaces == nil {
		t.Fatal(result, err, trace)
	}
	if strings.Contains(trace, "put") || strings.Contains(trace, "port:") || !strings.HasSuffix(trace, "list:floating_ip_address="+dispatchAddresses["pool"]) {
		t.Fatal(trace)
	}
}

// wait_for_server forwards reuse and nat_destination to get_active_server and
// add_ips_to_server; Go forwards the same Ensure options, plus a fixed
// address, through WaitForServer to the pool and automatic lanes. Reuse binds
// the free IP by PUT and no reuse allocates by POST, each on the selected
// port and fixed address, then waits for the IP and observes it in raw Nova.
func TestWaitForServerForwardsFixedAddressNATDestinationAndReuse(t *testing.T) {
	cases := []struct {
		name   string
		option []network.EnsureFloatingIPOption
		pool   bool
		port   string
		fixed  string
	}{
		{name: "pool/fixed", pool: true, option: []network.EnsureFloatingIPOption{network.WithEnsureFixedAddress("10.1.0.20")}, port: "port-n", fixed: "10.1.0.20"},
		{name: "pool/nat-id", pool: true, option: []network.EnsureFloatingIPOption{network.WithEnsureNATDestination(resource.ID("nat-net"))}, port: "port-n", fixed: "10.1.0.20"},
		{name: "pool/nat-fixed", pool: true, option: []network.EnsureFloatingIPOption{network.WithEnsureNATDestination(resource.Name("nat")), network.WithEnsureFixedAddress("10.1.0.20")}, port: "port-n", fixed: "10.1.0.20"},
		{name: "pool/fixed-private", pool: true, option: []network.EnsureFloatingIPOption{network.WithEnsureFixedAddress("10.0.0.10")}, port: "port-p", fixed: "10.0.0.10"},
		{name: "automatic/nat-name", option: []network.EnsureFloatingIPOption{network.WithEnsureNATDestination(resource.Name("nat"))}, port: "port-n", fixed: "10.1.0.20"},
		{name: "automatic/fixed", option: []network.EnsureFloatingIPOption{network.WithEnsureFixedAddress("10.1.0.20")}, port: "port-n", fixed: "10.1.0.20"},
	}
	for _, tc := range cases {
		for _, reuse := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/reuse=%v", tc.name, reuse), func(t *testing.T) {
				f := newLaneFixture(t)
				ensure := append([]network.EnsureFloatingIPOption{network.WithEnsureProject("owner"), network.WithEnsureReuse(reuse), network.WithEnsureWait(resource.WithPollInterval(time.Millisecond))}, tc.option...)
				extra := []compute.AutomaticFloatingIPOption{compute.WithAutomaticEnsureOptions(ensure...)}
				mode := compute.ServerIPAutomatic
				if tc.pool {
					extra, mode = append(extra, compute.WithFloatingIPPool(resource.ID("pool-network"))), compute.ServerIPPool
				}
				options := waitLaneOptions(f, extra...)
				f.free = reuse
				result, err := f.service.WaitForServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, options...)
				trace := f.trace()
				if err != nil || result == nil || result.Mode != mode || !result.Observed || result.Assignment == nil || result.Assignment.Allocated == reuse || result.Assignment.FloatingIP.Status != "ACTIVE" || result.Server.Name != "raw-3" || f.rawCalls.Load() != 3 || result.Interfaces == nil {
					t.Fatal(result, err, trace)
				}
				if len(f.bodies) != 1 || f.bodies[0]["port_id"] != tc.port || f.bodies[0]["fixed_ip_address"] != tc.fixed || reuse != strings.Contains(trace, "put") || reuse == strings.Contains(trace, "allocate") {
					t.Fatal(f.bodies, trace)
				}
				if selection := result.Decision.Selection; selection.PortID != tc.port || selection.FixedIPv4 != tc.fixed || selection.NetworkID != "pool-network" {
					t.Fatal(selection)
				}
			})
		}
	}
	t.Run("automatic/no-selector", func(t *testing.T) {
		f := newLaneFixture(t)
		options := waitLaneOptions(f, compute.WithAutomaticEnsureOptions(network.WithEnsureProject("owner")))
		result, err := f.service.WaitForServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, options...)
		trace := f.trace()
		if !errors.Is(err, resource.ErrAmbiguous) || result == nil || result.Assignment != nil || result.Decision.Needed || len(f.bodies) != 0 || f.rawCalls.Load() != 2 {
			t.Fatal(result, err, trace)
		}
	})
}

// Without a Network service (Python's has_service('network') false) or with
// the Nova floating IP source, wait_for_server reaches the Nova-network
// branches of add_ips_to_server. Go's WaitForServer polls raw ACTIVE once,
// then the shared dispatch reuses or allocates a Nova pool IP, sends one
// addFloatingIp action with the fixed address and observes the address.
func TestWaitForServerNovaNetworkFallbackLanes(t *testing.T) {
	cases := []struct {
		name   string
		extra  []compute.AutomaticFloatingIPOption
		mode   compute.ServerIPDispatchMode
		expect []string
	}{
		{name: "pool-reuse", extra: []compute.AutomaticFloatingIPOption{compute.WithFloatingIPPool(resource.Name("public"))}, mode: compute.ServerIPPool, expect: []string{"raw", "list", "get:2", "attach:" + dispatchAddresses["b"], "raw"}},
		{name: "pool-allocate", extra: []compute.AutomaticFloatingIPOption{compute.WithFloatingIPPool(resource.Name("public")), compute.WithAutomaticEnsureOptions(network.WithEnsureReuse(false))}, mode: compute.ServerIPPool, expect: []string{"raw", "allocate", "get:29", "get:29", "attach:" + dispatchAddresses["pool"], "raw"}},
		{name: "automatic", mode: compute.ServerIPAutomatic, expect: []string{"raw", "pools", "list", "get:2", "attach:" + dispatchAddresses["b"], "raw"}},
		{name: "address", extra: []compute.AutomaticFloatingIPOption{compute.WithFloatingIPAddresses(dispatchAddresses["b"])}, mode: compute.ServerIPExplicit, expect: []string{"raw", "list", "get:2", "attach:" + dispatchAddresses["b"], "raw"}},
	}
	for _, source := range []string{"no-network", "nova-source"} {
		for _, tc := range cases {
			t.Run(source+"/"+tc.name, func(t *testing.T) {
				f := newNovaIPFixture(t, "")
				f.status = "ACTIVE"
				address := []compute.ServerAddressOption{compute.WithAddressReachability(false)}
				if source == "nova-source" {
					address = append(address, compute.WithFloatingIPSource(compute.FloatingIPNova))
				}
				automatic := append([]compute.AutomaticFloatingIPOption{compute.WithAutomaticAddressOptions(address...), compute.WithAutomaticEnsureOptions(network.WithEnsureFixedAddress("10.0.0.10")), compute.WithAutomaticIPPollInterval(time.Millisecond)}, tc.extra...)
				server := automaticServer(t, autoFixed)
				server.Status = "BUILD"
				result, err := f.service.WaitForServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, compute.WithServerReadyAutomaticIPOptions(automatic...), compute.WithServerReadyWaitOptions(resource.WithPollInterval(time.Millisecond)))
				if err != nil || result == nil || result.Mode != tc.mode || result.Decision.Backend != compute.FloatingIPNova || result.NovaAssignment == nil || !result.NovaAssignment.ActionAccepted || result.Assignment != nil || !result.Observed || result.Interfaces == nil || strings.Join(f.trace(), ",") != strings.Join(tc.expect, ",") {
					t.Fatal(result, err, f.trace())
				}
			})
		}
	}
}
