package gophercloudsdk

import (
	"context"
	"gophercloudsdk/network"
)

// GetNetworkRoles shares one successful network/subnet discovery among all role
// getters. A missing catalog endpoint yields empty roles. HTTP/configuration
// failures remain errors and are not cached as an empty successful result.
func (c *Connection) GetNetworkRoles(ctx context.Context) (*network.NetworkRoleSnapshot, error) {
	if ctx == nil {
		return nil, invalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !c.UseExternalNetwork() && !c.UseInternalNetwork() {
		return &network.NetworkRoleSnapshot{}, nil
	}
	service, err := c.Network(ctx)
	if err != nil {
		if missingCatalogEndpoint(err) {
			return &network.NetworkRoleSnapshot{}, ctx.Err()
		}
		return nil, err
	}
	return service.Roles.Discover(ctx)
}

// ResetNetworkRoles does not make an HTTP request or construct a service.
// It invalidates results for subsequent getters, without cancelling callers.
func (c *Connection) ResetNetworkRoles() {
	c.mu.Lock()
	service := c.network
	c.mu.Unlock()
	if service != nil {
		service.Roles.Reset()
	}
}

func (c *Connection) UseExternalNetwork() bool { return c.options.networkRoles.UseExternalNetwork() }
func (c *Connection) UseInternalNetwork() bool { return c.options.networkRoles.UseInternalNetwork() }

func (c *Connection) GetExternalIPv4Networks(ctx context.Context) ([]*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return s.ExternalIPv4, nil
}
func (c *Connection) GetInternalIPv4Networks(ctx context.Context) ([]*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return s.InternalIPv4, nil
}
func (c *Connection) GetExternalIPv6Networks(ctx context.Context) ([]*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return s.ExternalIPv6, nil
}
func (c *Connection) GetInternalIPv6Networks(ctx context.Context) ([]*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return s.InternalIPv6, nil
}
func (c *Connection) GetExternalIPv4FloatingNetworks(ctx context.Context) ([]*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return s.ExternalIPv4Floating, nil
}
func (c *Connection) GetExternalNetworks(ctx context.Context) ([]*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return append(s.ExternalIPv4, s.ExternalIPv6...), nil
}
func (c *Connection) GetInternalNetworks(ctx context.Context) ([]*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return append(s.InternalIPv4, s.InternalIPv6...), nil
}
func (c *Connection) GetNATSource(ctx context.Context) (*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return s.NATSource, nil
}
func (c *Connection) GetNATDestination(ctx context.Context) (*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return s.NATDestination, nil
}
func (c *Connection) GetDefaultNetwork(ctx context.Context) (*network.RoleNetwork, error) {
	s, err := c.GetNetworkRoles(ctx)
	if err != nil {
		return nil, err
	}
	return s.DefaultNetwork, nil
}
