package openstack

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/compute"
)

func (c *Connection) DeleteUnattachedFloatingIPs(ctx context.Context, options ...compute.FloatingIPDeleteOption) (*compute.DeleteUnattachedFloatingIPsResult, error) {
	if c == nil {
		return nil, invalid("connection is required")
	}
	return c.addressFacade().DeleteUnattachedFloatingIPs(ctx, options...)
}
