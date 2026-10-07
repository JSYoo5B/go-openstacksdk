package compute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strconv"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/network"
)

// ServerAddress retains wire family, type and MAC, including missing-vs-empty
// MAC. Fields owns every original JSON field, including unknown extensions.
type ServerAddress struct {
	Version    int
	Address    string
	Type       string
	MACAddress string
	MACPresent bool
	// Supplemental is true for Neutron/Nova IP-list enrichment, rather than
	// an address observed in the supplied Nova server response.
	Supplemental bool
	Fields       map[string]json.RawMessage
}

// ServerAddressView owns the address snapshot and calculated access fields.
// The caller's native Server is never modified. SupplementalError records a
// best-effort Neutron failure; cancellation and source failures remain errors.
type ServerAddressView struct {
	ServerID                            string
	Addresses                           map[string][]ServerAddress
	NetworkOrder                        []string
	PublicIPv4, PublicIPv6, PrivateIPv4 string
	InterfaceIP, AccessIPv4, AccessIPv6 string
	SupplementalError                   error
}

type serverAddressState struct {
	view                   *ServerAddressView
	accessIPv4, accessIPv6 string
	options                serverAddressOptions
	roles                  *network.NetworkRoleSnapshot
	rolesLoaded            bool
	loadRoles              func(context.Context) (*network.NetworkRoleSnapshot, error)
	servers                *Servers
	sourceGuard            func(context.Context) error
}

func (s *Service) prepareAddressView(ctx context.Context, server *Server, options []ServerAddressOption, readAddresses bool) (*serverAddressState, error) {
	collection := s.Servers
	state, err := collection.prepareServerAddresses(ctx, server, options, readAddresses)
	if err != nil {
		return nil, err
	}
	var sourceError error
	state.servers = collection
	state.sourceGuard = func(ctx context.Context) error {
		if ctx.Err() != nil {
			return errors.Join(ctx.Err(), context.Cause(ctx))
		}
		if sourceError == nil && s.Servers != collection {
			sourceError = invalid("compute server collection changed during address calculation")
		}
		return sourceError
	}
	return state, state.sourceGuard(ctx)
}

func (s *Servers) prepareServerAddresses(ctx context.Context, server *Server, options []ServerAddressOption, readAddresses bool) (*serverAddressState, error) {
	if ctx == nil {
		return nil, invalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, context.Cause(ctx))
	}
	if s == nil || server == nil {
		return nil, invalid("server and server collection are required")
	}
	policy, err := prepareServerAddressPolicy(s.addressPolicy, options)
	if err != nil {
		return nil, err
	}
	view := &ServerAddressView{ServerID: server.ID}
	if readAddresses {
		view, err = parseServerAddresses(server, policy.options.networkOrder)
		if err != nil {
			return nil, err
		}
	}
	return &serverAddressState{view: view, accessIPv4: server.AccessIPv4, accessIPv6: server.AccessIPv6,
		options: policy.options, loadRoles: s.dependencies.NetworkRoles}, errors.Join(ctx.Err(), context.Cause(ctx))
}

func parseServerAddresses(server *Server, order []string) (*ServerAddressView, error) {
	view := &ServerAddressView{ServerID: server.ID, Addresses: make(map[string][]ServerAddress)}
	data, err := json.Marshal(server.Addresses)
	if err != nil {
		return nil, invalid("server addresses: %v", err)
	}
	var raw map[string][]map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, invalid("server addresses: %v", err)
	}
	if raw == nil {
		view.Addresses = nil
	}
	var keys []string
	for key, rows := range raw {
		keys = append(keys, key)
		for i, row := range rows {
			if row == nil {
				return nil, invalid("server addresses %q row %d is null", key, i)
			}
			var address ServerAddress
			address.Fields = row
			if _, present := row["version"]; !present {
				return nil, invalid("server addresses %q row %d requires version", key, i)
			}
			if value, present := row["addr"]; !present || string(value) == "null" {
				return nil, invalid("server addresses %q row %d requires a string addr", key, i)
			}
			for field, target := range map[string]any{"version": &address.Version, "addr": &address.Address,
				"OS-EXT-IPS:type": &address.Type, "OS-EXT-IPS-MAC:mac_addr": &address.MACAddress} {
				if value, present := row[field]; present {
					if err := json.Unmarshal(value, target); err != nil {
						return nil, invalid("server addresses %q row %d %s: %v", key, i, field, err)
					}
				}
			}
			mac, present := row["OS-EXT-IPS-MAC:mac_addr"]
			address.MACPresent = present && string(mac) != "null"
			view.Addresses[key] = append(view.Addresses[key], address)
		}
		if rows == nil {
			view.Addresses[key] = nil
		} else if len(rows) == 0 {
			view.Addresses[key] = []ServerAddress{}
		}
	}
	slices.Sort(keys)
	used := make(map[string]bool, len(keys))
	for _, key := range append(slices.Clone(order), keys...) {
		if _, present := raw[key]; present && !used[key] {
			view.NetworkOrder = append(view.NetworkOrder, key)
			used[key] = true
		}
	}
	return view, nil
}

