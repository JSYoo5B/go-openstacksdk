package openstack_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// cloudNetReuseFallbackHandler serves the Neutron-404 to Nova fallback of
// cloudNetNovaFallbackHandler with configurable server ports and Neutron PUT.
// The private network has a gateway subnet, so it is the discovered NAT
// destination unless the ports body is marked "no-nat".
func cloudNetReuseFallbackHandler(ports string, portsCode, put int, putBody string, serverBody func() string) func(r2CloudNetCall) (int, string) {
	deleted := false
	base := cloudNetNovaFallbackHandler("null", &deleted)
	return func(c r2CloudNetCall) (int, string) {
		switch c.key() {
		case "GET /network/v2.0/subnets":
			if strings.Contains(ports, "no-nat") {
				return 200, `{"subnets":[]}`
			}
			return 200, `{"subnets":[{"id":"sub","network_id":"external","ip_version":4},{"id":"priv","network_id":"private","ip_version":4,"gateway_ip":"10.0.0.1"}]}`
		case "GET /network/v2.0/ports":
			return portsCode, ports
		case "PUT /network/v2.0/floatingips/12":
			return put, putBody
		case "GET /compute/servers/srv-1":
			if serverBody != nil {
				return 200, serverBody()
			}
		}
		return base(c)
	}
}

const cloudNetTwoPorts = `{"ports":[` +
	`{"id":"port-old","device_id":"srv-1","network_id":"private","created_at":"2024-01-01T00:00:00Z","fixed_ips":[{"ip_address":"10.0.0.5"}]},` +
	`{"id":"port-new","device_id":"srv-1","network_id":"private","created_at":"2024-02-01T00:00:00Z","fixed_ips":[{"ip_address":"fd00::7"},{"ip_address":"10.0.0.9"}]}]}`

// TestCloudNetAddAutoIPReuseNovaFallbackRowUsesNeutronAttachAndIgnoresNotFound
// covers the pinned reuse path under the Neutron source when the available
// row came from Nova: Python sends the Neutron port update for that row,
// ignores its NotFound and never falls through to addFloatingIp.
func TestCloudNetAddAutoIPReuseNovaFallbackRowUsesNeutronAttachAndIgnoresNotFound(t *testing.T) {
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, cloudNetReuseFallbackHandler(cloudNetTwoPorts, 200, 404, cloudNetNeutronMissing, nil))
	server := &compute.Server{ID: "srv-1", Status: "ACTIVE"}
	result, err := conn.AddAutoIP(context.Background(), server)
	if err != nil || result == nil || !result.Reused || result.Created || result.Backend != compute.FloatingIPNova || !result.NeutronFloating || result.Attach != nil || result.AttachSkipped {
		t.Fatal(result, err, f.keys())
	}
	if result.Available == nil || result.Available.FallbackError == nil || result.NeutronNotFound == nil || result.NeutronAttach == nil || result.NeutronAttach.Sent {
		t.Fatal(result.Available, result.NeutronNotFound, result.NeutronAttach, f.keys())
	}
	// The newest port's first IPv4 address is the NAT destination, as in
	// _nat_destination_port; the IPv6 address on that port is skipped.
	put, _ := f.find("PUT /network/v2.0/floatingips/12")
	if !strings.Contains(put.body, `"port_id":"port-new"`) || !strings.Contains(put.body, `"fixed_ip_address":"10.0.0.9"`) || result.NeutronAttach.Selection.PortID != "port-new" {
		t.Fatal(put, result.NeutronAttach.Selection)
	}
	// Like Python, the Neutron allocation attempt whose 404 selects Nova precedes
	// the Nova reuse; the only attach is the Neutron PUT.
	if got := cloudNetMutations(f.keys()); len(got) != 2 || got[0] != "POST /network/v2.0/floatingips" || got[1] != "PUT /network/v2.0/floatingips/12" || f.count("POST /compute/servers/srv-1/action") != 0 {
		t.Fatal(got, f.keys())
	}
	if server.Addresses != nil || result.InterfaceIP != "" {
		t.Fatal(server.Addresses, result.InterfaceIP)
	}
}

