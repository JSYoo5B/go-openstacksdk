package gophercloudsdk

import (
	"context"
	"errors"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/compute"
	"gophercloudsdk/network"
)

func (c *Connection) addressNetworkService(ctx context.Context) (*network.Service, error) {
	service, err := c.Network(ctx)
	var missing *gophercloud.ErrEndpointNotFound
	var missingValue gophercloud.ErrEndpointNotFound
	if errors.As(err, &missing) || errors.As(err, &missingValue) {
		return nil, ctx.Err()
	}
	return service, err
}

func (c *Connection) addressComputeClient(ctx context.Context) (*gophercloud.ServiceClient, error) {
	service, err := c.Compute(ctx)
	if err != nil {
		return nil, err
	}
	return service.RawClient(), nil
}

func (c *Connection) addressFacade() *compute.Service {
	return compute.New(nil, compute.Dependencies{
		NetworkRoles: c.GetNetworkRoles, AddressNetworks: c.addressNetworkService, AddressCompute: c.addressComputeClient,
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
