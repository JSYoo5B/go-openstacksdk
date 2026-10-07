package gophercloudsdk

import (
	"context"
	"gophercloudsdk/compute"
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
