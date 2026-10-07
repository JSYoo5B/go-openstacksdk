package network

import "strings"

// ConfiguredNetwork describes routing roles by a literal network name or ID.
// The family flags describe cloud routing policy, not subnet IP versions.
type ConfiguredNetwork struct {
	Name                 string
	RoutesIPv4Externally bool
	RoutesIPv6Externally bool
	NATSource            bool
	NATDestination       bool
	DefaultInterface     bool
}

type networkRoleOptions struct {
	networks                           []ConfiguredNetwork
	externalDisabled, internalDisabled bool
}

// NetworkRolePolicy is an owned immutable configuration. Its zero value enables
// external and internal discovery with no configured role overrides.
type NetworkRolePolicy struct{ options networkRoleOptions }
type NetworkRoleOption func(*networkRoleOptions) error

// PrepareNetworkRoleOptions validates and snapshots a reusable policy.
func PrepareNetworkRoleOptions(options ...NetworkRoleOption) (NetworkRolePolicy, error) {
	o := networkRoleOptions{}
	for _, apply := range options {
		if apply == nil {
			return NetworkRolePolicy{}, floatingIPInvalid("nil network role option")
		}
		if err := apply(&o); err != nil {
			return NetworkRolePolicy{}, err
		}
	}
	o.networks = append([]ConfiguredNetwork(nil), o.networks...)
	return NetworkRolePolicy{options: o}, nil
}

// WithConfiguredNetworks replaces all configured roles; an empty list clears
// inherited roles. Only one default interface and NAT destination are allowed.
// As in the pinned loader, the first configured NAT source is used.
func WithConfiguredNetworks(rows ...ConfiguredNetwork) NetworkRoleOption {
	rows = append([]ConfiguredNetwork(nil), rows...)
	return func(o *networkRoleOptions) error {
		defaults, destinations := 0, 0
		for _, row := range rows {
			if strings.TrimSpace(row.Name) == "" {
				return floatingIPInvalid("configured network requires a nonempty name or ID")
			}
			if row.DefaultInterface {
				defaults++
			}
			if row.NATDestination {
				destinations++
			}
		}
		if defaults > 1 || destinations > 1 {
			return floatingIPInvalid("only one default interface and NAT destination may be configured")
		}
		o.networks = append([]ConfiguredNetwork(nil), rows...)
		return nil
	}
}

func WithExternalNetworkDiscovery(enabled bool) NetworkRoleOption {
	return func(o *networkRoleOptions) error { o.externalDisabled = !enabled; return nil }
}
func WithInternalNetworkDiscovery(enabled bool) NetworkRoleOption {
	return func(o *networkRoleOptions) error { o.internalDisabled = !enabled; return nil }
}
func (p NetworkRolePolicy) UseExternalNetwork() bool { return !p.options.externalDisabled }
func (p NetworkRolePolicy) UseInternalNetwork() bool { return !p.options.internalDisabled }
func (p NetworkRolePolicy) DefaultNetworkSelector() string {
	for _, row := range p.options.networks {
		if row.DefaultInterface {
			return row.Name
		}
	}
	return ""
}
