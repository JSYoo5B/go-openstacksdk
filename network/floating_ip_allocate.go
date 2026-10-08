package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// AllocateFloatingIPRequest always creates a fresh Neutron allocation. Network
// is an ordinary name-or-ID lookup; nil/empty uses the floating role/router.
// PortID wins over Server, FixedAddress and NATDestination. No owner or free-IP
// inventory is needed, and no wait or cleanup is performed by this primitive.
type AllocateFloatingIPRequest struct {
	Network                              *string
	Server                               resource.Ref
	PortID, FixedAddress, NATDestination string
}

// FloatingIPAllocation retains accepted response evidence even if decoding,
// reading, closing or a source/context check fails. Wire is a passive resource;
// it is not an executable assignment candidate or proof of port association.
type FloatingIPAllocation struct {
	Selection          FloatingIPSelection
	Allocated          bool
	Wire               *resource.RawResource
	AllocationResponse *FloatingIPAvailabilityResponse
}

func (f *FloatingIPs) Allocate(ctx context.Context, input AllocateFloatingIPRequest) (*FloatingIPAllocation, error) {
	if input.Network != nil {
		value := *input.Network
		input.Network = &value
	}
	p, err := f.NewPlanner(ctx)
	if err != nil {
		return nil, err
	}
	var networkID string
	if input.Network != nil && *input.Network != "" {
		row, findErr := f.allocateNetwork(p).FindIdentity(ctx, *input.Network, resource.WithIdentityFindIgnoreMissing(false))
		if findErr != nil {
			return nil, findErr
		}
		networkID = row.ID
	} else {
		networkID, err = f.planExternalNetwork(ctx, resource.Ref{}, p.Check, p.NetworkRoles)
		if err != nil {
			return nil, err
		}
	}
	if err := resource.ID(networkID).Validate(); err != nil {
		return nil, err
	}
	selection := FloatingIPSelection{NetworkID: networkID}
	if input.PortID != "" {
		if err := resource.ID(input.PortID).Validate(); err != nil {
			return nil, err
		}
		selection.PortID = input.PortID
	} else if input.Server != (resource.Ref{}) {
		if err := input.Server.Validate(); err != nil {
			return nil, err
		}
		serverID := input.Server.String()
		if input.Server.IsName() {
			if f.dependencies.Server == nil {
				return nil, floatingIPInvalid("named allocation server needs a Compute resolver")
			}
			serverID, err = f.dependencies.Server(ctx, input.Server)
			if err = errors.Join(err, p.Check(ctx)); err != nil {
				return nil, err
			}
		}
		if err := resource.ID(serverID).Validate(); err != nil {
			return nil, err
		}
		destination, err := f.allocateDestination(ctx, serverID, input, p)
		if err != nil {
			return nil, err
		}
		selection.ServerID, selection.PortID, selection.PortNetworkID, selection.FixedIPv4 = serverID, destination.PortID, destination.PortNetworkID, destination.FixedIPv4
	}
	return f.allocateSelected(ctx, selection, p)
}

// Both ordinary creation and prepared availability use this minimal raw POST.
func (f *FloatingIPs) allocateSelected(ctx context.Context, selection FloatingIPSelection, p *FloatingIPPlanner) (*FloatingIPAllocation, error) {
	if err := p.Check(ctx); err != nil {
		return nil, err
	}
	fields := map[string]any{"floating_network_id": selection.NetworkID}
	if selection.PortID != "" {
		fields["port_id"] = selection.PortID
	}
	if selection.FixedIPv4 != "" {
		fields["fixed_ip_address"] = selection.FixedIPv4
	}
	// Resource.create raises for HTTP >=400 rather than imposing native POST
	// defaults201/202. Fix the initial cloud policy through physical retries.
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	response, requestErr := rest.DoJSONGuarded(ctx, f.api.RawClient(), p.Check, http.MethodPost, f.api.RawClient().ServiceURL("floatingips"), map[string]any{"floatingip": fields}, nil, codes...)
	result := &FloatingIPAllocation{Selection: selection, Allocated: response != nil}
	if response == nil {
		return result, errors.Join(requestErr, p.Check(ctx))
	}
	result.AllocationResponse = &FloatingIPAvailabilityResponse{Metadata: resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode}, Envelope: bytes.Clone(response.Body)}
	key := "floatingip"
	var envelope map[string]json.RawMessage
	if json.Unmarshal(response.Body, &envelope) == nil {
		if _, present := envelope[key]; !present {
			key = ""
		}
	}
	wire, decodeErr := rest.Decode[resource.RawResource](response, key, func(row *resource.RawResource) *resource.Metadata { return &row.Metadata })
	result.Wire = wire
	return result, errors.Join(requestErr, decodeErr, p.Check(ctx))
}

func (f *FloatingIPs) allocateNetwork(p *FloatingIPPlanner) *resource.Collection[Network] {
	return rest.Collection(rest.CollectionSpec[Network]{Client: f.api.RawClient(), Path: "networks", Kind: "network", SingleKey: "network", PluralKey: "networks", Get: true, IdentityFind: true, Validate: p.Check, SourceGuard: p.Check, Metadata: planMetadata[Network], ID: func(n *Network) string { return n.ID }, Name: func(n *Network) string { return n.Name }, NameQuery: func(s string) string { return s }, ListCodes: []int{200}, Paging: rest.PagePolicy[Network]{HTTPLink: true}})
}

func allocationString(row *resource.RawResource, key string) string {
	var value string
	_ = json.Unmarshal(row.Body[key], &value)
	return value
}
