package bgpvpns

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// NetworkAssociationResources returns a Collection over the network
// associations of one fixed BGP VPN, bound to direct GET
// bgpvpn/bgpvpns/{bgpvpn_id}/network_associations/{id} for the shared wait
// engine, such as network.WaitForDelete. Only resource.ID references are
// supported; status waits require resource.WithStatusAttribute.
func (a *API) NetworkAssociationResources(bgpVPNID string) *resource.Collection[NetworkAssociation] {
	return resource.NewCollection(resource.Adapter[NetworkAssociation]{
		Kind: "bgpvpns",
		Get: func(ctx context.Context, id string) (*NetworkAssociation, error) {
			return a.GetNetworkAssociation(ctx, bgpVPNID, id)
		},
		ID: func(v *NetworkAssociation) string { return v.ID },
	})
}

// PortAssociationResources returns a Collection over the port associations of
// one fixed BGP VPN, bound to direct GET
// bgpvpn/bgpvpns/{bgpvpn_id}/port_associations/{id}. Only resource.ID
// references are supported; status waits require resource.WithStatusAttribute.
func (a *API) PortAssociationResources(bgpVPNID string) *resource.Collection[PortAssociation] {
	return resource.NewCollection(resource.Adapter[PortAssociation]{
		Kind: "bgpvpns",
		Get: func(ctx context.Context, id string) (*PortAssociation, error) {
			return a.GetPortAssociation(ctx, bgpVPNID, id)
		},
		ID: func(v *PortAssociation) string { return v.ID },
	})
}

// RouterAssociationResources returns a Collection over the router
// associations of one fixed BGP VPN, bound to direct GET
// bgpvpn/bgpvpns/{bgpvpn_id}/router_associations/{id}. Only resource.ID
// references are supported; status waits require resource.WithStatusAttribute.
func (a *API) RouterAssociationResources(bgpVPNID string) *resource.Collection[RouterAssociation] {
	return resource.NewCollection(resource.Adapter[RouterAssociation]{
		Kind: "bgpvpns",
		Get: func(ctx context.Context, id string) (*RouterAssociation, error) {
			return a.GetRouterAssociation(ctx, bgpVPNID, id)
		},
		ID: func(v *RouterAssociation) string { return v.ID },
	})
}
