package openstack

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/compute"
)

func (c *Connection) DeleteFloatingIP(ctx context.Context, input compute.DeleteFloatingIPRequest, options ...compute.FloatingIPDeleteOption) (*compute.DeleteFloatingIPResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().DeleteFloatingIP(ctx, input, options...)
}
