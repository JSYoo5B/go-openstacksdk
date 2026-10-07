package network

import (
	"context"
	"errors"
	"net/netip"
	"net/url"
	"sort"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/routers"
)

func planMetadata[T any](*T) *resource.Metadata { return &resource.Metadata{} }

func (f *FloatingIPs) planPorts(guard func(context.Context) error) *resource.Collection[Port] {
	return rest.Collection(f.planPortSpec(guard))
}

func (f *FloatingIPs) planPortSpec(guard func(context.Context) error) rest.CollectionSpec[Port] {
	return rest.CollectionSpec[Port]{Client: f.api.RawClient(), Path: "ports", Kind: "port",
		SingleKey: "port", PluralKey: "ports", Get: true, SourceGuard: guard, Validate: guard,
		ID: func(p *Port) string { return p.ID }, Name: func(p *Port) string { return p.Name }, Metadata: planMetadata[Port],
		NameQuery: func(name string) string { return name },
		ListCodes: []int{200, 204}, Paging: rest.PagePolicy[Port]{HTTPLink: true}}
}

func (f *FloatingIPs) planExternalNetwork(ctx context.Context, ref resource.Ref, guard func(context.Context) error, getRoles func(context.Context) (*NetworkRoleSnapshot, error)) (string, error) {
	if ref != (resource.Ref{}) {
		if !ref.IsName() {
			return ref.String(), guard(ctx)
		}
		spec := rest.CollectionSpec[externalNetwork]{Client: f.api.RawClient(), Path: "networks", Kind: "external network",
			PluralKey: "networks", SourceGuard: guard, Validate: guard, Metadata: planMetadata[externalNetwork], ListCodes: []int{200, 204}, Paging: rest.PagePolicy[externalNetwork]{HTTPLink: true}}
		var ids []string
		for row, err := range rest.List(ctx, spec, url.Values{"name": {ref.String()}, "router:external": {"true"}}) {
			if err != nil {
				return "", err
			}
			if row.External && row.Name == ref.String() {
				if err := resource.ID(row.ID).Validate(); err != nil {
					return "", err
				}
				ids = append(ids, row.ID)
			}
		}
		if len(ids) == 0 {
			return "", &resource.NotFoundError{Resource: "external network", Reference: ref.String()}
		}
		if len(ids) > 1 {
			sort.Strings(ids)
			return "", &resource.AmbiguousError{Resource: "external network", Name: ref.String(), IDs: ids}
		}
		return ids[0], guard(ctx)
	}
	roles, err := getRoles(ctx)
	if err = errors.Join(err, guard(ctx)); err != nil {
		return "", err
	}
	if len(roles.ExternalIPv4Floating) > 0 {
		id := roles.ExternalIPv4Floating[0].ID
		return id, resource.ID(id).Validate()
	}
	spec := rest.CollectionSpec[routers.Router]{Client: f.api.RawClient(), Path: "routers", Kind: "router",
		PluralKey: "routers", SourceGuard: guard, Validate: guard, Metadata: planMetadata[routers.Router], ListCodes: []int{200, 204}, Paging: rest.PagePolicy[routers.Router]{HTTPLink: true}}
	var gateway string
	for router, err := range rest.List(ctx, spec, nil) {
		if err != nil {
			return "", err
		}
		if gateway == "" && router.AdminStateUp && router.GatewayInfo.NetworkID != "" {
			gateway = router.GatewayInfo.NetworkID
			if err := resource.ID(gateway).Validate(); err != nil {
				return "", err
			}
		}
	}
	if gateway == "" {
		return "", &FloatingIPPlanUnavailableError{Reason: NoFloatingIPExternalNetwork}
	}
	return gateway, guard(ctx)
}

func (f *FloatingIPs) planDestination(ctx context.Context, serverID string, options createFloatingIPOptions, guard func(context.Context) error, getRoles func(context.Context) (*NetworkRoleSnapshot, error)) (FloatingIPSelection, error) {
	var zero FloatingIPSelection
	networkID := ""
	if options.destination != nil {
		collection := rest.Collection(rest.CollectionSpec[Network]{Client: f.api.RawClient(), Path: "networks", Kind: "network",
			PluralKey: "networks", SourceGuard: guard, Validate: guard, Metadata: planMetadata[Network], ListCodes: []int{200, 204},
			ID: func(n *Network) string { return n.ID }, Name: func(n *Network) string { return n.Name }, NameQuery: func(name string) string { return name }, Paging: rest.PagePolicy[Network]{HTTPLink: true}})
		var err error
		networkID, err = collection.ResolveID(ctx, *options.destination)
		if err != nil {
			return zero, err
		}
	}
	ports := f.planPorts(guard)
	var owned []*Port
	if options.port != nil {
		port, err := ports.Find(ctx, *options.port)
		if err != nil {
			return zero, err
		}
		if !options.port.IsName() && port.ID != options.port.String() {
			return zero, floatingIPInvalid("Neutron returned a different requested port")
		}
		if port.DeviceID == serverID {
			owned = append(owned, port)
		}
	} else {
		for port, err := range ports.List(ctx, resource.WithQuery("device_id", serverID)) {
			if err != nil {
				return zero, err
			}
			if port.DeviceID == serverID {
				owned = append(owned, port)
			}
		}
		if len(owned) == 0 {
			return zero, &FloatingIPPlanUnavailableError{Reason: NoFloatingIPServerPorts}
		}
		if options.destination == nil && options.fixedAddress == "" && len(owned) > 1 {
			roles, err := getRoles(ctx)
			if err = errors.Join(err, guard(ctx)); err != nil {
				return zero, err
			}
			if roles.NATDestination == nil {
				ids := make([]string, len(owned))
				for i, port := range owned {
					ids[i] = port.ID
				}
				sort.Strings(ids)
				return zero, &resource.AmbiguousError{Resource: "floating IP NAT destination", Name: serverID, IDs: ids}
			}
			networkID = roles.NATDestination.ID
			if err := resource.ID(networkID).Validate(); err != nil {
				return zero, err
			}
		}
	}
	var candidates []FloatingIPSelection
	seen := map[FloatingIPSelection]bool{}
	for _, port := range owned {
		if networkID != "" && port.NetworkID != networkID {
			continue
		}
		for _, fixed := range port.FixedIPs {
			address, err := netip.ParseAddr(fixed.IPAddress)
			if err != nil || !address.Is4() || options.fixedAddress != "" && address.String() != options.fixedAddress {
				continue
			}
			for _, id := range []string{port.ID, port.NetworkID} {
				if err := resource.ID(id).Validate(); err != nil {
					return zero, err
				}
			}
			candidate := FloatingIPSelection{ServerID: serverID, PortNetworkID: port.NetworkID, PortID: port.ID, FixedIPv4: address.String()}
			if !seen[candidate] {
				candidates = append(candidates, candidate)
				seen[candidate] = true
			}
		}
	}
	if len(candidates) == 0 {
		if options.fixedAddress != "" && len(owned) > 0 && options.destination == nil && options.port == nil {
			return zero, &FloatingIPPlanUnavailableError{Reason: NoFloatingIPFixedMatch}
		}
		return zero, &resource.NotFoundError{Resource: "floating IP destination IPv4", Reference: serverID}
	}
	if len(candidates) > 1 {
		ids := make([]string, len(candidates))
		for i, candidate := range candidates {
			ids[i] = candidate.PortID + "@" + candidate.FixedIPv4
		}
		sort.Strings(ids)
		return zero, &resource.AmbiguousError{Resource: "floating IP destination", Name: serverID, IDs: ids}
	}
	return candidates[0], guard(ctx)
}
