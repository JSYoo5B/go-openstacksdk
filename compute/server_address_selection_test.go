package compute_test

import (
	"context"
	"errors"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func addressRow(version int, ip, tag string, mac ...string) map[string]any {
	row := map[string]any{"version": version, "addr": ip}
	if tag != "" {
		row["OS-EXT-IPS:type"] = tag
	}
	if len(mac) != 0 {
		row["OS-EXT-IPS-MAC:mac_addr"] = mac[0]
	}
	return row
}

func addressRole(name string) *network.RoleNetwork {
	return &network.RoleNetwork{Network: network.Network{ID: "network-" + name, Name: name}}
}

func TestServerPrivateAddressRoleMACAndLegacyFallback(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		addresses map[string]any
		roles     []string
		want      string
	}{
		{"MAC selects second fixed", map[string]any{"inside": []any{addressRow(4, "10.0.0.1", "fixed", "first"), addressRow(4, "10.0.0.2", "fixed", "second")}, "outside": []any{addressRow(4, "8.8.8.8", "floating", "second")}}, []string{"inside"}, "10.0.0.2"},
		{"fixed pass precedes all legacy rows", map[string]any{"one": []any{addressRow(4, "10.0.0.1", "", "mac")}, "two": []any{addressRow(4, "10.0.0.2", "fixed", "mac")}}, []string{"one", "two"}, "10.0.0.2"},
		{"tagless second pass", map[string]any{"inside": []any{addressRow(4, "10.0.0.3", "", "mac")}}, []string{"inside"}, "10.0.0.3"},
		{"empty MAC differs from missing", map[string]any{"inside": []any{addressRow(4, "10.0.0.1", "fixed"), addressRow(4, "10.0.0.2", "fixed", "")}, "outside": []any{addressRow(4, "8.8.8.8", "floating", "")}}, []string{"inside"}, "10.0.0.2"},
		{"MAC comparison is case sensitive", map[string]any{"inside": []any{addressRow(4, "10.0.0.1", "fixed", "AA")}, "outside": []any{addressRow(4, "8.8.8.8", "floating", "aa")}}, []string{"inside"}, ""},
		{"literal private last pass drops MAC", map[string]any{"private": []any{addressRow(4, "10.0.0.4", "fixed", "other")}, "outside": []any{addressRow(4, "8.8.8.8", "floating", "mac")}}, nil, "10.0.0.4"},
		{"empty role name is an exact key", map[string]any{"": []any{addressRow(4, "10.0.0.5", "fixed")}, "aaa": []any{addressRow(4, "10.0.0.6", "fixed")}}, []string{""}, "10.0.0.5"},
		{"IPv6 floating MAC is ignored", map[string]any{"inside": []any{addressRow(4, "10.0.0.1", "fixed", "a")}, "outside": []any{addressRow(6, "2001:db8::1", "floating", "b")}}, []string{"inside"}, "10.0.0.1"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			calls := 0
			deps := compute.Dependencies{NetworkRoles: func(context.Context) (*network.NetworkRoleSnapshot, error) {
				calls++
				roles := &network.NetworkRoleSnapshot{}
				for _, name := range scenario.roles {
					roles.InternalIPv4 = append(roles.InternalIPv4, addressRole(name))
				}
				return roles, nil
			}}
			got, err := compute.New(nil, deps).GetServerPrivateIP(context.Background(), &compute.Server{Addresses: scenario.addresses}, compute.WithAddressReachability(false))
			if got != scenario.want || err != nil || calls != 1 {
				t.Fatalf("private=%q want=%q err=%v role calls=%d", got, scenario.want, err, calls)
			}
		})
	}
}

func TestServerPublicAddressOrderingRolesFloatingAndAccessShortCircuit(t *testing.T) {
	server := &compute.Server{Addresses: map[string]any{
		"aa":     []any{addressRow(4, "10.0.0.1", "fixed"), addressRow(4, "8.8.8.8", "floating")},
		"zz":     []any{addressRow(4, "1.1.1.1", "floating")},
		"public": []any{addressRow(4, "9.9.9.9", "fixed")},
	}}
	service := compute.New(nil, compute.Dependencies{})
	got, err := service.GetServerPublicIP(context.Background(), server, compute.WithAddressReachability(false))
	if got != "8.8.8.8" || err != nil {
		t.Fatalf("default public=%q err=%v", got, err)
	}
	order := []string{"zz", "aa"}
	option := compute.WithAddressNetworkOrder(order...)
	order[0] = "public"
	got, err = service.GetServerPublicIP(context.Background(), server, option, compute.WithAddressReachability(false))
	if got != "1.1.1.1" || err != nil {
		t.Fatalf("explicit order public=%q err=%v", got, err)
	}
	calls := 0
	service = compute.New(nil, compute.Dependencies{NetworkRoles: func(context.Context) (*network.NetworkRoleSnapshot, error) {
		calls++
		return &network.NetworkRoleSnapshot{ExternalIPv4: []*network.RoleNetwork{addressRole("public")}}, nil
	}})
	got, err = service.GetServerPublicIP(context.Background(), server, compute.WithAddressReachability(false))
	if got != "9.9.9.9" || err != nil || calls != 1 {
		t.Fatalf("role public=%q err=%v calls=%d", got, err, calls)
	}
	server.AccessIPv4 = "literal-access"
	server.Addresses = map[string]any{"bad": func() {}}
	got, err = service.GetServerPublicIP(context.Background(), server)
	if got != "literal-access" || err != nil || calls != 1 {
		t.Fatalf("access public=%q err=%v calls=%d", got, err, calls)
	}
}