func (state *serverAddressState) networkRoles(ctx context.Context) (*network.NetworkRoleSnapshot, error) {
	if !state.rolesLoaded {
		state.roles = &network.NetworkRoleSnapshot{}
		if state.loadRoles != nil {
			roles, err := state.loadRoles(ctx)
			if err != nil {
				return nil, errors.Join(err, state.sourceGuard(ctx))
			}
			if err := state.sourceGuard(ctx); err != nil {
				return nil, err
			}
			if roles != nil {
				state.roles = roles
			}
		}
		state.rolesLoaded = true
	}
	return state.roles, state.sourceGuard(ctx)
}

func (state *serverAddressState) candidates(version int, tag string, name, mac *string) []string {
	var floating, other []string
	for _, key := range state.view.NetworkOrder {
		if name != nil && key != *name {
			continue
		}
		for _, row := range state.view.Addresses[key] {
			if row.Version != version || (tag != "" && row.Type != tag) ||
				(mac != nil && (!row.MACPresent || row.MACAddress != *mac)) {
				continue
			}
			if row.Type == "floating" {
				floating = append(floating, row.Address)
			} else {
				other = append(other, row.Address)
			}
		}
	}
	return append(floating, other...)
}

func (state *serverAddressState) best(ctx context.Context, addresses []string, public, cloudPublic bool) (string, error) {
	if err := state.sourceGuard(ctx); err != nil {
		return "", err
	}
	if len(addresses) == 0 {
		return "", nil
	}
	if len(addresses) > 1 && !state.options.noProbe && public == cloudPublic {
		for _, address := range addresses {
			candidate, cancel := context.WithTimeout(ctx, state.options.probeBudget)
			reachable := probeAddress(candidate, address, state.options.probePort)
			cancel()
			if err := state.sourceGuard(ctx); err != nil {
				return "", err
			}
			if reachable {
				return address, nil
			}
		}
	}
	return addresses[0], state.sourceGuard(ctx)
}

func probeAddress(ctx context.Context, address string, port int) bool {
	dialer := net.Dialer{Timeout: time.Second}
	for ctx.Err() == nil {
		connection, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(address, strconv.Itoa(port)))
		if err == nil {
			connection.Close()
			return true
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false
		case <-timer.C:
		}
	}
	return false
}

func (state *serverAddressState) privateIPv4(ctx context.Context, enabled bool) (string, error) {
	if !enabled {
		return "", state.sourceGuard(ctx)
	}
	var mac *string
	for _, key := range state.view.NetworkOrder {
		for _, row := range state.view.Addresses[key] {
			if row.Version == 4 && row.Type == "floating" {
				if row.MACPresent {
					value := row.MACAddress
					mac = &value
				}
				goto selectedMAC
			}
		}
	}
selectedMAC:
	roles, err := state.networkRoles(ctx)
	if err != nil {
		return "", err
	}
	for _, tag := range []string{"fixed", ""} {
		for _, role := range roles.InternalIPv4 {
			if role == nil {
				continue
			}
			candidates := state.candidates(4, tag, &role.Name, mac)
			address, err := state.best(ctx, candidates, false, !state.options.private)
			if err != nil || len(candidates) != 0 {
				return address, err
			}
		}
	}
	privateName := "private"
	address, err := state.best(ctx, state.candidates(4, "fixed", &privateName, mac), false, true)
	if err != nil || address != "" {
		return address, err
	}
	return state.best(ctx, state.candidates(4, "", &privateName, nil), false, true)
}

