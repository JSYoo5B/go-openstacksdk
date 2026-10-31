package connectiontest_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const cloudNetNeutronMissing = `{"NeutronError":{"type":"HTTPNotFound","message":"not found"}}`

// cloudNetNovaFallbackHandler serves a Neutron endpoint whose floating IP
// routes return 404 and a Nova endpoint that lists, allocates, deletes and
// shows the server. It records nothing beyond the fixture's call log.
func cloudNetNovaFallbackHandler(instance string, deleted *bool) func(r2CloudNetCall) (int, string) {
	return func(c r2CloudNetCall) (int, string) {
		switch c.key() {
		case "GET /network/v2.0/floatingips", "POST /network/v2.0/floatingips", "DELETE /network/v2.0/floatingips/12", "GET /network/v2.0/floatingips/12":
			return 404, cloudNetNeutronMissing
		case "GET /network/v2.0/networks":
			return 200, `{"networks":[{"id":"external","name":"public","router:external":true,"subnets":["sub"]},{"id":"private","name":"private"}]}`
		case "GET /network/v2.0/subnets":
			return 200, `{"subnets":[{"id":"sub","network_id":"external","ip_version":4}]}`
		case "GET /network/v2.0/ports":
			return 200, `{"ports":[{"id":"port-1","device_id":"srv-1","network_id":"private","fixed_ips":[{"ip_address":"10.0.0.8"}]}]}`
		case "GET /compute/os-floating-ip-pools":
			return 200, `{"floating_ip_pools":[{"name":"public"}]}`
		case "POST /compute/os-floating-ips":
			return 200, `{"floating_ip":{"id":"12","ip":"198.51.100.21","pool":"public","instance_id":null,"fixed_ip":null}}`
		case "GET /compute/os-floating-ips":
			if *deleted {
				return 200, `{"floating_ips":[]}`
			}
			return 200, `{"floating_ips":[{"id":"12","ip":"198.51.100.21","pool":"public","instance_id":` + instance + `,"fixed_ip":"10.0.0.8"}]}`
		case "GET /compute/os-floating-ips/12":
			if *deleted {
				return 404, `{"itemNotFound":{"message":"gone"}}`
			}
			return 200, `{"floating_ip":{"id":"12","ip":"198.51.100.21","pool":"public","instance_id":null}}`
		case "DELETE /compute/os-floating-ips/12":
			*deleted = true
			return 202, ``
		case "GET /compute/servers/srv-1":
			return 200, `{"server":{"id":"srv-1","status":"ACTIVE","addresses":{"private":[{"version":4,"addr":"10.0.0.8","OS-EXT-IPS:type":"fixed"}]}}}`
		case "POST /compute/servers/srv-1/action":
			return 202, ``
		}
		return 0, ""
	}
}

func cloudNetMutations(keys []string) []string {
	var out []string
	for _, key := range keys {
		if !strings.HasPrefix(key, "GET ") {
			out = append(out, key)
		}
	}
	return out
}

// TestCloudNetDetachNeutronSourceRunsNeutronBranchOnNovaFallbackRow covers the
// pinned edge where the Neutron floating IP list returns 404, the lookup falls
// back to normalized Nova rows, and Python still runs its Neutron branch on
// that row: port_id is None, so it returns False and sends no mutation.
func TestCloudNetDetachNeutronSourceRunsNeutronBranchOnNovaFallbackRow(t *testing.T) {
	deleted := false
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, cloudNetNovaFallbackHandler(`"srv-1"`, &deleted))
	result, err := conn.DetachIPFromServer(context.Background(), compute.DetachFloatingIPRequest{ServerID: "srv-1", FloatingIPID: "12"})
	if err != nil || result == nil || result.Outcome != compute.DetachNotAttached || result.Detached || result.Backend != compute.FloatingIPNeutron || result.Response != nil {
		t.Fatal(result, err, f.keys())
	}
	if result.Lookup == nil || result.Lookup.Backend != compute.FloatingIPNova || result.Lookup.FallbackError == nil || result.Lookup.FloatingIP == nil || result.NeutronNotFound != nil {
		t.Fatal(result.Lookup, f.keys())
	}
	if got := cloudNetMutations(f.keys()); len(got) != 0 || f.count("GET /network/v2.0/floatingips") != 1 || f.count("GET /compute/os-floating-ips") != 1 {
		t.Fatal(got, f.keys())
	}
}

// TestCloudNetDetachNeutronSourceStrictFallbackRowHasNoPortID mirrors Python
// strict mode: the normalized Nova row has no port_id attribute, so the
// Neutron branch fails before any mutation instead of reading None.
func TestCloudNetDetachNeutronSourceStrictFallbackRowHasNoPortID(t *testing.T) {
	deleted := false
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, cloudNetNovaFallbackHandler(`"srv-1"`, &deleted))
	result, err := conn.DetachIPFromServer(context.Background(), compute.DetachFloatingIPRequest{ServerID: "srv-1", FloatingIPID: "12"}, compute.WithFloatingIPQueryStrict(true))
	if !errors.Is(err, resource.ErrInvalidOption) || result == nil || result.Outcome != "" || len(cloudNetMutations(f.keys())) != 0 {
		t.Fatal(result, err, f.keys())
	}
}