func TestServerAddressUseFlagsSkipMalformedAddressesAndNetwork(t *testing.T) {
	policy, err := network.PrepareNetworkRoleOptions(network.WithExternalNetworkDiscovery(false), network.WithInternalNetworkDiscovery(false))
	if err != nil {
		t.Fatal(err)
	}
	service := compute.New(nil, compute.Dependencies{NetworkPolicy: policy, NetworkRoles: func(context.Context) (*network.NetworkRoleSnapshot, error) {
		t.Fatal("disabled getter selected network roles")
		return nil, nil
	}})
	server := &compute.Server{AccessIPv4: "8.8.8.8", Addresses: map[string]any{"bad": func() {}}}
	public, pubErr := service.GetServerPublicIP(context.Background(), server)
	private, priErr := service.GetServerPrivateIP(context.Background(), server)
	if public != "" || private != "" || pubErr != nil || priErr != nil {
		t.Fatalf("public=%q private=%q errors=%v/%v", public, private, pubErr, priErr)
	}
}

func TestServerPublicAddressFinalFallbackUsesPython313Classification(t *testing.T) {
	for _, scenario := range []struct {
		address  string
		accepted bool
	}{
		{"8.8.8.8", true}, {"100.64.0.1", true}, {"224.0.0.1", true}, {"192.0.0.9", true}, {"192.0.0.10", true},
		{"0.0.0.0", false}, {"10.0.0.1", false}, {"127.0.0.1", false}, {"169.254.1.1", false}, {"172.16.0.1", false},
		{"192.0.0.11", false}, {"192.0.2.1", false}, {"192.168.0.1", false}, {"198.18.0.1", false}, {"198.51.100.1", false}, {"203.0.113.1", false},
		{"240.0.0.1", false}, {"255.255.255.255", false}, {"192.000.0.1", false}, {"::ffff:8.8.8.8", false}, {"2001:db8::1", false}, {"not-an-ip", false},
	} {
		t.Run(scenario.address, func(t *testing.T) {
			server := &compute.Server{Addresses: map[string]any{"unclassified": []any{addressRow(6, scenario.address, "")}}}
			got, err := compute.New(nil, compute.Dependencies{}).GetServerPublicIP(context.Background(), server)
			want := ""
			if scenario.accepted {
				want = scenario.address
			}
			if got != want || err != nil {
				t.Fatalf("fallback=%q want=%q err=%v", got, want, err)
			}
		})
	}
}

func TestServerAddressReachabilitySelectionDirectionFallbackAndContext(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	server := &compute.Server{Addresses: map[string]any{"public": []any{addressRow(4, "127.0.0.2", "fixed"), addressRow(4, "127.0.0.1", "fixed")}}}
	service := compute.New(nil, compute.Dependencies{})
	options := []compute.ServerAddressOption{compute.WithAddressProbePort(port), compute.WithAddressProbeBudget(20 * time.Millisecond)}
	got, err := service.GetServerPublicIP(context.Background(), server, options...)
	if got != "127.0.0.1" || err != nil {
		t.Fatalf("reachable=%q err=%v port=%s", got, err, strconv.Itoa(port))
	}
	got, err = service.GetServerPublicIP(context.Background(), server, append(options, compute.WithPrivateCloud(true))...)
	if got != "127.0.0.2" || err != nil {
		t.Fatalf("private cloud skips external probes: %q %v", got, err)
	}
	listener.Close()
	got, err = service.GetServerPublicIP(context.Background(), server, options...)
	if got != "127.0.0.2" || err != nil {
		t.Fatalf("all failed fallback=%q %v", got, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	got, err = service.GetServerPublicIP(ctx, server, compute.WithAddressProbePort(port), compute.WithAddressProbeBudget(time.Second))
	if got != "" || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("canceled=%q err=%v", got, err)
	}
}

func TestServerAddressValidationPrecedesDependencyAndPropagatesRoleFailure(t *testing.T) {
	wantError := errors.New("role inventory failed")
	calls := 0
	service := compute.New(nil, compute.Dependencies{NetworkRoles: func(context.Context) (*network.NetworkRoleSnapshot, error) { calls++; return nil, wantError }})
	server := &compute.Server{Addresses: map[string]any{"private": []any{addressRow(4, "10.0.0.1", "fixed")}}}
	for _, option := range []compute.ServerAddressOption{nil, compute.WithAddressProbeBudget(0), compute.WithAddressProbePort(0), compute.WithFloatingIPSource("bad"), compute.WithAddressNetworkOrder("same", "same")} {
		if _, err := service.GetServerPrivateIP(context.Background(), server, option); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatalf("invalid option err=%v calls=%d", err, calls)
		}
	}
	if _, err := service.GetServerPrivateIP(nil, server); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatalf("nil ctx=%v calls=%d", err, calls)
	}
	if _, err := service.GetServerPrivateIP(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatalf("nil server=%v calls=%d", err, calls)
	}
	bad := &compute.Server{Addresses: map[string]any{"private": []any{map[string]any{"version": "4", "addr": "10.0.0.1"}}}}
	if _, err := service.GetServerPrivateIP(context.Background(), bad); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatalf("bad addresses=%v calls=%d", err, calls)
	}
	if _, err := service.GetServerPrivateIP(context.Background(), server); !errors.Is(err, wantError) || calls != 1 {
		t.Fatalf("role failure=%v calls=%d", err, calls)
	}
}
