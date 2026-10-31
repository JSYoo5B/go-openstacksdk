package openstack

import (
	"context"
	"errors"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/network"

	"github.com/gophercloud/gophercloud/v2"
)

// addressNetworkService returns nil (no Neutron) for a missing endpoint and for
// a network service disabled by ServiceEnabled, the has_service('network')
// half of Python's _use_neutron_floating.
func (c *Connection) addressNetworkService(ctx context.Context) (*network.Service, error) {
	if !c.ServiceEnabled("network") {
		return nil, ctx.Err()
	}
	service, err := c.Network(ctx)
	if err != nil && c.networkLookupUnavailable(ctx, err) {
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

// AddIPsToServer provides standalone pool/list/automatic assignment with lazy
// services, async attachment and a single 60-second budget by default.
func (c *Connection) AddIPsToServer(ctx context.Context, input compute.AutomaticFloatingIPRequest, options ...compute.ServerIPOption) (*compute.AutomaticServerIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().AddIPsToServer(ctx, input, options...)
}

// AddIPList attaches positional addresses in order. Empty is a no-op and cannot
// fall through to automatic allocation; Compute discovery remains lazy.
func (c *Connection) AddIPList(ctx context.Context, server *compute.Server, addresses []string, options ...compute.ServerIPOption) (*compute.AutomaticServerIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().AddIPList(ctx, server, addresses, options...)
}

// GetActiveServer checks supplied metadata and keeps known branches free of
// endpoint discovery. Needed Neutron assignment obtains only lazy services.
func (c *Connection) GetActiveServer(ctx context.Context, input compute.AutomaticFloatingIPRequest, options ...compute.ServerReadyOption) (*compute.AutomaticServerIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().GetActiveServer(ctx, input, options...)
}

// WaitForServer validates policy before obtaining a raw Compute endpoint. The
// library supplies lazy service bindings and conditional floating-IP behavior.
func (c *Connection) WaitForServer(ctx context.Context, input compute.AutomaticFloatingIPRequest, options ...compute.ServerReadyOption) (*compute.AutomaticServerIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().WaitForServer(ctx, input, options...)
}

// CreateWithAutomaticFloatingIP discovers Compute lazily, then delegates all
// creation, readiness and automatic-address policy to the owned service lane.
func (c *Connection) CreateWithAutomaticFloatingIP(ctx context.Context, request compute.CreateServerRequest, options compute.AutomaticServerCreateOptions) (*compute.AutomaticServerCreateResult, error) {
	if ctx == nil {
		return nil, invalid("context is required")
	}
	if c == nil {
		return nil, invalid("connection is required")
	}
	service, err := c.Compute(ctx)
	if err != nil {
		return nil, errors.Join(err, ctx.Err(), context.Cause(ctx))
	}
	return service.CreateWithAutomaticFloatingIP(ctx, request, options)
}

// CreateRefreshedServer is Python's cloud create_server with wait=False on
// the Connection-owned Compute service: POST, one GET, the ERROR fault check
// and interface expansion with Connection-configured address policy.
func (c *Connection) CreateRefreshedServer(ctx context.Context, request compute.CreateServerRequest, serverOptions []compute.CreateServerOption, addressOptions ...compute.ServerAddressOption) (*compute.RefreshedServerCreateResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	if ctx == nil {
		return nil, invalid("context is required")
	}
	service, err := c.Compute(ctx)
	if err != nil {
		return nil, err
	}
	return service.CreateRefreshedServer(ctx, request, serverOptions, addressOptions...)
}
