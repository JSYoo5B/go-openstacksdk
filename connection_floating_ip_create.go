package openstack

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/compute"
)

func (c *Connection) CreateFloatingIP(ctx context.Context, input compute.CreateFloatingIPRequest, options ...compute.FloatingIPCreateOption) (*compute.CreateFloatingIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().CreateFloatingIP(ctx, input, options...)
}
