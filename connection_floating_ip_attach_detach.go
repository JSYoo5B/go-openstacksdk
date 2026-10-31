package openstack

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/compute"
)

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
