package openstack

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/compute"
)

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