func (state *serverAddressState) publicIPv4(ctx context.Context, enabled bool) (string, error) {
	if !enabled {
		return "", state.sourceGuard(ctx)
	}
	if state.accessIPv4 != "" {
		return state.accessIPv4, state.sourceGuard(ctx)
	}
	roles, err := state.networkRoles(ctx)
	if err != nil {
		return "", err
	}
	for _, role := range roles.ExternalIPv4 {
		if role == nil {
			continue
		}
		candidates := state.candidates(4, "", &role.Name, nil)
		address, err := state.best(ctx, candidates, true, !state.options.private)
		if err != nil || len(candidates) != 0 {
			return address, err
		}
	}
	for _, filter := range [][2]string{{"floating", ""}, {"", "public"}} {
		var name *string
		if filter[1] != "" {
			name = &filter[1]
		}
		candidates := state.candidates(4, filter[0], name, nil)
		address, err := state.best(ctx, candidates, true, !state.options.private)
		if err != nil || len(candidates) != 0 {
			return address, err
		}
	}
	for _, key := range state.view.NetworkOrder {
		for _, row := range state.view.Addresses[key] {
			ip, err := netip.ParseAddr(row.Address)
			if err == nil && ip.Is4() && !python313PrivateIPv4(ip) {
				return ip.String(), state.sourceGuard(ctx)
			}
		}
	}
	return "", state.sourceGuard(ctx)
}

func (state *serverAddressState) publicIPv6(ctx context.Context) (string, error) {
	if state.accessIPv6 != "" {
		return state.accessIPv6, state.sourceGuard(ctx)
	}
	return state.best(ctx, state.candidates(6, "", nil, nil), true, true)
}

func (state *serverAddressState) defaultIP(ctx context.Context) (string, bool, error) {
	roles, err := state.networkRoles(ctx)
	if err != nil || roles.DefaultNetwork == nil {
		return "", false, err
	}
	versions := []int{4}
	if state.options.localIPv6 && !state.options.forceIPv4 {
		versions = []int{6, 4}
	}
	for _, version := range versions {
		candidates := state.candidates(version, "", &roles.DefaultNetwork.Name, nil)
		address, err := state.best(ctx, candidates, true, !state.options.private)
		if err != nil || len(candidates) != 0 {
			return address, len(candidates) != 0, err
		}
	}
	return "", false, state.sourceGuard(ctx)
}

// Match the stable CPython 3.13 IPv4 is_private table, rather than RFC1918-only
// netip.IsPrivate or IsGlobalUnicast. The pinned SDK does not pin Python itself.
var addressPrivateIPv4 = func() []netip.Prefix {
	var result []netip.Prefix
	for _, prefix := range []string{"0.0.0.0/8", "10.0.0.0/8", "127.0.0.0/8", "169.254.0.0/16", "172.16.0.0/12",
		"192.0.0.0/24", "192.0.2.0/24", "192.168.0.0/16", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4"} {
		result = append(result, netip.MustParsePrefix(prefix))
	}
	return result
}()

func python313PrivateIPv4(ip netip.Addr) bool {
	if ip == netip.MustParseAddr("192.0.0.9") || ip == netip.MustParseAddr("192.0.0.10") {
		return false
	}
	for _, prefix := range addressPrivateIPv4 {
		if prefix.Contains(ip) {
			return true
		}
	}
	return false
}

// GetServerPublicIP implements the cloud IPv4 getter; it does not supplement
// Nova addresses or calculate public IPv6. An unavailable address is "".
func (s *Service) GetServerPublicIP(ctx context.Context, server *Server, options ...ServerAddressOption) (string, error) {
	if s == nil {
		return "", invalid("compute service is required")
	}
	readAddresses := server != nil && server.AccessIPv4 == ""
	if s.Servers != nil && !s.dependenciesUseExternal() {
		readAddresses = false
	}
	state, err := s.prepareAddressView(ctx, server, options, readAddresses)
	if err != nil {
		return "", err
	}
	address, err := state.publicIPv4(ctx, state.servers.dependencies.NetworkPolicy.UseExternalNetwork())
	return address, errors.Join(err, state.sourceGuard(ctx))
}

func (s *Service) GetServerPrivateIP(ctx context.Context, server *Server, options ...ServerAddressOption) (string, error) {
	if s == nil {
		return "", invalid("compute service is required")
	}
	readAddresses := s.Servers != nil && s.dependenciesUseInternal()
	state, err := s.prepareAddressView(ctx, server, options, readAddresses)
	if err != nil {
		return "", err
	}
	address, err := state.privateIPv4(ctx, state.servers.dependencies.NetworkPolicy.UseInternalNetwork())
	return address, errors.Join(err, state.sourceGuard(ctx))
}

func (s *Service) dependenciesUseExternal() bool {
	return s.Servers.dependencies.NetworkPolicy.UseExternalNetwork()
}
func (s *Service) dependenciesUseInternal() bool {
	return s.Servers.dependencies.NetworkPolicy.UseInternalNetwork()
}

func wrapServerAddress(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("server addresses/%s: %w", operation, err)
}
