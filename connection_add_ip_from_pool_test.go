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

const poolServerShowing = `{"server":{"id":"srv-1","status":"ACTIVE","addresses":{"private":[{"version":4,"addr":"10.0.0.8","OS-EXT-IPS:type":"fixed"},{"version":4,"addr":"%s","OS-EXT-IPS:type":"floating"}]}}}`

// poolNeutronHandler serves a Neutron pool "public" (ID external) whose
// floating IP list holds an IP already attached to the server and a free IP,
// a single server port, a creatable IP and a Nova server read whose floating
// address the test controls.
func poolNeutronHandler(put int, shown *string) func(r2CloudNetCall) (int, string) {
	return func(c r2CloudNetCall) (int, string) {
		switch c.key() {
		case "GET /network/v2.0/networks":
			return 200, `{"networks":[{"id":"external","name":"public","router:external":true},{"id":"private","name":"private"}]}`
		case "GET /network/v2.0/networks/public":
			return 404, cloudNetNeutronMissing
		case "GET /network/v2.0/subnets":
			return 200, `{"subnets":[{"id":"priv","network_id":"private","gateway_ip":"10.0.0.1"}]}`
		case "GET /network/v2.0/floatingips":
			return 200, `{"floatingips":[` +
				`{"id":"fip-att","floating_network_id":"external","floating_ip_address":"198.51.100.30","port_id":"port-1","fixed_ip_address":"10.0.0.8","project_id":"proj","status":"ACTIVE"},` +
				`{"id":"fip-free","floating_network_id":"external","floating_ip_address":"198.51.100.20","port_id":null,"project_id":"proj","status":"DOWN"},` +
				`{"id":"fip-new","floating_network_id":"external","floating_ip_address":"198.51.100.40","port_id":"port-1","fixed_ip_address":"10.0.0.8","project_id":"proj","status":"ACTIVE"}]}`
		case "GET /network/v2.0/ports":
			return 200, `{"ports":[{"id":"port-1","device_id":"srv-1","network_id":"private","fixed_ips":[{"ip_address":"10.0.0.8"}]}]}`
		case "POST /network/v2.0/floatingips":
			return 201, `{"floatingip":{"id":"fip-new","floating_network_id":"external","floating_ip_address":"198.51.100.40","port_id":"port-1","fixed_ip_address":"10.0.0.8","project_id":"proj","status":"ACTIVE"}}`
		case "PUT /network/v2.0/floatingips/fip-free", "PUT /network/v2.0/floatingips/fip-new":
			if put == 404 {
				return 404, cloudNetNeutronMissing
			}
			return put, `{"floatingip":{"id":"x","port_id":"port-1"}}`
		case "GET /compute/servers/srv-1":
			return 200, strings.Replace(poolServerShowing, "%s", *shown, 1)
		}
		return 0, ""
	}
}

func poolProject() compute.AddIPFromPoolOption {
	return compute.WithPoolIPAvailableOptions(compute.WithAvailableIPNetworkOptions(network.WithAvailableProject("proj")))
}

// TestAddIPFromPoolReuseChoosesUnattachedFirstAndAttachesByID covers the
// reuse lane of _add_ip_from_pool: available_floating_ip(network=pool) picks
// the first unattached IP on the pool network even when another IP is already
// attached to the server, and _attach_ip_to_server sends one Neutron PUT on
// the server port (honouring the fixed address); a PUT NotFound is ignored.
func TestAddIPFromPoolReuseChoosesUnattachedFirstAndAttachesByID(t *testing.T) {
	for _, put := range []int{200, 404} {
		shown := "198.51.100.30"
		conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, poolNeutronHandler(put, &shown))
		server := &compute.Server{ID: "srv-1", Status: "ACTIVE", Addresses: map[string]any{"private": []any{map[string]any{"version": 4, "addr": "198.51.100.30", "OS-EXT-IPS:type": "floating"}}}}
		result, err := conn.AddIPFromPool(context.Background(), server, resource.Name("public"), poolProject(), compute.WithPoolIPFixedAddress("10.0.0.8"))
		if err != nil || result == nil || !result.Reused || result.Created || result.FloatingIPID != "fip-free" || result.AlreadyAttached || result.Attach != nil || result.NeutronAttach == nil || result.Cleanup != nil {
			t.Fatal(put, result, err, f.keys())
		}
		if (put == 404) != (result.NeutronNotFound != nil) {
			t.Fatal(put, result.NeutronNotFound)
		}
		call, _ := f.find("PUT /network/v2.0/floatingips/fip-free")
		if call.body != `{"floatingip":{"fixed_ip_address":"10.0.0.8","port_id":"port-1"}}` {
			t.Fatal(call.body)
		}
		if got := cloudNetMutations(f.keys()); len(got) != 1 || f.count("POST /compute/servers/srv-1/action") != 0 {
			t.Fatal(got)
		}
	}
}

