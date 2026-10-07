package network

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"net/url"
	"sort"

	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

func (f *FloatingIPs) allocateDestination(ctx context.Context, serverID string, input AllocateFloatingIPRequest, p *FloatingIPPlanner) (FloatingIPSelection, error) {
	var zero FloatingIPSelection
	spec := rest.CollectionSpec[resource.RawResource]{Client: f.api.RawClient(), Path: "ports", Kind: "port", PluralKey: "ports", Validate: p.Check, SourceGuard: p.Check, ListCodes: []int{200}, Metadata: func(row *resource.RawResource) *resource.Metadata { return &row.Metadata }, Paging: rest.PagePolicy[resource.RawResource]{HTTPLink: true}}
	var ports []*resource.RawResource
	for port, err := range rest.List(ctx, spec, url.Values{"device_id": {serverID}}) {
		if err != nil {
			return zero, err
		}
		ports = append(ports, port)
	}
	if len(ports) == 0 {
		return zero, p.Check(ctx)
	}
	if input.FixedAddress == "" && len(ports) > 1 {
		var networkID string
		if input.NATDestination != "" {
			row, err := f.allocateNetwork(p).FindIdentity(ctx, input.NATDestination, resource.WithIdentityFindIgnoreMissing(false))
			if err != nil {
				return zero, errors.Join(floatingIPInvalid("allocation NAT destination unavailable"), err)
			}
			networkID = row.ID
		} else {
			roles, err := p.NetworkRoles(ctx)
			if err != nil {
				return zero, err
			}
			if roles.NATDestination == nil {
				return zero, floatingIPInvalid("multiple allocation ports need a NAT destination")
			}
			networkID = roles.NATDestination.ID
		}
		var matching []*resource.RawResource
		for _, port := range ports {
			if allocationString(port, "network_id") == networkID {
				matching = append(matching, port)
			}
		}
		if len(matching) == 0 {
			return zero, floatingIPInvalid("no server port matches the allocation NAT destination")
		}
		ports = matching
	}
	if input.FixedAddress == "" {
		sort.SliceStable(ports, func(i, j int) bool {
			return allocationString(ports[i], "created_at") > allocationString(ports[j], "created_at")
		})
	}
	for _, port := range ports {
		var fixed []json.RawMessage
		if raw, present := port.Body["fixed_ips"]; present {
			if err := json.Unmarshal(raw, &fixed); err != nil {
				return zero, err
			}
		}
		for _, item := range fixed {
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(item, &fields); err != nil || fields == nil {
				if input.FixedAddress != "" {
					return zero, floatingIPInvalid("fixed-address selection needs object entries")
				}
				continue
			}
			var literal string
			if json.Unmarshal(fields["ip_address"], &literal) != nil {
				continue
			}
			if input.FixedAddress != "" {
				if literal != input.FixedAddress {
					continue
				}
			} else {
				address, err := netip.ParseAddr(literal)
				if err != nil || !address.Is4() {
					continue
				}
			}
			id := allocationString(port, "id")
			if err := resource.ID(id).Validate(); err != nil {
				return zero, err
			}
			return FloatingIPSelection{ServerID: serverID, PortID: id, PortNetworkID: allocationString(port, "network_id"), FixedIPv4: literal}, p.Check(ctx)
		}
	}
	if input.FixedAddress != "" {
		return zero, p.Check(ctx)
	}
	return zero, floatingIPInvalid("no fixed IPv4 address on allocation server ports")
}
