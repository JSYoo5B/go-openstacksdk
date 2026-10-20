package openstack

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/gophercloud/gophercloud/v2"
)

func (c *Connection) addressNetworkService(ctx context.Context) (*network.Service, error) {
	service, err := c.Network(ctx)
	if missingCatalogEndpoint(err) {
		return nil, ctx.Err()
	}
	return service, err
}

func (c *Connection) addressComputeClient(ctx context.Context) (*gophercloud.ServiceClient, error) {
	service, err := c.Compute(ctx)
	if err != nil {
		return nil, err
	}
	if service == nil || service.API == nil || service.Servers == nil || service.RawClient() == nil {
		return nil, invalid("raw compute service is required")
	}
	api, servers, collection, flavors, client := service.API, service.Servers, service.Servers.Collection, service.Flavors, service.RawClient()
	apiServers := api.Servers
	provider := c.provider
	rest.RegisterOperationSource(ctx, func(ctx context.Context) error {
		c.mu.Lock()
		cached := c.compute
		c.mu.Unlock()
		if c.provider != provider || cached != service || service.API != api || service.Servers != servers || servers.Collection != collection || service.Flavors != flavors || service.RawClient() != client || api.Servers != apiServers || api.RawClient() != client {
			return invalid("lazy compute service source changed")
		}
		return nil
	})
	return service.RawClient(), nil
}

func (c *Connection) addressFacade() *compute.Service {
	return compute.New(nil, compute.Dependencies{
		CloudLocation: c.CurrentLocation,
		NetworkRoles:  c.GetNetworkRoles, AddressNetworks: c.addressNetworkService, AddressCompute: c.addressComputeClient,
		NetworkPolicy: c.options.networkRoles, ServerAddresses: c.options.serverAddresses,
	})
}

// GetServerPublicIP calculates IPv4 from a supplied Nova model. It needs no
// Compute endpoint and only discovers network roles when selection needs them.
func (c *Connection) GetServerPublicIP(ctx context.Context, server *compute.Server, options ...compute.ServerAddressOption) (string, error) {
	if c == nil {
		return "", invalid("connection is required")
	}
	return c.addressFacade().GetServerPublicIP(ctx, server, options...)
}

func (c *Connection) GetServerPrivateIP(ctx context.Context, server *compute.Server, options ...compute.ServerAddressOption) (string, error) {
	if c == nil {
		return "", invalid("connection is required")
	}
	return c.addressFacade().GetServerPrivateIP(ctx, server, options...)
}

// ExpandServerInterfaces returns an owned view; shared Network roles are cached
// by Connection while address supplementation is refreshed on every call.
func (c *Connection) ExpandServerInterfaces(ctx context.Context, server *compute.Server, options ...compute.ServerAddressOption) (*compute.ServerAddressView, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().ExpandServerInterfaces(ctx, server, options...)
}

// PlanServerFloatingIP keeps known skips free of endpoint discovery. A nil
// address refresh or Nova supplementation requires a lazy Compute endpoint;
// Neutron classification shares Connection's guarded role snapshot.
func (c *Connection) PlanServerFloatingIP(ctx context.Context, input compute.AutomaticFloatingIPRequest, options ...compute.AutomaticFloatingIPOption) (*compute.ServerFloatingIPDecision, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().PlanServerFloatingIP(ctx, input, options...)
}

func (c *Connection) EnsureServerFloatingIP(ctx context.Context, input compute.AutomaticFloatingIPRequest, options ...compute.AutomaticFloatingIPOption) (*compute.AutomaticServerIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().EnsureServerFloatingIP(ctx, input, options...)
}
