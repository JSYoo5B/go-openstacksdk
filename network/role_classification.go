package network

import "gophercloudsdk/resource"

func classifyNetworkRoles(all []*RoleNetwork, gateways map[string]bool, policy NetworkRolePolicy) (*NetworkRoleSnapshot, error) {
	result := &NetworkRoleSnapshot{}
	external4, external6, internal4, internal6 := map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	var external4Names, external6Names, internal4Names, internal6Names []string
	var natSource, natDestination, defaultNetwork string
	for _, row := range policy.options.networks {
		if row.RoutesIPv4Externally {
			external4[row.Name] = true
			external4Names = append(external4Names, row.Name)
		} else {
			internal4[row.Name] = true
			internal4Names = append(internal4Names, row.Name)
		}
		if row.RoutesIPv6Externally {
			external6[row.Name] = true
			external6Names = append(external6Names, row.Name)
		} else {
			internal6[row.Name] = true
			internal6Names = append(internal6Names, row.Name)
		}
		if row.NATSource && natSource == "" {
			natSource = row.Name
		}
		if row.NATDestination {
			natDestination = row.Name
		}
		if row.DefaultInterface {
			defaultNetwork = row.Name
		}
	}
	in := func(names map[string]bool, n *RoleNetwork) bool { return names[n.Name] || names[n.ID] }
	for _, n := range all {
		if in(external4, n) || ((n.RouterExternal || n.ProviderPhysicalNetwork != "") && !in(internal4, n)) {
			result.ExternalIPv4 = append(result.ExternalIPv4, n)
		}
		if in(internal4, n) || (!n.RouterExternal && n.ProviderPhysicalNetwork == "" && !in(external4, n)) {
			result.InternalIPv4 = append(result.InternalIPv4, n)
		}
		if in(external6, n) || (n.RouterExternal && !in(internal6, n)) {
			result.ExternalIPv6 = append(result.ExternalIPv6, n)
		}
		if in(internal6, n) || (!n.RouterExternal && !in(external6, n)) {
			result.InternalIPv6 = append(result.InternalIPv6, n)
		}
		if natSource == "" && n.RouterExternal {
			result.ExternalIPv4Floating = append(result.ExternalIPv4Floating, n)
			if result.NATSource == nil {
				result.NATSource = n
			}
		}
		// The pinned source retains the last network with a gateway subnet.
		if natDestination == "" && gateways[n.ID] {
			result.NATDestination = n
		}
	}
	for _, group := range []struct {
		names []string
		rows  []*RoleNetwork
		kind  string
	}{
		{external4Names, result.ExternalIPv4, "external IPv4 network"}, {internal4Names, result.InternalIPv4, "internal IPv4 network"},
		{external6Names, result.ExternalIPv6, "external IPv6 network"}, {internal6Names, result.InternalIPv6, "internal IPv6 network"},
	} {
		for _, selector := range group.names {
			found := false
			for _, n := range group.rows {
				found = found || n.Name == selector || n.ID == selector
			}
			if !found {
				return nil, &resource.NotFoundError{Resource: group.kind, Reference: selector}
			}
		}
	}
	selectOne := func(selector, kind string) (*RoleNetwork, error) {
		if selector == "" {
			return nil, nil
		}
		var selected *RoleNetwork
		var ids []string
		for _, n := range all {
			if n.Name == selector || n.ID == selector {
				selected = n
				ids = append(ids, n.ID)
			}
		}
		if len(ids) == 0 {
			return nil, &resource.NotFoundError{Resource: kind, Reference: selector}
		}
		if len(ids) > 1 {
			return nil, &resource.AmbiguousError{Resource: kind, Name: selector, IDs: ids}
		}
		return selected, nil
	}
	var err error
	if natSource != "" {
		result.NATSource, err = selectOne(natSource, "NAT source network")
		if err != nil {
			return nil, err
		}
		result.ExternalIPv4Floating = []*RoleNetwork{result.NATSource}
	}
	if natDestination != "" {
		result.NATDestination, err = selectOne(natDestination, "NAT destination network")
		if err != nil {
			return nil, err
		}
	}
	result.DefaultNetwork, err = selectOne(defaultNetwork, "default network")
	if err != nil {
		return nil, err
	}
	return result, nil
}
