package openstack

import (
	"fmt"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/network"
)

// Called after the existing default-network parser validates list/row/flag
// shapes. Secure > base > public already replaced the entire networks list.
func configuredNetworkRoles(settings map[string]any) (network.NetworkRolePolicy, error) {
	var rows []network.ConfiguredNetwork
	if raw, present := settings["networks"]; present {
		for i, rawRow := range raw.([]any) {
			row := rawRow.(map[any]any)
			base, _ := networkConfigBoolean(row["routes_externally"])
			ipv4, ipv6 := base, base
			if value, present := row["routes_ipv4_externally"]; present {
				ipv4, _ = networkConfigBoolean(value)
			}
			if value, present := row["routes_ipv6_externally"]; present {
				ipv6, _ = networkConfigBoolean(value)
			}
			source, _ := networkConfigBoolean(row["nat_source"])
			destination, _ := networkConfigBoolean(row["nat_destination"])
			defaultInterface, _ := networkConfigBoolean(row["default_interface"])
			name, ok := row["name"].(string)
			if !ok {
				return network.NetworkRolePolicy{}, invalid("network entry %d requires a string name or ID", i)
			}
			rows = append(rows, network.ConfiguredNetwork{Name: name, RoutesIPv4Externally: ipv4, RoutesIPv6Externally: ipv6, NATSource: source, NATDestination: destination, DefaultInterface: defaultInterface})
		}
	} else {
		for _, key := range []string{"external_network", "internal_network"} {
			value, present := settings[key]
			if !present || value == nil || value == "" {
				continue
			}
			name, ok := value.(string)
			if !ok || strings.TrimSpace(name) == "" {
				return network.NetworkRolePolicy{}, invalid("%s must be a nonempty string", key)
			}
			external := key == "external_network"
			rows = append(rows, network.ConfiguredNetwork{Name: name, RoutesIPv4Externally: external, RoutesIPv6Externally: external, NATDestination: !external, DefaultInterface: external})
		}
	}
	external, internal := true, true
	for _, flag := range []struct {
		name  string
		value *bool
	}{{"use_external_network", &external}, {"use_internal_network", &internal}} {
		if value, present := settings[flag.name]; present {
			parsed, err := networkConfigBoolean(value)
			if err != nil {
				return network.NetworkRolePolicy{}, fmt.Errorf("%s: %w", flag.name, err)
			}
			*flag.value = parsed
		}
	}
	return network.PrepareNetworkRoleOptions(network.WithConfiguredNetworks(rows...), network.WithExternalNetworkDiscovery(external), network.WithInternalNetworkDiscovery(internal))
}