// TestAddIPFromPoolNonReuseRefreshesServerBeforeAttach covers the allocation
// lane: CreateFloatingIP on the pool with the server, the server is read again
// unconditionally, an address already shown skips the attach, otherwise the
// IP is attached again like Python (skip_attach is false here), and a timed-out
// wait never deletes the allocation.
func TestAddIPFromPoolNonReuseRefreshesServerBeforeAttach(t *testing.T) {
	for _, scenario := range []string{"already shown", "attach again", "wait timeout"} {
		t.Run(scenario, func(t *testing.T) {
			shown := "198.51.100.40"
			if scenario != "already shown" {
				shown = "198.51.100.99"
			}
			conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, poolNeutronHandler(200, &shown))
			options := []compute.AddIPFromPoolOption{compute.WithPoolIPReuse(false), poolProject()}
			if scenario == "wait timeout" {
				options = append(options, compute.WithPoolIPWait(true), compute.WithPoolIPTimeout(60*time.Millisecond), compute.WithPoolIPPollInterval(5*time.Millisecond))
			}
			result, err := conn.AddIPFromPool(context.Background(), &compute.Server{ID: "srv-1", Status: "ACTIVE"}, resource.Name("public"), options...)
			mutations := cloudNetMutations(f.keys())
			if result == nil || !result.Created || result.FloatingIPID != "fip-new" || result.Cleanup != nil || f.count("DELETE /network/v2.0/floatingips/fip-new") != 0 {
				t.Fatal(result, err, f.keys())
			}
			keys := f.keys()
			post, refresh := -1, -1
			for i, key := range keys {
				if key == "POST /network/v2.0/floatingips" && post < 0 {
					post = i
				}
				if key == "GET /compute/servers/srv-1" && refresh < 0 {
					refresh = i
				}
			}
			if post < 0 || refresh < post {
				t.Fatal(keys)
			}
			switch scenario {
			case "already shown":
				if err != nil || !result.AlreadyAttached || len(mutations) != 1 || result.InterfaceIP != "198.51.100.40" {
					t.Fatal(result, err, mutations)
				}
			case "attach again":
				if err != nil || result.AlreadyAttached || len(mutations) != 2 || mutations[1] != "PUT /network/v2.0/floatingips/fip-new" {
					t.Fatal(result, err, mutations)
				}
			case "wait timeout":
				var timeout *compute.AddAutoIPTimeoutError
				if !errors.As(err, &timeout) || len(mutations) != 2 {
					t.Fatal(result, err, mutations)
				}
			}
		})
	}
}

// TestAddIPFromPoolNovaSourceAssociatesWithFixedAddress covers the Nova lane:
// the pool's unattached Nova IP is reused and associated by one addFloatingIp
// action carrying the fixed address, without Neutron floating IP or port requests.
func TestAddIPFromPoolNovaSourceAssociatesWithFixedAddress(t *testing.T) {
	attached := false
	body := func() string { return strings.Replace(poolServerShowing, "%s", "198.51.100.21", 1) }
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNova, r2CloudNetNovaAutoHandler(&attached, body))
	result, err := conn.AddIPFromPool(context.Background(), &compute.Server{ID: "srv-1", Status: "ACTIVE"}, resource.Name("public"), compute.WithPoolIPFixedAddress("10.0.0.8"), compute.WithPoolIPWait(true), compute.WithPoolIPPollInterval(time.Millisecond))
	if err != nil || result == nil || result.Backend != compute.FloatingIPNova || result.NeutronFloating || !result.Reused || result.Attach == nil || !result.Observed {
		t.Fatal(result, err, f.keys())
	}
	post, _ := f.find("POST /compute/servers/srv-1/action")
	if !strings.Contains(post.body, `"addFloatingIp"`) || !strings.Contains(post.body, `"fixed_address":"10.0.0.8"`) || !strings.Contains(post.body, "198.51.100.21") || f.count("POST /compute/servers/srv-1/action") != 1 {
		t.Fatal(post, f.keys())
	}
	// Only the final interface view may read network roles.
	for _, key := range f.keys() {
		if strings.Contains(key, "/floatingips") || strings.Contains(key, "/ports") {
			t.Fatal(f.keys())
		}
	}
	if _, err := conn.AddIPFromPool(context.Background(), &compute.Server{ID: "srv-1"}, resource.Ref{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