// TestCloudNetDetachNeutronSourceTruthyFallbackPortUsesNeutronUpdate shows a
// Nova row that carries a truthy port_id: the Neutron branch sends the Neutron
// PUT, and its 404 is a detach error that never reaches the Nova action.
func TestCloudNetDetachNeutronSourceTruthyFallbackPortUsesNeutronUpdate(t *testing.T) {
	handle := func(c r2CloudNetCall) (int, string) {
		switch c.key() {
		case "GET /network/v2.0/floatingips", "PUT /network/v2.0/floatingips/12":
			return 404, cloudNetNeutronMissing
		case "GET /compute/os-floating-ips":
			return 200, `{"floating_ips":[{"id":"12","ip":"198.51.100.21","pool":"public","instance_id":"srv-1","port_id":"p-1"}]}`
		}
		return 0, ""
	}
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, handle)
	result, err := conn.DetachIPFromServer(context.Background(), compute.DetachFloatingIPRequest{ServerID: "srv-1", FloatingIPID: "12"})
	var detach *compute.DetachFloatingIPError
	if !errors.As(err, &detach) || result == nil || result.Detached || result.Failure == nil || result.Failure.StatusCode != 404 {
		t.Fatal(result, err, f.keys())
	}
	if got := cloudNetMutations(f.keys()); len(got) != 1 || got[0] != "PUT /network/v2.0/floatingips/12" {
		t.Fatal(got)
	}
}

// TestCloudNetDetachNeutronLookupNotFoundRunsNovaBranch covers Python's
// NotFoundException catch: with direct get, the Neutron member GET 404 sends
// the helper to its Nova branch, whose repeated lookup also fails, so the
// result is the second NotFound and nothing is mutated.
func TestCloudNetDetachNeutronLookupNotFoundRunsNovaBranch(t *testing.T) {
	const id = "11111111-2222-3333-4444-555555555555"
	handle := func(c r2CloudNetCall) (int, string) {
		if c.key() == "GET /network/v2.0/floatingips/"+id {
			return 404, cloudNetNeutronMissing
		}
		return 0, ""
	}
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, handle)
	result, err := conn.DetachIPFromServer(context.Background(), compute.DetachFloatingIPRequest{ServerID: "srv-1", FloatingIPID: id}, compute.WithFloatingIPQueryDirectGet(true))
	if err == nil || result == nil || result.NeutronNotFound == nil || result.NeutronLookup == nil || result.Backend != compute.FloatingIPNova || result.Outcome != "" {
		t.Fatal(result, err, f.keys())
	}
	if f.count("GET /network/v2.0/floatingips/"+id) != 2 || len(cloudNetMutations(f.keys())) != 0 {
		t.Fatal(f.keys())
	}
}

// TestCloudNetAddAutoIPNeutronSourceNovaFallbackAllocationSkipsAttach covers
// Python's skip_attach for an allocation that fell back to Nova under the
// Neutron source: no attach request of either kind is sent.
func TestCloudNetAddAutoIPNeutronSourceNovaFallbackAllocationSkipsAttach(t *testing.T) {
	deleted := false
	conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, cloudNetNovaFallbackHandler("null", &deleted))
	result, err := conn.AddAutoIP(context.Background(), &compute.Server{ID: "srv-1"}, compute.WithAddAutoIPReuse(false))
	if err != nil || result == nil || !result.Created || result.Backend != compute.FloatingIPNova || !result.NeutronFloating || !result.AttachSkipped || result.Attach != nil || result.FloatingIPAddr != "198.51.100.21" {
		t.Fatal(result, err, f.keys())
	}
	if result.Creation == nil || result.Creation.FallbackError == nil {
		t.Fatal(result.Creation)
	}
	if got := cloudNetMutations(f.keys()); len(got) != 2 || got[0] != "POST /network/v2.0/floatingips" && got[0] != "POST /compute/os-floating-ips" || got[len(got)-1] != "POST /compute/os-floating-ips" || f.count("POST /compute/servers/srv-1/action") != 0 {
		t.Fatal(got, f.keys())
	}
}

// TestCloudNetAddAutoIPNeutronSourceNovaFallbackTimeoutCleansUp covers the
// Python timeout cleanup condition (_use_neutron_floating() and created),
// which includes the Nova-fallback allocation; WithAddAutoIPCleanup(false)
// keeps the IP.
func TestCloudNetAddAutoIPNeutronSourceNovaFallbackTimeoutCleansUp(t *testing.T) {
	for _, cleanup := range []bool{true, false} {
		deleted := false
		conn, f := newR2CloudNetFixture(t, compute.FloatingIPNeutron, cloudNetNovaFallbackHandler("null", &deleted))
		result, err := conn.AddAutoIP(context.Background(), &compute.Server{ID: "srv-1"}, compute.WithAddAutoIPReuse(false), compute.WithAddAutoIPWait(true),
			compute.WithAddAutoIPTimeout(80*time.Millisecond), compute.WithAddAutoIPPollInterval(5*time.Millisecond), compute.WithAddAutoIPCleanup(cleanup))
		var timeout *compute.AddAutoIPTimeoutError
		if !errors.As(err, &timeout) || result == nil || !result.AttachSkipped || f.count("POST /compute/servers/srv-1/action") != 0 {
			t.Fatal(cleanup, result, err, f.keys())
		}
		// Python deletes through Neutron (the configured source); its 404 is a
		// False result inside _neutron_delete_floating_ip, not a Nova fallback.
		want := 0
		if cleanup {
			want = 1
		}
		if cleanup != (result.Cleanup != nil) || (result.Cleanup != nil && result.Cleanup.Deleted) || deleted || f.count("DELETE /network/v2.0/floatingips/12") != want || f.count("DELETE /compute/os-floating-ips/12") != 0 {
			t.Fatal(cleanup, result.Cleanup, result.CleanupError, f.keys())
		}
	}
}