// TestCloudNetAddAutoIPReuseNovaFallbackRowAttachOutcomes covers the other
// outcomes of that Neutron branch: an accepted PUT and an observed address,
// a port-list NotFound (also ignored, no PUT), a server without ports, several
// ports without a NAT destination network and a rejected PUT (all returned,
// never followed by a Nova action), and a wait that times out without
// deleting the reused row.
func TestCloudNetAddAutoIPReuseNovaFallbackRowAttachOutcomes(t *testing.T) {
	shown := `{"server":{"id":"srv-1","status":"ACTIVE","addresses":{"private":[{"version":4,"addr":"10.0.0.9","OS-EXT-IPS:type":"fixed"},{"version":4,"addr":"198.51.100.21","OS-EXT-IPS:type":"floating"}]}}}`
	for _, scenario := range []string{"accepted and observed", "ports not found", "no ports", "put conflict", "wait timeout", "no NAT destination"} {
		t.Run(scenario, func(t *testing.T) {
			ports, portsCode, put, putBody := cloudNetTwoPorts, 200, 200, `{"floatingip":{"id":"12","port_id":"port-new"}}`
			var serverBody func() string
			options := []compute.AddAutoIPOption{}
			switch scenario {
			case "accepted and observed":
				serverBody = func() string { return shown }
				options = append(options, compute.WithAddAutoIPWait(true), compute.WithAddAutoIPPollInterval(time.Millisecond))
			case "ports not found":
				ports, portsCode = cloudNetNeutronMissing, 404
			case "no ports":
				ports = `{"ports":[]}`
			case "no NAT destination":
				ports = strings.Replace(cloudNetTwoPorts, `"private"`, `"no-nat"`, 1)
			case "put conflict":
				put, putBody = 409, `{"NeutronError":{"type":"Conflict","message":"busy"}}`
			case "wait timeout":
				put, putBody = 404, cloudNetNeutronMissing
				options = append(options, compute.WithAddAutoIPWait(true), compute.WithAddAutoIPTimeout(40*time.Millisecond), compute.WithAddAutoIPPollInterval(5*time.Millisecond))
			}
			conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, cloudNetReuseFallbackHandler(ports, portsCode, put, putBody, serverBody))
			result, err := conn.AddAutoIP(context.Background(), &compute.Server{ID: "srv-1", Status: "ACTIVE"}, options...)
			if result == nil || f.count("POST /compute/servers/srv-1/action") != 0 || f.count("DELETE /network/v2.0/floatingips/12") != 0 || f.count("DELETE /compute/os-floating-ips/12") != 0 {
				t.Fatal(result, err, f.keys())
			}
			puts := f.count("PUT /network/v2.0/floatingips/12")
			switch scenario {
			case "accepted and observed":
				if err != nil || !result.Observed || result.NeutronNotFound != nil || result.NeutronAttach == nil || result.NeutronAttach.Wire == nil || result.InterfaceIP != "198.51.100.21" || puts != 1 {
					t.Fatal(result, err, f.keys())
				}
			case "ports not found":
				if err != nil || result.NeutronNotFound == nil || puts != 0 {
					t.Fatal(result, err, f.keys())
				}
			case "no ports":
				if !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), "unable to find a port") || result.NeutronNotFound != nil || puts != 0 {
					t.Fatal(result, err, f.keys())
				}
			case "no NAT destination":
				if !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), "NAT destination") || puts != 0 {
					t.Fatal(result, err, f.keys())
				}
			case "put conflict":
				if err == nil || result.NeutronNotFound != nil || result.NeutronAttach == nil || puts != 1 {
					t.Fatal(result, err, f.keys())
				}
			case "wait timeout":
				var timeout *compute.AddAutoIPTimeoutError
				if !errors.As(err, &timeout) || result.NeutronNotFound == nil || result.Cleanup != nil || result.Created || puts != 1 {
					t.Fatal(result, err, f.keys())
				}
			}
		})
	}
}

// TestCloudNetFloatingIPAttachByIDSelectsFixedAddressAndValidatesInputs covers
// the Neutron port update primitive directly: an exact fixed address selects
// its port without NAT narrowing, an unmatched fixed address and invalid
// inputs fail before any mutation.
func TestCloudNetFloatingIPAttachByIDSelectsFixedAddressAndValidatesInputs(t *testing.T) {
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, cloudNetReuseFallbackHandler(strings.Replace(cloudNetTwoPorts, `"private"`, `"no-nat"`, 1), 200, 200, `{"floatingip":{"id":"12","port_id":"port-old"}}`, nil))
	service, err := conn.Network(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	update, err := service.FloatingIPs.AttachByID(context.Background(), network.AttachFloatingIPByIDRequest{Server: resource.ID("srv-1"), FloatingIPID: "12", FixedAddress: "10.0.0.5"})
	if err != nil || update == nil || !update.Sent || update.Selection.PortID != "port-old" || update.Selection.FixedIPv4 != "10.0.0.5" || update.Wire == nil || update.Response == nil || update.Response.StatusCode != 200 {
		t.Fatal(update, err, f.keys())
	}
	put, _ := f.find("PUT /network/v2.0/floatingips/12")
	if put.body != `{"floatingip":{"fixed_ip_address":"10.0.0.5","port_id":"port-old"}}` {
		t.Fatal(put.body)
	}
	if _, err := service.FloatingIPs.AttachByID(context.Background(), network.AttachFloatingIPByIDRequest{Server: resource.ID("srv-1"), FloatingIPID: "12", FixedAddress: "10.0.0.77"}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, input := range []network.AttachFloatingIPByIDRequest{{Server: resource.Name("web"), FloatingIPID: "12"}, {Server: resource.ID("srv-1")}, {FloatingIPID: "12"}} {
		if _, err := service.FloatingIPs.AttachByID(context.Background(), input); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(input, err)
		}
	}
	if f.count("PUT /network/v2.0/floatingips/12") != 1 {
		t.Fatal(f.keys())
	}
}
