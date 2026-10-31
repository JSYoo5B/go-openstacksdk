package openstack

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func (c *Connection) ListFloatingIPs(ctx context.Context, options ...compute.FloatingIPQueryOption) (*compute.FloatingIPQueryResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().ListFloatingIPs(ctx, options...)
}

func (c *Connection) SearchFloatingIPs(ctx context.Context, input compute.SearchFloatingIPsRequest, options ...compute.FloatingIPQueryOption) (*compute.FloatingIPQueryResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().SearchFloatingIPs(ctx, input, options...)
}

func (c *Connection) GetFloatingIP(ctx context.Context, input compute.GetFloatingIPRequest, options ...compute.FloatingIPQueryOption) (*compute.GetFloatingIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().GetFloatingIP(ctx, input, options...)
}

func (c *Connection) GetFloatingIPByID(ctx context.Context, input compute.GetFloatingIPByIDRequest, options ...compute.FloatingIPQueryOption) (*compute.GetFloatingIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().GetFloatingIPByID(ctx, input, options...)
}

func (c *Connection) ListFloatingIPPools(ctx context.Context, options ...compute.FloatingIPQueryOption) (*compute.FloatingIPPoolQueryResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().ListFloatingIPPools(ctx, options...)
}

func (c *Connection) SearchFloatingIPPools(ctx context.Context, input compute.SearchFloatingIPPoolsRequest, options ...compute.FloatingIPQueryOption) (*compute.FloatingIPPoolQueryResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().SearchFloatingIPPools(ctx, input, options...)
}

// AvailableFloatingIP supplies lazy service discovery and configured source
// policy for free-first availability. A Server is optional; this is not Ensure.
func (c *Connection) AvailableFloatingIP(ctx context.Context, input compute.AvailableFloatingIPRequest, options ...compute.AvailableFloatingIPOption) (*compute.AvailableFloatingIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().AvailableFloatingIP(ctx, input, options...)
}

func (c *Connection) CreateFloatingIP(ctx context.Context, input compute.CreateFloatingIPRequest, options ...compute.FloatingIPCreateOption) (*compute.CreateFloatingIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().CreateFloatingIP(ctx, input, options...)
}

func (c *Connection) DeleteFloatingIP(ctx context.Context, input compute.DeleteFloatingIPRequest, options ...compute.FloatingIPDeleteOption) (*compute.DeleteFloatingIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().DeleteFloatingIP(ctx, input, options...)
}

func (c *Connection) DeleteUnattachedFloatingIPs(ctx context.Context, options ...compute.FloatingIPDeleteOption) (*compute.DeleteUnattachedFloatingIPsResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().DeleteUnattachedFloatingIPs(ctx, options...)
}

// DetachIPFromServer implements detach_ip_from_server over lazily discovered
// Network and Compute services. See compute.Service.DetachIPFromServer for the
// branch, error and no-replay contract.
func (c *Connection) DetachIPFromServer(ctx context.Context, input compute.DetachFloatingIPRequest, options ...compute.FloatingIPQueryOption) (*compute.DetachFloatingIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().DetachIPFromServer(ctx, input, options...)
}

// AddAutoIP implements add_auto_ip over lazily discovered Network and Compute
// services. See compute.Service.AddAutoIP for the reuse, attach, wait and
// timeout-cleanup contract.
func (c *Connection) AddAutoIP(ctx context.Context, server *compute.Server, options ...compute.AddAutoIPOption) (*compute.AddAutoIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().AddAutoIP(ctx, server, options...)
}

// AddIPFromPool implements the pool branch of add_ips_to_server over lazily
// discovered Network and Compute services. See compute.Service.AddIPFromPool
// for the reuse, refresh, attach and wait contract.
func (c *Connection) AddIPFromPool(ctx context.Context, server *compute.Server, pool resource.Ref, options ...compute.AddIPFromPoolOption) (*compute.AddAutoIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().AddIPFromPool(ctx, server, pool, options...)
}
